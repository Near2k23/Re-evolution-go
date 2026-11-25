package webhooks

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func testService() *Service {
	s := New(nil, Settings{Timeout: 2 * time.Second, MaxAttempts: 4, BaseDelay: time.Millisecond})
	s.sleep = func(time.Duration) {}
	return s
}

func TestDeliverFirmaCabecerasYRutaPorEvento(t *testing.T) {
	payload := []byte(`{"event":"SendMessage","data":{}}`)
	var got *http.Request
	var body []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r
		body, _ = io.ReadAll(r.Body)
		w.WriteHeader(204)
	}))
	defer srv.Close()

	res := testService().Deliver(Target{URL: srv.URL + "/hook", Headers: map[string]string{"Authorization": "Bearer x"}, Secret: "s3cr3t", ByEvents: true}, payload, false)
	if !res.OK || res.Attempts != 1 || res.StatusCode != 204 {
		t.Fatalf("res = %+v", res)
	}
	if got.URL.Path != "/hook/send-message" {
		t.Fatalf("ruta = %s", got.URL.Path)
	}
	if got.Header.Get("Authorization") != "Bearer x" || got.Header.Get("X-Webhook-Event") != "SendMessage" || got.Header.Get("X-Webhook-Delivery") == "" {
		t.Fatalf("cabeceras = %v", got.Header)
	}
	ts := got.Header.Get("X-Webhook-Timestamp")
	if want := "sha256=" + Sign("s3cr3t", ts, body); got.Header.Get("X-Webhook-Signature") != want {
		t.Fatalf("firma = %s, want %s", got.Header.Get("X-Webhook-Signature"), want)
	}
}

func TestDeliverReintentaErrores5xx(t *testing.T) {
	var n int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&n, 1) < 3 {
			w.WriteHeader(503)
			return
		}
		w.WriteHeader(200)
	}))
	defer srv.Close()
	res := testService().Deliver(Target{URL: srv.URL}, []byte(`{"event":"Message"}`), false)
	if !res.OK || res.Attempts != 3 {
		t.Fatalf("res = %+v", res)
	}
}

func TestDeliverNoReintenta4xx(t *testing.T) {
	var n int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&n, 1)
		w.WriteHeader(401)
	}))
	defer srv.Close()
	res := testService().Deliver(Target{URL: srv.URL}, []byte(`{"event":"Message"}`), false)
	if res.OK || res.Attempts != 1 || n != 1 || res.StatusCode != 401 {
		t.Fatalf("res = %+v, llamadas %d", res, n)
	}
}

func TestDeliverReintenta429YAgota(t *testing.T) {
	var n int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&n, 1)
		w.WriteHeader(429)
	}))
	defer srv.Close()
	res := testService().Deliver(Target{URL: srv.URL}, []byte(`{"event":"Message"}`), false)
	if res.OK || res.Attempts != 4 || n != 4 {
		t.Fatalf("res = %+v, llamadas %d", res, n)
	}
}

func TestDeliverPruebaUnSoloIntento(t *testing.T) {
	var n int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&n, 1)
		w.WriteHeader(500)
	}))
	defer srv.Close()
	res := testService().Deliver(Target{URL: srv.URL}, []byte(`{"event":"Test"}`), true)
	if res.Attempts != 1 || n != 1 {
		t.Fatalf("res = %+v", res)
	}
}

func TestBackoffCreceConTope(t *testing.T) {
	s := New(nil, Settings{BaseDelay: time.Second})
	if d := s.backoff(1); d < 800*time.Millisecond || d > 1200*time.Millisecond {
		t.Fatalf("backoff(1) = %v", d)
	}
	if d := s.backoff(3); d < 3200*time.Millisecond || d > 4800*time.Millisecond {
		t.Fatalf("backoff(3) = %v", d)
	}
	if d := s.backoff(20); d > 72*time.Second {
		t.Fatalf("backoff(20) = %v", d)
	}
}

func TestKebabYCabecerasReservadas(t *testing.T) {
	if Kebab("SendMessage") != "send-message" || Kebab("QRCode") != "qrcode" && Kebab("QRCode") != "q-r-code" {
		t.Fatalf("kebab: %s %s", Kebab("SendMessage"), Kebab("QRCode"))
	}
	if !strings.Contains(Kebab("LabelAssociationChat"), "label-association-chat") {
		t.Fatal(Kebab("LabelAssociationChat"))
	}
	if !reserved["x-webhook-signature"] || headerName.MatchString("Bad Header") {
		t.Fatal("validación de cabeceras")
	}
}
