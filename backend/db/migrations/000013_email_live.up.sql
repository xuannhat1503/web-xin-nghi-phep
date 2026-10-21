-- LIVE delivers to the real recipients (manager, employee, Finance CC). It can only be selected when the
-- server has EMAIL_LIVE_ENABLED=true; ALLOWED_EMAILS / ALLOWED_EMAIL_DOMAIN gate every recipient.
ALTER TABLE company_settings MODIFY email_mode ENUM('MOCK', 'TEST_EMAIL', 'LIVE') NOT NULL DEFAULT 'MOCK';
