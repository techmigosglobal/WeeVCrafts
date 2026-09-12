package payment

import (
	"context"
	"errors"
	"testing"

	"github.com/wecratfs/commerce/internal/ports"
)

type fakePaymentGateway struct {
	created int
	request ports.PaymentOrderRequest
	order   ports.PaymentOrder
}

func (f *fakePaymentGateway) CreateOrder(_ context.Context, request ports.PaymentOrderRequest) (ports.PaymentOrder, error) {
	f.created++
	f.request = request
	return f.order, nil
}

func (f *fakePaymentGateway) VerifyPayment(context.Context, ports.PaymentVerification) error {
	return nil
}

type fakePaymentOrderRepository struct {
	record ports.PaymentOrderRecord
	attach ports.PaymentOrder
}

func (f *fakePaymentOrderRepository) GetPaymentOrder(context.Context, int64, string) (ports.PaymentOrderRecord, error) {
	return f.record, nil
}

func (f *fakePaymentOrderRepository) AttachProviderOrder(_ context.Context, _ int64, _ string, providerOrder ports.PaymentOrder) (ports.PaymentOrderRecord, error) {
	f.attach = providerOrder
	f.record.ProviderOrderID = providerOrder.ProviderOrderID
	f.record.PaymentStatus = "pending"
	return f.record, nil
}

func TestOrderServiceCreatesGatewayOrderFromAuthoritativeTotal(t *testing.T) {
	repository := &fakePaymentOrderRepository{record: ports.PaymentOrderRecord{
		OrderID:       42,
		OrderNumber:   "WC-42",
		OrderStatus:   "pending_payment",
		Provider:      "test-provider",
		PaymentStatus: "created",
		AmountCents:   13900,
		Currency:      "INR",
	}}
	gateway := &fakePaymentGateway{order: ports.PaymentOrder{ProviderOrderID: "order_42", AmountCents: 13900, Currency: "INR"}}
	service := NewOrderService(gateway, repository)

	intent, err := service.Create(context.Background(), 7, "WC-42")
	if err != nil {
		t.Fatalf("create payment order: %v", err)
	}
	if gateway.created != 1 || repository.attach.ProviderOrderID != "order_42" {
		t.Fatalf("expected one provider order attachment, created=%d attach=%+v", gateway.created, repository.attach)
	}
	if gateway.request.Reference != "WC-42" || gateway.request.AmountCents != 13900 || gateway.request.Currency != "INR" {
		t.Fatalf("gateway received non-authoritative request: %+v", gateway.request)
	}
	if intent.AmountCents != 13900 || intent.Currency != "INR" || intent.Status != "pending" {
		t.Fatalf("unexpected payment intent: %+v", intent)
	}
}

func TestOrderServiceReusesExistingProviderOrder(t *testing.T) {
	repository := &fakePaymentOrderRepository{record: ports.PaymentOrderRecord{
		OrderID: 42, OrderNumber: "WC-42", OrderStatus: "pending_payment", Provider: "test-provider",
		ProviderOrderID: "order_existing", PaymentStatus: "pending", AmountCents: 13900, Currency: "INR",
	}}
	gateway := &fakePaymentGateway{order: ports.PaymentOrder{ProviderOrderID: "order_new", AmountCents: 13900, Currency: "INR"}}
	intent, err := NewOrderService(gateway, repository).Create(context.Background(), 7, "WC-42")
	if err != nil {
		t.Fatalf("reuse payment order: %v", err)
	}
	if gateway.created != 0 || intent.ProviderOrderID != "order_existing" {
		t.Fatalf("expected existing provider order, created=%d intent=%+v", gateway.created, intent)
	}
}

func TestOrderServiceRejectsInconsistentGatewayAmount(t *testing.T) {
	repository := &fakePaymentOrderRepository{record: ports.PaymentOrderRecord{
		OrderID: 42, OrderNumber: "WC-42", OrderStatus: "pending_payment", Provider: "test-provider",
		PaymentStatus: "created", AmountCents: 13900, Currency: "INR",
	}}
	gateway := &fakePaymentGateway{order: ports.PaymentOrder{ProviderOrderID: "order_42", AmountCents: 1, Currency: "INR"}}
	_, err := NewOrderService(gateway, repository).Create(context.Background(), 7, "WC-42")
	if !errors.Is(err, ErrPaymentCreationFailed) {
		t.Fatalf("expected inconsistent gateway response rejection, got %v", err)
	}
}
