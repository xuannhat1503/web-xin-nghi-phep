-- In-app notifications. dedup_key makes retries and reminder scans idempotent.
CREATE TABLE notifications (
    id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    recipient_id BIGINT UNSIGNED NOT NULL,
    type         VARCHAR(40)     NOT NULL,
    reference_id BIGINT UNSIGNED NULL,
    title        VARCHAR(200)    NOT NULL,
    body         VARCHAR(1000)   NOT NULL DEFAULT '',
    link         VARCHAR(300)    NOT NULL DEFAULT '',
    dedup_key    VARCHAR(191)    NOT NULL,
    read_at      DATETIME(3)     NULL,
    created_at   DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uq_notifications_dedup (dedup_key),
    KEY idx_notifications_recipient (recipient_id, created_at, id),
    KEY idx_notifications_unread (recipient_id, read_at),
    CONSTRAINT fk_notifications_recipient FOREIGN KEY (recipient_id) REFERENCES employees (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Transactional outbox: written in the same transaction as the business change, processed by the worker.
CREATE TABLE notification_outbox (
    id              BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT,
    event_type      VARCHAR(40)      NOT NULL,
    aggregate_id    BIGINT UNSIGNED  NULL,
    payload         JSON             NOT NULL,
    status          ENUM('PENDING','PROCESSING','SENT','FAILED') NOT NULL DEFAULT 'PENDING',
    attempts        SMALLINT UNSIGNED NOT NULL DEFAULT 0,
    next_retry_at   DATETIME(3)      NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    locked_until    DATETIME(3)      NULL,
    last_error      VARCHAR(500)     NULL,
    idempotency_key VARCHAR(191)     NOT NULL,
    processed_at    DATETIME(3)      NULL,
    created_at      DATETIME(3)      NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at      DATETIME(3)      NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uq_notification_outbox_idem (idempotency_key),
    KEY idx_notification_outbox_due (status, next_retry_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Rendered Mock Emails. Nothing here is ever delivered.
CREATE TABLE mock_emails (
    id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    outbox_id    BIGINT UNSIGNED NOT NULL,
    email_type   VARCHAR(40)     NOT NULL,
    to_address   VARCHAR(320)    NOT NULL,
    cc_addresses VARCHAR(1000)   NOT NULL DEFAULT '',
    subject      VARCHAR(300)    NOT NULL,
    body_text    TEXT            NOT NULL,
    transport    VARCHAR(20)     NOT NULL DEFAULT 'MOCK',
    created_at   DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uq_mock_emails_outbox (outbox_id),
    KEY idx_mock_emails_created (created_at),
    CONSTRAINT fk_mock_emails_outbox FOREIGN KEY (outbox_id) REFERENCES notification_outbox (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
