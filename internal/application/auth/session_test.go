package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	domainidentity "github.com/wecratfs/commerce/internal/domain/identity"
	"github.com/wecratfs/commerce/internal/ports"
)

type sessionStore struct {
	records map[string]ports.SessionRecord
}

func (s *sessionStore) Create(_ context.Context, record ports.SessionRecord) error {
	if s.records == nil {
		s.records = map[string]ports.SessionRecord{}
	}
	s.records[record.ID] = record
	return nil
}
func (s *sessionStore) Get(_ context.Context, id string) (ports.SessionRecord, error) {
	record, ok := s.records[id]
	if !ok {
		return ports.SessionRecord{}, errors.New("missing")
	}
	return record, nil
}
func (s *sessionStore) Delete(_ context.Context, id string) error {
	delete(s.records, id)
	return nil
}
func (s *sessionStore) DeleteAll(_ context.Context, userID int64) error {
	for id, record := range s.records {
		if record.UserID == userID {
			delete(s.records, id)
		}
	}
	return nil
}
func (s *sessionStore) List(_ context.Context, userID int64) ([]ports.SessionRecord, error) {
	result := []ports.SessionRecord{}
	for _, record := range s.records {
		if record.UserID == userID {
			result = append(result, record)
		}
	}
	return result, nil
}

func TestSessionServiceCreatesAndExpiresOpaqueRecords(t *testing.T) {
	store := &sessionStore{}
	service := NewSessionService(store)
	now := time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	service.newID = func() (string, error) { return "opaque-token", nil }

	record, err := service.Start(context.Background(), domainidentity.User{
		ID:    42,
		Roles: []domainidentity.Role{domainidentity.RoleCustomer},
	}, "browser", "127.0.0.1", time.Hour)
	if err != nil {
		t.Fatalf("start session: %v", err)
	}
	if record.ID == "42" || record.CSRFToken == "" || record.ExpiresAt != now.Add(time.Hour) {
		t.Fatalf("session is not opaque or correctly bounded: %+v", record)
	}
	if _, err := service.Get(context.Background(), record.ID); err != nil {
		t.Fatalf("get session: %v", err)
	}
	now = now.Add(2 * time.Hour)
	if _, err := service.Get(context.Background(), record.ID); !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("expected expired session, got %v", err)
	}
}
