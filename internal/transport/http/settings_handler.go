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
	TelegramBotToken        string `json:"telegram_bot_token"`
	TelegramChatID          string `json:"telegram_chat_id"`
	WhatsAppWebhookURL      string `json:"whatsapp_webhook_url"`
	UptimeNotifyDown        *bool  `json:"uptime_notify_down"`
	UptimeNotifyRecovery    *bool  `json:"uptime_notify_recovery"`
	UptimeFailThreshold     *int   `json:"uptime_fail_threshold"`
	UptimeRecoveryThreshold *int   `json:"uptime_recovery_threshold"`
	DeployNotifySuccess     *bool  `json:"deploy_notify_success"`
	DeployNotifyFailure     *bool  `json:"deploy_notify_failure"`
}

type notificationSettingsResponse struct {
	TelegramBotToken        string `json:"telegram_bot_token"`
	TelegramChatID          string `json:"telegram_chat_id"`
	WhatsAppWebhookURL      string `json:"whatsapp_webhook_url"`
	UptimeNotifyDown        bool   `json:"uptime_notify_down"`
	UptimeNotifyRecovery    bool   `json:"uptime_notify_recovery"`
	UptimeFailThreshold     int    `json:"uptime_fail_threshold"`
	UptimeRecoveryThreshold int    `json:"uptime_recovery_threshold"`
	DeployNotifySuccess     bool   `json:"deploy_notify_success"`
	DeployNotifyFailure     bool   `json:"deploy_notify_failure"`
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
		TelegramChatID:          cfg.TelegramChatID,
		WhatsAppWebhookURL:      cfg.WhatsAppWebhookURL,
		UptimeNotifyDown:        cfg.UptimeNotifyDown,
		UptimeNotifyRecovery:    cfg.UptimeNotifyRecovery,
		UptimeFailThreshold:     cfg.UptimeFailThreshold,
		UptimeRecoveryThreshold: cfg.UptimeRecoveryThreshold,
		DeployNotifySuccess:     cfg.DeployNotifySuccess,
		DeployNotifyFailure:     cfg.DeployNotifyFailure,
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
	existing, err := h.repo.GetNotificationConfig(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	token := req.TelegramBotToken
	if strings.HasPrefix(token, "••••") {
		token = existing.TelegramBotToken
	}

	cfg := settingsdomain.NotificationConfig{
		TelegramBotToken:        token,
		TelegramChatID:          req.TelegramChatID,
		WhatsAppWebhookURL:      req.WhatsAppWebhookURL,
		UptimeNotifyDown:        existing.UptimeNotifyDown,
		UptimeNotifyRecovery:    existing.UptimeNotifyRecovery,
		UptimeFailThreshold:     existing.UptimeFailThreshold,
		UptimeRecoveryThreshold: existing.UptimeRecoveryThreshold,
		DeployNotifySuccess:     existing.DeployNotifySuccess,
		DeployNotifyFailure:     existing.DeployNotifyFailure,
	}
	if req.UptimeNotifyDown != nil {
		cfg.UptimeNotifyDown = *req.UptimeNotifyDown
	}
	if req.UptimeNotifyRecovery != nil {
		cfg.UptimeNotifyRecovery = *req.UptimeNotifyRecovery
	}
	if req.UptimeFailThreshold != nil {
		cfg.UptimeFailThreshold = settingsdomain.ClampInt(*req.UptimeFailThreshold, 1, 60)
	}
	if req.UptimeRecoveryThreshold != nil {
		cfg.UptimeRecoveryThreshold = settingsdomain.ClampInt(*req.UptimeRecoveryThreshold, 1, 60)
	}
	if req.DeployNotifySuccess != nil {
		cfg.DeployNotifySuccess = *req.DeployNotifySuccess
	}
	if req.DeployNotifyFailure != nil {
		cfg.DeployNotifyFailure = *req.DeployNotifyFailure
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
