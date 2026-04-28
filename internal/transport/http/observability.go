package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	sslsvc "github.com/your-org/ventopanel/internal/service/ssl"
)

type ObservabilityHandler struct {
	sslService *sslsvc.Service
}

func NewObservabilityHandler(sslService *sslsvc.Service) *ObservabilityHandler {
	return &ObservabilityHandler{sslService: sslService}
}

func (h *ObservabilityHandler) SSL(c *gin.Context) {
	c.JSON(http.StatusOK, h.sslService.Stats())
}
