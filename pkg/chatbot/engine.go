package chatbot

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/evolution-foundation/evolution-go/pkg/chatstore"
	"github.com/gomessguii/logger"
	"github.com/google/uuid"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"gorm.io/gorm"
)

// Sender envía mensajes por una línea (implementado con el servicio de envío,
// para que las respuestas generen webhooks y queden en el historial).
type Sender interface {
	SendText(instanceID, jid, text string) error
	SendMedia(instanceID, jid, url, mediaType, caption, fileName string) error
}

// ClientGetter devuelve el cliente de whatsmeow de una línea (o nil).
type ClientGetter func(instanceID string) *whatsmeow.Client

// Provider genera la respuesta de un tipo de bot.
type Provider interface {
	Reply(ctx context.Context, b *Bot, s *Session, in Input) (*Output, error)
}

type buffer struct {
	texts []string
	in    Input
	bot   *Bot
	timer *time.Timer
}

// Engine procesa los mensajes entrantes de todas las líneas.
type Engine struct {
	db        *gorm.DB
	send      Sender
	client    ClientGetter
	providers map[string]Provider

	cacheMu sync.RWMutex
	cache   map[string][]Bot // instanceID → bots

	locks sync.Map // instanceID|jid → *sync.Mutex

	bufMu   sync.Mutex
	buffers map[string]*buffer
}

func NewEngine(db *gorm.DB, send Sender, client ClientGetter) *Engine {
	e := &Engine{db: db, send: send, client: client, cache: map[string][]Bot{}, buffers: map[string]*buffer{}}
	e.providers = defaultProviders(e)
	return e
}

func (e *Engine) Migrate() error { return e.db.AutoMigrate(&Bot{}, &Session{}) }

// Invalidate olvida la caché de bots de una línea (tras crear, editar o borrar).
func (e *Engine) Invalidate(instanceID string) {
	e.cacheMu.Lock()
	delete(e.cache, instanceID)
	e.cacheMu.Unlock()
}

func (e *Engine) bots(instanceID string) []Bot {
	e.cacheMu.RLock()
	list, ok := e.cache[instanceID]
	e.cacheMu.RUnlock()
	if ok {
		return list
	}
	var out []Bot
	e.db.Where("instance_id = ? AND enabled", instanceID).Find(&out)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Priority != out[j].Priority {
			return out[i].Priority > out[j].Priority
		}
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	e.cacheMu.Lock()
	e.cache[instanceID] = out
	e.cacheMu.Unlock()
	return out
}

func (e *Engine) botByID(instanceID, id string) *Bot {
	for _, b := range e.bots(instanceID) {
		if b.ID == id {
			bb := b
			return &bb
		}
	}
	return nil
}

func (e *Engine) lock(key string) *sync.Mutex {
	m, _ := e.locks.LoadOrStore(key, &sync.Mutex{})
	return m.(*sync.Mutex)
}

var (
	defMu  sync.RWMutex
	defEng *Engine
)

func SetDefault(e *Engine) {
	defMu.Lock()
	defEng = e
	defMu.Unlock()
}

// Observe recibe los eventos de whatsmeow de una línea.
func Observe(instanceID string, evt interface{}) {
	defMu.RLock()
	e := defEng
	defMu.RUnlock()
	if e == nil {
		return
	}
	m, ok := evt.(*events.Message)
	if !ok {
		return
	}
	cp := *m // upstream modifica evt.Info después de esta llamada
	go e.handle(instanceID, &cp)
}

func (e *Engine) openSession(instanceID, jid string) *Session {
	var s Session
	err := e.db.Where("instance_id = ? AND remote_jid = ? AND status <> ?", instanceID, jid, SessionClosed).
		Order("last_activity DESC").First(&s).Error
	if err != nil {
		return nil
	}
	return &s
}

func (e *Engine) setStatus(s *Session, status string) {
	s.Status = status
	e.db.Model(&Session{}).Where("id = ?", s.ID).Updates(map[string]interface{}{"status": status, "last_activity": time.Now()})
}

