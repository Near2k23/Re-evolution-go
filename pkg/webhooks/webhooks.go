// Package webhooks entrega los eventos a los webhooks: cabeceras propias, firma
// HMAC, URL por evento, reintentos con espera creciente y registro de entregas.
//
// Variables de entorno:
//
//	WEBHOOK_TIMEOUT_MS     (15000) tiempo máximo de cada intento
//	WEBHOOK_RETRY_MAX      (8)     intentos en total (≈2 min de reintentos, como upstream)
//	WEBHOOK_RETRY_BASE_MS  (1000)  espera antes del 2.º intento; se duplica hasta 60 s
//	WEBHOOK_SECRET         ("")    secreto HMAC para el webhook global (WEBHOOK_URL)
//	WEBHOOK_LOG_DAYS       (3)     días que se guarda el registro de entregas
package webhooks

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	mrand "math/rand"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/evolution-foundation/evolution-go/pkg/evoformat"
	"github.com/gomessguii/logger"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Config es la configuración de webhook de una línea (la URL y los eventos
// siguen en la instancia, que es lo que fija /instance/connect).
type Config struct {
	InstanceID string            `json:"-" gorm:"primaryKey;size:64"`
	Headers    map[string]string `json:"headers" gorm:"serializer:json;type:jsonb"`
	ByEvents   bool              `json:"byEvents"`
	Format     string            `json:"format" gorm:"size:24"` // "" (Evolution GO) o "evolution-api"
	Secret     string            `json:"-"`
	UpdatedAt  time.Time         `json:"updatedAt"`
}

func (Config) TableName() string { return "webhook_configs" }

// FormatEvolutionAPI envía los eventos con los nombres y la forma de Evolution API v2.
const FormatEvolutionAPI = "evolution-api"

// ValidFormat indica si el formato es conocido.
func ValidFormat(f string) bool { return f == "" || f == FormatEvolutionAPI }

// Delivery es el resultado final de una entrega.
type Delivery struct {
	ID         uint64    `json:"id" gorm:"primaryKey;autoIncrement"`
	InstanceID string    `json:"instanceId" gorm:"size:64;index:idx_webhook_deliveries_inst_at,priority:1"`
	Event      string    `json:"event" gorm:"size:64"`
	URL        string    `json:"url"`
	StatusCode int       `json:"statusCode"`
	Attempts   int       `json:"attempts"`
	DurationMs int64     `json:"durationMs"`
	Error      string    `json:"error,omitempty"`
	OK         bool      `json:"ok"`
	Test       bool      `json:"test"`
	At         time.Time `json:"at" gorm:"index:idx_webhook_deliveries_inst_at,priority:2;index"`
}

func (Delivery) TableName() string { return "webhook_deliveries" }

// Settings son los ajustes globales de entrega.
type Settings struct {
	Timeout      time.Duration
	MaxAttempts  int
	BaseDelay    time.Duration
	GlobalSecret string
	GlobalFormat string // WEBHOOK_FORMAT
	ServerURL    string // SERVER_URL, va en el sobre de Evolution API
	LogDays      int
}

func envInt(key string, def int) int {
	if v, err := strconv.Atoi(os.Getenv(key)); err == nil && v >= 0 {
		return v
	}
	return def
}

func SettingsFromEnv() Settings {
	s := Settings{
		Timeout:      time.Duration(envInt("WEBHOOK_TIMEOUT_MS", 15000)) * time.Millisecond,
		MaxAttempts:  envInt("WEBHOOK_RETRY_MAX", 8),
		BaseDelay:    time.Duration(envInt("WEBHOOK_RETRY_BASE_MS", 1000)) * time.Millisecond,
		GlobalSecret: os.Getenv("WEBHOOK_SECRET"),
		GlobalFormat: strings.TrimSpace(os.Getenv("WEBHOOK_FORMAT")),
		ServerURL:    strings.TrimRight(os.Getenv("SERVER_URL"), "/"),
		LogDays:      envInt("WEBHOOK_LOG_DAYS", 3),
	}
	if s.MaxAttempts < 1 {
		s.MaxAttempts = 1
	}
	if s.Timeout <= 0 {
		s.Timeout = 15 * time.Second
	}
	return s
}

// Service guarda configuraciones y entregas y hace los envíos.
type Service struct {
	db       *gorm.DB
	settings Settings
	client   *http.Client
	cache    sync.Map // instanceID → *Config (nil = sin configuración)
	jids     sync.Map // instanceID → JID de la línea (para el sobre v2)
	sleep    func(time.Duration)
}

func New(db *gorm.DB, s Settings) *Service {
	return &Service{db: db, settings: s, client: &http.Client{Timeout: s.Timeout}, sleep: time.Sleep}
}

func (s *Service) Settings() Settings { return s.settings }

func (s *Service) Migrate() error {
	if err := s.db.AutoMigrate(&Config{}, &Delivery{}); err != nil {
		return err
	}
	s.prune()
	return nil
}

func (s *Service) prune() {
	if s.settings.LogDays <= 0 {
		return
	}
	s.db.Where("at < ?", time.Now().AddDate(0, 0, -s.settings.LogDays)).Delete(&Delivery{})
}

// StartPruner borra entregas antiguas una vez al día.
func (s *Service) StartPruner() {
	go func() {
		t := time.NewTicker(24 * time.Hour)
		for range t.C {
			s.prune()
		}
	}()
}

// Get devuelve la configuración de una línea (con caché).
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

var headerName = regexp.MustCompile(`^[A-Za-z0-9-]{1,64}$`)

// reserved son cabeceras que pone el servicio y no se pueden sobrescribir.
var reserved = map[string]bool{"content-type": true, "content-length": true, "host": true, "x-webhook-signature": true, "x-webhook-timestamp": true, "x-webhook-event": true, "x-webhook-delivery": true}

// ErrInvalidHeader indica una cabecera con nombre no válido o reservada.
var ErrInvalidHeader = errors.New("invalid or reserved header name")

// Save guarda cabeceras y URL por evento. secret: nil = no cambiar, "" = quitar.
func (s *Service) Save(instanceID string, headers map[string]string, byEvents bool, secret *string) (*Config, error) {
	clean := map[string]string{}
	for k, v := range headers {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		if !headerName.MatchString(k) || reserved[strings.ToLower(k)] {
			return nil, fmt.Errorf("%w: %s", ErrInvalidHeader, k)
		}
		clean[k] = strings.TrimSpace(v)
	}
	c := Config{InstanceID: instanceID, Headers: clean, ByEvents: byEvents, UpdatedAt: time.Now()}
	cols := []string{"headers", "by_events", "updated_at"}
	if secret != nil {
		c.Secret = *secret
		cols = append(cols, "secret")
	} else if old := s.Get(instanceID); old != nil {
		c.Secret = old.Secret
	}
	err := s.db.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "instance_id"}}, DoUpdates: clause.AssignmentColumns(cols)}).Create(&c).Error
	s.cache.Delete(instanceID)
	if err != nil {
		return nil, err
	}
	return s.Get(instanceID), nil
}

