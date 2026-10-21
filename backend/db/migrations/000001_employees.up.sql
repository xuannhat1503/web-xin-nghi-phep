-- Employees. Leader is not a role: review rights come from being someone's manager / assigned reviewer.
-- Self-manager is rejected in the application (MySQL forbids CHECK on AUTO_INCREMENT columns).
CREATE TABLE employees (
    id                BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    google_subject_id VARCHAR(255)    NOT NULL,
    email             VARCHAR(320)    NOT NULL,
    full_name         VARCHAR(200)    NOT NULL,
    role              ENUM('ADMIN','EMPLOYEE') NOT NULL DEFAULT 'EMPLOYEE',
    manager_id        BIGINT UNSIGNED NULL,
    active            BOOLEAN         NOT NULL DEFAULT FALSE,
    version           INT UNSIGNED    NOT NULL DEFAULT 1,
    last_login_at     DATETIME(3)     NULL,
    created_at        DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at        DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uq_employees_google_subject (google_subject_id),
    KEY idx_employees_manager (manager_id),
    KEY idx_employees_email (email),
    KEY idx_employees_active_role (active, role),
    CONSTRAINT fk_employees_manager FOREIGN KEY (manager_id) REFERENCES employees (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Append-only audit of activation / role changes.
CREATE TABLE employee_events (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    employee_id BIGINT UNSIGNED NOT NULL,
    actor_id    BIGINT UNSIGNED NULL,
    action      VARCHAR(40)     NOT NULL,
    before_data JSON            NULL,
    after_data  JSON            NULL,
    created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    KEY idx_employee_events_employee (employee_id, created_at),
    CONSTRAINT fk_employee_events_employee FOREIGN KEY (employee_id) REFERENCES employees (id) ON DELETE CASCADE,
    CONSTRAINT fk_employee_events_actor FOREIGN KEY (actor_id) REFERENCES employees (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Append-only history of manager assignments.
CREATE TABLE manager_assignment_events (
    id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    employee_id    BIGINT UNSIGNED NOT NULL,
    old_manager_id BIGINT UNSIGNED NULL,
    new_manager_id BIGINT UNSIGNED NULL,
    changed_by     BIGINT UNSIGNED NULL,
    reason         VARCHAR(500)    NOT NULL DEFAULT '',
    created_at     DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    KEY idx_manager_events_employee (employee_id, created_at),
    CONSTRAINT fk_manager_events_employee FOREIGN KEY (employee_id) REFERENCES employees (id) ON DELETE CASCADE,
    CONSTRAINT fk_manager_events_old FOREIGN KEY (old_manager_id) REFERENCES employees (id),
    CONSTRAINT fk_manager_events_new FOREIGN KEY (new_manager_id) REFERENCES employees (id),
    CONSTRAINT fk_manager_events_actor FOREIGN KEY (changed_by) REFERENCES employees (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
