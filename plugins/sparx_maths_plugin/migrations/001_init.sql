CREATE TABLE IF NOT EXISTS sparx_vendor_config (
  config_key TEXT PRIMARY KEY,
  api_key TEXT NOT NULL,
  key_preview TEXT NOT NULL,
  sync_endpoint_url TEXT NOT NULL,
  updated_by_staff_id TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS sparx_homework_scores (
  id TEXT PRIMARY KEY,
  sync_batch_id TEXT NOT NULL,
  group_id TEXT NOT NULL,
  pupil_id TEXT NOT NULL,
  pupil_name TEXT NOT NULL,
  academic_year_id TEXT NOT NULL,
  homework_week TEXT NOT NULL,
  completion_rate INTEGER NOT NULL,
  xp_earned INTEGER NOT NULL,
  bookwork_accuracy INTEGER NOT NULL,
  teacher_note TEXT NOT NULL,
  synced_by_staff_id TEXT NOT NULL,
  synced_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_sparx_homework_scores_pupil_id
  ON sparx_homework_scores (pupil_id, synced_at);

CREATE INDEX IF NOT EXISTS idx_sparx_homework_scores_group_id
  ON sparx_homework_scores (group_id, homework_week);

CREATE TABLE IF NOT EXISTS sparx_sync_runs (
  id TEXT PRIMARY KEY,
  sync_batch_id TEXT NOT NULL,
  group_id TEXT NOT NULL,
  academic_year_id TEXT NOT NULL,
  reference_date TEXT NOT NULL,
  homework_week TEXT NOT NULL,
  roster_count INTEGER NOT NULL,
  persisted_scores_count INTEGER NOT NULL,
  target_url TEXT NOT NULL,
  http_status INTEGER NOT NULL,
  status TEXT NOT NULL,
  two_phase_pattern TEXT NOT NULL,
  synced_by_staff_id TEXT NOT NULL,
  synced_at TEXT NOT NULL
);
