package dal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// ============================================================================
// SQL Projections & Shared Field Constants
// ============================================================================

const senRegisterPupilFieldsSQL = `
	DISTINCT p.id, p.upn, per.legal_first_name, per.legal_last_name,
	COALESCE(NULLIF(TRIM(per.preferred_first_name), ''), per.legal_first_name),
	COALESCE(NULLIF(TRIM(per.preferred_last_name), ''), per.legal_last_name),
	COALESCE(enr.nc_year_group, ''), COALESCE(enr.enrolment_status, 'C'),
	enr.academic_year_id, enr.effective_from, enr.effective_to
`

// ============================================================================
// Internal Helpers & Row Scanners
// ============================================================================

func scanSenRegisterPupil(s rowScanner) (Pupil, error) {
	var p Pupil
	var effTo sql.NullString
	if err := s.Scan(
		&p.ID, &p.UPN, &p.FirstName, &p.LastName,
		&p.PreferredFirstName, &p.PreferredLastName,
		&p.NCYearGroup, &p.EnrolmentStatus,
		&p.AcademicYearID, &p.EffectiveFrom, &effTo,
	); err != nil {
		return Pupil{}, err
	}
	p.PupilID = p.ID
	p.LegalFirstName = p.FirstName
	p.LegalLastName = p.LastName
	if effTo.Valid {
		val := effTo.String
		p.EffectiveTo = &val
	}
	return p, nil
}

func buildInPlaceholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat("?,", n-1) + "?"
}

// ============================================================================
// Types & Structs
// ============================================================================

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

// ============================================================================
// Batch SEN & Medical Alerts
// ============================================================================

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

	placeholders := buildInPlaceholders(len(uniqueIDs))

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
			`SELECT %s
			 FROM pupil_sen_records psr
			 WHERE psr.pupil_id IN (%s)
			   AND psr.effective_from <= ?
			   AND (psr.effective_to IS NULL OR psr.effective_to >= ?)
			 ORDER BY psr.pupil_id ASC, psr.effective_from DESC`,
			senRecordSummaryFieldsSQL,
			placeholders,
		),
		buildInArgs(scope.ReferenceDate, scope.ReferenceDate)...,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query SEN records batch: %w", err)
	}
	seenSenRec := make(map[string]struct{}, len(uniqueIDs))
	for senRows.Next() {
		r, scanErr := scanSenRecordRow(senRows)
		if scanErr != nil {
			_ = senRows.Close()
			return nil, fmt.Errorf("failed to scan SEN record row: %w", scanErr)
		}
		if _, already := seenSenRec[r.pupilID]; already {
			continue
		}
		seenSenRec[r.pupilID] = struct{}{}
		summary := result[r.pupilID]
		summary.SenProvisionCode = r.provisionCode
		summary.TeachingStrategies = r.teachingStrategies
		summary.PupilTargets = r.pupilTargets
		summary.SencoNotes = r.sencoNotes
		summary.EHCPStartDate = r.ehcpStartDate
		summary.EHCPReviewDueDate = r.ehcpReviewDueDate
		result[r.pupilID] = summary
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
			`SELECT %s
			 FROM pupil_sen_needs psn
			 WHERE psn.pupil_id IN (%s)
			   AND psn.effective_from <= ?
			   AND (psn.effective_to IS NULL OR psn.effective_to >= ?)
			 ORDER BY psn.pupil_id ASC, psn.ranking ASC, psn.effective_from DESC`,
			senNeedSummaryFieldsSQL,
			placeholders,
		),
		buildInArgs(scope.ReferenceDate, scope.ReferenceDate)...,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query SEN needs batch: %w", err)
	}
	for needRows.Next() {
		pid, need, scanErr := scanSenNeedSummary(needRows)
		if scanErr != nil {
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
			`SELECT %s
			 FROM pupil_sen_assistants psa
			 JOIN staff st ON st.id = psa.staff_id
			 JOIN persons per ON per.id = st.person_id
			 WHERE psa.pupil_id IN (%s)
			   AND psa.effective_from <= ?
			   AND (psa.effective_to IS NULL OR psa.effective_to >= ?)
			 ORDER BY psa.pupil_id ASC, psa.id ASC`,
			senAssistantSummaryFieldsSQL,
			placeholders,
		),
		buildInArgs(scope.ReferenceDate, scope.ReferenceDate)...,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query SEN assistants batch: %w", err)
	}
	for taRows.Next() {
		pid, ta, scanErr := scanSenAssistantSummary(taRows)
		if scanErr != nil {
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
			`SELECT %s
			 FROM pupil_medical_conditions mc
			 WHERE mc.pupil_id IN (%s)
			   AND mc.effective_from <= ?
			   AND (mc.effective_to IS NULL OR mc.effective_to >= ?)
			 ORDER BY mc.pupil_id ASC, mc.id ASC`,
			medicalConditionFieldsSQL,
			placeholders,
		),
		buildInArgs(scope.ReferenceDate, scope.ReferenceDate)...,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query medical conditions batch: %w", err)
	}
	for medRows.Next() {
		pid, cond, scanErr := scanMedicalConditionAlert(medRows)
		if scanErr != nil {
			_ = medRows.Close()
			return nil, fmt.Errorf("failed to scan medical condition: %w", scanErr)
		}
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

// ============================================================================
// Whole-School SEN Register
// ============================================================================

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

	codePlaceholders := buildInPlaceholders(len(cleanCodes))

	query := fmt.Sprintf(
		`SELECT %s
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
		senRegisterPupilFieldsSQL,
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
		p, scanErr := scanSenRegisterPupil(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("failed to scan SEN register pupil row: %w", scanErr)
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
