package dal

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type CensusTestFixtureInput struct {
	AcademicYearID      string `json:"academic_year_id"`
	CensusDate          string `json:"census_date"`
	TermCode            string `json:"term_code"`
	AttendanceStartDate string `json:"attendance_start_date"`
	AttendanceEndDate   string `json:"attendance_end_date"`
}

type CensusTestFixtureResult struct {
	FixtureID           string `json:"fixture_id"`
	AcademicYearID      string `json:"academic_year_id"`
	CensusDate          string `json:"census_date"`
	TermCode            string `json:"term_code"`
	AttendanceStartDate string `json:"attendance_start_date"`
	AttendanceEndDate   string `json:"attendance_end_date"`
	AdminStaffID        string `json:"admin_staff_id"`
	TeacherStaffID      string `json:"teacher_staff_id"`
	PupilSenKFsmID      string `json:"pupil_sen_k_fsm_id"`
	PupilEhcpLacID      string `json:"pupil_ehcp_lac_id"`
	PupilNoSenID        string `json:"pupil_no_sen_id"`
	OffRollPupilID      string `json:"off_roll_pupil_id"`
}

func (d *CoreDAL) SeedCensusTestFixtures(ctx context.Context, in CensusTestFixtureInput) (*CensusTestFixtureResult, error) {
	ayID := strings.TrimSpace(in.AcademicYearID)
	if ayID == "" {
		return nil, errors.New("missing mandatory field: academic_year_id")
	}
	censusDate := strings.TrimSpace(in.CensusDate)
	if censusDate == "" {
		return nil, errors.New("missing mandatory field: census_date")
	}
	termCode := strings.ToUpper(strings.TrimSpace(in.TermCode))
	if termCode == "" {
		termCode = "AUTUMN"
	}
	attStart := strings.TrimSpace(in.AttendanceStartDate)
	if attStart == "" {
		attStart = "2026-09-01"
	}
	attEnd := strings.TrimSpace(in.AttendanceEndDate)
	if attEnd == "" {
		attEnd = censusDate
	}

	u := uuid.New()
	sfx := strings.ReplaceAll(u.String(), "-", "")[:8]
	numSeed := int(u.ID()%800000) + 100000

	res := &CensusTestFixtureResult{
		FixtureID:           sfx,
		AcademicYearID:      ayID,
		CensusDate:          censusDate,
		TermCode:            termCode,
		AttendanceStartDate: attStart,
		AttendanceEndDate:   attEnd,
		AdminStaffID:        "stf_cadm_" + sfx,
		TeacherStaffID:      "stf_ctch_" + sfx,
		PupilSenKFsmID:      "pup_csenk_" + sfx,
		PupilEhcpLacID:      "pup_cehcp_" + sfx,
		PupilNoSenID:        "pup_cnosen_" + sfx,
		OffRollPupilID:      "pup_coff_" + sfx,
	}

	upn1 := fmt.Sprintf("H8234%06d01", numSeed)
	upn2 := fmt.Sprintf("H8234%06d02", numSeed)
	upn3 := fmt.Sprintf("H8234%06d03", numSeed)
	upn4 := fmt.Sprintf("H8234%06d04", numSeed)

	uln1 := fmt.Sprintf("10%06d01", numSeed)
	uln2 := fmt.Sprintf("10%06d02", numSeed)
	uln3 := fmt.Sprintf("10%06d03", numSeed)
	uln4 := fmt.Sprintf("10%06d04", numSeed)

	tx, err := d.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin SeedCensusTestFixtures transaction: %w", err)
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
			      VALUES ('HIGH', 'High Priority', 'HI', '#FFFFFF', '#D32F2F', 'Critical alert', 1),
			             ('MEDIUM', 'Medium Priority', 'MD', '#FFFFFF', '#F57C00', 'Medium alert', 2)
			      ON CONFLICT(id) DO NOTHING`,
		},
		{
			sql: `INSERT INTO sen_need_types (code, name, broad_area, sequence_number)
			      VALUES ('ASD', 'Autistic Spectrum Disorder', 'COMMUNICATION_AND_INTERACTION', 1),
			             ('SPLD', 'Specific Learning Difficulty', 'COGNITION_AND_LEARNING', 2)
			      ON CONFLICT(code) DO NOTHING`,
		},
		{
			sql: `INSERT INTO attendance_codes (code, name, statistical_category, physical_status, sequence_number)
			      VALUES ('/', 'Present (AM)', 'PRESENT', 'ONSITE', 1),
			             ('\', 'Present (PM)', 'PRESENT', 'ONSITE', 2),
			             ('I', 'Illness', 'AUTHORISED_ABSENCE', 'ABSENT', 3),
			             ('O', 'Unauthorised Absence', 'UNAUTHORISED_ABSENCE', 'ABSENT', 4)
			      ON CONFLICT(code) DO NOTHING`,
		},
		{
			sql: `INSERT INTO roles (id, category, name, name_short, description, default_scopes_json)
			      VALUES
			        ('ADMIN', 'STAFF', 'System Administrator / Headteacher', 'ADM', 'Full administrative & statutory census authority', '["dal.census.read:all","dal.pupils.read:all","dal.pupils.write:all","dal.staff.read:admin","dal.staff.write:admin","dal.groups.read:all","dal.groups.write:all","dal.attendance.read:all","dal.attendance.write:all","dal.guardians.read:all","dal.guardians.write:all","dal.sen.read:all","dal.medical.read:all"]'),
			        ('TEACHER', 'STAFF', 'Classroom Teacher', 'TCH', 'Scoped classroom teacher access', '["dal.pupils.read:own_groups","dal.groups.read:own_groups","dal.attendance.read:own_groups","dal.attendance.write:own_groups","dal.sen.read:own_groups","dal.medical.read:own_groups","dal.safeguarding.create"]')
			      ON CONFLICT(id) DO UPDATE SET default_scopes_json = excluded.default_scopes_json`,
		},
		{
			sql: `INSERT INTO persons (id, external_sso_sub, title, legal_first_name, legal_last_name, preferred_first_name, preferred_last_name, dob, sex, status)
			      VALUES
			        (?, ?, 'Dame', 'Eleanor', 'Vance', 'Eleanor', 'Vance', '1975-04-12', 'F', 'ACTIVE'),
			        (?, ?, 'Ms', 'Hannah', 'Abbott', 'Hannah', 'Abbott', '1990-08-19', 'F', 'ACTIVE'),
			        (?, NULL, '', 'Maya', 'Patel', 'Maya', 'Patel', '2011-05-19', 'F', 'ACTIVE'),
			        (?, NULL, '', 'Noah', 'Williams', 'Noah', 'Williams', '2011-11-23', 'M', 'ACTIVE'),
			        (?, NULL, '', 'Chloe', 'Sterling', 'Chloe', 'Sterling', '2011-02-04', 'F', 'ACTIVE'),
			        (?, NULL, '', 'Oliver', 'Grant', 'Oliver', 'Grant', '2011-03-29', 'M', 'LEFT')`,
			args: []any{
				"per_cadm_" + sfx, "sso_cadm_" + sfx,
				"per_ctch_" + sfx, "sso_ctch_" + sfx,
				"per_cp1_" + sfx,
				"per_cp2_" + sfx,
				"per_cp3_" + sfx,
				"per_cp4_" + sfx,
			},
		},
		{
			sql: `INSERT INTO person_roles (person_id, role_id, is_primary, effective_from)
			      VALUES
			        (?, 'ADMIN', 1, '2026-09-01'),
			        (?, 'TEACHER', 1, '2026-09-01')`,
			args: []any{
				"per_cadm_" + sfx,
				"per_ctch_" + sfx,
			},
		},
		{
			sql: `INSERT INTO staff (id, person_id, staff_code, staff_type, job_title, qts_flag, is_dsl, hire_date)
			      VALUES
			        (?, ?, ?, 'LEADERSHIP', 'Executive Headteacher', 1, 1, '2018-09-01'),
			        (?, ?, ?, 'TEACHING', 'Teacher of Mathematics', 1, 0, '2021-09-01')`,
			args: []any{
				res.AdminStaffID, "per_cadm_" + sfx, "CA_" + sfx[:4],
				res.TeacherStaffID, "per_ctch_" + sfx, "CT_" + sfx[:4],
			},
		},
		{
			sql: `INSERT INTO pupils (id, person_id, upn, uln, ethnicity_code, first_language_code, pupil_premium_flag, looked_after_flag, post_looked_after_code)
			      VALUES
			        (?, ?, ?, ?, 'INDN', 'GUJ', 1, 0, 'N'),
			        (?, ?, ?, ?, 'WBRI', 'ENG', 1, 1, 'A'),
			        (?, ?, ?, ?, 'WBRI', 'ENG', 0, 0, 'N'),
			        (?, ?, ?, ?, 'WBRI', 'ENG', 0, 0, 'N')`,
			args: []any{
				res.PupilSenKFsmID, "per_cp1_" + sfx, upn1, uln1,
				res.PupilEhcpLacID, "per_cp2_" + sfx, upn2, uln2,
				res.PupilNoSenID, "per_cp3_" + sfx, upn3, uln3,
				res.OffRollPupilID, "per_cp4_" + sfx, upn4, uln4,
			},
		},
		{
			sql: `INSERT INTO pupil_year_enrolments (id, pupil_id, academic_year_id, nc_year_group, enrolment_status, effective_from, effective_to)
			      VALUES
			        (?, ?, ?, '10', 'C', '2026-09-01', NULL),
			        (?, ?, ?, '10', 'C', '2026-09-01', NULL),
			        (?, ?, ?, '10', 'C', '2026-09-01', NULL),
			        (?, ?, ?, '10', 'C', '2026-09-01', '2026-09-15')`,
			args: []any{
				"cenr_1_" + sfx, res.PupilSenKFsmID, ayID,
				"cenr_2_" + sfx, res.PupilEhcpLacID, ayID,
				"cenr_3_" + sfx, res.PupilNoSenID, ayID,
				"cenr_4_" + sfx, res.OffRollPupilID, ayID,
			},
		},
		{
			sql: `INSERT INTO pupil_sen_records (id, pupil_id, provision_code, ehcp_start_date, teaching_strategies, effective_from, effective_to)
			      VALUES
			        (?, ?, 'K', NULL, 'Extra time and dyslexia overlays', '2026-09-01', NULL),
			        (?, ?, 'E', '2025-05-10', '1:1 TA support and structured visual schedule', '2026-09-01', NULL),
			        (?, ?, 'N', NULL, '', '2026-09-01', NULL)`,
			args: []any{
				"csen_1_" + sfx, res.PupilSenKFsmID,
				"csen_2_" + sfx, res.PupilEhcpLacID,
				"csen_3_" + sfx, res.PupilNoSenID,
			},
		},
		{
			sql: `INSERT INTO pupil_sen_needs (id, pupil_id, sen_need_code, ranking, alert_level_id, effective_from, effective_to)
			      VALUES
			        (?, ?, 'SPLD', 1, 'MEDIUM', '2026-09-01', NULL),
			        (?, ?, 'ASD', 1, 'HIGH', '2026-09-01', NULL)`,
			args: []any{
				"csneed_1_" + sfx, res.PupilSenKFsmID,
				"csneed_2_" + sfx, res.PupilEhcpLacID,
			},
		},
		{
			sql: `INSERT INTO pupil_fsm_periods (id, pupil_id, effective_from, effective_to, country_code, ecs_reference)
			      VALUES
			        (?, ?, '2025-09-01', NULL, 'ENG', 'ECS-001'),
			        (?, ?, '2025-09-01', NULL, 'ENG', 'ECS-002')`,
			args: []any{
				"cfsm_1_" + sfx, res.PupilSenKFsmID,
				"cfsm_2_" + sfx, res.PupilEhcpLacID,
			},
		},
		{
			sql: `INSERT INTO attendance_session_marks (id, pupil_id, academic_year_id, date, session, attendance_code, recorded_by_staff_id)
			      VALUES
			        (?, ?, ?, '2026-09-10', 'AM', '/', ?),
			        (?, ?, ?, '2026-09-10', 'PM', '\', ?),
			        (?, ?, ?, '2026-09-11', 'AM', 'I', ?),
			        (?, ?, ?, '2026-09-11', 'PM', 'O', ?),
			        (?, ?, ?, '2026-09-10', 'AM', '/', ?),
			        (?, ?, ?, '2026-09-10', 'PM', '\', ?),
			        (?, ?, ?, '2026-09-10', 'AM', '/', ?),
			        (?, ?, ?, '2026-09-10', 'PM', '\', ?)`,
			args: []any{
				"casm_1_" + sfx, res.PupilSenKFsmID, ayID, res.TeacherStaffID,
				"casm_2_" + sfx, res.PupilSenKFsmID, ayID, res.TeacherStaffID,
				"casm_3_" + sfx, res.PupilSenKFsmID, ayID, res.TeacherStaffID,
				"casm_4_" + sfx, res.PupilSenKFsmID, ayID, res.TeacherStaffID,
				"casm_5_" + sfx, res.PupilEhcpLacID, ayID, res.TeacherStaffID,
				"casm_6_" + sfx, res.PupilEhcpLacID, ayID, res.TeacherStaffID,
				"casm_7_" + sfx, res.PupilNoSenID, ayID, res.TeacherStaffID,
				"casm_8_" + sfx, res.PupilNoSenID, ayID, res.TeacherStaffID,
			},
		},
	}

	for _, stmt := range stmts {
		if _, err := tx.ExecContext(ctx, stmt.sql, stmt.args...); err != nil {
			return nil, fmt.Errorf("failed to execute census fixture seed statement: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit SeedCensusTestFixtures transaction: %w", err)
	}
	return res, nil
}

func (d *CoreDAL) DeleteCensusTestFixtures(ctx context.Context, fixtureID string) error {
	sfx := strings.TrimSpace(fixtureID)
	if sfx == "" {
		return errors.New("missing mandatory parameter: fixture_id")
	}

	p1 := "pup_csenk_" + sfx
	p2 := "pup_cehcp_" + sfx
	p3 := "pup_cnosen_" + sfx
	p4 := "pup_coff_" + sfx

	tx, err := d.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin DeleteCensusTestFixtures transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	cleanupStmts := []struct {
		sql  string
		args []any
	}{
		{
			sql:  `DELETE FROM attendance_session_marks WHERE pupil_id IN (?, ?, ?, ?)`,
			args: []any{p1, p2, p3, p4},
		},
		{
			sql:  `DELETE FROM pupil_fsm_periods WHERE pupil_id IN (?, ?, ?, ?)`,
			args: []any{p1, p2, p3, p4},
		},
		{
			sql:  `DELETE FROM pupil_sen_needs WHERE pupil_id IN (?, ?, ?, ?)`,
			args: []any{p1, p2, p3, p4},
		},
		{
			sql:  `DELETE FROM pupil_sen_records WHERE pupil_id IN (?, ?, ?, ?)`,
			args: []any{p1, p2, p3, p4},
		},
		{
			sql:  `DELETE FROM pupil_year_enrolments WHERE pupil_id IN (?, ?, ?, ?)`,
			args: []any{p1, p2, p3, p4},
		},
		{
			sql:  `DELETE FROM pupils WHERE id IN (?, ?, ?, ?)`,
			args: []any{p1, p2, p3, p4},
		},
		{
			sql:  `DELETE FROM staff WHERE id IN (?, ?)`,
			args: []any{"stf_cadm_" + sfx, "stf_ctch_" + sfx},
		},
		{
			sql:  `DELETE FROM person_roles WHERE person_id IN (?, ?)`,
			args: []any{"per_cadm_" + sfx, "per_ctch_" + sfx},
		},
		{
			sql: `DELETE FROM persons WHERE id IN (?, ?, ?, ?, ?, ?)`,
			args: []any{
				"per_cadm_" + sfx,
				"per_ctch_" + sfx,
				"per_cp1_" + sfx,
				"per_cp2_" + sfx,
				"per_cp3_" + sfx,
				"per_cp4_" + sfx,
			},
		},
	}

	for _, q := range cleanupStmts {
		if _, err := tx.ExecContext(ctx, q.sql, q.args...); err != nil {
			return fmt.Errorf("failed to execute census cleanup query (%s): %w", q.sql, err)
		}
	}

	return tx.Commit()
}
