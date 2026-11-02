package dal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type Pupil struct {
	ID                    string  `json:"id"`
	PupilID               string  `json:"pupil_id,omitempty"`
	PersonID              string  `json:"person_id,omitempty"`
	UPN                   string  `json:"upn"`
	ULN                   string  `json:"uln,omitempty"`
	FormerUPN             string  `json:"former_upn,omitempty"`
	NHSNumber             string  `json:"nhs_number,omitempty"`
	AdmissionNumber       string  `json:"admission_number,omitempty"`
	FirstName             string  `json:"first_name"`
	LastName              string  `json:"last_name"`
	LegalFirstName        string  `json:"legal_first_name,omitempty"`
	LegalMiddleNames      string  `json:"legal_middle_names,omitempty"`
	LegalLastName         string  `json:"legal_last_name,omitempty"`
	PreferredFirstName    string  `json:"preferred_first_name,omitempty"`
	PreferredLastName     string  `json:"preferred_last_name,omitempty"`
	FormerLastName        string  `json:"former_last_name,omitempty"`
	Sex                   string  `json:"sex,omitempty"`
	GenderIdentity        string  `json:"gender_identity,omitempty"`
	DOB                   string  `json:"dob,omitempty"`
	Email                 string  `json:"email,omitempty"`
	PhonePrimary          string  `json:"phone_primary,omitempty"`
	PhotoPath             string  `json:"photo_path,omitempty"`
	Status                string  `json:"status,omitempty"`
	EthnicityCode         string  `json:"ethnicity_code,omitempty"`
	FirstLanguageCode     string  `json:"first_language_code,omitempty"`
	EALFlag               bool    `json:"eal_flag"`
	CountryOfBirth        string  `json:"country_of_birth,omitempty"`
	ReligionCode          string  `json:"religion_code,omitempty"`
	PupilPremiumFlag      bool    `json:"pupil_premium_flag"`
	EarlyYearsPPFlag      bool    `json:"early_years_pp_flag"`
	ServiceChildFlag      bool    `json:"service_child_flag"`
	LookedAfterFlag       bool    `json:"looked_after_flag"`
	PostLookedAfterCode   string  `json:"post_looked_after_code,omitempty"`
	YoungCarerCode        string  `json:"young_carer_code,omitempty"`
	NCYearGroup           string  `json:"nc_year_group,omitempty"`
	EnrolmentStatus       string  `json:"enrolment_status,omitempty"`
	RegistrationGroupID   string  `json:"registration_group_id,omitempty"`
	HouseID               string  `json:"house_id,omitempty"`
	BoarderStatus         string  `json:"boarder_status,omitempty"`
	PartTimeFlag          bool    `json:"part_time_flag"`
	EYFundedHours         float64 `json:"ey_funded_hours,omitempty"`
	EYExtendedFundedHours float64 `json:"ey_extended_funded_hours,omitempty"`
	EYThirtyHourCode      string  `json:"ey_thirty_hour_code,omitempty"`
	EYDAFIndicator        bool    `json:"ey_daf_indicator"`
	PreviousSchoolURN     string  `json:"previous_school_urn,omitempty"`
	DestinationSchoolURN  string  `json:"destination_school_urn,omitempty"`
	LeavingReasonCode     string  `json:"leaving_reason_code,omitempty"`
	AcademicYearID        string  `json:"academic_year_id"`
	EffectiveFrom         string  `json:"effective_from"`
	EffectiveTo           *string `json:"effective_to,omitempty"`
}

