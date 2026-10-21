-- name: EnsureCalendarYear :exec
INSERT INTO calendar_years (year) VALUES (sqlc.arg(year))
ON DUPLICATE KEY UPDATE year = year;

-- name: LockCalendarYear :one
SELECT year, status, version FROM calendar_years WHERE year = sqlc.arg(year) FOR UPDATE;

-- name: GetCalendarDayBySeedKey :one
SELECT id, deleted_at, admin_modified FROM calendar_days WHERE seed_key = sqlc.arg(seed_key);

-- name: GetActiveCalendarDayByDate :one
SELECT id, name, seed_key FROM calendar_days WHERE active_date = sqlc.arg(day_date);

-- name: InsertSeedCalendarDay :execlastid
INSERT INTO calendar_days (day_date, name, kind, status, source, note, seed_key)
VALUES (sqlc.arg(day_date), sqlc.arg(name), sqlc.arg(kind), 'DRAFT', sqlc.arg(source), sqlc.arg(note), sqlc.arg(seed_key));

-- name: InsertCalendarDayEvent :exec
INSERT INTO calendar_day_events (year, calendar_day_id, action, actor_id, before_data, after_data)
VALUES (sqlc.arg(year), sqlc.narg(calendar_day_id), sqlc.arg(action), sqlc.narg(actor_id), sqlc.narg(before_data), sqlc.narg(after_data));

-- name: ListCalendarDaysByYear :many
SELECT id, day_date, name, kind, status, start_time, end_time, source, note, seed_key, admin_modified, version
FROM calendar_days
WHERE year = sqlc.arg(year) AND deleted_at IS NULL
ORDER BY day_date;

-- name: ListCalendarYears :many
SELECT y.year, y.status, y.version, y.confirmed_at,
       CAST(COUNT(d.id) AS SIGNED) AS total_days,
       CAST(COALESCE(SUM(d.status = 'DRAFT'), 0) AS SIGNED) AS draft_days
FROM calendar_years y
LEFT JOIN calendar_days d ON d.year = y.year AND d.deleted_at IS NULL
GROUP BY y.year, y.status, y.version, y.confirmed_at
ORDER BY y.year;

-- name: GetCalendarYear :one
SELECT y.year, y.status, y.version, y.confirmed_at, e.full_name AS confirmed_by_name
FROM calendar_years y LEFT JOIN employees e ON e.id = y.confirmed_by
WHERE y.year = sqlc.arg(year);

-- name: ListCalendarDaysAdmin :many
SELECT d.id, d.day_date, d.name, d.kind, d.status, d.start_time, d.end_time, d.source, d.note, d.seed_key,
       d.admin_modified, d.version, d.confirmed_at, e.full_name AS confirmed_by_name
FROM calendar_days d LEFT JOIN employees e ON e.id = d.confirmed_by
WHERE d.year = sqlc.arg(year) AND d.deleted_at IS NULL
ORDER BY d.day_date;

-- name: GetCalendarDayForUpdate :one
SELECT id, day_date, year, name, kind, status, start_time, end_time, source, note, version
FROM calendar_days WHERE id = sqlc.arg(id) AND deleted_at IS NULL FOR UPDATE;

-- name: InsertCalendarDay :execlastid
INSERT INTO calendar_days (day_date, name, kind, status, start_time, end_time, source, note, admin_modified, created_by, updated_by)
VALUES (sqlc.arg(day_date), sqlc.arg(name), sqlc.arg(kind), 'DRAFT', sqlc.narg(start_time), sqlc.narg(end_time),
        sqlc.arg(source), sqlc.arg(note), TRUE, sqlc.arg(actor_id), sqlc.arg(actor_id));

-- Any edit returns the day to DRAFT: a changed rule must be confirmed again.
-- name: UpdateCalendarDay :execrows
UPDATE calendar_days
SET day_date = sqlc.arg(day_date), name = sqlc.arg(name), kind = sqlc.arg(kind),
    start_time = sqlc.narg(start_time), end_time = sqlc.narg(end_time), source = sqlc.arg(source), note = sqlc.arg(note),
    status = 'DRAFT', confirmed_by = NULL, confirmed_at = NULL, admin_modified = TRUE,
    updated_by = sqlc.arg(actor_id), version = version + 1