// SetFormat cambia el formato de los eventos de una línea.
func (s *Service) SetFormat(instanceID, format string) error {
	if !ValidFormat(format) {
		return fmt.Errorf("unknown format %q", format)
	}
	c := Config{InstanceID: instanceID, Headers: map[string]string{}, Format: format, UpdatedAt: time.Now()}
	err := s.db.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "instance_id"}}, DoUpdates: clause.AssignmentColumns([]string{"format", "updated_at"})}).Create(&c).Error
	s.cache.Delete(instanceID)
	return err
}

// NewSecret genera un secreto aleatorio.
func NewSecret() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return "whsec_" + hex.EncodeToString(b)
}

// DeleteInstance borra configuración y entregas de una línea.
func (s *Service) DeleteInstance(instanceID string) {
	s.db.Where("instance_id = ?", instanceID).Delete(&Config{})
	s.db.Where("instance_id = ?", instanceID).Delete(&Delivery{})
	s.cache.Delete(instanceID)
}

// Recent devuelve las últimas entregas de una línea.
func (s *Service) Recent(instanceID string, limit int) ([]Delivery, error) {
	var out []Delivery
	err := s.db.Where("instance_id = ?", instanceID).Order("at DESC").Limit(limit).Find(&out).Error
	return out, err
}

