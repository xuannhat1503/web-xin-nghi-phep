-- Promoting an account to ADMIN always activates it, including manual edits in phpMyAdmin.
-- (An ADMIN can still be deactivated later by changing `active` alone; the app protects the last Admin.)
CREATE TRIGGER trg_employees_admin_active_ins BEFORE INSERT ON employees
FOR EACH ROW SET NEW.active = IF(NEW.role = 'ADMIN', TRUE, NEW.active);

CREATE TRIGGER trg_employees_admin_active_upd BEFORE UPDATE ON employees
FOR EACH ROW SET NEW.active = IF(NEW.role = 'ADMIN' AND OLD.role <> 'ADMIN', TRUE, NEW.active);
