-- name: InsertSession :exec
INSERT INTO sessions (token_hash, employee_id, created_at, last_seen_at, idle_expires_at, absolute_expires_at, user_agent)
VALUES (sqlc.arg(token_hash), sqlc.arg(employee_id), sqlc.arg(now), sqlc.arg(now), sqlc.arg(idle_expires_at), sqlc.arg(absolute_expires_at), sqlc.arg(user_agent));

-- name: GetValidSession :one
SELECT s.token_hash, s.employee_id, s.last_seen_at, s.idle_expires_at, s.absolute_expires_at,
       e.full_name, e.email, e.role, e.active, e.manager_id
FROM sessions s
JOIN employees e ON e.id = s.employee_id
WHERE s.token_hash = sqlc.arg(token_hash)
  AND s.revoked_at IS NULL
  AND s.idle_expires_at > sqlc.arg(now)
  AND s.absolute_expires_at > sqlc.arg(now);

-- name: TouchSession :exec
UPDATE sessions
SET last_seen_at = sqlc.arg(now), idle_expires_at = LEAST(sqlc.arg(idle_expires_at), absolute_expires_at)
WHERE token_hash = sqlc.arg(token_hash) AND revoked_at IS NULL;

-- name: RevokeSession :exec
UPDATE sessions SET revoked_at = sqlc.arg(now) WHERE token_hash = sqlc.arg(token_hash) AND revoked_at IS NULL;

-- name: RevokeEmployeeSessions :execrows
UPDATE sessions SET revoked_at = sqlc.arg(now) WHERE employee_id = sqlc.arg(employee_id) AND revoked_at IS NULL;

-- name: DeleteDeadSessions :execrows
DELETE FROM sessions
WHERE revoked_at IS NOT NULL OR idle_expires_at <= sqlc.arg(now) OR absolute_expires_at <= sqlc.arg(now);

-- name: InsertOAuthState :exec
INSERT INTO oauth_states (state_hash, nonce, code_verifier, return_to, created_at, expires_at)
VALUES (sqlc.arg(state_hash), sqlc.arg(nonce), sqlc.arg(code_verifier), sqlc.arg(return_to), sqlc.arg(now), sqlc.arg(expires_at));

-- name: GetOAuthStateForUpdate :one
SELECT nonce, code_verifier, return_to, expires_at FROM oauth_states WHERE state_hash = sqlc.arg(state_hash) FOR UPDATE;

-- name: DeleteOAuthState :exec
DELETE FROM oauth_states WHERE state_hash = sqlc.arg(state_hash);

-- name: DeleteExpiredOAuthStates :execrows
DELETE FROM oauth_states WHERE expires_at <= sqlc.arg(now);
