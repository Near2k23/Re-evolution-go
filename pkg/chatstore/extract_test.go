package chatstore

import (
	"strings"
	"testing"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

func TestExtract(t *testing.T) {
	cases := []struct {
		name string
		msg  *waE2E.Message
		typ  string
		text string
		ok   bool
	}{
		{"texto", &waE2E.Message{Conversation: proto.String("hola")}, TypeText, "hola", true},
		{"texto extendido con cita", &waE2E.Message{ExtendedTextMessage: &waE2E.ExtendedTextMessage{Text: proto.String("ok"), ContextInfo: &waE2E.ContextInfo{StanzaID: proto.String("Q1")}}}, TypeText, "ok", true},
		{"imagen", &waE2E.Message{ImageMessage: &waE2E.ImageMessage{Caption: proto.String("foto"), Mimetype: proto.String("image/jpeg")}}, TypeImage, "foto", true},
		{"documento", &waE2E.Message{DocumentMessage: &waE2E.DocumentMessage{FileName: proto.String("a.pdf")}}, TypeDocument, "", true},
		{"audio", &waE2E.Message{AudioMessage: &waE2E.AudioMessage{PTT: proto.Bool(true)}}, TypeAudio, "", true},
		{"reacción", &waE2E.Message{ReactionMessage: &waE2E.ReactionMessage{Text: proto.String("👍")}}, "", "", false},
		{"protocolo", &waE2E.Message{ProtocolMessage: &waE2E.ProtocolMessage{}}, "", "", false},
		{"solo claves", &waE2E.Message{SenderKeyDistributionMessage: &waE2E.SenderKeyDistributionMessage{}, MessageContextInfo: &waE2E.MessageContextInfo{}}, "", "", false},
		{"nil", nil, "", "", false},
	}
	for _, c := range cases {
		got, ok := Extract(c.msg)
		if ok != c.ok || got.Type != c.typ || got.Text != c.text {
			t.Errorf("%s: got (%+v, %v)", c.name, got, ok)
		}
	}
	got, _ := Extract(&waE2E.Message{ExtendedTextMessage: &waE2E.ExtendedTextMessage{Text: proto.String("x"), ContextInfo: &waE2E.ContextInfo{StanzaID: proto.String("Q1")}}})
	if got.QuotedID != "Q1" {
		t.Errorf("QuotedID = %q", got.QuotedID)
	}
}

func TestPreview(t *testing.T) {
	if p := Preview(Content{Type: TypeDocument, FileName: "factura.pdf"}); p != "factura.pdf" {
		t.Fatalf("documento sin pie: %q", p)
	}
	long := strings.Repeat("a", 200)
	if p := Preview(Content{Type: TypeText, Text: long}); len([]rune(p)) != 141 {
		t.Fatalf("recorte: %d", len([]rune(p)))
	}
}

func TestChatJIDPrefiereNumero(t *testing.T) {
	lid := types.NewJID("123", types.HiddenUserServer)
	pn := types.NewJID("573001112233", types.DefaultUserServer)

	in := types.MessageSource{Chat: lid, Sender: lid, SenderAlt: pn}
	if got := ChatJID(in); got != pn {
		t.Fatalf("entrante por LID: %s", got)
	}
	out := types.MessageSource{Chat: lid, IsFromMe: true, RecipientAlt: pn}
	if got := ChatJID(out); got != pn {
		t.Fatalf("saliente por LID: %s", got)
	}
	grp := types.NewJID("1203", types.GroupServer)
	if got := ChatJID(types.MessageSource{Chat: grp, IsGroup: true, SenderAlt: pn}); got != grp {
		t.Fatalf("grupo: %s", got)
	}
	if got := ChatJID(types.MessageSource{Chat: lid}); got != lid {
		t.Fatalf("LID sin número conocido: %s", got)
	}
}

func TestSkipEstados(t *testing.T) {
	if !Skip(types.StatusBroadcastJID) {
		t.Fatal("status@broadcast debe omitirse")
	}
	if Skip(types.NewJID("573001112233", types.DefaultUserServer)) {
		t.Fatal("un chat normal no debe omitirse")
	}
}
