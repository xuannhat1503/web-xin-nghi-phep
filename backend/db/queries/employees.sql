-- name: GetEmployeeBySubject :one
SELECT * FROM employees WHERE google_subject_id = sqlc.arg(google_subject_id);

-- name: GetEmployeeByID :one
SELECT * FROM employees WHERE id = sqlc.arg(id);

-- name: CreateEmployee :execlastid
INSERT INTO employees (google_subject_id, email, full_name, role, active, last_login_at)
VALUES (sqlc.arg(google_subject_id), sqlc.arg(email), sqlc.arg(full_name), sqlc.arg(role), sqlc.arg(active), sqlc.arg(last_login_at));

-- Profile fields always come from the verified ID token; role/active/manager are never touched here.
-- name: UpdateEmployeeLogin :exec
UPDATE employees
SET email = sqlc.arg(email), full_name = sqlc.arg(full_name), last_login_at = sqlc.arg(last_login_at)
WHERE id = sqlc.arg(id);

-- name: InsertEmployeeEvent :exec
INSERT INTO employee_events (employee_id, actor_id, action, before_data, after_data)
VALUES (sqlc.arg(employee_id), sqlc.narg(actor_id), sqlc.arg(action), sqlc.narg(before_data), sqlc.narg(after_data));

-- name: HasActiveDirectReports :one
SELECT EXISTS (
    SELECT 1 FROM employees WHERE manager_id = sqlc.arg(manager_id) AND active = TRUE
) AS has_reports;

-- name: EmailInUse :one
SELECT EXISTS (SELECT 1 FROM employees WHERE email = sqlc.arg(email)) AS in_use;
