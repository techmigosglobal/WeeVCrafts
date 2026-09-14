package ports

import (
	"context"
	"errors"

	domaincommerce "github.com/wecratfs/commerce/internal/domain/commerce"
)

var (
	ErrForbidden         = errors.New("forbidden")
	ErrInvalidState      = errors.New("invalid state transition")
	ErrInsufficientStock = errors.New("insufficient stock")
	ErrCartNotFound      = errors.New("cart not found")
	ErrOrderNotFound     = errors.New("order not found")
	ErrReturnNotFound    = errors.New("return request not found")
	ErrReturnState       = errors.New("return request state is invalid")
	ErrSupportNotFound   = errors.New("support ticket not found")
	ErrSupportState      = errors.New("support ticket state is invalid")
	ErrSellerNotFound    = errors.New("seller not found")
	ErrSellerState       = errors.New("seller state is invalid")
)

type CatalogManagementRepository interface {
	EnsureSeller(ctx context.Context, ownerUserID int64, displayName string) (domaincommerce.Seller, error)
	GetSeller(ctx context.Context, userID int64) (domaincommerce.Seller, error)
	CreateDraft(ctx context.Context, ownerUserID int64, input domaincommerce.ProductDraftInput) (domaincommerce.ManagedProduct, error)
	GetSellerProduct(ctx context.Context, ownerUserID, productID int64) (domaincommerce.ManagedProduct, error)
	UpdateDraft(ctx context.Context, ownerUserID, productID int64, input domaincommerce.ProductDraftInput) (domaincommerce.ManagedProduct, error)
	SubmitProduct(ctx context.Context, ownerUserID, productID int64) error
	ApproveProduct(ctx context.Context, actorID, productID int64, approved bool, reason string) error
	ListSellerProducts(ctx context.Context, ownerUserID int64) ([]domaincommerce.ManagedProduct, error)
	ListPendingProducts(ctx context.Context, actorID int64) ([]domaincommerce.ManagedProduct, error)
}

type SellerAdministrationRepository interface {
	ListSellerApplications(ctx context.Context, actorID int64) ([]domaincommerce.SellerAdminEntry, error)
	UpdateSellerStatus(ctx context.Context, actorID, sellerID int64, status, reason string) error
}

type AuditRepository interface {
	ListAuditEntries(ctx context.Context, actorID int64, limit int) ([]domaincommerce.AuditEntry, error)
	ListSellerAuditEntries(ctx context.Context, ownerUserID int64, limit int) ([]domaincommerce.AuditEntry, error)
}

type AdminOperationsRepository interface {
	ListAdminOrders(ctx context.Context, actorID int64, limit int) ([]domaincommerce.AdminOrder, error)
}

type BrandDirectoryRepository interface {
	ListBrandDirectory(ctx context.Context, actorID int64) ([]domaincommerce.AdminBrand, error)
}

type SellerStaffRepository interface {
	ListStaff(ctx context.Context, ownerUserID int64) ([]domaincommerce.SellerStaffMember, error)
	AddStaff(ctx context.Context, ownerUserID int64, email string, permissions []string) (domaincommerce.SellerStaffMember, error)
	RemoveStaff(ctx context.Context, ownerUserID, staffUserID int64) error
}

// SellerPermissionChecker is deliberately separate from RoleChecker because
// a seller staff permission is scoped to one seller, not to the user globally.
type SellerPermissionChecker interface {
	HasSellerPermission(ctx context.Context, userID int64, permission string) (bool, error)
}

type InventoryRepository interface {
	ListSellerInventory(ctx context.Context, userID int64) ([]domaincommerce.InventoryItem, error)
	AdjustSellerInventory(ctx context.Context, userID int64, adjustment domaincommerce.InventoryAdjustment) error
}

type SellerOrderRepository interface {
	ListSellerOrders(ctx context.Context, userID int64) ([]domaincommerce.SellerOrder, error)
	UpdateSellerFulfillment(ctx context.Context, userID, sellerID int64, orderNumber, status, carrier, trackingNumber, note string) error
}

type FinanceRepository interface {
	ListFinanceEntries(ctx context.Context, limit int) ([]domaincommerce.FinanceEntry, error)
}

type SupportRepository interface {
	CreateSupportTicket(ctx context.Context, userID int64, orderNumber, subject, message string) (domaincommerce.SupportTicket, error)
	ListCustomerSupportTickets(ctx context.Context, userID int64) ([]domaincommerce.SupportTicket, error)
	ListSellerSupportTickets(ctx context.Context, userID int64) ([]domaincommerce.SupportTicket, error)
	ListSupportTickets(ctx context.Context, limit int) ([]domaincommerce.SupportTicket, error)
	UpdateSupportTicket(ctx context.Context, actorID, ticketID int64, status, note string) error
}

type ReturnRepository interface {
	CreateReturnRequest(ctx context.Context, userID int64, orderNumber, reason string) (domaincommerce.ReturnRequest, error)
	GetReturnRequest(ctx context.Context, userID int64, orderNumber string) (domaincommerce.ReturnRequest, error)
	ListReturnRequests(ctx context.Context, actorID int64) ([]domaincommerce.ReturnRequest, error)
	UpdateReturnRequest(ctx context.Context, actorID, requestID int64, status, reason string) error
}

type CartRepository interface {
	GetOrCreateCart(ctx context.Context, userID int64, guestTokenHash string) (domaincommerce.Cart, error)
	AddCartItem(ctx context.Context, cartID, variantID int64, quantity int) error
	UpdateCartItem(ctx context.Context, cartID, variantID int64, quantity int) error
	RemoveCartItem(ctx context.Context, cartID, variantID int64) error
	MoveCartItemToWishlist(ctx context.Context, cartID, userID, variantID, productID int64) error
	MergeGuestCart(ctx context.Context, userID int64, guestTokenHash string) error
	GetCart(ctx context.Context, cartID int64) (domaincommerce.Cart, error)
	AddWishlist(ctx context.Context, userID, productID int64) error
	RemoveWishlist(ctx context.Context, userID, productID int64) error
	ListWishlist(ctx context.Context, userID int64) ([]domaincommerce.WishlistItem, error)
}

type OrderRepository interface {
	CreateOrder(ctx context.Context, userID int64, cartID int64, idempotencyKey string, address domaincommerce.AddressInput) (domaincommerce.Order, error)
	ListOrders(ctx context.Context, userID int64, limit int) ([]domaincommerce.Order, error)
	GetOrder(ctx context.Context, userID int64, orderNumber string) (domaincommerce.OrderDetail, error)
	CancelOrder(ctx context.Context, userID int64, orderNumber string) error
}

type ReservationReaper interface {
	ReleaseExpiredReservations(ctx context.Context, limit int) (released int, err error)
}
