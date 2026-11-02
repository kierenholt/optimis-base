package dal

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type SessionAttendanceMarkInput struct {
	PupilID        string `json:"pupil_id"`
	AttendanceCode string `json:"attendance_code"`
	MinutesLate    int    `json:"minutes_late"`
	SubCode        string `json:"sub_code"`
	Comment        string `json:"comment"`
}

type UpsertSessionAttendanceBatchInput struct {
	GroupID string                       `json:"group_id"`
	Date    string                       `json:"date"`
	Session string                       `json:"session"`
	Marks   []SessionAttendanceMarkInput `json:"marks"`
}

type UpsertSessionAttendanceBatchResult struct {
	SubmissionID  string `json:"submission_id"`
	UpsertedCount int    `json:"upserted_count"`
	GroupID       string `json:"group_id"`
	Date          string `json:"date"`
	Session       string `json:"session"`
}

type SessionAttendanceMark struct {
	ID                string `json:"id"`
	PupilID           string `json:"pupil_id"`
	AcademicYearID    string `json:"academic_year_id"`
	Date              string `json:"date"`
	Session           string `json:"session"`
	AttendanceCode    string `json:"attendance_code"`
	MinutesLate       int    `json:"minutes_late"`
	SubCode           string `json:"sub_code"`
	Comment           string `json:"comment"`
	SourceGroupID     string `json:"source_group_id"`
	RecordedByStaffID string `json:"recorded_by_staff_id"`
}

