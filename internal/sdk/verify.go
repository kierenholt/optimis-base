package sdk

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	lua "github.com/yuin/gopher-lua"
	_ "modernc.org/sqlite"

	"optimis/internal/host"
)

var (
	manifestIDPattern      = regexp.MustCompile(`^[a-z0-9]+(\.[a-z0-9_-]+)+$`)
	dbAliasPattern         = regexp.MustCompile(`^plugin_[a-z0-9_]+$`)
	dataMisRPCAttrRegex    = regexp.MustCompile(`data-mis-rpc\s*=\s*["']([^"']+)["']`)
	externalScriptSrcRegex = regexp.MustCompile(`(?i)<script[^>]+src\s*=\s*["'](https?://[^"']+)["']`)
	externalLinkHrefRegex  = regexp.MustCompile(`(?i)<link[^>]+href\s*=\s*["'](https?://[^"']+)["']`)
	externalCSSImportRegex = regexp.MustCompile(`(?i)@import\s+(?:url\s*\(\s*)?["']?(https?://[^"')]+)["']?`)
	externalFetchRegex     = regexp.MustCompile(`(?i)\bfetch\s*\(\s*["'](https?://[^"']+)["']`)
	destructiveSQLRegex    = regexp.MustCompile(`(?i)\b(DROP\s+COLUMN|RENAME\s+COLUMN|DROP\s+TABLE)\b`)
)

type VerifyReport struct {
	PluginID string
	Gates    []string
}

func VerifyPluginDir(pluginDir string, out io.Writer) (*VerifyReport, error) {
	trimmedDir := strings.TrimSpace(pluginDir)
	if trimmedDir == "" {
		return nil, errors.New("missing mandatory plugin directory path")
	}
	absDir, err := filepath.Abs(trimmedDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve plugin directory path: %w", err)
	}
	info, err := os.Stat(absDir)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("plugin directory does not exist: %s", absDir)
	}

	if out == nil {
		out = io.Discard
	}

	// Gate 1: Manifest <-> Lua <-> UI data-mis-rpc cross-check
	manifest, err := verifyManifestLuaAndUI(absDir)
	if err != nil {
		return nil, err
	}
	_, _ = fmt.Fprintf(out, "type=verify_gate gate=manifest_lua_ui status=MANIFEST_LUA_UI_OK plugin=%s\n", manifest.ID)

	// Gate 2: Offline CSP & Sandbox check (connect-src 'none')
	if err := verifyOfflineCSPAssets(absDir); err != nil {
		return nil, err
	}
	_, _ = fmt.Fprintf(out, "type=verify_gate gate=csp_offline status=CSP_OFFLINE_OK plugin=%s\n", manifest.ID)

	// Gate 3: Expand-and-Contract SQL migration check
	if err := verifyExpandAndContractMigrations(absDir); err != nil {
		return nil, err
	}
	_, _ = fmt.Fprintf(out, "type=verify_gate gate=migrations status=MIGRATIONS_OK plugin=%s\n", manifest.ID)

	// Gate 4: UK GDPR export_pupil_data & purge_pupil_data zero-orphan check
	if err := verifyGDPRHooksZeroOrphans(absDir, manifest); err != nil {
		return nil, err
	}
	_, _ = fmt.Fprintf(out, "type=verify_gate gate=gdpr_zero_orphans status=GDPR_ZERO_ORPHANS_OK plugin=%s\n", manifest.ID)

	return &VerifyReport{
		PluginID: manifest.ID,
		Gates: []string{
			"MANIFEST_LUA_UI_OK",
			"CSP_OFFLINE_OK",
			"MIGRATIONS_OK",
			"GDPR_ZERO_ORPHANS_OK",
		},
	}, nil
}

