package dal

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

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
	effToArg := optionalTrimmedString(in.EffectiveTo)

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
		personID, optionalTrimmedString(in.Email), firstName, strings.TrimSpace(in.LegalMiddleNames), lastName,
		prefFirst, prefLast, strings.TrimSpace(in.FormerLastName),
		sex, strings.TrimSpace(in.GenderIdentity), optionalTrimmedString(in.DOB), strings.TrimSpace(in.PhonePrimary), optionalTrimmedString(in.PhotoPath),
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
		pupilID, personID, upn, optionalTrimmedString(in.FormerUPN), optionalTrimmedString(in.ULN), optionalTrimmedString(in.NHSNumber), optionalTrimmedString(in.AdmissionNumber),
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
		optionalTrimmedString(in.RegistrationGroupID), optionalTrimmedString(in.HouseID), boarder, boolPtrToInt(in.PartTimeFlag, 0),
		in.EYFundedHours, in.EYExtendedFundedHours, optionalTrimmedString(in.EYThirtyHourCode), boolPtrToInt(in.EYDAFIndicator, 0),
		optionalTrimmedString(in.PreviousSchoolURN), effFrom,
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
		optionalTrimmedString(in.RegistrationGroupID), optionalTrimmedString(in.HouseID), boarder, boolPtrToInt(in.PartTimeFlag, 0),
		in.EYFundedHours, in.EYExtendedFundedHours, optionalTrimmedString(in.EYThirtyHourCode), boolPtrToInt(in.EYDAFIndicator, 0),
		optionalTrimmedString(in.PreviousSchoolURN), optionalTrimmedString(in.DestinationSchoolURN), optionalTrimmedString(in.LeavingReasonCode),
		effFrom, optionalTrimmedString(in.EffectiveTo),
	); err != nil {
		return nil, fmt.Errorf("failed to upsert pupil_year_enrolments row: %w", err)
	}

	return d.GetPupilByID(ctx, pupilID)
}
