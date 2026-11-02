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

type UpsertGroupStaffAssignmentInput struct {
	AssignmentID   string `json:"assignment_id,omitempty"`
	GroupID        string `json:"group_id"`
	StaffID        string `json:"staff_id"`
	Role           string `json:"role"`
	AcademicYearID string `json:"academic_year_id"`
	EffectiveFrom  string `json:"effective_from"`
	EffectiveTo    string `json:"effective_to,omitempty"`
}

type UpsertGroupStaffAssignmentResult struct {
	AssignmentID   string `json:"assignment_id"`
	GroupID        string `json:"group_id"`
	StaffID        string `json:"staff_id"`
	Role           string `json:"role"`
	AcademicYearID string `json:"academic_year_id"`
	EffectiveFrom  string `json:"effective_from"`
	EffectiveTo    string `json:"effective_to,omitempty"`
}

func (d *CoreDAL) UpsertGroupStaffAssignmentScoped(
	ctx context.Context,
	scope DALQueryScope,
	in UpsertGroupStaffAssignmentInput,
) (*UpsertGroupStaffAssignmentResult, error) {
	groupID := strings.TrimSpace(in.GroupID)
	staffID := strings.TrimSpace(in.StaffID)
	role := strings.ToUpper(strings.TrimSpace(in.Role))
	if role == "" {
		role = "CO_TEACHER"
	}
	switch role {
	case "LEAD_TEACHER", "CO_TEACHER", "COVER_TEACHER", "FORM_TUTOR", "TEACHING_ASSISTANT":
	default:
		return nil, fmt.Errorf("invalid group_staff_assignments role %q: must be LEAD_TEACHER, CO_TEACHER, COVER_TEACHER, FORM_TUTOR, or TEACHING_ASSISTANT", in.Role)
	}

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
	if staffID == "" {
		return nil, errors.New("missing mandatory parameter: staff_id")
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

	assignmentID := strings.TrimSpace(in.AssignmentID)
	if assignmentID == "" {
		assignmentID = "gsa_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	}

	var effToArg any
	trimmedEffTo := strings.TrimSpace(in.EffectiveTo)
	if trimmedEffTo != "" {
		effToArg = trimmedEffTo
	}

	tx, err := d.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin group staff assignment transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO group_staff_assignments (
			id, group_id, staff_id, role, academic_year_id, effective_from, effective_to
		) VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(group_id, staff_id, role, effective_from) DO UPDATE SET
			academic_year_id = excluded.academic_year_id,
			effective_to = excluded.effective_to`,
		assignmentID, groupID, staffID, role, ayID, effFrom, effToArg,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert group_staff_assignments row: %w", err)
	}

	var persistedID string
	if err := tx.QueryRowContext(
		ctx,
		`SELECT id FROM group_staff_assignments WHERE group_id = ? AND staff_id = ? AND role = ? AND effective_from = ?`,
		groupID, staffID, role, effFrom,
	).Scan(&persistedID); err == nil && strings.TrimSpace(persistedID) != "" {
		assignmentID = persistedID
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit group staff assignment transaction: %w", err)
	}

	return &UpsertGroupStaffAssignmentResult{
		AssignmentID:   assignmentID,
		GroupID:        groupID,
		StaffID:        staffID,
		Role:           role,
		AcademicYearID: ayID,
		EffectiveFrom:  effFrom,
		EffectiveTo:    trimmedEffTo,
	}, nil
}

type RevokeGroupStaffAssignmentInput struct {
	AssignmentID  string `json:"assignment_id,omitempty"`
	GroupID       string `json:"group_id,omitempty"`
	StaffID       string `json:"staff_id,omitempty"`
	Role          string `json:"role,omitempty"`
	EffectiveFrom string `json:"effective_from,omitempty"`
}

func (d *CoreDAL) RevokeGroupStaffAssignmentScoped(
	ctx context.Context,
	scope DALQueryScope,
	in RevokeGroupStaffAssignmentInput,
) (int64, error) {
	if err := validateScope(scope); err != nil {
		return 0, err
	}
	assignmentID := strings.TrimSpace(in.AssignmentID)
	groupID := strings.TrimSpace(in.GroupID)
	staffID := strings.TrimSpace(in.StaffID)
	role := strings.ToUpper(strings.TrimSpace(in.Role))
	effFrom := strings.TrimSpace(in.EffectiveFrom)

	if assignmentID == "" && (groupID == "" || staffID == "") {
		return 0, errors.New("missing mandatory parameter: assignment_id or (group_id and staff_id)")
	}
	if groupID != "" {
		if err := d.ensureGroupInOwnGroups(ctx, scope, groupID); err != nil {
			return 0, err
		}
	}

	tx, err := d.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to begin revoke group staff assignment tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var res sql.Result
	if assignmentID != "" {
		res, err = tx.ExecContext(ctx, `DELETE FROM group_staff_assignments WHERE id = ?`, assignmentID)
	} else if role != "" && effFrom != "" {
		res, err = tx.ExecContext(
			ctx,
			`DELETE FROM group_staff_assignments WHERE group_id = ? AND staff_id = ? AND role = ? AND effective_from = ?`,
			groupID, staffID, role, effFrom,
		)
	} else if effFrom != "" {
		res, err = tx.ExecContext(
			ctx,
			`DELETE FROM group_staff_assignments WHERE group_id = ? AND staff_id = ? AND effective_from = ?`,
			groupID, staffID, effFrom,
		)
	} else {
		res, err = tx.ExecContext(
			ctx,
			`DELETE FROM group_staff_assignments WHERE group_id = ? AND staff_id = ?`,
			groupID, staffID,
		)
	}
	if err != nil {
		return 0, fmt.Errorf("failed to delete group_staff_assignments row: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit revoke group staff assignment tx: %w", err)
	}
	return rowsAffected, nil
}

type RoomSummary struct {
	ID             string `json:"id"`
	Code           string `json:"code"`
	Name           string `json:"name"`
	RoomType       string `json:"room_type"`
	Capacity       int    `json:"capacity"`
	IsActive       bool   `json:"is_active"`
	PhoneExtension string `json:"phone_extension"`
}

func (d *CoreDAL) ListRooms(ctx context.Context) ([]RoomSummary, error) {
	rows, err := d.readDB.QueryContext(
		ctx,
		`SELECT id, code, name, room_type, capacity, is_active, COALESCE(phone_extension, '')
		 FROM rooms
		 ORDER BY code ASC, id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list rooms: %w", err)
	}
	defer rows.Close()

	var result []RoomSummary
	for rows.Next() {
		var r RoomSummary
		var activeInt int
		if err := rows.Scan(&r.ID, &r.Code, &r.Name, &r.RoomType, &r.Capacity, &activeInt, &r.PhoneExtension); err != nil {
			return nil, fmt.Errorf("failed to scan room row: %w", err)
		}
		r.IsActive = activeInt == 1
		result = append(result, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rooms: %w", err)
	}
	return result, nil
}

type CourseSummary struct {
	ID             string `json:"id"`
	AcademicYearID string `json:"academic_year_id"`
	DepartmentID   string `json:"department_id"`
	Code           string `json:"code"`
	Name           string `json:"name"`
	DfESubjectCode string `json:"dfe_subject_code"`
}

func (d *CoreDAL) ListCourses(ctx context.Context, academicYearID string) ([]CourseSummary, error) {
	ayID := strings.TrimSpace(academicYearID)
	var rows *sql.Rows
	var err error
	if ayID != "" {
		rows, err = d.readDB.QueryContext(
			ctx,
			`SELECT id, academic_year_id, COALESCE(department_id, ''), code, name, COALESCE(dfe_subject_code, '')
			 FROM courses
			 WHERE academic_year_id = ?
			 ORDER BY code ASC, id ASC`,
			ayID,
		)
	} else {
		rows, err = d.readDB.QueryContext(
			ctx,
			`SELECT id, academic_year_id, COALESCE(department_id, ''), code, name, COALESCE(dfe_subject_code, '')
			 FROM courses
			 ORDER BY code ASC, id ASC`,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to list courses: %w", err)
	}
	defer rows.Close()

	var result []CourseSummary
	for rows.Next() {
		var c CourseSummary
		if err := rows.Scan(&c.ID, &c.AcademicYearID, &c.DepartmentID, &c.Code, &c.Name, &c.DfESubjectCode); err != nil {
			return nil, fmt.Errorf("failed to scan course row: %w", err)
		}
		result = append(result, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating courses: %w", err)
	}
	return result, nil
}

type DepartmentSummary struct {
	ID                      string `json:"id"`
	Name                    string `json:"name"`
	NameShort               string `json:"name_short"`
	Type                    string `json:"type"`
	HeadOfDepartmentStaffID string `json:"head_of_department_staff_id"`
}

func (d *CoreDAL) ListDepartments(ctx context.Context) ([]DepartmentSummary, error) {
	rows, err := d.readDB.QueryContext(
		ctx,
		`SELECT id, name, name_short, type, COALESCE(head_of_department_staff_id, '')
		 FROM departments
		 ORDER BY name ASC, id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list departments: %w", err)
	}
	defer rows.Close()

	var result []DepartmentSummary
	for rows.Next() {
		var dep DepartmentSummary
		if err := rows.Scan(&dep.ID, &dep.Name, &dep.NameShort, &dep.Type, &dep.HeadOfDepartmentStaffID); err != nil {
			return nil, fmt.Errorf("failed to scan department row: %w", err)
		}
		result = append(result, dep)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating departments: %w", err)
	}
	return result, nil
}

type YearGroupSummary struct {
	ID                string `json:"id"`
	NCCode            string `json:"nc_code"`
	Name              string `json:"name"`
	Phase             string `json:"phase"`
	KeyStage          string `json:"key_stage"`
	SequenceNumber    int    `json:"sequence_number"`
	HeadOfYearStaffID string `json:"head_of_year_staff_id"`
}

func (d *CoreDAL) ListYearGroups(ctx context.Context) ([]YearGroupSummary, error) {
	rows, err := d.readDB.QueryContext(
		ctx,
		`SELECT id, nc_code, name, phase, key_stage, sequence_number, COALESCE(head_of_year_staff_id, '')
		 FROM year_groups
		 ORDER BY sequence_number ASC, id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list year_groups: %w", err)
	}
	defer rows.Close()

	var result []YearGroupSummary
	for rows.Next() {
		var yg YearGroupSummary
		if err := rows.Scan(&yg.ID, &yg.NCCode, &yg.Name, &yg.Phase, &yg.KeyStage, &yg.SequenceNumber, &yg.HeadOfYearStaffID); err != nil {
			return nil, fmt.Errorf("failed to scan year_group row: %w", err)
		}
		result = append(result, yg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating year_groups: %w", err)
	}
	return result, nil
}

type TimetablePeriodSummary struct {
	ID                     string `json:"id"`
	AcademicYearID         string `json:"academic_year_id"`
	CycleWeek              int    `json:"cycle_week"`
	DayOfWeekID            int    `json:"day_of_week_id"`
	PeriodCode             string `json:"period_code"`
	Name                   string `json:"name"`
	StartTime              string `json:"start_time"`
	EndTime                string `json:"end_time"`
	PeriodType             string `json:"period_type"`
	MapsToStatutorySession string `json:"maps_to_statutory_session"`
	SequenceNumber         int    `json:"sequence_number"`
}

func (d *CoreDAL) ListTimetablePeriodsScoped(
	ctx context.Context,
	scope DALQueryScope,
	cycleWeek int,
	dayOfWeekID int,
) ([]TimetablePeriodSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}

	query := `SELECT id, academic_year_id, cycle_week, day_of_week_id, period_code, name, start_time, end_time,
	                 period_type, COALESCE(maps_to_statutory_session, ''), sequence_number
	          FROM timetable_periods
	          WHERE academic_year_id = ?`
	args := []any{scope.AcademicYearID}
	if cycleWeek > 0 {
		query += ` AND cycle_week = ?`
		args = append(args, cycleWeek)
	}
	if dayOfWeekID > 0 {
		query += ` AND day_of_week_id = ?`
		args = append(args, dayOfWeekID)
	}
	query += ` ORDER BY cycle_week ASC, day_of_week_id ASC, sequence_number ASC, id ASC`

	rows, err := d.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list timetable_periods: %w", err)
	}
	defer rows.Close()

	var result []TimetablePeriodSummary
	for rows.Next() {
		var p TimetablePeriodSummary
		if err := rows.Scan(
			&p.ID, &p.AcademicYearID, &p.CycleWeek, &p.DayOfWeekID, &p.PeriodCode, &p.Name,
			&p.StartTime, &p.EndTime, &p.PeriodType, &p.MapsToStatutorySession, &p.SequenceNumber,
		); err != nil {
			return nil, fmt.Errorf("failed to scan timetable_period row: %w", err)
		}
		result = append(result, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating timetable_periods: %w", err)
	}
	return result, nil
}

type TimetableSlotFilter struct {
	GroupID     string
	StaffID     string
	RoomID      string
	CycleWeek   int
	DayOfWeekID int
}

func (d *CoreDAL) ListTimetableSlotsScoped(
	ctx context.Context,
	scope DALQueryScope,
	filter TimetableSlotFilter,
) ([]TimetableSlotSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}

	query := `SELECT ts.id, tp.id, tp.period_code, tp.name, tp.cycle_week, tp.day_of_week_id,
	                 tp.start_time, tp.end_time, tp.period_type, COALESCE(tp.maps_to_statutory_session, ''),
	                 ts.group_id, g.code, COALESCE(ts.staff_id, ''), COALESCE(ts.room_id, '')
	          FROM timetable_slots ts
	          JOIN timetable_periods tp ON tp.id = ts.timetable_period_id
	          JOIN groups g ON g.id = ts.group_id
	          WHERE tp.academic_year_id = ?
	            AND ts.effective_from <= ?
	            AND (ts.effective_to IS NULL OR ts.effective_to >= ?)`
	args := []any{scope.AcademicYearID, scope.ReferenceDate, scope.ReferenceDate}

	if scope.Mode == ScopeOwnGroups {
		query += ` AND EXISTS (
			SELECT 1 FROM group_staff_assignments gsa
			WHERE gsa.group_id = ts.group_id
			  AND gsa.staff_id = ?
			  AND gsa.academic_year_id = ?
			  AND gsa.effective_from <= ?
			  AND (gsa.effective_to IS NULL OR gsa.effective_to >= ?)
		)`
		args = append(args, scope.StaffID, scope.AcademicYearID, scope.ReferenceDate, scope.ReferenceDate)
	}

	if gID := strings.TrimSpace(filter.GroupID); gID != "" {
		query += ` AND ts.group_id = ?`
		args = append(args, gID)
	}
	if sID := strings.TrimSpace(filter.StaffID); sID != "" {
		query += ` AND ts.staff_id = ?`
		args = append(args, sID)
	}
	if rID := strings.TrimSpace(filter.RoomID); rID != "" {
		query += ` AND ts.room_id = ?`
		args = append(args, rID)
	}
	if filter.CycleWeek > 0 {
		query += ` AND tp.cycle_week = ?`
		args = append(args, filter.CycleWeek)
	}
	if filter.DayOfWeekID > 0 {
		query += ` AND tp.day_of_week_id = ?`
		args = append(args, filter.DayOfWeekID)
	}
	query += ` ORDER BY tp.cycle_week ASC, tp.day_of_week_id ASC, tp.sequence_number ASC, ts.id ASC`

	rows, err := d.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list timetable_slots: %w", err)
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
			return nil, fmt.Errorf("failed to scan timetable_slot row: %w", err)
		}
		result = append(result, slot)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating timetable_slots: %w", err)
	}
	return result, nil
}

type UpsertGroupInput struct {
	GroupID         string `json:"group_id,omitempty"`
	AcademicYearID  string `json:"academic_year_id"`
	GroupType       string `json:"group_type"`
	Code            string `json:"code"`
	Name            string `json:"name"`
	YearGroupID     string `json:"year_group_id,omitempty"`
	CourseID        string `json:"course_id,omitempty"`
	DefaultRoomID   string `json:"default_room_id,omitempty"`
	TakesAttendance *bool  `json:"takes_attendance,omitempty"`
	EffectiveFrom   string `json:"effective_from"`
	EffectiveTo     string `json:"effective_to,omitempty"`
}

func (d *CoreDAL) UpsertGroupScoped(
	ctx context.Context,
	scope DALQueryScope,
	in UpsertGroupInput,
) (*GroupSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	if scope.Mode != ScopeAll {
		return nil, errors.New("access denied: upserting groups requires dal.groups.write:all scope")
	}

	groupID := strings.TrimSpace(in.GroupID)
	if groupID == "" {
		groupID = "grp_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	}
	ayID := strings.TrimSpace(in.AcademicYearID)
	if ayID == "" {
		ayID = strings.TrimSpace(scope.AcademicYearID)
	}
	groupType := strings.ToUpper(strings.TrimSpace(in.GroupType))
	if groupType == "" {
		groupType = "TEACHING_CLASS"
	}
	switch groupType {
	case "REGISTRATION_FORM", "TEACHING_CLASS", "INTERVENTION", "CUSTOM":
	default:
		return nil, fmt.Errorf("invalid group_type %q: must be REGISTRATION_FORM, TEACHING_CLASS, INTERVENTION, or CUSTOM", in.GroupType)
	}

	code := strings.TrimSpace(in.Code)
	name := strings.TrimSpace(in.Name)
	if code == "" {
		return nil, errors.New("missing mandatory parameter: code")
	}
	if name == "" {
		return nil, errors.New("missing mandatory parameter: name")
	}
	effFrom := strings.TrimSpace(in.EffectiveFrom)
	if effFrom == "" {
		effFrom = strings.TrimSpace(scope.ReferenceDate)
	}
	if ayID == "" || effFrom == "" {
		return nil, errors.New("missing mandatory parameter: academic_year_id or effective_from")
	}

	takesAtt := true
	if in.TakesAttendance != nil {
		takesAtt = *in.TakesAttendance
	}
	takesAttInt := 0
	if takesAtt {
		takesAttInt = 1
	}

	var ygArg, courseArg, roomArg, effToArg any
	if yg := strings.TrimSpace(in.YearGroupID); yg != "" {
		ygArg = yg
	}
	if c := strings.TrimSpace(in.CourseID); c != "" {
		courseArg = c
	}
	trimmedRoom := strings.TrimSpace(in.DefaultRoomID)
	if trimmedRoom != "" {
		roomArg = trimmedRoom
	}
	trimmedEffTo := strings.TrimSpace(in.EffectiveTo)
	if trimmedEffTo != "" {
		effToArg = trimmedEffTo
	}

	tx, err := d.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin upsert group tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO groups (
			id, academic_year_id, group_type, code, name, year_group_id, course_id,
			default_room_id, takes_attendance, effective_from, effective_to
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(academic_year_id, code) DO UPDATE SET
			group_type = excluded.group_type,
			name = excluded.name,
			year_group_id = COALESCE(excluded.year_group_id, groups.year_group_id),
			course_id = COALESCE(excluded.course_id, groups.course_id),
			default_room_id = COALESCE(excluded.default_room_id, groups.default_room_id),
			takes_attendance = excluded.takes_attendance,
			effective_from = excluded.effective_from,
			effective_to = excluded.effective_to`,
		groupID, ayID, groupType, code, name, ygArg, courseArg, roomArg, takesAttInt, effFrom, effToArg,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert group row: %w", err)
	}

	var persistedID string
	if err := tx.QueryRowContext(
		ctx,
		`SELECT id FROM groups WHERE academic_year_id = ? AND code = ?`,
		ayID, code,
	).Scan(&persistedID); err == nil && strings.TrimSpace(persistedID) != "" {
		groupID = persistedID
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit upsert group tx: %w", err)
	}

	var effToPtr *string
	if trimmedEffTo != "" {
		effToPtr = &trimmedEffTo
	}

	return &GroupSummary{
		ID:              groupID,
		AcademicYearID:  ayID,
		GroupType:       groupType,
		Code:            code,
		Name:            name,
		DefaultRoomID:   trimmedRoom,
		TakesAttendance: takesAtt,
		EffectiveFrom:   effFrom,
		EffectiveTo:     effToPtr,
	}, nil
}

type UpsertTimetablePeriodInput struct {
	PeriodID               string `json:"period_id,omitempty"`
	AcademicYearID         string `json:"academic_year_id"`
	CycleWeek              int    `json:"cycle_week"`
	DayOfWeekID            int    `json:"day_of_week_id"`
	PeriodCode             string `json:"period_code"`
	Name                   string `json:"name"`
	StartTime              string `json:"start_time"`
	EndTime                string `json:"end_time"`
	PeriodType             string `json:"period_type"`
	MapsToStatutorySession string `json:"maps_to_statutory_session,omitempty"`
	SequenceNumber         int    `json:"sequence_number"`
}

func (d *CoreDAL) UpsertTimetablePeriodScoped(
	ctx context.Context,
	scope DALQueryScope,
	in UpsertTimetablePeriodInput,
) (*TimetablePeriodSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	if scope.Mode != ScopeAll {
		return nil, errors.New("access denied: upserting timetable periods requires dal.groups.write:all scope")
	}

	periodID := strings.TrimSpace(in.PeriodID)
	if periodID == "" {
		periodID = "ttp_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	}
	ayID := strings.TrimSpace(in.AcademicYearID)
	if ayID == "" {
		ayID = strings.TrimSpace(scope.AcademicYearID)
	}
	cycleWeek := in.CycleWeek
	if cycleWeek == 0 {
		cycleWeek = 1
	}
	if cycleWeek != 1 && cycleWeek != 2 {
		return nil, fmt.Errorf("invalid cycle_week %d: must be 1 or 2", cycleWeek)
	}
	dayOfWeekID := in.DayOfWeekID
	if dayOfWeekID < 1 || dayOfWeekID > 7 {
		return nil, fmt.Errorf("invalid day_of_week_id %d: must be 1..7", dayOfWeekID)
	}
	periodCode := strings.TrimSpace(in.PeriodCode)
	name := strings.TrimSpace(in.Name)
	startTime := strings.TrimSpace(in.StartTime)
	endTime := strings.TrimSpace(in.EndTime)
	if periodCode == "" || name == "" || startTime == "" || endTime == "" {
		return nil, errors.New("missing mandatory parameter: period_code, name, start_time, and end_time are required")
	}
	periodType := strings.ToUpper(strings.TrimSpace(in.PeriodType))
	if periodType == "" {
		periodType = "LESSON"
	}
	switch periodType {
	case "STATUTORY_AM_REG", "STATUTORY_PM_REG", "LESSON", "BREAK", "PASTORAL":
	default:
		return nil, fmt.Errorf("invalid period_type %q", in.PeriodType)
	}

	var statSessArg any
	statSess := strings.ToUpper(strings.TrimSpace(in.MapsToStatutorySession))
	if statSess != "" {
		if statSess != "AM" && statSess != "PM" {
			return nil, fmt.Errorf("invalid maps_to_statutory_session %q: must be AM or PM", in.MapsToStatutorySession)
		}
		statSessArg = statSess
	}

	seqNum := in.SequenceNumber
	if seqNum <= 0 {
		seqNum = 1
	}

	tx, err := d.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin upsert timetable_period tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO timetable_periods (
			id, academic_year_id, cycle_week, day_of_week_id, period_code, name,
			start_time, end_time, period_type, maps_to_statutory_session, sequence_number
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(academic_year_id, cycle_week, day_of_week_id, period_code) DO UPDATE SET
			name = excluded.name,
			start_time = excluded.start_time,
			end_time = excluded.end_time,
			period_type = excluded.period_type,
			maps_to_statutory_session = excluded.maps_to_statutory_session,
			sequence_number = excluded.sequence_number`,
		periodID, ayID, cycleWeek, dayOfWeekID, periodCode, name, startTime, endTime, periodType, statSessArg, seqNum,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert timetable_period row: %w", err)
	}

	var persistedID string
	if err := tx.QueryRowContext(
		ctx,
		`SELECT id FROM timetable_periods WHERE academic_year_id = ? AND cycle_week = ? AND day_of_week_id = ? AND period_code = ?`,
		ayID, cycleWeek, dayOfWeekID, periodCode,
	).Scan(&persistedID); err == nil && strings.TrimSpace(persistedID) != "" {
		periodID = persistedID
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit upsert timetable_period tx: %w", err)
	}

	return &TimetablePeriodSummary{
		ID:                     periodID,
		AcademicYearID:         ayID,
		CycleWeek:              cycleWeek,
		DayOfWeekID:            dayOfWeekID,
		PeriodCode:             periodCode,
		Name:                   name,
		StartTime:              startTime,
		EndTime:                endTime,
		PeriodType:             periodType,
		MapsToStatutorySession: statSess,
		SequenceNumber:         seqNum,
	}, nil
}

type UpsertTimetableSlotInput struct {
	SlotID            string `json:"slot_id,omitempty"`
	TimetablePeriodID string `json:"timetable_period_id"`
	GroupID           string `json:"group_id"`
	StaffID           string `json:"staff_id,omitempty"`
	RoomID            string `json:"room_id,omitempty"`
	EffectiveFrom     string `json:"effective_from"`
	EffectiveTo       string `json:"effective_to,omitempty"`
}

type UpsertTimetableSlotResult struct {
	SlotID            string `json:"slot_id"`
	TimetablePeriodID string `json:"timetable_period_id"`
	GroupID           string `json:"group_id"`
	StaffID           string `json:"staff_id"`
	RoomID            string `json:"room_id"`
	EffectiveFrom     string `json:"effective_from"`
	EffectiveTo       string `json:"effective_to,omitempty"`
}

func (d *CoreDAL) UpsertTimetableSlotScoped(
	ctx context.Context,
	scope DALQueryScope,
	in UpsertTimetableSlotInput,
) (*UpsertTimetableSlotResult, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}

	periodID := strings.TrimSpace(in.TimetablePeriodID)
	groupID := strings.TrimSpace(in.GroupID)
	effFrom := strings.TrimSpace(in.EffectiveFrom)
	if effFrom == "" {
		effFrom = strings.TrimSpace(scope.ReferenceDate)
	}
	if periodID == "" || groupID == "" || effFrom == "" {
		return nil, errors.New("missing mandatory parameter: timetable_period_id, group_id, and effective_from are required")
	}
	if err := d.ensureGroupInOwnGroups(ctx, scope, groupID); err != nil {
		return nil, err
	}

	slotID := strings.TrimSpace(in.SlotID)
	if slotID == "" {
		slotID = "tts_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	}

	var staffArg, roomArg, effToArg any
	trimmedStaff := strings.TrimSpace(in.StaffID)
	if trimmedStaff != "" {
		staffArg = trimmedStaff
	}
	trimmedRoom := strings.TrimSpace(in.RoomID)
	if trimmedRoom != "" {
		roomArg = trimmedRoom
	}
	trimmedEffTo := strings.TrimSpace(in.EffectiveTo)
	if trimmedEffTo != "" {
		effToArg = trimmedEffTo
	}

	tx, err := d.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin upsert timetable_slot tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO timetable_slots (
			id, timetable_period_id, group_id, staff_id, room_id, effective_from, effective_to
		) VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(timetable_period_id, group_id, effective_from) DO UPDATE SET
			staff_id = excluded.staff_id,
			room_id = excluded.room_id,
			effective_to = excluded.effective_to`,
		slotID, periodID, groupID, staffArg, roomArg, effFrom, effToArg,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert timetable_slot row: %w", err)
	}

	var persistedID string
	if err := tx.QueryRowContext(
		ctx,
		`SELECT id FROM timetable_slots WHERE timetable_period_id = ? AND group_id = ? AND effective_from = ?`,
		periodID, groupID, effFrom,
	).Scan(&persistedID); err == nil && strings.TrimSpace(persistedID) != "" {
		slotID = persistedID
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit upsert timetable_slot tx: %w", err)
	}

	return &UpsertTimetableSlotResult{
		SlotID:            slotID,
		TimetablePeriodID: periodID,
		GroupID:           groupID,
		StaffID:           trimmedStaff,
		RoomID:            trimmedRoom,
		EffectiveFrom:     effFrom,
		EffectiveTo:       trimmedEffTo,
	}, nil
}

type HouseSummary struct {
	ID             string `json:"id"`
	HouseID        string `json:"house_id"`
	Code           string `json:"code"`
	NameShort      string `json:"name_short"`
	Name           string `json:"name"`
	ColorHex       string `json:"color_hex"`
	SequenceNumber int    `json:"sequence_number,omitempty"`
}

func (d *CoreDAL) ListHouses(ctx context.Context) ([]HouseSummary, error) {
	rows, err := d.readDB.QueryContext(
		ctx,
		`SELECT id, name, name_short, color_hex
		 FROM houses
		 ORDER BY name ASC, id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list houses: %w", err)
	}
	defer rows.Close()

	result := make([]HouseSummary, 0)
	seq := 1
	for rows.Next() {
		var h HouseSummary
		if err := rows.Scan(&h.ID, &h.Name, &h.NameShort, &h.ColorHex); err != nil {
			return nil, fmt.Errorf("failed to scan house row: %w", err)
		}
		h.HouseID = h.ID
		h.Code = h.NameShort
		h.SequenceNumber = seq
		seq++
		result = append(result, h)
	}
	return result, rows.Err()
}

func (d *CoreDAL) UpsertHouseScoped(
	ctx context.Context,
	scope DALQueryScope,
	in HouseSummary,
) (*HouseSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	if scope.Mode != ScopeAll {
		return nil, errors.New("access denied: upserting houses requires dal.groups.write:all scope")
	}

	houseID := strings.TrimSpace(in.HouseID)
	if houseID == "" {
		houseID = strings.TrimSpace(in.ID)
	}
	if houseID == "" {
		houseID = "hse_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	}
	name := strings.TrimSpace(in.Name)
	nameShort := strings.ToUpper(strings.TrimSpace(in.NameShort))
	if nameShort == "" {
		nameShort = strings.ToUpper(strings.TrimSpace(in.Code))
	}
	if nameShort == "" && len(name) >= 3 {
		nameShort = strings.ToUpper(name[:3])
	}
	if name == "" || nameShort == "" {
		return nil, errors.New("missing mandatory parameter: name and code/name_short are required")
	}
	colorHex := strings.TrimSpace(in.ColorHex)
	if colorHex == "" {
		colorHex = "#1E40AF"
	}

	if _, err := d.writeDB.ExecContext(
		ctx,
		`INSERT INTO houses (id, name, name_short, color_hex)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		     name = excluded.name,
		     name_short = excluded.name_short,
		     color_hex = excluded.color_hex`,
		houseID, name, nameShort, colorHex,
	); err != nil {
		return nil, fmt.Errorf("failed to upsert houses row: %w", err)
	}

	seq := in.SequenceNumber
	if seq <= 0 {
		seq = 1
	}
	return &HouseSummary{
		ID:             houseID,
		HouseID:        houseID,
		Code:           nameShort,
		NameShort:      nameShort,
		Name:           name,
		ColorHex:       colorHex,
		SequenceNumber: seq,
	}, nil
}

func (d *CoreDAL) UpsertRoomScoped(
	ctx context.Context,
	scope DALQueryScope,
	in RoomSummary,
) (*RoomSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	if scope.Mode != ScopeAll {
		return nil, errors.New("access denied: upserting rooms requires dal.groups.write:all scope")
	}

	roomID := strings.TrimSpace(in.ID)
	code := strings.ToUpper(strings.TrimSpace(in.Code))
	name := strings.TrimSpace(in.Name)
	if code == "" || name == "" {
		return nil, errors.New("missing mandatory parameter: code and name are required")
	}
	if roomID == "" {
		roomID = "rm_" + strings.ToLower(code)
	}
	roomType := strings.ToUpper(strings.TrimSpace(in.RoomType))
	if roomType == "" {
		roomType = "CLASSROOM"
	}
	capVal := in.Capacity
	if capVal <= 0 {
		capVal = 30
	}

	if _, err := d.writeDB.ExecContext(
		ctx,
		`INSERT INTO rooms (id, code, name, room_type, capacity, is_active, phone_extension)
		 VALUES (?, ?, ?, ?, ?, 1, ?)
		 ON CONFLICT(id) DO UPDATE SET
		     code = excluded.code,
		     name = excluded.name,
		     room_type = excluded.room_type,
		     capacity = excluded.capacity,
		     phone_extension = excluded.phone_extension`,
		roomID, code, name, roomType, capVal, strings.TrimSpace(in.PhoneExtension),
	); err != nil {
		return nil, fmt.Errorf("failed to upsert rooms row: %w", err)
	}

	return &RoomSummary{
		ID:             roomID,
		Code:           code,
		Name:           name,
		RoomType:       roomType,
		Capacity:       capVal,
		IsActive:       true,
		PhoneExtension: strings.TrimSpace(in.PhoneExtension),
	}, nil
}

func (d *CoreDAL) UpsertDepartmentScoped(
	ctx context.Context,
	scope DALQueryScope,
	in DepartmentSummary,
) (*DepartmentSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	if scope.Mode != ScopeAll {
		return nil, errors.New("access denied: upserting departments requires dal.groups.write:all scope")
	}

	depID := strings.TrimSpace(in.ID)
	name := strings.TrimSpace(in.Name)
	nameShort := strings.ToUpper(strings.TrimSpace(in.NameShort))
	if nameShort == "" && len(name) >= 3 {
		nameShort = strings.ToUpper(name[:3])
	}
	if name == "" || nameShort == "" {
		return nil, errors.New("missing mandatory parameter: name and name_short/code are required")
	}
	if depID == "" {
		depID = "dep_" + strings.ToLower(nameShort)
	}
	depType := strings.ToUpper(strings.TrimSpace(in.Type))
	switch depType {
	case "", "ACADEMIC", "LEARNING_AREA":
		depType = "LEARNING_AREA"
	case "ADMINISTRATION", "PASTORAL":
	default:
		return nil, fmt.Errorf("invalid department type %q", in.Type)
	}
	var hodArg any
	if strings.TrimSpace(in.HeadOfDepartmentStaffID) != "" {
		hodArg = strings.TrimSpace(in.HeadOfDepartmentStaffID)
	}

	if _, err := d.writeDB.ExecContext(
		ctx,
		`INSERT INTO departments (id, name, name_short, type, head_of_department_staff_id)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		     name = excluded.name,
		     name_short = excluded.name_short,
		     type = excluded.type,
		     head_of_department_staff_id = COALESCE(excluded.head_of_department_staff_id, departments.head_of_department_staff_id)`,
		depID, name, nameShort, depType, hodArg,
	); err != nil {
		return nil, fmt.Errorf("failed to upsert departments row: %w", err)
	}

	return &DepartmentSummary{
		ID:                      depID,
		Name:                    name,
		NameShort:               nameShort,
		Type:                    depType,
		HeadOfDepartmentStaffID: strings.TrimSpace(in.HeadOfDepartmentStaffID),
	}, nil
}

func (d *CoreDAL) UpsertCourseScoped(
	ctx context.Context,
	scope DALQueryScope,
	in CourseSummary,
) (*CourseSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	if scope.Mode != ScopeAll {
		return nil, errors.New("access denied: upserting courses requires dal.groups.write:all scope")
	}

	courseID := strings.TrimSpace(in.ID)
	ayID := strings.TrimSpace(in.AcademicYearID)
	if ayID == "" {
		ayID = scope.AcademicYearID
	}
	code := strings.ToUpper(strings.TrimSpace(in.Code))
	name := strings.TrimSpace(in.Name)
	if code == "" || name == "" {
		return nil, errors.New("missing mandatory parameter: code and name are required")
	}
	if courseID == "" {
		courseID = "crs_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	}
	var depArg any
	if strings.TrimSpace(in.DepartmentID) != "" {
		depArg = strings.TrimSpace(in.DepartmentID)
	}

	if _, err := d.writeDB.ExecContext(
		ctx,
		`INSERT INTO courses (id, academic_year_id, department_id, code, name, dfe_subject_code)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(academic_year_id, code) DO UPDATE SET
		     department_id = COALESCE(excluded.department_id, courses.department_id),
		     name = excluded.name,
		     dfe_subject_code = excluded.dfe_subject_code`,
		courseID, ayID, depArg, code, name, strings.TrimSpace(in.DfESubjectCode),
	); err != nil {
		return nil, fmt.Errorf("failed to upsert courses row: %w", err)
	}

	var persistedID string
	if err := d.readDB.QueryRowContext(ctx, `SELECT id FROM courses WHERE academic_year_id = ? AND code = ?`, ayID, code).Scan(&persistedID); err == nil && persistedID != "" {
		courseID = persistedID
	}

	return &CourseSummary{
		ID:             courseID,
		AcademicYearID: ayID,
		DepartmentID:   strings.TrimSpace(in.DepartmentID),
		Code:           code,
		Name:           name,
		DfESubjectCode: strings.TrimSpace(in.DfESubjectCode),
	}, nil
}

func (d *CoreDAL) UpsertYearGroupScoped(
	ctx context.Context,
	scope DALQueryScope,
	in YearGroupSummary,
) (*YearGroupSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	if scope.Mode != ScopeAll {
		return nil, errors.New("access denied: upserting year groups requires dal.groups.write:all scope")
	}

	ygID := strings.TrimSpace(in.ID)
	ncCode := strings.ToUpper(strings.TrimSpace(in.NCCode))
	name := strings.TrimSpace(in.Name)
	if ncCode == "" || name == "" {
		return nil, errors.New("missing mandatory parameter: nc_code and name are required")
	}
	if ygID == "" {
		ygID = "yg_" + strings.ToLower(ncCode)
	}
	phase := strings.ToUpper(strings.TrimSpace(in.Phase))
	if phase == "" {
		phase = "SECONDARY"
	}
	keyStage := strings.ToUpper(strings.TrimSpace(in.KeyStage))
	if keyStage == "" {
		keyStage = "KS3"
	}
	seq := in.SequenceNumber
	if seq <= 0 {
		_ = d.readDB.QueryRowContext(ctx, `SELECT COALESCE(MAX(sequence_number), 0) + 1 FROM year_groups`).Scan(&seq)
		if seq <= 0 {
			seq = 1
		}
	}
	var hoyArg any
	if strings.TrimSpace(in.HeadOfYearStaffID) != "" {
		hoyArg = strings.TrimSpace(in.HeadOfYearStaffID)
	}

	if _, err := d.writeDB.ExecContext(
		ctx,
		`INSERT INTO year_groups (id, nc_code, name, phase, key_stage, sequence_number, head_of_year_staff_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		     nc_code = excluded.nc_code,
		     name = excluded.name,
		     phase = excluded.phase,
		     key_stage = excluded.key_stage,
		     sequence_number = excluded.sequence_number,
		     head_of_year_staff_id = COALESCE(excluded.head_of_year_staff_id, year_groups.head_of_year_staff_id)`,
		ygID, ncCode, name, phase, keyStage, seq, hoyArg,
	); err != nil {
		return nil, fmt.Errorf("failed to upsert year_groups row: %w", err)
	}

	return &YearGroupSummary{
		ID:                ygID,
		NCCode:            ncCode,
		Name:              name,
		Phase:             phase,
		KeyStage:          keyStage,
		SequenceNumber:    seq,
		HeadOfYearStaffID: strings.TrimSpace(in.HeadOfYearStaffID),
	}, nil
}

func (d *CoreDAL) GetPupilTimetableSlotsScoped(
	ctx context.Context,
	scope DALQueryScope,
	pupilID string,
	cycleWeek int,
	dayOfWeekID int,
) ([]TimetableSlotSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	trimmedPupil := strings.TrimSpace(pupilID)
	if trimmedPupil == "" {
		return nil, errors.New("missing mandatory parameter: pupil_id")
	}
	if scope.Mode == ScopeOwnGroups {
		if err := d.ensurePupilInOwnGroups(ctx, scope, trimmedPupil); err != nil {
			return nil, err
		}
	}

	query := `
		SELECT ts.id, tp.id, tp.period_code, tp.name, tp.cycle_week, tp.day_of_week_id,
		       tp.start_time, tp.end_time, tp.period_type, COALESCE(tp.maps_to_statutory_session, ''),
		       ts.group_id, g.code, COALESCE(ts.staff_id, ''), COALESCE(ts.room_id, '')
		FROM group_pupil_memberships gpm
		JOIN timetable_slots ts ON ts.group_id = gpm.group_id
		JOIN timetable_periods tp ON tp.id = ts.timetable_period_id
		JOIN groups g ON g.id = ts.group_id
		WHERE gpm.pupil_id = ?
		  AND gpm.academic_year_id = ?
		  AND gpm.effective_from <= ?
		  AND (gpm.effective_to IS NULL OR gpm.effective_to >= ?)
		  AND tp.academic_year_id = ?
		  AND ts.effective_from <= ?
		  AND (ts.effective_to IS NULL OR ts.effective_to >= ?)`
	args := []any{
		trimmedPupil,
		scope.AcademicYearID, scope.ReferenceDate, scope.ReferenceDate,
		scope.AcademicYearID, scope.ReferenceDate, scope.ReferenceDate,
	}
	if cycleWeek > 0 {
		query += ` AND tp.cycle_week = ?`
		args = append(args, cycleWeek)
	}
	if dayOfWeekID > 0 {
		query += ` AND tp.day_of_week_id = ?`
		args = append(args, dayOfWeekID)
	}
	query += ` ORDER BY tp.cycle_week ASC, tp.day_of_week_id ASC, tp.sequence_number ASC, ts.id ASC`

	rows, err := d.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query pupil timetable slots: %w", err)
	}
	defer rows.Close()

	result := make([]TimetableSlotSummary, 0)
	for rows.Next() {
		var slot TimetableSlotSummary
		if err := rows.Scan(
			&slot.SlotID, &slot.PeriodID, &slot.PeriodCode, &slot.PeriodName,
			&slot.CycleWeek, &slot.DayOfWeekID, &slot.StartTime, &slot.EndTime,
			&slot.PeriodType, &slot.MapsToStatutorySession,
			&slot.GroupID, &slot.GroupCode, &slot.StaffID, &slot.RoomID,
		); err != nil {
			return nil, fmt.Errorf("failed to scan pupil timetable slot: %w", err)
		}
		result = append(result, slot)
	}
	return result, rows.Err()
}

type GroupPupilMembershipSummary struct {
	ID             string `json:"id"`
	MembershipID   string `json:"membership_id"`
	GroupID        string `json:"group_id"`
	GroupCode      string `json:"group_code"`
	GroupName      string `json:"group_name"`
	GroupType      string `json:"group_type"`
	PupilID        string `json:"pupil_id"`
	AcademicYearID string `json:"academic_year_id"`
	EffectiveFrom  string `json:"effective_from"`
	EffectiveTo    string `json:"effective_to,omitempty"`
}

func (d *CoreDAL) ListGroupPupilMembershipsScoped(
	ctx context.Context,
	scope DALQueryScope,
	pupilID string,
	groupID string,
) ([]GroupPupilMembershipSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	trimmedPupil := strings.TrimSpace(pupilID)
	trimmedGroup := strings.TrimSpace(groupID)
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
		SELECT gpm.id, gpm.group_id, g.code, g.name, g.group_type,
		       gpm.pupil_id, gpm.academic_year_id, gpm.effective_from, COALESCE(gpm.effective_to, '')
		FROM group_pupil_memberships gpm
		JOIN groups g ON g.id = gpm.group_id
		WHERE gpm.academic_year_id = ?`
	args := []any{scope.AcademicYearID}
	if trimmedPupil != "" {
		query += ` AND gpm.pupil_id = ?`
		args = append(args, trimmedPupil)
	}
	if trimmedGroup != "" {
		query += ` AND gpm.group_id = ?`
		args = append(args, trimmedGroup)
	}
	if scope.Mode == ScopeOwnGroups && trimmedGroup == "" {
		query += ` AND EXISTS (
			SELECT 1 FROM group_staff_assignments gsa
			WHERE gsa.group_id = gpm.group_id
			  AND gsa.staff_id = ?
			  AND gsa.academic_year_id = ?
			  AND gsa.effective_from <= ?
			  AND (gsa.effective_to IS NULL OR gsa.effective_to >= ?)
		)`
		args = append(args, scope.StaffID, scope.AcademicYearID, scope.ReferenceDate, scope.ReferenceDate)
	}
	query += ` ORDER BY g.code ASC, gpm.pupil_id ASC, gpm.effective_from DESC`

	rows, err := d.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list group_pupil_memberships: %w", err)
	}
	defer rows.Close()

	result := make([]GroupPupilMembershipSummary, 0)
	for rows.Next() {
		var r GroupPupilMembershipSummary
		if err := rows.Scan(
			&r.ID, &r.GroupID, &r.GroupCode, &r.GroupName, &r.GroupType,
			&r.PupilID, &r.AcademicYearID, &r.EffectiveFrom, &r.EffectiveTo,
		); err != nil {
			return nil, fmt.Errorf("failed to scan group_pupil_memberships row: %w", err)
		}
		r.MembershipID = r.ID
		result = append(result, r)
	}
	return result, rows.Err()
}

