package ports

import (
	"context"
	"errors"
)

var (
	ErrPaymentNotFound = errors.New("payment order not found")
	ErrPaymentState    = errors.New("payment order is not payable")
	ErrPaymentMismatch = errors.New("payment webhook does not match the order")
	ErrRefundNotFound  = errors.New("refund not found")
	ErrRefundState     = errors.New("refund is not in a refundable state")
)

// PaymentRefundGateway is intentionally separate from PaymentGateway: refund
// support was added after the original order/verification contract and should
// not force existing gateway fakes or adapters to change their interface.
type PaymentRefundGateway interface {
	CreateRefund(ctx context.Context, request PaymentRefundRequest) (PaymentRefund, error)
}

type PaymentRefundRequest struct {
	ProviderPaymentID string
	AmountCents       int64
	Currency          string
	Receipt           string
}

type PaymentRefund struct {
	ProviderRefundID  string
	ProviderPaymentID string
	AmountCents       int64
	Currency          string
	Status            string
}

type PaymentRefundRecord struct {
	ID                int64
	OrderID           int64
	OrderNumber       string
	OrderStatus       string
	PaymentStatus     string
	ProviderPaymentID string
	ProviderRefundID  string
	Status            string
	AmountCents       int64
	Currency          string
	Reason            string
	RequestKey        string
}

type PaymentRefundRepository interface {
	// BeginRefund derives the refundable amount and currency from the locked
	// PostgreSQL order/payment rows. V1 intentionally supports one full refund
	// per order; callers cannot override the server-authoritative total.
	BeginRefund(ctx context.Context, userID int64, orderNumber, reason, requestKey string) (PaymentRefundRecord, error)
	CompleteRefund(ctx context.Context, refundID int64, refund PaymentRefund) (PaymentRefundRecord, error)
	FailRefund(ctx context.Context, refundID int64, reason string) error
}

type PaymentWebhookRepository interface {
	ProcessWebhook(ctx context.Context, webhook PaymentWebhook, payload []byte) (processed bool, err error)
}

// PaymentOrderRecord is the provider-neutral payment intent read model used
// while creating a gateway order. It contains only the server-authoritative
// amount, currency, ownership, and current provider reference.
type PaymentOrderRecord struct {
	OrderID         int64
	OrderNumber     string
	OrderStatus     string
	Provider        string
	ProviderOrderID string
	PaymentStatus   string
	AmountCents     int64
	Currency        string
}

// PaymentOrderRepository persists the provider reference after the gateway
// order is created. The adapter owns all SQL and locking details.
type PaymentOrderRepository interface {
	GetPaymentOrder(ctx context.Context, userID int64, orderNumber string) (PaymentOrderRecord, error)
	AttachProviderOrder(ctx context.Context, userID int64, orderNumber string, providerOrder PaymentOrder) (PaymentOrderRecord, error)
}
