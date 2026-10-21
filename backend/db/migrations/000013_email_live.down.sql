UPDATE company_settings SET email_mode = 'MOCK' WHERE email_mode = 'LIVE';
ALTER TABLE company_settings MODIFY email_mode ENUM('MOCK', 'TEST_EMAIL') NOT NULL DEFAULT 'MOCK';
