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

const senRecordSummaryFieldsSQL = `
	psr.pupil_id, psr.provision_code, COALESCE(psr.teaching_strategies, ''),
	COALESCE(psr.pupil_targets, ''), COALESCE(psr.senco_notes, ''),
	COALESCE(psr.ehcp_start_date, ''), COALESCE(psr.ehcp_review_due_date, '')
`

const senNeedSummaryFieldsSQL = `
	psn.id, psn.pupil_id, psn.sen_need_code, psn.ranking,
	COALESCE(psn.alert_level_id, ''), psn.effective_from
`

const senAssistantSummaryFieldsSQL = `
	psa.id, psa.pupil_id, psa.staff_id,
	TRIM(COALESCE(per.title || ' ', '') || COALESCE(NULLIF(TRIM(per.preferred_first_name), ''), per.legal_first_name) || ' ' || COALESCE(NULLIF(TRIM(per.preferred_last_name), ''), per.legal_last_name)),
	COALESCE(psa.allocation_notes, ''), psa.effective_from
`

const senNeedTypeFieldsSQL = `
	code, name, broad_area, COALESCE(description, ''), sequence_number
`

// ============================================================================
// Internal Helpers & Row Scanners
// ============================================================================

type senRecordRow struct {
	pupilID            string
	provisionCode      string
	teachingStrategies string
	pupilTargets       string
	sencoNotes         string
	ehcpStartDate      string
	ehcpReviewDueDate  string
}

func scanSenRecordRow(s rowScanner) (senRecordRow, error) {
	var r senRecordRow
	err := s.Scan(
		&r.pupilID,
		&r.provisionCode,
		&r.teachingStrategies,
		&r.pupilTargets,
		&r.sencoNotes,
		&r.ehcpStartDate,
		&r.ehcpReviewDueDate,
	)
	return r, err
}

func scanSenNeedSummary(s rowScanner) (string, PupilSenNeedSummary, error) {
	var need PupilSenNeedSummary
	var pid string
	if err := s.Scan(&need.ID, &pid, &need.SenNeedCode, &need.Ranking, &need.AlertLevelID, &need.EffectiveFrom); err != nil {
		return "", PupilSenNeedSummary{}, err
	}
	return pid, need, nil
}

func scanSenAssistantSummary(s rowScanner) (string, PupilSenAssistantSummary, error) {
	var ta PupilSenAssistantSummary
	var pid string
	if err := s.Scan(&ta.ID, &pid, &ta.StaffID, &ta.DisplayName, &ta.AllocationNotes, &ta.EffectiveFrom); err != nil {
		return "", PupilSenAssistantSummary{}, err
	}
	return pid, ta, nil
}

func scanSenNeedTypeSummary(s rowScanner) (SenNeedTypeSummary, error) {
	var r SenNeedTypeSummary
	if err := s.Scan(&r.Code, &r.Name, &r.BroadArea, &r.Description, &r.SequenceNumber); err != nil {
		return SenNeedTypeSummary{}, err
	}
	return r, nil
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

func normalizeBroadArea(raw string) (string, error) {
	broadArea := strings.ToUpper(strings.TrimSpace(raw))
	if broadArea == "" {
		broadArea = "OTHER"
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
		return broadArea, nil
	default:
		return "", fmt.Errorf("invalid SEN need broad_area %q", raw)
	}
}

// ============================================================================
// Types & Structs
// ============================================================================

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

type SenNeedTypeSummary struct {
	Code           string `json:"code"`
	Name           string `json:"name"`
	BroadArea      string `json:"broad_area"`
	Description    string `json:"description,omitempty"`
	SequenceNumber int    `json:"sequence_number"`
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

type UpsertPupilSenNeedInput struct {
	ID            string `json:"id,omitempty"`
	PupilID       string `json:"pupil_id"`
	SenNeedCode   string `json:"sen_need_code"`
	Ranking       int    `json:"ranking"`
	AlertLevelID  string `json:"alert_level_id,omitempty"`
	EffectiveFrom string `json:"effective_from"`
	EffectiveTo   string `json:"effective_to,omitempty"`
}

type UpsertPupilSenAssistantInput struct {
	ID              string `json:"id,omitempty"`
	PupilID         string `json:"pupil_id"`
	StaffID         string `json:"staff_id"`
	AllocationNotes string `json:"allocation_notes,omitempty"`
	EffectiveFrom   string `json:"effective_from"`
	EffectiveTo     string `json:"effective_to,omitempty"`
}

// ============================================================================
// SEN Need Types (Taxonomy)
// ============================================================================

func (d *CoreDAL) ListSenNeedTypes(ctx context.Context) ([]SenNeedTypeSummary, error) {
	rows, err := d.readDB.QueryContext(
		ctx,
		`SELECT `+senNeedTypeFieldsSQL+`
		 FROM sen_need_types
		 ORDER BY sequence_number ASC, code ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list sen_need_types: %w", err)
	}
	defer rows.Close()

	result := make([]SenNeedTypeSummary, 0)
	for rows.Next() {
		r, scanErr := scanSenNeedTypeSummary(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("failed to scan sen_need_types row: %w", scanErr)
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
	if code == "" || name == "" {
		return nil, errors.New("missing mandatory parameter: code and name are required")
	}

	broadArea, err := normalizeBroadArea(in.BroadArea)
	if err != nil {
		return nil, err
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

// ============================================================================
// Pupil SEN Records & Provisions
// ============================================================================

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
		recID, pupilID, provCode, optTrimmedString(in.EHCPStartDate), optTrimmedString(in.EHCPReviewDueDate),
		boolToInt(in.SENUnitIndicator), boolToInt(in.ResourcedProvisionIndicator),
		strings.TrimSpace(in.TeachingStrategies), strings.TrimSpace(in.PupilTargets), strings.TrimSpace(in.SencoNotes),
		effFrom, optTrimmedString(in.EffectiveTo),
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

// ============================================================================
// Pupil SEN Needs
// ============================================================================

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
	effToArg := optTrimmedString(in.EffectiveTo)

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

// ============================================================================
// Pupil SEN Assistants
// ============================================================================

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

	effToArg := optTrimmedString(in.EffectiveTo)

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
