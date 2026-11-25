package chatbot

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

var httpClient = &http.Client{Timeout: 80 * time.Second}

func defaultProviders(_ *Engine) map[string]Provider {
	return map[string]Provider{
		TypeHTTP:    httpProvider{},
		TypeN8N:     httpProvider{n8n: true},
		TypeOpenAI:  openAIProvider{},
		TypeTypebot: typebotProvider{},
		TypeDify:    difyProvider{},
		TypeFlowise: flowiseProvider{},
	}
}

// postJSON envía JSON y devuelve el cuerpo; error si no es 2xx.
func postJSON(ctx context.Context, url string, body interface{}, headers map[string]string) ([]byte, int, error) {
	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		if v != "" {
			req.Header.Set(k, v)
		}
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return data, resp.StatusCode, fmt.Errorf("%s: %s", resp.Status, truncate(string(data), 300))
	}
	return data, resp.StatusCode, nil
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}

func decodeConfig(b *Bot, v interface{}) {
	if len(b.Config) > 0 {
		_ = json.Unmarshal(b.Config, v)
	}
}

// ---------- HTTP genérico y n8n ----------

// HTTPConfig: URL que recibe el mensaje y devuelve la respuesta.
type HTTPConfig struct {
	URL      string `json:"url"`
	APIKey   string `json:"apiKey"`   // Authorization: Bearer …
	Username string `json:"username"` // n8n: autenticación básica
	Password string `json:"password"`
}

type httpProvider struct{ n8n bool }

func (p httpProvider) Reply(ctx context.Context, b *Bot, s *Session, in Input) (*Output, error) {
	var c HTTPConfig
	decodeConfig(b, &c)
	if c.URL == "" {
		return nil, errors.New("url is required")
	}
	body := map[string]interface{}{
		"instanceId":  in.InstanceID,
		"remoteJid":   in.RemoteJID,
		"pushName":    in.PushName,
		"message":     in.Text,
		"messageType": in.Type,
		"messageId":   in.MessageID,
		"fromMe":      in.FromMe,
		"isGroup":     in.IsGroup,
		"sessionId":   s.ID,
		"botId":       b.ID,
	}
	headers := map[string]string{}
	if c.APIKey != "" {
		headers["Authorization"] = "Bearer " + c.APIKey
	}
	req, _ := json.Marshal(body)
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URL, bytes.NewReader(req))
	if err != nil {
		return nil, err
	}
	r.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	if c.Username != "" {
		r.SetBasicAuth(c.Username, c.Password)
	}
	resp, err := httpClient.Do(r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%s: %s", resp.Status, truncate(string(data), 300))
	}
	return &Output{Replies: ParseHTTPReply(data)}, nil
}

// ParseHTTPReply entiende las respuestas habituales: texto plano; un objeto con
// message/output/reply/text/answer/response; un objeto con messages[]; o un
// arreglo de esos objetos (n8n devuelve [{"output": "…"}]).
func ParseHTTPReply(data []byte) []Reply {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return nil
	}
	var any interface{}
	if err := json.Unmarshal(data, &any); err != nil {
		return []Reply{{Text: string(data)}}
	}
	var out []Reply
	var walk func(v interface{})
	walk = func(v interface{}) {
		switch x := v.(type) {
		case string:
			if strings.TrimSpace(x) != "" {
				out = append(out, Reply{Text: x})
			}
		case []interface{}:
			for _, it := range x {
				walk(it)
			}
		case map[string]interface{}:
			if url, ok := x["mediaUrl"].(string); ok && url != "" {
				t, _ := x["mediaType"].(string)
				cap, _ := x["caption"].(string)
				out = append(out, Reply{MediaURL: url, MediaType: t, Text: cap})
				return
			}
			for _, k := range []string{"messages", "message", "output", "reply", "text", "answer", "response", "data"} {
				if v, ok := x[k]; ok {
					walk(v)
					return
				}
			}
		}
	}
	walk(any)
	return out
}

// ---------- OpenAI (y compatibles) ----------

type OpenAIConfig struct {
	BaseURL            string  `json:"baseUrl"`
	APIKey             string  `json:"apiKey"`
	Model              string  `json:"model"`
	SystemPrompt       string  `json:"systemPrompt"`
	MaxTokens          int     `json:"maxTokens"`
	Temperature        float64 `json:"temperature"`
	HistoryLimit       int     `json:"historyLimit"`
	Transcribe         bool    `json:"transcribe"`
	TranscriptionModel string  `json:"transcriptionModel"`
}

