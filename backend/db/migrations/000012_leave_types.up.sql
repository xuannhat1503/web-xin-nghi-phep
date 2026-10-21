-- Leave types become Admin-managed data instead of a fixed ENUM. Per-type rules stay in the versioned
-- leave_type_policies (a type without a row there uses the default rule until the policy is saved).
CREATE TABLE leave_types (
    code       VARCHAR(40)  NOT NULL,
    name_vi    VARCHAR(100) NOT NULL,
    name_en    VARCHAR(100) NOT NULL,
    sort_order SMALLINT     NOT NULL DEFAULT 0,
    active     BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (code),
    KEY idx_leave_types_active (active, sort_order)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT INTO leave_types (code, name_vi, name_en, sort_order) VALUES
    ('ANNUAL_LEAVE', 'Nghỉ phép năm',    'Annual Leave', 10),
    ('UNPAID_LEAVE', 'Nghỉ không lương', 'Unpaid Leave', 20),
    ('SICK_LEAVE',   'Nghỉ ốm',          'Sick Leave',   30),
    ('OTHER',        'Khác',             'Other',        999);

ALTER TABLE leave_type_policies MODIFY leave_type VARCHAR(40) NOT NULL;

ALTER TABLE leave_requests
    MODIFY leave_type VARCHAR(40) NULL,
    ADD CONSTRAINT fk_leave_requests_type FOREIGN KEY (leave_type) REFERENCES leave_types (code);

ALTER TABLE leave_request_versions
    MODIFY leave_type VARCHAR(40) NOT NULL,
    ADD CONSTRAINT fk_leave_request_versions_type FOREIGN KEY (leave_type) REFERENCES leave_types (code);

-- Reasons get an English label next to the (now Vietnamese) display label, so Admin edits show in both languages.
ALTER TABLE leave_reasons ADD COLUMN label_en VARCHAR(200) NOT NULL DEFAULT '' AFTER label;
UPDATE leave_reasons SET label_en = label;
UPDATE leave_reasons SET label = CASE code
    WHEN 'PERSONAL' THEN 'Việc cá nhân'
    WHEN 'FAMILY' THEN 'Việc gia đình'
    WHEN 'ILLNESS' THEN 'Ốm đau'
    WHEN 'MEDICAL' THEN 'Khám bệnh'
    WHEN 'WEDDING' THEN 'Cưới hỏi'
    WHEN 'BEREAVEMENT' THEN 'Tang lễ'
    WHEN 'TRAVEL' THEN 'Du lịch / nghỉ ngơi'
    WHEN 'OTHER' THEN 'Khác (ghi rõ)'
    ELSE label END
WHERE label_en IN ('Personal matters', 'Family matters', 'Illness', 'Medical appointment', 'Wedding',
                   'Bereavement', 'Travel / vacation', 'Other (please specify)');
