// Package chatstore_handler expone el almacén de chats por HTTP:
// /store/:instanceId/{status,chats,messages,messages/:id,contacts,read}.
package chatstore_handler

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/evolution-foundation/evolution-go/pkg/chatstore"
	instance_model "github.com/evolution-foundation/evolution-go/pkg/instance/model"
	auth_middleware "github.com/evolution-foundation/evolution-go/pkg/middleware"
	"github.com/gin-gonic/gin"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

// ClientGetter devuelve el cliente de whatsmeow de una instancia (o nil).
type ClientGetter func(instanceID string) *whatsmeow.Client

func clamp(v string, def, max int) int {
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return def
	}
	if n > max {
		return max
	}
	return n
}

func parseBefore(v string) *time.Time {
	if v == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339Nano, v)
	if err != nil {
		return nil
	}
	return &t
}

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

// Register monta /store/:instanceId/... (clave global o token de esa instancia).
func Register(r *gin.Engine, mw auth_middleware.Middleware, s *chatstore.Store, client ClientGetter) {
	g := r.Group("/store/:instanceId", mw.AuthInstanceOrAdmin, func(c *gin.Context) {
		if !canAccess(c, c.Param("instanceId")) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.Next()
	})

	g.GET("/status", func(c *gin.Context) {
		cfg := s.Config()
		c.JSON(http.StatusOK, gin.H{"message": "success", "data": gin.H{"enabled": cfg.Enabled, "history": cfg.History, "retentionDays": cfg.RetentionDays}})
	})

	g.GET("/chats", func(c *gin.Context) {
		list, err := s.ListChats(c.Param("instanceId"), c.Query("q"), c.Query("archived") == "true", clamp(c.Query("limit"), 50, 200), parseBefore(c.Query("before")))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "success", "data": list})
	})

	g.GET("/messages", func(c *gin.Context) {
		chat := c.Query("chat")
		if chat == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "chat is required"})
			return
		}
		list, err := s.ListMessages(c.Param("instanceId"), chat, clamp(c.Query("limit"), 50, 200), parseBefore(c.Query("before")))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "success", "data": list})
	})

	g.GET("/messages/:messageId", func(c *gin.Context) {
		m, err := s.GetMessage(c.Param("instanceId"), c.Param("messageId"))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "message not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "success", "data": m})
	})

	g.GET("/contacts", func(c *gin.Context) {
		list, err := s.ListContacts(c.Param("instanceId"), c.Query("q"), clamp(c.Query("limit"), 100, 500))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "success", "data": list})
	})

	// Marca el chat como leído aquí y, si la línea está conectada, envía los
	// acuses de lectura a WhatsApp.
	g.POST("/read", func(c *gin.Context) {
		var b struct {
			Chat string `json:"chat"`
		}
		if err := c.ShouldBindJSON(&b); err != nil || b.Chat == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "chat is required"})
			return
		}
		id := c.Param("instanceId")
		msgs, err := s.MarkRead(id, b.Chat)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		receipts := 0
		if cli := client(id); cli != nil && cli.IsConnected() && len(msgs) > 0 {
			chatJID, err := types.ParseJID(b.Chat)
			if err == nil {
				bySender := map[string][]types.MessageID{}
				for _, m := range msgs {
					bySender[m.SenderJID] = append(bySender[m.SenderJID], m.MessageID)
				}
				for sender, ids := range bySender {
					sj, _ := types.ParseJID(sender)
					if chatJID.Server != types.GroupServer {
						sj = types.EmptyJID
					}
					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					if cli.MarkRead(ctx, ids, time.Now(), chatJID, sj) == nil {
						receipts += len(ids)
					}
					cancel()
				}
			}
		}
		c.JSON(http.StatusOK, gin.H{"message": "success", "data": gin.H{"receipts": receipts}})
	})
}
