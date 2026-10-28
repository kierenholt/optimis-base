package dal_test

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"optimis/internal/dal"
)

type dalTestLogTap struct {
	mu      sync.Mutex
	entries []string
}

func (l *dalTestLogTap) Logf(format string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = append(l.entries, fmt.Sprintf(format, args...))
}

func (l *dalTestLogTap) Dump() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return strings.Join(l.entries, "\n")
}

type milestone1Fixtures struct {
	AcademicYearID           string
	ReferenceDate            string
	TeacherStaffID           string
	OtherTeacherStaffID      string
	DSLStaffID               string
	OwnGroupID               string
	OtherGroupID             string
	ActivePupilID            string
	TransferredPupilID       string
	OtherGroupPupilID        string
	UnrestrictedMotherPersID string
	RestrictedFatherPersID   string
	NonPRGrandparentPersID   string
	SafeHouseholdID          string
	ConfidentialHouseholdID  string
	TimetableSlotAMID        string
	TimetableSlotPMID        string
	HostKMSKeyHandle         string
}

func setupIsolatedCoreDALWithFixtures(t *testing.T) (*dal.CoreDAL, * milestone1Fixtures, *dalTestLogTap) {
	t.Helper()

	tempDir := t.TempDir()
	logTap := &dalTestLogTap{}

	coreDAL, err := dal.NewCoreDAL(tempDir, "100001")
	if err != nil {
		t.Fatalf("failed to create isolated CoreDAL: %v", err)
	}
	logTap.Logf("initialized isolated CoreDAL in %s", tempDir)

	fx := &milestone1Fixtures{
		AcademicYearID:           "ay_2026_2027",
		ReferenceDate:            "2026-10-27",
		TeacherStaffID:           "stf_teacher_1",
		OtherTeacherStaffID:      "stf_teacher_2",
		DSLStaffID:               "stf_dsl_1",
		OwnGroupID:               "grp_10ma1",
		OtherGroupID:             "grp_10ma2",
		ActivePupilID:            "pup_active_1",
		TransferredPupilID:       "pup_transferred_2",
		OtherGroupPupilID:        "pup_other_3",
		UnrestrictedMotherPersID: "per_mother_1",
		RestrictedFatherPersID:   "per_father_1",
		NonPRGrandparentPersID:   "per_grandparent_1",
		SafeHouseholdID:          "hh_safe_1",
		ConfidentialHouseholdID:  "hh_refuge_1",
		TimetableSlotAMID:        "slot_wk1_mon_am",
		TimetableSlotPMID:        "slot_wk2_mon_pm",
		HostKMSKeyHandle:         "kms-dsl-vault-key-001",
	}

	seedQueries := []string{
		`INSERT INTO academic_years (id, name, status, sequence_number, first_day, last_day)
		 VALUES ('ay_2026_2027', '2026/2027', 'CURRENT', 1, '2026-09-01', '2027-07-23')`,
		`INSERT INTO academic_terms (id, academic_year_id, code, name, name_short, sequence_number, first_day, last_day)
		 VALUES ('term_aut_26', 'ay_2026_2027', 'AUTUMN', 'Autumn Term 2026', 'AUT', 1, '2026-09-01', '2026-12-18')`,
		`INSERT INTO days_of_week (id, name, name_short, is_school_day, school_open_time, am_register_time, pm_register_time, school_close_time)
		 VALUES (1, 'Monday', 'Mon', 1, '08:00', '08:45', '13:15', '15:30')`,
		`INSERT INTO alert_levels (id, name, name_short, color_hex, bg_color_hex, description, sequence_number)
		 VALUES ('HIGH', 'High Priority', 'HI', '#FFFFFF', '#D32F2F', 'Critical medical or SEN alert', 1),
		        ('MEDIUM', 'Medium Priority', 'MD', '#000000', '#FFA000', 'Moderate alert', 2)`,
		`INSERT INTO sen_need_types (code, name, broad_area, sequence_number)
		 VALUES ('ASD', 'Autistic Spectrum Disorder', 'COMMUNICATION_AND_INTERACTION', 1),
		        ('SPLD', 'Specific Learning Difficulty', 'COGNITION_AND_LEARNING', 2)`,
		`INSERT INTO attendance_codes (code, name, statistical_category, physical_status, sequence_number)
		 VALUES ('/', 'Present (AM)', 'PRESENT', 'ONSITE', 1),
		        ('\', 'Present (PM)', 'PRESENT', 'ONSITE', 2),
		        ('L', 'Late before register closed', 'PRESENT', 'ONSITE_LATE', 3),
		        ('I', 'Illness', 'AUTHORISED_ABSENCE', 'ABSENT', 4)`,
		`INSERT INTO persons (id, email, legal_first_name, legal_last_name, sex, dob, phone_primary, phone_primary_type, status)
		 VALUES ('per_pupil_1', 'ada@optimis.school', 'Ada', 'Lovelace', 'F', '2012-12-10', '', '', 'ACTIVE'),
		        ('per_pupil_2', 'charles@optimis.school', 'Charles', 'Babbage', 'M', '2012-10-15', '', '', 'ACTIVE'),
		        ('per_pupil_3', 'alan@optimis.school', 'Alan', 'Turing', 'M', '2012-06-23', '', '', 'ACTIVE'),
		        ('per_teacher_1', 'ghopper@optimis.school', 'Grace', 'Hopper', 'F', '1985-05-01', '', '', 'ACTIVE'),
		        ('per_teacher_2', 'kjohn@optimis.school', 'Katherine', 'Johnson', 'F', '1984-08-26', '', '', 'ACTIVE'),
		        ('per_dsl_1', 'mhamilton@optimis.school', 'Margaret', 'Hamilton', 'F', '1980-08-17', '', '', 'ACTIVE'),
		        ('per_mother_1', 'anne.byron@parent.org', 'Anne', 'Byron', 'F', '1988-03-14', '07700900111', 'MOBILE', 'ACTIVE'),
		        ('per_father_1', 'george.byron@parent.org', 'George', 'Byron', 'M', '1987-01-22', '07700900222', 'MOBILE', 'ACTIVE'),
		        ('per_grandparent_1', 'judith.milbanke@parent.org', 'Judith', 'Milbanke', 'F', '1960-04-10', '07700900333', 'MOBILE', 'ACTIVE')`,
		`INSERT INTO pupils (id, person_id, upn, uln, ethnicity_code, first_language_code)
		 VALUES ('pup_active_1', 'per_pupil_1', 'H823400026001', '1000000001', 'WBRI', 'ENG'),
		        ('pup_transferred_2', 'per_pupil_2', 'H823400026002', '1000000002', 'WBRI', 'ENG'),
		        ('pup_other_3', 'per_pupil_3', 'H823400026003', '1000000003', 'WBRI', 'ENG')`,
		`INSERT INTO pupil_year_enrolments (id, pupil_id, academic_year_id, nc_year_group, enrolment_status, effective_from, effective_to)
		 VALUES ('enr_1', 'pup_active_1', 'ay_2026_2027', '10', 'C', '2026-09-01', NULL),
		        ('enr_2', 'pup_transferred_2', 'ay_2026_2027', '10', 'C', '2026-09-01', NULL),
		        ('enr_3', 'pup_other_3', 'ay_2026_2027', '10', 'C', '2026-09-01', NULL)`,
		`INSERT INTO staff (id, person_id, staff_code, trn, staff_type, job_title, qts_flag, is_dsl, hire_date)
		 VALUES ('stf_teacher_1', 'per_teacher_1', 'GHO', '7654321', 'TEACHING', 'Teacher of Mathematics', 1, 0, '2020-09-01'),
		        ('stf_teacher_2', 'per_teacher_2', 'KJO', '7654322', 'TEACHING', 'Teacher of Mathematics', 1, 0, '2021-09-01'),
		        ('stf_dsl_1', 'per_dsl_1', 'MHA', '1122334', 'LEADERSHIP', 'Designated Safeguarding Lead', 1, 1, '2018-09-01')`,
		`INSERT INTO rooms (id, code, name, room_type, capacity)
		 VALUES ('rm_ma1', 'MA1', 'Maths Room 1', 'CLASSROOM', 30),
		        ('rm_ma2', 'MA2', 'Maths Room 2', 'CLASSROOM', 30)`,
		`INSERT INTO groups (id, academic_year_id, group_type, code, name, default_room_id, takes_attendance, effective_from, effective_to)
		 VALUES ('grp_10ma1', 'ay_2026_2027', 'TEACHING_CLASS', '10Ma1', 'Year 10 Maths Set 1', 'rm_ma1', 1, '2026-09-01', NULL),
		        ('grp_10ma2', 'ay_2026_2027', 'TEACHING_CLASS', '10Ma2', 'Year 10 Maths Set 2', 'rm_ma2', 1, '2026-09-01', NULL),
		        ('grp_expired', 'ay_2026_2027', 'INTERVENTION', '10IntOld', 'Expired Intervention', 'rm_ma1', 1, '2026-09-01', '2026-10-01')`,
		`INSERT INTO group_staff_assignments (id, group_id, staff_id, role, academic_year_id, effective_from, effective_to)
		 VALUES ('gsa_1', 'grp_10ma1', 'stf_teacher_1', 'LEAD_TEACHER', 'ay_2026_2027', '2026-09-01', NULL),
		        ('gsa_expired', 'grp_expired', 'stf_teacher_1', 'LEAD_TEACHER', 'ay_2026_2027', '2026-09-01', '2026-10-01'),
		        ('gsa_2', 'grp_10ma2', 'stf_teacher_2', 'LEAD_TEACHER', 'ay_2026_2027', '2026-09-01', NULL)`,
		`INSERT INTO group_pupil_memberships (id, group_id, pupil_id, academic_year_id, effective_from, effective_to)
		 VALUES ('gpm_active_1', 'grp_10ma1', 'pup_active_1', 'ay_2026_2027', '2026-09-01', NULL),
		        ('gpm_trans_old', 'grp_10ma1', 'pup_transferred_2', 'ay_2026_2027', '2026-09-01', '2026-10-15'),
		        ('gpm_trans_new', 'grp_10ma2', 'pup_transferred_2', 'ay_2026_2027', '2026-10-16', NULL),
		        ('gpm_other_3', 'grp_10ma2', 'pup_other_3', 'ay_2026_2027', '2026-09-01', NULL)`,
		`INSERT INTO timetable_periods (
			id, academic_year_id, cycle_week, day_of_week_id, period_code, name,
			start_time, end_time, period_type, maps_to_statutory_session, sequence_number
		) VALUES
			('ttp_wk1_mon_am', 'ay_2026_2027', 1, 1, 'P1_AM', 'Week 1 Mon Period 1 (AM)', '08:45', '09:45', 'LESSON', 'AM', 1),
			('ttp_wk2_mon_pm', 'ay_2026_2027', 2, 1, 'P4_PM', 'Week 2 Mon Period 4 (PM)', '13:15', '14:15', 'LESSON', 'PM', 2)`,
		`INSERT INTO timetable_slots (id, timetable_period_id, group_id, staff_id, room_id, effective_from, effective_to)
		 VALUES ('slot_wk1_mon_am', 'ttp_wk1_mon_am', 'grp_10ma1', 'stf_teacher_1', 'rm_ma1', '2026-09-01', NULL),
		        ('slot_wk2_mon_pm', 'ttp_wk2_mon_pm', 'grp_10ma1', 'stf_teacher_1', 'rm_ma1', '2026-09-01', NULL)`,
		`INSERT INTO pupil_sen_records (id, pupil_id, provision_code, ehcp_start_date, teaching_strategies, effective_from, effective_to)
		 VALUES ('sen_hist_1', 'pup_active_1', 'K', NULL, 'Historical SEN Support', '2026-09-01', '2026-10-10'),
		        ('sen_curr_1', 'pup_active_1', 'E', '2026-10-11', 'Chunk instructions and provide visual timetable', '2026-10-11', NULL)`,
		`INSERT INTO pupil_sen_needs (id, pupil_id, sen_need_code, ranking, alert_level_id, effective_from, effective_to)
		 VALUES ('sneed_curr_1', 'pup_active_1', 'ASD', 1, 'HIGH', '2026-09-01', NULL)`,
		`INSERT INTO pupil_medical_conditions (
			id, pupil_id, condition_name, condition_category, alert_level_id,
			has_ihcp, emergency_response_plan, parental_consent_given, effective_from, effective_to
		) VALUES
			('med_expired_1', 'pup_active_1', 'Resolved Childhood Rash', 'MEDICAL', 'MEDIUM', 0, 'None', 1, '2025-09-01', '2026-08-31'),
			('med_active_1', 'pup_active_1', 'Anaphylaxis (Peanut Allergy)', 'ALLERGY', 'HIGH', 1, 'Administer EpiPen immediately and call 999', 1, '2026-09-01', NULL)`,
		`INSERT INTO households (id, name, address_line_1, town_city, postcode, is_confidential_address)
		 VALUES ('hh_safe_1', 'Byron Household', '42 High Street', 'London', 'SW1A 2BB', 0),
		        ('hh_refuge_1', 'Confidential Refuge', '10 Safe Haven Way', 'London', 'SW1A 1AA', 1)`,
		`INSERT INTO household_memberships (id, household_id, person_id, member_role, residency_type, effective_from, effective_to)
		 VALUES ('hm_mother_safe', 'hh_safe_1', 'per_mother_1', 'ADULT', 'PRIMARY', '2026-09-01', NULL),
		        ('hm_pupil_safe', 'hh_safe_1', 'per_pupil_1', 'CHILD', 'PRIMARY', '2026-09-01', NULL),
		        ('hm_grandparent_refuge', 'hh_refuge_1', 'per_grandparent_1', 'ADULT', 'SECONDARY_SHARED', '2026-09-01', NULL)`,
		`INSERT INTO guardian_relationships (
			id, pupil_id, guardian_person_id, relationship_code,
			has_parental_responsibility, court_order_restricted, court_order_notes,
			contact_priority, is_emergency_contact, authorised_to_collect,
			portal_data_access, receive_reports, contact_by_phone, contact_by_sms, contact_by_email, contact_by_post,
			effective_from, effective_to
		) VALUES
			('grel_mother', 'pup_active_1', 'per_mother_1', 'MOTHER', 1, 0, '', 1, 1, 1, 1, 1, 1, 1, 1, 1, '2026-09-01', NULL),
			('grel_father_restricted', 'pup_active_1', 'per_father_1', 'FATHER', 1, 1, 'Prohibited Steps Order - Do Not Contact', 2, 0, 0, 0, 0, 0, 0, 0, 0, '2026-09-01', NULL),
			('grel_grandparent_nopr', 'pup_active_1', 'per_grandparent_1', 'GRANDPARENT', 0, 0, '', 3, 1, 1, 0, 0, 1, 1, 1, 0, '2026-09-01', NULL)`,
	}

	for _, q := range seedQueries {
		if _, err := coreDAL.WriteDB().Exec(q); err != nil {
			_ = coreDAL.Close()
			t.Fatalf("failed to seed Milestone 1 test fixture query (%s): %v", q, err)
		}
	}
	logTap.Logf("seeded Milestone 1 school fixtures for %s on %s", fx.AcademicYearID, fx.ReferenceDate)

	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("--- TAPPED CORE DAL SERVICE & DIAGNOSTIC LOGS ---\n%s", logTap.Dump())
		}
		_ = coreDAL.Close()
	})

	return coreDAL, fx, logTap
}

