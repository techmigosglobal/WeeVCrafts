package privacy

import (
	"context"
	"errors"
	"strings"

	domainprivacy "github.com/wecratfs/commerce/internal/domain/privacy"
	"github.com/wecratfs/commerce/internal/ports"
)

var ErrInvalidRequest = errors.New("privacy request is invalid")

type Service struct {
	repository ports.PrivacyRepository
}

func NewService(repository ports.PrivacyRepository) *Service {
	return &Service{repository: repository}
}

func (s *Service) Center(ctx context.Context, userID int64) (domainprivacy.Center, error) {
	if userID <= 0 {
		return domainprivacy.Center{}, ports.ErrForbidden
	}
	return s.repository.GetCenter(ctx, userID)
}

func (s *Service) SetConsent(ctx context.Context, userID int64, purpose string, granted bool) error {
	if userID <= 0 || !validPurpose(purpose) {
		return ErrInvalidRequest
	}
	return s.repository.RecordConsent(ctx, userID, purpose, "privacy-2026-09", granted, "account-center")
}

func (s *Service) SetMarketing(ctx context.Context, userID int64, enabled bool) error {
	if userID <= 0 {
		return ports.ErrForbidden
	}
	return s.repository.SetEmailMarketing(ctx, userID, enabled)
}

func (s *Service) Request(ctx context.Context, userID int64, requestType, details string) (domainprivacy.Request, error) {
	requestType = strings.TrimSpace(requestType)
	details = strings.TrimSpace(details)
	if userID <= 0 || !validRequestType(requestType) || len(details) > 2000 {
		return domainprivacy.Request{}, ErrInvalidRequest
	}
	return s.repository.CreatePrivacyRequest(ctx, userID, requestType, details)
}

func (s *Service) ExecuteDeletion(ctx context.Context, userID, requestID int64) error {
	if userID <= 0 || requestID <= 0 {
		return ErrInvalidRequest
	}
	return s.repository.ExecuteDeletion(ctx, userID, requestID)
}

func validPurpose(value string) bool {
	return value == "marketing" || value == "analytics" || value == "necessary"
}

func validRequestType(value string) bool {
	return value == "access" || value == "correction" || value == "deletion" || value == "grievance"
}