func (d *CoreDAL) GetGroupSessionAttendanceScoped(
	ctx context.Context,
	scope DALQueryScope,
	groupID string,
	date string,
	session string,
) ([]SessionAttendanceMark, error) {
	if strings.TrimSpace(groupID) == "" {
		return nil, errors.New("missing mandatory parameter: group_id")
	}
	if strings.TrimSpace(date) == "" {
		return nil, errors.New("missing mandatory parameter: date")
	}
	if session != "AM" && session != "PM" {
		return nil, fmt.Errorf("invalid statutory session %q: must be 'AM' or 'PM'", session)
	}
	if err := d.ensureGroupInOwnGroups(ctx, scope, groupID); err != nil {
		return nil, err
	}

	rows, err := d.readDB.QueryContext(
		ctx,
		`SELECT asm.id, asm.pupil_id, asm.academic_year_id, asm.date, asm.session,
		        asm.attendance_code, asm.minutes_late, asm.sub_code, asm.comment,
		        COALESCE(asm.source_group_id, ''), COALESCE(asm.recorded_by_staff_id, '')
		 FROM attendance_session_marks asm
		 JOIN group_pupil_memberships gpm ON gpm.pupil_id = asm.pupil_id AND gpm.group_id = ?
		 WHERE asm.date = ?
		   AND asm.session = ?
		   AND gpm.academic_year_id = ?
		   AND gpm.effective_from <= ?
		   AND (gpm.effective_to IS NULL OR gpm.effective_to >= ?)
		 ORDER BY asm.pupil_id ASC`,
		groupID, date, session, scope.AcademicYearID, date, date,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query group session attendance: %w", err)
	}
	defer rows.Close()

	var result []SessionAttendanceMark
	for rows.Next() {
		var m SessionAttendanceMark
		if err := rows.Scan(
			&m.ID,
			&m.PupilID,
			&m.AcademicYearID,
			&m.Date,
			&m.Session,
			&m.AttendanceCode,
			&m.MinutesLate,
			&m.SubCode,
			&m.Comment,
			&m.SourceGroupID,
			&m.RecordedByStaffID,
		); err != nil {
			return nil, fmt.Errorf("failed to scan attendance session mark: %w", err)
		}
		result = append(result, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating attendance session marks: %w", err)
	}
	return result, nil
}

func (d *CoreDAL) UpsertSessionAttendanceBatchScoped(
	ctx context.Context,
	scope DALQueryScope,
	in UpsertSessionAttendanceBatchInput,
) (*UpsertSessionAttendanceBatchResult, error) {
	if strings.TrimSpace(in.GroupID) == "" {
		return nil, errors.New("missing mandatory parameter: group_id")
	}
	if strings.TrimSpace(in.Date) == "" {
		return nil, errors.New("missing mandatory parameter: date")
	}
	if in.Session != "AM" && in.Session != "PM" {
		return nil, fmt.Errorf("invalid statutory session %q: must be 'AM' or 'PM'", in.Session)
	}
	if len(in.Marks) == 0 {
		return nil, errors.New("marks batch must contain at least one attendance mark")
	}
	if strings.TrimSpace(scope.StaffID) == "" {
		return nil, errors.New("missing mandatory StaffID for recording attendance submission")
	}
	if err := d.ensureGroupInOwnGroups(ctx, scope, in.GroupID); err != nil {
		return nil, err
	}

	tx, err := d.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin attendance batch transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	recordedPupilIDs := make([]string, 0, len(in.Marks))
	for _, mark := range in.Marks {
		pupilID := strings.TrimSpace(mark.PupilID)
		code := strings.TrimSpace(mark.AttendanceCode)
		if pupilID == "" || code == "" {
			return nil, errors.New("each attendance mark requires pupil_id and attendance_code")
		}
		if mark.MinutesLate < 0 {
			return nil, errors.New("minutes_late cannot be negative")
		}

		var memberCount int
		if err := tx.QueryRowContext(
			ctx,
			`SELECT COUNT(*)
			 FROM group_pupil_memberships
			 WHERE group_id = ?
			   AND pupil_id = ?
			   AND academic_year_id = ?
			   AND effective_from <= ?
			   AND (effective_to IS NULL OR effective_to >= ?)`,
			in.GroupID, pupilID, scope.AcademicYearID, in.Date, in.Date,
		).Scan(&memberCount); err != nil {
			return nil, fmt.Errorf("failed to verify group membership for attendance mark: %w", err)
		}
		if memberCount == 0 {
			return nil, fmt.Errorf("pupil is not an active member of group %s on %s", in.GroupID, in.Date)
		}

		markID := "asm_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
		_, err := tx.ExecContext(
			ctx,
			`INSERT INTO attendance_session_marks (
				id, pupil_id, academic_year_id, date, session,
				attendance_code, minutes_late, sub_code, comment,
				source_group_id, recorded_by_staff_id
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(pupil_id, date, session) DO UPDATE SET
				attendance_code = excluded.attendance_code,
				minutes_late = excluded.minutes_late,
				sub_code = excluded.sub_code,
				comment = excluded.comment,
				source_group_id = excluded.source_group_id,
				recorded_by_staff_id = excluded.recorded_by_staff_id,
				recorded_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')`,
			markID, pupilID, scope.AcademicYearID, in.Date, in.Session,
			code, mark.MinutesLate, mark.SubCode, mark.Comment,
			in.GroupID, scope.StaffID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to upsert attendance session mark: %w", err)
		}
		recordedPupilIDs = append(recordedPupilIDs, pupilID)
	}

	newSubID := "ars_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO attendance_register_submissions (
			id, group_id, date, session_or_slot_key, taken_by_staff_id
		) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(group_id, date, session_or_slot_key) DO UPDATE SET
			taken_by_staff_id = excluded.taken_by_staff_id,
			taken_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')`,
		newSubID, in.GroupID, in.Date, in.Session, scope.StaffID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert attendance_register_submissions row: %w", err)
	}

	var persistedSubID string
	if err := tx.QueryRowContext(
		ctx,
		`SELECT id FROM attendance_register_submissions WHERE group_id = ? AND date = ? AND session_or_slot_key = ?`,
		in.GroupID, in.Date, in.Session,
	).Scan(&persistedSubID); err != nil {
		return nil, fmt.Errorf("failed to read attendance register submission ID: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit attendance batch transaction: %w", err)
	}

	firstPupilID := ""
	if len(recordedPupilIDs) > 0 {
		firstPupilID = recordedPupilIDs[0]
	}
	d.emitPostCommitEvent(ctx, EventCoreAttendanceSessionRecorded, map[string]any{
		"submission_id":        persistedSubID,
		"group_id":             in.GroupID,
		"date":                 in.Date,
		"session":              in.Session,
		"academic_year_id":     scope.AcademicYearID,
		"upserted_count":       len(in.Marks),
		"recorded_by_staff_id": scope.StaffID,
		"pupil_id":             firstPupilID,
		"pupil_ids":            recordedPupilIDs,
	})

	return &UpsertSessionAttendanceBatchResult{
		SubmissionID:  persistedSubID,
		UpsertedCount: len(in.Marks),
		GroupID:       in.GroupID,
		Date:          in.Date,
		Session:       in.Session,
	}, nil
}