// Stats cuenta entregas correctas y fallidas desde since (sin las de prueba).
type Stats struct {
	OK     int64 `json:"ok"`
	Failed int64 `json:"failed"`
}

func (s *Service) Stats(instanceID string, since time.Time) Stats {
	var st Stats
	q := s.db.Model(&Delivery{}).Where("at >= ? AND NOT test", since)
	if instanceID != "" {
		q = q.Where("instance_id = ?", instanceID)
	}
	q.Session(&gorm.Session{}).Where("ok").Count(&st.OK)
	q.Session(&gorm.Session{}).Where("NOT ok").Count(&st.Failed)
	return st
}

// StatsByInstance cuenta entregas por línea desde since (para el panel).
func (s *Service) StatsByInstance(since time.Time) map[string]Stats {
	type row struct {
		InstanceID string
		OK         bool
		N          int64
	}
	var rows []row
	s.db.Model(&Delivery{}).Select("instance_id, ok, count(*) AS n").Where("at >= ? AND NOT test", since).Group("instance_id, ok").Scan(&rows)
	out := map[string]Stats{}
	for _, r := range rows {
		st := out[r.InstanceID]
		if r.OK {
			st.OK = r.N
		} else {
			st.Failed = r.N
		}
		out[r.InstanceID] = st
	}
	return out
}

var kebabRe = regexp.MustCompile(`([a-z0-9])([A-Z])`)

// Kebab convierte «SendMessage» en «send-message».
func Kebab(event string) string {
	return strings.ToLower(kebabRe.ReplaceAllString(event, "$1-$2"))
}

// Sign calcula la firma: hex(HMAC-SHA256(secret, timestamp + "." + body)).
func Sign(secret, timestamp string, body []byte) string {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write([]byte(timestamp))
	m.Write([]byte("."))
	m.Write(body)
	return hex.EncodeToString(m.Sum(nil))
}

// EventOf lee el campo "event" del payload.
func EventOf(payload []byte) string {
	var p struct {
		Event string `json:"event"`
	}
	_ = json.Unmarshal(payload, &p)
	return p.Event
}

// retryable indica si vale la pena repetir tras ese código de respuesta.
func retryable(code int) bool {
	if code == 0 || code >= 500 {
		return true
	}
	return code == 408 || code == 425 || code == 429
}

// backoff es la espera antes del intento n (n ≥ 1): base·2^(n-1) ±20 %, máx. 60 s.
func (s *Service) backoff(n int) time.Duration {
	d := float64(s.settings.BaseDelay) * math.Pow(2, float64(n-1))
	if d > float64(60*time.Second) {
		d = float64(60 * time.Second)
	}
	jitter := 0.8 + 0.4*mrand.Float64()
	return time.Duration(d * jitter)
}

// Target es un destino concreto de una entrega.
type Target struct {
	InstanceID string
	URL        string
	Headers    map[string]string
	Secret     string
	ByEvents   bool
	Format     string
}

// TargetFor arma el destino de la línea con su configuración.
func (s *Service) TargetFor(instanceID, url string) Target {
	t := Target{InstanceID: instanceID, URL: url}
	if c := s.Get(instanceID); c != nil {
		t.Headers, t.Secret, t.ByEvents, t.Format = c.Headers, c.Secret, c.ByEvents, c.Format
	}
	return t
}

// GlobalTarget es el webhook global (WEBHOOK_URL).
func (s *Service) GlobalTarget(instanceID, url string) Target {
	return Target{InstanceID: instanceID, URL: url, Secret: s.settings.GlobalSecret, Format: s.settings.GlobalFormat}
}

// Result es el resultado de una entrega.
type Result struct {
	StatusCode int    `json:"statusCode"`
	Attempts   int    `json:"attempts"`
	DurationMs int64  `json:"durationMs"`
	Error      string `json:"error,omitempty"`
	OK         bool   `json:"ok"`
	URL        string `json:"url"`
}