func ReadAndValidateManifest(pluginDir string) (*host.Manifest, error) {
	manifestPath := filepath.Join(pluginDir, "manifest.json")
	rawBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("missing or unreadable manifest.json at %s: %w", manifestPath, err)
	}

	var manifest host.Manifest
	dec := json.NewDecoder(bytes.NewReader(rawBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&manifest); err != nil {
		return nil, fmt.Errorf("invalid manifest.json schema in %s: %w", manifestPath, err)
	}

	if !manifestIDPattern.MatchString(strings.TrimSpace(manifest.ID)) {
		return nil, fmt.Errorf("invalid manifest.id %q: must be reverse-DNS identifier", manifest.ID)
	}
	if strings.TrimSpace(manifest.Version) == "" {
		return nil, errors.New("manifest.json missing mandatory field: version")
	}
	if strings.TrimSpace(manifest.CoreDALVersion) == "" {
		return nil, errors.New("manifest.json missing mandatory field: core_dal_version")
	}
	if len(manifest.Endpoints) == 0 {
		return nil, errors.New("manifest.json must declare at least one endpoint in endpoints")
	}

	if manifest.Database != nil && strings.TrimSpace(manifest.Database.Alias) != "" {
		if !dbAliasPattern.MatchString(strings.TrimSpace(manifest.Database.Alias)) {
			return nil, fmt.Errorf("invalid database.alias %q: must match ^plugin_[a-z0-9_]+$", manifest.Database.Alias)
		}
		if manifest.GDPRHooks == nil ||
			strings.TrimSpace(manifest.GDPRHooks.ExportPupilData) == "" ||
			strings.TrimSpace(manifest.GDPRHooks.PurgePupilData) == "" {
			return nil, fmt.Errorf("plugin %s attaches database %s but is missing mandatory gdpr_hooks (export_pupil_data and purge_pupil_data)", manifest.ID, manifest.Database.Alias)
		}
	}

	return &manifest, nil
}