type PupilAttendanceSummary struct {
	PupilID           string `json:"pupil_id"`
	AcademicYearID    string `json:"academic_year_id"`
	ReferenceDate     string `json:"reference_date"`
	TotalSessions     int    `json:"total_sessions"`
	PresentSessions   int    `json:"present_sessions"`
	AMPresentSessions int    `json:"am_present_sessions"`
	PMPresentSessions int    `json:"pm_present_sessions"`
}

func (d *CoreDAL) GetPupilAttendanceSummaryScoped(
	ctx context.Context,
	scope DALQueryScope,
	pupilID string,
) (*PupilAttendanceSummary, error) {
	trimmedPupilID := strings.TrimSpace(pupilID)
	if trimmedPupilID == "" {
		return nil, errors.New("missing mandatory parameter: pupil_id")
	}
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	if err := d.ensurePupilInOwnGroups(ctx, scope, trimmedPupilID); err != nil {
		return nil, err
	}

	var totalSessions, presentSessions, amPresent, pmPresent int
	if err := d.readDB.QueryRowContext(
		ctx,
		`SELECT
			COUNT(*),
			COALESCE(SUM(CASE WHEN ac.statistical_category IN ('PRESENT', 'APPROVED_EDUCATIONAL_ACTIVITY') OR asm.attendance_code IN ('/', '\', 'L', 'B', 'P', 'V', 'W') THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN asm.session = 'AM' AND (ac.statistical_category IN ('PRESENT', 'APPROVED_EDUCATIONAL_ACTIVITY') OR asm.attendance_code IN ('/', '\', 'L', 'B', 'P', 'V', 'W')) THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN asm.session = 'PM' AND (ac.statistical_category IN ('PRESENT', 'APPROVED_EDUCATIONAL_ACTIVITY') OR asm.attendance_code IN ('/', '\', 'L', 'B', 'P', 'V', 'W')) THEN 1 ELSE 0 END), 0)
		 FROM attendance_session_marks asm
		 LEFT JOIN attendance_codes ac ON ac.code = asm.attendance_code
		 WHERE asm.pupil_id = ?
		   AND asm.academic_year_id = ?
		   AND asm.date <= ?`,
		trimmedPupilID, scope.AcademicYearID, scope.ReferenceDate,
	).Scan(&totalSessions, &presentSessions, &amPresent, &pmPresent); err != nil {
		return nil, fmt.Errorf("failed to query pupil attendance summary: %w", err)
	}

	return &PupilAttendanceSummary{
		PupilID:           trimmedPupilID,
		AcademicYearID:    scope.AcademicYearID,
		ReferenceDate:     scope.ReferenceDate,
		TotalSessions:     totalSessions,
		PresentSessions:   presentSessions,
		AMPresentSessions: amPresent,
		PMPresentSessions: pmPresent,
	}, nil
}

type AttendanceCodeSummary struct {
	Code                string `json:"code"`
	Name                string `json:"name"`
	Description         string `json:"description"`
	StatisticalCategory string `json:"statistical_category"`
	StatisticalMeaning  string `json:"statistical_meaning"`
	PhysicalStatus      string `json:"physical_status"`
	IsStatutoryCode     bool   `json:"is_statutory_code"`
	IsActive            bool   `json:"is_active"`
	RestrictedToAdmin   bool   `json:"restricted_to_admin"`
	SequenceNumber      int    `json:"sequence_number"`
}

