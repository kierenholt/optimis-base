package dal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type GroupSummary struct {
	ID              string  `json:"id"`
	AcademicYearID  string  `json:"academic_year_id"`
	GroupType       string  `json:"group_type"`
	Code            string  `json:"code"`
	Name            string  `json:"name"`
	DefaultRoomID   string  `json:"default_room_id"`
	TakesAttendance bool    `json:"takes_attendance"`
	EffectiveFrom   string  `json:"effective_from"`
	EffectiveTo     *string `json:"effective_to,omitempty"`
}

type TimetableSlotSummary struct {
	SlotID                 string `json:"slot_id"`
	PeriodID               string `json:"period_id"`
	PeriodCode             string `json:"period_code"`
	PeriodName             string `json:"period_name"`
	CycleWeek              int    `json:"cycle_week"`
	DayOfWeekID            int    `json:"day_of_week_id"`
	StartTime              string `json:"start_time"`
	EndTime                string `json:"end_time"`
	PeriodType             string `json:"period_type"`
	MapsToStatutorySession string `json:"maps_to_statutory_session"`
	GroupID                string `json:"group_id"`
	GroupCode              string `json:"group_code"`
	StaffID                string `json:"staff_id"`
	RoomID                 string `json:"room_id"`
}

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

func (d *CoreDAL) ListGroupsScoped(ctx context.Context, scope DALQueryScope, groupType string) ([]GroupSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}

	var rows *sql.Rows
	var err error
	trimmedType := strings.TrimSpace(groupType)

	if scope.Mode == ScopeOwnGroups {
		if trimmedType != "" {
			rows, err = d.readDB.QueryContext(
				ctx,
				`SELECT DISTINCT g.id, g.academic_year_id, g.group_type, g.code, g.name, COALESCE(g.default_room_id, ''), g.takes_attendance, g.effective_from, g.effective_to
				 FROM groups g
				 JOIN group_staff_assignments gsa ON gsa.group_id = g.id
				 WHERE g.academic_year_id = ?
				   AND g.group_type = ?
				   AND g.effective_from <= ?
				   AND (g.effective_to IS NULL OR g.effective_to >= ?)
				   AND gsa.staff_id = ?
				   AND gsa.academic_year_id = ?
				   AND gsa.effective_from <= ?
				   AND (gsa.effective_to IS NULL OR gsa.effective_to >= ?)
				 ORDER BY g.code ASC, g.id ASC`,
				scope.AcademicYearID, trimmedType, scope.ReferenceDate, scope.ReferenceDate,
				scope.StaffID, scope.AcademicYearID, scope.ReferenceDate, scope.ReferenceDate,
			)
		} else {
			rows, err = d.readDB.QueryContext(
				ctx,
				`SELECT DISTINCT g.id, g.academic_year_id, g.group_type, g.code, g.name, COALESCE(g.default_room_id, ''), g.takes_attendance, g.effective_from, g.effective_to
				 FROM groups g
				 JOIN group_staff_assignments gsa ON gsa.group_id = g.id
				 WHERE g.academic_year_id = ?
				   AND g.effective_from <= ?
				   AND (g.effective_to IS NULL OR g.effective_to >= ?)
				   AND gsa.staff_id = ?
				   AND gsa.academic_year_id = ?
				   AND gsa.effective_from <= ?
				   AND (gsa.effective_to IS NULL OR gsa.effective_to >= ?)
				 ORDER BY g.code ASC, g.id ASC`,
				scope.AcademicYearID, scope.ReferenceDate, scope.ReferenceDate,
				scope.StaffID, scope.AcademicYearID, scope.ReferenceDate, scope.ReferenceDate,
			)
		}
	} else {
		if trimmedType != "" {
			rows, err = d.readDB.QueryContext(
				ctx,
				`SELECT g.id, g.academic_year_id, g.group_type, g.code, g.name, COALESCE(g.default_room_id, ''), g.takes_attendance, g.effective_from, g.effective_to
				 FROM groups g
				 WHERE g.academic_year_id = ?
				   AND g.group_type = ?
				   AND g.effective_from <= ?
				   AND (g.effective_to IS NULL OR g.effective_to >= ?)
				 ORDER BY g.code ASC, g.id ASC`,
				scope.AcademicYearID, trimmedType, scope.ReferenceDate, scope.ReferenceDate,
			)
		} else {
			rows, err = d.readDB.QueryContext(
				ctx,
				`SELECT g.id, g.academic_year_id, g.group_type, g.code, g.name, COALESCE(g.default_room_id, ''), g.takes_attendance, g.effective_from, g.effective_to
				 FROM groups g
				 WHERE g.academic_year_id = ?
				   AND g.effective_from <= ?
				   AND (g.effective_to IS NULL OR g.effective_to >= ?)
				 ORDER BY g.code ASC, g.id ASC`,
				scope.AcademicYearID, scope.ReferenceDate, scope.ReferenceDate,
			)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to list scoped groups: %w", err)
	}
	defer rows.Close()

	var result []GroupSummary
	for rows.Next() {
		var g GroupSummary
		var takesAttInt int
		var effTo sql.NullString
		if err := rows.Scan(
			&g.ID,
			&g.AcademicYearID,
			&g.GroupType,
			&g.Code,
			&g.Name,
			&g.DefaultRoomID,
			&takesAttInt,
			&g.EffectiveFrom,
			&effTo,
		); err != nil {
			return nil, fmt.Errorf("failed to scan group row: %w", err)
		}
		g.TakesAttendance = takesAttInt == 1
		if effTo.Valid {
			val := effTo.String
			g.EffectiveTo = &val
		}
		result = append(result, g)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating groups: %w", err)
	}
	return result, nil
}

