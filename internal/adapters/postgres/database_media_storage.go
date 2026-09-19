package postgres

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wecratfs/commerce/internal/ports"
)

const netlifyMediaUploadLimit int64 = 4 << 20

// DatabaseMediaStorage stores small product images with their PostgreSQL
// metadata. The 4 MiB cap leaves room for Netlify's base64 event envelope.
type DatabaseMediaStorage struct{ pool *pgxpool.Pool }

func NewDatabaseMediaStorage(pool *pgxpool.Pool) *DatabaseMediaStorage {
	return &DatabaseMediaStorage{pool: pool}
}

func (s *DatabaseMediaStorage) MaxUploadBytes() int64 { return netlifyMediaUploadLimit }

func (s *DatabaseMediaStorage) CreateUploadURL(_ context.Context, request ports.UploadRequest) (ports.UploadURL, error) {
	if s.pool == nil || !validDatabaseMediaKey(request.ObjectKey) || !validProductMediaType(request.ContentType) || request.MaxBytes <= 0 || request.MaxBytes > netlifyMediaUploadLimit {
		return ports.UploadURL{}, errors.New("invalid upload request")
	}
	query := url.Values{}
	query.Set("object_key", request.ObjectKey)
	return ports.UploadURL{
		URL:           "/api/v1/seller/products/media-upload?" + query.Encode(),
		ExpiresAtUnix: time.Now().Add(15 * time.Minute).Unix(),
	}, nil
}

func (s *DatabaseMediaStorage) SaveObject(ctx context.Context, ownerID int64, objectKey, contentType string, body []byte) error {
	if s.pool == nil || ownerID <= 0 || !validDatabaseMediaKey(objectKey) || !validProductMediaType(contentType) || len(body) == 0 || int64(len(body)) > netlifyMediaUploadLimit {
		return errors.New("invalid media object")
	}
	var mediaID int64
	err := s.pool.QueryRow(ctx, `
		UPDATE product_media pm SET object_bytes = $1
		FROM sellers s
		WHERE pm.seller_id = s.id AND s.owner_user_id = $2 AND pm.object_key = $3
		  AND pm.content_type = $4 AND pm.byte_size = octet_length($1)
		  AND pm.status = 'pending' AND pm.object_bytes IS NULL
		RETURNING pm.id`, body, ownerID, objectKey, contentType).Scan(&mediaID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ports.ErrForbidden
	}
	return err
}

func (s *DatabaseMediaStorage) InspectObject(ctx context.Context, objectKey string) (ports.ObjectInfo, error) {
	if s.pool == nil || !validDatabaseMediaKey(objectKey) {
		return ports.ObjectInfo{}, errors.New("invalid media object key")
	}
	var info ports.ObjectInfo
	err := s.pool.QueryRow(ctx, `
		SELECT content_type, octet_length(object_bytes)
		FROM product_media
		WHERE object_key = $1 AND object_bytes IS NOT NULL AND status IN ('pending', 'ready')`, objectKey).
		Scan(&info.ContentType, &info.ByteSize)
	if errors.Is(err, pgx.ErrNoRows) {
		return ports.ObjectInfo{}, ports.ErrNotFound
	}
	return info, err
}

func (s *DatabaseMediaStorage) ReadObject(ctx context.Context, objectKey string) ([]byte, ports.ObjectInfo, error) {
	if s.pool == nil || !validDatabaseMediaKey(objectKey) {
		return nil, ports.ObjectInfo{}, errors.New("invalid media object key")
	}
	var body []byte
	var info ports.ObjectInfo
	err := s.pool.QueryRow(ctx, `
		SELECT object_bytes, content_type, octet_length(object_bytes)
		FROM product_media
		WHERE object_key = $1 AND object_bytes IS NOT NULL AND status = 'ready' AND visibility = 'public'`, objectKey).
		Scan(&body, &info.ContentType, &info.ByteSize)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ports.ObjectInfo{}, ports.ErrNotFound
	}
	return body, info, err
}

func (s *DatabaseMediaStorage) CreateDownloadURL(ctx context.Context, objectKey string, lifetime time.Duration) (ports.DownloadURL, error) {
	if s.pool == nil || !validDatabaseMediaKey(objectKey) || lifetime <= 0 || lifetime > 24*time.Hour {
		return ports.DownloadURL{}, errors.New("invalid download request")
	}
	var mediaID int64
	err := s.pool.QueryRow(ctx, `
		SELECT id FROM product_media WHERE object_key = $1 AND status = 'ready' AND visibility = 'public'`, objectKey).Scan(&mediaID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ports.DownloadURL{}, ports.ErrNotFound
	}
	if err != nil {
		return ports.DownloadURL{}, err
	}
	return ports.DownloadURL{URL: fmt.Sprintf("/media/products/%d", mediaID), ExpiresAtUnix: time.Now().Add(lifetime).Unix()}, nil
}

func (s *DatabaseMediaStorage) DeleteObject(ctx context.Context, objectKey string) error {
	if s.pool == nil || !validDatabaseMediaKey(objectKey) {
		return errors.New("invalid media object key")
	}
	_, err := s.pool.Exec(ctx, `UPDATE product_media SET object_bytes = NULL WHERE object_key = $1`, objectKey)
	return err
}

func validDatabaseMediaKey(value string) bool {
	return strings.HasPrefix(value, "products/") && !strings.Contains(value, "..") &&
		!strings.HasPrefix(value, "/") && !strings.ContainsAny(value, "\\\n\r\t")
}

func validProductMediaType(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return value == "image/jpeg" || value == "image/png" || value == "image/webp"
}
