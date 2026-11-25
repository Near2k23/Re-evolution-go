// Package chatbot es el motor común de bots: elige el bot de cada chat por su
// disparador, mantiene una sesión por chat, llama al proveedor (HTTP, n8n,
// OpenAI, Typebot, Dify, Flowise) y envía sus respuestas por la línea.
package chatbot

import (
	"encoding/json"
	"time"
)

// Tipos de bot.
const (
	TypeHTTP    = "http"
	TypeN8N     = "n8n"
	TypeOpenAI  = "openai"
	TypeTypebot = "typebot"
	TypeDify    = "dify"
	TypeFlowise = "flowise"
)

var Types = []string{TypeHTTP, TypeN8N, TypeOpenAI, TypeTypebot, TypeDify, TypeFlowise}

// Disparadores.
const (
	TriggerAll     = "all"     // cualquier mensaje
	TriggerKeyword = "keyword" // solo si el texto coincide
)

// Operadores del disparador por palabra clave.
const (
	OpContains   = "contains"
	OpEquals     = "equals"
	OpStartsWith = "startsWith"
	OpEndsWith   = "endsWith"
	OpRegex      = "regex"
)

// Bot es un bot de una línea.
type Bot struct {
	ID              string          `json:"id" gorm:"type:uuid;primaryKey"`
	InstanceID      string          `json:"instanceId" gorm:"size:64;index;not null"`
	Type            string          `json:"type" gorm:"size:16;not null"`
	Name            string          `json:"name" gorm:"size:120;not null"`
	Enabled         bool            `json:"enabled"`
	Priority        int             `json:"priority"`
	TriggerType     string          `json:"triggerType" gorm:"size:16"`
	TriggerOperator string          `json:"triggerOperator" gorm:"size:16"`
	TriggerValue    string          `json:"triggerValue"`
	AllowGroups     bool            `json:"allowGroups"`
	ListenFromMe    bool            `json:"listenFromMe"`
	StopOnFromMe    bool            `json:"stopOnFromMe"`
	KeywordFinish   string          `json:"keywordFinish"`
	ExpireMinutes   int             `json:"expireMinutes"`
	DelayMs         int             `json:"delayMs"`
	DebounceMs      int             `json:"debounceMs"`
	SplitMessages   bool            `json:"splitMessages"`
	UnknownMessage  string          `json:"unknownMessage"`
	IgnoreJIDs      []string        `json:"ignoreJids" gorm:"serializer:json;type:jsonb"`
	Config          json.RawMessage `json:"config" gorm:"type:jsonb"`
	CreatedAt       time.Time       `json:"createdAt"`
	UpdatedAt       time.Time       `json:"updatedAt"`
}

func (Bot) TableName() string { return "chatbots" }

// Estados de sesión.
const (
	SessionOpen   = "open"
	SessionPaused = "paused"
	SessionClosed = "closed"
)

// Session es la conversación de un chat con un bot.
type Session struct {
	ID           string          `json:"id" gorm:"type:uuid;primaryKey"`
	InstanceID   string          `json:"instanceId" gorm:"size:64;not null;index:idx_chatbot_sess_chat,priority:1"`
	RemoteJID    string          `json:"remoteJid" gorm:"column:remote_jid;size:128;not null;index:idx_chatbot_sess_chat,priority:2"`
	BotID        string          `json:"botId" gorm:"type:uuid;index;not null"`
	Status       string          `json:"status" gorm:"size:16;not null;index:idx_chatbot_sess_chat,priority:3"`
	PushName     string          `json:"pushName"`
	ExternalID   string          `json:"externalId"`
	Context      json.RawMessage `json:"-" gorm:"type:jsonb"`
	LastActivity time.Time       `json:"lastActivity"`
	CreatedAt    time.Time       `json:"createdAt"`
}

func (Session) TableName() string { return "chatbot_sessions" }

// Input es lo que recibe un proveedor.
type Input struct {
	InstanceID string
	RemoteJID  string
	PushName   string
	MessageID  string
	Type       string // text, audio, image…
	Text       string
	FromMe     bool
	IsGroup    bool
	Audio      []byte // audio descargado (para transcribir)
	AudioMime  string
}

// Reply es una respuesta del bot: texto o un archivo por URL.
type Reply struct {
	Text      string
	MediaURL  string
	MediaType string // image, video, audio, document
}

// Output es lo que devuelve un proveedor.
type Output struct {
	Replies    []Reply
	ExternalID string          // id de sesión/conversación del proveedor
	Context    json.RawMessage // estado propio (p. ej. historial de OpenAI)
	End        bool            // el proveedor terminó la conversación
}
