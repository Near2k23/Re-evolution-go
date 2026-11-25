package connlog

import "time"

// Cut es un corte: desde que se perdió la conexión hasta que volvió (End nulo si sigue caída).
type Cut struct {
	Start  time.Time  `json:"start"`
	End    *time.Time `json:"end,omitempty"`
	Kind   string     `json:"kind"`
	Reason string     `json:"reason,omitempty"`
}

// Seconds es la duración del corte (hasta now si sigue abierto).
func (c Cut) Seconds(now time.Time) int {
	end := now
	if c.End != nil {
		end = *c.End
	}
	return int(end.Sub(c.Start).Seconds())
}

// Cuts reconstruye los cortes a partir de eventos ordenados por fecha.
func Cuts(evts []ConnectionEvent) []Cut {
	var out []Cut
	var open *Cut
	for _, e := range evts {
		switch e.Kind {
		case KindDisconnected, KindLoggedOut, KindTemporaryBan, KindConnectFailure, KindStreamReplaced:
			if open == nil {
				open = &Cut{Start: e.At, Kind: e.Kind, Reason: e.Reason}
			} else if e.Kind != KindDisconnected {
				// Un cierre de sesión o bloqueo durante un corte define su causa.
				open.Kind, open.Reason = e.Kind, e.Reason
			}
		case KindConnected, KindPairSuccess:
			if open != nil {
				end := e.At
				open.End = &end
				out = append(out, *open)
				open = nil
			}
		}
	}
	if open != nil {
		out = append(out, *open)
	}
	return out
}

// Summary resume la conexión reciente de una instancia.
type Summary struct {
	Cuts24h         int        `json:"cuts24h"`
	LongestCutSec   int        `json:"longestCutSec"`
	LastCutAt       *time.Time `json:"lastCutAt,omitempty"`
	LastReconnectAt *time.Time `json:"lastReconnectAt,omitempty"`
	ConnectedSince  *time.Time `json:"connectedSince,omitempty"`
	LastKind        string     `json:"lastKind,omitempty"`
	LastReason      string     `json:"lastReason,omitempty"`
	LastEventAt     *time.Time `json:"lastEventAt,omitempty"`
}

// Summarize resume los eventos de las últimas 24 h; last es el último evento conocido
// (puede ser más antiguo que la ventana).
func Summarize(evts []ConnectionEvent, last *ConnectionEvent, now time.Time) Summary {
	var s Summary
	dayAgo := now.Add(-24 * time.Hour)
	for _, c := range Cuts(evts) {
		if c.Start.Before(dayAgo) {
			continue
		}
		s.Cuts24h++
		if d := c.Seconds(now); d > s.LongestCutSec {
			s.LongestCutSec = d
		}
		start := c.Start
		s.LastCutAt = &start
		if c.End != nil {
			end := *c.End
			s.LastReconnectAt = &end
		}
	}
	for i := len(evts) - 1; i >= 0; i-- {
		if evts[i].Kind == KindConnected || evts[i].Kind == KindPairSuccess {
			at := evts[i].At
			s.ConnectedSince = &at
			break
		}
	}
	if last != nil {
		s.LastKind, s.LastReason = last.Kind, last.Reason
		at := last.At
		s.LastEventAt = &at
	}
	return s
}

// LiveState es el estado en memoria del cliente de whatsmeow.
type LiveState struct {
	Exists    bool
	Connected bool
	LoggedIn  bool
}

// Classify decide el estado que ve el manager.
func Classify(live LiveState, jid string, s Summary) string {
	if live.Exists && live.Connected && live.LoggedIn {
		if s.Cuts24h > 0 {
			return StateReconnected
		}
		return StateConnected
	}
	if live.Exists && jid != "" && !NeedsPairing(jid, s) {
		return StateReconnecting
	}
	return StateDown
}

// NeedsPairing indica que hay que escanear el QR o usar el código otra vez
// (no hay sesión, o WhatsApp la cerró).
func NeedsPairing(jid string, s Summary) bool {
	return jid == "" || s.LastKind == KindLoggedOut
}

// Estados de un bloque horario.
const (
	HourUp      = "ok"
	HourPartial = "corte"
	HourDown    = "caida"
)

// HourBlocks divide las últimas 24 h en 24 bloques de una hora (el último termina
// en now) y marca cada uno como ok, corte (caída parcial) o caida (toda la hora).
// upNow es el estado actual; sirve cuando no hubo eventos en la ventana.
func HourBlocks(evts []ConnectionEvent, upNow bool, now time.Time) []string {
	start := now.Add(-24 * time.Hour)

	// Estado al inicio de la ventana: lo contrario del primer evento, o el actual.
	up := upNow
	if len(evts) > 0 {
		switch evts[0].Kind {
		case KindConnected, KindPairSuccess:
			up = false
		default:
			up = true
		}
	}

	// Intervalos caídos dentro de la ventana.
	type span struct{ a, b time.Time }
	var down []span
	var downFrom *time.Time
	if !up {
		s := start
		downFrom = &s
	}
	for _, e := range evts {
		if e.At.Before(start) {
			continue
		}
		switch e.Kind {
		case KindConnected, KindPairSuccess:
			if downFrom != nil {
				down = append(down, span{*downFrom, e.At})
				downFrom = nil
			}
		default:
			if downFrom == nil {
				at := e.At
				downFrom = &at
			}
		}
	}
	if downFrom != nil {
		down = append(down, span{*downFrom, now})
	}

	out := make([]string, 24)
	for h := 0; h < 24; h++ {
		a := start.Add(time.Duration(h) * time.Hour)
		b := a.Add(time.Hour)
		var covered time.Duration
		for _, d := range down {
			lo, hi := d.a, d.b
			if lo.Before(a) {
				lo = a
			}
			if hi.After(b) {
				hi = b
			}
			if hi.After(lo) {
				covered += hi.Sub(lo)
			}
		}
		switch {
		case covered <= 0:
			out[h] = HourUp
		case covered >= time.Hour-time.Second:
			out[h] = HourDown
		default:
			out[h] = HourPartial
		}
	}
	return out
}