// Deliver envía con reintentos y guarda el resultado. Bloquea: llamarla en una
// goroutine. Con formato Evolution API el evento se traduce antes (puede dar
// varios envíos, o ninguno si no tiene equivalente).
func (s *Service) Deliver(t Target, payload []byte, test bool) Result {
	if t.Format != FormatEvolutionAPI || test {
		event := EventOf(payload)
		path := ""
		if event != "" {
			path = Kebab(event)
		}
		return s.deliver(t, payload, event, path, test)
	}
	conv := evoformat.Convert(payload, evoformat.Meta{ServerURL: s.settings.ServerURL, Sender: s.jidOf(t.InstanceID), Destination: t.URL})
	res := Result{OK: true, URL: t.URL}
	for _, c := range conv {
		res = s.deliver(t, c.Payload, c.Event, evoformat.Path(c.Event), false)
	}
	return res
}

// jidOf devuelve el JID de la línea (sin dispositivo), con caché.
func (s *Service) jidOf(instanceID string) string {
	if v, ok := s.jids.Load(instanceID); ok && v.(string) != "" {
		return v.(string)
	}
	if s.db == nil {
		return ""
	}
	var jid string
	s.db.Table("instances").Select("jid").Where("id = ?", instanceID).Scan(&jid)
	if i := strings.Index(jid, ":"); i > 0 {
		if at := strings.Index(jid, "@"); at > i {
			jid = jid[:i] + jid[at:]
		}
	}
	s.jids.Store(instanceID, jid)
	return jid
}

func (s *Service) deliver(t Target, payload []byte, event, path string, test bool) Result {
	url := t.URL
	if t.ByEvents && path != "" {
		url = strings.TrimRight(url, "/") + "/" + path
	}
	attempts := s.settings.MaxAttempts
	if test {
		attempts = 1
	}
	deliveryID := uuid.NewString()
	start := time.Now()
	var res Result
	for n := 1; n <= attempts; n++ {
		if n > 1 {
			s.sleep(s.backoff(n - 1))
		}
		res = s.attempt(t, url, event, deliveryID, payload)
		res.Attempts = n
		if res.OK || !retryable(res.StatusCode) {
			break
		}
	}
	res.URL = url
	res.DurationMs = time.Since(start).Milliseconds()
	if s.db != nil && t.InstanceID != "" {
		d := Delivery{InstanceID: t.InstanceID, Event: event, URL: url, StatusCode: res.StatusCode, Attempts: res.Attempts,
			DurationMs: res.DurationMs, Error: res.Error, OK: res.OK, Test: test, At: time.Now()}
		if err := s.db.Create(&d).Error; err != nil {
			logger.LogError("[webhooks] save delivery: %v", err)
		}
	}
	return res
}

func (s *Service) attempt(t Target, url, event, deliveryID string, payload []byte) Result {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return Result{Error: err.Error()}
	}
	for k, v := range t.Headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Delivery", deliveryID)
	if event != "" {
		req.Header.Set("X-Webhook-Event", event)
	}
	if t.Secret != "" {
		ts := strconv.FormatInt(time.Now().Unix(), 10)
		req.Header.Set("X-Webhook-Timestamp", ts)
		req.Header.Set("X-Webhook-Signature", "sha256="+Sign(t.Secret, ts, payload))
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return Result{Error: err.Error()}
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64*1024))
	r := Result{StatusCode: resp.StatusCode, OK: resp.StatusCode >= 200 && resp.StatusCode < 300}
	if !r.OK {
		r.Error = resp.Status
	}
	return r
}

var (
	defMu  sync.RWMutex
	defSvc *Service
)

func SetDefault(s *Service) {
	defMu.Lock()
	defSvc = s
	defMu.Unlock()
}

// Default devuelve el servicio global (nil si no se inicializó).
func Default() *Service {
	defMu.RLock()
	defer defMu.RUnlock()
	return defSvc
}
