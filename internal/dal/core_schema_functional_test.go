package dal_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"

	_ "modernc.org/sqlite"
)

type schemaTestHarness struct {
	writeDB *sql.DB
	readDB  *sql.DB
	dbPath  string
	tempDir string
	logs    []string
	mu      sync.Mutex
}

func (h *schemaTestHarness) logf(format string, args ...any) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.logs = append(h.logs, fmt.Sprintf(format, args...))
}

func setupCoreSchemaHarness(t *testing.T) *schemaTestHarness {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "optimis_core_schema_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tempDir, "core.db")
	dsn := fmt.Sprintf(
		"file:%s?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)",
		dbPath,
	)

	writeDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("failed to open WriteDB: %v", err)
	}
	writeDB.SetMaxOpenConns(1)
	writeDB.SetMaxIdleConns(1)

	readDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		_ = writeDB.Close()
		os.RemoveAll(tempDir)
		t.Fatalf("failed to open ReadDB: %v", err)
	}
	numCPU := runtime.NumCPU()
	if numCPU < 2 {
		numCPU = 2
	}
	readDB.SetMaxOpenConns(numCPU)

	schemaDir, err := filepath.Abs(filepath.Join("..", "..", "docs", "schema", "core"))
	if err != nil {
		_ = readDB.Close()
		_ = writeDB.Close()
		os.RemoveAll(tempDir)
		t.Fatalf("failed to resolve docs/schema/core path: %v", err)
	}

	entries, err := os.ReadDir(schemaDir)
	if err != nil {
		_ = readDB.Close()
		_ = writeDB.Close()
		os.RemoveAll(tempDir)
		t.Fatalf("failed to read schema dir %s: %v", schemaDir, err)
	}

	var sqlFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			sqlFiles = append(sqlFiles, filepath.Join(schemaDir, e.Name()))
		}
	}
	sort.Strings(sqlFiles)
	if len(sqlFiles) != 7 {
		_ = readDB.Close()
		_ = writeDB.Close()
		os.RemoveAll(tempDir)
		t.Fatalf("expected 7 modular core schema SQL files in %s, found %d", schemaDir, len(sqlFiles))
	}

	harness := &schemaTestHarness{
		writeDB: writeDB,
		readDB:  readDB,
		dbPath:  dbPath,
		tempDir: tempDir,
	}

	for _, file := range sqlFiles {
		ddlBytes, err := os.ReadFile(file)
		if err != nil {
			_ = readDB.Close()
			_ = writeDB.Close()
			os.RemoveAll(tempDir)
			t.Fatalf("failed to read %s: %v", file, err)
		}
		if _, err := writeDB.Exec(string(ddlBytes)); err != nil {
			_ = readDB.Close()
			_ = writeDB.Close()
			os.RemoveAll(tempDir)
			t.Fatalf("failed to execute DDL in %s: %v", filepath.Base(file), err)
		}
		harness.logf("loaded schema module: %s", filepath.Base(file))
	}

	t.Cleanup(func() {
		if t.Failed() {
			harness.mu.Lock()
			t.Logf("--- TAPPED SCHEMA HARNESS LOGS ---\n%s", strings.Join(harness.logs, "\n"))
			harness.mu.Unlock()
		}
		_ = readDB.Close()
		_ = writeDB.Close()
		_ = os.RemoveAll(tempDir)
	})

	return harness
}

