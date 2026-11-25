package chatstore

import (
	"context"
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gomessguii/logger"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Config se lee del entorno:
//
//	CHAT_STORE_ENABLED         (true)  guardar chats, contactos y mensajes
//	CHAT_STORE_HISTORY         (true)  importar el historial que WhatsApp envía al vincular
//	CHAT_STORE_RETENTION_DAYS  (90)    días que se guardan los mensajes; 0 = sin límite
//	CHAT_STORE_RAW             (true)  guardar también el mensaje original en JSON
type Config struct {
	Enabled       bool
	History       bool
	RetentionDays int
	Raw           bool
}

func envBool(key string, def bool) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if v == "" {
		return def
	}
	return v == "true" || v == "1" || v == "yes"
}

func ConfigFromEnv() Config {
	days := 90
	if v := os.Getenv("CHAT_STORE_RETENTION_DAYS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			days = n
		}
	}
	return Config{
		Enabled:       envBool("CHAT_STORE_ENABLED", true),
		History:       envBool("CHAT_STORE_HISTORY", true),
		RetentionDays: days,
		Raw:           envBool("CHAT_STORE_RAW", true),
	}
}

const maxRawBytes = 256 * 1024

// Store escribe en una cola de un solo trabajador: los eventos de una misma
// línea se aplican en orden y la base no recibe ráfagas concurrentes.
type Store struct {
	db   *gorm.DB
	cfg  Config
	jobs chan func()
}

func New(db *gorm.DB, cfg Config) *Store {
	return &Store{db: db, cfg: cfg, jobs: make(chan func(), 20000)}
}

func (s *Store) Config() Config { return s.cfg }

func (s *Store) Migrate() error {
	return s.db.AutoMigrate(&Chat{}, &Contact{}, &Message{})
}

// Start lanza el trabajador y la limpieza diaria por antigüedad.
func (s *Store) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case job := <-s.jobs:
				func() {
					defer func() {
						if r := recover(); r != nil {
							logger.LogError("[chatstore] job panic: %v", r)
						}
					}()
					job()
				}()
			}
		}
	}()
	go func() {
		s.prune()
		t := time.NewTicker(24 * time.Hour)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				s.prune()
			}
		}
	}()
}

func (s *Store) prune() {
	if s.cfg.RetentionDays <= 0 {
		return
	}
	cut := time.Now().AddDate(0, 0, -s.cfg.RetentionDays)
	if err := s.db.Where("timestamp < ?", cut).Delete(&Message{}).Error; err != nil {
		logger.LogError("[chatstore] prune: %v", err)
	}
}

func (s *Store) enqueue(job func()) {
	select {
	case s.jobs <- job:
	default:
		logger.LogWarn("[chatstore] queue full, event dropped")
	}
}

var (
	defMu  sync.RWMutex
	defSto *Store
)

// SetDefault fija el almacén que usan Observe y ObserveSent.
func SetDefault(s *Store) {
	defMu.Lock()
	defSto = s
	defMu.Unlock()
}

func current() *Store {
	defMu.RLock()
	defer defMu.RUnlock()
	if defSto == nil || !defSto.cfg.Enabled {
		return nil
	}
	return defSto
}

