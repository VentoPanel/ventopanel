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

// TestNotifyAllContinuesOnPartialFailure verifies that all notifiers are attempted
// even when one of them returns an error, and the combined error is returned.
func TestNotifyAllContinuesOnPartialFailure(t *testing.T) {
	telegram := &notifierStub{err: errors.New("telegram unavailable")}
	whatsApp := &notifierStub{}
	service := NewService(telegram, whatsApp)

	err := service.NotifyAll(context.Background(), "server down")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// WhatsApp must still have been called despite Telegram failing.
	if len(whatsApp.messages) != 1 {
		t.Fatalf("whatsapp should be called even when telegram fails; messages: %#v", whatsApp.messages)
	}
}

func TestNotifyAllNoNotifiers(t *testing.T) {
	service := NewService()
	if err := service.NotifyAll(context.Background(), "test"); err != nil {
		t.Fatalf("expected nil error with no notifiers, got: %v", err)
	}
}
