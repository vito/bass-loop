DELETE FROM services;
DELETE FROM runtimes;

DROP INDEX service_user_runtime_service_idx;

ALTER TABLE runtimes DROP COLUMN priority;
ALTER TABLE runtimes ADD COLUMN driver TEXT NOT NULL
ALTER TABLE runtimes ADD COLUMN config BLOB NOT NULL;