const pupilBaseSelectColumns = `
	p.id, p.person_id, p.upn, COALESCE(p.uln, ''), COALESCE(p.former_upn, ''),
	COALESCE(p.nhs_number, ''), COALESCE(p.admission_number, ''),
	per.legal_first_name, COALESCE(per.legal_middle_names, ''), per.legal_last_name,
	COALESCE(NULLIF(TRIM(per.preferred_first_name), ''), per.legal_first_name),
	COALESCE(NULLIF(TRIM(per.preferred_last_name), ''), per.legal_last_name),
	COALESCE(per.former_last_name, ''), COALESCE(per.sex, 'U'), COALESCE(per.gender_identity, ''),
	COALESCE(per.dob, ''), COALESCE(per.email, ''), COALESCE(per.phone_primary, ''),
	COALESCE(per.photo_path, ''), COALESCE(per.status, 'ACTIVE'),
	COALESCE(p.ethnicity_code, ''), COALESCE(p.first_language_code, ''),
	p.eal_flag, COALESCE(p.country_of_birth, 'GBR'), COALESCE(p.religion_code, ''),
	p.pupil_premium_flag, p.early_years_pp_flag, p.service_child_flag, p.looked_after_flag,
	COALESCE(p.post_looked_after_code, 'N'), COALESCE(p.young_carer_code, 'N'),
	COALESCE(enr.nc_year_group, ''), COALESCE(enr.enrolment_status, 'C'),
	COALESCE(enr.registration_group_id, ''), COALESCE(enr.house_id, ''),
	COALESCE(enr.boarder_status, 'N'), COALESCE(enr.part_time_flag, 0),
	COALESCE(enr.ey_funded_hours, 0), COALESCE(enr.ey_extended_funded_hours, 0),
	COALESCE(enr.ey_thirty_hour_code, ''), COALESCE(enr.ey_daf_indicator, 0),
	COALESCE(enr.previous_school_urn, ''), COALESCE(enr.destination_school_urn, ''),
	COALESCE(enr.leaving_reason_code, '')`

const pupilSelectColumns = pupilBaseSelectColumns + `,
	enr.academic_year_id, enr.effective_from, enr.effective_to`

type rowScanner interface {
	Scan(dest ...any) error
}

func optionalTrimmedString(s string) any {
	if trimmed := strings.TrimSpace(s); trimmed != "" {
		return trimmed
	}
	return nil
}

func scanPupilRow(s rowScanner) (Pupil, error) {
	var p Pupil
	var ealInt, ppInt, eyppInt, lacInt, ptInt, eydafInt int
	var svcStr string
	var effTo sql.NullString
	err := s.Scan(
		&p.ID, &p.PersonID, &p.UPN, &p.ULN, &p.FormerUPN,
		&p.NHSNumber, &p.AdmissionNumber,
		&p.FirstName, &p.LegalMiddleNames, &p.LastName,
		&p.PreferredFirstName, &p.PreferredLastName,
		&p.FormerLastName, &p.Sex, &p.GenderIdentity,
		&p.DOB, &p.Email, &p.PhonePrimary,
		&p.PhotoPath, &p.Status,
		&p.EthnicityCode, &p.FirstLanguageCode,
		&ealInt, &p.CountryOfBirth, &p.ReligionCode,
		&ppInt, &eyppInt, &svcStr, &lacInt,
		&p.PostLookedAfterCode, &p.YoungCarerCode,
		&p.NCYearGroup, &p.EnrolmentStatus,
		&p.RegistrationGroupID, &p.HouseID,
		&p.BoarderStatus, &ptInt,
		&p.EYFundedHours, &p.EYExtendedFundedHours,
		&p.EYThirtyHourCode, &eydafInt,
		&p.PreviousSchoolURN, &p.DestinationSchoolURN,
		&p.LeavingReasonCode,
		&p.AcademicYearID, &p.EffectiveFrom, &effTo,
	)
	if err != nil {
		return Pupil{}, err
	}
	p.PupilID = p.ID
	p.LegalFirstName = p.FirstName
	p.LegalLastName = p.LastName
	p.EALFlag = ealInt == 1
	p.PupilPremiumFlag = ppInt == 1
	p.EarlyYearsPPFlag = eyppInt == 1
	p.ServiceChildFlag = strings.EqualFold(strings.TrimSpace(svcStr), "Y") || strings.TrimSpace(svcStr) == "1"
	p.LookedAfterFlag = lacInt == 1
	p.PartTimeFlag = ptInt == 1
	p.EYDAFIndicator = eydafInt == 1
	if effTo.Valid {
		val := effTo.String
		p.EffectiveTo = &val
	}
	return p, nil
}

type SeedPupilInput struct {
	UPN            string `json:"upn"`
	FirstName      string `json:"first_name"`
	LastName       string `json:"last_name"`
	AcademicYearID string `json:"academic_year_id"`
	EffectiveFrom  string `json:"effective_from"`
}

