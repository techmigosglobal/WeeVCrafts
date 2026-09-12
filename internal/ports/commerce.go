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
)

type CatalogManagementRepository interface {
	EnsureSeller(ctx context.Context, ownerUserID int64, displayName string) (domaincommerce.Seller, error)
	CreateDraft(ctx context.Context, ownerUserID int64, input domaincommerce.ProductDraftInput) (domaincommerce.ManagedProduct, error)
	SubmitProduct(ctx context.Context, ownerUserID, productID int64) error
	ApproveProduct(ctx context.Context, actorID, productID int64, approved bool, reason string) error
	ListSellerProducts(ctx context.Context, ownerUserID int64) ([]domaincommerce.ManagedProduct, error)
	ListPendingProducts(ctx context.Context, actorID int64) ([]domaincommerce.ManagedProduct, error)
}

type CartRepository interface {
	GetOrCreateCart(ctx context.Context, userID int64, guestTokenHash string) (domaincommerce.Cart, error)
	AddCartItem(ctx context.Context, cartID, variantID int64, quantity int) error
	UpdateCartItem(ctx context.Context, cartID, variantID int64, quantity int) error
	RemoveCartItem(ctx context.Context, cartID, variantID int64) error
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