// ============================================================================
// 1. Pupils, SEN & Medical Domain (`ctx.dal.pupils`, `ctx.dal.sen`, `ctx.dal.medical`)
// ============================================================================

func TestGetPupilByIDScoped_CallerWithOwnGroupsAndPupilInAssignedGroup_ReturnsPupilSummary_test(t *testing.T) {
	ctx := context.Background()
	d, fx, logTap := setupIsolatedCoreDALWithFixtures(t)

	scope := dal.DALQueryScope{
		Mode:           dal.ScopeOwnGroups,
		StaffID:        fx.TeacherStaffID,
		AcademicYearID: fx.AcademicYearID,
		ReferenceDate:  fx.ReferenceDate,
	}

	pupil, err := d.GetPupilByIDScoped(ctx, scope, fx.ActivePupilID)
	if err != nil {
		t.Fatalf("expected teacher with :own_groups to read assigned pupil %s, got error: %v", fx.ActivePupilID, err)
	}
	logTap.Logf("fetched pupil %s (%s %s) under :own_groups", pupil.ID, pupil.FirstName, pupil.LastName)

	if pupil.ID != fx.ActivePupilID {
		t.Errorf("expected pupil ID %q, got %q", fx.ActivePupilID, pupil.ID)
	}
	if pupil.UPN != "H823400026001" {
		t.Errorf("expected UPN H823400026001, got %q", pupil.UPN)
	}
	if pupil.FirstName != "Ada" || pupil.LastName != "Lovelace" {
		t.Errorf("expected Ada Lovelace, got %s %s", pupil.FirstName, pupil.LastName)
	}
}

