// Package webhooks_handler expone la configuración de webhooks por HTTP:
// GET/PUT /webhook/:instanceId, POST /webhook/:instanceId/test y
// GET /webhook/:instanceId/deliveries.
package webhooks_handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	instance_model "github.com/evolution-foundation/evolution-go/pkg/instance/model"
	instance_service "github.com/evolution-foundation/evolution-go/pkg/instance/service"
	auth_middleware "github.com/evolution-foundation/evolution-go/pkg/middleware"
	"github.com/evolution-foundation/evolution-go/pkg/webhooks"
	"github.com/gin-gonic/gin"
)

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

type view struct {
	URL        string            `json:"url"`
	Events     []string          `json:"events"`
	Headers    map[string]string `json:"headers"`
	ByEvents   bool              `json:"byEvents"`
	Format     string            `json:"format"`
	HasSecret  bool              `json:"hasSecret"`
	Secret     string            `json:"secret,omitempty"` // solo al generarlo
	Stats24h   webhooks.Stats    `json:"stats24h"`
	GlobalURL  bool              `json:"globalUrl"`
	Retry      map[string]int64  `json:"retry"`
	SignatureV string            `json:"signature"`
}

func Register(r *gin.Engine, mw auth_middleware.Middleware, svc *webhooks.Service, instances instance_service.InstanceService, globalURL string) {
	g := r.Group("/webhook/:instanceId", mw.AuthInstanceOrAdmin, func(c *gin.Context) {
		if !canAccess(c, c.Param("instanceId")) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.Next()
	})

	build := func(id string, inst *instance_model.Instance) view {
		v := view{URL: inst.Webhook, Headers: map[string]string{}, GlobalURL: globalURL != "", SignatureV: "sha256=hex(HMAC_SHA256(secret, X-Webhook-Timestamp + \".\" + body))"}
		for _, e := range strings.Split(inst.Events, ",") {
			if e = strings.TrimSpace(e); e != "" {
				v.Events = append(v.Events, e)
			}
		}
		if cfg := svc.Get(id); cfg != nil {
			if cfg.Headers != nil {
				v.Headers = cfg.Headers
			}
			v.ByEvents, v.HasSecret, v.Format = cfg.ByEvents, cfg.Secret != "", cfg.Format
		}
		v.Stats24h = svc.Stats(id, time.Now().Add(-24*time.Hour))
		st := svc.Settings()
		v.Retry = map[string]int64{"maxAttempts": int64(st.MaxAttempts), "baseDelayMs": st.BaseDelay.Milliseconds(), "timeoutMs": st.Timeout.Milliseconds()}
		return v
	}

	g.GET("", func(c *gin.Context) {
		id := c.Param("instanceId")
		inst, err := instances.Info(id)
		if err != nil || inst == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "success", "data": build(id, inst)})
	})

	g.PUT("", func(c *gin.Context) {
		id := c.Param("instanceId")
		inst, err := instances.Info(id)
		if err != nil || inst == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
			return
		}
		var b struct {
			Headers  map[string]string `json:"headers"`
			ByEvents bool              `json:"byEvents"`
			// Format: "" (Evolution GO) o "evolution-api"; nil = no cambiar.
			Format *string `json:"format"`
			// Secret: "keep" (por defecto), "new" genera uno nuevo, "remove" lo quita.
			Secret string `json:"secret"`
		}
		if err := c.ShouldBindJSON(&b); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if b.Format != nil && !webhooks.ValidFormat(*b.Format) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "format must be \"\" or \"evolution-api\""})
			return
		}
		var secret *string
		generated := ""
		switch b.Secret {
		case "new":
			generated = webhooks.NewSecret()
			secret = &generated
		case "remove":
			empty := ""
			secret = &empty
		}
		if _, err := svc.Save(id, b.Headers, b.ByEvents, secret); err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, webhooks.ErrInvalidHeader) {
				status = http.StatusBadRequest
			}
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}
		if b.Format != nil {
			if err := svc.SetFormat(id, *b.Format); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}
		v := build(id, inst)
		v.Secret = generated
		c.JSON(http.StatusOK, gin.H{"message": "success", "data": v})
	})

	g.POST("/test", func(c *gin.Context) {
		id := c.Param("instanceId")
		inst, err := instances.Info(id)
		if err != nil || inst == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
			return
		}
		if inst.Webhook == "" || inst.Webhook == "disabled" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "this instance has no webhook URL"})
			return
		}
		payload, _ := json.Marshal(gin.H{
			"event":        "Test",
			"instanceId":   inst.Id,
			"instanceName": inst.Name,
			"data":         gin.H{"message": "Test event", "sentAt": time.Now().UTC().Format(time.RFC3339)},
		})
		res := svc.Deliver(svc.TargetFor(id, inst.Webhook), payload, true)
		c.JSON(http.StatusOK, gin.H{"message": "success", "data": res})
	})

	g.GET("/deliveries", func(c *gin.Context) {
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
		if limit <= 0 || limit > 200 {
			limit = 50
		}
		list, err := svc.Recent(c.Param("instanceId"), limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "success", "data": list})
	})
}
