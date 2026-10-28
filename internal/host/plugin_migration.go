package host

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"optimis/internal/dal"
)

func copyFileSync(srcPath string, dstPath string) error {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", filepath.Base(srcPath), err)
	}
	defer srcFile.Close()

	dstFile, err := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open destination file %s: %w", filepath.Base(dstPath), err)
	}

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		_ = dstFile.Close()
		return fmt.Errorf("failed to copy %s to %s: %w", filepath.Base(srcPath), filepath.Base(dstPath), err)
	}
	if err := dstFile.Sync(); err != nil {
		_ = dstFile.Close()
		return fmt.Errorf("failed to sync %s: %w", filepath.Base(dstPath), err)
	}
	return dstFile.Close()
}

func computeMaxPageCount(maxDBSizeMB int, pageSize int) int {
	if maxDBSizeMB <= 0 {
		maxDBSizeMB = defaultMaxDBSizeMB
	}
	if pageSize <= 0 {
		pageSize = defaultSQLitePageSz
	}
	pages := (maxDBSizeMB * 1024 * 1024) / pageSize
	if pages < 1 {
		pages = 1
	}
	return pages
}

func (h *Host) provisionAndMigratePluginDB(ctx context.Context, plugin *LoadedPlugin) error {
	if plugin.Manifest.Database == nil || strings.TrimSpace(plugin.Manifest.Database.Alias) == "" {
		return nil
	}

	alias := strings.TrimSpace(plugin.Manifest.Database.Alias)
	if err := dal.ValidatePluginDatabaseAlias(alias); err != nil {
		return err
	}

	maxDBSizeMB := plugin.Manifest.Database.MaxDBSizeMB
	if maxDBSizeMB <= 0 {
		maxDBSizeMB = defaultMaxDBSizeMB
	}

	dbFilePath := filepath.Join(h.cfg.DataDirPath, alias+".db")
	snapshotPath := dbFilePath + ".snapshot"

	if h.coreDAL.IsPluginDBAttached(alias) {
		if err := h.coreDAL.DetachPluginDB(ctx, alias); err != nil {
			return fmt.Errorf("failed to detach %s prior to migration check: %w", alias, err)
		}
	}

	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)", dbFilePath)
	bootDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return fmt.Errorf("failed to open plugin DB %s: %w", alias, err)
	}
	bootDB.SetMaxOpenConns(1)

	var pageSize int
	if err := bootDB.QueryRowContext(ctx, `PRAGMA page_size;`).Scan(&pageSize); err != nil || pageSize <= 0 {
		pageSize = defaultSQLitePageSz
	}
	maxPageCount := computeMaxPageCount(maxDBSizeMB, pageSize)

	if _, err := bootDB.ExecContext(ctx, `PRAGMA journal_mode=WAL; PRAGMA synchronous=NORMAL; PRAGMA busy_timeout=5000; PRAGMA foreign_keys=ON;`); err != nil {
		_ = bootDB.Close()
		return fmt.Errorf("failed to configure WAL pragmas on %s.db: %w", alias, err)
	}
	if _, err := bootDB.ExecContext(ctx, fmt.Sprintf(`PRAGMA max_page_count = %d;`, maxPageCount)); err != nil {
		_ = bootDB.Close()
		return fmt.Errorf("failed to set max_page_count on %s.db: %w", alias, err)
	}
	if _, err := bootDB.ExecContext(
		ctx,
		`CREATE TABLE IF NOT EXISTS __plugin_migrations (
			filename TEXT PRIMARY KEY,
			applied_at TEXT NOT NULL
		);`,
	); err != nil {
		_ = bootDB.Close()
		return fmt.Errorf("failed to ensure __plugin_migrations table in %s.db: %w", alias, err)
	}

	appliedMigrations := make(map[string]bool)
	rows, err := bootDB.QueryContext(ctx, `SELECT filename FROM __plugin_migrations`)
	if err != nil {
		_ = bootDB.Close()
		return fmt.Errorf("failed to query applied migrations in %s.db: %w", alias, err)
	}
	for rows.Next() {
		var fn string
		if err := rows.Scan(&fn); err != nil {
			_ = rows.Close()
			_ = bootDB.Close()
			return fmt.Errorf("failed to scan migration filename: %w", err)
		}
		appliedMigrations[fn] = true
	}
	_ = rows.Close()

	_, _ = bootDB.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE);`)
	if err := bootDB.Close(); err != nil {
		return fmt.Errorf("failed to close bootstrap handle for %s.db: %w", alias, err)
	}

	migrationsDir := filepath.Join(plugin.DirPath, "migrations")
	var pendingFiles []string
	if info, statErr := os.Stat(migrationsDir); statErr == nil && info.IsDir() {
		entries, readErr := os.ReadDir(migrationsDir)
		if readErr != nil {
			return fmt.Errorf("failed to read migrations dir for %s: %w", plugin.Manifest.ID, readErr)
		}
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
				if !appliedMigrations[entry.Name()] {
					pendingFiles = append(pendingFiles, filepath.Join(migrationsDir, entry.Name()))
				}
			}
		}
		sort.Strings(pendingFiles)
	}

	if len(pendingFiles) > 0 {
		if err := copyFileSync(dbFilePath, snapshotPath); err != nil {
			return fmt.Errorf("failed to create file-level snapshot for %s.db: %w", alias, err)
		}
		h.logInfo(fmt.Sprintf("Created migration file snapshot %s", filepath.Base(snapshotPath)))

		migErr := h.runMigrationsInImmediateTx(ctx, dsn, maxPageCount, pendingFiles)
		if migErr != nil {
			_ = os.Remove(dbFilePath + "-wal")
			_ = os.Remove(dbFilePath + "-shm")
			if restoreErr := copyFileSync(snapshotPath, dbFilePath); restoreErr != nil {
				_ = os.Remove(snapshotPath)
				return fmt.Errorf("migration failed (%v) and snapshot restore failed: %w", migErr, restoreErr)
			}
			_ = os.Remove(snapshotPath)
			_ = h.coreDAL.AttachPluginDB(ctx, alias, dbFilePath, maxPageCount)
			h.recordDiagnostic(plugin.Manifest.ID, "migrations", "MIGRATION_ROLLBACK", 0)
			h.logError(fmt.Sprintf("Rolled back %s.db from file snapshot after migration failure: %v", alias, migErr))
			return fmt.Errorf("plugin %s migration failed (rolled back from snapshot): %w", plugin.Manifest.ID, migErr)
		}

		_ = os.Remove(snapshotPath)
		h.logInfo(fmt.Sprintf("Applied %d migration(s) to %s.db inside BEGIN IMMEDIATE", len(pendingFiles), alias))
	}

	if err := h.coreDAL.AttachPluginDB(ctx, alias, dbFilePath, maxPageCount); err != nil {
		return err
	}
	h.logInfo(fmt.Sprintf("Attached %s.db (max_page_count=%d) across WriteDB and ReadDB", alias, maxPageCount))
	return nil
}

func (h *Host) runMigrationsInImmediateTx(ctx context.Context, dsn string, maxPageCount int, pendingFiles []string) error {
	migDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return err
	}
	migDB.SetMaxOpenConns(1)
	defer migDB.Close()

	conn, err := migDB.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, fmt.Sprintf(`PRAGMA max_page_count = %d;`, maxPageCount)); err != nil {
		return err
	}

	if _, err := conn.ExecContext(ctx, `BEGIN IMMEDIATE;`); err != nil {
		return fmt.Errorf("failed to begin immediate transaction: %w", err)
	}

	for _, file := range pendingFiles {
		sqlBytes, readErr := os.ReadFile(file)
		if readErr != nil {
			_, _ = conn.ExecContext(ctx, `ROLLBACK;`)
			return fmt.Errorf("failed to read migration %s: %w", filepath.Base(file), readErr)
		}
		sqlText := strings.TrimSpace(string(sqlBytes))
		if sqlText == "" {
			_, _ = conn.ExecContext(ctx, `ROLLBACK;`)
			return fmt.Errorf("migration file %s is empty", filepath.Base(file))
		}
		if _, execErr := conn.ExecContext(ctx, sqlText); execErr != nil {
			_, _ = conn.ExecContext(ctx, `ROLLBACK;`)
			return fmt.Errorf("failed executing migration %s: %w", filepath.Base(file), execErr)
		}
		if _, recErr := conn.ExecContext(
			ctx,
			`INSERT INTO __plugin_migrations (filename, applied_at) VALUES (?, ?)`,
			filepath.Base(file),
			time.Now().UTC().Format(time.RFC3339Nano),
		); recErr != nil {
			_, _ = conn.ExecContext(ctx, `ROLLBACK;`)
			return fmt.Errorf("failed recording migration %s: %w", filepath.Base(file), recErr)
		}
	}

	if _, err := conn.ExecContext(ctx, `COMMIT;`); err != nil {
		_, _ = conn.ExecContext(ctx, `ROLLBACK;`)
		return fmt.Errorf("failed to commit migration transaction: %w", err)
	}

	_, _ = conn.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE);`)
	return nil
}
