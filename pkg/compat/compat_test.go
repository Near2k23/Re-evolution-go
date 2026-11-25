package compat

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	instance_model "github.com/evolution-foundation/evolution-go/pkg/instance/model"
	"github.com/evolution-foundation/evolution-go/pkg/webhooks"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// fakeGo simula el router de Evolution GO y anota lo que recibe.
type fakeGo struct {
	mu    sync.Mutex
	calls []string
	last  map[string]map[string]interface{}
	db    *gorm.DB
}

func (f *fakeGo) note(c *gin.Context) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var b map[string]interface{}
	raw, _ := io.ReadAll(c.Request.Body)
	_ = json.Unmarshal(raw, &b)
	key := c.Request.Method + " " + c.Request.URL.Path
	f.calls = append(f.calls, key+" "+c.GetHeader("apikey"))
	f.last[key] = b
}

func (f *fakeGo) engine() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	ok := func(data interface{}) gin.HandlerFunc {
		return func(c *gin.Context) { f.note(c); c.JSON(200, gin.H{"message": "success", "data": data}) }
	}
	sent := map[string]interface{}{"Info": map[string]interface{}{"Chat": "573001112233@s.whatsapp.net", "IsFromMe": true, "ID": "3EB0AA", "Timestamp": "2026-11-25T15:00:00Z"},
		"Message": map[string]interface{}{"conversation": "hola"}}
	r.POST("/send/text", ok(sent))
	r.POST("/send/media", ok(sent))
	r.POST("/send/contact", ok(sent))
	r.GET("/instance/status", ok(map[string]bool{"Connected": true, "LoggedIn": true}))
	r.POST("/group/create", ok("go-route"))
	r.DELETE("/instance/logout", ok("go-logout"))
	r.DELETE("/instance/delete/:id", ok("go-delete"))
	r.POST("/message/markread", ok(nil))
	r.POST("/user/check", ok(map[string]interface{}{"Users": []map[string]interface{}{{"Query": "573001112233", "IsInWhatsapp": true, "JID": "573001112233@s.whatsapp.net"}}}))
	r.POST("/instance/create", func(c *gin.Context) {
		f.note(c)
		f.mu.Lock()
		b := f.last["POST /instance/create"]
		f.mu.Unlock()
		name, _ := b["name"].(string)
		token, _ := b["token"].(string)
		if name == "" {
			c.JSON(400, gin.H{"error": "name is required"})
			return
		}
		inst := instance_model.Instance{Id: uuid.NewString(), Name: name, Token: token}
		if f.db != nil {
			f.db.Create(&inst)
		}
		c.JSON(200, gin.H{"message": "success", "data": inst})
	})
	r.GET("/manager", func(c *gin.Context) { c.String(200, "manager") })
	return r
}

func newFake(db *gorm.DB) *fakeGo { return &fakeGo{last: map[string]map[string]interface{}{}, db: db} }

func do(h http.Handler, method, path, key string, body interface{}) (*httptest.ResponseRecorder, map[string]interface{}) {
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, rd)
	if key != "" {
		req.Header.Set("apikey", key)
	}
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	var out map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w, out
}

// Las rutas Go que comparten prefijo con v2 siguen llegando al router Go.
func TestPassThroughGoRoutes(t *testing.T) {
	f := newFake(nil)
	s := New(Options{GlobalKey: "global"}, f.engine())
	for _, tc := range []struct{ method, path, body string }{
		{http.MethodPost, "/group/create", "go-route"},
		{http.MethodDelete, "/instance/logout", "go-logout"},
	} {
		w, out := do(s, tc.method, tc.path, "tok", map[string]interface{}{"groupName": "x"})
		if w.Code != 200 || out["data"] != tc.body {
			t.Fatalf("%s %s: %d %s", tc.method, tc.path, w.Code, w.Body.String())
		}
	}
	if w, _ := do(s, http.MethodGet, "/manager", "", nil); w.Body.String() != "manager" {
		t.Fatalf("manager: %s", w.Body.String())
	}
	// POST /instance/create con el cuerpo de Go ({name, token}) va al router Go.
	w, out := do(s, http.MethodPost, "/instance/create", "global", map[string]interface{}{"name": "a", "token": "t"})
	if w.Code != 200 || out["message"] != "success" {
		t.Fatalf("go create: %d %s", w.Code, w.Body.String())
	}
}

