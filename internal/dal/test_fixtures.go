package dal

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type Milestone1SeedInput struct {
	AcademicYearID string `json:"academic_year_id"`
	ReferenceDate  string `json:"reference_date"`
}

type Milestone1SeedResult struct {
	FixtureID                  string `json:"fixture_id"`
	AcademicYearID             string `json:"academic_year_id"`
	ReferenceDate              string `json:"reference_date"`
	TeacherStaffID             string `json:"teacher_staff_id"`
	OtherTeacherStaffID        string `json:"other_teacher_staff_id"`
	DSLStaffID                 string `json:"dsl_staff_id"`
	OwnGroupID                 string `json:"own_group_id"`
	OtherGroupID               string `json:"other_group_id"`
	ActivePupilID              string `json:"active_pupil_id"`
	TransferredPupilID         string `json:"transferred_pupil_id"`
	OtherGroupPupilID          string `json:"other_group_pupil_id"`
	UnrestrictedMotherPersonID string `json:"unrestricted_mother_person_id"`
	RestrictedFatherPersonID   string `json:"restricted_father_person_id"`
	SafeHouseholdID            string `json:"safe_household_id"`
	ConfidentialHouseholdID    string `json:"confidential_household_id"`
}

func (d *CoreDAL) SeedMilestone1Fixtures(ctx context.Context, in Milestone1SeedInput) (*Milestone1SeedResult, error) {
	ayID := strings.TrimSpace(in.AcademicYearID)
	refDate := strings.TrimSpace(in.ReferenceDate)
	if ayID == "" {
		return nil, errors.New("missing mandatory field: academic_year_id")
	}
	if refDate == "" {
		return nil, errors.New("missing mandatory field: reference_date")
	}

	sfx := strings.ReplaceAll(uuid.NewString(), "-", "")[:8]
	res := &Milestone1SeedResult{
		FixtureID:                  sfx,
		AcademicYearID:             ayID,
		ReferenceDate:              refDate,
		TeacherStaffID:             "stf_t1_" + sfx,
		OtherTeacherStaffID:        "stf_t2_" + sfx,
		DSLStaffID:                 "stf_dsl_" + sfx,
		OwnGroupID:                 "grp_own_" + sfx,
		OtherGroupID:               "grp_oth_" + sfx,
		ActivePupilID:              "pup_act_" + sfx,
		TransferredPupilID:         "pup_trn_" + sfx,
		OtherGroupPupilID:          "pup_oth_" + sfx,
		UnrestrictedMotherPersonID: "per_mom_" + sfx,
		RestrictedFatherPersonID:   "per_dad_" + sfx,
		SafeHouseholdID:            "hh_safe_" + sfx,
		ConfidentialHouseholdID:    "hh_conf_" + sfx,
	}

	tx, err := d.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin SeedMilestone1Fixtures transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	stmts := []struct {
		sql  string
		args []any
	}{
		{
			sql: `INSERT INTO academic_years (id, name, status, sequence_number, first_day, last_day)
			      VALUES (?, '2026/2027', 'CURRENT', (SELECT COALESCE(MAX(sequence_number), 0) + 1 FROM academic_years), '2026-09-01', '2027-07-23')
			      ON CONFLICT(id) DO NOTHING`,
			args: []any{ayID},
		},
		{
			sql: `INSERT INTO alert_levels (id, name, name_short, color_hex, bg_color_hex, description, sequence_number)
			      VALUES ('HIGH', 'High Priority', 'HI', '#FFFFFF', '#D32F2F', 'Critical alert', 1)
			      ON CONFLICT(id) DO NOTHING`,
		},
		{
			sql: `INSERT INTO sen_need_types (code, name, broad_area, sequence_number)
			      VALUES ('ASD', 'Autistic Spectrum Disorder', 'COMMUNICATION_AND_INTERACTION', 1)
			      ON CONFLICT(code) DO NOTHING`,
		},
		{
			sql: `INSERT INTO attendance_codes (code, name, statistical_category, physical_status, sequence_number)
			      VALUES ('/', 'Present (AM)', 'PRESENT', 'ONSITE', 1),
			             ('\', 'Present (PM)', 'PRESENT', 'ONSITE', 2),
			             ('L', 'Late before register closed', 'PRESENT', 'ONSITE_LATE', 3)
			      ON CONFLICT(code) DO NOTHING`,
		},
		{
			sql: `INSERT INTO persons (id, legal_first_name, legal_last_name, sex, dob, phone_primary, status)
			      VALUES (?, 'Ada', 'Lovelace', 'F', '2012-12-10', '', 'ACTIVE'),
			             (?, 'Charles', 'Babbage', 'M', '2012-10-15', '', 'ACTIVE'),
			             (?, 'Alan', 'Turing', 'M', '2012-06-23', '', 'ACTIVE'),
			             (?, 'Grace', 'Hopper', 'F', '1985-05-01', '', 'ACTIVE'),
			             (?, 'Katherine', 'Johnson', 'F', '1984-08-26', '', 'ACTIVE'),
			             (?, 'Margaret', 'Hamilton', 'F', '1980-08-17', '', 'ACTIVE'),
			             (?, 'Anne', 'Byron', 'F', '1988-03-14', '07700900111', 'ACTIVE'),
			             (?, 'George', 'Byron', 'M', '1987-01-22', '07700900222', 'ACTIVE'),
			             (?, 'Judith', 'Milbanke', 'F', '1960-04-10', '07700900333', 'ACTIVE')`,
			args: []any{
				"per_p1_" + sfx,
				"per_p2_" + sfx,
				"per_p3_" + sfx,
				"per_t1_" + sfx,
				"per_t2_" + sfx,
				"per_dsl_" + sfx,
				res.UnrestrictedMotherPersonID,
				res.RestrictedFatherPersonID,
				"per_gp_" + sfx,
			},
		},
		{
			sql: `INSERT INTO pupils (id, person_id, upn, ethnicity_code, first_language_code)
			      VALUES (?, ?, ?, 'WBRI', 'ENG'),
			             (?, ?, ?, 'WBRI', 'ENG'),
			             (?, ?, ?, 'WBRI', 'ENG')`,
			args: []any{
				res.ActivePupilID, "per_p1_" + sfx, "U1" + strings.ToUpper(sfx) + "001",
				res.TransferredPupilID, "per_p2_" + sfx, "U2" + strings.ToUpper(sfx) + "002",
				res.OtherGroupPupilID, "per_p3_" + sfx, "U3" + strings.ToUpper(sfx) + "003",
			},
		},
		{
			sql: `INSERT INTO pupil_year_enrolments (id, pupil_id, academic_year_id, nc_year_group, enrolment_status, effective_from, effective_to)
			      VALUES (?, ?, ?, '10', 'C', '2026-09-01', NULL),
			             (?, ?, ?, '10', 'C', '2026-09-01', NULL),
			             (?, ?, ?, '10', 'C', '2026-09-01', NULL)`,
			args: []any{
				"enr_1_" + sfx, res.ActivePupilID, ayID,
				"enr_2_" + sfx, res.TransferredPupilID, ayID,
				"enr_3_" + sfx, res.OtherGroupPupilID, ayID,
			},
		},
		{
			sql: `INSERT INTO staff (id, person_id, staff_code, trn, staff_type, job_title, qts_flag, is_dsl, hire_date)
			      VALUES (?, ?, ?, ?, 'TEACHING', 'Teacher of Maths', 1, 0, '2020-09-01'),
			             (?, ?, ?, ?, 'TEACHING', 'Teacher of Maths', 1, 0, '2021-09-01'),
			             (?, ?, ?, ?, 'LEADERSHIP', 'DSL', 1, 1, '2018-09-01')`,
			args: []any{
				res.TeacherStaffID, "per_t1_" + sfx, "T1_" + sfx[:4], "71" + sfx[:5],
				res.OtherTeacherStaffID, "per_t2_" + sfx, "T2_" + sfx[:4], "72" + sfx[:5],
				res.DSLStaffID, "per_dsl_" + sfx, "DS_" + sfx[:4], "73" + sfx[:5],
			},
		},
		{
			sql: `INSERT INTO groups (id, academic_year_id, group_type, code, name, takes_attendance, effective_from, effective_to)
			      VALUES (?, ?, 'TEACHING_CLASS', ?, 'Year 10 Maths Set 1', 1, '2026-09-01', NULL),
			             (?, ?, 'TEACHING_CLASS', ?, 'Year 10 Maths Set 2', 1, '2026-09-01', NULL)`,
			args: []any{
				res.OwnGroupID, ayID, "10Ma1_" + sfx[:4],
				res.OtherGroupID, ayID, "10Ma2_" + sfx[:4],
			},
		},
		{
			sql: `INSERT INTO group_staff_assignments (id, group_id, staff_id, role, academic_year_id, effective_from, effective_to)
			      VALUES (?, ?, ?, 'LEAD_TEACHER', ?, '2026-09-01', NULL),
			             (?, ?, ?, 'LEAD_TEACHER', ?, '2026-09-01', NULL)`,
			args: []any{
				"gsa_1_" + sfx, res.OwnGroupID, res.TeacherStaffID, ayID,
				"gsa_2_" + sfx, res.OtherGroupID, res.OtherTeacherStaffID, ayID,
			},
		},
		{
			sql: `INSERT INTO group_pupil_memberships (id, group_id, pupil_id, academic_year_id, effective_from, effective_to)
			      VALUES (?, ?, ?, ?, '2026-09-01', NULL),
			             (?, ?, ?, ?, '2026-09-01', '2026-10-15'),
			             (?, ?, ?, ?, '2026-10-16', NULL),
			             (?, ?, ?, ?, '2026-09-01', NULL)`,
			args: []any{
				"gpm_act_" + sfx, res.OwnGroupID, res.ActivePupilID, ayID,
				"gpm_tro_" + sfx, res.OwnGroupID, res.TransferredPupilID, ayID,
				"gpm_trn_" + sfx, res.OtherGroupID, res.TransferredPupilID, ayID,
				"gpm_oth_" + sfx, res.OtherGroupID, res.OtherGroupPupilID, ayID,
			},
		},
		{
			sql: `INSERT INTO pupil_sen_records (id, pupil_id, provision_code, ehcp_start_date, teaching_strategies, effective_from, effective_to)
			      VALUES (?, ?, 'E', '2026-10-11', 'Chunk instructions and provide visual timetable', '2026-10-11', NULL)`,
			args: []any{"sen_" + sfx, res.ActivePupilID},
		},
		{
			sql: `INSERT INTO pupil_sen_needs (id, pupil_id, sen_need_code, ranking, alert_level_id, effective_from, effective_to)
			      VALUES (?, ?, 'ASD', 1, 'HIGH', '2026-09-01', NULL)`,
			args: []any{"sneed_" + sfx, res.ActivePupilID},
		},
		{
			sql: `INSERT INTO pupil_medical_conditions (
				id, pupil_id, condition_name, condition_category, alert_level_id,
				has_ihcp, emergency_response_plan, parental_consent_given, effective_from, effective_to
			) VALUES (?, ?, 'Anaphylaxis (Peanut Allergy)', 'ALLERGY', 'HIGH', 1, 'Administer EpiPen immediately and call 999', 1, '2026-09-01', NULL)`,
			args: []any{"med_" + sfx, res.ActivePupilID},
		},
		{
			sql: `INSERT INTO households (id, name, address_line_1, town_city, postcode, is_confidential_address)
			      VALUES (?, 'Byron Household', '42 High Street', 'London', 'SW1A 2BB', 0),
			             (?, 'Confidential Refuge', '10 Safe Haven Way', 'London', 'SW1A 1AA', 1)`,
			args: []any{res.SafeHouseholdID, res.ConfidentialHouseholdID},
		},
		{
			sql: `INSERT INTO household_memberships (id, household_id, person_id, member_role, residency_type, effective_from, effective_to)
			      VALUES (?, ?, ?, 'ADULT', 'PRIMARY', '2026-09-01', NULL),
			             (?, ?, ?, 'CHILD', 'PRIMARY', '2026-09-01', NULL),
			             (?, ?, ?, 'ADULT', 'SECONDARY_SHARED', '2026-09-01', NULL)`,
			args: []any{
				"hm_mom_" + sfx, res.SafeHouseholdID, res.UnrestrictedMotherPersonID,
				"hm_pup_" + sfx, res.SafeHouseholdID, "per_p1_" + sfx,
				"hm_gp_" + sfx, res.ConfidentialHouseholdID, "per_gp_" + sfx,
			},
		},
		{
			sql: `INSERT INTO guardian_relationships (
				id, pupil_id, guardian_person_id, relationship_code,
				has_parental_responsibility, court_order_restricted, court_order_notes,
				contact_priority, is_emergency_contact, authorised_to_collect,
				portal_data_access, receive_reports, contact_by_phone, contact_by_sms, contact_by_email, contact_by_post,
				effective_from, effective_to
			) VALUES
				(?, ?, ?, 'MOTHER', 1, 0, '', 1, 1, 1, 1, 1, 1, 1, 1, 1, '2026-09-01', NULL),
				(?, ?, ?, 'FATHER', 1, 1, 'Prohibited Steps Order - Do Not Contact', 2, 0, 0, 0, 0, 0, 0, 0, 0, '2026-09-01', NULL),
				(?, ?, ?, 'GRANDPARENT', 0, 0, '', 3, 1, 1, 0, 0, 1, 1, 1, 0, '2026-09-01', NULL)`,
			args: []any{
				"grel_mom_" + sfx, res.ActivePupilID, res.UnrestrictedMotherPersonID,
				"grel_dad_" + sfx, res.ActivePupilID, res.RestrictedFatherPersonID,
				"grel_gp_" + sfx, res.ActivePupilID, "per_gp_" + sfx,
			},
		},
	}

	for _, stmt := range stmts {
		if _, err := tx.ExecContext(ctx, stmt.sql, stmt.args...); err != nil {
			return nil, fmt.Errorf("failed to execute Milestone 1 seed statement: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit SeedMilestone1Fixtures transaction: %w", err)
	}
	return res, nil
}

func (d *CoreDAL) DeleteMilestone1Fixtures(ctx context.Context, fixtureID string) error {
	sfx := strings.TrimSpace(fixtureID)
	if sfx == "" {
		return errors.New("missing mandatory parameter: fixture_id")
	}

	activePupilID := "pup_act_" + sfx
	transPupilID := "pup_trn_" + sfx
	otherPupilID := "pup_oth_" + sfx
	ownGroupID := "grp_own_" + sfx
	otherGroupID := "grp_oth_" + sfx

	tx, err := d.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin DeleteMilestone1Fixtures transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.ExecContext(
		ctx,
		`UPDATE safeguarding_notes SET retention_purge_eligible = 1 WHERE pupil_id IN (?, ?, ?)`,
		activePupilID, transPupilID, otherPupilID,
	); err != nil {
		return fmt.Errorf("failed to mark fixture safeguarding_notes purge eligible: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DROP TRIGGER IF EXISTS trg_safeguarding_audit_no_delete;`); err != nil {
		return fmt.Errorf("failed to temporarily drop audit delete trigger for fixture cleanup: %w", err)
	}
	if _, err := tx.ExecContext(
		ctx,
		`DELETE FROM safeguarding_audit_logs WHERE pupil_id IN (?, ?, ?)`,
		activePupilID, transPupilID, otherPupilID,
	); err != nil {
		return fmt.Errorf("failed to delete fixture safeguarding_audit_logs: %w", err)
	}
	if _, err := tx.ExecContext(
		ctx,
		`CREATE TRIGGER IF NOT EXISTS trg_safeguarding_audit_no_delete
		 BEFORE DELETE ON safeguarding_audit_logs
		 BEGIN
		     SELECT RAISE(ABORT, 'Security Violation: Statutory safeguarding audit logs are immutable and cannot be deleted.');
		 END;`,
	); err != nil {
		return fmt.Errorf("failed to restore trg_safeguarding_audit_no_delete: %w", err)
	}
	if _, err := tx.ExecContext(
		ctx,
		`DELETE FROM safeguarding_notes WHERE pupil_id IN (?, ?, ?)`,
		activePupilID, transPupilID, otherPupilID,
	); err != nil {
		return fmt.Errorf("failed to delete fixture safeguarding_notes: %w", err)
	}

	cleanupQueries := []struct {
		sql  string
		args []any
	}{
		{
			sql:  `DELETE FROM attendance_session_marks WHERE pupil_id IN (?, ?, ?)`,
			args: []any{activePupilID, transPupilID, otherPupilID},
		},
		{
			sql:  `DELETE FROM attendance_register_submissions WHERE group_id IN (?, ?)`,
			args: []any{ownGroupID, otherGroupID},
		},
		{
			sql:  `DELETE FROM guardian_relationships WHERE pupil_id IN (?, ?, ?)`,
			args: []any{activePupilID, transPupilID, otherPupilID},
		},
		{
			sql:  `DELETE FROM household_memberships WHERE household_id IN (?, ?)`,
			args: []any{"hh_safe_" + sfx, "hh_conf_" + sfx},
		},
		{
			sql:  `DELETE FROM households WHERE id IN (?, ?)`,
			args: []any{"hh_safe_" + sfx, "hh_conf_" + sfx},
		},
		{
			sql:  `DELETE FROM pupil_medical_conditions WHERE pupil_id IN (?, ?, ?)`,
			args: []any{activePupilID, transPupilID, otherPupilID},
		},
		{
			sql:  `DELETE FROM pupil_sen_needs WHERE pupil_id IN (?, ?, ?)`,
			args: []any{activePupilID, transPupilID, otherPupilID},
		},
		{
			sql:  `DELETE FROM pupil_sen_records WHERE pupil_id IN (?, ?, ?)`,
			args: []any{activePupilID, transPupilID, otherPupilID},
		},
		{
			sql:  `DELETE FROM group_pupil_memberships WHERE group_id IN (?, ?)`,
			args: []any{ownGroupID, otherGroupID},
		},
		{
			sql:  `DELETE FROM group_staff_assignments WHERE group_id IN (?, ?)`,
			args: []any{ownGroupID, otherGroupID},
		},
		{
			sql:  `DELETE FROM groups WHERE id IN (?, ?)`,
			args: []any{ownGroupID, otherGroupID},
		},
		{
			sql:  `DELETE FROM staff WHERE id IN (?, ?, ?)`,
			args: []any{"stf_t1_" + sfx, "stf_t2_" + sfx, "stf_dsl_" + sfx},
		},
		{
			sql:  `DELETE FROM pupil_year_enrolments WHERE pupil_id IN (?, ?, ?)`,
			args: []any{activePupilID, transPupilID, otherPupilID},
		},
		{
			sql:  `DELETE FROM pupils WHERE id IN (?, ?, ?)`,
			args: []any{activePupilID, transPupilID, otherPupilID},
		},
		{
			sql: `DELETE FROM persons WHERE id IN (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			args: []any{
				"per_p1_" + sfx,
				"per_p2_" + sfx,
				"per_p3_" + sfx,
				"per_t1_" + sfx,
				"per_t2_" + sfx,
				"per_dsl_" + sfx,
				"per_mom_" + sfx,
				"per_dad_" + sfx,
				"per_gp_" + sfx,
			},
		},
	}

	for _, q := range cleanupQueries {
		if _, err := tx.ExecContext(ctx, q.sql, q.args...); err != nil {
			return fmt.Errorf("failed to execute Milestone 1 cleanup query (%s): %w", q.sql, err)
		}
	}

	return tx.Commit()
}
