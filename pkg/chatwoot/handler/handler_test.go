package chatwoot_handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/evolution-foundation/evolution-go/pkg/chatwoot"
	"github.com/gin-gonic/gin"
)

type fakeMW struct{}

func (fakeMW) Auth(c *gin.Context)                { c.Next() }
func (fakeMW) AuthAdmin(c *gin.Context)           { c.Next() }
func (fakeMW) AuthInstanceOrAdmin(c *gin.Context) { c.Set("isAdmin", true); c.Next() }

func TestRutas(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/chat/pin", func(c *gin.Context) {}) // rutas existentes de upstream
	s := chatwoot.New(nil, nil, nil)
	Register(r, fakeMW{}, s)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/chatwoot/webhook/abc?secret=x", strings.NewReader(`{`)))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("webhook no enrutado: %d", w.Code)
	}
}
