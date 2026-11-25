package evoformat

import (
	"encoding/json"
	"reflect"
	"testing"
)

func decode(t *testing.T, b []byte) M {
	t.Helper()
	var m M
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestConvertMessage(t *testing.T) {
	in := `{"event":"Message","instanceId":"i1","instanceName":"ventas","instanceToken":"tok",
	"data":{"Info":{"Chat":"123@g.us","Sender":"57300@s.whatsapp.net","IsFromMe":false,"IsGroup":true,"ID":"3EB0ABC","PushName":"Ana","Timestamp":"2026-11-25T10:00:00-05:00"},
	"Message":{"extendedTextMessage":{"text":"hola"},"messageContextInfo":{}}}}`
	out := Convert([]byte(in), Meta{ServerURL: "https://api", Sender: "57111@s.whatsapp.net", Destination: "https://hook"})
	if len(out) != 1 || out[0].Event != MessagesUpsert {
		t.Fatalf("got %+v", out)
	}
	m := decode(t, out[0].Payload)
	if Str(m, "event") != "messages.upsert" || Str(m, "instance") != "ventas" || Str(m, "apikey") != "tok" ||
		Str(m, "sender") != "57111@s.whatsapp.net" || Str(m, "server_url") != "https://api" || Str(m, "destination") != "https://hook" {
		t.Fatalf("envelope: %v", m)
	}
	d := Obj(m, "data")
	if Str(d, "key.remoteJid") != "123@g.us" || Str(d, "key.id") != "3EB0ABC" || Str(d, "key.participant") != "57300@s.whatsapp.net" ||
		Bool(d, "key.fromMe") || Str(d, "messageType") != "extendedTextMessage" || Str(d, "pushName") != "Ana" ||
		Str(d, "source") != "web" || Str(d, "status") != "DELIVERY_ACK" || Str(d, "message.extendedTextMessage.text") != "hola" {
		t.Fatalf("data: %v", d)
	}
	if ts, _ := d["messageTimestamp"].(float64); int64(ts) != 1795618800 {
		t.Fatalf("timestamp %v", d["messageTimestamp"])
	}
}

func TestConvertReceiptPerMessage(t *testing.T) {
	in := `{"event":"Receipt","state":"Read","instanceId":"i1","instanceName":"n","data":{"Chat":"57300@s.whatsapp.net","Type":"read","MessageIDs":["A","B"]}}`
	out := Convert([]byte(in), Meta{})
	if len(out) != 2 {
		t.Fatalf("want 2 events, got %d", len(out))
	}
	d := Obj(decode(t, out[1].Payload), "data")
	if Str(d, "keyId") != "B" || Str(d, "status") != "READ" || Str(d, "remoteJid") != "57300@s.whatsapp.net" {
		t.Fatalf("data: %v", d)
	}
}

func TestConvertConnectionAndQR(t *testing.T) {
	qr := Convert([]byte(`{"event":"QRCode","instanceName":"n","data":{"qrcode":"data:image/png;base64,xx","code":"2@abc"}}`), Meta{})
	if len(qr) != 1 || Str(decode(t, qr[0].Payload), "data.qrcode.code") != "2@abc" {
		t.Fatalf("qr: %+v", qr)
	}
	c := Convert([]byte(`{"event":"LoggedOut","instanceName":"n","data":{}}`), Meta{})
	d := Obj(decode(t, c[0].Payload), "data")
	if Str(d, "state") != "close" || d["statusReason"].(float64) != 401 {
		t.Fatalf("logout: %v", d)
	}
	if got := Convert([]byte(`{"event":"PasskeyRequest","data":{}}`), Meta{}); got != nil {
		t.Fatalf("unknown events must be dropped: %+v", got)
	}
}

func TestNames(t *testing.T) {
	if Upper("group-participants.update") != "GROUP_PARTICIPANTS_UPDATE" || Path("messages.upsert") != "messages-upsert" {
		t.Fatal("names")
	}
	if g := GroupsFor([]string{"MESSAGES_UPSERT", "SEND_MESSAGE", "PRESENCE_UPDATE", "messages.upsert"}); !reflect.DeepEqual(g, []string{"MESSAGE", "SEND_MESSAGE", "PRESENCE", "CHAT_PRESENCE"}) {
		t.Fatalf("groups %v", g)
	}
	if g := GroupsFor(nil); !reflect.DeepEqual(g, []string{"ALL"}) {
		t.Fatalf("empty %v", g)
	}
	if e := EventsFor([]string{"MESSAGE", "CONNECTION"}); !reflect.DeepEqual(e, []string{"MESSAGES_UPSERT", "CONNECTION_UPDATE"}) {
		t.Fatalf("events %v", e)
	}
	if MessageType(M{"messageContextInfo": M{}, "imageMessage": M{}}) != "imageMessage" || MessageType(M{"conversation": "x", "zzz": 1}) != "conversation" {
		t.Fatal("message type")
	}
}

func TestFromSend(t *testing.T) {
	r := FromSend(M{"Info": M{"Chat": "57300@s.whatsapp.net", "IsFromMe": true, "ID": "3EB0X", "Timestamp": "2026-11-25T15:00:00Z"},
		"Message": M{"conversation": "hi"}}, "i1")
	if Str(r, "key.remoteJid") != "57300@s.whatsapp.net" || !Bool(r, "key.fromMe") || Str(r, "status") != "PENDING" || Str(r, "messageType") != "conversation" {
		t.Fatalf("%v", r)
	}
	if _, has := Obj(r, "key")["participant"]; has {
		t.Fatal("no participant outside groups")
	}
}
