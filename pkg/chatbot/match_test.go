package chatbot

import (
	"testing"
	"time"
)

func TestMatches(t *testing.T) {
	cases := []struct {
		typ, op, val, text string
		want               bool
	}{
		{"all", "", "", "hola", true},
		{"", "", "", "hola", true},
		{"keyword", "contains", "Precio", "¿cuál es el precio?", true},
		{"keyword", "equals", "menu", " MENU ", true},
		{"keyword", "equals", "menu", "menu por favor", false},
		{"keyword", "startsWith", "hola", "Hola, buenas", true},
		{"keyword", "endsWith", "gracias", "muchas gracias", true},
		{"keyword", "regex", `^pedido\s+\d+$`, "pedido 4821", true},
		{"keyword", "regex", `(`, "x", false},
		{"keyword", "contains", "", "x", false},
	}
	for _, c := range cases {
		b := &Bot{TriggerType: c.typ, TriggerOperator: c.op, TriggerValue: c.val}
		if got := Matches(b, c.text); got != c.want {
			t.Errorf("%s/%s/%q con %q = %v", c.typ, c.op, c.val, c.text, got)
		}
	}
}

func TestIgnoredExpiredFinishSplit(t *testing.T) {
	b := &Bot{IgnoreJIDs: []string{"+573001112233", "1203@g.us"}, ExpireMinutes: 10, KeywordFinish: "salir"}
	if !Ignored(b, "573001112233@s.whatsapp.net") || !Ignored(b, "1203@g.us") || Ignored(b, "573009998877@s.whatsapp.net") {
		t.Fatal("ignorados")
	}
	now := time.Now()
	if !Expired(b, &Session{LastActivity: now.Add(-11 * time.Minute)}, now) || Expired(b, &Session{LastActivity: now.Add(-time.Minute)}, now) {
		t.Fatal("caducidad")
	}
	if !IsFinish(b, " SALIR ") || IsFinish(b, "salir ya") {
		t.Fatal("palabra de salida")
	}
	if p := Split("uno\n\ndos\n \n tres"); len(p) != 3 || p[2] != "tres" {
		t.Fatalf("split = %q", p)
	}
}

func TestIgnoredWildcards(t *testing.T) {
	b := &Bot{IgnoreJIDs: []string{"@g.us"}}
	if !Ignored(b, "120363000@g.us") || Ignored(b, "57300@s.whatsapp.net") {
		t.Fatal("@g.us must ignore only groups")
	}
	b.IgnoreJIDs = []string{"@s.whatsapp.net"}
	if Ignored(b, "120363000@g.us") || !Ignored(b, "57300@s.whatsapp.net") {
		t.Fatal("@s.whatsapp.net must ignore only private chats")
	}
}
