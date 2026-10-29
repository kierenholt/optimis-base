package dal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

type PupilMedicalConditionAlert struct {
	ID                    string `json:"id"`
	ConditionName         string `json:"condition_name"`
	ConditionCategory     string `json:"condition_category"`
	AlertLevelID          string `json:"alert_level_id"`
	HasIHCP               bool   `json:"has_ihcp"`
	Triggers              string `json:"triggers"`
	ReactionSymptoms      string `json:"reaction_symptoms"`
	EmergencyResponsePlan string `json:"emergency_response_plan"`
	MedicationNameAndDose string `json:"medication_name_and_dose"`
	MedicationLocation    string `json:"medication_location"`
	ParentalConsentGiven  bool   `json:"parental_consent_given"`
	EffectiveFrom         string `json:"effective_from"`
	EffectiveTo           string `json:"effective_to"`
}

type PupilSenNeedSummary struct {
	ID            string `json:"id"`
	SenNeedCode   string `json:"sen_need_code"`
	Ranking       int    `json:"ranking"`
	AlertLevelID  string `json:"alert_level_id"`
	EffectiveFrom string `json:"effective_from"`
}

type PupilSenAssistantSummary struct {
	ID              string `json:"id"`
	StaffID         string `json:"staff_id"`
	DisplayName     string `json:"display_name"`
	AllocationNotes string `json:"allocation_notes"`
	EffectiveFrom   string `json:"effective_from"`
}

type PupilSenAndMedicalSummary struct {
	PupilID              string                       `json:"pupil_id"`
	SenProvisionCode     string                       `json:"sen_provision_code"`
	PrimarySenNeedCode   string                       `json:"primary_sen_need_code"`
	SecondarySenNeedCode string                       `json:"secondary_sen_need_code"`
	TeachingStrategies   string                       `json:"teaching_strategies"`
	PupilTargets         string                       `json:"pupil_targets"`
	SencoNotes           string                       `json:"senco_notes"`
	EHCPStartDate        string                       `json:"ehcp_start_date"`
	EHCPReviewDueDate    string                       `json:"ehcp_review_due_date"`
	SenNeeds             []PupilSenNeedSummary        `json:"sen_needs"`
	SenAssistants        []PupilSenAssistantSummary   `json:"sen_assistants"`
	MedicalConditions    []PupilMedicalConditionAlert `json:"medical_conditions"`
}

type SenRegisterFilterOptions struct {
	GroupID        string   `json:"group_id"`
	NCYearGroup    string   `json:"nc_year_group"`
	ProvisionCodes []string `json:"provision_codes"`
}

