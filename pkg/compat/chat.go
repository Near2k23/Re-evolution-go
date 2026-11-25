package compat

import (
	"encoding/json"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/evolution-foundation/evolution-go/pkg/chatstore"
	"github.com/evolution-foundation/evolution-go/pkg/evoformat"
	instance_model "github.com/evolution-foundation/evolution-go/pkg/instance/model"
	"github.com/gin-gonic/gin"
)

func (s *Server) registerChat() {
	a := s.auth
	p := func(method, name string, h gin.HandlerFunc) { s.handle(method, "/chat/"+name+"/:instanceName", a, h) }
	p(http.MethodPost, "whatsappNumbers", s.whatsappNumbers)
	p(http.MethodPost, "markMessageAsRead", s.markRead)
	p(http.MethodDelete, "deleteMessageForEveryone", s.deleteForEveryone)
	p(http.MethodPost, "updateMessage", s.updateMessage)
	p(http.MethodPost, "sendPresence", s.sendPresence)
	p(http.MethodPost, "fetchProfilePictureUrl", s.profilePicture)
	p(http.MethodPost, "fetchProfile", s.fetchProfile)
	p(http.MethodPost, "updateBlockStatus", s.blockStatus)
	p(http.MethodPost, "findChats", s.findChats)
	p(http.MethodPost, "findMessages", s.findMessages)
	p(http.MethodPost, "findContacts", s.findContacts)
	p(http.MethodGet, "findChatByRemoteJid", s.findChatByRemoteJid)
	p(http.MethodPost, "getBase64FromMediaMessage", s.mediaBase64)
	p(http.MethodPost, "updateProfileName", s.simple("/user/profileName", func(b M) M { return M{"name": str(b, "name")} }))
	p(http.MethodPost, "updateProfileStatus", s.simple("/user/profileStatus", func(b M) M { return M{"status": str(b, "status")} }))
	p(http.MethodPost, "updateProfilePicture", s.simple("/user/profilePicture", func(b M) M { return M{"image": str(b, "picture")} }))
	p(http.MethodGet, "fetchPrivacySettings", s.privacy)
	p(http.MethodPost, "archiveChat", s.archiveChat)
	p(http.MethodPost, "markChatUnread", s.markChatUnread)
	p(http.MethodPost, "fetchBusinessProfile", s.businessProfile)
}

// chatOf lee el chat de un cuerpo v2 (chat o lastMessage.key.remoteJid).
func chatOf(b M) string {
	return str(b, "chat", "lastMessage.key.remoteJid", "number")
}

func (s *Server) archiveChat(c *gin.Context) {
	i := inst(c)
	b, ok := body(c)
	if !ok {
		return
	}
	chat := chatOf(b)
	if chat == "" {
		v2Error(c, http.StatusBadRequest, "chat is required")
		return
	}
	archive, has := boolOf(b, "archive")
	path := "/chat/archive"
	if has && !archive {
		path = "/chat/unarchive"
	}
	if r := s.call(http.MethodPost, path, i.Token, M{"chat": toJID(chat)}); !r.ok() {
		fail(c, r)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"chatId": toJID(chat), "archived": !has || archive})
}

func (s *Server) markChatUnread(c *gin.Context) {
	i := inst(c)
	b, ok := body(c)
	if !ok {
		return
	}
	chat := chatOf(b)
	if chat == "" {
		v2Error(c, http.StatusBadRequest, "chat is required")
		return
	}
	if r := s.call(http.MethodPost, "/chat/markunread", i.Token, M{"chat": toJID(chat)}); !r.ok() {
		fail(c, r)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"chatId": toJID(chat), "markedChatUnread": true})
}

func (s *Server) businessProfile(c *gin.Context) {
	i := inst(c)
	b, ok := body(c)
	if !ok {
		return
	}
	n := str(b, "number")
	r := s.call(http.MethodPost, "/user/business", i.Token, M{"number": n})
	if !r.ok() {
		fail(c, r)
		return
	}
	var p M
	_ = r.decode(&p)
	cats := []string{}
	if list, ok := p["Categories"].([]interface{}); ok {
		for _, it := range list {
			if m, ok := it.(map[string]interface{}); ok {
				cats = append(cats, evoformat.Str(m, "Name"))
			}
		}
	}
	desc := ""
	if opts, ok := p["ProfileOptions"].(map[string]interface{}); ok {
		desc, _ = opts["description"].(string)
	}
	c.JSON(http.StatusOK, gin.H{"wid": nilIfEmpty(evoformat.Str(p, "JID")), "address": nilIfEmpty(evoformat.Str(p, "Address")),
		"email": nilIfEmpty(evoformat.Str(p, "Email")), "description": nilIfEmpty(desc), "website": []string{},
		"category": nilIfEmpty(strings.Join(cats, ", ")), "business_hours": p["BusinessHours"], "isBusiness": evoformat.Str(p, "JID") != ""})
}

