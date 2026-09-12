package email

import (
	"context"
	"net/mail"
	"strings"
	"testing"

	"github.com/wecratfs/commerce/internal/ports"
)

func TestSMTPRejectsHeaderInjectionBeforeDialing(t *testing.T) {
	sender := NewSMTP("127.0.0.1:1", "", "", "noreply@example.invalid")
	err := sender.Send(context.Background(), ports.EmailMessage{
		To:          "customer@example.invalid",
		Subject:     "reset\r\nBcc: attacker@example.invalid",
		ActionURL:   "https://example.invalid/reset?token=opaque",
		DisplayName: "Customer",
	})
	if err == nil || !strings.Contains(err.Error(), "configured") && !strings.Contains(err.Error(), "recipient") {
		t.Fatalf("expected validation failure before network use, got %v", err)
	}
}

func TestFormatMessageDoesNotUseSMTPProviderTypes(t *testing.T) {
	from, _ := mail.ParseAddress("noreply@example.invalid")
	recipient, _ := mail.ParseAddress("customer@example.invalid")
	message := formatMessage(from, recipient, ports.EmailMessage{Subject: "Verify", ActionURL: "https://example.invalid/verify?token=opaque"})
	if !strings.Contains(string(message), "https://example.invalid/verify?token=opaque") || !strings.Contains(string(message), "Subject: Verify") {
		t.Fatalf("unexpected formatted message: %q", message)
	}
}
