package postgres

import (
	"context"
	"testing"
	"time"
)

func TestMergeGuestCartClampsDuplicatesAndSkipsUnavailableProducts(t *testing.T) {
	pool := openIntegrationPool(t)
	fixture := newCommerceFixture(t, pool, 1, 5)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	guestHash := "guest-merge-test-" + time.Now().Format("20060102150405.000000000")
	var guestCartID int64
	if err := pool.QueryRow(ctx, `INSERT INTO carts (guest_token_hash, expires_at) VALUES ($1, NOW() + INTERVAL '1 hour') RETURNING id`, guestHash).Scan(&guestCartID); err != nil {
		t.Fatalf("insert guest cart: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE cart_items SET quantity = 2 WHERE cart_id = $1 AND variant_id = $2`, fixture.cartIDs[0], fixture.variantID); err != nil {
		t.Fatalf("update authenticated cart item: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO cart_items (cart_id, variant_id, quantity) VALUES ($1, $2, 98)`, guestCartID, fixture.variantID); err != nil {
		t.Fatalf("insert duplicate guest item: %v", err)
	}

	repository := NewCommerceRepository(pool)
	if err := repository.MergeGuestCart(ctx, fixture.userIDs[0], guestHash); err != nil {
		t.Fatalf("merge guest cart: %v", err)
	}
	var mergedQuantity int
	if err := pool.QueryRow(ctx, `SELECT quantity FROM cart_items WHERE cart_id = $1 AND variant_id = $2`, fixture.cartIDs[0], fixture.variantID).Scan(&mergedQuantity); err != nil {
		t.Fatalf("read merged quantity: %v", err)
	}
	if mergedQuantity != 99 {
		t.Fatalf("expected duplicate quantity clamp at 99, got %d", mergedQuantity)
	}
	var guestStatus string
	if err := pool.QueryRow(ctx, `SELECT status FROM carts WHERE id = $1`, guestCartID).Scan(&guestStatus); err != nil {
		t.Fatalf("read guest cart status: %v", err)
	}
	if guestStatus != "expired" {
		t.Fatalf("expected guest cart to be expired after merge, got %q", guestStatus)
	}

	if _, err := pool.Exec(ctx, `UPDATE products SET status = 'draft' WHERE id = $1`, fixture.productID); err != nil {
		t.Fatalf("make product unavailable: %v", err)
	}
	unavailableHash := guestHash + "-unavailable"
	var unavailableCartID int64
	if err := pool.QueryRow(ctx, `INSERT INTO carts (guest_token_hash, expires_at) VALUES ($1, NOW() + INTERVAL '1 hour') RETURNING id`, unavailableHash).Scan(&unavailableCartID); err != nil {
		t.Fatalf("insert unavailable guest cart: %v", err)
	}
	t.Cleanup(func() {
		cleanupContext, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = pool.Exec(cleanupContext, `DELETE FROM carts WHERE guest_token_hash IN ($1, $2)`, guestHash, unavailableHash)
	})
	if _, err := pool.Exec(ctx, `INSERT INTO cart_items (cart_id, variant_id, quantity) VALUES ($1, $2, 1)`, unavailableCartID, fixture.variantID); err != nil {
		t.Fatalf("insert unavailable guest item: %v", err)
	}
	if err := repository.MergeGuestCart(ctx, fixture.userIDs[0], unavailableHash); err != nil {
		t.Fatalf("merge unavailable guest cart: %v", err)
	}
	var quantityAfterUnavailableMerge int
	if err := pool.QueryRow(ctx, `SELECT quantity FROM cart_items WHERE cart_id = $1 AND variant_id = $2`, fixture.cartIDs[0], fixture.variantID).Scan(&quantityAfterUnavailableMerge); err != nil {
		t.Fatalf("check unavailable merge: %v", err)
	}
	if quantityAfterUnavailableMerge != 99 {
		t.Fatal("unavailable guest item changed the authenticated cart")
	}
}
