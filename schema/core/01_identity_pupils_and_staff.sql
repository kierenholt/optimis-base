-- ============================================================================
-- Optimis Core Schema: 01_identity_pupils_and_staff.sql
-- Domain: Persons, Roles, Statutory Pupils (DfE CBDS), Enrolments & Staff HR
-- Engine: SQLite 3 (WAL mode, Foreign Keys ON)
-- Identity Invariant:
--   `persons`, `pupils`, and `staff` are strictly 1-row-per-human entity roots
--   (NOT temporal rows). All time-bounded changes (enrolments, contracts,
--   roles, absences) live in effective-dated child tables.
-- Gibbon References: gibbonPerson, gibbonRole, gibbonStudentEnrolment,
--                    gibbonStaff, gibbonStaffContract, gibbonStaffAbsence*,
--                    gibbonStaffCoverage*
-- ============================================================================

PRAGMA foreign_keys = ON;

-- @domain: identity
-- @dal_scope: dal.identity.read | dal.identity.write
-- @gibbon_ref: gibbonRole
CREATE TABLE IF NOT EXISTS roles (
    id TEXT PRIMARY KEY,                              -- e.g. 'TEACHER', 'HOD', 'DSL', 'ADMIN', 'PUPIL', 'GUARDIAN'
    category TEXT NOT NULL CHECK (category IN ('STAFF', 'PUPIL', 'GUARDIAN', 'SYSTEM')),
    name TEXT NOT NULL UNIQUE,
    name_short TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    default_scopes_json TEXT NOT NULL DEFAULT '[]'    -- JSON array of DAL scope strings
);

-- @domain: identity
-- @dal_scope: dal.identity.read | dal.identity.write
-- @dfe_cbds: Shared Identity Spine (1 row per human; single source of truth for names, DOB, sex)
-- @gibbon_ref: gibbonPerson
CREATE TABLE IF NOT EXISTS persons (
    id TEXT PRIMARY KEY,                              -- e.g. 'per_...'
    external_sso_sub TEXT UNIQUE,                     -- Entra ID / Google Workspace OIDC subject
    email TEXT UNIQUE,
    title TEXT NOT NULL DEFAULT '',
    legal_first_name TEXT NOT NULL,
    legal_middle_names TEXT NOT NULL DEFAULT '',
    legal_last_name TEXT NOT NULL,
    preferred_first_name TEXT NOT NULL DEFAULT '',
    preferred_last_name TEXT NOT NULL DEFAULT '',
    former_last_name TEXT NOT NULL DEFAULT '',
    sex TEXT NOT NULL DEFAULT 'U' CHECK (sex IN ('M', 'F', 'U', 'X')), -- DfE CBDS sex code
    gender_identity TEXT NOT NULL DEFAULT '',
    dob TEXT,                                         -- ISO-8601 Date (YYYY-MM-DD)
    phone_primary TEXT NOT NULL DEFAULT '',
    phone_primary_type TEXT NOT NULL DEFAULT '' CHECK (phone_primary_type IN ('', 'MOBILE', 'HOME', 'WORK', 'OTHER')),
    phone_secondary TEXT NOT NULL DEFAULT '',
    phone_secondary_type TEXT NOT NULL DEFAULT '' CHECK (phone_secondary_type IN ('', 'MOBILE', 'HOME', 'WORK', 'OTHER')),
    photo_path TEXT,
    status TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'EXPECTED', 'LEFT', 'DECEASED', 'ARCHIVED')),
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);
CREATE INDEX IF NOT EXISTS idx_persons_last_first ON persons(legal_last_name, legal_first_name);

-- @domain: identity
-- @dal_scope: dal.identity.read | dal.identity.write
-- @gibbon_ref: gibbonPerson.gibbonRoleIDAll
CREATE TABLE IF NOT EXISTS person_roles (
    person_id TEXT NOT NULL REFERENCES persons(id) ON DELETE RESTRICT,
    role_id TEXT NOT NULL REFERENCES roles(id) ON DELETE RESTRICT,
    is_primary INTEGER NOT NULL DEFAULT 0 CHECK (is_primary IN (0, 1)),
    effective_from TEXT NOT NULL,
    effective_to TEXT,
    CHECK (effective_to IS NULL OR effective_to >= effective_from),
    PRIMARY KEY (person_id, role_id, effective_from)
);
CREATE INDEX IF NOT EXISTS idx_person_roles_role ON person_roles(role_id);

