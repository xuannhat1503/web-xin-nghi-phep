-- Year-level confirmation. Submitting leave that touches a non-CONFIRMED year is rejected.
CREATE TABLE calendar_years (
    year         SMALLINT UNSIGNED NOT NULL,
    status       ENUM('DRAFT','CONFIRMED') NOT NULL DEFAULT 'DRAFT',
    version      INT UNSIGNED      NOT NULL DEFAULT 1,
    confirmed_by BIGINT UNSIGNED   NULL,
    confirmed_at DATETIME(3)       NULL,
    created_at   DATETIME(3)       NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at   DATETIME(3)       NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (year),
    CONSTRAINT fk_calendar_years_confirmer FOREIGN KEY (confirmed_by) REFERENCES employees (id),
    CONSTRAINT chk_calendar_years_range CHECK (year BETWEEN 2000 AND 2200)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- One effective rule per date: active_date is NULL for soft-deleted rows, so the UNIQUE key only covers live rows.
-- start_time/end_time set = partial-day company holiday (e.g. afternoon off); NULL = full day.
CREATE TABLE calendar_days (
    id             BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT,
    day_date       DATE             NOT NULL,
    year           SMALLINT UNSIGNED AS (YEAR(day_date)) STORED NOT NULL,
    name           VARCHAR(200)     NOT NULL,
    kind           ENUM('NATIONAL_HOLIDAY','COMPANY_HOLIDAY','COMPENSATORY_HOLIDAY','MAKEUP_WORKDAY') NOT NULL,
    status         ENUM('DRAFT','CONFIRMED') NOT NULL DEFAULT 'DRAFT',
    start_time     TIME             NULL,
    end_time       TIME             NULL,
    source         VARCHAR(255)     NOT NULL DEFAULT '',
    note           VARCHAR(1000)    NOT NULL DEFAULT '',
    seed_key       VARCHAR(100)     NULL,
    admin_modified BOOLEAN          NOT NULL DEFAULT FALSE,
    confirmed_by   BIGINT UNSIGNED  NULL,
    confirmed_at   DATETIME(3)      NULL,
    created_by     BIGINT UNSIGNED  NULL,
    updated_by     BIGINT UNSIGNED  NULL,
    deleted_by     BIGINT UNSIGNED  NULL,
    deleted_at     DATETIME(3)      NULL,
    version        INT UNSIGNED     NOT NULL DEFAULT 1,
    created_at     DATETIME(3)      NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at     DATETIME(3)      NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    active_date    DATE AS (IF(deleted_at IS NULL, day_date, NULL)) STORED,
    PRIMARY KEY (id),
    UNIQUE KEY uq_calendar_days_active_date (active_date),
    UNIQUE KEY uq_calendar_days_seed_key (seed_key),
    KEY idx_calendar_days_year (year, status, deleted_at),
    KEY idx_calendar_days_date (day_date),
    CONSTRAINT fk_calendar_days_year FOREIGN KEY (year) REFERENCES calendar_years (year),
    CONSTRAINT fk_calendar_days_confirmer FOREIGN KEY (confirmed_by) REFERENCES employees (id),
    CONSTRAINT fk_calendar_days_creator FOREIGN KEY (created_by) REFERENCES employees (id),
    CONSTRAINT fk_calendar_days_updater FOREIGN KEY (updated_by) REFERENCES employees (id),
    CONSTRAINT fk_calendar_days_deleter FOREIGN KEY (deleted_by) REFERENCES employees (id),
    CONSTRAINT chk_calendar_days_partial CHECK (
        (start_time IS NULL AND end_time IS NULL)
        OR (start_time IS NOT NULL AND end_time IS NOT NULL AND start_time < end_time AND kind = 'COMPANY_HOLIDAY')
    )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Append-only audit for days and year confirmation (calendar_day_id NULL = year-level event).
CREATE TABLE calendar_day_events (
    id              BIGINT UNSIGNED   NOT NULL AUTO_INCREMENT,
    year            SMALLINT UNSIGNED NOT NULL,
    calendar_day_id BIGINT UNSIGNED   NULL,
    action          VARCHAR(40)       NOT NULL,
    actor_id        BIGINT UNSIGNED   NULL,
    before_data     JSON              NULL,
    after_data      JSON              NULL,
    created_at      DATETIME(3)       NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    KEY idx_calendar_day_events_year (year, created_at),
    KEY idx_calendar_day_events_day (calendar_day_id),
    CONSTRAINT fk_calendar_day_events_year FOREIGN KEY (year) REFERENCES calendar_years (year) ON DELETE CASCADE,
    CONSTRAINT fk_calendar_day_events_day FOREIGN KEY (calendar_day_id) REFERENCES calendar_days (id) ON DELETE CASCADE,
    CONSTRAINT fk_calendar_day_events_actor FOREIGN KEY (actor_id) REFERENCES employees (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
