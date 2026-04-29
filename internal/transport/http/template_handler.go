package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	tmpl "github.com/your-org/ventopanel/internal/domain/template"
)

type TemplateHandler struct{}

func NewTemplateHandler() *TemplateHandler {
	return &TemplateHandler{}
}

// List handles GET /templates — returns the full template catalog.
func (h *TemplateHandler) List(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"items": tmpl.Catalog})
}

// GetByID handles GET /templates/:id — returns one template including its Dockerfile.
func (h *TemplateHandler) GetByID(c *gin.Context) {
	t := tmpl.ByID(c.Param("id"))
	if t == nil {
		c.JSON(http.StatusNotFound, errorResponse{Error: "template not found"})
		return
	}
	c.JSON(http.StatusOK, t)
}