func seedBaseSchoolFixtures(t *testing.T, h *schemaTestHarness) {
	t.Helper()
	stmts := []string{
		`INSERT INTO academic_years (id, name, status, sequence_number, first_day, last_day)
		 VALUES ('ay_2026_2027', '2026/2027', 'CURRENT', 1, '2026-09-01', '2027-07-23')`,
		`INSERT INTO academic_terms (id, academic_year_id, code, name, name_short, sequence_number, first_day, last_day)
		 VALUES ('term_aut_26', 'ay_2026_2027', 'AUTUMN', 'Autumn Term 2026', 'AUT', 1, '2026-09-01', '2026-12-18')`,
		`INSERT INTO days_of_week (id, name, name_short, is_school_day, school_open_time, am_register_time, pm_register_time, school_close_time)
		 VALUES (1, 'Monday', 'Mon', 1, '08:00', '08:45', '13:15', '15:30')`,
		`INSERT INTO alert_levels (id, name, name_short, color_hex, bg_color_hex, description, sequence_number)
		 VALUES ('HIGH', 'High Priority', 'HI', '#FFFFFF', '#D32F2F', 'Critical medical or SEN alert', 1)`,
		`INSERT INTO persons (id, legal_first_name, legal_last_name, sex, dob, status)
		 VALUES ('per_pupil_1', 'Ada', 'Lovelace', 'F', '2014-12-10', 'ACTIVE'),
		        ('per_teacher_1', 'Grace', 'Hopper', 'F', '1985-05-01', 'ACTIVE'),
		        ('per_dsl_1', 'Margaret', 'Hamilton', 'F', '1980-08-17', 'ACTIVE'),
		        ('per_mother_1', 'Anne', 'Byron', 'F', '1988-03-14', 'ACTIVE'),
		        ('per_father_1', 'George', 'Byron', 'M', '1987-01-22', 'ACTIVE')`,
		`INSERT INTO pupils (id, person_id, upn, ulN, ethnicity_code, first_language_code)
		 VALUES ('pup_1', 'per_pupil_1', 'H823400026001', '1234567890', 'WBRI', 'ENG')`,
		`INSERT INTO staff (id, person_id, staff_code, trn, staff_type, job_title, qts_flag, is_dsl, hire_date)
		 VALUES ('stf_teacher_1', 'per_teacher_1', 'GHO', '7654321', 'TEACHING', 'Teacher of Mathematics', 1, 0, '2020-09-01'),
		        ('stf_dsl_1', 'per_dsl_1', 'MHA', '1122334', 'LEADERSHIP', 'Designated Safeguarding Lead', 1, 1, '2018-09-01')`,
	}

	for _, q := range stmts {
		if _, err := h.writeDB.Exec(q); err != nil {
			t.Fatalf("failed to seed base fixture query (%s): %v", q, err)
		}
	}
	h.logf("seeded base school fixtures for ay_2026_2027")
}

// 1. Safeguarding & Court-Order Invariant Tests
func TestCreateGuardianRelationship_CourtOrderRestrictedWithContactEnabled_ThrowsConstraintError_test(t *testing.T) {
	h := setupCoreSchemaHarness(t)
	seedBaseSchoolFixtures(t, h)

	// Attempting to insert a court_order_restricted=1 guardian who still has contact_by_sms=1 or receive_reports=1 MUST fail
	invalidInserts := []struct {
		name  string
		query string
	}{
		{
			name: "court_order_with_sms_enabled",
			query: `INSERT INTO guardian_relationships (
				id, pupil_id, guardian_person_id, relationship_code,
				has_parental_responsibility, court_order_restricted,
				portal_data_access, authorised_to_collect, is_emergency_contact,
				receive_reports, contact_by_phone, contact_by_sms, contact_by_email, contact_by_post,
				effective_from
			) VALUES ('grel_bad_1', 'pup_1', 'per_father_1', 'FATHER', 0, 1, 0, 0, 0, 0, 0, 1, 0, 0, '2026-09-01')`,
		},
		{
			name: "court_order_with_reports_enabled",
			query: `INSERT INTO guardian_relationships (
				id, pupil_id, guardian_person_id, relationship_code,
				has_parental_responsibility, court_order_restricted,
				portal_data_access, authorised_to_collect, is_emergency_contact,
				receive_reports, contact_by_phone, contact_by_sms, contact_by_email, contact_by_post,
				effective_from
			) VALUES ('grel_bad_2', 'pup_1', 'per_father_1', 'FATHER', 0, 1, 0, 0, 0, 1, 0, 0, 0, 0, '2026-09-01')`,
		},
	}

	for _, tc := range invalidInserts {
		if _, err := h.writeDB.Exec(tc.query); err == nil {
			t.Fatalf("expected SQLite CHECK constraint failure for %s, but insert succeeded", tc.name)
		}
	}

	// Valid court-order-restricted row with all 8 access/contact flags zeroed out MUST succeed
	validQuery := `INSERT INTO guardian_relationships (
		id, pupil_id, guardian_person_id, relationship_code,
		has_parental_responsibility, court_order_restricted, court_order_notes,
		portal_data_access, authorised_to_collect, is_emergency_contact,
		receive_reports, contact_by_phone, contact_by_sms, contact_by_email, contact_by_post,
		effective_from
	) VALUES (
		'grel_restricted_ok', 'pup_1', 'per_father_1', 'FATHER',
		0, 1, 'Prohibited Steps Order - Do Not Contact',
		0, 0, 0, 0, 0, 0, 0, 0,
		'2026-09-01'
	)`
	if _, err := h.writeDB.Exec(validQuery); err != nil {
		t.Fatalf("expected valid court-order restricted insert to succeed, got: %v", err)
	}
}

