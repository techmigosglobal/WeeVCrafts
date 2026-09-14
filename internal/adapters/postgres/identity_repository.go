package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wecratfs/commerce/internal/adapters/postgres/generated"
	domainidentity "github.com/wecratfs/commerce/internal/domain/identity"
	"github.com/wecratfs/commerce/internal/ports"
)

type UserRepository struct {
	pool    *pgxpool.Pool
	queries *generated.Queries
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool, queries: generated.New(pool)}
}

func (r *UserRepository) CreateCustomer(ctx context.Context, email, displayName, passwordHash string) (domainidentity.User, error) {
	var user generated.InsertUserRow
	err := WithinTransactionWithQueries(ctx, r.pool, func(transactionContext context.Context, queries *generated.Queries) error {
		var err error
		user, err = queries.InsertUser(transactionContext, generated.InsertUserParams{
			Email:       email,
			DisplayName: displayName,
			Status:      "active",
		})
		if err != nil {
			return err
		}
		if err := queries.InsertCredential(transactionContext, generated.InsertCredentialParams{
			UserID:       user.ID,
			PasswordHash: passwordHash,
		}); err != nil {
			return err
		}
		return queries.AssignUserRole(transactionContext, generated.AssignUserRoleParams{
			UserID:   user.ID,
			RoleSlug: string(domainidentity.RoleCustomer),
		})
	})
	if err != nil {
		var pgError *pgconn.PgError
		if errors.As(err, &pgError) && pgError.Code == "23505" {
			return domainidentity.User{}, ports.ErrUserAlreadyExists
		}
		return domainidentity.User{}, fmt.Errorf("create customer: %w", err)
	}
	return mapCreatedUser(user), nil
}

func (r *UserRepository) GetCredentialsByEmail(ctx context.Context, email string) (domainidentity.Credentials, error) {
	row, err := r.queries.GetCredentialsByEmail(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		return domainidentity.Credentials{}, ports.ErrCredentialsNotFound
	}
	if err != nil {
		return domainidentity.Credentials{}, fmt.Errorf("get credentials: %w", err)
	}
	roles, err := r.queries.GetUserRoles(ctx, row.ID)
	if err != nil {
		return domainidentity.Credentials{}, fmt.Errorf("get user roles: %w", err)
	}
	return domainidentity.Credentials{
		User:           mapCredentialUser(row, roles),
		PasswordHash:   row.PasswordHash,
		FailedAttempts: int(row.FailedAttempts),
		LockedUntil:    optionalTime(row.LockedUntil),
		EmailVerified:  row.EmailVerifiedAt.Valid,
	}, nil
}

func (r *UserRepository) RecordFailedLogin(ctx context.Context, userID int64) error {
	return WithinTransaction(ctx, r.pool, func(transactionContext context.Context, tx pgx.Tx) error {
		result, err := tx.Exec(transactionContext, `
			UPDATE credentials
			SET failed_attempts = failed_attempts + 1,
				locked_until = CASE
					WHEN failed_attempts + 1 >= 5 THEN NOW() + INTERVAL '15 minutes'
					ELSE locked_until
				END,
				updated_at = NOW()
			WHERE user_id = $1`, userID)
		if err != nil {
			return fmt.Errorf("record failed login: %w", err)
		}
		if result.RowsAffected() != 1 {
			return ports.ErrCredentialsNotFound
		}
		if _, err := tx.Exec(transactionContext, `
			INSERT INTO audit_logs (actor_id, action, resource_type, resource_id, request_id, metadata)
			VALUES ($1, 'auth.login_failed', 'user', $1, $2, '{"reason":"invalid_credentials"}'::jsonb)`, fmt.Sprint(userID), ports.RequestID(transactionContext)); err != nil {
			return fmt.Errorf("audit failed login: %w", err)
		}
		return nil
	})
}

func (r *UserRepository) ResetFailedLogin(ctx context.Context, userID int64) error {
	if err := r.queries.ResetFailedLogin(ctx, userID); err != nil {
		return fmt.Errorf("reset failed login: %w", err)
	}
	return nil
}

func (r *UserRepository) ListCustomers(ctx context.Context, actorID int64, limit int) ([]domainidentity.AdminCustomer, error) {
	var allowed bool
	if err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM user_roles WHERE user_id = $1 AND role_slug IN ($2, $3))`, actorID, string(domainidentity.RoleMarketplaceAdmin), string(domainidentity.RoleSuperAdmin)).Scan(&allowed); err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ports.ErrForbidden
	}
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	rows, err := r.pool.Query(ctx, `
		SELECT u.id, u.display_name, u.email, u.status, u.created_at, COUNT(DISTINCT o.id)
		FROM users u
		JOIN user_roles ur ON ur.user_id = u.id AND ur.role_slug = 'customer'
		LEFT JOIN orders o ON o.user_id = u.id
		GROUP BY u.id, u.display_name, u.email, u.status, u.created_at
		ORDER BY u.created_at DESC, u.id DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	customers := make([]domainidentity.AdminCustomer, 0)
	for rows.Next() {
		var customer domainidentity.AdminCustomer
		var email string
		if err := rows.Scan(&customer.ID, &customer.DisplayName, &email, &customer.Status, &customer.CreatedAt, &customer.OrderCount); err != nil {
			return nil, err
		}
		customer.EmailMasked = maskCustomerEmail(email)
		customers = append(customers, customer)
	}
	return customers, rows.Err()
}

func maskCustomerEmail(email string) string {
	parts := strings.SplitN(strings.TrimSpace(email), "@", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "hidden"
	}
	return parts[0][:1] + "***@" + parts[1]
}

var _ ports.CustomerDirectoryRepository = (*UserRepository)(nil)

func mapUser(user generated.User, roles []string) domainidentity.User {
	roleValues := make([]domainidentity.Role, 0, len(roles))
	for _, role := range roles {
		roleValues = append(roleValues, domainidentity.Role(role))
	}
	return domainidentity.User{
		ID:          user.ID,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		Status:      user.Status,
		Roles:       roleValues,
		CreatedAt:   user.CreatedAt.Time,
	}
}

func mapCreatedUser(user generated.InsertUserRow) domainidentity.User {
	return domainidentity.User{
		ID:          user.ID,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		Status:      user.Status,
		Roles:       []domainidentity.Role{domainidentity.RoleCustomer},
		CreatedAt:   user.CreatedAt.Time,
	}
}

func mapCredentialUser(row generated.GetCredentialsByEmailRow, roles []string) domainidentity.User {
	user := mapUser(generated.User{
		ID:          row.ID,
		Email:       row.Email,
		DisplayName: row.DisplayName,
		Status:      row.Status,
		CreatedAt:   row.CreatedAt,
	}, roles)
	user.EmailVerified = row.EmailVerifiedAt.Valid
	return user
}
