-- ============================================================================
-- Optimis Core Schema: 03_groups_and_timetable.sql
-- Domain: Year Groups, Teaching/Registration Groups, Enrolments, Rooms & Timetable
-- Engine: SQLite 3 (WAL mode, Foreign Keys ON)
-- Mandatory Optimis DAL Invariant (GEMINI.md §5A & §9.2):
--   All group memberships (`group_pupil_memberships` and `group_staff_assignments`)
--   and `timetable_slots` are effective-dated so `:own_groups` RBAC scopes
--   dynamically restrict SQL queries to a teacher's active groups.
-- Gibbon References: gibbonYearGroup, gibbonHouse, gibbonDepartment*,
--                    gibbonFormGroup, gibbonCourse*, gibbonGroup*,
--                    gibbonSpace*, gibbonTT*
-- ============================================================================

PRAGMA foreign_keys = ON;

-- @domain: groups
-- @dal_scope: dal.groups.read | dal.groups.write
-- @dfe_cbds: National Curriculum Year Group ('E1','E2','N1','N2','R','1'..'14')
-- @gibbon_ref: gibbonYearGroup
CREATE TABLE IF NOT EXISTS year_groups (
    id TEXT PRIMARY KEY,                              -- e.g. 'yg_7'
    nc_code TEXT NOT NULL UNIQUE,                     -- DfE NC Year code ('E1','E2','N1','N2','R','1'..'14')
    name TEXT NOT NULL UNIQUE,                        -- e.g. 'Year 7'
    phase TEXT NOT NULL CHECK (phase IN ('NURSERY', 'PRIMARY', 'SECONDARY', 'SIXTH_FORM')),
    key_stage TEXT NOT NULL CHECK (key_stage IN ('EYFS', 'KS1', 'KS2', 'KS3', 'KS4', 'KS5')),
    sequence_number INTEGER NOT NULL UNIQUE,
    head_of_year_staff_id TEXT REFERENCES staff(id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_year_groups_hoy ON year_groups(head_of_year_staff_id);

-- @domain: groups
-- @dal_scope: dal.groups.read | dal.groups.write
-- @gibbon_ref: gibbonHouse
CREATE TABLE IF NOT EXISTS houses (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    name_short TEXT NOT NULL UNIQUE,
    color_hex TEXT NOT NULL DEFAULT '#000000'
);

-- @domain: groups
-- @dal_scope: dal.groups.read | dal.groups.write
-- @gibbon_ref: gibbonDepartment, gibbonDepartmentStaff
CREATE TABLE IF NOT EXISTS departments (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    name_short TEXT NOT NULL UNIQUE,
    type TEXT NOT NULL DEFAULT 'LEARNING_AREA' CHECK (type IN ('LEARNING_AREA', 'ADMINISTRATION', 'PASTORAL')),
    head_of_department_staff_id TEXT REFERENCES staff(id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_departments_hod ON departments(head_of_department_staff_id);

-- @domain: groups
-- @dal_scope: dal.groups.read | dal.groups.write
-- @gibbon_ref: gibbonSpace
CREATE TABLE IF NOT EXISTS rooms (
    id TEXT PRIMARY KEY,                              -- e.g. 'rm_sci1'
    code TEXT NOT NULL UNIQUE,                        -- e.g. 'SCI1'
    name TEXT NOT NULL UNIQUE,
    room_type TEXT NOT NULL DEFAULT 'CLASSROOM',      -- 'CLASSROOM', 'LAB', 'HALL', 'SPORTS', 'OFFICE'
    capacity INTEGER NOT NULL DEFAULT 30,
    is_active INTEGER NOT NULL DEFAULT 1 CHECK (is_active IN (0, 1)),
    phone_extension TEXT NOT NULL DEFAULT ''
);

-- @domain: groups
-- @dal_scope: dal.groups.read | dal.groups.write
-- @dfe_cbds: Qualification / Subject Classification (JCQ / DfE Subject Codes)
-- @gibbon_ref: gibbonCourse
CREATE TABLE IF NOT EXISTS courses (
    id TEXT PRIMARY KEY,
    academic_year_id TEXT NOT NULL REFERENCES academic_years(id) ON DELETE RESTRICT,
    department_id TEXT REFERENCES departments(id) ON DELETE SET NULL,
    code TEXT NOT NULL,                               -- e.g. 'MAT-KS3'
    name TEXT NOT NULL,                               -- e.g. 'Key Stage 3 Mathematics'
    dfe_subject_code TEXT NOT NULL DEFAULT '',
    UNIQUE (academic_year_id, code)
);
CREATE INDEX IF NOT EXISTS idx_courses_department ON courses(department_id);

-- @domain: groups
-- @dal_scope: dal.groups.read | dal.groups.write
-- @description: Unified Group Spine covering Form/Registration Groups, Teaching Classes, and Intervention Groups
-- @gibbon_ref: gibbonFormGroup, gibbonCourseClass, gibbonGroup
CREATE TABLE IF NOT EXISTS groups (
    id TEXT PRIMARY KEY,                              -- e.g. 'grp_...'
    academic_year_id TEXT NOT NULL REFERENCES academic_years(id) ON DELETE RESTRICT,
    group_type TEXT NOT NULL CHECK (group_type IN ('REGISTRATION_FORM', 'TEACHING_CLASS', 'INTERVENTION', 'CUSTOM')),
    code TEXT NOT NULL,                               -- e.g. '7A' or '10Ma1'
    name TEXT NOT NULL,
    year_group_id TEXT REFERENCES year_groups(id) ON DELETE SET NULL,
    course_id TEXT REFERENCES courses(id) ON DELETE SET NULL,
    default_room_id TEXT REFERENCES rooms(id) ON DELETE SET NULL,
    takes_attendance INTEGER NOT NULL DEFAULT 1 CHECK (takes_attendance IN (0, 1)),
    effective_from TEXT NOT NULL,
    effective_to TEXT,
    CHECK (effective_to IS NULL OR effective_to >= effective_from),
    UNIQUE (academic_year_id, code)
);
CREATE INDEX IF NOT EXISTS idx_groups_year_type ON groups(academic_year_id, group_type);
CREATE INDEX IF NOT EXISTS idx_groups_year_group ON groups(year_group_id);
CREATE INDEX IF NOT EXISTS idx_groups_course ON groups(course_id);
CREATE INDEX IF NOT EXISTS idx_groups_room ON groups(default_room_id);

-- @domain: groups
-- @dal_scope: dal.groups.read | dal.groups.write
-- @description: Temporal pupil membership in Form/Teaching groups (powers `:own_groups` RBAC)
-- @gibbon_ref: gibbonCourseClassPerson, gibbonStudentEnrolment, gibbonGroupPerson
CREATE TABLE IF NOT EXISTS group_pupil_memberships (
    id TEXT PRIMARY KEY,
    group_id TEXT NOT NULL REFERENCES groups(id) ON DELETE RESTRICT,
    pupil_id TEXT NOT NULL REFERENCES pupils(id) ON DELETE RESTRICT,
    academic_year_id TEXT NOT NULL REFERENCES academic_years(id) ON DELETE RESTRICT,
    effective_from TEXT NOT NULL,
    effective_to TEXT,
    CHECK (effective_to IS NULL OR effective_to >= effective_from),
    UNIQUE (group_id, pupil_id, effective_from)
);
CREATE INDEX IF NOT EXISTS idx_group_pupil_member_lookup ON group_pupil_memberships(pupil_id, academic_year_id, effective_from, effective_to);
CREATE INDEX IF NOT EXISTS idx_group_pupil_group_lookup ON group_pupil_memberships(group_id, effective_from, effective_to);
CREATE INDEX IF NOT EXISTS idx_group_pupil_year ON group_pupil_memberships(academic_year_id);

-- @domain: groups
-- @dal_scope: dal.groups.read | dal.groups.write
-- @description: Temporal staff assignment to Form/Teaching groups (powers `:own_groups` RBAC)
-- @gibbon_ref: gibbonCourseClassPerson, gibbonFormGroup (Tutor1..3)
CREATE TABLE IF NOT EXISTS group_staff_assignments (
    id TEXT PRIMARY KEY,
    group_id TEXT NOT NULL REFERENCES groups(id) ON DELETE RESTRICT,
    staff_id TEXT NOT NULL REFERENCES staff(id) ON DELETE RESTRICT,
    role TEXT NOT NULL CHECK (role IN ('LEAD_TEACHER', 'CO_TEACHER', 'COVER_TEACHER', 'FORM_TUTOR', 'TEACHING_ASSISTANT')),
    academic_year_id TEXT NOT NULL REFERENCES academic_years(id) ON DELETE RESTRICT,
    effective_from TEXT NOT NULL,
    effective_to TEXT,
    CHECK (effective_to IS NULL OR effective_to >= effective_from),
    UNIQUE (group_id, staff_id, role, effective_from)
);
CREATE INDEX IF NOT EXISTS idx_group_staff_rbac ON group_staff_assignments(staff_id, effective_from, effective_to);
CREATE INDEX IF NOT EXISTS idx_group_staff_year ON group_staff_assignments(academic_year_id);

-- @domain: timetable
-- @dal_scope: dal.groups.read | dal.groups.write
-- @gibbon_ref: gibbonTT, gibbonTTDay, gibbonTTColumn, gibbonTTColumnRow
CREATE TABLE IF NOT EXISTS timetable_periods (
    id TEXT PRIMARY KEY,                              -- e.g. 'ttp_wkA_mon_p1'
    academic_year_id TEXT NOT NULL REFERENCES academic_years(id) ON DELETE RESTRICT,
    cycle_week INTEGER NOT NULL DEFAULT 1 CHECK (cycle_week IN (1, 2)), -- Supports 1-week or 2-week (Week A/B) rolling timetables
    day_of_week_id INTEGER NOT NULL REFERENCES days_of_week(id) ON DELETE RESTRICT,
    period_code TEXT NOT NULL,                        -- e.g. 'AM_REG', 'P1', 'P2', 'BREAK', 'P3', 'LUNCH', 'PM_REG', 'P4', 'P5'
    name TEXT NOT NULL,
    start_time TEXT NOT NULL,                         -- HH:MM
    end_time TEXT NOT NULL,                           -- HH:MM
    period_type TEXT NOT NULL CHECK (period_type IN ('STATUTORY_AM_REG', 'STATUTORY_PM_REG', 'LESSON', 'BREAK', 'PASTORAL')),
    maps_to_statutory_session TEXT CHECK (maps_to_statutory_session IN ('AM', 'PM')), -- Links period to statutory AM/PM session
    sequence_number INTEGER NOT NULL,
    CHECK (end_time > start_time),
    UNIQUE (academic_year_id, cycle_week, day_of_week_id, period_code)
);
CREATE INDEX IF NOT EXISTS idx_timetable_periods_day ON timetable_periods(day_of_week_id);

-- @domain: timetable
-- @dal_scope: dal.groups.read | dal.groups.write
-- @gibbon_ref: gibbonTTDayRowClass, gibbonTTSpaceChange
CREATE TABLE IF NOT EXISTS timetable_slots (
    id TEXT PRIMARY KEY,
    timetable_period_id TEXT NOT NULL REFERENCES timetable_periods(id) ON DELETE RESTRICT,
    group_id TEXT NOT NULL REFERENCES groups(id) ON DELETE RESTRICT,
    staff_id TEXT REFERENCES staff(id) ON DELETE SET NULL,
    room_id TEXT REFERENCES rooms(id) ON DELETE SET NULL,
    effective_from TEXT NOT NULL,
    effective_to TEXT,
    CHECK (effective_to IS NULL OR effective_to >= effective_from),
    UNIQUE (timetable_period_id, group_id, effective_from)
);
CREATE INDEX IF NOT EXISTS idx_timetable_slots_group ON timetable_slots(group_id, effective_from, effective_to);
CREATE INDEX IF NOT EXISTS idx_timetable_slots_staff ON timetable_slots(staff_id);
CREATE INDEX IF NOT EXISTS idx_timetable_slots_room ON timetable_slots(room_id);
