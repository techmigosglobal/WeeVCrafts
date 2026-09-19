package api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	applicationauth "github.com/wecratfs/commerce/internal/application/auth"
	applicationcommerce "github.com/wecratfs/commerce/internal/application/commerce"
	"github.com/wecratfs/commerce/internal/ports"
)

type uploadSessionStore struct{ session ports.SessionRecord }

func (s *uploadSessionStore) Create(context.Context, ports.SessionRecord) error { return nil }
func (s *uploadSessionStore) Get(_ context.Context, id string) (ports.SessionRecord, error) {
	if id != s.session.ID {
		return ports.SessionRecord{}, ports.ErrNotFound
	}
	return s.session, nil
}
func (s *uploadSessionStore) Delete(context.Context, string) error   { return nil }
func (s *uploadSessionStore) DeleteAll(context.Context, int64) error { return nil }
func (s *uploadSessionStore) List(context.Context, int64) ([]ports.SessionRecord, error) {
	return []ports.SessionRecord{s.session}, nil
}

type uploadReceiver struct {
	maxBytes int64
	ownerID  int64
	key      string
	body     []byte
}

func (s *uploadReceiver) CreateUploadURL(context.Context, ports.UploadRequest) (ports.UploadURL, error) {
	return ports.UploadURL{}, nil
}
func (s *uploadReceiver) MaxUploadBytes() int64 { return s.maxBytes }
func (s *uploadReceiver) SaveObject(_ context.Context, ownerID int64, key, _ string, body []byte) error {
	s.ownerID, s.key, s.body = ownerID, key, append([]byte(nil), body...)
	return nil
}

type uploadMediaRepository struct{}

func (uploadMediaRepository) PrepareUpload(context.Context, int64, int64, string, string, int64) (int64, error) {
	return 1, nil
}

func TestMediaUploadRequiresSessionCSRFAndEnforcesPlatformLimit(t *testing.T) {
	store := &uploadSessionStore{session: ports.SessionRecord{ID: "session", UserID: 7, CSRFToken: "csrf", ExpiresAt: time.Now().Add(time.Hour)}}
	sessionService := applicationauth.NewSessionService(store)
	receiver := &uploadReceiver{maxBytes: 8}
	mediaService := applicationcommerce.NewMediaService(receiver, uploadMediaRepository{})
	handler := &Handler{sessions: sessionService, media: mediaService}

	request := httptest.NewRequest(http.MethodPut, "/api/v1/seller/products/media-upload?object_key=products%2F7%2Fitem", bytes.NewReader([]byte("image")))
	request.AddCookie(&http.Cookie{Name: "wecratfs_session", Value: "session"})
	request.Header.Set("X-CSRF-Token", "csrf")
	request.Header.Set("Content-Type", "image/png")
	response := httptest.NewRecorder()
	handler.MediaUpload(response, request)
	if response.Code != http.StatusNoContent || receiver.ownerID != 7 || receiver.key != "products/7/item" || !bytes.Equal(receiver.body, []byte("image")) {
		t.Fatalf("authenticated upload failed: status=%d owner=%d key=%q bytes=%q", response.Code, receiver.ownerID, receiver.key, receiver.body)
	}

	request = httptest.NewRequest(http.MethodPut, "/api/v1/seller/products/media-upload?object_key=products%2F7%2Fitem", bytes.NewReader([]byte("image")))
	request.AddCookie(&http.Cookie{Name: "wecratfs_session", Value: "session"})
	request.Header.Set("Content-Type", "image/png")
	response = httptest.NewRecorder()
	handler.MediaUpload(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("upload without CSRF token returned %d", response.Code)
	}

	request = httptest.NewRequest(http.MethodPut, "/api/v1/seller/products/media-upload?object_key=products%2F7%2Fitem", bytes.NewReader([]byte("too-large-image")))
	request.AddCookie(&http.Cookie{Name: "wecratfs_session", Value: "session"})
	request.Header.Set("X-CSRF-Token", "csrf")
	request.Header.Set("Content-Type", "image/png")
	response = httptest.NewRecorder()
	handler.MediaUpload(response, request)
	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized upload returned %d", response.Code)
	}
}
