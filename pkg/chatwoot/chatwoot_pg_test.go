package chatwoot

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type fakeCW struct {
	mu            sync.Mutex
	inboxes       int
	webhook       string
	contacts      int
	conversations map[int]string // id → status
	messages      []map[string]interface{}
}

func (f *fakeCW) handler(t *testing.T) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if r.Header.Get("api_access_token") != "tok" {
			w.WriteHeader(401)
			return
		}
		p := strings.TrimPrefix(r.URL.Path, "/api/v1/accounts/7")
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		switch {
		case r.Method == "POST" && p == "/inboxes":
			f.inboxes++
			f.webhook = body["channel"].(map[string]interface{})["webhook_url"].(string)
			_, _ = w.Write([]byte(`{"id":55}`))
		case r.Method == "GET" && p == "/contacts/search":
			_, _ = w.Write([]byte(`{"payload":[]}`))
		case r.Method == "POST" && p == "/contacts":
			f.contacts++
			_, _ = w.Write([]byte(`{"payload":{"contact":{"id":900},"contact_inbox":{"source_id":"SRC1"}}}`))
		case r.Method == "POST" && p == "/conversations":
			id := 300 + len(f.conversations)
			f.conversations[id] = body["status"].(string)
			_ = json.NewEncoder(w).Encode(map[string]int{"id": id})
		case r.Method == "GET" && strings.HasPrefix(p, "/conversations/") && !strings.Contains(p, "/messages"):
			var id int
			_, _ = fmtSscan(strings.TrimPrefix(p, "/conversations/"), &id)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": f.conversations[id]})
		case r.Method == "POST" && strings.HasSuffix(p, "/toggle_status"):
			var id int
			_, _ = fmtSscan(strings.TrimSuffix(strings.TrimPrefix(p, "/conversations/"), "/toggle_status"), &id)
			f.conversations[id] = body["status"].(string)
			_, _ = w.Write([]byte(`{}`))
		case r.Method == "POST" && strings.HasSuffix(p, "/messages"):
			body["_path"] = p
			f.messages = append(f.messages, body)
			_, _ = w.Write([]byte(`{"id":1}`))
		default:
			t.Logf("ruta no simulada %s %s", r.Method, p)
			w.WriteHeader(404)
		}
	})
}

type recSender struct {
	mu   sync.Mutex
	sent []string
	svc  *Service
}

func (r *recSender) SendText(inst, jid, text string) error {
	r.mu.Lock()
	r.sent = append(r.sent, jid+"|"+text)
	r.mu.Unlock()
	// Como el servicio real: lo enviado pasa por ObserveSent.
	j, _ := types.ParseJID(jid)
	ObserveSent(inst, types.MessageInfo{MessageSource: types.MessageSource{Chat: j, IsFromMe: true}, ID: uuid.NewString(), Timestamp: time.Now()}, &waE2E.Message{Conversation: proto.String(text)})
	return nil
}
func (r *recSender) SendMedia(inst, jid, url, _, caption, _ string) error {
	return r.SendText(inst, jid, url+" "+caption)
}