func (e *Engine) handle(instanceID string, m *events.Message) {
	info := m.Info
	chat := chatstore.ChatJID(info.MessageSource)
	if chatstore.Skip(chat) || chat.Server == types.NewsletterServer || chat.Server == types.BroadcastServer {
		return
	}
	content, ok := chatstore.Extract(m.Message)
	if !ok {
		return
	}
	jid := chat.String()
	isGroup := info.IsGroup || chat.Server == types.GroupServer
	key := instanceID + "|" + jid
	mu := e.lock(key)
	mu.Lock()
	defer mu.Unlock()

	now := time.Now()
	sess := e.openSession(instanceID, jid)
	var bot *Bot
	if sess != nil {
		bot = e.botByID(instanceID, sess.BotID)
		if bot == nil { // bot borrado o desactivado
			e.setStatus(sess, SessionClosed)
			sess = nil
		}
	}

	// Mensajes enviados desde el teléfono de la línea: un humano toma el chat.
	if info.IsFromMe {
		if sess != nil && bot != nil && !bot.ListenFromMe {
			if bot.StopOnFromMe && sess.Status == SessionOpen {
				e.setStatus(sess, SessionPaused)
			}
			return
		}
		if sess == nil && !e.anyListensFromMe(instanceID) {
			return
		}
	}

	if sess != nil && Expired(bot, sess, now) {
		e.setStatus(sess, SessionClosed)
		sess, bot = nil, nil
	}
	if sess != nil && sess.Status == SessionPaused {
		return
	}
	if sess != nil && IsFinish(bot, content.Text) {
		e.setStatus(sess, SessionClosed)
		return
	}
	if sess != nil && isGroup && !bot.AllowGroups {
		return
	}

	if sess == nil {
		bot = e.pick(instanceID, jid, content.Text, isGroup, info.IsFromMe)
		if bot == nil {
			return
		}
	}

	in := Input{InstanceID: instanceID, RemoteJID: jid, PushName: info.PushName, MessageID: info.ID, Type: content.Type, Text: content.Text, FromMe: info.IsFromMe, IsGroup: isGroup}
	if content.Type == chatstore.TypeAudio && needsAudio(bot) {
		if cli := e.client(instanceID); cli != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			if data, err := cli.Download(ctx, m.Message.GetAudioMessage()); err == nil {
				in.Audio, in.AudioMime = data, m.Message.GetAudioMessage().GetMimetype()
			}
			cancel()
		}
	}

	if bot.DebounceMs > 0 {
		e.debounce(key, bot, in)
		return
	}
	e.run(bot, in)
}

func (e *Engine) anyListensFromMe(instanceID string) bool {
	for _, b := range e.bots(instanceID) {
		if b.ListenFromMe {
			return true
		}
	}
	return false
}

func (e *Engine) pick(instanceID, jid, text string, isGroup, fromMe bool) *Bot {
	for _, b := range e.bots(instanceID) {
		if isGroup && !b.AllowGroups {
			continue
		}
		if fromMe && !b.ListenFromMe {
			continue
		}
		if Ignored(&b, jid) || !Matches(&b, text) {
			continue
		}
		bb := b
		return &bb
	}
	return nil
}

// debounce junta los mensajes seguidos de un chat y responde una sola vez.
func (e *Engine) debounce(key string, bot *Bot, in Input) {
	e.bufMu.Lock()
	defer e.bufMu.Unlock()
	b := e.buffers[key]
	if b == nil {
		b = &buffer{bot: bot}
		e.buffers[key] = b
	}
	if in.Text != "" {
		b.texts = append(b.texts, in.Text)
	}
	b.in = in
	if b.timer != nil {
		b.timer.Stop()
	}
	b.timer = time.AfterFunc(time.Duration(bot.DebounceMs)*time.Millisecond, func() {
		e.bufMu.Lock()
		cur := e.buffers[key]
		delete(e.buffers, key)
		e.bufMu.Unlock()
		if cur == nil {
			return
		}
		joined := cur.in
		joined.Text = strings.Join(cur.texts, "\n")
		mu := e.lock(key)
		mu.Lock()
		defer mu.Unlock()
		e.run(cur.bot, joined)
	})
}

