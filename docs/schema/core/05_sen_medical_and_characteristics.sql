-- ============================================================================
-- Optimis Core Schema: 05_sen_medical_and_characteristics.sql
-- Domain: Special Educational Needs (SEN), Free School Meals (FSM),
--         Suspensions/Exclusions, Alternative Provision, Post-16 Learning Aims,
--         Medical Conditions & Pastoral Notes
-- Engine: SQLite 3 (WAL mode, Foreign Keys ON)
-- Statutory Retention Note (compliance.md §5):
--   SEN records must be retained until the pupil reaches age 31 (DOB + 31 yrs).
--   All statutory tables use `ON DELETE RESTRICT` to prevent accidental cascade
--   deletion prior to statutory retention expiry.
-- Gibbon References: gibbonIN*, gibbonMedicalCondition, gibbonPersonMedical*,
--                    gibbonStudentNote*
-- ============================================================================

PRAGMA foreign_keys = ON;

-- @domain: pupils
-- @dal_scope: dal.pupils.read | dal.pupils.write
-- @dfe_cbds: N00018 (FSM Eligibility Start Date), N00019 (FSM Eligibility End Date), N00178 (FSM Country)
CREATE TABLE IF NOT EXISTS pupil_fsm_periods (
    id TEXT PRIMARY KEY,
    pupil_id TEXT NOT NULL REFERENCES pupils(id) ON DELETE RESTRICT,
    effective_from TEXT NOT NULL,                     -- Statutory FSM start date (YYYY-MM-DD)
    effective_to TEXT,                                -- Statutory FSM end date (NULL if ongoing / Transitional Protection)
    country_code TEXT NOT NULL DEFAULT 'ENG',
    ecs_reference TEXT NOT NULL DEFAULT '',           -- Eligibility Checking Service reference
    verified_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    CHECK (effective_to IS NULL OR effective_to >= effective_from)
);
CREATE INDEX IF NOT EXISTS idx_pupil_fsm_lookup ON pupil_fsm_periods(pupil_id, effective_from, effective_to);

-- @domain: sen
-- @dal_scope: dal.sen.read | dal.sen.write
-- @dfe_cbds: DfE SEN Type Code Set ('SPLD','MLD','SLD','PMLD','SEMH','SLCN','HI','VI','MSI','PD','ASD','OTH','NSA')
-- @gibbon_ref: gibbonINDescriptor
CREATE TABLE IF NOT EXISTS sen_need_types (
    code TEXT PRIMARY KEY,                            -- Official DfE SEN Need Code (e.g. 'ASD', 'SEMH', 'SPLD')
    name TEXT NOT NULL UNIQUE,
    broad_area TEXT NOT NULL CHECK (
        broad_area IN (
            'COGNITION_AND_LEARNING',
            'COMMUNICATION_AND_INTERACTION',
            'SOCIAL_EMOTIONAL_AND_MENTAL_HEALTH',
            'SENSORY_AND_PHYSICAL',
            'OTHER'
        )
    ),
    description TEXT NOT NULL DEFAULT '',
    sequence_number INTEGER NOT NULL
);

-- @domain: sen
-- @dal_scope: dal.sen.read | dal.sen.write
-- @dfe_cbds: N00020 (SEN Provision: 'N'=No SEN, 'K'=SEN Support, 'E'=EHC Plan) — Retained until DOB + 31 years
-- @gibbon_ref: gibbonIN, gibbonINArchive
CREATE TABLE IF NOT EXISTS pupil_sen_records (
    id TEXT PRIMARY KEY,
    pupil_id TEXT NOT NULL REFERENCES pupils(id) ON DELETE RESTRICT,
    provision_code TEXT NOT NULL DEFAULT 'N' CHECK (provision_code IN ('N', 'K', 'E')), -- DfE CBDS: N, K, E
    ehcp_start_date TEXT,                             -- Date EHC Plan was issued (if provision_code = 'E')
    ehcp_review_due_date TEXT,                        -- Statutory annual review deadline
    sen_unit_indicator INTEGER NOT NULL DEFAULT 0 CHECK (sen_unit_indicator IN (0, 1)),
    resourced_provision_indicator INTEGER NOT NULL DEFAULT 0 CHECK (resourced_provision_indicator IN (0, 1)),
    teaching_strategies TEXT NOT NULL DEFAULT '',     -- Classroom differentiation & support strategies
    pupil_targets TEXT NOT NULL DEFAULT '',           -- IEP / Pupil Passport targets
    senco_notes TEXT NOT NULL DEFAULT '',
    effective_from TEXT NOT NULL,
    effective_to TEXT,
    CHECK (effective_to IS NULL OR effective_to >= effective_from)
);
CREATE INDEX IF NOT EXISTS idx_pupil_sen_pupil ON pupil_sen_records(pupil_id, effective_from, effective_to);

