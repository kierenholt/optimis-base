package dal

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type PersonaTestFixtures struct {
	FixtureID                string `json:"fixture_id"`
	PupilID                  string `json:"pupil_id"`
	AdminStaffID             string `json:"admin_staff_id"`
	DSLStaffID               string `json:"dsl_staff_id"`
	HoDStaffID               string `json:"hod_staff_id"`
	TeacherStaffID           string `json:"teacher_staff_id"`
	AttendanceOfficerStaffID string `json:"attendance_officer_staff_id"`
	SchoolNurseStaffID       string `json:"school_nurse_staff_id"`
	TerminatedStaffID        string `json:"terminated_staff_id"`
}

func (d *CoreDAL) SeedPersonaTestFixtures(ctx context.Context, academicYearID string, effectiveFrom string) (*PersonaTestFixtures, error) {
	if strings.TrimSpace(academicYearID) == "" {
		return nil, errors.New("missing mandatory parameter: academic_year_id")
	}
	if strings.TrimSpace(effectiveFrom) == "" {
		return nil, errors.New("missing mandatory parameter: effective_from")
	}

	fxID := strings.ReplaceAll(uuid.NewString(), "-", "")[:10]
	out := &PersonaTestFixtures{
		FixtureID:                fxID,
		PupilID:                  "pup_pf_" + fxID,
		AdminStaffID:             "stf_admin_" + fxID,
		DSLStaffID:               "stf_dsl_" + fxID,
		HoDStaffID:               "stf_hod_" + fxID,
		TeacherStaffID:           "stf_tch_" + fxID,
		AttendanceOfficerStaffID: "stf_att_" + fxID,
		SchoolNurseStaffID:       "stf_att_" + fxID,
		TerminatedStaffID:        "stf_term_" + fxID,
	}

	tx, err := d.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin SeedPersonaTestFixtures transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO academic_years (id, name, status, sequence_number, first_day, last_day)
		 VALUES (?, ?, 'CURRENT', (SELECT COALESCE(MAX(sequence_number), 0) + 1 FROM academic_years), ?, '2099-08-31')
		 ON CONFLICT(id) DO NOTHING`,
		academicYearID, academicYearID, effectiveFrom,
	); err != nil {
		return nil, fmt.Errorf("failed to ensure academic year: %w", err)
	}

	roles := []struct {
		id, cat, name, short, desc, scopes string
	}{
		{
			"ADMIN", "STAFF", "System Administrator / Headteacher", "ADM", "Full school administrative access",
			`["dal.census.read:all","dal.pupils.read:all","dal.pupils.write:all","dal.staff.read:admin","dal.staff.write:admin","dal.groups.read:all","dal.groups.write:all","dal.attendance.read:all","dal.attendance.write:all","dal.guardians.read:all","dal.guardians.write:all","dal.sen.read:all","dal.medical.read:all","plugin.config.write"]`,
		},
		{
			"DSL", "STAFF", "Designated Safeguarding Lead", "DSL", "KCSIE Safeguarding Vault read & triage authority",
			`["dal.safeguarding.create","dal.safeguarding.read","dal.pupils.read:all","dal.guardians.read:all","dal.attendance.read:all","dal.sen.read:all","dal.medical.read:all"]`,
		},
		{
			"HOD", "STAFF", "Head of Department", "HOD", "Departmental curriculum and cohort lead",
			`["dal.pupils.read:all","dal.groups.read:all","dal.groups.write:all","dal.attendance.read:all","dal.sen.read:all","dal.medical.read:all","dal.safeguarding.create","plugin.config.write"]`,
		},
		{
			"TEACHER", "STAFF", "Classroom Teacher", "TCH", "Scoped access to assigned teaching and registration groups",
			`["dal.pupils.read:own_groups","dal.groups.read:own_groups","dal.attendance.read:own_groups","dal.attendance.write:own_groups","dal.guardians.read:own_groups","dal.sen.read:own_groups","dal.medical.read:own_groups","dal.safeguarding.create"]`,
		},
		{
			"SUPPORT", "STAFF", "School Office & Pastoral Support", "SUP", "Attendance officer, First Aider, and administrative support",
			`["dal.pupils.read:all","dal.attendance.read:all","dal.attendance.write:all","dal.guardians.read:all","dal.medical.read:all","dal.safeguarding.create"]`,
		},
	}
	for _, r := range roles {
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO roles (id, category, name, name_short, description, default_scopes_json)
			 VALUES (?, ?, ?, ?, ?, ?)
			 ON CONFLICT(id) DO UPDATE SET default_scopes_json = excluded.default_scopes_json`,
			r.id, r.cat, r.name, r.short, r.desc, r.scopes,
		); err != nil {
			return nil, fmt.Errorf("failed to upsert role %s: %w", r.id, err)
		}
	}

	staffSpecs := []struct {
		staffID   string
		personID  string
		ssoSub    string
		code      string
		title     string
		first     string
		last      string
		staffType string
		jobTitle  string
		roleID    string
		isDSL     int
		termDate  any
	}{
		{out.AdminStaffID, "per_adm_" + fxID, "sso_adm_" + fxID, "ADM" + fxID[:3], "Dame", "Eleanor", "Vance", "LEADERSHIP", "Executive Headteacher", "ADMIN", 1, nil},
		{out.DSLStaffID, "per_dsl_" + fxID, "sso_dsl_" + fxID, "DSL" + fxID[:3], "Mr", "Marcus", "Thorne", "LEADERSHIP", "Designated Safeguarding Lead", "DSL", 1, nil},
		{out.HoDStaffID, "per_hod_" + fxID, "sso_hod_" + fxID, "HOD" + fxID[:3], "Mr", "David", "Sterling", "TEACHING", "Head of Mathematics", "HOD", 0, nil},
		{out.TeacherStaffID, "per_tch_" + fxID, "sso_tch_" + fxID, "TCH" + fxID[:3], "Ms", "Hannah", "Abbott", "TEACHING", "Teacher of Mathematics", "TEACHER", 0, nil},
		{out.AttendanceOfficerStaffID, "per_att_" + fxID, "sso_att_" + fxID, "ATT" + fxID[:3], "Mrs", "Clara", "Oswald", "SUPPORT", "Attendance Officer", "SUPPORT", 0, nil},
		{out.TerminatedStaffID, "per_trm_" + fxID, "sso_trm_" + fxID, "TRM" + fxID[:3], "Mr", "Former", "Staff", "TEACHING", "Former Teacher", "TEACHER", 0, "2026-08-31"},
	}

	for _, s := range staffSpecs {
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO persons (id, external_sso_sub, title, legal_first_name, legal_last_name, preferred_first_name, preferred_last_name, sex, status)
			 VALUES (?, ?, ?, ?, ?, ?, ?, 'U', 'ACTIVE')`,
			s.personID, s.ssoSub, s.title, s.first, s.last, s.first, s.last,
		); err != nil {
			return nil, fmt.Errorf("failed to insert person %s: %w", s.personID, err)
		}
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO person_roles (person_id, role_id, is_primary, effective_from)
			 VALUES (?, ?, 1, ?)`,
			s.personID, s.roleID, effectiveFrom,
		); err != nil {
			return nil, fmt.Errorf("failed to insert person_role for %s: %w", s.personID, err)
		}
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO staff (id, person_id, staff_code, staff_type, job_title, qts_flag, is_dsl, hire_date, termination_date)
			 VALUES (?, ?, ?, ?, ?, 1, ?, ?, ?)`,
			s.staffID, s.personID, s.code, s.staffType, s.jobTitle, s.isDSL, effectiveFrom, s.termDate,
		); err != nil {
			return nil, fmt.Errorf("failed to insert staff %s: %w", s.staffID, err)
		}
	}

	pupilPersonID := "per_pup_" + fxID
	upn := "P" + strings.ToUpper(fxID) + "01"
	if len(upn) > 13 {
		upn = upn[:13]
	}
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO persons (id, legal_first_name, legal_last_name, sex, status)
		 VALUES (?, 'Ada', 'Lovelace', 'F', 'ACTIVE')`,
		pupilPersonID,
	); err != nil {
		return nil, fmt.Errorf("failed to insert pupil person: %w", err)
	}
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO pupils (id, person_id, upn)
		 VALUES (?, ?, ?)`,
		out.PupilID, pupilPersonID, upn,
	); err != nil {
		return nil, fmt.Errorf("failed to insert pupil: %w", err)
	}
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO pupil_year_enrolments (id, pupil_id, academic_year_id, nc_year_group, effective_from, effective_to)
		 VALUES (?, ?, ?, '7', ?, NULL)`,
		"enr_pf_"+fxID, out.PupilID, academicYearID, effectiveFrom,
	); err != nil {
		return nil, fmt.Errorf("failed to insert pupil enrolment: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit SeedPersonaTestFixtures: %w", err)
	}

	return out, nil
}

