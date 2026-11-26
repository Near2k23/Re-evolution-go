package sinks

import (
	"context"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/evolution-foundation/evolution-go/pkg/evoformat"
)

type fakeSink struct {
	mu   sync.Mutex
	got  []Message
	done chan struct{}
}

func (f *fakeSink) Name() string { return "fake" }
func (f *fakeSink) Send(_ context.Context, m Message) error {
	f.mu.Lock()
	f.got = append(f.got, m)
	f.mu.Unlock()
	f.done <- struct{}{}
	return nil
}
func (f *fakeSink) Close() error { return nil }

func wait(t *testing.T, ch chan struct{}, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		select {
		case <-ch:
		case <-time.After(2 * time.Second):
			t.Fatalf("got %d of %d messages", i, n)
		}
	}
}

const msgEvent = `{"event":"Message","instanceId":"i1","instanceName":"ventas","instanceToken":"tok","data":{"Info":{"Chat":"57300@s.whatsapp.net","ID":"3EB0","Timestamp":"2026-11-25T15:00:00Z"},"Message":{"conversation":"hola"}}}`

func TestHubFilterAndFormat(t *testing.T) {
	goSink := &fakeSink{done: make(chan struct{}, 10)}
	v2Sink := &fakeSink{done: make(chan struct{}, 10)}
	h := NewHub(Options{Group: func(e string) string {
		return map[string]string{"Message": "MESSAGE", "Receipt": "READ_RECEIPT"}[e]
	}, Meta: func(string) evoformat.Meta { return evoformat.Meta{Sender: "57111@s.whatsapp.net"} }})
	h.Add(goSink, Filter{Groups: map[string]bool{"MESSAGE": true}})
	h.Add(v2Sink, Filter{All: true, V2: true})

	h.Publish([]byte(msgEvent))
	h.Publish([]byte(`{"event":"Receipt","instanceId":"i1","data":{"Chat":"x","MessageIDs":["A","B"]}}`))
	wait(t, goSink.done, 1)
	wait(t, v2Sink.done, 3) // messages.upsert + 2 messages.update
	if goSink.got[0].Event != "Message" || goSink.got[0].InstanceName != "ventas" {
		t.Fatalf("go sink: %+v", goSink.got[0])
	}
	events := []string{}
	for _, m := range v2Sink.got {
		events = append(events, m.Event)
	}
	sort.Strings(events)
	if strings.Join(events, ",") != "messages.update,messages.update,messages.upsert" {
		t.Fatalf("v2 events: %v", events)
	}
	h.Close()
}

func TestFilterFromEnv(t *testing.T) {
	t.Setenv("X_EVENTS", "message, CONNECTION")
	t.Setenv("X_FORMAT", "evolution-api")
	f := FilterFromEnv("X")
	if f.All || !f.V2 || !f.Allows("Message", "") || !f.Allows("LoggedOut", "CONNECTION") || f.Allows("Receipt", "READ_RECEIPT") {
		t.Fatalf("filter: %+v", f)
	}
	if !FilterFromEnv("NONE").All {
		t.Fatal("empty events means all")
	}
}

func TestPusherSignature(t *testing.T) {
	var gotQuery, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotQuery, gotBody = r.URL.RawQuery, string(b)
		if r.URL.Path != "/apps/123/events" {
			http.Error(w, "path", 404)
			return
		}
		w.Write([]byte("{}"))
	}))
	defer srv.Close()
	t.Setenv("PUSHER_APP_ID", "123")
	t.Setenv("PUSHER_KEY", "key")
	t.Setenv("PUSHER_SECRET", "secret")
	t.Setenv("PUSHER_HOST", srv.URL)
	p, err := NewPusherFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	p.now = func() time.Time { return time.Unix(1700000000, 0) }
	if err := p.Send(context.Background(), Message{Event: "Message", InstanceName: "Ventas 1", Payload: []byte(`{"a":1}`)}); err != nil {
		t.Fatal(err)
	}
	var body map[string]interface{}
	_ = json.Unmarshal([]byte(gotBody), &body)
	if body["name"] != "Message" || body["channels"].([]interface{})[0] != "Ventas-1" || body["data"] != `{"a":1}` {
		t.Fatalf("body: %s", gotBody)
	}
	sum := md5.Sum([]byte(gotBody))
	base := "auth_key=key&auth_timestamp=1700000000&auth_version=1.0&body_md5=" + hex.EncodeToString(sum[:])
	mac := hmac.New(sha256.New, []byte("secret"))
	mac.Write([]byte("POST\n/apps/123/events\n" + base))
	if gotQuery != base+"&auth_signature="+hex.EncodeToString(mac.Sum(nil)) {
		t.Fatalf("query: %s", gotQuery)
	}
	// Más de 10 KB: se manda un aviso sin el contenido.
	_ = p.Send(context.Background(), Message{Event: "Message", InstanceName: "v", Payload: []byte(strings.Repeat("x", 11*1024))})
	if !strings.Contains(gotBody, `truncated`) || len(gotBody) > 1024 {
		t.Fatalf("large payload body: %d bytes", len(gotBody))
	}
}

func TestSQSSend(t *testing.T) {
	var got map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &got)
		body, _ := got["MessageBody"].(string)
		sum := md5.Sum([]byte(body))
		w.Header().Set("Content-Type", "application/x-amz-json-1.0")
		w.Write([]byte(`{"MessageId":"m1","MD5OfMessageBody":"` + hex.EncodeToString(sum[:]) + `"}`))
	}))
	defer srv.Close()
	t.Setenv("SQS_QUEUE_URL", srv.URL+"/000000000000/eventos.fifo")
	t.Setenv("SQS_REGION", "us-east-1")
	t.Setenv("SQS_ACCESS_KEY_ID", "AKIDTEST")
	t.Setenv("SQS_SECRET_ACCESS_KEY", "secret")
	t.Setenv("SQS_ENDPOINT", srv.URL)
	s, err := NewSQSFromEnv(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Send(context.Background(), Message{Event: "Message", InstanceID: "i1", InstanceName: "ventas", Payload: []byte(msgEvent)}); err != nil {
		t.Fatal(err)
	}
	if got["MessageBody"] != msgEvent || got["MessageGroupId"] != "i1" || got["MessageDeduplicationId"] == "" {
		t.Fatalf("sqs request: %v", got)
	}
	attrs, _ := got["MessageAttributes"].(map[string]interface{})
	if ev, _ := attrs["event"].(map[string]interface{}); ev["StringValue"] != "Message" {
		t.Fatalf("attributes: %v", attrs)
	}
}

func TestKafkaTopic(t *testing.T) {
	t.Setenv("KAFKA_BROKERS", "localhost:9092")
	t.Setenv("KAFKA_TOPIC", "evo")
	t.Setenv("KAFKA_TOPIC_PER_EVENT", "true")
	k, err := NewKafkaFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	defer k.Close()
	if k.TopicFor("messages.upsert") != "evo.messages.upsert" || k.TopicFor("SendMessage") != "evo.sendmessage" {
		t.Fatalf("topics: %s %s", k.TopicFor("messages.upsert"), k.TopicFor("SendMessage"))
	}
}
