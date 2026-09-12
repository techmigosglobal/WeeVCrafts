package postgres

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	razorpayadapter "github.com/wecratfs/commerce/internal/adapters/razorpay"
	applicationpayment "github.com/wecratfs/commerce/internal/application/payment"
	domaincommerce "github.com/wecratfs/commerce/internal/domain/commerce"
	"github.com/wecratfs/commerce/internal/ports"
)

func TestVerifiedPaymentWebhookTransitionsOrderOnceAndCommitsReservation(t *testing.T) {
	pool := openIntegrationPool(t)
	fixture := newCommerceFixture(t, pool, 1, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	address := domaincommerce.AddressInput{RecipientName: "Payment Fixture", Line1: "1 Test Lane", City: "Pune", State: "MH", PostalCode: "411001", CountryCode: "IN"}
	order, err := NewCommerceRepository(pool).CreateOrder(ctx, fixture.userIDs[0], fixture.cartIDs[0], "webhook-integration-key-1234", address)
	if err != nil {
		t.Fatalf("create pending order: %v", err)
	}
	paymentRepository := NewPaymentRepository(pool)
	if _, err := paymentRepository.AttachProviderOrder(ctx, fixture.userIDs[0], order.OrderNumber, ports.PaymentOrder{ProviderOrderID: "order_fixture_" + fmt.Sprint(order.ID), AmountCents: order.TotalCents, Currency: order.Currency}); err != nil {
		t.Fatalf("attach provider order: %v", err)
	}
	eventID := "evt_fixture_" + fmt.Sprint(order.ID)
	paymentID := "pay_fixture_" + fmt.Sprint(order.ID)
	t.Cleanup(func() {
		cleanupContext, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = pool.Exec(cleanupContext, `DELETE FROM payment_webhook_events WHERE event_id = $1`, eventID)
		_, _ = pool.Exec(cleanupContext, `DELETE FROM audit_logs WHERE action = 'payment.webhook_processed' AND resource_id = $1`, fmt.Sprint(order.ID))
	})
	raw := []byte(fmt.Sprintf(`{"id":%q,"event":"payment.captured","payload":{"payment":{"entity":{"id":%q,"order_id":%q,"amount":%d,"currency":%q,"status":"captured"}}}}`, eventID, paymentID, "order_fixture_"+fmt.Sprint(order.ID), order.TotalCents, order.Currency))
	secret := "webhook-fixture-secret"
	digest := hmac.New(sha256.New, []byte(secret))
	_, _ = digest.Write(raw)
	signature := hex.EncodeToString(digest.Sum(nil))
	gateway := razorpayadapter.NewGateway("key", "secret", secret, "http://payments.invalid")
	service := applicationpayment.NewService(gateway, paymentRepository)

	processed, err := service.HandleWebhook(ctx, raw, signature)
	if err != nil || !processed {
		t.Fatalf("process verified capture: processed=%v err=%v", processed, err)
	}
	processed, err = service.HandleWebhook(ctx, raw, signature)
	if err != nil || processed {
		t.Fatalf("replayed capture was not harmless: processed=%v err=%v", processed, err)
	}

	var orderStatus, paymentStatus, reservationStatus string
	var available, reserved int
	if err := pool.QueryRow(ctx, `SELECT o.status, pa.status, r.status, i.available_quantity, i.reserved_quantity FROM orders o JOIN payment_attempts pa ON pa.order_id = o.id JOIN inventory_reservations r ON r.order_id = o.id JOIN inventory_reservation_items ri ON ri.reservation_id = r.id JOIN inventory_stock i ON i.variant_id = ri.variant_id WHERE o.id = $1`, order.ID).Scan(&orderStatus, &paymentStatus, &reservationStatus, &available, &reserved); err != nil {
		t.Fatalf("read captured state: %v", err)
	}
	if orderStatus != "paid" || paymentStatus != "captured" || reservationStatus != "committed" || available != 0 || reserved != 0 {
		t.Fatalf("unexpected captured state: order=%s payment=%s reservation=%s available=%d reserved=%d", orderStatus, paymentStatus, reservationStatus, available, reserved)
	}
	var auditCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit_logs WHERE action = 'payment.webhook_processed' AND resource_id = $1`, fmt.Sprint(order.ID)).Scan(&auditCount); err != nil {
		t.Fatalf("read webhook audit: %v", err)
	}
	if auditCount != 1 {
		t.Fatalf("expected one processed-webhook audit record, got %d", auditCount)
	}

	unknownRaw := []byte(fmt.Sprintf(`{"id":"evt_unknown_%d","event":"payment.unknown","payload":{}}`, order.ID))
	unknownDigest := hmac.New(sha256.New, []byte(secret))
	_, _ = unknownDigest.Write(unknownRaw)
	if _, err := service.HandleWebhook(ctx, unknownRaw, hex.EncodeToString(unknownDigest.Sum(nil))); err != applicationpayment.ErrWebhookRejected {
		t.Fatalf("expected unknown event rejection, got %v", err)
	}
}