func (d *CoreDAL) ListAttendanceCodes(ctx context.Context) ([]AttendanceCodeSummary, error) {
	rows, err := d.readDB.QueryContext(
		ctx,
		`SELECT code, name, statistical_category, physical_status, is_statutory_code, is_active, restricted_to_admin, sequence_number
		 FROM attendance_codes
		 ORDER BY sequence_number ASC, code ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list attendance_codes: %w", err)
	}
	defer rows.Close()

	result := make([]AttendanceCodeSummary, 0)
	for rows.Next() {
		var r AttendanceCodeSummary
		var statInt, actInt, restInt int
		if err := rows.Scan(
			&r.Code, &r.Name, &r.StatisticalCategory, &r.PhysicalStatus,
			&statInt, &actInt, &restInt, &r.SequenceNumber,
		); err != nil {
			return nil, fmt.Errorf("failed to scan attendance_codes row: %w", err)
		}
		r.Description = r.Name
		r.StatisticalMeaning = r.StatisticalCategory
		r.IsStatutoryCode = statInt == 1
		r.IsActive = actInt == 1
		r.RestrictedToAdmin = restInt == 1
		result = append(result, r)
	}
	return result, rows.Err()
}

func (d *CoreDAL) UpsertAttendanceCodeScoped(
	ctx context.Context,
	scope DALQueryScope,
	in AttendanceCodeSummary,
) (*AttendanceCodeSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	if scope.Mode != ScopeAll {
		return nil, errors.New("access denied: upserting attendance codes requires dal.attendance.write:all scope")
	}

	code := strings.TrimSpace(in.Code)
	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = strings.TrimSpace(in.Description)
	}
	statCat := strings.ToUpper(strings.TrimSpace(in.StatisticalCategory))
	if statCat == "" {
		statCat = strings.ToUpper(strings.TrimSpace(in.StatisticalMeaning))
	}
	if code == "" || name == "" || statCat == "" {
		return nil, errors.New("missing mandatory parameter: code, name/description, and statistical_category/statistical_meaning are required")
	}
	switch statCat {
	case "PRESENT", "APPROVED_EDUCATIONAL_ACTIVITY", "AUTHORISED_ABSENCE", "UNAUTHORISED_ABSENCE", "UNABLE_TO_ATTEND_EXCEPTIONAL", "NOT_REQUIRED_TO_ATTEND":
	default:
		return nil, fmt.Errorf("invalid statistical_category %q", statCat)
	}
	physStatus := strings.ToUpper(strings.TrimSpace(in.PhysicalStatus))
	if physStatus == "" {
		switch statCat {
		case "PRESENT":
			physStatus = "ONSITE"
		case "APPROVED_EDUCATIONAL_ACTIVITY":
			physStatus = "OFFSITE_EDUCATIONAL"
		case "NOT_REQUIRED_TO_ATTEND", "UNABLE_TO_ATTEND_EXCEPTIONAL":
			physStatus = "CLOSURE"
		default:
			physStatus = "ABSENT"
		}
	}
	seq := in.SequenceNumber
	if seq <= 0 {
		_ = d.readDB.QueryRowContext(ctx, `SELECT COALESCE(MAX(sequence_number), 0) + 1 FROM attendance_codes`).Scan(&seq)
		if seq <= 0 {
			seq = 1
		}
	}
	statInt := 1
	actInt := 1
	if !in.IsActive && in.SequenceNumber < 0 {
		actInt = 0
	}
	restInt := 0
	if in.RestrictedToAdmin {
		restInt = 1
	}

	if _, err := d.writeDB.ExecContext(
		ctx,
		`INSERT INTO attendance_codes (
			code, name, statistical_category, physical_status,
			is_statutory_code, is_active, restricted_to_admin, sequence_number
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(code) DO UPDATE SET
			name = excluded.name,
			statistical_category = excluded.statistical_category,
			physical_status = excluded.physical_status,
			is_active = excluded.is_active,
			restricted_to_admin = excluded.restricted_to_admin,
			sequence_number = excluded.sequence_number`,
		code, name, statCat, physStatus, statInt, actInt, restInt, seq,
	); err != nil {
		return nil, fmt.Errorf("failed to upsert attendance_codes row: %w", err)
	}

	return &AttendanceCodeSummary{
		Code:                code,
		Name:                name,
		Description:         name,
		StatisticalCategory: statCat,
		StatisticalMeaning:  statCat,
		PhysicalStatus:      physStatus,
		IsStatutoryCode:     statInt == 1,
		IsActive:            actInt == 1,
		RestrictedToAdmin:   restInt == 1,
		SequenceNumber:      seq,
	}, nil
}

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

type ListSessionMarksFilter struct {
	PupilID string `json:"pupil_id,omitempty"`
	GroupID string `json:"group_id,omitempty"`
	Date    string `json:"date,omitempty"`
	Session string `json:"session,omitempty"`
}

func (d *CoreDAL) ListSessionAttendanceMarksScoped(
	ctx context.Context,
	scope DALQueryScope,
	filter ListSessionMarksFilter,
) ([]SessionAttendanceMark, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	trimmedPupil := strings.TrimSpace(filter.PupilID)
	trimmedGroup := strings.TrimSpace(filter.GroupID)
	if trimmedPupil != "" && scope.Mode == ScopeOwnGroups {
		if err := d.ensurePupilInOwnGroups(ctx, scope, trimmedPupil); err != nil {
			return nil, err
		}
	}
	if trimmedGroup != "" && scope.Mode == ScopeOwnGroups {
		if err := d.ensureGroupInOwnGroups(ctx, scope, trimmedGroup); err != nil {
			return nil, err
		}
	}

	query := `
		SELECT asm.id, asm.pupil_id, asm.academic_year_id, asm.date, asm.session,
		       asm.attendance_code, asm.minutes_late, asm.sub_code, asm.comment,
		       COALESCE(asm.source_group_id, ''), COALESCE(asm.recorded_by_staff_id, '')
		FROM attendance_session_marks asm
		WHERE asm.academic_year_id = ?`
	args := []any{scope.AcademicYearID}

	if trimmedPupil != "" {
		query += ` AND asm.pupil_id = ?`
		args = append(args, trimmedPupil)
	}
	if trimmedGroup != "" {
		query += ` AND asm.source_group_id = ?`
		args = append(args, trimmedGroup)
	}
	if strings.TrimSpace(filter.Date) != "" {
		query += ` AND asm.date = ?`
		args = append(args, strings.TrimSpace(filter.Date))
	}
	if strings.TrimSpace(filter.Session) != "" {
		query += ` AND asm.session = ?`
		args = append(args, strings.ToUpper(strings.TrimSpace(filter.Session)))
	}
	if scope.Mode == ScopeOwnGroups && trimmedPupil == "" && trimmedGroup == "" {
		query += ` AND EXISTS (
			SELECT 1 FROM group_pupil_memberships gpm
			JOIN group_staff_assignments gsa ON gsa.group_id = gpm.group_id
			WHERE gpm.pupil_id = asm.pupil_id
			  AND gsa.staff_id = ?
			  AND gpm.academic_year_id = ?
			  AND gpm.effective_from <= ?
			  AND (gpm.effective_to IS NULL OR gpm.effective_to >= ?)
		)`
		args = append(args, scope.StaffID, scope.AcademicYearID, scope.ReferenceDate, scope.ReferenceDate)
	}
	query += ` ORDER BY asm.date DESC, asm.session ASC, asm.pupil_id ASC`

	rows, err := d.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list attendance_session_marks: %w", err)
	}
	defer rows.Close()

	result := make([]SessionAttendanceMark, 0)
	for rows.Next() {
		var m SessionAttendanceMark
		if err := rows.Scan(
			&m.ID, &m.PupilID, &m.AcademicYearID, &m.Date, &m.Session,
			&m.AttendanceCode, &m.MinutesLate, &m.SubCode, &m.Comment,
			&m.SourceGroupID, &m.RecordedByStaffID,
		); err != nil {
			return nil, fmt.Errorf("failed to scan attendance_session_marks row: %w", err)
		}
		result = append(result, m)
	}
	return result, rows.Err()
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

