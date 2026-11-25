// Package connlog guarda el historial de conexión de cada instancia (conexiones,
// cortes, cierres de sesión, bloqueos) y clasifica su estado para el manager:
// conectada, reconectada (volvió sola tras un corte reciente), reconectando o
// caída.
package connlog

import (
	"sync"
	"time"

	"go.mau.fi/whatsmeow/types/events"
	"gorm.io/gorm"
)

// Tipos de evento guardados.
const (
	KindConnected      = "connected"
	KindDisconnected   = "disconnected"
	KindLoggedOut      = "logged_out"
	KindTemporaryBan   = "temporary_ban"
	KindConnectFailure = "connect_failure"
	KindStreamReplaced = "stream_replaced"
	KindPairSuccess    = "pair_success"
)

// Estados que ve el manager.
const (
	StateConnected    = "conectada"
	StateReconnected  = "reconectada"
	StateReconnecting = "reconectando"
	StateDown         = "caida"
)

// Retention es cuánto se guarda el historial.
const Retention = 30 * 24 * time.Hour

// ConnectionEvent es una fila del historial.
type ConnectionEvent struct {
	ID         uint      `json:"-" gorm:"primaryKey"`
	InstanceID string    `json:"instanceId" gorm:"index:idx_conn_events_instance_at,priority:1;not null"`
	Kind       string    `json:"kind" gorm:"size:32;not null"`
	Reason     string    `json:"reason,omitempty"`
	At         time.Time `json:"at" gorm:"index:idx_conn_events_instance_at,priority:2;index;not null"`
}

func (ConnectionEvent) TableName() string { return "connection_events" }

// Repository lee y escribe el historial.
type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

// Migrate crea la tabla y borra lo que pasa de Retention.
func (r *Repository) Migrate() error {
	if err := r.db.AutoMigrate(&ConnectionEvent{}); err != nil {
		return err
	}
	return r.db.Where("at < ?", time.Now().Add(-Retention)).Delete(&ConnectionEvent{}).Error
}

func (r *Repository) Record(instanceID, kind, reason string, at time.Time) error {
	return r.db.Create(&ConnectionEvent{InstanceID: instanceID, Kind: kind, Reason: reason, At: at}).Error
}

// List devuelve los eventos de una instancia desde since, del más antiguo al más reciente.
func (r *Repository) List(instanceID string, since time.Time) ([]ConnectionEvent, error) {
	var out []ConnectionEvent
	err := r.db.Where("instance_id = ? AND at >= ?", instanceID, since).Order("at ASC").Find(&out).Error
	return out, err
}

// ListAll devuelve los eventos de todas las instancias desde since, agrupados por instancia.
func (r *Repository) ListAll(since time.Time) (map[string][]ConnectionEvent, error) {
	var rows []ConnectionEvent
	if err := r.db.Where("at >= ?", since).Order("at ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string][]ConnectionEvent)
	for _, e := range rows {
		out[e.InstanceID] = append(out[e.InstanceID], e)
	}
	return out, nil
}

// LastByInstance devuelve el último evento de cada instancia (sin límite de fecha).
func (r *Repository) LastByInstance() (map[string]ConnectionEvent, error) {
	var rows []ConnectionEvent
	err := r.db.Raw(`SELECT DISTINCT ON (instance_id) id, instance_id, kind, reason, at
		FROM connection_events ORDER BY instance_id, at DESC`).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[string]ConnectionEvent, len(rows))
	for _, e := range rows {
		out[e.InstanceID] = e
	}
	return out, nil
}

// DeleteInstance borra el historial de una instancia.
func (r *Repository) DeleteInstance(instanceID string) error {
	return r.db.Where("instance_id = ?", instanceID).Delete(&ConnectionEvent{}).Error
}

var (
	defaultMu   sync.RWMutex
	defaultRepo *Repository
)

// SetDefault fija el repositorio que usa Observe.
func SetDefault(r *Repository) {
	defaultMu.Lock()
	defaultRepo = r
	defaultMu.Unlock()
}

// KindOf traduce un evento de whatsmeow a un tipo del historial ("" si no aplica).
func KindOf(evt interface{}) (kind, reason string) {
	switch e := evt.(type) {
	case *events.Connected:
		return KindConnected, ""
	case *events.Disconnected:
		return KindDisconnected, ""
	case *events.LoggedOut:
		return KindLoggedOut, e.Reason.String()
	case *events.TemporaryBan:
		return KindTemporaryBan, e.String()
	case *events.ConnectFailure:
		return KindConnectFailure, e.Reason.String()
	case *events.StreamReplaced:
		return KindStreamReplaced, ""
	case *events.PairSuccess:
		return KindPairSuccess, ""
	}
	return "", ""
}

// Observe guarda el evento si es de conexión. No bloquea al manejador de eventos.
func Observe(instanceID string, evt interface{}) {
	kind, reason := KindOf(evt)
	if kind == "" {
		return
	}
	defaultMu.RLock()
	r := defaultRepo
	defaultMu.RUnlock()
	if r == nil {
		return
	}
	at := time.Now()
	go func() { _ = r.Record(instanceID, kind, reason, at) }()
}
