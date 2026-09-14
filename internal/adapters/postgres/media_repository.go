package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	domaincommerce "github.com/wecratfs/commerce/internal/domain/commerce"
	"github.com/wecratfs/commerce/internal/ports"
)

type MediaRepository struct{ pool *pgxpool.Pool }

func NewMediaRepository(pool *pgxpool.Pool) *MediaRepository { return &MediaRepository{pool: pool} }

func (r *MediaRepository) PrepareUpload(ctx context.Context, sellerOwnerID, productID int64, objectKey, contentType string, byteSize int64) (int64, error) {
	var mediaID int64
	err := r.pool.QueryRow(ctx, `INSERT INTO product_media (product_id, seller_id, object_key, content_type, byte_size) SELECT p.id, s.id, $3, $4, $5 FROM products p JOIN sellers s ON s.id = p.seller_id WHERE p.id = $1 AND s.owner_user_id = $2 RETURNING product_media.id`, productID, sellerOwnerID, objectKey, contentType, byteSize).Scan(&mediaID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ports.ErrForbidden
	}
	return mediaID, err
}

const mediaRecordSelect = `
	SELECT pm.id, pm.product_id, pm.seller_id, pm.object_key, pm.content_type,
	       pm.byte_size, pm.visibility, pm.status, pm.public_url, pm.created_at
	FROM product_media pm
`

func scanMediaRecord(row pgx.Row) (domaincommerce.MediaRecord, error) {
	var record domaincommerce.MediaRecord
	err := row.Scan(&record.ID, &record.ProductID, &record.SellerID, &record.ObjectKey, &record.ContentType, &record.ByteSize, &record.Visibility, &record.Status, &record.PublicPath, &record.CreatedAt)
	return record, err
}

func (r *MediaRepository) GetOwnedMedia(ctx context.Context, sellerOwnerID, mediaID int64) (domaincommerce.MediaRecord, error) {
	record, err := scanMediaRecord(r.pool.QueryRow(ctx, mediaRecordSelect+`JOIN sellers s ON s.id = pm.seller_id WHERE pm.id = $1 AND s.owner_user_id = $2 AND pm.status <> 'deleted'`, mediaID, sellerOwnerID))
	if errors.Is(err, pgx.ErrNoRows) {
		return domaincommerce.MediaRecord{}, ports.ErrForbidden
	}
	return record, err
}

func (r *MediaRepository) FinalizeMedia(ctx context.Context, sellerOwnerID, mediaID int64, publicPath, contentType string, byteSize int64) (domaincommerce.MediaRecord, error) {
	var record domaincommerce.MediaRecord
	err := WithinTransaction(ctx, r.pool, func(transactionContext context.Context, tx pgx.Tx) error {
		row := tx.QueryRow(transactionContext, mediaRecordSelect+`JOIN sellers s ON s.id = pm.seller_id WHERE pm.id = $1 AND s.owner_user_id = $2 FOR UPDATE`, mediaID, sellerOwnerID)
		current, err := scanMediaRecord(row)
		if errors.Is(err, pgx.ErrNoRows) {
			return ports.ErrForbidden
		}
		if err != nil {
			return err
		}
		if current.Status == "ready" {
			record = current
			return nil
		}
		if current.Status != "pending" || publicPath == "" || contentType == "" || byteSize <= 0 {
			return ports.ErrInvalidState
		}
		row = tx.QueryRow(transactionContext, `
			UPDATE product_media
			SET content_type = $1, byte_size = $2, status = 'ready', public_url = $3, updated_at = NOW()
			WHERE id = $4
			RETURNING id, product_id, seller_id, object_key, content_type, byte_size, visibility, status, public_url, created_at`, contentType, byteSize, publicPath, mediaID)
		record, err = scanMediaRecord(row)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(transactionContext, `UPDATE products SET image_url = $1, updated_at = NOW() WHERE id = $2`, publicPath, record.ProductID); err != nil {
			return err
		}
		_, err = tx.Exec(transactionContext, `INSERT INTO audit_logs (actor_id, action, resource_type, resource_id, request_id, metadata) VALUES ($1, 'catalog.media_finalized', 'product_media', $2, $3, jsonb_build_object('product_id', $4::bigint, 'content_type', $5::text, 'byte_size', $6::bigint))`, fmt.Sprint(sellerOwnerID), fmt.Sprint(mediaID), ports.RequestID(transactionContext), record.ProductID, contentType, byteSize)
		return err
	})
	return record, err
}

