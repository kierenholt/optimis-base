package dal

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type DashboardWidgetItem struct {
	PlugID   string `json:"plug_id"`
	WidgetID string `json:"widget_id"`
	Col      int    `json:"col,omitempty"`
	Row      int    `json:"row,omitempty"`
}

func ValidateAndNormalizeDashboardWidgetItems(items []DashboardWidgetItem) ([]DashboardWidgetItem, error) {
	if items == nil {
		return []DashboardWidgetItem{}, nil
	}
	seen := make(map[string]struct{}, len(items))
	normalized := make([]DashboardWidgetItem, 0, len(items))
	for idx, it := range items {
		plugID := strings.TrimSpace(it.PlugID)
		widgetID := strings.TrimSpace(it.WidgetID)
		if plugID == "" || widgetID == "" {
			return nil, fmt.Errorf("INVALID_DASHBOARD_WIDGET_ITEM: item[%d] is missing mandatory plug_id or widget_id", idx)
		}
		key := plugID + ":" + widgetID
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		col := 0
		if it.Col > 0 {
			col = it.Col
		}
		row := 0
		if it.Row > 0 {
			row = it.Row
		}
		normalized = append(normalized, DashboardWidgetItem{
			PlugID:   plugID,
			WidgetID: widgetID,
			Col:      col,
			Row:      row,
		})
	}
	return normalized, nil
}

func decodeDashboardWidgetsJSON(raw string) ([]DashboardWidgetItem, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return []DashboardWidgetItem{}, nil
	}
	var items []DashboardWidgetItem
	if err := json.Unmarshal([]byte(trimmed), &items); err != nil {
		return nil, fmt.Errorf("failed to parse dashboard widgets_json: %w", err)
	}
	return ValidateAndNormalizeDashboardWidgetItems(items)
}

const (
	DefaultDashboardColumnCount = 4
	MinDashboardColumnCount     = 2
	MaxDashboardColumnCount     = 8
)

func ValidateAndNormalizeDashboardColumnCount(columnCount int) (int, error) {
	if columnCount == 0 {
		return DefaultDashboardColumnCount, nil
	}
	if columnCount < MinDashboardColumnCount || columnCount > MaxDashboardColumnCount {
		return 0, fmt.Errorf(
			"INVALID_DASHBOARD_COLUMN_COUNT: column_count must be between %d and %d (got %d)",
			MinDashboardColumnCount,
			MaxDashboardColumnCount,
			columnCount,
		)
	}
	return columnCount, nil
}

