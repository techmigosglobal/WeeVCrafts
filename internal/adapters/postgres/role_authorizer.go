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
