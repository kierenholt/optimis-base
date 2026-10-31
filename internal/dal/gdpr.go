package dal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

type SARSection struct {
	SourceID         string           `json:"source_id"`
	SourceType       string           `json:"source_type"`
	PluginState      string           `json:"plugin_state,omitempty"`
	Title            string           `json:"title"`
	Records          []map[string]any `json:"records"`
	RedactableFields []string         `json:"redactable_fields"`
}

func (d *CoreDAL) ExportCorePupilSAR(ctx context.Context, pupilID string) ([]SARSection, error) {
	trimmedPupilID := strings.TrimSpace(pupilID)
	if trimmedPupilID == "" {
		return nil, errors.New("missing mandatory parameter: pupil_id")
	}

	// 1. Verify pupil exists and export Core Pupil Profile & Enrolments
	var upn, firstName, lastName, dob, sex, ethnicity, firstLang string
	err := d.readDB.QueryRowContext(
		ctx,
		`SELECT p.upn, per.legal_first_name, per.legal_last_name, COALESCE(per.dob, ''),
		        per.sex, p.ethnicity_code, p.first_language_code
		 FROM pupils p
		 JOIN persons per ON per.id = p.person_id
		 WHERE p.id = ?`,
		trimmedPupilID,
	).Scan(&upn, &firstName, &lastName, &dob, &sex, &ethnicity, &firstLang)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("pupil not found: %s", trimmedPupilID)
		}
		return nil, fmt.Errorf("failed to query core pupil profile for SAR: %w", err)
	}

	enrRows, err := d.readDB.QueryContext(
		ctx,
		`SELECT id, academic_year_id, nc_year_group, enrolment_status, effective_from, COALESCE(effective_to, '')
		 FROM pupil_year_enrolments
		 WHERE pupil_id = ?
		 ORDER BY effective_from DESC, id ASC`,
		trimmedPupilID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query pupil_year_enrolments for SAR: %w", err)
	}
	var profileRecords []map[string]any
	for enrRows.Next() {
		var enrID, ayID, ncYear, status, effFrom, effTo string
		if scanErr := enrRows.Scan(&enrID, &ayID, &ncYear, &status, &effFrom, &effTo); scanErr != nil {
			_ = enrRows.Close()
			return nil, scanErr
		}
		profileRecords = append(profileRecords, map[string]any{
			"pupil_id":            trimmedPupilID,
			"upn":                 upn,
			"first_name":          firstName,
			"last_name":           lastName,
			"dob":                 dob,
			"sex":                 sex,
			"ethnicity_code":      ethnicity,
			"first_language_code": firstLang,
			"enrolment_id":        enrID,
			"academic_year_id":    ayID,
			"nc_year_group":       ncYear,
			"enrolment_status":    status,
			"effective_from":      effFrom,
			"effective_to":        effTo,
		})
	}
	_ = enrRows.Close()
	if len(profileRecords) == 0 {
		profileRecords = append(profileRecords, map[string]any{
			"pupil_id":            trimmedPupilID,
			"upn":                 upn,
			"first_name":          firstName,
			"last_name":           lastName,
			"dob":                 dob,
			"sex":                 sex,
			"ethnicity_code":      ethnicity,
			"first_language_code": firstLang,
		})
	}

	// 2. Core Guardians & Household Contacts (enforcing court_order_restricted = 0 & confidential address redaction)
	guardianRows, err := d.readDB.QueryContext(
		ctx,
		`SELECT gr.id, gr.relationship_code, per.legal_first_name, per.legal_last_name,
		        COALESCE(per.email, ''), per.phone_primary, gr.has_parental_responsibility,
		        COALESCE(h.name, ''), COALESCE(h.address_line_1, ''), COALESCE(h.postcode, ''),
		        COALESCE(h.is_confidential_address, 0)
		 FROM guardian_relationships gr
		 JOIN persons per ON per.id = gr.guardian_person_id
		 LEFT JOIN household_memberships hm ON hm.person_id = gr.guardian_person_id
		 LEFT JOIN households h ON h.id = hm.household_id
		 WHERE gr.pupil_id = ?
		   AND gr.court_order_restricted = 0
		 ORDER BY gr.contact_priority ASC, gr.id ASC`,
		trimmedPupilID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query guardian_relationships for SAR: %w", err)
	}
	guardianRecords := make([]map[string]any, 0)
	for guardianRows.Next() {
		var relID, relCode, gFirst, gLast, gEmail, gPhone, hName, addr1, postcode string
		var hasPR, isConf int
		if scanErr := guardianRows.Scan(&relID, &relCode, &gFirst, &gLast, &gEmail, &gPhone, &hasPR, &hName, &addr1, &postcode, &isConf); scanErr != nil {
			_ = guardianRows.Close()
			return nil, scanErr
		}
		if isConf == 1 {
			hName = "[CONFIDENTIAL ADDRESS REDACTED]"
			addr1 = "[CONFIDENTIAL ADDRESS REDACTED]"
			postcode = "[CONFIDENTIAL ADDRESS REDACTED]"
		}
		guardianRecords = append(guardianRecords, map[string]any{
			"relationship_id":             relID,
			"relationship_code":           relCode,
			"guardian_name":               strings.TrimSpace(gFirst + " " + gLast),
			"email":                       gEmail,
			"phone_primary":               gPhone,
			"has_parental_responsibility": hasPR == 1,
			"household_name":              hName,
			"address_line_1":              addr1,
			"postcode":                    postcode,
		})
	}
	_ = guardianRows.Close()

	// 3. Core Statutory Session Attendance (`AM`/`PM`)
	attRows, err := d.readDB.QueryContext(
		ctx,
		`SELECT id, date, session, attendance_code, minutes_late, comment
		 FROM attendance_session_marks
		 WHERE pupil_id = ?
		 ORDER BY date ASC, session ASC`,
		trimmedPupilID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query attendance_session_marks for SAR: %w", err)
	}
	attRecords := make([]map[string]any, 0)
	for attRows.Next() {
		var id, date, session, code, comment string
		var minsLate int
		if scanErr := attRows.Scan(&id, &date, &session, &code, &minsLate, &comment); scanErr != nil {
			_ = attRows.Close()
			return nil, scanErr
		}
		attRecords = append(attRecords, map[string]any{
			"id":              id,
			"date":            date,
			"session":         session,
			"attendance_code": code,
			"minutes_late":    minsLate,
			"comment":         comment,
		})
	}
	_ = attRows.Close()

	// 4. Core SEN & Medical Records
	senMedRecords := make([]map[string]any, 0)
	senRows, err := d.readDB.QueryContext(
		ctx,
		`SELECT id, provision_code, teaching_strategies, pupil_targets, senco_notes, effective_from
		 FROM pupil_sen_records
		 WHERE pupil_id = ?
		 ORDER BY effective_from DESC, id ASC`,
		trimmedPupilID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query pupil_sen_records for SAR: %w", err)
	}
	for senRows.Next() {
		var id, prov, strategies, targets, sencoNotes, effFrom string
		if scanErr := senRows.Scan(&id, &prov, &strategies, &targets, &sencoNotes, &effFrom); scanErr != nil {
			_ = senRows.Close()
			return nil, scanErr
		}
		senMedRecords = append(senMedRecords, map[string]any{
			"record_type":         "SEN",
			"id":                  id,
			"provision_code":      prov,
			"teaching_strategies": strategies,
			"pupil_targets":       targets,
			"senco_notes":         sencoNotes,
			"effective_from":      effFrom,
		})
	}
	_ = senRows.Close()

	medRows, err := d.readDB.QueryContext(
		ctx,
		`SELECT id, condition_name, condition_category, emergency_response_plan, effective_from
		 FROM pupil_medical_conditions
		 WHERE pupil_id = ?
		 ORDER BY effective_from DESC, id ASC`,
		trimmedPupilID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query pupil_medical_conditions for SAR: %w", err)
	}
	for medRows.Next() {
		var id, condName, condCat, erp, effFrom string
		if scanErr := medRows.Scan(&id, &condName, &condCat, &erp, &effFrom); scanErr != nil {
			_ = medRows.Close()
			return nil, scanErr
		}
		senMedRecords = append(senMedRecords, map[string]any{
			"record_type":             "MEDICAL",
			"id":                      id,
			"condition_name":          condName,
			"condition_category":      condCat,
			"emergency_response_plan": erp,
			"effective_from":          effFrom,
		})
	}
	_ = medRows.Close()

	// 5. Core Pastoral Notes (redactable free-text fields: title, note_text)
	pastoralRows, err := d.readDB.QueryContext(
		ctx,
		`SELECT id, category, title, note_text, created_by_staff_id, created_at
		 FROM pupil_pastoral_notes
		 WHERE pupil_id = ?
		 ORDER BY created_at ASC, id ASC`,
		trimmedPupilID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query pupil_pastoral_notes for SAR: %w", err)
	}
	pastoralRecords := make([]map[string]any, 0)
	for pastoralRows.Next() {
		var id, cat, title, noteText, staffID, createdAt string
		if scanErr := pastoralRows.Scan(&id, &cat, &title, &noteText, &staffID, &createdAt); scanErr != nil {
			_ = pastoralRows.Close()
			return nil, scanErr
		}
		pastoralRecords = append(pastoralRecords, map[string]any{
			"id":                  id,
			"category":            cat,
			"title":               title,
			"note_text":           noteText,
			"created_by_staff_id": staffID,
			"created_at":          createdAt,
		})
	}
	_ = pastoralRows.Close()

	return []SARSection{
		{
			SourceID:         "core",
			SourceType:       "CORE",
			Title:            "Core Pupil Profile & Enrolments",
			Records:          profileRecords,
			RedactableFields: []string{},
		},
		{
			SourceID:         "core",
			SourceType:       "CORE",
			Title:            "Core Guardians & Household Contacts",
			Records:          guardianRecords,
			RedactableFields: []string{"guardian_name", "email", "phone_primary", "address_line_1"},
		},
		{
			SourceID:         "core",
			SourceType:       "CORE",
			Title:            "Core Statutory Session Attendance",
			Records:          attRecords,
			RedactableFields: []string{"comment"},
		},
		{
			SourceID:         "core",
			SourceType:       "CORE",
			Title:            "Core SEN & Medical Records",
			Records:          senMedRecords,
			RedactableFields: []string{"teaching_strategies", "pupil_targets", "senco_notes", "emergency_response_plan"},
		},
		{
			SourceID:         "core",
			SourceType:       "CORE",
			Title:            "Core Pastoral Notes",
			Records:          pastoralRecords,
			RedactableFields: []string{"title", "note_text"},
		},
	}, nil
}

