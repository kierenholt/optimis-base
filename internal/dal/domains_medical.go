package dal

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// ============================================================================
// SQL Projections & Shared Field Constants
// ============================================================================

const medicalConditionFieldsSQL = `
	mc.id, mc.pupil_id, mc.condition_name, mc.condition_category,
	COALESCE(mc.alert_level_id, ''), mc.has_ihcp,
	COALESCE(mc.triggers, ''), COALESCE(mc.reaction_symptoms, ''),
	COALESCE(mc.emergency_response_plan, ''), COALESCE(mc.medication_name_and_dose, ''),
	COALESCE(mc.medication_location, ''), mc.parental_consent_given,
	mc.effective_from, COALESCE(mc.effective_to, '')
`

// ============================================================================
// Internal Helpers & Row Scanners
// ============================================================================

func scanMedicalConditionAlert(s rowScanner) (string, PupilMedicalConditionAlert, error) {
	var cond PupilMedicalConditionAlert
	var pid string
	var hasIHCPInt, consentInt int
	if err := s.Scan(
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
	); err != nil {
		return "", PupilMedicalConditionAlert{}, err
	}
	cond.HasIHCP = hasIHCPInt == 1
	cond.ParentalConsentGiven = consentInt == 1
	return pid, cond, nil
}

func scanMedicalConditionInput(s rowScanner) (UpsertPupilMedicalConditionInput, error) {
	var r UpsertPupilMedicalConditionInput
	var hasIHCPInt, consentInt int
	if err := s.Scan(
		&r.ID,
		&r.PupilID,
		&r.ConditionName,
		&r.ConditionCategory,
		&r.AlertLevelID,
		&hasIHCPInt,
		&r.Triggers,
		&r.ReactionSymptoms,
		&r.EmergencyResponsePlan,
		&r.MedicationNameAndDose,
		&r.MedicationLocation,
		&consentInt,
		&r.EffectiveFrom,
		&r.EffectiveTo,
	); err != nil {
		return UpsertPupilMedicalConditionInput{}, err
	}
	r.HasIHCP = hasIHCPInt == 1
	r.ParentalConsentGiven = consentInt == 1
	return r, nil
}

func normalizeConditionCategory(raw string) string {
	condCat := strings.ToUpper(strings.TrimSpace(raw))
	switch condCat {
	case "MEDICAL", "ALLERGY", "DIETARY":
		return condCat
	case "ANAPHYLAXIS", "PEANUT_ALLERGY", "HAYFEVER":
		return "ALLERGY"
	case "HALAL", "KOSHER", "VEGAN", "VEGETARIAN", "COELIAC", "GLUTEN_FREE":
		return "DIETARY"
	default:
		return "MEDICAL"
	}
}

// ============================================================================
// Types & Structs
// ============================================================================

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

// ============================================================================
// Pupil Medical Conditions
// ============================================================================

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
		SELECT ` + medicalConditionFieldsSQL + `
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
		r, scanErr := scanMedicalConditionInput(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("failed to scan pupil_medical_conditions row: %w", scanErr)
		}
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
	condCat := normalizeConditionCategory(in.ConditionCategory)
	effFrom := strings.TrimSpace(in.EffectiveFrom)
	if effFrom == "" {
		effFrom = scope.ReferenceDate
	}
	if pupilID == "" || condName == "" || effFrom == "" {
		return nil, errors.New("missing mandatory parameter: pupil_id, condition_name, and effective_from are required")
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
		condID, pupilID, condName, condCat, alertArg, boolToInt(in.HasIHCP),
		strings.TrimSpace(in.Triggers), strings.TrimSpace(in.ReactionSymptoms), strings.TrimSpace(in.EmergencyResponsePlan),
		strings.TrimSpace(in.MedicationNameAndDose), strings.TrimSpace(in.MedicationLocation), boolToInt(in.ParentalConsentGiven),
		effFrom, optTrimmedString(in.EffectiveTo),
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
