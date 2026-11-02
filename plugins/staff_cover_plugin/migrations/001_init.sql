CREATE TABLE IF NOT EXISTS cover_absences (
    id TEXT PRIMARY KEY,
    absent_staff_id TEXT NOT NULL,
    academic_year_id TEXT NOT NULL,
    cover_date TEXT NOT NULL,
    absence_reason TEXT NOT NULL DEFAULT '',
    recorded_by_staff_id TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_cover_absences_staff_date ON cover_absences(absent_staff_id, cover_date);

CREATE TABLE IF NOT EXISTS cover_slot_assignments (
    id TEXT PRIMARY KEY,
    absence_id TEXT NOT NULL REFERENCES cover_absences(id) ON DELETE CASCADE,
    group_id TEXT NOT NULL,
    slot_id TEXT NOT NULL,
    cover_staff_id TEXT NOT NULL,
    session TEXT NOT NULL DEFAULT 'AM' CHECK (session IN ('AM', 'PM')),
    assigned_role TEXT NOT NULL DEFAULT 'COVER_TEACHER',
    staff_assignment_id TEXT NOT NULL DEFAULT '',
    cover_notes TEXT NOT NULL DEFAULT '',
    assigned_by_staff_id TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_cover_slot_assignments_group_date ON cover_slot_assignments(group_id, absence_id);
CREATE INDEX IF NOT EXISTS idx_cover_slot_assignments_cover_staff ON cover_slot_assignments(cover_staff_id);

CREATE TABLE IF NOT EXISTS cover_handover_pupil_notes (
    id TEXT PRIMARY KEY,
    cover_assignment_id TEXT NOT NULL REFERENCES cover_slot_assignments(id) ON DELETE CASCADE,
    pupil_id TEXT NOT NULL,
    group_id TEXT NOT NULL,
    academic_year_id TEXT NOT NULL,
    cover_date TEXT NOT NULL,
    session TEXT NOT NULL CHECK (session IN ('AM', 'PM')),
    register_comment TEXT NOT NULL DEFAULT '',
    taken_by_staff_id TEXT NOT NULL,
    submission_id TEXT NOT NULL,
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_cover_handover_pupil ON cover_handover_pupil_notes(pupil_id);
CREATE INDEX IF NOT EXISTS idx_cover_handover_group_date ON cover_handover_pupil_notes(group_id, cover_date);

