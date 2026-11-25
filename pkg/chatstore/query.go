package chatstore

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

// ChatView es un chat con el nombre resuelto a partir de los contactos.
type ChatView struct {
	Chat
	DisplayName string `json:"displayName"`
}

// ListChats devuelve los chats ordenados: fijados primero, luego por último mensaje.
func (s *Store) chatQuery(instanceID string) *gorm.DB {
	return s.db.Table("chat_store_chats c").
		Select(`c.*, COALESCE(NULLIF(c.name, ''), NULLIF(ct.full_name, ''), NULLIF(ct.push_name, ''), NULLIF(ct.business_name, ''), '') AS display_name`).
		Joins("LEFT JOIN chat_store_contacts ct ON ct.instance_id = c.instance_id AND ct.jid = c.jid").
		Where("c.instance_id = ?", instanceID)
}

// GetChat devuelve un chat concreto, archivado o no.
func (s *Store) GetChat(instanceID, jid string) ([]ChatView, error) {
	var out []ChatView
	err := s.chatQuery(instanceID).Where("c.jid = ?", jid).Limit(1).Scan(&out).Error
	return out, err
}

func (s *Store) ListChats(instanceID, q string, archived bool, limit int, before *time.Time) ([]ChatView, error) {
	var out []ChatView
	tx := s.chatQuery(instanceID).Where("c.archived = ? AND c.last_message_at IS NOT NULL", archived)
	if q = strings.TrimSpace(q); q != "" {
		like := "%" + strings.ToLower(q) + "%"
		tx = tx.Where("LOWER(c.jid) LIKE ? OR LOWER(c.name) LIKE ? OR LOWER(ct.full_name) LIKE ? OR LOWER(ct.push_name) LIKE ? OR LOWER(c.last_preview) LIKE ?", like, like, like, like, like)
	}
	if before != nil {
		tx = tx.Where("c.last_message_at < ?", *before)
	}
	err := tx.Order("c.pinned DESC, c.last_message_at DESC").Limit(limit).Scan(&out).Error
	return out, err
}

// ListMessages devuelve los mensajes de un chat, del más nuevo al más antiguo.
func (s *Store) ListMessages(instanceID, chat string, limit int, before *time.Time) ([]Message, error) {
	var out []Message
	tx := s.db.Where("instance_id = ? AND chat_jid = ?", instanceID, chat)
	if before != nil {
		tx = tx.Where("timestamp < ?", *before)
	}
	err := tx.Order("timestamp DESC, id DESC").Limit(limit).Omit("raw").Find(&out).Error
	return out, err
}

// GetMessage devuelve un mensaje con su JSON original.
func (s *Store) GetMessage(instanceID, messageID string) (*Message, error) {
	var m Message
	err := s.db.Where("instance_id = ? AND message_id = ?", instanceID, messageID).First(&m).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (s *Store) ListContacts(instanceID, q string, limit int) ([]Contact, error) {
	var out []Contact
	tx := s.db.Where("instance_id = ?", instanceID)
	if q = strings.TrimSpace(q); q != "" {
		like := "%" + strings.ToLower(q) + "%"
		tx = tx.Where("LOWER(jid) LIKE ? OR LOWER(full_name) LIKE ? OR LOWER(push_name) LIKE ? OR LOWER(business_name) LIKE ?", like, like, like, like)
	}
	err := tx.Order("COALESCE(NULLIF(full_name, ''), NULLIF(push_name, ''), jid)").Limit(limit).Find(&out).Error
	return out, err
}

// MarkRead pone a cero los no leídos y devuelve los últimos mensajes entrantes
// sin leer, para enviar los acuses.
func (s *Store) MarkRead(instanceID, chat string) ([]Message, error) {
	var unread int
	s.db.Model(&Chat{}).Select("unread_count").Where("instance_id = ? AND jid = ?", instanceID, chat).Scan(&unread)
	if err := s.db.Model(&Chat{}).Where("instance_id = ? AND jid = ?", instanceID, chat).Update("unread_count", 0).Error; err != nil {
		return nil, err
	}
	if unread <= 0 {
		return nil, nil
	}
	if unread > 50 {
		unread = 50
	}
	var msgs []Message
	err := s.db.Where("instance_id = ? AND chat_jid = ? AND NOT from_me", instanceID, chat).
		Order("timestamp DESC").Limit(unread).Omit("raw").Find(&msgs).Error
	return msgs, err
}

// LastMessage es el último mensaje guardado de un chat; WhatsApp lo pide para
// archivar o marcar como no leído.
type LastMessage struct {
	ID        string
	FromMe    bool
	Sender    string
	Timestamp time.Time
}

// LastOf devuelve el último mensaje de un chat con el almacén global (ok=false
// si el historial está apagado o no hay mensajes).
func LastOf(instanceID, chat string) (LastMessage, bool) {
	s := current()
	if s == nil {
		return LastMessage{}, false
	}
	var m Message
	if err := s.db.Select("message_id", "from_me", "sender_jid", "timestamp").
		Where("instance_id = ? AND chat_jid = ?", instanceID, chat).
		Order("timestamp DESC, id DESC").Limit(1).Find(&m).Error; err != nil || m.MessageID == "" {
		return LastMessage{}, false
	}
	return LastMessage{ID: m.MessageID, FromMe: m.FromMe, Sender: m.SenderJID, Timestamp: m.Timestamp}, true
}
