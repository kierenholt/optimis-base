package sdk

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	_ "modernc.org/sqlite"
)

var (
	disallowedUIIconEmojiRegex = regexp.MustCompile(`[\x{1F300}-\x{1FAFF}\x{2600}-\x{27BF}]`)
	externalScriptSrcRegex     = regexp.MustCompile(`(?i)<script[^>]+src\s*=\s*["'](https?://[^"']+)["']`)
	externalLinkHrefRegex      = regexp.MustCompile(`(?i)<link[^>]+href\s*=\s*["'](https?://[^"']+)["']`)
	externalCSSImportRegex     = regexp.MustCompile(`(?i)@import\s+(?:url\s*\(\s*)?["']?(https?://[^"')]+)["']?`)
	externalFetchRegex         = regexp.MustCompile(`(?i)\bfetch\s*\(\s*["'](https?://[^"']+)["']`)
	destructiveSQLRegex        = regexp.MustCompile(`(?i)\b(DROP\s+COLUMN|RENAME\s+COLUMN|DROP\s+TABLE)\b`)
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

		if m := disallowedUIIconEmojiRegex.FindString(text); m != "" {
			return fmt.Errorf("NON_MONOCHROME_ICON_EMOJI_VIOLATION: disallowed emoji glyph %q in %s (all UI icons must be monochrome Lucide SVGs with stroke='currentColor')", m, relPath)
		}

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