func (d *CoreDAL) DeletePersonaTestFixtures(ctx context.Context, fixtureID string) error {
	fxID := strings.TrimSpace(fixtureID)
	if fxID == "" {
		return errors.New("missing mandatory parameter: fixture_id")
	}

	tx, err := d.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin DeletePersonaTestFixtures transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	pupilID := "pup_pf_" + fxID
	pupilPersonID := "per_pup_" + fxID

	if _, err := tx.ExecContext(
		ctx,
		`UPDATE safeguarding_notes SET retention_purge_eligible = 1 WHERE pupil_id = ?`,
		pupilID,
	); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DROP TRIGGER IF EXISTS trg_safeguarding_audit_no_delete;`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM safeguarding_audit_logs WHERE pupil_id = ?`, pupilID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(
		ctx,
		`CREATE TRIGGER IF NOT EXISTS trg_safeguarding_audit_no_delete
		 BEFORE DELETE ON safeguarding_audit_logs
		 BEGIN
		     SELECT RAISE(ABORT, 'Security Violation: Statutory safeguarding audit logs are immutable and cannot be deleted.');
		 END;`,
	); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM safeguarding_notes WHERE pupil_id = ?`, pupilID); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM pupil_year_enrolments WHERE pupil_id = ?`, pupilID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM pupils WHERE id = ?`, pupilID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM persons WHERE id = ?`, pupilPersonID); err != nil {
		return err
	}

	staffPrefixes := []struct {
		staffID  string
		personID string
	}{
		{"stf_admin_" + fxID, "per_adm_" + fxID},
		{"stf_dsl_" + fxID, "per_dsl_" + fxID},
		{"stf_hod_" + fxID, "per_hod_" + fxID},
		{"stf_tch_" + fxID, "per_tch_" + fxID},
		{"stf_att_" + fxID, "per_att_" + fxID},
		{"stf_term_" + fxID, "per_trm_" + fxID},
	}

	for _, sp := range staffPrefixes {
		if _, err := tx.ExecContext(ctx, `DELETE FROM staff WHERE id = ?`, sp.staffID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM person_roles WHERE person_id = ?`, sp.personID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM persons WHERE id = ?`, sp.personID); err != nil {
			return err
		}
	}

	return tx.Commit()
}
