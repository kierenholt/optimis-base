CREATE TABLE IF NOT EXISTS sen_provisions (
    id TEXT PRIMARY KEY,
    pupil_id TEXT NOT NULL,
    academic_year_id TEXT NOT NULL,
    sen_provision_code TEXT NOT NULL CHECK (sen_provision_code IN ('K', 'E')),
    primary_sen_need_code TEXT NOT NULL DEFAULT '',
    iep_target TEXT NOT NULL,
    classroom_adjustment TEXT NOT NULL,
    ta_allocation_hours REAL NOT NULL DEFAULT 0,
    assigned_ta_name TEXT NOT NULL DEFAULT '',
    review_notes TEXT NOT NULL DEFAULT '',
    logged_by_staff_id TEXT NOT NULL,
    effective_date TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_sen_provisions_pupil_year
    ON sen_provisions(pupil_id, academic_year_id, created_at);
