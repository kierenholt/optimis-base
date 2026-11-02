package dal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

type InstalledPluginRecord struct {
	ManifestID         string `json:"manifest_id"`
	PlugID             string `json:"plug_id"`
	Version            string `json:"version"`
	State              string `json:"state"`
	DatabaseAlias      string `json:"database_alias,omitempty"`
	Folder             string `json:"folder"`
	ApprovedScopesJSON string `json:"approved_scopes_json"`
	InstalledAt        string `json:"installed_at"`
	UpdatedAt          string `json:"updated_at"`
}

func (d *CoreDAL) GetInstalledPlugin(ctx context.Context, manifestOrPlugID string) (*InstalledPluginRecord, error) {
	trimmed := strings.TrimSpace(manifestOrPlugID)
	if trimmed == "" {
		return nil, errors.New("missing mandatory parameter: manifestOrPlugID")
	}

	var rec InstalledPluginRecord
	var dbAlias sql.NullString
	err := d.readDB.QueryRowContext(
		ctx,
		`SELECT manifest_id, plug_id, version, state, database_alias, folder, approved_scopes_json, installed_at, updated_at
		 FROM installed_plugins
		 WHERE manifest_id = ? OR plug_id = ?
		 LIMIT 1`,
		trimmed, trimmed,
	).Scan(
		&rec.ManifestID,
		&rec.PlugID,
		&rec.Version,
		&rec.State,
		&dbAlias,
		&rec.Folder,
		&rec.ApprovedScopesJSON,
		&rec.InstalledAt,
		&rec.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("installed plugin not found: %s", trimmed)
		}
		return nil, fmt.Errorf("failed to query installed_plugins for %s: %w", trimmed, err)
	}
	if dbAlias.Valid {
		rec.DatabaseAlias = dbAlias.String
	}
	return &rec, nil
}

