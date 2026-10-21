-- Admin employee management. Every query here is reachable only through ADMIN-authorized handlers.

-- name: CountEmployeesAdmin :one
SELECT COUNT(*) FROM employees e
WHERE (sqlc.arg(search) = '' OR e.full_name LIKE sqlc.arg(pattern) OR e.email LIKE sqlc.arg(pattern))
  AND (sqlc.arg(status) = 'all'
       OR (sqlc.arg(status) = 'active' AND e.active = TRUE)
       OR (sqlc.arg(status) = 'inactive' AND e.active = FALSE))
  AND (sqlc.arg(no_manager) = FALSE OR e.manager_id IS NULL);

-- name: ListEmployeesAdmin :many
SELECT e.id, e.full_name, e.email, e.role, e.active, e.manager_id, e.last_login_at, e.created_at, e.version,
       m.full_name AS manager_name, m.active AS manager_active
FROM employees e
LEFT JOIN employees m ON m.id = e.manager_id
WHERE (sqlc.arg(search) = '' OR e.full_name LIKE sqlc.arg(pattern) OR e.email LIKE sqlc.arg(pattern))
  AND (sqlc.arg(status) = 'all'
       OR (sqlc.arg(status) = 'active' AND e.active = TRUE)
       OR (sqlc.arg(status) = 'inactive' AND e.active = FALSE))
  AND (sqlc.arg(no_manager) = FALSE OR e.manager_id IS NULL)
ORDER BY e.full_name, e.id
LIMIT ? OFFSET ?;

-- name: EmployeeSummaryCounts :one
SELECT
    CAST(COALESCE(SUM(active = FALSE), 0) AS SIGNED) AS inactive,
    CAST(COALESCE(SUM(active = TRUE AND manager_id IS NULL), 0) AS SIGNED) AS active_without_manager,
    CAST(COUNT(*) AS SIGNED) AS total
FROM employees;

-- name: GetEmployeeAdmin :one
SELECT e.id, e.full_name, e.email, e.role, e.active, e.manager_id, e.last_login_at, e.created_at, e.updated_at, e.version,
       m.full_name AS manager_name, m.email AS manager_email, m.active AS manager_active,
       (SELECT COUNT(*) FROM employees r WHERE r.manager_id = e.id AND r.active = TRUE) AS active_reports
FROM employees e
LEFT JOIN employees m ON m.id = e.manager_id
WHERE e.id = sqlc.arg(id);

-- name: GetEmployeeForUpdate :one
SELECT id, role, active, manager_id, version FROM employees WHERE id = sqlc.arg(id) FOR UPDATE;

-- Locks every active admin row so concurrent demotions/deactivations are serialized (last-admin rule).
-- name: LockActiveAdmins :many
SELECT id FROM employees WHERE role = 'ADMIN' AND active = TRUE ORDER BY id FOR UPDATE;

-- name: UpdateEmployeeAccess :execrows
UPDATE employees SET active = sqlc.arg(active), role = sqlc.arg(role), version = version + 1
WHERE id = sqlc.arg(id) AND version = sqlc.arg(version);

-- name: UpdateEmployeeManager :execrows
UPDATE employees SET manager_id = sqlc.narg(manager_id), version = version + 1
WHERE id = sqlc.arg(id) AND version = sqlc.arg(version);

-- name: GetManagerOf :one
SELECT manager_id FROM employees WHERE id = sqlc.arg(id);

-- name: InsertManagerEvent :exec
INSERT INTO manager_assignment_events (employee_id, old_manager_id, new_manager_id, changed_by, reason)
VALUES (sqlc.arg(employee_id), sqlc.narg(old_manager_id), sqlc.narg(new_manager_id), sqlc.arg(changed_by), sqlc.arg(reason));

-- name: ListEmployeeEvents :many
SELECT ev.id, ev.action, ev.before_data, ev.after_data, ev.created_at, a.full_name AS actor_name
FROM employee_events ev
LEFT JOIN employees a ON a.id = ev.actor_id
WHERE ev.employee_id = sqlc.arg(employee_id)
ORDER BY ev.created_at DESC, ev.id DESC
LIMIT 100;

-- name: ListManagerEvents :many
SELECT ev.id, ev.reason, ev.created_at,
       o.full_name AS old_manager_name, n.full_name AS new_manager_name, a.full_name AS actor_name
FROM manager_assignment_events ev
LEFT JOIN employees o ON o.id = ev.old_manager_id
LEFT JOIN employees n ON n.id = ev.new_manager_id
LEFT JOIN employees a ON a.id = ev.changed_by
WHERE ev.employee_id = sqlc.arg(employee_id)
ORDER BY ev.created_at DESC, ev.id DESC
LIMIT 100;

-- Candidate managers for the admin picker: active employees only, never the employee themself.
-- name: SearchManagerCandidates :many
SELECT id, full_name, email FROM employees
WHERE active = TRUE AND id <> sqlc.arg(exclude_id)
  AND (sqlc.arg(search) = '' OR full_name LIKE sqlc.arg(pattern) OR email LIKE sqlc.arg(pattern))
ORDER BY full_name, id
LIMIT 20;