// run llama al proveedor y envía la respuesta. Debe llamarse con el candado del chat.
func (e *Engine) run(bot *Bot, in Input) {
	p := e.providers[bot.Type]
	if p == nil {
		logger.LogError("[chatbot] unknown bot type %q", bot.Type)
		return
	}
	sess := e.openSession(in.InstanceID, in.RemoteJID)
	if sess == nil || sess.BotID != bot.ID {
		sess = &Session{ID: uuid.NewString(), InstanceID: in.InstanceID, RemoteJID: in.RemoteJID, BotID: bot.ID, Status: SessionOpen,
			PushName: in.PushName, LastActivity: time.Now(), CreatedAt: time.Now()}
		if err := e.db.Create(sess).Error; err != nil {
			logger.LogError("[chatbot] create session: %v", err)
			return
		}
	}

	if strings.TrimSpace(in.Text) == "" && len(in.Audio) == 0 {
		if bot.UnknownMessage != "" {
			e.deliver(bot, in, []Reply{{Text: bot.UnknownMessage}})
		}
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	out, err := p.Reply(ctx, bot, sess, in)
	if err != nil {
		logger.LogError("[chatbot] %s bot %s: %v", bot.Type, bot.ID, err)
		e.db.Model(&Session{}).Where("id = ?", sess.ID).Update("last_activity", time.Now())
		return
	}

	e.deliver(bot, in, out.Replies)

	upd := map[string]interface{}{"last_activity": time.Now()}
	if out.ExternalID != "" {
		upd["external_id"] = out.ExternalID
	}
	if out.Context != nil {
		upd["context"] = out.Context
	}
	if out.End {
		upd["status"] = SessionClosed
	}
	e.db.Model(&Session{}).Where("id = ?", sess.ID).Updates(upd)
}

// deliver envía las respuestas con «escribiendo…» y la espera configurada.
func (e *Engine) deliver(bot *Bot, in Input, replies []Reply) {
	var parts []Reply
	for _, r := range replies {
		if r.MediaURL == "" && bot.SplitMessages {
			for _, t := range Split(r.Text) {
				parts = append(parts, Reply{Text: t})
			}
			continue
		}
		if r.MediaURL != "" || strings.TrimSpace(r.Text) != "" {
			parts = append(parts, r)
		}
	}
	jid, _ := types.ParseJID(in.RemoteJID)
	for _, r := range parts {
		if d := TypingDelay(bot); d > 0 {
			if cli := e.client(in.InstanceID); cli != nil {
				_ = cli.SendChatPresence(context.Background(), jid, types.ChatPresenceComposing, types.ChatPresenceMediaText)
			}
			time.Sleep(d)
		}
		var err error
		if r.MediaURL != "" {
			err = e.send.SendMedia(in.InstanceID, in.RemoteJID, r.MediaURL, r.MediaType, r.Text, "")
		} else {
			err = e.send.SendText(in.InstanceID, in.RemoteJID, r.Text)
		}
		if err != nil {
			logger.LogError("[chatbot] send reply to %s: %v", in.RemoteJID, err)
			return
		}
	}
}

func needsAudio(b *Bot) bool {
	return b.Type == TypeOpenAI && openAIConfigOf(b).Transcribe
}

// DeleteInstance borra los bots y sesiones de una línea.
func (e *Engine) DeleteInstance(instanceID string) {
	e.db.Where("instance_id = ?", instanceID).Delete(&Session{})
	e.db.Where("instance_id = ?", instanceID).Delete(&Bot{})
	e.Invalidate(instanceID)
}
