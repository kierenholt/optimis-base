package dal

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

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
		optionalTrimmedString(in.CourseID), glh, in.PlannedEEPHours,
		compStatus, outcome, startDate, plannedEnd, optionalTrimmedString(in.ActualEndDate),
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
