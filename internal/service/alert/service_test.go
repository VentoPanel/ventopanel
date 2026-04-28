package alert

import (
	"context"
	"errors"
	"testing"
)

type notifierStub struct {
	err      error
	messages []string
}

func (n *notifierStub) Notify(_ context.Context, message string) error {
	n.messages = append(n.messages, message)
	return n.err
}

func TestNotifyAll(t *testing.T) {
	telegram := &notifierStub{}
	whatsApp := &notifierStub{}
	service := NewService(telegram, whatsApp)

	if err := service.NotifyAll(context.Background(), "disk is almost full"); err != nil {
		t.Fatalf("NotifyAll returned error: %v", err)
	}

	if len(telegram.messages) != 1 || telegram.messages[0] != "disk is almost full" {
		t.Fatalf("unexpected telegram messages: %#v", telegram.messages)
	}

	if len(whatsApp.messages) != 1 || whatsApp.messages[0] != "disk is almost full" {
		t.Fatalf("unexpected whatsapp messages: %#v", whatsApp.messages)
	}
}

func TestNotifyAllStopsOnTelegramFailure(t *testing.T) {
	telegram := &notifierStub{err: errors.New("telegram unavailable")}
	whatsApp := &notifierStub{}
	service := NewService(telegram, whatsApp)

	if err := service.NotifyAll(context.Background(), "server down"); err == nil {
		t.Fatal("expected error, got nil")
	}

	if len(whatsApp.messages) != 0 {
		t.Fatalf("whatsapp should not be called when telegram fails: %#v", whatsApp.messages)
	}
}
