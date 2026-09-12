package auth

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/wecratfs/commerce/internal/ports"
)

type fakeAuthTokenRepository struct {
	recipient    ports.EmailRecipient
	found        bool
	resetUserID  int64
	verification int64
	resetHash    string
	verifyHash   string
	revokedHash  string
}

func (f *fakeAuthTokenRepository) IssueEmailVerification(context.Context, int64, string, time.Time) (ports.EmailRecipient, error) {
	f.verifyHash = "issued"
	return f.recipient, nil
}

func (f *fakeAuthTokenRepository) IssuePasswordReset(_ context.Context, _ string, tokenHash string, _ time.Time) (ports.EmailRecipient, bool, error) {
	f.resetHash = tokenHash
	return f.recipient, f.found, nil
}

func (f *fakeAuthTokenRepository) ConsumeEmailVerification(context.Context, string) (int64, error) {
	return f.verification, nil
}

func (f *fakeAuthTokenRepository) ResetPassword(_ context.Context, tokenHash, passwordHash string) (int64, error) {
	if tokenHash != f.resetHash || passwordHash != "hash" {
		return 0, errors.New("unexpected reset arguments")
	}
	return f.resetUserID, nil
}

func (f *fakeAuthTokenRepository) RevokeAuthToken(_ context.Context, tokenHash string) error {
	f.revokedHash = tokenHash
	return nil
}

type fakeEmailSender struct {
	message ports.EmailMessage
	err     error
}

func (f *fakeEmailSender) Send(_ context.Context, message ports.EmailMessage) error {
	f.message = message
	return f.err
}

func TestRecoveryServiceSendsHashedPasswordResetTokenWithoutEnumeration(t *testing.T) {
	repository := &fakeAuthTokenRepository{found: true, recipient: ports.EmailRecipient{UserID: 7, Email: "maker@example.invalid", DisplayName: "Maker"}}
	sender := &fakeEmailSender{}
	service := NewRecoveryService(repository, fakeHasher{}, sender, "https://wecratfs.example")
	service.newToken = func() (string, error) { return "raw-token-abc", nil }

	if err := service.RequestPasswordReset(context.Background(), " MAKER@EXAMPLE.INVALID "); err != nil {
		t.Fatalf("request reset: %v", err)
	}
	if repository.resetHash != hashActionToken("raw-token-abc") || strings.Contains(repository.resetHash, "raw-token-abc") {
		t.Fatalf("repository received an unsafe token value: %q", repository.resetHash)
	}
	if sender.message.To != "maker@example.invalid" || !strings.Contains(sender.message.ActionURL, "token=raw-token-abc") {
		t.Fatalf("unexpected recovery email: %+v", sender.message)
	}

	repository.found = false
	sender.message = ports.EmailMessage{}
	if err := service.RequestPasswordReset(context.Background(), "unknown@example.invalid"); err != nil {
		t.Fatalf("unknown reset request leaked an error: %v", err)
	}
	if sender.message.To != "" {
		t.Fatalf("unknown email triggered delivery: %+v", sender.message)
	}
}

func TestRecoveryServiceRevokesWhenDeliveryFailsAndResetsPassword(t *testing.T) {
	repository := &fakeAuthTokenRepository{found: true, recipient: ports.EmailRecipient{Email: "maker@example.invalid"}, resetUserID: 9}
	sender := &fakeEmailSender{err: errors.New("smtp unavailable")}
	service := NewRecoveryService(repository, fakeHasher{}, sender, "https://wecratfs.example")
	service.newToken = func() (string, error) { return "raw-token-reset", nil }

	if err := service.RequestPasswordReset(context.Background(), "maker@example.invalid"); !errors.Is(err, ErrRecoveryUnavailable) {
		t.Fatalf("expected delivery failure, got %v", err)
	}
	if repository.revokedHash != hashActionToken("raw-token-reset") {
		t.Fatalf("failed delivery did not revoke token: %q", repository.revokedHash)
	}

	userID, err := service.ResetPassword(context.Background(), "raw-token-reset", "a sufficiently long password")
	if err != nil || userID != 9 {
		t.Fatalf("reset password: user=%d err=%v", userID, err)
	}
}

func TestRecoveryServiceValidatesPasswordAndVerificationInput(t *testing.T) {
	repository := &fakeAuthTokenRepository{verification: 12}
	service := NewRecoveryService(repository, fakeHasher{}, &fakeEmailSender{}, "")
	if _, err := service.ResetPassword(context.Background(), "token", "short"); !errors.Is(err, ErrRecoveryInvalidRequest) {
		t.Fatalf("expected short password rejection, got %v", err)
	}
	if _, err := service.VerifyEmail(context.Background(), ""); !errors.Is(err, ErrRecoveryInvalidRequest) {
		t.Fatalf("expected empty verification token rejection, got %v", err)
	}
	if userID, err := service.VerifyEmail(context.Background(), "verification-token"); err != nil || userID != 12 {
		t.Fatalf("verify email: user=%d err=%v", userID, err)
	}
}