func TestGetPupilByIDScoped_CallerWithOwnGroupsAndPupilOutsideAssignedGroups_ReturnsAccessDeniedError_test(t *testing.T) {
	ctx := context.Background()
	d, fx, logTap := setupIsolatedCoreDALWithFixtures(t)

	scope := dal.DALQueryScope{
		Mode:           dal.ScopeOwnGroups,
		StaffID:        fx.TeacherStaffID,
		AcademicYearID: fx.AcademicYearID,
		ReferenceDate:  fx.ReferenceDate,
	}

	// 1. Pupil in another teacher's group (10Ma2) MUST be denied
	_, err := d.GetPupilByIDScoped(ctx, scope, fx.OtherGroupPupilID)
	if err == nil {
		t.Fatalf("expected access-denied error when teacher requests pupil %s outside :own_groups, got nil", fx.OtherGroupPupilID)
	}
	logTap.Logf("verified :own_groups denied access to other group pupil: %v", err)

	// 2. Pupil who transferred out of 10Ma1 on 2026-10-15 MUST be denied on ReferenceDate 2026-10-27
	_, err = d.GetPupilByIDScoped(ctx, scope, fx.TransferredPupilID)
	if err == nil {
		t.Fatalf("expected access-denied error on %s for pupil %s who transferred out on 2026-10-15, got nil", fx.ReferenceDate, fx.TransferredPupilID)
	}
	logTap.Logf("verified :own_groups denied access to transferred pupil on %s: %v", fx.ReferenceDate, err)
}

func TestListPupilsByGroupScoped_MidYearSetTransfer_FiltersByEffectiveFromAndEffectiveTo_test(t *testing.T) {
	ctx := context.Background()
	d, fx, logTap := setupIsolatedCoreDALWithFixtures(t)

	// 1. Query on 2026-10-27 (after Charles Babbage transferred from 10Ma1 to 10Ma2 on 2026-10-16): only Ada is active in 10Ma1
	currentScope := dal.DALQueryScope{
		Mode:           dal.ScopeOwnGroups,
		StaffID:        fx.TeacherStaffID,
		AcademicYearID: fx.AcademicYearID,
		ReferenceDate:  "2026-10-27",
	}
	currentRoster, err := d.ListPupilsByGroupScoped(ctx, currentScope, fx.OwnGroupID)
	if err != nil {
		t.Fatalf("unexpected error listing current group roster: %v", err)
	}
	logTap.Logf("roster count on 2026-10-27: %d", len(currentRoster))
	if len(currentRoster) != 1 || currentRoster[0].ID != fx.ActivePupilID {
		t.Fatalf("expected only [%s] in 10Ma1 on 2026-10-27, got %+v", fx.ActivePupilID, currentRoster)
	}

	// 2. Historical query on 2026-10-01 (before set transfer): both Ada and Charles were active in 10Ma1
	historicalScope := dal.DALQueryScope{
		Mode:           dal.ScopeOwnGroups,
		StaffID:        fx.TeacherStaffID,
		AcademicYearID: fx.AcademicYearID,
		ReferenceDate:  "2026-10-01",
	}
	historicalRoster, err := d.ListPupilsByGroupScoped(ctx, historicalScope, fx.OwnGroupID)
	if err != nil {
		t.Fatalf("unexpected error listing historical group roster: %v", err)
	}
	logTap.Logf("roster count on 2026-10-01: %d", len(historicalRoster))
	if len(historicalRoster) != 2 {
		t.Fatalf("expected 2 pupils in 10Ma1 on 2026-10-01, got %d (%+v)", len(historicalRoster), historicalRoster)
	}

	// 3. Teacher 1 requesting Teacher 2's group (10Ma2) under :own_groups MUST fail fast
	_, err = d.ListPupilsByGroupScoped(ctx, currentScope, fx.OtherGroupID)
	if err == nil {
		t.Fatalf("expected error when teacher with :own_groups requests unassigned group %s", fx.OtherGroupID)
	}
}