// simple llama a una ruta Go y responde {update: "success"}.
func (s *Server) simple(goPath string, build func(M) M) gin.HandlerFunc {
	return func(c *gin.Context) {
		i := inst(c)
		b, ok := body(c)
		if !ok {
			return
		}
		if r := s.call(http.MethodPost, goPath, i.Token, build(b)); !r.ok() {
			fail(c, r)
			return
		}
		c.JSON(http.StatusOK, gin.H{"update": "success"})
	}
}

func (s *Server) whatsappNumbers(c *gin.Context) {
	i := inst(c)
	b, ok := body(c)
	if !ok {
		return
	}
	nums := strs(b, "numbers")
	if len(nums) == 0 {
		v2Error(c, http.StatusBadRequest, "numbers is required")
		return
	}
	r := s.call(http.MethodPost, "/user/check", i.Token, M{"number": nums})
	if !r.ok() {
		fail(c, r)
		return
	}
	var data struct {
		Users []struct {
			Query        string
			IsInWhatsapp bool
			JID          string
			LID          *string
			VerifiedName string
		}
	}
	_ = r.decode(&data)
	out := make([]gin.H, 0, len(data.Users))
	for _, u := range data.Users {
		item := gin.H{"exists": u.IsInWhatsapp, "jid": u.JID, "number": strings.Split(u.Query, "@")[0]}
		if u.VerifiedName != "" {
			item["name"] = u.VerifiedName
		}
		if u.LID != nil {
			item["lid"] = *u.LID
		}
		out = append(out, item)
	}
	c.JSON(http.StatusOK, out)
}

func (s *Server) markRead(c *gin.Context) {
	i := inst(c)
	b, ok := body(c)
	if !ok {
		return
	}
	list, _ := evoformat.Get(b, "readMessages").([]interface{})
	byChat := map[string][]string{}
	var order []string
	for _, it := range list {
		m, _ := it.(map[string]interface{})
		if m == nil {
			continue
		}
		if fromMe, _ := boolOf(m, "fromMe"); fromMe {
			continue
		}
		chat, id := str(m, "remoteJid"), str(m, "id")
		if chat == "" || id == "" {
			continue
		}
		if _, seen := byChat[chat]; !seen {
			order = append(order, chat)
		}
		byChat[chat] = append(byChat[chat], id)
	}
	for _, chat := range order {
		if r := s.call(http.MethodPost, "/message/markread", i.Token, M{"number": chat, "id": byChat[chat]}); !r.ok() {
			fail(c, r)
			return
		}
	}
	c.JSON(http.StatusCreated, gin.H{"message": "Read messages", "read": "success"})
}

func (s *Server) deleteForEveryone(c *gin.Context) {
	i := inst(c)
	b, ok := body(c)
	if !ok {
		return
	}
	chat, id := str(b, "remoteJid"), str(b, "id")
	if chat == "" || id == "" {
		v2Error(c, http.StatusBadRequest, "remoteJid and id are required")
		return
	}
	r := s.call(http.MethodPost, "/message/delete", i.Token, M{"chat": chat, "messageId": id})
	if !r.ok() {
		fail(c, r)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"key": gin.H{"remoteJid": chat, "fromMe": true, "id": id}, "status": "PENDING",
		"message":     gin.H{"protocolMessage": gin.H{"key": gin.H{"remoteJid": chat, "fromMe": true, "id": id}, "type": "REVOKE"}},
		"messageType": "protocolMessage", "messageTimestamp": time.Now().Unix(), "instanceId": i.Id})
}

func (s *Server) updateMessage(c *gin.Context) {
	i := inst(c)
	b, ok := body(c)
	if !ok {
		return
	}
	chat := str(b, "number", "key.remoteJid")
	r := s.call(http.MethodPost, "/message/edit", i.Token, M{"chat": toJID(chat), "messageId": str(b, "key.id"), "message": str(b, "text")})
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
	c.JSON(http.StatusCreated, gin.H{"key": gin.H{"remoteJid": toJID(chat), "fromMe": true, "id": str(b, "key.id")}, "status": "PENDING"})
}

