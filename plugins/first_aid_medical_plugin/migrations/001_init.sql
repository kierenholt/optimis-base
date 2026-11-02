CREATE TABLE IF NOT EXISTS first_aid_treatments (
    id TEXT PRIMARY KEY,
    pupil_id TEXT NOT NULL,
    incident_date TEXT NOT NULL,
    incident_location TEXT NOT NULL,
    injury_summary TEXT NOT NULL,
    treatment_given TEXT NOT NULL,
    outcome TEXT NOT NULL,
    critical_alert_snapshot TEXT NOT NULL,
    notified_guardian_id TEXT NOT NULL,
    recorded_by_staff_id TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_first_aid_treatments_pupil
    ON first_aid_treatments(pupil_id);

CREATE TABLE IF NOT EXISTS first_aid_event_logs (
    id TEXT PRIMARY KEY,
    treatment_id TEXT NOT NULL REFERENCES first_aid_treatments(id) ON DELETE CASCADE,
    pupil_id TEXT NOT NULL,
    event_name TEXT NOT NULL,
    hop_count INTEGER NOT NULL DEFAULT 1,
    dispatch_status TEXT NOT NULL,
    dispatched_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_first_aid_event_logs_pupil
    ON first_aid_event_logs(pupil_id);

