// Package sinks envía los eventos de todas las líneas a destinos externos
// configurados por entorno: Amazon SQS, Kafka y Pusher. Cada destino tiene su
// filtro de eventos y puede usar el formato de Evolution API v2. Los envíos
// van por una cola en memoria con reintentos, sin bloquear a WhatsApp.
package sinks

import (
	"context"
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/evolution-foundation/evolution-go/pkg/evoformat"
	"github.com/gomessguii/logger"
)

// Message es un evento listo para un destino.
type Message struct {
	Event        string // nombre del evento (Go o v2 según el formato)
	InstanceID   string
	InstanceName string
	Payload      []byte
}

// Sink es un destino externo.
type Sink interface {
	Name() string
	Send(ctx context.Context, m Message) error
	Close() error
}

// Filter decide qué eventos recibe un destino y en qué formato.
type Filter struct {
	All    bool
	Groups map[string]bool // grupos de Go (MESSAGE, CONNECTION…) o nombres de evento
	V2     bool            // formato Evolution API v2
}

// FilterFromEnv lee <PREFIX>_EVENTS y <PREFIX>_FORMAT.
func FilterFromEnv(prefix string) Filter {
	f := Filter{Groups: map[string]bool{}, V2: strings.EqualFold(os.Getenv(prefix+"_FORMAT"), "evolution-api")}
	raw := strings.TrimSpace(os.Getenv(prefix + "_EVENTS"))
	if raw == "" || strings.EqualFold(raw, "ALL") {
		f.All = true
		return f
	}
	for _, e := range strings.Split(raw, ",") {
		if e = strings.TrimSpace(e); e != "" {
			f.Groups[strings.ToUpper(e)] = true
		}
	}
	return f
}

// Allows indica si el evento (nombre Go y su grupo) pasa el filtro.
func (f Filter) Allows(event, group string) bool {
	return f.All || f.Groups[strings.ToUpper(event)] || (group != "" && f.Groups[group])
}

type entry struct {
	sink   Sink
	filter Filter
}

type job struct {
	e entry
	m Message
}

// Hub reparte los eventos entre los destinos.
type Hub struct {
	entries []entry
	queue   chan job
	group   func(event string) string
	meta    func(instanceID string) evoformat.Meta
	retries int
	wg      sync.WaitGroup
}

// Options configura el Hub.
type Options struct {
	Group   func(event string) string           // grupo de suscripción de un evento
	Meta    func(instanceID string) evoformat.Meta // datos del sobre v2
	Workers int
	Queue   int
	Retries int
}

// NewHub crea el repartidor. Sin destinos, Publish no hace nada.
func NewHub(o Options) *Hub {
	if o.Workers <= 0 {
		o.Workers = 4
	}
	if o.Queue <= 0 {
		o.Queue = 10000
	}
	if o.Retries <= 0 {
		o.Retries = 3
	}
	if o.Meta == nil {
		o.Meta = func(string) evoformat.Meta { return evoformat.Meta{} }
	}
	if o.Group == nil {
		o.Group = func(string) string { return "" }
	}
	h := &Hub{queue: make(chan job, o.Queue), group: o.Group, meta: o.Meta, retries: o.Retries}
	for i := 0; i < o.Workers; i++ {
		h.wg.Add(1)
		go h.worker()
	}
	return h
}

// Add registra un destino.
func (h *Hub) Add(s Sink, f Filter) {
	h.entries = append(h.entries, entry{sink: s, filter: f})
	logger.LogInfo("[sinks] %s enabled", s.Name())
}

// Len es el número de destinos.
func (h *Hub) Len() int { return len(h.entries) }

func (h *Hub) worker() {
	defer h.wg.Done()
	for j := range h.queue {
		var err error
		for n := 0; n < h.retries; n++ {
			if n > 0 {
				time.Sleep(time.Duration(n) * time.Second)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			err = j.e.sink.Send(ctx, j.m)
			cancel()
			if err == nil {
				break
			}
		}
		if err != nil {
			logger.LogError("[sinks] %s: %s for %s: %v", j.e.sink.Name(), j.m.Event, j.m.InstanceName, err)
		}
	}
}

// Publish encola un evento de Go ({event, data, instanceId, instanceName…})
// para cada destino que lo acepte.
func (h *Hub) Publish(payload []byte) {
	if h == nil || len(h.entries) == 0 {
		return
	}
	var head struct {
		Event        string `json:"event"`
		InstanceID   string `json:"instanceId"`
		InstanceName string `json:"instanceName"`
	}
	if json.Unmarshal(payload, &head) != nil || head.Event == "" {
		return
	}
	group := h.group(head.Event)
	var v2 []evoformat.Converted
	v2Done := false
	for _, e := range h.entries {
		if !e.filter.Allows(head.Event, group) {
			continue
		}
		msgs := []Message{{Event: head.Event, InstanceID: head.InstanceID, InstanceName: head.InstanceName, Payload: payload}}
		if e.filter.V2 {
			if !v2Done {
				v2, v2Done = evoformat.Convert(payload, h.meta(head.InstanceID)), true
			}
			msgs = msgs[:0]
			for _, c := range v2 {
				msgs = append(msgs, Message{Event: c.Event, InstanceID: head.InstanceID, InstanceName: head.InstanceName, Payload: c.Payload})
			}
		}
		for _, m := range msgs {
			select {
			case h.queue <- job{e: e, m: m}:
			default:
				logger.LogWarn("[sinks] queue full, %s dropped for %s", m.Event, e.sink.Name())
			}
		}
	}
}

// Close vacía la cola y cierra los destinos.
func (h *Hub) Close() {
	if h == nil {
		return
	}
	close(h.queue)
	h.wg.Wait()
	for _, e := range h.entries {
		_ = e.sink.Close()
	}
}

var (
	defMu  sync.RWMutex
	defHub *Hub
)

// SetDefault fija el Hub que usa Publish.
func SetDefault(h *Hub) {
	defMu.Lock()
	defHub = h
	defMu.Unlock()
}

// Publish envía un evento con el Hub global.
func Publish(payload []byte) {
	defMu.RLock()
	h := defHub
	defMu.RUnlock()
	h.Publish(payload)
}

func envBool(k string) bool {
	b, _ := strconv.ParseBool(os.Getenv(k))
	return b
}

func envList(k string) []string {
	var out []string
	for _, v := range strings.Split(os.Getenv(k), ",") {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// sanitize deja letras, números, guion, guion bajo y punto.
func sanitize(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.' {
			return r
		}
		return '_'
	}, s)
}
