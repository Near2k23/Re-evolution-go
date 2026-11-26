package compat

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/evolution-foundation/evolution-go/pkg/chatbot"
	chatbot_handler "github.com/evolution-foundation/evolution-go/pkg/chatbot/handler"
	instance_model "github.com/evolution-foundation/evolution-go/pkg/instance/model"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// tokenMW es un middleware de prueba: el token apikey identifica la línea.
type tokenMW struct{ db *gorm.DB }

func (m tokenMW) set(c *gin.Context) {
	var i instance_model.Instance
	if m.db.First(&i, "token = ?", c.GetHeader("apikey")).Error != nil {
		c.AbortWithStatus(401)
		return
	}
	c.Set("instance", &i)
	c.Next()
}
func (m tokenMW) Auth(c *gin.Context)                { m.set(c) }
func (m tokenMW) AuthAdmin(c *gin.Context)           { m.set(c) }
func (m tokenMW) AuthInstanceOrAdmin(c *gin.Context) { m.set(c) }

func TestV2Bots(t *testing.T) {
	db := pgDB(t)
	eng := chatbot.NewEngine(db, nil, nil)
	if err := eng.Migrate(); err != nil {
		t.Fatal(err)
	}
	db.Exec("DELETE FROM chatbots")
	gin.SetMode(gin.TestMode)
	r := gin.New()
	chatbot_handler.Register(r, tokenMW{db}, db, eng)
	var cwHost, cwProto string
	var cwBody map[string]interface{}
	r.PUT("/chatwoot/:id", func(c *gin.Context) {
		cwHost, cwProto = c.Request.Host, c.GetHeader("X-Forwarded-Proto")
		_ = c.ShouldBindJSON(&cwBody)
		c.JSON(200, gin.H{"data": gin.H{"enabled": true, "url": cwBody["url"], "accountId": cwBody["accountId"], "inboxName": cwBody["inboxName"],
			"ignoreGroups": cwBody["ignoreGroups"], "webhookUrl": "https://" + c.Request.Host + "/chatwoot/webhook/" + c.Param("id")}})
	})
	inst := instance_model.Instance{Id: uuid.NewString(), Name: "bots", Token: "tok-bots"}
	db.Create(&inst)
	s := New(Options{DB: db, GlobalKey: "global"}, r)

	// typebot: create → find → update → delete con la forma v2.
	w, out := do(s, http.MethodPost, "/typebot/create/bots", "tok-bots", map[string]interface{}{
		"enabled": true, "description": "Ventas", "url": "https://typebot.io", "typebot": "ventas-abc",
		"triggerType": "keyword", "triggerOperator": "contains", "triggerValue": "hola", "expire": 20, "debounceTime": 3,
		"ignoreJids": []string{"@g.us"}, "listeningFromMe": false, "stopBotFromMe": true})
	if w.Code != 201 || out["typebot"] != "ventas-abc" || out["debounceTime"].(float64) != 3 || out["triggerValue"] != "hola" {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	id := out["id"].(string)
	var stored chatbot.Bot
	db.First(&stored, "id = ?", id)
	if stored.Type != "typebot" || stored.AllowGroups || stored.DebounceMs != 3000 || !stored.StopOnFromMe || stored.ExpireMinutes != 20 {
		t.Fatalf("stored: %+v", stored)
	}
	req := httptest.NewRequest(http.MethodGet, "/typebot/find/bots", nil)
	req.Header.Set("apikey", "tok-bots")
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	var list []map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &list)
	if len(list) != 1 {
		t.Fatalf("find: %s", rec.Body.String())
	}
	// Un bot de otra familia no aparece en /typebot/find.
	if w, _ := do(s, http.MethodPost, "/n8n/create/bots", "tok-bots", map[string]interface{}{"description": "Flujo", "webhookUrl": "https://n8n/x", "triggerType": "all"}); w.Code != 201 {
		t.Fatalf("n8n create: %d", w.Code)
	}
	rec = httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	_ = json.Unmarshal(rec.Body.Bytes(), &list)
	if len(list) != 1 {
		t.Fatalf("typebot/find must not list n8n bots: %s", rec.Body.String())
	}
	if w, out := do(s, http.MethodPut, "/typebot/update/"+id+"/bots", "tok-bots", map[string]interface{}{"description": "Ventas 2", "triggerType": "none"}); w.Code != 200 ||
		out["description"] != "Ventas 2" || out["triggerType"] != "none" || out["url"] != "https://typebot.io" {
		t.Fatalf("update: %d %v", w.Code, out)
	}
	if w, _ := do(s, http.MethodPost, "/typebot/create/bots", "tok-bots", map[string]interface{}{"description": "x", "triggerType": "advanced"}); w.Code != 400 {
		t.Fatalf("advanced trigger must fail: %d", w.Code)
	}

	// openai con credenciales v2.
	_, cred := do(s, http.MethodPost, "/openai/creds/bots", "tok-bots", map[string]interface{}{"name": "main", "apiKey": "sk-real"})
	w, out = do(s, http.MethodPost, "/openai/create/bots", "tok-bots", map[string]interface{}{"description": "IA", "openaiCredsId": cred["id"],
		"botType": "chatCompletion", "model": "gpt-4o-mini", "systemMessages": []string{"Eres amable."}, "triggerType": "all"})
	if w.Code != 201 || out["openaiCredsId"] != cred["id"] || out["systemMessages"].([]interface{})[0] != "Eres amable." {
		t.Fatalf("openai: %d %s", w.Code, w.Body.String())
	}
	var ai chatbot.Bot
	db.First(&ai, "id = ?", out["id"])
	if !strings.Contains(string(ai.Config), "sk-real") {
		t.Fatalf("openai key not stored: %s", ai.Config)
	}

	if w, _ := do(s, http.MethodDelete, "/typebot/delete/"+id+"/bots", "tok-bots", nil); w.Code != 200 {
		t.Fatalf("delete: %d", w.Code)
	}

	// chatwoot/set traduce los campos y conserva el host público.
	req = httptest.NewRequest(http.MethodPost, "/chatwoot/set/bots", strings.NewReader(`{"enabled":true,"accountId":"7","token":"t","url":"https://cw.test","nameInbox":"Ventas","ignoreJids":["@g.us"]}`))
	req.Host = "api.publica.test"
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("apikey", "tok-bots")
	rec = httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != 201 || cwHost != "api.publica.test" || cwProto != "https" || cwBody["accountId"].(float64) != 7 || cwBody["inboxName"] != "Ventas" || cwBody["ignoreGroups"] != true {
		t.Fatalf("chatwoot/set: %d %s host=%q body=%v", rec.Code, rec.Body.String(), cwHost, cwBody)
	}
	if !strings.Contains(rec.Body.String(), `"webhook_url":"https://api.publica.test/chatwoot/webhook/`) {
		t.Fatalf("webhook_url: %s", rec.Body.String())
	}
}
