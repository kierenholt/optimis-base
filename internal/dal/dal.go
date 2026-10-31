package dal

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"

	_ "modernc.org/sqlite"
)

var pluginAliasPattern = regexp.MustCompile(`^plugin_[a-z0-9_]+$`)

const maxSQLiteAttachedDBs = 10

type AttachedPluginDB struct {
	Alias        string
	FilePath     string
	MaxPageCount int
	AttachSeq    uint64
}

type ConnectHook func(ctx context.Context, conn driver.Conn) error

type sqliteConnector struct {
	dsn         string
	baseDriver  driver.Driver
	connectHook ConnectHook
}

func (c *sqliteConnector) Connect(ctx context.Context) (driver.Conn, error) {
	conn, err := c.baseDriver.Open(c.dsn)
	if err != nil {
		return nil, err
	}
	if c.connectHook != nil {
		if err := c.connectHook(ctx, conn); err != nil {
			_ = conn.Close()
			return nil, err
		}
	}
	return conn, nil
}

func (c *sqliteConnector) Driver() driver.Driver {
	return c.baseDriver
}

func execDriverConn(ctx context.Context, conn driver.Conn, query string) error {
	if execer, ok := conn.(driver.ExecerContext); ok {
		_, err := execer.ExecContext(ctx, query, nil)
		return err
	}
	stmt, err := conn.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(nil)
	return err
}

type CoreDAL struct {
	writeDB             *sql.DB
	readDB              *sql.DB
	dbPath              string
	numCPU              int
	attachMu            sync.RWMutex
	attachedDBs         map[string]AttachedPluginDB
	attachSeq           uint64
	connectHook         ConnectHook
	postCommitEventHook PostCommitEventHook
}

func NewCoreDAL(dataDirPath string, schoolURN string) (*CoreDAL, error) {
	return NewCoreDALWithPath(dataDirPath, schoolURN, "")
}

func NewCoreDALWithPath(dataDirPath string, schoolURN string, customDBPath string) (*CoreDAL, error) {
	if strings.TrimSpace(dataDirPath) == "" {
		return nil, errors.New("mandatory parameter dataDirPath is missing")
	}
	if strings.TrimSpace(schoolURN) == "" {
		return nil, errors.New("mandatory parameter schoolURN is missing")
	}

	dbPath := filepath.Join(dataDirPath, "core.db")
	if trimmedCustom := strings.TrimSpace(customDBPath); trimmedCustom != "" {
		dbPath = trimmedCustom
	}
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)", dbPath)

	probeDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve sqlite driver: %w", err)
	}
	baseDriver := probeDB.Driver()
	_ = probeDB.Close()

	numCPU := runtime.NumCPU()
	if numCPU < 1 {
		numCPU = 1
	}

	d := &CoreDAL{
		dbPath:      dbPath,
		numCPU:      numCPU,
		attachedDBs: make(map[string]AttachedPluginDB),
	}

	d.connectHook = func(ctx context.Context, conn driver.Conn) error {
		d.attachMu.RLock()
		attachments := make([]AttachedPluginDB, 0, len(d.attachedDBs))
		for _, att := range d.attachedDBs {
			attachments = append(attachments, att)
		}
		d.attachMu.RUnlock()

		sort.Slice(attachments, func(i, j int) bool {
			return attachments[i].Alias < attachments[j].Alias
		})

		for _, att := range attachments {
			escapedPath := strings.ReplaceAll(att.FilePath, "'", "''")
			attachSQL := fmt.Sprintf("ATTACH DATABASE '%s' AS %s;", escapedPath, att.Alias)
			if err := execDriverConn(ctx, conn, attachSQL); err != nil {
				return fmt.Errorf("ConnectHook failed to attach %s: %w", att.Alias, err)
			}
			pragmaSQL := fmt.Sprintf("PRAGMA %s.max_page_count = %d;", att.Alias, att.MaxPageCount)
			if err := execDriverConn(ctx, conn, pragmaSQL); err != nil {
				return fmt.Errorf("ConnectHook failed to set max_page_count on %s: %w", att.Alias, err)
			}
		}
		return nil
	}

	writeDB := sql.OpenDB(&sqliteConnector{
		dsn:         dsn,
		baseDriver:  baseDriver,
		connectHook: d.connectHook,
	})
	writeDB.SetMaxOpenConns(1)
	writeDB.SetMaxIdleConns(1)

	readDB := sql.OpenDB(&sqliteConnector{
		dsn:         dsn,
		baseDriver:  baseDriver,
		connectHook: d.connectHook,
	})
	readDB.SetMaxOpenConns(numCPU)
	readDB.SetMaxIdleConns(numCPU)

	d.writeDB = writeDB
	d.readDB = readDB

	if err := d.initSchema(schoolURN); err != nil {
		_ = d.Close()
		return nil, err
	}

	return d, nil
}

