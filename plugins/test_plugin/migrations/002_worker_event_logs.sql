CREATE TABLE IF NOT EXISTS worker_event_logs (
    id TEXT PRIMARY KEY,
    pupil_id TEXT NOT NULL,
    event_name TEXT NOT NULL,
    hop INTEGER NOT NULL,
    caller_type TEXT NOT NULL,
    caller_staff_id TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_worker_event_logs_pupil_id ON worker_event_logs (pupil_id);
