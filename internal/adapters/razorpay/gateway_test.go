package razorpay

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/wecratfs/commerce/internal/ports"
)

func TestGatewayCreatesServerSideOrderAndChecksAmount(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/orders" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		username, password, ok := r.BasicAuth()
		if !ok || username != "key" || password != "secret" {
			t.Fatalf("unexpected basic auth: %q %q", username, password)
		}
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if request["amount"] != float64(13900) || request["currency"] != "INR" || request["receipt"] != "WC-42" {
			t.Fatalf("unexpected provider request: %+v", request)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"order_42","amount":13900,"currency":"INR"}`))
	}))
	defer server.Close()

	order, err := NewGateway("key", "secret", "webhook", server.URL).CreateOrder(context.Background(), ports.PaymentOrderRequest{Reference: "WC-42", AmountCents: 13900, Currency: "INR"})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	if order.ProviderOrderID != "order_42" {
		t.Fatalf("unexpected provider order: %+v", order)
	}
}

func TestGatewayRejectsProviderAmountMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":"order_bad","amount":1,"currency":"INR"}`))
	}))
	defer server.Close()

	_, err := NewGateway("key", "secret", "webhook", server.URL).CreateOrder(context.Background(), ports.PaymentOrderRequest{Reference: "WC-42", AmountCents: 13900, Currency: "INR"})
	if err == nil {
		t.Fatal("expected provider amount mismatch error")
	}
}

func TestGatewayCreatesRefundWithAuthoritativeAmount(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/payments/pay_42/refund" {
			t.Fatalf("unexpected refund request: %s %s", r.Method, r.URL.Path)
		}
		username, password, ok := r.BasicAuth()
		if !ok || username != "key" || password != "secret" {
			t.Fatalf("unexpected basic auth: %q %q", username, password)
		}
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode refund request: %v", err)
		}
		if request["amount"] != float64(13900) || request["receipt"] != "refund-7" {
			t.Fatalf("unexpected provider refund request: %+v", request)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"rfnd_42","amount":13900,"currency":"INR","payment_id":"pay_42","status":"processed"}`))
	}))
	defer server.Close()

	refund, err := NewGateway("key", "secret", "webhook", server.URL).CreateRefund(context.Background(), ports.PaymentRefundRequest{ProviderPaymentID: "pay_42", AmountCents: 13900, Currency: "INR", Receipt: "refund-7"})
	if err != nil {
		t.Fatalf("create refund: %v", err)
	}
	if refund.ProviderRefundID != "rfnd_42" || refund.ProviderPaymentID != "pay_42" || refund.Status != "processed" {
		t.Fatalf("unexpected refund: %+v", refund)
	}
}

func TestGatewayVerifiesRawWebhookSignatureAndPayload(t *testing.T) {
	gateway := NewGateway("key", "secret", "webhook-secret", "http://payments.invalid")
	raw := []byte(`{"id":"evt_123","event":"payment.captured","payload":{"payment":{"entity":{"id":"pay_123","order_id":"order_123","amount":12900,"currency":"INR","status":"captured"}}}}`)
	digest := hmac.New(sha256.New, []byte("webhook-secret"))
	_, _ = digest.Write(raw)
	webhook, err := gateway.VerifyWebhook(context.Background(), raw, hex.EncodeToString(digest.Sum(nil)))
	if err != nil {
		t.Fatalf("verify webhook: %v", err)
	}
	if webhook.EventID != "evt_123" || webhook.ProviderPaymentID != "pay_123" || webhook.AmountCents != 12900 {
		t.Fatalf("unexpected webhook: %+v", webhook)
	}
	if _, err := gateway.VerifyWebhook(context.Background(), raw, "bad"); err != ErrInvalidSignature {
		t.Fatalf("expected invalid signature, got %v", err)
	}
}