func ValidatePluginDatabaseAlias(alias string) error {
	trimmed := strings.TrimSpace(alias)
	if !pluginAliasPattern.MatchString(trimmed) {
		return fmt.Errorf("invalid plugin database.alias %q: must match ^plugin_[a-z0-9_]+$", alias)
	}
	return nil
}

func (d *CoreDAL) DBPath() string {
	return d.dbPath
}

func (d *CoreDAL) IsPluginDBAttached(alias string) bool {
	d.attachMu.RLock()
	defer d.attachMu.RUnlock()
	_, ok := d.attachedDBs[alias]
	return ok
}

func (d *CoreDAL) AttachPluginDB(ctx context.Context, alias string, dbFilePath string, maxPageCount int) error {
	if err := ValidatePluginDatabaseAlias(alias); err != nil {
		return err
	}
	if strings.TrimSpace(dbFilePath) == "" {
		return errors.New("missing mandatory parameter: dbFilePath")
	}
	if maxPageCount <= 0 {
		return fmt.Errorf("invalid maxPageCount %d: must be > 0", maxPageCount)
	}

	for {
		d.attachMu.RLock()
		_, alreadyAttached := d.attachedDBs[alias]
		currentCount := len(d.attachedDBs)
		var evictAlias string
		var lowestSeq uint64
		if !alreadyAttached && currentCount >= maxSQLiteAttachedDBs {
			for k, v := range d.attachedDBs {
				if evictAlias == "" || v.AttachSeq < lowestSeq {
					evictAlias = k
					lowestSeq = v.AttachSeq
				}
			}
		}
		d.attachMu.RUnlock()

		if evictAlias == "" {
			break
		}
		if err := d.DetachPluginDB(ctx, evictAlias); err != nil {
			return fmt.Errorf("failed to evict attached plugin DB %s to make room for %s: %w", evictAlias, alias, err)
		}
	}

	var attachedCount int
	if err := d.writeDB.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM pragma_database_list WHERE name = ?`,
		alias,
	).Scan(&attachedCount); err != nil {
		return fmt.Errorf("failed to inspect pragma_database_list on WriteDB: %w", err)
	}

	if attachedCount == 0 {
		escapedPath := strings.ReplaceAll(dbFilePath, "'", "''")
		attachSQL := fmt.Sprintf("ATTACH DATABASE '%s' AS %s;", escapedPath, alias)
		if _, err := d.writeDB.ExecContext(ctx, attachSQL); err != nil {
			return fmt.Errorf("failed to attach %s on WriteDB: %w", alias, err)
		}
	}

	pragmaSQL := fmt.Sprintf("PRAGMA %s.max_page_count = %d;", alias, maxPageCount)
	if _, err := d.writeDB.ExecContext(ctx, pragmaSQL); err != nil {
		return fmt.Errorf("failed to enforce max_page_count on WriteDB for %s: %w", alias, err)
	}

	d.attachMu.Lock()
	d.attachSeq++
	d.attachedDBs[alias] = AttachedPluginDB{
		Alias:        alias,
		FilePath:     dbFilePath,
		MaxPageCount: maxPageCount,
		AttachSeq:    d.attachSeq,
	}
	d.attachMu.Unlock()

	// Cycle idle connections in ReadDB so all subsequent connections execute ConnectHook
	d.readDB.SetMaxIdleConns(0)
	d.readDB.SetMaxIdleConns(d.numCPU)

	return nil
}

func (d *CoreDAL) DetachPluginDB(ctx context.Context, alias string) error {
	if err := ValidatePluginDatabaseAlias(alias); err != nil {
		return err
	}

	d.attachMu.Lock()
	delete(d.attachedDBs, alias)
	d.attachMu.Unlock()

	d.readDB.SetMaxIdleConns(0)
	d.readDB.SetMaxIdleConns(d.numCPU)

	var attachedCount int
	if err := d.writeDB.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM pragma_database_list WHERE name = ?`,
		alias,
	).Scan(&attachedCount); err != nil {
		return fmt.Errorf("failed to inspect pragma_database_list on WriteDB during detach: %w", err)
	}

	if attachedCount > 0 {
		detachSQL := fmt.Sprintf("DETACH DATABASE %s;", alias)
		if _, err := d.writeDB.ExecContext(ctx, detachSQL); err != nil {
			return fmt.Errorf("failed to detach %s from WriteDB: %w", alias, err)
		}
	}

	return nil
}

