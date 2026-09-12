package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	domainidentity "github.com/wecratfs/commerce/internal/domain/identity"
	"github.com/wecratfs/commerce/internal/ports"
)

var ErrInvalidSession = errors.New("session is invalid or expired")

const DefaultSessionTTL = 24 * time.Hour

type SessionService struct {
	store ports.SessionStore
	now   func() time.Time
	newID func() (string, error)
}

func NewSessionService(store ports.SessionStore) *SessionService {
	return &SessionService{store: store, now: time.Now, newID: randomToken}
}

func (s *SessionService) Start(ctx context.Context, user domainidentity.User, userAgent, ipAddress string, ttl time.Duration) (ports.SessionRecord, error) {
	if ttl <= 0 {
		ttl = DefaultSessionTTL
	}
	id, err := s.newID()
	if err != nil {
		return ports.SessionRecord{}, err
	}
	csrfToken, err := s.newID()
	if err != nil {
		return ports.SessionRecord{}, err
	}
	now := s.now().UTC()
	record := ports.SessionRecord{
		ID:         id,
		UserID:     user.ID,
		RoleSlugs:  roleSlugs(user.Roles),
		CSRFToken:  csrfToken,
		UserAgent:  userAgent,
		IPAddress:  ipAddress,
		CreatedAt:  now,
		LastSeenAt: now,
		ExpiresAt:  now.Add(ttl),
	}
	if err := s.store.Create(ctx, record); err != nil {
		return ports.SessionRecord{}, err
	}
	return record, nil
}

func (s *SessionService) Get(ctx context.Context, sessionID string) (ports.SessionRecord, error) {
	if sessionID == "" {
		return ports.SessionRecord{}, ErrInvalidSession
	}
	record, err := s.store.Get(ctx, sessionID)
	if err != nil {
		return ports.SessionRecord{}, ErrInvalidSession
	}
	if record.ExpiresAt.IsZero() || !record.ExpiresAt.After(s.now()) {
		_ = s.store.Delete(ctx, sessionID)
		return ports.SessionRecord{}, ErrInvalidSession
	}
	return record, nil
}

func (s *SessionService) Revoke(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return nil
	}
	return s.store.Delete(ctx, sessionID)
}

func (s *SessionService) RevokeAll(ctx context.Context, userID int64) error {
	if userID <= 0 {
		return nil
	}
	return s.store.DeleteAll(ctx, userID)
}

func (s *SessionService) List(ctx context.Context, userID int64) ([]ports.SessionRecord, error) {
	return s.store.List(ctx, userID)
}

func roleSlugs(roles []domainidentity.Role) []string {
	result := make([]string, 0, len(roles))
	for _, role := range roles {
		result = append(result, string(role))
	}
	return result
}

func randomToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}
