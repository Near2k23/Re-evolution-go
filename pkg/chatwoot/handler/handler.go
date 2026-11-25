// Package chatwoot_handler expone la integración con Chatwoot por HTTP.
package chatwoot_handler

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/evolution-foundation/evolution-go/pkg/chatwoot"
	instance_model "github.com/evolution-foundation/evolution-go/pkg/instance/model"
	auth_middleware "github.com/evolution-foundation/evolution-go/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/gomessguii/logger"
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

// publicBase deduce la URL pública del servidor a partir de la petición.
func publicBase(c *gin.Context) string {
	proto := c.GetHeader("X-Forwarded-Proto")
	if proto == "" {
		proto = "http"
		if c.Request.TLS != nil {
			proto = "https"
		}
	}
	host := c.GetHeader("X-Forwarded-Host")
	if host == "" {
		host = c.Request.Host
	}
	return proto + "://" + host
}

type view struct {
	*chatwoot.Config
	HasToken   bool   `json:"hasToken"`
	WebhookURL string `json:"webhookUrl,omitempty"`
}

func Register(r *gin.Engine, mw auth_middleware.Middleware, s *chatwoot.Service) {
	// Webhook público de la bandeja (lo protege el secreto de la URL).
	r.POST("/chatwoot/webhook/:instanceId", func(c *gin.Context) {
		var ev chatwoot.WebhookEvent
		if err := c.ShouldBindJSON(&ev); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := s.HandleWebhook(c.Param("instanceId"), c.Query("secret"), ev); err != nil {
			if errors.Is(err, chatwoot.ErrUnauthorized) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
				return
			}
			logger.LogError("[chatwoot] webhook: %v", err)
			c.JSON(http.StatusOK, gin.H{"message": "error", "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	g := r.Group("/chatwoot/:instanceId", mw.AuthInstanceOrAdmin, func(c *gin.Context) {
		if !canAccess(c, c.Param("instanceId")) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.Next()
	})

	build := func(c *gin.Context, cfg *chatwoot.Config) view {
		if cfg == nil {
			return view{Config: &chatwoot.Config{}}
		}
		v := view{Config: cfg, HasToken: cfg.Token != ""}
		if cfg.WebhookSecret != "" {
			v.WebhookURL = chatwoot.WebhookURL(publicBase(c), c.Param("instanceId"), cfg.WebhookSecret)
		}
		return v
	}

	g.GET("", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success", "data": build(c, s.Get(c.Param("instanceId")))})
	})

	g.PUT("", func(c *gin.Context) {
		var in chatwoot.SaveInput
		if err := c.ShouldBindJSON(&in); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
		defer cancel()
		cfg, err := s.Save(ctx, c.Param("instanceId"), publicBase(c), in)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "success", "data": build(c, cfg)})
	})

	// Prueba la conexión (URL, cuenta y token; sin token usa el guardado).
	g.POST("/test", func(c *gin.Context) {
		var in struct {
			URL       string `json:"url"`
			AccountID int    `json:"accountId"`
			Token     string `json:"token"`
		}
		if err := c.ShouldBindJSON(&in); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if in.Token == "" {
			if cfg := s.Get(c.Param("instanceId")); cfg != nil {
				in.Token = cfg.Token
			}
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
		defer cancel()
		if err := s.TestConnection(ctx, strings.TrimSpace(in.URL), in.AccountID, in.Token); err != nil {
			c.JSON(http.StatusOK, gin.H{"message": "success", "data": gin.H{"ok": false, "error": err.Error()}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "success", "data": gin.H{"ok": true}})
	})
}
