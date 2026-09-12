package commerce

import "time"

type Seller struct {
	ID          int64  `json:"id"`
	OwnerUserID int64  `json:"owner_user_id"`
	DisplayName string `json:"display_name"`
	Status      string `json:"status"`
}

type ProductDraftInput struct {
	Slug           string
	BrandSlug      string
	BrandName      string
	CategorySlug   string
	CategoryName   string
	Name           string
	Description    string
	PriceCents     int64
	CompareAtCents *int64
	SKU            string
	InitialStock   int
	DeliveryLabel  string
}

type ManagedProduct struct {
	ID           int64  `json:"id"`
	VariantID    int64  `json:"variant_id"`
	SellerID     int64  `json:"seller_id"`
	SellerName   string `json:"seller_name"`
	Slug         string `json:"slug"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	SKU          string `json:"sku"`
	PriceCents   int64  `json:"price_cents"`
	Stock        int    `json:"stock"`
	Status       string `json:"status"`
	CategorySlug string `json:"category_slug"`
}

type CartItem struct {
	VariantID      int64  `json:"variant_id"`
	ProductID      int64  `json:"product_id"`
	ProductSlug    string `json:"product_slug"`
	ProductName    string `json:"product_name"`
	SKU            string `json:"sku"`
	Quantity       int    `json:"quantity"`
	UnitPriceCents int64  `json:"unit_price_cents"`
	LineTotalCents int64  `json:"line_total_cents"`
	Available      int    `json:"available"`
}

type Cart struct {
	ID            int64      `json:"id"`
	Items         []CartItem `json:"items"`
	SubtotalCents int64      `json:"subtotal_cents"`
	ItemCount     int        `json:"item_count"`
	ExpiresAt     time.Time  `json:"expires_at"`
}

type WishlistItem struct {
	ProductID  int64  `json:"product_id"`
	Slug       string `json:"slug"`
	Name       string `json:"name"`
	Brand      string `json:"brand"`
	PriceCents int64  `json:"price_cents"`
	Available  int    `json:"available"`
}

type AddressInput struct {
	RecipientName string `json:"recipient_name"`
	Line1         string `json:"line1"`
	Line2         string `json:"line2"`
	City          string `json:"city"`
	State         string `json:"state"`
	PostalCode    string `json:"postal_code"`
	CountryCode   string `json:"country_code"`
}

type Order struct {
	ID            int64     `json:"id"`
	OrderNumber   string    `json:"order_number"`
	Status        string    `json:"status"`
	Currency      string    `json:"currency"`
	SubtotalCents int64     `json:"subtotal_cents"`
	ShippingCents int64     `json:"shipping_cents"`
	TotalCents    int64     `json:"total_cents"`
	CreatedAt     time.Time `json:"created_at"`
}

type OrderItem struct {
	ProductName    string `json:"product_name"`
	SKU            string `json:"sku"`
	Quantity       int    `json:"quantity"`
	UnitPriceCents int64  `json:"unit_price_cents"`
	LineTotalCents int64  `json:"line_total_cents"`
}

type OrderDetail struct {
	Order
	Items []OrderItem `json:"items"`
}

type PaymentIntent struct {
	OrderID         int64  `json:"order_id"`
	OrderNumber     string `json:"order_number"`
	Provider        string `json:"provider"`
	ProviderOrderID string `json:"provider_order_id"`
	Status          string `json:"status"`
	AmountCents     int64  `json:"amount_cents"`
	Currency        string `json:"currency"`
}

type Refund struct {
	ID          int64     `json:"id"`
	OrderID     int64     `json:"order_id"`
	OrderNumber string    `json:"order_number"`
	Status      string    `json:"status"`
	AmountCents int64     `json:"amount_cents"`
	Currency    string    `json:"currency"`
	Reason      string    `json:"reason"`
	CreatedAt   time.Time `json:"created_at"`
}
