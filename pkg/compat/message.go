package compat

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/evolution-foundation/evolution-go/pkg/evoformat"
	instance_model "github.com/evolution-foundation/evolution-go/pkg/instance/model"
	"github.com/gin-gonic/gin"
)

var nonDigit = regexp.MustCompile(`\D`)

// toJID convierte un número en JID; deja igual lo que ya es un JID.
func toJID(n string) string {
	if strings.Contains(n, "@") {
		return n
	}
	return nonDigit.ReplaceAllString(n, "") + "@s.whatsapp.net"
}

// common traduce los campos comunes de v2 (number, delay, quoted, mentions),
// también dentro de options como en v2.0/2.1.
func common(b M, i *instance_model.Instance) M {
	out := M{"number": str(b, "number")}
	if d := num(b, "delay", "options.delay"); d > 0 {
		out["delay"] = int(d)
	}
	if all, _ := boolOf(b, "mentionsEveryOne", "options.mentionsEveryOne", "options.mentions.everyOne"); all {
		out["mentionAll"] = true
	}
	var mentioned []string
	for _, k := range []string{"mentioned", "options.mentioned", "options.mentions.mentioned"} {
		for _, m := range strs(b, k) {
			mentioned = append(mentioned, toJID(m))
		}
	}
	if len(mentioned) > 0 {
		out["mentionedJid"] = mentioned
	}
	q := evoformat.Obj(b, "quoted")
	if q == nil {
		q = evoformat.Obj(b, "options.quoted")
	}
	if q != nil {
		id := str(q, "key.id")
		if id != "" {
			part := str(q, "key.participant")
			if part == "" {
				if fromMe, _ := boolOf(q, "key.fromMe"); fromMe {
					part = bareJID(i.Jid)
				} else {
					part = str(q, "key.remoteJid")
				}
			}
			out["quoted"] = M{"messageId": id, "participant": part}
		}
	}
	return out
}

// sendVia arma un manejador de envío: build completa el cuerpo de la ruta Go.
func (s *Server) sendVia(goPath string, build func(b, out M) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		i := inst(c)
		b, ok := body(c)
		if !ok {
			return
		}
		out := common(b, i)
		if msg := build(b, out); msg != "" {
			v2Error(c, http.StatusBadRequest, msg)
			return
		}
		if goPath != "/send/status/text" && goPath != "/send/status/media" && str(out, "number") == "" {
			v2Error(c, http.StatusBadRequest, "number is required")
			return
		}
		r := s.call(http.MethodPost, goPath, i.Token, out)
		if !r.ok() {
			fail(c, r)
			return
		}
		var data M
		_ = r.decode(&data)
		c.JSON(http.StatusCreated, evoformat.FromSend(data, i.Id))
	}
}

// media quita el prefijo data:…;base64, que v2 acepta y Go no.
func media(v string) string {
	if strings.HasPrefix(v, "data:") {
		if i := strings.Index(v, ","); i > 0 {
			return v[i+1:]
		}
	}
	return v
}

func (s *Server) registerMessages() {
	a := s.auth
	s.handle(http.MethodPost, "/message/sendText/:instanceName", a, s.sendVia("/send/text", func(b, out M) string {
		out["text"] = str(b, "text", "textMessage.text")
		if out["text"] == "" {
			return "text is required"
		}
		return ""
	}))
	s.handle(http.MethodPost, "/message/sendMedia/:instanceName", a, s.sendVia("/send/media", func(b, out M) string {
		src := b
		if mm := evoformat.Obj(b, "mediaMessage"); mm != nil {
			src = mm
		}
		t := str(src, "mediatype", "mediaType")
		if t == "" {
			return "mediatype is required"
		}
		out["type"], out["url"], out["caption"], out["filename"] = t, media(str(src, "media")), str(src, "caption"), str(src, "fileName")
		if out["url"] == "" {
			return "media is required"
		}
		return ""
	}))
	s.handle(http.MethodPost, "/message/sendPtv/:instanceName", a, s.sendVia("/send/media", func(b, out M) string {
		out["type"], out["url"] = "ptv", media(str(b, "video"))
		if out["url"] == "" {
			return "video is required"
		}
		return ""
	}))
	s.handle(http.MethodPost, "/message/sendWhatsAppAudio/:instanceName", a, s.sendVia("/send/media", func(b, out M) string {
		out["type"], out["url"] = "audio", media(str(b, "audio", "audioMessage.audio"))
		if out["url"] == "" {
			return "audio is required"
		}
		return ""
	}))
	s.handle(http.MethodPost, "/message/sendSticker/:instanceName", a, s.sendVia("/send/sticker", func(b, out M) string {
		out["sticker"] = media(str(b, "sticker", "stickerMessage.image"))
		if out["sticker"] == "" {
			return "sticker is required"
		}
		return ""
	}))
	s.handle(http.MethodPost, "/message/sendLocation/:instanceName", a, s.sendVia("/send/location", func(b, out M) string {
		src := b
		if lm := evoformat.Obj(b, "locationMessage"); lm != nil {
			src = lm
		}
		out["latitude"], out["longitude"], out["name"], out["address"] = num(src, "latitude"), num(src, "longitude"), str(src, "name"), str(src, "address")
		return ""
	}))
	s.handle(http.MethodPost, "/message/sendPoll/:instanceName", a, s.sendVia("/send/poll", func(b, out M) string {
		src := b
		if pm := evoformat.Obj(b, "pollMessage"); pm != nil {
			src = pm
		}
		out["question"], out["options"] = str(src, "name"), strs(src, "values")
		max := int(num(src, "selectableCount"))
		if max < 1 {
			max = 1
		}
		out["maxAnswer"] = max
		return ""
	}))
	s.handle(http.MethodPost, "/message/sendList/:instanceName", a, s.sendVia("/send/list", func(b, out M) string {
		src := b
		if lm := evoformat.Obj(b, "listMessage"); lm != nil {
			src = lm
		}
		out["title"], out["description"], out["buttonText"], out["footerText"] = str(src, "title"), str(src, "description"), str(src, "buttonText"), str(src, "footerText")
		out["sections"] = evoformat.Get(src, "sections")
		return ""
	}))
	s.handle(http.MethodPost, "/message/sendButtons/:instanceName", a, s.sendVia("/send/button", func(b, out M) string {
		out["title"], out["description"], out["footer"], out["buttons"] = str(b, "title"), str(b, "description"), str(b, "footer"), evoformat.Get(b, "buttons")
		if u := str(b, "thumbnailUrl"); u != "" {
			out["imageUrl"] = u
		}
		return ""
	}))
	s.handle(http.MethodPost, "/message/sendStatus/:instanceName", a, s.sendStatus)
	s.handle(http.MethodPost, "/message/sendContact/:instanceName", a, s.sendContact)
	s.handle(http.MethodPost, "/message/sendReaction/:instanceName", a, s.sendReaction)
}

