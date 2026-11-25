package websocket_producer

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/evolution-foundation/evolution-go/pkg/config"
	logger_wrapper "github.com/evolution-foundation/evolution-go/pkg/logger"
	"github.com/gorilla/websocket"
)

func dial(t *testing.T, srv *httptest.Server, instanceID string) *websocket.Conn {
	t.Helper()
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/?instanceId=" + instanceID
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	return conn
}

func waitClients(t *testing.T, p *websocketProducer, instanceID string, want int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		p.clientsMux.RLock()
		n := len(p.clients[instanceID])
		if instanceID == "" {
			n = len(p.broadcast)
		}
		p.clientsMux.RUnlock()
		if n == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("esperaba %d clientes para %q", want, instanceID)
}

func TestProduceLlegaATodasLasConexionesDeLaInstancia(t *testing.T) {
	p := NewWebsocketProducer(logger_wrapper.NewLoggerManager(&config.Config{LogDirectory: t.TempDir()}))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ServeWs(w, r, r.URL.Query().Get("instanceId"), p)
	}))
	defer srv.Close()

	a1 := dial(t, srv, "a")
	defer a1.Close()
	a2 := dial(t, srv, "a")
	defer a2.Close()
	b := dial(t, srv, "b")
	defer b.Close()
	all := dial(t, srv, "")
	defer all.Close()
	waitClients(t, p, "a", 2)
	waitClients(t, p, "", 1)

	if err := p.Produce("Message", []byte(`{"x":1}`), "a", ""); err != nil {
		t.Fatal(err)
	}

	for name, c := range map[string]*websocket.Conn{"a1": a1, "a2": a2, "broadcast": all} {
		_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
		var msg map[string]interface{}
		if err := c.ReadJSON(&msg); err != nil {
			t.Fatalf("%s no recibió el evento: %v", name, err)
		}
		if msg["queue"] != "message" {
			t.Fatalf("%s: queue = %v", name, msg["queue"])
		}
	}

	_ = b.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	var msg map[string]interface{}
	if err := b.ReadJSON(&msg); err == nil {
		t.Fatal("la instancia b no debía recibir eventos de a")
	}

	a1.Close()
	waitClients(t, p, "a", 1)
}
