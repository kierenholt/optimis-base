CREATE TABLE IF NOT EXISTS trips (
    id TEXT PRIMARY KEY,
    group_id TEXT NOT NULL,
    academic_year_id TEXT NOT NULL,
    trip_name TEXT NOT NULL,
    destination TEXT NOT NULL,
    trip_date TEXT NOT NULL,
    lead_staff_id TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_trips_group_date
    ON trips (group_id, trip_date);

CREATE TABLE IF NOT EXISTS trip_consents (
    id TEXT PRIMARY KEY,
    trip_id TEXT NOT NULL,
    pupil_id TEXT NOT NULL,
    guardian_person_id TEXT NOT NULL,
    guardian_name TEXT NOT NULL,
    relationship_code TEXT NOT NULL,
    has_parental_responsibility INTEGER NOT NULL CHECK (has_parental_responsibility = 1),
    consent_status TEXT NOT NULL CHECK (consent_status IN ('CONSENTED', 'DECLINED')),
    dietary_notes TEXT NOT NULL DEFAULT '',
    medical_notes TEXT NOT NULL DEFAULT '',
    emergency_phone TEXT NOT NULL DEFAULT '',
    recorded_by_staff_id TEXT NOT NULL,
    recorded_at TEXT NOT NULL,
    UNIQUE (trip_id, pupil_id)
);

CREATE INDEX IF NOT EXISTS idx_trip_consents_trip_id
    ON trip_consents (trip_id);

CREATE INDEX IF NOT EXISTS idx_trip_consents_pupil_id
    ON trip_consents (pupil_id);