func (d *CoreDAL) ListInstalledPlugins(ctx context.Context) ([]*InstalledPluginRecord, error) {
	rows, err := d.readDB.QueryContext(
		ctx,
		`SELECT manifest_id, plug_id, version, state, database_alias, folder, approved_scopes_json, installed_at, updated_at
		 FROM installed_plugins
		 ORDER BY manifest_id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list installed_plugins: %w", err)
	}
	defer rows.Close()

	var out []*InstalledPluginRecord
	for rows.Next() {
		var rec InstalledPluginRecord
		var dbAlias sql.NullString
		if err := rows.Scan(
			&rec.ManifestID,
			&rec.PlugID,
			&rec.Version,
			&rec.State,
			&dbAlias,
			&rec.Folder,
			&rec.ApprovedScopesJSON,
			&rec.InstalledAt,
			&rec.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan installed_plugins row: %w", err)
		}
		if dbAlias.Valid {
			rec.DatabaseAlias = dbAlias.String
		}
		out = append(out, &rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating installed_plugins rows: %w", err)
	}
	return out, nil
}

func (d *CoreDAL) UpsertInstalledPlugin(ctx context.Context, rec InstalledPluginRecord) error {
	trimmedManifestID := strings.TrimSpace(rec.ManifestID)
	if trimmedManifestID == "" {
		return errors.New("missing mandatory parameter: ManifestID")
	}
	trimmedPlugID := strings.TrimSpace(rec.PlugID)
	if len(trimmedPlugID) != 8 {
		return fmt.Errorf("invalid plug_id %q: must be exactly 8 characters", trimmedPlugID)
	}
	trimmedVersion := strings.TrimSpace(rec.Version)
	if trimmedVersion == "" {
		return errors.New("missing mandatory parameter: Version")
	}
	state := strings.TrimSpace(rec.State)
	if state != "ACTIVE" && state != "UNINSTALLED_DORMANT" {
		return fmt.Errorf("invalid state %q: must be ACTIVE or UNINSTALLED_DORMANT", state)
	}
	folder := strings.TrimSpace(rec.Folder)
	if folder == "" {
		return errors.New("missing mandatory parameter: Folder")
	}
	scopesJSON := strings.TrimSpace(rec.ApprovedScopesJSON)
	if scopesJSON == "" {
		scopesJSON = "[]"
	}

	installedAt := strings.TrimSpace(rec.InstalledAt)
	if installedAt == "" {
		installedAt = time.Now().UTC().Format(time.RFC3339)
	}
	updatedAt := strings.TrimSpace(rec.UpdatedAt)
	if updatedAt == "" {
		updatedAt = time.Now().UTC().Format(time.RFC3339)
	}

	_, err := d.writeDB.ExecContext(
		ctx,
		`INSERT INTO installed_plugins (
			manifest_id, plug_id, version, state, database_alias, folder, approved_scopes_json, installed_at, updated_at
		 ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(manifest_id) DO UPDATE SET
			plug_id = excluded.plug_id,
			version = excluded.version,
			state = excluded.state,
			database_alias = excluded.database_alias,
			folder = excluded.folder,
			approved_scopes_json = excluded.approved_scopes_json,
			updated_at = excluded.updated_at`,
		trimmedManifestID,
		trimmedPlugID,
		trimmedVersion,
		state,
		strings.TrimSpace(rec.DatabaseAlias),
		folder,
		scopesJSON,
		installedAt,
		updatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert installed_plugin for %s: %w", trimmedManifestID, err)
	}
	return nil
}

func (d *CoreDAL) SetInstalledPluginState(ctx context.Context, manifestOrPlugID string, state string) error {
	trimmed := strings.TrimSpace(manifestOrPlugID)
	if trimmed == "" {
		return errors.New("missing mandatory parameter: manifestOrPlugID")
	}
	trimmedState := strings.TrimSpace(state)
	if trimmedState != "ACTIVE" && trimmedState != "UNINSTALLED_DORMANT" {
		return fmt.Errorf("invalid state %q: must be ACTIVE or UNINSTALLED_DORMANT", trimmedState)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	res, err := d.writeDB.ExecContext(
		ctx,
		`UPDATE installed_plugins
		 SET state = ?, updated_at = ?
		 WHERE manifest_id = ? OR plug_id = ?`,
		trimmedState, now, trimmed, trimmed,
	)
	if err != nil {
		return fmt.Errorf("failed to update installed_plugin state for %s: %w", trimmed, err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to verify rows affected for %s: %w", trimmed, err)
	}
	if affected == 0 {
		return fmt.Errorf("installed plugin not found: %s", trimmed)
	}
	return nil
}

func (d *CoreDAL) DeleteInstalledPlugin(ctx context.Context, manifestOrPlugID string) error {
	trimmed := strings.TrimSpace(manifestOrPlugID)
	if trimmed == "" {
		return errors.New("missing mandatory parameter: manifestOrPlugID")
	}

	_, err := d.writeDB.ExecContext(
		ctx,
		`DELETE FROM installed_plugins WHERE manifest_id = ? OR plug_id = ?`,
		trimmed, trimmed,
	)
	if err != nil {
		return fmt.Errorf("failed to delete installed_plugin %s: %w", trimmed, err)
	}
	return nil
}

func (d *CoreDAL) IsPlugIDAllocated(ctx context.Context, plugID string, excludeManifestID string) (bool, error) {
	trimmedPlugID := strings.TrimSpace(plugID)
	if trimmedPlugID == "" {
		return false, errors.New("missing mandatory parameter: plugID")
	}

	var dummy int
	err := d.readDB.QueryRowContext(
		ctx,
		`SELECT 1 FROM installed_plugins WHERE plug_id = ? AND manifest_id != ? LIMIT 1`,
		trimmedPlugID, strings.TrimSpace(excludeManifestID),
	).Scan(&dummy)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check plug_id allocation: %w", err)
	}
	return true, nil
}
