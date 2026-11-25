package chatstore

import (
	"context"
	"os"
	"testing"
	"time"

	"go.mau.fi/whatsmeow/proto/waCommon"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/proto/waSyncAction"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Prueba de integración: CHATSTORE_TEST_DSN=postgres://... go test ./pkg/chatstore -run PG
func TestPGFlujoCompleto(t *testing.T) {
	dsn := os.Getenv("CHATSTORE_TEST_DSN")
	if dsn == "" {
		t.Skip("CHATSTORE_TEST_DSN no definido")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	s := New(db, Config{Enabled: true, History: true, Raw: true})
	if err := s.Migrate(); err != nil {
		t.Fatal(err)
	}
	inst := "test-" + time.Now().Format("150405.000")
	defer s.DeleteInstance(inst)

	peer := types.NewJID("573001112233", types.DefaultUserServer)
	me := types.NewJID("573009998877", types.DefaultUserServer)
	t0 := time.Now().Add(-time.Hour)

	msg := func(id, text string, fromMe bool, at time.Time) *events.Message {
		src := types.MessageSource{Chat: peer, Sender: peer, IsFromMe: fromMe}
		if fromMe {
			src.Sender = me
		}
		return &events.Message{Info: types.MessageInfo{MessageSource: src, ID: id, Timestamp: at, PushName: "Ana"}, Message: &waE2E.Message{Conversation: proto.String(text)}}
	}

	// Dos entrantes: 2 sin leer; repetir un mensaje no duplica.
	s.saveMessage(inst, msg("A1", "hola", false, t0), false)
	s.saveMessage(inst, msg("A2", "¿hay stock?", false, t0.Add(time.Minute)), false)
	s.saveMessage(inst, msg("A2", "¿hay stock?", false, t0.Add(time.Minute)), false)

	chats, err := s.ListChats(inst, "", false, 50, nil)
	if err != nil || len(chats) != 1 {
		t.Fatalf("chats = %+v, err %v", chats, err)
	}
	if chats[0].UnreadCount != 2 || chats[0].LastPreview != "¿hay stock?" || chats[0].DisplayName != "Ana" {
		t.Fatalf("chat = %+v", chats[0])
	}

	// Un mensaje viejo (historial) no cambia el último mensaje.
	s.saveMessage(inst, msg("OLD", "antiguo", false, t0.Add(-24*time.Hour)), true)
	chats, _ = s.ListChats(inst, "", false, 50, nil)
	if chats[0].LastMessageID != "A2" {
		t.Fatalf("el historial pisó el último mensaje: %s", chats[0].LastMessageID)
	}

	// Respuesta propia: no leídos a 0 y estado sube con los acuses, sin bajar.
	s.saveMessage(inst, msg("B1", "sí", true, t0.Add(2*time.Minute)), false)
	s.receipt(inst, &events.Receipt{MessageSource: types.MessageSource{Chat: peer, Sender: peer}, MessageIDs: []types.MessageID{"B1"}, Type: types.ReceiptTypeRead})
	s.receipt(inst, &events.Receipt{MessageSource: types.MessageSource{Chat: peer, Sender: peer}, MessageIDs: []types.MessageID{"B1"}, Type: types.ReceiptTypeDelivered})
	chats, _ = s.ListChats(inst, "", false, 50, nil)
	if chats[0].UnreadCount != 0 || !chats[0].LastFromMe {
		t.Fatalf("tras responder: %+v", chats[0])
	}
	m, err := s.GetMessage(inst, "B1")
	if err != nil || m.Status != StatusRead {
		t.Fatalf("estado = %v, err %v", m, err)
	}

	// Edición y borrado para todos.
	edit := &events.Message{Info: types.MessageInfo{MessageSource: types.MessageSource{Chat: peer, Sender: peer}, ID: "E1", Timestamp: time.Now()},
		Message: &waE2E.Message{ProtocolMessage: &waE2E.ProtocolMessage{Type: waE2E.ProtocolMessage_MESSAGE_EDIT.Enum(), Key: &waCommon.MessageKey{ID: proto.String("A1")}, EditedMessage: &waE2E.Message{Conversation: proto.String("hola!")}}}}
	s.saveMessage(inst, edit, false)
	revoke := &events.Message{Info: types.MessageInfo{MessageSource: types.MessageSource{Chat: peer, Sender: peer}, ID: "R1", Timestamp: time.Now()},
		Message: &waE2E.Message{ProtocolMessage: &waE2E.ProtocolMessage{Type: waE2E.ProtocolMessage_REVOKE.Enum(), Key: &waCommon.MessageKey{ID: proto.String("A2")}}}}
	s.saveMessage(inst, revoke, false)
	msgs, err := s.ListMessages(inst, peer.String(), 50, nil)
	if err != nil || len(msgs) != 4 {
		t.Fatalf("mensajes = %d, err %v", len(msgs), err)
	}
	for _, x := range msgs {
		if x.MessageID == "A1" && (!x.Edited || x.Text != "hola!") {
			t.Fatalf("edición no aplicada: %+v", x)
		}
		if x.MessageID == "A2" && (!x.Deleted || x.Text != "") {
			t.Fatalf("borrado no aplicado: %+v", x)
		}
	}
	if msgs[0].MessageID != "B1" {
		t.Fatalf("orden: primero %s", msgs[0].MessageID)
	}

	// Paginación hacia atrás.
	older, _ := s.ListMessages(inst, peer.String(), 50, &msgs[1].Timestamp)
	if len(older) != 2 {
		t.Fatalf("antes de %v: %d", msgs[1].Timestamp, len(older))
	}

	// MarkRead devuelve los entrantes sin leer y pone el contador a 0.
	s.saveMessage(inst, msg("C1", "gracias", false, t0.Add(3*time.Minute)), false)
	unread, err := s.MarkRead(inst, peer.String())
	if err != nil || len(unread) != 1 || unread[0].MessageID != "C1" {
		t.Fatalf("MarkRead = %+v, err %v", unread, err)
	}

	// LastOf da el último mensaje; «no leído» deja 1 pendiente y «leído» lo vuelve a 0.
	SetDefault(s)
	defer SetDefault(nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.Start(ctx)
	if last, ok := LastOf(inst, peer.String()); !ok || last.ID != "C1" {
		t.Fatalf("LastOf = %+v %v", last, ok)
	}
	unreadNow := func() int {
		cs, _ := s.GetChat(inst, peer.String())
		return cs[0].UnreadCount
	}
	waitUnread := func(want int) {
		t.Helper()
		for i := 0; i < 50 && unreadNow() != want; i++ {
			time.Sleep(20 * time.Millisecond)
		}
		if got := unreadNow(); got != want {
			t.Fatalf("unread = %d, want %d", got, want)
		}
	}
	mark := func(read bool) {
		Observe(inst, nil, &events.MarkChatAsRead{JID: peer, Action: &waSyncAction.MarkChatAsReadAction{Read: proto.Bool(read)}})
	}
	mark(false)
	waitUnread(1)
	mark(true)
	waitUnread(0)

	// Búsqueda y contactos.
	if found, _ := s.ListChats(inst, "ana", false, 50, nil); len(found) != 1 {
		t.Fatal("la búsqueda por nombre no encontró el chat")
	}
	if cs, _ := s.ListContacts(inst, "", 10); len(cs) != 1 || cs[0].PushName != "Ana" {
		t.Fatalf("contactos = %+v", cs)
	}

	// Borrar la instancia limpia todo.
	if err := s.DeleteInstance(inst); err != nil {
		t.Fatal(err)
	}
	if cs, _ := s.ListChats(inst, "", false, 50, nil); len(cs) != 0 {
		t.Fatal("quedaron chats tras borrar la instancia")
	}
}
