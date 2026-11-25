// Package linegroups permite agrupar líneas (instancias) con nombre y color
// propios, para filtrarlas y verlas juntas en el manager.
package linegroups

import (
	"errors"
	"net/http"
	"strings"
	"time"

	auth_middleware "github.com/evolution-foundation/evolution-go/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Colors son los colores que acepta un grupo; el manager los traduce a su paleta.
var Colors = []string{"green", "blue", "orange", "violet", "clay", "slate", "teal", "rose"}

// Group es un grupo de líneas.
type Group struct {
	ID        string    `json:"id" gorm:"type:uuid;primaryKey"`
	Name      string    `json:"name" gorm:"size:80;not null;uniqueIndex"`
	Color     string    `json:"color" gorm:"size:16;not null"`
	CreatedAt time.Time `json:"createdAt" gorm:"autoCreateTime"`
}

func (Group) TableName() string { return "line_groups" }

// Member asigna una línea a un grupo (como mucho uno por línea).
type Member struct {
	InstanceID string `gorm:"primaryKey"`
	GroupID    string `gorm:"index;not null"`
}

func (Member) TableName() string { return "line_group_members" }

var (
	ErrNotFound  = errors.New("group not found")
	ErrNameTaken = errors.New("a group with that name already exists")
	ErrInvalid   = errors.New("invalid group")
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Migrate() error { return r.db.AutoMigrate(&Group{}, &Member{}) }

func validColor(c string) bool {
	for _, x := range Colors {
		if x == c {
			return true
		}
	}
	return false
}

func clean(name, color string) (string, string, error) {
	name = strings.TrimSpace(name)
	if name == "" || len([]rune(name)) > 80 {
		return "", "", ErrInvalid
	}
	if color == "" {
		color = Colors[0]
	}
	if !validColor(color) {
		return "", "", ErrInvalid
	}
	return name, color, nil
}

func (r *Repository) nameTaken(name, exceptID string) (bool, error) {
	var n int64
	q := r.db.Model(&Group{}).Where("LOWER(name) = LOWER(?)", name)
	if exceptID != "" {
		q = q.Where("id <> ?", exceptID)
	}
	err := q.Count(&n).Error
	return n > 0, err
}

func (r *Repository) List() ([]Group, error) {
	var out []Group
	err := r.db.Order("name ASC").Find(&out).Error
	return out, err
}

func (r *Repository) Create(name, color string) (*Group, error) {
	name, color, err := clean(name, color)
	if err != nil {
		return nil, err
	}
	if taken, err := r.nameTaken(name, ""); err != nil {
		return nil, err
	} else if taken {
		return nil, ErrNameTaken
	}
	g := &Group{ID: uuid.NewString(), Name: name, Color: color}
	return g, r.db.Create(g).Error
}

func (r *Repository) Update(id, name, color string) (*Group, error) {
	name, color, err := clean(name, color)
	if err != nil {
		return nil, err
	}
	var g Group
	if err := r.db.First(&g, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if taken, err := r.nameTaken(name, id); err != nil {
		return nil, err
	} else if taken {
		return nil, ErrNameTaken
	}
	g.Name, g.Color = name, color
	return &g, r.db.Save(&g).Error
}

// Delete borra el grupo; sus líneas quedan sin grupo.
func (r *Repository) Delete(id string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("group_id = ?", id).Delete(&Member{}).Error; err != nil {
			return err
		}
		res := tx.Where("id = ?", id).Delete(&Group{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrNotFound
		}
		return nil
	})
}

// Assign pone las líneas en un grupo; groupID vacío las deja sin grupo.
func (r *Repository) Assign(instanceIDs []string, groupID string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if groupID == "" {
			return tx.Where("instance_id IN ?", instanceIDs).Delete(&Member{}).Error
		}
		var n int64
		if err := tx.Model(&Group{}).Where("id = ?", groupID).Count(&n).Error; err != nil {
			return err
		}
		if n == 0 {
			return ErrNotFound
		}
		for _, id := range instanceIDs {
			m := Member{InstanceID: id, GroupID: groupID}
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "instance_id"}},
				DoUpdates: clause.AssignmentColumns([]string{"group_id"}),
			}).Create(&m).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// Members devuelve instancia → grupo.
func (r *Repository) Members() (map[string]string, error) {
	var rows []Member
	if err := r.db.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string]string, len(rows))
	for _, m := range rows {
		out[m.InstanceID] = m.GroupID
	}
	return out, nil
}

// Register monta /line-groups (clave global).
func Register(r *gin.Engine, mw auth_middleware.Middleware, repo *Repository) {
	g := r.Group("/line-groups", mw.AuthAdmin)
	g.GET("", func(c *gin.Context) {
		list, err := repo.List()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "success", "data": list, "colors": Colors})
	})

	type body struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	}
	g.POST("", func(c *gin.Context) {
		var b body
		if err := c.ShouldBindJSON(&b); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		grp, err := repo.Create(b.Name, b.Color)
		if err != nil {
			c.JSON(statusOf(err), gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "success", "data": grp})
	})
	g.PUT("/:id", func(c *gin.Context) {
		var b body
		if err := c.ShouldBindJSON(&b); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		grp, err := repo.Update(c.Param("id"), b.Name, b.Color)
		if err != nil {
			c.JSON(statusOf(err), gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "success", "data": grp})
	})
	g.DELETE("/:id", func(c *gin.Context) {
		if err := repo.Delete(c.Param("id")); err != nil {
			c.JSON(statusOf(err), gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})
	g.POST("/assign", func(c *gin.Context) {
		var b struct {
			InstanceIDs []string `json:"instanceIds"`
			GroupID     string   `json:"groupId"`
		}
		if err := c.ShouldBindJSON(&b); err != nil || len(b.InstanceIDs) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "instanceIds is required"})
			return
		}
		if err := repo.Assign(b.InstanceIDs, b.GroupID); err != nil {
			c.JSON(statusOf(err), gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})
}

func statusOf(err error) int {
	switch {
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrNameTaken):
		return http.StatusConflict
	case errors.Is(err, ErrInvalid):
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}
