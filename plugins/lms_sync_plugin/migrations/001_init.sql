CREATE TABLE IF NOT EXISTS lms_sync_runs (
    id TEXT PRIMARY KEY,
    event_name TEXT NOT NULL,
    hop INTEGER NOT NULL DEFAULT 1,
    group_id TEXT NOT NULL,
    academic_year_id TEXT NOT NULL,
    roster_count INTEGER NOT NULL DEFAULT 0,
    target_url TEXT NOT NULL,
    http_status INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL CHECK (status IN ('SYNCED', 'FAILED')),
    external_sync_id TEXT NOT NULL DEFAULT '',
    caller_type TEXT NOT NULL DEFAULT 'SYSTEM',
    caller_staff_id TEXT NOT NULL DEFAULT '',
    synced_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);
CREATE INDEX IF NOT EXISTS idx_lms_sync_runs_group ON lms_sync_runs(group_id, synced_at);

CREATE TABLE IF NOT EXISTS lms_sync_pupil_entries (
    id TEXT PRIMARY KEY,
    sync_run_id TEXT NOT NULL REFERENCES lms_sync_runs(id) ON DELETE CASCADE,
    pupil_id TEXT NOT NULL,
    membership_id TEXT NOT NULL DEFAULT '',
    effective_from TEXT NOT NULL,
    effective_to TEXT NOT NULL DEFAULT '',
    sync_notes TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);
CREATE INDEX IF NOT EXISTS idx_lms_sync_pupil_entries_pupil ON lms_sync_pupil_entries(pupil_id);
CREATE INDEX IF NOT EXISTS idx_lms_sync_pupil_entries_run ON lms_sync_pupil_entries(sync_run_id);

