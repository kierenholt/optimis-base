package dal

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type GDPRTestFixtureInput struct {
	AcademicYearID   string `json:"academic_year_id"`
	ReferenceDate    string `json:"reference_date"`
	HostKMSKeyHandle string `json:"host_kms_key_handle"`
}

type GDPRTestFixtureResult struct {
	FixtureID          string `json:"fixture_id"`
	AcademicYearID     string `json:"academic_year_id"`
	ReferenceDate      string `json:"reference_date"`
	AdminStaffID       string `json:"admin_staff_id"`
	TeacherStaffID     string `json:"teacher_staff_id"`
	DSLStaffID         string `json:"dsl_staff_id"`
	PupilID            string `json:"pupil_id"`
	ProtectedPupilID   string `json:"protected_pupil_id"`
	PastoralNoteID     string `json:"pastoral_note_id"`
	SafeguardingNoteID string `json:"safeguarding_note_id"`
}

func (d *CoreDAL) SeedGDPRTestFixtures(ctx context.Context, in GDPRTestFixtureInput) (*GDPRTestFixtureResult, error) {
	ayID := strings.TrimSpace(in.AcademicYearID)
	refDate := strings.TrimSpace(in.ReferenceDate)
	kmsKey := strings.TrimSpace(in.HostKMSKeyHandle)
	if ayID == "" {
		return nil, errors.New("missing mandatory field: academic_year_id")
	}
	if refDate == "" {
		return nil, errors.New("missing mandatory field: reference_date")
	}
	if kmsKey == "" {
		kmsKey = "kms-test-key-001"
	}

	fxID := strings.ReplaceAll(uuid.NewString(), "-", "")[:8]
	res := &GDPRTestFixtureResult{
		FixtureID:          fxID,
		AcademicYearID:     ayID,
		ReferenceDate:      refDate,
		AdminStaffID:       "stf_gadm_" + fxID,
		TeacherStaffID:     "stf_gtch_" + fxID,
		DSLStaffID:         "stf_gdsl_" + fxID,
		PupilID:            "pup_gsar_" + fxID,
		ProtectedPupilID:   "pup_gpro_" + fxID,
		PastoralNoteID:     "pas_gsar_" + fxID,
		SafeguardingNoteID: "sfg_gpro_" + fxID,
	}

	tx, err := d.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin SeedGDPRTestFixtures transaction: %w", err)
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
			      VALUES (?, ?, 'CURRENT', (SELECT COALESCE(MAX(sequence_number), 0) + 1 FROM academic_years), '2026-09-01', '2027-07-23')
			      ON CONFLICT(id) DO NOTHING`,
			args: []any{ayID, ayID},
		},
		{
			sql: `INSERT INTO attendance_codes (code, name, statistical_category, physical_status, sequence_number)
			      VALUES ('/', 'Present (AM)', 'PRESENT', 'ONSITE', 1)
			      ON CONFLICT(code) DO NOTHING`,
		},
		{
			sql: `INSERT INTO roles (id, category, name, name_short, description, default_scopes_json)
			      VALUES
			        ('ADMIN', 'STAFF', 'System Administrator / DPO', 'ADM', 'Full administrative & GDPR SAR/Purge authority', '["dal.census.read:all","dal.pupils.read:all","dal.pupils.write:all","dal.staff.read:admin","dal.staff.write:admin","dal.groups.read:all","dal.groups.write:all","dal.attendance.read:all","dal.attendance.write:all","dal.guardians.read:all","dal.guardians.write:all","dal.sen.read:all","dal.medical.read:all"]'),
			        ('DSL', 'STAFF', 'Designated Safeguarding Lead', 'DSL', 'KCSIE Safeguarding Vault authority', '["dal.safeguarding.create","dal.safeguarding.read","dal.pupils.read:all","dal.guardians.read:all","dal.attendance.read:all","dal.sen.read:all","dal.medical.read:all"]'),
			        ('TEACHER', 'STAFF', 'Classroom Teacher', 'TCH', 'Scoped classroom teacher access', '["dal.pupils.read:own_groups","dal.groups.read:own_groups","dal.attendance.read:own_groups","dal.attendance.write:own_groups","dal.sen.read:own_groups","dal.medical.read:own_groups","dal.safeguarding.create"]')
			      ON CONFLICT(id) DO UPDATE SET default_scopes_json = excluded.default_scopes_json`,
		},
		{
			sql: `INSERT INTO persons (id, external_sso_sub, title, legal_first_name, legal_last_name, preferred_first_name, preferred_last_name, dob, sex, phone_primary, status)
			      VALUES
			        (?, ?, 'Dame', 'Eleanor', 'Vance', 'Eleanor', 'Vance', '1975-04-12', 'F', '07700900001', 'ACTIVE'),
			        (?, ?, 'Ms', 'Hannah', 'Abbott', 'Hannah', 'Abbott', '1990-08-19', 'F', '07700900002', 'ACTIVE'),
			        (?, ?, 'Mr', 'Marcus', 'Thorne', 'Marcus', 'Thorne', '1982-11-03', 'M', '07700900003', 'ACTIVE'),
			        (?, NULL, '', 'Clara', 'Oswald', 'Clara', 'Oswald', '2013-11-23', 'F', '', 'ACTIVE'),
			        (?, NULL, '', 'Arthur', 'Pendragon', 'Arthur', 'Pendragon', '2013-05-14', 'M', '', 'ACTIVE'),
			        (?, NULL, 'Mrs', 'Sarah', 'Oswald', 'Sarah', 'Oswald', '1985-02-10', 'F', '07700900123', 'ACTIVE')`,
			args: []any{
				"per_gadm_" + fxID, "sso_gadm_" + fxID,
				"per_gtch_" + fxID, "sso_gtch_" + fxID,
				"per_gdsl_" + fxID, "sso_gdsl_" + fxID,
				"per_gsar_" + fxID,
				"per_gpro_" + fxID,
				"per_gmom_" + fxID,
			},
		},
		{
			sql: `INSERT INTO person_roles (person_id, role_id, is_primary, effective_from)
			      VALUES
			        (?, 'ADMIN', 1, '2026-09-01'),
			        (?, 'TEACHER', 1, '2026-09-01'),
			        (?, 'DSL', 1, '2026-09-01')`,
			args: []any{
				"per_gadm_" + fxID,
				"per_gtch_" + fxID,
				"per_gdsl_" + fxID,
			},
		},
		{
			sql: `INSERT INTO staff (id, person_id, staff_code, staff_type, job_title, qts_flag, is_dsl, hire_date)
			      VALUES
			        (?, ?, ?, 'LEADERSHIP', 'Data Protection Officer & Headteacher', 1, 1, '2019-09-01'),
			        (?, ?, ?, 'TEACHING', 'Teacher of Mathematics', 1, 0, '2022-09-01'),
			        (?, ?, ?, 'LEADERSHIP', 'Designated Safeguarding Lead', 1, 1, '2020-09-01')`,
			args: []any{
				res.AdminStaffID, "per_gadm_" + fxID, "GA_" + fxID[:4],
				res.TeacherStaffID, "per_gtch_" + fxID, "GT_" + fxID[:4],
				res.DSLStaffID, "per_gdsl_" + fxID, "GD_" + fxID[:4],
			},
		},
		{
			sql: `INSERT INTO pupils (id, person_id, upn, ethnicity_code, first_language_code)
			      VALUES
			        (?, ?, ?, 'WBRI', 'ENG'),
			        (?, ?, ?, 'WBRI', 'ENG')`,
			args: []any{
				res.PupilID, "per_gsar_" + fxID, "G1" + strings.ToUpper(fxID) + "001",
				res.ProtectedPupilID, "per_gpro_" + fxID, "G2" + strings.ToUpper(fxID) + "002",
			},
		},
		{
			sql: `INSERT INTO pupil_year_enrolments (id, pupil_id, academic_year_id, nc_year_group, enrolment_status, effective_from, effective_to)
			      VALUES
			        (?, ?, ?, '9', 'C', '2026-09-01', NULL),
			        (?, ?, ?, '9', 'C', '2026-09-01', NULL)`,
			args: []any{
				"enr_gsar_" + fxID, res.PupilID, ayID,
				"enr_gpro_" + fxID, res.ProtectedPupilID, ayID,
			},
		},
		{
			sql: `INSERT INTO guardian_relationships (
				id, pupil_id, guardian_person_id, relationship_code,
				has_parental_responsibility, court_order_restricted, contact_priority,
				is_emergency_contact, authorised_to_collect, receive_reports,
				effective_from, effective_to
			) VALUES (?, ?, ?, 'MOTHER', 1, 0, 1, 1, 1, 1, '2026-09-01', NULL)`,
			args: []any{"grel_gsar_" + fxID, res.PupilID, "per_gmom_" + fxID},
		},
		{
			sql: `INSERT INTO attendance_session_marks (
				id, pupil_id, academic_year_id, date, session, attendance_code, minutes_late, sub_code, comment, recorded_by_staff_id
			) VALUES (?, ?, ?, ?, 'AM', '/', 0, '', 'Arrived on time', ?)`,
			args: []any{"asm_gsar_" + fxID, res.PupilID, ayID, refDate, res.TeacherStaffID},
		},
		{
			sql: `INSERT INTO pupil_sen_records (
				id, pupil_id, provision_code, teaching_strategies, pupil_targets, senco_notes, effective_from, effective_to
			) VALUES (?, ?, 'K', 'Seat near front of classroom', 'Complete independent starter tasks', 'Reviewed with SENCO and Mrs Oswald', '2026-09-01', NULL)`,
			args: []any{"sen_gsar_" + fxID, res.PupilID},
		},
		{
			sql: `INSERT INTO pupil_medical_conditions (
				id, pupil_id, condition_name, condition_category, has_ihcp, emergency_response_plan, parental_consent_given, effective_from, effective_to
			) VALUES (?, ?, 'Mild Asthma', 'MEDICAL', 0, 'Use blue salbutamol inhaler as required', 1, '2026-09-01', NULL)`,
			args: []any{"med_gsar_" + fxID, res.PupilID},
		},
		{
			sql: `INSERT INTO pupil_pastoral_notes (
				id, pupil_id, category, title, note_text, created_by_staff_id, created_at
			) VALUES (?, ?, 'ACADEMIC_WELFARE', 'Corridor incident note', 'Conflict in corridor involving third-party pupil Harry Potter before morning registration.', ?, '2026-10-20T08:45:00Z')`,
			args: []any{res.PastoralNoteID, res.PupilID, res.TeacherStaffID},
		},
		{
			sql: `INSERT INTO safeguarding_notes (
				id, pupil_id, reported_by_staff_id, assigned_dsl_staff_id,
				concern_category, status, incident_date,
				kms_key_id, wrapped_dek, nonce, ciphertext_payload, retention_purge_eligible
			) VALUES (?, ?, ?, ?, 'PHYSICAL', 'OPEN_TRIAGE', ?, ?, X'0102030405060708090A0B0C0D0E0F10', X'0102030405060708090A0B0C', X'AA', 0)`,
			args: []any{res.SafeguardingNoteID, res.ProtectedPupilID, res.TeacherStaffID, res.DSLStaffID, refDate, kmsKey},
		},
	}

	for _, stmt := range stmts {
		if _, err := tx.ExecContext(ctx, stmt.sql, stmt.args...); err != nil {
			return nil, fmt.Errorf("failed executing SeedGDPRTestFixtures statement: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed committing SeedGDPRTestFixtures: %w", err)
	}
	return res, nil
}

func (d *CoreDAL) DeleteGDPRTestFixtures(ctx context.Context, fixtureID string) error {
	fxID := strings.TrimSpace(fixtureID)
	if fxID == "" {
		return errors.New("missing mandatory parameter: fixture_id")
	}

	tx, err := d.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin DeleteGDPRTestFixtures transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	pupilID := "pup_gsar_" + fxID
	protectedPupilID := "pup_gpro_" + fxID
	sfgNoteID := "sfg_gpro_" + fxID

	// Unlock safeguarding note for test fixture teardown
	if _, err := tx.ExecContext(ctx, `UPDATE safeguarding_notes SET retention_purge_eligible = 1 WHERE id = ?`, sfgNoteID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM safeguarding_notes WHERE id = ?`, sfgNoteID); err != nil {
		return err
	}

	for _, pid := range []string{pupilID, protectedPupilID} {
		for _, q := range []string{
			`DELETE FROM pupil_pastoral_notes WHERE pupil_id = ?`,
			`DELETE FROM pupil_medical_conditions WHERE pupil_id = ?`,
			`DELETE FROM pupil_sen_records WHERE pupil_id = ?`,
			`DELETE FROM attendance_session_marks WHERE pupil_id = ?`,
			`DELETE FROM guardian_relationships WHERE pupil_id = ?`,
			`DELETE FROM pupil_year_enrolments WHERE pupil_id = ?`,
			`DELETE FROM pupils WHERE id = ?`,
		} {
			if _, err := tx.ExecContext(ctx, q, pid); err != nil {
				return err
			}
		}
	}

	for _, stfID := range []string{"stf_gadm_" + fxID, "stf_gtch_" + fxID, "stf_gdsl_" + fxID} {
		if _, err := tx.ExecContext(ctx, `DELETE FROM staff WHERE id = ?`, stfID); err != nil {
			return err
		}
	}

	for _, perID := range []string{
		"per_gadm_" + fxID,
		"per_gtch_" + fxID,
		"per_gdsl_" + fxID,
		"per_gsar_" + fxID,
		"per_gpro_" + fxID,
		"per_gmom_" + fxID,
	} {
		if _, err := tx.ExecContext(ctx, `DELETE FROM person_roles WHERE person_id = ?`, perID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM persons WHERE id = ?`, perID); err != nil {
			return err
		}
	}

	return tx.Commit()
}
