package dal

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type LessonAttendanceMark struct {
	ID                string `json:"id"`
	PupilID           string `json:"pupil_id"`
	TimetableSlotID   string `json:"timetable_slot_id"`
	GroupID           string `json:"group_id"`
	Date              string `json:"date"`
	AttendanceCode    string `json:"attendance_code"`
	MinutesLate       int    `json:"minutes_late"`
	Comment           string `json:"comment"`
	RecordedByStaffID string `json:"recorded_by_staff_id"`
}

func (d *CoreDAL) GetGroupLessonAttendanceScoped(
	ctx context.Context,
	scope DALQueryScope,
	groupID string,
	timetableSlotID string,
	date string,
) ([]LessonAttendanceMark, error) {
	if strings.TrimSpace(groupID) == "" || strings.TrimSpace(timetableSlotID) == "" || strings.TrimSpace(date) == "" {
		return nil, errors.New("missing mandatory parameter: group_id, timetable_slot_id, and date are required")
	}
	if err := d.ensureGroupInOwnGroups(ctx, scope, groupID); err != nil {
		return nil, err
	}

	rows, err := d.readDB.QueryContext(
		ctx,
		`SELECT alm.id, alm.pupil_id, alm.timetable_slot_id, alm.group_id, alm.date,
		        alm.attendance_code, alm.minutes_late, alm.comment, COALESCE(alm.recorded_by_staff_id, '')
		 FROM attendance_lesson_marks alm
		 WHERE alm.group_id = ?
		   AND alm.timetable_slot_id = ?
		   AND alm.date = ?
		 ORDER BY alm.pupil_id ASC`,
		strings.TrimSpace(groupID), strings.TrimSpace(timetableSlotID), strings.TrimSpace(date),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query group lesson attendance: %w", err)
	}
	defer rows.Close()

	result := make([]LessonAttendanceMark, 0)
	for rows.Next() {
		var m LessonAttendanceMark
		if err := rows.Scan(
			&m.ID, &m.PupilID, &m.TimetableSlotID, &m.GroupID, &m.Date,
			&m.AttendanceCode, &m.MinutesLate, &m.Comment, &m.RecordedByStaffID,
		); err != nil {
			return nil, fmt.Errorf("failed to scan attendance lesson mark: %w", err)
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

type UpsertLessonAttendanceBatchInput struct {
	GroupID         string                       `json:"group_id"`
	TimetableSlotID string                       `json:"timetable_slot_id"`
	Date            string                       `json:"date"`
	Marks           []SessionAttendanceMarkInput `json:"marks"`
}

type UpsertLessonAttendanceBatchResult struct {
	SubmissionID    string `json:"submission_id"`
	UpsertedCount   int    `json:"upserted_count"`
	GroupID         string `json:"group_id"`
	TimetableSlotID string `json:"timetable_slot_id"`
	Date            string `json:"date"`
}

func (d *CoreDAL) UpsertLessonAttendanceBatchScoped(
	ctx context.Context,
	scope DALQueryScope,
	in UpsertLessonAttendanceBatchInput,
) (*UpsertLessonAttendanceBatchResult, error) {
	groupID := strings.TrimSpace(in.GroupID)
	slotID := strings.TrimSpace(in.TimetableSlotID)
	date := strings.TrimSpace(in.Date)
	if groupID == "" || slotID == "" || date == "" {
		return nil, errors.New("missing mandatory parameter: group_id, timetable_slot_id, and date are required")
	}
	if len(in.Marks) == 0 {
		return nil, errors.New("marks batch must contain at least one attendance mark")
	}
	if strings.TrimSpace(scope.StaffID) == "" {
		return nil, errors.New("missing mandatory StaffID for recording lesson attendance")
	}
	if err := d.ensureGroupInOwnGroups(ctx, scope, groupID); err != nil {
		return nil, err
	}

	tx, err := d.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin lesson attendance batch tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	for _, mark := range in.Marks {
		pupilID := strings.TrimSpace(mark.PupilID)
		code := strings.TrimSpace(mark.AttendanceCode)
		if pupilID == "" || code == "" {
			return nil, errors.New("each lesson mark requires pupil_id and attendance_code")
		}
		if mark.MinutesLate < 0 {
			return nil, errors.New("minutes_late cannot be negative")
		}

		markID := "alm_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO attendance_lesson_marks (
				id, pupil_id, timetable_slot_id, group_id, date,
				attendance_code, minutes_late, comment, recorded_by_staff_id
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(pupil_id, timetable_slot_id, date) DO UPDATE SET
				attendance_code = excluded.attendance_code,
				minutes_late = excluded.minutes_late,
				comment = excluded.comment,
				recorded_by_staff_id = excluded.recorded_by_staff_id,
				recorded_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')`,
			markID, pupilID, slotID, groupID, date,
			code, mark.MinutesLate, strings.TrimSpace(mark.Comment), scope.StaffID,
		); err != nil {
			return nil, fmt.Errorf("failed to upsert attendance_lesson_marks row: %w", err)
		}
	}

	newSubID := "ars_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO attendance_register_submissions (
			id, group_id, date, session_or_slot_key, taken_by_staff_id
		) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(group_id, date, session_or_slot_key) DO UPDATE SET
			taken_by_staff_id = excluded.taken_by_staff_id,
			taken_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')`,
		newSubID, groupID, date, slotID, scope.StaffID,
	); err != nil {
		return nil, fmt.Errorf("failed to upsert lesson attendance_register_submissions row: %w", err)
	}

	var persistedSubID string
	if err := tx.QueryRowContext(
		ctx,
		`SELECT id FROM attendance_register_submissions WHERE group_id = ? AND date = ? AND session_or_slot_key = ?`,
		groupID, date, slotID,
	).Scan(&persistedSubID); err != nil {
		return nil, fmt.Errorf("failed to read lesson attendance register submission ID: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit lesson attendance batch tx: %w", err)
	}

	return &UpsertLessonAttendanceBatchResult{
		SubmissionID:    persistedSubID,
		UpsertedCount:   len(in.Marks),
		GroupID:         groupID,
		TimetableSlotID: slotID,
		Date:            date,
	}, nil
}

