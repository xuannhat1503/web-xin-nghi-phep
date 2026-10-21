-- Fails on purpose if Admin-created leave types are in use (they cannot fit the old ENUM).
UPDATE leave_reasons SET label = label_en WHERE label_en <> '';
ALTER TABLE leave_reasons DROP COLUMN label_en;

ALTER TABLE leave_request_versions
    DROP FOREIGN KEY fk_leave_request_versions_type,
    MODIFY leave_type ENUM('ANNUAL_LEAVE','UNPAID_LEAVE','SICK_LEAVE','OTHER') NOT NULL;
ALTER TABLE leave_requests
    DROP FOREIGN KEY fk_leave_requests_type,
    MODIFY leave_type ENUM('ANNUAL_LEAVE','UNPAID_LEAVE','SICK_LEAVE','OTHER') NULL;
ALTER TABLE leave_type_policies MODIFY leave_type ENUM('ANNUAL_LEAVE','UNPAID_LEAVE','SICK_LEAVE','OTHER') NOT NULL;

DROP TABLE leave_types;
