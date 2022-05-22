CREATE TABLE vertexes (
  run_id TEXT NOT NULL,
  digest TEXT NOT NULL,
  name TEXT NOT NULL,
  cached INTEGER NOT NULL,
  start_time INTEGER NULL,
  end_time INTEGER NULL,
  error TEXT NULL,
  PRIMARY KEY (run_id, digest),

  FOREIGN KEY (run_id) REFERENCES runs (id) ON DELETE CASCADE
);

CREATE INDEX idx_vertexes_run_id ON vertexes (run_id);

CREATE TABLE vertex_edges (
  source_digest TEXT NOT NULL,
  target_digest TEXT NOT NULL,
  PRIMARY KEY (source_digest, target_digest),
  FOREIGN KEY (target_digest) REFERENCES thunks (digest) ON DELETE CASCADE
);

CREATE INDEX idx_vertex_edges_target_digest ON vertex_edges (target_digest);
CREATE INDEX idx_vertex_edges_source_digest ON vertex_edges (source_digest);
