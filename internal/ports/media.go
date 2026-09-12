package ports

import (
	"context"
	"time"

	domaincommerce "github.com/wecratfs/commerce/internal/domain/commerce"
)

type MediaRepository interface {
	PrepareUpload(ctx context.Context, sellerOwnerID, productID int64, objectKey, contentType string, byteSize int64) (int64, error)
}

// MediaLifecycleRepository is additive to MediaRepository so upload-only
// callers remain source-compatible while the full lifecycle is introduced.
type MediaLifecycleRepository interface {
	MediaRepository
	GetOwnedMedia(ctx context.Context, sellerOwnerID, mediaID int64) (domaincommerce.MediaRecord, error)
	FinalizeMedia(ctx context.Context, sellerOwnerID, mediaID int64, publicPath, contentType string, byteSize int64) (domaincommerce.MediaRecord, error)
	GetPublicMedia(ctx context.Context, mediaID int64) (domaincommerce.MediaRecord, error)
	ClaimOwnedMedia(ctx context.Context, sellerOwnerID, mediaID int64) (domaincommerce.MediaRecord, error)
	MarkMediaDeleted(ctx context.Context, mediaID int64) error
	RestoreMedia(ctx context.Context, mediaID int64) error
	ClaimExpiredMedia(ctx context.Context, olderThan time.Time, limit int) ([]domaincommerce.MediaRecord, error)
	ListExpiredMedia(ctx context.Context, olderThan time.Time, limit int) ([]domaincommerce.MediaRecord, error)
}

type MediaServicePort interface {
	PrepareUpload(ctx context.Context, sellerOwnerID, productID int64, contentType string, byteSize int64) (domaincommerce.MediaUpload, error)
}
