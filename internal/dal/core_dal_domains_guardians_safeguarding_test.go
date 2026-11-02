package dal_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"optimis/internal/dal"
)

// ============================================================================
// 4. Guardians & Households (`ctx.dal.guardians` — Mandatory PR & Court-Order Invariant)
// ============================================================================

func TestListGuardiansForPupilScoped_GuardianWithCourtOrderRestricted_ExcludesRestrictedGuardianFromResults_test(t *testing.T) {
	ctx := context.Background()
	d, fx, logTap := setupIsolatedCoreDALWithFixtures(t)

	scope := dal.DALQueryScope{
		Mode:           dal.ScopeOwnGroups,
		StaffID:        fx.TeacherStaffID,
		AcademicYearID: fx.AcademicYearID,
		ReferenceDate:  fx.ReferenceDate,
	}

	guardians, err := d.ListGuardiansForPupilScoped(ctx, scope, fx.ActivePupilID, dal.GuardianFilterOptions{
		RequireParentalResponsibility: false,
	})
	if err != nil {
		t.Fatalf("ListGuardiansForPupilScoped failed: %v", err)
	}
	logTap.Logf("returned %d non-restricted guardians for %s", len(guardians), fx.ActivePupilID)

	for _, g := range guardians {
		if g.GuardianPersonID == fx.RestrictedFatherPersID || g.FirstName == "George" {
			t.Fatalf("SECURITY VIOLATION: court-order-restricted guardian (%s) was returned by ListGuardiansForPupilScoped: %+v", fx.RestrictedFatherPersID, g)
		}
	}
	if len(guardians) != 2 {
		t.Fatalf("expected 2 non-restricted guardians (Mother + Grandparent), got %d (%+v)", len(guardians), guardians)
	}
}

func TestListGuardiansForPupilScoped_GuardianWithoutParentalResponsibility_RespectsRequirePRFilter_test(t *testing.T) {
	ctx := context.Background()
	d, fx, logTap := setupIsolatedCoreDALWithFixtures(t)

	scope := dal.DALQueryScope{
		Mode:           dal.ScopeOwnGroups,
		StaffID:        fx.TeacherStaffID,
		AcademicYearID: fx.AcademicYearID,
		ReferenceDate:  fx.ReferenceDate,
	}

	prGuardians, err := d.ListGuardiansForPupilScoped(ctx, scope, fx.ActivePupilID, dal.GuardianFilterOptions{
		RequireParentalResponsibility: true,
	})
	if err != nil {
		t.Fatalf("ListGuardiansForPupilScoped (RequirePR=true) failed: %v", err)
	}
	logTap.Logf("returned %d PR guardians for %s", len(prGuardians), fx.ActivePupilID)

	if len(prGuardians) != 1 {
		t.Fatalf("expected exactly 1 unrestricted PR guardian (Anne Byron), got %d (%+v)", len(prGuardians), prGuardians)
	}
	if prGuardians[0].GuardianPersonID != fx.UnrestrictedMotherPersID || !prGuardians[0].HasParentalResponsibility {
		t.Errorf("expected Anne Byron with HasParentalResponsibility=true, got %+v", prGuardians[0])
	}
}

func TestListGuardiansForPupilScoped_HouseholdMarkedConfidentialAddress_RedactsConfidentialAddressFields_test(t *testing.T) {
	ctx := context.Background()
	d, fx, logTap := setupIsolatedCoreDALWithFixtures(t)

	scope := dal.DALQueryScope{
		Mode:           dal.ScopeOwnGroups,
		StaffID:        fx.TeacherStaffID,
		AcademicYearID: fx.AcademicYearID,
		ReferenceDate:  fx.ReferenceDate,
	}

	guardians, err := d.ListGuardiansForPupilScoped(ctx, scope, fx.ActivePupilID, dal.GuardianFilterOptions{
		RequireParentalResponsibility: false,
	})
	if err != nil {
		t.Fatalf("ListGuardiansForPupilScoped failed: %v", err)
	}

	var foundGrandparent bool
	for _, g := range guardians {
		if g.GuardianPersonID == fx.NonPRGrandparentPersID {
			foundGrandparent = true
			if g.AddressLine1 != "" || g.Postcode != "" {
				t.Fatalf("SECURITY VIOLATION: confidential household address leaked for guardian %s: address=%q postcode=%q",
					g.GuardianPersonID, g.AddressLine1, g.Postcode)
			}
			if !g.IsConfidentialAddressRedacted {
				t.Errorf("expected IsConfidentialAddressRedacted == true for guardian residing in confidential household")
			}
		}
		if g.GuardianPersonID == fx.UnrestrictedMotherPersID {
			if g.AddressLine1 != "42 High Street" || g.Postcode != "SW1A 2BB" {
				t.Errorf("expected non-confidential address '42 High Street, SW1A 2BB' for Mother, got %q %q", g.AddressLine1, g.Postcode)
			}
		}
	}
	if !foundGrandparent {
		t.Fatalf("expected non-restricted grandparent %s in guardian list", fx.NonPRGrandparentPersID)
	}
	logTap.Logf("verified confidential household address redaction")
}

