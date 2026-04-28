package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type MetricsHandler struct {
	handler http.Handler
}

func NewMetricsHandler() *MetricsHandler {
	return &MetricsHandler{handler: promhttp.Handler()}
}

func (h *MetricsHandler) Get(c *gin.Context) {
	h.handler.ServeHTTP(c.Writer, c.Request)
}
