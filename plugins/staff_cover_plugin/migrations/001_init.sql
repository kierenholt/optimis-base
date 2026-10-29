CREATE TABLE IF NOT EXISTS cover_absences_and_assignments (
    id TEXT PRIMARY KEY,
    absent_staff_id TEXT NOT NULL,
    cover_staff_id TEXT NOT NULL,
    group_id TEXT NOT NULL,
    slot_id TEXT NOT NULL,
    academic_year_id TEXT NOT NULL,
    cover_date TEXT NOT NULL,
    session TEXT NOT NULL DEFAULT 'AM',
    assigned_role TEXT NOT NULL DEFAULT 'CO_TEACHER',
    staff_assignment_id TEXT NOT NULL,
    absence_reason TEXT NOT NULL DEFAULT '',
    cover_notes TEXT NOT NULL DEFAULT '',
    assigned_by_staff_id TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_cover_assignments_group_date ON cover_absences_and_assignments(group_id, cover_date);
CREATE INDEX IF NOT EXISTS idx_cover_assignments_cover_staff ON cover_absences_and_assignments(cover_staff_id, cover_date);

CREATE TABLE IF NOT EXISTS cover_pupil_register_logs (
    id TEXT PRIMARY KEY,
    cover_assignment_id TEXT NOT NULL DEFAULT '',
    pupil_id TEXT NOT NULL,
    group_id TEXT NOT NULL,
    academic_year_id TEXT NOT NULL,
    cover_date TEXT NOT NULL,
    session TEXT NOT NULL,
    attendance_code TEXT NOT NULL,
    minutes_late INTEGER NOT NULL DEFAULT 0,
    register_comment TEXT NOT NULL DEFAULT '',
    taken_by_staff_id TEXT NOT NULL,
    submission_id TEXT NOT NULL,
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_cover_pupil_logs_pupil ON cover_pupil_register_logs(pupil_id);
CREATE INDEX IF NOT EXISTS idx_cover_pupil_logs_group_date ON cover_pupil_register_logs(group_id, cover_date);
