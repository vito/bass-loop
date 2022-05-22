CREATE TABLE thunks (
  digest TEXT NOT NULL PRIMARY KEY,
  sensitive INTEGER NOT NULL DEFAULT FALSE
);

CREATE TABLE runs (
  thunk_digest TEXT NOT NULL,
  id TEXT NOT NULL PRIMARY KEY,
  start_time INTEGER NOT NULL,
  end_time INTEGER NULL,
  succeeded INTEGER NULL,
  FOREIGN KEY (thunk_digest) REFERENCES thunks (digest) ON DELETE CASCADE
);

CREATE INDEX idx_thunk_runs_digest ON runs (thunk_digest);