func TestGetGuardianContacts_SeparatedParentsWithConfidentialAddress_ExcludesConfidentialHousehold_test(t *testing.T) {
	h := setupCoreSchemaHarness(t)
	seedBaseSchoolFixtures(t, h)

	_, err := h.writeDB.Exec(`
		INSERT INTO households (id, name, address_line_1, town_city, postcode, is_confidential_address)
		VALUES ('hh_refuge', 'Confidential Household', '10 Safe Haven Way', 'London', 'SW1A 1AA', 1),
		       ('hh_father', 'Byron Household', '42 High Street', 'London', 'SW1A 2BB', 0);

		INSERT INTO household_memberships (id, household_id, person_id, member_role, residency_type, effective_from)
		VALUES ('hm_1', 'hh_refuge', 'per_mother_1', 'ADULT', 'PRIMARY', '2026-09-01'),
		       ('hm_2', 'hh_refuge', 'per_pupil_1', 'CHILD', 'PRIMARY', '2026-09-01'),
		       ('hm_3', 'hh_father', 'per_father_1', 'ADULT', 'SECONDARY_SHARED', '2026-09-01'),
		       ('hm_4', 'hh_father', 'per_pupil_1', 'CHILD', 'SECONDARY_SHARED', '2026-09-01');
	`)
	if err != nil {
		t.Fatalf("failed to seed households: %v", err)
	}

	// Non-confidential address query for parent portal / general staff
	rows, err := h.readDB.Query(`
		SELECT h.id, h.postcode
		FROM household_memberships hm
		JOIN households h ON h.id = hm.household_id
		WHERE hm.person_id = 'per_pupil_1'
		  AND h.is_confidential_address = 0
		  AND (hm.effective_to IS NULL OR hm.effective_to >= '2026-10-27')
	`)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	defer rows.Close()

	var returnedHouseholds []string
	for rows.Next() {
		var id, postcode string
		if err := rows.Scan(&id, &postcode); err != nil {
			t.Fatalf("scan failed: %v", err)
		}
		returnedHouseholds = append(returnedHouseholds, id)
	}

	if len(returnedHouseholds) != 1 || returnedHouseholds[0] != "hh_father" {
		t.Fatalf("expected only non-confidential household [hh_father], got %v", returnedHouseholds)
	}
}

