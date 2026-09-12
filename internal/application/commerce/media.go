package commerce

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	domaincommerce "github.com/wecratfs/commerce/internal/domain/commerce"
	"github.com/wecratfs/commerce/internal/ports"
)

var ErrInvalidMedia = errors.New("media upload is invalid")
var ErrMediaNotReady = errors.New("media upload is not ready")

type MediaService struct {
	storage    ports.ObjectStorage
	repository ports.MediaRepository
}

func NewMediaService(storage ports.ObjectStorage, repository ports.MediaRepository) *MediaService {
	return &MediaService{storage: storage, repository: repository}
}

func (s *MediaService) PrepareUpload(ctx context.Context, sellerOwnerID, productID int64, contentType string, byteSize int64) (domaincommerce.MediaUpload, error) {
	if sellerOwnerID <= 0 || productID <= 0 || byteSize <= 0 || byteSize > 25<<20 || !allowedMediaType(contentType) || s.storage == nil || s.repository == nil {
		return domaincommerce.MediaUpload{}, ErrInvalidMedia
	}
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return domaincommerce.MediaUpload{}, err
	}
	objectKey := fmt.Sprintf("products/%d/%s", sellerOwnerID, hex.EncodeToString(bytes))
	mediaID, err := s.repository.PrepareUpload(ctx, sellerOwnerID, productID, objectKey, contentType, byteSize)
	if err != nil {
		return domaincommerce.MediaUpload{}, err
	}
	upload, err := s.storage.CreateUploadURL(ctx, ports.UploadRequest{ObjectKey: objectKey, ContentType: contentType, MaxBytes: 25 << 20})
	if err != nil {
		return domaincommerce.MediaUpload{}, err
	}
	return domaincommerce.MediaUpload{ID: mediaID, ProductID: productID, ObjectKey: objectKey, ContentType: contentType, MaxBytes: 25 << 20, UploadURL: upload.URL, ExpiresAtUnix: upload.ExpiresAtUnix}, nil
}

func (s *MediaService) FinalizeUpload(ctx context.Context, sellerOwnerID, mediaID int64) (domaincommerce.MediaRecord, error) {
	lifecycle, reader, err := s.lifecycle()
	if err != nil || sellerOwnerID <= 0 || mediaID <= 0 {
		return domaincommerce.MediaRecord{}, ErrInvalidMedia
	}
	record, err := lifecycle.GetOwnedMedia(ctx, sellerOwnerID, mediaID)
	if err != nil {
		return domaincommerce.MediaRecord{}, err
	}
	if record.Status == "ready" {
		return record, nil
	}
	if record.Status != "pending" {
		return domaincommerce.MediaRecord{}, ErrMediaNotReady
	}
	info, err := reader.InspectObject(ctx, record.ObjectKey)
	if err != nil || info.ByteSize <= 0 || info.ByteSize > 25<<20 || info.ByteSize != record.ByteSize || !allowedMediaType(info.ContentType) || info.ContentType != record.ContentType {
		return domaincommerce.MediaRecord{}, ErrInvalidMedia
	}
	return lifecycle.FinalizeMedia(ctx, sellerOwnerID, mediaID, fmt.Sprintf("/media/products/%d", mediaID), info.ContentType, info.ByteSize)
}

func (s *MediaService) ResolvePublicURL(ctx context.Context, mediaID int64) (string, error) {
	lifecycle, reader, err := s.lifecycle()
	if err != nil || mediaID <= 0 {
		return "", ErrInvalidMedia
	}
	record, err := lifecycle.GetPublicMedia(ctx, mediaID)
	if err != nil {
		return "", err
	}
	download, err := reader.CreateDownloadURL(ctx, record.ObjectKey, 15*time.Minute)
	if err != nil {
		return "", err
	}
	return download.URL, nil
}

func (s *MediaService) DeleteUpload(ctx context.Context, sellerOwnerID, mediaID int64) error {
	lifecycle, reader, err := s.lifecycle()
	if err != nil || sellerOwnerID <= 0 || mediaID <= 0 {
		return ErrInvalidMedia
	}
	record, err := lifecycle.ClaimOwnedMedia(ctx, sellerOwnerID, mediaID)
	if err != nil {
		return err
	}
	if err := reader.DeleteObject(ctx, record.ObjectKey); err != nil {
		_ = lifecycle.RestoreMedia(ctx, record.ID)
		return err
	}
	return lifecycle.MarkMediaDeleted(ctx, mediaID)
}

func (s *MediaService) CleanupExpired(ctx context.Context, limit int) (int, error) {
	lifecycle, reader, err := s.lifecycle()
	if err != nil {
		return 0, ErrInvalidMedia
	}
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	media, err := lifecycle.ClaimExpiredMedia(ctx, time.Now().Add(-time.Hour), limit)
	if err != nil {
		return 0, err
	}
	deleted := 0
	for _, record := range media {
		if err := reader.DeleteObject(ctx, record.ObjectKey); err != nil {
			_ = lifecycle.RestoreMedia(ctx, record.ID)
			continue
		}
		if err := lifecycle.MarkMediaDeleted(ctx, record.ID); err != nil {
			return deleted, err
		}
		deleted++
	}
	return deleted, nil
}

func (s *MediaService) lifecycle() (ports.MediaLifecycleRepository, ports.ObjectStorageReader, error) {
	lifecycle, ok := s.repository.(ports.MediaLifecycleRepository)
	if !ok || s.storage == nil {
		return nil, nil, ErrInvalidMedia
	}
	reader, ok := s.storage.(ports.ObjectStorageReader)
	if !ok {
		return nil, nil, ErrInvalidMedia
	}
	return lifecycle, reader, nil
}

func allowedMediaType(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return value == "image/jpeg" || value == "image/png" || value == "image/webp"
}
