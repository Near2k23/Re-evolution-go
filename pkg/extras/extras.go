// Package extras añade rutas que Evolution GO no tenía: información de una
// invitación de grupo, mensajes temporales y perfil de empresa.
package extras

import (
	"context"
	"net/http"
	"strings"
	"time"

	instance_model "github.com/evolution-foundation/evolution-go/pkg/instance/model"
	auth_middleware "github.com/evolution-foundation/evolution-go/pkg/middleware"
	"github.com/evolution-foundation/evolution-go/pkg/utils"
	"github.com/gin-gonic/gin"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

// ClientGetter devuelve el cliente de una línea (o nil).
type ClientGetter func(instanceID string) *whatsmeow.Client

// Ephemeral son los tiempos que admite WhatsApp para mensajes temporales.
var Ephemeral = map[int64]bool{0: true, 86400: true, 604800: true, 7776000: true}

// InviteCode extrae el código de un enlace de invitación.
func InviteCode(s string) string {
	s = strings.TrimSpace(s)
	return s[strings.LastIndex(s, "/")+1:]
}

func client(c *gin.Context, get ClientGetter) *whatsmeow.Client {
	inst, ok := c.MustGet("instance").(*instance_model.Instance)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "instance not found"})
		return nil
	}
	cli := get(inst.Id)
	if cli == nil || !cli.IsConnected() || !cli.IsLoggedIn() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "instance is not connected"})
		return nil
	}
	return cli
}

func parseJID(raw, server string) (types.JID, bool) {
	if raw == "" {
		return types.JID{}, false
	}
	if !strings.Contains(raw, "@") {
		if server == types.GroupServer {
			raw += "@" + server
		} else if j, err := utils.CreateJID(raw); err == nil {
			raw = j
		}
	}
	return utils.ParseJID(raw)
}

// Register monta las rutas (token de la línea).
func Register(r *gin.Engine, mw auth_middleware.Middleware, get ClientGetter) {
	g := r.Group("", mw.Auth)

	// POST /group/inviteinfo {code} → datos del grupo de un enlace sin unirse.
	g.POST("/group/inviteinfo", func(c *gin.Context) {
		var b struct {
			Code string `json:"code"`
		}
		if err := c.ShouldBindJSON(&b); err != nil || InviteCode(b.Code) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "code is required"})
			return
		}
		cli := client(c, get)
		if cli == nil {
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
		defer cancel()
		info, err := cli.GetGroupInfoFromLink(ctx, InviteCode(b.Code))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "success", "data": info})
	})

	// POST /group/ephemeral {groupJid, expiration} → mensajes temporales
	// (0, 86400, 604800 o 7776000 segundos). Sirve también para chats.
	g.POST("/group/ephemeral", func(c *gin.Context) {
		var b struct {
			GroupJID   string `json:"groupJid"`
			Expiration int64  `json:"expiration"`
		}
		if err := c.ShouldBindJSON(&b); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		jid, ok := parseJID(b.GroupJID, types.GroupServer)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "groupJid is required"})
			return
		}
		if !Ephemeral[b.Expiration] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "expiration must be 0, 86400, 604800 or 7776000"})
			return
		}
		cli := client(c, get)
		if cli == nil {
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
		defer cancel()
		if err := cli.SetDisappearingTimer(ctx, jid, time.Duration(b.Expiration)*time.Second, time.Now()); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "success", "data": gin.H{"jid": jid.String(), "expiration": b.Expiration}})
	})

	// POST /user/business {number} → perfil de empresa (vacío si no es business).
	g.POST("/user/business", func(c *gin.Context) {
		var b struct {
			Number string `json:"number"`
		}
		_ = c.ShouldBindJSON(&b)
		jid, ok := parseJID(b.Number, types.DefaultUserServer)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "number is required"})
			return
		}
		cli := client(c, get)
		if cli == nil {
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
		defer cancel()
		p, err := cli.GetBusinessProfile(ctx, jid)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "success", "data": p})
	})
}
