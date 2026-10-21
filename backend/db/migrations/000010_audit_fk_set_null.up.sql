-- Audit rows must survive if an employee row is ever removed by a DBA (the app never deletes employees):
-- references to people in audit tables become NULL instead of blocking the delete.
-- FK actions do not fire triggers, so the append-only triggers stay intact.
ALTER TABLE manager_assignment_events
    DROP FOREIGN KEY fk_manager_events_old,
    DROP FOREIGN KEY fk_manager_events_new,
    DROP FOREIGN KEY fk_manager_events_actor;
ALTER TABLE manager_assignment_events
    ADD CONSTRAINT fk_manager_events_old FOREIGN KEY (old_manager_id) REFERENCES employees (id) ON DELETE SET NULL,
    ADD CONSTRAINT fk_manager_events_new FOREIGN KEY (new_manager_id) REFERENCES employees (id) ON DELETE SET NULL,
    ADD CONSTRAINT fk_manager_events_actor FOREIGN KEY (changed_by) REFERENCES employees (id) ON DELETE SET NULL;

ALTER TABLE employee_events DROP FOREIGN KEY fk_employee_events_actor;
ALTER TABLE employee_events
    ADD CONSTRAINT fk_employee_events_actor FOREIGN KEY (actor_id) REFERENCES employees (id) ON DELETE SET NULL;

ALTER TABLE calendar_day_events DROP FOREIGN KEY fk_calendar_day_events_actor;
ALTER TABLE calendar_day_events
    ADD CONSTRAINT fk_calendar_day_events_actor FOREIGN KEY (actor_id) REFERENCES employees (id) ON DELETE SET NULL;

ALTER TABLE settings_events DROP FOREIGN KEY fk_settings_events_actor;
ALTER TABLE settings_events
    ADD CONSTRAINT fk_settings_events_actor FOREIGN KEY (actor_id) REFERENCES employees (id) ON DELETE SET NULL;

ALTER TABLE calendar_days
    DROP FOREIGN KEY fk_calendar_days_confirmer,
    DROP FOREIGN KEY fk_calendar_days_creator,
    DROP FOREIGN KEY fk_calendar_days_updater,
    DROP FOREIGN KEY fk_calendar_days_deleter;
ALTER TABLE calendar_days
    ADD CONSTRAINT fk_calendar_days_confirmer FOREIGN KEY (confirmed_by) REFERENCES employees (id) ON DELETE SET NULL,
    ADD CONSTRAINT fk_calendar_days_creator FOREIGN KEY (created_by) REFERENCES employees (id) ON DELETE SET NULL,
    ADD CONSTRAINT fk_calendar_days_updater FOREIGN KEY (updated_by) REFERENCES employees (id) ON DELETE SET NULL,
    ADD CONSTRAINT fk_calendar_days_deleter FOREIGN KEY (deleted_by) REFERENCES employees (id) ON DELETE SET NULL;

ALTER TABLE calendar_years DROP FOREIGN KEY fk_calendar_years_confirmer;
ALTER TABLE calendar_years
    ADD CONSTRAINT fk_calendar_years_confirmer FOREIGN KEY (confirmed_by) REFERENCES employees (id) ON DELETE SET NULL;
