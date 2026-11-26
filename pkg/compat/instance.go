package compat

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/evolution-foundation/evolution-go/pkg/evoformat"
	instance_model "github.com/evolution-foundation/evolution-go/pkg/instance/model"
	"github.com/evolution-foundation/evolution-go/pkg/webhooks"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.mau.fi/whatsmeow/types"
)

const integration = "WHATSAPP-BAILEYS"

// ---------- utilidades de cuerpo ----------

type M = evoformat.M

func str(m M, keys ...string) string {
	for _, k := range keys {
		if v := evoformat.Get(m, k); v != nil {
			switch t := v.(type) {
			case string:
				if t != "" {
					return t
				}
			case float64:
				return strconv.FormatFloat(t, 'f', -1, 64)
			}
		}
	}
	return ""
}

func num(m M, keys ...string) float64 {
	for _, k := range keys {
		switch t := evoformat.Get(m, k).(type) {
		case float64:
			return t
		case string:
			var f float64
			if json.Unmarshal([]byte(t), &f) == nil {
				return f
			}
		}
	}
	return 0
}

func boolOf(m M, keys ...string) (bool, bool) {
	for _, k := range keys {
		switch t := evoformat.Get(m, k).(type) {
		case bool:
			return t, true
		case string:
			return t == "true", true
		}
	}
	return false, false
}

func strs(m M, key string) []string {
	var out []string
	switch t := evoformat.Get(m, key).(type) {
	case []interface{}:
		for _, v := range t {
			if s, ok := v.(string); ok && s != "" {
				out = append(out, s)
			}
		}
	case string:
		for _, s := range strings.Split(t, ",") {
			if s = strings.TrimSpace(s); s != "" {
				out = append(out, s)
			}
		}
	}
	return out
}

// bareJID quita el dispositivo de un JID ("57300:12@s.whatsapp.net" → "57300@s.whatsapp.net").
func bareJID(jid string) string {
	if i := strings.Index(jid, ":"); i > 0 {
		if at := strings.Index(jid, "@"); at > i {
			return jid[:i] + jid[at:]
		}
	}
	return jid
}

// ---------- estado ----------

// state devuelve open, connecting o close, como v2.
func (s *Server) state(i *instance_model.Instance) string {
	r := s.call(http.MethodGet, "/instance/status", i.Token, nil)
	var st struct{ Connected, LoggedIn bool }
	_ = r.decode(&st)
	switch {
	case st.Connected && st.LoggedIn:
		return "open"
	case st.Connected || st.LoggedIn:
		return "connecting"
	}
	return "close"
}

func (s *Server) connectionState(c *gin.Context) {
	i := inst(c)
	c.JSON(http.StatusOK, gin.H{"instance": gin.H{"instanceName": i.Name, "state": s.state(i)}})
}

// startClient arranca la línea sin cambiar su webhook ni sus eventos.
func (s *Server) startClient(i *instance_model.Instance) result {
	var subs []string
	for _, e := range strings.Split(i.Events, ",") {
		if e = strings.TrimSpace(e); e != "" {
			subs = append(subs, e)
		}
	}
	return s.call(http.MethodPost, "/instance/connect", i.Token, M{
		"webhookUrl": i.Webhook, "subscribe": subs,
		"rabbitmqEnable": i.RabbitmqEnable, "websocketEnable": i.WebSocketEnable, "natsEnable": i.NatsEnable,
	})
}

// qrcode pide el QR (y el código de emparejamiento si hay número).
func (s *Server) qrcode(i *instance_model.Instance, number string) gin.H {
	out := gin.H{"pairingCode": nil, "code": nil, "base64": nil, "count": 1}
	r := s.call(http.MethodGet, "/instance/qr", i.Token, nil)
	var q struct{ Qrcode, Code string }
	if r.ok() && r.decode(&q) == nil {
		out["base64"], out["code"] = q.Qrcode, q.Code
	}
	if number != "" {
		p := s.call(http.MethodPost, "/instance/pair", i.Token, M{"phone": number})
		var pr struct{ PairingCode string }
		if p.ok() && p.decode(&pr) == nil && pr.PairingCode != "" {
			out["pairingCode"] = pr.PairingCode
		}
	}
	return out
}

