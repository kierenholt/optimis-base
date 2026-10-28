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
	EmergencyResponsePlan string `json:"emergency_response_plan"`
}

type PupilSenAndMedicalSummary struct {
	PupilID            string                       `json:"pupil_id"`
	SenProvisionCode   string                       `json:"sen_provision_code"`
	PrimarySenNeedCode string                       `json:"primary_sen_need_code"`
	TeachingStrategies string                       `json:"teaching_strategies"`
	MedicalConditions  []PupilMedicalConditionAlert `json:"medical_conditions"`
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
		`SELECT p.id, p.upn, per.legal_first_name, per.legal_last_name, enr.academic_year_id, enr.effective_from, enr.effective_to
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
	if err := row.Scan(&p.ID, &p.UPN, &p.FirstName, &p.LastName, &p.AcademicYearID, &p.EffectiveFrom, &effTo); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("pupil not found or not enrolled on %s", scope.ReferenceDate)
		}
		return nil, fmt.Errorf("failed to query scoped pupil: %w", err)
	}
	if effTo.Valid {
		val := effTo.String
		p.EffectiveTo = &val
	}
	return &p, nil
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
		`SELECT p.id, p.upn, per.legal_first_name, per.legal_last_name, gpm.academic_year_id, gpm.effective_from, gpm.effective_to
		 FROM group_pupil_memberships gpm
		 JOIN pupils p ON p.id = gpm.pupil_id
		 JOIN persons per ON per.id = p.person_id
		 WHERE gpm.group_id = ?
		   AND gpm.academic_year_id = ?
		   AND gpm.effective_from <= ?
		   AND (gpm.effective_to IS NULL OR gpm.effective_to >= ?)
		 ORDER BY per.legal_last_name ASC, per.legal_first_name ASC, p.id ASC`,
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
		if err := rows.Scan(&p.ID, &p.UPN, &p.FirstName, &p.LastName, &p.AcademicYearID, &p.EffectiveFrom, &effTo); err != nil {
			return nil, fmt.Errorf("failed to scan group pupil row: %w", err)
		}
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

	result := make(map[string]PupilSenAndMedicalSummary, len(pupilIDs))
	for _, rawID := range pupilIDs {
		pupilID := strings.TrimSpace(rawID)
		if pupilID == "" {
			return nil, errors.New("pupilIDs contains an empty pupil_id")
		}
		if err := d.ensurePupilInOwnGroups(ctx, scope, pupilID); err != nil {
			return nil, err
		}

		summary := PupilSenAndMedicalSummary{
			PupilID:           pupilID,
			SenProvisionCode:  "N",
			MedicalConditions: []PupilMedicalConditionAlert{},
		}

		var provCode, strategies string
		err := d.readDB.QueryRowContext(
			ctx,
			`SELECT provision_code, teaching_strategies
			 FROM pupil_sen_records
			 WHERE pupil_id = ?
			   AND effective_from <= ?
			   AND (effective_to IS NULL OR effective_to >= ?)
			 ORDER BY effective_from DESC
			 LIMIT 1`,
			pupilID, scope.ReferenceDate, scope.ReferenceDate,
		).Scan(&provCode, &strategies)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("failed to query SEN record: %w", err)
		}
		if err == nil {
			summary.SenProvisionCode = provCode
			summary.TeachingStrategies = strategies
		}

		var primaryNeed string
		err = d.readDB.QueryRowContext(
			ctx,
			`SELECT sen_need_code
			 FROM pupil_sen_needs
			 WHERE pupil_id = ?
			   AND ranking = 1
			   AND effective_from <= ?
			   AND (effective_to IS NULL OR effective_to >= ?)
			 ORDER BY effective_from DESC
			 LIMIT 1`,
			pupilID, scope.ReferenceDate, scope.ReferenceDate,
		).Scan(&primaryNeed)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("failed to query primary SEN need: %w", err)
		}
		if err == nil {
			summary.PrimarySenNeedCode = primaryNeed
		}

		medRows, err := d.readDB.QueryContext(
			ctx,
			`SELECT id, condition_name, condition_category, COALESCE(alert_level_id, ''), has_ihcp, emergency_response_plan
			 FROM pupil_medical_conditions
			 WHERE pupil_id = ?
			   AND effective_from <= ?
			   AND (effective_to IS NULL OR effective_to >= ?)
			 ORDER BY id ASC`,
			pupilID, scope.ReferenceDate, scope.ReferenceDate,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to query medical conditions: %w", err)
		}
		for medRows.Next() {
			var cond PupilMedicalConditionAlert
			var hasIHCPInt int
			if scanErr := medRows.Scan(
				&cond.ID,
				&cond.ConditionName,
				&cond.ConditionCategory,
				&cond.AlertLevelID,
				&hasIHCPInt,
				&cond.EmergencyResponsePlan,
			); scanErr != nil {
				_ = medRows.Close()
				return nil, fmt.Errorf("failed to scan medical condition: %w", scanErr)
			}
			cond.HasIHCP = hasIHCPInt == 1
			summary.MedicalConditions = append(summary.MedicalConditions, cond)
		}
		if err := medRows.Err(); err != nil {
			_ = medRows.Close()
			return nil, fmt.Errorf("error iterating medical conditions: %w", err)
		}
		_ = medRows.Close()

		result[pupilID] = summary
	}

	return result, nil
}
