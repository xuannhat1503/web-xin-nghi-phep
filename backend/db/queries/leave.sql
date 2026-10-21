-- Leave requests (owner side). Every query is scoped by employee_id so another user's id simply finds nothing.

-- name: CreateLeaveDraft :execlastid
INSERT INTO leave_requests (employee_id) VALUES (sqlc.arg(employee_id));

-- name: CountOpenDrafts :one
SELECT COUNT(*) FROM leave_requests
WHERE employee_id = sqlc.arg(employee_id) AND status = 'DRAFT' AND submit_count = 0;

-- name: ListLeaveDrafts :many
SELECT id, leave_type, start_date, end_date, start_time, end_time, total_minutes, version, created_at, updated_at
FROM leave_requests
WHERE employee_id = sqlc.arg(employee_id) AND status = 'DRAFT'
ORDER BY updated_at DESC, id DESC
LIMIT 20;

-- name: GetOwnLeaveRequest :one
SELECT lr.id, lr.request_code, lr.employee_id, lr.reviewer_id, lr.leave_type, lr.start_date, lr.end_date, lr.start_time, lr.end_time,
       lr.total_minutes, lr.reason_code, lr.reason_text, lr.status, lr.rejection_reason, lr.change_request_reason, lr.reviewed_at,
       lr.manager_name_snapshot, lr.manager_email_snapshot, lr.policy_version_id, lr.calculation_snapshot, lr.calendar_changed,
       lr.submit_count, lr.version, lr.created_at, lr.first_submitted_at, lr.submitted_at, lr.cancelled_at, lr.updated_at,
       r.label AS reason_label
FROM leave_requests lr
LEFT JOIN leave_reasons r ON r.code = lr.reason_code
WHERE lr.id = sqlc.arg(id) AND lr.employee_id = sqlc.arg(employee_id);

-- name: LockOwnLeaveRequest :one
SELECT id, request_code, status, version, submit_count, leave_type, start_date, end_date, start_time, end_time, reason_code, reason_text
FROM leave_requests
WHERE id = sqlc.arg(id) AND employee_id = sqlc.arg(employee_id)
FOR UPDATE;

-- Auto-save: only the owner, only editable statuses, only the version they last saw.
-- name: UpdateLeaveDraft :execrows
UPDATE leave_requests
SET leave_type = sqlc.narg(leave_type), start_date = sqlc.narg(start_date), end_date = sqlc.narg(end_date),
    start_time = sqlc.narg(start_time), end_time = sqlc.narg(end_time),
    reason_code = sqlc.narg(reason_code), reason_text = sqlc.arg(reason_text),
    total_minutes = sqlc.arg(total_minutes), version = version + 1
WHERE id = sqlc.arg(id) AND employee_id = sqlc.arg(employee_id) AND version = sqlc.arg(version)
  AND status IN ('DRAFT', 'NEEDS_CHANGES');

-- Never-submitted drafts are the only requests that can be hard-deleted (their events cascade).
-- name: DeleteLeaveDraft :execrows
DELETE FROM leave_requests
WHERE id = sqlc.arg(id) AND employee_id = sqlc.arg(employee_id) AND version = sqlc.arg(version)
  AND status = 'DRAFT' AND submit_count = 0;

-- name: LastDraftSavedEventAt :one
SELECT MAX(created_at) AS last_at FROM leave_request_events
WHERE leave_request_id = sqlc.arg(leave_request_id) AND action = 'DRAFT_SAVED';

-- name: InsertLeaveEvent :exec
INSERT INTO leave_request_events (leave_request_id, action, actor_id, from_status, to_status, request_version, submit_no, comment, metadata)
VALUES (sqlc.arg(leave_request_id), sqlc.arg(action), sqlc.narg(actor_id), sqlc.narg(from_status), sqlc.narg(to_status),
        sqlc.arg(request_version), sqlc.narg(submit_no), sqlc.narg(comment), sqlc.narg(metadata));

-- Serializes submissions of one employee so overlapping requests cannot be submitted concurrently.
-- name: LockEmployeeForSubmit :one
SELECT id, full_name, email, active, manager_id FROM employees WHERE id = sqlc.arg(id) FOR UPDATE;

