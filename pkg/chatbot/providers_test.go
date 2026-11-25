package chatbot

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func botWith(typ string, cfg interface{}) *Bot {
	b, _ := json.Marshal(cfg)
	return &Bot{ID: "b1", Type: typ, Config: b}
}

var in = Input{InstanceID: "i1", RemoteJID: "573001112233@s.whatsapp.net", PushName: "Ana", Text: "hola", Type: "text"}

func TestParseHTTPReply(t *testing.T) {
	cases := map[string][]string{
		`texto plano`:                         {"texto plano"},
		`{"message":"hola"}`:                  {"hola"},
		`[{"output":"uno"},{"output":"dos"}]`: {"uno", "dos"},
		`{"messages":["a","b"]}`:              {"a", "b"},
		`{"data":{"answer":"x"}}`:             {"x"},
		`{"mediaUrl":"https://x/y.jpg","mediaType":"image","caption":"c"}`: {"c"},
	}
	for body, want := range cases {
		got := ParseHTTPReply([]byte(body))
		if len(got) != len(want) {
			t.Fatalf("%s → %+v", body, got)
		}
		for i := range want {
			if got[i].Text != want[i] {
				t.Fatalf("%s → %+v", body, got)
			}
		}
	}
}

func TestN8NEnviaYAutentica(t *testing.T) {
	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, _ := r.BasicAuth()
		if u != "user" || p != "pass" {
			w.WriteHeader(401)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = w.Write([]byte(`[{"output":"respuesta"}]`))
	}))
	defer srv.Close()
	out, err := httpProvider{n8n: true}.Reply(context.Background(), botWith(TypeN8N, HTTPConfig{URL: srv.URL, Username: "user", Password: "pass"}), &Session{ID: "s1"}, in)
	if err != nil || len(out.Replies) != 1 || out.Replies[0].Text != "respuesta" {
		t.Fatalf("out=%+v err=%v", out, err)
	}
	if gotBody["message"] != "hola" || gotBody["remoteJid"] != in.RemoteJID || gotBody["sessionId"] != "s1" {
		t.Fatalf("body=%v", gotBody)
	}
}

func TestOpenAIHistorial(t *testing.T) {
	var last struct {
		Model    string    `json:"model"`
		Messages []chatMsg `json:"messages"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sk-x" || r.URL.Path != "/v1/chat/completions" {
			w.WriteHeader(401)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&last)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"¡Hola Ana!"}}]}`))
	}))
	defer srv.Close()
	b := botWith(TypeOpenAI, OpenAIConfig{BaseURL: srv.URL + "/v1", APIKey: "sk-x", Model: "m1", SystemPrompt: "Eres amable", HistoryLimit: 1})
	s := &Session{}
	out, err := openAIProvider{}.Reply(context.Background(), b, s, in)
	if err != nil || out.Replies[0].Text != "¡Hola Ana!" {
		t.Fatalf("out=%+v err=%v", out, err)
	}
	if last.Model != "m1" || last.Messages[0].Role != "system" || last.Messages[1].Content != "hola" {
		t.Fatalf("petición=%+v", last)
	}
	// Segundo turno: el historial se recorta a HistoryLimit (1 intercambio = 2 mensajes).
	s.Context = out.Context
	in2 := in
	in2.Text = "¿y el precio?"
	if _, err := (openAIProvider{}).Reply(context.Background(), b, s, in2); err != nil {
		t.Fatal(err)
	}
	if len(last.Messages) != 3 || last.Messages[1].Role != "assistant" || last.Messages[2].Content != "¿y el precio?" {
		t.Fatalf("historial=%+v", last.Messages)
	}
}

func TestTypebotIniciaContinuaYTermina(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/api/v1/typebots/pub1/startChat"):
			_, _ = w.Write([]byte(`{"sessionId":"S1","messages":[{"type":"text","content":{"richText":[{"type":"p","children":[{"text":"Hola "},{"text":"Ana","bold":true}]}]}},{"type":"image","content":{"url":"https://x/a.png"}}],"input":{"type":"choice input","items":[{"content":"Ventas"},{"content":"Soporte"}]}}`))
		case strings.HasSuffix(r.URL.Path, "/api/v1/sessions/S1/continueChat"):
			_, _ = w.Write([]byte(`{"messages":[{"type":"text","content":{"richText":[{"type":"p","children":[{"text":"Gracias"}]}]}}]}`))
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()
	b := botWith(TypeTypebot, TypebotConfig{URL: srv.URL, PublicID: "pub1"})
	out, err := typebotProvider{}.Reply(context.Background(), b, &Session{}, in)
	if err != nil || out.ExternalID != "S1" || out.End || len(out.Replies) != 3 {
		t.Fatalf("inicio out=%+v err=%v", out, err)
	}
	if out.Replies[0].Text != "Hola *Ana*" || out.Replies[1].MediaURL != "https://x/a.png" || out.Replies[2].Text != "1. Ventas\n2. Soporte" {
		t.Fatalf("respuestas=%+v", out.Replies)
	}
	out, err = typebotProvider{}.Reply(context.Background(), b, &Session{ExternalID: "S1"}, in)
	if err != nil || !out.End || out.Replies[0].Text != "Gracias" || out.ExternalID != "S1" {
		t.Fatalf("continuar out=%+v err=%v", out, err)
	}
}

func TestDifyChatAgenteYWorkflow(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		switch r.URL.Path {
		case "/v1/chat-messages":
			if body["response_mode"] == "streaming" {
				_, _ = io.WriteString(w, "data: {\"event\":\"agent_message\",\"answer\":\"Hola \",\"conversation_id\":\"C2\"}\n\ndata: {\"event\":\"agent_message\",\"answer\":\"desde agente\"}\n\ndata: {\"event\":\"message_end\"}\n\n")
				return
			}
			_, _ = w.Write([]byte(`{"answer":"hola chat","conversation_id":"C1"}`))
		case "/v1/workflows/run":
			_, _ = w.Write([]byte(`{"data":{"outputs":{"text":"salida"}}}`))
		}
	}))
	defer srv.Close()
	for typ, want := range map[string][2]string{"chat": {"hola chat", "C1"}, "agent": {"Hola desde agente", "C2"}, "workflow": {"salida", ""}} {
		out, err := difyProvider{}.Reply(context.Background(), botWith(TypeDify, DifyConfig{APIURL: srv.URL + "/v1", APIKey: "k", BotType: typ}), &Session{}, in)
		if err != nil || out.Replies[0].Text != want[0] || out.ExternalID != want[1] {
			t.Fatalf("%s: out=%+v err=%v", typ, out, err)
		}
	}
}

func TestFlowise(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"text":"respuesta flowise"}`))
	}))
	defer srv.Close()
	out, err := flowiseProvider{}.Reply(context.Background(), botWith(TypeFlowise, FlowiseConfig{APIURL: srv.URL}), &Session{ID: "s"}, in)
	if err != nil || out.Replies[0].Text != "respuesta flowise" {
		t.Fatalf("out=%+v err=%v", out, err)
	}
}

func TestHTTPWantsEnd(t *testing.T) {
	if !httpWantsEnd([]byte(`{"message":"bye","end":true}`)) {
		t.Fatal("expected end")
	}
	if httpWantsEnd([]byte(`{"message":"hi"}`)) || httpWantsEnd([]byte(`[{"output":"x"}]`)) || httpWantsEnd([]byte("plain")) {
		t.Fatal("unexpected end")
	}
}