// CHATWOOT_TEST_DSN=postgres://... go test ./pkg/chatwoot -run PG
func TestPGChatwoot(t *testing.T) {
	dsn := os.Getenv("CHATWOOT_TEST_DSN")
	if dsn == "" {
		t.Skip("CHATWOOT_TEST_DSN no definido")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	fake := &fakeCW{conversations: map[int]string{}}
	srv := httptest.NewServer(fake.handler(t))
	defer srv.Close()

	snd := &recSender{}
	s := New(db, snd, func(string) *whatsmeow.Client { return nil })
	snd.svc = s
	if err := s.Migrate(); err != nil {
		t.Fatal(err)
	}
	SetDefault(s)
	inst := "t-" + uuid.NewString()[:8]
	defer s.DeleteInstance(inst)

	c, err := s.Save(context.Background(), inst, "https://api.example.com", SaveInput{Enabled: true, URL: srv.URL, AccountID: 7, Token: "tok", SignMessages: true})
	if err != nil {
		t.Fatal(err)
	}
	if c.InboxID != 55 || fake.inboxes != 1 || !strings.HasPrefix(fake.webhook, "https://api.example.com/chatwoot/webhook/"+inst+"?secret=") {
		t.Fatalf("bandeja: %+v %s", c, fake.webhook)
	}
	// Guardar otra vez no crea otra bandeja y conserva el token.
	if c2, err := s.Save(context.Background(), inst, "https://api.example.com", SaveInput{Enabled: true, URL: srv.URL, AccountID: 7, InboxID: 55, SignMessages: true}); err != nil || c2.Token != "tok" || fake.inboxes != 1 {
		t.Fatalf("resave: %v %+v", err, c2)
	}

	peer := types.NewJID("573001112233", types.DefaultUserServer)
	info := func(id string, fromMe bool) types.MessageInfo {
		return types.MessageInfo{MessageSource: types.MessageSource{Chat: peer, Sender: peer, IsFromMe: fromMe}, ID: id, Timestamp: time.Now(), PushName: "Ana"}
	}
	txt := func(s string) *waE2E.Message { return &waE2E.Message{Conversation: proto.String(s)} }

	cfg := s.Get(inst)
	s.forward(cfg, inst, info("W1", false), txt("hola"), false)
	s.forward(cfg, inst, info("W2", false), txt("¿precio?"), false)
	if fake.contacts != 1 || len(fake.conversations) != 1 || len(fake.messages) != 2 {
		t.Fatalf("contactos %d conversaciones %d mensajes %d", fake.contacts, len(fake.conversations), len(fake.messages))
	}
	if fake.messages[0]["message_type"] != "incoming" || fake.messages[0]["source_id"] != "WAID:W1" || fake.messages[1]["content"] != "¿precio?" {
		t.Fatalf("mensajes = %v", fake.messages)
	}

	// Respuesta del agente → WhatsApp con firma; no vuelve a Chatwoot.
	ev := WebhookEvent{Event: "message_created", MessageType: "outgoing", Content: "Cuesta 10"}
	ev.Sender.AvailableName = "Luis"
	ev.Conversation.ID = 300
	if err := s.HandleWebhook(inst, "mal", ev); err != ErrUnauthorized {
		t.Fatalf("secreto incorrecto: %v", err)
	}
	if err := s.HandleWebhook(inst, cfg.WebhookSecret, ev); err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)
	if len(snd.sent) != 1 || snd.sent[0] != peer.String()+"|*Luis*:\nCuesta 10" {
		t.Fatalf("enviado = %v", snd.sent)
	}
	if len(fake.messages) != 2 {
		t.Fatalf("lo que vino de Chatwoot no debe volver: %d mensajes", len(fake.messages))
	}
	// Mensajes creados por este servidor (WAID:) no se reenvían.
	ev.SourceID = "WAID:W1"
	_ = s.HandleWebhook(inst, cfg.WebhookSecret, ev)
	if len(snd.sent) != 1 {
		t.Fatal("eco de WAID reenviado")
	}

	// Lo enviado por la API (bots, otros sistemas) aparece como saliente.
	ObserveSent(inst, types.MessageInfo{MessageSource: types.MessageSource{Chat: peer, IsFromMe: true}, ID: "API1", Timestamp: time.Now()}, txt("recordatorio"))
	time.Sleep(300 * time.Millisecond)
	fake.mu.Lock()
	last := fake.messages[len(fake.messages)-1]
	fake.mu.Unlock()
	if last["message_type"] != "outgoing" || last["content"] != "recordatorio" {
		t.Fatalf("saliente por API = %v", last)
	}

	// Conversación resuelta: sin reabrir → conversación nueva; con reabrir → misma.
	fake.mu.Lock()
	fake.conversations[300] = "resolved"
	fake.mu.Unlock()
	s.forward(cfg, inst, info("W3", false), txt("otra vez"), false)
	if len(fake.conversations) != 2 {
		t.Fatalf("debió crear conversación nueva: %v", fake.conversations)
	}
	s.Save(context.Background(), inst, "https://api.example.com", SaveInput{Enabled: true, URL: srv.URL, AccountID: 7, InboxID: 55, ReopenConversation: true})
	cfg = s.Get(inst)
	fake.mu.Lock()
	fake.conversations[301] = "resolved"
	fake.mu.Unlock()
	s.forward(cfg, inst, info("W4", false), txt("y ahora"), false)
	if len(fake.conversations) != 2 || fake.conversations[301] != "open" {
		t.Fatalf("debió reabrir: %v", fake.conversations)
	}
}
