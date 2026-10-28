package dal

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type StaffPersona struct {
	StaffID     string   `json:"staff_id"`
	PersonID    string   `json:"person_id"`
	UserID      string   `json:"user_id"`
	StaffCode   string   `json:"staff_code"`
	DisplayName string   `json:"display_name"`
	JobTitle    string   `json:"job_title"`
	IsDSL       bool     `json:"is_dsl"`
	ActiveRoles []string `json:"active_roles"`
	Scopes      []string `json:"scopes"`
}

type PersonaTestFixtures struct {
	FixtureID         string `json:"fixture_id"`
	PupilID           string `json:"pupil_id"`
	AdminStaffID      string `json:"admin_staff_id"`
	DSLStaffID        string `json:"dsl_staff_id"`
	HoDStaffID        string `json:"hod_staff_id"`
	TeacherStaffID    string `json:"teacher_staff_id"`
	TerminatedStaffID string `json:"terminated_staff_id"`
}

func (d *CoreDAL) GetSystemConfigValue(ctx context.Context, key string) (string, error) {
	trimmedKey := strings.TrimSpace(key)
	if trimmedKey == "" {
		return "", errors.New("missing mandatory parameter: key")
	}

	var val string
	err := d.readDB.QueryRowContext(
		ctx,
		`SELECT value FROM __system_config WHERE key = ?`,
		trimmedKey,
	).Scan(&val)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("missing mandatory __system_config key: %s", trimmedKey)
		}
		return "", fmt.Errorf("failed to query __system_config key %s: %w", trimmedKey, err)
	}
	if strings.TrimSpace(val) == "" {
		return "", fmt.Errorf("__system_config key %s has empty value", trimmedKey)
	}
	return val, nil
}

func formatPersonaDisplayName(title, prefFirst, legalFirst, prefLast, legalLast string) string {
	first := strings.TrimSpace(prefFirst)
	if first == "" {
		first = strings.TrimSpace(legalFirst)
	}
	last := strings.TrimSpace(prefLast)
	if last == "" {
		last = strings.TrimSpace(legalLast)
	}
	t := strings.TrimSpace(title)
	if t != "" {
		return fmt.Sprintf("%s %s %s", t, first, last)
	}
	return strings.TrimSpace(fmt.Sprintf("%s %s", first, last))
}