func (s *Server) sendStatus(c *gin.Context) {
	i := inst(c)
	b, ok := body(c)
	if !ok {
		return
	}
	src := b
	if sm := evoformat.Obj(b, "statusMessage"); sm != nil {
		src = sm
	}
	t := str(src, "type")
	var r result
	if t == "text" {
		r = s.call(http.MethodPost, "/send/status/text", i.Token, M{"text": str(src, "content")})
	} else {
		r = s.call(http.MethodPost, "/send/status/media", i.Token, M{"type": t, "url": media(str(src, "content")), "caption": str(src, "caption")})
	}
	if !r.ok() {
		fail(c, r)
		return
	}
	var data M
	_ = r.decode(&data)
	c.JSON(http.StatusCreated, evoformat.FromSend(data, i.Id))
}

// sendContact: v2 manda varios contactos en un mensaje; Go uno por envío.
func (s *Server) sendContact(c *gin.Context) {
	i := inst(c)
	b, ok := body(c)
	if !ok {
		return
	}
	list, _ := evoformat.Get(b, "contact").([]interface{})
	if list == nil {
		list, _ = evoformat.Get(b, "contactMessage").([]interface{})
	}
	if len(list) == 0 {
		v2Error(c, http.StatusBadRequest, "contact is required")
		return
	}
	var cards []M
	for _, it := range list {
		ct, _ := it.(map[string]interface{})
		if ct == nil {
			continue
		}
		phone := str(ct, "phoneNumber")
		if phone == "" {
			phone = str(ct, "wuid")
		}
		cards = append(cards, M{"fullName": str(ct, "fullName"), "organization": str(ct, "organization"),
			"phone": nonDigit.ReplaceAllString(strings.Split(phone, "@")[0], ""), "email": str(ct, "email"), "url": str(ct, "url")})
	}
	if len(cards) == 0 {
		v2Error(c, http.StatusBadRequest, "contact is required")
		return
	}
	out := common(b, i)
	out["vcard"] = cards[0]
	if len(cards) > 1 {
		out["vcards"] = cards
	}
	r := s.call(http.MethodPost, "/send/contact", i.Token, out)
	if !r.ok() {
		fail(c, r)
		return
	}
	var data M
	_ = r.decode(&data)
	c.JSON(http.StatusCreated, evoformat.FromSend(data, i.Id))
}

func (s *Server) sendReaction(c *gin.Context) {
	i := inst(c)
	b, ok := body(c)
	if !ok {
		return
	}
	src := b
	if rm := evoformat.Obj(b, "reactionMessage"); rm != nil {
		src = rm
	}
	fromMe, _ := boolOf(src, "key.fromMe")
	out := M{"number": str(src, "key.remoteJid"), "id": str(src, "key.id"), "reaction": str(src, "reaction"), "fromMe": fromMe}
	if p := str(src, "key.participant"); p != "" {
		out["participant"] = p
	}
	if out["number"] == "" || out["id"] == "" {
		v2Error(c, http.StatusBadRequest, "key.remoteJid and key.id are required")
		return
	}
	r := s.call(http.MethodPost, "/message/react", i.Token, out)
	if !r.ok() {
		fail(c, r)
		return
	}
	var data M
	_ = r.decode(&data)
	if evoformat.Obj(data, "Info") != nil {
		c.JSON(http.StatusCreated, evoformat.FromSend(data, i.Id))
		return
	}
	c.JSON(http.StatusCreated, gin.H{"key": gin.H{"remoteJid": out["number"], "fromMe": true, "id": out["id"]}, "status": "PENDING"})
}