type PupilSenRegisterEntry struct {
	Pupil
	SenProvisionCode     string                     `json:"sen_provision_code"`
	PrimarySenNeedCode   string                     `json:"primary_sen_need_code"`
	SecondarySenNeedCode string                     `json:"secondary_sen_need_code"`
	TeachingStrategies   string                     `json:"teaching_strategies"`
	PupilTargets         string                     `json:"pupil_targets"`
	SencoNotes           string                     `json:"senco_notes"`
	EHCPStartDate        string                     `json:"ehcp_start_date"`
	EHCPReviewDueDate    string                     `json:"ehcp_review_due_date"`
	SenNeeds             []PupilSenNeedSummary      `json:"sen_needs"`
	SenAssistants        []PupilSenAssistantSummary `json:"sen_assistants"`
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
		`SELECT p.id, p.upn, per.legal_first_name, per.legal_last_name,
		        COALESCE(NULLIF(TRIM(per.preferred_first_name), ''), per.legal_first_name),
		        COALESCE(NULLIF(TRIM(per.preferred_last_name), ''), per.legal_last_name),
		        COALESCE(enr.nc_year_group, ''), COALESCE(enr.enrolment_status, 'C'),
		        enr.academic_year_id, enr.effective_from, enr.effective_to
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

	var p Pupil
	var effTo sql.NullString
	if err := row.Scan(
		&p.ID, &p.UPN, &p.FirstName, &p.LastName,
		&p.PreferredFirstName, &p.PreferredLastName,
		&p.NCYearGroup, &p.EnrolmentStatus,
		&p.AcademicYearID, &p.EffectiveFrom, &effTo,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("pupil not found or not enrolled on %s", scope.ReferenceDate)
		}
		return nil, fmt.Errorf("failed to query scoped pupil: %w", err)
	}
	p.PupilID = p.ID
	p.LegalFirstName = p.FirstName
	p.LegalLastName = p.LastName
	if effTo.Valid {
		val := effTo.String
		p.EffectiveTo = &val
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
		SELECT p.id, p.upn, per.legal_first_name, per.legal_last_name,
		       COALESCE(NULLIF(TRIM(per.preferred_first_name), ''), per.legal_first_name),
		       COALESCE(NULLIF(TRIM(per.preferred_last_name), ''), per.legal_last_name),
		       COALESCE(enr.nc_year_group, ''), COALESCE(enr.enrolment_status, 'C'),
		       enr.academic_year_id, enr.effective_from, enr.effective_to
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
		var p Pupil
		var effTo sql.NullString
		if err := rows.Scan(
			&p.ID, &p.UPN, &p.FirstName, &p.LastName,
			&p.PreferredFirstName, &p.PreferredLastName,
			&p.NCYearGroup, &p.EnrolmentStatus,
			&p.AcademicYearID, &p.EffectiveFrom, &effTo,
		); err != nil {
			return nil, fmt.Errorf("failed to scan batch pupil row: %w", err)
		}
		p.PupilID = p.ID
		p.LegalFirstName = p.FirstName
		p.LegalLastName = p.LastName
		if effTo.Valid {
			val := effTo.String
			p.EffectiveTo = &val
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

func (d *CoreDAL) ListAllPupilsScoped(ctx context.Context, scope DALQueryScope) ([]Pupil, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}

	rows, err := d.readDB.QueryContext(
		ctx,
		`SELECT p.id, p.upn, per.legal_first_name, per.legal_last_name,
		        COALESCE(NULLIF(TRIM(per.preferred_first_name), ''), per.legal_first_name),
		        COALESCE(NULLIF(TRIM(per.preferred_last_name), ''), per.legal_last_name),
		        COALESCE(enr.nc_year_group, ''), COALESCE(enr.enrolment_status, 'C'),
		        enr.academic_year_id, enr.effective_from, enr.effective_to
		 FROM pupils p
		 JOIN persons per ON per.id = p.person_id
		 JOIN pupil_year_enrolments enr ON enr.pupil_id = p.id
		 WHERE enr.academic_year_id = ?
		   AND enr.effective_from <= ?
		   AND (enr.effective_to IS NULL OR enr.effective_to >= ?)
		 ORDER BY per.legal_last_name ASC, per.legal_first_name ASC, p.id ASC`,
		scope.AcademicYearID, scope.ReferenceDate, scope.ReferenceDate,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list all active pupils: %w", err)
	}
	defer rows.Close()

	result := make([]Pupil, 0)
	for rows.Next() {
		var p Pupil
		var effTo sql.NullString
		if err := rows.Scan(
			&p.ID, &p.UPN, &p.FirstName, &p.LastName,
			&p.PreferredFirstName, &p.PreferredLastName,
			&p.NCYearGroup, &p.EnrolmentStatus,
			&p.AcademicYearID, &p.EffectiveFrom, &effTo,
		); err != nil {
			return nil, fmt.Errorf("failed to scan active pupil row: %w", err)
		}
		p.PupilID = p.ID
		p.LegalFirstName = p.FirstName
		p.LegalLastName = p.LastName
		if effTo.Valid {
			val := effTo.String
			p.EffectiveTo = &val
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
		`SELECT p.id, p.upn, per.legal_first_name, per.legal_last_name,
		        COALESCE(NULLIF(TRIM(per.preferred_first_name), ''), per.legal_first_name),
		        COALESCE(NULLIF(TRIM(per.preferred_last_name), ''), per.legal_last_name),
		        COALESCE((
		            SELECT enr.nc_year_group FROM pupil_year_enrolments enr
		            WHERE enr.pupil_id = p.id
		              AND enr.academic_year_id = gpm.academic_year_id
		              AND enr.effective_from <= ?
		              AND (enr.effective_to IS NULL OR enr.effective_to >= ?)
		            ORDER BY enr.effective_from DESC LIMIT 1
		        ), ''),
		        COALESCE((
		            SELECT enr.enrolment_status FROM pupil_year_enrolments enr
		            WHERE enr.pupil_id = p.id
		              AND enr.academic_year_id = gpm.academic_year_id
		              AND enr.effective_from <= ?
		              AND (enr.effective_to IS NULL OR enr.effective_to >= ?)
		            ORDER BY enr.effective_from DESC LIMIT 1
		        ), 'C'),
		        gpm.academic_year_id, gpm.effective_from, gpm.effective_to
		 FROM group_pupil_memberships gpm
		 JOIN pupils p ON p.id = gpm.pupil_id
		 JOIN persons per ON per.id = p.person_id
		 WHERE gpm.group_id = ?
		   AND gpm.academic_year_id = ?
		   AND gpm.effective_from <= ?
		   AND (gpm.effective_to IS NULL OR gpm.effective_to >= ?)
		 ORDER BY per.legal_last_name ASC, per.legal_first_name ASC, p.id ASC`,
		scope.ReferenceDate, scope.ReferenceDate,
		scope.ReferenceDate, scope.ReferenceDate,
		groupID, scope.AcademicYearID, scope.ReferenceDate, scope.ReferenceDate,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list pupils for group %s: %w", groupID, err)
	}
	defer rows.Close()

	var result []Pupil
	for rows.Next() {
		var p Pupil
		var effTo sql.NullString
		if err := rows.Scan(
			&p.ID, &p.UPN, &p.FirstName, &p.LastName,
			&p.PreferredFirstName, &p.PreferredLastName,
			&p.NCYearGroup, &p.EnrolmentStatus,
			&p.AcademicYearID, &p.EffectiveFrom, &effTo,
		); err != nil {
			return nil, fmt.Errorf("failed to scan group pupil row: %w", err)
		}
		p.PupilID = p.ID
		p.LegalFirstName = p.FirstName
		p.LegalLastName = p.LastName
		if effTo.Valid {
			val := effTo.String
			p.EffectiveTo = &val
		}
		result = append(result, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating group pupils: %w", err)
	}
	return result, nil
}

func (d *CoreDAL) ListPupilSenAndMedicalBatch(
	ctx context.Context,
	scope DALQueryScope,
	pupilIDs []string,
) (map[string]PupilSenAndMedicalSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	if len(pupilIDs) == 0 {
		return map[string]PupilSenAndMedicalSummary{}, nil
	}

	uniqueIDs := make([]string, 0, len(pupilIDs))
	seen := make(map[string]struct{}, len(pupilIDs))
	result := make(map[string]PupilSenAndMedicalSummary, len(pupilIDs))

	for _, rawID := range pupilIDs {
		pupilID := strings.TrimSpace(rawID)
		if pupilID == "" {
			return nil, errors.New("pupilIDs contains an empty pupil_id")
		}
		if err := d.ensurePupilInOwnGroups(ctx, scope, pupilID); err != nil {
			return nil, err
		}
		if _, ok := seen[pupilID]; !ok {
			seen[pupilID] = struct{}{}
			uniqueIDs = append(uniqueIDs, pupilID)
		}
		result[pupilID] = PupilSenAndMedicalSummary{
			PupilID:           pupilID,
			SenProvisionCode:  "N",
			SenNeeds:          []PupilSenNeedSummary{},
			SenAssistants:     []PupilSenAssistantSummary{},
			MedicalConditions: []PupilMedicalConditionAlert{},
		}
	}

	placeholders := strings.Repeat("?,", len(uniqueIDs))
	placeholders = strings.TrimSuffix(placeholders, ",")

	buildInArgs := func(trailing ...any) []any {
		args := make([]any, 0, len(uniqueIDs)+len(trailing))
		for _, id := range uniqueIDs {
			args = append(args, id)
		}
		args = append(args, trailing...)
		return args
	}

	// 1. Batched query for pupil_sen_records (ordered by effective_from DESC so first seen per pupil is active latest)
	senRows, err := d.readDB.QueryContext(
		ctx,
		fmt.Sprintf(
			`SELECT pupil_id, provision_code, COALESCE(teaching_strategies, ''),
			        COALESCE(pupil_targets, ''), COALESCE(senco_notes, ''),
			        COALESCE(ehcp_start_date, ''), COALESCE(ehcp_review_due_date, '')
			 FROM pupil_sen_records
			 WHERE pupil_id IN (%s)
			   AND effective_from <= ?
			   AND (effective_to IS NULL OR effective_to >= ?)
			 ORDER BY pupil_id ASC, effective_from DESC`,
			placeholders,
		),
		buildInArgs(scope.ReferenceDate, scope.ReferenceDate)...,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query SEN records batch: %w", err)
	}
	seenSenRec := make(map[string]struct{}, len(uniqueIDs))
	for senRows.Next() {
		var pid, provCode, strategies, targets, notes, ehcpStart, ehcpReview string
		if scanErr := senRows.Scan(&pid, &provCode, &strategies, &targets, &notes, &ehcpStart, &ehcpReview); scanErr != nil {
			_ = senRows.Close()
			return nil, fmt.Errorf("failed to scan SEN record row: %w", scanErr)
		}
		if _, already := seenSenRec[pid]; already {
			continue
		}
		seenSenRec[pid] = struct{}{}
		summary := result[pid]
		summary.SenProvisionCode = provCode
		summary.TeachingStrategies = strategies
		summary.PupilTargets = targets
		summary.SencoNotes = notes
		summary.EHCPStartDate = ehcpStart
		summary.EHCPReviewDueDate = ehcpReview
		result[pid] = summary
	}
	if err := senRows.Err(); err != nil {
		_ = senRows.Close()
		return nil, fmt.Errorf("error iterating SEN records batch: %w", err)
	}
	_ = senRows.Close()

	// 2. Batched query for pupil_sen_needs (ranking 1 and 2)
	needRows, err := d.readDB.QueryContext(
		ctx,
		fmt.Sprintf(
			`SELECT id, pupil_id, sen_need_code, ranking, COALESCE(alert_level_id, ''), effective_from
			 FROM pupil_sen_needs
			 WHERE pupil_id IN (%s)
			   AND effective_from <= ?
			   AND (effective_to IS NULL OR effective_to >= ?)
			 ORDER BY pupil_id ASC, ranking ASC, effective_from DESC`,
			placeholders,
		),
		buildInArgs(scope.ReferenceDate, scope.ReferenceDate)...,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query SEN needs batch: %w", err)
	}
	for needRows.Next() {
		var need PupilSenNeedSummary
		var pid string
		if scanErr := needRows.Scan(&need.ID, &pid, &need.SenNeedCode, &need.Ranking, &need.AlertLevelID, &need.EffectiveFrom); scanErr != nil {
			_ = needRows.Close()
			return nil, fmt.Errorf("failed to scan SEN need row: %w", scanErr)
		}
		summary := result[pid]
		if need.Ranking == 1 && summary.PrimarySenNeedCode == "" {
			summary.PrimarySenNeedCode = need.SenNeedCode
		} else if need.Ranking == 2 && summary.SecondarySenNeedCode == "" {
			summary.SecondarySenNeedCode = need.SenNeedCode
		}
		summary.SenNeeds = append(summary.SenNeeds, need)
		result[pid] = summary
	}
	if err := needRows.Err(); err != nil {
		_ = needRows.Close()
		return nil, fmt.Errorf("error iterating SEN needs batch: %w", err)
	}
	_ = needRows.Close()

	// 3. Batched query for pupil_sen_assistants
	taRows, err := d.readDB.QueryContext(
		ctx,
		fmt.Sprintf(
			`SELECT psa.id, psa.pupil_id, psa.staff_id,
			        TRIM(COALESCE(per.title || ' ', '') || COALESCE(NULLIF(TRIM(per.preferred_first_name), ''), per.legal_first_name) || ' ' || COALESCE(NULLIF(TRIM(per.preferred_last_name), ''), per.legal_last_name)),
			        COALESCE(psa.allocation_notes, ''), psa.effective_from
			 FROM pupil_sen_assistants psa
			 JOIN staff st ON st.id = psa.staff_id
			 JOIN persons per ON per.id = st.person_id
			 WHERE psa.pupil_id IN (%s)
			   AND psa.effective_from <= ?
			   AND (psa.effective_to IS NULL OR psa.effective_to >= ?)
			 ORDER BY psa.pupil_id ASC, psa.id ASC`,
			placeholders,
		),
		buildInArgs(scope.ReferenceDate, scope.ReferenceDate)...,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query SEN assistants batch: %w", err)
	}
	for taRows.Next() {
		var ta PupilSenAssistantSummary
		var pid string
		if scanErr := taRows.Scan(&ta.ID, &pid, &ta.StaffID, &ta.DisplayName, &ta.AllocationNotes, &ta.EffectiveFrom); scanErr != nil {
			_ = taRows.Close()
			return nil, fmt.Errorf("failed to scan SEN assistant row: %w", scanErr)
		}
		summary := result[pid]
		summary.SenAssistants = append(summary.SenAssistants, ta)
		result[pid] = summary
	}
	if err := taRows.Err(); err != nil {
		_ = taRows.Close()
		return nil, fmt.Errorf("error iterating SEN assistants batch: %w", err)
	}
	_ = taRows.Close()

	// 4. Batched query for pupil_medical_conditions
	medRows, err := d.readDB.QueryContext(
		ctx,
		fmt.Sprintf(
			`SELECT id, pupil_id, condition_name, condition_category, COALESCE(alert_level_id, ''), has_ihcp,
			        COALESCE(triggers, ''), COALESCE(reaction_symptoms, ''), COALESCE(emergency_response_plan, ''),
			        COALESCE(medication_name_and_dose, ''), COALESCE(medication_location, ''),
			        parental_consent_given, effective_from, COALESCE(effective_to, '')
			 FROM pupil_medical_conditions
			 WHERE pupil_id IN (%s)
			   AND effective_from <= ?
			   AND (effective_to IS NULL OR effective_to >= ?)
			 ORDER BY pupil_id ASC, id ASC`,
			placeholders,
		),
		buildInArgs(scope.ReferenceDate, scope.ReferenceDate)...,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query medical conditions batch: %w", err)
	}
	for medRows.Next() {
		var cond PupilMedicalConditionAlert
		var pid string
		var hasIHCPInt int
		var consentInt int
		if scanErr := medRows.Scan(
			&cond.ID,
			&pid,
			&cond.ConditionName,
			&cond.ConditionCategory,
			&cond.AlertLevelID,
			&hasIHCPInt,
			&cond.Triggers,
			&cond.ReactionSymptoms,
			&cond.EmergencyResponsePlan,
			&cond.MedicationNameAndDose,
			&cond.MedicationLocation,
			&consentInt,
			&cond.EffectiveFrom,
			&cond.EffectiveTo,
		); scanErr != nil {
			_ = medRows.Close()
			return nil, fmt.Errorf("failed to scan medical condition: %w", scanErr)
		}
		cond.HasIHCP = hasIHCPInt == 1
		cond.ParentalConsentGiven = consentInt == 1
		summary := result[pid]
		summary.MedicalConditions = append(summary.MedicalConditions, cond)
		result[pid] = summary
	}
	if err := medRows.Err(); err != nil {
		_ = medRows.Close()
		return nil, fmt.Errorf("error iterating medical conditions batch: %w", err)
	}
	_ = medRows.Close()

	return result, nil
}

func (d *CoreDAL) ListSenRegisterScoped(
	ctx context.Context,
	scope DALQueryScope,
	opts SenRegisterFilterOptions,
) ([]PupilSenRegisterEntry, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}

	groupID := strings.TrimSpace(opts.GroupID)
	if groupID != "" && scope.Mode == ScopeOwnGroups {
		if err := d.ensureGroupInOwnGroups(ctx, scope, groupID); err != nil {
			return nil, err
		}
	}

	provCodes := opts.ProvisionCodes
	if len(provCodes) == 0 {
		provCodes = []string{"K", "E"}
	}
	cleanCodes := make([]string, 0, len(provCodes))
	for _, c := range provCodes {
		trimmed := strings.ToUpper(strings.TrimSpace(c))
		if trimmed != "" {
			cleanCodes = append(cleanCodes, trimmed)
		}
	}
	if len(cleanCodes) == 0 {
		cleanCodes = []string{"K", "E"}
	}

	codePlaceholders := strings.Repeat("?,", len(cleanCodes))
	codePlaceholders = strings.TrimSuffix(codePlaceholders, ",")

	query := fmt.Sprintf(
		`SELECT DISTINCT p.id, p.upn, per.legal_first_name, per.legal_last_name,
		        COALESCE(NULLIF(TRIM(per.preferred_first_name), ''), per.legal_first_name),
		        COALESCE(NULLIF(TRIM(per.preferred_last_name), ''), per.legal_last_name),
		        COALESCE(enr.nc_year_group, ''), COALESCE(enr.enrolment_status, 'C'),
		        enr.academic_year_id, enr.effective_from, enr.effective_to
		 FROM pupil_sen_records psr
		 JOIN pupils p ON p.id = psr.pupil_id
		 JOIN persons per ON per.id = p.person_id
		 JOIN pupil_year_enrolments enr ON enr.pupil_id = p.id
		 WHERE psr.provision_code IN (%s)
		   AND psr.effective_from <= ?
		   AND (psr.effective_to IS NULL OR psr.effective_to >= ?)
		   AND enr.academic_year_id = ?
		   AND enr.effective_from <= ?
		   AND (enr.effective_to IS NULL OR enr.effective_to >= ?)`,
		codePlaceholders,
	)

	args := make([]any, 0, len(cleanCodes)+8)
	for _, c := range cleanCodes {
		args = append(args, c)
	}
	args = append(args, scope.ReferenceDate, scope.ReferenceDate, scope.AcademicYearID, scope.ReferenceDate, scope.ReferenceDate)

	if ncYear := strings.TrimSpace(opts.NCYearGroup); ncYear != "" {
		query += ` AND enr.nc_year_group = ?`
		args = append(args, ncYear)
	}

	if groupID != "" {
		query += ` AND EXISTS (
			SELECT 1 FROM group_pupil_memberships gpm
			WHERE gpm.pupil_id = p.id
			  AND gpm.group_id = ?
			  AND gpm.academic_year_id = ?
			  AND gpm.effective_from <= ?
			  AND (gpm.effective_to IS NULL OR gpm.effective_to >= ?)
		)`
		args = append(args, groupID, scope.AcademicYearID, scope.ReferenceDate, scope.ReferenceDate)
	} else if scope.Mode == ScopeOwnGroups {
		query += ` AND EXISTS (
			SELECT 1 FROM group_pupil_memberships gpm
			JOIN group_staff_assignments gsa ON gsa.group_id = gpm.group_id
			WHERE gpm.pupil_id = p.id
			  AND gpm.academic_year_id = ?
			  AND gpm.effective_from <= ?
			  AND (gpm.effective_to IS NULL OR gpm.effective_to >= ?)
			  AND gsa.staff_id = ?
			  AND gsa.academic_year_id = ?
			  AND gsa.effective_from <= ?
			  AND (gsa.effective_to IS NULL OR gsa.effective_to >= ?)
		)`
		args = append(args,
			scope.AcademicYearID, scope.ReferenceDate, scope.ReferenceDate,
			scope.StaffID,
			scope.AcademicYearID, scope.ReferenceDate, scope.ReferenceDate,
		)
	}

	query += ` ORDER BY per.legal_last_name ASC, per.legal_first_name ASC, p.id ASC`

	rows, err := d.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query SEN register: %w", err)
	}
	defer rows.Close()

	var pupils []Pupil
	var pupilIDs []string
	for rows.Next() {
		var p Pupil
		var effTo sql.NullString
		if err := rows.Scan(
			&p.ID, &p.UPN, &p.FirstName, &p.LastName,
			&p.PreferredFirstName, &p.PreferredLastName,
			&p.NCYearGroup, &p.EnrolmentStatus,
			&p.AcademicYearID, &p.EffectiveFrom, &effTo,
		); err != nil {
			return nil, fmt.Errorf("failed to scan SEN register pupil row: %w", err)
		}
		p.PupilID = p.ID
		p.LegalFirstName = p.FirstName
		p.LegalLastName = p.LastName
		if effTo.Valid {
			val := effTo.String
			p.EffectiveTo = &val
		}
		pupils = append(pupils, p)
		pupilIDs = append(pupilIDs, p.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating SEN register rows: %w", err)
	}

	if len(pupils) == 0 {
		return []PupilSenRegisterEntry{}, nil
	}

	batchScope := scope
	batchScope.Mode = ScopeAll
	summaries, err := d.ListPupilSenAndMedicalBatch(ctx, batchScope, pupilIDs)
	if err != nil {
		return nil, err
	}

	entries := make([]PupilSenRegisterEntry, 0, len(pupils))
	for _, p := range pupils {
		s := summaries[p.ID]
		entries = append(entries, PupilSenRegisterEntry{
			Pupil:                p,
			SenProvisionCode:     s.SenProvisionCode,
			PrimarySenNeedCode:   s.PrimarySenNeedCode,
			SecondarySenNeedCode: s.SecondarySenNeedCode,
			TeachingStrategies:   s.TeachingStrategies,
			PupilTargets:         s.PupilTargets,
			SencoNotes:           s.SencoNotes,
			EHCPStartDate:        s.EHCPStartDate,
			EHCPReviewDueDate:    s.EHCPReviewDueDate,
			SenNeeds:             s.SenNeeds,
			SenAssistants:        s.SenAssistants,
		})
	}

	return entries, nil
}

