package ports

import (
	"context"
	"time"
)

// PaymentGateway is the application boundary for creating and verifying a
// payment-provider order. Provider SDK types must not cross this boundary.
type PaymentGateway interface {
	CreateOrder(ctx context.Context, request PaymentOrderRequest) (PaymentOrder, error)
	VerifyPayment(ctx context.Context, verification PaymentVerification) error
}

type PaymentOrderRequest struct {
	Reference   string
	AmountCents int64
	Currency    string
}

type PaymentOrder struct {
	ProviderOrderID string
	AmountCents     int64
	Currency        string
}

type PaymentVerification struct {
	ProviderOrderID   string
	ProviderPaymentID string
	Signature         string
}

type PaymentWebhookVerifier interface {
	VerifyWebhook(ctx context.Context, rawBody []byte, signature string) (PaymentWebhook, error)
}

type PaymentWebhook struct {
	EventID           string
	EventType         string
	ProviderOrderID   string
	ProviderPaymentID string
	Status            string
	AmountCents       int64
	Currency          string
}

// SearchEngine is disposable read infrastructure. PostgreSQL remains the
// source of truth when the index is unavailable or stale.
type SearchEngine interface {
	Search(ctx context.Context, request SearchRequest) (SearchResult, error)
}

type SearchDocument struct {
	ID           int64  `json:"id"`
	Slug         string `json:"slug"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Brand        string `json:"brand"`
	CategorySlug string `json:"category_slug"`
	PriceCents   int64  `json:"price_cents"`
	Available    int    `json:"available"`
}

type SearchIndexer interface {
	Rebuild(ctx context.Context, documents []SearchDocument) error
}

type SearchRequest struct {
	Query        string
	CategorySlug string
	Sort         string
	Limit        int
	Offset       int
}

type SearchResult struct {
	ProductSlugs []string
	Total        int
	Facets       []SearchFacet
}

// ObjectStorage owns file bytes and returns opaque references to them. Product
// and order records store metadata, never provider SDK objects.
type ObjectStorage interface {
	CreateUploadURL(ctx context.Context, request UploadRequest) (UploadURL, error)
}

// ObjectStorageReader provides the bounded read/delete capabilities needed to
// verify a browser upload and resolve a public media reference. It is kept
// separate from ObjectStorage so existing upload-only adapters remain valid.
type ObjectStorageReader interface {
	InspectObject(ctx context.Context, objectKey string) (ObjectInfo, error)
	CreateDownloadURL(ctx context.Context, objectKey string, lifetime time.Duration) (DownloadURL, error)
	DeleteObject(ctx context.Context, objectKey string) error
}

// ObjectStorageUploadReceiver is implemented by same-origin storage adapters
// that accept upload bytes through the application rather than a signed URL.
type ObjectStorageUploadReceiver interface {
	SaveObject(ctx context.Context, ownerID int64, objectKey, contentType string, body []byte) error
}

// ObjectStorageUploadCapacity lets an adapter advertise a stricter request
// size ceiling imposed by a serverless platform.
type ObjectStorageUploadCapacity interface {
	MaxUploadBytes() int64
}

// ObjectStorageObjectReader permits first-party routes to serve stored images
// directly; external providers can keep using short-lived redirect URLs.
type ObjectStorageObjectReader interface {
	ReadObject(ctx context.Context, objectKey string) ([]byte, ObjectInfo, error)
}

type UploadRequest struct {
	ObjectKey   string
	ContentType string
	MaxBytes    int64
}

type UploadURL struct {
	URL           string
	ExpiresAtUnix int64
}

type ObjectInfo struct {
	ContentType string
	ByteSize    int64
}

type DownloadURL struct {
	URL           string
	ExpiresAtUnix int64
}

// Cache is disposable infrastructure. A cache miss is expected behavior and
// must not change business correctness.
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttlSeconds int) error
	Delete(ctx context.Context, key string) error
}

// CacheInvalidator lets bounded background rebuilds discard disposable read
// models after authoritative catalog changes. It never participates in a
// business mutation's correctness decision.
type CacheInvalidator interface {
	DeletePrefix(ctx context.Context, prefix string) error
}

type RateLimiter interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error)
}

// SessionStore owns opaque web-session persistence. The browser receives only
// the session identifier; session contents stay in the disposable adapter.
type SessionStore interface {
	Create(ctx context.Context, session SessionRecord) error
	Get(ctx context.Context, sessionID string) (SessionRecord, error)
	Delete(ctx context.Context, sessionID string) error
	DeleteAll(ctx context.Context, userID int64) error
	List(ctx context.Context, userID int64) ([]SessionRecord, error)
}

type SessionRecord struct {
	ID         string
	UserID     int64
	RoleSlugs  []string
	CSRFToken  string
	UserAgent  string
	IPAddress  string
	CreatedAt  time.Time
	LastSeenAt time.Time
	ExpiresAt  time.Time
}

// TransactionRunner makes atomic application mutations explicit without
// leaking a PostgreSQL transaction type into business code.
type TransactionRunner interface {
	WithinTransaction(ctx context.Context, fn func(context.Context) error) error
}
