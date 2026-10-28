package main

import (
	"bufio"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

func loadDotEnv(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("mandatory .env file missing at %s: %w", path, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, val)
		}
	}
	return scanner.Err()
}

func requireEnv(key string) (string, error) {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return "", fmt.Errorf("missing mandatory environment variable: %s", key)
	}
	return val, nil
}

func run() error {
	var customDBPath string
	var overwriteDevDB bool
	var seedVal uint64

	flag.StringVar(&customDBPath, "db", "", "Explicit target SQLite database path (overrides default <DATA_DIR_PATH>/qa_core.db)")
	flag.BoolVar(&overwriteDevDB, "overwrite-dev-db", false, "Write directly to <DATA_DIR_PATH>/core.db instead of <DATA_DIR_PATH>/qa_core.db")
	flag.Uint64Var(&seedVal, "seed", 42, "Deterministic PRNG seed for synthetic school cohort generation")
	flag.Parse()

	if err := loadDotEnv(".env"); err != nil {
		return err
	}

	dataDirPath, err := requireEnv("DATA_DIR_PATH")
	if err != nil {
		return err
	}
	schoolURN, err := requireEnv("SCHOOL_URN")
	if err != nil {
		return err
	}
	kmsKeyHandle, err := requireEnv("HOST_KMS_KEY_HANDLE")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dataDirPath, 0o755); err != nil {
		return fmt.Errorf("failed to create DATA_DIR_PATH %s: %w", dataDirPath, err)
	}

	targetDBPath := filepath.Join(dataDirPath, "qa_core.db")
	if overwriteDevDB {
		targetDBPath = filepath.Join(dataDirPath, "core.db")
	}
	if strings.TrimSpace(customDBPath) != "" {
		targetDBPath = strings.TrimSpace(customDBPath)
	}

	// Clean any prior target DB and WAL/SHM artifacts so QA starts from a clean deterministic state
	for _, suffix := range []string{"", "-wal", "-shm"} {
		_ = os.Remove(targetDBPath + suffix)
	}

	schemaDir := filepath.Clean("schema/core")
	entries, err := os.ReadDir(schemaDir)
	if err != nil {
		return fmt.Errorf("failed to read schema directory %s: %w", schemaDir, err)
	}
	var sqlFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			sqlFiles = append(sqlFiles, filepath.Join(schemaDir, e.Name()))
		}
	}
	sort.Strings(sqlFiles)
	if len(sqlFiles) != 7 {
		return fmt.Errorf("expected 7 core schema SQL files in %s, found %d", schemaDir, len(sqlFiles))
	}

	start := time.Now()
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)", targetDBPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return fmt.Errorf("failed to open target SQLite database %s: %w", targetDBPath, err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	if _, err := db.Exec(`PRAGMA journal_mode=WAL; PRAGMA synchronous=NORMAL; PRAGMA busy_timeout=5000; PRAGMA foreign_keys=ON;`); err != nil {
		return fmt.Errorf("failed to set SQLite pragmas: %w", err)
	}

	for _, f := range sqlFiles {
		ddl, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", f, err)
		}
		if _, err := db.Exec(string(ddl)); err != nil {
			return fmt.Errorf("failed to execute DDL %s: %w", filepath.Base(f), err)
		}
	}

	st := &qaSeedState{
		schoolURN:      schoolURN,
		kmsKeyHandle:   kmsKeyHandle,
		gen:            newDetGen(seedVal),
		pupilByID:      make(map[string]*seededPupilMeta),
		slotIDsByGroup: make(map[string][]string),
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin seed transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if err := seedDomain00SystemAndCalendar(tx, st); err != nil {
		return fmt.Errorf("Domain 00 seed failed: %w", err)
	}
	if err := seedDomain01And03StaffGroupsAndPupils(tx, st); err != nil {
		return fmt.Errorf("Domain 01/03 seed failed: %w", err)
	}
	if err := seedDomain02HouseholdsAndGuardians(tx, st); err != nil {
		return fmt.Errorf("Domain 02 seed failed: %w", err)
	}
	if err := seedDomain04StatutoryAttendance(tx, st); err != nil {
		return fmt.Errorf("Domain 04 seed failed: %w", err)
	}
	if err := seedDomain05And06SenMedicalCensusAndSafeguarding(tx, st); err != nil {
		return fmt.Errorf("Domain 05/06 seed failed: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit seed transaction: %w", err)
	}

	// Post-Seed Integrity Verification: Foreign Key Check + 45-Table Non-Empty Verification
	fkRows, err := db.Query(`PRAGMA foreign_key_check;`)
	if err != nil {
		return fmt.Errorf("PRAGMA foreign_key_check failed: %w", err)
	}
	defer fkRows.Close()
	if fkRows.Next() {
		var tbl, parent string
		var rowid, fkid int
		_ = fkRows.Scan(&tbl, &rowid, &parent, &fkid)
		return fmt.Errorf("foreign key violation detected in table %s (rowid=%d -> %s)", tbl, rowid, parent)
	}

	tblRows, err := db.Query(`SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' ORDER BY name`)
	if err != nil {
		return err
	}
	var tables []string
	for tblRows.Next() {
		var name string
		if err := tblRows.Scan(&name); err != nil {
			tblRows.Close()
			return err
		}
		tables = append(tables, name)
	}
	tblRows.Close()

	if len(tables) != 45 {
		return fmt.Errorf("expected 45 tables in seeded database, found %d", len(tables))
	}

	totalRows := 0
	counts := make(map[string]int, len(tables))
	for _, tName := range tables {
		var c int
		if err := db.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM "%s"`, tName)).Scan(&c); err != nil {
			return fmt.Errorf("failed to count rows in %s: %w", tName, err)
		}
		if c == 0 {
			return fmt.Errorf("post-seed verification failed: table %s has 0 rows", tName)
		}
		counts[tName] = c
		totalRows += c
	}

	elapsed := time.Since(start)
	printQACheatSheet(targetDBPath, seedVal, elapsed, tables, counts, totalRows)
	return nil
}

func printQACheatSheet(dbPath string, seed uint64, elapsed time.Duration, tables []string, counts map[string]int, totalRows int) {
	fmt.Printf("================================================================================\n")
	fmt.Printf(" OPTIMIS QA SEED COMPLETE — Oakridge All-Through Academy (URN 100001)\n")
	fmt.Printf("================================================================================\n")
	fmt.Printf(" Target Database : %s\n", dbPath)
	fmt.Printf(" PRNG Seed       : %d\n", seed)
	fmt.Printf(" Tables Populated: %d / 45 (0 FK violations)\n", len(tables))
	fmt.Printf(" Total DB Rows   : %d rows inserted in %s\n", totalRows, elapsed.Round(time.Millisecond))
	fmt.Printf("--------------------------------------------------------------------------------\n")
	fmt.Printf(" PINNED STAFF PERSONAS (Persona Switcher & RBAC Testing)\n")
	fmt.Printf("   stf_admin_01   | Dame Eleanor Vance | ADMIN   | Executive Headteacher (:all)\n")
	fmt.Printf("   stf_dsl_01     | Mr Marcus Thorne   | DSL     | Safeguarding Vault Lead (is_dsl=1)\n")
	fmt.Printf("   stf_senco_01   | Dr Priya Nair      | SENCO   | Whole-School SENCO (SEN :all)\n")
	fmt.Printf("   stf_hod_maths  | Mr David Sterling  | HOD     | Head of Maths (10AMa, KS5 Maths)\n")
	fmt.Printf("   stf_tch_7a     | Ms Hannah Abbott   | TEACHER | Form 7A Tutor & 7AMa (:own_groups)\n")
	fmt.Printf("   stf_tch_7b     | Mr Simon Barker    | TEACHER | Form 7B Tutor & 7BEn (Negative Control)\n")
	fmt.Printf("   stf_ta_01      | Mrs Sarah Jenkins  | TA      | HLTA assigned to Noah (pup_yr7_noah)\n")
	fmt.Printf("   stf_supply_01  | Mr Gareth Cole     | TEACHER | Agency Cover Teacher\n")
	fmt.Printf("--------------------------------------------------------------------------------\n")
	fmt.Printf(" PINNED EDGE-CASE PUPIL ARCHETYPES\n")
	fmt.Printf("   pup_eyfs_leo    | Leo Carter    (N2A) | EYFS 30hr Funded + DAF Indicator\n")
	fmt.Printf("   pup_pri_maya    | Maya Patel    (4A)  | FSM + PP + Critical Peanut Anaphylaxis IHCP\n")
	fmt.Printf("   pup_yr7_alice   | Alice Evans   (7A)  | Refuge Address + Court-Order Father + Vault Note\n")
	fmt.Printf("   pup_yr7_noah    | Noah Williams (7A)  | EHCP (E) ASD/SEMH + TA stf_ta_01 + INT-LIT-7\n")
	fmt.Printf("   pup_yr7_zara    | Zara Khan     (7B)  | :own_groups Negative Control + EAL + SEN (K)\n")
	fmt.Printf("   pup_yr7_liam    | Liam O'Connor (7B)  | Temporal Transfer (7A -> 7B on 2026-10-01)\n")
	fmt.Printf("   pup_yr10_ryan   | Ryan Mitchell (10A) | LAC Foster Care + Dual Roll (S) + AP + Suspension\n")
	fmt.Printf("   pup_yr12_chloe  | Chloe Sterling(12A) | KS5 Post-16 Learning Aims (3 A-Levels)\n")
	fmt.Printf("   pup_left_oliver | Oliver Grant  (10B) | Off-Rolled Historical Pupil (LEFT 2026-10-02)\n")
	fmt.Printf("--------------------------------------------------------------------------------\n")
	fmt.Printf(" ROW COUNTS ACROSS ALL 45 CORE TABLES\n")
	for i := 0; i < len(tables); i += 2 {
		left := fmt.Sprintf(" %-33s : %6d", tables[i], counts[tables[i]])
		right := ""
		if i+1 < len(tables) {
			right = fmt.Sprintf(" | %-33s : %6d", tables[i+1], counts[tables[i+1]])
		}
		fmt.Printf("%s%s\n", left, right)
	}
	fmt.Printf("================================================================================\n")
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: %v\n", err)
		os.Exit(1)
	}
}
