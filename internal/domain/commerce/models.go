package commerce

import "time"

const (
	SellerPermissionProductRead    = "PRODUCT_READ"
	SellerPermissionProductWrite   = "PRODUCT_WRITE"
	SellerPermissionInventoryRead  = "INVENTORY_READ"
	SellerPermissionInventoryWrite = "INVENTORY_WRITE"
	SellerPermissionOrderRead      = "ORDER_READ"
	SellerPermissionOrderFulfill   = "ORDER_FULFILL"
	SellerPermissionReturnRead     = "RETURN_READ"
	SellerPermissionReturnProcess  = "RETURN_PROCESS"
	SellerPermissionReportRead     = "REPORT_READ"
)

var SellerPermissions = []string{
	SellerPermissionProductRead,
	SellerPermissionProductWrite,
	SellerPermissionInventoryRead,
	SellerPermissionInventoryWrite,
	SellerPermissionOrderRead,
	SellerPermissionOrderFulfill,
	SellerPermissionReturnRead,
	SellerPermissionReturnProcess,
	SellerPermissionReportRead,
}

type Seller struct {
	ID          int64  `json:"id"`
	OwnerUserID int64  `json:"owner_user_id"`
	DisplayName string `json:"display_name"`
	Status      string `json:"status"`
}

type SellerAdminEntry struct {
	ID          int64
	OwnerUserID int64
	DisplayName string
	OwnerEmail  string
	OwnerName   string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type AuditEntry struct {
	ID           int64
	ActorID      string
	Action       string
	ResourceType string
	ResourceID   string
	RequestID    string
	Metadata     string
	CreatedAt    time.Time
}

type AdminOrder struct {
	ID                  int64
	OrderNumber         string
	Status              string
	CustomerLabel       string
	CustomerEmailMasked string
	PaymentStatus       string
	FulfillmentStatus   string
	TotalCents          int64
	Currency            string
	CreatedAt           time.Time
}

type AdminBrand struct {
	Slug          string
	Name          string
	ProductCount  int
	CategoryCount int
}

type SellerStaffMember struct {
	UserID      int64
	Email       string
	DisplayName string
	Status      string
	Permissions []string
	AddedAt     time.Time
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
	ID             int64  `json:"id"`
	VariantID      int64  `json:"variant_id"`
	SellerID       int64  `json:"seller_id"`
	SellerName     string `json:"seller_name"`
	Slug           string `json:"slug"`
	BrandSlug      string `json:"brand_slug"`
	BrandName      string `json:"brand_name"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	SKU            string `json:"sku"`
	PriceCents     int64  `json:"price_cents"`
	CompareAtCents int64  `json:"compare_at_cents"`
	HasCompareAt   bool   `json:"has_compare_at"`
	Stock          int    `json:"stock"`
	Status         string `json:"status"`
	CategorySlug   string `json:"category_slug"`
	CategoryName   string `json:"category_name"`
	DeliveryLabel  string `json:"delivery_label"`
}

// InventoryItem is the seller-scoped stock record used by the live seller
// workspace. Available and reserved quantities are kept separate so a seller
// cannot accidentally make stock reserved for a customer available again.
type InventoryItem struct {
	VariantID         int64
	ProductID         int64
	SellerID          int64
	ProductName       string
	SKU               string
	Status            string
	AvailableQuantity int
	ReservedQuantity  int
	UpdatedAt         time.Time
}

type InventoryAdjustment struct {
	VariantID int64
	Delta     int
	Reason    string
}

type SellerOrderItem struct {
	ProductName    string
	SKU            string
	Quantity       int
	UnitPriceCents int64
	LineTotalCents int64
}

type SellerOrder struct {
	ID                  int64
	SellerID            int64
	OrderNumber         string
	OverallStatus       string
	FulfillmentStatus   string
	Currency            string
	SellerSubtotalCents int64
	Carrier             string
	TrackingNumber      string
	LastNote            string
	CreatedAt           time.Time
	Items               []SellerOrderItem
}

// SellerReport is a read-only summary built from the seller-scoped order
// projection. It deliberately contains no settlement or commission values;
// those require the separate finance ledger that is not part of V1 yet.
type SellerReport struct {
	OrderCount       int
	UnitsSold        int
	GrossSalesCents  int64
	Currency         string
	MixedCurrencies  bool
	PendingOrders    int
	ProcessingOrders int
	ShippedOrders    int
	DeliveredOrders  int
}

type ReturnRequest struct {
	ID            int64
	OrderNumber   string
	UserID        int64
	CustomerEmail string
	AmountCents   int64
	Currency      string
	Status        string
	Reason        string
	CreatedAt     time.Time
	UpdatedAt     time.Time
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
	Items         []OrderItem        `json:"items"`
	PaymentStatus string             `json:"payment_status"`
	Fulfillments  []OrderFulfillment `json:"fulfillments"`
}

// OrderFulfillment is the customer-safe delivery projection. It contains
// only fulfilment information that belongs to the customer's order; seller
// permissions and internal operational fields never cross this boundary.
type OrderFulfillment struct {
	SellerName     string    `json:"seller_name"`
	Status         string    `json:"status"`
	Carrier        string    `json:"carrier"`
	TrackingNumber string    `json:"tracking_number"`
	LastNote       string    `json:"last_note"`
	UpdatedAt      time.Time `json:"updated_at"`
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

type FinanceEntry struct {
	OrderID             int64
	OrderNumber         string
	OrderStatus         string
	PaymentStatus       string
	RefundStatus        string
	Provider            string
	ProviderPaymentID   string
	AmountCents         int64
	RefundedAmountCents int64
	Currency            string
	CreatedAt           time.Time
}

type SupportTicket struct {
	ID                  int64
	TicketNumber        string
	OrderNumber         string
	Subject             string
	Message             string
	Status              string
	Priority            string
	CustomerLabel       string
	CustomerEmailMasked string
	AgentNote           string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}
