ALTER TABLE leave_request_versions DROP COLUMN employee_name;
ALTER TABLE leave_requests DROP COLUMN employee_name;
ALTER TABLE employees ADD COLUMN name_custom BOOLEAN NOT NULL DEFAULT FALSE AFTER full_name;
