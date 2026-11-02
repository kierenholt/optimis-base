package dal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// ============================================================================
// SQL Projections & Shared Field Constants
// ============================================================================

const staffSummaryFieldsSQL = `
	s.id, s.person_id, s.staff_code,
	COALESCE(per.title, ''),
	COALESCE(NULLIF(TRIM(per.preferred_first_name), ''), per.legal_first_name),
	COALESCE(NULLIF(TRIM(per.preferred_last_name), ''), per.legal_last_name),
	per.legal_first_name, per.legal_last_name,
	COALESCE(s.job_title, ''), COALESCE(s.staff_type, ''), s.is_dsl
`

const staffMemberDetailFieldsSQL = `
	s.id, s.person_id, s.staff_code,
	COALESCE(per.title, ''),
	COALESCE(NULLIF(TRIM(per.preferred_first_name), ''), per.legal_first_name),
	COALESCE(NULLIF(TRIM(per.preferred_last_name), ''), per.legal_last_name),
	per.legal_first_name, per.legal_last_name,
	COALESCE(per.email, ''), COALESCE(per.phone_primary, ''),
	COALESCE(per.sex, 'U'), COALESCE(per.dob, ''), COALESCE(per.status, 'ACTIVE'),
	COALESCE(s.job_title, ''), COALESCE(s.staff_type, ''), COALESCE(s.trn, ''),
	s.qts_flag, s.eyts_flag, s.hlta_flag, s.is_dsl,
	s.exclude_from_cover, s.cover_priority,
	COALESCE(s.hire_date, ''), COALESCE(s.termination_date, '')
`

// ============================================================================
// Internal Helpers & Row Scanners
// ============================================================================

func optTrimmedString(s string) any {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func boolPtrToAny(b *bool) any {
	if b == nil {
		return nil
	}
	if *b {
		return 1
	}
	return 0
}

func scanStaffSummary(s rowScanner) (StaffSummary, error) {
	var st StaffSummary
	var title, prefFirst, prefLast, legFirst, legLast string
	var isDSLInt int
	if err := s.Scan(
		&st.StaffID, &st.PersonID, &st.StaffCode,
		&title, &prefFirst, &prefLast, &legFirst, &legLast,
		&st.JobTitle, &st.StaffType, &isDSLInt,
	); err != nil {
		return StaffSummary{}, err
	}
	st.FirstName = prefFirst
	st.LastName = prefLast
	st.DisplayName = formatPersonaDisplayName(title, prefFirst, legFirst, prefLast, legLast)
	st.IsDSL = isDSLInt == 1
	return st, nil
}

func scanStaffMemberDetail(s rowScanner) (StaffMemberDetail, error) {
	var st StaffMemberDetail
	var prefFirst, prefLast, legFirst, legLast string
	var qtsInt, eytsInt, hltaInt, dslInt, exclInt int
	if err := s.Scan(
		&st.StaffID, &st.PersonID, &st.StaffCode,
		&st.Title, &prefFirst, &prefLast, &legFirst, &legLast,
		&st.Email, &st.PhonePrimary,
		&st.Sex, &st.DOB, &st.Status,
		&st.JobTitle, &st.StaffType, &st.TRN,
		&qtsInt, &eytsInt, &hltaInt, &dslInt,
		&exclInt, &st.CoverPriority,
		&st.HireDate, &st.TerminationDate,
	); err != nil {
		return StaffMemberDetail{}, err
	}
	st.FirstName = legFirst
	st.LastName = legLast
	st.PreferredFirstName = prefFirst
	st.PreferredLastName = prefLast
	st.DisplayName = formatPersonaDisplayName(st.Title, prefFirst, legFirst, prefLast, legLast)
	st.QTSFlag = qtsInt == 1
	st.EYTSFlag = eytsInt == 1
	st.HLTAFlag = hltaInt == 1
	st.IsDSL = dslInt == 1
	st.ExcludeFromCover = exclInt == 1
	return st, nil
}

func scanRoleSummary(s rowScanner) (RoleSummary, error) {
	var r RoleSummary
	if err := s.Scan(&r.ID, &r.Category, &r.Name, &r.NameShort, &r.Description, &r.DefaultScopesJSON); err != nil {
		return RoleSummary{}, err
	}
	return r, nil
}

// ============================================================================
// Section 1: Active Staff & Directory Queries
// ============================================================================

type StaffSummary struct {
	StaffID     string `json:"staff_id"`
	PersonID    string `json:"person_id"`
	StaffCode   string `json:"staff_code"`
	DisplayName string `json:"display_name"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	JobTitle    string `json:"job_title"`
	StaffType   string `json:"staff_type"`
	IsDSL       bool   `json:"is_dsl"`
}

func (d *CoreDAL) ListActiveStaff(ctx context.Context) ([]StaffSummary, error) {
	rows, err := d.readDB.QueryContext(
		ctx,
		`SELECT `+staffSummaryFieldsSQL+`
		 FROM staff s
		 JOIN persons per ON per.id = s.person_id
		 WHERE per.status = 'ACTIVE'
		   AND (s.termination_date IS NULL OR TRIM(s.termination_date) = '')
		 ORDER BY per.legal_last_name ASC, per.legal_first_name ASC, s.id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list active staff: %w", err)
	}
	defer rows.Close()

	result := make([]StaffSummary, 0)
	for rows.Next() {
		st, err := scanStaffSummary(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan active staff row: %w", err)
		}
		result = append(result, st)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating active staff: %w", err)
	}
	return result, nil
}

