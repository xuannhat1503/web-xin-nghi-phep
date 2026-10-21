-- Append-only audit tables: block UPDATE/DELETE at the database level.
-- (MySQL does not fire triggers for FK cascades, so deleting a never-submitted draft still cascades its events.)
-- Kept in a separate migration because sqlc's parser does not support CREATE TRIGGER.

CREATE TRIGGER trg_employee_events_no_update BEFORE UPDATE ON employee_events
FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'employee_events is append-only';
CREATE TRIGGER trg_employee_events_no_delete BEFORE DELETE ON employee_events
FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'employee_events is append-only';

CREATE TRIGGER trg_manager_events_no_update BEFORE UPDATE ON manager_assignment_events
FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'manager_assignment_events is append-only';
CREATE TRIGGER trg_manager_events_no_delete BEFORE DELETE ON manager_assignment_events
FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'manager_assignment_events is append-only';

CREATE TRIGGER trg_settings_events_no_update BEFORE UPDATE ON settings_events
FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'settings_events is append-only';
CREATE TRIGGER trg_settings_events_no_delete BEFORE DELETE ON settings_events
FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'settings_events is append-only';

CREATE TRIGGER trg_calendar_day_events_no_update BEFORE UPDATE ON calendar_day_events
FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'calendar_day_events is append-only';
CREATE TRIGGER trg_calendar_day_events_no_delete BEFORE DELETE ON calendar_day_events
FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'calendar_day_events is append-only';

CREATE TRIGGER trg_leave_request_events_no_update BEFORE UPDATE ON leave_request_events
FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'leave_request_events is append-only';
CREATE TRIGGER trg_leave_request_events_no_delete BEFORE DELETE ON leave_request_events
FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'leave_request_events is append-only';

CREATE TRIGGER trg_leave_request_versions_no_update BEFORE UPDATE ON leave_request_versions
FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'leave_request_versions is append-only';
CREATE TRIGGER trg_leave_request_versions_no_delete BEFORE DELETE ON leave_request_versions
FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'leave_request_versions is append-only';

CREATE TRIGGER trg_policy_versions_no_update BEFORE UPDATE ON policy_versions
FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'policy_versions is immutable';
CREATE TRIGGER trg_policy_versions_no_delete BEFORE DELETE ON policy_versions
FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'policy_versions is immutable';

CREATE TRIGGER trg_leave_type_policies_no_update BEFORE UPDATE ON leave_type_policies
FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'leave_type_policies is immutable';
CREATE TRIGGER trg_leave_type_policies_no_delete BEFORE DELETE ON leave_type_policies
FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'leave_type_policies is immutable';
