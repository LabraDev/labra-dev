CREATE TABLE IF NOT EXISTS ai_request_logs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  deployment_id INTEGER NOT NULL,
  prompt_version TEXT NOT NULL,
  provider TEXT NOT NULL,
  model TEXT NOT NULL,
  input_redacted INTEGER NOT NULL DEFAULT 0,
  fallback_used INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL,
  input_excerpt TEXT,
  output_excerpt TEXT,
  created_at INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE INDEX IF NOT EXISTS idx_ai_request_logs_user_created
  ON ai_request_logs(user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_ai_request_logs_deployment_created
  ON ai_request_logs(deployment_id, created_at DESC);
