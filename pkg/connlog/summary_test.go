package connlog

import (
	"testing"
	"time"

	"go.mau.fi/whatsmeow/types/events"
)

func ev(kind string, at time.Time) ConnectionEvent { return ConnectionEvent{Kind: kind, At: at} }

func TestSummarizeCortesCerradosYAbiertos(t *testing.T) {
	now := time.Date(2026, 11, 25, 16, 0, 0, 0, time.UTC)
	evts := []ConnectionEvent{
		ev(KindConnected, now.Add(-30*time.Hour)),
		ev(KindDisconnected, now.Add(-26*time.Hour)), // fuera de 24 h
		ev(KindConnected, now.Add(-25*time.Hour)),
		ev(KindDisconnected, now.Add(-2*time.Hour)),
		ev(KindConnected, now.Add(-2*time.Hour+41*time.Second)),
		ev(KindDisconnected, now.Add(-10*time.Minute)),
		ev(KindConnected, now.Add(-10*time.Minute+6*time.Second)),
	}
	last := evts[len(evts)-1]
	s := Summarize(evts, &last, now)
	if s.Cuts24h != 2 {
		t.Fatalf("Cuts24h = %d, want 2", s.Cuts24h)
	}
	if s.LongestCutSec != 41 {
		t.Fatalf("LongestCutSec = %d, want 41", s.LongestCutSec)
	}
	if s.ConnectedSince == nil || !s.ConnectedSince.Equal(now.Add(-10*time.Minute+6*time.Second)) {
		t.Fatalf("ConnectedSince = %v", s.ConnectedSince)
	}

	open := append(evts, ev(KindDisconnected, now.Add(-time.Minute)))
	s = Summarize(open, &open[len(open)-1], now)
	if s.Cuts24h != 3 || s.LongestCutSec != 60 {
		t.Fatalf("corte abierto: Cuts24h=%d LongestCutSec=%d", s.Cuts24h, s.LongestCutSec)
	}
}

func TestCutsTomaLaCausaDelCierreDeSesion(t *testing.T) {
	now := time.Now()
	cuts := Cuts([]ConnectionEvent{
		ev(KindDisconnected, now.Add(-time.Hour)),
		{Kind: KindLoggedOut, Reason: "logged out from another device", At: now.Add(-59 * time.Minute)},
	})
	if len(cuts) != 1 || cuts[0].Kind != KindLoggedOut || cuts[0].End != nil {
		t.Fatalf("cuts = %+v", cuts)
	}
}

func TestClassify(t *testing.T) {
	up := LiveState{Exists: true, Connected: true, LoggedIn: true}
	cases := []struct {
		name string
		live LiveState
		jid  string
		s    Summary
		want string
	}{
		{"estable", up, "57@s", Summary{}, StateConnected},
		{"volvió sola", up, "57@s", Summary{Cuts24h: 1}, StateReconnected},
		{"cliente sin socket", LiveState{Exists: true}, "57@s", Summary{LastKind: KindDisconnected}, StateReconnecting},
		{"sesión cerrada", LiveState{Exists: true}, "57@s", Summary{LastKind: KindLoggedOut}, StateDown},
		{"sin cliente", LiveState{}, "57@s", Summary{}, StateDown},
		{"nunca vinculada", LiveState{Exists: true}, "", Summary{}, StateDown},
	}
	for _, c := range cases {
		if got := Classify(c.live, c.jid, c.s); got != c.want {
			t.Errorf("%s: got %s, want %s", c.name, got, c.want)
		}
	}
}

func TestKindOf(t *testing.T) {
	if k, _ := KindOf(&events.Connected{}); k != KindConnected {
		t.Fatal(k)
	}
	if k, _ := KindOf(&events.Disconnected{}); k != KindDisconnected {
		t.Fatal(k)
	}
	if k, _ := KindOf(&events.Message{}); k != "" {
		t.Fatal(k)
	}
}

func TestHourBlocks(t *testing.T) {
	now := time.Date(2026, 11, 25, 16, 0, 0, 0, time.UTC)
	// Sin eventos: todo según el estado actual.
	for _, b := range HourBlocks(nil, true, now) {
		if b != HourUp {
			t.Fatalf("arriba sin eventos: %s", b)
		}
	}
	for _, b := range HourBlocks(nil, false, now) {
		if b != HourDown {
			t.Fatalf("caída sin eventos: %s", b)
		}
	}
	// Corte de 41 s en la hora 22 (de 14:00 a 15:00).
	evts := []ConnectionEvent{
		ev(KindDisconnected, now.Add(-90*time.Minute)),
		ev(KindConnected, now.Add(-90*time.Minute+41*time.Second)),
	}
	b := HourBlocks(evts, true, now)
	if b[22] != HourPartial || b[21] != HourUp || b[23] != HourUp {
		t.Fatalf("bloques = %v", b)
	}
	// Caída desde hace 3 h y sigue caída.
	b = HourBlocks([]ConnectionEvent{ev(KindLoggedOut, now.Add(-3*time.Hour))}, false, now)
	if b[20] != HourUp || b[21] != HourDown || b[23] != HourDown {
		t.Fatalf("bloques = %v", b)
	}
}
