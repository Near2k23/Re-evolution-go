package compat

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/evolution-foundation/evolution-go/pkg/evoformat"
	instance_model "github.com/evolution-foundation/evolution-go/pkg/instance/model"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// OpenAICred es una credencial de OpenAI con la forma de v2 (/openai/creds).
type OpenAICred struct {
	ID         string    `json:"id" gorm:"type:uuid;primaryKey"`
	InstanceID string    `json:"instanceId" gorm:"size:64;index;not null"`
	Name       string    `json:"name"`
	APIKey     string    `json:"-"`
	CreatedAt  time.Time `json:"createdAt"`
}

func (OpenAICred) TableName() string { return "compat_openai_creds" }

// botKinds asigna cada familia de rutas v2 al tipo de bot de Go.
var botKinds = map[string]string{
	"typebot": "typebot", "openai": "openai", "dify": "dify", "flowise": "flowise", "n8n": "n8n", "evolutionBot": "http",
}

// v2Bot convierte un bot de Go (JSON de /bots) en el objeto de v2.
func v2Bot(kind string, b M) gin.H {
	cfg := evoformat.Obj(b, "config")
	if cfg == nil {
		cfg = M{}
	}
	trigger := str(b, "triggerType")
	if trigger == "keyword" && strings.TrimSpace(str(b, "triggerValue")) == "" {
		trigger = "none"
	}
	out := gin.H{
		"id": str(b, "id"), "enabled": b["enabled"], "description": str(b, "name"), "expire": int(num(b, "expireMinutes")),
		"keywordFinish": str(b, "keywordFinish"), "delayMessage": int(num(b, "delayMs")), "unknownMessage": str(b, "unknownMessage"),
		"listeningFromMe": b["listenFromMe"], "stopBotFromMe": b["stopOnFromMe"], "keepOpen": false,
		"debounceTime": int(num(b, "debounceMs")) / 1000, "ignoreJids": b["ignoreJids"], "triggerType": trigger,
		"triggerOperator": str(b, "triggerOperator"), "triggerValue": str(b, "triggerValue"), "splitMessages": b["splitMessages"],
		"timePerChar": 0, "instanceId": str(b, "instanceId"), "createdAt": b["createdAt"], "updatedAt": b["updatedAt"],
	}
	switch kind {
	case "typebot":
		out["url"], out["typebot"] = str(cfg, "url"), str(cfg, "publicId")
	case "dify":
		out["apiUrl"], out["apiKey"] = str(cfg, "apiUrl"), str(cfg, "apiKey")
		out["botType"] = map[string]string{"chat": "chatBot", "completion": "textGenerator", "agent": "agent", "workflow": "workflow"}[str(cfg, "botType")]
	case "flowise", "evolutionBot":
		out["apiUrl"], out["apiKey"] = str(cfg, "url"), str(cfg, "apiKey")
	case "n8n":
		out["webhookUrl"], out["basicAuthUser"], out["basicAuthPass"] = str(cfg, "url"), str(cfg, "username"), str(cfg, "password")
	case "openai":
		out["openaiCredsId"], out["botType"], out["model"] = str(cfg, "credsId"), "chatCompletion", str(cfg, "model")
		out["systemMessages"] = []string{}
		if p := str(cfg, "systemPrompt"); p != "" {
			out["systemMessages"] = []string{p}
		}
		out["maxTokens"] = int(num(cfg, "maxTokens"))
	}
	return out
}

