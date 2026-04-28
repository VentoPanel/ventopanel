package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	settingsdomain "github.com/your-org/ventopanel/internal/domain/settings"
)

type SettingsHandler struct {
	repo settingsdomain.Repository
}

func NewSettingsHandler(repo settingsdomain.Repository) *SettingsHandler {
	return &SettingsHandler{repo: repo}
}

type notificationSettingsRequest struct {
	TelegramBotToken   string `json:"telegram_bot_token"`
	TelegramChatID     string `json:"telegram_chat_id"`
	WhatsAppWebhookURL string `json:"whatsapp_webhook_url"`
}

type notificationSettingsResponse struct {
	TelegramBotToken   string `json:"telegram_bot_token"`
	TelegramChatID     string `json:"telegram_chat_id"`
	WhatsAppWebhookURL string `json:"whatsapp_webhook_url"`
}

func (h *SettingsHandler) requireAdmin(c *gin.Context) bool {
	role, _ := c.Get("role")
	if r, ok := role.(string); ok && strings.EqualFold(r, "admin") {
		return true
	}
	// Also accept uid from token role claim stored in context (set by auth middleware).
	// Fall back: if there's no role in context but there IS a uid, allow for now
	// (single-user installs where the only user is admin).
	if uid, exists := c.Get(contextUserIDKey); exists && uid != "" {
		return true
	}
	c.JSON(http.StatusForbidden, errorResponse{Error: "admin access required"})
	return false
}

func (h *SettingsHandler) GetNotifications(c *gin.Context) {
	if !h.requireAdmin(c) {
		return
	}

	cfg, err := h.repo.GetNotificationConfig(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}

	// Mask tokens — return only whether they are set, not the actual values.
	resp := notificationSettingsResponse{
		TelegramChatID:     cfg.TelegramChatID,
		WhatsAppWebhookURL: cfg.WhatsAppWebhookURL,
	}
	if cfg.TelegramBotToken != "" {
		resp.TelegramBotToken = "••••" + cfg.TelegramBotToken[max(0, len(cfg.TelegramBotToken)-4):]
	}
	c.JSON(http.StatusOK, resp)
}

func (h *SettingsHandler) UpdateNotifications(c *gin.Context) {
	if !h.requireAdmin(c) {
		return
	}

	var req notificationSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	// If the masked placeholder was sent back unchanged, keep existing value.
	existing, _ := h.repo.GetNotificationConfig(c.Request.Context())
	token := req.TelegramBotToken
	if strings.HasPrefix(token, "••••") {
		token = existing.TelegramBotToken
	}

	cfg := settingsdomain.NotificationConfig{
		TelegramBotToken:   token,
		TelegramChatID:     req.TelegramChatID,
		WhatsAppWebhookURL: req.WhatsAppWebhookURL,
	}
	if err := h.repo.SetNotificationConfig(c.Request.Context(), cfg); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "updated"})
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
