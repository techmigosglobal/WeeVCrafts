package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	domainidentity "github.com/wecratfs/commerce/internal/domain/identity"
)

// RoleAuthorizer reads the authoritative role assignments from PostgreSQL.
// It intentionally returns a boolean decision so application services can
// keep authorization failures indistinguishable from missing scope.
type RoleAuthorizer struct {
	pool *pgxpool.Pool
}

func NewRoleAuthorizer(pool *pgxpool.Pool) *RoleAuthorizer {
	return &RoleAuthorizer{pool: pool}
}

func (r *RoleAuthorizer) HasAnyRole(ctx context.Context, userID int64, roles ...domainidentity.Role) (bool, error) {
	if userID <= 0 || len(roles) == 0 {
		return false, nil
	}
	roleSlugs := make([]string, 0, len(roles))
	for _, role := range roles {
		roleSlugs = append(roleSlugs, string(role))
	}
	var allowed bool
	if err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM user_roles
			WHERE user_id = $1 AND role_slug = ANY($2::text[])
		)`, userID, roleSlugs).Scan(&allowed); err != nil {
		return false, fmt.Errorf("check user roles: %w", err)
	}
	return allowed, nil
}

func (r *RoleAuthorizer) HasSellerPermission(ctx context.Context, userID int64, permission string) (bool, error) {
	if userID <= 0 || permission == "" {
		return false, nil
	}
	var allowed bool
	if err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM seller_users su
			JOIN sellers s ON s.id = su.seller_id AND s.status = 'active'
			WHERE su.user_id = $1 AND su.status = 'active' AND $2 = ANY(su.permissions)
		)`, userID, permission).Scan(&allowed); err != nil {
		return false, fmt.Errorf("check seller permission: %w", err)
	}
	return allowed, nil
}
