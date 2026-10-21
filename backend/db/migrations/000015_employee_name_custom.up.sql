-- An employee may correct the name Google returned; once corrected, later Google logins keep it.
ALTER TABLE employees ADD COLUMN name_custom BOOLEAN NOT NULL DEFAULT FALSE AFTER full_name;