-- @domain: sen
-- @dal_scope: dal.sen.read | dal.sen.write
-- @dfe_cbds: N00022 / N00166 (SEN Type Ranking: 1 = Primary Need, 2 = Secondary Need)
-- @gibbon_ref: gibbonINPersonDescriptor
CREATE TABLE IF NOT EXISTS pupil_sen_needs (
    id TEXT PRIMARY KEY,
    pupil_id TEXT NOT NULL REFERENCES pupils(id) ON DELETE RESTRICT,
    sen_need_code TEXT NOT NULL REFERENCES sen_need_types(code) ON DELETE RESTRICT,
    ranking INTEGER NOT NULL CHECK (ranking IN (1, 2)), -- DfE Census requires rank 1 (Primary) and optional rank 2 (Secondary)
    alert_level_id TEXT REFERENCES alert_levels(id) ON DELETE SET NULL,
    effective_from TEXT NOT NULL,
    effective_to TEXT,
    CHECK (effective_to IS NULL OR effective_to >= effective_from),
    UNIQUE (pupil_id, ranking, effective_from)
);
CREATE INDEX IF NOT EXISTS idx_pupil_sen_needs_pupil ON pupil_sen_needs(pupil_id, effective_from);
CREATE INDEX IF NOT EXISTS idx_pupil_sen_needs_code ON pupil_sen_needs(sen_need_code);
CREATE INDEX IF NOT EXISTS idx_pupil_sen_needs_alert ON pupil_sen_needs(alert_level_id);

-- @domain: sen
-- @dal_scope: dal.sen.read | dal.sen.write
-- @gibbon_ref: gibbonINAssistant
CREATE TABLE IF NOT EXISTS pupil_sen_assistants (
    id TEXT PRIMARY KEY,
    pupil_id TEXT NOT NULL REFERENCES pupils(id) ON DELETE RESTRICT,
    staff_id TEXT NOT NULL REFERENCES staff(id) ON DELETE RESTRICT,
    allocation_notes TEXT NOT NULL DEFAULT '',
    effective_from TEXT NOT NULL,
    effective_to TEXT,
    CHECK (effective_to IS NULL OR effective_to >= effective_from),
    UNIQUE (pupil_id, staff_id, effective_from)
);
CREATE INDEX IF NOT EXISTS idx_pupil_sen_assistants_staff ON pupil_sen_assistants(staff_id);

-- @domain: statutory_census
-- @dal_scope: dal.pupils.read | dal.pupils.write
-- @dfe_cbds: Statutory Suspensions & Permanent Exclusions Module (Termly School Census)
CREATE TABLE IF NOT EXISTS pupil_suspensions_exclusions (
    id TEXT PRIMARY KEY,
    pupil_id TEXT NOT NULL REFERENCES pupils(id) ON DELETE RESTRICT,
    academic_year_id TEXT NOT NULL REFERENCES academic_years(id) ON DELETE RESTRICT,
    exclusion_category TEXT NOT NULL CHECK (exclusion_category IN ('SUSPENSION', 'PERMANENT', 'LUNCHTIME')),
    reason_code_1 TEXT NOT NULL,                      -- DfE Exclusion Reason ('PP','PA','VA','VA','BU','RA','SM','DM','LG','MT','OT', etc.)
    reason_code_2 TEXT,                               -- Optional 2nd DfE reason code
    reason_code_3 TEXT,                               -- Optional 3rd DfE reason code
    start_date TEXT NOT NULL,                         -- ISO-8601 Date (YYYY-MM-DD)
    end_date TEXT,                                    -- ISO-8601 Date (NULL for Permanent Exclusion)
    sessions_missed INTEGER NOT NULL CHECK (sessions_missed >= 1), -- Statutory half-day AM/PM sessions missed
    sen_provision_at_exclusion TEXT NOT NULL CHECK (sen_provision_at_exclusion IN ('N', 'K', 'E')),
    governor_review_date TEXT,
    governor_review_outcome TEXT CHECK (governor_review_outcome IN ('UPHELD', 'REINSTATED', 'WITHDRAWN', 'PENDING')),
    notes TEXT NOT NULL DEFAULT '',
    CHECK (end_date IS NULL OR end_date >= start_date)
);
CREATE INDEX IF NOT EXISTS idx_pupil_exclusions_pupil ON pupil_suspensions_exclusions(pupil_id, start_date);
CREATE INDEX IF NOT EXISTS idx_pupil_exclusions_year ON pupil_suspensions_exclusions(academic_year_id, start_date);

