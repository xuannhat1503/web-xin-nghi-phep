-- name: GetCurrentWorkdaysMask :one
SELECT p.workdays_mask
FROM company_settings s
JOIN policy_versions p ON p.id = s.current_policy_version_id
WHERE s.id = 1;

-- name: GetCurrentPolicy :one
SELECT p.id, p.version_no, p.timezone, p.work_start, p.work_end, p.lunch_start, p.lunch_end, p.workdays_mask,
       p.minutes_per_day, p.min_leave_minutes, p.leave_unit_minutes, p.reminder_basis, p.reminder_interval_minutes,
       p.reminder_max_count, p.change_note, p.created_at,
       s.version AS settings_version, s.finance_cc_emails
FROM company_settings s
JOIN policy_versions p ON p.id = s.current_policy_version_id
WHERE s.id = 1;

-- name: ListLeaveTypePolicies :many
SELECT leave_type, advance_notice_days, allow_past_dates, max_past_days, reason_required
FROM leave_type_policies WHERE policy_version_id = sqlc.arg(policy_version_id) ORDER BY leave_type;

-- name: ListPolicyVersions :many
SELECT p.id, p.version_no, p.change_note, p.created_at, e.full_name AS created_by_name
FROM policy_versions p LEFT JOIN employees e ON e.id = p.created_by
ORDER BY p.version_no DESC LIMIT 50;

-- name: LockCompanySettings :one
SELECT version, current_policy_version_id, finance_cc_emails FROM company_settings WHERE id = 1 FOR UPDATE;

-- name: NextPolicyVersionNo :one
SELECT CAST(COALESCE(MAX(version_no), 0) + 1 AS UNSIGNED) FROM policy_versions;

-- name: InsertPolicyVersion :execlastid
INSERT INTO policy_versions (version_no, timezone, work_start, work_end, lunch_start, lunch_end, workdays_mask,
    min_leave_minutes, leave_unit_minutes, reminder_basis, reminder_interval_minutes, reminder_max_count, change_note, created_by)
VALUES (sqlc.arg(version_no), sqlc.arg(timezone), sqlc.arg(work_start), sqlc.arg(work_end), sqlc.arg(lunch_start), sqlc.arg(lunch_end),
    sqlc.arg(workdays_mask), sqlc.arg(min_leave_minutes), sqlc.arg(leave_unit_minutes), sqlc.arg(reminder_basis),
    sqlc.arg(reminder_interval_minutes), sqlc.arg(reminder_max_count), sqlc.arg(change_note), sqlc.arg(created_by));

-- name: InsertLeaveTypePolicy :exec
INSERT INTO leave_type_policies (policy_version_id, leave_type, advance_notice_days, allow_past_dates, max_past_days, reason_required)
VALUES (sqlc.arg(policy_version_id), sqlc.arg(leave_type), sqlc.arg(advance_notice_days), sqlc.arg(allow_past_dates), sqlc.arg(max_past_days), sqlc.arg(reason_required));

-- name: SetCurrentPolicy :execrows
UPDATE company_settings SET current_policy_version_id = sqlc.arg(policy_version_id), updated_by = sqlc.arg(updated_by), version = version + 1
WHERE id = 1 AND version = sqlc.arg(version);

-- name: SetFinanceCC :execrows
UPDATE company_settings SET finance_cc_emails = sqlc.arg(finance_cc_emails), updated_by = sqlc.arg(updated_by), version = version + 1
WHERE id = 1 AND version = sqlc.arg(version);

-- name: InsertSettingsEvent :exec
INSERT INTO settings_events (actor_id, action, before_data, after_data)
VALUES (sqlc.arg(actor_id), sqlc.arg(action), sqlc.narg(before_data), sqlc.narg(after_data));

-- name: ListLeaveReasons :many
SELECT id, code, label, label_en, requires_text, sort_order, active FROM leave_reasons ORDER BY sort_order, id;

-- name: GetLeaveReasonForUpdate :one
SELECT id, code, label, label_en, requires_text, sort_order, active FROM leave_reasons WHERE id = sqlc.arg(id) FOR UPDATE;

-- name: InsertLeaveReason :execlastid
INSERT INTO leave_reasons (code, label, label_en, requires_text, sort_order) VALUES (sqlc.arg(code), sqlc.arg(label), sqlc.arg(label_en), sqlc.arg(requires_text), sqlc.arg(sort_order));

-- name: UpdateLeaveReason :exec
UPDATE leave_reasons SET label = sqlc.arg(label), label_en = sqlc.arg(label_en), requires_text = sqlc.arg(requires_text), sort_order = sqlc.arg(sort_order), active = sqlc.arg(active)
WHERE id = sqlc.arg(id);

-- Requests that will be (re)validated against a new policy. Approved/rejected/cancelled ones are never recalculated.
-- name: CountOpenRequestsByStatus :many
SELECT status, COUNT(*) AS n FROM leave_requests
WHERE status IN ('DRAFT', 'PENDING', 'NEEDS_CHANGES') GROUP BY status;

-- name: DeleteLeaveReason :execrows
DELETE FROM leave_reasons WHERE id = sqlc.arg(id);

-- name: CountReasonUsage :one
SELECT COUNT(*) FROM leave_reasons x WHERE x.code = sqlc.arg(code) AND (
    EXISTS (SELECT 1 FROM leave_requests r WHERE r.reason_code = x.code)
    OR EXISTS (SELECT 1 FROM leave_request_versions v WHERE v.reason_code = x.code));

-- name: ListLeaveTypes :many
SELECT code, name_vi, name_en, sort_order, active FROM leave_types ORDER BY sort_order, code;

-- name: GetLeaveTypeForUpdate :one
SELECT code, name_vi, name_en, sort_order, active FROM leave_types WHERE code = sqlc.arg(code) FOR UPDATE;

-- name: InsertLeaveType :exec
INSERT INTO leave_types (code, name_vi, name_en, sort_order) VALUES (sqlc.arg(code), sqlc.arg(name_vi), sqlc.arg(name_en), sqlc.arg(sort_order));

-- name: UpdateLeaveType :exec
UPDATE leave_types SET name_vi = sqlc.arg(name_vi), name_en = sqlc.arg(name_en), sort_order = sqlc.arg(sort_order), active = sqlc.arg(active)
WHERE code = sqlc.arg(code);

-- name: DeleteLeaveType :execrows
DELETE FROM leave_types WHERE code = sqlc.arg(code);

-- name: CountLeaveTypeUsage :one
SELECT COUNT(*) FROM leave_types t WHERE t.code = sqlc.arg(code) AND (
    EXISTS (SELECT 1 FROM leave_requests r WHERE r.leave_type = t.code)
    OR EXISTS (SELECT 1 FROM leave_request_versions v WHERE v.leave_type = t.code));