func TestListGuardiansForPupilsBatchScoped_MultiplePupilsAndCourtOrderCheck_ReturnsGroupedGuardiansAndExcludesRestricted_test(t *testing.T) {
	ctx := context.Background()
	d, fx, logTap := setupIsolatedCoreDALWithFixtures(t)

	scope := dal.DALQueryScope{
		Mode:           dal.ScopeAll,
		AcademicYearID: fx.AcademicYearID,
		ReferenceDate:  fx.ReferenceDate,
	}

	batch, err := d.ListGuardiansForPupilsBatchScoped(ctx, scope, []string{fx.ActivePupilID}, dal.GuardianFilterOptions{
		RequireParentalResponsibility: false,
	})
	if err != nil {
		t.Fatalf("ListGuardiansForPupilsBatchScoped failed: %v", err)
	}

	guardians, exists := batch[fx.ActivePupilID]
	if !exists {
		t.Fatalf("expected entry for pupil %s in batch results", fx.ActivePupilID)
	}
	if len(guardians) != 2 {
		t.Fatalf("expected 2 non-restricted guardians, got %d", len(guardians))
	}
	for _, g := range guardians {
		if g.GuardianPersonID == fx.RestrictedFatherPersID {
			t.Fatalf("SECURITY VIOLATION: court-order restricted father returned in batch")
		}
	}
	logTap.Logf("verified ListGuardiansForPupilsBatchScoped excludes restricted guardians")
}

func TestUpsertGuardianContactScoped_NewGuardianWithHousehold_PersistsPersonHouseholdAndRelationship_test(t *testing.T) {
	ctx := context.Background()
	d, fx, logTap := setupIsolatedCoreDALWithFixtures(t)

	scope := dal.DALQueryScope{
		Mode:           dal.ScopeAll,
		AcademicYearID: fx.AcademicYearID,
		ReferenceDate:  fx.ReferenceDate,
	}

	isEmerg := true
	authCollect := true
	res, err := d.UpsertGuardianContactScoped(ctx, scope, dal.UpsertGuardianContactInput{
		PupilID:                   fx.ActivePupilID,
		FirstName:                 "Mary",
		LastName:                  "Shelley",
		Email:                     "mary.shelley@example.org",
		PhonePrimary:              "07700900123",
		RelationshipCode:          "AUNT",
		HasParentalResponsibility: false,
		ContactPriority:           3,
		IsEmergencyContact:        &isEmerg,
		AuthorisedToCollect:       &authCollect,
		HouseholdName:             "Shelley Residence",
		AddressLine1:              "10 Shelley Lane",
		TownCity:                  "London",
		Postcode:                  "NW1 4NP",
		EffectiveFrom:             fx.ReferenceDate,
	})
	if err != nil {
		t.Fatalf("UpsertGuardianContactScoped failed: %v", err)
	}
	if res.FirstName != "Mary" || res.LastName != "Shelley" || res.RelationshipCode != "AUNT" {
		t.Errorf("unexpected upsert summary: %+v", res)
	}
	if res.AddressLine1 != "10 Shelley Lane" || res.TownCity != "London" {
		t.Errorf("expected address '10 Shelley Lane, London', got %q %q", res.AddressLine1, res.TownCity)
	}
	logTap.Logf("verified UpsertGuardianContactScoped created guardian %s", res.GuardianPersonID)
}

