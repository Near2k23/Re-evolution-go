package overview

import (
	"context"
	"io"
	"net/http"
	"sync"
	"time"

	auth_middleware "github.com/evolution-foundation/evolution-go/pkg/middleware"
	"github.com/gin-gonic/gin"
	"go.mau.fi/whatsmeow"
)

// ClientGetter devuelve el cliente de WhatsApp de una línea, o nil.
type ClientGetter func(instanceID string) *whatsmeow.Client

const (
	pictureTTL     = 6 * time.Hour
	noPictureTTL   = 30 * time.Minute
	maxPictureSize = 2 << 20
)

type picture struct {
	data    []byte
	ctype   string
	expires time.Time
}

type pictures struct {
	client ClientGetter
	http   *http.Client
	mu     sync.Mutex
	cache  map[string]picture
}

// RegisterPictures monta GET /instance/picture/:instanceId, que devuelve la
// foto de perfil de la cuenta vinculada a la línea. Se guarda en memoria
// unas horas: las URL de WhatsApp caducan y así no se pide en cada carga.
func RegisterPictures(r *gin.Engine, mw auth_middleware.Middleware, client ClientGetter) {
	p := &pictures{client: client, http: &http.Client{Timeout: 15 * time.Second}, cache: map[string]picture{}}
	r.GET("/instance/picture/:instanceId", mw.AuthInstanceOrAdmin, p.serve)
}

func (p *pictures) serve(c *gin.Context) {
	id := c.Param("instanceId")
	if !canAccess(c, id) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	now := time.Now()
	p.mu.Lock()
	pic, ok := p.cache[id]
	p.mu.Unlock()
	if !ok || now.After(pic.expires) {
		fresh, err := p.fetch(c.Request.Context(), id)
		switch {
		case err == nil:
			pic = fresh
			p.mu.Lock()
			p.cache[id] = pic
			p.mu.Unlock()
		case !ok:
			// Sin conexión y sin copia: no hay nada que mostrar todavía.
			c.Status(http.StatusNotFound)
			return
		}
		// Si falla con copia caducada, se sirve la copia.
	}
	if len(pic.data) == 0 {
		c.Status(http.StatusNotFound)
		return
	}
	c.Header("Cache-Control", "private, max-age=3600")
	c.Data(http.StatusOK, pic.ctype, pic.data)
}

// fetch pide la foto a WhatsApp. Una cuenta sin foto (o que la oculta) se
// guarda vacía para no volver a preguntar en cada carga.
func (p *pictures) fetch(ctx context.Context, id string) (picture, error) {
	cli := p.client(id)
	if cli == nil || cli.Store.ID == nil || !cli.IsConnected() {
		return picture{}, errNotConnected
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	info, err := cli.GetProfilePictureInfo(ctx, cli.Store.ID.ToNonAD(), &whatsmeow.GetProfilePictureParams{Preview: true})
	if err != nil || info == nil || info.URL == "" {
		if err == nil || err == whatsmeow.ErrProfilePictureNotSet || err == whatsmeow.ErrProfilePictureUnauthorized {
			return picture{expires: time.Now().Add(noPictureTTL)}, nil
		}
		return picture{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, info.URL, nil)
	if err != nil {
		return picture{}, err
	}
	res, err := p.http.Do(req)
	if err != nil {
		return picture{}, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return picture{}, errBadStatus
	}
	data, err := io.ReadAll(io.LimitReader(res.Body, maxPictureSize))
	if err != nil {
		return picture{}, err
	}
	ctype := res.Header.Get("Content-Type")
	if ctype == "" {
		ctype = http.DetectContentType(data)
	}
	return picture{data: data, ctype: ctype, expires: time.Now().Add(pictureTTL)}, nil
}

type pictureError string

func (e pictureError) Error() string { return string(e) }

const (
	errNotConnected = pictureError("line not connected")
	errBadStatus    = pictureError("unexpected status from whatsapp")
)
