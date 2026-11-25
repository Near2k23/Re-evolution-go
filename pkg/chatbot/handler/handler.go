// Package chatbot_handler expone los bots por HTTP: /bots/:instanceId/...
package chatbot_handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/evolution-foundation/evolution-go/pkg/chatbot"
	instance_model "github.com/evolution-foundation/evolution-go/pkg/instance/model"
	auth_middleware "github.com/evolution-foundation/evolution-go/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Mask sustituye los secretos de la configuración al devolverla.
const Mask = "********"

var secretKeys = []string{"apiKey", "password"}

func canAccess(c *gin.Context, instanceID string) bool {
	if v, ok := c.Get(auth_middleware.ContextKeyIsAdmin); ok && v == true {
		return true
	}
	v, ok := c.Get("instance")
	if !ok {
		return false
	}
	inst, ok := v.(*instance_model.Instance)
	return ok && inst != nil && inst.Id == instanceID
}

func maskConfig(raw json.RawMessage) json.RawMessage {
	var m map[string]interface{}
	if json.Unmarshal(raw, &m) != nil {
		return raw
	}
	for _, k := range secretKeys {
		if v, ok := m[k].(string); ok && v != "" {
			m[k] = Mask
		}
	}
	b, _ := json.Marshal(m)
	return b
}

// mergeSecrets conserva los secretos que llegan enmascarados.
func mergeSecrets(newRaw, oldRaw json.RawMessage) json.RawMessage {
	var nm, om map[string]interface{}
	if json.Unmarshal(newRaw, &nm) != nil {
		return newRaw
	}
	_ = json.Unmarshal(oldRaw, &om)
	for _, k := range secretKeys {
		if v, ok := nm[k].(string); ok && v == Mask {
			nm[k] = om[k]
		}
	}
	b, _ := json.Marshal(nm)
	return b
}

type botInput struct {
	Type            string          `json:"type"`
	Name            string          `json:"name"`
	Enabled         bool            `json:"enabled"`
	Priority        int             `json:"priority"`
	TriggerType     string          `json:"triggerType"`
	TriggerOperator string          `json:"triggerOperator"`
	TriggerValue    string          `json:"triggerValue"`
	AllowGroups     bool            `json:"allowGroups"`
	ListenFromMe    bool            `json:"listenFromMe"`
	StopOnFromMe    bool            `json:"stopOnFromMe"`
	KeywordFinish   string          `json:"keywordFinish"`
	ExpireMinutes   int             `json:"expireMinutes"`
	DelayMs         int             `json:"delayMs"`
	DebounceMs      int             `json:"debounceMs"`
	SplitMessages   bool            `json:"splitMessages"`
	UnknownMessage  string          `json:"unknownMessage"`
	IgnoreJIDs      []string        `json:"ignoreJids"`
	Config          json.RawMessage `json:"config"`
}

func validType(t string) bool {
	for _, x := range chatbot.Types {
		if x == t {
			return true
		}
	}
	return false
}

func apply(b *chatbot.Bot, in botInput) {
	b.Type, b.Name, b.Enabled, b.Priority = in.Type, strings.TrimSpace(in.Name), in.Enabled, in.Priority
	b.TriggerType, b.TriggerOperator, b.TriggerValue = in.TriggerType, in.TriggerOperator, in.TriggerValue
	b.AllowGroups, b.ListenFromMe, b.StopOnFromMe = in.AllowGroups, in.ListenFromMe, in.StopOnFromMe
	b.KeywordFinish, b.ExpireMinutes, b.DelayMs, b.DebounceMs = strings.TrimSpace(in.KeywordFinish), in.ExpireMinutes, in.DelayMs, in.DebounceMs
	b.SplitMessages, b.UnknownMessage, b.IgnoreJIDs = in.SplitMessages, in.UnknownMessage, in.IgnoreJIDs
	if b.TriggerType == "" {
		b.TriggerType = chatbot.TriggerAll
	}
	if b.TriggerOperator == "" {
		b.TriggerOperator = chatbot.OpContains
	}
}

