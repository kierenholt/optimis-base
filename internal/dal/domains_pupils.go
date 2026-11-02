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

const pupilSelectColumns = `
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
	COALESCE(enr.leaving_reason_code, ''),
	enr.academic_year_id, enr.effective_from, enr.effective_to`

type rowScanner interface {
	Scan(dest ...any) error
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
		`SELECT p.id, p.person_id, p.upn, COALESCE(p.uln, ''), COALESCE(p.former_upn, ''),
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
		        COALESCE(enr.leaving_reason_code, ''),
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

type PupilFSMPeriodSummary struct {
	ID            string `json:"id"`
	PupilID       string `json:"pupil_id"`
	FSMType       string `json:"fsm_type,omitempty"`
	Ever6FSMFlag  bool   `json:"ever6_fsm_flag,omitempty"`
	EffectiveFrom string `json:"effective_from"`
	EffectiveTo   string `json:"effective_to,omitempty"`
	CountryCode   string `json:"country_code"`
	ECSReference  string `json:"ecs_reference,omitempty"`
	IsActive      bool   `json:"is_active"`
}

func (d *CoreDAL) ListPupilFSMPeriodsBatchScoped(
	ctx context.Context,
	scope DALQueryScope,
	pupilIDs []string,
) (map[string][]PupilFSMPeriodSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	if len(pupilIDs) == 0 {
		return map[string][]PupilFSMPeriodSummary{}, nil
	}

	out := make(map[string][]PupilFSMPeriodSummary, len(pupilIDs))
	placeholders := make([]string, 0, len(pupilIDs))
	args := make([]any, 0, len(pupilIDs))
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
		if _, exists := out[pid]; !exists {
			out[pid] = []PupilFSMPeriodSummary{}
			placeholders = append(placeholders, "?")
			args = append(args, pid)
		}
	}

	query := fmt.Sprintf(`
		SELECT id, pupil_id, effective_from, COALESCE(effective_to, ''), COALESCE(country_code, 'ENG'), COALESCE(ecs_reference, '')
		FROM pupil_fsm_periods
		WHERE pupil_id IN (%s)
		ORDER BY pupil_id ASC, effective_from DESC, id ASC`,
		strings.Join(placeholders, ","),
	)
	rows, err := d.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query pupil_fsm_periods batch: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var rec PupilFSMPeriodSummary
		if err := rows.Scan(&rec.ID, &rec.PupilID, &rec.EffectiveFrom, &rec.EffectiveTo, &rec.CountryCode, &rec.ECSReference); err != nil {
			return nil, fmt.Errorf("failed to scan pupil_fsm_periods row: %w", err)
		}
		rec.FSMType = "STATUTORY_FSM"
		rec.Ever6FSMFlag = true
		rec.IsActive = rec.EffectiveFrom <= scope.ReferenceDate && (rec.EffectiveTo == "" || rec.EffectiveTo >= scope.ReferenceDate)
		out[rec.PupilID] = append(out[rec.PupilID], rec)
	}
	return out, rows.Err()
}

type UpsertPupilFSMPeriodInput struct {
	ID            string `json:"id,omitempty"`
	PupilID       string `json:"pupil_id"`
	FSMType       string `json:"fsm_type,omitempty"`
	Ever6FSMFlag  bool   `json:"ever6_fsm_flag,omitempty"`
	EffectiveFrom string `json:"effective_from"`
	EffectiveTo   string `json:"effective_to,omitempty"`
	CountryCode   string `json:"country_code,omitempty"`
	ECSReference  string `json:"ecs_reference,omitempty"`
}

func (d *CoreDAL) UpsertPupilFSMPeriodScoped(
	ctx context.Context,
	scope DALQueryScope,
	in UpsertPupilFSMPeriodInput,
) (*PupilFSMPeriodSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	if scope.Mode != ScopeAll {
		return nil, errors.New("access denied: upserting FSM periods requires dal.pupils.write:all scope")
	}
	pupilID := strings.TrimSpace(in.PupilID)
	effFrom := strings.TrimSpace(in.EffectiveFrom)
	if effFrom == "" {
		effFrom = scope.ReferenceDate
	}
	if pupilID == "" || effFrom == "" {
		return nil, errors.New("missing mandatory parameter: pupil_id and effective_from are required")
	}
	fsmID := strings.TrimSpace(in.ID)
	if fsmID == "" {
		fsmID = "fsm_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	}
	country := strings.ToUpper(strings.TrimSpace(in.CountryCode))
	if country == "" {
		country = "ENG"
	}
	var effToArg any
	if strings.TrimSpace(in.EffectiveTo) != "" {
		effToArg = strings.TrimSpace(in.EffectiveTo)
	}

	if _, err := d.writeDB.ExecContext(
		ctx,
		`INSERT INTO pupil_fsm_periods (id, pupil_id, effective_from, effective_to, country_code, ecs_reference)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		     effective_from = excluded.effective_from,
		     effective_to = excluded.effective_to,
		     country_code = excluded.country_code,
		     ecs_reference = excluded.ecs_reference`,
		fsmID, pupilID, effFrom, effToArg, country, strings.TrimSpace(in.ECSReference),
	); err != nil {
		return nil, fmt.Errorf("failed to upsert pupil_fsm_periods row: %w", err)
	}

	effTo := strings.TrimSpace(in.EffectiveTo)
	return &PupilFSMPeriodSummary{
		ID:            fsmID,
		PupilID:       pupilID,
		FSMType:       strings.TrimSpace(in.FSMType),
		Ever6FSMFlag:  in.Ever6FSMFlag,
		EffectiveFrom: effFrom,
		EffectiveTo:   effTo,
		CountryCode:   country,
		ECSReference:  strings.TrimSpace(in.ECSReference),
		IsActive:      effFrom <= scope.ReferenceDate && (effTo == "" || effTo >= scope.ReferenceDate),
	}, nil
}

type UpsertPupilAdmissionInput struct {
	PupilID               string  `json:"pupil_id,omitempty"`
	PersonID              string  `json:"person_id,omitempty"`
	UPN                   string  `json:"upn"`
	ULN                   string  `json:"uln,omitempty"`
	FormerUPN             string  `json:"former_upn,omitempty"`
	NHSNumber             string  `json:"nhs_number,omitempty"`
	AdmissionNumber       string  `json:"admission_number,omitempty"`
	FirstName             string  `json:"first_name"`
	LegalMiddleNames      string  `json:"legal_middle_names,omitempty"`
	LastName              string  `json:"last_name"`
	PreferredFirstName    string  `json:"preferred_first_name,omitempty"`
	PreferredLastName     string  `json:"preferred_last_name,omitempty"`
	FormerLastName        string  `json:"former_last_name,omitempty"`
	Sex                   string  `json:"sex,omitempty"`
	GenderIdentity        string  `json:"gender_identity,omitempty"`
	DOB                   string  `json:"dob,omitempty"`
	Email                 string  `json:"email,omitempty"`
	PhonePrimary          string  `json:"phone_primary,omitempty"`
	PhotoPath             string  `json:"photo_path,omitempty"`
	EthnicityCode         string  `json:"ethnicity_code,omitempty"`
	FirstLanguageCode     string  `json:"first_language_code,omitempty"`
	EALFlag               *bool   `json:"eal_flag,omitempty"`
	CountryOfBirth        string  `json:"country_of_birth,omitempty"`
	ReligionCode          string  `json:"religion_code,omitempty"`
	PupilPremiumFlag      *bool   `json:"pupil_premium_flag,omitempty"`
	EarlyYearsPPFlag      *bool   `json:"early_years_pp_flag,omitempty"`
	ServiceChildFlag      *bool   `json:"service_child_flag,omitempty"`
	LookedAfterFlag       *bool   `json:"looked_after_flag,omitempty"`
	PostLookedAfterCode   string  `json:"post_looked_after_code,omitempty"`
	YoungCarerCode        string  `json:"young_carer_code,omitempty"`
	NCYearGroup           string  `json:"nc_year_group"`
	EnrolmentStatus       string  `json:"enrolment_status,omitempty"`
	AcademicYearID        string  `json:"academic_year_id"`
	RegistrationGroupID   string  `json:"registration_group_id,omitempty"`
	HouseID               string  `json:"house_id,omitempty"`
	BoarderStatus         string  `json:"boarder_status,omitempty"`
	PartTimeFlag          *bool   `json:"part_time_flag,omitempty"`
	EYFundedHours         float64 `json:"ey_funded_hours,omitempty"`
	EYExtendedFundedHours float64 `json:"ey_extended_funded_hours,omitempty"`
	EYThirtyHourCode      string  `json:"ey_thirty_hour_code,omitempty"`
	EYDAFIndicator        *bool   `json:"ey_daf_indicator,omitempty"`
	PreviousSchoolURN     string  `json:"previous_school_urn,omitempty"`
	EffectiveFrom         string  `json:"effective_from"`
	FSMEligible           bool    `json:"fsm_eligible,omitempty"`
	FSMEffectiveFrom      string  `json:"fsm_effective_from,omitempty"`
}

func boolPtrToInt(p *bool, def int) int {
	if p == nil {
		return def
	}
	if *p {
		return 1
	}
	return 0
}

func (d *CoreDAL) UpsertPupilAdmissionScoped(
	ctx context.Context,
	scope DALQueryScope,
	in UpsertPupilAdmissionInput,
) (*Pupil, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	if scope.Mode != ScopeAll {
		return nil, errors.New("access denied: pupil admission requires dal.pupils.write:all scope")
	}

	upn := strings.TrimSpace(in.UPN)
	firstName := strings.TrimSpace(in.FirstName)
	lastName := strings.TrimSpace(in.LastName)
	ncYear := strings.TrimSpace(in.NCYearGroup)
	if ncYear == "" {
		ncYear = "7"
	}
	ayID := strings.TrimSpace(in.AcademicYearID)
	if ayID == "" {
		ayID = scope.AcademicYearID
	}
	effFrom := strings.TrimSpace(in.EffectiveFrom)
	if effFrom == "" {
		effFrom = scope.ReferenceDate
	}
	if upn == "" {
		return nil, errors.New("missing mandatory parameter: upn")
	}
	if firstName == "" {
		return nil, errors.New("missing mandatory parameter: first_name")
	}
	if lastName == "" {
		return nil, errors.New("missing mandatory parameter: last_name")
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
	eth := strings.TrimSpace(in.EthnicityCode)
	if eth == "" {
		eth = "NOBT"
	}
	lang := strings.TrimSpace(in.FirstLanguageCode)
	if lang == "" {
		lang = "ENG"
	}
	countryBirth := strings.TrimSpace(in.CountryOfBirth)
	if countryBirth == "" {
		countryBirth = "GBR"
	}
	plac := strings.ToUpper(strings.TrimSpace(in.PostLookedAfterCode))
	if plac == "" {
		plac = "N"
	}
	yCarer := strings.ToUpper(strings.TrimSpace(in.YoungCarerCode))
	if yCarer == "" {
		yCarer = "N"
	}
	enrStatus := strings.ToUpper(strings.TrimSpace(in.EnrolmentStatus))
	if enrStatus == "" {
		enrStatus = "C"
	}
	boarder := strings.ToUpper(strings.TrimSpace(in.BoarderStatus))
	if boarder == "" {
		boarder = "N"
	}

	uid := strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	pupilID := strings.TrimSpace(in.PupilID)
	if pupilID == "" {
		pupilID = "pup_" + uid
	}
	personID := strings.TrimSpace(in.PersonID)
	if personID == "" {
		personID = "per_" + uid
	}

	tx, err := d.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin UpsertPupilAdmissionScoped tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var existingPupilID, existingPersonID string
	err = tx.QueryRowContext(ctx, `SELECT id, person_id FROM pupils WHERE upn = ? OR id = ? LIMIT 1`, upn, pupilID).Scan(&existingPupilID, &existingPersonID)
	if err == nil && existingPupilID != "" {
		pupilID = existingPupilID
		personID = existingPersonID
	}

	optStr := func(s string) any {
		if strings.TrimSpace(s) == "" {
			return nil
		}
		return strings.TrimSpace(s)
	}

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO persons (
			id, email, legal_first_name, legal_middle_names, legal_last_name,
			preferred_first_name, preferred_last_name, former_last_name,
			sex, gender_identity, dob, phone_primary, photo_path, status
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'ACTIVE')
		 ON CONFLICT(id) DO UPDATE SET
		     email = COALESCE(excluded.email, persons.email),
		     legal_first_name = excluded.legal_first_name,
		     legal_middle_names = CASE WHEN excluded.legal_middle_names != '' THEN excluded.legal_middle_names ELSE persons.legal_middle_names END,
		     legal_last_name = excluded.legal_last_name,
		     preferred_first_name = excluded.preferred_first_name,
		     preferred_last_name = excluded.preferred_last_name,
		     former_last_name = CASE WHEN excluded.former_last_name != '' THEN excluded.former_last_name ELSE persons.former_last_name END,
		     sex = excluded.sex,
		     gender_identity = CASE WHEN excluded.gender_identity != '' THEN excluded.gender_identity ELSE persons.gender_identity END,
		     dob = COALESCE(excluded.dob, persons.dob),
		     phone_primary = CASE WHEN excluded.phone_primary != '' THEN excluded.phone_primary ELSE persons.phone_primary END,
		     photo_path = COALESCE(excluded.photo_path, persons.photo_path),
		     status = 'ACTIVE'`,
		personID, optStr(in.Email), firstName, strings.TrimSpace(in.LegalMiddleNames), lastName,
		prefFirst, prefLast, strings.TrimSpace(in.FormerLastName),
		sex, strings.TrimSpace(in.GenderIdentity), optStr(in.DOB), strings.TrimSpace(in.PhonePrimary), optStr(in.PhotoPath),
	); err != nil {
		return nil, fmt.Errorf("failed to upsert person for pupil admission: %w", err)
	}

	ealDefault := 0
	if lang != "ENG" && lang != "ENGB" {
		ealDefault = 1
	}

	svcFlag := "N"
	if in.ServiceChildFlag != nil && *in.ServiceChildFlag {
		svcFlag = "Y"
	}

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO pupils (
			id, person_id, upn, former_upn, uln, nhs_number, admission_number,
			ethnicity_code, first_language_code, eal_flag, country_of_birth, religion_code,
			pupil_premium_flag, early_years_pp_flag, service_child_flag, looked_after_flag,
			post_looked_after_code, young_carer_code
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		     upn = excluded.upn,
		     former_upn = COALESCE(excluded.former_upn, pupils.former_upn),
		     uln = COALESCE(excluded.uln, pupils.uln),
		     nhs_number = COALESCE(excluded.nhs_number, pupils.nhs_number),
		     admission_number = COALESCE(excluded.admission_number, pupils.admission_number),
		     ethnicity_code = excluded.ethnicity_code,
		     first_language_code = excluded.first_language_code,
		     eal_flag = excluded.eal_flag,
		     country_of_birth = excluded.country_of_birth,
		     religion_code = CASE WHEN excluded.religion_code != '' THEN excluded.religion_code ELSE pupils.religion_code END,
		     pupil_premium_flag = excluded.pupil_premium_flag,
		     early_years_pp_flag = excluded.early_years_pp_flag,
		     service_child_flag = excluded.service_child_flag,
		     looked_after_flag = excluded.looked_after_flag,
		     post_looked_after_code = excluded.post_looked_after_code,
		     young_carer_code = excluded.young_carer_code`,
		pupilID, personID, upn, optStr(in.FormerUPN), optStr(in.ULN), optStr(in.NHSNumber), optStr(in.AdmissionNumber),
		eth, lang, boolPtrToInt(in.EALFlag, ealDefault), countryBirth, strings.TrimSpace(in.ReligionCode),
		boolPtrToInt(in.PupilPremiumFlag, 0), boolPtrToInt(in.EarlyYearsPPFlag, 0),
		svcFlag, boolPtrToInt(in.LookedAfterFlag, 0),
		plac, yCarer,
	); err != nil {
		return nil, fmt.Errorf("failed to upsert pupil row: %w", err)
	}

	enrID := "enr_" + uid
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO pupil_year_enrolments (
			id, pupil_id, academic_year_id, nc_year_group, enrolment_status,
			registration_group_id, house_id, boarder_status, part_time_flag,
			ey_funded_hours, ey_extended_funded_hours, ey_thirty_hour_code, ey_daf_indicator,
			previous_school_urn, effective_from, effective_to
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL)
		ON CONFLICT(pupil_id, academic_year_id, effective_from) DO UPDATE SET
			nc_year_group = excluded.nc_year_group,
			enrolment_status = excluded.enrolment_status,
			registration_group_id = COALESCE(excluded.registration_group_id, pupil_year_enrolments.registration_group_id),
			house_id = COALESCE(excluded.house_id, pupil_year_enrolments.house_id),
			boarder_status = excluded.boarder_status,
			part_time_flag = excluded.part_time_flag,
			ey_funded_hours = excluded.ey_funded_hours,
			ey_extended_funded_hours = excluded.ey_extended_funded_hours,
			ey_thirty_hour_code = COALESCE(excluded.ey_thirty_hour_code, pupil_year_enrolments.ey_thirty_hour_code),
			ey_daf_indicator = excluded.ey_daf_indicator,
			previous_school_urn = COALESCE(excluded.previous_school_urn, pupil_year_enrolments.previous_school_urn),
			effective_to = NULL`,
		enrID, pupilID, ayID, ncYear, enrStatus,
		optStr(in.RegistrationGroupID), optStr(in.HouseID), boarder, boolPtrToInt(in.PartTimeFlag, 0),
		in.EYFundedHours, in.EYExtendedFundedHours, optStr(in.EYThirtyHourCode), boolPtrToInt(in.EYDAFIndicator, 0),
		optStr(in.PreviousSchoolURN), effFrom,
	); err != nil {
		return nil, fmt.Errorf("failed to upsert pupil_year_enrolments row: %w", err)
	}

	if in.FSMEligible {
		fsmFrom := strings.TrimSpace(in.FSMEffectiveFrom)
		if fsmFrom == "" {
			fsmFrom = effFrom
		}
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO pupil_fsm_periods (id, pupil_id, effective_from, effective_to, country_code)
			 VALUES (?, ?, ?, NULL, 'ENG')
			 ON CONFLICT(id) DO NOTHING`,
			"fsm_"+uid, pupilID, fsmFrom,
		); err != nil {
			return nil, fmt.Errorf("failed to insert pupil_fsm_periods row: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit UpsertPupilAdmissionScoped tx: %w", err)
	}

	d.emitPostCommitEvent(ctx, EventCorePupilEnrolled, map[string]any{
		"pupil_id":         pupilID,
		"upn":              upn,
		"academic_year_id": ayID,
		"effective_from":   effFrom,
	})

	return d.GetPupilByID(ctx, pupilID)
}

type UpsertPupilYearEnrolmentInput struct {
	EnrolmentID           string  `json:"enrolment_id,omitempty"`
	PupilID               string  `json:"pupil_id"`
	AcademicYearID        string  `json:"academic_year_id"`
	NCYearGroup           string  `json:"nc_year_group"`
	EnrolmentStatus       string  `json:"enrolment_status,omitempty"`
	RegistrationGroupID   string  `json:"registration_group_id,omitempty"`
	HouseID               string  `json:"house_id,omitempty"`
	BoarderStatus         string  `json:"boarder_status,omitempty"`
	PartTimeFlag          *bool   `json:"part_time_flag,omitempty"`
	EYFundedHours         float64 `json:"ey_funded_hours,omitempty"`
	EYExtendedFundedHours float64 `json:"ey_extended_funded_hours,omitempty"`
	EYThirtyHourCode      string  `json:"ey_thirty_hour_code,omitempty"`
	EYDAFIndicator        *bool   `json:"ey_daf_indicator,omitempty"`
	PreviousSchoolURN     string  `json:"previous_school_urn,omitempty"`
	DestinationSchoolURN  string  `json:"destination_school_urn,omitempty"`
	LeavingReasonCode     string  `json:"leaving_reason_code,omitempty"`
	EffectiveFrom         string  `json:"effective_from"`
	EffectiveTo           string  `json:"effective_to,omitempty"`
}

func (d *CoreDAL) UpsertPupilYearEnrolmentScoped(
	ctx context.Context,
	scope DALQueryScope,
	in UpsertPupilYearEnrolmentInput,
) (*Pupil, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	if scope.Mode != ScopeAll {
		return nil, errors.New("access denied: year enrolment update requires dal.pupils.write:all scope")
	}

	pupilID := strings.TrimSpace(in.PupilID)
	ncYear := strings.TrimSpace(in.NCYearGroup)
	ayID := strings.TrimSpace(in.AcademicYearID)
	if ayID == "" {
		ayID = scope.AcademicYearID
	}
	effFrom := strings.TrimSpace(in.EffectiveFrom)
	if effFrom == "" {
		effFrom = scope.ReferenceDate
	}
	if pupilID == "" {
		return nil, errors.New("missing mandatory parameter: pupil_id")
	}
	if ncYear == "" {
		return nil, errors.New("missing mandatory parameter: nc_year_group")
	}

	enrStatus := strings.ToUpper(strings.TrimSpace(in.EnrolmentStatus))
	if enrStatus == "" {
		enrStatus = "C"
	}
	boarder := strings.ToUpper(strings.TrimSpace(in.BoarderStatus))
	if boarder == "" {
		boarder = "N"
	}

	enrID := strings.TrimSpace(in.EnrolmentID)
	if enrID == "" {
		enrID = "enr_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	}

	optStr := func(s string) any {
		if strings.TrimSpace(s) == "" {
			return nil
		}
		return strings.TrimSpace(s)
	}

	if _, err := d.writeDB.ExecContext(
		ctx,
		`INSERT INTO pupil_year_enrolments (
			id, pupil_id, academic_year_id, nc_year_group, enrolment_status,
			registration_group_id, house_id, boarder_status, part_time_flag,
			ey_funded_hours, ey_extended_funded_hours, ey_thirty_hour_code, ey_daf_indicator,
			previous_school_urn, destination_school_urn, leaving_reason_code,
			effective_from, effective_to
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(pupil_id, academic_year_id, effective_from) DO UPDATE SET
			nc_year_group = excluded.nc_year_group,
			enrolment_status = excluded.enrolment_status,
			registration_group_id = COALESCE(excluded.registration_group_id, pupil_year_enrolments.registration_group_id),
			house_id = COALESCE(excluded.house_id, pupil_year_enrolments.house_id),
			boarder_status = excluded.boarder_status,
			part_time_flag = excluded.part_time_flag,
			ey_funded_hours = excluded.ey_funded_hours,
			ey_extended_funded_hours = excluded.ey_extended_funded_hours,
			ey_thirty_hour_code = COALESCE(excluded.ey_thirty_hour_code, pupil_year_enrolments.ey_thirty_hour_code),
			ey_daf_indicator = excluded.ey_daf_indicator,
			previous_school_urn = COALESCE(excluded.previous_school_urn, pupil_year_enrolments.previous_school_urn),
			destination_school_urn = COALESCE(excluded.destination_school_urn, pupil_year_enrolments.destination_school_urn),
			leaving_reason_code = COALESCE(excluded.leaving_reason_code, pupil_year_enrolments.leaving_reason_code),
			effective_to = excluded.effective_to`,
		enrID, pupilID, ayID, ncYear, enrStatus,
		optStr(in.RegistrationGroupID), optStr(in.HouseID), boarder, boolPtrToInt(in.PartTimeFlag, 0),
		in.EYFundedHours, in.EYExtendedFundedHours, optStr(in.EYThirtyHourCode), boolPtrToInt(in.EYDAFIndicator, 0),
		optStr(in.PreviousSchoolURN), optStr(in.DestinationSchoolURN), optStr(in.LeavingReasonCode),
		effFrom, optStr(in.EffectiveTo),
	); err != nil {
		return nil, fmt.Errorf("failed to upsert pupil_year_enrolments row: %w", err)
	}

	return d.GetPupilByID(ctx, pupilID)
}

type PupilExclusionSummary struct {
	ID                      string `json:"id"`
	ExclusionID             string `json:"exclusion_id"`
	PupilID                 string `json:"pupil_id"`
	AcademicYearID          string `json:"academic_year_id"`
	Category                string `json:"category"`
	ExclusionCategory       string `json:"exclusion_category"`
	DfEReasonCode           string `json:"dfe_reason_code"`
	ReasonCode1             string `json:"reason_code_1"`
	ReasonCode2             string `json:"reason_code_2,omitempty"`
	ReasonCode3             string `json:"reason_code_3,omitempty"`
	StartDate               string `json:"start_date"`
	EndDate                 string `json:"end_date,omitempty"`
	SessionsLost            int    `json:"sessions_lost"`
	SessionsMissed          int    `json:"sessions_missed"`
	SENProvisionAtExclusion string `json:"sen_provision_at_exclusion"`
	ReviewDate              string `json:"review_date,omitempty"`
	ReviewOutcome           string `json:"review_outcome,omitempty"`
	AuthorisingStaffID      string `json:"authorising_staff_id,omitempty"`
	Notes                   string `json:"notes,omitempty"`
}

func (d *CoreDAL) ListPupilExclusionsScoped(
	ctx context.Context,
	scope DALQueryScope,
	pupilID string,
	academicYearID string,
) ([]PupilExclusionSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	trimmedPupil := strings.TrimSpace(pupilID)
	if trimmedPupil != "" && scope.Mode == ScopeOwnGroups {
		if err := d.ensurePupilInOwnGroups(ctx, scope, trimmedPupil); err != nil {
			return nil, err
		}
	}
	trimmedAY := strings.TrimSpace(academicYearID)

	query := `
		SELECT e.id, e.pupil_id, e.academic_year_id, e.exclusion_category, e.reason_code_1,
		       COALESCE(e.reason_code_2, ''), COALESCE(e.reason_code_3, ''),
		       e.start_date, COALESCE(e.end_date, ''), e.sessions_missed,
		       COALESCE(e.sen_provision_at_exclusion, 'N'),
		       COALESCE(e.governor_review_date, ''), COALESCE(e.governor_review_outcome, ''),
		       COALESCE(e.notes, '')
		FROM pupil_suspensions_exclusions e WHERE 1=1`
	var args []any
	if trimmedPupil != "" {
		query += ` AND e.pupil_id = ?`
		args = append(args, trimmedPupil)
	}
	if trimmedAY != "" {
		query += ` AND e.academic_year_id = ?`
		args = append(args, trimmedAY)
	}
	if scope.Mode == ScopeOwnGroups && trimmedPupil == "" {
		query += ` AND EXISTS (
			SELECT 1 FROM group_pupil_memberships gpm
			JOIN group_staff_assignments gsa ON gsa.group_id = gpm.group_id
			WHERE gpm.pupil_id = e.pupil_id
			  AND gsa.staff_id = ?
			  AND gpm.academic_year_id = ?
			  AND gpm.effective_from <= ?
			  AND (gpm.effective_to IS NULL OR gpm.effective_to >= ?)
		)`
		args = append(args, scope.StaffID, scope.AcademicYearID, scope.ReferenceDate, scope.ReferenceDate)
	}
	query += ` ORDER BY e.start_date DESC, e.id ASC`

	rows, err := d.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list pupil_suspensions_exclusions: %w", err)
	}
	defer rows.Close()

	result := make([]PupilExclusionSummary, 0)
	for rows.Next() {
		var r PupilExclusionSummary
		if err := rows.Scan(
			&r.ID, &r.PupilID, &r.AcademicYearID, &r.Category, &r.DfEReasonCode,
			&r.ReasonCode2, &r.ReasonCode3, &r.StartDate, &r.EndDate, &r.SessionsLost,
			&r.SENProvisionAtExclusion, &r.ReviewDate, &r.ReviewOutcome, &r.Notes,
		); err != nil {
			return nil, fmt.Errorf("failed to scan pupil_suspensions_exclusions row: %w", err)
		}
		r.ExclusionID = r.ID
		r.ExclusionCategory = r.Category
		r.ReasonCode1 = r.DfEReasonCode
		r.SessionsMissed = r.SessionsLost
		r.AuthorisingStaffID = scope.StaffID
		result = append(result, r)
	}
	return result, rows.Err()
}

func (d *CoreDAL) UpsertPupilExclusionScoped(
	ctx context.Context,
	scope DALQueryScope,
	in PupilExclusionSummary,
) (*PupilExclusionSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	if scope.Mode != ScopeAll {
		return nil, errors.New("access denied: upserting pupil exclusions requires dal.pupils.write:all scope")
	}

	pupilID := strings.TrimSpace(in.PupilID)
	ayID := strings.TrimSpace(in.AcademicYearID)
	if ayID == "" {
		ayID = scope.AcademicYearID
	}
	category := strings.ToUpper(strings.TrimSpace(in.Category))
	if category == "" {
		category = strings.ToUpper(strings.TrimSpace(in.ExclusionCategory))
	}
	if category == "" {
		category = "SUSPENSION"
	}
	reason1 := strings.ToUpper(strings.TrimSpace(in.DfEReasonCode))
	if reason1 == "" {
		reason1 = strings.ToUpper(strings.TrimSpace(in.ReasonCode1))
	}
	startDate := strings.TrimSpace(in.StartDate)
	if pupilID == "" || reason1 == "" || startDate == "" {
		return nil, errors.New("missing mandatory parameter: pupil_id, dfe_reason_code, and start_date are required")
	}
	exclID := strings.TrimSpace(in.ExclusionID)
	if exclID == "" {
		exclID = strings.TrimSpace(in.ID)
	}
	if exclID == "" {
		exclID = "exc_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	}
	sessionsLost := in.SessionsLost
	if sessionsLost <= 0 {
		sessionsLost = in.SessionsMissed
	}
	if sessionsLost <= 0 {
		sessionsLost = 1
	}
	senProv := strings.ToUpper(strings.TrimSpace(in.SENProvisionAtExclusion))
	if senProv != "K" && senProv != "E" {
		senProv = "N"
	}

	optStr := func(s string) any {
		if strings.TrimSpace(s) == "" {
			return nil
		}
		return strings.TrimSpace(s)
	}

	if _, err := d.writeDB.ExecContext(
		ctx,
		`INSERT INTO pupil_suspensions_exclusions (
			id, pupil_id, academic_year_id, exclusion_category, reason_code_1,
			reason_code_2, reason_code_3, start_date, end_date, sessions_missed,
			sen_provision_at_exclusion, governor_review_date, governor_review_outcome, notes
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			exclusion_category = excluded.exclusion_category,
			reason_code_1 = excluded.reason_code_1,
			reason_code_2 = excluded.reason_code_2,
			reason_code_3 = excluded.reason_code_3,
			start_date = excluded.start_date,
			end_date = excluded.end_date,
			sessions_missed = excluded.sessions_missed,
			sen_provision_at_exclusion = excluded.sen_provision_at_exclusion,
			governor_review_date = excluded.governor_review_date,
			governor_review_outcome = excluded.governor_review_outcome,
			notes = excluded.notes`,
		exclID, pupilID, ayID, category, reason1,
		optStr(in.ReasonCode2), optStr(in.ReasonCode3), startDate, optStr(in.EndDate), sessionsLost,
		senProv, optStr(in.ReviewDate), optStr(in.ReviewOutcome), strings.TrimSpace(in.Notes),
	); err != nil {
		return nil, fmt.Errorf("failed to upsert pupil_suspensions_exclusions row: %w", err)
	}

	authStaff := strings.TrimSpace(in.AuthorisingStaffID)
	if authStaff == "" {
		authStaff = scope.StaffID
	}
	out := in
	out.ID = exclID
	out.ExclusionID = exclID
	out.PupilID = pupilID
	out.AcademicYearID = ayID
	out.Category = category
	out.ExclusionCategory = category
	out.DfEReasonCode = reason1
	out.ReasonCode1 = reason1
	out.StartDate = startDate
	out.SessionsLost = sessionsLost
	out.SessionsMissed = sessionsLost
	out.SENProvisionAtExclusion = senProv
	out.AuthorisingStaffID = authStaff
	return &out, nil
}

type PupilAlternativeProvisionSummary struct {
	ID                     string `json:"id"`
	ProvisionID            string `json:"provision_id"`
	PupilID                string `json:"pupil_id"`
	AcademicYearID         string `json:"academic_year_id"`
	ProviderURN            string `json:"provider_urn,omitempty"`
	ProviderUKPRN          string `json:"provider_ukprn,omitempty"`
	CompaniesHouseNumber   string `json:"companies_house_number,omitempty"`
	APSettingTypeCode      string `json:"ap_setting_type_code,omitempty"`
	ReasonCode             string `json:"reason_code"`
	SENProvision           string `json:"sen_provision,omitempty"`
	SessionsPerWeek        int    `json:"sessions_per_week"`
	EffectiveFrom          string `json:"effective_from"`
	EffectiveTo            string `json:"effective_to,omitempty"`
}

func (d *CoreDAL) ListPupilAlternativeProvisionsScoped(
	ctx context.Context,
	scope DALQueryScope,
	pupilID string,
	academicYearID string,
) ([]PupilAlternativeProvisionSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	trimmedPupil := strings.TrimSpace(pupilID)
	if trimmedPupil != "" && scope.Mode == ScopeOwnGroups {
		if err := d.ensurePupilInOwnGroups(ctx, scope, trimmedPupil); err != nil {
			return nil, err
		}
	}
	trimmedAY := strings.TrimSpace(academicYearID)

	query := `
		SELECT id, pupil_id, academic_year_id,
		       COALESCE(provider_urn, ''), COALESCE(provider_ukprn, ''),
		       COALESCE(companies_house_number, ''), COALESCE(ap_setting_type_code, ''),
		       ap_reason_code, COALESCE(sen_provision_during_ap, 'N'),
		       sessions_per_week, effective_from, COALESCE(effective_to, '')
		FROM pupil_alternative_provisions WHERE 1=1`
	var args []any
	if trimmedPupil != "" {
		query += ` AND pupil_id = ?`
		args = append(args, trimmedPupil)
	}
	if trimmedAY != "" {
		query += ` AND academic_year_id = ?`
		args = append(args, trimmedAY)
	}
	query += ` ORDER BY effective_from DESC, id ASC`

	rows, err := d.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list pupil_alternative_provisions: %w", err)
	}
	defer rows.Close()

	result := make([]PupilAlternativeProvisionSummary, 0)
	for rows.Next() {
		var r PupilAlternativeProvisionSummary
		if err := rows.Scan(
			&r.ID, &r.PupilID, &r.AcademicYearID, &r.ProviderURN, &r.ProviderUKPRN,
			&r.CompaniesHouseNumber, &r.APSettingTypeCode,
			&r.ReasonCode, &r.SENProvision, &r.SessionsPerWeek, &r.EffectiveFrom, &r.EffectiveTo,
		); err != nil {
			return nil, fmt.Errorf("failed to scan pupil_alternative_provisions row: %w", err)
		}
		r.ProvisionID = r.ID
		result = append(result, r)
	}
	return result, rows.Err()
}

func (d *CoreDAL) UpsertPupilAlternativeProvisionScoped(
	ctx context.Context,
	scope DALQueryScope,
	in PupilAlternativeProvisionSummary,
) (*PupilAlternativeProvisionSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	if scope.Mode != ScopeAll {
		return nil, errors.New("access denied: upserting alternative provisions requires dal.pupils.write:all scope")
	}

	pupilID := strings.TrimSpace(in.PupilID)
	ayID := strings.TrimSpace(in.AcademicYearID)
	if ayID == "" {
		ayID = scope.AcademicYearID
	}
	reasonCode := strings.ToUpper(strings.TrimSpace(in.ReasonCode))
	effFrom := strings.TrimSpace(in.EffectiveFrom)
	if effFrom == "" {
		effFrom = scope.ReferenceDate
	}
	if pupilID == "" || reasonCode == "" || effFrom == "" {
		return nil, errors.New("missing mandatory parameter: pupil_id, reason_code, and effective_from are required")
	}
	provID := strings.TrimSpace(in.ProvisionID)
	if provID == "" {
		provID = strings.TrimSpace(in.ID)
	}
	if provID == "" {
		provID = "ap_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	}
	sessPerWeek := in.SessionsPerWeek
	if sessPerWeek <= 0 {
		sessPerWeek = 1
	}
	senProv := strings.ToUpper(strings.TrimSpace(in.SENProvision))
	if senProv != "K" && senProv != "E" {
		senProv = "N"
	}

	optStr := func(s string) any {
		if strings.TrimSpace(s) == "" {
			return nil
		}
		return strings.TrimSpace(s)
	}

	if _, err := d.writeDB.ExecContext(
		ctx,
		`INSERT INTO pupil_alternative_provisions (
			id, pupil_id, academic_year_id, provider_urn, provider_ukprn,
			companies_house_number, ap_setting_type_code, ap_reason_code,
			sen_provision_during_ap, sessions_per_week, effective_from, effective_to
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			provider_urn = excluded.provider_urn,
			provider_ukprn = excluded.provider_ukprn,
			companies_house_number = excluded.companies_house_number,
			ap_setting_type_code = excluded.ap_setting_type_code,
			ap_reason_code = excluded.ap_reason_code,
			sen_provision_during_ap = excluded.sen_provision_during_ap,
			sessions_per_week = excluded.sessions_per_week,
			effective_from = excluded.effective_from,
			effective_to = excluded.effective_to`,
		provID, pupilID, ayID, optStr(in.ProviderURN), optStr(in.ProviderUKPRN),
		optStr(in.CompaniesHouseNumber), optStr(in.APSettingTypeCode),
		reasonCode, senProv, sessPerWeek, effFrom, optStr(in.EffectiveTo),
	); err != nil {
		return nil, fmt.Errorf("failed to upsert pupil_alternative_provisions row: %w", err)
	}

	out := in
	out.ID = provID
	out.ProvisionID = provID
	out.PupilID = pupilID
	out.AcademicYearID = ayID
	out.ReasonCode = reasonCode
	out.SENProvision = senProv
	out.SessionsPerWeek = sessPerWeek
	out.EffectiveFrom = effFrom
	return &out, nil
}

type PupilPost16LearningAimSummary struct {
	ID               string `json:"id"`
	AimID            string `json:"aim_id"`
	PupilID          string `json:"pupil_id"`
	AcademicYearID   string `json:"academic_year_id"`
	QANCode          string `json:"qan_code"`
	DiscountCode     string `json:"discount_code,omitempty"`
	SubjectTitle     string `json:"subject_title"`
	CourseID         string `json:"course_id,omitempty"`
	PlannedGLH       int    `json:"planned_glh"`
	PlannedEEPHours  int    `json:"planned_eep_hours,omitempty"`
	StartDate        string `json:"start_date"`
	PlannedEndDate   string `json:"planned_end_date"`
	ActualEndDate    string `json:"actual_end_date,omitempty"`
	CompletionStatus string `json:"completion_status"`
	Outcome          string `json:"outcome"`
}

func (d *CoreDAL) ListPupilPost16LearningAimsScoped(
	ctx context.Context,
	scope DALQueryScope,
	pupilID string,
	academicYearID string,
) ([]PupilPost16LearningAimSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	trimmedPupil := strings.TrimSpace(pupilID)
	if trimmedPupil != "" && scope.Mode == ScopeOwnGroups {
		if err := d.ensurePupilInOwnGroups(ctx, scope, trimmedPupil); err != nil {
			return nil, err
		}
	}
	trimmedAY := strings.TrimSpace(academicYearID)

	query := `
		SELECT id, pupil_id, academic_year_id, qn_code, COALESCE(subject_classification_code, ''),
		       COALESCE(course_id, ''), planned_learning_hours, planned_eep_hours,
		       effective_from, planned_end_date, COALESCE(actual_end_date, ''),
		       completion_status, COALESCE(outcome_code, '8')
		FROM pupil_post16_learning_aims WHERE 1=1`
	var args []any
	if trimmedPupil != "" {
		query += ` AND pupil_id = ?`
		args = append(args, trimmedPupil)
	}
	if trimmedAY != "" {
		query += ` AND academic_year_id = ?`
		args = append(args, trimmedAY)
	}
	query += ` ORDER BY effective_from DESC, id ASC`

	rows, err := d.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list pupil_post16_learning_aims: %w", err)
	}
	defer rows.Close()

	result := make([]PupilPost16LearningAimSummary, 0)
	for rows.Next() {
		var r PupilPost16LearningAimSummary
		var subjCode string
		if err := rows.Scan(
			&r.ID, &r.PupilID, &r.AcademicYearID, &r.QANCode, &subjCode,
			&r.CourseID, &r.PlannedGLH, &r.PlannedEEPHours,
			&r.StartDate, &r.PlannedEndDate, &r.ActualEndDate,
			&r.CompletionStatus, &r.Outcome,
		); err != nil {
			return nil, fmt.Errorf("failed to scan pupil_post16_learning_aims row: %w", err)
		}
		r.AimID = r.ID
		r.DiscountCode = subjCode
		r.SubjectTitle = subjCode
		if r.SubjectTitle == "" {
			r.SubjectTitle = r.QANCode
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

func (d *CoreDAL) UpsertPupilPost16LearningAimScoped(
	ctx context.Context,
	scope DALQueryScope,
	in PupilPost16LearningAimSummary,
) (*PupilPost16LearningAimSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	if scope.Mode != ScopeAll {
		return nil, errors.New("access denied: upserting post-16 learning aims requires dal.pupils.write:all scope")
	}

	pupilID := strings.TrimSpace(in.PupilID)
	ayID := strings.TrimSpace(in.AcademicYearID)
	if ayID == "" {
		ayID = scope.AcademicYearID
	}
	qanCode := strings.ToUpper(strings.TrimSpace(in.QANCode))
	subjTitle := strings.TrimSpace(in.SubjectTitle)
	if subjTitle == "" {
		subjTitle = strings.TrimSpace(in.DiscountCode)
	}
	startDate := strings.TrimSpace(in.StartDate)
	if startDate == "" {
		startDate = scope.ReferenceDate
	}
	plannedEnd := strings.TrimSpace(in.PlannedEndDate)
	if plannedEnd == "" {
		plannedEnd = startDate
	}
	if pupilID == "" || qanCode == "" {
		return nil, errors.New("missing mandatory parameter: pupil_id and qan_code are required")
	}
	aimID := strings.TrimSpace(in.AimID)
	if aimID == "" {
		aimID = strings.TrimSpace(in.ID)
	}
	if aimID == "" {
		aimID = "aim_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	}
	glh := in.PlannedGLH
	if glh < 0 {
		glh = 0
	}
	compStatus := strings.TrimSpace(in.CompletionStatus)
	if compStatus == "" {
		compStatus = "1"
	}
	outcome := strings.TrimSpace(in.Outcome)
	if outcome != "1" && outcome != "2" && outcome != "3" && outcome != "8" {
		outcome = "8"
	}
	subjClassCode := strings.TrimSpace(in.DiscountCode)
	if subjClassCode == "" {
		subjClassCode = subjTitle
	}
	if subjClassCode == "" {
		subjClassCode = "0000"
	}

	optStr := func(s string) any {
		if strings.TrimSpace(s) == "" {
			return nil
		}
		return strings.TrimSpace(s)
	}

	if _, err := d.writeDB.ExecContext(
		ctx,
		`INSERT INTO pupil_post16_learning_aims (
			id, pupil_id, academic_year_id, qn_code, subject_classification_code,
			course_id, planned_learning_hours, planned_eep_hours,
			completion_status, outcome_code, effective_from, planned_end_date, actual_end_date
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			qn_code = excluded.qn_code,
			subject_classification_code = excluded.subject_classification_code,
			course_id = excluded.course_id,
			planned_learning_hours = excluded.planned_learning_hours,
			planned_eep_hours = excluded.planned_eep_hours,
			completion_status = excluded.completion_status,
			outcome_code = excluded.outcome_code,
			effective_from = excluded.effective_from,
			planned_end_date = excluded.planned_end_date,
			actual_end_date = excluded.actual_end_date`,
		aimID, pupilID, ayID, qanCode, subjClassCode,
		optStr(in.CourseID), glh, in.PlannedEEPHours,
		compStatus, outcome, startDate, plannedEnd, optStr(in.ActualEndDate),
	); err != nil {
		return nil, fmt.Errorf("failed to upsert pupil_post16_learning_aims row: %w", err)
	}

	out := in
	out.ID = aimID
	out.AimID = aimID
	out.PupilID = pupilID
	out.AcademicYearID = ayID
	out.QANCode = qanCode
	out.DiscountCode = subjClassCode
	if subjTitle == "" {
		subjTitle = qanCode
	}
	out.SubjectTitle = subjTitle
	out.PlannedGLH = glh
	out.StartDate = startDate
	out.PlannedEndDate = plannedEnd
	out.CompletionStatus = compStatus
	out.Outcome = outcome
	return &out, nil
}

type PupilPastoralNoteSummary struct {
	ID                string `json:"id"`
	NoteID            string `json:"note_id"`
	PupilID           string `json:"pupil_id"`
	RecordedByStaffID string `json:"recorded_by_staff_id"`
	Category          string `json:"category"`
	AlertLevelID      int    `json:"alert_level_id"`
	Title             string `json:"title,omitempty"`
	Summary           string `json:"summary"`
	CreatedAt         string `json:"created_at,omitempty"`
}

func (d *CoreDAL) ListPupilPastoralNotesScoped(
	ctx context.Context,
	scope DALQueryScope,
	pupilID string,
) ([]PupilPastoralNoteSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	trimmedPupil := strings.TrimSpace(pupilID)
	if trimmedPupil == "" {
		return nil, errors.New("missing mandatory parameter: pupil_id")
	}
	if scope.Mode == ScopeOwnGroups {
		if err := d.ensurePupilInOwnGroups(ctx, scope, trimmedPupil); err != nil {
			return nil, err
		}
	}

	rows, err := d.readDB.QueryContext(
		ctx,
		`SELECT id, pupil_id, COALESCE(created_by_staff_id, ''), category, COALESCE(title, ''), note_text, COALESCE(created_at, '')
		 FROM pupil_pastoral_notes
		 WHERE pupil_id = ?
		 ORDER BY created_at DESC, id ASC`,
		trimmedPupil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list pupil_pastoral_notes: %w", err)
	}
	defer rows.Close()

	result := make([]PupilPastoralNoteSummary, 0)
	for rows.Next() {
		var r PupilPastoralNoteSummary
		if err := rows.Scan(&r.ID, &r.PupilID, &r.RecordedByStaffID, &r.Category, &r.Title, &r.Summary, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan pupil_pastoral_notes row: %w", err)
		}
		r.NoteID = r.ID
		r.AlertLevelID = 3
		result = append(result, r)
	}
	return result, rows.Err()
}

func (d *CoreDAL) UpsertPupilPastoralNoteScoped(
	ctx context.Context,
	scope DALQueryScope,
	in PupilPastoralNoteSummary,
) (*PupilPastoralNoteSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	pupilID := strings.TrimSpace(in.PupilID)
	summary := strings.TrimSpace(in.Summary)
	staffID := strings.TrimSpace(in.RecordedByStaffID)
	if staffID == "" {
		staffID = scope.StaffID
	}
	if pupilID == "" || summary == "" {
		return nil, errors.New("missing mandatory parameter: pupil_id and summary are required")
	}
	if scope.Mode == ScopeOwnGroups {
		if err := d.ensurePupilInOwnGroups(ctx, scope, pupilID); err != nil {
			return nil, err
		}
	}
	noteID := strings.TrimSpace(in.NoteID)
	if noteID == "" {
		noteID = strings.TrimSpace(in.ID)
	}
	if noteID == "" {
		noteID = "ppn_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	}
	category := strings.ToUpper(strings.TrimSpace(in.Category))
	if category == "" {
		category = "PASTORAL"
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = category + " Note"
	}
	alertLevelID := in.AlertLevelID
	if alertLevelID < 1 || alertLevelID > 4 {
		alertLevelID = 3
	}
	var resolvedStaffID string
	if staffID != "" {
		var exists int
		if err := d.readDB.QueryRowContext(ctx, `SELECT 1 FROM staff WHERE id = ?`, staffID).Scan(&exists); err == nil && exists == 1 {
			resolvedStaffID = staffID
		}
	}
	if resolvedStaffID == "" {
		_ = d.readDB.QueryRowContext(ctx, `SELECT id FROM staff ORDER BY id ASC LIMIT 1`).Scan(&resolvedStaffID)
	}
	if resolvedStaffID == "" {
		return nil, errors.New("missing mandatory parameter: recorded_by_staff_id must reference an existing staff row")
	}

	if _, err := d.writeDB.ExecContext(
		ctx,
		`INSERT INTO pupil_pastoral_notes (
			id, pupil_id, category, title, note_text, created_by_staff_id
		) VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			category = excluded.category,
			title = excluded.title,
			note_text = excluded.note_text,
			created_by_staff_id = excluded.created_by_staff_id`,
		noteID, pupilID, category, title, summary, resolvedStaffID,
	); err != nil {
		return nil, fmt.Errorf("failed to upsert pupil_pastoral_notes row: %w", err)
	}

	return &PupilPastoralNoteSummary{
		ID:                noteID,
		NoteID:            noteID,
		PupilID:           pupilID,
		RecordedByStaffID: staffID,
		Category:          category,
		AlertLevelID:      alertLevelID,
		Title:             title,
		Summary:           summary,
	}, nil
}
