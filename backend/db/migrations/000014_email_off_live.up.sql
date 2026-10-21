-- Two delivery modes only: OFF (emails are created and logged, never sent) and LIVE (sent to the real
-- recipients). The test-inbox mode and its TO/CC columns are removed.
ALTER TABLE company_settings MODIFY email_mode ENUM('MOCK', 'TEST_EMAIL', 'LIVE', 'OFF') NOT NULL DEFAULT 'OFF';
UPDATE company_settings SET email_mode = 'OFF' WHERE email_mode IN ('MOCK', 'TEST_EMAIL');
ALTER TABLE company_settings
    MODIFY email_mode ENUM('OFF', 'LIVE') NOT NULL DEFAULT 'OFF',
    DROP COLUMN test_email_to,
    DROP COLUMN test_email_cc;

-- Background jobs were renamed (no more "mock" wording); keep their history.
UPDATE worker_status SET name = 'email_outbox' WHERE name = 'mock_email_outbox';
UPDATE worker_status SET name = 'email_log_retention' WHERE name = 'mock_email_retention';