func TestSafeguardingVault_UpdateOrDeleteAuditLogAndPrematureNoteDelete_ThrowsTriggerAbort_test(t *testing.T) {
	h := setupCoreSchemaHarness(t)
	seedBaseSchoolFixtures(t, h)

	_, err := h.writeDB.Exec(`
		INSERT INTO safeguarding_notes (
			id, pupil_id, reported_by_staff_id, assigned_dsl_staff_id,
			concern_category, status, incident_date,
			kms_key_id, wrapped_dek, nonce, ciphertext_payload, retention_purge_eligible
		) VALUES (
			'sfg_1', 'pup_1', 'stf_teacher_1', 'stf_dsl_1',
			'NEGLECT', 'OPEN_TRIAGE', '2026-10-27',
			'kms_dsl_key_1', X'01020304', X'05060708090A0B0C0D0E0F10', X'DEADBEEF', 0
		);

		INSERT INTO safeguarding_audit_logs (
			audit_id, note_id, pupil_id, actor_person_id, action_type, request_id, client_ip
		) VALUES (
			'aud_1', 'sfg_1', 'pup_1', 'per_teacher_1', 'CREATE_POSTBOX', 'req_999', '10.0.0.12'
		);
	`)
	if err != nil {
		t.Fatalf("failed to insert safeguarding note and audit log: %v", err)
	}

	// 1. Updating safeguarding_audit_logs MUST abort via trigger
	if _, err := h.writeDB.Exec(`UPDATE safeguarding_audit_logs SET action_type = 'SAR_REVIEW' WHERE audit_id = 'aud_1'`); err == nil {
		t.Fatalf("expected BEFORE UPDATE trigger on safeguarding_audit_logs to abort, but update succeeded")
	}

	// 2. Deleting safeguarding_audit_logs MUST abort via trigger
	if _, err := h.writeDB.Exec(`DELETE FROM safeguarding_audit_logs WHERE audit_id = 'aud_1'`); err == nil {
		t.Fatalf("expected BEFORE DELETE trigger on safeguarding_audit_logs to abort, but delete succeeded")
	}

	// 3. Deleting safeguarding_notes when retention_purge_eligible=0 MUST abort via trigger
	if _, err := h.writeDB.Exec(`DELETE FROM safeguarding_notes WHERE id = 'sfg_1'`); err == nil {
		t.Fatalf("expected BEFORE DELETE trigger on safeguarding_notes to abort before retention expiry, but delete succeeded")
	}
}

// 2. Temporal `:own_groups` RBAC & Mid-Year Set-Move Tests
func TestQueryOwnGroupsPupils_PupilTransferredBetweenTeachingSets_ResolvesCurrentAndHistoricalRosters_test(t *testing.T) {
	h := setupCoreSchemaHarness(t)
	seedBaseSchoolFixtures(t, h)

	_, err := h.writeDB.Exec(`
		INSERT INTO groups (id, academic_year_id, group_type, code, name, effective_from)
		VALUES ('grp_10ma1', 'ay_2026_2027', 'TEACHING_CLASS', '10Ma1', 'Year 10 Maths Set 1', '2026-09-01'),
		       ('grp_10ma2', 'ay_2026_2027', 'TEACHING_CLASS', '10Ma2', 'Year 10 Maths Set 2', '2026-09-01');

		-- Grace Hopper teaches 10Ma1 only
		INSERT INTO group_staff_assignments (id, group_id, staff_id, role, academic_year_id, effective_from)
		VALUES ('gsa_1', 'grp_10ma1', 'stf_teacher_1', 'LEAD_TEACHER', 'ay_2026_2027', '2026-09-01');

		-- Ada was in 10Ma1 from Sep 1 to Oct 15, then moved to 10Ma2 on Oct 16
		INSERT INTO group_pupil_memberships (id, group_id, pupil_id, academic_year_id, effective_from, effective_to)
		VALUES ('gpm_old', 'grp_10ma1', 'pup_1', 'ay_2026_2027', '2026-09-01', '2026-10-15'),
		       ('gpm_new', 'grp_10ma2', 'pup_1', 'ay_2026_2027', '2026-10-16', NULL);
	`)
	if err != nil {
		t.Fatalf("failed to seed group memberships: %v", err)
	}

	ownGroupsQuery := `
		SELECT p.id
		FROM pupils p
		JOIN group_pupil_memberships gpm ON gpm.pupil_id = p.id
		JOIN group_staff_assignments gsa ON gsa.group_id = gpm.group_id
		WHERE gsa.staff_id = ?
		  AND gpm.effective_from <= ? AND (gpm.effective_to IS NULL OR gpm.effective_to >= ?)
		  AND gsa.effective_from <= ? AND (gsa.effective_to IS NULL OR gsa.effective_to >= ?)
	`

	// Historical query on 2026-10-01: Teacher SHOULD see Ada in 10Ma1
	var foundID string
	err = h.readDB.QueryRow(ownGroupsQuery, "stf_teacher_1", "2026-10-01", "2026-10-01", "2026-10-01", "2026-10-01").Scan(&foundID)
	if err != nil || foundID != "pup_1" {
		t.Fatalf("expected teacher to have :own_groups access to pup_1 on 2026-10-01, got id=%q err=%v", foundID, err)
	}

	// Current query on 2026-10-27: Teacher MUST NOT see Ada in 10Ma1
	err = h.readDB.QueryRow(ownGroupsQuery, "stf_teacher_1", "2026-10-27", "2026-10-27", "2026-10-27", "2026-10-27").Scan(&foundID)
	if err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows for teacher :own_groups on 2026-10-27 after set transfer, got err=%v", err)
	}
}

