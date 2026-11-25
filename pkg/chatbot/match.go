package chatbot

import (
	"regexp"
	"strings"
	"time"
)

// Matches indica si el texto activa el disparador del bot.
func Matches(b *Bot, text string) bool {
	switch b.TriggerType {
	case "", TriggerAll:
		return true
	case TriggerKeyword:
		v := strings.TrimSpace(b.TriggerValue)
		if v == "" {
			return false
		}
		t := strings.ToLower(strings.TrimSpace(text))
		lv := strings.ToLower(v)
		switch b.TriggerOperator {
		case OpEquals:
			return t == lv
		case OpStartsWith:
			return strings.HasPrefix(t, lv)
		case OpEndsWith:
			return strings.HasSuffix(t, lv)
		case OpRegex:
			re, err := regexp.Compile(v)
			return err == nil && re.MatchString(text)
		default:
			return strings.Contains(t, lv)
		}
	}
	return false
}

// Ignored indica si el chat está en la lista de ignorados del bot.
func Ignored(b *Bot, remoteJID string) bool {
	user := strings.Split(remoteJID, "@")[0]
	for _, j := range b.IgnoreJIDs {
		j = strings.TrimSpace(j)
		if j == "" {
			continue
		}
		if j == remoteJID || strings.TrimPrefix(j, "+") == user {
			return true
		}
	}
	return false
}

// Expired indica si la sesión caducó por inactividad.
func Expired(b *Bot, s *Session, now time.Time) bool {
	return b.ExpireMinutes > 0 && now.Sub(s.LastActivity) > time.Duration(b.ExpireMinutes)*time.Minute
}

// IsFinish indica si el texto es la palabra para terminar la sesión.
func IsFinish(b *Bot, text string) bool {
	k := strings.TrimSpace(b.KeywordFinish)
	return k != "" && strings.EqualFold(strings.TrimSpace(text), k)
}

// Split parte una respuesta larga por párrafos (líneas en blanco).
func Split(text string) []string {
	var out []string
	for _, p := range regexp.MustCompile(`\n\s*\n`).Split(text, -1) {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 && strings.TrimSpace(text) != "" {
		out = []string{strings.TrimSpace(text)}
	}
	return out
}

// TypingDelay calcula la espera con «escribiendo…»: la fija del bot o, si es 0,
// nada.
func TypingDelay(b *Bot) time.Duration {
	if b.DelayMs <= 0 {
		return 0
	}
	if b.DelayMs > 15000 {
		return 15 * time.Second
	}
	return time.Duration(b.DelayMs) * time.Millisecond
}
