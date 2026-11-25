// Package metrics expone métricas en formato de texto de Prometheus en
// GET /metrics, sin dependencias externas: líneas por estado, mensajes
// recibidos y enviados, entregas de webhook y datos del proceso.
package metrics

import (
	"crypto/subtle"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"go.mau.fi/whatsmeow/types/events"
)

var started = time.Now()

// counterVec es un contador con una etiqueta.
type counterVec struct {
	m sync.Map // etiqueta → *uint64
}

func (c *counterVec) inc(label string, n uint64) {
	v, _ := c.m.LoadOrStore(label, new(uint64))
	atomic.AddUint64(v.(*uint64), n)
}

func (c *counterVec) snapshot() map[string]uint64 {
	out := map[string]uint64{}
	c.m.Range(func(k, v interface{}) bool {
		out[k.(string)] = atomic.LoadUint64(v.(*uint64))
		return true
	})
	return out
}

// histogram es un histograma acumulativo simple.
type histogram struct {
	mu      sync.Mutex
	buckets []float64
	counts  []uint64
	sum     float64
	count   uint64
}

func newHistogram(b ...float64) *histogram {
	return &histogram{buckets: b, counts: make([]uint64, len(b))}
}

func (h *histogram) observe(v float64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for i, b := range h.buckets {
		if v <= b {
			h.counts[i]++
		}
	}
	h.sum += v
	h.count++
}

var (
	received      counterVec // instanceID
	sent          counterVec // instanceID
	calls         counterVec // instanceID
	webhookResult counterVec // ok | failed
	webhookTime   = newHistogram(0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60, 120)
)

// Observe cuenta los eventos de whatsmeow de una línea.
func Observe(instanceID string, evt interface{}) {
	switch e := evt.(type) {
	case *events.Message:
		if e.Info.IsFromMe {
			return // los enviados desde la API se cuentan en ObserveSent
		}
		received.inc(instanceID, 1)
	case *events.CallOffer:
		calls.inc(instanceID, 1)
	}
}

// ObserveSent cuenta un mensaje enviado por la API.
func ObserveSent(instanceID string) { sent.inc(instanceID, 1) }

// ObserveWebhook cuenta una entrega de webhook terminada.
func ObserveWebhook(ok bool, d time.Duration) {
	if ok {
		webhookResult.inc("ok", 1)
	} else {
		webhookResult.inc("failed", 1)
	}
	webhookTime.observe(d.Seconds())
}

// Line es el estado de una línea en el momento de la lectura.
type Line struct {
	ID, Name  string
	Connected bool // conectada y con sesión
	LoggedIn  bool // tiene sesión (aunque esté reconectando)
}

// Options configura el endpoint.
type Options struct {
	Version   string
	GlobalKey string
	User      string // METRICS_USER (autenticación básica opcional)
	Password  string // METRICS_PASSWORD
	Lines     func() []Line
}

// OptionsFromEnv lee METRICS_USER y METRICS_PASSWORD.
func OptionsFromEnv() Options {
	return Options{User: os.Getenv("METRICS_USER"), Password: os.Getenv("METRICS_PASSWORD")}
}

// Enabled indica si /metrics está activo (METRICS_ENABLED, por defecto sí).
func Enabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("METRICS_ENABLED")))
	return v != "false" && v != "0" && v != "no"
}

func eq(a, b string) bool { return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1 }

// authorized acepta la autenticación básica configurada o la clave global
// (cabecera apikey o Authorization: Bearer).
func (o Options) authorized(r *http.Request) bool {
	if o.User != "" {
		if u, p, ok := r.BasicAuth(); ok && eq(u, o.User) && eq(p, o.Password) {
			return true
		}
	}
	if o.GlobalKey == "" {
		return false
	}
	if k := r.Header.Get("apikey"); k != "" && eq(k, o.GlobalKey) {
		return true
	}
	if b := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "); b != r.Header.Get("Authorization") && eq(b, o.GlobalKey) {
		return true
	}
	return false
}