func (d *CoreDAL) ensureDashboardLayoutsColumnCount(ctx context.Context) {
	var count int
	err := d.readDB.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM pragma_table_info('dashboard_layouts') WHERE name = 'column_count'`,
	).Scan(&count)
	if err == nil && count == 0 {
		_, _ = d.writeDB.ExecContext(
			ctx,
			`ALTER TABLE dashboard_layouts ADD COLUMN column_count INTEGER NOT NULL DEFAULT 4`,
		)
	}
}

func (d *CoreDAL) GetStaffDashboardLayout(ctx context.Context, staffID string) ([]DashboardWidgetItem, bool, error) {
	items, _, found, err := d.GetStaffDashboardLayoutWithColumns(ctx, staffID)
	return items, found, err
}

func (d *CoreDAL) GetStaffDashboardLayoutWithColumns(ctx context.Context, staffID string) ([]DashboardWidgetItem, int, bool, error) {
	trimmedStaff := strings.TrimSpace(staffID)
	if trimmedStaff == "" {
		return nil, DefaultDashboardColumnCount, false, errors.New("missing mandatory parameter: staff_id")
	}

	d.ensureDashboardLayoutsColumnCount(ctx)

	var rawJSON string
	var colCount sql.NullInt64
	err := d.readDB.QueryRowContext(
		ctx,
		`SELECT widgets_json, column_count FROM dashboard_layouts WHERE scope_type = 'STAFF' AND scope_target = ?`,
		trimmedStaff,
	).Scan(&rawJSON, &colCount)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, DefaultDashboardColumnCount, false, nil
		}
		return nil, DefaultDashboardColumnCount, false, fmt.Errorf("failed to query staff dashboard layout for %s: %w", trimmedStaff, err)
	}

	items, decErr := decodeDashboardWidgetsJSON(rawJSON)
	if decErr != nil {
		return nil, DefaultDashboardColumnCount, false, decErr
	}
	resolvedCols := DefaultDashboardColumnCount
	if colCount.Valid && int(colCount.Int64) >= MinDashboardColumnCount && int(colCount.Int64) <= MaxDashboardColumnCount {
		resolvedCols = int(colCount.Int64)
	}
	return items, resolvedCols, true, nil
}

func (d *CoreDAL) SaveStaffDashboardLayout(
	ctx context.Context,
	staffID string,
	items []DashboardWidgetItem,
	updatedByStaffID string,
) ([]DashboardWidgetItem, error) {
	saved, _, err := d.SaveStaffDashboardLayoutWithColumns(ctx, staffID, items, 0, updatedByStaffID)
	return saved, err
}

func (d *CoreDAL) SaveStaffDashboardLayoutWithColumns(
	ctx context.Context,
	staffID string,
	items []DashboardWidgetItem,
	columnCount int,
	updatedByStaffID string,
) ([]DashboardWidgetItem, int, error) {
	trimmedStaff := strings.TrimSpace(staffID)
	if trimmedStaff == "" {
		return nil, DefaultDashboardColumnCount, errors.New("missing mandatory parameter: staff_id")
	}
	normalized, err := ValidateAndNormalizeDashboardWidgetItems(items)
	if err != nil {
		return nil, DefaultDashboardColumnCount, err
	}

	d.ensureDashboardLayoutsColumnCount(ctx)

	resolvedCols := columnCount
	if resolvedCols == 0 {
		_, existingCols, found, _ := d.GetStaffDashboardLayoutWithColumns(ctx, trimmedStaff)
		if found {
			resolvedCols = existingCols
		} else {
			resolvedCols = DefaultDashboardColumnCount
		}
	} else {
		validCols, colErr := ValidateAndNormalizeDashboardColumnCount(resolvedCols)
		if colErr != nil {
			return nil, DefaultDashboardColumnCount, colErr
		}
		resolvedCols = validCols
	}

	rawBytes, err := json.Marshal(normalized)
	if err != nil {
		return nil, DefaultDashboardColumnCount, fmt.Errorf("failed to marshal dashboard layout JSON: %w", err)
	}

	rowID := "staff:" + trimmedStaff
	actor := strings.TrimSpace(updatedByStaffID)
	if actor == "" {
		actor = trimmedStaff
	}

	if _, err := d.writeDB.ExecContext(
		ctx,
		`INSERT INTO dashboard_layouts (id, scope_type, scope_target, widgets_json, column_count, updated_by_staff_id, updated_at)
		 VALUES (?, 'STAFF', ?, ?, ?, ?, strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
		 ON CONFLICT(scope_type, scope_target) DO UPDATE SET
		     widgets_json = excluded.widgets_json,
		     column_count = excluded.column_count,
		     updated_by_staff_id = excluded.updated_by_staff_id,
		     updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')`,
		rowID,
		trimmedStaff,
		string(rawBytes),
		resolvedCols,
		actor,
	); err != nil {
		return nil, DefaultDashboardColumnCount, fmt.Errorf("failed to save staff dashboard layout for %s: %w", trimmedStaff, err)
	}

	return normalized, resolvedCols, nil
}

func (d *CoreDAL) DeleteStaffDashboardLayout(ctx context.Context, staffID string) (bool, error) {
	trimmedStaff := strings.TrimSpace(staffID)
	if trimmedStaff == "" {
		return false, errors.New("missing mandatory parameter: staff_id")
	}
	res, err := d.writeDB.ExecContext(
		ctx,
		`DELETE FROM dashboard_layouts WHERE scope_type = 'STAFF' AND scope_target = ?`,
		trimmedStaff,
	)
	if err != nil {
		return false, fmt.Errorf("failed to delete staff dashboard layout for %s: %w", trimmedStaff, err)
	}
	affected, _ := res.RowsAffected()
	return affected > 0, nil
}

func (d *CoreDAL) GetRoleDefaultDashboardLayout(ctx context.Context, roleID string) ([]DashboardWidgetItem, bool, error) {
	normalizedRole := strings.ToUpper(strings.TrimSpace(roleID))
	if normalizedRole == "" {
		return nil, false, errors.New("missing mandatory parameter: role_id")
	}

	var rawJSON string
	err := d.readDB.QueryRowContext(
		ctx,
		`SELECT widgets_json FROM dashboard_layouts WHERE scope_type = 'ROLE_DEFAULT' AND UPPER(scope_target) = ?`,
		normalizedRole,
	).Scan(&rawJSON)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("failed to query role default dashboard layout for %s: %w", normalizedRole, err)
	}

	items, decErr := decodeDashboardWidgetsJSON(rawJSON)
	if decErr != nil {
		return nil, false, decErr
	}
	return items, true, nil
}

