package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	domaincommerce "github.com/wecratfs/commerce/internal/domain/commerce"
	"github.com/wecratfs/commerce/internal/ports"
)

type CommerceRepository struct {
	pool *pgxpool.Pool
}

func NewCommerceRepository(pool *pgxpool.Pool) *CommerceRepository {
	return &CommerceRepository{pool: pool}
}

func (r *CommerceRepository) EnsureSeller(ctx context.Context, ownerUserID int64, displayName string) (domaincommerce.Seller, error) {
	var seller domaincommerce.Seller
	err := WithinTransaction(ctx, r.pool, func(transactionContext context.Context, tx pgx.Tx) error {
		row := tx.QueryRow(transactionContext, `
			INSERT INTO sellers (owner_user_id, display_name, status)
			VALUES ($1, $2, 'pending')
			ON CONFLICT (owner_user_id) DO UPDATE SET display_name = EXCLUDED.display_name, updated_at = NOW()
			RETURNING id, owner_user_id, display_name, status`, ownerUserID, displayName)
		if err := row.Scan(&seller.ID, &seller.OwnerUserID, &seller.DisplayName, &seller.Status); err != nil {
			return err
		}
		_, err := tx.Exec(transactionContext, `INSERT INTO user_roles (user_id, role_slug) VALUES ($1, 'seller_owner') ON CONFLICT DO NOTHING`, ownerUserID)
		return err
	})
	return seller, err
}

func (r *CommerceRepository) GetSeller(ctx context.Context, userID int64) (domaincommerce.Seller, error) {
	var seller domaincommerce.Seller
	err := r.pool.QueryRow(ctx, `
		SELECT s.id, s.owner_user_id, s.display_name, s.status
		FROM sellers s
		WHERE s.owner_user_id = $1
		   OR EXISTS (SELECT 1 FROM seller_users su WHERE su.seller_id = s.id AND su.user_id = $1 AND su.status = 'active')
		ORDER BY CASE WHEN s.owner_user_id = $1 THEN 0 ELSE 1 END, s.id
		LIMIT 1`, userID).Scan(&seller.ID, &seller.OwnerUserID, &seller.DisplayName, &seller.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return domaincommerce.Seller{}, ports.ErrSellerNotFound
	}
	return seller, err
}

func (r *CommerceRepository) CreateDraft(ctx context.Context, ownerUserID int64, input domaincommerce.ProductDraftInput) (domaincommerce.ManagedProduct, error) {
	var product domaincommerce.ManagedProduct
	err := WithinTransaction(ctx, r.pool, func(transactionContext context.Context, tx pgx.Tx) error {
		var sellerID int64
		if err := tx.QueryRow(transactionContext, `SELECT id FROM sellers WHERE status = 'active' AND (owner_user_id = $1 OR EXISTS (SELECT 1 FROM seller_users WHERE seller_id = sellers.id AND user_id = $1 AND status = 'active'))`, ownerUserID).Scan(&sellerID); err != nil {
			return ports.ErrForbidden
		}
		if _, err := tx.Exec(transactionContext, `INSERT INTO brands (slug, name) VALUES ($1, $2) ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name`, input.BrandSlug, input.BrandName); err != nil {
			return err
		}
		if _, err := tx.Exec(transactionContext, `INSERT INTO categories (slug, name) VALUES ($1, $2) ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name`, input.CategorySlug, input.CategoryName); err != nil {
			return err
		}
		var productID int64
		if err := tx.QueryRow(transactionContext, `
			INSERT INTO products (slug, brand_slug, category_slug, name, description, price_cents, compare_at_cents, delivery_label, seller_id, status)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'draft')
			RETURNING id`, input.Slug, input.BrandSlug, input.CategorySlug, input.Name, input.Description, input.PriceCents, input.CompareAtCents, input.DeliveryLabel, sellerID).Scan(&productID); err != nil {
			return mapCommerceError(err)
		}
		if err := tx.QueryRow(transactionContext, `
			INSERT INTO product_variants (product_id, sku, price_cents, compare_at_cents)
			VALUES ($1, $2, $3, $4)
			RETURNING id`, productID, input.SKU, input.PriceCents, input.CompareAtCents).Scan(&product.VariantID); err != nil {
			return mapCommerceError(err)
		}
		if _, err := tx.Exec(transactionContext, `INSERT INTO inventory_stock (variant_id, available_quantity) VALUES ($1, $2)`, product.VariantID, input.InitialStock); err != nil {
			return err
		}
		product = domaincommerce.ManagedProduct{
			ID:             productID,
			VariantID:      product.VariantID,
			SellerID:       sellerID,
			Slug:           input.Slug,
			BrandSlug:      input.BrandSlug,
			BrandName:      input.BrandName,
			Name:           input.Name,
			Description:    input.Description,
			SKU:            input.SKU,
			PriceCents:     input.PriceCents,
			CompareAtCents: 0,
			HasCompareAt:   input.CompareAtCents != nil,
			Stock:          input.InitialStock,
			Status:         "draft",
			CategorySlug:   input.CategorySlug,
			CategoryName:   input.CategoryName,
			DeliveryLabel:  input.DeliveryLabel,
		}
		if input.CompareAtCents != nil {
			product.CompareAtCents = *input.CompareAtCents
		}
		_, err := tx.Exec(transactionContext, `INSERT INTO audit_logs (actor_id, action, resource_type, resource_id, request_id, metadata) VALUES ($1, 'catalog.product_created', 'product', $2, $3, $4)`, fmt.Sprint(ownerUserID), fmt.Sprint(productID), ports.RequestID(transactionContext), []byte(`{"source":"seller"}`))
		return err
	})
	return product, err
}

func (r *CommerceRepository) GetSellerProduct(ctx context.Context, ownerUserID, productID int64) (domaincommerce.ManagedProduct, error) {
	product, err := r.getSellerProduct(ctx, r.pool, ownerUserID, productID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domaincommerce.ManagedProduct{}, ports.ErrForbidden
	}
	return product, err
}

