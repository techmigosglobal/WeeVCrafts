package postgres

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	domaincommerce "github.com/wecratfs/commerce/internal/domain/commerce"
	"github.com/wecratfs/commerce/internal/ports"
)

type commerceFixture struct {
	userIDs   []int64
	ownerID   int64
	cartIDs   []int64
	productID int64
	variantID int64
	brandSlug string
	category  string
}

func newCommerceFixture(t *testing.T, pool *pgxpool.Pool, userCount, stock int) commerceFixture {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	brandSlug := "fixture-brand-" + suffix
	categorySlug := "fixture-category-" + suffix
	if _, err := pool.Exec(ctx, `INSERT INTO brands (slug, name) VALUES ($1, $2)`, brandSlug, "Fixture Brand"); err != nil {
		t.Fatalf("insert fixture brand: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO categories (slug, name) VALUES ($1, $2)`, categorySlug, "Fixture Category"); err != nil {
		t.Fatalf("insert fixture category: %v", err)
	}

	var ownerID int64
	if err := pool.QueryRow(ctx, `INSERT INTO users (email, display_name, status) VALUES ($1, $2, 'active') RETURNING id`, "fixture-owner-"+suffix+"@example.invalid", "Fixture Owner").Scan(&ownerID); err != nil {
		t.Fatalf("insert fixture owner: %v", err)
	}
	var sellerID int64
	if err := pool.QueryRow(ctx, `INSERT INTO sellers (owner_user_id, display_name, status) VALUES ($1, $2, 'active') RETURNING id`, ownerID, "Fixture Seller").Scan(&sellerID); err != nil {
		t.Fatalf("insert fixture seller: %v", err)
	}
	var productID int64
	if err := pool.QueryRow(ctx, `INSERT INTO products (slug, brand_slug, category_slug, name, description, price_cents, seller_id, status) VALUES ($1, $2, $3, 'Fixture Product', 'Concurrency fixture', 1000, $4, 'approved') RETURNING id`, "fixture-product-"+suffix, brandSlug, categorySlug, sellerID).Scan(&productID); err != nil {
		t.Fatalf("insert fixture product: %v", err)
	}
	var variantID int64
	if err := pool.QueryRow(ctx, `INSERT INTO product_variants (product_id, sku, price_cents) VALUES ($1, $2, 1000) RETURNING id`, productID, "FIXTURE-"+suffix).Scan(&variantID); err != nil {
		t.Fatalf("insert fixture variant: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO inventory_stock (variant_id, available_quantity) VALUES ($1, $2)`, variantID, stock); err != nil {
		t.Fatalf("insert fixture stock: %v", err)
	}

	userIDs := make([]int64, 0, userCount)
	cartIDs := make([]int64, 0, userCount)
	for i := 0; i < userCount; i++ {
		var userID, cartID int64
		if err := pool.QueryRow(ctx, `INSERT INTO users (email, display_name, status) VALUES ($1, $2, 'active') RETURNING id`, fmt.Sprintf("fixture-customer-%s-%d@example.invalid", suffix, i), "Fixture Customer").Scan(&userID); err != nil {
			t.Fatalf("insert fixture customer %d: %v", i, err)
		}
		if err := pool.QueryRow(ctx, `INSERT INTO carts (user_id, expires_at) VALUES ($1, NOW() + INTERVAL '1 hour') RETURNING id`, userID).Scan(&cartID); err != nil {
			t.Fatalf("insert fixture cart %d: %v", i, err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO cart_items (cart_id, variant_id, quantity) VALUES ($1, $2, 1)`, cartID, variantID); err != nil {
			t.Fatalf("insert fixture cart item %d: %v", i, err)
		}
		userIDs = append(userIDs, userID)
		cartIDs = append(cartIDs, cartID)
	}

	fixture := commerceFixture{userIDs: userIDs, ownerID: ownerID, cartIDs: cartIDs, productID: productID, variantID: variantID, brandSlug: brandSlug, category: categorySlug}
	t.Cleanup(func() {
		cleanupContext, cleanupCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cleanupCancel()
		_, _ = pool.Exec(cleanupContext, `DELETE FROM outbox_events WHERE aggregate_type = 'Order' AND aggregate_id IN (SELECT order_number FROM orders WHERE user_id = ANY($1))`, userIDs)
		_, _ = pool.Exec(cleanupContext, `DELETE FROM orders WHERE user_id = ANY($1)`, userIDs)
		_, _ = pool.Exec(cleanupContext, `DELETE FROM carts WHERE user_id = ANY($1)`, userIDs)
		_, _ = pool.Exec(cleanupContext, `DELETE FROM inventory_transactions WHERE variant_id = $1`, variantID)
		_, _ = pool.Exec(cleanupContext, `DELETE FROM users WHERE id = ANY($1)`, userIDs)
		_, _ = pool.Exec(cleanupContext, `DELETE FROM inventory_stock WHERE variant_id = $1`, variantID)
		_, _ = pool.Exec(cleanupContext, `DELETE FROM products WHERE id = $1`, productID)
		_, _ = pool.Exec(cleanupContext, `DELETE FROM sellers WHERE id = $1`, sellerID)
		_, _ = pool.Exec(cleanupContext, `DELETE FROM users WHERE id = $1`, ownerID)
		_, _ = pool.Exec(cleanupContext, `DELETE FROM brands WHERE slug = $1`, brandSlug)
		_, _ = pool.Exec(cleanupContext, `DELETE FROM categories WHERE slug = $1`, categorySlug)
	})
	return fixture
}

func TestInventoryReservationConcurrencyCheckpoint(t *testing.T) {
	pool := openIntegrationPool(t)
	fixture := newCommerceFixture(t, pool, 100, 10)
	repository := NewCommerceRepository(pool)
	address := domaincommerce.AddressInput{RecipientName: "Fixture", Line1: "1 Test Lane", City: "Pune", State: "MH", PostalCode: "411001", CountryCode: "IN"}

	results := make(chan error, len(fixture.userIDs))
	var group sync.WaitGroup
	for index, userID := range fixture.userIDs {
		userID := userID
		cartID := fixture.cartIDs[index]
		group.Add(1)
		go func() {
			defer group.Done()
			_, err := repository.CreateOrder(context.Background(), userID, cartID, fmt.Sprintf("concurrency-%d", userID), address)
			results <- err
		}()
	}
	group.Wait()
	close(results)

	var successes, stockErrors int
	for err := range results {
		if err == nil {
			successes++
		} else if errors.Is(err, ports.ErrInsufficientStock) {
			stockErrors++
		} else {
			t.Fatalf("unexpected concurrent checkout error: %v", err)
		}
	}
	if successes != 10 || stockErrors != 90 {
		t.Fatalf("expected 10 successes and 90 stock errors, got successes=%d stock_errors=%d", successes, stockErrors)
	}
	var available, reserved int
	if err := pool.QueryRow(context.Background(), `SELECT available_quantity, reserved_quantity FROM inventory_stock WHERE variant_id = $1`, fixture.variantID).Scan(&available, &reserved); err != nil {
		t.Fatalf("read fixture stock: %v", err)
	}
	if available != 0 || reserved != 10 {
		t.Fatalf("inventory oversold or reservation count incorrect: available=%d reserved=%d", available, reserved)
	}
}

func TestManualPaymentOrderPersistsAndEntersSellerFulfillment(t *testing.T) {
	pool := openIntegrationPool(t)
	fixture := newCommerceFixture(t, pool, 1, 3)
	repository := NewCommerceRepository(pool)
	address := domaincommerce.AddressInput{RecipientName: "Fixture", Line1: "1 Test Lane", City: "Pune", State: "MH", PostalCode: "411001", CountryCode: "IN"}
	order, err := repository.CreateManualOrder(context.Background(), fixture.userIDs[0], fixture.cartIDs[0], "manual-checkout-key-12345", address)
	if err != nil {
		t.Fatalf("create manual order: %v", err)
	}
	if order.Status != "processing" || order.PaymentMethod != "cash_on_delivery" {
		t.Fatalf("manual order state = status %q, payment method %q", order.Status, order.PaymentMethod)
	}
	var available, reserved, committedReservations, paymentAttempts int
	if err := pool.QueryRow(context.Background(), `SELECT available_quantity, reserved_quantity FROM inventory_stock WHERE variant_id = $1`, fixture.variantID).Scan(&available, &reserved); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM inventory_reservations WHERE order_id = $1 AND status = 'committed'`, order.ID).Scan(&committedReservations); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM payment_attempts WHERE order_id = $1`, order.ID).Scan(&paymentAttempts); err != nil {
		t.Fatal(err)
	}
	if available != 2 || reserved != 0 || committedReservations != 1 || paymentAttempts != 0 {
		t.Fatalf("stock/payment state = available:%d reserved:%d committed:%d payment-attempts:%d", available, reserved, committedReservations, paymentAttempts)
	}
	detail, err := repository.GetOrder(context.Background(), fixture.userIDs[0], order.OrderNumber)
	if err != nil {
		t.Fatalf("read manual order: %v", err)
	}
	if detail.PaymentStatus != "due_on_delivery" || detail.PaymentMethod != "cash_on_delivery" {
		t.Fatalf("manual payment view = method %q, status %q", detail.PaymentMethod, detail.PaymentStatus)
	}
	sellerOrders, err := repository.ListSellerOrders(context.Background(), fixture.ownerID)
	if err != nil {
		t.Fatalf("list seller orders: %v", err)
	}
	if len(sellerOrders) != 1 || sellerOrders[0].PaymentStatus != "due_on_delivery" || sellerOrders[0].OrderNumber != order.OrderNumber {
		t.Fatalf("seller order projection did not include the unpaid order: %#v", sellerOrders)
	}
	if err := repository.UpdateSellerFulfillment(context.Background(), fixture.ownerID, sellerOrders[0].SellerID, order.OrderNumber, "processing", "", "", "COD order accepted"); err != nil {
		t.Fatalf("start seller fulfilment: %v", err)
	}
	detail, err = repository.GetOrder(context.Background(), fixture.userIDs[0], order.OrderNumber)
	if err != nil || len(detail.Fulfillments) != 1 || detail.Fulfillments[0].Status != "processing" {
		t.Fatalf("customer fulfilment projection = %#v, err=%v", detail.Fulfillments, err)
	}
}

func TestCustomerCanCancelManualOrderBeforeSellerFulfillment(t *testing.T) {
	pool := openIntegrationPool(t)
	fixture := newCommerceFixture(t, pool, 1, 1)
	repository := NewCommerceRepository(pool)
	address := domaincommerce.AddressInput{RecipientName: "Fixture", Line1: "1 Test Lane", City: "Pune", State: "MH", PostalCode: "411001", CountryCode: "IN"}
	order, err := repository.CreateManualOrder(context.Background(), fixture.userIDs[0], fixture.cartIDs[0], "manual-cancel-checkout-key", address)
	if err != nil {
		t.Fatalf("create manual order: %v", err)
	}
	if err := repository.CancelOrder(context.Background(), fixture.userIDs[0], order.OrderNumber); err != nil {
		t.Fatalf("cancel before seller fulfilment: %v", err)
	}
	var status, reservationStatus string
	var available, reserved int
	if err := pool.QueryRow(context.Background(), `SELECT status FROM orders WHERE id = $1`, order.ID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT status FROM inventory_reservations WHERE order_id = $1`, order.ID).Scan(&reservationStatus); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT available_quantity, reserved_quantity FROM inventory_stock WHERE variant_id = $1`, fixture.variantID).Scan(&available, &reserved); err != nil {
		t.Fatal(err)
	}
	if status != "cancelled" || reservationStatus != "released" || available != 1 || reserved != 0 {
		t.Fatalf("cancel result order=%q reservation=%q available=%d reserved=%d", status, reservationStatus, available, reserved)
	}
	if err := repository.CancelOrder(context.Background(), fixture.userIDs[0], order.OrderNumber); !errors.Is(err, ports.ErrInvalidState) {
		t.Fatalf("second cancel error = %v, want invalid state", err)
	}
}

func TestExpiredReservationReleasesStockAndMarksPaymentFailed(t *testing.T) {
	pool := openIntegrationPool(t)
	fixture := newCommerceFixture(t, pool, 1, 1)
	repository := NewCommerceRepository(pool)
	address := domaincommerce.AddressInput{RecipientName: "Fixture", Line1: "1 Test Lane", City: "Pune", State: "MH", PostalCode: "411001", CountryCode: "IN"}
	cartID := fixture.cartIDs[0]
	order, err := repository.CreateOrder(context.Background(), fixture.userIDs[0], cartID, "expiry-checkout-key-1234", address)
	if err != nil {
		t.Fatalf("create expiry fixture order: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `UPDATE inventory_reservations SET expires_at = NOW() - INTERVAL '1 minute' WHERE order_id = $1`, order.ID); err != nil {
		t.Fatalf("expire fixture reservation: %v", err)
	}
	released, err := repository.ReleaseExpiredReservations(context.Background(), 100)
	if err != nil || released < 1 {
		t.Fatalf("release expired reservation: released=%d err=%v", released, err)
	}
	var available, reserved int
	if err := pool.QueryRow(context.Background(), `SELECT available_quantity, reserved_quantity FROM inventory_stock WHERE variant_id = $1`, fixture.variantID).Scan(&available, &reserved); err != nil {
		t.Fatalf("read released stock: %v", err)
	}
	if available != 1 || reserved != 0 {
		t.Fatalf("expired reservation did not release stock: available=%d reserved=%d", available, reserved)
	}
	var orderStatus, reservationStatus, paymentStatus string
	if err := pool.QueryRow(context.Background(), `SELECT o.status, r.status, pa.status FROM orders o JOIN inventory_reservations r ON r.order_id = o.id JOIN payment_attempts pa ON pa.order_id = o.id WHERE o.id = $1`, order.ID).Scan(&orderStatus, &reservationStatus, &paymentStatus); err != nil {
		t.Fatalf("read expired states: %v", err)
	}
	if orderStatus != "payment_failed" || reservationStatus != "expired" || paymentStatus != "failed" {
		t.Fatalf("unexpected expired states: order=%s reservation=%s payment=%s", orderStatus, reservationStatus, paymentStatus)
	}
}

func TestCancelPendingOrderReleasesReservation(t *testing.T) {
	pool := openIntegrationPool(t)
	fixture := newCommerceFixture(t, pool, 1, 1)
	repository := NewCommerceRepository(pool)
	address := domaincommerce.AddressInput{RecipientName: "Fixture", Line1: "1 Test Lane", City: "Pune", State: "MH", PostalCode: "411001", CountryCode: "IN"}
	order, err := repository.CreateOrder(context.Background(), fixture.userIDs[0], fixture.cartIDs[0], "cancel-checkout-key-1234", address)
	if err != nil {
		t.Fatalf("create cancellable order: %v", err)
	}
	if err := repository.CancelOrder(context.Background(), fixture.userIDs[0], order.OrderNumber); err != nil {
		t.Fatalf("cancel pending order: %v", err)
	}
	var available, reserved int
	if err := pool.QueryRow(context.Background(), `SELECT available_quantity, reserved_quantity FROM inventory_stock WHERE variant_id = $1`, fixture.variantID).Scan(&available, &reserved); err != nil {
		t.Fatalf("read cancelled stock: %v", err)
	}
	if available != 1 || reserved != 0 {
		t.Fatalf("cancelled order did not release stock: available=%d reserved=%d", available, reserved)
	}
}