type AttendanceRegisterSubmissionSummary struct {
	ID               string `json:"id"`
	SubmissionID     string `json:"submission_id"`
	GroupID          string `json:"group_id"`
	Date             string `json:"date"`
	SessionOrSlotKey string `json:"session_or_slot_key"`
	TakenByStaffID   string `json:"taken_by_staff_id"`
	TakenAt          string `json:"taken_at"`
}

func (d *CoreDAL) ListAttendanceRegisterSubmissionsScoped(
	ctx context.Context,
	scope DALQueryScope,
	date string,
	groupID string,
) ([]AttendanceRegisterSubmissionSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	trimmedGroup := strings.TrimSpace(groupID)
	if trimmedGroup != "" && scope.Mode == ScopeOwnGroups {
		if err := d.ensureGroupInOwnGroups(ctx, scope, trimmedGroup); err != nil {
			return nil, err
		}
	}

	query := `
		SELECT ars.id, ars.group_id, ars.date, ars.session_or_slot_key, ars.taken_by_staff_id, COALESCE(ars.taken_at, '')
		FROM attendance_register_submissions ars
		WHERE 1=1`
	var args []any
	if strings.TrimSpace(date) != "" {
		query += ` AND ars.date = ?`
		args = append(args, strings.TrimSpace(date))
	}
	if trimmedGroup != "" {
		query += ` AND ars.group_id = ?`
		args = append(args, trimmedGroup)
	}
	if scope.Mode == ScopeOwnGroups && trimmedGroup == "" {
		query += ` AND EXISTS (
			SELECT 1 FROM group_staff_assignments gsa
			WHERE gsa.group_id = ars.group_id
			  AND gsa.staff_id = ?
			  AND gsa.academic_year_id = ?
			  AND gsa.effective_from <= ?
			  AND (gsa.effective_to IS NULL OR gsa.effective_to >= ?)
		)`
		args = append(args, scope.StaffID, scope.AcademicYearID, scope.ReferenceDate, scope.ReferenceDate)
	}
	query += ` ORDER BY ars.date DESC, ars.group_id ASC, ars.session_or_slot_key ASC`

	rows, err := d.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list attendance_register_submissions: %w", err)
	}
	defer rows.Close()

	result := make([]AttendanceRegisterSubmissionSummary, 0)
	for rows.Next() {
		var r AttendanceRegisterSubmissionSummary
		if err := rows.Scan(&r.ID, &r.GroupID, &r.Date, &r.SessionOrSlotKey, &r.TakenByStaffID, &r.TakenAt); err != nil {
			return nil, fmt.Errorf("failed to scan attendance_register_submissions row: %w", err)
		}
		r.SubmissionID = r.ID
		result = append(result, r)
	}
	return result, rows.Err()
}