func view(b chatbot.Bot) chatbot.Bot {
	b.Config = maskConfig(b.Config)
	if b.IgnoreJIDs == nil {
		b.IgnoreJIDs = []string{}
	}
	return b
}

func Register(r *gin.Engine, mw auth_middleware.Middleware, db *gorm.DB, eng *chatbot.Engine) {
	g := r.Group("/bots/:instanceId", mw.AuthInstanceOrAdmin, func(c *gin.Context) {
		if !canAccess(c, c.Param("instanceId")) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.Next()
	})

	g.GET("", func(c *gin.Context) {
		var list []chatbot.Bot
		if err := db.Where("instance_id = ?", c.Param("instanceId")).Order("priority DESC, created_at").Find(&list).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		out := make([]chatbot.Bot, 0, len(list))
		for _, b := range list {
			out = append(out, view(b))
		}
		c.JSON(http.StatusOK, gin.H{"message": "success", "data": out})
	})

	g.POST("", func(c *gin.Context) {
		var in botInput
		if err := c.ShouldBindJSON(&in); err != nil || !validType(in.Type) || strings.TrimSpace(in.Name) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "type and name are required; type must be one of " + strings.Join(chatbot.Types, ", ")})
			return
		}
		b := chatbot.Bot{ID: uuid.NewString(), InstanceID: c.Param("instanceId"), CreatedAt: time.Now(), UpdatedAt: time.Now()}
		apply(&b, in)
		b.Config = in.Config
		if len(b.Config) == 0 {
			b.Config = json.RawMessage(`{}`)
		}
		if err := db.Create(&b).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		eng.Invalidate(b.InstanceID)
		c.JSON(http.StatusOK, gin.H{"message": "success", "data": view(b)})
	})

	g.PUT("/:botId", func(c *gin.Context) {
		var b chatbot.Bot
		if err := db.First(&b, "id = ? AND instance_id = ?", c.Param("botId"), c.Param("instanceId")).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "bot not found"})
			return
		}
		var in botInput
		if err := c.ShouldBindJSON(&in); err != nil || !validType(in.Type) || strings.TrimSpace(in.Name) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "type and name are required"})
			return
		}
		old := b.Config
		apply(&b, in)
		if len(in.Config) > 0 {
			b.Config = mergeSecrets(in.Config, old)
		}
		b.UpdatedAt = time.Now()
		if err := db.Save(&b).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		eng.Invalidate(b.InstanceID)
		c.JSON(http.StatusOK, gin.H{"message": "success", "data": view(b)})
	})

	g.DELETE("/:botId", func(c *gin.Context) {
		id, inst := c.Param("botId"), c.Param("instanceId")
		db.Where("bot_id = ? AND instance_id = ?", id, inst).Delete(&chatbot.Session{})
		res := db.Where("id = ? AND instance_id = ?", id, inst).Delete(&chatbot.Bot{})
		if res.RowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "bot not found"})
			return
		}
		eng.Invalidate(inst)
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	g.GET("/sessions", func(c *gin.Context) {
		q := db.Where("instance_id = ?", c.Param("instanceId"))
		if id := c.Query("botId"); id != "" {
			q = q.Where("bot_id = ?", id)
		}
		if c.Query("all") != "true" {
			q = q.Where("status <> ?", chatbot.SessionClosed)
		}
		var list []chatbot.Session
		if err := q.Order("last_activity DESC").Limit(200).Find(&list).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "success", "data": list})
	})

	// Cambia el estado de una sesión: open (reanudar), paused (pausar) o closed.
	g.POST("/sessions/:sessionId", func(c *gin.Context) {
		var b struct {
			Status string `json:"status"`
		}
		if err := c.ShouldBindJSON(&b); err != nil || (b.Status != chatbot.SessionOpen && b.Status != chatbot.SessionPaused && b.Status != chatbot.SessionClosed) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "status must be open, paused or closed"})
			return
		}
		res := db.Model(&chatbot.Session{}).Where("id = ? AND instance_id = ?", c.Param("sessionId"), c.Param("instanceId")).
			Updates(map[string]interface{}{"status": b.Status, "last_activity": time.Now()})
		if res.RowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})
}
