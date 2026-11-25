// Package overview expone al manager el estado de las líneas: conectada,
// reconectada, reconectando o caída, con su historial de cortes.
package overview

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/evolution-foundation/evolution-go/pkg/connlog"
	instance_model "github.com/evolution-foundation/evolution-go/pkg/instance/model"
	instance_service "github.com/evolution-foundation/evolution-go/pkg/instance/service"
	auth_middleware "github.com/evolution-foundation/evolution-go/pkg/middleware"
	"github.com/gin-gonic/gin"
)

// Line es una línea tal como la ve el manager.
type Line struct {
	ID               string     `json:"id"`
	Name             string     `json:"name"`
	Token            string     `json:"token"`
	Jid              string     `json:"jid"`
	Phone            string     `json:"phone"`
	State            string     `json:"state"`
	NeedsPairing     bool       `json:"needsPairing"`
	Webhook          string     `json:"webhook"`
	Events           string     `json:"events"`
	DisconnectReason string     `json:"disconnectReason"`
	CreatedAt        time.Time  `json:"createdAt"`
	Cuts24h          int        `json:"cuts24h"`
	LongestCutSec    int        `json:"longestCutSec"`
	LastCutAt        *time.Time `json:"lastCutAt,omitempty"`
	LastReconnectAt  *time.Time `json:"lastReconnectAt,omitempty"`
	ConnectedSince   *time.Time `json:"connectedSince,omitempty"`
	LastKind         string     `json:"lastKind,omitempty"`
	LastReason       string     `json:"lastReason,omitempty"`
	LastEventAt      *time.Time `json:"lastEventAt,omitempty"`
	Hours            []string   `json:"hours"`
	GroupID          string     `json:"groupId,omitempty"`
}

// GroupLister devuelve instancia → grupo.
type GroupLister func() (map[string]string, error)

// DeviceLister devuelve los JID con sesión guardada en el almacén de whatsmeow.
type DeviceLister func() (map[string]bool, error)

type handler struct {
	instances instance_service.InstanceService
	repo      *connlog.Repository
	devices   DeviceLister
	groups    GroupLister
}

// Register monta las rutas del manager.
func Register(r *gin.Engine, mw auth_middleware.Middleware, instances instance_service.InstanceService, repo *connlog.Repository, devices DeviceLister, groups GroupLister) {
	h := &handler{instances: instances, repo: repo, devices: devices, groups: groups}
	admin := r.Group("/instance", mw.AuthAdmin)
	admin.GET("/overview", h.overview)
	admin.POST("/reconnect/:instanceId", h.reconnect)

	scoped := r.Group("/instance", mw.AuthInstanceOrAdmin)
	scoped.GET("/history/:instanceId", h.history)
}

// phoneFromJid toma el número de "573001234567:3@s.whatsapp.net".
func phoneFromJid(jid string) string {
	user := jid
	if i := strings.IndexAny(user, ":@"); i >= 0 {
		user = user[:i]
	}
	return user
}

func (h *handler) overview(c *gin.Context) {
	list, err := h.instances.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	now := time.Now()
	recent, err := h.repo.ListAll(now.Add(-24 * time.Hour))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	last, err := h.repo.LastByInstance()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Sin acceso al almacén (nil) se asume que el dispositivo existe.
	var stored map[string]bool
	if h.devices != nil {
		stored, _ = h.devices()
	}

	var members map[string]string
	if h.groups != nil {
		members, _ = h.groups()
	}

	out := make([]Line, 0, len(list))
	for _, inst := range list {
		hasDevice := stored == nil || stored[inst.Jid]
		l := h.line(inst, recent[inst.Id], last, now, hasDevice)
		l.GroupID = members[inst.Id]
		out = append(out, l)
	}
	c.JSON(http.StatusOK, gin.H{"message": "success", "data": out})
}

func (h *handler) line(inst *instance_model.Instance, evts []connlog.ConnectionEvent, last map[string]connlog.ConnectionEvent, now time.Time, hasDevice bool) Line {
	var lastEvt *connlog.ConnectionEvent
	if e, ok := last[inst.Id]; ok {
		lastEvt = &e
	}
	s := connlog.Summarize(evts, lastEvt, now)
	exists, connected, loggedIn := h.instances.LiveState(inst.Id)
	live := connlog.LiveState{Exists: exists, Connected: connected, LoggedIn: loggedIn}
	state := connlog.Classify(live, inst.Jid, s, hasDevice)
	upNow := state == connlog.StateConnected || state == connlog.StateReconnected

	return Line{
		ID:               inst.Id,
		Name:             inst.Name,
		Token:            inst.Token,
		Jid:              inst.Jid,
		Phone:            phoneFromJid(inst.Jid),
		State:            state,
		NeedsPairing:     connlog.NeedsPairing(inst.Jid, s, hasDevice),
		Webhook:          inst.Webhook,
		Events:           inst.Events,
		DisconnectReason: inst.DisconnectReason,
		CreatedAt:        inst.CreatedAt,
		Cuts24h:          s.Cuts24h,
		LongestCutSec:    s.LongestCutSec,
		LastCutAt:        s.LastCutAt,
		LastReconnectAt:  s.LastReconnectAt,
		ConnectedSince:   s.ConnectedSince,
		LastKind:         s.LastKind,
		LastReason:       s.LastReason,
		LastEventAt:      s.LastEventAt,
		Hours:            connlog.HourBlocks(evts, upNow, now),
	}
}

func (h *handler) reconnect(c *gin.Context) {
	inst, err := h.instances.Info(c.Param("instanceId"))
	if err != nil || inst == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}
	if err := h.instances.Reconnect(inst); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "success"})
}

func (h *handler) history(c *gin.Context) {
	id := c.Param("instanceId")
	if !canAccess(c, id) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))
	if days < 1 || days > 30 {
		days = 7
	}
	evts, err := h.repo.List(id, time.Now().Add(-time.Duration(days)*24*time.Hour))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "success", "data": gin.H{"events": evts, "cuts": connlog.Cuts(evts)}})
}

func canAccess(c *gin.Context, instanceId string) bool {
	if v, ok := c.Get(auth_middleware.ContextKeyIsAdmin); ok && v == true {
		return true
	}
	v, ok := c.Get("instance")
	if !ok {
		return false
	}
	inst, ok := v.(*instance_model.Instance)
	return ok && inst != nil && inst.Id == instanceId
}
