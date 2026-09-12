package commerce

import (
	"context"
	"errors"

	domaincommerce "github.com/wecratfs/commerce/internal/domain/commerce"
	"github.com/wecratfs/commerce/internal/ports"
)

var ErrInvalidQuantity = errors.New("quantity must be between 1 and 99")

type CartService struct {
	repository ports.CartRepository
}

func NewCartService(repository ports.CartRepository) *CartService {
	return &CartService{repository: repository}
}

func (s *CartService) GetOrCreate(ctx context.Context, userID int64, guestTokenHash string) (domaincommerce.Cart, error) {
	if userID <= 0 && guestTokenHash == "" {
		return domaincommerce.Cart{}, ports.ErrCartNotFound
	}
	return s.repository.GetOrCreateCart(ctx, userID, guestTokenHash)
}

func (s *CartService) Add(ctx context.Context, userID int64, guestTokenHash string, variantID, quantity int64) (domaincommerce.Cart, error) {
	if quantity < 1 || quantity > 99 {
		return domaincommerce.Cart{}, ErrInvalidQuantity
	}
	cart, err := s.GetOrCreate(ctx, userID, guestTokenHash)
	if err != nil {
		return domaincommerce.Cart{}, err
	}
	if err := s.repository.AddCartItem(ctx, cart.ID, variantID, int(quantity)); err != nil {
		return domaincommerce.Cart{}, err
	}
	return s.repository.GetCart(ctx, cart.ID)
}

func (s *CartService) Update(ctx context.Context, userID int64, guestTokenHash string, variantID, quantity int64) (domaincommerce.Cart, error) {
	if quantity < 1 || quantity > 99 {
		return domaincommerce.Cart{}, ErrInvalidQuantity
	}
	cart, err := s.GetOrCreate(ctx, userID, guestTokenHash)
	if err != nil {
		return domaincommerce.Cart{}, err
	}
	if err := s.repository.UpdateCartItem(ctx, cart.ID, variantID, int(quantity)); err != nil {
		return domaincommerce.Cart{}, err
	}
	return s.repository.GetCart(ctx, cart.ID)
}

func (s *CartService) Remove(ctx context.Context, userID int64, guestTokenHash string, variantID int64) (domaincommerce.Cart, error) {
	cart, err := s.GetOrCreate(ctx, userID, guestTokenHash)
	if err != nil {
		return domaincommerce.Cart{}, err
	}
	if err := s.repository.RemoveCartItem(ctx, cart.ID, variantID); err != nil {
		return domaincommerce.Cart{}, err
	}
	return s.repository.GetCart(ctx, cart.ID)
}

func (s *CartService) MergeGuest(ctx context.Context, userID int64, guestTokenHash string) error {
	if userID <= 0 || guestTokenHash == "" {
		return nil
	}
	return s.repository.MergeGuestCart(ctx, userID, guestTokenHash)
}

func (s *CartService) AddWishlist(ctx context.Context, userID, productID int64) error {
	if userID <= 0 || productID <= 0 {
		return ports.ErrForbidden
	}
	return s.repository.AddWishlist(ctx, userID, productID)
}

func (s *CartService) RemoveWishlist(ctx context.Context, userID, productID int64) error {
	if userID <= 0 || productID <= 0 {
		return ports.ErrForbidden
	}
	return s.repository.RemoveWishlist(ctx, userID, productID)
}

func (s *CartService) ListWishlist(ctx context.Context, userID int64) ([]domaincommerce.WishlistItem, error) {
	if userID <= 0 {
		return nil, ports.ErrForbidden
	}
	return s.repository.ListWishlist(ctx, userID)
}
