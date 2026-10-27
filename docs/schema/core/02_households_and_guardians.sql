-- ============================================================================
-- Optimis Core Schema: 02_households_and_guardians.sql
-- Domain: Households, Postal Addresses (BS7666/CBDS), Guardians & Court Orders
-- Engine: SQLite 3 (WAL mode, Foreign Keys ON)
-- Mandatory Optimis DAL Invariants (GEMINI.md §3):
--   1. Every guardian relationship explicitly tracks `has_parental_responsibility`
--      and `court_order_restricted`.
--   2. When `court_order_restricted = 1`, a strict SQL `CHECK` constraint
--      zeroes out ALL portal, collection, emergency, report, and contact flags.
--   3. `households.is_confidential_address` protects separated-parent refuge
--      or confidential addresses from leaking into SAR exports or parent portals.
-- Gibbon References: gibbonFamily, gibbonFamilyAdult, gibbonFamilyChild,
--                    gibbonFamilyRelationship, gibbonFamilyUpdate
-- ============================================================================

PRAGMA foreign_keys = ON;

-- @domain: guardians
-- @dal_scope: dal.guardians.read | dal.guardians.write
-- @dfe_cbds: N00185 (UPRN), BS7666 UK Address Structure (PAON, SAON, Street, Town, Postcode)
-- @gibbon_ref: gibbonFamily
CREATE TABLE IF NOT EXISTS households (
    id TEXT PRIMARY KEY,                              -- e.g. 'hh_...'
    name TEXT NOT NULL,                               -- e.g. 'Smith Household'
    formal_salutation TEXT NOT NULL DEFAULT '',       -- e.g. 'Mr & Mrs J. Smith'
    uprn TEXT,                                        -- UK Unique Property Reference Number (up to 12 digits)
    address_line_1 TEXT NOT NULL,                     -- PAON / SAON + Street
    address_line_2 TEXT NOT NULL DEFAULT '',
    locality TEXT NOT NULL DEFAULT '',
    town_city TEXT NOT NULL,
    county TEXT NOT NULL DEFAULT '',
    postcode TEXT NOT NULL,                           -- UK Postcode (DfE CBDS validated)
    country_code TEXT NOT NULL DEFAULT 'GBR',
    home_language_primary TEXT NOT NULL DEFAULT 'ENG',
    home_language_secondary TEXT NOT NULL DEFAULT '',
    is_confidential_address INTEGER NOT NULL DEFAULT 0 CHECK (is_confidential_address IN (0, 1)), -- Prevents separated-parent address leakage
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);
CREATE INDEX IF NOT EXISTS idx_households_postcode ON households(postcode);

-- @domain: guardians
-- @dal_scope: dal.guardians.read | dal.guardians.write
-- @gibbon_ref: gibbonFamilyAdult, gibbonFamilyChild
CREATE TABLE IF NOT EXISTS household_memberships (
    id TEXT PRIMARY KEY,
    household_id TEXT NOT NULL REFERENCES households(id) ON DELETE RESTRICT,
    person_id TEXT NOT NULL REFERENCES persons(id) ON DELETE RESTRICT,
    member_role TEXT NOT NULL CHECK (member_role IN ('ADULT', 'CHILD')),
    residency_type TEXT NOT NULL DEFAULT 'PRIMARY' CHECK (residency_type IN ('PRIMARY', 'SECONDARY_SHARED', 'TERM_TIME_ONLY', 'OTHER')),
    effective_from TEXT NOT NULL,
    effective_to TEXT,
    CHECK (effective_to IS NULL OR effective_to >= effective_from),
    UNIQUE (household_id, person_id, effective_from)
);
CREATE INDEX IF NOT EXISTS idx_household_memberships_person ON household_memberships(person_id, effective_from, effective_to);

-- @domain: guardians
-- @dal_scope: dal.guardians.read | dal.guardians.write
-- @dfe_cbds: Guardian Relationship, Parental Responsibility (PR), Court Order Restriction
-- @gibbon_ref: gibbonFamilyRelationship, gibbonFamilyAdult
CREATE TABLE IF NOT EXISTS guardian_relationships (
    id TEXT PRIMARY KEY,                              -- e.g. 'grel_...'
    pupil_id TEXT NOT NULL REFERENCES pupils(id) ON DELETE RESTRICT,
    guardian_person_id TEXT NOT NULL REFERENCES persons(id) ON DELETE RESTRICT,
    relationship_code TEXT NOT NULL,                  -- 'MOTHER', 'FATHER', 'STEP_PARENT', 'FOSTER_CARER', 'SOCIAL_WORKER', 'GRANDPARENT', 'OTHER'
    -- MANDATORY OPTIMIS CORE DAL INVARIANTS (GEMINI.md §3):
    has_parental_responsibility INTEGER NOT NULL DEFAULT 1 CHECK (has_parental_responsibility IN (0, 1)),
    court_order_restricted INTEGER NOT NULL DEFAULT 0 CHECK (court_order_restricted IN (0, 1)),
    court_order_notes TEXT NOT NULL DEFAULT '',       -- Details of restriction (visible only to Admin/DSL)
    contact_priority INTEGER NOT NULL DEFAULT 1 CHECK (contact_priority BETWEEN 1 AND 10),
    is_emergency_contact INTEGER NOT NULL DEFAULT 1 CHECK (is_emergency_contact IN (0, 1)),
    authorised_to_collect INTEGER NOT NULL DEFAULT 1 CHECK (authorised_to_collect IN (0, 1)),
    portal_data_access INTEGER NOT NULL DEFAULT 1 CHECK (portal_data_access IN (0, 1)),
    receive_reports INTEGER NOT NULL DEFAULT 1 CHECK (receive_reports IN (0, 1)),
    contact_by_phone INTEGER NOT NULL DEFAULT 1 CHECK (contact_by_phone IN (0, 1)),
    contact_by_sms INTEGER NOT NULL DEFAULT 1 CHECK (contact_by_sms IN (0, 1)),
    contact_by_email INTEGER NOT NULL DEFAULT 1 CHECK (contact_by_email IN (0, 1)),
    contact_by_post INTEGER NOT NULL DEFAULT 1 CHECK (contact_by_post IN (0, 1)),
    effective_from TEXT NOT NULL,
    effective_to TEXT,
    CHECK (effective_to IS NULL OR effective_to >= effective_from),
    -- Structural safeguard: A court-order-restricted person CANNOT have portal access, collection rights, emergency status, reports, or any outbound contact enabled
    CHECK (
        court_order_restricted = 0 OR (
            portal_data_access = 0
            AND authorised_to_collect = 0
            AND is_emergency_contact = 0
            AND receive_reports = 0
            AND contact_by_phone = 0
            AND contact_by_sms = 0
            AND contact_by_email = 0
            AND contact_by_post = 0
        )
    ),
    UNIQUE (pupil_id, guardian_person_id, effective_from)
);
CREATE INDEX IF NOT EXISTS idx_guardian_rel_pupil ON guardian_relationships(pupil_id, contact_priority);
CREATE INDEX IF NOT EXISTS idx_guardian_rel_guardian ON guardian_relationships(guardian_person_id);
