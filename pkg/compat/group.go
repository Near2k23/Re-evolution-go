package compat

import (
	"net/http"
	"strings"

	"github.com/evolution-foundation/evolution-go/pkg/evoformat"
	instance_model "github.com/evolution-foundation/evolution-go/pkg/instance/model"
	"github.com/gin-gonic/gin"
)

// groupView convierte un types.GroupInfo (JSON de Go) en un grupo de v2.
func groupView(g M, withParticipants bool) gin.H {
	parts, _ := g["Participants"].([]interface{})
	out := gin.H{
		"id": evoformat.Str(g, "JID"), "subject": evoformat.Str(g, "Name"), "subjectOwner": nilIfEmpty(evoformat.Str(g, "NameSetBy")),
		"subjectTime": evoformat.Unix(g["NameSetAt"]), "pictureUrl": nil, "size": len(parts), "creation": evoformat.Unix(g["GroupCreated"]),
		"owner": nilIfEmpty(evoformat.Str(g, "OwnerJID")), "desc": nilIfEmpty(evoformat.Str(g, "Topic")), "descId": nilIfEmpty(evoformat.Str(g, "TopicID")),
		"restrict": evoformat.Bool(g, "IsLocked"), "announce": evoformat.Bool(g, "IsAnnounce"), "isCommunity": evoformat.Bool(g, "IsParent"),
	}
	if withParticipants {
		list := make([]gin.H, 0, len(parts))
		for _, p := range parts {
			pm, _ := p.(map[string]interface{})
			if pm == nil {
				continue
			}
			var admin interface{}
			if evoformat.Bool(pm, "IsSuperAdmin") {
				admin = "superadmin"
			} else if evoformat.Bool(pm, "IsAdmin") {
				admin = "admin"
			}
			item := gin.H{"id": evoformat.Str(pm, "JID"), "admin": admin}
			if pn := evoformat.Str(pm, "PhoneNumber"); pn != "" {
				item["phoneNumber"] = pn
			}
			if lid := evoformat.Str(pm, "LID"); lid != "" {
				item["lid"] = lid
			}
			list = append(list, item)
		}
		out["participants"] = list
	}
	return out
}

func groupJID(c *gin.Context) string {
	g := c.Query("groupJid")
	if g != "" && !strings.Contains(g, "@") {
		g += "@g.us"
	}
	return g
}

func (s *Server) groupInfo(i *instance_model.Instance, jid string) (M, result) {
	r := s.call(http.MethodPost, "/group/info", i.Token, M{"groupJid": jid})
	var g M
	if r.ok() {
		_ = r.decode(&g)
	}
	return g, r
}

