package postgres

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/wecratfs/commerce/internal/ports"
)

func TestMediaLifecyclePersistsOnlyVerifiedReadyReferences(t *testing.T) {
	pool := openIntegrationPool(t)
	fixture := newCommerceFixture(t, pool, 1, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	repository := NewMediaRepository(pool)

	mediaID, err := repository.PrepareUpload(ctx, fixture.ownerID, fixture.productID, "products/media-fixture/object", "image/png", 12)
	if err != nil {
		t.Fatalf("prepare media: %v", err)
	}
	t.Cleanup(func() {
		cleanupContext, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = pool.Exec(cleanupContext, `DELETE FROM audit_logs WHERE action = 'catalog.media_finalized' AND resource_id = $1`, strconv.FormatInt(mediaID, 10))
	})
	if _, err := repository.GetOwnedMedia(ctx, fixture.ownerID, mediaID); err != nil {
		t.Fatalf("read owned pending media: %v", err)
	}
	if _, err := repository.GetOwnedMedia(ctx, fixture.userIDs[0]+1000, mediaID); !errors.Is(err, ports.ErrForbidden) {
		t.Fatalf("expected cross-seller media denial, got %v", err)
	}

	ready, err := repository.FinalizeMedia(ctx, fixture.ownerID, mediaID, "/media/products/"+strconv.FormatInt(mediaID, 10), "image/png", 12)
	if err != nil {
		t.Fatalf("finalize media: %v", err)
	}
	if ready.Status != "ready" || ready.Visibility != "public" || ready.PublicPath == "" {
		t.Fatalf("unexpected ready media: %+v", ready)
	}
	public, err := repository.GetPublicMedia(ctx, mediaID)
	if err != nil || public.PublicPath != ready.PublicPath {
		t.Fatalf("read public media: record=%+v err=%v", public, err)
	}
	var productImage string
	if err := pool.QueryRow(ctx, `SELECT image_url FROM products WHERE id = $1`, fixture.productID).Scan(&productImage); err != nil {
		t.Fatalf("read product media reference: %v", err)
	}
	if productImage != ready.PublicPath {
		t.Fatalf("product image reference was not finalized: %q", productImage)
	}

	replayed, err := repository.FinalizeMedia(ctx, fixture.ownerID, mediaID, "/media/products/ignored", "image/png", 12)
	if err != nil || replayed.PublicPath != ready.PublicPath {
		t.Fatalf("finalize replay changed media: record=%+v err=%v", replayed, err)
	}
	var auditCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit_logs WHERE action = 'catalog.media_finalized' AND resource_id = $1`, strconv.FormatInt(mediaID, 10)).Scan(&auditCount); err != nil {
		t.Fatalf("count media audit: %v", err)
	}
	if auditCount != 1 {
		t.Fatalf("expected one media finalization audit, got %d", auditCount)
	}
	claimed, err := repository.ClaimOwnedMedia(ctx, fixture.ownerID, mediaID)
	if err != nil || claimed.Status != "deleting" {
		t.Fatalf("claim media deletion: record=%+v err=%v", claimed, err)
	}
	if err := repository.MarkMediaDeleted(ctx, mediaID); err != nil {
		t.Fatalf("mark media deleted: %v", err)
	}
	if _, err := repository.GetPublicMedia(ctx, mediaID); !errors.Is(err, ports.ErrNotFound) {
		t.Fatalf("deleted media remained public: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT image_url FROM products WHERE id = $1`, fixture.productID).Scan(&productImage); err != nil {
		t.Fatalf("read deleted product media reference: %v", err)
	}
	if productImage != "" {
		t.Fatalf("deleted media reference remained on product: %q", productImage)
	}
}
