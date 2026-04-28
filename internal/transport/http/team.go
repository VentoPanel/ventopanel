package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	teamsvc "github.com/your-org/ventopanel/internal/service/team"
)

type TeamHandler struct {
	service *teamsvc.Service
}

func NewTeamHandler(service *teamsvc.Service) *TeamHandler {
	return &TeamHandler{service: service}
}

func (h *TeamHandler) List(c *gin.Context) {
	_ = h.service

	c.JSON(http.StatusOK, gin.H{
		"items": []gin.H{
			{"role": "owner", "scope": "all"},
		},
	})
}
