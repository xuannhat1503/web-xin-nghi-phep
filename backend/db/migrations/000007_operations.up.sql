-- Worker heartbeats for /admin/system.
CREATE TABLE worker_status (
    name            VARCHAR(60)  NOT NULL,
    last_run_at     DATETIME(3)  NULL,
    last_success_at DATETIME(3)  NULL,
    last_error      VARCHAR(500) NULL,
    updated_at      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Sanitized operational errors (no SQL text, stack traces, tokens or personal data).
CREATE TABLE operational_errors (
    id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    component  VARCHAR(60)     NOT NULL,
    code       VARCHAR(60)     NOT NULL,
    message    VARCHAR(500)    NOT NULL,
    created_at DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    KEY idx_operational_errors_created (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
