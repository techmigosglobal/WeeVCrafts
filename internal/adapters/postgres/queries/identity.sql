-- name: InsertUser :one
INSERT INTO users (email, display_name, status)
VALUES (@email, @display_name, @status)
RETURNING id, email, display_name, status, created_at, updated_at;

-- name: GetCredentialsByEmail :one
SELECT u.id, u.email, u.display_name, u.status, u.created_at,
       u.email_verified_at,
       c.password_hash, c.failed_attempts, c.locked_until
FROM users u
JOIN credentials c ON c.user_id = u.id
WHERE LOWER(u.email) = LOWER(@email);

-- name: GetUserRoles :many
SELECT role_slug
FROM user_roles
WHERE user_id = @user_id
ORDER BY role_slug;

-- name: InsertCredential :exec
INSERT INTO credentials (user_id, password_hash)
VALUES (@user_id, @password_hash);

-- name: AssignUserRole :exec
INSERT INTO user_roles (user_id, role_slug)
VALUES (@user_id, @role_slug)
ON CONFLICT (user_id, role_slug) DO NOTHING;

-- name: RecordFailedLogin :exec
UPDATE credentials
SET failed_attempts = failed_attempts + 1,
    locked_until = CASE
        WHEN failed_attempts + 1 >= @lock_threshold
            THEN NOW() + @lock_duration::interval
        ELSE locked_until
    END,
    updated_at = NOW()
WHERE user_id = @user_id;

-- name: ResetFailedLogin :exec
UPDATE credentials
SET failed_attempts = 0, locked_until = NULL, updated_at = NOW()
WHERE user_id = @user_id;
