-- Immutable policy snapshots. Editing settings inserts a new row; leave requests keep the version they were calculated with.
CREATE TABLE policy_versions (
    id                        BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT,
    version_no                INT UNSIGNED     NOT NULL,
    timezone                  VARCHAR(64)      NOT NULL,
    work_start                TIME             NOT NULL,
    work_end                  TIME             NOT NULL,
    lunch_start               TIME             NOT NULL,
    lunch_end                 TIME             NOT NULL,
    -- bit 0 = Monday ... bit 6 = Sunday
    workdays_mask             TINYINT UNSIGNED NOT NULL,
    minutes_per_day           SMALLINT UNSIGNED AS (
        (TIME_TO_SEC(work_end) - TIME_TO_SEC(work_start) - (TIME_TO_SEC(lunch_end) - TIME_TO_SEC(lunch_start))) DIV 60
    ) STORED,
    min_leave_minutes         SMALLINT UNSIGNED NOT NULL,
    leave_unit_minutes        SMALLINT UNSIGNED NOT NULL,
    reminder_basis            ENUM('WORKING_HOURS','CLOCK_HOURS') NOT NULL,
    reminder_interval_minutes INT UNSIGNED     NOT NULL,
    reminder_max_count        TINYINT UNSIGNED NOT NULL,
    change_note               VARCHAR(500)     NOT NULL DEFAULT '',
    created_by                BIGINT UNSIGNED  NULL,
    created_at                DATETIME(3)      NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uq_policy_versions_no (version_no),
    CONSTRAINT fk_policy_versions_creator FOREIGN KEY (created_by) REFERENCES employees (id),
    CONSTRAINT chk_policy_work_hours CHECK (work_start < work_end),
    CONSTRAINT chk_policy_lunch CHECK (lunch_start <= lunch_end AND lunch_start >= work_start AND lunch_end <= work_end),
    CONSTRAINT chk_policy_workdays CHECK (workdays_mask BETWEEN 1 AND 127),
    CONSTRAINT chk_policy_units CHECK (leave_unit_minutes BETWEEN 1 AND 480 AND min_leave_minutes >= 1),
    CONSTRAINT chk_policy_reminder CHECK (reminder_interval_minutes >= 15 AND reminder_max_count BETWEEN 0 AND 20)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE leave_type_policies (
    policy_version_id   BIGINT UNSIGNED  NOT NULL,
    leave_type          ENUM('ANNUAL_LEAVE','UNPAID_LEAVE','SICK_LEAVE','OTHER') NOT NULL,
    advance_notice_days SMALLINT UNSIGNED NOT NULL DEFAULT 0,
    allow_past_dates    BOOLEAN          NOT NULL DEFAULT FALSE,
    max_past_days       SMALLINT UNSIGNED NOT NULL DEFAULT 0,
    reason_required     BOOLEAN          NOT NULL DEFAULT TRUE,
    PRIMARY KEY (policy_version_id, leave_type),
    CONSTRAINT fk_leave_type_policies_version FOREIGN KEY (policy_version_id) REFERENCES policy_versions (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Single-row pointer to the active policy plus non-calculation settings.
CREATE TABLE company_settings (
    id                        TINYINT UNSIGNED NOT NULL,
    current_policy_version_id BIGINT UNSIGNED  NOT NULL,
    -- Comma-separated. Shown in Mock Email previews only; never used for real delivery.
    finance_cc_emails         VARCHAR(1000)    NOT NULL DEFAULT '',
    version                   INT UNSIGNED     NOT NULL DEFAULT 1,
    updated_by                BIGINT UNSIGNED  NULL,
    updated_at                DATETIME(3)      NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    CONSTRAINT chk_company_settings_singleton CHECK (id = 1),
    CONSTRAINT fk_company_settings_policy FOREIGN KEY (current_policy_version_id) REFERENCES policy_versions (id),
    CONSTRAINT fk_company_settings_updater FOREIGN KEY (updated_by) REFERENCES employees (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Admin-managed predefined reasons; requires_text marks "Other" style entries.
CREATE TABLE leave_reasons (
    id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    code          VARCHAR(40)     NOT NULL,
    label         VARCHAR(200)    NOT NULL,
    requires_text BOOLEAN         NOT NULL DEFAULT FALSE,
    sort_order    SMALLINT        NOT NULL DEFAULT 0,
    active        BOOLEAN         NOT NULL DEFAULT TRUE,
    created_at    DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at    DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uq_leave_reasons_code (code),
    KEY idx_leave_reasons_active (active, sort_order)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Append-only audit for settings, policies and reasons.
CREATE TABLE settings_events (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    actor_id    BIGINT UNSIGNED NULL,
    action      VARCHAR(40)     NOT NULL,
    before_data JSON            NULL,
    after_data  JSON            NULL,
    created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    KEY idx_settings_events_created (created_at),
    CONSTRAINT fk_settings_events_actor FOREIGN KEY (actor_id) REFERENCES employees (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Initial defaults (confirmed 2026-10-19; Admin edits them in /admin/settings).
INSERT INTO policy_versions
    (id, version_no, timezone, work_start, work_end, lunch_start, lunch_end, workdays_mask,
     min_leave_minutes, leave_unit_minutes, reminder_basis, reminder_interval_minutes, reminder_max_count, change_note)
VALUES
    (1, 1, 'Asia/Ho_Chi_Minh', '08:30:00', '17:30:00', '12:00:00', '13:00:00', 31,
     30, 30, 'WORKING_HOURS', 480, 5, 'Initial default policy');

INSERT INTO leave_type_policies (policy_version_id, leave_type, advance_notice_days, allow_past_dates, max_past_days, reason_required) VALUES
    (1, 'ANNUAL_LEAVE', 1, FALSE, 0, TRUE),
    (1, 'UNPAID_LEAVE', 1, FALSE, 0, TRUE),
    (1, 'SICK_LEAVE',   0, TRUE,  3, TRUE),
    (1, 'OTHER',        0, FALSE, 0, TRUE);

INSERT INTO company_settings (id, current_policy_version_id) VALUES (1, 1);

INSERT INTO leave_reasons (code, label, requires_text, sort_order) VALUES
    ('PERSONAL',    'Personal matters',      FALSE, 10),
    ('FAMILY',      'Family matters',        FALSE, 20),
    ('ILLNESS',     'Illness',               FALSE, 30),
    ('MEDICAL',     'Medical appointment',   FALSE, 40),
    ('WEDDING',     'Wedding',               FALSE, 50),
    ('BEREAVEMENT', 'Bereavement',           FALSE, 60),
    ('TRAVEL',      'Travel / vacation',     FALSE, 70),
    ('OTHER',       'Other (please specify)', TRUE, 999);
