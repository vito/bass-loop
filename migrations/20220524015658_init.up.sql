-- thunks, which are run by users
CREATE TABLE thunks (
  -- a content-addressable identifier for the thunk
  digest TEXT NOT NULL PRIMARY KEY,

  -- the thunk itself
  json BLOB NOT NULL
);

-- runs of thunks by users
CREATE TABLE runs (
  -- an arbitrary ID, i.e. a GUID
  id TEXT NOT NULL PRIMARY KEY,

  -- the user running the thunk
  user_id TEXT NOT NULL,

  -- the thunk being run
  thunk_digest TEXT NOT NULL,

  -- the time the thunk was run
  start_time TIMESTAMP NOT NULL,

  -- the time the thunk finished
  end_time TIMESTAMP NULL,

  -- whether the thunk completed successfully
  succeeded INTEGER NULL,

  FOREIGN KEY (thunk_digest) REFERENCES thunks (digest) ON DELETE CASCADE
);

-- it's reasonable to want to list runs of a thunk
CREATE INDEX idx_thunk_runs_digest ON runs (thunk_digest);

-- nodes in the DAG that have their own status and output
--
-- NB: vertex logs are stored in a blobstore rather than SQLite.
CREATE TABLE vertexes (
  -- the run the vertex belongs to
  run_id TEXT NOT NULL,

  -- the vertex's content addressed ID
  digest TEXT NOT NULL,

  -- the vertex's human-friendly name
  name TEXT NOT NULL,

  -- whether the vertex was a cache hit
  cached INTEGER NOT NULL,

  -- the vertex's start time, if any
  start_time TIMESTAMP NULL,

  -- the vertex's end time, if any
  end_time TIMESTAMP NULL,

  -- the vertex's error, if any
  --
  -- e.g. command exited nonzero, other error
  error TEXT NULL,

  PRIMARY KEY (run_id, digest),
  FOREIGN KEY (run_id) REFERENCES runs (id) ON DELETE CASCADE
);

-- when viewing a run, you'll want to fetch all of its vertexes
CREATE INDEX idx_vertexes_run_id ON vertexes (run_id);

-- dependency relationships between vertexes
--
-- not currently used, but worth collecting for future use
CREATE TABLE vertex_edges (
  source_digest TEXT NOT NULL,
  target_digest TEXT NOT NULL,
  PRIMARY KEY (source_digest, target_digest),
  FOREIGN KEY (target_digest) REFERENCES thunks (digest) ON DELETE CASCADE
);

-- finding inputs/outputs of a vertex seems useful
CREATE INDEX idx_vertex_edges_target_digest ON vertex_edges (target_digest);
CREATE INDEX idx_vertex_edges_source_digest ON vertex_edges (source_digest);

-- GitHub users
CREATE TABLE users (
  id TEXT NOT NULL,
  login TEXT NOT NULL,
  PRIMARY KEY (id)
);

-- runtimes registered by a user
CREATE TABLE runtimes (
  -- the authenticated user registering the runtime
  user_id TEXT NOT NULL,

  -- a name used for tying the runtime to concurrently-forwarded services
  name TEXT NOT NULL,

  -- platform attributes used to select the runtime
  os TEXT NOT NULL,
  arch TEXT NOT NULL,

  -- which runtime driver to use (must be supported by bass-loop)
  driver TEXT NOT NULL, -- "buildkit"

  -- arbitrary JSON config to pass to the runtime
  config BLOB NOT NULL,

  -- timestamp after which this row is no longer used and can be garbage
  -- collected.
  expires_at TIMESTAMP NOT NULL,

  PRIMARY KEY (user_id, name),
  FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

-- need to find a user's runtimes when running their thunks
CREATE INDEX idx_runtimes_user_id ON runtimes (user_id);

-- services provided to runtimes
CREATE TABLE services (
  -- the authenticated user forwarding the service
  user_id TEXT NOT NULL,

  -- the runtime config this service is attached to
  runtime_name TEXT NOT NULL,

  -- service the address corresponds to
  --
  -- this will be autodetected as the base name of the socket path:
  --
  --   ssh -R /buildkitd:/run/buildkit/buildkitd.sock ...
  --   ssh -R buildkitd:0:/run/buildkit/buildkitd.sock ...
  service TEXT NOT NULL,

  -- address used to reach the forwarded server.
  --
  -- in a single-node installation, this may be a local Unix socket; in a
  -- multi-node installation, it may be a TCP/IP address reachable by peers.
  addr TEXT NOT NULL,

  PRIMARY KEY (runtime_name, service),
  FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

-- need to be able to list services available to a runtime
CREATE INDEX idx_services_runtime_name ON services (user_id, runtime_name);