func (d *CoreDAL) GetStaffTimetableSlotsScoped(
	ctx context.Context,
	scope DALQueryScope,
	staffID string,
	cycleWeek int,
	dayOfWeekID int,
) ([]TimetableSlotSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	targetStaffID := strings.TrimSpace(staffID)
	if targetStaffID == "" {
		return nil, errors.New("missing mandatory parameter: staff_id")
	}
	if scope.Mode == ScopeOwnGroups && targetStaffID != scope.StaffID {
		return nil, fmt.Errorf("access denied under own_groups scope: staff %s cannot read timetable for staff %s", scope.StaffID, targetStaffID)
	}
	if cycleWeek != 1 && cycleWeek != 2 {
		return nil, fmt.Errorf("invalid cycleWeek %d: must be 1 or 2", cycleWeek)
	}

	rows, err := d.readDB.QueryContext(
		ctx,
		`SELECT ts.id, tp.id, tp.period_code, tp.name, tp.cycle_week, tp.day_of_week_id,
		        tp.start_time, tp.end_time, tp.period_type, COALESCE(tp.maps_to_statutory_session, ''),
		        ts.group_id, g.code, COALESCE(ts.staff_id, ''), COALESCE(ts.room_id, '')
		 FROM timetable_slots ts
		 JOIN timetable_periods tp ON tp.id = ts.timetable_period_id
		 JOIN groups g ON g.id = ts.group_id
		 WHERE ts.staff_id = ?
		   AND tp.academic_year_id = ?
		   AND tp.cycle_week = ?
		   AND tp.day_of_week_id = ?
		   AND ts.effective_from <= ?
		   AND (ts.effective_to IS NULL OR ts.effective_to >= ?)
		 ORDER BY tp.sequence_number ASC, ts.id ASC`,
		targetStaffID, scope.AcademicYearID, cycleWeek, dayOfWeekID, scope.ReferenceDate, scope.ReferenceDate,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query staff timetable slots: %w", err)
	}
	defer rows.Close()

	var result []TimetableSlotSummary
	for rows.Next() {
		var slot TimetableSlotSummary
		if err := rows.Scan(
			&slot.SlotID,
			&slot.PeriodID,
			&slot.PeriodCode,
			&slot.PeriodName,
			&slot.CycleWeek,
			&slot.DayOfWeekID,
			&slot.StartTime,
			&slot.EndTime,
			&slot.PeriodType,
			&slot.MapsToStatutorySession,
			&slot.GroupID,
			&slot.GroupCode,
			&slot.StaffID,
			&slot.RoomID,
		); err != nil {
			return nil, fmt.Errorf("failed to scan timetable slot: %w", err)
		}
		result = append(result, slot)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating timetable slots: %w", err)
	}
	return result, nil
}

