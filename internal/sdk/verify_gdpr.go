package sdk

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"optimis/internal/host"
)

type columnMeta struct {
	Name       string
	ColType    string
	NotNull    int
	DefaultVal sql.NullString
	PK         int
}

func verifyGDPRHooksZeroOrphans(pluginDir string, manifest *host.Manifest) error {
	if manifest.Database == nil || strings.TrimSpace(manifest.Database.Alias) == "" {
		return nil
	}
	alias := strings.TrimSpace(manifest.Database.Alias)

	tmpDir, err := os.MkdirTemp("", "mis-verify-gdpr-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	pluginsRoot := filepath.Join(tmpDir, "plugins")
	targetPluginCopy := filepath.Join(pluginsRoot, "target_plugin")
	if err := copyDirectory(pluginDir, targetPluginCopy); err != nil {
		return fmt.Errorf("failed to stage plugin for GDPR verification: %w", err)
	}

	h, err := host.NewHost(host.Config{
		DataDirPath:      filepath.Join(tmpDir, "data"),
		SchoolURN:        "100001",
		HostKMSKeyHandle: "kms_verify_key_01",
		PluginsDir:       pluginsRoot,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize headless host for GDPR verification: %w", err)
	}
	defer h.Close()

	if err := h.LoadPlugin("target_plugin"); err != nil {
		return fmt.Errorf("failed to load plugin in headless GDPR verifier: %w", err)
	}

	ctx := context.Background()
	writeDB := h.DAL().WriteDB()

	// Discover all plugin tables that have a pupil_id column
	rows, err := writeDB.QueryContext(
		ctx,
		fmt.Sprintf("SELECT name, COALESCE(sql, '') FROM %s.sqlite_master WHERE type = 'table' ORDER BY name ASC", alias),
	)
	if err != nil {
		return fmt.Errorf("failed to inspect %s.sqlite_master: %w", alias, err)
	}
	type tableDDL struct {
		name string
		ddl  string
	}
	var tables []tableDDL
	for rows.Next() {
		var name, ddl string
		if err := rows.Scan(&name, &ddl); err != nil {
			rows.Close()
			return err
		}
		if strings.HasPrefix(strings.ToLower(name), "sqlite_") || strings.HasPrefix(name, "__") {
			continue
		}
		tables = append(tables, tableDDL{name: name, ddl: ddl})
	}
	rows.Close()

	const syntheticPupilID = "pup_verify_gdpr_01"
	var pupilTables []string

	tableMap := make(map[string]tableDDL, len(tables))
	colsMap := make(map[string][]columnMeta, len(tables))
	fkMap := make(map[string]map[string]string, len(tables))

	for _, tbl := range tables {
		tableMap[tbl.name] = tbl
		cols, err := inspectTableColumns(ctx, writeDB, alias, tbl.name)
		if err != nil {
			return err
		}
		colsMap[tbl.name] = cols

		fks, err := inspectTableForeignKeys(ctx, writeDB, alias, tbl.name)
		if err != nil {
			return err
		}
		fkMap[tbl.name] = fks

		for _, c := range cols {
			if strings.EqualFold(c.Name, "pupil_id") {
				pupilTables = append(pupilTables, tbl.name)
				break
			}
		}
	}

	seededTables := make(map[string]bool, len(tables))
	var ensureTableSeeded func(tableName string) error
	ensureTableSeeded = func(tableName string) error {
		if seededTables[tableName] {
			return nil
		}
		tbl, ok := tableMap[tableName]
		if !ok {
			return nil
		}
		for _, parentTable := range fkMap[tableName] {
			if parentTable != "" && parentTable != tableName {
				if err := ensureTableSeeded(parentTable); err != nil {
					return err
				}
			}
		}
		if err := insertSyntheticPupilRow(ctx, writeDB, alias, tbl.name, tbl.ddl, colsMap[tableName], fkMap[tableName], syntheticPupilID); err != nil {
			return fmt.Errorf("failed to seed synthetic GDPR row in %s.%s: %w", alias, tbl.name, err)
		}
		seededTables[tableName] = true
		return nil
	}

	for _, tblName := range pupilTables {
		if err := ensureTableSeeded(tblName); err != nil {
			return err
		}
	}

	adminCaller := host.CallerContext{
		Type:        "USER",
		UserID:      "usr_verify_admin",
		StaffID:     "stf_verify_admin",
		ActiveRoles: []string{"ADMIN", "DPO"},
		Scopes: []string{
			"dal.pupils.read:all",
			"dal.pupils.write:all",
			"dal.staff.write:admin",
			"dal.census.read:all",
		},
		RequestID: "req_verify_gdpr",
	}

	sections, err := h.InvokePluginExportHook(ctx, manifest.ID, adminCaller, syntheticPupilID)
	if err != nil {
		return fmt.Errorf("gdpr_hooks.export_pupil_data verification failed: %w", err)
	}
	if len(sections) == 0 {
		return fmt.Errorf("gdpr_hooks.export_pupil_data returned 0 sections for %s", manifest.ID)
	}

	if _, err := h.InvokePluginPurgeHook(ctx, manifest.ID, adminCaller, syntheticPupilID); err != nil {
		return fmt.Errorf("gdpr_hooks.purge_pupil_data verification failed: %w", err)
	}

	for _, tbl := range pupilTables {
		var remaining int
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s.%s WHERE pupil_id = ?", alias, tbl)
		if err := writeDB.QueryRowContext(ctx, query, syntheticPupilID).Scan(&remaining); err != nil {
			return fmt.Errorf("failed to count remaining rows in %s.%s: %w", alias, tbl, err)
		}
		if remaining > 0 {
			return fmt.Errorf("GDPR_ORPHAN_ROWS_DETECTED: table %s.%s retained %d orphaned row(s) for pupil_id after purge_pupil_data", alias, tbl, remaining)
		}
	}

	return nil
}

func inspectTableForeignKeys(ctx context.Context, db *sql.DB, alias string, table string) (map[string]string, error) {
	rows, err := db.QueryContext(ctx, fmt.Sprintf("PRAGMA %s.foreign_key_list(%s)", alias, table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	fks := make(map[string]string)
	for rows.Next() {
		var id, seq int
		var refTable, from, to, onUpdate, onDelete, match string
		if err := rows.Scan(&id, &seq, &refTable, &from, &to, &onUpdate, &onDelete, &match); err != nil {
			return nil, err
		}
		fks[strings.ToLower(from)] = refTable
	}
	return fks, rows.Err()
}

func inspectTableColumns(ctx context.Context, db *sql.DB, alias string, table string) ([]columnMeta, error) {
	rows, err := db.QueryContext(ctx, fmt.Sprintf("PRAGMA %s.table_info(%s)", alias, table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cols []columnMeta
	for rows.Next() {
		var cid int
		var cm columnMeta
		if err := rows.Scan(&cid, &cm.Name, &cm.ColType, &cm.NotNull, &cm.DefaultVal, &cm.PK); err != nil {
			return nil, err
		}
		cols = append(cols, cm)
	}
	return cols, rows.Err()
}

func extractCheckAllowedValue(ddl string, colName string) (string, bool) {
	pattern := fmt.Sprintf(`(?i)\b%s\b[^,)]*CHECK\s*\(\s*\b%s\b\s+IN\s*\(\s*'([^']+)'`, regexp.QuoteMeta(colName), regexp.QuoteMeta(colName))
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", false
	}
	if m := re.FindStringSubmatch(ddl); len(m) >= 2 {
		return m[1], true
	}
	return "", false
}

func insertSyntheticPupilRow(ctx context.Context, db *sql.DB, alias string, table string, ddl string, cols []columnMeta, fks map[string]string, pupilID string) error {
	var colNames []string
	var placeholders []string
	var args []any

	for idx, c := range cols {
		lowerName := strings.ToLower(c.Name)
		upperType := strings.ToUpper(c.ColType)

		// Skip INTEGER PRIMARY KEY autoincrement columns unless named id with TEXT type
		if c.PK > 0 && strings.Contains(upperType, "INT") && lowerName != "pupil_id" {
			continue
		}

		colNames = append(colNames, c.Name)
		placeholders = append(placeholders, "?")

		if parentTable, isFK := fks[lowerName]; isFK && parentTable != "" {
			args = append(args, fmt.Sprintf("syn_fk_%s", parentTable))
			continue
		}

		if c.PK > 0 && lowerName != "pupil_id" {
			args = append(args, fmt.Sprintf("syn_fk_%s", table))
			continue
		}

		if checkVal, ok := extractCheckAllowedValue(ddl, c.Name); ok {
			args = append(args, checkVal)
			continue
		}

		switch {
		case lowerName == "pupil_id":
			args = append(args, pupilID)
		case lowerName == "id" || strings.HasSuffix(lowerName, "_id"):
			args = append(args, fmt.Sprintf("syn_%s_%d", table, idx))
		case strings.Contains(lowerName, "point") || strings.Contains(lowerName, "count") || strings.Contains(lowerName, "score") || strings.Contains(upperType, "INT"):
			args = append(args, 1)
		case strings.Contains(upperType, "REAL") || strings.Contains(upperType, "FLOA") || strings.Contains(upperType, "DOUB"):
			args = append(args, 1.0)
		default:
			args = append(args, fmt.Sprintf("synthetic_%s", lowerName))
		}
	}

	sqlStmt := fmt.Sprintf(
		"INSERT INTO %s.%s (%s) VALUES (%s)",
		alias,
		table,
		strings.Join(colNames, ", "),
		strings.Join(placeholders, ", "),
	)
	_, err := db.ExecContext(ctx, sqlStmt, args...)
	return err
}

func copyDirectory(srcDir string, dstDir string) error {
	return filepath.WalkDir(srcDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(dstDir, rel)
		if d.IsDir() {
			return os.MkdirAll(targetPath, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(targetPath, data, 0o644)
	})
}