func TestListPupilSenAndMedicalBatch_BatchOfPupils_ReturnsActiveSenAndMedicalAlertsAndExcludesExpired_test(t *testing.T) {
	ctx := context.Background()
	d, fx, logTap := setupIsolatedCoreDALWithFixtures(t)

	scope := dal.DALQueryScope{
		Mode:           dal.ScopeOwnGroups,
		StaffID:        fx.TeacherStaffID,
		AcademicYearID: fx.AcademicYearID,
		ReferenceDate:  fx.ReferenceDate,
	}

	summaries, err := d.ListPupilSenAndMedicalBatch(ctx, scope, []string{fx.ActivePupilID})
	if err != nil {
		t.Fatalf("ListPupilSenAndMedicalBatch failed: %v", err)
	}

	summary, ok := summaries[fx.ActivePupilID]
	if !ok {
		t.Fatalf("expected SEN/Medical summary for pupil %s", fx.ActivePupilID)
	}
	logTap.Logf("SEN/Medical summary for %s: provision=%s need=%s med_count=%d",
		summary.PupilID, summary.SenProvisionCode, summary.PrimarySenNeedCode, len(summary.MedicalConditions))

	// On 2026-10-27, SEN provision is 'E' (upgraded on 2026-10-11; historical 'K' record expired 2026-10-10)
	if summary.SenProvisionCode != "E" {
		t.Errorf("expected active SEN provision code 'E' on %s, got %q", fx.ReferenceDate, summary.SenProvisionCode)
	}
	if summary.PrimarySenNeedCode != "ASD" {
		t.Errorf("expected primary SEN need code 'ASD', got %q", summary.PrimarySenNeedCode)
	}
	if len(summary.MedicalConditions) != 1 {
		t.Fatalf("expected exactly 1 active medical condition (excluding expired childhood rash), got %d", len(summary.MedicalConditions))
	}
	if summary.MedicalConditions[0].ConditionName != "Anaphylaxis (Peanut Allergy)" {
		t.Errorf("expected 'Anaphylaxis (Peanut Allergy)', got %q", summary.MedicalConditions[0].ConditionName)
	}
	if !summary.MedicalConditions[0].HasIHCP {
		t.Errorf("expected HasIHCP == true for active anaphylaxis condition")
	}
}

// ============================================================================
// 2. Groups & Timetable Spine (`ctx.dal.groups`)
// ============================================================================

func TestListGroupsScoped_TeacherWithOwnGroupsScope_ReturnsOnlyTemporallyAssignedGroups_test(t *testing.T) {
	ctx := context.Background()
	d, fx, logTap := setupIsolatedCoreDALWithFixtures(t)

	scope := dal.DALQueryScope{
		Mode:           dal.ScopeOwnGroups,
		StaffID:        fx.TeacherStaffID,
		AcademicYearID: fx.AcademicYearID,
		ReferenceDate:  fx.ReferenceDate,
	}

	groups, err := d.ListGroupsScoped(ctx, scope, "")
	if err != nil {
		t.Fatalf("ListGroupsScoped failed: %v", err)
	}
	logTap.Logf("teacher :own_groups count on %s: %d", fx.ReferenceDate, len(groups))

	// Must return grp_10ma1 only (excluding grp_10ma2 taught by stf_teacher_2 and grp_expired which ended 2026-10-01)
	if len(groups) != 1 {
		t.Fatalf("expected 1 active group for teacher 1 on %s, got %d (%+v)", fx.ReferenceDate, len(groups), groups)
	}
	if groups[0].ID != fx.OwnGroupID || groups[0].Code != "10Ma1" {
		t.Errorf("expected group %s (10Ma1), got %+v", fx.OwnGroupID, groups[0])
	}
}

func TestListGroupsScoped_CallerWithAllScope_ReturnsAllActiveGroupsForAcademicYear_test(t *testing.T) {
	ctx := context.Background()
	d, fx, logTap := setupIsolatedCoreDALWithFixtures(t)

	scope := dal.DALQueryScope{
		Mode:           dal.ScopeAll,
		StaffID:        fx.TeacherStaffID,
		AcademicYearID: fx.AcademicYearID,
		ReferenceDate:  fx.ReferenceDate,
	}

	groups, err := d.ListGroupsScoped(ctx, scope, "TEACHING_CLASS")
	if err != nil {
		t.Fatalf("ListGroupsScoped (:all) failed: %v", err)
	}
	logTap.Logf(":all scope returned %d active TEACHING_CLASS groups", len(groups))

	if len(groups) != 2 {
		t.Fatalf("expected 2 active TEACHING_CLASS groups (10Ma1 and 10Ma2), got %d (%+v)", len(groups), groups)
	}
}

func TestGetStaffTimetableSlotsScoped_TwoWeekRollingCycle_ReturnsSlotsWithMappedStatutorySession_test(t *testing.T) {
	ctx := context.Background()
	d, fx, logTap := setupIsolatedCoreDALWithFixtures(t)

	scope := dal.DALQueryScope{
		Mode:           dal.ScopeOwnGroups,
		StaffID:        fx.TeacherStaffID,
		AcademicYearID: fx.AcademicYearID,
		ReferenceDate:  fx.ReferenceDate,
	}

	wk1Slots, err := d.GetStaffTimetableSlotsScoped(ctx, scope, fx.TeacherStaffID, 1, 1)
	if err != nil {
		t.Fatalf("GetStaffTimetableSlotsScoped (Week 1) failed: %v", err)
	}
	if len(wk1Slots) != 1 {
		t.Fatalf("expected 1 slot on CycleWeek 1 Day 1, got %d", len(wk1Slots))
	}
	if wk1Slots[0].SlotID != fx.TimetableSlotAMID || wk1Slots[0].MapsToStatutorySession != "AM" {
		t.Errorf("expected slot %s mapped to statutory 'AM', got %+v", fx.TimetableSlotAMID, wk1Slots[0])
	}

	wk2Slots, err := d.GetStaffTimetableSlotsScoped(ctx, scope, fx.TeacherStaffID, 2, 1)
	if err != nil {
		t.Fatalf("GetStaffTimetableSlotsScoped (Week 2) failed: %v", err)
	}
	if len(wk2Slots) != 1 {
		t.Fatalf("expected 1 slot on CycleWeek 2 Day 1, got %d", len(wk2Slots))
	}
	if wk2Slots[0].SlotID != fx.TimetableSlotPMID || wk2Slots[0].MapsToStatutorySession != "PM" {
		t.Errorf("expected slot %s mapped to statutory 'PM', got %+v", fx.TimetableSlotPMID, wk2Slots[0])
	}
	logTap.Logf("verified 2-week rolling timetable statutory session mapping (Wk1=AM, Wk2=PM)")
}