func TestInsertTemporalRecords_EffectiveToBeforeEffectiveFrom_ThrowsCheckConstraintError_test(t *testing.T) {
	h := setupCoreSchemaHarness(t)
	seedBaseSchoolFixtures(t, h)

	_, err := h.writeDB.Exec(`
		INSERT INTO pupil_year_enrolments (
			id, pupil_id, academic_year_id, nc_year_group, effective_from, effective_to
		) VALUES ('enr_inverted', 'pup_1', 'ay_2026_2027', '7', '2026-09-01', '2026-08-01')
	`)
	if err == nil {
		t.Fatalf("expected CHECK constraint failure when effective_to < effective_from, but insert succeeded")
	}
}

// 3. Statutory AM/PM Attendance & Timetable-Edit Protection Tests
func TestDeleteTimetableSlotAndRecordSessionAttendance_ExistingMarksAndDuplicateAMSession_EnforcesRestrictAndUniqueAMPM_test(t *testing.T) {
	h := setupCoreSchemaHarness(t)
	seedBaseSchoolFixtures(t, h)

	_, err := h.writeDB.Exec(`
		INSERT INTO groups (id, academic_year_id, group_type, code, name, effective_from)
		VALUES ('grp_7a', 'ay_2026_2027', 'REGISTRATION_FORM', '7A', 'Form 7A', '2026-09-01');

		INSERT INTO timetable_periods (
			id, academic_year_id, cycle_week, day_of_week_id, period_code, name,
			start_time, end_time, period_type, maps_to_statutory_session, sequence_number
		) VALUES (
			'ttp_mon_am', 'ay_2026_2027', 1, 1, 'AM_REG', 'Morning Registration',
			'08:45', '09:00', 'STATUTORY_AM_REG', 'AM', 1
		);

		INSERT INTO timetable_slots (id, timetable_period_id, group_id, staff_id, effective_from)
		VALUES ('slot_mon_am_7a', 'ttp_mon_am', 'grp_7a', 'stf_teacher_1', '2026-09-01');

		INSERT INTO attendance_codes (code, name, statistical_category, physical_status, sequence_number)
		VALUES ('/', 'Present (AM)', 'PRESENT', 'ONSITE', 1),
		       ('L', 'Late before register closed', 'PRESENT', 'ONSITE_LATE', 2);

		INSERT INTO attendance_session_marks (
			id, pupil_id, academic_year_id, date, session, attendance_code, source_group_id, recorded_by_staff_id
		) VALUES (
			'asm_1', 'pup_1', 'ay_2026_2027', '2026-10-27', 'AM', '/', 'grp_7a', 'stf_teacher_1'
		);

		INSERT INTO attendance_lesson_marks (
			id, pupil_id, timetable_slot_id, group_id, date, attendance_code, recorded_by_staff_id
		) VALUES (
			'alm_1', 'pup_1', 'slot_mon_am_7a', 'grp_7a', '2026-10-27', '/', 'stf_teacher_1'
		);
	`)
	if err != nil {
		t.Fatalf("failed to seed attendance & timetable records: %v", err)
	}

	// 1. Deleting timetable_slots when attendance_lesson_marks exist MUST be blocked by ON DELETE RESTRICT
	if _, err := h.writeDB.Exec(`DELETE FROM timetable_slots WHERE id = 'slot_mon_am_7a'`); err == nil {
		t.Fatalf("expected ON DELETE RESTRICT to block deleting timetable_slot with recorded lesson attendance")
	}

	// 2. Inserting a non-AM/PM session code MUST be blocked by CHECK constraint
	if _, err := h.writeDB.Exec(`
		INSERT INTO attendance_session_marks (id, pupil_id, academic_year_id, date, session, attendance_code)
		VALUES ('asm_bad_sess', 'pup_1', 'ay_2026_2027', '2026-10-27', 'LUNCH', '/')
	`); err == nil {
		t.Fatalf("expected CHECK (session IN ('AM','PM')) to reject 'LUNCH'")
	}

	// 3. Duplicate (pupil_id, date, session) without ON CONFLICT upsert MUST be blocked by UNIQUE constraint
	if _, err := h.writeDB.Exec(`
		INSERT INTO attendance_session_marks (id, pupil_id, academic_year_id, date, session, attendance_code)
		VALUES ('asm_dup', 'pup_1', 'ay_2026_2027', '2026-10-27', 'AM', 'L')
	`); err == nil {
		t.Fatalf("expected UNIQUE (pupil_id, date, session) to reject duplicate AM mark")
	}
}

