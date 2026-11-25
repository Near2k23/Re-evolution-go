// Package chatwoot conecta cada línea con una bandeja de Chatwoot: los mensajes
// de WhatsApp llegan como conversaciones y lo que responden los agentes sale
// por la línea.
package chatwoot

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/evolution-foundation/evolution-go/pkg/chatstore"
	"github.com/gomessguii/logger"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Config es la conexión de una línea con Chatwoot.
type Config struct {
	InstanceID          string    `json:"-" gorm:"primaryKey;size:64"`
	Enabled             bool      `json:"enabled"`
	URL                 string    `json:"url"`
	AccountID           int       `json:"accountId"`
	Token               string    `json:"-"`
	InboxID             int       `json:"inboxId"`
	InboxName           string    `json:"inboxName"`
	SignMessages        bool      `json:"signMessages"`
	ReopenConversation  bool      `json:"reopenConversation"`
	ConversationPending bool      `json:"conversationPending"`
	IgnoreGroups        bool      `json:"ignoreGroups"`
	WebhookSecret       string    `json:"-"`
	UpdatedAt           time.Time `json:"updatedAt"`
}

func (Config) TableName() string { return "chatwoot_configs" }

// Link guarda qué contacto y conversación de Chatwoot corresponden a un chat.
type Link struct {
	InstanceID     string `gorm:"primaryKey;size:64"`
	JID            string `gorm:"column:jid;primaryKey;size:128"`
	ContactID      int
	SourceID       string
	ConversationID int `gorm:"index"`
}

func (Link) TableName() string { return "chatwoot_links" }

// Sender envía por la línea (servicio de envío del servidor).
type Sender interface {
	SendText(instanceID, jid, text string) error
	SendMedia(instanceID, jid, url, mediaType, caption, fileName string) error
}

type ClientGetter func(instanceID string) *whatsmeow.Client

const sourcePrefix = "WAID:"

// Service es la integración con Chatwoot.
type Service struct {
	db     *gorm.DB
	send   Sender
	client ClientGetter
	http   *http.Client
	cache  sync.Map // instanceID → *Config
	locks  sync.Map // instanceID|jid → *sync.Mutex
	// Envíos en curso que vienen de Chatwoot (no deben volver a Chatwoot).
	fromCW sync.Map // instanceID|jid → int (contador)
}

func New(db *gorm.DB, send Sender, client ClientGetter) *Service {
	return &Service{db: db, send: send, client: client, http: &http.Client{Timeout: 60 * time.Second}}
}

func (s *Service) Migrate() error { return s.db.AutoMigrate(&Config{}, &Link{}) }

func (s *Service) Get(instanceID string) *Config {
	if v, ok := s.cache.Load(instanceID); ok {
		c, _ := v.(*Config)
		return c
	}
	var c Config
	if err := s.db.First(&c, "instance_id = ?", instanceID).Error; err != nil {
		s.cache.Store(instanceID, (*Config)(nil))
		return nil
	}
	s.cache.Store(instanceID, &c)
	return &c
}

// SaveInput es lo que se puede cambiar desde la API.
type SaveInput struct {
	Enabled             bool   `json:"enabled"`
	URL                 string `json:"url"`
	AccountID           int    `json:"accountId"`
	Token               string `json:"token"` // vacío = conservar
	InboxID             int    `json:"inboxId"`
	InboxName           string `json:"inboxName"`
	SignMessages        bool   `json:"signMessages"`
	ReopenConversation  bool   `json:"reopenConversation"`
	ConversationPending bool   `json:"conversationPending"`
	IgnoreGroups        bool   `json:"ignoreGroups"`
}

