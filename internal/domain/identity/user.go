package identity

import "time"

type Role string

const (
	RoleCustomer         Role = "customer"
	RoleSellerOwner      Role = "seller_owner"
	RoleSellerStaff      Role = "seller_staff"
	RoleMarketplaceAdmin Role = "marketplace_admin"
	RoleSuperAdmin       Role = "super_admin"
	RoleSupportAgent     Role = "support_agent"
	RoleFinanceOperator  Role = "finance_operator"
	RoleOperations       Role = "operations_returns"
)

type User struct {
	ID            int64     `json:"id"`
	Email         string    `json:"email"`
	DisplayName   string    `json:"display_name"`
	Status        string    `json:"status"`
	EmailVerified bool      `json:"email_verified"`
	Roles         []Role    `json:"roles"`
	CreatedAt     time.Time `json:"created_at"`
}

type Credentials struct {
	User           User
	PasswordHash   string
	FailedAttempts int
	LockedUntil    *time.Time
	EmailVerified  bool
}
