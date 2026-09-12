package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	domainidentity "github.com/wecratfs/commerce/internal/domain/identity"
	"github.com/wecratfs/commerce/internal/ports"
)

var (
	ErrInvalidRegistration = errors.New("registration details are invalid")
	ErrInvalidCredentials  = errors.New("email or password is incorrect")
	ErrAccountLocked       = errors.New("account is temporarily locked")
)

type Service struct {
	repository ports.UserRepository
	hasher     ports.PasswordHasher
	now        func() time.Time
}

func NewService(repository ports.UserRepository, hasher ports.PasswordHasher) *Service {
	return &Service{repository: repository, hasher: hasher, now: time.Now}
}

type RegisterInput struct {
	Email       string
	DisplayName string
	Password    string
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (domainidentity.User, error) {
	email := normalizeEmail(input.Email)
	if !validEmail(email) || strings.TrimSpace(input.DisplayName) == "" || len(input.Password) < 12 || len(input.Password) > 1024 {
		return domainidentity.User{}, ErrInvalidRegistration
	}
	passwordHash, err := s.hasher.Hash(input.Password)
	if err != nil {
		return domainidentity.User{}, err
	}
	return s.repository.CreateCustomer(ctx, email, strings.TrimSpace(input.DisplayName), passwordHash)
}

func (s *Service) Login(ctx context.Context, email, password string) (domainidentity.User, error) {
	credentials, err := s.repository.GetCredentialsByEmail(ctx, normalizeEmail(email))
	if errors.Is(err, ports.ErrCredentialsNotFound) {
		return domainidentity.User{}, ErrInvalidCredentials
	}
	if err != nil {
		return domainidentity.User{}, err
	}
	if credentials.User.Status != "active" {
		return domainidentity.User{}, ErrInvalidCredentials
	}
	if credentials.LockedUntil != nil && credentials.LockedUntil.After(s.now()) {
		return domainidentity.User{}, ErrAccountLocked
	}
	if !s.hasher.Compare(credentials.PasswordHash, password) {
		if err := s.repository.RecordFailedLogin(ctx, credentials.User.ID); err != nil {
			return domainidentity.User{}, err
		}
		return domainidentity.User{}, ErrInvalidCredentials
	}
	if err := s.repository.ResetFailedLogin(ctx, credentials.User.ID); err != nil {
		return domainidentity.User{}, err
	}
	return credentials.User, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func validEmail(email string) bool {
	if len(email) < 3 || len(email) > 320 {
		return false
	}
	parts := strings.Split(email, "@")
	return len(parts) == 2 && parts[0] != "" && parts[1] != "" && !strings.ContainsAny(email, " \t\r\n")
}