func (d *CoreDAL) CheckPupilRetentionPurgeEligible(ctx context.Context, pupilID string) error {
	trimmedPupilID := strings.TrimSpace(pupilID)
	if trimmedPupilID == "" {
		return errors.New("missing mandatory parameter: pupil_id")
	}

	var exists int
	if err := d.readDB.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM pupils WHERE id = ?`,
		trimmedPupilID,
	).Scan(&exists); err != nil {
		return fmt.Errorf("failed to check pupil existence: %w", err)
	}
	if exists == 0 {
		return fmt.Errorf("pupil not found: %s", trimmedPupilID)
	}

	var lockedSafeguardingCount int
	if err := d.readDB.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM safeguarding_notes WHERE pupil_id = ? AND retention_purge_eligible = 0`,
		trimmedPupilID,
	).Scan(&lockedSafeguardingCount); err != nil {
		return fmt.Errorf("failed to inspect safeguarding_notes retention status: %w", err)
	}
	if lockedSafeguardingCount > 0 {
		return fmt.Errorf("KCSIE_RETENTION_LOCK: pupil %s has %d safeguarding record(s) locked under statutory KCSIE retention (DOB + 25/35 years)", trimmedPupilID, lockedSafeguardingCount)
	}

	var auditCount int
	if err := d.readDB.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM safeguarding_audit_logs WHERE pupil_id = ?`,
		trimmedPupilID,
	).Scan(&auditCount); err != nil {
		return fmt.Errorf("failed to inspect safeguarding_audit_logs retention status: %w", err)
	}
	if auditCount > 0 {
		return fmt.Errorf("KCSIE_RETENTION_LOCK: pupil %s has immutable safeguarding audit trail entries locked under KCSIE retention", trimmedPupilID)
	}

	return nil
}

func (d *CoreDAL) PurgeCorePupilData(ctx context.Context, pupilID string) error {
	if err := d.CheckPupilRetentionPurgeEligible(ctx, pupilID); err != nil {
		return err
	}
	trimmedPupilID := strings.TrimSpace(pupilID)

	tx, err := d.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin PurgeCorePupilData transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var personID string
	if err := tx.QueryRowContext(
		ctx,
		`SELECT person_id FROM pupils WHERE id = ?`,
		trimmedPupilID,
	).Scan(&personID); err != nil {
		return fmt.Errorf("failed to resolve person_id for pupil %s: %w", trimmedPupilID, err)
	}

	deleteQueries := []string{
		`DELETE FROM safeguarding_notes WHERE pupil_id = ? AND retention_purge_eligible = 1`,
		`DELETE FROM pupil_pastoral_notes WHERE pupil_id = ?`,
		`DELETE FROM pupil_medical_conditions WHERE pupil_id = ?`,
		`DELETE FROM pupil_post16_learning_aims WHERE pupil_id = ?`,
		`DELETE FROM pupil_alternative_provisions WHERE pupil_id = ?`,
		`DELETE FROM pupil_suspensions_exclusions WHERE pupil_id = ?`,
		`DELETE FROM pupil_sen_assistants WHERE pupil_id = ?`,
		`DELETE FROM pupil_sen_needs WHERE pupil_id = ?`,
		`DELETE FROM pupil_sen_records WHERE pupil_id = ?`,
		`DELETE FROM pupil_fsm_periods WHERE pupil_id = ?`,
		`DELETE FROM attendance_lesson_marks WHERE pupil_id = ?`,
		`DELETE FROM attendance_session_marks WHERE pupil_id = ?`,
		`DELETE FROM group_pupil_memberships WHERE pupil_id = ?`,
		`DELETE FROM guardian_relationships WHERE pupil_id = ?`,
		`DELETE FROM pupil_year_enrolments WHERE pupil_id = ?`,
		`DELETE FROM pupils WHERE id = ?`,
	}

	for _, q := range deleteQueries {
		if _, err := tx.ExecContext(ctx, q, trimmedPupilID); err != nil {
			return fmt.Errorf("failed executing core GDPR purge query (%s): %w", q, err)
		}
	}

	if personID != "" {
		if _, err := tx.ExecContext(ctx, `DELETE FROM household_memberships WHERE person_id = ?`, personID); err != nil {
			return fmt.Errorf("failed deleting household_memberships for person %s: %w", personID, err)
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM person_roles WHERE person_id = ?`, personID); err != nil {
			return fmt.Errorf("failed deleting person_roles for person %s: %w", personID, err)
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM persons WHERE id = ?`, personID); err != nil {
			return fmt.Errorf("failed deleting person %s: %w", personID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	d.emitPostCommitEvent(ctx, EventCoreGDPRPupilPurged, map[string]any{
		"pupil_id": trimmedPupilID,
	})
	return nil
}