// Register monta GET /metrics.
func Register(r *gin.Engine, o Options) {
	r.GET("/metrics", func(c *gin.Context) {
		if !o.authorized(c.Request) {
			if o.User != "" {
				c.Header("WWW-Authenticate", `Basic realm="metrics"`)
			}
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Header("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		c.Status(http.StatusOK)
		Write(c.Writer, o)
	})
}

func esc(s string) string {
	return strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`).Replace(s)
}

func header(w io.Writer, name, typ, help string) {
	fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s %s\n", name, help, name, typ)
}

// Write escribe todas las métricas.
func Write(w io.Writer, o Options) {
	var lines []Line
	if o.Lines != nil {
		lines = o.Lines()
	}
	names := map[string]string{}
	state := map[string]int{"open": 0, "connecting": 0, "close": 0}
	for _, l := range lines {
		names[l.ID] = l.Name
		switch {
		case l.Connected:
			state["open"]++
		case l.LoggedIn:
			state["connecting"]++
		default:
			state["close"]++
		}
	}
	label := func(id string) string {
		n := names[id]
		if n == "" {
			n = id
		}
		return fmt.Sprintf(`instance="%s",instance_id="%s"`, esc(n), esc(id))
	}

	header(w, "evolution_build_info", "gauge", "Version of the running server.")
	fmt.Fprintf(w, "evolution_build_info{version=\"%s\"} 1\n", esc(o.Version))

	header(w, "evolution_instances", "gauge", "Lines by connection state (open, connecting, close).")
	for _, s := range []string{"open", "connecting", "close"} {
		fmt.Fprintf(w, "evolution_instances{state=\"%s\"} %d\n", s, state[s])
	}
	header(w, "evolution_instance_up", "gauge", "1 when the line is connected and logged in.")
	sort.Slice(lines, func(i, j int) bool { return lines[i].Name < lines[j].Name })
	for _, l := range lines {
		up := 0
		if l.Connected {
			up = 1
		}
		fmt.Fprintf(w, "evolution_instance_up{%s} %d\n", label(l.ID), up)
	}

	vec := func(name, help string, c *counterVec) {
		header(w, name, "counter", help)
		snap := c.snapshot()
		ids := make([]string, 0, len(snap))
		for id := range snap {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			fmt.Fprintf(w, "%s{%s} %d\n", name, label(id), snap[id])
		}
	}
	vec("evolution_messages_received_total", "Messages received by each line since start.", &received)
	vec("evolution_messages_sent_total", "Messages sent through the API by each line since start.", &sent)
	vec("evolution_calls_received_total", "Calls received by each line since start.", &calls)

	header(w, "evolution_webhook_deliveries_total", "counter", "Finished webhook deliveries by result.")
	res := webhookResult.snapshot()
	for _, r := range []string{"ok", "failed"} {
		fmt.Fprintf(w, "evolution_webhook_deliveries_total{result=\"%s\"} %d\n", r, res[r])
	}
	header(w, "evolution_webhook_delivery_seconds", "histogram", "Time to finish a webhook delivery, retries included.")
	webhookTime.mu.Lock()
	for i, b := range webhookTime.buckets {
		fmt.Fprintf(w, "evolution_webhook_delivery_seconds_bucket{le=\"%s\"} %d\n", fmtFloat(b), webhookTime.counts[i])
	}
	fmt.Fprintf(w, "evolution_webhook_delivery_seconds_bucket{le=\"+Inf\"} %d\n", webhookTime.count)
	fmt.Fprintf(w, "evolution_webhook_delivery_seconds_sum %s\n", fmtFloat(webhookTime.sum))
	fmt.Fprintf(w, "evolution_webhook_delivery_seconds_count %d\n", webhookTime.count)
	webhookTime.mu.Unlock()

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	header(w, "process_start_time_seconds", "gauge", "Start time of the process since unix epoch in seconds.")
	fmt.Fprintf(w, "process_start_time_seconds %d\n", started.Unix())
	header(w, "go_goroutines", "gauge", "Number of goroutines that currently exist.")
	fmt.Fprintf(w, "go_goroutines %d\n", runtime.NumGoroutine())
	header(w, "go_memstats_heap_alloc_bytes", "gauge", "Number of heap bytes allocated and still in use.")
	fmt.Fprintf(w, "go_memstats_heap_alloc_bytes %d\n", ms.HeapAlloc)
	header(w, "go_memstats_sys_bytes", "gauge", "Number of bytes obtained from the OS.")
	fmt.Fprintf(w, "go_memstats_sys_bytes %d\n", ms.Sys)
}

func fmtFloat(f float64) string {
	if f == math.Trunc(f) {
		return fmt.Sprintf("%.0f", f)
	}
	return fmt.Sprintf("%g", f)
}