type UpsertGroupPupilMembershipInput struct {
	MembershipID   string `json:"membership_id,omitempty"`
	GroupID        string `json:"group_id"`
	PupilID        string `json:"pupil_id"`
	AcademicYearID string `json:"academic_year_id"`
	EffectiveFrom  string `json:"effective_from"`
	EffectiveTo    string `json:"effective_to,omitempty"`
}

type UpsertGroupPupilMembershipResult struct {
	MembershipID   string `json:"membership_id"`
	GroupID        string `json:"group_id"`
	PupilID        string `json:"pupil_id"`
	AcademicYearID string `json:"academic_year_id"`
	EffectiveFrom  string `json:"effective_from"`
	EffectiveTo    string `json:"effective_to,omitempty"`
}

func (d *CoreDAL) UpsertGroupPupilMembershipScoped(
	ctx context.Context,
	scope DALQueryScope,
	in UpsertGroupPupilMembershipInput,
) (*UpsertGroupPupilMembershipResult, error) {
	groupID := strings.TrimSpace(in.GroupID)
	pupilID := strings.TrimSpace(in.PupilID)
	ayID := strings.TrimSpace(in.AcademicYearID)
	if ayID == "" {
		ayID = strings.TrimSpace(scope.AcademicYearID)
	}
	effFrom := strings.TrimSpace(in.EffectiveFrom)
	if effFrom == "" {
		effFrom = strings.TrimSpace(scope.ReferenceDate)
	}
	if groupID == "" {
		return nil, errors.New("missing mandatory parameter: group_id")
	}
	if pupilID == "" {
		return nil, errors.New("missing mandatory parameter: pupil_id")
	}
	if ayID == "" {
		return nil, errors.New("missing mandatory parameter: academic_year_id")
	}
	if effFrom == "" {
		return nil, errors.New("missing mandatory parameter: effective_from")
	}
	if err := d.ensureGroupInOwnGroups(ctx, scope, groupID); err != nil {
		return nil, err
	}

	membershipID := strings.TrimSpace(in.MembershipID)
	if membershipID == "" {
		membershipID = "gpm_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	}

	var effToArg any
	trimmedEffTo := strings.TrimSpace(in.EffectiveTo)
	if trimmedEffTo != "" {
		effToArg = trimmedEffTo
	}

	tx, err := d.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin group membership transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO group_pupil_memberships (
			id, group_id, pupil_id, academic_year_id, effective_from, effective_to
		) VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			effective_from = excluded.effective_from,
			effective_to = excluded.effective_to`,
		membershipID, groupID, pupilID, ayID, effFrom, effToArg,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert group_pupil_memberships row: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit group membership transaction: %w", err)
	}

	d.emitPostCommitEvent(ctx, EventCoreGroupMembershipChanged, map[string]any{
		"membership_id":    membershipID,
		"group_id":         groupID,
		"pupil_id":         pupilID,
		"academic_year_id": ayID,
		"effective_from":   effFrom,
		"effective_to":     trimmedEffTo,
	})

	return &UpsertGroupPupilMembershipResult{
		MembershipID:   membershipID,
		GroupID:        groupID,
		PupilID:        pupilID,
		AcademicYearID: ayID,
		EffectiveFrom:  effFrom,
		EffectiveTo:    trimmedEffTo,
	}, nil
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
	PupilID            string `json:"pupil_id"`
	AcademicYearID     string `json:"academic_year_id"`
	ReferenceDate      string `json:"reference_date"`
	TotalSessions      int    `json:"total_sessions"`
	PresentSessions    int    `json:"present_sessions"`
	AMPresentSessions  int    `json:"am_present_sessions"`
	PMPresentSessions  int    `json:"pm_present_sessions"`
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

