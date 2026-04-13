ALTER TABLE apps
  ADD COLUMN current_release_version_id INTEGER;

ALTER TABLE deployments
  ADD COLUMN release_version_id INTEGER;

CREATE TABLE IF NOT EXISTS release_versions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  app_id INTEGER NOT NULL,
  deployment_id INTEGER NOT NULL,
  version_number INTEGER NOT NULL,
  artifact_path TEXT NOT NULL,
  artifact_checksum TEXT,
  is_retained INTEGER NOT NULL DEFAULT 1,
  created_at INTEGER NOT NULL DEFAULT (unixepoch()),
  UNIQUE(app_id, version_number),
  UNIQUE(deployment_id)
);

CREATE INDEX IF NOT EXISTS idx_release_versions_app_created
  ON release_versions(app_id, created_at DESC);

CREATE TABLE IF NOT EXISTS rollback_events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  app_id INTEGER NOT NULL,
  user_id INTEGER NOT NULL,
  from_release_version_id INTEGER,
  to_release_version_id INTEGER NOT NULL,
  deployment_id INTEGER NOT NULL,
  reason TEXT,
  created_at INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE INDEX IF NOT EXISTS idx_rollback_events_app_created
  ON rollback_events(app_id, created_at DESC);

ALTER TABLE app_health_metrics
  ADD COLUMN total_duration_seconds INTEGER NOT NULL DEFAULT 0;

ALTER TABLE app_health_metrics
  ADD COLUMN latest_duration_seconds INTEGER NOT NULL DEFAULT 0;

ALTER TABLE app_health_metrics
  ADD COLUMN last_deploy_at INTEGER NOT NULL DEFAULT 0;

ALTER TABLE app_health_metrics
  ADD COLUMN rollback_count INTEGER NOT NULL DEFAULT 0;