func pgDB(t *testing.T) *gorm.DB {
	dsn := os.Getenv("COMPAT_TEST_DSN")
	if dsn == "" {
		t.Skip("COMPAT_TEST_DSN not set")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&instance_model.Instance{}, &webhooks.Config{}, &webhooks.Delivery{}); err != nil {
		t.Fatal(err)
	}
	db.Exec("DELETE FROM instances")
	db.Exec("DELETE FROM webhook_configs")
	return db
}

func TestV2Flow(t *testing.T) {
	db := pgDB(t)
	f := newFake(db)
	wh := webhooks.New(db, webhooks.Settings{MaxAttempts: 1})
	s := New(Options{DB: db, GlobalKey: "global", Webhooks: wh}, f.engine())

	// Crear con la forma v2: 201, token propio y webhook en formato v2.
	w, out := do(s, http.MethodPost, "/instance/create", "global", map[string]interface{}{
		"instanceName": "ventas", "token": "tok-ventas", "integration": "WHATSAPP-BAILEYS", "rejectCall": true,
		"webhook": map[string]interface{}{"url": "https://hook.test/x", "byEvents": true, "events": []string{"MESSAGES_UPSERT", "CONNECTION_UPDATE"}},
	})
	if w.Code != http.StatusCreated || out["hash"] != "tok-ventas" {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	if adv := f.last["POST /instance/create"]["advancedSettings"].(map[string]interface{}); adv["rejectCall"] != true {
		t.Fatalf("settings not forwarded: %v", adv)
	}
	var inst instance_model.Instance
	db.First(&inst, "name = ?", "ventas")
	if inst.Webhook != "https://hook.test/x" || inst.Events != "MESSAGE,CONNECTION" {
		t.Fatalf("webhook not applied: %q %q", inst.Webhook, inst.Events)
	}
	if cfg := wh.Get(inst.Id); cfg == nil || cfg.Format != webhooks.FormatEvolutionAPI || !cfg.ByEvents {
		t.Fatalf("webhook config: %+v", cfg)
	}
	if w, _ := do(s, http.MethodPost, "/instance/create", "global", map[string]interface{}{"instanceName": "ventas"}); w.Code != http.StatusForbidden {
		t.Fatalf("duplicate name: %d", w.Code)
	}

	// Autenticación: clave global o token de la línea; otra clave no.
	db.Create(&instance_model.Instance{Id: uuid.NewString(), Name: "otra", Token: "tok-otra"})
	if w, _ := do(s, http.MethodPost, "/message/sendText/ventas", "tok-otra", map[string]interface{}{"number": "1", "text": "x"}); w.Code != 401 {
		t.Fatalf("foreign token: %d", w.Code)
	}
	if w, _ := do(s, http.MethodPost, "/message/sendText/nadie", "global", map[string]interface{}{"number": "1", "text": "x"}); w.Code != 404 {
		t.Fatalf("unknown instance: %d", w.Code)
	}

	// sendText: traduce el cuerpo, llama con el token de la línea y responde como v2.
	w, out = do(s, http.MethodPost, "/message/sendText/ventas", "global", map[string]interface{}{
		"number": "573001112233", "text": "hola", "delay": 1200, "mentioned": []string{"573009998877"},
		"quoted": map[string]interface{}{"key": map[string]interface{}{"id": "Q1", "remoteJid": "573001112233@s.whatsapp.net", "fromMe": false}},
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("sendText: %d %s", w.Code, w.Body.String())
	}
	key := out["key"].(map[string]interface{})
	if key["id"] != "3EB0AA" || out["status"] != "PENDING" || out["messageType"] != "conversation" {
		t.Fatalf("sendText response: %v", out)
	}
	b := f.last["POST /send/text"]
	q := b["quoted"].(map[string]interface{})
	if b["text"] != "hola" || b["delay"].(float64) != 1200 || q["messageId"] != "Q1" || q["participant"] != "573001112233@s.whatsapp.net" ||
		b["mentionedJid"].([]interface{})[0] != "573009998877@s.whatsapp.net" {
		t.Fatalf("forwarded body: %v", b)
	}
	if last := f.calls[len(f.calls)-1]; !strings.HasSuffix(last, " tok-ventas") {
		t.Fatalf("must call Go with the instance token: %s", last)
	}

	// sendMedia acepta data URI.
	if w, _ := do(s, http.MethodPost, "/message/sendMedia/ventas", "tok-ventas", map[string]interface{}{
		"number": "573001112233", "mediatype": "image", "media": "data:image/png;base64,QUJD", "caption": "c"}); w.Code != 201 {
		t.Fatalf("sendMedia: %d", w.Code)
	}
	if b := f.last["POST /send/media"]; b["url"] != "QUJD" || b["type"] != "image" {
		t.Fatalf("media body: %v", b)
	}

	// Varios contactos van en un solo envío (vcards).
	if w, _ := do(s, http.MethodPost, "/message/sendContact/ventas", "global", map[string]interface{}{"number": "573001112233",
		"contact": []map[string]interface{}{{"fullName": "Ana", "wuid": "573001", "phoneNumber": "+57 300 1"}, {"fullName": "Luis", "phoneNumber": "573002"}}}); w.Code != 201 {
		t.Fatalf("sendContact: %d", w.Code)
	}
	if v := f.last["POST /send/contact"]["vcards"].([]interface{}); len(v) != 2 || v[0].(map[string]interface{})["phone"] != "573001" {
		t.Fatalf("vcards: %v", v)
	}

	// Estado, listado y números.
	if _, out := do(s, http.MethodGet, "/instance/connectionState/ventas", "tok-ventas", nil); out["instance"].(map[string]interface{})["state"] != "open" {
		t.Fatalf("state: %v", out)
	}
	w = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/instance/fetchInstances", nil)
	req.Header.Set("apikey", "tok-ventas")
	s.ServeHTTP(w, req)
	var list []map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &list)
	if len(list) != 1 || list[0]["name"] != "ventas" || list[0]["Setting"] == nil {
		t.Fatalf("fetchInstances with token: %s", w.Body.String())
	}
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/chat/whatsappNumbers/ventas", strings.NewReader(`{"numbers":["573001112233"]}`))
	req.Header.Set("apikey", "global")
	s.ServeHTTP(w, req)
	if !strings.Contains(w.Body.String(), `"exists":true`) {
		t.Fatalf("whatsappNumbers: %s", w.Body.String())
	}

	// markMessageAsRead agrupa por chat y omite los propios.
	do(s, http.MethodPost, "/chat/markMessageAsRead/ventas", "global", map[string]interface{}{"readMessages": []map[string]interface{}{
		{"remoteJid": "a@s.whatsapp.net", "id": "1"}, {"remoteJid": "a@s.whatsapp.net", "id": "2"}, {"remoteJid": "a@s.whatsapp.net", "id": "3", "fromMe": true}}})
	if ids := f.last["POST /message/markread"]["id"].([]interface{}); len(ids) != 2 {
		t.Fatalf("markread ids: %v", ids)
	}

	// webhook/find devuelve la configuración en forma v2.
	_, out = do(s, http.MethodGet, "/webhook/find/ventas", "tok-ventas", nil)
	if out["url"] != "https://hook.test/x" || out["webhookByEvents"] != true || len(out["events"].([]interface{})) != 2 {
		t.Fatalf("webhook/find: %v", out)
	}

	// DELETE /instance/delete/:id con la clave global es la ruta Go; con el nombre, la v2.
	_, out = do(s, http.MethodDelete, "/instance/delete/"+inst.Id, "global", nil)
	if out["data"] != "go-delete" {
		t.Fatalf("delete by id must reach Go: %v", out)
	}
	_, out = do(s, http.MethodDelete, "/instance/delete/ventas", "tok-ventas", nil)
	if out["status"] != "SUCCESS" {
		t.Fatalf("delete by name: %v", out)
	}
}