func (d *CoreDAL) GetStaffByID(ctx context.Context, staffID string) (*StaffSummary, error) {
	trimmed := strings.TrimSpace(staffID)
	if trimmed == "" {
		return nil, errors.New("missing mandatory parameter: staff_id")
	}

	row := d.readDB.QueryRowContext(
		ctx,
		`SELECT `+staffSummaryFieldsSQL+`
		 FROM staff s
		 JOIN persons per ON per.id = s.person_id
		 WHERE s.id = ?
		   AND per.status = 'ACTIVE'
		   AND (s.termination_date IS NULL OR TRIM(s.termination_date) = '')`,
		trimmed,
	)
	st, err := scanStaffSummary(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("active staff not found: %s", trimmed)
		}
		return nil, fmt.Errorf("failed to query staff %s: %w", trimmed, err)
	}
	return &st, nil
}

type ListStaffOptions struct {
	IncludeLeavers bool   `json:"include_leavers"`
	StaffType      string `json:"staff_type,omitempty"`
}

type StaffMemberDetail struct {
	StaffID            string `json:"staff_id"`
	PersonID           string `json:"person_id"`
	StaffCode          string `json:"staff_code"`
	DisplayName        string `json:"display_name"`
	Title              string `json:"title,omitempty"`
	FirstName          string `json:"first_name"`
	LastName           string `json:"last_name"`
	PreferredFirstName string `json:"preferred_first_name,omitempty"`
	PreferredLastName  string `json:"preferred_last_name,omitempty"`
	Email              string `json:"email,omitempty"`
	PhonePrimary       string `json:"phone_primary,omitempty"`
	Sex                string `json:"sex,omitempty"`
	DOB                string `json:"dob,omitempty"`
	Status             string `json:"status,omitempty"`
	JobTitle           string `json:"job_title"`
	StaffType          string `json:"staff_type"`
	TRN                string `json:"trn,omitempty"`
	QTSFlag            bool   `json:"qts_flag"`
	EYTSFlag           bool   `json:"eyts_flag"`
	HLTAFlag           bool   `json:"hlta_flag"`
	IsDSL              bool   `json:"is_dsl"`
	ExcludeFromCover   bool   `json:"exclude_from_cover"`
	CoverPriority      int    `json:"cover_priority"`
	HireDate           string `json:"hire_date,omitempty"`
	TerminationDate    string `json:"termination_date,omitempty"`
}

func (d *CoreDAL) ListAllStaff(ctx context.Context, opts ListStaffOptions) ([]StaffMemberDetail, error) {
	query := `
		SELECT ` + staffMemberDetailFieldsSQL + `
		FROM staff s
		JOIN persons per ON per.id = s.person_id
		WHERE 1=1`
	var args []any
	if !opts.IncludeLeavers {
		query += ` AND per.status = 'ACTIVE' AND (s.termination_date IS NULL OR TRIM(s.termination_date) = '')`
	}
	if strings.TrimSpace(opts.StaffType) != "" {
		query += ` AND s.staff_type = ?`
		args = append(args, strings.ToUpper(strings.TrimSpace(opts.StaffType)))
	}
	query += ` ORDER BY per.legal_last_name ASC, per.legal_first_name ASC, s.id ASC`

	rows, err := d.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list staff: %w", err)
	}
	defer rows.Close()

	result := make([]StaffMemberDetail, 0)
	for rows.Next() {
		st, err := scanStaffMemberDetail(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan staff row: %w", err)
		}
		result = append(result, st)
	}
	return result, rows.Err()
}

