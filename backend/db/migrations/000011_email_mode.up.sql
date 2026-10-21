-- Email delivery mode, set by an Admin. MOCK never sends. TEST_EMAIL sends every message ONLY to the
-- configured test inbox (plus test CC), never to the real recipients. SMTP credentials stay in .env.
ALTER TABLE company_settings
    ADD COLUMN email_mode ENUM('MOCK', 'TEST_EMAIL') NOT NULL DEFAULT 'MOCK' AFTER finance_cc_emails,
    ADD COLUMN test_email_to VARCHAR(320) NOT NULL DEFAULT 'xnnht1503@gmail.com' AFTER email_mode,
    ADD COLUMN test_email_cc VARCHAR(1000) NOT NULL DEFAULT 'longnhat@gmail.com' AFTER test_email_to;

-- Which messages actually went out over SMTP (prevents a second send when a task is retried).
ALTER TABLE mock_emails
    ADD COLUMN delivered_to VARCHAR(1000) NULL AFTER transport,
    ADD COLUMN delivered_at DATETIME(3) NULL AFTER delivered_to;