// fromV2 aplica un cuerpo v2 sobre un bot de Go (cur: el actual o vacío).
func (s *Server) fromV2(kind string, i *instance_model.Instance, b, cur M) (M, string) {
	out := M{}
	for k, v := range cur {
		out[k] = v
	}
	out["type"] = botKinds[kind]
	cfg := M{}
	if c := evoformat.Obj(cur, "config"); c != nil {
		for k, v := range c {
			cfg[k] = v
		}
	}
	set := func(v2, goKey string, conv func(interface{}) interface{}) {
		if v, ok := b[v2]; ok && v != nil {
			if conv != nil {
				v = conv(v)
			}
			out[goKey] = v
		}
	}
	id := func(v interface{}) interface{} { return v }
	set("description", "name", id)
	set("enabled", "enabled", id)
	set("expire", "expireMinutes", id)
	set("keywordFinish", "keywordFinish", id)
	set("delayMessage", "delayMs", id)
	set("unknownMessage", "unknownMessage", id)
	set("listeningFromMe", "listenFromMe", id)
	set("stopBotFromMe", "stopOnFromMe", id)
	set("splitMessages", "splitMessages", id)
	set("debounceTime", "debounceMs", func(v interface{}) interface{} { f, _ := v.(float64); return int(f * 1000) })
	set("triggerOperator", "triggerOperator", func(v interface{}) interface{} {
		if v == "none" {
			return "contains"
		}
		return v
	})
	set("triggerValue", "triggerValue", id)
	if _, ok := b["ignoreJids"]; ok {
		jids := strs(b, "ignoreJids")
		out["ignoreJids"] = jids
		allow := true
		for _, j := range jids {
			if j == "@g.us" {
				allow = false
			}
		}
		out["allowGroups"] = allow
	} else if cur == nil || cur["allowGroups"] == nil {
		out["allowGroups"] = true // en v2 los bots responden en grupos salvo que se ignoren
	}
	switch str(b, "triggerType") {
	case "":
	case "all":
		out["triggerType"] = "all"
	case "keyword":
		out["triggerType"] = "keyword"
	case "none":
		out["triggerType"], out["triggerValue"] = "keyword", "" // solo por sesiones abiertas a mano
	default:
		return nil, "triggerType " + str(b, "triggerType") + " is not supported (use all, keyword or none)"
	}
	if str(out, "name") == "" {
		return nil, "description is required"
	}

	cs := func(v2, goKey string) {
		if v, ok := b[v2]; ok && v != nil {
			cfg[goKey] = v
		}
	}
	switch kind {
	case "typebot":
		cs("url", "url")
		cs("typebot", "publicId")
	case "dify":
		cs("apiUrl", "apiUrl")
		cs("apiKey", "apiKey")
		if bt := str(b, "botType"); bt != "" {
			cfg["botType"] = map[string]string{"chatBot": "chat", "textGenerator": "completion", "agent": "agent", "workflow": "workflow"}[bt]
		}
	case "flowise", "evolutionBot":
		cs("apiUrl", "url")
		cs("apiKey", "apiKey")
	case "n8n":
		cs("webhookUrl", "url")
		cs("basicAuthUser", "username")
		cs("basicAuthPass", "password")
	case "openai":
		if bt := str(b, "botType"); bt != "" && bt != "chatCompletion" {
			return nil, "only botType chatCompletion is supported"
		}
		if credID := str(b, "openaiCredsId"); credID != "" {
			var cred OpenAICred
			if s.opt.DB.First(&cred, "id = ? AND instance_id = ?", credID, i.Id).Error != nil {
				return nil, "openaiCredsId not found"
			}
			cfg["apiKey"], cfg["credsId"] = cred.APIKey, cred.ID
		}
		cs("model", "model")
		cs("maxTokens", "maxTokens")
		if _, ok := b["systemMessages"]; ok {
			cfg["systemPrompt"] = strings.Join(strs(b, "systemMessages"), "\n")
		}
	}
	out["config"] = cfg
	return out, ""
}

// botsOf lista los bots de Go de una familia v2.
func (s *Server) botsOf(i *instance_model.Instance, kind string) ([]M, result) {
	r := s.call(http.MethodGet, "/bots/"+i.Id, i.Token, nil)
	var all []M
	if r.ok() {
		_ = r.decode(&all)
	}
	out := make([]M, 0, len(all))
	for _, b := range all {
		if str(b, "type") == botKinds[kind] {
			out = append(out, b)
		}
	}
	return out, r
}

func (s *Server) botByID(i *instance_model.Instance, kind, id string) M {
	list, _ := s.botsOf(i, kind)
	for _, b := range list {
		if str(b, "id") == id {
			return b
		}
	}
	return nil
}

func v2SessionStatus(st string) string {
	return map[string]string{"open": "opened", "paused": "paused", "closed": "closed"}[st]
}