func openAIConfigOf(b *Bot) OpenAIConfig {
	c := OpenAIConfig{BaseURL: "https://api.openai.com/v1", Model: "gpt-4o-mini", HistoryLimit: 10, TranscriptionModel: "whisper-1", Temperature: 0.7}
	decodeConfig(b, &c)
	if c.BaseURL == "" {
		c.BaseURL = "https://api.openai.com/v1"
	}
	c.BaseURL = strings.TrimRight(c.BaseURL, "/")
	if c.HistoryLimit <= 0 {
		c.HistoryLimit = 10
	}
	if c.TranscriptionModel == "" {
		c.TranscriptionModel = "whisper-1"
	}
	return c
}

type chatMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIProvider struct{}

func (openAIProvider) Reply(ctx context.Context, b *Bot, s *Session, in Input) (*Output, error) {
	c := openAIConfigOf(b)
	if c.APIKey == "" {
		return nil, errors.New("apiKey is required")
	}
	text := in.Text
	if len(in.Audio) > 0 && c.Transcribe {
		t, err := transcribe(ctx, c, in.Audio, in.AudioMime)
		if err != nil {
			return nil, fmt.Errorf("transcription: %w", err)
		}
		text = strings.TrimSpace(strings.Join([]string{text, t}, "\n"))
	}
	if text == "" {
		return &Output{}, nil
	}
	var history []chatMsg
	_ = json.Unmarshal(s.Context, &history)
	history = append(history, chatMsg{Role: "user", Content: text})
	if max := c.HistoryLimit * 2; len(history) > max {
		history = history[len(history)-max:]
	}
	msgs := history
	if c.SystemPrompt != "" {
		msgs = append([]chatMsg{{Role: "system", Content: c.SystemPrompt}}, history...)
	}
	body := map[string]interface{}{"model": c.Model, "messages": msgs, "temperature": c.Temperature}
	if c.MaxTokens > 0 {
		body["max_tokens"] = c.MaxTokens
	}
	data, _, err := postJSON(ctx, c.BaseURL+"/chat/completions", body, map[string]string{"Authorization": "Bearer " + c.APIKey})
	if err != nil {
		return nil, err
	}
	var res struct {
		Choices []struct {
			Message chatMsg `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(data, &res); err != nil || len(res.Choices) == 0 {
		return nil, fmt.Errorf("unexpected response: %s", truncate(string(data), 200))
	}
	answer := strings.TrimSpace(res.Choices[0].Message.Content)
	history = append(history, chatMsg{Role: "assistant", Content: answer})
	ctxJSON, _ := json.Marshal(history)
	return &Output{Replies: []Reply{{Text: answer}}, Context: ctxJSON}, nil
}

func transcribe(ctx context.Context, c OpenAIConfig, audio []byte, mime string) (string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	ext := "ogg"
	if strings.Contains(mime, "mpeg") || strings.Contains(mime, "mp3") {
		ext = "mp3"
	} else if strings.Contains(mime, "mp4") || strings.Contains(mime, "m4a") {
		ext = "m4a"
	}
	fw, _ := w.CreateFormFile("file", "audio."+ext)
	_, _ = fw.Write(audio)
	_ = w.WriteField("model", c.TranscriptionModel)
	_ = w.Close()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/audio/transcriptions", &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("%s: %s", resp.Status, truncate(string(data), 200))
	}
	var r struct {
		Text string `json:"text"`
	}
	_ = json.Unmarshal(data, &r)
	return r.Text, nil
}

// ---------- Typebot ----------

type TypebotConfig struct {
	URL      string `json:"url"`      // https://typebot.io o tu instalación
	PublicID string `json:"publicId"` // id público del typebot
}

type typebotProvider struct{}

type tbResponse struct {
	SessionID string            `json:"sessionId"`
	Messages  []json.RawMessage `json:"messages"`
	Input     json.RawMessage   `json:"input"`
}

func (typebotProvider) Reply(ctx context.Context, b *Bot, s *Session, in Input) (*Output, error) {
	var c TypebotConfig
	decodeConfig(b, &c)
	if c.URL == "" || c.PublicID == "" {
		return nil, errors.New("url and publicId are required")
	}
	base := strings.TrimRight(c.URL, "/")
	msg := map[string]interface{}{"type": "text", "text": in.Text}
	var data []byte
	var code int
	var err error
	if s.ExternalID != "" {
		data, code, err = postJSON(ctx, base+"/api/v1/sessions/"+s.ExternalID+"/continueChat", map[string]interface{}{"message": msg}, nil)
		if err != nil && code != 404 && code != 400 {
			return nil, err
		}
	}
	if s.ExternalID == "" || err != nil {
		data, _, err = postJSON(ctx, base+"/api/v1/typebots/"+c.PublicID+"/startChat", map[string]interface{}{
			"message":            msg,
			"prefilledVariables": map[string]string{"remoteJid": in.RemoteJID, "pushName": in.PushName, "instanceId": in.InstanceID},
		}, nil)
		if err != nil {
			return nil, err
		}
	}
	var r tbResponse
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	out := &Output{ExternalID: r.SessionID, End: len(bytes.TrimSpace(r.Input)) == 0 || string(bytes.TrimSpace(r.Input)) == "null"}
	if out.ExternalID == "" {
		out.ExternalID = s.ExternalID
	}
	for _, raw := range r.Messages {
		if rep, ok := ParseTypebotMessage(raw); ok {
			out.Replies = append(out.Replies, rep)
		}
	}
	// Opciones de botones como texto, para que se puedan responder.
	var input struct {
		Type  string `json:"type"`
		Items []struct {
			Content string `json:"content"`
		} `json:"items"`
	}
	if json.Unmarshal(r.Input, &input) == nil && len(input.Items) > 0 {
		var lines []string
		for i, it := range input.Items {
			lines = append(lines, fmt.Sprintf("%d. %s", i+1, it.Content))
		}
		out.Replies = append(out.Replies, Reply{Text: strings.Join(lines, "\n")})
	}
	return out, nil
}

// ParseTypebotMessage convierte un mensaje de Typebot en respuesta.
func ParseTypebotMessage(raw json.RawMessage) (Reply, bool) {
	var m struct {
		Type    string `json:"type"`
		Content struct {
			RichText  []json.RawMessage `json:"richText"`
			Markdown  string            `json:"markdown"`
			PlainText string            `json:"plainText"`
			URL       string            `json:"url"`
		} `json:"content"`
	}
	if json.Unmarshal(raw, &m) != nil {
		return Reply{}, false
	}
	switch m.Type {
	case "text":
		t := m.Content.Markdown
		if t == "" {
			t = m.Content.PlainText
		}
		if t == "" {
			var paras []string
			for _, p := range m.Content.RichText {
				paras = append(paras, richText(p))
			}
			t = strings.Join(paras, "\n")
		}
		t = strings.TrimSpace(t)
		return Reply{Text: t}, t != ""
	case "image", "video", "audio":
		return Reply{MediaURL: m.Content.URL, MediaType: m.Type}, m.Content.URL != ""
	case "embed":
		return Reply{Text: m.Content.URL}, m.Content.URL != ""
	}
	return Reply{}, false
}

func richText(raw json.RawMessage) string {
	var n struct {
		Text     string            `json:"text"`
		Children []json.RawMessage `json:"children"`
		Bold     bool              `json:"bold"`
		Italic   bool              `json:"italic"`
	}
	if json.Unmarshal(raw, &n) != nil {
		return ""
	}
	if n.Text != "" {
		t := n.Text
		if n.Bold {
			t = "*" + t + "*"
		}
		if n.Italic {
			t = "_" + t + "_"
		}
		return t
	}
	var b strings.Builder
	for _, c := range n.Children {
		b.WriteString(richText(c))
	}
	return b.String()
}

// ---------- Dify ----------

type DifyConfig struct {
	APIURL  string `json:"apiUrl"` // https://api.dify.ai/v1
	APIKey  string `json:"apiKey"`
	BotType string `json:"botType"` // chat, agent, workflow, completion
}

type difyProvider struct{}

func (difyProvider) Reply(ctx context.Context, b *Bot, s *Session, in Input) (*Output, error) {
	c := DifyConfig{APIURL: "https://api.dify.ai/v1", BotType: "chat"}
	decodeConfig(b, &c)
	if c.APIKey == "" {
		return nil, errors.New("apiKey is required")
	}
	base := strings.TrimRight(c.APIURL, "/")
	auth := map[string]string{"Authorization": "Bearer " + c.APIKey}
	inputs := map[string]string{"remoteJid": in.RemoteJID, "pushName": in.PushName, "instanceId": in.InstanceID}
	user := in.RemoteJID

	switch c.BotType {
	case "workflow":
		inputs["query"] = in.Text
		data, _, err := postJSON(ctx, base+"/workflows/run", map[string]interface{}{"inputs": inputs, "response_mode": "blocking", "user": user}, auth)
		if err != nil {
			return nil, err
		}
		var r struct {
			Data struct {
				Outputs map[string]interface{} `json:"outputs"`
			} `json:"data"`
		}
		_ = json.Unmarshal(data, &r)
		for _, k := range []string{"text", "answer", "output", "result"} {
			if v, ok := r.Data.Outputs[k].(string); ok && v != "" {
				return &Output{Replies: []Reply{{Text: v}}}, nil
			}
		}
		for _, v := range r.Data.Outputs {
			if s, ok := v.(string); ok && s != "" {
				return &Output{Replies: []Reply{{Text: s}}}, nil
			}
		}
		return &Output{}, nil
	case "completion":
		inputs["query"] = in.Text
		data, _, err := postJSON(ctx, base+"/completion-messages", map[string]interface{}{"inputs": inputs, "response_mode": "blocking", "user": user}, auth)
		if err != nil {
			return nil, err
		}
		var r struct {
			Answer string `json:"answer"`
		}
		_ = json.Unmarshal(data, &r)
		return &Output{Replies: []Reply{{Text: r.Answer}}}, nil
	}

	// chat y agent comparten /chat-messages; agent solo admite streaming.
	mode := "blocking"
	if c.BotType == "agent" {
		mode = "streaming"
	}
	body := map[string]interface{}{"inputs": inputs, "query": in.Text, "response_mode": mode, "conversation_id": s.ExternalID, "user": user}
	if mode == "blocking" {
		data, _, err := postJSON(ctx, base+"/chat-messages", body, auth)
		if err != nil {
			return nil, err
		}
		var r struct {
			Answer         string `json:"answer"`
			ConversationID string `json:"conversation_id"`
		}
		_ = json.Unmarshal(data, &r)
		return &Output{Replies: []Reply{{Text: r.Answer}}, ExternalID: r.ConversationID}, nil
	}
	return difyStream(ctx, base+"/chat-messages", body, auth)
}

func difyStream(ctx context.Context, url string, body interface{}, headers map[string]string) (*Output, error) {
	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%s: %s", resp.Status, truncate(string(data), 300))
	}
	var answer strings.Builder
	conv := ""
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 64*1024), 4<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		var ev struct {
			Event          string `json:"event"`
			Answer         string `json:"answer"`
			ConversationID string `json:"conversation_id"`
		}
		if json.Unmarshal([]byte(strings.TrimSpace(strings.TrimPrefix(line, "data:"))), &ev) != nil {
			continue
		}
		if ev.ConversationID != "" {
			conv = ev.ConversationID
		}
		if ev.Event == "agent_message" || ev.Event == "message" {
			answer.WriteString(ev.Answer)
		}
		if ev.Event == "message_end" {
			break
		}
	}
	return &Output{Replies: []Reply{{Text: answer.String()}}, ExternalID: conv}, nil
}

// ---------- Flowise ----------

type FlowiseConfig struct {
	APIURL string `json:"apiUrl"` // https://host/api/v1/prediction/<chatflowId>
	APIKey string `json:"apiKey"`
}

type flowiseProvider struct{}

func (flowiseProvider) Reply(ctx context.Context, b *Bot, s *Session, in Input) (*Output, error) {
	var c FlowiseConfig
	decodeConfig(b, &c)
	if c.APIURL == "" {
		return nil, errors.New("apiUrl is required")
	}
	headers := map[string]string{}
	if c.APIKey != "" {
		headers["Authorization"] = "Bearer " + c.APIKey
	}
	data, _, err := postJSON(ctx, c.APIURL, map[string]interface{}{
		"question":       in.Text,
		"overrideConfig": map[string]interface{}{"sessionId": s.ID, "vars": map[string]string{"remoteJid": in.RemoteJID, "pushName": in.PushName}},
	}, headers)
	if err != nil {
		return nil, err
	}
	var r struct {
		Text string `json:"text"`
	}
	if json.Unmarshal(data, &r) == nil && r.Text != "" {
		return &Output{Replies: []Reply{{Text: r.Text}}}, nil
	}
	return &Output{Replies: ParseHTTPReply(data)}, nil
}
