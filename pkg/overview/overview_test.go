package overview

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestPhoneFromJid(t *testing.T) {
	cases := map[string]string{
		"573001234567:3@s.whatsapp.net": "573001234567",
		"573001234567@s.whatsapp.net":   "573001234567",
		"":                              "",
	}
	for in, want := range cases {
		if got := phoneFromJid(in); got != want {
			t.Errorf("phoneFromJid(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRutasNoChocanConLasDeInstancia(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	noop := func(c *gin.Context) {}
	// Rutas existentes bajo /instance (pkg/routes/routes.go).
	r.POST("/instance/create", noop)
	r.GET("/instance/all", noop)
	r.GET("/instance/info/:instanceId", noop)
	r.DELETE("/instance/delete/:instanceId", noop)
	r.POST("/instance/proxy/:instanceId", noop)
	r.POST("/instance/forcereconnect/:instanceId", noop)
	r.GET("/instance/logs/:instanceId", noop)
	r.GET("/instance/status", noop)
	r.GET("/instance/qr", noop)
	r.POST("/instance/reconnect", noop)
	r.GET("/instance/:instanceId/advanced-settings", noop)
	r.PUT("/instance/:instanceId/advanced-settings", noop)

	Register(r, fakeMW{}, nil, nil, nil)

	for _, p := range []string{"/instance/overview", "/instance/history/x", "/instance/status", "/instance/x/advanced-settings"} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, p, nil)
		r.ServeHTTP(w, req)
		if w.Code == http.StatusNotFound {
			t.Errorf("%s no está enrutada", p)
		}
	}
}

type fakeMW struct{}

func (fakeMW) Auth(c *gin.Context)                { c.AbortWithStatus(http.StatusUnauthorized) }
func (fakeMW) AuthAdmin(c *gin.Context)           { c.AbortWithStatus(http.StatusUnauthorized) }
func (fakeMW) AuthInstanceOrAdmin(c *gin.Context) { c.AbortWithStatus(http.StatusUnauthorized) }