func (r *CommerceRepository) UpdateDraft(ctx context.Context, ownerUserID, productID int64, input domaincommerce.ProductDraftInput) (domaincommerce.ManagedProduct, error) {
	var product domaincommerce.ManagedProduct
	err := WithinTransaction(ctx, r.pool, func(transactionContext context.Context, tx pgx.Tx) error {
		current, err := r.getSellerProduct(transactionContext, tx, ownerUserID, productID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ports.ErrForbidden
		}
		if err != nil {
			return err
		}
		if current.Status != "draft" && current.Status != "rejected" {
			return ports.ErrInvalidState
		}
		if _, err := tx.Exec(transactionContext, `INSERT INTO brands (slug, name) VALUES ($1, $2) ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name`, input.BrandSlug, input.BrandName); err != nil {
			return err
		}
		if _, err := tx.Exec(transactionContext, `INSERT INTO categories (slug, name) VALUES ($1, $2) ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name`, input.CategorySlug, input.CategoryName); err != nil {
			return err
		}
		if _, err := tx.Exec(transactionContext, `UPDATE products SET slug = $1, brand_slug = $2, category_slug = $3, name = $4, description = $5, price_cents = $6, compare_at_cents = $7, delivery_label = $8, status = CASE WHEN status = 'rejected' THEN 'draft' ELSE status END, updated_at = NOW() WHERE id = $9 AND seller_id = $10 AND status IN ('draft', 'rejected')`, input.Slug, input.BrandSlug, input.CategorySlug, input.Name, input.Description, input.PriceCents, input.CompareAtCents, input.DeliveryLabel, productID, current.SellerID); err != nil {
			return mapCommerceError(err)
		}
		if _, err := tx.Exec(transactionContext, `UPDATE product_variants SET sku = $1, price_cents = $2, compare_at_cents = $3, updated_at = NOW() WHERE id = $4 AND product_id = $5 AND status = 'active'`, input.SKU, input.PriceCents, input.CompareAtCents, current.VariantID, productID); err != nil {
			return mapCommerceError(err)
		}
		metadata, _ := json.Marshal(map[string]string{"source": "seller", "previous_status": current.Status})
		if _, err := tx.Exec(transactionContext, `INSERT INTO audit_logs (actor_id, action, resource_type, resource_id, request_id, metadata) VALUES ($1, 'catalog.product_updated', 'product', $2, $3, $4)`, fmt.Sprint(ownerUserID), fmt.Sprint(productID), ports.RequestID(transactionContext), metadata); err != nil {
			return err
		}
		product = current
		product.Slug = input.Slug
		product.BrandSlug = input.BrandSlug
		product.BrandName = input.BrandName
		product.Name = input.Name
		product.Description = input.Description
		product.SKU = input.SKU
		product.PriceCents = input.PriceCents
		product.CompareAtCents = 0
		product.HasCompareAt = input.CompareAtCents != nil
		if input.CompareAtCents != nil {
			product.CompareAtCents = *input.CompareAtCents
		}
		// Stock is an operational record. Product writers must use the
		// permission-scoped inventory workflow rather than changing it while
		// editing catalogue copy or price.
		product.Stock = current.Stock
		product.Status = "draft"
		product.CategorySlug = input.CategorySlug
		product.CategoryName = input.CategoryName
		product.DeliveryLabel = input.DeliveryLabel
		return nil
	})
	return product, err
}

func (r *CommerceRepository) SubmitProduct(ctx context.Context, ownerUserID, productID int64) error {
	result, err := r.pool.Exec(ctx, `UPDATE products SET status = 'pending_review', updated_at = NOW() WHERE id = $1 AND seller_id IN (SELECT s.id FROM sellers s WHERE s.status = 'active' AND (s.owner_user_id = $2 OR EXISTS (SELECT 1 FROM seller_users su WHERE su.seller_id = s.id AND su.user_id = $2 AND su.status = 'active'))) AND status IN ('draft', 'rejected')`, productID, ownerUserID)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return ports.ErrForbidden
	}
	return nil
}

func (r *CommerceRepository) ApproveProduct(ctx context.Context, actorID, productID int64, approved bool, reason string) error {
	status := "rejected"
	if approved {
		status = "approved"
	}
	return WithinTransaction(ctx, r.pool, func(transactionContext context.Context, tx pgx.Tx) error {
		var allowed bool
		if err := tx.QueryRow(transactionContext, `SELECT EXISTS (SELECT 1 FROM user_roles WHERE user_id = $1 AND role_slug IN ('marketplace_admin', 'super_admin'))`, actorID).Scan(&allowed); err != nil {
			return err
		}
		if !allowed {
			return ports.ErrForbidden
		}
		result, err := tx.Exec(transactionContext, `UPDATE products SET status = $1, updated_at = NOW() WHERE id = $2 AND status = 'pending_review'`, status, productID)
		if err != nil {
			return err
		}
		if result.RowsAffected() != 1 {
			return ports.ErrInvalidState
		}
		metadata, _ := json.Marshal(map[string]string{"reason": reason, "approved": fmt.Sprint(approved)})
		_, err = tx.Exec(transactionContext, `INSERT INTO audit_logs (actor_id, action, resource_type, resource_id, request_id, metadata) VALUES ($1, $2, 'product', $3, $4, $5)`, fmt.Sprint(actorID), "catalog.product_reviewed", fmt.Sprint(productID), ports.RequestID(transactionContext), metadata)
		if err != nil {
			return err
		}
		payload, _ := json.Marshal(map[string]any{"product_id": productID, "status": status})
		_, err = tx.Exec(transactionContext, `INSERT INTO outbox_events (event_type, aggregate_type, aggregate_id, payload) VALUES ('ProductIndexRequested', 'Product', $1, $2)`, fmt.Sprint(productID), payload)
		return err
	})
}

