-- name: GetAppUserByEmail :one
SELECT * FROM app_users
WHERE email = $1 LIMIT 1;

-- name: CreateAppSession :one
INSERT INTO app_sessions (
    user_id
) VALUES (
    $1
)
RETURNING *;

-- name: GetAppUserBySessionID :one
SELECT u.* FROM app_users u
JOIN app_sessions s ON u.user_id = s.user_id
WHERE s.session_id = $1 AND s.expires_at > NOW();

-- name: CreateAppUser :one
INSERT INTO app_users (
    name,
    email,
    hashed_password,
    role,
    dni
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: ListAppUsers :many
SELECT * FROM app_users
ORDER BY created_at DESC;

