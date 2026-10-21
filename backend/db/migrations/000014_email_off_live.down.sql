UPDATE worker_status SET name = 'mock_email_outbox' WHERE name = 'email_outbox';
UPDATE worker_status SET name = 'mock_email_retention' WHERE name = 'email_log_retention';
ALTER TABLE company_settings
    MODIFY email_mode ENUM('MOCK', 'TEST_EMAIL', 'LIVE', 'OFF') NOT NULL DEFAULT 'MOCK',
    ADD COLUMN test_email_to VARCHAR(320) NOT NULL DEFAULT '' AFTER email_mode,
    ADD COLUMN test_email_cc VARCHAR(1000) NOT NULL DEFAULT '' AFTER test_email_to;
UPDATE company_settings SET email_mode = 'MOCK' WHERE email_mode = 'OFF';
ALTER TABLE company_settings MODIFY email_mode ENUM('MOCK', 'TEST_EMAIL', 'LIVE') NOT NULL DEFAULT 'MOCK';
