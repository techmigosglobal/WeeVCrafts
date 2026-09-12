package ports

import (
	"context"
	"errors"
	"time"

	domainidentity "github.com/wecratfs/commerce/internal/domain/identity"
)

var (
	ErrCredentialsNotFound = errors.New("credentials not found")
	ErrUserAlreadyExists   = errors.New("user already exists")
	ErrAuthTokenInvalid    = errors.New("authentication token is invalid or expired")
	ErrEmailUnavailable    = errors.New("email delivery is unavailable")
)

type UserRepository interface {
	CreateCustomer(ctx context.Context, email, displayName, passwordHash string) (domainidentity.User, error)
	GetCredentialsByEmail(ctx context.Context, email string) (domainidentity.Credentials, error)
	RecordFailedLogin(ctx context.Context, userID int64) error
	ResetFailedLogin(ctx context.Context, userID int64) error
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(encodedHash, password string) bool
}

// AuthTokenRepository stores only hashes of one-time account-action tokens.
// It is separate from UserRepository so existing identity fakes and callers
// remain source-compatible while recovery/verification is added.
type AuthTokenRepository interface {
	IssueEmailVerification(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) (EmailRecipient, error)
	IssuePasswordReset(ctx context.Context, email, tokenHash string, expiresAt time.Time) (EmailRecipient, bool, error)
	ConsumeEmailVerification(ctx context.Context, tokenHash string) (int64, error)
	ResetPassword(ctx context.Context, tokenHash, passwordHash string) (int64, error)
	RevokeAuthToken(ctx context.Context, tokenHash string) error
}

type EmailRecipient struct {
	UserID      int64
	Email       string
	DisplayName string
}

// EmailSender is the provider-neutral delivery seam. Implementations must
// not log ActionURL because it contains a one-time secret.
type EmailSender interface {
	Send(ctx context.Context, message EmailMessage) error
}

type EmailMessage struct {
	To          string
	DisplayName string
	Subject     string
	ActionURL   string
}

// RoleChecker is the application authorization boundary. Implementations may
// load roles from PostgreSQL, but role policy decisions stay in application
// services rather than being hidden only in templates or HTTP handlers.
type RoleChecker interface {
	HasAnyRole(ctx context.Context, userID int64, roles ...domainidentity.Role) (bool, error)
}
