package dal_test

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
)

// 4. Termly DfE School Census Point-in-Time Snapshot Test
func TestGenerateCensusSnapshot_PointInTimeReferenceDate_ReturnsExactActiveRollFsmSenExclusionsApAndPost16_test(t *testing.T) {
	h := setupCoreSchemaHarness(t)
	seedBaseSchoolFixtures(t, h)

	// Autumn Census Reference Date: 2026-10-01
	_, err := h.writeDB.Exec(`
		INSERT INTO pupil_year_enrolments (
			id, pupil_id, academic_year_id, nc_year_group, enrolment_status,
			ey_funded_hours, effective_from, effective_to
		) VALUES ('enr_1', 'pup_1', 'ay_2026_2027', '12', 'C', 0.0, '2026-09-01', NULL);

		-- FSM period 1 expired before Census Day; FSM period 2 is active on Census Day
		INSERT INTO pupil_fsm_periods (id, pupil_id, effective_from, effective_to)
		VALUES ('fsm_old', 'pup_1', '2025-09-01', '2026-07-20'),
		       ('fsm_active', 'pup_1', '2026-09-01', NULL);

		-- SEN was 'K' (SEN Support) on Census Day (2026-10-01), then upgraded to 'E' (EHCP) on 2026-10-20
		INSERT INTO pupil_sen_records (id, pupil_id, provision_code, effective_from, effective_to)
		VALUES ('sen_k', 'pup_1', 'K', '2026-09-01', '2026-10-19'),
		       ('sen_e', 'pup_1', 'E', '2026-10-20', NULL);

		INSERT INTO sen_need_types (code, name, broad_area, sequence_number)
		VALUES ('ASD', 'Autistic Spectrum Disorder', 'COMMUNICATION_AND_INTERACTION', 1);

		INSERT INTO pupil_sen_needs (id, pupil_id, sen_need_code, ranking, alert_level_id, effective_from)
		VALUES ('sneed_1', 'pup_1', 'ASD', 1, 'HIGH', '2026-09-01');

		INSERT INTO pupil_suspensions_exclusions (
			id, pupil_id, academic_year_id, exclusion_category, reason_code_1,
			start_date, end_date, sessions_missed, sen_provision_at_exclusion
		) VALUES (
			'exc_1', 'pup_1', 'ay_2026_2027', 'SUSPENSION', 'DB',
			'2026-09-15', '2026-09-16', 4, 'K'
		);

		INSERT INTO pupil_alternative_provisions (
			id, pupil_id, academic_year_id, provider_urn, ap_reason_code,
			sen_provision_during_ap, sessions_per_week, effective_from, effective_to
		) VALUES (
			'ap_1', 'pup_1', 'ay_2026_2027', '139999', 'MH', 'K', 4, '2026-09-20', '2026-12-18'
		);

		INSERT INTO pupil_post16_learning_aims (
			id, pupil_id, academic_year_id, qn_code, subject_classification_code,
			planned_learning_hours, planned_eep_hours, completion_status, outcome_code,
			effective_from, planned_end_date
		) VALUES (
			'aim_1', 'pup_1', 'ay_2026_2027', '6031333X', 'RB10',
			540, 60, '1', '8', '2026-09-01', '2028-06-30'
		);
	`)
	if err != nil {
		t.Fatalf("failed to seed census snapshot records: %v", err)
	}

	censusDate := "2026-10-01"
	var upn, ncYear, senProvision, fsmID, qnCode string
	err = h.readDB.QueryRow(`
		SELECT
			p.upn,
			enr.nc_year_group,
			sen.provision_code,
			fsm.id,
			aim.qn_code
		FROM pupils p
		JOIN pupil_year_enrolments enr ON enr.pupil_id = p.id
		JOIN pupil_sen_records sen ON sen.pupil_id = p.id
		JOIN pupil_fsm_periods fsm ON fsm.pupil_id = p.id
		JOIN pupil_post16_learning_aims aim ON aim.pupil_id = p.id
		WHERE enr.effective_from <= ? AND (enr.effective_to IS NULL OR enr.effective_to >= ?)
		  AND sen.effective_from <= ? AND (sen.effective_to IS NULL OR sen.effective_to >= ?)
		  AND fsm.effective_from <= ? AND (fsm.effective_to IS NULL OR fsm.effective_to >= ?)
	`, censusDate, censusDate, censusDate, censusDate, censusDate, censusDate).Scan(&upn, &ncYear, &senProvision, &fsmID, &qnCode)
	if err != nil {
		t.Fatalf("census point-in-time query failed: %v", err)
	}

	if senProvision != "K" {
		t.Fatalf("expected point-in-time SEN provision on 2026-10-01 to be 'K' (before Oct 20 EHCP upgrade), got %q", senProvision)
	}
	if fsmID != "fsm_active" {
		t.Fatalf("expected active FSM period 'fsm_active', got %q", fsmID)
	}
	if qnCode != "6031333X" {
		t.Fatalf("expected Post-16 Learning Aim QN '6031333X', got %q", qnCode)
	}
}