func (r *CommerceRepository) ListSellerProducts(ctx context.Context, ownerUserID int64) ([]domaincommerce.ManagedProduct, error) {
	return r.listManagedProducts(ctx, `WHERE s.owner_user_id = $1 OR EXISTS (SELECT 1 FROM seller_users su WHERE su.seller_id = s.id AND su.user_id = $1 AND su.status = 'active')`, ownerUserID)
}

func (r *CommerceRepository) ListPendingProducts(ctx context.Context, actorID int64) ([]domaincommerce.ManagedProduct, error) {
	var allowed bool
	if err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM user_roles WHERE user_id = $1 AND role_slug IN ('marketplace_admin', 'super_admin'))`, actorID).Scan(&allowed); err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ports.ErrForbidden
	}
	return r.listManagedProducts(ctx, `WHERE p.status = 'pending_review'`)
}

func (r *CommerceRepository) listManagedProducts(ctx context.Context, filter string, args ...any) ([]domaincommerce.ManagedProduct, error) {
	query := `SELECT p.id, s.id, s.display_name, p.slug, p.brand_slug, b.name, p.name, p.description, pv.sku, pv.price_cents, COALESCE(p.compare_at_cents, 0), p.compare_at_cents IS NOT NULL, COALESCE(i.available_quantity, 0), p.status, p.category_slug, c.name, p.delivery_label, pv.id FROM products p JOIN sellers s ON s.id = p.seller_id JOIN brands b ON b.slug = p.brand_slug JOIN categories c ON c.slug = p.category_slug JOIN product_variants pv ON pv.product_id = p.id AND pv.status = 'active' LEFT JOIN inventory_stock i ON i.variant_id = pv.id ` + filter + ` ORDER BY p.updated_at DESC, p.id DESC LIMIT 100`
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	products := make([]domaincommerce.ManagedProduct, 0)
	for rows.Next() {
		var product domaincommerce.ManagedProduct
		if err := rows.Scan(&product.ID, &product.SellerID, &product.SellerName, &product.Slug, &product.BrandSlug, &product.BrandName, &product.Name, &product.Description, &product.SKU, &product.PriceCents, &product.CompareAtCents, &product.HasCompareAt, &product.Stock, &product.Status, &product.CategorySlug, &product.CategoryName, &product.DeliveryLabel, &product.VariantID); err != nil {
			return nil, err
		}
		products = append(products, product)
	}
	return products, rows.Err()
}

type catalogProductQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func (r *CommerceRepository) getSellerProduct(ctx context.Context, querier catalogProductQuerier, ownerUserID, productID int64) (domaincommerce.ManagedProduct, error) {
	var product domaincommerce.ManagedProduct
	err := querier.QueryRow(ctx, `SELECT p.id, s.id, s.display_name, p.slug, p.brand_slug, b.name, p.name, p.description, pv.sku, pv.price_cents, COALESCE(p.compare_at_cents, 0), p.compare_at_cents IS NOT NULL, COALESCE(i.available_quantity, 0), p.status, p.category_slug, c.name, p.delivery_label, pv.id FROM products p JOIN sellers s ON s.id = p.seller_id AND s.status = 'active' JOIN brands b ON b.slug = p.brand_slug JOIN categories c ON c.slug = p.category_slug JOIN product_variants pv ON pv.product_id = p.id AND pv.status = 'active' LEFT JOIN inventory_stock i ON i.variant_id = pv.id WHERE (s.owner_user_id = $1 OR EXISTS (SELECT 1 FROM seller_users su WHERE su.seller_id = s.id AND su.user_id = $1 AND su.status = 'active')) AND p.id = $2`, ownerUserID, productID).Scan(&product.ID, &product.SellerID, &product.SellerName, &product.Slug, &product.BrandSlug, &product.BrandName, &product.Name, &product.Description, &product.SKU, &product.PriceCents, &product.CompareAtCents, &product.HasCompareAt, &product.Stock, &product.Status, &product.CategorySlug, &product.CategoryName, &product.DeliveryLabel, &product.VariantID)
	return product, err
}

func (r *CommerceRepository) GetOrCreateCart(ctx context.Context, userID int64, guestTokenHash string) (domaincommerce.Cart, error) {
	var cartID int64
	query := `SELECT id FROM carts WHERE status = 'active' AND expires_at > NOW() AND (($1 > 0 AND user_id = $1) OR ($2 <> '' AND guest_token_hash = $2)) LIMIT 1`
	err := r.pool.QueryRow(ctx, query, userID, guestTokenHash).Scan(&cartID)
	if errors.Is(err, pgx.ErrNoRows) {
		err = r.pool.QueryRow(ctx, `INSERT INTO carts (user_id, guest_token_hash, expires_at) VALUES (NULLIF($1, 0), NULLIF($2, ''), NOW() + INTERVAL '30 days') RETURNING id`, userID, guestTokenHash).Scan(&cartID)
	}
	if err != nil {
		return domaincommerce.Cart{}, mapCommerceError(err)
	}
	return r.GetCart(ctx, cartID)
}

func (r *CommerceRepository) AddCartItem(ctx context.Context, cartID, variantID int64, quantity int) error {
	if err := r.assertAvailableVariant(ctx, variantID); err != nil {
		return err
	}
	result, err := r.pool.Exec(ctx, `INSERT INTO cart_items (cart_id, variant_id, quantity) VALUES ($1, $2, $3) ON CONFLICT (cart_id, variant_id) DO UPDATE SET quantity = LEAST(99, cart_items.quantity + EXCLUDED.quantity), updated_at = NOW() WHERE cart_items.cart_id IN (SELECT id FROM carts WHERE status = 'active' AND expires_at > NOW())`, cartID, variantID, quantity)
	if err != nil {
		return mapCommerceError(err)
	}
	if result.RowsAffected() != 1 {
		return ports.ErrCartNotFound
	}
	return nil
}

func (r *CommerceRepository) UpdateCartItem(ctx context.Context, cartID, variantID int64, quantity int) error {
	result, err := r.pool.Exec(ctx, `UPDATE cart_items SET quantity = $3, updated_at = NOW() WHERE cart_id = $1 AND variant_id = $2 AND EXISTS (SELECT 1 FROM carts WHERE id = $1 AND status = 'active' AND expires_at > NOW())`, cartID, variantID, quantity)
	if err != nil {
		return mapCommerceError(err)
	}
	if result.RowsAffected() != 1 {
		return ports.ErrCartNotFound
	}
	return nil
}

func (r *CommerceRepository) RemoveCartItem(ctx context.Context, cartID, variantID int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM cart_items WHERE cart_id = $1 AND variant_id = $2`, cartID, variantID)
	return err
}