func (d *CoreDAL) ListAllRoleDefaultDashboardLayouts(ctx context.Context) (map[string][]DashboardWidgetItem, error) {
	result := make(map[string][]DashboardWidgetItem)
	rows, err := d.readDB.QueryContext(
		ctx,
		`SELECT UPPER(scope_target), widgets_json FROM dashboard_layouts WHERE scope_type = 'ROLE_DEFAULT' ORDER BY scope_target ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list role default dashboard layouts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var roleKey, rawJSON string
		if scanErr := rows.Scan(&roleKey, &rawJSON); scanErr != nil {
			return nil, scanErr
		}
		items, decErr := decodeDashboardWidgetsJSON(rawJSON)
		if decErr != nil {
			continue
		}
		result[roleKey] = items
	}
	return result, rows.Err()
}

func (d *CoreDAL) SaveRoleDefaultDashboardLayout(
	ctx context.Context,
	roleID string,
	items []DashboardWidgetItem,
	appendToExistingStaff bool,
	updatedByStaffID string,
) ([]DashboardWidgetItem, int, error) {
	normalizedRole := strings.ToUpper(strings.TrimSpace(roleID))
	if normalizedRole == "" {
		return nil, 0, errors.New("missing mandatory parameter: role_id")
	}
	normalizedItems, err := ValidateAndNormalizeDashboardWidgetItems(items)
	if err != nil {
		return nil, 0, err
	}

	rawBytes, err := json.Marshal(normalizedItems)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to marshal role default dashboard layout: %w", err)
	}

	tx, err := d.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to begin transaction for role default dashboard layout: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	actor := strings.TrimSpace(updatedByStaffID)
	rowID := "role:" + normalizedRole
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO dashboard_layouts (id, scope_type, scope_target, widgets_json, updated_by_staff_id, updated_at)
		 VALUES (?, 'ROLE_DEFAULT', ?, ?, ?, strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
		 ON CONFLICT(scope_type, scope_target) DO UPDATE SET
		     widgets_json = excluded.widgets_json,
		     updated_by_staff_id = excluded.updated_by_staff_id,
		     updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')`,
		rowID,
		normalizedRole,
		string(rawBytes),
		actor,
	); err != nil {
		return nil, 0, fmt.Errorf("failed to upsert role default dashboard layout for %s: %w", normalizedRole, err)
	}

	appendedStaffCount := 0
	if appendToExistingStaff && len(normalizedItems) > 0 {
		rows, qErr := tx.QueryContext(
			ctx,
			`SELECT dl.scope_target, dl.widgets_json
			 FROM dashboard_layouts dl
			 WHERE dl.scope_type = 'STAFF'
			   AND EXISTS (
			       SELECT 1
			       FROM staff s
			       JOIN person_roles pr ON pr.person_id = s.person_id
			       WHERE s.id = dl.scope_target
			         AND UPPER(pr.role_id) = ?
			   )`,
			normalizedRole,
		)
		if qErr != nil {
			return nil, 0, fmt.Errorf("failed to query existing staff layouts for role %s: %w", normalizedRole, qErr)
		}

		type staffUpdate struct {
			staffID     string
			mergedBytes string
		}
		var updates []staffUpdate

		for rows.Next() {
			var targetStaffID, existingJSON string
			if scanErr := rows.Scan(&targetStaffID, &existingJSON); scanErr != nil {
				rows.Close()
				return nil, 0, scanErr
			}
			existingItems, decErr := decodeDashboardWidgetsJSON(existingJSON)
			if decErr != nil {
				existingItems = []DashboardWidgetItem{}
			}
			merged := make([]DashboardWidgetItem, 0, len(existingItems)+len(normalizedItems))
			merged = append(merged, existingItems...)
			changed := false
			for _, candidate := range normalizedItems {
				exists := false
				for _, cur := range merged {
					if cur.PlugID == candidate.PlugID && cur.WidgetID == candidate.WidgetID {
						exists = true
						break
					}
				}
				if !exists {
					merged = append(merged, candidate)
					changed = true
				}
			}
			if changed {
				b, mErr := json.Marshal(merged)
				if mErr == nil {
					updates = append(updates, staffUpdate{
						staffID:     targetStaffID,
						mergedBytes: string(b),
					})
				}
			}
		}
		rows.Close()

		for _, u := range updates {
			if _, uErr := tx.ExecContext(
				ctx,
				`UPDATE dashboard_layouts
				 SET widgets_json = ?,
				     updated_by_staff_id = ?,
				     updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
				 WHERE scope_type = 'STAFF' AND scope_target = ?`,
				u.mergedBytes,
				actor,
				u.staffID,
			); uErr != nil {
				return nil, 0, fmt.Errorf("failed to append role widgets to staff layout %s: %w", u.staffID, uErr)
			}
			appendedStaffCount++
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, 0, fmt.Errorf("failed to commit role default dashboard layout: %w", err)
	}

	return normalizedItems, appendedStaffCount, nil
}

func (d *CoreDAL) AppendWidgetToRoleDashboardLayout(
	ctx context.Context,
	roleID string,
	item DashboardWidgetItem,
	updatedByStaffID string,
) ([]DashboardWidgetItem, int, error) {
	normalizedRole := strings.ToUpper(strings.TrimSpace(roleID))
	if normalizedRole == "" {
		return nil, 0, errors.New("missing mandatory parameter: role_id")
	}
	validated, err := ValidateAndNormalizeDashboardWidgetItems([]DashboardWidgetItem{item})
	if err != nil || len(validated) == 0 {
		if err != nil {
			return nil, 0, err
		}
		return nil, 0, errors.New("INVALID_DASHBOARD_WIDGET_ITEM: missing plug_id or widget_id")
	}

	existingRoleItems, found, err := d.GetRoleDefaultDashboardLayout(ctx, normalizedRole)
	if err != nil {
		return nil, 0, err
	}
	nextItems := make([]DashboardWidgetItem, 0, len(existingRoleItems)+1)
	if found && len(existingRoleItems) > 0 {
		nextItems = append(nextItems, existingRoleItems...)
	}
	nextItems = append(nextItems, validated[0])

	return d.SaveRoleDefaultDashboardLayout(ctx, normalizedRole, nextItems, true, updatedByStaffID)
}

func (d *CoreDAL) ResolveEffectiveDashboardLayout(
	ctx context.Context,
	staffID string,
	activeRoles []string,
	fallbackDefault []DashboardWidgetItem,
) ([]DashboardWidgetItem, string, error) {
	items, _, source, err := d.ResolveEffectiveDashboardLayoutWithColumns(ctx, staffID, activeRoles, fallbackDefault)
	return items, source, err
}

func (d *CoreDAL) ResolveEffectiveDashboardLayoutWithColumns(
	ctx context.Context,
	staffID string,
	activeRoles []string,
	fallbackDefault []DashboardWidgetItem,
) ([]DashboardWidgetItem, int, string, error) {
	if trimmedStaff := strings.TrimSpace(staffID); trimmedStaff != "" {
		staffItems, colCount, found, err := d.GetStaffDashboardLayoutWithColumns(ctx, trimmedStaff)
		if err != nil {
			return nil, DefaultDashboardColumnCount, "", err
		}
		if found {
			return staffItems, colCount, "STAFF", nil
		}
	}

	roleCandidates := make([]string, 0, len(activeRoles)+1)
	for _, r := range activeRoles {
		if norm := strings.ToUpper(strings.TrimSpace(r)); norm != "" {
			roleCandidates = append(roleCandidates, norm)
		}
	}
	if len(roleCandidates) == 0 {
		roleCandidates = append(roleCandidates, "TEACHER")
	}

	for _, r := range roleCandidates {
		roleItems, found, err := d.GetRoleDefaultDashboardLayout(ctx, r)
		if err != nil {
			return nil, DefaultDashboardColumnCount, "", err
		}
		if found && len(roleItems) > 0 {
			return roleItems, DefaultDashboardColumnCount, "ROLE_DEFAULT:" + r, nil
		}
	}

	normalizedFallback, err := ValidateAndNormalizeDashboardWidgetItems(fallbackDefault)
	if err != nil {
		normalizedFallback = []DashboardWidgetItem{}
	}
	return normalizedFallback, DefaultDashboardColumnCount, "SYSTEM_FALLBACK", nil
}

func (d *CoreDAL) ResetAllDashboardLayoutsForTest(ctx context.Context) error {
	_, err := d.writeDB.ExecContext(ctx, `DELETE FROM dashboard_layouts`)
	return err
}