func (s *Server) registerGroups() {
	a := s.auth
	p := func(method, name string, h gin.HandlerFunc) { s.handle(method, "/group/"+name+"/:instanceName", a, h) }
	needJID := func(h func(*gin.Context, *instance_model.Instance, string)) gin.HandlerFunc {
		return func(c *gin.Context) {
			jid := groupJID(c)
			if jid == "" {
				v2Error(c, http.StatusBadRequest, "groupJid is required")
				return
			}
			h(c, inst(c), jid)
		}
	}
	update := func(goPath string, build func(b M, jid string) M) gin.HandlerFunc {
		return needJID(func(c *gin.Context, i *instance_model.Instance, jid string) {
			b, ok := body(c)
			if !ok {
				return
			}
			if r := s.call(http.MethodPost, goPath, i.Token, build(b, jid)); !r.ok() {
				fail(c, r)
				return
			}
			c.JSON(http.StatusCreated, gin.H{"update": "success"})
		})
	}

	p(http.MethodPost, "create", func(c *gin.Context) {
		i := inst(c)
		b, ok := body(c)
		if !ok {
			return
		}
		var parts []string
		for _, n := range strs(b, "participants") {
			parts = append(parts, strings.Split(n, "@")[0])
		}
		r := s.call(http.MethodPost, "/group/create", i.Token, M{"groupName": str(b, "subject"), "participants": parts})
		if !r.ok() {
			fail(c, r)
			return
		}
		var created M
		_ = r.decode(&created)
		jid := evoformat.Str(created, "jid")
		if d := str(b, "description"); d != "" && jid != "" {
			s.call(http.MethodPost, "/group/description", i.Token, M{"groupJid": jid, "description": d})
		}
		if g, gr := s.groupInfo(i, jid); gr.ok() && g != nil {
			c.JSON(http.StatusCreated, groupView(g, true))
			return
		}
		c.JSON(http.StatusCreated, gin.H{"id": jid, "subject": str(b, "subject")})
	})
	p(http.MethodPost, "updateGroupSubject", update("/group/name", func(b M, jid string) M { return M{"groupJid": jid, "name": str(b, "subject")} }))
	p(http.MethodPost, "updateGroupDescription", update("/group/description", func(b M, jid string) M { return M{"groupJid": jid, "description": str(b, "description")} }))
	p(http.MethodPost, "updateGroupPicture", update("/group/photo", func(b M, jid string) M { return M{"groupJid": jid, "image": str(b, "image")} }))
	p(http.MethodPost, "updateSetting", update("/group/settings", func(b M, jid string) M { return M{"groupJid": jid, "action": str(b, "action")} }))

	invite := func(reset bool) gin.HandlerFunc {
		return needJID(func(c *gin.Context, i *instance_model.Instance, jid string) {
			r := s.call(http.MethodPost, "/group/invitelink", i.Token, M{"groupJid": jid, "reset": reset})
			if !r.ok() {
				fail(c, r)
				return
			}
			var link string
			_ = r.decode(&link)
			code := link[strings.LastIndex(link, "/")+1:]
			if reset {
				c.JSON(http.StatusCreated, gin.H{"revoked": true, "inviteCode": code})
				return
			}
			c.JSON(http.StatusOK, gin.H{"inviteUrl": link, "inviteCode": code})
		})
	}
	p(http.MethodGet, "inviteCode", invite(false))
	p(http.MethodPost, "revokeInviteCode", invite(true))
	p(http.MethodGet, "acceptInviteCode", func(c *gin.Context) {
		i := inst(c)
		code := c.Query("inviteCode")
		code = code[strings.LastIndex(code, "/")+1:]
		r := s.call(http.MethodPost, "/group/join", i.Token, M{"code": code})
		if !r.ok() {
			fail(c, r)
			return
		}
		c.JSON(http.StatusOK, gin.H{"accepted": true})
	})

	p(http.MethodGet, "fetchAllGroups", func(c *gin.Context) {
		i := inst(c)
		r := s.call(http.MethodGet, "/group/list", i.Token, nil)
		if !r.ok() {
			fail(c, r)
			return
		}
		var list []M
		_ = r.decode(&list)
		with := c.Query("getParticipants") == "true"
		out := make([]gin.H, 0, len(list))
		for _, g := range list {
			out = append(out, groupView(g, with))
		}
		c.JSON(http.StatusOK, out)
	})
	p(http.MethodGet, "findGroupInfos", needJID(func(c *gin.Context, i *instance_model.Instance, jid string) {
		g, r := s.groupInfo(i, jid)
		if !r.ok() {
			fail(c, r)
			return
		}
		c.JSON(http.StatusOK, groupView(g, true))
	}))
	p(http.MethodGet, "participants", needJID(func(c *gin.Context, i *instance_model.Instance, jid string) {
		g, r := s.groupInfo(i, jid)
		if !r.ok() {
			fail(c, r)
			return
		}
		c.JSON(http.StatusOK, gin.H{"participants": groupView(g, true)["participants"]})
	}))
	p(http.MethodPost, "updateParticipant", needJID(func(c *gin.Context, i *instance_model.Instance, jid string) {
		b, ok := body(c)
		if !ok {
			return
		}
		var parts []string
		for _, n := range strs(b, "participants") {
			parts = append(parts, strings.Split(n, "@")[0])
		}
		action := str(b, "action")
		r := s.call(http.MethodPost, "/group/participant", i.Token, M{"groupJid": jid, "participants": parts, "action": action})
		if !r.ok() {
			fail(c, r)
			return
		}
		res := make([]gin.H, 0, len(parts))
		for _, n := range parts {
			res = append(res, gin.H{"status": "200", "jid": toJID(n)})
		}
		c.JSON(http.StatusCreated, gin.H{"updateParticipants": res})
	}))
	p(http.MethodDelete, "leaveGroup", needJID(func(c *gin.Context, i *instance_model.Instance, jid string) {
		if r := s.call(http.MethodPost, "/group/leave", i.Token, M{"groupJid": jid}); !r.ok() {
			fail(c, r)
			return
		}
		c.JSON(http.StatusOK, gin.H{"groupJid": jid, "leave": true})
	}))
}

