CREATE TABLE IF NOT EXISTS first_aid_treatments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    pupil_id TEXT NOT NULL,
    pupil_name TEXT NOT NULL,
    nc_year_group TEXT NOT NULL,
    incident_date TEXT NOT NULL,
    incident_location TEXT NOT NULL,
    injury_summary TEXT NOT NULL,
    treatment_given TEXT NOT NULL,
    outcome TEXT NOT NULL,
    critical_alert_snapshot TEXT NOT NULL,
    notified_guardian_id TEXT NOT NULL,
    notified_guardian_name TEXT NOT NULL,
    recorded_by_staff_id TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_first_aid_treatments_pupil
    ON first_aid_treatments(pupil_id);

CREATE TABLE IF NOT EXISTS first_aid_event_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    treatment_id INTEGER NOT NULL,
    pupil_id TEXT NOT NULL,
    event_name TEXT NOT NULL,
    hop_count INTEGER NOT NULL DEFAULT 1,
    outcome TEXT NOT NULL,
    dispatch_status TEXT NOT NULL,
    dispatched_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_first_aid_event_logs_pupil
    ON first_aid_event_logs(pupil_id);