func (r *CommerceRepository) MoveCartItemToWishlist(ctx context.Context, cartID, userID, variantID, productID int64) error {
	return WithinTransaction(ctx, r.pool, func(transactionContext context.Context, tx pgx.Tx) error {
		var cartProductID int64
		err := tx.QueryRow(transactionContext, `
			SELECT p.id
			FROM carts c
			JOIN cart_items ci ON ci.cart_id = c.id AND ci.variant_id = $3
			JOIN product_variants pv ON pv.id = ci.variant_id AND pv.status = 'active'
			JOIN products p ON p.id = pv.product_id AND p.status = 'approved'
			WHERE c.id = $1 AND c.user_id = $2 AND c.status = 'active' AND c.expires_at > NOW()
			FOR UPDATE OF c, ci`, cartID, userID, variantID).Scan(&cartProductID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ports.ErrCartNotFound
			}
			return err
		}
		if cartProductID != productID {
			return ports.ErrCartNotFound
		}
		if _, err := tx.Exec(transactionContext, `INSERT INTO wishlist_items (user_id, product_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, userID, productID); err != nil {
			return err
		}
		result, err := tx.Exec(transactionContext, `DELETE FROM cart_items WHERE cart_id = $1 AND variant_id = $2`, cartID, variantID)
		if err != nil {
			return err
		}
		if result.RowsAffected() != 1 {
			return ports.ErrCartNotFound
		}
		return nil
	})
}

func (r *CommerceRepository) MergeGuestCart(ctx context.Context, userID int64, guestTokenHash string) error {
	return WithinTransaction(ctx, r.pool, func(transactionContext context.Context, tx pgx.Tx) error {
		var guestID, userCartID int64
		if err := tx.QueryRow(transactionContext, `SELECT id FROM carts WHERE guest_token_hash = $1 AND status = 'active' AND expires_at > NOW() FOR UPDATE`, guestTokenHash).Scan(&guestID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}
			return err
		}
		if err := tx.QueryRow(transactionContext, `SELECT id FROM carts WHERE user_id = $1 AND status = 'active' AND expires_at > NOW() FOR UPDATE`, userID).Scan(&userCartID); errors.Is(err, pgx.ErrNoRows) {
			if err := tx.QueryRow(transactionContext, `INSERT INTO carts (user_id, expires_at) VALUES ($1, NOW() + INTERVAL '30 days') RETURNING id`, userID).Scan(&userCartID); err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		if _, err := tx.Exec(transactionContext, `INSERT INTO cart_items (cart_id, variant_id, quantity)
			SELECT $1, ci.variant_id, ci.quantity
			FROM cart_items ci
			JOIN product_variants pv ON pv.id = ci.variant_id AND pv.status = 'active'
			JOIN products p ON p.id = pv.product_id AND p.status = 'approved'
			JOIN inventory_stock i ON i.variant_id = pv.id
			WHERE ci.cart_id = $2
			ON CONFLICT (cart_id, variant_id) DO UPDATE SET quantity = LEAST(99, cart_items.quantity + EXCLUDED.quantity), updated_at = NOW()`, userCartID, guestID); err != nil {
			return err
		}
		_, err := tx.Exec(transactionContext, `UPDATE carts SET status = 'expired', updated_at = NOW() WHERE id = $1`, guestID)
		return err
	})
}

func (r *CommerceRepository) GetCart(ctx context.Context, cartID int64) (domaincommerce.Cart, error) {
	var cart domaincommerce.Cart
	if err := r.pool.QueryRow(ctx, `SELECT id, expires_at FROM carts WHERE id = $1 AND status = 'active' AND expires_at > NOW()`, cartID).Scan(&cart.ID, &cart.ExpiresAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domaincommerce.Cart{}, ports.ErrCartNotFound
		}
		return domaincommerce.Cart{}, err
	}
	rows, err := r.pool.Query(ctx, `SELECT ci.variant_id, p.id, p.slug, p.name, pv.sku, ci.quantity, pv.price_cents, i.available_quantity FROM cart_items ci JOIN product_variants pv ON pv.id = ci.variant_id AND pv.status = 'active' JOIN products p ON p.id = pv.product_id AND p.status = 'approved' JOIN inventory_stock i ON i.variant_id = pv.id WHERE ci.cart_id = $1 ORDER BY ci.created_at, ci.variant_id`, cartID)
	if err != nil {
		return domaincommerce.Cart{}, err
	}
	defer rows.Close()
	cart.Items = make([]domaincommerce.CartItem, 0)
	for rows.Next() {
		var item domaincommerce.CartItem
		if err := rows.Scan(&item.VariantID, &item.ProductID, &item.ProductSlug, &item.ProductName, &item.SKU, &item.Quantity, &item.UnitPriceCents, &item.Available); err != nil {
			return domaincommerce.Cart{}, err
		}
		item.LineTotalCents = item.UnitPriceCents * int64(item.Quantity)
		cart.SubtotalCents += item.LineTotalCents
		cart.ItemCount += item.Quantity
		cart.Items = append(cart.Items, item)
	}
	return cart, rows.Err()
}

func (r *CommerceRepository) AddWishlist(ctx context.Context, userID, productID int64) error {
	var exists bool
	if err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM products WHERE id = $1 AND status = 'approved')`, productID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ports.ErrNotFound
	}
	_, err := r.pool.Exec(ctx, `INSERT INTO wishlist_items (user_id, product_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, userID, productID)
	if err != nil {
		return err
	}
	return nil
}

func (r *CommerceRepository) RemoveWishlist(ctx context.Context, userID, productID int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM wishlist_items WHERE user_id = $1 AND product_id = $2`, userID, productID)
	return err
}

func (r *CommerceRepository) ListWishlist(ctx context.Context, userID int64) ([]domaincommerce.WishlistItem, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT wi.product_id, p.slug, p.name, b.name, p.price_cents,
		       COALESCE(i.available_quantity, 0)
		FROM wishlist_items wi
		JOIN products p ON p.id = wi.product_id AND p.status = 'approved'
		JOIN brands b ON b.slug = p.brand_slug
		LEFT JOIN LATERAL (
			SELECT pv.id
			FROM product_variants pv
			WHERE pv.product_id = p.id AND pv.status = 'active'
			ORDER BY pv.id
			LIMIT 1
		) pv ON TRUE
		LEFT JOIN inventory_stock i ON i.variant_id = pv.id
		WHERE wi.user_id = $1
		ORDER BY wi.created_at DESC, wi.product_id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domaincommerce.WishlistItem, 0)
	for rows.Next() {
		var item domaincommerce.WishlistItem
		if err := rows.Scan(&item.ProductID, &item.Slug, &item.Name, &item.Brand, &item.PriceCents, &item.Available); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *CommerceRepository) CreateOrder(ctx context.Context, userID, cartID int64, idempotencyKey string, address domaincommerce.AddressInput) (domaincommerce.Order, error) {
	return r.createOrder(ctx, userID, cartID, idempotencyKey, address, false)
}

func (r *CommerceRepository) CreateManualOrder(ctx context.Context, userID, cartID int64, idempotencyKey string, address domaincommerce.AddressInput) (domaincommerce.Order, error) {
	return r.createOrder(ctx, userID, cartID, idempotencyKey, address, true)
}

func (r *CommerceRepository) createOrder(ctx context.Context, userID, cartID int64, idempotencyKey string, address domaincommerce.AddressInput, manualPayment bool) (domaincommerce.Order, error) {
	var order domaincommerce.Order
	addressJSON, err := json.Marshal(address)
	if err != nil {
		return order, err
	}
	requestHash := checkoutRequestHash(cartID, addressJSON)
	status, paymentMethod, historyReason := "pending_payment", "gateway", "checkout created"
	if manualPayment {
		status, paymentMethod, historyReason = "processing", "cash_on_delivery", "cash-on-delivery order placed; payment due on delivery"
		requestHash = checkoutRequestHashMode(cartID, addressJSON, paymentMethod)
	}
	err = WithinTransaction(ctx, r.pool, func(transactionContext context.Context, tx pgx.Tx) error {
		var existingHash *string
		if err := tx.QueryRow(transactionContext, `SELECT id, order_number, status, payment_method, currency, subtotal_cents, shipping_cents, total_cents, created_at, idempotency_request_hash FROM orders WHERE idempotency_scope = $1 AND idempotency_key = $2`, fmt.Sprintf("checkout:%d", userID), idempotencyKey).Scan(&order.ID, &order.OrderNumber, &order.Status, &order.PaymentMethod, &order.Currency, &order.SubtotalCents, &order.ShippingCents, &order.TotalCents, &order.CreatedAt, &existingHash); err == nil {
			if existingHash != nil && *existingHash != requestHash {
				return ports.ErrIdempotencyConflict
			}
			return nil
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		var cartOwner int64
		if err := tx.QueryRow(transactionContext, `SELECT user_id FROM carts WHERE id = $1 AND status = 'active' AND expires_at > NOW() FOR UPDATE`, cartID).Scan(&cartOwner); err != nil || cartOwner != userID {
			var existingHash *string
			if existingErr := tx.QueryRow(transactionContext, `SELECT id, order_number, status, payment_method, currency, subtotal_cents, shipping_cents, total_cents, created_at, idempotency_request_hash FROM orders WHERE idempotency_scope = $1 AND idempotency_key = $2`, fmt.Sprintf("checkout:%d", userID), idempotencyKey).Scan(&order.ID, &order.OrderNumber, &order.Status, &order.PaymentMethod, &order.Currency, &order.SubtotalCents, &order.ShippingCents, &order.TotalCents, &order.CreatedAt, &existingHash); existingErr == nil {
				if existingHash != nil && *existingHash != requestHash {
					return ports.ErrIdempotencyConflict
				}
				return nil
			}
			return ports.ErrCartNotFound
		}
		rows, err := tx.Query(transactionContext, `SELECT ci.variant_id, p.name, pv.sku, pv.price_cents, ci.quantity, i.available_quantity FROM cart_items ci JOIN product_variants pv ON pv.id = ci.variant_id JOIN products p ON p.id = pv.product_id AND p.status = 'approved' JOIN inventory_stock i ON i.variant_id = pv.id WHERE ci.cart_id = $1 FOR UPDATE OF i`, cartID)
		if err != nil {
			return err
		}
		defer rows.Close()
		type line struct {
			variantID           int64
			name, sku           string
			price               int64
			quantity, available int
		}
		lines := make([]line, 0)
		var subtotal int64
		for rows.Next() {
			var item line
			if err := rows.Scan(&item.variantID, &item.name, &item.sku, &item.price, &item.quantity, &item.available); err != nil {
				return err
			}
			if item.quantity > item.available {
				return ports.ErrInsufficientStock
			}
			subtotal += item.price * int64(item.quantity)
			lines = append(lines, item)
		}
		if err := rows.Err(); err != nil {
			return err
		}
		if len(lines) == 0 {
			return ports.ErrCartNotFound
		}
		orderNumber := newOrderNumber(userID)
		shipping := int64(0)
		if subtotal < 150000 {
			shipping = 9900
		}
		total := subtotal + shipping
		err = tx.QueryRow(transactionContext, `INSERT INTO orders (order_number, user_id, status, payment_method, subtotal_cents, shipping_cents, total_cents, address_snapshot, idempotency_scope, idempotency_key, idempotency_request_hash) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) ON CONFLICT (idempotency_scope, idempotency_key) DO NOTHING RETURNING id, order_number, status, payment_method, currency, subtotal_cents, shipping_cents, total_cents, created_at`, orderNumber, userID, status, paymentMethod, subtotal, shipping, total, addressJSON, fmt.Sprintf("checkout:%d", userID), idempotencyKey, requestHash).Scan(&order.ID, &order.OrderNumber, &order.Status, &order.PaymentMethod, &order.Currency, &order.SubtotalCents, &order.ShippingCents, &order.TotalCents, &order.CreatedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			var existingHash *string
			if selectErr := tx.QueryRow(transactionContext, `SELECT id, order_number, status, payment_method, currency, subtotal_cents, shipping_cents, total_cents, created_at, idempotency_request_hash FROM orders WHERE idempotency_scope = $1 AND idempotency_key = $2`, fmt.Sprintf("checkout:%d", userID), idempotencyKey).Scan(&order.ID, &order.OrderNumber, &order.Status, &order.PaymentMethod, &order.Currency, &order.SubtotalCents, &order.ShippingCents, &order.TotalCents, &order.CreatedAt, &existingHash); selectErr != nil {
				return selectErr
			} else if existingHash != nil && *existingHash != requestHash {
				return ports.ErrIdempotencyConflict
			}
			return nil
		}
		if err != nil {
			return mapCommerceError(err)
		}
		var reservationID int64
		if err := tx.QueryRow(transactionContext, `INSERT INTO inventory_reservations (order_id, expires_at) VALUES ($1, NOW() + INTERVAL '30 minutes') RETURNING id`, order.ID).Scan(&reservationID); err != nil {
			return err
		}
		for _, item := range lines {
			if _, err := tx.Exec(transactionContext, `INSERT INTO order_items (order_id, variant_id, product_name, sku, unit_price_cents, quantity, line_total_cents) VALUES ($1, $2, $3, $4, $5, $6, $7)`, order.ID, item.variantID, item.name, item.sku, item.price, item.quantity, item.price*int64(item.quantity)); err != nil {
				return err
			}
			if _, err := tx.Exec(transactionContext, `INSERT INTO inventory_reservation_items (reservation_id, variant_id, quantity) VALUES ($1, $2, $3)`, reservationID, item.variantID, item.quantity); err != nil {
				return err
			}
			if _, err := tx.Exec(transactionContext, `UPDATE inventory_stock SET available_quantity = available_quantity - $2, reserved_quantity = reserved_quantity + $2, updated_at = NOW() WHERE variant_id = $1`, item.variantID, item.quantity); err != nil {
				return err
			}
		}
		if manualPayment {
			if _, err := tx.Exec(transactionContext, `UPDATE inventory_stock i SET reserved_quantity = reserved_quantity - ri.quantity, updated_at = NOW() FROM inventory_reservation_items ri WHERE ri.reservation_id = $1 AND i.variant_id = ri.variant_id`, reservationID); err != nil {
				return err
			}
			if _, err := tx.Exec(transactionContext, `UPDATE inventory_reservations SET status = 'committed', updated_at = NOW() WHERE id = $1`, reservationID); err != nil {
				return err
			}
		} else if _, err := tx.Exec(transactionContext, `INSERT INTO payment_attempts (order_id, provider, status, amount_cents) VALUES ($1, 'razorpay', 'created', $2)`, order.ID, order.TotalCents); err != nil {
			return err
		}
		if _, err := tx.Exec(transactionContext, `INSERT INTO order_status_history (order_id, to_status, actor_id, reason) VALUES ($1, $2, $3, $4)`, order.ID, status, userID, historyReason); err != nil {
			return err
		}
		payload, _ := json.Marshal(map[string]any{"order_id": order.ID, "order_number": order.OrderNumber, "status": order.Status, "payment_method": paymentMethod})
		_, err = tx.Exec(transactionContext, `INSERT INTO outbox_events (event_type, aggregate_type, aggregate_id, payload) VALUES ('OrderCreated', 'Order', $1, $2)`, order.OrderNumber, payload)
		if err != nil {
			return err
		}
		_, err = tx.Exec(transactionContext, `UPDATE carts SET status = 'checked_out', updated_at = NOW() WHERE id = $1`, cartID)
		return err
	})
	return order, err
}

func (r *CommerceRepository) ListOrders(ctx context.Context, userID int64, limit int) ([]domaincommerce.Order, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, order_number, status, payment_method, currency, subtotal_cents, shipping_cents, total_cents, created_at FROM orders WHERE user_id = $1 ORDER BY created_at DESC, id DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	orders := make([]domaincommerce.Order, 0)
	for rows.Next() {
		var order domaincommerce.Order
		if err := rows.Scan(&order.ID, &order.OrderNumber, &order.Status, &order.PaymentMethod, &order.Currency, &order.SubtotalCents, &order.ShippingCents, &order.TotalCents, &order.CreatedAt); err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	return orders, rows.Err()
}

func (r *CommerceRepository) GetOrder(ctx context.Context, userID int64, orderNumber string) (domaincommerce.OrderDetail, error) {
	var detail domaincommerce.OrderDetail
	if err := r.pool.QueryRow(ctx, `
		SELECT o.id, o.order_number, o.status, o.payment_method, o.currency, o.subtotal_cents,
		       o.shipping_cents, o.total_cents, o.created_at,
	       CASE WHEN o.payment_method = 'cash_on_delivery' AND o.status = 'cancelled' THEN 'cancelled_unpaid'
	            WHEN o.payment_method = 'cash_on_delivery' THEN 'due_on_delivery'
		            ELSE COALESCE((SELECT pa.status FROM payment_attempts pa WHERE pa.order_id = o.id ORDER BY pa.id DESC LIMIT 1), '') END
		FROM orders o
		WHERE o.user_id = $1 AND o.order_number = $2`, userID, orderNumber).Scan(&detail.ID, &detail.OrderNumber, &detail.Status, &detail.PaymentMethod, &detail.Currency, &detail.SubtotalCents, &detail.ShippingCents, &detail.TotalCents, &detail.CreatedAt, &detail.PaymentStatus); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domaincommerce.OrderDetail{}, ports.ErrOrderNotFound
		}
		return domaincommerce.OrderDetail{}, err
	}
	rows, err := r.pool.Query(ctx, `SELECT product_name, sku, quantity, unit_price_cents, line_total_cents FROM order_items WHERE order_id = $1 ORDER BY id`, detail.ID)
	if err != nil {
		return domaincommerce.OrderDetail{}, err
	}
	defer rows.Close()
	detail.Items = make([]domaincommerce.OrderItem, 0)
	for rows.Next() {
		var item domaincommerce.OrderItem
		if err := rows.Scan(&item.ProductName, &item.SKU, &item.Quantity, &item.UnitPriceCents, &item.LineTotalCents); err != nil {
			return domaincommerce.OrderDetail{}, err
		}
		detail.Items = append(detail.Items, item)
	}
	if err := rows.Err(); err != nil {
		return domaincommerce.OrderDetail{}, err
	}
	fulfillmentRows, err := r.pool.Query(ctx, `
		SELECT s.display_name, sf.status, sf.carrier, sf.tracking_number,
		       sf.last_note, sf.updated_at
		FROM seller_fulfillments sf
		JOIN sellers s ON s.id = sf.seller_id
		WHERE sf.order_id = $1
		ORDER BY sf.updated_at DESC, sf.id DESC`, detail.ID)
	if err != nil {
		return domaincommerce.OrderDetail{}, err
	}
	defer fulfillmentRows.Close()
	detail.Fulfillments = make([]domaincommerce.OrderFulfillment, 0)
	for fulfillmentRows.Next() {
		var fulfillment domaincommerce.OrderFulfillment
		if err := fulfillmentRows.Scan(&fulfillment.SellerName, &fulfillment.Status, &fulfillment.Carrier, &fulfillment.TrackingNumber, &fulfillment.LastNote, &fulfillment.UpdatedAt); err != nil {
			return domaincommerce.OrderDetail{}, err
		}
		detail.Fulfillments = append(detail.Fulfillments, fulfillment)
	}
	return detail, fulfillmentRows.Err()
}

func (r *CommerceRepository) CancelOrder(ctx context.Context, userID int64, orderNumber string) error {
	return WithinTransaction(ctx, r.pool, func(transactionContext context.Context, tx pgx.Tx) error {
		var orderID, reservationID int64
		var status, paymentMethod string
		if err := tx.QueryRow(transactionContext, `SELECT id, status, payment_method FROM orders WHERE user_id = $1 AND order_number = $2 FOR UPDATE`, userID, orderNumber).Scan(&orderID, &status, &paymentMethod); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ports.ErrOrderNotFound
			}
			return err
		}
		if status == "processing" && paymentMethod == "cash_on_delivery" {
			var fulfillmentStarted bool
			if err := tx.QueryRow(transactionContext, `SELECT EXISTS (SELECT 1 FROM seller_fulfillments WHERE order_id = $1 AND status IN ('processing', 'shipped', 'delivered'))`, orderID).Scan(&fulfillmentStarted); err != nil {
				return err
			}
			if fulfillmentStarted {
				return ports.ErrInvalidState
			}
			if err := tx.QueryRow(transactionContext, `SELECT id FROM inventory_reservations WHERE order_id = $1 AND status = 'committed' FOR UPDATE`, orderID).Scan(&reservationID); err != nil {
				return err
			}
			type cancelledLine struct {
				sellerID, variantID int64
				available, quantity int
			}
			rows, err := tx.Query(transactionContext, `
				SELECT s.id, i.variant_id, i.available_quantity, ri.quantity
				FROM inventory_reservation_items ri
				JOIN inventory_stock i ON i.variant_id = ri.variant_id
				JOIN product_variants pv ON pv.id = i.variant_id
				JOIN products p ON p.id = pv.product_id
				JOIN sellers s ON s.id = p.seller_id
				WHERE ri.reservation_id = $1
				ORDER BY i.variant_id FOR UPDATE OF i`, reservationID)
			if err != nil {
				return err
			}
			lines := make([]cancelledLine, 0)
			for rows.Next() {
				var line cancelledLine
				if err := rows.Scan(&line.sellerID, &line.variantID, &line.available, &line.quantity); err != nil {
					rows.Close()
					return err
				}
				lines = append(lines, line)
			}
			if err := rows.Err(); err != nil {
				rows.Close()
				return err
			}
			rows.Close()
			for _, line := range lines {
				after := line.available + line.quantity
				if _, err := tx.Exec(transactionContext, `UPDATE inventory_stock SET available_quantity = $1, updated_at = NOW() WHERE variant_id = $2`, after, line.variantID); err != nil {
					return err
				}
				if _, err := tx.Exec(transactionContext, `INSERT INTO inventory_transactions (seller_id, variant_id, actor_id, delta, before_quantity, after_quantity, reason) VALUES ($1, $2, $3, $4, $5, $6, 'customer_cancelled_unpaid_order')`, line.sellerID, line.variantID, userID, line.quantity, line.available, after); err != nil {
					return err
				}
			}
			if _, err := tx.Exec(transactionContext, `UPDATE inventory_reservations SET status = 'released', updated_at = NOW() WHERE id = $1`, reservationID); err != nil {
				return err
			}
			if _, err := tx.Exec(transactionContext, `UPDATE orders SET status = 'cancelled', updated_at = NOW() WHERE id = $1`, orderID); err != nil {
				return err
			}
			if _, err := tx.Exec(transactionContext, `INSERT INTO order_status_history (order_id, from_status, to_status, actor_id, reason) VALUES ($1, 'processing', 'cancelled', $2, 'customer cancelled unpaid cash-on-delivery order before fulfilment')`, orderID, userID); err != nil {
				return err
			}
			_, err = tx.Exec(transactionContext, `INSERT INTO audit_logs (actor_id, action, resource_type, resource_id, request_id, metadata) VALUES ($1, 'order.manual_payment_cancelled', 'order', $2, $3, jsonb_build_object('payment_method', 'cash_on_delivery'))`, fmt.Sprint(userID), orderNumber, ports.RequestID(transactionContext))
			return err
		}
		if status != "pending_payment" {
			return ports.ErrInvalidState
		}
		if err := tx.QueryRow(transactionContext, `SELECT id FROM inventory_reservations WHERE order_id = $1 AND status = 'active' FOR UPDATE`, orderID).Scan(&reservationID); err != nil {
			return err
		}
		if _, err := releaseReservationWithStatus(transactionContext, tx, reservationID, "released"); err != nil {
			return err
		}
		if _, err := tx.Exec(transactionContext, `UPDATE orders SET status = 'cancelled', updated_at = NOW() WHERE id = $1`, orderID); err != nil {
			return err
		}
		_, err := tx.Exec(transactionContext, `INSERT INTO order_status_history (order_id, from_status, to_status, actor_id, reason) VALUES ($1, 'pending_payment', 'cancelled', $2, 'customer cancelled before payment')`, orderID, userID)
		return err
	})
}