func (s *Server) sendPresence(c *gin.Context) {
	i := inst(c)
	b, ok := body(c)
	if !ok {
		return
	}
	presence := str(b, "presence", "options.presence")
	state, audio := "composing", false
	switch presence {
	case "recording":
		audio = true
	case "paused", "available", "unavailable":
		state = "paused"
	}
	out := M{"number": str(b, "number"), "state": state, "isAudio": audio}
	if d := num(b, "delay", "options.delay"); d > 0 {
		out["delay"] = int(d)
	}
	if r := s.call(http.MethodPost, "/message/presence", i.Token, out); !r.ok() {
		fail(c, r)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"presence": presence})
}

func (s *Server) avatar(i *instance_model.Instance, number string) interface{} {
	r := s.call(http.MethodPost, "/user/avatar", i.Token, M{"number": number})
	var pic struct{ URL string }
	if r.ok() && r.decode(&pic) == nil && pic.URL != "" {
		return pic.URL
	}
	return nil
}

func (s *Server) profilePicture(c *gin.Context) {
	i := inst(c)
	b, ok := body(c)
	if !ok {
		return
	}
	n := str(b, "number")
	c.JSON(http.StatusOK, gin.H{"wuid": toJID(n), "profilePictureUrl": s.avatar(i, n)})
}

func (s *Server) fetchProfile(c *gin.Context) {
	i := inst(c)
	b, ok := body(c)
	if !ok {
		return
	}
	n := str(b, "number")
	if n == "" {
		n = strings.Split(bareJID(i.Jid), "@")[0]
	}
	out := gin.H{"wuid": toJID(n), "name": nil, "numberExists": false, "picture": s.avatar(i, n), "status": nil, "isBusiness": false}
	r := s.call(http.MethodPost, "/user/info", i.Token, M{"number": []string{n}})
	var data struct {
		Users map[string]struct {
			VerifiedName *struct {
				Details *struct {
					VerifiedName string `json:"verifiedName"`
				} `json:"Details"`
			}
			Status string
		}
	}
	if r.ok() && r.decode(&data) == nil {
		for jid, u := range data.Users {
			out["wuid"], out["numberExists"], out["status"] = jid, true, nilIfEmpty(u.Status)
			if u.VerifiedName != nil {
				out["isBusiness"] = true
				if u.VerifiedName.Details != nil {
					out["name"] = nilIfEmpty(u.VerifiedName.Details.VerifiedName)
				}
			}
			break
		}
	}
	c.JSON(http.StatusOK, out)
}

func (s *Server) blockStatus(c *gin.Context) {
	i := inst(c)
	b, ok := body(c)
	if !ok {
		return
	}
	status := str(b, "status")
	path := "/user/block"
	if status == "unblock" {
		path = "/user/unblock"
	} else if status != "block" {
		v2Error(c, http.StatusBadRequest, "status must be block or unblock")
		return
	}
	if r := s.call(http.MethodPost, path, i.Token, M{"number": str(b, "number")}); !r.ok() {
		fail(c, r)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"accepted": true})
}

func (s *Server) privacy(c *gin.Context) {
	i := inst(c)
	r := s.call(http.MethodGet, "/user/privacy", i.Token, nil)
	if !r.ok() {
		fail(c, r)
		return
	}
	var p map[string]interface{}
	_ = r.decode(&p)
	c.JSON(http.StatusOK, gin.H{"readreceipts": p["ReadReceipts"], "profile": p["Profile"], "status": p["Status"],
		"online": p["Online"], "last": p["LastSeen"], "groupadd": p["GroupAdd"], "calladd": p["CallAdd"]})
}

// ---------- historial (chatstore) ----------

var typeNames = map[string]string{
	chatstore.TypeText: "conversation", chatstore.TypeImage: "imageMessage", chatstore.TypeVideo: "videoMessage",
	chatstore.TypeAudio: "audioMessage", chatstore.TypeDocument: "documentMessage", chatstore.TypeSticker: "stickerMessage",
	chatstore.TypeLocation: "locationMessage", chatstore.TypeContact: "contactMessage", chatstore.TypePoll: "pollCreationMessageV3",
}

