-- Review workflow (Phase 5). Authorization is part of every WHERE clause.

-- name: LockLeaveRequest :one
SELECT id, request_code, employee_id, reviewer_id, status, version, submit_count
FROM leave_requests WHERE id = sqlc.arg(id) FOR UPDATE;

-- Only the current reviewer, never the owner, only while PENDING and at the version they saw.
-- name: DecideLeaveRequest :execrows
UPDATE leave_requests
SET status = sqlc.arg(status), rejection_reason = sqlc.narg(rejection_reason),
    change_request_reason = sqlc.narg(change_request_reason), reviewed_at = sqlc.narg(reviewed_at),
    version = version + 1
WHERE id = sqlc.arg(id) AND version = sqlc.arg(version) AND status = 'PENDING'
  AND reviewer_id = sqlc.arg(reviewer_id) AND employee_id <> sqlc.arg(not_employee_id);

-- name: CancelLeaveRequest :execrows
UPDATE leave_requests SET status = 'CANCELLED', cancelled_at = sqlc.arg(now), version = version + 1
WHERE id = sqlc.arg(id) AND employee_id = sqlc.arg(employee_id) AND version = sqlc.arg(version)
  AND status IN ('PENDING', 'NEEDS_CHANGES');

-- name: TransferLeaveReviewer :execrows
UPDATE leave_requests
SET reviewer_id = sqlc.arg(reviewer_id), manager_name_snapshot = sqlc.arg(manager_name),
    manager_email_snapshot = sqlc.arg(manager_email), version = version + 1
WHERE id = sqlc.arg(id) AND version = sqlc.arg(version) AND status = 'PENDING';

-- name: ListLeaveRequestEvents :many
SELECT ev.id, ev.action, ev.from_status, ev.to_status, ev.submit_no, ev.comment, ev.created_at, e.full_name AS actor_name
FROM leave_request_events ev LEFT JOIN employees e ON e.id = ev.actor_id
WHERE ev.leave_request_id = sqlc.arg(leave_request_id) AND ev.action <> 'DRAFT_SAVED'
ORDER BY ev.created_at, ev.id;

-- name: ListLeaveRequestVersions :many
SELECT submit_no, leave_type, start_date, end_date, start_time, end_time, total_minutes,
       reason_code, reason_text, manager_name_snapshot, created_at
FROM leave_request_versions WHERE leave_request_id = sqlc.arg(leave_request_id) ORDER BY submit_no;

-- Leader view: only requests currently assigned to the viewer, never their own.
-- name: GetReviewRequest :one
SELECT lr.id, lr.request_code, lr.employee_id, lr.leave_type, lr.start_date, lr.end_date, lr.start_time, lr.end_time,
       lr.total_minutes, lr.reason_code, lr.reason_text, lr.status, lr.rejection_reason, lr.change_request_reason,
       lr.reviewed_at, lr.calculation_snapshot, lr.calendar_changed, lr.submit_count, lr.version,
       lr.created_at, lr.first_submitted_at, lr.submitted_at, lr.cancelled_at,
       e.full_name AS employee_name, e.email AS employee_email, r.label AS reason_label
FROM leave_requests lr
JOIN employees e ON e.id = lr.employee_id
LEFT JOIN leave_reasons r ON r.code = lr.reason_code
WHERE lr.id = sqlc.arg(id) AND lr.reviewer_id = sqlc.arg(reviewer_id) AND lr.employee_id <> sqlc.arg(viewer_id)
  AND lr.status <> 'DRAFT';

-- name: HasAssignedReviews :one
SELECT EXISTS (
    SELECT 1 FROM leave_requests WHERE reviewer_id = sqlc.arg(reviewer_id) AND status IN ('PENDING', 'NEEDS_CHANGES')
) AS has_reviews;

-- name: InsertOperationalError :exec
INSERT INTO operational_errors (component, code, message) VALUES (sqlc.arg(component), sqlc.arg(code), sqlc.arg(message));