func (r *CommerceRepository) ReleaseExpiredReservations(ctx context.Context, limit int) (released int, err error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	err = WithinTransaction(ctx, r.pool, func(transactionContext context.Context, tx pgx.Tx) error {
		rows, queryErr := tx.Query(transactionContext, `
			SELECT id, order_id
			FROM inventory_reservations
			WHERE status = 'active' AND expires_at <= NOW()
			ORDER BY id
			FOR UPDATE SKIP LOCKED
			LIMIT $1`, limit)
		if queryErr != nil {
			return queryErr
		}
		type expiredReservation struct {
			id      int64
			orderID int64
		}
		reservations := make([]expiredReservation, 0, limit)
		for rows.Next() {
			var reservationID, orderID int64
			if scanErr := rows.Scan(&reservationID, &orderID); scanErr != nil {
				rows.Close()
				return scanErr
			}
			reservations = append(reservations, expiredReservation{id: reservationID, orderID: orderID})
		}
		if rowsErr := rows.Err(); rowsErr != nil {
			rows.Close()
			return rowsErr
		}
		rows.Close()
		for _, reservation := range reservations {
			if _, releaseErr := releaseReservationWithStatus(transactionContext, tx, reservation.id, "expired"); releaseErr != nil {
				return releaseErr
			}
			if _, updateErr := tx.Exec(transactionContext, `UPDATE orders SET status = 'payment_failed', updated_at = NOW() WHERE id = $1 AND status = 'pending_payment'`, reservation.orderID); updateErr != nil {
				return updateErr
			}
			if _, historyErr := tx.Exec(transactionContext, `INSERT INTO order_status_history (order_id, from_status, to_status, reason) VALUES ($1, 'pending_payment', 'payment_failed', 'payment reservation expired')`, reservation.orderID); historyErr != nil {
				return historyErr
			}
			if _, paymentErr := tx.Exec(transactionContext, `UPDATE payment_attempts SET status = 'failed', failure_code = 'reservation_expired', updated_at = NOW() WHERE order_id = $1 AND status IN ('created', 'pending')`, reservation.orderID); paymentErr != nil {
				return paymentErr
			}
			released++
		}
		return nil
	})
	return released, err
}

