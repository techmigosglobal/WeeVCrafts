package razorpay

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/wecratfs/commerce/internal/ports"
)

var ErrInvalidSignature = errors.New("payment signature is invalid")

type Gateway struct {
	keyID         string
	keySecret     string
	webhookSecret string
	baseURL       string
	client        *http.Client
}

func NewGateway(keyID, keySecret, webhookSecret, baseURL string) *Gateway {
	if baseURL == "" {
		baseURL = "https://api.razorpay.com"
	}
	return &Gateway{keyID: keyID, keySecret: keySecret, webhookSecret: webhookSecret, baseURL: strings.TrimRight(baseURL, "/"), client: &http.Client{Timeout: 5 * time.Second}}
}

func (g *Gateway) CreateOrder(ctx context.Context, request ports.PaymentOrderRequest) (ports.PaymentOrder, error) {
	if g.keyID == "" || g.keySecret == "" {
		return ports.PaymentOrder{}, errors.New("razorpay credentials are not configured")
	}
	body, err := json.Marshal(map[string]any{"amount": request.AmountCents, "currency": request.Currency, "receipt": request.Reference, "payment_capture": 0})
	if err != nil {
		return ports.PaymentOrder{}, err
	}
	var response struct {
		ID       string `json:"id"`
		Amount   int64  `json:"amount"`
		Currency string `json:"currency"`
	}
	if err := g.doJSON(ctx, http.MethodPost, "/v1/orders", body, &response); err != nil {
		return ports.PaymentOrder{}, err
	}
	if response.ID == "" || response.Amount != request.AmountCents || response.Currency != request.Currency {
		return ports.PaymentOrder{}, errors.New("razorpay returned an inconsistent order")
	}
	return ports.PaymentOrder{ProviderOrderID: response.ID, AmountCents: response.Amount, Currency: response.Currency}, nil
}

// CreateRefund uses Razorpay's normal refund endpoint. Provider JSON stays in
// this adapter; the application only receives the provider-neutral result.
func (g *Gateway) CreateRefund(ctx context.Context, request ports.PaymentRefundRequest) (ports.PaymentRefund, error) {
	if g.keyID == "" || g.keySecret == "" || request.ProviderPaymentID == "" || request.AmountCents <= 0 || request.Currency == "" || request.Receipt == "" {
		return ports.PaymentRefund{}, errors.New("razorpay refund request is incomplete")
	}
	body, err := json.Marshal(map[string]any{"amount": request.AmountCents, "receipt": request.Receipt})
	if err != nil {
		return ports.PaymentRefund{}, err
	}
	var response struct {
		ID        string `json:"id"`
		Amount    int64  `json:"amount"`
		Currency  string `json:"currency"`
		PaymentID string `json:"payment_id"`
		Status    string `json:"status"`
	}
	path := "/v1/payments/" + url.PathEscape(request.ProviderPaymentID) + "/refund"
	if err := g.doJSON(ctx, http.MethodPost, path, body, &response); err != nil {
		return ports.PaymentRefund{}, err
	}
	if response.ID == "" || response.Amount != request.AmountCents || (response.Currency != "" && response.Currency != request.Currency) {
		return ports.PaymentRefund{}, errors.New("razorpay returned an inconsistent refund")
	}
	if response.PaymentID == "" {
		response.PaymentID = request.ProviderPaymentID
	}
	if response.Status == "" {
		return ports.PaymentRefund{}, errors.New("razorpay returned a refund without a status")
	}
	if response.Currency == "" {
		response.Currency = request.Currency
	}
	return ports.PaymentRefund{ProviderRefundID: response.ID, ProviderPaymentID: response.PaymentID, AmountCents: response.Amount, Currency: response.Currency, Status: response.Status}, nil
}

func (g *Gateway) VerifyPayment(_ context.Context, verification ports.PaymentVerification) error {
	message := verification.ProviderOrderID + "|" + verification.ProviderPaymentID
	if !secureEqualHex(g.keySecret, message, verification.Signature) {
		return ErrInvalidSignature
	}
	return nil
}

func (g *Gateway) VerifyWebhook(_ context.Context, rawBody []byte, signature string) (ports.PaymentWebhook, error) {
	if !secureEqualHex(g.webhookSecret, string(rawBody), signature) {
		return ports.PaymentWebhook{}, ErrInvalidSignature
	}
	var payload struct {
		ID      string `json:"id"`
		Event   string `json:"event"`
		Payload struct {
			Payment struct {
				Entity struct {
					ID       string `json:"id"`
					OrderID  string `json:"order_id"`
					Amount   int64  `json:"amount"`
					Currency string `json:"currency"`
					Status   string `json:"status"`
				} `json:"entity"`
			} `json:"payment"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		return ports.PaymentWebhook{}, err
	}
	if payload.ID == "" || payload.Event == "" {
		return ports.PaymentWebhook{}, errors.New("webhook event is incomplete")
	}
	entity := payload.Payload.Payment.Entity
	return ports.PaymentWebhook{EventID: payload.ID, EventType: payload.Event, ProviderOrderID: entity.OrderID, ProviderPaymentID: entity.ID, Status: entity.Status, AmountCents: entity.Amount, Currency: entity.Currency}, nil
}

func (g *Gateway) doJSON(ctx context.Context, method, path string, body []byte, destination any) error {
	req, err := http.NewRequestWithContext(ctx, method, g.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.SetBasicAuth(g.keyID, g.keySecret)
	req.Header.Set("Content-Type", "application/json")
	response, err := g.client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	limited := io.LimitReader(response.Body, 1<<20)
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		message, _ := io.ReadAll(limited)
		return fmt.Errorf("razorpay returned status %d: %s", response.StatusCode, strings.TrimSpace(string(message)))
	}
	return json.NewDecoder(limited).Decode(destination)
}

func secureEqualHex(secret, message, signature string) bool {
	if secret == "" || signature == "" {
		return false
	}
	digest := hmac.New(sha256.New, []byte(secret))
	_, _ = digest.Write([]byte(message))
	want := make([]byte, hex.DecodedLen(len(signature)))
	n, err := hex.Decode(want, []byte(signature))
	if err != nil {
		return false
	}
	return hmac.Equal(digest.Sum(nil), want[:n])
}