type UpsertStaffMemberInput struct {
	StaffID            string   `json:"staff_id,omitempty"`
	PersonID           string   `json:"person_id,omitempty"`
	StaffCode          string   `json:"staff_code"`
	Title              string   `json:"title,omitempty"`
	FirstName          string   `json:"first_name"`
	LastName           string   `json:"last_name"`
	PreferredFirstName string   `json:"preferred_first_name,omitempty"`
	PreferredLastName  string   `json:"preferred_last_name,omitempty"`
	Email              string   `json:"email,omitempty"`
	PhonePrimary       string   `json:"phone_primary,omitempty"`
	Sex                string   `json:"sex,omitempty"`
	DOB                string   `json:"dob,omitempty"`
	Status             string   `json:"status,omitempty"`
	StaffType          string   `json:"staff_type"`
	JobTitle           string   `json:"job_title"`
	TRN                string   `json:"trn,omitempty"`
	QTSFlag            bool     `json:"qts_flag"`
	EYTSFlag           bool     `json:"eyts_flag"`
	HLTAFlag           bool     `json:"hlta_flag"`
	IsDSL              bool     `json:"is_dsl"`
	ExcludeFromCover   bool     `json:"exclude_from_cover"`
	CoverPriority      int      `json:"cover_priority"`
	HireDate           string   `json:"hire_date"`
	TerminationDate    string   `json:"termination_date,omitempty"`
	Roles              []string `json:"roles,omitempty"`
}