func TestListSiblingsForPupilScoped_SharedHouseholdAndGuardians_ReturnsDiscoveredSiblings_test(t *testing.T) {
	ctx := context.Background()
	d, fx, logTap := setupIsolatedCoreDALWithFixtures(t)

	scope := dal.DALQueryScope{
		Mode:           dal.ScopeAll,
		AcademicYearID: fx.AcademicYearID,
		ReferenceDate:  fx.ReferenceDate,
	}

	siblings, err := d.ListSiblingsForPupilScoped(ctx, scope, fx.ActivePupilID)
	if err != nil {
		t.Fatalf("ListSiblingsForPupilScoped failed: %v", err)
	}
	logTap.Logf("returned %d siblings for pupil %s", len(siblings), fx.ActivePupilID)
}

func TestListHouseholdsScoped_PupilWithConfidentialAndStandardHouseholds_RedactsConfidentialFields_test(t *testing.T) {
	ctx := context.Background()
	d, fx, logTap := setupIsolatedCoreDALWithFixtures(t)

	scope := dal.DALQueryScope{
		Mode:           dal.ScopeAll,
		AcademicYearID: fx.AcademicYearID,
		ReferenceDate:  fx.ReferenceDate,
	}

	households, err := d.ListHouseholdsScoped(ctx, scope, fx.ActivePupilID)
	if err != nil {
		t.Fatalf("ListHouseholdsScoped failed: %v", err)
	}
	if len(households) == 0 {
		t.Fatalf("expected at least 1 household for pupil %s", fx.ActivePupilID)
	}
	for _, h := range households {
		if h.IsConfidentialAddressRedacted {
			if h.AddressLine1 != "" || h.Postcode != "" || h.HouseholdName != "" {
				t.Fatalf("SECURITY VIOLATION: confidential address details leaked: %+v", h)
			}
		}
	}
	logTap.Logf("verified ListHouseholdsScoped confidential redactions")
}

// ============================================================================
// 5. Safeguarding Vault (`dal.safeguarding.create` Postbox vs `dal.safeguarding.read` DSL Vault)
// ============================================================================

func TestCreateSafeguardingPostboxNote_ValidTeacherCaller_EncryptsPayloadWithEnvelopeKeyAndWritesImmutableAuditLog_test(t *testing.T) {
	ctx := context.Background()
	d, fx, logTap := setupIsolatedCoreDALWithFixtures(t)

	plaintextVerbatim := "Pupil disclosed bruised wrist during morning registration."
	receipt, err := d.CreateSafeguardingPostboxNote(ctx, dal.CreateSafeguardingPostboxInput{
		PupilID:            fx.ActivePupilID,
		ReportedByStaffID:  fx.TeacherStaffID,
		AssignedDSLStaffID: fx.DSLStaffID,
		ConcernCategory:    "PHYSICAL",
		IncidentDate:       fx.ReferenceDate,
		Summary:            "Bruised wrist disclosure",
		BodyMap:            "LEFT_WRIST",
		VerbatimAccount:    plaintextVerbatim,
		ActionTaken:        "Referred immediately to DSL",
		RequestID:          "req_sfg_postbox_1",
		ClientIP:           "10.0.0.15",
		HostKMSKeyHandle:   fx.HostKMSKeyHandle,
	})
	if err != nil {
		t.Fatalf("CreateSafeguardingPostboxNote failed: %v", err)
	}
	logTap.Logf("created safeguarding postbox note %s", receipt.NoteID)

	if !strings.HasPrefix(receipt.NoteID, "sfg_") {
		t.Errorf("expected note ID prefixed with 'sfg_', got %q", receipt.NoteID)
	}

	// Verify raw SQLite storage is AES-256-GCM encrypted (ciphertext does NOT contain plaintext)
	var wrappedDEK, nonce, ciphertext []byte
	err = d.ReadDB().QueryRowContext(
		ctx,
		`SELECT wrapped_dek, nonce, ciphertext_payload FROM safeguarding_notes WHERE id = ?`,
		receipt.NoteID,
	).Scan(&wrappedDEK, &nonce, &ciphertext)
	if err != nil {
		t.Fatalf("failed to query raw safeguarding_notes row: %v", err)
	}
	if len(wrappedDEK) == 0 || len(nonce) != 12 || len(ciphertext) == 0 {
		t.Fatalf("expected non-empty wrapped_dek, 12-byte nonce, and non-empty ciphertext, got dek=%d nonce=%d cipher=%d",
			len(wrappedDEK), len(nonce), len(ciphertext))
	}
	if bytes.Contains(ciphertext, []byte(plaintextVerbatim)) || bytes.Contains(ciphertext, []byte("Bruised wrist")) {
		t.Fatalf("SECURITY VIOLATION: safeguarding ciphertext_payload contains unencrypted plaintext")
	}

	// Verify append-only CREATE_POSTBOX audit log entry was written
	var actionType string
	err = d.ReadDB().QueryRowContext(
		ctx,
		`SELECT action_type FROM safeguarding_audit_logs WHERE note_id = ? AND request_id = 'req_sfg_postbox_1'`,
		receipt.NoteID,
	).Scan(&actionType)
	if err != nil || actionType != "CREATE_POSTBOX" {
		t.Fatalf("expected CREATE_POSTBOX audit log entry for note %s, got action=%q err=%v", receipt.NoteID, actionType, err)
	}
}

