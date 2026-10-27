-- ============================================================================
-- Optimis Core Schema: 00_system_and_academic_cycle.sql
-- Domain: System Configuration, Academic Calendar & Shared Lookups
-- Engine: SQLite 3 (WAL mode, Foreign Keys ON)
-- Gibbon References: gibbonSetting, gibbonSchoolYear, gibbonSchoolYearTerm,
--                    gibbonSchoolYearSpecialDay, gibbonDaysOfWeek,
--                    gibbonAlertLevel, gibbonDataRetention
-- ============================================================================

PRAGMA foreign_keys = ON;

-- @domain: system
-- @dal_scope: host.internal
-- @description: Self-contained runtime configuration table (GEMINI.md §7C).
CREATE TABLE IF NOT EXISTS __system_config (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

-- @domain: academic_cycle
-- @dal_scope: dal.academic.read | dal.academic.write
-- @dfe_cbds: School Census Academic Year (e.g. '2026/2027')
-- @gibbon_ref: gibbonSchoolYear
CREATE TABLE IF NOT EXISTS academic_years (
    id TEXT PRIMARY KEY,                              -- e.g. 'ay_2026_2027'
    name TEXT NOT NULL UNIQUE,                        -- e.g. '2026/2027'
    status TEXT NOT NULL CHECK (status IN ('PAST', 'CURRENT', 'UPCOMING')),
    sequence_number INTEGER NOT NULL UNIQUE,
    first_day TEXT NOT NULL,                          -- ISO-8601 Date (YYYY-MM-DD)
    last_day TEXT NOT NULL,                           -- ISO-8601 Date (YYYY-MM-DD)
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    CHECK (last_day >= first_day)
);

-- @domain: academic_cycle
-- @dal_scope: dal.academic.read | dal.academic.write
-- @dfe_cbds: Termly Census Windows (AUTUMN, SPRING, SUMMER)
-- @gibbon_ref: gibbonSchoolYearTerm
CREATE TABLE IF NOT EXISTS academic_terms (
    id TEXT PRIMARY KEY,
    academic_year_id TEXT NOT NULL REFERENCES academic_years(id) ON DELETE RESTRICT,
    code TEXT NOT NULL CHECK (code IN ('AUTUMN', 'SPRING', 'SUMMER', 'HALF_TERM_1', 'HALF_TERM_2', 'HALF_TERM_3', 'HALF_TERM_4', 'HALF_TERM_5', 'HALF_TERM_6')),
    name TEXT NOT NULL,
    name_short TEXT NOT NULL,
    sequence_number INTEGER NOT NULL,
    first_day TEXT NOT NULL,
    last_day TEXT NOT NULL,
    CHECK (last_day >= first_day),
    UNIQUE (academic_year_id, sequence_number)
);
CREATE INDEX IF NOT EXISTS idx_academic_terms_year ON academic_terms(academic_year_id);

-- @domain: academic_cycle
-- @dal_scope: dal.academic.read | dal.academic.write
-- @gibbon_ref: gibbonDaysOfWeek
CREATE TABLE IF NOT EXISTS days_of_week (
    id INTEGER PRIMARY KEY CHECK (id BETWEEN 1 AND 7), -- 1=Monday .. 7=Sunday (ISO-8601)
    name TEXT NOT NULL UNIQUE,
    name_short TEXT NOT NULL UNIQUE,
    is_school_day INTEGER NOT NULL DEFAULT 1 CHECK (is_school_day IN (0, 1)),
    school_open_time TEXT,                             -- HH:MM
    am_register_time TEXT,                             -- HH:MM (Statutory AM roll call)
    pm_register_time TEXT,                             -- HH:MM (Statutory PM roll call)
    school_close_time TEXT                             -- HH:MM
);

-- @domain: academic_cycle
-- @dal_scope: dal.academic.read | dal.academic.write
-- @dfe_cbds: Attendance Code '#' (Planned whole or partial school closure)
-- @gibbon_ref: gibbonSchoolYearSpecialDay
CREATE TABLE IF NOT EXISTS academic_special_days (
    id TEXT PRIMARY KEY,
    academic_term_id TEXT NOT NULL REFERENCES academic_terms(id) ON DELETE RESTRICT,
    date TEXT NOT NULL UNIQUE,                         -- ISO-8601 Date (YYYY-MM-DD)
    type TEXT NOT NULL CHECK (type IN ('SCHOOL_CLOSURE', 'INSET_DAY', 'TIMING_CHANGE', 'OFF_TIMETABLE')),
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    school_open_time TEXT,
    school_start_time TEXT,
    school_end_time TEXT,
    school_close_time TEXT,
    applies_to_year_group_ids TEXT,                    -- JSON array of year_group IDs if partial closure
    applies_to_group_ids TEXT                          -- JSON array of group IDs if partial closure
);
CREATE INDEX IF NOT EXISTS idx_special_days_term ON academic_special_days(academic_term_id);

-- @domain: lookups
-- @dal_scope: dal.pupils.read
-- @gibbon_ref: gibbonAlertLevel
CREATE TABLE IF NOT EXISTS alert_levels (
    id TEXT PRIMARY KEY,                               -- e.g. 'LOW', 'MEDIUM', 'HIGH', 'CRITICAL'
    name TEXT NOT NULL UNIQUE,
    name_short TEXT NOT NULL UNIQUE,
    color_hex TEXT NOT NULL DEFAULT '#000000',
    bg_color_hex TEXT NOT NULL DEFAULT '#FFFFFF',
    description TEXT NOT NULL DEFAULT '',
    sequence_number INTEGER NOT NULL UNIQUE
);

-- @domain: compliance
-- @dal_scope: host.gdpr
-- @gibbon_ref: gibbonDataRetention
CREATE TABLE IF NOT EXISTS data_retention_policies (
    domain_key TEXT PRIMARY KEY,                       -- e.g. 'PUPIL_ADMISSIONS', 'ATTENDANCE', 'SEN', 'SAFEGUARDING'
    description TEXT NOT NULL,
    retention_rule TEXT NOT NULL,                      -- e.g. 'DOB_PLUS_25_YEARS', 'DOB_PLUS_31_YEARS', 'LEAVING_PLUS_3_YEARS'
    retention_years INTEGER NOT NULL,
    statutory_basis TEXT NOT NULL                      -- e.g. 'KCSIE', 'DfE School Attendance Regulations', 'SEND_CODE_OF_PRACTICE'
);
