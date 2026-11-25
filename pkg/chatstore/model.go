// Package chatstore guarda chats, contactos y mensajes de cada línea para
// poder consultarlos después (manager, integraciones). Escribe en una cola en
// segundo plano para no frenar el manejador de eventos de whatsmeow.
package chatstore

import (
	"encoding/json"
	"time"
)

// Chat es una conversación de una línea.
type Chat struct {
	InstanceID    string     `json:"-" gorm:"primaryKey;size:64"`
	JID           string     `json:"jid" gorm:"column:jid;primaryKey;size:128"`
	Name          string     `json:"name"`
	IsGroup       bool       `json:"isGroup"`
	LastMessageAt *time.Time `json:"lastMessageAt,omitempty" gorm:"index"`
	LastMessageID string     `json:"lastMessageId,omitempty"`
	LastPreview   string     `json:"lastPreview"`
	LastType      string     `json:"lastType"`
	LastFromMe    bool       `json:"lastFromMe"`
	UnreadCount   int        `json:"unread"`
	Archived      bool       `json:"archived"`
	Pinned        bool       `json:"pinned"`
	MutedUntil    *time.Time `json:"mutedUntil,omitempty"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

func (Chat) TableName() string { return "chat_store_chats" }

// Contact es un contacto conocido por una línea.
type Contact struct {
	InstanceID   string    `json:"-" gorm:"primaryKey;size:64"`
	JID          string    `json:"jid" gorm:"column:jid;primaryKey;size:128"`
	FullName     string    `json:"fullName"`
	PushName     string    `json:"pushName"`
	BusinessName string    `json:"businessName"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

func (Contact) TableName() string { return "chat_store_contacts" }

// Estados de un mensaje propio, de menor a mayor.
const (
	StatusReceived  = "received"
	StatusSent      = "sent"
	StatusDelivered = "delivered"
	StatusRead      = "read"
	StatusPlayed    = "played"
)

var statusRank = map[string]int{StatusReceived: 0, StatusSent: 1, StatusDelivered: 2, StatusRead: 3, StatusPlayed: 4}

// Message es un mensaje guardado.
type Message struct {
	ID         uint64          `json:"-" gorm:"primaryKey;autoIncrement"`
	InstanceID string          `json:"-" gorm:"size:64;not null;uniqueIndex:idx_chat_store_msg,priority:1;index:idx_chat_store_chat_ts,priority:1"`
	MessageID  string          `json:"id" gorm:"size:128;not null;uniqueIndex:idx_chat_store_msg,priority:2"`
	ChatJID    string          `json:"chat" gorm:"column:chat_jid;size:128;not null;index:idx_chat_store_chat_ts,priority:2"`
	SenderJID  string          `json:"sender" gorm:"column:sender_jid"`
	SenderName string          `json:"senderName,omitempty"`
	FromMe     bool            `json:"fromMe"`
	Type       string          `json:"type"`
	Text       string          `json:"text"`
	MediaMime  string          `json:"mime,omitempty"`
	FileName   string          `json:"fileName,omitempty"`
	QuotedID   string          `json:"quotedId,omitempty"`
	Status     string          `json:"status"`
	StatusRank int             `json:"-"`
	Edited     bool            `json:"edited"`
	Deleted    bool            `json:"deleted"`
	Timestamp  time.Time       `json:"timestamp" gorm:"index:idx_chat_store_chat_ts,priority:3"`
	Raw        json.RawMessage `json:"raw,omitempty" gorm:"type:jsonb"`
}

func (Message) TableName() string { return "chat_store_messages" }
