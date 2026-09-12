package commerce

import (
	"context"
	"errors"
	"testing"
	"time"

	domaincommerce "github.com/wecratfs/commerce/internal/domain/commerce"
	"github.com/wecratfs/commerce/internal/ports"
)

type mediaLifecycleRepository struct {
	record       domaincommerce.MediaRecord
	finalized    bool
	deleted      bool
	finalizePath string
}

func (r *mediaLifecycleRepository) PrepareUpload(context.Context, int64, int64, string, string, int64) (int64, error) {
	return r.record.ID, nil
}

func (r *mediaLifecycleRepository) GetOwnedMedia(context.Context, int64, int64) (domaincommerce.MediaRecord, error) {
	return r.record, nil
}

func (r *mediaLifecycleRepository) FinalizeMedia(_ context.Context, _ int64, _ int64, publicPath, contentType string, byteSize int64) (domaincommerce.MediaRecord, error) {
	r.record.Status = "ready"
	r.record.PublicPath = publicPath
	r.record.ContentType = contentType
	r.record.ByteSize = byteSize
	r.finalized = true
	r.finalizePath = publicPath
	return r.record, nil
}

func (r *mediaLifecycleRepository) GetPublicMedia(context.Context, int64) (domaincommerce.MediaRecord, error) {
	return r.record, nil
}

func (r *mediaLifecycleRepository) ClaimOwnedMedia(context.Context, int64, int64) (domaincommerce.MediaRecord, error) {
	r.record.Status = "deleting"
	return r.record, nil
}

func (r *mediaLifecycleRepository) MarkMediaDeleted(context.Context, int64) error {
	r.deleted = true
	r.record.Status = "deleted"
	return nil
}

func (r *mediaLifecycleRepository) RestoreMedia(context.Context, int64) error {
	r.record.Status = "pending"
	return nil
}

func (r *mediaLifecycleRepository) ClaimExpiredMedia(context.Context, time.Time, int) ([]domaincommerce.MediaRecord, error) {
	r.record.Status = "deleting"
	return []domaincommerce.MediaRecord{r.record}, nil
}

func (r *mediaLifecycleRepository) ListExpiredMedia(context.Context, time.Time, int) ([]domaincommerce.MediaRecord, error) {
	return []domaincommerce.MediaRecord{r.record}, nil
}

type mediaStorage struct {
	info       ports.ObjectInfo
	deleted    bool
	inspected  int
	downloaded int
}

func (s *mediaStorage) CreateUploadURL(context.Context, ports.UploadRequest) (ports.UploadURL, error) {
	return ports.UploadURL{URL: "https://storage.invalid/upload"}, nil
}

func (s *mediaStorage) InspectObject(context.Context, string) (ports.ObjectInfo, error) {
	s.inspected++
	return s.info, nil
}

func (s *mediaStorage) CreateDownloadURL(context.Context, string, time.Duration) (ports.DownloadURL, error) {
	s.downloaded++
	return ports.DownloadURL{URL: "https://storage.invalid/read"}, nil
}

func (s *mediaStorage) DeleteObject(context.Context, string) error {
	s.deleted = true
	return nil
}

func TestMediaFinalizeVerifiesStoredObjectBeforePublishing(t *testing.T) {
	repository := &mediaLifecycleRepository{record: domaincommerce.MediaRecord{ID: 7, ObjectKey: "products/9/object", ContentType: "image/png", ByteSize: 12, Status: "pending"}}
	storage := &mediaStorage{info: ports.ObjectInfo{ContentType: "image/png", ByteSize: 12}}
	service := NewMediaService(storage, repository)

	record, err := service.FinalizeUpload(context.Background(), 9, 7)
	if err != nil {
		t.Fatalf("finalize media: %v", err)
	}
	if !repository.finalized || record.Status != "ready" || record.PublicPath != "/media/products/7" {
		t.Fatalf("unexpected finalized media: %+v", record)
	}
	if storage.inspected != 1 {
		t.Fatalf("expected one object inspection, got %d", storage.inspected)
	}

	if _, err := service.FinalizeUpload(context.Background(), 9, 7); err != nil {
		t.Fatalf("ready media should be idempotent: %v", err)
	}
	if storage.inspected != 1 {
		t.Fatalf("ready replay re-inspected the object: %d", storage.inspected)
	}
}

func TestMediaFinalizeRejectsObjectMetadataMismatch(t *testing.T) {
	repository := &mediaLifecycleRepository{record: domaincommerce.MediaRecord{ID: 8, ObjectKey: "products/9/object", ContentType: "image/jpeg", ByteSize: 20, Status: "pending"}}
	storage := &mediaStorage{info: ports.ObjectInfo{ContentType: "image/png", ByteSize: 20}}
	service := NewMediaService(storage, repository)

	if _, err := service.FinalizeUpload(context.Background(), 9, 8); !errors.Is(err, ErrInvalidMedia) {
		t.Fatalf("expected metadata mismatch rejection, got %v", err)
	}
	if repository.finalized {
		t.Fatal("metadata mismatch finalized media")
	}
}

func TestMediaCleanupDeletesOnlyObjectsThatWereRemoved(t *testing.T) {
	repository := &mediaLifecycleRepository{record: domaincommerce.MediaRecord{ID: 9, ObjectKey: "products/9/object", Status: "pending"}}
	storage := &mediaStorage{}
	service := NewMediaService(storage, repository)

	deleted, err := service.CleanupExpired(context.Background(), 1)
	if err != nil || deleted != 1 || !storage.deleted || !repository.deleted {
		t.Fatalf("unexpected cleanup result: deleted=%d err=%v storage_deleted=%v repository_deleted=%v", deleted, err, storage.deleted, repository.deleted)
	}
}

func TestMediaDeleteClaimsReadyMediaBeforeRemovingObject(t *testing.T) {
	repository := &mediaLifecycleRepository{record: domaincommerce.MediaRecord{ID: 10, ObjectKey: "products/9/object", Status: "ready", PublicPath: "/media/products/10"}}
	storage := &mediaStorage{}
	service := NewMediaService(storage, repository)

	if err := service.DeleteUpload(context.Background(), 9, 10); err != nil {
		t.Fatalf("delete ready media: %v", err)
	}
	if !storage.deleted || !repository.deleted || repository.record.Status != "deleted" {
		t.Fatalf("ready media was not deleted safely: storage=%v repository=%v status=%s", storage.deleted, repository.deleted, repository.record.Status)
	}
}
