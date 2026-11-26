package sinks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

const pusherMaxBytes = 10 * 1024

// Pusher publica en un canal de Pusher (o Soketi, con PUSHER_HOST).
type Pusher struct {
	base, appID, key, secret, channel string
	client                            *http.Client
	now                               func() time.Time
}

// NewPusherFromEnv usa PUSHER_APP_ID, PUSHER_KEY, PUSHER_SECRET,
// PUSHER_CLUSTER (o PUSHER_HOST) y PUSHER_CHANNEL (por defecto, el nombre de
// cada línea).
func NewPusherFromEnv() (*Pusher, error) {
	p := &Pusher{appID: os.Getenv("PUSHER_APP_ID"), key: os.Getenv("PUSHER_KEY"), secret: os.Getenv("PUSHER_SECRET"),
		channel: os.Getenv("PUSHER_CHANNEL"), client: &http.Client{Timeout: 10 * time.Second}, now: time.Now}
	if p.appID == "" || p.key == "" || p.secret == "" {
		return nil, errors.New("PUSHER_APP_ID, PUSHER_KEY and PUSHER_SECRET are required")
	}
	switch {
	case os.Getenv("PUSHER_HOST") != "":
		p.base = strings.TrimRight(os.Getenv("PUSHER_HOST"), "/")
		if !strings.Contains(p.base, "://") {
			p.base = "https://" + p.base
		}
	case os.Getenv("PUSHER_CLUSTER") != "":
		p.base = "https://api-" + os.Getenv("PUSHER_CLUSTER") + ".pusher.com"
	default:
		p.base = "https://api.pusherapp.com"
	}
	return p, nil
}

func (p *Pusher) Name() string { return "pusher" }

// channelFor usa PUSHER_CHANNEL o el nombre de la línea.
func (p *Pusher) channelFor(m Message) string {
	if p.channel != "" {
		return p.channel
	}
	c := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("-_=@,.;", r) {
			return r
		}
		return '-'
	}, m.InstanceName)
	if c == "" {
		c = m.InstanceID
	}
	return c
}

// signedURL firma la petición como pide la API HTTP de Pusher.
func (p *Pusher) signedURL(path string, body []byte) string {
	sum := md5.Sum(body)
	q := map[string]string{
		"auth_key":       p.key,
		"auth_timestamp": strconv.FormatInt(p.now().Unix(), 10),
		"auth_version":   "1.0",
		"body_md5":       hex.EncodeToString(sum[:]),
	}
	keys := make([]string, 0, len(q))
	for k := range q {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+q[k])
	}
	query := strings.Join(parts, "&")
	mac := hmac.New(sha256.New, []byte(p.secret))
	mac.Write([]byte("POST\n" + path + "\n" + query))
	return p.base + path + "?" + query + "&auth_signature=" + hex.EncodeToString(mac.Sum(nil))
}

func (p *Pusher) Send(ctx context.Context, m Message) error {
	data := string(m.Payload)
	if len(m.Payload) > pusherMaxBytes {
		// Pusher acepta hasta 10 KB: se avisa del evento sin su contenido.
		small, _ := json.Marshal(map[string]interface{}{"event": m.Event, "instance": m.InstanceName, "instanceId": m.InstanceID, "truncated": true})
		data = string(small)
	}
	body, _ := json.Marshal(map[string]interface{}{"name": m.Event, "channels": []string{p.channelFor(m)}, "data": data})
	path := "/apps/" + p.appID + "/events"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.signedURL(path, body), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 300))
		return fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	return nil
}

func (p *Pusher) Close() error { return nil }
