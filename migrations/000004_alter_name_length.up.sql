-- Restrict name length for secrets and device names.
-- Both columns are part of unique indexes and displayed in the UI,
-- so unbounded TEXT is not appropriate.

ALTER TABLE secrets ALTER COLUMN name TYPE VARCHAR(255);

ALTER TABLE tokens ALTER COLUMN device_name TYPE VARCHAR(255);