-- name: GetManagerInfo :one
SELECT id, full_name, email, active FROM employees WHERE id = sqlc.arg(id);

-- name: FindOverlappingRequest :one
SELECT id, request_code, status FROM leave_requests
WHERE employee_id = sqlc.arg(employee_id) AND id <> sqlc.arg(exclude_id)
  AND status IN ('PENDING', 'APPROVED', 'NEEDS_CHANGES')
  AND TIMESTAMP(start_date, start_time) < sqlc.arg(range_end)
  AND TIMESTAMP(end_date, end_time) > sqlc.arg(range_start)
ORDER BY start_date
LIMIT 1;

-- name: EnsureRequestCodeSequence :exec
INSERT INTO request_code_sequences (year, next_value) VALUES (sqlc.arg(year), 1)
ON DUPLICATE KEY UPDATE year = year;

-- name: LockRequestCodeSequence :one
SELECT next_value FROM request_code_sequences WHERE year = sqlc.arg(year) FOR UPDATE;

-- name: BumpRequestCodeSequence :exec
UPDATE request_code_sequences SET next_value = next_value + 1 WHERE year = sqlc.arg(year);

-- name: SubmitLeaveRequest :execrows
UPDATE leave_requests
SET status = 'PENDING', request_code = sqlc.arg(request_code), reviewer_id = sqlc.arg(reviewer_id),
    total_minutes = sqlc.arg(total_minutes), policy_version_id = sqlc.arg(policy_version_id),
    calculation_snapshot = sqlc.arg(calculation_snapshot),
    manager_name_snapshot = sqlc.arg(manager_name), manager_email_snapshot = sqlc.arg(manager_email),
    calendar_changed = FALSE, submit_count = submit_count + 1,
    first_submitted_at = COALESCE(first_submitted_at, sqlc.arg(now)), submitted_at = sqlc.arg(now),
    version = version + 1
WHERE id = sqlc.arg(id) AND employee_id = sqlc.arg(employee_id) AND version = sqlc.arg(version)
  AND status IN ('DRAFT', 'NEEDS_CHANGES');

-- name: DeleteLeaveDays :exec
DELETE FROM leave_request_days WHERE leave_request_id = sqlc.arg(leave_request_id);

-- name: InsertLeaveDay :exec
INSERT INTO leave_request_days (leave_request_id, work_date, minutes) VALUES (sqlc.arg(leave_request_id), sqlc.arg(work_date), sqlc.arg(minutes));

-- name: InsertLeaveVersion :exec
INSERT INTO leave_request_versions (leave_request_id, submit_no, leave_type, start_date, end_date, start_time, end_time,
    total_minutes, reason_code, reason_text, reviewer_id, manager_name_snapshot, manager_email_snapshot,
    policy_version_id, calculation_snapshot, created_by)
VALUES (sqlc.arg(leave_request_id), sqlc.arg(submit_no), sqlc.arg(leave_type), sqlc.arg(start_date), sqlc.arg(end_date),
    sqlc.arg(start_time), sqlc.arg(end_time), sqlc.arg(total_minutes), sqlc.narg(reason_code), sqlc.arg(reason_text),
    sqlc.arg(reviewer_id), sqlc.arg(manager_name), sqlc.arg(manager_email), sqlc.arg(policy_version_id),
    sqlc.arg(calculation_snapshot), sqlc.arg(created_by));

-- name: InsertIdempotencyKey :exec
INSERT INTO idempotency_keys (actor_id, scope, key_hash, request_hash, resource_id, response_status, expires_at)
VALUES (sqlc.arg(actor_id), sqlc.arg(scope), sqlc.arg(key_hash), sqlc.arg(request_hash), sqlc.arg(resource_id), 200, sqlc.arg(expires_at));

-- name: GetIdempotencyKey :one
SELECT request_hash, resource_id FROM idempotency_keys
WHERE actor_id = sqlc.arg(actor_id) AND scope = sqlc.arg(scope) AND key_hash = sqlc.arg(key_hash);

-- name: DeleteExpiredIdempotencyKeys :execrows
DELETE FROM idempotency_keys WHERE expires_at <= sqlc.arg(now);

-- name: GetCalendarYearStatus :one
SELECT status FROM calendar_years WHERE year = sqlc.arg(year);
