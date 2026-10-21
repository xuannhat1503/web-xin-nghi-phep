-- Server-side sessions. Only the SHA-256 of the opaque token is stored.
CREATE TABLE sessions (
    token_hash          BINARY(32)      NOT NULL,
    employee_id         BIGINT UNSIGNED NOT NULL,
    created_at          DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    last_seen_at        DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    idle_expires_at     DATETIME(3)     NOT NULL,
    absolute_expires_at DATETIME(3)     NOT NULL,
    revoked_at          DATETIME(3)     NULL,
    user_agent          VARCHAR(255)    NOT NULL DEFAULT '',
    PRIMARY KEY (token_hash),
    KEY idx_sessions_employee (employee_id),
    KEY idx_sessions_absolute_expiry (absolute_expires_at),
    CONSTRAINT fk_sessions_employee FOREIGN KEY (employee_id) REFERENCES employees (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Short-lived, single-use OIDC login attempts (state, nonce, PKCE verifier).
CREATE TABLE oauth_states (
    state_hash    BINARY(32)   NOT NULL,
    nonce         VARCHAR(64)  NOT NULL,
    code_verifier VARCHAR(128) NOT NULL,
    return_to     VARCHAR(300) NOT NULL DEFAULT '/',
    created_at    DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    expires_at    DATETIME(3)  NOT NULL,
    PRIMARY KEY (state_hash),
    KEY idx_oauth_states_expiry (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Idempotency keys for important writes (e.g. submit). request_hash detects key reuse with a different body.
CREATE TABLE idempotency_keys (
    id              BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT,
    actor_id        BIGINT UNSIGNED  NOT NULL,
    scope           VARCHAR(60)      NOT NULL,
    key_hash        BINARY(32)       NOT NULL,
    request_hash    BINARY(32)       NOT NULL,
    resource_id     BIGINT UNSIGNED  NULL,
    response_status SMALLINT UNSIGNED NULL,
    created_at      DATETIME(3)      NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    expires_at      DATETIME(3)      NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_idempotency_keys (actor_id, scope, key_hash),
    KEY idx_idempotency_keys_expiry (expires_at),
    CONSTRAINT fk_idempotency_keys_actor FOREIGN KEY (actor_id) REFERENCES employees (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
