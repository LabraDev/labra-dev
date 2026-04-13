PRAGMA foreign_keys=off;

DROP TABLE IF EXISTS rollback_events;
DROP TABLE IF EXISTS release_versions;

CREATE TABLE IF NOT EXISTS apps_phase5_phase7_down (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  name TEXT NOT NULL,
  repo_full_name TEXT NOT NULL,
  branch TEXT NOT NULL DEFAULT 'main',
  build_type TEXT NOT NULL DEFAULT 'static',
  output_dir TEXT NOT NULL DEFAULT 'dist',
  root_dir TEXT,
  site_url TEXT,
  auto_deploy_enabled INTEGER NOT NULL DEFAULT 1,
  created_at INTEGER NOT NULL DEFAULT (unixepoch()),
  updated_at INTEGER NOT NULL DEFAULT (unixepoch())
);

INSERT INTO apps_phase5_phase7_down (
  id, user_id, name, repo_full_name, branch, build_type, output_dir, root_dir, site_url, auto_deploy_enabled, created_at, updated_at
)
SELECT
  id, user_id, name, repo_full_name, branch, build_type, output_dir, root_dir, site_url, auto_deploy_enabled, created_at, updated_at
FROM apps;

DROP TABLE apps;
ALTER TABLE apps_phase5_phase7_down RENAME TO apps;

CREATE UNIQUE INDEX IF NOT EXISTS idx_apps_user_repo_branch
  ON apps(user_id, repo_full_name, branch);

CREATE INDEX IF NOT EXISTS idx_apps_repo
  ON apps(repo_full_name);

CREATE TABLE IF NOT EXISTS deployments_phase5_phase7_down (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  app_id INTEGER NOT NULL,
  user_id INTEGER NOT NULL,
  status TEXT NOT NULL,
  trigger_type TEXT NOT NULL,
  commit_sha TEXT,
  commit_message TEXT,
  commit_author TEXT,
  branch TEXT,
  site_url TEXT,
  failure_reason TEXT,
  correlation_id TEXT,
  created_at INTEGER NOT NULL DEFAULT (unixepoch()),
  updated_at INTEGER NOT NULL DEFAULT (unixepoch()),
  started_at INTEGER,
  finished_at INTEGER
);

INSERT INTO deployments_phase5_phase7_down (
  id, app_id, user_id, status, trigger_type, commit_sha, commit_message, commit_author, branch, site_url, failure_reason, correlation_id,
  created_at, updated_at, started_at, finished_at
)
SELECT
  id, app_id, user_id, status, trigger_type, commit_sha, commit_message, commit_author, branch, site_url, failure_reason, correlation_id,
  created_at, updated_at, started_at, finished_at
FROM deployments;

DROP TABLE deployments;
ALTER TABLE deployments_phase5_phase7_down RENAME TO deployments;

CREATE INDEX IF NOT EXISTS idx_deployments_app_created
  ON deployments(app_id, created_at DESC);

CREATE TABLE IF NOT EXISTS app_health_metrics_phase5_phase7_down (
  app_id INTEGER PRIMARY KEY,
  success_count INTEGER NOT NULL DEFAULT 0,
  failure_count INTEGER NOT NULL DEFAULT 0,
  last_success_at INTEGER NOT NULL DEFAULT 0,
  last_failure_at INTEGER NOT NULL DEFAULT 0,
  updated_at INTEGER NOT NULL DEFAULT (unixepoch())
);

INSERT INTO app_health_metrics_phase5_phase7_down (
  app_id, success_count, failure_count, last_success_at, last_failure_at, updated_at
)
SELECT
  app_id, success_count, failure_count, last_success_at, last_failure_at, updated_at
FROM app_health_metrics;

DROP TABLE app_health_metrics;
ALTER TABLE app_health_metrics_phase5_phase7_down RENAME TO app_health_metrics;

PRAGMA foreign_keys=on;