func (d *CoreDAL) resolveRolesAndScopesForPerson(ctx context.Context, personID string) ([]string, []string, error) {
	rows, err := d.readDB.QueryContext(
		ctx,
		`SELECT r.id, r.default_scopes_json
		 FROM person_roles pr
		 JOIN roles r ON r.id = pr.role_id
		 WHERE pr.person_id = ?
		   AND (pr.effective_to IS NULL OR pr.effective_to >= date('now'))
		 ORDER BY pr.is_primary DESC, r.id ASC`,
		personID,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query person_roles for %s: %w", personID, err)
	}
	defer rows.Close()

	var activeRoles []string
	var scopes []string
	seenScopes := make(map[string]struct{})

	for rows.Next() {
		var roleID, scopesJSON string
		if err := rows.Scan(&roleID, &scopesJSON); err != nil {
			return nil, nil, fmt.Errorf("failed to scan role row for %s: %w", personID, err)
		}
		activeRoles = append(activeRoles, roleID)

		var parsedScopes []string
		if err := json.Unmarshal([]byte(scopesJSON), &parsedScopes); err != nil {
			return nil, nil, fmt.Errorf("invalid default_scopes_json on role %s: %w", roleID, err)
		}
		for _, s := range parsedScopes {
			trimmed := strings.TrimSpace(s)
			if trimmed == "" {
				continue
			}
			if _, exists := seenScopes[trimmed]; !exists {
				seenScopes[trimmed] = struct{}{}
				scopes = append(scopes, trimmed)
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("error iterating roles for %s: %w", personID, err)
	}

	return activeRoles, scopes, nil
}

func (d *CoreDAL) ListActiveStaffPersonas(ctx context.Context) ([]StaffPersona, error) {
	rows, err := d.readDB.QueryContext(
		ctx,
		`SELECT stf.id, stf.person_id, COALESCE(per.external_sso_sub, ''), COALESCE(stf.staff_code, ''),
		        per.title, per.preferred_first_name, per.legal_first_name,
		        per.preferred_last_name, per.legal_last_name, stf.job_title, stf.is_dsl
		 FROM staff stf
		 JOIN persons per ON per.id = stf.person_id
		 WHERE per.status = 'ACTIVE'
		   AND (stf.termination_date IS NULL OR TRIM(stf.termination_date) = '')
		 ORDER BY stf.id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list active staff personas: %w", err)
	}

	type rawStaff struct {
		staffID   string
		personID  string
		ssoSub    string
		staffCode string
		title     string
		prefFirst string
		legFirst  string
		prefLast  string
		legLast   string
		jobTitle  string
		isDSL     int
	}
	var rawList []rawStaff
	for rows.Next() {
		var r rawStaff
		if err := rows.Scan(
			&r.staffID, &r.personID, &r.ssoSub, &r.staffCode,
			&r.title, &r.prefFirst, &r.legFirst,
			&r.prefLast, &r.legLast, &r.jobTitle, &r.isDSL,
		); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("failed to scan staff persona row: %w", err)
		}
		rawList = append(rawList, r)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	_ = rows.Close()

	personas := make([]StaffPersona, 0, len(rawList))
	for _, r := range rawList {
		activeRoles, scopes, err := d.resolveRolesAndScopesForPerson(ctx, r.personID)
		if err != nil {
			return nil, err
		}
		if len(activeRoles) == 0 {
			continue
		}
		userID := strings.TrimSpace(r.ssoSub)
		if userID == "" {
			userID = r.personID
		}
		personas = append(personas, StaffPersona{
			StaffID:     r.staffID,
			PersonID:    r.personID,
			UserID:      userID,
			StaffCode:   r.staffCode,
			DisplayName: formatPersonaDisplayName(r.title, r.prefFirst, r.legFirst, r.prefLast, r.legLast),
			JobTitle:    r.jobTitle,
			IsDSL:       r.isDSL == 1,
			ActiveRoles: activeRoles,
			Scopes:      scopes,
		})
	}

	return personas, nil
}

func (d *CoreDAL) ResolveStaffPersonaByID(ctx context.Context, staffID string) (*StaffPersona, error) {
	trimmedStaffID := strings.TrimSpace(staffID)
	if trimmedStaffID == "" {
		return nil, errors.New("missing mandatory parameter: staff_id")
	}

	var personID, ssoSub, staffCode, title, prefFirst, legFirst, prefLast, legLast, jobTitle, personStatus string
	var termDate sql.NullString
	var isDSL int

	err := d.readDB.QueryRowContext(
		ctx,
		`SELECT stf.person_id, COALESCE(per.external_sso_sub, ''), COALESCE(stf.staff_code, ''),
		        per.title, per.preferred_first_name, per.legal_first_name,
		        per.preferred_last_name, per.legal_last_name, stf.job_title, stf.is_dsl,
		        per.status, stf.termination_date
		 FROM staff stf
		 JOIN persons per ON per.id = stf.person_id
		 WHERE stf.id = ?`,
		trimmedStaffID,
	).Scan(
		&personID, &ssoSub, &staffCode,
		&title, &prefFirst, &legFirst,
		&prefLast, &legLast, &jobTitle, &isDSL,
		&personStatus, &termDate,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("staff persona not found: %s", trimmedStaffID)
		}
		return nil, fmt.Errorf("failed to query staff persona %s: %w", trimmedStaffID, err)
	}

	if personStatus != "ACTIVE" {
		return nil, fmt.Errorf("staff persona %s is not ACTIVE (status=%s)", trimmedStaffID, personStatus)
	}
	if termDate.Valid && strings.TrimSpace(termDate.String) != "" {
		return nil, fmt.Errorf("staff persona %s is terminated (termination_date=%s)", trimmedStaffID, termDate.String)
	}

	activeRoles, scopes, err := d.resolveRolesAndScopesForPerson(ctx, personID)
	if err != nil {
		return nil, err
	}
	if len(activeRoles) == 0 {
		return nil, fmt.Errorf("staff persona %s has no active roles", trimmedStaffID)
	}

	userID := strings.TrimSpace(ssoSub)
	if userID == "" {
		userID = personID
	}

	return &StaffPersona{
		StaffID:     trimmedStaffID,
		PersonID:    personID,
		UserID:      userID,
		StaffCode:   staffCode,
		DisplayName: formatPersonaDisplayName(title, prefFirst, legFirst, prefLast, legLast),
		JobTitle:    jobTitle,
		IsDSL:       isDSL == 1,
		ActiveRoles: activeRoles,
		Scopes:      scopes,
	}, nil
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
		FixtureID:         fxID,
		PupilID:           "pup_pf_" + fxID,
		AdminStaffID:      "stf_admin_" + fxID,
		DSLStaffID:        "stf_dsl_" + fxID,
		HoDStaffID:        "stf_hod_" + fxID,
		TeacherStaffID:    "stf_tch_" + fxID,
		TerminatedStaffID: "stf_term_" + fxID,
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
			`["dal.census.read:all","dal.pupils.read:all","dal.pupils.write:all","dal.staff.read:admin","dal.staff.write:admin","dal.groups.read:all","dal.groups.write:all","dal.attendance.read:all","dal.attendance.write:all","dal.guardians.read:all","dal.guardians.write:all","dal.sen.read:all","dal.medical.read:all"]`,
		},
		{
			"DSL", "STAFF", "Designated Safeguarding Lead", "DSL", "KCSIE Safeguarding Vault read & triage authority",
			`["dal.safeguarding.create","dal.safeguarding.read","dal.pupils.read:all","dal.guardians.read:all","dal.attendance.read:all","dal.sen.read:all","dal.medical.read:all"]`,
		},
		{
			"HOD", "STAFF", "Head of Department", "HOD", "Departmental curriculum and cohort lead",
			`["dal.pupils.read:all","dal.groups.read:all","dal.attendance.read:all","dal.sen.read:all","dal.medical.read:all","dal.safeguarding.create"]`,
		},
		{
			"TEACHER", "STAFF", "Classroom Teacher", "TCH", "Scoped access to assigned teaching and registration groups",
			`["dal.pupils.read:own_groups","dal.groups.read:own_groups","dal.attendance.read:own_groups","dal.attendance.write:own_groups","dal.sen.read:own_groups","dal.medical.read:own_groups","dal.safeguarding.create"]`,
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