var statusNames = map[string]string{
	chatstore.StatusReceived: "DELIVERY_ACK", chatstore.StatusSent: "SERVER_ACK", chatstore.StatusDelivered: "DELIVERY_ACK",
	chatstore.StatusRead: "READ", chatstore.StatusPlayed: "PLAYED",
}

func messageRecord(m *chatstore.Message, instanceID string) gin.H {
	var msg M
	if len(m.Raw) > 0 {
		_ = json.Unmarshal(m.Raw, &msg)
	}
	if msg == nil {
		msg = M{"conversation": m.Text}
	}
	key := gin.H{"id": m.MessageID, "fromMe": m.FromMe, "remoteJid": m.ChatJID}
	if strings.HasSuffix(m.ChatJID, "@g.us") && m.SenderJID != "" {
		key["participant"] = m.SenderJID
	}
	mt := evoformat.MessageType(msg)
	if len(m.Raw) == 0 && typeNames[m.Type] != "" {
		mt = typeNames[m.Type]
	}
	status := statusNames[m.Status]
	if status == "" {
		status = "PENDING"
	}
	return gin.H{"id": m.ID, "key": key, "pushName": m.SenderName, "participant": nilIfEmpty(m.SenderJID), "messageType": mt,
		"message": msg, "contextInfo": nil, "source": evoformat.Source(m.MessageID), "messageTimestamp": m.Timestamp.Unix(),
		"instanceId": instanceID, "status": status, "MessageUpdate": []interface{}{}}
}

func (s *Server) needStore(c *gin.Context) bool {
	if s.opt.Store == nil {
		v2Error(c, http.StatusBadRequest, "chat history is disabled (CHAT_STORE_ENABLED=false)")
		return false
	}
	return true
}

func (s *Server) findMessages(c *gin.Context) {
	if !s.needStore(c) {
		return
	}
	i := inst(c)
	b, ok := body(c)
	if !ok {
		return
	}
	q := s.opt.DB.Model(&chatstore.Message{}).Where("instance_id = ?", i.Id)
	if v := str(b, "where.key.remoteJid", "where.remoteJid"); v != "" {
		q = q.Where("chat_jid = ?", v)
	}
	if v := str(b, "where.key.id", "where.id"); v != "" {
		q = q.Where("message_id = ?", v)
	}
	if v, has := boolOf(b, "where.key.fromMe"); has {
		q = q.Where("from_me = ?", v)
	}
	if v := str(b, "where.key.participant", "where.key.participants"); v != "" {
		q = q.Where("sender_jid = ?", v)
	}
	if v := num(b, "where.messageTimestamp.gte"); v > 0 {
		q = q.Where("timestamp >= ?", time.Unix(int64(v), 0))
	}
	if v := num(b, "where.messageTimestamp.lte"); v > 0 {
		q = q.Where("timestamp <= ?", time.Unix(int64(v), 0))
	}
	var total int64
	q.Count(&total)
	size := int(num(b, "offset", "limit"))
	if size <= 0 || size > 500 {
		size = 50
	}
	page := int(num(b, "page"))
	if page < 1 {
		page = 1
	}
	var rows []chatstore.Message
	if err := q.Order("timestamp DESC, id DESC").Offset((page - 1) * size).Limit(size).Find(&rows).Error; err != nil {
		v2Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	records := make([]gin.H, 0, len(rows))
	for idx := range rows {
		records = append(records, messageRecord(&rows[idx], i.Id))
	}
	c.JSON(http.StatusOK, gin.H{"messages": gin.H{"total": total, "pages": int(math.Ceil(float64(total) / float64(size))),
		"currentPage": page, "records": records}})
}

func chatRecord(ch *chatstore.ChatView, instanceID string) gin.H {
	out := gin.H{"id": ch.JID, "remoteJid": ch.JID, "pushName": nilIfEmpty(ch.DisplayName), "name": nilIfEmpty(ch.DisplayName),
		"profilePicUrl": nil, "updatedAt": ch.UpdatedAt, "unreadCount": ch.UnreadCount, "archived": ch.Archived, "pinned": ch.Pinned,
		"isGroup": ch.IsGroup, "instanceId": instanceID}
	if ch.LastMessageID != "" && ch.LastMessageAt != nil {
		mt := typeNames[ch.LastType]
		if mt == "" {
			mt = "conversation"
		}
		out["lastMessage"] = gin.H{"key": gin.H{"id": ch.LastMessageID, "fromMe": ch.LastFromMe, "remoteJid": ch.JID},
			"messageType": mt, "message": gin.H{"conversation": ch.LastPreview}, "messageTimestamp": ch.LastMessageAt.Unix()}
	}
	return out
}

func (s *Server) chatList(i *instance_model.Instance, jid string, limit int) ([]gin.H, error) {
	var rows []chatstore.ChatView
	var err error
	if jid != "" {
		rows, err = s.opt.Store.GetChat(i.Id, jid)
	} else {
		var a, b []chatstore.ChatView
		if a, err = s.opt.Store.ListChats(i.Id, "", false, limit, nil); err == nil {
			b, err = s.opt.Store.ListChats(i.Id, "", true, limit, nil)
			rows = append(a, b...)
		}
	}
	out := make([]gin.H, 0, len(rows))
	for idx := range rows {
		out = append(out, chatRecord(&rows[idx], i.Id))
	}
	return out, err
}

func (s *Server) findChats(c *gin.Context) {
	if !s.needStore(c) {
		return
	}
	i := inst(c)
	b, ok := body(c)
	if !ok {
		return
	}
	limit := int(num(b, "take", "limit"))
	if limit <= 0 || limit > 2000 {
		limit = 1000
	}
	out, err := s.chatList(i, str(b, "where.remoteJid", "where.id"), limit)
	if err != nil {
		v2Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, out)
}

func (s *Server) findChatByRemoteJid(c *gin.Context) {
	if !s.needStore(c) {
		return
	}
	out, err := s.chatList(inst(c), c.Query("remoteJid"), 1)
	if err != nil || len(out) == 0 {
		c.JSON(http.StatusOK, nil)
		return
	}
	c.JSON(http.StatusOK, out[0])
}

func (s *Server) findContacts(c *gin.Context) {
	if !s.needStore(c) {
		return
	}
	i := inst(c)
	b, ok := body(c)
	if !ok {
		return
	}
	var rows []chatstore.Contact
	q := s.opt.DB.Where("instance_id = ?", i.Id)
	if v := str(b, "where.remoteJid", "where.id"); v != "" {
		q = q.Where("jid = ?", v)
	}
	if err := q.Order("jid").Limit(5000).Find(&rows).Error; err != nil {
		v2Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]gin.H, 0, len(rows))
	for _, ct := range rows {
		name := ct.FullName
		if name == "" {
			name = ct.PushName
		}
		out = append(out, gin.H{"id": ct.JID, "remoteJid": ct.JID, "pushName": nilIfEmpty(name), "profilePicUrl": nil,
			"updatedAt": ct.UpdatedAt, "instanceId": i.Id, "isGroup": strings.HasSuffix(ct.JID, "@g.us")})
	}
	c.JSON(http.StatusOK, out)
}

