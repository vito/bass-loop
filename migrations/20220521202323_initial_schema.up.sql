CREATE TABLE thunks (
  sha256 TEXT NOT NULL PRIMARY KEY,
  sensitive INTEGER NOT NULL DEFAULT FALSE
);

CREATE TABLE thunk_inputs (
  target_sha256 TEXT NOT NULL,
  source_sha256 TEXT NOT NULL,
  PRIMARY KEY (target_sha256, source_sha256)
);

CREATE TABLE thunk_runs (
  id TEXT NOT NULL PRIMARY KEY,
  thunk_sha256 TEXT NOT NULL,
  start_time INTEGER NOT NULL,
  end_time INTEGER NULL,
  succeeded INTEGER NULL,
  FOREIGN KEY (thunk_sha256)
    REFERENCES thunks (sha256)
      ON DELETE CASCADE
      ON UPDATE NO ACTION
);

CREATE INDEX idx_thunk_runs_sha256
ON thunk_runs (thunk_sha256);