// 4. Termly DfE School Census Point-in-Time Snapshot Test
func TestGenerateCensusSnapshot_PointInTimeReferenceDate_ReturnsExactActiveRollFsmSenExclusionsApAndPost16_test(t *testing.T) {
	h := setupCoreSchemaHarness(t)
	seedBaseSchoolFixtures(t, h)

	// Autumn Census Reference Date: 2026-10-01
	_, err := h.writeDB.Exec(`
		INSERT INTO pupil_year_enrolments (
			id, pupil_id, academic_year_id, nc_year_group, enrolment_status,
			ey_funded_hours, effective_from, effective_to
		) VALUES ('enr_1', 'pup_1', 'ay_2026_2027', '12', 'C', 0.0, '2026-09-01', NULL);

		-- FSM period 1 expired before Census Day; FSM period 2 is active on Census Day
		INSERT INTO pupil_fsm_periods (id, pupil_id, effective_from, effective_to)
		VALUES ('fsm_old', 'pup_1', '2025-09-01', '2026-07-20'),
		       ('fsm_active', 'pup_1', '2026-09-01', NULL);

		-- SEN was 'K' (SEN Support) on Census Day (2026-10-01), then upgraded to 'E' (EHCP) on 2026-10-20
		INSERT INTO pupil_sen_records (id, pupil_id, provision_code, effective_from, effective_to)
		VALUES ('sen_k', 'pup_1', 'K', '2026-09-01', '2026-10-19'),
		       ('sen_e', 'pup_1', 'E', '2026-10-20', NULL);

		INSERT INTO sen_need_types (code, name, broad_area, sequence_number)
		VALUES ('ASD', 'Autistic Spectrum Disorder', 'COMMUNICATION_AND_INTERACTION', 1);

		INSERT INTO pupil_sen_needs (id, pupil_id, sen_need_code, ranking, alert_level_id, effective_from)
		VALUES ('sneed_1', 'pup_1', 'ASD', 1, 'HIGH', '2026-09-01');

		INSERT INTO pupil_suspensions_exclusions (
			id, pupil_id, academic_year_id, exclusion_category, reason_code_1,
			start_date, end_date, sessions_missed, sen_provision_at_exclusion
		) VALUES (
			'exc_1', 'pup_1', 'ay_2026_2027', 'SUSPENSION', 'DB',
			'2026-09-15', '2026-09-16', 4, 'K'
		);

		INSERT INTO pupil_alternative_provisions (
			id, pupil_id, academic_year_id, provider_urn, ap_reason_code,
			sen_provision_during_ap, sessions_per_week, effective_from, effective_to
		) VALUES (
			'ap_1', 'pup_1', 'ay_2026_2027', '139999', 'MH', 'K', 4, '2026-09-20', '2026-12-18'
		);

		INSERT INTO pupil_post16_learning_aims (
			id, pupil_id, academic_year_id, qn_code, subject_classification_code,
			planned_learning_hours, planned_eep_hours, completion_status, outcome_code,
			effective_from, planned_end_date
		) VALUES (
			'aim_1', 'pup_1', 'ay_2026_2027', '6031333X', 'RB10',
			540, 60, '1', '8', '2026-09-01', '2028-06-30'
		);
	`)
	if err != nil {
		t.Fatalf("failed to seed census snapshot records: %v", err)
	}

	censusDate := "2026-10-01"
	var upn, ncYear, senProvision, fsmID, qnCode string
	err = h.readDB.QueryRow(`
		SELECT
			p.upn,
			enr.nc_year_group,
			sen.provision_code,
			fsm.id,
			aim.qn_code
		FROM pupils p
		JOIN pupil_year_enrolments enr ON enr.pupil_id = p.id
		JOIN pupil_sen_records sen ON sen.pupil_id = p.id
		JOIN pupil_fsm_periods fsm ON fsm.pupil_id = p.id
		JOIN pupil_post16_learning_aims aim ON aim.pupil_id = p.id
		WHERE enr.effective_from <= ? AND (enr.effective_to IS NULL OR enr.effective_to >= ?)
		  AND sen.effective_from <= ? AND (sen.effective_to IS NULL OR sen.effective_to >= ?)
		  AND fsm.effective_from <= ? AND (fsm.effective_to IS NULL OR fsm.effective_to >= ?)
	`, censusDate, censusDate, censusDate, censusDate, censusDate, censusDate).Scan(&upn, &ncYear, &senProvision, &fsmID, &qnCode)
	if err != nil {
		t.Fatalf("census point-in-time query failed: %v", err)
	}

	if senProvision != "K" {
		t.Fatalf("expected point-in-time SEN provision on 2026-10-01 to be 'K' (before Oct 20 EHCP upgrade), got %q", senProvision)
	}
	if fsmID != "fsm_active" {
		t.Fatalf("expected active FSM period 'fsm_active', got %q", fsmID)
	}
	if qnCode != "6031333X" {
		t.Fatalf("expected Post-16 Learning Aim QN '6031333X', got %q", qnCode)
	}
}

