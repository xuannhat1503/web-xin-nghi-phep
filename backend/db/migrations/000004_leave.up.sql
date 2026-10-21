-- Per-year counter for request codes like LR-2026-000123 (allocated with SELECT ... FOR UPDATE).
CREATE TABLE request_code_sequences (
    year       SMALLINT UNSIGNED NOT NULL,
    next_value INT UNSIGNED      NOT NULL,
    PRIMARY KEY (year)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Drafts may be incomplete; anything past DRAFT must be fully specified (chk_leave_requests_submitted).
CREATE TABLE leave_requests (
    id                     BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT,
    request_code           VARCHAR(20)      NULL,
    employee_id            BIGINT UNSIGNED  NOT NULL,
    reviewer_id            BIGINT UNSIGNED  NULL,
    leave_type             ENUM('ANNUAL_LEAVE','UNPAID_LEAVE','SICK_LEAVE','OTHER') NULL,
    start_date             DATE             NULL,
    end_date               DATE             NULL,
    start_time             TIME             NULL,
    end_time               TIME             NULL,
    total_minutes          INT UNSIGNED     NOT NULL DEFAULT 0,
    reason_code            VARCHAR(40)      NULL,
    reason_text            VARCHAR(1000)    NOT NULL DEFAULT '',
    status                 ENUM('DRAFT','PENDING','NEEDS_CHANGES','APPROVED','REJECTED','CANCELLED') NOT NULL DEFAULT 'DRAFT',
    rejection_reason       VARCHAR(1000)    NULL,
    change_request_reason  VARCHAR(1000)    NULL,
    reviewed_at            DATETIME(3)      NULL,
    manager_name_snapshot  VARCHAR(200)     NULL,
    manager_email_snapshot VARCHAR(320)     NULL,
    policy_version_id      BIGINT UNSIGNED  NULL,
    calculation_snapshot   JSON             NULL,
    calendar_changed       BOOLEAN          NOT NULL DEFAULT FALSE,
    submit_count           SMALLINT UNSIGNED NOT NULL DEFAULT 0,
    version                INT UNSIGNED     NOT NULL DEFAULT 1,
    created_at             DATETIME(3)      NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    first_submitted_at     DATETIME(3)      NULL,
    submitted_at           DATETIME(3)      NULL,
    cancelled_at           DATETIME(3)      NULL,
    updated_at             DATETIME(3)      NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uq_leave_requests_code (request_code),
    KEY idx_leave_requests_employee_created (employee_id, created_at, id),
    KEY idx_leave_requests_employee_status_dates (employee_id, status, start_date, end_date),
    KEY idx_leave_requests_reviewer_status (reviewer_id, status, submitted_at),
    KEY idx_leave_requests_reviewer_created (reviewer_id, created_at, id),
    KEY idx_leave_requests_status_submitted (status, submitted_at),
    KEY idx_leave_requests_first_submitted (first_submitted_at),
    KEY idx_leave_requests_reviewed (reviewed_at),
    CONSTRAINT fk_leave_requests_employee FOREIGN KEY (employee_id) REFERENCES employees (id),
    CONSTRAINT fk_leave_requests_reviewer FOREIGN KEY (reviewer_id) REFERENCES employees (id),
    CONSTRAINT fk_leave_requests_policy FOREIGN KEY (policy_version_id) REFERENCES policy_versions (id),
    CONSTRAINT chk_leave_requests_dates CHECK (start_date IS NULL OR end_date IS NULL OR start_date <= end_date),
    CONSTRAINT chk_leave_requests_submitted CHECK (
        status = 'DRAFT' OR (
            request_code IS NOT NULL AND reviewer_id IS NOT NULL AND leave_type IS NOT NULL
            AND start_date IS NOT NULL AND end_date IS NOT NULL AND start_time IS NOT NULL AND end_time IS NOT NULL
            AND total_minutes > 0 AND policy_version_id IS NOT NULL AND first_submitted_at IS NOT NULL
        )
    )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Minutes per calendar date, written at submit/resubmit. Drives month filters and analytics allocation.
CREATE TABLE leave_request_days (
    leave_request_id BIGINT UNSIGNED   NOT NULL,
    work_date        DATE              NOT NULL,
    minutes          SMALLINT UNSIGNED NOT NULL,
    PRIMARY KEY (leave_request_id, work_date),
    KEY idx_leave_request_days_date (work_date, leave_request_id),
    CONSTRAINT fk_leave_request_days_request FOREIGN KEY (leave_request_id) REFERENCES leave_requests (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Immutable snapshot of each submission (submit_no 1 = first submit, 2+ = resubmits).
CREATE TABLE leave_request_versions (
    id                     BIGINT UNSIGNED   NOT NULL AUTO_INCREMENT,
    leave_request_id       BIGINT UNSIGNED   NOT NULL,
    submit_no              SMALLINT UNSIGNED NOT NULL,
    leave_type             ENUM('ANNUAL_LEAVE','UNPAID_LEAVE','SICK_LEAVE','OTHER') NOT NULL,
    start_date             DATE              NOT NULL,
    end_date               DATE              NOT NULL,
    start_time             TIME              NOT NULL,
    end_time               TIME              NOT NULL,
    total_minutes          INT UNSIGNED      NOT NULL,
    reason_code            VARCHAR(40)       NULL,
    reason_text            VARCHAR(1000)     NOT NULL DEFAULT '',
    reviewer_id            BIGINT UNSIGNED   NOT NULL,
    manager_name_snapshot  VARCHAR(200)      NOT NULL,
    manager_email_snapshot VARCHAR(320)      NOT NULL,
    policy_version_id      BIGINT UNSIGNED   NOT NULL,
    calculation_snapshot   JSON              NOT NULL,
    created_by             BIGINT UNSIGNED   NOT NULL,
    created_at             DATETIME(3)       NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uq_leave_request_versions (leave_request_id, submit_no),
    CONSTRAINT fk_leave_request_versions_request FOREIGN KEY (leave_request_id) REFERENCES leave_requests (id) ON DELETE CASCADE,
    CONSTRAINT fk_leave_request_versions_reviewer FOREIGN KEY (reviewer_id) REFERENCES employees (id),
    CONSTRAINT fk_leave_request_versions_policy FOREIGN KEY (policy_version_id) REFERENCES policy_versions (id),
    CONSTRAINT fk_leave_request_versions_creator FOREIGN KEY (created_by) REFERENCES employees (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Append-only timeline. Cascade only matters for never-submitted drafts (submitted requests are never deleted).
CREATE TABLE leave_request_events (
    id               BIGINT UNSIGNED   NOT NULL AUTO_INCREMENT,
    leave_request_id BIGINT UNSIGNED   NOT NULL,
    action           VARCHAR(40)       NOT NULL,
    actor_id         BIGINT UNSIGNED   NULL,
    from_status      VARCHAR(20)       NULL,
    to_status        VARCHAR(20)       NULL,
    request_version  INT UNSIGNED      NOT NULL,
    submit_no        SMALLINT UNSIGNED NULL,
    comment          VARCHAR(1000)     NULL,
    metadata         JSON              NULL,
    created_at       DATETIME(3)       NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    KEY idx_leave_request_events_request (leave_request_id, created_at, id),
    KEY idx_leave_request_events_action (action, created_at),
    CONSTRAINT fk_leave_request_events_request FOREIGN KEY (leave_request_id) REFERENCES leave_requests (id) ON DELETE CASCADE,
    CONSTRAINT fk_leave_request_events_actor FOREIGN KEY (actor_id) REFERENCES employees (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
