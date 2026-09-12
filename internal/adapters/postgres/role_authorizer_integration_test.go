package postgres

import (
	"context"
	"fmt"
	"testing"
	"time"

	domainidentity "github.com/wecratfs/commerce/internal/domain/identity"
)

func TestRoleAuthorizerReadsAuthoritativeAssignments(t *testing.T) {
	pool := openIntegrationPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	var userID int64
	if err := pool.QueryRow(ctx, `INSERT INTO users (email, display_name, status) VALUES ($1, 'Role Fixture', 'active') RETURNING id`, "role-authorizer-"+suffix+"@example.invalid").Scan(&userID); err != nil {
		t.Fatalf("insert role fixture: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})
	if _, err := pool.Exec(ctx, `INSERT INTO user_roles (user_id, role_slug) VALUES ($1, 'marketplace_admin')`, userID); err != nil {
		t.Fatalf("assign role fixture: %v", err)
	}

	authorizer := NewRoleAuthorizer(pool)
	allowed, err := authorizer.HasAnyRole(ctx, userID, domainidentity.RoleMarketplaceAdmin, domainidentity.RoleSuperAdmin)
	if err != nil || !allowed {
		t.Fatalf("expected assigned role to authorize, allowed=%v err=%v", allowed, err)
	}
	allowed, err = authorizer.HasAnyRole(ctx, userID, domainidentity.RoleFinanceOperator)
	if err != nil {
		t.Fatalf("check denied role: %v", err)
	}
	if allowed {
		t.Fatal("unassigned role unexpectedly authorized")
	}
}
