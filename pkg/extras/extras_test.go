package extras

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	instance_model "github.com/evolution-foundation/evolution-go/pkg/instance/model"
	"github.com/gin-gonic/gin"
	"go.mau.fi/whatsmeow"
)

func TestInviteCode(t *testing.T) {
	for in, want := range map[string]string{"https://chat.whatsapp.com/AbC123": "AbC123", " AbC123 ": "AbC123"} {
		if got := InviteCode(in); got != want {
			t.Fatalf("%q → %q", in, got)
		}
	}
}

type mw struct{}

func (mw) Auth(c *gin.Context) {
	c.Set("instance", &instance_model.Instance{Id: "i1"})
	c.Next()
}
func (mw) AuthAdmin(c *gin.Context)           { c.Next() }
func (mw) AuthInstanceOrAdmin(c *gin.Context) { c.Next() }

func TestValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Register(r, mw{}, func(string) *whatsmeow.Client { return nil })
	for _, tc := range []struct{ path, body, want string }{
		{"/group/ephemeral", `{"groupJid":"123","expiration":5}`, "expiration must be"},
		{"/group/ephemeral", `{"expiration":0}`, "groupJid is required"},
		{"/group/ephemeral", `{"groupJid":"123","expiration":86400}`, "not connected"},
		{"/group/inviteinfo", `{}`, "code is required"},
		{"/user/business", `{}`, "number is required"},
	} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body)))
		if w.Code != 400 || !strings.Contains(w.Body.String(), tc.want) {
			t.Errorf("%s %s: %d %s", tc.path, tc.body, w.Code, w.Body.String())
		}
	}
}
