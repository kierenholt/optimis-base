package dal

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type ScopeMode string

const (
	ScopeAll       ScopeMode = "all"
	ScopeOwnGroups ScopeMode = "own_groups"
)

type DALQueryScope struct {
	Mode           ScopeMode `json:"mode"`
	StaffID        string    `json:"staff_id"`
	AcademicYearID string    `json:"academic_year_id"`
	ReferenceDate  string    `json:"reference_date"`
}

func validateScope(scope DALQueryScope) error {
	if scope.Mode != ScopeAll && scope.Mode != ScopeOwnGroups {
		return fmt.Errorf("invalid or missing DALQueryScope.Mode %q", scope.Mode)
	}
	if scope.Mode == ScopeOwnGroups && strings.TrimSpace(scope.StaffID) == "" {
		return errors.New("missing mandatory StaffID for :own_groups scope")
	}
	if strings.TrimSpace(scope.AcademicYearID) == "" {
		return errors.New("missing mandatory AcademicYearID in DALQueryScope")
	}
	if strings.TrimSpace(scope.ReferenceDate) == "" {
		return errors.New("missing mandatory ReferenceDate in DALQueryScope")
	}
	return nil
}

func (d *CoreDAL) ResolveScopeDefaults(ctx context.Context, scope DALQueryScope) (DALQueryScope, error) {
	out := scope
	if strings.TrimSpace(out.ReferenceDate) == "" {
		var today string
		if err := d.readDB.QueryRowContext(ctx, `SELECT date('now')`).Scan(&today); err != nil {
			return out, fmt.Errorf("failed to resolve current reference date: %w", err)
		}
		out.ReferenceDate = today
	}
	if strings.TrimSpace(out.AcademicYearID) == "" {
		var ayID string
		err := d.readDB.QueryRowContext(
			ctx,
			`SELECT id FROM academic_years WHERE status = 'CURRENT' ORDER BY sequence_number DESC LIMIT 1`,
		).Scan(&ayID)
		if err != nil {
			return out, fmt.Errorf("missing mandatory AcademicYearID and no CURRENT academic_year found: %w", err)
		}
		out.AcademicYearID = ayID
	}
	return out, nil
}

func (d *CoreDAL) ensureGroupInOwnGroups(ctx context.Context, scope DALQueryScope, groupID string) error {
	if err := validateScope(scope); err != nil {
		return err
	}
	if strings.TrimSpace(groupID) == "" {
		return errors.New("missing mandatory parameter: group_id")
	}

	if scope.Mode == ScopeAll {
		var count int
		err := d.readDB.QueryRowContext(
			ctx,
			`SELECT COUNT(*) FROM groups
			 WHERE id = ?
			   AND academic_year_id = ?
			   AND effective_from <= ?
			   AND (effective_to IS NULL OR effective_to >= ?)`,
			groupID, scope.AcademicYearID, scope.ReferenceDate, scope.ReferenceDate,
		).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to verify group %s: %w", groupID, err)
		}
		if count == 0 {
			return fmt.Errorf("group %s not found or inactive on %s", groupID, scope.ReferenceDate)
		}
		return nil
	}

	var count int
	err := d.readDB.QueryRowContext(
		ctx,
		`SELECT COUNT(*)
		 FROM groups g
		 JOIN group_staff_assignments gsa ON gsa.group_id = g.id
		 WHERE g.id = ?
		   AND g.academic_year_id = ?
		   AND g.effective_from <= ?
		   AND (g.effective_to IS NULL OR g.effective_to >= ?)
		   AND gsa.staff_id = ?
		   AND gsa.academic_year_id = ?
		   AND gsa.effective_from <= ?
		   AND (gsa.effective_to IS NULL OR gsa.effective_to >= ?)`,
		groupID,
		scope.AcademicYearID, scope.ReferenceDate, scope.ReferenceDate,
		scope.StaffID,
		scope.AcademicYearID, scope.ReferenceDate, scope.ReferenceDate,
	).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to verify :own_groups assignment for group %s: %w", groupID, err)
	}
	if count == 0 {
		return fmt.Errorf("access denied under own_groups scope: staff %s is not assigned to active group %s on %s", scope.StaffID, groupID, scope.ReferenceDate)
	}
	return nil
}

func (d *CoreDAL) ensurePupilInOwnGroups(ctx context.Context, scope DALQueryScope, pupilID string) error {
	if err := validateScope(scope); err != nil {
		return err
	}
	if strings.TrimSpace(pupilID) == "" {
		return errors.New("missing mandatory parameter: pupil_id")
	}
	if scope.Mode == ScopeAll {
		return nil
	}

	var count int
	err := d.readDB.QueryRowContext(
		ctx,
		`SELECT COUNT(*)
		 FROM group_pupil_memberships gpm
		 JOIN groups g ON g.id = gpm.group_id
		 JOIN group_staff_assignments gsa ON gsa.group_id = g.id
		 WHERE gpm.pupil_id = ?
		   AND gpm.academic_year_id = ?
		   AND gpm.effective_from <= ?
		   AND (gpm.effective_to IS NULL OR gpm.effective_to >= ?)
		   AND g.academic_year_id = ?
		   AND g.effective_from <= ?
		   AND (g.effective_to IS NULL OR g.effective_to >= ?)
		   AND gsa.staff_id = ?
		   AND gsa.academic_year_id = ?
		   AND gsa.effective_from <= ?
		   AND (gsa.effective_to IS NULL OR gsa.effective_to >= ?)`,
		pupilID,
		scope.AcademicYearID, scope.ReferenceDate, scope.ReferenceDate,
		scope.AcademicYearID, scope.ReferenceDate, scope.ReferenceDate,
		scope.StaffID,
		scope.AcademicYearID, scope.ReferenceDate, scope.ReferenceDate,
	).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to verify own_groups pupil membership: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("access denied under own_groups scope: pupil is not in any active group assigned to staff %s on %s", scope.StaffID, scope.ReferenceDate)
	}
	return nil
}
