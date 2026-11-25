package chatbot

import (
	"context"
	"encoding/json"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type fakeSender struct {
	mu   sync.Mutex
	sent []string
}

func (f *fakeSender) SendText(_, jid, text string) error {
	f.mu.Lock()
	f.sent = append(f.sent, jid+"|"+text)
	f.mu.Unlock()
	return nil
}
func (f *fakeSender) SendMedia(_, jid, url, _, _, _ string) error { return f.SendText("", jid, url) }
func (f *fakeSender) all() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.sent...)
}

type echoProvider struct{}

func (echoProvider) Reply(_ context.Context, _ *Bot, _ *Session, in Input) (*Output, error) {
	return &Output{Replies: []Reply{{Text: "eco: " + in.Text}}}, nil
}

// CHATBOT_TEST_DSN=postgres://... go test ./pkg/chatbot -run PG
func TestPGMotor(t *testing.T) {
	dsn := os.Getenv("CHATBOT_TEST_DSN")
	if dsn == "" {
		t.Skip("CHATBOT_TEST_DSN no definido")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	snd := &fakeSender{}
	e := NewEngine(db, snd, func(string) *whatsmeow.Client { return nil })
	e.providers[TypeHTTP] = echoProvider{}
	if err := e.Migrate(); err != nil {
		t.Fatal(err)
	}
	inst := "t-" + uuid.NewString()[:8]
	defer e.DeleteInstance(inst)

	cfg, _ := json.Marshal(HTTPConfig{URL: "http://x"})
	bot := Bot{ID: uuid.NewString(), InstanceID: inst, Type: TypeHTTP, Name: "eco", Enabled: true, TriggerType: TriggerKeyword, TriggerOperator: OpStartsWith,
		TriggerValue: "hola", StopOnFromMe: true, KeywordFinish: "salir", Config: cfg}
	if err := db.Create(&bot).Error; err != nil {
		t.Fatal(err)
	}

	peer := types.NewJID("573001112233", types.DefaultUserServer)
	msg := func(text string, fromMe bool, chat types.JID) *events.Message {
		return &events.Message{Info: types.MessageInfo{MessageSource: types.MessageSource{Chat: chat, Sender: chat, IsFromMe: fromMe, IsGroup: chat.Server == types.GroupServer},
			ID: uuid.NewString(), Timestamp: time.Now(), PushName: "Ana"}, Message: &waE2E.Message{Conversation: proto.String(text)}}
	}
	status := func() string {
		s := e.openSession(inst, peer.String())
		if s == nil {
			return "none"
		}
		return s.Status
	}

	e.handle(inst, msg("buenas", false, peer)) // no dispara
	if len(snd.all()) != 0 || status() != "none" {
		t.Fatalf("sin disparador no debe responder: %v %s", snd.all(), status())
	}
	e.handle(inst, msg("Hola, ¿precio?", false, peer)) // dispara y abre sesión
	if got := snd.all(); len(got) != 1 || got[0] != peer.String()+"|eco: Hola, ¿precio?" || status() != SessionOpen {
		t.Fatalf("disparo: %v %s", got, status())
	}
	e.handle(inst, msg("otra cosa", false, peer)) // sigue la sesión sin disparador
	if len(snd.all()) != 2 {
		t.Fatalf("sesión abierta debe responder: %v", snd.all())
	}
	e.handle(inst, msg("lo atiendo yo", true, peer)) // humano desde el teléfono → pausa
	if status() != SessionPaused {
		t.Fatalf("debió pausarse: %s", status())
	}
	e.handle(inst, msg("¿sigue ahí?", false, peer)) // pausada: no responde
	if len(snd.all()) != 2 {
		t.Fatalf("pausada no debe responder: %v", snd.all())
	}
	// Reanudar y salir con la palabra clave.
	db.Model(&Session{}).Where("instance_id = ?", inst).Update("status", SessionOpen)
	e.handle(inst, msg(" SALIR ", false, peer))
	if status() != "none" || len(snd.all()) != 2 {
		t.Fatalf("salir debe cerrar sin responder: %s %v", status(), snd.all())
	}

	// Grupos: ignorados salvo AllowGroups.
	grp := types.NewJID("120363000000000001", types.GroupServer)
	e.handle(inst, msg("hola grupo", false, grp))
	if len(snd.all()) != 2 {
		t.Fatalf("grupo no permitido: %v", snd.all())
	}
	db.Model(&Bot{}).Where("id = ?", bot.ID).Update("allow_groups", true)
	e.Invalidate(inst)
	e.handle(inst, msg("hola grupo", false, grp))
	if len(snd.all()) != 3 {
		t.Fatalf("grupo permitido: %v", snd.all())
	}

	// Agrupación: tres mensajes seguidos → una respuesta con los tres.
	db.Model(&Bot{}).Where("id = ?", bot.ID).Updates(map[string]interface{}{"debounce_ms": 300, "trigger_type": TriggerAll})
	e.Invalidate(inst)
	other := types.NewJID("573009998877", types.DefaultUserServer)
	for _, t := range []string{"uno", "dos", "tres"} {
		e.handle(inst, msg(t, false, other))
	}
	time.Sleep(700 * time.Millisecond)
	got := snd.all()
	if len(got) != 4 || got[3] != other.String()+"|eco: uno\ndos\ntres" {
		t.Fatalf("debounce: %q", got)
	}
}