func (s *Server) connect(c *gin.Context) {
	i := inst(c)
	if st := s.state(i); st == "open" {
		c.JSON(http.StatusOK, gin.H{"instance": gin.H{"instanceName": i.Name, "state": st}})
		return
	}
	if r := s.startClient(i); !r.ok() {
		fail(c, r)
		return
	}
	c.JSON(http.StatusOK, s.qrcode(i, c.Query("number")))
}

func (s *Server) restart(c *gin.Context) {
	i := inst(c)
	if s.opt.Client != nil && s.opt.Client(i.Id) == nil {
		if r := s.startClient(i); !r.ok() {
			fail(c, r)
			return
		}
	} else if r := s.call(http.MethodPost, "/instance/reconnect", i.Token, nil); !r.ok() {
		fail(c, r)
		return
	}
	time.Sleep(2 * time.Second)
	c.JSON(http.StatusOK, gin.H{"instance": gin.H{"instanceName": i.Name, "status": s.state(i)}})
}

func (s *Server) logout(c *gin.Context) {
	i := inst(c)
	if r := s.call(http.MethodDelete, "/instance/logout", i.Token, nil); !r.ok() {
		fail(c, r)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "SUCCESS", "error": false, "response": gin.H{"message": "Instance logged out"}})
}

// deleteInstance: DELETE /instance/delete/:x existe en ambas APIs. Con el ID de
// una línea y la clave global es la ruta Go; con un nombre, la de v2.
func (s *Server) deleteInstance(c *gin.Context) {
	param := c.Param("instanceName")
	if s.isAdmin(c) {
		var n int64
		s.opt.DB.Model(&instance_model.Instance{}).Where("id = ?", param).Count(&n)
		if n > 0 && uuidLike(param) {
			s.passToGo(c, nil)
			return
		}
	}
	s.auth(c)
	if c.IsAborted() {
		return
	}
	i := inst(c)
	if r := s.call(http.MethodDelete, "/instance/delete/"+i.Id, s.opt.GlobalKey, nil); !r.ok() {
		fail(c, r)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "SUCCESS", "error": false, "response": gin.H{"message": "Instance deleted"}})
}

func uuidLike(s string) bool { _, err := uuid.Parse(s); return err == nil }