// Observe procesa un evento de whatsmeow de la instancia.
func Observe(instanceID string, cli *whatsmeow.Client, evt interface{}) {
	s := current()
	if s == nil {
		return
	}
	switch e := evt.(type) {
	case *events.Message:
		s.enqueue(func() { s.saveMessage(instanceID, e, false) })
	case *events.Receipt:
		s.enqueue(func() { s.receipt(instanceID, e) })
	case *events.HistorySync:
		if s.cfg.History && cli != nil {
			s.enqueue(func() { s.history(instanceID, cli, e) })
		}
	case *events.PushName:
		s.enqueue(func() { s.contact(instanceID, preferPN(e.JID, e.JIDAlt), "", e.NewPushName, "") })
	case *events.BusinessName:
		s.enqueue(func() { s.contact(instanceID, e.JID, "", "", e.NewBusinessName) })
	case *events.Contact:
		if e.Action != nil {
			s.enqueue(func() { s.contact(instanceID, e.JID, e.Action.GetFullName(), "", "") })
		}
	case *events.GroupInfo:
		if e.Name != nil {
			s.enqueue(func() { s.chatName(instanceID, e.JID, e.Name.Name, true) })
		}
	case *events.JoinedGroup:
		s.enqueue(func() { s.chatName(instanceID, e.JID, e.GroupInfo.Name, true) })
	case *events.Archive:
		if e.Action != nil {
			s.enqueue(func() { s.chatFlag(instanceID, e.JID, map[string]interface{}{"archived": e.Action.GetArchived()}) })
		}
	case *events.Pin:
		if e.Action != nil {
			s.enqueue(func() { s.chatFlag(instanceID, e.JID, map[string]interface{}{"pinned": e.Action.GetPinned()}) })
		}
	case *events.Mute:
		if e.Action != nil {
			var until *time.Time
			if e.Action.GetMuted() {
				t := time.UnixMilli(e.Action.GetMuteEndTimestamp())
				if e.Action.GetMuteEndTimestamp() <= 0 {
					t = time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)
				}
				until = &t
			}
			s.enqueue(func() { s.chatFlag(instanceID, e.JID, map[string]interface{}{"muted_until": until}) })
		}
	case *events.MarkChatAsRead:
		if e.Action != nil && e.Action.GetRead() {
			s.enqueue(func() { s.chatFlag(instanceID, e.JID, map[string]interface{}{"unread_count": 0}) })
		}
	case *events.DeleteForMe:
		s.enqueue(func() {
			s.db.Where("instance_id = ? AND message_id = ?", instanceID, e.MessageID).Delete(&Message{})
		})
	}
}

// ObserveSent guarda un mensaje enviado por la API (whatsmeow no emite evento para él).
func ObserveSent(instanceID string, info types.MessageInfo, msg *waE2E.Message) {
	s := current()
	if s == nil {
		return
	}
	evt := &events.Message{Info: info, Message: msg}
	s.enqueue(func() { s.saveMessage(instanceID, evt, false) })
}

func preferPN(jid, alt types.JID) types.JID {
	if jid.Server == types.HiddenUserServer && alt.Server == types.DefaultUserServer {
		return alt
	}
	return jid
}

func (s *Store) raw(m *waE2E.Message) json.RawMessage {
	if !s.cfg.Raw || m == nil {
		return nil
	}
	b, err := json.Marshal(m)
	if err != nil || len(b) > maxRawBytes {
		return nil
	}
	return b
}

// buildMessage convierte un evento en fila; ok=false si no se guarda.
func (s *Store) buildMessage(instanceID string, e *events.Message) (Message, types.JID, Content, bool) {
	chat := ChatJID(e.Info.MessageSource)
	if Skip(chat) || e.Info.ID == "" {
		return Message{}, chat, Content{}, false
	}
	c, ok := Extract(e.Message)
	if !ok {
		return Message{}, chat, c, false
	}
	status := StatusReceived
	if e.Info.IsFromMe {
		status = StatusSent
	}
	ts := e.Info.Timestamp
	if ts.IsZero() {
		ts = time.Now()
	}
	return Message{
		InstanceID: instanceID,
		MessageID:  e.Info.ID,
		ChatJID:    chat.String(),
		SenderJID:  preferPN(e.Info.Sender.ToNonAD(), e.Info.SenderAlt.ToNonAD()).String(),
		SenderName: e.Info.PushName,
		FromMe:     e.Info.IsFromMe,
		Type:       c.Type,
		Text:       c.Text,
		MediaMime:  c.Mime,
		FileName:   c.FileName,
		QuotedID:   c.QuotedID,
		Status:     status,
		StatusRank: statusRank[status],
		Timestamp:  ts,
		Raw:        s.raw(e.Message),
	}, chat, c, true
}