// Save guarda la configuración y, si no hay bandeja, crea una de tipo API con
// el webhook apuntando a este servidor (publicBase + /chatwoot/webhook/…).
func (s *Service) Save(ctx context.Context, instanceID, publicBase string, in SaveInput) (*Config, error) {
	c := Config{InstanceID: instanceID}
	if old := s.Get(instanceID); old != nil {
		c = *old
	}
	c.Enabled, c.URL, c.AccountID = in.Enabled, strings.TrimRight(strings.TrimSpace(in.URL), "/"), in.AccountID
	if in.Token != "" {
		c.Token = strings.TrimSpace(in.Token)
	}
	c.InboxID, c.InboxName = in.InboxID, strings.TrimSpace(in.InboxName)
	c.SignMessages, c.ReopenConversation, c.ConversationPending, c.IgnoreGroups = in.SignMessages, in.ReopenConversation, in.ConversationPending, in.IgnoreGroups
	if c.WebhookSecret == "" {
		b := make([]byte, 16)
		_, _ = rand.Read(b)
		c.WebhookSecret = hex.EncodeToString(b)
	}
	if c.Enabled {
		if c.URL == "" || c.AccountID == 0 || c.Token == "" {
			return nil, errors.New("url, accountId and token are required")
		}
		if c.InboxName == "" {
			c.InboxName = "WhatsApp"
		}
		hook := WebhookURL(publicBase, instanceID, c.WebhookSecret)
		if c.InboxID == 0 {
			id, err := s.createInbox(ctx, &c, hook)
			if err != nil {
				return nil, fmt.Errorf("create inbox: %w", err)
			}
			c.InboxID = id
		} else if err := s.updateInboxWebhook(ctx, &c, hook); err != nil {
			logger.LogWarn("[chatwoot] update inbox webhook: %v", err)
		}
	}
	c.UpdatedAt = time.Now()
	if err := s.db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&c).Error; err != nil {
		return nil, err
	}
	s.cache.Delete(instanceID)
	return s.Get(instanceID), nil
}

// WebhookURL es la URL que Chatwoot llama al haber respuestas de agentes.
func WebhookURL(base, instanceID, secret string) string {
	return strings.TrimRight(base, "/") + "/chatwoot/webhook/" + instanceID + "?secret=" + secret
}

func (s *Service) DeleteInstance(instanceID string) {
	s.db.Where("instance_id = ?", instanceID).Delete(&Link{})
	s.db.Where("instance_id = ?", instanceID).Delete(&Config{})
	s.cache.Delete(instanceID)
}

// ---------- API de Chatwoot ----------

func (s *Service) api(ctx context.Context, c *Config, method, path string, body interface{}, out interface{}) error {
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.URL+"/api/v1/accounts/"+strconv.Itoa(c.AccountID)+path, rd)
	if err != nil {
		return err
	}
	req.Header.Set("api_access_token", c.Token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return s.do(req, out)
}

type apiError struct {
	Status int
	Body   string
}

func (e *apiError) Error() string { return fmt.Sprintf("chatwoot %d: %s", e.Status, e.Body) }

func (s *Service) do(req *http.Request, out interface{}) error {
	resp, err := s.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b := string(data)
		if len(b) > 300 {
			b = b[:300]
		}
		return &apiError{Status: resp.StatusCode, Body: b}
	}
	if out != nil && len(data) > 0 {
		return json.Unmarshal(data, out)
	}
	return nil
}

func (s *Service) createInbox(ctx context.Context, c *Config, hook string) (int, error) {
	var r struct {
		ID int `json:"id"`
	}
	err := s.api(ctx, c, http.MethodPost, "/inboxes", map[string]interface{}{
		"name":    c.InboxName,
		"channel": map[string]interface{}{"type": "api", "webhook_url": hook},
	}, &r)
	return r.ID, err
}

func (s *Service) updateInboxWebhook(ctx context.Context, c *Config, hook string) error {
	return s.api(ctx, c, http.MethodPatch, "/inboxes/"+strconv.Itoa(c.InboxID), map[string]interface{}{
		"channel": map[string]interface{}{"webhook_url": hook},
	}, nil)
}

// TestConnection comprueba URL, cuenta y token.
func (s *Service) TestConnection(ctx context.Context, urlStr string, accountID int, token string) error {
	c := &Config{URL: strings.TrimRight(urlStr, "/"), AccountID: accountID, Token: token}
	return s.api(ctx, c, http.MethodGet, "/inboxes", nil, nil)
}

func (s *Service) lock(key string) *sync.Mutex {
	m, _ := s.locks.LoadOrStore(key, &sync.Mutex{})
	return m.(*sync.Mutex)
}

func phoneOf(jid types.JID) string {
	if jid.Server == types.DefaultUserServer {
		return "+" + jid.User
	}
	return ""
}

