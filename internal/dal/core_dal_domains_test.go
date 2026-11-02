package dal_test

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

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

func setupIsolatedCoreDALWithFixtures(t *testing.T) (*dal.CoreDAL, *milestone1Fixtures, *dalTestLogTap) {
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
