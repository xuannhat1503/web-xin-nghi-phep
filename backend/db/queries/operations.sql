-- name: RecordWorkerRun :exec
INSERT INTO worker_status (name, last_run_at, last_success_at, last_error)
VALUES (sqlc.arg(name), sqlc.arg(run_at), sqlc.narg(success_at), sqlc.narg(last_error))
ON DUPLICATE KEY UPDATE
    last_run_at = VALUES(last_run_at),
    last_success_at = COALESCE(VALUES(last_success_at), last_success_at),
    last_error = VALUES(last_error);
