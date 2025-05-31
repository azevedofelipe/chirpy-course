-- name: CreateToken :one
INSERT INTO refresh_tokens (token, created_at, updated_at, user_id, expires_at, revoked_at)
VALUES (
	$1,
    NOW(),
    NOW(),
    $2,
    $3,
	NULL

)
RETURNING *;


-- name: GetUserFromRefreshToken :one
SELECT 
	users.id,
	refresh_tokens.revoked_at,
	refresh_tokens.expires_at
FROM refresh_tokens 
JOIN users ON refresh_tokens.user_id = users.id
WHERE refresh_tokens.token = $1;

