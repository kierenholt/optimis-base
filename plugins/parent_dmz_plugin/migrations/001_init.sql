CREATE TABLE IF NOT EXISTS dmz_inbox_submissions (
    submission_id TEXT PRIMARY KEY,
    pupil_id TEXT NOT NULL,
    submitted_by_guardian_person_id TEXT NOT NULL,
    guardian_name TEXT NOT NULL DEFAULT '',
    submission_type TEXT NOT NULL,
    date TEXT NOT NULL,
    session TEXT NOT NULL DEFAULT '',
    reason TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL CHECK (status IN ('ACCEPTED', 'REJECTED_SAFEGUARDING_BLOCK')),
    processed_by_staff_id TEXT NOT NULL DEFAULT '',
    processed_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_dmz_inbox_submissions_pupil
    ON dmz_inbox_submissions(pupil_id, date);

CREATE TABLE IF NOT EXISTS dmz_projection_snapshots (
    id TEXT PRIMARY KEY,
    pupil_id TEXT NOT NULL,
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    academic_year_id TEXT NOT NULL,
    reference_date TEXT NOT NULL,
    present_sessions INTEGER NOT NULL DEFAULT 0,
    total_sessions INTEGER NOT NULL DEFAULT 0,
    authorized_guardian_ids_json TEXT NOT NULL DEFAULT '[]',
    redacted_confidential_address_count INTEGER NOT NULL DEFAULT 0,
    upn_stripped INTEGER NOT NULL DEFAULT 1 CHECK (upn_stripped = 1),
    sanitized_payload_json TEXT NOT NULL,
    notes TEXT NOT NULL DEFAULT '',
    pushed_by_staff_id TEXT NOT NULL DEFAULT '',
    pushed_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_dmz_projection_snapshots_pupil
    ON dmz_projection_snapshots(pupil_id, reference_date);
