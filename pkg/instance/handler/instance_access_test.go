package instance_handler

import (
	"net/http/httptest"
	"testing"

	instance_model "github.com/evolution-foundation/evolution-go/pkg/instance/model"
	"github.com/gin-gonic/gin"
)

func newTestContext() *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	return c
}

func TestCanAccessInstance(t *testing.T) {
	t.Run("admin accede a cualquier instancia", func(t *testing.T) {
		c := newTestContext()
		c.Set("isAdmin", true)
		if !canAccessInstance(c, "otra") {
			t.Fatal("la clave global debe poder acceder")
		}
	})

	t.Run("token de la misma instancia", func(t *testing.T) {
		c := newTestContext()
		c.Set("instance", &instance_model.Instance{Id: "a"})
		if !canAccessInstance(c, "a") {
			t.Fatal("el token propio debe poder acceder")
		}
	})

	t.Run("token de otra instancia", func(t *testing.T) {
		c := newTestContext()
		c.Set("instance", &instance_model.Instance{Id: "a"})
		if canAccessInstance(c, "b") {
			t.Fatal("un token no debe acceder a otra instancia")
		}
	})

	t.Run("sin autenticación", func(t *testing.T) {
		c := newTestContext()
		if canAccessInstance(c, "a") {
			t.Fatal("sin instancia ni admin no hay acceso")
		}
	})
}