func (s *Store) saveMessage(instanceID string, e *events.Message, fromHistory bool) {
	// Ediciones y borrados para todos llegan como mensajes de protocolo.
	if p := e.Message.GetProtocolMessage(); p != nil {
		id := p.GetKey().GetID()
		switch p.GetType() {
		case waE2E.ProtocolMessage_REVOKE:
			s.db.Model(&Message{}).Where("instance_id = ? AND message_id = ?", instanceID, id).
				Updates(map[string]interface{}{"deleted": true, "text": "", "raw": nil})
		case waE2E.ProtocolMessage_MESSAGE_EDIT:
			if c, ok := Extract(p.GetEditedMessage()); ok {
				s.db.Model(&Message{}).Where("instance_id = ? AND message_id = ?", instanceID, id).
					Updates(map[string]interface{}{"edited": true, "text": c.Text})
			}
		}
		return
	}

	row, chat, c, ok := s.buildMessage(instanceID, e)
	if !ok {
		return
	}
	res := s.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&row)
	if res.Error != nil {
		logger.LogError("[chatstore] save message: %v", res.Error)
		return
	}
	if res.RowsAffected == 0 {
		return // ya estaba guardado
	}
	unread := 0
	if !e.Info.IsFromMe && !fromHistory {
		unread = 1
	}
	s.touchChat(instanceID, chat, e.Info.IsGroup, row, c, unread)
	if !e.Info.IsFromMe && e.Info.PushName != "" && !e.Info.IsGroup {
		s.contact(instanceID, preferPN(e.Info.Sender.ToNonAD(), e.Info.SenderAlt.ToNonAD()), "", e.Info.PushName, "")
	}
}

// touchChat crea el chat o actualiza su último mensaje si este es más nuevo.
func (s *Store) touchChat(instanceID string, chat types.JID, isGroup bool, m Message, c Content, unread int) {
	ts := m.Timestamp
	row := Chat{
		InstanceID: instanceID, JID: chat.String(), IsGroup: isGroup || chat.Server == types.GroupServer,
		LastMessageAt: &ts, LastMessageID: m.MessageID, LastPreview: Preview(c), LastType: c.Type, LastFromMe: m.FromMe,
		UnreadCount: unread, UpdatedAt: time.Now(),
	}
	err := s.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "instance_id"}, {Name: "jid"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"last_message_at": gorm.Expr("CASE WHEN chat_store_chats.last_message_at IS NULL OR chat_store_chats.last_message_at <= EXCLUDED.last_message_at THEN EXCLUDED.last_message_at ELSE chat_store_chats.last_message_at END"),
			"last_message_id": gorm.Expr("CASE WHEN chat_store_chats.last_message_at IS NULL OR chat_store_chats.last_message_at <= EXCLUDED.last_message_at THEN EXCLUDED.last_message_id ELSE chat_store_chats.last_message_id END"),
			"last_preview":    gorm.Expr("CASE WHEN chat_store_chats.last_message_at IS NULL OR chat_store_chats.last_message_at <= EXCLUDED.last_message_at THEN EXCLUDED.last_preview ELSE chat_store_chats.last_preview END"),
			"last_type":       gorm.Expr("CASE WHEN chat_store_chats.last_message_at IS NULL OR chat_store_chats.last_message_at <= EXCLUDED.last_message_at THEN EXCLUDED.last_type ELSE chat_store_chats.last_type END"),
			"last_from_me":    gorm.Expr("CASE WHEN chat_store_chats.last_message_at IS NULL OR chat_store_chats.last_message_at <= EXCLUDED.last_message_at THEN EXCLUDED.last_from_me ELSE chat_store_chats.last_from_me END"),
			"unread_count":    gorm.Expr("CASE WHEN EXCLUDED.last_from_me THEN 0 ELSE chat_store_chats.unread_count + EXCLUDED.unread_count END"),
			"updated_at":      gorm.Expr("EXCLUDED.updated_at"),
		}),
	}).Create(&row).Error
	if err != nil {
		logger.LogError("[chatstore] touch chat: %v", err)
	}
}

func (s *Store) receipt(instanceID string, e *events.Receipt) {
	switch e.Type {
	case types.ReceiptTypeReadSelf:
		// Se leyó el chat desde el teléfono.
		s.chatFlag(instanceID, ChatJID(e.MessageSource), map[string]interface{}{"unread_count": 0})
		return
	case types.ReceiptTypeDelivered, types.ReceiptTypeRead, types.ReceiptTypePlayed:
	default:
		return
	}
	status := StatusDelivered
	if e.Type == types.ReceiptTypeRead {
		status = StatusRead
	} else if e.Type == types.ReceiptTypePlayed {
		status = StatusPlayed
	}
	if len(e.MessageIDs) == 0 {
		return
	}
	s.db.Model(&Message{}).
		Where("instance_id = ? AND message_id IN ? AND from_me AND status_rank < ?", instanceID, e.MessageIDs, statusRank[status]).
		Updates(map[string]interface{}{"status": status, "status_rank": statusRank[status]})
}