func (s *Server) registerBots() {
	a := s.auth
	for kind := range botKinds {
		kind := kind
		p := func(method, path string, h gin.HandlerFunc) {
			s.handle(method, "/"+kind+"/"+path+"/:instanceName", a, h)
		}

		p(http.MethodPost, "create", func(c *gin.Context) {
			i := inst(c)
			b, ok := body(c)
			if !ok {
				return
			}
			in, msg := s.fromV2(kind, i, b, M{"enabled": true})
			if msg != "" {
				v2Error(c, http.StatusBadRequest, msg)
				return
			}
			r := s.call(http.MethodPost, "/bots/"+i.Id, i.Token, in)
			if !r.ok() {
				fail(c, r)
				return
			}
			var created M
			_ = r.decode(&created)
			c.JSON(http.StatusCreated, v2Bot(kind, created))
		})
		p(http.MethodGet, "find", func(c *gin.Context) {
			list, r := s.botsOf(inst(c), kind)
			if !r.ok() {
				fail(c, r)
				return
			}
			out := make([]gin.H, 0, len(list))
			for _, b := range list {
				out = append(out, v2Bot(kind, b))
			}
			c.JSON(http.StatusOK, out)
		})
		p(http.MethodGet, "fetch/:botId", func(c *gin.Context) {
			b := s.botByID(inst(c), kind, c.Param("botId"))
			if b == nil {
				v2Error(c, http.StatusNotFound, "bot not found")
				return
			}
			c.JSON(http.StatusOK, v2Bot(kind, b))
		})
		p(http.MethodPut, "update/:botId", func(c *gin.Context) {
			i := inst(c)
			cur := s.botByID(i, kind, c.Param("botId"))
			if cur == nil {
				v2Error(c, http.StatusNotFound, "bot not found")
				return
			}
			b, ok := body(c)
			if !ok {
				return
			}
			in, msg := s.fromV2(kind, i, b, cur)
			if msg != "" {
				v2Error(c, http.StatusBadRequest, msg)
				return
			}
			r := s.call(http.MethodPut, "/bots/"+i.Id+"/"+c.Param("botId"), i.Token, in)
			if !r.ok() {
				fail(c, r)
				return
			}
			var upd M
			_ = r.decode(&upd)
			c.JSON(http.StatusOK, v2Bot(kind, upd))
		})
		p(http.MethodDelete, "delete/:botId", func(c *gin.Context) {
			i := inst(c)
			if s.botByID(i, kind, c.Param("botId")) == nil {
				v2Error(c, http.StatusNotFound, "bot not found")
				return
			}
			if r := s.call(http.MethodDelete, "/bots/"+i.Id+"/"+c.Param("botId"), i.Token, nil); !r.ok() {
				fail(c, r)
				return
			}
			c.JSON(http.StatusOK, gin.H{"bot": gin.H{"id": c.Param("botId")}})
		})
		p(http.MethodGet, "fetchSessions/:botId", func(c *gin.Context) {
			i := inst(c)
			r := s.call(http.MethodGet, "/bots/"+i.Id+"/sessions?all=true&botId="+c.Param("botId"), i.Token, nil)
			if !r.ok() {
				fail(c, r)
				return
			}
			var list []M
			_ = r.decode(&list)
			out := make([]gin.H, 0, len(list))
			for _, ss := range list {
				out = append(out, gin.H{"id": str(ss, "id"), "remoteJid": str(ss, "remoteJid"), "pushName": str(ss, "pushName"),
					"sessionId": str(ss, "externalId"), "status": v2SessionStatus(str(ss, "status")), "awaitUser": false,
					"botId": str(ss, "botId"), "instanceId": i.Id, "createdAt": ss["createdAt"], "updatedAt": ss["lastActivity"]})
			}
			c.JSON(http.StatusOK, out)
		})
		// changeStatus {remoteJid, status: opened|paused|closed|delete}.
		p(http.MethodPost, "changeStatus", func(c *gin.Context) {
			i := inst(c)
			b, ok := body(c)
			if !ok {
				return
			}
			jid, status := str(b, "remoteJid"), str(b, "status")
			goStatus := map[string]string{"opened": "open", "paused": "paused", "closed": "closed", "delete": "closed"}[status]
			if jid == "" || goStatus == "" {
				v2Error(c, http.StatusBadRequest, "remoteJid and status (opened, paused, closed or delete) are required")
				return
			}
			r := s.call(http.MethodGet, "/bots/"+i.Id+"/sessions", i.Token, nil)
			var list []M
			_ = r.decode(&list)
			changed := 0
			for _, ss := range list {
				bot := s.botByID(i, kind, str(ss, "botId"))
				if str(ss, "remoteJid") != toJID(jid) && str(ss, "remoteJid") != jid || bot == nil {
					continue
				}
				if rr := s.call(http.MethodPost, "/bots/"+i.Id+"/sessions/"+str(ss, "id"), i.Token, M{"status": goStatus}); rr.ok() {
					changed++
				}
			}
			c.JSON(http.StatusOK, gin.H{kind: gin.H{"remoteJid": jid, "status": status, "sessions": changed}})
		})
		unsupported := func(c *gin.Context) {
			v2Error(c, http.StatusBadRequest, "per-instance bot settings are not supported: set expire, delays, ignoreJids… on each bot")
		}
		p(http.MethodPost, "settings", unsupported)
		p(http.MethodPost, "ignoreJid", unsupported)
		p(http.MethodGet, "fetchSettings", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"expire": 0, "keywordFinish": "", "delayMessage": 0, "unknownMessage": "", "listeningFromMe": false,
				"stopBotFromMe": false, "keepOpen": false, "debounceTime": 0, "ignoreJids": []string{}, "splitMessages": false, "timePerChar": 0, "fallback": nil})
		})
	}

	// Credenciales de OpenAI.
	s.handle(http.MethodPost, "/openai/creds/:instanceName", a, func(c *gin.Context) {
		i := inst(c)
		b, ok := body(c)
		if !ok {
			return
		}
		name, key := strings.TrimSpace(str(b, "name")), strings.TrimSpace(str(b, "apiKey"))
		if name == "" || key == "" {
			v2Error(c, http.StatusBadRequest, "name and apiKey are required")
			return
		}
		cred := OpenAICred{ID: uuid.NewString(), InstanceID: i.Id, Name: name, APIKey: key, CreatedAt: time.Now()}
		if err := s.opt.DB.Create(&cred).Error; err != nil {
			v2Error(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusCreated, gin.H{"id": cred.ID, "name": cred.Name, "apiKey": "********", "instanceId": i.Id, "createdAt": cred.CreatedAt})
	})
	s.handle(http.MethodGet, "/openai/creds/:instanceName", a, func(c *gin.Context) {
		var list []OpenAICred
		s.opt.DB.Where("instance_id = ?", inst(c).Id).Order("created_at").Find(&list)
		out := make([]gin.H, 0, len(list))
		for _, cr := range list {
			out = append(out, gin.H{"id": cr.ID, "name": cr.Name, "apiKey": "********", "instanceId": cr.InstanceID, "createdAt": cr.CreatedAt})
		}
		c.JSON(http.StatusOK, out)
	})
	s.handle(http.MethodDelete, "/openai/creds/:credsId/:instanceName", a, func(c *gin.Context) {
		res := s.opt.DB.Where("id = ? AND instance_id = ?", c.Param("credsId"), inst(c).Id).Delete(&OpenAICred{})
		if res.RowsAffected == 0 {
			v2Error(c, http.StatusNotFound, "credentials not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"openaiCreds": gin.H{"id": c.Param("credsId")}})
	})
	s.handle(http.MethodGet, "/openai/getModels/:instanceName", a, func(c *gin.Context) {
		c.JSON(http.StatusOK, []gin.H{{"id": "gpt-4o-mini"}, {"id": "gpt-4o"}, {"id": "gpt-4.1-mini"}, {"id": "gpt-4.1"}})
	})

	// Chatwoot con la forma de v2.
	s.handle(http.MethodPost, "/chatwoot/set/:instanceName", a, s.setChatwoot)
	s.handle(http.MethodGet, "/chatwoot/find/:instanceName", a, s.findChatwoot)
}

func chatwootV2(cfg M) gin.H {
	return gin.H{"enabled": cfg["enabled"], "accountId": jsonString(cfg["accountId"]), "token": "********", "url": str(cfg, "url"),
		"nameInbox": str(cfg, "inboxName"), "signMsg": cfg["signMessages"], "signDelimiter": "\n", "number": nil,
		"reopenConversation": cfg["reopenConversation"], "conversationPending": cfg["conversationPending"], "mergeBrazilContacts": false,
		"importContacts": false, "importMessages": false, "daysLimitImportMessages": 0, "autoCreate": true, "organization": "", "logo": "",
		"ignoreJids": map[bool][]string{true: {"@g.us"}, false: {}}[cfg["ignoreGroups"] == true], "webhook_url": str(cfg, "webhookUrl")}
}

func jsonString(v interface{}) string {
	if f, ok := v.(float64); ok {
		b, _ := json.Marshal(int64(f))
		return string(b)
	}
	s, _ := v.(string)
	return s
}

func (s *Server) setChatwoot(c *gin.Context) {
	i := inst(c)
	b, ok := body(c)
	if !ok {
		return
	}
	enabled, has := boolOf(b, "enabled")
	if !has {
		enabled = true
	}
	ignoreGroups := false
	for _, j := range strs(b, "ignoreJids") {
		if j == "@g.us" {
			ignoreGroups = true
		}
	}
	sign, _ := boolOf(b, "signMsg")
	reopen, _ := boolOf(b, "reopenConversation")
	pending, _ := boolOf(b, "conversationPending")
	in := M{"enabled": enabled, "url": str(b, "url"), "accountId": int(num(b, "accountId")), "token": str(b, "token"),
		"inboxName": str(b, "nameInbox"), "signMessages": sign, "reopenConversation": reopen, "conversationPending": pending, "ignoreGroups": ignoreGroups}
	r := s.callFrom(c, http.MethodPut, "/chatwoot/"+i.Id, i.Token, in)
	if !r.ok() {
		fail(c, r)
		return
	}
	var cfg M
	_ = r.decode(&cfg)
	c.JSON(http.StatusCreated, chatwootV2(cfg))
}

func (s *Server) findChatwoot(c *gin.Context) {
	i := inst(c)
	r := s.callFrom(c, http.MethodGet, "/chatwoot/"+i.Id, i.Token, nil)
	if !r.ok() {
		fail(c, r)
		return
	}
	var cfg M
	_ = r.decode(&cfg)
	c.JSON(http.StatusOK, chatwootV2(cfg))
}
