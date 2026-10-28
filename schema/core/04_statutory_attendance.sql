-- ============================================================================
-- Optimis Core Schema: 04_statutory_attendance.sql
-- Domain: UK Statutory AM/PM Roll-Call Attendance & Lesson Registers
-- Engine: SQLite 3 (WAL mode, Foreign Keys ON)
-- Mandatory Optimis DAL Invariants (GEMINI.md §3 & compliance.md §5):
--   1. All school phases (Nursery, Primary, Secondary, Sixth Form) resolve down
--      to statutory AM/PM sessions (`attendance_session_marks.session IN ('AM', 'PM')`).
--   2. Statutory attendance records use `ON DELETE RESTRICT` so timetable or
--      group edits NEVER cascade-delete historical roll-call or lesson marks.
-- Gibbon References: gibbonAttendanceCode, gibbonAttendanceLogPerson,
--                    gibbonAttendanceLogFormGroup, gibbonAttendanceLogCourseClass
-- ============================================================================

PRAGMA foreign_keys = ON;

-- @domain: attendance
-- @dal_scope: dal.attendance.read | dal.attendance.write
-- @dfe_cbds: DfE Statutory Attendance & Absence Codes (August 2024+ Statutory Guidance)
-- @gibbon_ref: gibbonAttendanceCode
CREATE TABLE IF NOT EXISTS attendance_codes (
    code TEXT PRIMARY KEY,                            -- Official DfE 1- or 2-char code: '/', '\', 'L', 'B', 'C', 'C1', 'C2', 'D', 'E', 'G', 'I', 'J1', 'K', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y1'..'Y7', 'Z', '#'
    name TEXT NOT NULL UNIQUE,                        -- e.g. 'Present (AM)', 'Illness', 'Unauthorised Absence'
    statistical_category TEXT NOT NULL CHECK (
        statistical_category IN (
            'PRESENT',
            'APPROVED_EDUCATIONAL_ACTIVITY',
            'AUTHORISED_ABSENCE',
            'UNAUTHORISED_ABSENCE',
            'UNABLE_TO_ATTEND_EXCEPTIONAL',
            'NOT_REQUIRED_TO_ATTEND'
        )
    ),
    physical_status TEXT NOT NULL CHECK (physical_status IN ('ONSITE', 'ONSITE_LATE', 'OFFSITE_EDUCATIONAL', 'ABSENT', 'CLOSURE')),
    is_statutory_code INTEGER NOT NULL DEFAULT 1 CHECK (is_statutory_code IN (0, 1)),
    is_active INTEGER NOT NULL DEFAULT 1 CHECK (is_active IN (0, 1)),
    restricted_to_admin INTEGER NOT NULL DEFAULT 0 CHECK (restricted_to_admin IN (0, 1)),
    sequence_number INTEGER NOT NULL
);

-- @domain: attendance
-- @dal_scope: dal.attendance.read | dal.attendance.write
-- @dfe_cbds: Statutory Roll-Call Session Marks (AM / PM) — Retained 3+ years
-- @gibbon_ref: gibbonAttendanceLogPerson (Form Group / Daily Roll Call)
CREATE TABLE IF NOT EXISTS attendance_session_marks (
    id TEXT PRIMARY KEY,                              -- e.g. 'asm_...'
    pupil_id TEXT NOT NULL REFERENCES pupils(id) ON DELETE RESTRICT,
    academic_year_id TEXT NOT NULL REFERENCES academic_years(id) ON DELETE RESTRICT,
    date TEXT NOT NULL,                               -- ISO-8601 Date (YYYY-MM-DD)
    session TEXT NOT NULL CHECK (session IN ('AM', 'PM')), -- MANDATORY INVARIANT: Statutory AM or PM session
    attendance_code TEXT NOT NULL REFERENCES attendance_codes(code) ON DELETE RESTRICT,
    minutes_late INTEGER NOT NULL DEFAULT 0 CHECK (minutes_late >= 0),
    sub_code TEXT NOT NULL DEFAULT '',                -- Optional school sub-code (e.g. 'I01' Respiratory)
    comment TEXT NOT NULL DEFAULT '',
    source_group_id TEXT REFERENCES groups(id) ON DELETE SET NULL,
    recorded_by_staff_id TEXT REFERENCES staff(id) ON DELETE SET NULL,
    recorded_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    UNIQUE (pupil_id, date, session)
);
CREATE INDEX IF NOT EXISTS idx_att_session_pupil_date ON attendance_session_marks(pupil_id, date, session);
CREATE INDEX IF NOT EXISTS idx_att_session_year_date ON attendance_session_marks(academic_year_id, date, session);
CREATE INDEX IF NOT EXISTS idx_att_session_code ON attendance_session_marks(attendance_code, date);
CREATE INDEX IF NOT EXISTS idx_att_session_group ON attendance_session_marks(source_group_id);
CREATE INDEX IF NOT EXISTS idx_att_session_staff ON attendance_session_marks(recorded_by_staff_id);

