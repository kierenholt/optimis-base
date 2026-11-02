package dal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

type StaffSettings struct {
	StaffID           string `json:"staff_id"`
	BaseFontSize      string `json:"base_font_size"`
	ThemeMode         string `json:"theme_mode"`
	PinnedRecentsJSON string `json:"-"`
	UpdatedAt         string `json:"updated_at,omitempty"`
}

func ValidateBaseFontSize(raw string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	switch normalized {
	case "small", "medium", "large", "extra-large":
		return normalized, nil
	default:
		return "", fmt.Errorf("INVALID_BASE_FONT_SIZE: base_font_size %q must be one of 'small', 'medium', 'large', or 'extra-large'", raw)
	}
}

func ValidateThemeMode(raw string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	switch normalized {
	case "light", "high-contrast":
		return normalized, nil
	default:
		return "", fmt.Errorf("INVALID_THEME_MODE: theme_mode %q must be one of 'light' or 'high-contrast'", raw)
	}
}

func (d *CoreDAL) GetStaffSettings(ctx context.Context, staffID string) (StaffSettings, error) {
	trimmedStaff := strings.TrimSpace(staffID)
	if trimmedStaff == "" {
		return StaffSettings{}, errors.New("MISSING_MANDATORY_PARAMETER: staff_id is required for GetStaffSettings")
	}

	s := StaffSettings{
		StaffID:           trimmedStaff,
		BaseFontSize:      "medium",
		ThemeMode:         "light",
		PinnedRecentsJSON: "[]",
	}

	var prefFontSize, prefThemeMode, prefUpdated sql.NullString
	err := d.readDB.QueryRowContext(
		ctx,
		`SELECT base_font_size, theme_mode, updated_at
		 FROM staff_ui_preferences
		 WHERE staff_id = ?`,
		trimmedStaff,
	).Scan(&prefFontSize, &prefThemeMode, &prefUpdated)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return StaffSettings{}, fmt.Errorf("failed to query staff_ui_preferences for %s: %w", trimmedStaff, err)
	}
	if prefFontSize.Valid && strings.TrimSpace(prefFontSize.String) != "" {
		s.BaseFontSize = prefFontSize.String
	}
	if prefThemeMode.Valid && strings.TrimSpace(prefThemeMode.String) != "" {
		s.ThemeMode = prefThemeMode.String
	}
	if prefUpdated.Valid && strings.TrimSpace(prefUpdated.String) != "" {
		s.UpdatedAt = prefUpdated.String
	}

	var pinJSON, pinUpdated sql.NullString
	err = d.readDB.QueryRowContext(
		ctx,
		`SELECT pinned_recents_json, updated_at
		 FROM staff_pinned_recents
		 WHERE staff_id = ?`,
		trimmedStaff,
	).Scan(&pinJSON, &pinUpdated)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return StaffSettings{}, fmt.Errorf("failed to query staff_pinned_recents for %s: %w", trimmedStaff, err)
	}
	if pinJSON.Valid && strings.TrimSpace(pinJSON.String) != "" {
		s.PinnedRecentsJSON = pinJSON.String
	}
	if pinUpdated.Valid && strings.TrimSpace(pinUpdated.String) != "" && (s.UpdatedAt == "" || pinUpdated.String > s.UpdatedAt) {
		s.UpdatedAt = pinUpdated.String
	}

	return s, nil
}

func (d *CoreDAL) UpsertStaffUISettings(
	ctx context.Context,
	staffID string,
	baseFontSize *string,
	themeMode *string,
) (StaffSettings, error) {
	trimmedStaff := strings.TrimSpace(staffID)
	if trimmedStaff == "" {
		return StaffSettings{}, errors.New("MISSING_MANDATORY_PARAMETER: staff_id is required for UpsertStaffUISettings")
	}
	if baseFontSize == nil && themeMode == nil {
		return StaffSettings{}, errors.New("MISSING_MANDATORY_PARAMETER: at least one of base_font_size or theme_mode must be provided")
	}

	existing, err := d.GetStaffSettings(ctx, trimmedStaff)
	if err != nil {
		return StaffSettings{}, err
	}

	nextFontSize := existing.BaseFontSize
	if baseFontSize != nil {
		validatedFont, fErr := ValidateBaseFontSize(*baseFontSize)
		if fErr != nil {
			return StaffSettings{}, fErr
		}
		nextFontSize = validatedFont
	}

	nextTheme := existing.ThemeMode
	if themeMode != nil {
		validatedTheme, tErr := ValidateThemeMode(*themeMode)
		if tErr != nil {
			return StaffSettings{}, tErr
		}
		nextTheme = validatedTheme
	}

	_, err = d.writeDB.ExecContext(
		ctx,
		`INSERT INTO staff_ui_preferences (staff_id, base_font_size, theme_mode, updated_at)
		 VALUES (?, ?, ?, strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
		 ON CONFLICT(staff_id) DO UPDATE SET
		     base_font_size = excluded.base_font_size,
		     theme_mode = excluded.theme_mode,
		     updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')`,
		trimmedStaff,
		nextFontSize,
		nextTheme,
	)
	if err != nil {
		return StaffSettings{}, fmt.Errorf("failed to upsert staff_ui_preferences for %s: %w", trimmedStaff, err)
	}

	return d.GetStaffSettings(ctx, trimmedStaff)
}

func (d *CoreDAL) SaveStaffPinnedRecentsJSON(ctx context.Context, staffID string, pinnedRecentsJSON string) error {
	trimmedStaff := strings.TrimSpace(staffID)
	if trimmedStaff == "" {
		return errors.New("MISSING_MANDATORY_PARAMETER: staff_id is required for SaveStaffPinnedRecentsJSON")
	}
	cleanJSON := strings.TrimSpace(pinnedRecentsJSON)
	if cleanJSON == "" {
		cleanJSON = "[]"
	}

	_, err := d.writeDB.ExecContext(
		ctx,
		`INSERT INTO staff_pinned_recents (staff_id, pinned_recents_json, updated_at)
		 VALUES (?, ?, strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
		 ON CONFLICT(staff_id) DO UPDATE SET
		     pinned_recents_json = excluded.pinned_recents_json,
		     updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')`,
		trimmedStaff,
		cleanJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to save pinned_recents_json in staff_pinned_recents for %s: %w", trimmedStaff, err)
	}
	return nil
}

