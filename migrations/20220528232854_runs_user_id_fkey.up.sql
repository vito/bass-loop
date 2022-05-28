CREATE TEMPORARY TABLE runs_nofkey AS
SELECT * FROM runs;

DROP TABLE runs;

CREATE TABLE runs (
  id TEXT NOT NULL PRIMARY KEY,
  user_id TEXT NOT NULL,
  thunk_digest TEXT NOT NULL,
  start_time TIMESTAMP NOT NULL,
  end_time TIMESTAMP NULL,
  succeeded INTEGER NULL,
  FOREIGN KEY (thunk_digest) REFERENCES thunks (digest) ON DELETE CASCADE,

  -- only difference from initial migration
  FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

INSERT INTO runs (id, user_id, thunk_digest, start_time, end_time, succeeded)
SELECT id, user_id, thunk_digest, start_time, end_time, succeeded
FROM runs_nofkey;

-- restore from initial migration
CREATE INDEX idx_thunk_runs_digest ON runs (thunk_digest);

-- add an index for user_id while we're at it
CREATE INDEX idx_thunk_runs_user_id ON runs (user_id);