func (d *CoreDAL) UpsertStaffMember(ctx context.Context, in UpsertStaffMemberInput) (*StaffMemberDetail, error) {
	staffCode := strings.ToUpper(strings.TrimSpace(in.StaffCode))
	firstName := strings.TrimSpace(in.FirstName)
	lastName := strings.TrimSpace(in.LastName)
	staffType := strings.ToUpper(strings.TrimSpace(in.StaffType))
	if staffType == "" {
		staffType = "TEACHING"
	}
	hireDate := strings.TrimSpace(in.HireDate)
	if hireDate == "" {
		hireDate = "2026-09-01"
	}
	if staffCode == "" || firstName == "" || lastName == "" {
		return nil, errors.New("missing mandatory parameter: staff_code, first_name, and last_name are required")
	}
	switch staffType {
	case "TEACHING", "SUPPORT", "LEADERSHIP", "SUPPLY_INTERNAL", "SUPPLY_AGENCY":
	default:
		return nil, fmt.Errorf("invalid staff_type %q", in.StaffType)
	}

	uid := strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	staffID := strings.TrimSpace(in.StaffID)
	if staffID == "" {
		staffID = "stf_" + uid
	}
	personID := strings.TrimSpace(in.PersonID)
	if personID == "" {
		personID = "per_" + uid
	}

	tx, err := d.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin UpsertStaffMember tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var existingStaffID, existingPersonID string
	err = tx.QueryRowContext(ctx, `SELECT id, person_id FROM staff WHERE staff_code = ? OR id = ? LIMIT 1`, staffCode, staffID).Scan(&existingStaffID, &existingPersonID)
	if err == nil && existingStaffID != "" {
		staffID = existingStaffID
		personID = existingPersonID
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to check existing staff member: %w", err)
	}

	prefFirst := strings.TrimSpace(in.PreferredFirstName)
	if prefFirst == "" {
		prefFirst = firstName
	}
	prefLast := strings.TrimSpace(in.PreferredLastName)
	if prefLast == "" {
		prefLast = lastName
	}
	sex := strings.ToUpper(strings.TrimSpace(in.Sex))
	if sex == "" {
		sex = "U"
	}
	status := strings.ToUpper(strings.TrimSpace(in.Status))
	if status == "" {
		if strings.TrimSpace(in.TerminationDate) != "" {
			status = "LEFT"
		} else {
			status = "ACTIVE"
		}
	}

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO persons (
			id, email, title, legal_first_name, legal_last_name,
			preferred_first_name, preferred_last_name, sex, dob, phone_primary, status
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			email = COALESCE(excluded.email, persons.email),
			title = excluded.title,
			legal_first_name = excluded.legal_first_name,
			legal_last_name = excluded.legal_last_name,
			preferred_first_name = excluded.preferred_first_name,
			preferred_last_name = excluded.preferred_last_name,
			sex = excluded.sex,
			dob = COALESCE(excluded.dob, persons.dob),
			phone_primary = excluded.phone_primary,
			status = excluded.status`,
		personID, optTrimmedString(in.Email), strings.TrimSpace(in.Title), firstName, lastName,
		prefFirst, prefLast, sex, optTrimmedString(in.DOB), strings.TrimSpace(in.PhonePrimary), status,
	); err != nil {
		return nil, fmt.Errorf("failed to upsert person for staff: %w", err)
	}

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO staff (
			id, person_id, staff_code, trn, staff_type, job_title,
			qts_flag, eyts_flag, hlta_flag, is_dsl,
			exclude_from_cover, cover_priority, hire_date, termination_date
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			staff_code = excluded.staff_code,
			trn = COALESCE(excluded.trn, staff.trn),
			staff_type = excluded.staff_type,
			job_title = excluded.job_title,
			qts_flag = excluded.qts_flag,
			eyts_flag = excluded.eyts_flag,
			hlta_flag = excluded.hlta_flag,
			is_dsl = excluded.is_dsl,
			exclude_from_cover = excluded.exclude_from_cover,
			cover_priority = excluded.cover_priority,
			hire_date = excluded.hire_date,
			termination_date = excluded.termination_date`,
		staffID, personID, staffCode, optTrimmedString(in.TRN), staffType, strings.TrimSpace(in.JobTitle),
		boolToInt(in.QTSFlag), boolToInt(in.EYTSFlag), boolToInt(in.HLTAFlag), boolToInt(in.IsDSL),
		boolToInt(in.ExcludeFromCover), in.CoverPriority, hireDate, optTrimmedString(in.TerminationDate),
	); err != nil {
		return nil, fmt.Errorf("failed to upsert staff row: %w", err)
	}

	for idx, rawRole := range in.Roles {
		roleID := strings.ToUpper(strings.TrimSpace(rawRole))
		if roleID == "" {
			continue
		}
		isPrim := 0
		if idx == 0 {
			isPrim = 1
		}
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO person_roles (person_id, role_id, is_primary, effective_from)
			 VALUES (?, ?, ?, ?)
			 ON CONFLICT(person_id, role_id, effective_from) DO NOTHING`,
			personID, roleID, isPrim, hireDate,
		); err != nil {
			return nil, fmt.Errorf("failed to assign role %s to staff %s: %w", roleID, staffID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit UpsertStaffMember tx: %w", err)
	}

	return &StaffMemberDetail{
		StaffID:            staffID,
		PersonID:           personID,
		StaffCode:          staffCode,
		DisplayName:        formatPersonaDisplayName(in.Title, prefFirst, firstName, prefLast, lastName),
		Title:              strings.TrimSpace(in.Title),
		FirstName:          firstName,
		LastName:           lastName,
		PreferredFirstName: prefFirst,
		PreferredLastName:  prefLast,
		Email:              strings.TrimSpace(in.Email),
		PhonePrimary:       strings.TrimSpace(in.PhonePrimary),
		Sex:                sex,
		DOB:                strings.TrimSpace(in.DOB),
		Status:             status,
		JobTitle:           strings.TrimSpace(in.JobTitle),
		StaffType:          staffType,
		TRN:                strings.TrimSpace(in.TRN),
		QTSFlag:            in.QTSFlag,
		EYTSFlag:           in.EYTSFlag,
		HLTAFlag:           in.HLTAFlag,
		IsDSL:              in.IsDSL,
		ExcludeFromCover:   in.ExcludeFromCover,
		CoverPriority:      in.CoverPriority,
		HireDate:           hireDate,
		TerminationDate:    strings.TrimSpace(in.TerminationDate),
	}, nil
}

// ============================================================================
// Section 2: Staff Roles
// ============================================================================

type RoleSummary struct {
	ID                string `json:"id"`
	Category          string `json:"category"`
	Name              string `json:"name"`
	NameShort         string `json:"name_short"`
	Description       string `json:"description,omitempty"`
	DefaultScopesJSON string `json:"default_scopes_json"`
}

func (d *CoreDAL) ListRoles(ctx context.Context) ([]RoleSummary, error) {
	rows, err := d.readDB.QueryContext(
		ctx,
		`SELECT id, category, name, name_short, COALESCE(description, ''), COALESCE(default_scopes_json, '[]')
		 FROM roles
		 ORDER BY category ASC, id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}
	defer rows.Close()

	result := make([]RoleSummary, 0)
	for rows.Next() {
		r, err := scanRoleSummary(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan roles row: %w", err)
		}
		result = append(result, r)
	}
	return result, rows.Err()
}
