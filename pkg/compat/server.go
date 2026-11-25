// Package compat expone rutas con la forma de Evolution API v2
// (/message/sendText/:instanceName, /instance/connect/:instanceName…) sobre la
// API de Evolution GO. Cada ruta traduce el cuerpo, llama internamente a la
// ruta Go equivalente y devuelve la respuesta con la forma de v2, así que no
// duplica lógica y las rutas Go siguen igual.
package compat

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"

	"github.com/evolution-foundation/evolution-go/pkg/chatstore"
	instance_model "github.com/evolution-foundation/evolution-go/pkg/instance/model"
	"github.com/evolution-foundation/evolution-go/pkg/webhooks"
	"github.com/gin-gonic/gin"
	"go.mau.fi/whatsmeow"
	"gorm.io/gorm"
)

// Options son las dependencias de la capa.
type Options struct {
	DB         *gorm.DB
	GlobalKey  string
	ClientName string
	Store      *chatstore.Store
	Webhooks   *webhooks.Service
	// Client devuelve el cliente de whatsmeow de una línea (o nil).
	Client func(instanceID string) *whatsmeow.Client
	// Reload recarga la configuración de una línea en ejecución.
	Reload func(instanceID string) error
}

// Server atiende las rutas v2.
type Server struct {
	opt    Options
	main   http.Handler
	v2     *gin.Engine
	routes map[string]bool // "POST /message/sendText"
	mu     sync.Mutex
}

// New crea la capa. main es el router de Evolution GO.
func New(opt Options, main http.Handler) *Server {
	gin.SetMode(gin.ReleaseMode)
	s := &Server{opt: opt, main: main, v2: gin.New(), routes: map[string]bool{}}
	s.v2.RedirectTrailingSlash = false
	s.v2.RedirectFixedPath = false
	s.v2.Use(gin.Recovery(), cors)
	s.register()
	return s
}

func cors(c *gin.Context) {
	h := c.Writer.Header()
	h.Set("Access-Control-Allow-Origin", "*")
	h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
	h.Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, apikey, ApiKey")
	c.Next()
}

// handle registra una ruta v2 y la anota para el despacho.
func (s *Server) handle(method, path string, h ...gin.HandlerFunc) {
	s.v2.Handle(method, path, h...)
	s.routes[method+" "+prefix(path)] = true
}

// prefix son los dos primeros segmentos de la ruta ("/message/sendText").
func prefix(p string) string {
	parts := strings.SplitN(strings.TrimPrefix(p, "/"), "/", 3)
	if len(parts) < 2 {
		return "/" + parts[0]
	}
	return "/" + parts[0] + "/" + parts[1]
}

// ServeHTTP envía a la capa v2 las rutas suyas y el resto al router Go.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	method := r.Method
	if method == http.MethodOptions {
		method = r.Header.Get("Access-Control-Request-Method")
	}
	if s.routes[method+" "+prefix(r.URL.Path)] {
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, apikey, ApiKey")
			w.WriteHeader(http.StatusOK)
			return
		}
		s.v2.ServeHTTP(w, r)
		return
	}
	s.main.ServeHTTP(w, r)
}

// passToGo devuelve la petición al router Go (rutas con el mismo camino en
// ambas APIs, como POST /instance/create).
func (s *Server) passToGo(c *gin.Context, body []byte) {
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	s.main.ServeHTTP(c.Writer, c.Request)
	c.Abort()
}

// ---------- llamadas internas ----------

// result es la respuesta de una ruta Go.
type result struct {
	Status int
	Data   json.RawMessage
	Error  string
	Raw    []byte
}

func (r result) ok() bool { return r.Status >= 200 && r.Status < 300 }

// decode lee data en v (o la respuesta entera si no hay data).
func (r result) decode(v interface{}) error {
	if len(r.Data) > 0 {
		return json.Unmarshal(r.Data, v)
	}
	return json.Unmarshal(r.Raw, v)
}

// call ejecuta una ruta Go con la clave dada y devuelve su respuesta.
func (s *Server) call(method, path, key string, body interface{}) result {
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, rd)
	req.Header.Set("apikey", key)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	s.main.ServeHTTP(rec, req)
	out := result{Status: rec.Code, Raw: rec.Body.Bytes()}
	var env struct {
		Data  json.RawMessage `json:"data"`
		Error string          `json:"error"`
	}
	if json.Unmarshal(out.Raw, &env) == nil {
		out.Data, out.Error = env.Data, env.Error
	}
	if !out.ok() && out.Error == "" {
		out.Error = strings.TrimSpace(string(out.Raw))
		if out.Error == "" {
			out.Error = http.StatusText(out.Status)
		}
	}
	return out
}

// ---------- autenticación ----------

const ctxInstance = "compat.instance"

// v2Error responde con la forma de error de Evolution API v2.
func v2Error(c *gin.Context, status int, msg interface{}) {
	if _, ok := msg.([]string); !ok {
		if s, ok := msg.(string); ok {
			msg = []string{s}
		}
	}
	c.AbortWithStatusJSON(status, gin.H{"status": status, "error": http.StatusText(status), "response": gin.H{"message": msg}})
}

// fail traduce el error de una ruta Go.
func fail(c *gin.Context, r result) {
	status := r.Status
	if status < 400 {
		status = http.StatusInternalServerError
	}
	if status == http.StatusInternalServerError {
		status = http.StatusBadRequest // v2 responde 400 a los errores de negocio
	}
	v2Error(c, status, r.Error)
}

func (s *Server) instanceByName(name string) *instance_model.Instance {
	var inst instance_model.Instance
	q := s.opt.DB.Where("name = ?", name)
	if s.opt.ClientName != "" {
		q = q.Where("client_name = ? OR client_name = '' OR client_name IS NULL", s.opt.ClientName)
	}
	if err := q.Order("created_at").First(&inst).Error; err != nil {
		return nil
	}
	return &inst
}

// auth carga la línea de :instanceName y valida apikey (clave global o token
// de la línea, como en v2).
func (s *Server) auth(c *gin.Context) {
	key := c.GetHeader("apikey")
	if key == "" {
		key = strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
	}
	name := c.Param("instanceName")
	inst := s.instanceByName(name)
	if key == "" || (key != s.opt.GlobalKey && (inst == nil || key != inst.Token)) {
		v2Error(c, http.StatusUnauthorized, "Unauthorized")
		return
	}
	if inst == nil {
		v2Error(c, http.StatusNotFound, `The "`+name+`" instance does not exist`)
		return
	}
	c.Set(ctxInstance, inst)
	c.Next()
}

// admin exige la clave global.
func (s *Server) admin(c *gin.Context) {
	if c.GetHeader("apikey") != s.opt.GlobalKey || s.opt.GlobalKey == "" {
		v2Error(c, http.StatusUnauthorized, "Unauthorized")
		return
	}
	c.Next()
}

func inst(c *gin.Context) *instance_model.Instance {
	return c.MustGet(ctxInstance).(*instance_model.Instance)
}

// isAdmin indica si la petición usa la clave global.
func (s *Server) isAdmin(c *gin.Context) bool {
	return s.opt.GlobalKey != "" && c.GetHeader("apikey") == s.opt.GlobalKey
}

// body lee el cuerpo JSON como mapa (vacío si no hay).
func body(c *gin.Context) (map[string]interface{}, bool) {
	m := map[string]interface{}{}
	raw, _ := io.ReadAll(c.Request.Body)
	if len(bytes.TrimSpace(raw)) == 0 {
		return m, true
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		v2Error(c, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return nil, false
	}
	return m, true
}
