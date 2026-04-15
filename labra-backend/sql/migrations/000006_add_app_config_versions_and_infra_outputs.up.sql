CREATE TABLE IF NOT EXISTS app_config_versions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  app_id INTEGER NOT NULL,
  user_id INTEGER NOT NULL,
  source TEXT NOT NULL,
  config_json TEXT NOT NULL,
  created_at INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE INDEX IF NOT EXISTS idx_app_config_versions_app_created
  ON app_config_versions(app_id, created_at DESC);

CREATE TABLE IF NOT EXISTS app_infra_outputs (
  app_id INTEGER PRIMARY KEY,
  user_id INTEGER NOT NULL,
  bucket_name TEXT NOT NULL,
  distribution_id TEXT NOT NULL,
  site_url TEXT,
  updated_at INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE INDEX IF NOT EXISTS idx_app_infra_outputs_user
  ON app_infra_outputs(user_id, updated_at DESC);