func (r *MediaRepository) GetPublicMedia(ctx context.Context, mediaID int64) (domaincommerce.MediaRecord, error) {
	record, err := scanMediaRecord(r.pool.QueryRow(ctx, mediaRecordSelect+`WHERE pm.id = $1 AND pm.status = 'ready' AND pm.visibility = 'public'`, mediaID))
	if errors.Is(err, pgx.ErrNoRows) {
		return domaincommerce.MediaRecord{}, ports.ErrNotFound
	}
	return record, err
}

func (r *MediaRepository) ClaimOwnedMedia(ctx context.Context, sellerOwnerID, mediaID int64) (domaincommerce.MediaRecord, error) {
	row := r.pool.QueryRow(ctx, `
		WITH candidate AS (
			SELECT pm.id
			FROM product_media pm
			JOIN sellers s ON s.id = pm.seller_id
			WHERE pm.id = $1 AND s.owner_user_id = $2 AND pm.status IN ('pending', 'ready')
			FOR UPDATE
		)
		UPDATE product_media pm
		SET status = 'deleting', updated_at = NOW()
		FROM candidate c
		WHERE pm.id = c.id
		RETURNING pm.id, pm.product_id, pm.seller_id, pm.object_key, pm.content_type,
		          pm.byte_size, pm.visibility, pm.status, pm.public_url, pm.created_at`, mediaID, sellerOwnerID)
	record, err := scanMediaRecord(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domaincommerce.MediaRecord{}, ports.ErrForbidden
	}
	return record, err
}

func (r *MediaRepository) MarkMediaDeleted(ctx context.Context, mediaID int64) error {
	return WithinTransaction(ctx, r.pool, func(transactionContext context.Context, tx pgx.Tx) error {
		if _, err := tx.Exec(transactionContext, `UPDATE products p SET image_url = '' FROM product_media pm WHERE pm.id = $1 AND p.id = pm.product_id AND p.image_url = pm.public_url`, mediaID); err != nil {
			return err
		}
		_, err := tx.Exec(transactionContext, `UPDATE product_media SET status = 'deleted', public_url = '', updated_at = NOW() WHERE id = $1 AND status IN ('pending', 'deleting')`, mediaID)
		return err
	})
}

func (r *MediaRepository) RestoreMedia(ctx context.Context, mediaID int64) error {
	_, err := r.pool.Exec(ctx, `UPDATE product_media SET status = CASE WHEN public_url <> '' THEN 'ready' ELSE 'pending' END, updated_at = NOW() WHERE id = $1 AND status = 'deleting'`, mediaID)
	return err
}

func (r *MediaRepository) ClaimExpiredMedia(ctx context.Context, olderThan time.Time, limit int) ([]domaincommerce.MediaRecord, error) {
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	rows, err := r.pool.Query(ctx, `
		WITH candidates AS (
			SELECT id FROM product_media
			WHERE status = 'pending' AND created_at < $1
			ORDER BY created_at, id
			FOR UPDATE SKIP LOCKED
			LIMIT $2
		)
		UPDATE product_media pm
		SET status = 'deleting', updated_at = NOW()
		FROM candidates c
		WHERE pm.id = c.id
		RETURNING pm.id, pm.product_id, pm.seller_id, pm.object_key, pm.content_type,
		          pm.byte_size, pm.visibility, pm.status, pm.public_url, pm.created_at`, olderThan, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	media := make([]domaincommerce.MediaRecord, 0, limit)
	for rows.Next() {
		record, scanErr := scanMediaRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		media = append(media, record)
	}
	return media, rows.Err()
}

func (r *MediaRepository) ListExpiredMedia(ctx context.Context, olderThan time.Time, limit int) ([]domaincommerce.MediaRecord, error) {
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	rows, err := r.pool.Query(ctx, mediaRecordSelect+`WHERE pm.status = 'pending' AND pm.created_at < $1 ORDER BY pm.created_at, pm.id LIMIT $2`, olderThan, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	media := make([]domaincommerce.MediaRecord, 0, limit)
	for rows.Next() {
		record, scanErr := scanMediaRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		media = append(media, record)
	}
	return media, rows.Err()
}
