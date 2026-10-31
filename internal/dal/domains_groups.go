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
