package dal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
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

type SenNeedTypeSummary struct {
	Code           string `json:"code"`
	Name           string `json:"name"`
	BroadArea      string `json:"broad_area"`
	Description    string `json:"description,omitempty"`
	SequenceNumber int    `json:"sequence_number"`
}

func (d *CoreDAL) ListSenNeedTypes(ctx context.Context) ([]SenNeedTypeSummary, error) {
	rows, err := d.readDB.QueryContext(
		ctx,
		`SELECT code, name, broad_area, COALESCE(description, ''), sequence_number
		 FROM sen_need_types
		 ORDER BY sequence_number ASC, code ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list sen_need_types: %w", err)
	}
	defer rows.Close()

	result := make([]SenNeedTypeSummary, 0)
	for rows.Next() {
		var r SenNeedTypeSummary
		if err := rows.Scan(&r.Code, &r.Name, &r.BroadArea, &r.Description, &r.SequenceNumber); err != nil {
			return nil, fmt.Errorf("failed to scan sen_need_types row: %w", err)
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

func (d *CoreDAL) UpsertSenNeedType(
	ctx context.Context,
	scope DALQueryScope,
	in SenNeedTypeSummary,
) (*SenNeedTypeSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	if scope.Mode != ScopeAll {
		return nil, errors.New("access denied: upserting SEN need types requires dal.sen.write:all scope")
	}

	code := strings.ToUpper(strings.TrimSpace(in.Code))
	name := strings.TrimSpace(in.Name)
	broadArea := strings.ToUpper(strings.TrimSpace(in.BroadArea))
	if broadArea == "" {
		broadArea = "OTHER"
	}
	if code == "" || name == "" {
		return nil, errors.New("missing mandatory parameter: code and name are required")
	}
	switch broadArea {
	case "COGNITION_LEARNING":
		broadArea = "COGNITION_AND_LEARNING"
	case "COMMUNICATION_INTERACTION":
		broadArea = "COMMUNICATION_AND_INTERACTION"
	case "SOCIAL_EMOTIONAL_MENTAL_HEALTH":
		broadArea = "SOCIAL_EMOTIONAL_AND_MENTAL_HEALTH"
	case "SENSORY_PHYSICAL":
		broadArea = "SENSORY_AND_PHYSICAL"
	}
	switch broadArea {
	case "COGNITION_AND_LEARNING", "COMMUNICATION_AND_INTERACTION", "SOCIAL_EMOTIONAL_AND_MENTAL_HEALTH", "SENSORY_AND_PHYSICAL", "OTHER":
	default:
		return nil, fmt.Errorf("invalid SEN need broad_area %q", in.BroadArea)
	}
	seq := in.SequenceNumber
	if seq <= 0 {
		_ = d.readDB.QueryRowContext(ctx, `SELECT COALESCE(MAX(sequence_number), 0) + 1 FROM sen_need_types`).Scan(&seq)
		if seq <= 0 {
			seq = 1
		}
	}

	if _, err := d.writeDB.ExecContext(
		ctx,
		`INSERT INTO sen_need_types (code, name, broad_area, description, sequence_number)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(code) DO UPDATE SET
		     name = excluded.name,
		     broad_area = excluded.broad_area,
		     description = excluded.description,
		     sequence_number = excluded.sequence_number`,
		code, name, broadArea, strings.TrimSpace(in.Description), seq,
	); err != nil {
		return nil, fmt.Errorf("failed to upsert sen_need_types row: %w", err)
	}

	return &SenNeedTypeSummary{
		Code:           code,
		Name:           name,
		BroadArea:      broadArea,
		Description:    strings.TrimSpace(in.Description),
		SequenceNumber: seq,
	}, nil
}

type UpsertPupilSenRecordInput struct {
	ID                          string `json:"id,omitempty"`
	PupilID                     string `json:"pupil_id"`
	ProvisionCode               string `json:"provision_code"`
	SENStatusCode               string `json:"sen_status_code,omitempty"`
	EHCPStartDate               string `json:"ehcp_start_date,omitempty"`
	EHCPReviewDueDate           string `json:"ehcp_review_due_date,omitempty"`
	SENUnitIndicator            bool   `json:"sen_unit_indicator,omitempty"`
	ResourcedProvisionIndicator bool   `json:"resourced_provision_indicator,omitempty"`
	TeachingStrategies          string `json:"teaching_strategies,omitempty"`
	PupilTargets                string `json:"pupil_targets,omitempty"`
	SencoNotes                  string `json:"senco_notes,omitempty"`
	EffectiveFrom               string `json:"effective_from"`
	EffectiveTo                 string `json:"effective_to,omitempty"`
}

func (d *CoreDAL) UpsertPupilSenRecordScoped(
	ctx context.Context,
	scope DALQueryScope,
	in UpsertPupilSenRecordInput,
) (*UpsertPupilSenRecordInput, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	pupilID := strings.TrimSpace(in.PupilID)
	provCode := strings.ToUpper(strings.TrimSpace(in.ProvisionCode))
	if provCode == "" {
		provCode = strings.ToUpper(strings.TrimSpace(in.SENStatusCode))
	}
	effFrom := strings.TrimSpace(in.EffectiveFrom)
	if effFrom == "" {
		effFrom = scope.ReferenceDate
	}
	if pupilID == "" || provCode == "" || effFrom == "" {
		return nil, errors.New("missing mandatory parameter: pupil_id, provision_code, and effective_from are required")
	}
	switch provCode {
	case "N", "K", "E":
	default:
		return nil, fmt.Errorf("invalid SEN provision_code %q: must be 'N', 'K', or 'E'", provCode)
	}
	if scope.Mode == ScopeOwnGroups {
		if err := d.ensurePupilInOwnGroups(ctx, scope, pupilID); err != nil {
			return nil, err
		}
	}

	recID := strings.TrimSpace(in.ID)
	if recID == "" {
		recID = "senr_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	}

	optStr := func(s string) any {
		if strings.TrimSpace(s) == "" {
			return nil
		}
		return strings.TrimSpace(s)
	}
	unitInt := 0
	if in.SENUnitIndicator {
		unitInt = 1
	}
	resInt := 0
	if in.ResourcedProvisionIndicator {
		resInt = 1
	}

	if _, err := d.writeDB.ExecContext(
		ctx,
		`INSERT INTO pupil_sen_records (
			id, pupil_id, provision_code, ehcp_start_date, ehcp_review_due_date,
			sen_unit_indicator, resourced_provision_indicator,
			teaching_strategies, pupil_targets, senco_notes,
			effective_from, effective_to
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			provision_code = excluded.provision_code,
			ehcp_start_date = excluded.ehcp_start_date,
			ehcp_review_due_date = excluded.ehcp_review_due_date,
			sen_unit_indicator = excluded.sen_unit_indicator,
			resourced_provision_indicator = excluded.resourced_provision_indicator,
			teaching_strategies = excluded.teaching_strategies,
			pupil_targets = excluded.pupil_targets,
			senco_notes = excluded.senco_notes,
			effective_from = excluded.effective_from,
			effective_to = excluded.effective_to`,
		recID, pupilID, provCode, optStr(in.EHCPStartDate), optStr(in.EHCPReviewDueDate),
		unitInt, resInt,
		strings.TrimSpace(in.TeachingStrategies), strings.TrimSpace(in.PupilTargets), strings.TrimSpace(in.SencoNotes),
		effFrom, optStr(in.EffectiveTo),
	); err != nil {
		return nil, fmt.Errorf("failed to upsert pupil_sen_records row: %w", err)
	}

	out := in
	out.ID = recID
	out.PupilID = pupilID
	out.ProvisionCode = provCode
	out.SENStatusCode = provCode
	out.EffectiveFrom = effFrom
	return &out, nil
}

type UpsertPupilSenNeedInput struct {
	ID            string `json:"id,omitempty"`
	PupilID       string `json:"pupil_id"`
	SenNeedCode   string `json:"sen_need_code"`
	Ranking       int    `json:"ranking"`
	AlertLevelID  string `json:"alert_level_id,omitempty"`
	EffectiveFrom string `json:"effective_from"`
	EffectiveTo   string `json:"effective_to,omitempty"`
}

func resolveAlertLevelID(ctx context.Context, db *sql.DB, raw string) any {
	trimmed := strings.ToUpper(strings.TrimSpace(raw))
	switch trimmed {
	case "1":
		trimmed = "CRITICAL"
	case "2":
		trimmed = "HIGH"
	case "3":
		trimmed = "MEDIUM"
	case "4":
		trimmed = "LOW"
	}
	if trimmed == "" {
		return nil
	}
	var exists int
	if err := db.QueryRowContext(ctx, `SELECT 1 FROM alert_levels WHERE id = ?`, trimmed).Scan(&exists); err == nil && exists == 1 {
		return trimmed
	}
	return nil
}

func (d *CoreDAL) UpsertPupilSenNeedScoped(
	ctx context.Context,
	scope DALQueryScope,
	in UpsertPupilSenNeedInput,
) (*PupilSenNeedSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	pupilID := strings.TrimSpace(in.PupilID)
	needCode := strings.ToUpper(strings.TrimSpace(in.SenNeedCode))
	effFrom := strings.TrimSpace(in.EffectiveFrom)
	if effFrom == "" {
		effFrom = scope.ReferenceDate
	}
	if pupilID == "" || needCode == "" || effFrom == "" {
		return nil, errors.New("missing mandatory parameter: pupil_id, sen_need_code, and effective_from are required")
	}
	if scope.Mode == ScopeOwnGroups {
		if err := d.ensurePupilInOwnGroups(ctx, scope, pupilID); err != nil {
			return nil, err
		}
	}
	ranking := in.Ranking
	if ranking != 2 {
		ranking = 1
	}
	needID := strings.TrimSpace(in.ID)
	if needID == "" {
		needID = "senn_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	}

	// Ensure sen_need_types has this code if not already seeded
	_, _ = d.writeDB.ExecContext(
		ctx,
		`INSERT INTO sen_need_types (code, name, broad_area, sequence_number)
		 VALUES (?, ?, 'OTHER', (SELECT COALESCE(MAX(sequence_number), 0) + 1 FROM sen_need_types))
		 ON CONFLICT(code) DO NOTHING`,
		needCode, needCode,
	)

	alertArg := resolveAlertLevelID(ctx, d.readDB, in.AlertLevelID)
	var effToArg any
	if strings.TrimSpace(in.EffectiveTo) != "" {
		effToArg = strings.TrimSpace(in.EffectiveTo)
	}

	if _, err := d.writeDB.ExecContext(
		ctx,
		`INSERT INTO pupil_sen_needs (
			id, pupil_id, sen_need_code, ranking, alert_level_id, effective_from, effective_to
		) VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			sen_need_code = excluded.sen_need_code,
			ranking = excluded.ranking,
			alert_level_id = excluded.alert_level_id,
			effective_from = excluded.effective_from,
			effective_to = excluded.effective_to`,
		needID, pupilID, needCode, ranking, alertArg, effFrom, effToArg,
	); err != nil {
		return nil, fmt.Errorf("failed to upsert pupil_sen_needs row: %w", err)
	}

	alertStr := ""
	if s, ok := alertArg.(string); ok {
		alertStr = s
	}
	return &PupilSenNeedSummary{
		ID:            needID,
		SenNeedCode:   needCode,
		Ranking:       ranking,
		AlertLevelID:  alertStr,
		EffectiveFrom: effFrom,
	}, nil
}

type UpsertPupilSenAssistantInput struct {
	ID              string `json:"id,omitempty"`
	PupilID         string `json:"pupil_id"`
	StaffID         string `json:"staff_id"`
	AllocationNotes string `json:"allocation_notes,omitempty"`
	EffectiveFrom   string `json:"effective_from"`
	EffectiveTo     string `json:"effective_to,omitempty"`
}

func (d *CoreDAL) UpsertPupilSenAssistantScoped(
	ctx context.Context,
	scope DALQueryScope,
	in UpsertPupilSenAssistantInput,
) (*PupilSenAssistantSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	pupilID := strings.TrimSpace(in.PupilID)
	staffID := strings.TrimSpace(in.StaffID)
	effFrom := strings.TrimSpace(in.EffectiveFrom)
	if effFrom == "" {
		effFrom = scope.ReferenceDate
	}
	if pupilID == "" || staffID == "" || effFrom == "" {
		return nil, errors.New("missing mandatory parameter: pupil_id, staff_id, and effective_from are required")
	}
	if scope.Mode == ScopeOwnGroups {
		if err := d.ensurePupilInOwnGroups(ctx, scope, pupilID); err != nil {
			return nil, err
		}
	}
	taID := strings.TrimSpace(in.ID)
	if taID == "" {
		taID = "senta_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	}

	var effToArg any
	if strings.TrimSpace(in.EffectiveTo) != "" {
		effToArg = strings.TrimSpace(in.EffectiveTo)
	}

	if _, err := d.writeDB.ExecContext(
		ctx,
		`INSERT INTO pupil_sen_assistants (
			id, pupil_id, staff_id, allocation_notes, effective_from, effective_to
		) VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			staff_id = excluded.staff_id,
			allocation_notes = excluded.allocation_notes,
			effective_from = excluded.effective_from,
			effective_to = excluded.effective_to`,
		taID, pupilID, staffID, strings.TrimSpace(in.AllocationNotes), effFrom, effToArg,
	); err != nil {
		return nil, fmt.Errorf("failed to upsert pupil_sen_assistants row: %w", err)
	}

	var displayName string
	_ = d.readDB.QueryRowContext(
		ctx,
		`SELECT TRIM(COALESCE(per.title || ' ', '') || COALESCE(NULLIF(TRIM(per.preferred_first_name), ''), per.legal_first_name) || ' ' || COALESCE(NULLIF(TRIM(per.preferred_last_name), ''), per.legal_last_name))
		 FROM staff st JOIN persons per ON per.id = st.person_id WHERE st.id = ?`,
		staffID,
	).Scan(&displayName)

	return &PupilSenAssistantSummary{
		ID:              taID,
		StaffID:         staffID,
		DisplayName:     displayName,
		AllocationNotes: strings.TrimSpace(in.AllocationNotes),
		EffectiveFrom:   effFrom,
	}, nil
}

type UpsertPupilMedicalConditionInput struct {
	ID                    string `json:"id,omitempty"`
	PupilID               string `json:"pupil_id"`
	ConditionName         string `json:"condition_name"`
	ConditionCategory     string `json:"condition_category"`
	AlertLevelID          string `json:"alert_level_id,omitempty"`
	HasIHCP               bool   `json:"has_ihcp"`
	Triggers              string `json:"triggers,omitempty"`
	ReactionSymptoms      string `json:"reaction_symptoms,omitempty"`
	EmergencyResponsePlan string `json:"emergency_response_plan,omitempty"`
	MedicationNameAndDose string `json:"medication_name_and_dose,omitempty"`
	MedicationLocation    string `json:"medication_location,omitempty"`
	ParentalConsentGiven  bool   `json:"parental_consent_given"`
	EffectiveFrom         string `json:"effective_from"`
	EffectiveTo           string `json:"effective_to,omitempty"`
}

func (d *CoreDAL) ListPupilMedicalConditionsScoped(
	ctx context.Context,
	scope DALQueryScope,
	pupilID string,
	conditionCategory string,
) ([]UpsertPupilMedicalConditionInput, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	trimmedPupil := strings.TrimSpace(pupilID)
	if trimmedPupil != "" && scope.Mode == ScopeOwnGroups {
		if err := d.ensurePupilInOwnGroups(ctx, scope, trimmedPupil); err != nil {
			return nil, err
		}
	}
	trimmedCat := strings.ToUpper(strings.TrimSpace(conditionCategory))

	query := `
		SELECT mc.id, mc.pupil_id, mc.condition_name, mc.condition_category,
		       COALESCE(mc.alert_level_id, ''), mc.has_ihcp,
		       COALESCE(mc.triggers, ''), COALESCE(mc.reaction_symptoms, ''),
		       COALESCE(mc.emergency_response_plan, ''), COALESCE(mc.medication_name_and_dose, ''),
		       COALESCE(mc.medication_location, ''), mc.parental_consent_given,
		       mc.effective_from, COALESCE(mc.effective_to, '')
		FROM pupil_medical_conditions mc WHERE 1=1`
	var args []any
	if trimmedPupil != "" {
		query += ` AND mc.pupil_id = ?`
		args = append(args, trimmedPupil)
	}
	if trimmedCat != "" {
		query += ` AND mc.condition_category = ?`
		args = append(args, trimmedCat)
	}
	if scope.Mode == ScopeOwnGroups && trimmedPupil == "" {
		query += ` AND EXISTS (
			SELECT 1 FROM group_pupil_memberships gpm
			JOIN group_staff_assignments gsa ON gsa.group_id = gpm.group_id
			WHERE gpm.pupil_id = mc.pupil_id
			  AND gsa.staff_id = ?
			  AND gpm.academic_year_id = ?
			  AND gpm.effective_from <= ?
			  AND (gpm.effective_to IS NULL OR gpm.effective_to >= ?)
		)`
		args = append(args, scope.StaffID, scope.AcademicYearID, scope.ReferenceDate, scope.ReferenceDate)
	}
	query += ` ORDER BY mc.pupil_id ASC, mc.id ASC`

	rows, err := d.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list pupil_medical_conditions: %w", err)
	}
	defer rows.Close()

	result := make([]UpsertPupilMedicalConditionInput, 0)
	for rows.Next() {
		var r UpsertPupilMedicalConditionInput
		var ihcpInt, consentInt int
		if err := rows.Scan(
			&r.ID, &r.PupilID, &r.ConditionName, &r.ConditionCategory,
			&r.AlertLevelID, &ihcpInt,
			&r.Triggers, &r.ReactionSymptoms,
			&r.EmergencyResponsePlan, &r.MedicationNameAndDose,
			&r.MedicationLocation, &consentInt,
			&r.EffectiveFrom, &r.EffectiveTo,
		); err != nil {
			return nil, fmt.Errorf("failed to scan pupil_medical_conditions row: %w", err)
		}
		r.HasIHCP = ihcpInt == 1
		r.ParentalConsentGiven = consentInt == 1
		result = append(result, r)
	}
	return result, rows.Err()
}

func (d *CoreDAL) UpsertPupilMedicalConditionScoped(
	ctx context.Context,
	scope DALQueryScope,
	in UpsertPupilMedicalConditionInput,
) (*UpsertPupilMedicalConditionInput, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	pupilID := strings.TrimSpace(in.PupilID)
	condName := strings.TrimSpace(in.ConditionName)
	condCat := strings.ToUpper(strings.TrimSpace(in.ConditionCategory))
	if condCat == "" {
		condCat = "MEDICAL"
	}
	effFrom := strings.TrimSpace(in.EffectiveFrom)
	if effFrom == "" {
		effFrom = scope.ReferenceDate
	}
	if pupilID == "" || condName == "" || effFrom == "" {
		return nil, errors.New("missing mandatory parameter: pupil_id, condition_name, and effective_from are required")
	}
	switch condCat {
	case "MEDICAL", "ALLERGY", "DIETARY":
	case "ANAPHYLAXIS", "PEANUT_ALLERGY", "HAYFEVER":
		condCat = "ALLERGY"
	case "HALAL", "KOSHER", "VEGAN", "VEGETARIAN", "COELIAC", "GLUTEN_FREE":
		condCat = "DIETARY"
	default:
		condCat = "MEDICAL"
	}
	if scope.Mode == ScopeOwnGroups {
		if err := d.ensurePupilInOwnGroups(ctx, scope, pupilID); err != nil {
			return nil, err
		}
	}

	condID := strings.TrimSpace(in.ID)
	if condID == "" {
		condID = "med_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	}

	alertArg := resolveAlertLevelID(ctx, d.readDB, in.AlertLevelID)
	ihcpInt := 0
	if in.HasIHCP {
		ihcpInt = 1
	}
	consentInt := 0
	if in.ParentalConsentGiven {
		consentInt = 1
	}
	optStr := func(s string) any {
		if strings.TrimSpace(s) == "" {
			return nil
		}
		return strings.TrimSpace(s)
	}

	if _, err := d.writeDB.ExecContext(
		ctx,
		`INSERT INTO pupil_medical_conditions (
			id, pupil_id, condition_name, condition_category, alert_level_id, has_ihcp,
			triggers, reaction_symptoms, emergency_response_plan,
			medication_name_and_dose, medication_location, parental_consent_given,
			effective_from, effective_to
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			condition_name = excluded.condition_name,
			condition_category = excluded.condition_category,
			alert_level_id = excluded.alert_level_id,
			has_ihcp = excluded.has_ihcp,
			triggers = excluded.triggers,
			reaction_symptoms = excluded.reaction_symptoms,
			emergency_response_plan = excluded.emergency_response_plan,
			medication_name_and_dose = excluded.medication_name_and_dose,
			medication_location = excluded.medication_location,
			parental_consent_given = excluded.parental_consent_given,
			effective_from = excluded.effective_from,
			effective_to = excluded.effective_to`,
		condID, pupilID, condName, condCat, alertArg, ihcpInt,
		strings.TrimSpace(in.Triggers), strings.TrimSpace(in.ReactionSymptoms), strings.TrimSpace(in.EmergencyResponsePlan),
		strings.TrimSpace(in.MedicationNameAndDose), strings.TrimSpace(in.MedicationLocation), consentInt,
		effFrom, optStr(in.EffectiveTo),
	); err != nil {
		return nil, fmt.Errorf("failed to upsert pupil_medical_conditions row: %w", err)
	}

	out := in
	out.ID = condID
	out.PupilID = pupilID
	out.ConditionName = condName
	out.ConditionCategory = condCat
	if s, ok := alertArg.(string); ok {
		out.AlertLevelID = s
	} else {
		out.AlertLevelID = strings.TrimSpace(in.AlertLevelID)
	}
	out.EffectiveFrom = effFrom
	return &out, nil
}