-- @domain: statutory_census
-- @dal_scope: dal.pupils.read | dal.pupils.write
-- @dfe_cbds: Statutory Alternative Provision (AP) & Off-Site Placements Module (School Census)
CREATE TABLE IF NOT EXISTS pupil_alternative_provisions (
    id TEXT PRIMARY KEY,
    pupil_id TEXT NOT NULL REFERENCES pupils(id) ON DELETE RESTRICT,
    academic_year_id TEXT NOT NULL REFERENCES academic_years(id) ON DELETE RESTRICT,
    provider_urn TEXT,                                -- 6-digit URN (if registered school/AP)
    provider_ukprn TEXT,                              -- 8-digit UKPRN (if FE/registered provider)
    companies_house_number TEXT,                      -- 8-char Companies House No (if unregistered provider)
    ap_setting_type_code TEXT,                        -- DfE AP Setting Type ('NHS', 'FE', 'OTH', etc.)
    ap_reason_code TEXT NOT NULL,                     -- DfE AP Reason ('EHC', 'MH', 'PEX', 'BHV', 'OTH')
    sen_provision_during_ap TEXT NOT NULL DEFAULT 'N' CHECK (sen_provision_during_ap IN ('N', 'K', 'E')),
    sessions_per_week INTEGER CHECK (sessions_per_week BETWEEN 1 AND 10),
    effective_from TEXT NOT NULL,                     -- AP entry date
    effective_to TEXT,                                -- AP leaving date
    CHECK (effective_to IS NULL OR effective_to >= effective_from)
);
CREATE INDEX IF NOT EXISTS idx_pupil_ap_pupil ON pupil_alternative_provisions(pupil_id, effective_from);
CREATE INDEX IF NOT EXISTS idx_pupil_ap_year ON pupil_alternative_provisions(academic_year_id);

-- @domain: statutory_census
-- @dal_scope: dal.pupils.read | dal.pupils.write
-- @dfe_cbds: Post-16 Learning Aims & 16-19 Funding Module (Autumn School Census for Sixth Form / KS5)
CREATE TABLE IF NOT EXISTS pupil_post16_learning_aims (
    id TEXT PRIMARY KEY,
    pupil_id TEXT NOT NULL REFERENCES pupils(id) ON DELETE RESTRICT,
    academic_year_id TEXT NOT NULL REFERENCES academic_years(id) ON DELETE RESTRICT,
    qn_code TEXT NOT NULL,                            -- 8-char Ofqual Qualification Number (QN)
    subject_classification_code TEXT NOT NULL,        -- 4-char LDCS / SSI Subject Code
    course_id TEXT REFERENCES courses(id) ON DELETE SET NULL,
    planned_learning_hours INTEGER NOT NULL DEFAULT 0 CHECK (planned_learning_hours >= 0),
    planned_eep_hours INTEGER NOT NULL DEFAULT 0 CHECK (planned_eep_hours >= 0), -- Employability, Enrichment & Pastoral hours
    completion_status TEXT NOT NULL DEFAULT '1' CHECK (completion_status IN ('1', '2', '3', '4')), -- 1=Continuing, 2=Completed, 3=Withdrawn, 4=Transferred
    outcome_code TEXT NOT NULL DEFAULT '8' CHECK (outcome_code IN ('1', '2', '3', '8')),          -- 1=Achieved, 2=Partial, 3=No achievement, 8=Not yet known
    withdrawal_reason_code TEXT,
    maths_gcse_funding_exemption TEXT,                -- Condition of Funding exemption code
    english_gcse_funding_exemption TEXT,              -- Condition of Funding exemption code
    effective_from TEXT NOT NULL,                     -- Learning Aim start date
    planned_end_date TEXT NOT NULL,                   -- Learning Aim planned end date
    actual_end_date TEXT,                             -- Learning Aim actual end date
    CHECK (planned_end_date >= effective_from),
    CHECK (actual_end_date IS NULL OR actual_end_date >= effective_from)
);
CREATE INDEX IF NOT EXISTS idx_post16_aims_pupil ON pupil_post16_learning_aims(pupil_id, academic_year_id);
CREATE INDEX IF NOT EXISTS idx_post16_aims_year ON pupil_post16_learning_aims(academic_year_id);
CREATE INDEX IF NOT EXISTS idx_post16_aims_course ON pupil_post16_learning_aims(course_id);