func verifyManifestLuaAndUI(pluginDir string) (*host.Manifest, error) {
	manifest, err := ReadAndValidateManifest(pluginDir)
	if err != nil {
		return nil, err
	}

	luaPath := filepath.Join(pluginDir, "src", "init.lua")
	luaBytes, err := os.ReadFile(luaPath)
	if err != nil {
		return nil, fmt.Errorf("missing or unreadable src/init.lua at %s: %w", luaPath, err)
	}

	L := newSandboxedLuaState()
	defer L.Close()

	if err := L.DoString(string(luaBytes)); err != nil {
		return nil, fmt.Errorf("syntax or load error in src/init.lua: %w", err)
	}

	for endpointName := range manifest.Endpoints {
		if L.GetGlobal(endpointName).Type() != lua.LTFunction {
			return nil, fmt.Errorf("MANIFEST_LUA_MISMATCH: endpoint %q declared in manifest.json is not a function in src/init.lua", endpointName)
		}
	}

	for workerName := range manifest.Workers {
		if L.GetGlobal(workerName).Type() != lua.LTFunction {
			return nil, fmt.Errorf("MANIFEST_LUA_MISMATCH: worker %q declared in manifest.json is not a function in src/init.lua", workerName)
		}
	}

	if manifest.GDPRHooks != nil {
		for _, hookName := range []string{
			strings.TrimSpace(manifest.GDPRHooks.ExportPupilData),
			strings.TrimSpace(manifest.GDPRHooks.PurgePupilData),
		} {
			if hookName != "" && L.GetGlobal(hookName).Type() != lua.LTFunction {
				return nil, fmt.Errorf("MANIFEST_LUA_MISMATCH: gdpr_hook %q declared in manifest.json is not a function in src/init.lua", hookName)
			}
		}
	}

	uiDir := filepath.Join(pluginDir, "ui")
	if info, statErr := os.Stat(uiDir); statErr == nil && info.IsDir() {
		err = filepath.WalkDir(uiDir, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(path))
			if ext != ".html" && ext != ".js" {
				return nil
			}
			content, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			matches := dataMisRPCAttrRegex.FindAllStringSubmatch(string(content), -1)
			for _, m := range matches {
				rpcMethod := strings.TrimSpace(m[1])
				if rpcMethod == "" {
					continue
				}
				if _, declared := manifest.Endpoints[rpcMethod]; !declared {
					relPath, _ := filepath.Rel(pluginDir, path)
					return fmt.Errorf("UI_MANIFEST_MISMATCH: undeclared data-mis-rpc method %q in %s (not declared in manifest.json endpoints)", rpcMethod, relPath)
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return manifest, nil
}

func verifyOfflineCSPAssets(pluginDir string) error {
	uiDir := filepath.Join(pluginDir, "ui")
	info, err := os.Stat(uiDir)
	if err != nil || !info.IsDir() {
		return nil
	}

	return filepath.WalkDir(uiDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".html" && ext != ".js" && ext != ".css" {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		text := string(raw)
		relPath, _ := filepath.Rel(pluginDir, path)

		for _, re := range []*regexp.Regexp{
			externalScriptSrcRegex,
			externalLinkHrefRegex,
			externalCSSImportRegex,
			externalFetchRegex,
		} {
			if m := re.FindStringSubmatch(text); len(m) >= 2 {
				return fmt.Errorf("CSP_OFFLINE_VIOLATION: external URL %q in %s violates offline connect-src 'none' sandbox rule", m[1], relPath)
			}
		}
		return nil
	})
}

func stripSQLComments(sqlText string) string {
	lines := strings.Split(sqlText, "\n")
	cleaned := make([]string, 0, len(lines))
	for _, line := range lines {
		if idx := strings.Index(line, "--"); idx >= 0 {
			line = line[:idx]
		}
		cleaned = append(cleaned, line)
	}
	return strings.Join(cleaned, "\n")
}

func verifyExpandAndContractMigrations(pluginDir string) error {
	migDir := filepath.Join(pluginDir, "migrations")
	info, err := os.Stat(migDir)
	if err != nil || !info.IsDir() {
		return nil
	}

	entries, err := os.ReadDir(migDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var sqlFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".sql") {
			sqlFiles = append(sqlFiles, filepath.Join(migDir, entry.Name()))
		}
	}
	sort.Strings(sqlFiles)
	if len(sqlFiles) == 0 {
		return nil
	}

	tmpDir, err := os.MkdirTemp("", "mis-verify-mig-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "verify_migrations.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	for _, file := range sqlFiles {
		rawSQL, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		stripped := stripSQLComments(string(rawSQL))
		if m := destructiveSQLRegex.FindStringSubmatch(stripped); len(m) >= 2 {
			relPath, _ := filepath.Rel(pluginDir, file)
			return fmt.Errorf("EXPAND_AND_CONTRACT_VIOLATION: destructive SQL %q in %s is forbidden on minor migrations", m[1], relPath)
		}

		tx, err := db.Begin()
		if err != nil {
			return err
		}
		if _, execErr := tx.Exec(string(rawSQL)); execErr != nil {
			_ = tx.Rollback()
			relPath, _ := filepath.Rel(pluginDir, file)
			return fmt.Errorf("SQL_MIGRATION_ERROR in %s: %w", relPath, execErr)
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}

	return nil
}

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

	for _, tbl := range tables {
		cols, err := inspectTableColumns(ctx, writeDB, alias, tbl.name)
		if err != nil {
			return err
		}
		hasPupilID := false
		for _, c := range cols {
			if strings.EqualFold(c.Name, "pupil_id") {
				hasPupilID = true
				break
			}
		}
		if !hasPupilID {
			continue
		}
		pupilTables = append(pupilTables, tbl.name)

		if err := insertSyntheticPupilRow(ctx, writeDB, alias, tbl.name, tbl.ddl, cols, syntheticPupilID); err != nil {
			return fmt.Errorf("failed to seed synthetic GDPR row in %s.%s: %w", alias, tbl.name, err)
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

func insertSyntheticPupilRow(ctx context.Context, db *sql.DB, alias string, table string, ddl string, cols []columnMeta, pupilID string) error {
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

func newSandboxedLuaState() *lua.LState {
	L := lua.NewState(lua.Options{SkipOpenLibs: true})
	for _, lib := range []struct {
		name string
		fn   lua.LGFunction
	}{
		{lua.LoadLibName, lua.OpenPackage},
		{lua.BaseLibName, lua.OpenBase},
		{lua.TabLibName, lua.OpenTable},
		{lua.StringLibName, lua.OpenString},
		{lua.MathLibName, lua.OpenMath},
	} {
		L.Push(L.NewFunction(lib.fn))
		L.Push(lua.LString(lib.name))
		L.Call(1, 0)
	}
	L.SetGlobal("dofile", lua.LNil)
	L.SetGlobal("loadfile", lua.LNil)
	L.SetGlobal("package", lua.LNil)
	L.SetGlobal("require", lua.LNil)
	L.SetGlobal("os", lua.LNil)
	L.SetGlobal("io", lua.LNil)
	L.SetGlobal("debug", lua.LNil)
	return L
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
