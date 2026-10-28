CREATE TABLE IF NOT EXISTS census_returns (
    id TEXT PRIMARY KEY,
    academic_year_id TEXT NOT NULL,
    term_code TEXT NOT NULL,
    census_date TEXT NOT NULL,
    school_urn TEXT NOT NULL,
    school_name TEXT NOT NULL,
    on_roll_total INTEGER NOT NULL DEFAULT 0,
    sen_k_count INTEGER NOT NULL DEFAULT 0,
    sen_e_count INTEGER NOT NULL DEFAULT 0,
    sen_n_count INTEGER NOT NULL DEFAULT 0,
    fsm_eligible_count INTEGER NOT NULL DEFAULT 0,
    lac_count INTEGER NOT NULL DEFAULT 0,
    total_possible_sessions INTEGER NOT NULL DEFAULT 0,
    total_present_sessions INTEGER NOT NULL DEFAULT 0,
    total_authorised_absence_sessions INTEGER NOT NULL DEFAULT 0,
    total_unauthorised_absence_sessions INTEGER NOT NULL DEFAULT 0,
    xsd_valid INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'VALIDATED',
    collect_submission_id TEXT NOT NULL DEFAULT '',
    xml_payload TEXT NOT NULL,
    generated_by_staff_id TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE TABLE IF NOT EXISTS census_pupil_snapshots (
    id TEXT PRIMARY KEY,
    return_id TEXT NOT NULL,
    pupil_id TEXT NOT NULL,
    academic_year_id TEXT NOT NULL,
    term_code TEXT NOT NULL,
    census_date TEXT NOT NULL,
    upn TEXT NOT NULL,
    nc_year_group TEXT NOT NULL,
    enrolment_status TEXT NOT NULL,
    sen_provision_code TEXT NOT NULL,
    primary_sen_need_code TEXT NOT NULL DEFAULT '',
    fsm_eligible INTEGER NOT NULL DEFAULT 0,
    lac_episode_status TEXT NOT NULL DEFAULT 'NONE',
    possible_sessions INTEGER NOT NULL DEFAULT 0,
    present_sessions INTEGER NOT NULL DEFAULT 0,
    authorised_absence_sessions INTEGER NOT NULL DEFAULT 0,
    unauthorised_absence_sessions INTEGER NOT NULL DEFAULT 0,
    notes TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);
CREATE INDEX IF NOT EXISTS idx_census_pupil_snapshots_pupil ON census_pupil_snapshots(pupil_id);
CREATE INDEX IF NOT EXISTS idx_census_pupil_snapshots_return ON census_pupil_snapshots(return_id);
