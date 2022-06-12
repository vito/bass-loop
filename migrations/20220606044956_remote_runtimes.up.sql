DELETE FROM services;
DELETE FROM runtimes;

ALTER TABLE runtimes DROP COLUMN driver;
ALTER TABLE runtimes DROP COLUMN config;
ALTER TABLE runtimes ADD COLUMN priority INTEGER NOT NULL DEFAULT 0;

-- from here on 'services' actually contains GRPC services for the higher-level
-- runtime interface, rather than the underlying services like buildkitd.
--
-- the schema is kept as-is since it maps cleanly to how SSH forwarding
-- actually works, and still fits the generic 'service' terminology, but now
-- we'll have a well-known service name like 'runtime' to look up.
CREATE UNIQUE INDEX service_user_runtime_service_idx ON services (user_id, runtime_name, service);
