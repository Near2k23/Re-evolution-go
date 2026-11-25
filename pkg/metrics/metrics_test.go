package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func TestWriteAndAuth(t *testing.T) {
	Observe("id-1", &events.Message{Info: types.MessageInfo{}})
	Observe("id-1", &events.Message{Info: types.MessageInfo{MessageSource: types.MessageSource{IsFromMe: true}}})
	ObserveSent("id-1")
	ObserveWebhook(true, 300*time.Millisecond)
	ObserveWebhook(false, 3*time.Second)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	Register(r, Options{Version: "1.2.3", GlobalKey: "global", User: "prom", Password: "pw", Lines: func() []Line {
		return []Line{{ID: "id-1", Name: "Ventas \"1\"", Connected: true, LoggedIn: true}, {ID: "id-2", Name: "B", LoggedIn: true}, {ID: "id-3", Name: "C"}}
	}})

	get := func(set func(*http.Request)) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		set(req)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}
	if w := get(func(*http.Request) {}); w.Code != 401 {
		t.Fatalf("no auth: %d", w.Code)
	}
	if w := get(func(q *http.Request) { q.SetBasicAuth("prom", "bad") }); w.Code != 401 {
		t.Fatalf("bad password: %d", w.Code)
	}
	if w := get(func(q *http.Request) { q.Header.Set("Authorization", "Bearer global") }); w.Code != 200 {
		t.Fatalf("bearer: %d", w.Code)
	}
	w := get(func(q *http.Request) { q.SetBasicAuth("prom", "pw") })
	body := w.Body.String()
	for _, want := range []string{
		`evolution_build_info{version="1.2.3"} 1`,
		`evolution_instances{state="open"} 1`,
		`evolution_instances{state="connecting"} 1`,
		`evolution_instances{state="close"} 1`,
		`evolution_instance_up{instance="Ventas \"1\"",instance_id="id-1"} 1`,
		`evolution_messages_received_total{instance="Ventas \"1\"",instance_id="id-1"} 1`,
		`evolution_messages_sent_total{instance="Ventas \"1\"",instance_id="id-1"} 1`,
		`evolution_webhook_deliveries_total{result="failed"} 1`,
		`evolution_webhook_delivery_seconds_bucket{le="0.5"} 1`,
		`evolution_webhook_delivery_seconds_bucket{le="+Inf"} 2`,
		"# TYPE evolution_webhook_delivery_seconds histogram",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q in:\n%s", want, body)
		}
	}
}
