package payment

import (
	"context"
	"errors"
	"testing"

	"github.com/wecratfs/commerce/internal/ports"
)

type fakeRefundGateway struct {
	calls    int
	request  ports.PaymentRefundRequest
	response ports.PaymentRefund
	err      error
}

func (f *fakeRefundGateway) CreateRefund(_ context.Context, request ports.PaymentRefundRequest) (ports.PaymentRefund, error) {
	f.calls++
	f.request = request
	return f.response, f.err
}

type fakeRefundRepository struct {
	record       ports.PaymentRefundRecord
	completed    ports.PaymentRefund
	completeCall int
	failReason   string
}

func (f *fakeRefundRepository) BeginRefund(context.Context, int64, string, string, string) (ports.PaymentRefundRecord, error) {
	return f.record, nil
}

func (f *fakeRefundRepository) CompleteRefund(_ context.Context, _ int64, refund ports.PaymentRefund) (ports.PaymentRefundRecord, error) {
	f.completeCall++
	f.completed = refund
	f.record.ProviderRefundID = refund.ProviderRefundID
	f.record.Status = refund.Status
	return f.record, nil
}

func (f *fakeRefundRepository) FailRefund(_ context.Context, _ int64, reason string) error {
	f.failReason = reason
	return nil
}

func TestRefundServiceUsesAuthoritativeFullRefundAndReplays(t *testing.T) {
	repository := &fakeRefundRepository{record: ports.PaymentRefundRecord{
		ID: 7, OrderID: 42, OrderNumber: "WC-42", OrderStatus: "paid", PaymentStatus: "captured",
		ProviderPaymentID: "pay_42", Status: "pending", AmountCents: 13900, Currency: "INR", Reason: "damaged",
	}}
	gateway := &fakeRefundGateway{response: ports.PaymentRefund{
		ProviderRefundID: "rfnd_42", ProviderPaymentID: "pay_42", AmountCents: 13900, Currency: "INR", Status: "processed",
	}}
	service := NewRefundService(gateway, repository)

	refund, err := service.Refund(context.Background(), 7, "WC-42", "refund-key-123456", "damaged")
	if err != nil {
		t.Fatalf("refund: %v", err)
	}
	if gateway.calls != 1 || repository.completeCall != 1 || refund.Status != "processed" || refund.AmountCents != 13900 {
		t.Fatalf("unexpected first refund: gateway=%d complete=%d refund=%+v", gateway.calls, repository.completeCall, refund)
	}
	if gateway.request.ProviderPaymentID != "pay_42" || gateway.request.AmountCents != 13900 || gateway.request.Receipt != "refund-7" {
		t.Fatalf("gateway did not receive authoritative request: %+v", gateway.request)
	}

	replayed, err := service.Refund(context.Background(), 7, "WC-42", "refund-key-123456", "damaged")
	if err != nil {
		t.Fatalf("replay refund: %v", err)
	}
	if gateway.calls != 1 || replayed.ID != refund.ID || replayed.Status != "processed" {
		t.Fatalf("refund replay called provider or changed result: calls=%d replay=%+v", gateway.calls, replayed)
	}
}

func TestRefundServiceRejectsProviderMismatchAndRecordsFailure(t *testing.T) {
	repository := &fakeRefundRepository{record: ports.PaymentRefundRecord{
		ID: 8, OrderNumber: "WC-43", Status: "pending", AmountCents: 1000, Currency: "INR", ProviderPaymentID: "pay_43",
	}}
	gateway := &fakeRefundGateway{response: ports.PaymentRefund{
		ProviderRefundID: "rfnd_43", ProviderPaymentID: "pay_43", AmountCents: 999, Currency: "INR", Status: "processed",
	}}
	_, err := NewRefundService(gateway, repository).Refund(context.Background(), 7, "WC-43", "refund-key-123456", "")
	if !errors.Is(err, ErrRefundResponseMismatch) {
		t.Fatalf("expected provider mismatch rejection, got %v", err)
	}
	if gateway.calls != 1 || repository.failReason != ErrRefundResponseMismatch.Error() {
		t.Fatalf("provider mismatch was not recorded: calls=%d reason=%q", gateway.calls, repository.failReason)
	}
}

func TestRefundServiceValidatesRequestBeforeRepository(t *testing.T) {
	gateway := &fakeRefundGateway{}
	repository := &fakeRefundRepository{}
	_, err := NewRefundService(gateway, repository).Refund(context.Background(), 7, "WC-44", "short", "")
	if !errors.Is(err, ErrRefundInvalidRequest) {
		t.Fatalf("expected invalid request, got %v", err)
	}
	if gateway.calls != 0 {
		t.Fatalf("invalid refund reached provider")
	}
}
