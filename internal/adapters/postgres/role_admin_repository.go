package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	domainidentity "github.com/wecratfs/commerce/internal/domain/identity"
	"github.com/wecratfs/commerce/internal/ports"
)

type RoleAdministrationRepository struct {
	pool *pgxpool.Pool
}

func NewRoleAdministrationRepository(pool *pgxpool.Pool) *RoleAdministrationRepository {
	return &RoleAdministrationRepository{pool: pool}
}

func (r *RoleAdministrationRepository) ListRoleAssignments(ctx context.Context, actorID int64) ([]domainidentity.RoleAssignment, error) {
	if err := r.requireSuperAdmin(ctx, actorID); err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `
		SELECT u.id, u.email, u.display_name, u.status,
		       COALESCE(array_agg(ur.role_slug ORDER BY ur.role_slug) FILTER (WHERE ur.role_slug IS NOT NULL), ARRAY[]::text[])
		FROM users u
		LEFT JOIN user_roles ur ON ur.user_id = u.id
		GROUP BY u.id, u.email, u.display_name, u.status
		ORDER BY u.created_at DESC, u.id DESC
		LIMIT 200`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	assignments := make([]domainidentity.RoleAssignment, 0)
	for rows.Next() {
		var assignment domainidentity.RoleAssignment
		if err := rows.Scan(&assignment.ID, &assignment.Email, &assignment.DisplayName, &assignment.Status, &assignment.Roles); err != nil {
			return nil, err
		}
		assignments = append(assignments, assignment)
	}
	return assignments, rows.Err()
}

func (r *RoleAdministrationRepository) UpdateRoleAssignment(ctx context.Context, actorID, userID int64, role domainidentity.Role, grant bool, reason string) error {
	if err := r.requireSuperAdmin(ctx, actorID); err != nil {
		return err
	}
	return WithinTransaction(ctx, r.pool, func(transactionContext context.Context, tx pgx.Tx) error {
		var exists bool
		if err := tx.QueryRow(transactionContext, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, userID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return ports.ErrNotFound
		}
		action := "identity.role_revoked"
		var rowsAffected int64
		if grant {
			action = "identity.role_granted"
			commandTag, err := tx.Exec(transactionContext, `INSERT INTO user_roles (user_id, role_slug) VALUES ($1, $2) ON CONFLICT DO NOTHING`, userID, string(role))
			if err != nil {
				return err
			}
			rowsAffected = commandTag.RowsAffected()
			if rowsAffected != 1 {
				return ports.ErrRoleState
			}
		} else {
			commandTag, err := tx.Exec(transactionContext, `DELETE FROM user_roles WHERE user_id = $1 AND role_slug = $2`, userID, string(role))
			if err != nil {
				return err
			}
			rowsAffected = commandTag.RowsAffected()
		}
		if !grant && rowsAffected != 1 {
			return ports.ErrRoleState
		}
		metadata, _ := json.Marshal(map[string]any{"role": role, "grant": grant, "reason": reason})
		_, err := tx.Exec(transactionContext, `INSERT INTO audit_logs (actor_id, action, resource_type, resource_id, request_id, metadata) VALUES ($1, $2, 'user', $3, $4, $5)`, fmt.Sprint(actorID), action, fmt.Sprint(userID), ports.RequestID(transactionContext), metadata)
		return err
	})
}

func (r *RoleAdministrationRepository) requireSuperAdmin(ctx context.Context, actorID int64) error {
	var allowed bool
	if err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM user_roles WHERE user_id = $1 AND role_slug = $2)`, actorID, string(domainidentity.RoleSuperAdmin)).Scan(&allowed); err != nil {
		return err
	}
	if !allowed {
		return ports.ErrForbidden
	}
	return nil
}

var _ ports.RoleAdministrationRepository = (*RoleAdministrationRepository)(nil)