// mediaBase64 descarga la media de un mensaje (del cuerpo o del historial).
func (s *Server) mediaBase64(c *gin.Context) {
	i := inst(c)
	b, ok := body(c)
	if !ok {
		return
	}
	msg := evoformat.Obj(b, "message.message")
	id := str(b, "message.key.id")
	if msg == nil && id != "" && s.opt.Store != nil {
		if m, err := s.opt.Store.GetMessage(i.Id, id); err == nil && len(m.Raw) > 0 {
			_ = json.Unmarshal(m.Raw, &msg)
		}
	}
	if msg == nil {
		v2Error(c, http.StatusBadRequest, "message not found: send message.message or enable CHAT_STORE_RAW")
		return
	}
	r := s.call(http.MethodPost, "/message/downloadmedia", i.Token, M{"message": msg})
	if !r.ok() {
		fail(c, r)
		return
	}
	var data struct {
		Base64 string `json:"base64"`
	}
	_ = r.decode(&data)
	mime, b64 := "", data.Base64
	if strings.HasPrefix(b64, "data:") {
		if k := strings.Index(b64, ","); k > 0 {
			mime = strings.TrimSuffix(strings.TrimPrefix(b64[:k], "data:"), ";base64")
			b64 = b64[k+1:]
		}
	}
	mt := evoformat.MessageType(msg)
	inner := evoformat.Obj(msg, mt)
	c.JSON(http.StatusCreated, gin.H{"mediaType": strings.TrimSuffix(mt, "Message"), "fileName": nilIfEmpty(str(inner, "fileName")),
		"caption": nilIfEmpty(str(inner, "caption")), "size": gin.H{"fileLength": inner["fileLength"], "height": inner["height"], "width": inner["width"]},
		"mimetype": mime, "base64": b64, "buffer": nil})
}