WHERE id = sqlc.arg(id) AND version = sqlc.arg(version) AND deleted_at IS NULL;

-- name: ConfirmCalendarDay :execrows
UPDATE calendar_days
SET status = 'CONFIRMED', confirmed_by = sqlc.arg(actor_id), confirmed_at = sqlc.arg(now),
    updated_by = sqlc.arg(actor_id), version = version + 1
WHERE id = sqlc.arg(id) AND version = sqlc.arg(version) AND status = 'DRAFT' AND deleted_at IS NULL;

-- name: UnconfirmCalendarDay :execrows
UPDATE calendar_days
SET status = 'DRAFT', confirmed_by = NULL, confirmed_at = NULL, admin_modified = TRUE,
    updated_by = sqlc.arg(actor_id), version = version + 1
WHERE id = sqlc.arg(id) AND version = sqlc.arg(version) AND status = 'CONFIRMED' AND deleted_at IS NULL;

-- Only drafts can be removed (soft delete keeps the audit trail and blocks re-seeding).
-- name: SoftDeleteCalendarDay :execrows
UPDATE calendar_days
SET deleted_at = sqlc.arg(now), deleted_by = sqlc.arg(actor_id), admin_modified = TRUE, version = version + 1
WHERE id = sqlc.arg(id) AND version = sqlc.arg(version) AND status = 'DRAFT' AND deleted_at IS NULL;

-- name: ConfirmDraftDaysOfYear :execrows
UPDATE calendar_days
SET status = 'CONFIRMED', confirmed_by = sqlc.arg(actor_id), confirmed_at = sqlc.arg(now), version = version + 1
WHERE year = sqlc.arg(year) AND status = 'DRAFT' AND deleted_at IS NULL;

-- name: ConfirmCalendarYear :execrows
UPDATE calendar_years SET status = 'CONFIRMED', confirmed_by = sqlc.arg(actor_id), confirmed_at = sqlc.arg(now), version = version + 1
WHERE year = sqlc.arg(year) AND version = sqlc.arg(version);

-- name: ReopenCalendarYear :execrows
UPDATE calendar_years SET status = 'DRAFT', confirmed_by = NULL, confirmed_at = NULL, version = version + 1
WHERE year = sqlc.arg(year) AND status = 'CONFIRMED';

-- Flags (never recalculates) open and approved requests whose date range includes a changed calendar date.
-- name: FlagRequestsForCalendarChange :execrows
UPDATE leave_requests SET calendar_changed = TRUE
WHERE status IN ('PENDING', 'NEEDS_CHANGES', 'APPROVED')
  AND start_date <= sqlc.arg(day_date) AND end_date >= sqlc.arg(day_date);

-- name: CountRequestsCoveringDate :many
SELECT status, COUNT(*) AS n FROM leave_requests
WHERE status IN ('PENDING', 'NEEDS_CHANGES', 'APPROVED')
  AND start_date <= sqlc.arg(day_date) AND end_date >= sqlc.arg(day_date)
GROUP BY status;

-- name: CountRequestsInYear :many
SELECT status, COUNT(*) AS n FROM leave_requests
WHERE status IN ('PENDING', 'NEEDS_CHANGES', 'APPROVED')
  AND start_date <= sqlc.arg(year_end) AND end_date >= sqlc.arg(year_start)
GROUP BY status;

-- name: ListCalendarYearEvents :many
SELECT ev.id, ev.calendar_day_id, ev.action, ev.before_data, ev.after_data, ev.created_at, e.full_name AS actor_name
FROM calendar_day_events ev LEFT JOIN employees e ON e.id = ev.actor_id
WHERE ev.year = sqlc.arg(year)
ORDER BY ev.created_at DESC, ev.id DESC
LIMIT 200;

-- Employees and leaders only ever see CONFIRMED days.
-- name: ListConfirmedDays :many
SELECT day_date, name, kind, start_time, end_time FROM calendar_days
WHERE status = 'CONFIRMED' AND deleted_at IS NULL AND day_date BETWEEN sqlc.arg(from_date) AND sqlc.arg(to_date)
ORDER BY day_date;