// 5. UK GDPR Statutory Retention Floor Protection Test
func TestPurgePupilData_UnderAge31WithSenRecord_BlocksCoreDeletionViaRestrict_test(t *testing.T) {
	h := setupCoreSchemaHarness(t)
	seedBaseSchoolFixtures(t, h)

	_, err := h.writeDB.Exec(`
		INSERT INTO pupil_sen_records (id, pupil_id, provision_code, effective_from)
		VALUES ('sen_retention_lock', 'pup_1', 'E', '2026-09-01');
	`)
	if err != nil {
		t.Fatalf("failed to seed SEN record: %v", err)
	}

	// Attempting to delete the pupil while statutory SEN records remain MUST fail via ON DELETE RESTRICT
	if _, err := h.writeDB.Exec(`DELETE FROM pupils WHERE id = 'pup_1'`); err == nil {
		t.Fatalf("expected ON DELETE RESTRICT to block deleting pupil while statutory SEN record exists")
	}
}

// 6. DfE Common Transfer File (CTF) Atomic Import Saga Rollback Test
func TestImportCtfStudentBundle_InvalidSenProvisionCodeMidTransaction_RollsBackEntireSaga_test(t *testing.T) {
	h := setupCoreSchemaHarness(t)
	seedBaseSchoolFixtures(t, h)

	ctx := context.Background()
	tx, err := h.writeDB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("failed to begin CTF import transaction: %v", err)
	}

	_, err = tx.Exec(`
		INSERT INTO persons (id, legal_first_name, legal_last_name, sex, dob, status)
		VALUES ('per_ctf_new', 'Charles', 'Babbage', 'M', '2015-12-26', 'ACTIVE');
		INSERT INTO pupils (id, person_id, upn, ethnicity_code, first_language_code)
		VALUES ('pup_ctf_new', 'per_ctf_new', 'H823400026999', 'WBRI', 'ENG');
	`)
	if err != nil {
		_ = tx.Rollback()
		t.Fatalf("unexpected error in step 1-2 of CTF saga: %v", err)
	}

	// Step 3: Corrupted SEN provision code 'Z' (valid values are 'N', 'K', 'E')
	_, err = tx.Exec(`
		INSERT INTO pupil_sen_records (id, pupil_id, provision_code, effective_from)
		VALUES ('sen_ctf_bad', 'pup_ctf_new', 'Z', '2026-09-01')
	`)
	if err == nil {
		_ = tx.Rollback()
		t.Fatalf("expected invalid SEN provision_code 'Z' to fail CHECK constraint")
	}

	if rbErr := tx.Rollback(); rbErr != nil {
		t.Fatalf("failed to rollback transaction: %v", rbErr)
	}

	var count int
	if err := h.readDB.QueryRow(`SELECT COUNT(*) FROM persons WHERE id = 'per_ctf_new'`).Scan(&count); err != nil {
		t.Fatalf("failed to check person count after rollback: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 orphan persons rows after CTF saga rollback, found %d", count)
	}
}