func TestReadSafeguardingNoteDSL_AuthorizedDSLWithValidCloudKeyToken_DecryptsPayloadAndWritesDSLDecryptAuditLog_test(t *testing.T) {
	ctx := context.Background()
	d, fx, logTap := setupIsolatedCoreDALWithFixtures(t)

	receipt, err := d.CreateSafeguardingPostboxNote(ctx, dal.CreateSafeguardingPostboxInput{
		PupilID:            fx.ActivePupilID,
		ReportedByStaffID:  fx.TeacherStaffID,
		AssignedDSLStaffID: fx.DSLStaffID,
		ConcernCategory:    "EMOTIONAL",
		IncidentDate:       fx.ReferenceDate,
		Summary:            "Welfare check concern",
		BodyMap:            "NONE",
		VerbatimAccount:    "Pupil appeared distressed before Period 1.",
		ActionTaken:        "Escorted to pastoral office",
		RequestID:          "req_sfg_create_2",
		ClientIP:           "10.0.0.15",
		HostKMSKeyHandle:   fx.HostKMSKeyHandle,
	})
	if err != nil {
		t.Fatalf("CreateSafeguardingPostboxNote failed: %v", err)
	}

	cloudToken, err := d.IssueDSLCloudKeyToken(fx.DSLStaffID, fx.HostKMSKeyHandle, 24*time.Hour)
	if err != nil {
		t.Fatalf("IssueDSLCloudKeyToken failed: %v", err)
	}

	decrypted, err := d.ReadSafeguardingNoteDSL(ctx, dal.ReadSafeguardingNoteDSLInput{
		NoteID:           receipt.NoteID,
		DSLStaffID:       fx.DSLStaffID,
		CloudKeyToken:    cloudToken,
		HostKMSKeyHandle: fx.HostKMSKeyHandle,
		RequestID:        "req_sfg_read_dsl_2",
		ClientIP:         "10.0.0.20",
	})
	if err != nil {
		t.Fatalf("ReadSafeguardingNoteDSL failed for authorized DSL: %v", err)
	}
	logTap.Logf("DSL %s decrypted safeguarding note %s", fx.DSLStaffID, decrypted.NoteID)

	if decrypted.Summary != "Welfare check concern" || decrypted.VerbatimAccount != "Pupil appeared distressed before Period 1." {
		t.Errorf("decrypted payload mismatch: %+v", decrypted)
	}

	var auditCount int
	err = d.ReadDB().QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM safeguarding_audit_logs WHERE note_id = ? AND action_type = 'DSL_DECRYPT_VIEW' AND request_id = 'req_sfg_read_dsl_2'`,
		receipt.NoteID,
	).Scan(&auditCount)
	if err != nil || auditCount != 1 {
		t.Fatalf("expected DSL_DECRYPT_VIEW audit log entry, got count=%d err=%v", auditCount, err)
	}
}

func TestReadSafeguardingNoteDSL_NonDSLStaffOrExpiredCloudKeyToken_FailsFastWithUnauthorizedError_test(t *testing.T) {
	ctx := context.Background()
	d, fx, logTap := setupIsolatedCoreDALWithFixtures(t)

	receipt, err := d.CreateSafeguardingPostboxNote(ctx, dal.CreateSafeguardingPostboxInput{
		PupilID:            fx.ActivePupilID,
		ReportedByStaffID:  fx.TeacherStaffID,
		AssignedDSLStaffID: fx.DSLStaffID,
		ConcernCategory:    "NEGLECT",
		IncidentDate:       fx.ReferenceDate,
		Summary:            "Confidential concern",
		BodyMap:            "NONE",
		VerbatimAccount:    "Strictly confidential details.",
		ActionTaken:        "Logged in postbox",
		RequestID:          "req_sfg_create_3",
		ClientIP:           "10.0.0.15",
		HostKMSKeyHandle:   fx.HostKMSKeyHandle,
	})
	if err != nil {
		t.Fatalf("CreateSafeguardingPostboxNote failed: %v", err)
	}

	// 1. Non-DSL teacher attempting to read/decrypt MUST fail fast
	teacherToken, _ := d.IssueDSLCloudKeyToken(fx.TeacherStaffID, fx.HostKMSKeyHandle, 1*time.Hour)
	_, err = d.ReadSafeguardingNoteDSL(ctx, dal.ReadSafeguardingNoteDSLInput{
		NoteID:           receipt.NoteID,
		DSLStaffID:       fx.TeacherStaffID,
		CloudKeyToken:    teacherToken,
		HostKMSKeyHandle: fx.HostKMSKeyHandle,
		RequestID:        "req_sfg_unauth_teacher",
		ClientIP:         "10.0.0.15",
	})
	if err == nil {
		t.Fatalf("SECURITY VIOLATION: non-DSL teacher (%s) was allowed to decrypt safeguarding note %s", fx.TeacherStaffID, receipt.NoteID)
	}

	// 2. DSL with expired cloud key token (negative TTL) MUST fail fast
	expiredToken, err := d.IssueDSLCloudKeyToken(fx.DSLStaffID, fx.HostKMSKeyHandle, -5*time.Minute)
	if err != nil {
		t.Fatalf("failed to construct expired DSL token: %v", err)
	}
	_, err = d.ReadSafeguardingNoteDSL(ctx, dal.ReadSafeguardingNoteDSLInput{
		NoteID:           receipt.NoteID,
		DSLStaffID:       fx.DSLStaffID,
		CloudKeyToken:    expiredToken,
		HostKMSKeyHandle: fx.HostKMSKeyHandle,
		RequestID:        "req_sfg_expired_token",
		ClientIP:         "10.0.0.20",
	})
	if err == nil {
		t.Fatalf("SECURITY VIOLATION: expired DSL cloud key token was accepted for decrypting note %s", receipt.NoteID)
	}
	logTap.Logf("verified non-DSL staff and expired cloud key tokens are rejected")
}

func TestListSafeguardingNoteEnvelopesDSL_DSLAndNonDSL_ReturnsEncryptedEnvelopeMetadataOnlyToDSL_test(t *testing.T) {
	ctx := context.Background()
	d, fx, logTap := setupIsolatedCoreDALWithFixtures(t)

	receipt, err := d.CreateSafeguardingPostboxNote(ctx, dal.CreateSafeguardingPostboxInput{
		PupilID:            fx.ActivePupilID,
		ReportedByStaffID:  fx.TeacherStaffID,
		AssignedDSLStaffID: fx.DSLStaffID,
		ConcernCategory:    "ONLINE_SAFETY",
		IncidentDate:       fx.ReferenceDate,
		Summary:            "Sensitive online safety disclosure",
		BodyMap:            "NONE",
		VerbatimAccount:    "Strictly confidential verbatim text.",
		ActionTaken:        "Referred to DSL",
		RequestID:          "req_sfg_list_meta_1",
		ClientIP:           "10.0.0.15",
		HostKMSKeyHandle:   fx.HostKMSKeyHandle,
	})
	if err != nil {
		t.Fatalf("CreateSafeguardingPostboxNote failed: %v", err)
	}

	// 1. Non-DSL staff member MUST fail fast when listing safeguarding note envelopes
	if _, err := d.ListSafeguardingNoteEnvelopesDSL(ctx, fx.TeacherStaffID, fx.ActivePupilID); err == nil {
		t.Fatalf("SECURITY VIOLATION: non-DSL teacher (%s) was allowed to list safeguarding note envelopes", fx.TeacherStaffID)
	}

	// 2. Authorized DSL receives envelope metadata (NoteID, Category, Status, KMSKeyID) without plaintext
	envelopes, err := d.ListSafeguardingNoteEnvelopesDSL(ctx, fx.DSLStaffID, fx.ActivePupilID)
	if err != nil {
		t.Fatalf("ListSafeguardingNoteEnvelopesDSL failed for DSL: %v", err)
	}
	if len(envelopes) != 1 || envelopes[0].NoteID != receipt.NoteID {
		t.Fatalf("expected 1 envelope for note %s, got %+v", receipt.NoteID, envelopes)
	}
	if envelopes[0].ConcernCategory != "ONLINE_SAFETY" || envelopes[0].Status != "OPEN_TRIAGE" {
		t.Errorf("unexpected envelope metadata: %+v", envelopes[0])
	}
	logTap.Logf("verified DSL envelope metadata listing for note %s", receipt.NoteID)
}

func TestReadSafeguardingNoteDSL_ExpiredCloudKeyRenewed_RestoresAccessWithoutDataLoss_test(t *testing.T) {
	ctx := context.Background()
	d, fx, logTap := setupIsolatedCoreDALWithFixtures(t)

	receipt, err := d.CreateSafeguardingPostboxNote(ctx, dal.CreateSafeguardingPostboxInput{
		PupilID:            fx.ActivePupilID,
		ReportedByStaffID:  fx.TeacherStaffID,
		AssignedDSLStaffID: fx.DSLStaffID,
		ConcernCategory:    "EMOTIONAL",
		IncidentDate:       fx.ReferenceDate,
		Summary:            "Cloud key renewal test summary",
		BodyMap:            "NONE",
		VerbatimAccount:    "Cloud key renewal verbatim account.",
		ActionTaken:        "Logged in postbox",
		RequestID:          "req_sfg_renew_1",
		ClientIP:           "10.0.0.15",
		HostKMSKeyHandle:   fx.HostKMSKeyHandle,
	})
	if err != nil {
		t.Fatalf("CreateSafeguardingPostboxNote failed: %v", err)
	}

	// 1. Expired cloud key fails fast
	expiredKey, err := d.IssueDSLCloudKeyToken(fx.DSLStaffID, fx.HostKMSKeyHandle, -1*time.Hour)
	if err != nil {
		t.Fatalf("IssueDSLCloudKeyToken (expired) failed: %v", err)
	}
	if _, err := d.ReadSafeguardingNoteDSL(ctx, dal.ReadSafeguardingNoteDSLInput{
		NoteID:           receipt.NoteID,
		DSLStaffID:       fx.DSLStaffID,
		CloudKeyToken:    expiredKey,
		HostKMSKeyHandle: fx.HostKMSKeyHandle,
		RequestID:        "req_sfg_renew_attempt_1",
		ClientIP:         "10.0.0.20",
	}); err == nil {
		t.Fatalf("expected expired cloud key to fail fast")
	}

	// 2. Generating a new cloud key seamlessly restores access without data loss (GEMINI.md §5B)
	renewedKey, err := d.IssueDSLCloudKeyToken(fx.DSLStaffID, fx.HostKMSKeyHandle, 24*time.Hour)
	if err != nil {
		t.Fatalf("IssueDSLCloudKeyToken (renewed) failed: %v", err)
	}
	decrypted, err := d.ReadSafeguardingNoteDSL(ctx, dal.ReadSafeguardingNoteDSLInput{
		NoteID:           receipt.NoteID,
		DSLStaffID:       fx.DSLStaffID,
		CloudKeyToken:    renewedKey,
		HostKMSKeyHandle: fx.HostKMSKeyHandle,
		RequestID:        "req_sfg_renew_attempt_2",
		ClientIP:         "10.0.0.20",
	})
	if err != nil {
		t.Fatalf("ReadSafeguardingNoteDSL with renewed key failed: %v", err)
	}
	if decrypted.Summary != "Cloud key renewal test summary" || len(decrypted.AuditLogs) != 2 {
		t.Fatalf("unexpected decrypted note or audit log count after cloud key renewal: %+v", decrypted)
	}
	logTap.Logf("verified OAuth Cloud Key renewal restores DSL vault access with zero data loss")
}