// ensureConversation devuelve la conversación del chat, creando contacto y
// conversación si hace falta.
func (s *Service) ensureConversation(ctx context.Context, c *Config, instanceID string, chat types.JID, name string) (*Link, error) {
	var link Link
	err := s.db.First(&link, "instance_id = ? AND jid = ?", instanceID, chat.String()).Error
	if err == nil && link.ConversationID != 0 {
		status, err := s.conversationStatus(ctx, c, link.ConversationID)
		if err == nil && status != "resolved" {
			return &link, nil
		}
		if err == nil && status == "resolved" && c.ReopenConversation {
			target := "open"
			if c.ConversationPending {
				target = "pending"
			}
			_ = s.api(ctx, c, http.MethodPost, "/conversations/"+strconv.Itoa(link.ConversationID)+"/toggle_status", map[string]string{"status": target}, nil)
			return &link, nil
		}
		// resuelta sin reabrir, o borrada: conversación nueva
	}
	if link.ContactID == 0 {
		if err := s.ensureContact(ctx, c, chat, name, &link); err != nil {
			return nil, err
		}
	}
	status := "open"
	if c.ConversationPending {
		status = "pending"
	}
	var conv struct {
		ID int `json:"id"`
	}
	err = s.api(ctx, c, http.MethodPost, "/conversations", map[string]interface{}{
		"source_id": link.SourceID, "inbox_id": c.InboxID, "contact_id": link.ContactID, "status": status,
	}, &conv)
	if err != nil {
		var ae *apiError
		if errors.As(err, &ae) && (ae.Status == 404 || ae.Status == 422) {
			// el contacto pudo borrarse en Chatwoot: se vuelve a crear
			link.ContactID = 0
			if err := s.ensureContact(ctx, c, chat, name, &link); err != nil {
				return nil, err
			}
			err = s.api(ctx, c, http.MethodPost, "/conversations", map[string]interface{}{
				"source_id": link.SourceID, "inbox_id": c.InboxID, "contact_id": link.ContactID, "status": status,
			}, &conv)
		}
		if err != nil {
			return nil, err
		}
	}
	link.InstanceID, link.JID, link.ConversationID = instanceID, chat.String(), conv.ID
	s.db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&link)
	return &link, nil
}

func (s *Service) conversationStatus(ctx context.Context, c *Config, id int) (string, error) {
	var r struct {
		Status string `json:"status"`
	}
	err := s.api(ctx, c, http.MethodGet, "/conversations/"+strconv.Itoa(id), nil, &r)
	return r.Status, err
}

type cwContact struct {
	ID           int    `json:"id"`
	Identifier   string `json:"identifier"`
	PhoneNumber  string `json:"phone_number"`
	ContactInbox []struct {
		SourceID string `json:"source_id"`
		Inbox    struct {
			ID int `json:"id"`
		} `json:"inbox"`
	} `json:"contact_inboxes"`
}

func (s *Service) ensureContact(ctx context.Context, c *Config, chat types.JID, name string, link *Link) error {
	q := chat.String()
	if p := phoneOf(chat); p != "" {
		q = p
	}
	var found struct {
		Payload []cwContact `json:"payload"`
	}
	_ = s.api(ctx, c, http.MethodGet, "/contacts/search?q="+url.QueryEscape(q), nil, &found)
	var ct *cwContact
	for i := range found.Payload {
		if found.Payload[i].Identifier == chat.String() || (phoneOf(chat) != "" && found.Payload[i].PhoneNumber == phoneOf(chat)) {
			ct = &found.Payload[i]
			break
		}
	}
	if ct == nil {
		if name == "" {
			name = chat.User
		}
		body := map[string]interface{}{"inbox_id": c.InboxID, "name": name, "identifier": chat.String()}
		if p := phoneOf(chat); p != "" {
			body["phone_number"] = p
		}
		var created struct {
			Payload struct {
				Contact      cwContact `json:"contact"`
				ContactInbox struct {
					SourceID string `json:"source_id"`
				} `json:"contact_inbox"`
			} `json:"payload"`
		}
		if err := s.api(ctx, c, http.MethodPost, "/contacts", body, &created); err != nil {
			return fmt.Errorf("create contact: %w", err)
		}
		link.ContactID, link.SourceID = created.Payload.Contact.ID, created.Payload.ContactInbox.SourceID
		if link.SourceID != "" {
			return nil
		}
		ct = &created.Payload.Contact
	} else {
		link.ContactID = ct.ID
	}
	for _, ci := range ct.ContactInbox {
		if ci.Inbox.ID == c.InboxID {
			link.SourceID = ci.SourceID
			return nil
		}
	}
	var ci struct {
		SourceID string `json:"source_id"`
	}
	if err := s.api(ctx, c, http.MethodPost, "/contacts/"+strconv.Itoa(link.ContactID)+"/contact_inboxes", map[string]interface{}{"inbox_id": c.InboxID}, &ci); err != nil {
		return fmt.Errorf("contact inbox: %w", err)
	}
	link.SourceID = ci.SourceID
	return nil
}