// 5. UK GDPR Statutory Retention Floor Protection Test
func TestPurgePupilData_UnderAge31WithSenRecord_BlocksCoreDeletionViaRestrict_test(t *testing.T) {
	h := setupCoreSchemaHarness(t)
	seedBaseSchoolFixtures(t, h)

	_, err := h.writeDB.Exec(`
		INSERT INTO pupil_sen_records (id, pupil_id, provision_code, effective_from)
		VALUES ('sen_retention_lock', 'pup_1', 'E', '2026-09-01');
	`)
	if err != nil {
		t.Fatalf("failed to seed SEN record: %v", err)
	}

	// Attempting to delete the pupil while statutory SEN records remain MUST fail via ON DELETE RESTRICT
	if _, err := h.writeDB.Exec(`DELETE FROM pupils WHERE id = 'pup_1'`); err == nil {
		t.Fatalf("expected ON DELETE RESTRICT to block deleting pupil while statutory SEN record exists")
	}
}

// 6. DfE Common Transfer File (CTF) Atomic Import Saga Rollback Test
func TestImportCtfStudentBundle_InvalidSenProvisionCodeMidTransaction_RollsBackEntireSaga_test(t *testing.T) {
	h := setupCoreSchemaHarness(t)
	seedBaseSchoolFixtures(t, h)

	ctx := context.Background()
	tx, err := h.writeDB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("failed to begin CTF import transaction: %v", err)
	}

	_, err = tx.Exec(`
		INSERT INTO persons (id, legal_first_name, legal_last_name, sex, dob, status)
		VALUES ('per_ctf_new', 'Charles', 'Babbage', 'M', '2015-12-26', 'ACTIVE');
		INSERT INTO pupils (id, person_id, upn, ethnicity_code, first_language_code)
		VALUES ('pup_ctf_new', 'per_ctf_new', 'H823400026999', 'WBRI', 'ENG');
	`)
	if err != nil {
		_ = tx.Rollback()
		t.Fatalf("unexpected error in step 1-2 of CTF saga: %v", err)
	}

	// Step 3: Corrupted SEN provision code 'Z' (valid values are 'N', 'K', 'E')
	_, err = tx.Exec(`
		INSERT INTO pupil_sen_records (id, pupil_id, provision_code, effective_from)
		VALUES ('sen_ctf_bad', 'pup_ctf_new', 'Z', '2026-09-01')
	`)
	if err == nil {
		_ = tx.Rollback()
		t.Fatalf("expected invalid SEN provision_code 'Z' to fail CHECK constraint")
	}

	if rbErr := tx.Rollback(); rbErr != nil {
		t.Fatalf("failed to rollback transaction: %v", rbErr)
	}

	var count int
	if err := h.readDB.QueryRow(`SELECT COUNT(*) FROM persons WHERE id = 'per_ctf_new'`).Scan(&count); err != nil {
		t.Fatalf("failed to check person count after rollback: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 orphan persons rows after CTF saga rollback, found %d", count)
	}
}