// ============================================================================
// 3. Statutory Attendance (`ctx.dal.attendance`)
// ============================================================================

func TestGetGroupSessionAttendanceScoped_GroupInTeacherOwnGroups_ReturnsStatutoryAmPmMarks_test(t *testing.T) {
	ctx := context.Background()
	d, fx, logTap := setupIsolatedCoreDALWithFixtures(t)

	scope := dal.DALQueryScope{
		Mode:           dal.ScopeOwnGroups,
		StaffID:        fx.TeacherStaffID,
		AcademicYearID: fx.AcademicYearID,
		ReferenceDate:  fx.ReferenceDate,
	}

	_, err := d.UpsertSessionAttendanceBatchScoped(ctx, scope, dal.UpsertSessionAttendanceBatchInput{
		GroupID: fx.OwnGroupID,
		Date:    fx.ReferenceDate,
		Session: "AM",
		Marks: []dal.SessionAttendanceMarkInput{
			{
				PupilID:        fx.ActivePupilID,
				AttendanceCode: "/",
				MinutesLate:    0,
				Comment:        "On time",
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to seed AM attendance mark via UpsertSessionAttendanceBatchScoped: %v", err)
	}

	marks, err := d.GetGroupSessionAttendanceScoped(ctx, scope, fx.OwnGroupID, fx.ReferenceDate, "AM")
	if err != nil {
		t.Fatalf("GetGroupSessionAttendanceScoped failed: %v", err)
	}
	logTap.Logf("fetched %d AM attendance marks for %s", len(marks), fx.OwnGroupID)

	if len(marks) != 1 {
		t.Fatalf("expected 1 AM session mark, got %d", len(marks))
	}
	if marks[0].PupilID != fx.ActivePupilID || marks[0].AttendanceCode != "/" || marks[0].Session != "AM" {
		t.Errorf("unexpected AM session mark: %+v", marks[0])
	}
}

func TestUpsertSessionAttendanceBatchScoped_ValidAmAndPmMarks_PersistsMarksAndRegisterSubmissionAtomically_test(t *testing.T) {
	ctx := context.Background()
	d, fx, logTap := setupIsolatedCoreDALWithFixtures(t)

	scope := dal.DALQueryScope{
		Mode:           dal.ScopeOwnGroups,
		StaffID:        fx.TeacherStaffID,
		AcademicYearID: fx.AcademicYearID,
		ReferenceDate:  fx.ReferenceDate,
	}

	// 1. Insert initial AM mark ('/')
	res1, err := d.UpsertSessionAttendanceBatchScoped(ctx, scope, dal.UpsertSessionAttendanceBatchInput{
		GroupID: fx.OwnGroupID,
		Date:    fx.ReferenceDate,
		Session: "AM",
		Marks: []dal.SessionAttendanceMarkInput{
			{
				PupilID:        fx.ActivePupilID,
				AttendanceCode: "/",
				MinutesLate:    0,
			},
		},
	})
	if err != nil {
		t.Fatalf("initial AM batch upsert failed: %v", err)
	}
	if res1.UpsertedCount != 1 || res1.SubmissionID == "" {
		t.Fatalf("expected UpsertedCount=1 and non-empty SubmissionID, got %+v", res1)
	}

	// 2. Upsert updated AM mark ('L' with 12 minutes late) and verify idempotent update + audit submission row
	res2, err := d.UpsertSessionAttendanceBatchScoped(ctx, scope, dal.UpsertSessionAttendanceBatchInput{
		GroupID: fx.OwnGroupID,
		Date:    fx.ReferenceDate,
		Session: "AM",
		Marks: []dal.SessionAttendanceMarkInput{
			{
				PupilID:        fx.ActivePupilID,
				AttendanceCode: "L",
				MinutesLate:    12,
				Comment:        "Bus delayed",
			},
		},
	})
	if err != nil {
		t.Fatalf("second AM batch upsert failed: %v", err)
	}
	logTap.Logf("upserted AM register submission %s", res2.SubmissionID)

	marks, err := d.GetGroupSessionAttendanceScoped(ctx, scope, fx.OwnGroupID, fx.ReferenceDate, "AM")
	if err != nil {
		t.Fatalf("GetGroupSessionAttendanceScoped failed: %v", err)
	}
	if len(marks) != 1 || marks[0].AttendanceCode != "L" || marks[0].MinutesLate != 12 {
		t.Fatalf("expected updated mark code='L' minutes_late=12, got %+v", marks)
	}

	var submissionCount int
	err = d.ReadDB().QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM attendance_register_submissions WHERE group_id = ? AND date = ? AND session_or_slot_key = 'AM'`,
		fx.OwnGroupID, fx.ReferenceDate,
	).Scan(&submissionCount)
	if err != nil || submissionCount != 1 {
		t.Fatalf("expected 1 attendance_register_submissions row, got count=%d err=%v", submissionCount, err)
	}
}

func TestUpsertSessionAttendanceBatchScoped_InvalidNonAmPmSessionOrUnauthorizedGroup_FailsFastAndRollsBack_test(t *testing.T) {
	ctx := context.Background()
	d, fx, logTap := setupIsolatedCoreDALWithFixtures(t)

	scope := dal.DALQueryScope{
		Mode:           dal.ScopeOwnGroups,
		StaffID:        fx.TeacherStaffID,
		AcademicYearID: fx.AcademicYearID,
		ReferenceDate:  fx.ReferenceDate,
	}

	// 1. Non-AM/PM session ('LUNCH') MUST fail fast
	_, err := d.UpsertSessionAttendanceBatchScoped(ctx, scope, dal.UpsertSessionAttendanceBatchInput{
		GroupID: fx.OwnGroupID,
		Date:    fx.ReferenceDate,
		Session: "LUNCH",
		Marks: []dal.SessionAttendanceMarkInput{
			{PupilID: fx.ActivePupilID, AttendanceCode: "/"},
		},
	})
	if err == nil {
		t.Fatalf("expected fail-fast error for non-statutory session 'LUNCH', got nil")
	}

	// 2. Unauthorized group (grp_10ma2) under :own_groups MUST fail fast
	_, err = d.UpsertSessionAttendanceBatchScoped(ctx, scope, dal.UpsertSessionAttendanceBatchInput{
		GroupID: fx.OtherGroupID,
		Date:    fx.ReferenceDate,
		Session: "AM",
		Marks: []dal.SessionAttendanceMarkInput{
			{PupilID: fx.OtherGroupPupilID, AttendanceCode: "/"},
		},
	})
	if err == nil {
		t.Fatalf("expected fail-fast error when recording attendance for unassigned group %s", fx.OtherGroupID)
	}
	logTap.Logf("verified fail-fast rollback for invalid session and unauthorized group")
}

// ============================================================================
// 4. Guardians & Households (`ctx.dal.guardians` — Mandatory PR & Court-Order Invariant)
// ============================================================================

func TestListGuardiansForPupilScoped_GuardianWithCourtOrderRestricted_ExcludesRestrictedGuardianFromResults_test(t *testing.T) {
	ctx := context.Background()
	d, fx, logTap := setupIsolatedCoreDALWithFixtures(t)

	scope := dal.DALQueryScope{
		Mode:           dal.ScopeOwnGroups,
		StaffID:        fx.TeacherStaffID,
		AcademicYearID: fx.AcademicYearID,
		ReferenceDate:  fx.ReferenceDate,
	}

	guardians, err := d.ListGuardiansForPupilScoped(ctx, scope, fx.ActivePupilID, dal.GuardianFilterOptions{
		RequireParentalResponsibility: false,
	})
	if err != nil {
		t.Fatalf("ListGuardiansForPupilScoped failed: %v", err)
	}
	logTap.Logf("returned %d non-restricted guardians for %s", len(guardians), fx.ActivePupilID)

	for _, g := range guardians {
		if g.GuardianPersonID == fx.RestrictedFatherPersID || g.FirstName == "George" {
			t.Fatalf("SECURITY VIOLATION: court-order-restricted guardian (%s) was returned by ListGuardiansForPupilScoped: %+v", fx.RestrictedFatherPersID, g)
		}
	}
	if len(guardians) != 2 {
		t.Fatalf("expected 2 non-restricted guardians (Mother + Grandparent), got %d (%+v)", len(guardians), guardians)
	}
}

func TestListGuardiansForPupilScoped_GuardianWithoutParentalResponsibility_RespectsRequirePRFilter_test(t *testing.T) {
	ctx := context.Background()
	d, fx, logTap := setupIsolatedCoreDALWithFixtures(t)

	scope := dal.DALQueryScope{
		Mode:           dal.ScopeOwnGroups,
		StaffID:        fx.TeacherStaffID,
		AcademicYearID: fx.AcademicYearID,
		ReferenceDate:  fx.ReferenceDate,
	}

	prGuardians, err := d.ListGuardiansForPupilScoped(ctx, scope, fx.ActivePupilID, dal.GuardianFilterOptions{
		RequireParentalResponsibility: true,
	})
	if err != nil {
		t.Fatalf("ListGuardiansForPupilScoped (RequirePR=true) failed: %v", err)
	}
	logTap.Logf("returned %d PR guardians for %s", len(prGuardians), fx.ActivePupilID)

	if len(prGuardians) != 1 {
		t.Fatalf("expected exactly 1 unrestricted PR guardian (Anne Byron), got %d (%+v)", len(prGuardians), prGuardians)
	}
	if prGuardians[0].GuardianPersonID != fx.UnrestrictedMotherPersID || !prGuardians[0].HasParentalResponsibility {
		t.Errorf("expected Anne Byron with HasParentalResponsibility=true, got %+v", prGuardians[0])
	}
}

func TestListGuardiansForPupilScoped_HouseholdMarkedConfidentialAddress_RedactsConfidentialAddressFields_test(t *testing.T) {
	ctx := context.Background()
	d, fx, logTap := setupIsolatedCoreDALWithFixtures(t)

	scope := dal.DALQueryScope{
		Mode:           dal.ScopeOwnGroups,
		StaffID:        fx.TeacherStaffID,
		AcademicYearID: fx.AcademicYearID,
		ReferenceDate:  fx.ReferenceDate,
	}

	guardians, err := d.ListGuardiansForPupilScoped(ctx, scope, fx.ActivePupilID, dal.GuardianFilterOptions{
		RequireParentalResponsibility: false,
	})
	if err != nil {
		t.Fatalf("ListGuardiansForPupilScoped failed: %v", err)
	}

	var foundGrandparent bool
	for _, g := range guardians {
		if g.GuardianPersonID == fx.NonPRGrandparentPersID {
			foundGrandparent = true
			if g.AddressLine1 != "" || g.Postcode != "" {
				t.Fatalf("SECURITY VIOLATION: confidential household address leaked for guardian %s: address=%q postcode=%q",
					g.GuardianPersonID, g.AddressLine1, g.Postcode)
			}
			if !g.IsConfidentialAddressRedacted {
				t.Errorf("expected IsConfidentialAddressRedacted == true for guardian residing in confidential household")
			}
		}
		if g.GuardianPersonID == fx.UnrestrictedMotherPersID {
			if g.AddressLine1 != "42 High Street" || g.Postcode != "SW1A 2BB" {
				t.Errorf("expected non-confidential address '42 High Street, SW1A 2BB' for Mother, got %q %q", g.AddressLine1, g.Postcode)
			}
		}
	}
	if !foundGrandparent {
		t.Fatalf("expected non-restricted grandparent %s in guardian list", fx.NonPRGrandparentPersID)
	}
	logTap.Logf("verified confidential household address redaction")
}

// ============================================================================
// 5. Safeguarding Vault (`dal.safeguarding.create` Postbox vs `dal.safeguarding.read` DSL Vault)
// ============================================================================

func TestCreateSafeguardingPostboxNote_ValidTeacherCaller_EncryptsPayloadWithEnvelopeKeyAndWritesImmutableAuditLog_test(t *testing.T) {
	ctx := context.Background()
	d, fx, logTap := setupIsolatedCoreDALWithFixtures(t)

	plaintextVerbatim := "Pupil disclosed bruised wrist during morning registration."
	receipt, err := d.CreateSafeguardingPostboxNote(ctx, dal.CreateSafeguardingPostboxInput{
		PupilID:            fx.ActivePupilID,
		ReportedByStaffID:  fx.TeacherStaffID,
		AssignedDSLStaffID: fx.DSLStaffID,
		ConcernCategory:    "PHYSICAL",
		IncidentDate:       fx.ReferenceDate,
		Summary:            "Bruised wrist disclosure",
		BodyMap:            "LEFT_WRIST",
		VerbatimAccount:    plaintextVerbatim,
		ActionTaken:        "Referred immediately to DSL",
		RequestID:          "req_sfg_postbox_1",
		ClientIP:           "10.0.0.15",
		HostKMSKeyHandle:   fx.HostKMSKeyHandle,
	})
	if err != nil {
		t.Fatalf("CreateSafeguardingPostboxNote failed: %v", err)
	}
	logTap.Logf("created safeguarding postbox note %s", receipt.NoteID)

	if !strings.HasPrefix(receipt.NoteID, "sfg_") {
		t.Errorf("expected note ID prefixed with 'sfg_', got %q", receipt.NoteID)
	}

	// Verify raw SQLite storage is AES-256-GCM encrypted (ciphertext does NOT contain plaintext)
	var wrappedDEK, nonce, ciphertext []byte
	err = d.ReadDB().QueryRowContext(
		ctx,
		`SELECT wrapped_dek, nonce, ciphertext_payload FROM safeguarding_notes WHERE id = ?`,
		receipt.NoteID,
	).Scan(&wrappedDEK, &nonce, &ciphertext)
	if err != nil {
		t.Fatalf("failed to query raw safeguarding_notes row: %v", err)
	}
	if len(wrappedDEK) == 0 || len(nonce) != 12 || len(ciphertext) == 0 {
		t.Fatalf("expected non-empty wrapped_dek, 12-byte nonce, and non-empty ciphertext, got dek=%d nonce=%d cipher=%d",
			len(wrappedDEK), len(nonce), len(ciphertext))
	}
	if bytes.Contains(ciphertext, []byte(plaintextVerbatim)) || bytes.Contains(ciphertext, []byte("Bruised wrist")) {
		t.Fatalf("SECURITY VIOLATION: safeguarding ciphertext_payload contains unencrypted plaintext")
	}

	// Verify append-only CREATE_POSTBOX audit log entry was written
	var actionType string
	err = d.ReadDB().QueryRowContext(
		ctx,
		`SELECT action_type FROM safeguarding_audit_logs WHERE note_id = ? AND request_id = 'req_sfg_postbox_1'`,
		receipt.NoteID,
	).Scan(&actionType)
	if err != nil || actionType != "CREATE_POSTBOX" {
		t.Fatalf("expected CREATE_POSTBOX audit log entry for note %s, got action=%q err=%v", receipt.NoteID, actionType, err)
	}
}

func TestReadSafeguardingNoteDSL_AuthorizedDSLWithValidCloudKeyToken_DecryptsPayloadAndWritesDSLDecryptAuditLog_test(t *testing.T) {
	ctx := context.Background()
	d, fx, logTap := setupIsolatedCoreDALWithFixtures(t)

	receipt, err := d.CreateSafeguardingPostboxNote(ctx, dal.CreateSafeguardingPostboxInput{
		PupilID:            fx.ActivePupilID,
		ReportedByStaffID:  fx.TeacherStaffID,
		AssignedDSLStaffID: fx.DSLStaffID,
		ConcernCategory:    "EMOTIONAL",
		IncidentDate:       fx.ReferenceDate,
		Summary:            "Welfare check concern",
		BodyMap:            "NONE",
		VerbatimAccount:    "Pupil appeared distressed before Period 1.",
		ActionTaken:        "Escorted to pastoral office",
		RequestID:          "req_sfg_create_2",
		ClientIP:           "10.0.0.15",
		HostKMSKeyHandle:   fx.HostKMSKeyHandle,
	})
	if err != nil {
		t.Fatalf("CreateSafeguardingPostboxNote failed: %v", err)
	}

	cloudToken, err := d.IssueDSLCloudKeyToken(fx.DSLStaffID, fx.HostKMSKeyHandle, 24*time.Hour)
	if err != nil {
		t.Fatalf("IssueDSLCloudKeyToken failed: %v", err)
	}

	decrypted, err := d.ReadSafeguardingNoteDSL(ctx, dal.ReadSafeguardingNoteDSLInput{
		NoteID:           receipt.NoteID,
		DSLStaffID:       fx.DSLStaffID,
		CloudKeyToken:    cloudToken,
		HostKMSKeyHandle: fx.HostKMSKeyHandle,
		RequestID:        "req_sfg_read_dsl_2",
		ClientIP:         "10.0.0.20",
	})
	if err != nil {
		t.Fatalf("ReadSafeguardingNoteDSL failed for authorized DSL: %v", err)
	}
	logTap.Logf("DSL %s decrypted safeguarding note %s", fx.DSLStaffID, decrypted.NoteID)

	if decrypted.Summary != "Welfare check concern" || decrypted.VerbatimAccount != "Pupil appeared distressed before Period 1." {
		t.Errorf("decrypted payload mismatch: %+v", decrypted)
	}

	var auditCount int
	err = d.ReadDB().QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM safeguarding_audit_logs WHERE note_id = ? AND action_type = 'DSL_DECRYPT_VIEW' AND request_id = 'req_sfg_read_dsl_2'`,
		receipt.NoteID,
	).Scan(&auditCount)
	if err != nil || auditCount != 1 {
		t.Fatalf("expected DSL_DECRYPT_VIEW audit log entry, got count=%d err=%v", auditCount, err)
	}
}

func TestReadSafeguardingNoteDSL_NonDSLStaffOrExpiredCloudKeyToken_FailsFastWithUnauthorizedError_test(t *testing.T) {
	ctx := context.Background()
	d, fx, logTap := setupIsolatedCoreDALWithFixtures(t)

	receipt, err := d.CreateSafeguardingPostboxNote(ctx, dal.CreateSafeguardingPostboxInput{
		PupilID:            fx.ActivePupilID,
		ReportedByStaffID:  fx.TeacherStaffID,
		AssignedDSLStaffID: fx.DSLStaffID,
		ConcernCategory:    "NEGLECT",
		IncidentDate:       fx.ReferenceDate,
		Summary:            "Confidential concern",
		BodyMap:            "NONE",
		VerbatimAccount:    "Strictly confidential details.",
		ActionTaken:        "Logged in postbox",
		RequestID:          "req_sfg_create_3",
		ClientIP:           "10.0.0.15",
		HostKMSKeyHandle:   fx.HostKMSKeyHandle,
	})
	if err != nil {
		t.Fatalf("CreateSafeguardingPostboxNote failed: %v", err)
	}

	// 1. Non-DSL teacher attempting to read/decrypt MUST fail fast
	teacherToken, _ := d.IssueDSLCloudKeyToken(fx.TeacherStaffID, fx.HostKMSKeyHandle, 1*time.Hour)
	_, err = d.ReadSafeguardingNoteDSL(ctx, dal.ReadSafeguardingNoteDSLInput{
		NoteID:           receipt.NoteID,
		DSLStaffID:       fx.TeacherStaffID,
		CloudKeyToken:    teacherToken,
		HostKMSKeyHandle: fx.HostKMSKeyHandle,
		RequestID:        "req_sfg_unauth_teacher",
		ClientIP:         "10.0.0.15",
	})
	if err == nil {
		t.Fatalf("SECURITY VIOLATION: non-DSL teacher (%s) was allowed to decrypt safeguarding note %s", fx.TeacherStaffID, receipt.NoteID)
	}

	// 2. DSL with expired cloud key token (negative TTL) MUST fail fast
	expiredToken, err := d.IssueDSLCloudKeyToken(fx.DSLStaffID, fx.HostKMSKeyHandle, -5*time.Minute)
	if err != nil {
		t.Fatalf("failed to construct expired DSL token: %v", err)
	}
	_, err = d.ReadSafeguardingNoteDSL(ctx, dal.ReadSafeguardingNoteDSLInput{
		NoteID:           receipt.NoteID,
		DSLStaffID:       fx.DSLStaffID,
		CloudKeyToken:    expiredToken,
		HostKMSKeyHandle: fx.HostKMSKeyHandle,
		RequestID:        "req_sfg_expired_token",
		ClientIP:         "10.0.0.20",
	})
	if err == nil {
		t.Fatalf("SECURITY VIOLATION: expired DSL cloud key token was accepted for decrypting note %s", receipt.NoteID)
	}
	logTap.Logf("verified non-DSL staff and expired cloud key tokens are rejected")
}

func TestListSafeguardingNoteEnvelopesDSL_DSLAndNonDSL_ReturnsEncryptedEnvelopeMetadataOnlyToDSL_test(t *testing.T) {
	ctx := context.Background()
	d, fx, logTap := setupIsolatedCoreDALWithFixtures(t)

	receipt, err := d.CreateSafeguardingPostboxNote(ctx, dal.CreateSafeguardingPostboxInput{
		PupilID:            fx.ActivePupilID,
		ReportedByStaffID:  fx.TeacherStaffID,
		AssignedDSLStaffID: fx.DSLStaffID,
		ConcernCategory:    "ONLINE_SAFETY",
		IncidentDate:       fx.ReferenceDate,
		Summary:            "Sensitive online safety disclosure",
		BodyMap:            "NONE",
		VerbatimAccount:    "Strictly confidential verbatim text.",
		ActionTaken:        "Referred to DSL",
		RequestID:          "req_sfg_list_meta_1",
		ClientIP:           "10.0.0.15",
		HostKMSKeyHandle:   fx.HostKMSKeyHandle,
	})
	if err != nil {
		t.Fatalf("CreateSafeguardingPostboxNote failed: %v", err)
	}

	// 1. Non-DSL staff member MUST fail fast when listing safeguarding note envelopes
	if _, err := d.ListSafeguardingNoteEnvelopesDSL(ctx, fx.TeacherStaffID, fx.ActivePupilID); err == nil {
		t.Fatalf("SECURITY VIOLATION: non-DSL teacher (%s) was allowed to list safeguarding note envelopes", fx.TeacherStaffID)
	}

	// 2. Authorized DSL receives envelope metadata (NoteID, Category, Status, KMSKeyID) without plaintext
	envelopes, err := d.ListSafeguardingNoteEnvelopesDSL(ctx, fx.DSLStaffID, fx.ActivePupilID)
	if err != nil {
		t.Fatalf("ListSafeguardingNoteEnvelopesDSL failed for DSL: %v", err)
	}
	if len(envelopes) != 1 || envelopes[0].NoteID != receipt.NoteID {
		t.Fatalf("expected 1 envelope for note %s, got %+v", receipt.NoteID, envelopes)
	}
	if envelopes[0].ConcernCategory != "ONLINE_SAFETY" || envelopes[0].Status != "OPEN_TRIAGE" {
		t.Errorf("unexpected envelope metadata: %+v", envelopes[0])
	}
	logTap.Logf("verified DSL envelope metadata listing for note %s", receipt.NoteID)
}

func TestReadSafeguardingNoteDSL_ExpiredCloudKeyRenewed_RestoresAccessWithoutDataLoss_test(t *testing.T) {
	ctx := context.Background()
	d, fx, logTap := setupIsolatedCoreDALWithFixtures(t)

	receipt, err := d.CreateSafeguardingPostboxNote(ctx, dal.CreateSafeguardingPostboxInput{
		PupilID:            fx.ActivePupilID,
		ReportedByStaffID:  fx.TeacherStaffID,
		AssignedDSLStaffID: fx.DSLStaffID,
		ConcernCategory:    "EMOTIONAL",
		IncidentDate:       fx.ReferenceDate,
		Summary:            "Cloud key renewal test summary",
		BodyMap:            "NONE",
		VerbatimAccount:    "Cloud key renewal verbatim account.",
		ActionTaken:        "Logged in postbox",
		RequestID:          "req_sfg_renew_1",
		ClientIP:           "10.0.0.15",
		HostKMSKeyHandle:   fx.HostKMSKeyHandle,
	})
	if err != nil {
		t.Fatalf("CreateSafeguardingPostboxNote failed: %v", err)
	}

	// 1. Expired cloud key fails fast
	expiredKey, err := d.IssueDSLCloudKeyToken(fx.DSLStaffID, fx.HostKMSKeyHandle, -1*time.Hour)
	if err != nil {
		t.Fatalf("IssueDSLCloudKeyToken (expired) failed: %v", err)
	}
	if _, err := d.ReadSafeguardingNoteDSL(ctx, dal.ReadSafeguardingNoteDSLInput{
		NoteID:           receipt.NoteID,
		DSLStaffID:       fx.DSLStaffID,
		CloudKeyToken:    expiredKey,
		HostKMSKeyHandle: fx.HostKMSKeyHandle,
		RequestID:        "req_sfg_renew_attempt_1",
		ClientIP:         "10.0.0.20",
	}); err == nil {
		t.Fatalf("expected expired cloud key to fail fast")
	}

	// 2. Generating a new cloud key seamlessly restores access without data loss (GEMINI.md §5B)
	renewedKey, err := d.IssueDSLCloudKeyToken(fx.DSLStaffID, fx.HostKMSKeyHandle, 24*time.Hour)
	if err != nil {
		t.Fatalf("IssueDSLCloudKeyToken (renewed) failed: %v", err)
	}
	decrypted, err := d.ReadSafeguardingNoteDSL(ctx, dal.ReadSafeguardingNoteDSLInput{
		NoteID:           receipt.NoteID,
		DSLStaffID:       fx.DSLStaffID,
		CloudKeyToken:    renewedKey,
		HostKMSKeyHandle: fx.HostKMSKeyHandle,
		RequestID:        "req_sfg_renew_attempt_2",
		ClientIP:         "10.0.0.20",
	})
	if err != nil {
		t.Fatalf("ReadSafeguardingNoteDSL with renewed key failed: %v", err)
	}
	if decrypted.Summary != "Cloud key renewal test summary" || len(decrypted.AuditLogs) != 2 {
		t.Fatalf("unexpected decrypted note or audit log count after cloud key renewal: %+v", decrypted)
	}
	logTap.Logf("verified OAuth Cloud Key renewal restores DSL vault access with zero data loss")
}

