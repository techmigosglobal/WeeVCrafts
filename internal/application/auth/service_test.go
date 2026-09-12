package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	domainidentity "github.com/wecratfs/commerce/internal/domain/identity"
	"github.com/wecratfs/commerce/internal/ports"
)

type fakeUserRepository struct {
	credentials domainidentity.Credentials
	created     domainidentity.User
	createErr   error
	getErr      error
	failed      int
	reset       int
	seenContext context.Context
}

func (f *fakeUserRepository) CreateCustomer(ctx context.Context, email, displayName, passwordHash string) (domainidentity.User, error) {
	f.seenContext = ctx
	if f.createErr != nil {
		return domainidentity.User{}, f.createErr
	}
	f.created = domainidentity.User{Email: email, DisplayName: displayName}
	return f.created, nil
}

func (f *fakeUserRepository) GetCredentialsByEmail(ctx context.Context, _ string) (domainidentity.Credentials, error) {
	f.seenContext = ctx
	if f.getErr != nil {
		return domainidentity.Credentials{}, f.getErr
	}
	return f.credentials, nil
}

func (f *fakeUserRepository) RecordFailedLogin(context.Context, int64) error {
	f.failed++
	return nil
}

func (f *fakeUserRepository) ResetFailedLogin(context.Context, int64) error {
	f.reset++
	return nil
}

type fakeHasher struct{}

func (fakeHasher) Hash(string) (string, error) { return "hash", nil }
func (fakeHasher) Compare(hash, password string) bool {
	return hash == "hash" && password == "correct-password"
}

func TestRegisterValidatesAndNormalizesInput(t *testing.T) {
	repository := &fakeUserRepository{}
	service := NewService(repository, fakeHasher{})
	ctx := context.Background()
	user, err := service.Register(ctx, RegisterInput{
		Email:       "  PERSON@Example.COM ",
		DisplayName: " A maker ",
		Password:    "correct-password",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if user.Email != "person@example.com" || user.DisplayName != "A maker" {
		t.Fatalf("unexpected normalized user: %+v", user)
	}
	if repository.seenContext != ctx {
		t.Fatal("register did not propagate context")
	}
}

func TestLoginRejectsUnknownAndRecordsBadPassword(t *testing.T) {
	repository := &fakeUserRepository{getErr: ports.ErrCredentialsNotFound}
	service := NewService(repository, fakeHasher{})
	if _, err := service.Login(context.Background(), "missing@example.com", "anything"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials, got %v", err)
	}

	repository.getErr = nil
	repository.credentials = domainidentity.Credentials{
		User:         domainidentity.User{ID: 7, Status: "active"},
		PasswordHash: "hash",
	}
	if _, err := service.Login(context.Background(), "person@example.com", "wrong-password"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected invalid password, got %v", err)
	}
	if repository.failed != 1 {
		t.Fatalf("expected one failed-login record, got %d", repository.failed)
	}
}

func TestLoginResetsFailuresAndRejectsLockedAccount(t *testing.T) {
	repository := &fakeUserRepository{credentials: domainidentity.Credentials{
		User:           domainidentity.User{ID: 8, Status: "active"},
		PasswordHash:   "hash",
		LockedUntil:    nil,
		FailedAttempts: 3,
	}}
	service := NewService(repository, fakeHasher{})
	user, err := service.Login(context.Background(), "person@example.com", "correct-password")
	if err != nil || user.ID != 8 {
		t.Fatalf("expected successful login, user=%+v err=%v", user, err)
	}
	if repository.reset != 1 {
		t.Fatalf("expected one reset, got %d", repository.reset)
	}

	lockedUntil := time.Now().Add(time.Minute)
	repository.credentials.LockedUntil = &lockedUntil
	if _, err := service.Login(context.Background(), "person@example.com", "correct-password"); !errors.Is(err, ErrAccountLocked) {
		t.Fatalf("expected locked account, got %v", err)
	}
}