func resolveSchemaDir() (string, error) {
	_, callerFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.New("failed to resolve caller path for CoreDAL schema directory")
	}
	schemaDir := filepath.Clean(filepath.Join(filepath.Dir(callerFile), "..", "..", "schema", "core"))
	info, err := os.Stat(schemaDir)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("mandatory schema directory not found at %s: %w", schemaDir, err)
	}
	return schemaDir, nil
}

func (d *CoreDAL) initSchema(schoolURN string) error {
	schemaDir, err := resolveSchemaDir()
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(schemaDir)
	if err != nil {
		return fmt.Errorf("failed to read schema directory %s: %w", schemaDir, err)
	}

	var sqlFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			sqlFiles = append(sqlFiles, filepath.Join(schemaDir, entry.Name()))
		}
	}
	sort.Strings(sqlFiles)
	if len(sqlFiles) == 0 {
		return fmt.Errorf("no SQL schema files found in %s", schemaDir)
	}

	if _, err := d.writeDB.Exec(`PRAGMA journal_mode=WAL; PRAGMA synchronous=NORMAL; PRAGMA busy_timeout=5000; PRAGMA foreign_keys=ON;`); err != nil {
		return fmt.Errorf("failed to configure SQLite pragmas: %w", err)
	}

	// If an older 1-table MVP core.db exists without the normalized person_id column on pupils,
	// drop the legacy pupils table so the 45-table modular Core Schema can be created cleanly.
	var hasPersonID int
	_ = d.writeDB.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('pupils') WHERE name = 'person_id'`).Scan(&hasPersonID)
	var hasPupilsTable int
	_ = d.writeDB.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'pupils'`).Scan(&hasPupilsTable)
	if hasPupilsTable > 0 && hasPersonID == 0 {
		if _, err := d.writeDB.Exec(`DROP TABLE pupils;`); err != nil {
			return fmt.Errorf("failed to drop legacy MVP pupils table: %w", err)
		}
	}

	for _, file := range sqlFiles {
		ddlBytes, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read schema file %s: %w", filepath.Base(file), err)
		}
		if _, err := d.writeDB.Exec(string(ddlBytes)); err != nil {
			return fmt.Errorf("failed to execute schema file %s: %w", filepath.Base(file), err)
		}
	}

	var hasSysConfigUpdatedAt int
	_ = d.writeDB.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('__system_config') WHERE name = 'updated_at'`).Scan(&hasSysConfigUpdatedAt)
	if hasSysConfigUpdatedAt == 0 {
		if _, err := d.writeDB.Exec(`ALTER TABLE __system_config ADD COLUMN updated_at TEXT NOT NULL DEFAULT '';`); err != nil {
			return fmt.Errorf("failed to add updated_at column to __system_config: %w", err)
		}
		if _, err := d.writeDB.Exec(`UPDATE __system_config SET updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE updated_at = '';`); err != nil {
			return fmt.Errorf("failed to backfill updated_at on __system_config: %w", err)
		}
	}

	_, err = d.writeDB.Exec(
		`INSERT INTO __system_config (key, value, updated_at)
		 VALUES ('school_urn', ?, strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
		 ON CONFLICT(key) DO UPDATE SET
		     value = excluded.value,
		     updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')`,
		schoolURN,
	)
	if err != nil {
		return fmt.Errorf("failed to record school_urn in __system_config: %w", err)
	}

	return nil
}

func (d *CoreDAL) WriteDB() *sql.DB {
	return d.writeDB
}

func (d *CoreDAL) ReadDB() *sql.DB {
	return d.readDB
}

func (d *CoreDAL) Close() error {
	var firstErr error
	if d.readDB != nil {
		if err := d.readDB.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if d.writeDB != nil {
		if err := d.writeDB.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