func (d *CoreDAL) SeedPupil(ctx context.Context, in SeedPupilInput) (*Pupil, error) {
	if strings.TrimSpace(in.UPN) == "" {
		return nil, errors.New("missing mandatory field: upn")
	}
	if strings.TrimSpace(in.FirstName) == "" {
		return nil, errors.New("missing mandatory field: first_name")
	}
	if strings.TrimSpace(in.LastName) == "" {
		return nil, errors.New("missing mandatory field: last_name")
	}
	if strings.TrimSpace(in.AcademicYearID) == "" {
		return nil, errors.New("missing mandatory field: academic_year_id")
	}
	if strings.TrimSpace(in.EffectiveFrom) == "" {
		return nil, errors.New("missing mandatory field: effective_from")
	}

	uid := strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	personID := "per_" + uid
	pupilID := "pup_" + uid
	enrolmentID := "enr_" + uid

	tx, err := d.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin SeedPupil transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO academic_years (id, name, status, sequence_number, first_day, last_day)
		 VALUES (?, ?, 'CURRENT', (SELECT COALESCE(MAX(sequence_number), 0) + 1 FROM academic_years), ?, '2099-08-31')
		 ON CONFLICT(id) DO NOTHING`,
		in.AcademicYearID, in.AcademicYearID, in.EffectiveFrom,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure academic_year %s: %w", in.AcademicYearID, err)
	}

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO persons (id, legal_first_name, legal_last_name, sex, status)
		 VALUES (?, ?, ?, 'U', 'ACTIVE')`,
		personID, in.FirstName, in.LastName,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert person for pupil: %w", err)
	}

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO pupils (id, person_id, upn)
		 VALUES (?, ?, ?)`,
		pupilID, personID, in.UPN,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert pupil: %w", err)
	}

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO pupil_year_enrolments (id, pupil_id, academic_year_id, nc_year_group, effective_from, effective_to)
		 VALUES (?, ?, ?, '7', ?, NULL)`,
		enrolmentID, pupilID, in.AcademicYearID, in.EffectiveFrom,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert pupil_year_enrolment: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit SeedPupil transaction: %w", err)
	}

	d.emitPostCommitEvent(ctx, EventCorePupilEnrolled, map[string]any{
		"pupil_id":         pupilID,
		"upn":              in.UPN,
		"academic_year_id": in.AcademicYearID,
		"effective_from":   in.EffectiveFrom,
	})

	return &Pupil{
		ID:                 pupilID,
		PupilID:            pupilID,
		PersonID:           personID,
		UPN:                in.UPN,
		FirstName:          in.FirstName,
		LastName:           in.LastName,
		LegalFirstName:     in.FirstName,
		LegalLastName:      in.LastName,
		PreferredFirstName: in.FirstName,
		PreferredLastName:  in.LastName,
		Sex:                "U",
		Status:             "ACTIVE",
		NCYearGroup:        "7",
		EnrolmentStatus:    "C",
		AcademicYearID:     in.AcademicYearID,
		EffectiveFrom:      in.EffectiveFrom,
	}, nil
}

