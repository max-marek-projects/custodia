-- Revert the length restriction on name and device_name back to TEXT.
-- Note: this does NOT shrink data, so any value longer than 255 chars
-- that was added while the constraint was active will remain unaffected.

ALTER TABLE secrets ALTER COLUMN name TYPE TEXT;

ALTER TABLE tokens ALTER COLUMN device_name TYPE TEXT;