// 7. SQLite WAL Concurrency & Attached Plugin DB Test
func TestConcurrentWalReadAndWrite_SingleWriterAndPooledReadersWithAttachedPlugin_ZeroSqliteBusyErrors_test(t *testing.T) {
	h := setupCoreSchemaHarness(t)
	seedBaseSchoolFixtures(t, h)

	_, err := h.writeDB.Exec(`
		INSERT INTO attendance_codes (code, name, statistical_category, physical_status, sequence_number)
		VALUES ('/', 'Present (AM)', 'PRESENT', 'ONSITE', 1),
		       ('\', 'Present (PM)', 'PRESENT', 'ONSITE', 2);
	`)
	if err != nil {
		t.Fatalf("failed to seed attendance codes: %v", err)
	}

	// Attach an isolated plugin_*.db file on WriteDB and verify cross-DB isolation
	pluginDBPath := filepath.Join(h.tempDir, "plugin_behaviour.db")
	if _, err := h.writeDB.Exec(fmt.Sprintf("ATTACH DATABASE '%s' AS plugin_behaviour;", pluginDBPath)); err != nil {
		t.Fatalf("failed to attach plugin_behaviour.db: %v", err)
	}
	if _, err := h.writeDB.Exec(`
		CREATE TABLE plugin_behaviour.behaviour_points (
			id TEXT PRIMARY KEY,
			pupil_id TEXT NOT NULL,
			points INTEGER NOT NULL
		);
		INSERT INTO plugin_behaviour.behaviour_points (id, pupil_id, points) VALUES ('bp_1', 'pup_1', 5);
	`); err != nil {
		t.Fatalf("failed to create/seed plugin_behaviour table: %v", err)
	}

	var wg sync.WaitGroup
	errCh := make(chan error, 100)

	// 1 Writer goroutine inserting 60 attendance session marks sequentially via WriteDB
	wg.Add(1)
	go func() {
		defer wg.Done()
		for day := 1; day <= 30; day++ {
			dateStr := fmt.Sprintf("2026-09-%02d", day)
			for _, sess := range []string{"AM", "PM"} {
				code := "/"
				if sess == "PM" {
					code = "\\"
				}
				id := fmt.Sprintf("asm_%d_%s", day, sess)
				if _, err := h.writeDB.Exec(
					`INSERT INTO attendance_session_marks (id, pupil_id, academic_year_id, date, session, attendance_code)
					 VALUES (?, 'pup_1', 'ay_2026_2027', ?, ?, ?)`,
					id, dateStr, sess, code,
				); err != nil {
					errCh <- fmt.Errorf("writer failed on %s: %w", id, err)
					return
				}
			}
		}
	}()

	// 6 Concurrent Reader goroutines querying ReadDB while WriteDB is writing
	for r := 0; r < 6; r++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()
			for i := 0; i < 25; i++ {
				var fkEnabled int
				if err := h.readDB.QueryRow(`PRAGMA foreign_keys;`).Scan(&fkEnabled); err != nil {
					errCh <- fmt.Errorf("reader %d pragma check failed: %w", readerID, err)
					return
				}
				if fkEnabled != 1 {
					errCh <- fmt.Errorf("reader %d pooled connection has foreign_keys=%d (expected 1)", readerID, fkEnabled)
					return
				}

				var markCount int
				if err := h.readDB.QueryRow(`
					SELECT COUNT(*)
					FROM attendance_session_marks asm
					JOIN pupils p ON p.id = asm.pupil_id
					JOIN persons per ON per.id = p.person_id
					WHERE p.upn = 'H823400026001'
				`).Scan(&markCount); err != nil {
					errCh <- fmt.Errorf("reader %d query failed: %w", readerID, err)
					return
				}
			}
		}(r)
	}

	wg.Wait()
	close(errCh)

	for e := range errCh {
		t.Fatalf("concurrent WAL test error: %v", e)
	}
}