-- @domain: attendance
-- @dal_scope: dal.attendance.read | dal.attendance.write
-- @description: Period-by-period class register marks (Protected from timetable deletion via ON DELETE RESTRICT)
-- @gibbon_ref: gibbonAttendanceLogPerson (Class context), gibbonAttendanceLogCourseClass
CREATE TABLE IF NOT EXISTS attendance_lesson_marks (
    id TEXT PRIMARY KEY,                              -- e.g. 'alm_...'
    pupil_id TEXT NOT NULL REFERENCES pupils(id) ON DELETE RESTRICT,
    timetable_slot_id TEXT NOT NULL REFERENCES timetable_slots(id) ON DELETE RESTRICT,
    group_id TEXT NOT NULL REFERENCES groups(id) ON DELETE RESTRICT,
    date TEXT NOT NULL,                               -- ISO-8601 Date (YYYY-MM-DD)
    attendance_code TEXT NOT NULL REFERENCES attendance_codes(code) ON DELETE RESTRICT,
    minutes_late INTEGER NOT NULL DEFAULT 0 CHECK (minutes_late >= 0),
    comment TEXT NOT NULL DEFAULT '',
    recorded_by_staff_id TEXT REFERENCES staff(id) ON DELETE SET NULL,
    recorded_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    UNIQUE (pupil_id, timetable_slot_id, date)
);
CREATE INDEX IF NOT EXISTS idx_att_lesson_group_date ON attendance_lesson_marks(group_id, date);
CREATE INDEX IF NOT EXISTS idx_att_lesson_pupil_date ON attendance_lesson_marks(pupil_id, date);
CREATE INDEX IF NOT EXISTS idx_att_lesson_slot ON attendance_lesson_marks(timetable_slot_id, date);
CREATE INDEX IF NOT EXISTS idx_att_lesson_code ON attendance_lesson_marks(attendance_code);
CREATE INDEX IF NOT EXISTS idx_att_lesson_staff ON attendance_lesson_marks(recorded_by_staff_id);

-- @domain: attendance
-- @dal_scope: dal.attendance.read | dal.attendance.write
-- @description: Audit log of completed session/lesson registers (for missing-register alerts)
-- @gibbon_ref: gibbonAttendanceLogFormGroup, gibbonAttendanceLogCourseClass
CREATE TABLE IF NOT EXISTS attendance_register_submissions (
    id TEXT PRIMARY KEY,
    group_id TEXT NOT NULL REFERENCES groups(id) ON DELETE RESTRICT,
    date TEXT NOT NULL,
    session_or_slot_key TEXT NOT NULL,                -- 'AM', 'PM', or `timetable_slot_id`
    taken_by_staff_id TEXT NOT NULL REFERENCES staff(id) ON DELETE RESTRICT,
    taken_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    UNIQUE (group_id, date, session_or_slot_key)
);
CREATE INDEX IF NOT EXISTS idx_att_submissions_date ON attendance_register_submissions(date, group_id);
CREATE INDEX IF NOT EXISTS idx_att_submissions_staff ON attendance_register_submissions(taken_by_staff_id);
