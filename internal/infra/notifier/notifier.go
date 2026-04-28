package notifier

import "context"

type Telegram struct {
	token  string
	chatID string
}

func NewTelegram(token, chatID string) *Telegram {
	return &Telegram{token: token, chatID: chatID}
}

func (t *Telegram) Notify(ctx context.Context, message string) error {
	_ = ctx
	_ = message
	return nil
}

type WhatsApp struct {
	webhookURL string
}

func NewWhatsApp(webhookURL string) *WhatsApp {
	return &WhatsApp{webhookURL: webhookURL}
}

func (w *WhatsApp) Notify(ctx context.Context, message string) error {
	_ = ctx
	_ = message
	return nil
}