func (s *Server) registerLabels() {
	a := s.auth
	s.handle(http.MethodGet, "/label/findLabels/:instanceName", a, func(c *gin.Context) {
		i := inst(c)
		r := s.call(http.MethodGet, "/label/list", i.Token, nil)
		if !r.ok() {
			fail(c, r)
			return
		}
		var list []M
		_ = r.decode(&list)
		out := make([]gin.H, 0, len(list))
		for _, l := range list {
			out = append(out, gin.H{"id": evoformat.Str(l, "label_id"), "name": evoformat.Str(l, "label_name"),
				"color": evoformat.Str(l, "label_color"), "predefinedId": nilIfEmpty(evoformat.Str(l, "predefined_id"))})
		}
		c.JSON(http.StatusOK, out)
	})
	s.handle(http.MethodPost, "/label/handleLabel/:instanceName", a, func(c *gin.Context) {
		i := inst(c)
		b, ok := body(c)
		if !ok {
			return
		}
		action, path := str(b, "action"), "/label/chat"
		if action == "remove" {
			path = "/unlabel/chat"
		} else if action != "add" {
			v2Error(c, http.StatusBadRequest, "action must be add or remove")
			return
		}
		number, label := toJID(str(b, "number")), str(b, "labelId")
		if r := s.call(http.MethodPost, path, i.Token, M{"jid": number, "labelId": label}); !r.ok() {
			fail(c, r)
			return
		}
		c.JSON(http.StatusCreated, gin.H{"numberJid": number, "labelId": label, action: true})
	})
}

// register declara todas las rutas v2.
func (s *Server) register() {
	s.v2.NoRoute(func(c *gin.Context) { s.main.ServeHTTP(c.Writer, c.Request) })

	s.handle(http.MethodGet, "/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": 200, "message": "Welcome to the Evolution API, it is working!", "version": "2.3.0",
			"clientName": s.opt.ClientName, "manager": "/manager", "documentation": "/swagger/index.html", "compatibility": "evolution-go"})
	})
	s.handle(http.MethodPost, "/instance/create", s.createInstance)
	s.handle(http.MethodGet, "/instance/fetchInstances", s.fetchInstances)
	s.handle(http.MethodGet, "/instance/connect/:instanceName", s.auth, s.connect)
	s.handle(http.MethodGet, "/instance/connectionState/:instanceName", s.auth, s.connectionState)
	s.handle(http.MethodPost, "/instance/restart/:instanceName", s.auth, s.restart)
	s.handle(http.MethodPut, "/instance/restart/:instanceName", s.auth, s.restart)
	s.handle(http.MethodDelete, "/instance/logout/:instanceName", s.auth, s.logout)
	s.handle(http.MethodDelete, "/instance/delete/:instanceName", s.deleteInstance)
	s.handle(http.MethodPost, "/instance/setPresence/:instanceName", s.auth, s.setPresence)

	s.handle(http.MethodPost, "/settings/set/:instanceName", s.auth, s.setSettings)
	s.handle(http.MethodGet, "/settings/find/:instanceName", s.auth, s.findSettings)
	s.handle(http.MethodPost, "/proxy/set/:instanceName", s.auth, s.setProxy)
	s.handle(http.MethodGet, "/proxy/find/:instanceName", s.auth, s.findProxy)
	s.handle(http.MethodPost, "/webhook/set/:instanceName", s.auth, s.setWebhook)
	s.handle(http.MethodGet, "/webhook/find/:instanceName", s.auth, s.findWebhook)

	s.registerMessages()
	s.registerChat()
	s.registerGroups()
	s.registerLabels()
}
