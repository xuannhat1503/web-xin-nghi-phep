-- The applicant name is per request (shown in its emails), editable while drafting; the app-wide display name
-- stays the Google profile name. Replaces employees.name_custom from 000015.
ALTER TABLE employees DROP COLUMN name_custom;
ALTER TABLE leave_requests ADD COLUMN employee_name VARCHAR(200) NULL AFTER employee_id;
ALTER TABLE leave_request_versions ADD COLUMN employee_name VARCHAR(200) NULL AFTER submit_no;