-- @domain: medical
-- @dal_scope: dal.medical.read | dal.medical.write
-- @gibbon_ref: gibbonPersonMedical, gibbonPersonMedicalCondition, gibbonMedicalCondition
CREATE TABLE IF NOT EXISTS pupil_medical_conditions (
    id TEXT PRIMARY KEY,
    pupil_id TEXT NOT NULL REFERENCES pupils(id) ON DELETE RESTRICT,
    condition_name TEXT NOT NULL,                     -- e.g. 'Anaphylaxis (Peanut Allergy)', 'Asthma', 'Type 1 Diabetes'
    condition_category TEXT NOT NULL DEFAULT 'MEDICAL' CHECK (condition_category IN ('MEDICAL', 'ALLERGY', 'DIETARY')),
    alert_level_id TEXT REFERENCES alert_levels(id) ON DELETE SET NULL,
    has_ihcp INTEGER NOT NULL DEFAULT 0 CHECK (has_ihcp IN (0, 1)), -- Individual Healthcare Plan
    triggers TEXT NOT NULL DEFAULT '',
    reaction_symptoms TEXT NOT NULL DEFAULT '',
    emergency_response_plan TEXT NOT NULL DEFAULT '',
    medication_name_and_dose TEXT NOT NULL DEFAULT '',
    medication_location TEXT NOT NULL DEFAULT '',     -- e.g. 'Carried by pupil + Spare in Medical Room'
    parental_consent_given INTEGER NOT NULL DEFAULT 0 CHECK (parental_consent_given IN (0, 1)),
    effective_from TEXT NOT NULL,
    effective_to TEXT,
    CHECK (effective_to IS NULL OR effective_to >= effective_from)
);
CREATE INDEX IF NOT EXISTS idx_pupil_medical_pupil ON pupil_medical_conditions(pupil_id, effective_from);
CREATE INDEX IF NOT EXISTS idx_pupil_medical_alert ON pupil_medical_conditions(alert_level_id);

-- @domain: pastoral
-- @dal_scope: dal.pupils.read | dal.pupils.write
-- @description: General non-safeguarding pastoral notes (Safeguarding/Child Protection lives strictly in 06_safeguarding_vault.sql)
-- @gibbon_ref: gibbonStudentNote, gibbonStudentNoteCategory
CREATE TABLE IF NOT EXISTS pupil_pastoral_notes (
    id TEXT PRIMARY KEY,
    pupil_id TEXT NOT NULL REFERENCES pupils(id) ON DELETE RESTRICT,
    category TEXT NOT NULL DEFAULT 'GENERAL',         -- 'GENERAL', 'ACADEMIC_WELFARE', 'PARENT_CONTACT', 'TRANSITION'
    title TEXT NOT NULL,
    note_text TEXT NOT NULL,                          -- Redactable free-text field during GDPR SAR export
    created_by_staff_id TEXT NOT NULL REFERENCES staff(id) ON DELETE RESTRICT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);
CREATE INDEX IF NOT EXISTS idx_pupil_pastoral_pupil ON pupil_pastoral_notes(pupil_id, created_at);
CREATE INDEX IF NOT EXISTS idx_pupil_pastoral_staff ON pupil_pastoral_notes(created_by_staff_id);
