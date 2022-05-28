CREATE TEMPORARY TABLE runs_fkey AS
SELECT * FROM runs;

DROP TABLE runs;

-- same as initial migration
CREATE TABLE runs (
  id TEXT NOT NULL PRIMARY KEY,
  user_id TEXT NOT NULL,
  thunk_digest TEXT NOT NULL,
  start_time TIMESTAMP NOT NULL,
  end_time TIMESTAMP NULL,
  succeeded INTEGER NULL,
  FOREIGN KEY (thunk_digest) REFERENCES thunks (digest) ON DELETE CASCADE
);

INSERT INTO runs (id, user_id, thunk_digest, start_time, end_time, succeeded)
SELECT id, user_id, thunk_digest, start_time, end_time, succeeded
FROM runs_fkey;

-- same as initial migration
CREATE INDEX idx_thunk_runs_digest ON runs (thunk_digest);
