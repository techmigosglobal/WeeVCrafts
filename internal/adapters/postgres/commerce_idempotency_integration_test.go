package postgres

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	domaincommerce "github.com/wecratfs/commerce/internal/domain/commerce"
	"github.com/wecratfs/commerce/internal/ports"
)

func TestCheckoutIdenticalRetriesCreateOneOrderReservationPaymentAndOutbox(t *testing.T) {
	pool := openIntegrationPool(t)
	fixture := newCommerceFixture(t, pool, 1, 20)
	repository := NewCommerceRepository(pool)
	address := domaincommerce.AddressInput{RecipientName: "Fixture", Line1: "1 Test Lane", City: "Pune", State: "MH", PostalCode: "411001", CountryCode: "IN"}
	const attempts = 20
	orders := make(chan domaincommerce.Order, attempts)
	errorsSeen := make(chan error, attempts)
	var group sync.WaitGroup
	for i := 0; i < attempts; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			order, err := repository.CreateOrder(context.Background(), fixture.userIDs[0], fixture.cartIDs[0], "identical-checkout-key", address)
			orders <- order
			errorsSeen <- err
		}()
	}
	group.Wait()
	close(orders)
	close(errorsSeen)

	var firstOrder domaincommerce.Order
	for order := range orders {
		if firstOrder.OrderNumber == "" {
			firstOrder = order
		}
		if order.OrderNumber != firstOrder.OrderNumber {
			t.Fatalf("identical retries returned different orders: first=%+v current=%+v", firstOrder, order)
		}
	}
	for err := range errorsSeen {
		if err != nil {
			t.Fatalf("identical checkout retry failed: %v", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var orderCount, reservationCount, paymentCount, outboxCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM orders WHERE user_id = $1 AND idempotency_key = 'identical-checkout-key'`, fixture.userIDs[0]).Scan(&orderCount); err != nil {
		t.Fatalf("count idempotent orders: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM inventory_reservations WHERE order_id IN (SELECT id FROM orders WHERE user_id = $1 AND idempotency_key = 'identical-checkout-key')`, fixture.userIDs[0]).Scan(&reservationCount); err != nil {
		t.Fatalf("count idempotent reservations: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM payment_attempts WHERE order_id IN (SELECT id FROM orders WHERE user_id = $1 AND idempotency_key = 'identical-checkout-key')`, fixture.userIDs[0]).Scan(&paymentCount); err != nil {
		t.Fatalf("count idempotent payment attempts: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM outbox_events WHERE event_type = 'OrderCreated' AND aggregate_id = $1`, firstOrder.OrderNumber).Scan(&outboxCount); err != nil {
		t.Fatalf("count idempotent outbox events: %v", err)
	}
	if orderCount != 1 || reservationCount != 1 || paymentCount != 1 || outboxCount != 1 {
		t.Fatalf("expected one order/reservation/payment/outbox, got %d/%d/%d/%d", orderCount, reservationCount, paymentCount, outboxCount)
	}
}

func TestCheckoutIdempotencyKeyRejectsDifferentRequest(t *testing.T) {
	pool := openIntegrationPool(t)
	fixture := newCommerceFixture(t, pool, 1, 2)
	repository := NewCommerceRepository(pool)
	firstAddress := domaincommerce.AddressInput{RecipientName: "Fixture", Line1: "1 Test Lane", City: "Pune", State: "MH", PostalCode: "411001", CountryCode: "IN"}
	secondAddress := firstAddress
	secondAddress.Line1 = "2 Different Lane"
	key := fmt.Sprintf("conflict-checkout-%d", time.Now().UnixNano())
	if _, err := repository.CreateOrder(context.Background(), fixture.userIDs[0], fixture.cartIDs[0], key, firstAddress); err != nil {
		t.Fatalf("create initial checkout: %v", err)
	}
	if _, err := repository.CreateOrder(context.Background(), fixture.userIDs[0], fixture.cartIDs[0], key, secondAddress); !errors.Is(err, ports.ErrIdempotencyConflict) {
		t.Fatalf("expected idempotency conflict, got %v", err)
	}
}