// postMessage crea un mensaje en la conversación (con adjunto si hay).
func (s *Service) postMessage(ctx context.Context, c *Config, convID int, msgType, waID, text string, file []byte, fileName, mime string) error {
	path := "/conversations/" + strconv.Itoa(convID) + "/messages"
	if len(file) == 0 {
		return s.api(ctx, c, http.MethodPost, path, map[string]interface{}{
			"content": text, "message_type": msgType, "private": false, "source_id": sourcePrefix + waID,
		}, nil)
	}
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("content", text)
	_ = w.WriteField("message_type", msgType)
	_ = w.WriteField("private", "false")
	_ = w.WriteField("source_id", sourcePrefix+waID)
	h := make(map[string][]string)
	h["Content-Disposition"] = []string{fmt.Sprintf(`form-data; name="attachments[]"; filename="%s"`, strings.ReplaceAll(fileName, `"`, ""))}
	if mime == "" {
		mime = "application/octet-stream"
	}
	h["Content-Type"] = []string{mime}
	fw, _ := w.CreatePart(h)
	_, _ = fw.Write(file)
	_ = w.Close()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URL+"/api/v1/accounts/"+strconv.Itoa(c.AccountID)+path, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("api_access_token", c.Token)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return s.do(req, nil)
}

// ---------- WhatsApp → Chatwoot ----------

var (
	defMu  sync.RWMutex
	defSvc *Service
)

func SetDefault(s *Service) {
	defMu.Lock()
	defSvc = s
	defMu.Unlock()
}

func current() *Service {
	defMu.RLock()
	defer defMu.RUnlock()
	return defSvc
}

// Observe recibe los eventos de whatsmeow de una línea.
func Observe(instanceID string, evt interface{}) {
	s := current()
	if s == nil {
		return
	}
	m, ok := evt.(*events.Message)
	if !ok {
		return
	}
	c := s.Get(instanceID)
	if c == nil || !c.Enabled {
		return
	}
	cp := *m
	go s.forward(c, instanceID, cp.Info, cp.Message, false)
}

// ObserveSent recibe lo enviado por la API (bots, integraciones) para mostrarlo
// en Chatwoot como saliente; lo que vino de Chatwoot no vuelve.
func ObserveSent(instanceID string, info types.MessageInfo, msg *waE2E.Message) {
	s := current()
	if s == nil {
		return
	}
	c := s.Get(instanceID)
	if c == nil || !c.Enabled {
		return
	}
	if n, ok := s.fromCW.Load(instanceID + "|" + info.Chat.ToNonAD().String()); ok && n.(int) > 0 {
		return
	}
	go s.forward(c, instanceID, info, msg, true)
}

func downloadable(m *waE2E.Message) (whatsmeow.DownloadableMessage, string, string) {
	switch {
	case m.GetImageMessage() != nil:
		return m.GetImageMessage(), m.GetImageMessage().GetMimetype(), "image.jpg"
	case m.GetVideoMessage() != nil:
		return m.GetVideoMessage(), m.GetVideoMessage().GetMimetype(), "video.mp4"
	case m.GetAudioMessage() != nil:
		return m.GetAudioMessage(), m.GetAudioMessage().GetMimetype(), "audio.ogg"
	case m.GetDocumentMessage() != nil:
		name := m.GetDocumentMessage().GetFileName()
		if name == "" {
			name = "document"
		}
		return m.GetDocumentMessage(), m.GetDocumentMessage().GetMimetype(), name
	case m.GetStickerMessage() != nil:
		return m.GetStickerMessage(), m.GetStickerMessage().GetMimetype(), "sticker.webp"
	}
	return nil, "", ""
}

func (s *Service) forward(c *Config, instanceID string, info types.MessageInfo, msg *waE2E.Message, sentByAPI bool) {
	chat := chatstore.ChatJID(info.MessageSource)
	if chatstore.Skip(chat) || chat.Server == types.NewsletterServer || chat.Server == types.BroadcastServer {
		return
	}
	isGroup := info.IsGroup || chat.Server == types.GroupServer
	if isGroup && c.IgnoreGroups {
		return
	}
	content, ok := chatstore.Extract(msg)
	if !ok {
		return
	}
	mu := s.lock(instanceID + "|" + chat.String())
	mu.Lock()
	defer mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	name := info.PushName
	if info.IsFromMe || isGroup {
		name = ""
	}
	link, err := s.ensureConversation(ctx, c, instanceID, chat, name)
	if err != nil {
		logger.LogError("[chatwoot] %s: %v", instanceID, err)
		return
	}
	text := content.Text
	if isGroup && !info.IsFromMe {
		who := info.PushName
		if who == "" {
			who = info.Sender.User
		}
		text = "*" + who + "*: " + text
	}
	msgType := "incoming"
	if info.IsFromMe || sentByAPI {
		msgType = "outgoing"
	}

	var file []byte
	var mime, fileName string
	if d, mt, fn := downloadable(msg); d != nil {
		if cli := s.client(instanceID); cli != nil {
			if data, err := cli.Download(ctx, d); err == nil {
				file, mime, fileName = data, mt, fn
			} else {
				logger.LogWarn("[chatwoot] download media: %v", err)
			}
		}
	}
	if text == "" && len(file) == 0 {
		return
	}
	if err := s.postMessage(ctx, c, link.ConversationID, msgType, info.ID, text, file, fileName, mime); err != nil {
		var ae *apiError
		if errors.As(err, &ae) && ae.Status == 404 {
			// conversación borrada en Chatwoot: se crea otra y se reintenta
			s.db.Where("instance_id = ? AND jid = ?", instanceID, chat.String()).Delete(&Link{})
			if link, err = s.ensureConversation(ctx, c, instanceID, chat, name); err == nil {
				err = s.postMessage(ctx, c, link.ConversationID, msgType, info.ID, text, file, fileName, mime)
			}
		}
		if err != nil {
			logger.LogError("[chatwoot] post message: %v", err)
		}
	}
}