func (d *CoreDAL) DeletePupil(ctx context.Context, pupilID string) error {
	if strings.TrimSpace(pupilID) == "" {
		return errors.New("missing mandatory parameter: pupil_id")
	}

	tx, err := d.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin DeletePupil transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var personID string
	err = tx.QueryRowContext(ctx, `SELECT person_id FROM pupils WHERE id = ?`, pupilID).Scan(&personID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to look up person_id for pupil %s: %w", pupilID, err)
	}

	childTables := []string{
		"pupil_fsm_periods",
		"pupil_suspensions_exclusions",
		"pupil_alternative_provisions",
		"pupil_post16_learning_aims",
		"pupil_pastoral_notes",
		"pupil_sen_records",
		"pupil_sen_needs",
		"pupil_sen_assistants",
		"pupil_medical_conditions",
		"guardian_relationships",
		"attendance_lesson_marks",
		"attendance_session_marks",
		"group_pupil_memberships",
		"pupil_year_enrolments",
	}
	for _, tbl := range childTables {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(`DELETE FROM %s WHERE pupil_id = ?`, tbl), pupilID); err != nil {
			return fmt.Errorf("failed to delete %s: %w", tbl, err)
		}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM pupils WHERE id = ?`, pupilID); err != nil {
		return fmt.Errorf("failed to delete pupil: %w", err)
	}
	if personID != "" {
		_, _ = tx.ExecContext(ctx, `DELETE FROM household_memberships WHERE person_id = ?`, personID)
		_, _ = tx.ExecContext(ctx, `DELETE FROM person_roles WHERE person_id = ?`, personID)
		if _, err := tx.ExecContext(ctx, `DELETE FROM persons WHERE id = ?`, personID); err != nil {
			return fmt.Errorf("failed to delete person: %w", err)
		}
	}

	return tx.Commit()
}

func (d *CoreDAL) GetPupilByID(ctx context.Context, pupilID string) (*Pupil, error) {
	if strings.TrimSpace(pupilID) == "" {
		return nil, errors.New("missing mandatory parameter: pupil_id")
	}

	row := d.readDB.QueryRowContext(
		ctx,
		`SELECT `+pupilSelectColumns+`
		 FROM pupils p
		 JOIN persons per ON per.id = p.person_id
		 JOIN pupil_year_enrolments enr ON enr.pupil_id = p.id
		 WHERE p.id = ?
		 ORDER BY (CASE WHEN enr.effective_to IS NULL OR enr.effective_to >= date('now') THEN 0 ELSE 1 END) ASC,
		          enr.effective_from DESC
		 LIMIT 1`,
		pupilID,
	)

	p, err := scanPupilRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("pupil not found: %s", pupilID)
		}
		return nil, fmt.Errorf("failed to query pupil: %w", err)
	}
	return &p, nil
}

func (d *CoreDAL) GetPupilByIDScoped(ctx context.Context, scope DALQueryScope, pupilID string) (*Pupil, error) {
	if strings.TrimSpace(pupilID) == "" {
		return nil, errors.New("missing mandatory parameter: pupil_id")
	}
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	if err := d.ensurePupilInOwnGroups(ctx, scope, pupilID); err != nil {
		return nil, err
	}

	row := d.readDB.QueryRowContext(
		ctx,
		`SELECT `+pupilSelectColumns+`
		 FROM pupils p
		 JOIN persons per ON per.id = p.person_id
		 JOIN pupil_year_enrolments enr ON enr.pupil_id = p.id
		 WHERE p.id = ?
		   AND enr.academic_year_id = ?
		   AND enr.effective_from <= ?
		   AND (enr.effective_to IS NULL OR enr.effective_to >= ?)
		 ORDER BY enr.effective_from DESC
		 LIMIT 1`,
		pupilID, scope.AcademicYearID, scope.ReferenceDate, scope.ReferenceDate,
	)

	p, err := scanPupilRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("pupil not found or not enrolled on %s", scope.ReferenceDate)
		}
		return nil, fmt.Errorf("failed to query scoped pupil: %w", err)
	}
	return &p, nil
}

func (d *CoreDAL) GetPupilsByIDsBatchScoped(ctx context.Context, scope DALQueryScope, pupilIDs []string) ([]Pupil, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	if len(pupilIDs) == 0 {
		return []Pupil{}, nil
	}

	cleaned := make([]string, 0, len(pupilIDs))
	placeholders := make([]string, 0, len(pupilIDs))
	args := make([]any, 0, len(pupilIDs)+4)
	args = append(args, scope.AcademicYearID, scope.ReferenceDate, scope.ReferenceDate)

	for _, rawID := range pupilIDs {
		pid := strings.TrimSpace(rawID)
		if pid == "" {
			return nil, errors.New("pupilIDs contains an empty pupil_id")
		}
		if scope.Mode == ScopeOwnGroups {
			if err := d.ensurePupilInOwnGroups(ctx, scope, pid); err != nil {
				return nil, err
			}
		}
		cleaned = append(cleaned, pid)
		placeholders = append(placeholders, "?")
		args = append(args, pid)
	}

	query := fmt.Sprintf(`
		SELECT `+pupilSelectColumns+`
		FROM pupils p
		JOIN persons per ON per.id = p.person_id
		JOIN pupil_year_enrolments enr ON enr.pupil_id = p.id
		  AND enr.academic_year_id = ?
		  AND enr.effective_from <= ?
		  AND (enr.effective_to IS NULL OR enr.effective_to >= ?)
		WHERE p.id IN (%s)
		ORDER BY per.legal_last_name ASC, per.legal_first_name ASC, p.id ASC`,
		strings.Join(placeholders, ","),
	)

	rows, err := d.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to batch query pupils: %w", err)
	}
	defer rows.Close()

	byID := make(map[string]Pupil, len(cleaned))
	for rows.Next() {
		p, err := scanPupilRow(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan batch pupil row: %w", err)
		}
		byID[p.ID] = p
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating batch pupils: %w", err)
	}

	result := make([]Pupil, 0, len(cleaned))
	seen := make(map[string]bool, len(cleaned))
	for _, pid := range cleaned {
		if seen[pid] {
			continue
		}
		if p, ok := byID[pid]; ok {
			seen[pid] = true
			result = append(result, p)
		}
	}
	return result, nil
}

type ListPupilsOptions struct {
	IncludeLeavers bool `json:"include_leavers"`
}

func (d *CoreDAL) ListAllPupilsScoped(ctx context.Context, scope DALQueryScope) ([]Pupil, error) {
	return d.ListAllPupilsWithOptionsScoped(ctx, scope, ListPupilsOptions{})
}

func (d *CoreDAL) ListAllPupilsWithOptionsScoped(ctx context.Context, scope DALQueryScope, opts ListPupilsOptions) ([]Pupil, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}

	query := `
		SELECT ` + pupilSelectColumns + `
		FROM pupils p
		JOIN persons per ON per.id = p.person_id
		JOIN pupil_year_enrolments enr ON enr.pupil_id = p.id
		WHERE enr.academic_year_id = ?`
	args := []any{scope.AcademicYearID}
	if !opts.IncludeLeavers {
		query += ` AND enr.effective_from <= ? AND (enr.effective_to IS NULL OR enr.effective_to >= ?)`
		args = append(args, scope.ReferenceDate, scope.ReferenceDate)
	}
	query += ` ORDER BY per.legal_last_name ASC, per.legal_first_name ASC, p.id ASC`

	rows, err := d.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list all pupils: %w", err)
	}
	defer rows.Close()

	result := make([]Pupil, 0)
	for rows.Next() {
		p, err := scanPupilRow(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan active pupil row: %w", err)
		}
		result = append(result, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating active pupils: %w", err)
	}
	return result, nil
}

func (d *CoreDAL) ListPupilsByGroupScoped(ctx context.Context, scope DALQueryScope, groupID string) ([]Pupil, error) {
	if strings.TrimSpace(groupID) == "" {
		return nil, errors.New("missing mandatory parameter: group_id")
	}
	if err := d.ensureGroupInOwnGroups(ctx, scope, groupID); err != nil {
		return nil, err
	}

	rows, err := d.readDB.QueryContext(
		ctx,
		`SELECT `+pupilBaseSelectColumns+`,
		        gpm.academic_year_id, gpm.effective_from, gpm.effective_to
		 FROM group_pupil_memberships gpm
		 JOIN pupils p ON p.id = gpm.pupil_id
		 JOIN persons per ON per.id = p.person_id
		 LEFT JOIN pupil_year_enrolments enr ON enr.pupil_id = p.id
		   AND enr.academic_year_id = gpm.academic_year_id
		   AND enr.effective_from <= ?
		   AND (enr.effective_to IS NULL OR enr.effective_to >= ?)
		 WHERE gpm.group_id = ?
		   AND gpm.academic_year_id = ?
		   AND gpm.effective_from <= ?
		   AND (gpm.effective_to IS NULL OR gpm.effective_to >= ?)
		 ORDER BY per.legal_last_name ASC, per.legal_first_name ASC, p.id ASC`,
		scope.ReferenceDate, scope.ReferenceDate,
		groupID, scope.AcademicYearID, scope.ReferenceDate, scope.ReferenceDate,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list pupils for group %s: %w", groupID, err)
	}
	defer rows.Close()

	var result []Pupil
	for rows.Next() {
		p, err := scanPupilRow(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan group pupil row: %w", err)
		}
		result = append(result, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating group pupils: %w", err)
	}
	return result, nil
}