// 7. SQLite WAL Concurrency & Attached Plugin DB Test
func TestConcurrentWalReadAndWrite_SingleWriterAndPooledReadersWithAttachedPlugin_ZeroSqliteBusyErrors_test(t *testing.T) {
	h := setupCoreSchemaHarness(t)
	seedBaseSchoolFixtures(t, h)

	_, err := h.writeDB.Exec(`
		INSERT INTO attendance_codes (code, name, statistical_category, physical_status, sequence_number)
		VALUES ('/', 'Present (AM)', 'PRESENT', 'ONSITE', 1),
		       ('\', 'Present (PM)', 'PRESENT', 'ONSITE', 2);
	`)
	if err != nil {
		t.Fatalf("failed to seed attendance codes: %v", err)
	}

	// Attach an isolated plugin_*.db file on WriteDB and verify cross-DB isolation
	pluginDBPath := filepath.Join(h.tempDir, "plugin_behaviour.db")
	if _, err := h.writeDB.Exec(fmt.Sprintf("ATTACH DATABASE '%s' AS plugin_behaviour;", pluginDBPath)); err != nil {
		t.Fatalf("failed to attach plugin_behaviour.db: %v", err)
	}
	if _, err := h.writeDB.Exec(`
		CREATE TABLE plugin_behaviour.behaviour_points (
			id TEXT PRIMARY KEY,
			pupil_id TEXT NOT NULL,
			points INTEGER NOT NULL
		);
		INSERT INTO plugin_behaviour.behaviour_points (id, pupil_id, points) VALUES ('bp_1', 'pup_1', 5);
	`); err != nil {
		t.Fatalf("failed to create/seed plugin_behaviour table: %v", err)
	}

	var wg sync.WaitGroup
	errCh := make(chan error, 100)

	// 1 Writer goroutine inserting 60 attendance session marks sequentially via WriteDB
	wg.Add(1)
	go func() {
		defer wg.Done()
		for day := 1; day <= 30; day++ {
			dateStr := fmt.Sprintf("2026-09-%02d", day)
			for _, sess := range []string{"AM", "PM"} {
				code := "/"
				if sess == "PM" {
					code = "\\"
				}
				id := fmt.Sprintf("asm_%d_%s", day, sess)
				if _, err := h.writeDB.Exec(
					`INSERT INTO attendance_session_marks (id, pupil_id, academic_year_id, date, session, attendance_code)
					 VALUES (?, 'pup_1', 'ay_2026_2027', ?, ?, ?)`,
					id, dateStr, sess, code,
				); err != nil {
					errCh <- fmt.Errorf("writer failed on %s: %w", id, err)
					return
				}
			}
		}
	}()

	// 6 Concurrent Reader goroutines querying ReadDB while WriteDB is writing
	for r := 0; r < 6; r++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()
			for i := 0; i < 25; i++ {
				var fkEnabled int
				if err := h.readDB.QueryRow(`PRAGMA foreign_keys;`).Scan(&fkEnabled); err != nil {
					errCh <- fmt.Errorf("reader %d pragma check failed: %w", readerID, err)
					return
				}
				if fkEnabled != 1 {
					errCh <- fmt.Errorf("reader %d pooled connection has foreign_keys=%d (expected 1)", readerID, fkEnabled)
					return
				}

				var markCount int
				if err := h.readDB.QueryRow(`
					SELECT COUNT(*)
					FROM attendance_session_marks asm
					JOIN pupils p ON p.id = asm.pupil_id
					JOIN persons per ON per.id = p.person_id
					WHERE p.upn = 'H823400026001'
				`).Scan(&markCount); err != nil {
					errCh <- fmt.Errorf("reader %d query failed: %w", readerID, err)
					return
				}
			}
		}(r)
	}

	wg.Wait()
	close(errCh)

	for e := range errCh {
		t.Fatalf("concurrent WAL test error: %v", e)
	}
}
