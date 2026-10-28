CREATE TABLE IF NOT EXISTS behaviour_events (
    id TEXT PRIMARY KEY,
    pupil_id TEXT NOT NULL,
    academic_year_id TEXT NOT NULL,
    event_date TEXT NOT NULL,
    event_type TEXT NOT NULL CHECK (event_type IN ('MERIT', 'INCIDENT')),
    category TEXT NOT NULL,
    points INTEGER NOT NULL,
    incident_notes TEXT NOT NULL DEFAULT '',
    logged_by_staff_id TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_behaviour_events_pupil ON behaviour_events(pupil_id);
CREATE INDEX IF NOT EXISTS idx_behaviour_events_date ON behaviour_events(event_date);