func (s *Store) contact(instanceID string, jid types.JID, full, push, business string) {
	if jid.IsEmpty() || jid.Server == types.GroupServer {
		return
	}
	row := Contact{InstanceID: instanceID, JID: jid.ToNonAD().String(), FullName: full, PushName: push, BusinessName: business, UpdatedAt: time.Now()}
	set := map[string]interface{}{"updated_at": gorm.Expr("EXCLUDED.updated_at")}
	if full != "" {
		set["full_name"] = gorm.Expr("EXCLUDED.full_name")
	}
	if push != "" {
		set["push_name"] = gorm.Expr("EXCLUDED.push_name")
	}
	if business != "" {
		set["business_name"] = gorm.Expr("EXCLUDED.business_name")
	}
	s.db.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "instance_id"}, {Name: "jid"}}, DoUpdates: clause.Assignments(set)}).Create(&row)
}

func (s *Store) chatName(instanceID string, jid types.JID, name string, isGroup bool) {
	if name == "" {
		return
	}
	row := Chat{InstanceID: instanceID, JID: jid.ToNonAD().String(), Name: name, IsGroup: isGroup, UpdatedAt: time.Now()}
	s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "instance_id"}, {Name: "jid"}},
		DoUpdates: clause.Assignments(map[string]interface{}{"name": name, "is_group": isGroup}),
	}).Create(&row)
}

func (s *Store) chatFlag(instanceID string, jid types.JID, set map[string]interface{}) {
	s.db.Model(&Chat{}).Where("instance_id = ? AND jid = ?", instanceID, jid.ToNonAD().String()).Updates(set)
}

// history importa la sincronización inicial: chats, nombres y mensajes.
func (s *Store) history(instanceID string, cli *whatsmeow.Client, e *events.HistorySync) {
	if e.Data == nil {
		return
	}
	for _, pn := range e.Data.GetPushnames() {
		if jid, err := types.ParseJID(pn.GetID()); err == nil {
			s.contact(instanceID, jid, "", pn.GetPushname(), "")
		}
	}
	for _, conv := range e.Data.GetConversations() {
		jid, err := types.ParseJID(conv.GetID())
		if err != nil || Skip(jid) {
			continue
		}
		var rows []Message
		var newest *Message
		var newestContent Content
		chatKey := jid.ToNonAD()
		for _, hm := range conv.GetMessages() {
			evt, err := cli.ParseWebMessage(jid, hm.GetMessage())
			if err != nil || evt.Message.GetProtocolMessage() != nil {
				continue
			}
			row, key, c, ok := s.buildMessage(instanceID, evt)
			if !ok {
				continue
			}
			chatKey = key
			if evt.Info.IsFromMe {
				row.Status, row.StatusRank = StatusRead, statusRank[StatusRead]
			}
			rows = append(rows, row)
			if newest == nil || row.Timestamp.After(newest.Timestamp) {
				r := row
				newest, newestContent = &r, c
			}
		}
		if len(rows) > 0 {
			if err := s.db.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(rows, 200).Error; err != nil {
				logger.LogError("[chatstore] history batch: %v", err)
			}
			s.touchChat(instanceID, chatKey, jid.Server == types.GroupServer, *newest, newestContent, 0)
		}
		name := conv.GetName()
		if name == "" {
			name = conv.GetDisplayName()
		}
		if name != "" && jid.Server == types.GroupServer {
			s.chatName(instanceID, jid, name, true)
		}
		set := map[string]interface{}{
			"archived":     conv.GetArchived(),
			"pinned":       conv.GetPinned() > 0,
			"unread_count": int(conv.GetUnreadCount()),
		}
		if end := conv.GetMuteEndTime(); end > 0 {
			t := time.Unix(int64(end), 0)
			set["muted_until"] = &t
		}
		if len(rows) > 0 {
			s.chatFlag(instanceID, chatKey, set)
		}
	}
}

// DeleteInstance borra todo lo guardado de una instancia.
func (s *Store) DeleteInstance(instanceID string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		for _, m := range []interface{}{&Message{}, &Chat{}, &Contact{}} {
			if err := tx.Where("instance_id = ?", instanceID).Delete(m).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
