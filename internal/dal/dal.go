package dal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

type Pupil struct {
	ID             string  `json:"id"`
	UPN            string  `json:"upn"`
	FirstName      string  `json:"first_name"`
	LastName       string  `json:"last_name"`
	AcademicYearID string  `json:"academic_year_id"`
	EffectiveFrom  string  `json:"effective_from"`
	EffectiveTo    *string `json:"effective_to,omitempty"`
}

type SeedPupilInput struct {
	UPN            string `json:"upn"`
	FirstName      string `json:"first_name"`
	LastName       string `json:"last_name"`
	AcademicYearID string `json:"academic_year_id"`
	EffectiveFrom  string `json:"effective_from"`
}

type CoreDAL struct {
	writeDB *sql.DB
	readDB  *sql.DB
	dbPath  string
}

func NewCoreDAL(dataDirPath string, schoolURN string) (*CoreDAL, error) {
	if strings.TrimSpace(dataDirPath) == "" {
		return nil, errors.New("mandatory parameter dataDirPath is missing")
	}
	if strings.TrimSpace(schoolURN) == "" {
		return nil, errors.New("mandatory parameter schoolURN is missing")
	}

	dbPath := filepath.Join(dataDirPath, "core.db")
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(5000)", dbPath)

	writeDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open WriteDB: %w", err)
	}
	writeDB.SetMaxOpenConns(1)
	writeDB.SetMaxIdleConns(1)

	readDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		_ = writeDB.Close()
		return nil, fmt.Errorf("failed to open ReadDB: %w", err)
	}
	numCPU := runtime.NumCPU()
	if numCPU < 1 {
		numCPU = 1
	}
	readDB.SetMaxOpenConns(numCPU)

	d := &CoreDAL{
		writeDB: writeDB,
		readDB:  readDB,
		dbPath:  dbPath,
	}

	if err := d.initSchema(schoolURN); err != nil {
		_ = d.Close()
		return nil, err
	}

	return d, nil
}

func (d *CoreDAL) initSchema(schoolURN string) error {
	ddl := `
	PRAGMA journal_mode=WAL;
	PRAGMA synchronous=NORMAL;
	PRAGMA busy_timeout=5000;

	CREATE TABLE IF NOT EXISTS __system_config (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS pupils (
		id TEXT PRIMARY KEY,
		upn TEXT NOT NULL,
		first_name TEXT NOT NULL,
		last_name TEXT NOT NULL,
		academic_year_id TEXT NOT NULL,
		effective_from TEXT NOT NULL,
		effective_to TEXT
	);
	`
	if _, err := d.writeDB.Exec(ddl); err != nil {
		return fmt.Errorf("failed to initialize core.db schema: %w", err)
	}

	_, err := d.writeDB.Exec(
		`INSERT INTO __system_config (key, value) VALUES ('school_urn', ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
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

func (d *CoreDAL) SeedPupil(ctx context.Context, in SeedPupilInput) (*Pupil, error) {
	if strings.TrimSpace(in.UPN) == "" {
		return nil, errors.New("missing mandatory field: upn")
	}
	if strings.TrimSpace(in.FirstName) == "" {
		return nil, errors.New("missing mandatory field: first_name")
	}
	if strings.TrimSpace(in.LastName) == "" {
		return nil, errors.New("missing mandatory field: last_name")
	}
	if strings.TrimSpace(in.AcademicYearID) == "" {
		return nil, errors.New("missing mandatory field: academic_year_id")
	}
	if strings.TrimSpace(in.EffectiveFrom) == "" {
		return nil, errors.New("missing mandatory field: effective_from")
	}

	id := "pup_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]

	_, err := d.writeDB.ExecContext(
		ctx,
		`INSERT INTO pupils (id, upn, first_name, last_name, academic_year_id, effective_from, effective_to)
		 VALUES (?, ?, ?, ?, ?, ?, NULL)`,
		id, in.UPN, in.FirstName, in.LastName, in.AcademicYearID, in.EffectiveFrom,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert pupil: %w", err)
	}

	return &Pupil{
		ID:             id,
		UPN:            in.UPN,
		FirstName:      in.FirstName,
		LastName:       in.LastName,
		AcademicYearID: in.AcademicYearID,
		EffectiveFrom:  in.EffectiveFrom,
	}, nil
}

func (d *CoreDAL) DeletePupil(ctx context.Context, pupilID string) error {
	if strings.TrimSpace(pupilID) == "" {
		return errors.New("missing mandatory parameter: pupil_id")
	}
	_, err := d.writeDB.ExecContext(ctx, `DELETE FROM pupils WHERE id = ?`, pupilID)
	return err
}

func (d *CoreDAL) GetPupilByID(ctx context.Context, pupilID string) (*Pupil, error) {
	if strings.TrimSpace(pupilID) == "" {
		return nil, errors.New("missing mandatory parameter: pupil_id")
	}

	row := d.readDB.QueryRowContext(
		ctx,
		`SELECT id, upn, first_name, last_name, academic_year_id, effective_from, effective_to
		 FROM pupils
		 WHERE id = ? AND (effective_to IS NULL OR effective_to >= date('now'))`,
		pupilID,
	)

	var p Pupil
	var effTo sql.NullString
	if err := row.Scan(&p.ID, &p.UPN, &p.FirstName, &p.LastName, &p.AcademicYearID, &p.EffectiveFrom, &effTo); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("pupil not found: %s", pupilID)
		}
		return nil, fmt.Errorf("failed to query pupil: %w", err)
	}
	if effTo.Valid {
		val := effTo.String
		p.EffectiveTo = &val
	}
	return &p, nil
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
