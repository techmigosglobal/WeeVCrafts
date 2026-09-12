package ports

import (
	"context"
	"errors"

	domainprivacy "github.com/wecratfs/commerce/internal/domain/privacy"
)

var (
	ErrPrivacyRequestNotFound = errors.New("privacy request not found")
	ErrPrivacyRequestState    = errors.New("privacy request is not executable")
)

type PrivacyRepository interface {
	GetCenter(ctx context.Context, userID int64) (domainprivacy.Center, error)
	RecordConsent(ctx context.Context, userID int64, purpose, policyVersion string, granted bool, source string) error
	SetEmailMarketing(ctx context.Context, userID int64, enabled bool) error
	CreatePrivacyRequest(ctx context.Context, userID int64, requestType, details string) (domainprivacy.Request, error)
	ExecuteDeletion(ctx context.Context, userID, requestID int64) error
}