-- @domain: pupils
-- @dal_scope: dal.pupils.read | dal.pupils.write
-- @dfe_cbds: N00001 (UPN), N00015 (ULN), N00177 (Ethnicity), N00016 (First Language),
--            Looked After (LAC) / Service Child / Young Carer permanent characteristics
-- @gibbon_ref: gibbonPerson (student attributes)
CREATE TABLE IF NOT EXISTS pupils (
    id TEXT PRIMARY KEY,                              -- e.g. 'pup_...' (1 permanent row per pupil)
    person_id TEXT NOT NULL UNIQUE REFERENCES persons(id) ON DELETE RESTRICT,
    upn TEXT NOT NULL UNIQUE,                         -- 13-char UK Unique Pupil Number (statutory checksum)
    former_upn TEXT,
    uln TEXT UNIQUE,                                  -- 10-digit Unique Learner Number (Secondary/Post-16)
    nhs_number TEXT UNIQUE,                           -- 10-digit NHS Number
    admission_number TEXT UNIQUE,
    ethnicity_code TEXT NOT NULL DEFAULT 'NOBT',      -- DfE CBDS 4-char Ethnicity code (e.g. 'WBRI', 'NOBT', 'REFU')
    first_language_code TEXT NOT NULL DEFAULT 'ENG',  -- DfE CBDS 3-char Language code (e.g. 'ENG', 'POL', 'URD')
    eal_flag INTEGER NOT NULL DEFAULT 0 CHECK (eal_flag IN (0, 1)),
    country_of_birth TEXT NOT NULL DEFAULT 'GBR',
    religion_code TEXT NOT NULL DEFAULT '',
    pupil_premium_flag INTEGER NOT NULL DEFAULT 0 CHECK (pupil_premium_flag IN (0, 1)),
    early_years_pp_flag INTEGER NOT NULL DEFAULT 0 CHECK (early_years_pp_flag IN (0, 1)),
    service_child_flag TEXT NOT NULL DEFAULT 'N' CHECK (service_child_flag IN ('Y', 'N', 'U', 'R')),
    looked_after_flag INTEGER NOT NULL DEFAULT 0 CHECK (looked_after_flag IN (0, 1)),
    post_looked_after_code TEXT NOT NULL DEFAULT 'N' CHECK (post_looked_after_code IN ('N', 'A', 'G', 'R', 'O')),
    young_carer_code TEXT NOT NULL DEFAULT 'N' CHECK (young_carer_code IN ('N', 'S', 'P', 'U')),
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

-- @domain: pupils
-- @dal_scope: dal.pupils.read | dal.pupils.write
-- @dfe_cbds: Temporal Admission & Year-Group Enrolment (N00064 Enrolment Status, Early Years Funded Hours)
-- @gibbon_ref: gibbonStudentEnrolment
CREATE TABLE IF NOT EXISTS pupil_year_enrolments (
    id TEXT PRIMARY KEY,
    pupil_id TEXT NOT NULL REFERENCES pupils(id) ON DELETE RESTRICT,
    academic_year_id TEXT NOT NULL REFERENCES academic_years(id) ON DELETE RESTRICT,
    nc_year_group TEXT NOT NULL,                      -- DfE National Curriculum Year ('E1','E2','N1','N2','R','1'..'14')
    enrolment_status TEXT NOT NULL DEFAULT 'C' CHECK (enrolment_status IN ('C', 'M', 'S', 'F', 'O')), -- Current, Main Dual, Subsidiary Dual, FE, Other
    boarder_status TEXT NOT NULL DEFAULT 'N' CHECK (boarder_status IN ('N', 'B', '6', '7')),
    part_time_flag INTEGER NOT NULL DEFAULT 0 CHECK (part_time_flag IN (0, 1)),
    registration_group_id TEXT,                       -- Form/Tutor group
    house_id TEXT,
    roll_order INTEGER,
    -- Early Years (EYFS / Nursery) Statutory Census Fields:
    ey_funded_hours REAL NOT NULL DEFAULT 0.0 CHECK (ey_funded_hours >= 0.0 AND ey_funded_hours <= 40.0),
    ey_extended_funded_hours REAL NOT NULL DEFAULT 0.0 CHECK (ey_extended_funded_hours >= 0.0 AND ey_extended_funded_hours <= 40.0),
    ey_thirty_hour_code TEXT,                         -- 11-digit HMRC Extended Childcare Entitlement code
    ey_daf_indicator INTEGER NOT NULL DEFAULT 0 CHECK (ey_daf_indicator IN (0, 1)), -- Disability Access Fund
    previous_school_urn TEXT,
    destination_school_urn TEXT,
    leaving_reason_code TEXT,                         -- DfE statutory off-rolling reason code
    effective_from TEXT NOT NULL,                     -- Statutory admission / enrolment start date (YYYY-MM-DD)
    effective_to TEXT,                                -- Statutory leaving / enrolment end date (NULL = active on roll)
    CHECK (effective_to IS NULL OR effective_to >= effective_from),
    UNIQUE (pupil_id, academic_year_id, effective_from)
);
CREATE INDEX IF NOT EXISTS idx_pupil_enrolments_year ON pupil_year_enrolments(academic_year_id, nc_year_group);
CREATE INDEX IF NOT EXISTS idx_pupil_enrolments_pupil_temporal ON pupil_year_enrolments(pupil_id, effective_from, effective_to);

-- @domain: staff
-- @dal_scope: dal.staff.read | dal.staff.write
-- @dfe_cbds: School Workforce Census (SWC) & Single Central Record (SCR) — 1 permanent row per staff member
-- @gibbon_ref: gibbonStaff, gibbonSubstitute
CREATE TABLE IF NOT EXISTS staff (
    id TEXT PRIMARY KEY,                              -- e.g. 'stf_...' (1 permanent row per staff member)
    person_id TEXT NOT NULL UNIQUE REFERENCES persons(id) ON DELETE RESTRICT,
    staff_code TEXT UNIQUE,                           -- Timetable 3/4-letter initials
    trn TEXT UNIQUE,                                  -- 7-digit DfE Teacher Reference Number
    ni_number_encrypted TEXT,                         -- Encrypted National Insurance Number for SWC
    staff_type TEXT NOT NULL CHECK (staff_type IN ('TEACHING', 'SUPPORT', 'LEADERSHIP', 'SUPPLY_INTERNAL', 'SUPPLY_AGENCY')),
    job_title TEXT NOT NULL DEFAULT '',
    qts_flag INTEGER NOT NULL DEFAULT 0 CHECK (qts_flag IN (0, 1)),     -- Qualified Teacher Status
    eyts_flag INTEGER NOT NULL DEFAULT 0 CHECK (eyts_flag IN (0, 1)),    -- Early Years Teacher Status
    hlta_flag INTEGER NOT NULL DEFAULT 0 CHECK (hlta_flag IN (0, 1)),    -- Higher Level Teaching Assistant
    is_dsl INTEGER NOT NULL DEFAULT 0 CHECK (is_dsl IN (0, 1)),          -- Designated Safeguarding Lead
    first_aid_qualified INTEGER NOT NULL DEFAULT 0 CHECK (first_aid_qualified IN (0, 1)),
    first_aid_expiry_date TEXT,
    dbs_check_date TEXT,                              -- Single Central Record (SCR) DBS clearance date
    dbs_certificate_number TEXT,
    prohibition_check_date TEXT,
    right_to_work_check_date TEXT,
    exclude_from_cover INTEGER NOT NULL DEFAULT 0 CHECK (exclude_from_cover IN (0, 1)),
    cover_priority INTEGER NOT NULL DEFAULT 0,
    hire_date TEXT NOT NULL,                          -- Employment start date
    termination_date TEXT                             -- Employment end date (NULL = currently employed)
);

-- @domain: staff
-- @dal_scope: dal.staff.read:admin | dal.staff.write:admin
-- @dfe_cbds: School Workforce Census Contract Module (Post, Pay Range, FTE, Hours)
-- @gibbon_ref: gibbonStaffContract
CREATE TABLE IF NOT EXISTS staff_contracts (
    id TEXT PRIMARY KEY,
    staff_id TEXT NOT NULL REFERENCES staff(id) ON DELETE RESTRICT,
    contract_type TEXT NOT NULL CHECK (contract_type IN ('PERMANENT', 'FIXED_TERM', 'TEMPORARY', 'AGENCY')),
    swc_post_code TEXT NOT NULL DEFAULT 'TCH',        -- DfE SWC Post: EHT, HDT, DHD, AHD, TCH, SUP, etc.
    job_title TEXT NOT NULL,
    pay_scale TEXT,                                   -- e.g. 'MPS', 'UPS', 'L', 'SUPPORT'
    pay_spine_point TEXT,                             -- e.g. 'M1'..'M6', 'U1'..'U3', 'L1'..'L43'
    fte REAL NOT NULL DEFAULT 1.0 CHECK (fte > 0.0 AND fte <= 1.5),
    hours_per_week REAL,
    weeks_per_year REAL,
    effective_from TEXT NOT NULL,
    effective_to TEXT,
    CHECK (effective_to IS NULL OR effective_to >= effective_from)
);
CREATE INDEX IF NOT EXISTS idx_staff_contracts_staff ON staff_contracts(staff_id, effective_from);

-- @domain: staff
-- @dal_scope: dal.staff.read | dal.staff.write
-- @dfe_cbds: School Workforce Census Sickness/Absence Codes
-- @gibbon_ref: gibbonStaffAbsenceType, gibbonStaffAbsence, gibbonStaffAbsenceDate
CREATE TABLE IF NOT EXISTS staff_absences (
    id TEXT PRIMARY KEY,
    staff_id TEXT NOT NULL REFERENCES staff(id) ON DELETE RESTRICT,
    academic_year_id TEXT NOT NULL REFERENCES academic_years(id) ON DELETE RESTRICT,
    absence_category_code TEXT NOT NULL,              -- DfE SWC code: 'SIC' (Sickness), 'MAT' (Maternity), 'TRN' (Training), 'OTH'
    reason_detail TEXT NOT NULL DEFAULT '',
    confidential_notes TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'APPROVED' CHECK (status IN ('PENDING', 'APPROVED', 'DECLINED', 'CANCELLED')),
    cover_required INTEGER NOT NULL DEFAULT 0 CHECK (cover_required IN (0, 1)),
    start_date TEXT NOT NULL,
    end_date TEXT NOT NULL,
    working_days_lost REAL NOT NULL DEFAULT 1.0,
    approved_by_staff_id TEXT REFERENCES staff(id) ON DELETE SET NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    CHECK (end_date >= start_date)
);
CREATE INDEX IF NOT EXISTS idx_staff_absences_staff_dates ON staff_absences(staff_id, start_date, end_date);
CREATE INDEX IF NOT EXISTS idx_staff_absences_year ON staff_absences(academic_year_id);
CREATE INDEX IF NOT EXISTS idx_staff_absences_approver ON staff_absences(approved_by_staff_id);

-- @domain: staff
-- @dal_scope: dal.staff.read | dal.staff.write
-- @gibbon_ref: gibbonStaffCoverage, gibbonStaffCoverageDate
CREATE TABLE IF NOT EXISTS staff_cover_assignments (
    id TEXT PRIMARY KEY,
    staff_absence_id TEXT REFERENCES staff_absences(id) ON DELETE RESTRICT,
    timetable_slot_id TEXT,                           -- Lesson slot being covered
    date TEXT NOT NULL,                               -- ISO-8601 Date (YYYY-MM-DD)
    absent_staff_id TEXT NOT NULL REFERENCES staff(id) ON DELETE RESTRICT,
    covering_staff_id TEXT REFERENCES staff(id) ON DELETE SET NULL,
    status TEXT NOT NULL DEFAULT 'REQUESTED' CHECK (status IN ('REQUESTED', 'ACCEPTED', 'DECLINED', 'CANCELLED')),
    cover_notes TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);
CREATE INDEX IF NOT EXISTS idx_staff_cover_date_covering ON staff_cover_assignments(covering_staff_id, date);
CREATE INDEX IF NOT EXISTS idx_staff_cover_absent ON staff_cover_assignments(absent_staff_id, date);
CREATE INDEX IF NOT EXISTS idx_staff_cover_absence_id ON staff_cover_assignments(staff_absence_id);