func (s *Server) setPresence(c *gin.Context) {
	i := inst(c)
	b, ok := body(c)
	if !ok {
		return
	}
	p := str(b, "presence")
	if p != "available" && p != "unavailable" {
		v2Error(c, http.StatusBadRequest, "presence must be available or unavailable")
		return
	}
	cli := s.opt.Client(i.Id)
	if cli == nil {
		v2Error(c, http.StatusBadRequest, "instance is not connected")
		return
	}
	if err := cli.SendPresence(context.Background(), types.Presence(p)); err != nil {
		v2Error(c, http.StatusBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusCreated, gin.H{"presence": p})
}

// ---------- listado ----------

func (s *Server) fetchInstances(c *gin.Context) {
	key := c.GetHeader("apikey")
	q := s.opt.DB.Model(&instance_model.Instance{})
	if key == "" {
		v2Error(c, http.StatusUnauthorized, "Unauthorized")
		return
	}
	if !s.isAdmin(c) {
		q = q.Where("token = ?", key)
	} else if s.opt.ClientName != "" {
		q = q.Where("client_name = ?", s.opt.ClientName)
	}
	if n := c.Query("instanceName"); n != "" {
		q = q.Where("name = ?", n)
	}
	if id := c.Query("instanceId"); id != "" {
		q = q.Where("id = ?", id)
	}
	var list []instance_model.Instance
	if err := q.Order("created_at").Find(&list).Error; err != nil {
		v2Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	if !s.isAdmin(c) && len(list) == 0 {
		v2Error(c, http.StatusUnauthorized, "Unauthorized")
		return
	}
	counts := s.counts()
	out := make([]gin.H, 0, len(list))
	for idx := range list {
		out = append(out, s.instanceView(&list[idx], counts))
	}
	c.JSON(http.StatusOK, out)
}

type tally struct{ Message, Contact, Chat int64 }

// counts cuenta mensajes, contactos y chats guardados por línea.
func (s *Server) counts() map[string]*tally {
	out := map[string]*tally{}
	if s.opt.Store == nil {
		return out
	}
	type row struct {
		InstanceID string
		N          int64
	}
	for _, t := range []struct {
		table string
		set   func(*tally, int64)
	}{
		{"chat_store_messages", func(x *tally, n int64) { x.Message = n }},
		{"chat_store_contacts", func(x *tally, n int64) { x.Contact = n }},
		{"chat_store_chats", func(x *tally, n int64) { x.Chat = n }},
	} {
		var rows []row
		if s.opt.DB.Table(t.table).Select("instance_id, count(*) as n").Group("instance_id").Scan(&rows).Error != nil {
			continue
		}
		for _, r := range rows {
			if out[r.InstanceID] == nil {
				out[r.InstanceID] = &tally{}
			}
			t.set(out[r.InstanceID], r.N)
		}
	}
	return out
}

func (s *Server) instanceView(i *instance_model.Instance, counts map[string]*tally) gin.H {
	status := "close"
	profile := ""
	if s.opt.Client != nil {
		if cli := s.opt.Client(i.Id); cli != nil {
			switch {
			case cli.IsConnected() && cli.IsLoggedIn():
				status = "open"
			case cli.IsConnected() || cli.IsLoggedIn():
				status = "connecting"
			}
			if cli.Store != nil {
				profile = cli.Store.PushName
			}
		}
	}
	owner := bareJID(i.Jid)
	number := strings.Split(owner, "@")[0]
	t := counts[i.Id]
	if t == nil {
		t = &tally{}
	}
	return gin.H{
		"id": i.Id, "name": i.Name, "connectionStatus": status, "ownerJid": nilIfEmpty(owner), "profileName": nilIfEmpty(profile),
		"profilePicUrl": nil, "integration": integration, "number": nilIfEmpty(number), "businessId": nil, "token": i.Token,
		"clientName": i.ClientName, "disconnectionReasonCode": nil, "disconnectionObject": nilIfEmpty(i.DisconnectReason),
		"disconnectionAt": nil, "createdAt": i.CreatedAt, "updatedAt": i.CreatedAt,
		"Setting": settingsView(i),
		"_count":  gin.H{"Message": t.Message, "Contact": t.Contact, "Chat": t.Chat},
	}
}

func nilIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func settingsView(i *instance_model.Instance) gin.H {
	return gin.H{
		"rejectCall": i.RejectCall, "msgCall": i.MsgRejectCall, "groupsIgnore": i.IgnoreGroups, "alwaysOnline": i.AlwaysOnline,
		"readMessages": i.ReadMessages, "readStatus": i.ReadStatus, "syncFullHistory": true,
	}
}

// ---------- creación ----------

func (s *Server) createInstance(c *gin.Context) {
	raw, _ := c.GetRawData()
	var b M
	_ = json.Unmarshal(raw, &b)
	name := str(b, "instanceName")
	if name == "" { // cuerpo de Evolution GO ({name, token})
		s.passToGo(c, raw)
		return
	}
	s.admin(c)
	if c.IsAborted() {
		return
	}
	if s.instanceByName(name) != nil {
		v2Error(c, http.StatusForbidden, `This name "`+name+`" is already in use.`)
		return
	}
	token := str(b, "token")
	if token == "" {
		token = strings.ToUpper(uuid.NewString())
	}
	create := M{"name": name, "token": token, "advancedSettings": advancedFrom(b, nil)}
	if id := str(b, "instanceId"); id != "" && uuidLike(id) {
		create["instanceId"] = id
	}
	if h := str(b, "proxyHost"); h != "" {
		create["proxy"] = M{"host": h, "port": str(b, "proxyPort"), "protocol": str(b, "proxyProtocol"),
			"username": str(b, "proxyUsername"), "password": str(b, "proxyPassword")}
	}
	r := s.call(http.MethodPost, "/instance/create", s.opt.GlobalKey, create)
	if !r.ok() {
		fail(c, r)
		return
	}
	var created instance_model.Instance
	if err := r.decode(&created); err != nil || created.Id == "" {
		v2Error(c, http.StatusInternalServerError, "could not read the created instance")
		return
	}
	i := &created
	if wh := webhookFrom(b); wh != nil {
		if err := s.applyWebhook(i, wh); err != nil {
			v2Error(c, http.StatusBadRequest, err.Error())
			return
		}
	}
	resp := gin.H{
		"instance": gin.H{"instanceName": i.Name, "instanceId": i.Id, "integration": integration, "webhookWaBusiness": nil,
			"accessTokenWaBusiness": "", "status": "created"},
		"hash":      token,
		"webhook":   s.webhookView(i),
		"websocket": gin.H{"enabled": i.WebSocketEnable == "enabled" || i.WebSocketEnable == "true"},
		"rabbitmq":  gin.H{"enabled": i.RabbitmqEnable == "enabled" || i.RabbitmqEnable == "true"},
		"nats":      gin.H{"enabled": i.NatsEnable == "enabled" || i.NatsEnable == "true"},
		"sqs":       gin.H{"enabled": false},
		"settings":  settingsView(i),
	}
	if q, _ := boolOf(b, "qrcode"); q || str(b, "number") != "" {
		if r := s.startClient(i); r.ok() {
			qr := s.qrcode(i, str(b, "number"))
			resp["qrcode"] = qr
			resp["instance"].(gin.H)["status"] = "connecting"
		}
	}
	c.JSON(http.StatusCreated, resp)
}

// advancedFrom arma los ajustes de Go desde un cuerpo v2, sobre los actuales.
func advancedFrom(b M, cur *instance_model.AdvancedSettings) M {
	if cur == nil {
		cur = &instance_model.AdvancedSettings{}
	}
	out := M{"alwaysOnline": cur.AlwaysOnline, "rejectCall": cur.RejectCall, "msgRejectCall": cur.MsgRejectCall,
		"readMessages": cur.ReadMessages, "ignoreGroups": cur.IgnoreGroups, "ignoreStatus": cur.IgnoreStatus, "readStatus": cur.ReadStatus}
	for v2, goKey := range map[string]string{"alwaysOnline": "alwaysOnline", "rejectCall": "rejectCall", "readMessages": "readMessages",
		"groupsIgnore": "ignoreGroups", "readStatus": "readStatus", "ignoreStatus": "ignoreStatus"} {
		if v, ok := boolOf(b, v2); ok {
			out[goKey] = v
		}
	}
	if _, ok := b["msgCall"]; ok {
		out["msgRejectCall"] = str(b, "msgCall")
	}
	return out
}

// ---------- ajustes ----------

func (s *Server) currentSettings(i *instance_model.Instance) *instance_model.AdvancedSettings {
	r := s.call(http.MethodGet, "/instance/"+i.Id+"/advanced-settings", i.Token, nil)
	var st instance_model.AdvancedSettings
	if r.ok() {
		_ = r.decode(&st)
	}
	return &st
}

func settingsOut(st *instance_model.AdvancedSettings) gin.H {
	return gin.H{"rejectCall": st.RejectCall, "msgCall": st.MsgRejectCall, "groupsIgnore": st.IgnoreGroups,
		"alwaysOnline": st.AlwaysOnline, "readMessages": st.ReadMessages, "readStatus": st.ReadStatus, "syncFullHistory": true}
}

func (s *Server) setSettings(c *gin.Context) {
	i := inst(c)
	b, ok := body(c)
	if !ok {
		return
	}
	if inner := evoformat.Obj(b, "settings"); inner != nil {
		b = inner
	}
	next := advancedFrom(b, s.currentSettings(i))
	if r := s.call(http.MethodPut, "/instance/"+i.Id+"/advanced-settings", i.Token, next); !r.ok() {
		fail(c, r)
		return
	}
	out := settingsOut(s.currentSettings(i))
	out["instanceName"] = i.Name
	c.JSON(http.StatusCreated, gin.H{"settings": out})
}

func (s *Server) findSettings(c *gin.Context) {
	c.JSON(http.StatusOK, settingsOut(s.currentSettings(inst(c))))
}

// ---------- proxy ----------

func (s *Server) setProxy(c *gin.Context) {
	i := inst(c)
	b, ok := body(c)
	if !ok {
		return
	}
	if inner := evoformat.Obj(b, "proxy"); inner != nil {
		b = inner
	}
	enabled, has := boolOf(b, "enabled")
	if has && !enabled {
		if r := s.call(http.MethodDelete, "/instance/proxy/"+i.Id, s.opt.GlobalKey, nil); !r.ok() {
			fail(c, r)
			return
		}
		c.JSON(http.StatusCreated, gin.H{"enabled": false, "host": "", "port": "", "protocol": "http", "username": "", "password": ""})
		return
	}
	p := M{"host": str(b, "host"), "port": str(b, "port"), "protocol": str(b, "protocol"), "username": str(b, "username"), "password": str(b, "password")}
	if r := s.call(http.MethodPost, "/instance/proxy/"+i.Id, s.opt.GlobalKey, p); !r.ok() {
		fail(c, r)
		return
	}
	p["enabled"] = true
	c.JSON(http.StatusCreated, p)
}

func (s *Server) findProxy(c *gin.Context) {
	i := inst(c)
	r := s.call(http.MethodGet, "/instance/proxy/"+i.Id, s.opt.GlobalKey, nil)
	var p map[string]interface{}
	if !r.ok() || r.decode(&p) != nil || p == nil {
		c.JSON(http.StatusOK, gin.H{"enabled": false, "host": "", "port": "", "protocol": "http", "username": "", "password": ""})
		return
	}
	c.JSON(http.StatusOK, gin.H{"enabled": true, "host": p["host"], "port": p["port"], "protocol": p["protocol"], "username": p["username"], "password": ""})
}

// ---------- webhook ----------

type webhookCfg struct {
	Enabled  bool
	URL      string
	Events   []string
	Headers  map[string]string
	ByEvents bool
	Base64   bool
}

// webhookFrom lee el webhook de un cuerpo v2 ({webhook:{…}} o los campos sueltos).
func webhookFrom(b M) *webhookCfg {
	w := evoformat.Obj(b, "webhook")
	if w == nil {
		if u, ok := b["webhook"].(string); ok && u != "" {
			w = M{"url": u, "enabled": true, "byEvents": b["webhookByEvents"], "events": b["events"]}
			if w["byEvents"] == nil {
				w["byEvents"] = b["webhook_by_events"]
			}
		} else if str(b, "url") != "" || b["enabled"] != nil {
			w = b
		}
	}
	if w == nil {
		return nil
	}
	cfg := &webhookCfg{URL: str(w, "url"), Events: strs(w, "events"), Headers: map[string]string{}}
	enabled, has := boolOf(w, "enabled")
	cfg.Enabled = enabled || (!has && cfg.URL != "")
	cfg.ByEvents, _ = boolOf(w, "byEvents", "webhookByEvents", "webhook_by_events")
	cfg.Base64, _ = boolOf(w, "base64", "webhookBase64", "webhook_base64")
	if h := evoformat.Obj(w, "headers"); h != nil {
		for k, v := range h {
			if s, ok := v.(string); ok {
				cfg.Headers[k] = s
			}
		}
	}
	return cfg
}

// applyWebhook guarda el webhook de una línea y lo deja en formato v2.
func (s *Server) applyWebhook(i *instance_model.Instance, w *webhookCfg) error {
	url := ""
	if w.Enabled {
		url = w.URL
	}
	events := strings.Join(evoformat.GroupsFor(w.Events), ",")
	if err := s.opt.DB.Model(&instance_model.Instance{}).Where("id = ?", i.Id).
		Updates(map[string]interface{}{"webhook": url, "events": events}).Error; err != nil {
		return err
	}
	i.Webhook, i.Events = url, events
	if s.opt.Webhooks != nil {
		if _, err := s.opt.Webhooks.Save(i.Id, w.Headers, w.ByEvents, nil); err != nil {
			return err
		}
		if err := s.opt.Webhooks.SetFormat(i.Id, webhooks.FormatEvolutionAPI); err != nil {
			return err
		}
		if err := s.opt.Webhooks.SetBase64(i.Id, w.Base64); err != nil {
			return err
		}
	}
	if s.opt.Reload != nil {
		_ = s.opt.Reload(i.Id) // si la línea no está en ejecución, la toma al arrancar
	}
	return nil
}

func (s *Server) webhookView(i *instance_model.Instance) gin.H {
	v := gin.H{"enabled": i.Webhook != "" && i.Webhook != "disabled", "url": i.Webhook, "webhookUrl": i.Webhook,
		"events": evoformat.EventsFor(strings.Split(i.Events, ",")), "headers": gin.H{}, "webhookHeaders": gin.H{},
		"byEvents": false, "webhookByEvents": false, "base64": false, "webhookBase64": false}
	if s.opt.Webhooks != nil {
		if cfg := s.opt.Webhooks.Get(i.Id); cfg != nil {
			v["headers"], v["webhookHeaders"] = cfg.Headers, cfg.Headers
			v["byEvents"], v["webhookByEvents"] = cfg.ByEvents, cfg.ByEvents
			v["base64"], v["webhookBase64"] = cfg.Base64, cfg.Base64
		}
	}
	return v
}

func (s *Server) setWebhook(c *gin.Context) {
	i := inst(c)
	b, ok := body(c)
	if !ok {
		return
	}
	w := webhookFrom(b)
	if w == nil {
		v2Error(c, http.StatusBadRequest, "webhook is required")
		return
	}
	if w.Enabled && w.URL == "" {
		v2Error(c, http.StatusBadRequest, "url is required")
		return
	}
	if err := s.applyWebhook(i, w); err != nil {
		v2Error(c, http.StatusBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusCreated, s.webhookView(i))
}

func (s *Server) findWebhook(c *gin.Context) {
	c.JSON(http.StatusOK, s.webhookView(inst(c)))
}

// ---------- websocket, rabbitmq y nats ----------

// destination es un destino de eventos de la línea (columna de la instancia).
type destination struct {
	column    string
	available func(o Options) bool
	get       func(i *instance_model.Instance) string
	set       func(i *instance_model.Instance, v string)
}

var destinations = map[string]destination{
	"websocket": {"web_socket_enable", func(Options) bool { return true },
		func(i *instance_model.Instance) string { return i.WebSocketEnable }, func(i *instance_model.Instance, v string) { i.WebSocketEnable = v }},
	"rabbitmq": {"rabbitmq_enable", func(o Options) bool { return o.RabbitMQ },
		func(i *instance_model.Instance) string { return i.RabbitmqEnable }, func(i *instance_model.Instance, v string) { i.RabbitmqEnable = v }},
	"nats": {"nats_enable", func(o Options) bool { return o.NATS },
		func(i *instance_model.Instance) string { return i.NatsEnable }, func(i *instance_model.Instance, v string) { i.NatsEnable = v }},
}

func enabledFlag(v string) bool { return v == "enabled" || v == "true" }

func (s *Server) destinationView(i *instance_model.Instance, d destination) gin.H {
	return gin.H{"enabled": enabledFlag(d.get(i)), "events": evoformat.EventsFor(strings.Split(i.Events, ","))}
}

// setDestination activa o desactiva un destino. Los eventos de Evolution GO
// son una sola lista por línea (compartida con el webhook): si llegan, la
// reemplazan.
func (s *Server) setDestination(name string) gin.HandlerFunc {
	d := destinations[name]
	return func(c *gin.Context) {
		i := inst(c)
		b, ok := body(c)
		if !ok {
			return
		}
		if inner := evoformat.Obj(b, name); inner != nil {
			b = inner
		}
		enabled, _ := boolOf(b, "enabled")
		if enabled && !d.available(s.opt) {
			v2Error(c, http.StatusBadRequest, name+" is not configured on this server")
			return
		}
		val := ""
		if enabled {
			val = "enabled"
		}
		upd := map[string]interface{}{d.column: val}
		if ev := strs(b, "events"); len(ev) > 0 {
			upd["events"] = strings.Join(evoformat.GroupsFor(ev), ",")
		}
		if err := s.opt.DB.Model(&instance_model.Instance{}).Where("id = ?", i.Id).Updates(upd).Error; err != nil {
			v2Error(c, http.StatusInternalServerError, err.Error())
			return
		}
		d.set(i, val)
		if e, ok := upd["events"].(string); ok {
			i.Events = e
		}
		if s.opt.Reload != nil {
			_ = s.opt.Reload(i.Id)
		}
		c.JSON(http.StatusCreated, gin.H{name: gin.H{"instanceName": i.Name, name: s.destinationView(i, d)}})
	}
}

func (s *Server) findDestination(name string) gin.HandlerFunc {
	d := destinations[name]
	return func(c *gin.Context) { c.JSON(http.StatusOK, s.destinationView(inst(c), d)) }
}