// ---------- Chatwoot → WhatsApp ----------

// WebhookEvent es lo que Chatwoot envía al webhook de la bandeja.
type WebhookEvent struct {
	Event       string `json:"event"`
	ID          int    `json:"id"`
	Content     string `json:"content"`
	MessageType string `json:"message_type"`
	Private     bool   `json:"private"`
	SourceID    string `json:"source_id"`
	Sender      struct {
		Name          string `json:"name"`
		AvailableName string `json:"available_name"`
		Type          string `json:"type"`
	} `json:"sender"`
	Conversation struct {
		ID   int `json:"id"`
		Meta struct {
			Sender struct {
				Identifier  string `json:"identifier"`
				PhoneNumber string `json:"phone_number"`
			} `json:"sender"`
		} `json:"meta"`
	} `json:"conversation"`
	Attachments []struct {
		DataURL  string `json:"data_url"`
		FileType string `json:"file_type"`
	} `json:"attachments"`
}

// ErrUnauthorized indica un secreto de webhook no válido.
var ErrUnauthorized = errors.New("unauthorized")

// HandleWebhook procesa un evento de Chatwoot para la línea.
func (s *Service) HandleWebhook(instanceID, secret string, ev WebhookEvent) error {
	c := s.Get(instanceID)
	if c == nil || c.WebhookSecret == "" || secret != c.WebhookSecret {
		return ErrUnauthorized
	}
	if !c.Enabled || ev.Event != "message_created" || ev.MessageType != "outgoing" || ev.Private {
		return nil
	}
	if strings.HasPrefix(ev.SourceID, sourcePrefix) {
		return nil // lo creó este servidor: no se reenvía
	}
	var link Link
	jid := ""
	if s.db.First(&link, "instance_id = ? AND conversation_id = ?", instanceID, ev.Conversation.ID).Error == nil {
		jid = link.JID
	} else if id := ev.Conversation.Meta.Sender.Identifier; strings.Contains(id, "@") {
		jid = id
	} else if p := strings.TrimPrefix(ev.Conversation.Meta.Sender.PhoneNumber, "+"); p != "" {
		jid = p + "@" + types.DefaultUserServer
	}
	if jid == "" {
		return fmt.Errorf("conversation %d has no WhatsApp chat", ev.Conversation.ID)
	}

	text := ev.Content
	if c.SignMessages && text != "" {
		name := ev.Sender.AvailableName
		if name == "" {
			name = ev.Sender.Name
		}
		if name != "" {
			text = "*" + name + "*:\n" + text
		}
	}

	key := instanceID + "|" + jid
	s.markFromCW(key, 1)
	defer s.markFromCW(key, -1)

	for i, a := range ev.Attachments {
		caption := ""
		if i == 0 {
			caption = text
		}
		mt := map[string]string{"image": "image", "video": "video", "audio": "audio"}[a.FileType]
		if mt == "" {
			mt = "document"
		}
		if err := s.send.SendMedia(instanceID, jid, a.DataURL, mt, caption, ""); err != nil {
			return err
		}
	}
	if len(ev.Attachments) == 0 && strings.TrimSpace(text) != "" {
		return s.send.SendText(instanceID, jid, text)
	}
	return nil
}

func (s *Service) markFromCW(key string, d int) {
	for {
		v, _ := s.fromCW.LoadOrStore(key, 0)
		n := v.(int) + d
		if s.fromCW.CompareAndSwap(key, v, n) {
			return
		}
	}
}
