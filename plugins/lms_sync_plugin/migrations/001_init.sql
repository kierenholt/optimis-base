CREATE TABLE IF NOT EXISTS lms_sync_logs (
    id TEXT PRIMARY KEY,
    event_name TEXT NOT NULL,
    hop INTEGER NOT NULL DEFAULT 1,
    membership_id TEXT NOT NULL DEFAULT '',
    group_id TEXT NOT NULL,
    pupil_id TEXT NOT NULL,
    academic_year_id TEXT NOT NULL,
    effective_from TEXT NOT NULL,
    effective_to TEXT NOT NULL DEFAULT '',
    roster_count INTEGER NOT NULL DEFAULT 0,
    target_url TEXT NOT NULL,
    http_status INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL,
    external_sync_id TEXT NOT NULL DEFAULT '',
    caller_type TEXT NOT NULL DEFAULT 'SYSTEM',
    caller_staff_id TEXT NOT NULL DEFAULT '',
    sync_notes TEXT NOT NULL DEFAULT '',
    synced_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_lms_sync_logs_pupil ON lms_sync_logs(pupil_id);
CREATE INDEX IF NOT EXISTS idx_lms_sync_logs_group ON lms_sync_logs(group_id);