func (r *CommerceRepository) assertAvailableVariant(ctx context.Context, variantID int64) error {
	var exists bool
	if err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM product_variants pv JOIN products p ON p.id = pv.product_id JOIN inventory_stock i ON i.variant_id = pv.id WHERE pv.id = $1 AND pv.status = 'active' AND p.status = 'approved')`, variantID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ports.ErrNotFound
	}
	return nil
}

func mapCommerceError(err error) error {
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) && pgError.Code == "23505" {
		return fmt.Errorf("duplicate commerce record: %w", err)
	}
	return err
}

func hashGuestToken(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}

func checkoutRequestHash(cartID int64, addressJSON []byte) string {
	digest := sha256.Sum256([]byte(fmt.Sprintf("%d:", cartID) + string(addressJSON)))
	return hex.EncodeToString(digest[:])
}

func checkoutRequestHashMode(cartID int64, addressJSON []byte, paymentMethod string) string {
	digest := sha256.Sum256([]byte(fmt.Sprintf("%d:%s:", cartID, paymentMethod) + string(addressJSON)))
	return hex.EncodeToString(digest[:])
}

func newOrderNumber(userID int64) string {
	return fmt.Sprintf("WC-%s-%d", time.Now().UTC().Format("20060102-150405.000000000"), userID)
}

func normalizeGuestToken(token string) string {
	return strings.TrimSpace(token)
}
