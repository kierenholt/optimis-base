package host_test

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"optimis/internal/dal"
	"optimis/internal/host"
)

func TestFirstAidMedicalPlugin_ProfileTreatmentEventAndGdprSarHooks_test(t *testing.T) {
	tempDir := t.TempDir()
	pluginsDir, err := filepath.Abs(filepath.Join("..", "..", "plugins"))
	if err != nil {
		t.Fatalf("failed to resolve plugins dir: %v", err)
	}

	h, err := host.NewHost(host.Config{
		DataDirPath:      filepath.Join(tempDir, "data"),
		SchoolURN:        "100001",
		HostKMSKeyHandle: "kms-first-aid-test-key-001",
		PluginsDir:       pluginsDir,
	})
	if err != nil {
		t.Fatalf("failed to create host: %v", err)
	}
	defer func() {
		_ = h.Close()
	}()

	ctx := context.Background()
	personas, err := h.DAL().SeedPersonaTestFixtures(ctx, "ay_2026_2027", "2026-09-01")
	if err != nil {
		t.Fatalf("failed to seed personas: %v", err)
	}
	m1, err := h.DAL().SeedMilestone1Fixtures(ctx, dal.Milestone1SeedInput{
		AcademicYearID: "ay_2026_2027",
		ReferenceDate:  "2026-10-28",
	})
	if err != nil {
		t.Fatalf("failed to seed Milestone 1 fixtures: %v", err)
	}

	if err := h.LoadPlugin("first_aid_medical_plugin"); err != nil {
		t.Fatalf("failed to load first_aid_medical_plugin: %v", err)
	}

	firstAiderCaller := host.CallerContext{
		Type:        "USER",
		UserID:      "usr_" + personas.SchoolNurseStaffID,
		StaffID:     personas.SchoolNurseStaffID,
		ActiveRoles: []string{"SUPPORT"},
		RequestID:   "req_fa_001",
		Scopes: []string{
			"dal.pupils.read:all",
			"dal.medical.read:all",
			"dal.guardians.read:all",
		},
	}

	// 1. Get Pupil First Aid Profile (verifying CRITICAL anaphylaxis/IHCP alert + safe emergency contacts)
	profileRes, err := h.ExecuteRPC(ctx, "uk.org.optimis.medical.firstaid", "get_pupil_first_aid_profile", firstAiderCaller, map[string]any{
		"pupil_id":         m1.ActivePupilID,
		"academic_year_id": m1.AcademicYearID,
		"reference_date":   m1.ReferenceDate,
	})
	if err != nil {
		t.Fatalf("get_pupil_first_aid_profile failed: %v", err)
	}
	if !profileRes.OK {
		t.Fatalf("expected OK=true from get_pupil_first_aid_profile, got error: %s", profileRes.ErrorCode)
	}

	rawProfileJSON, err := json.Marshal(profileRes.Data)
	if err != nil {
		t.Fatalf("failed to marshal profile response: %v", err)
	}
	profileStr := string(rawProfileJSON)

	if !strings.Contains(profileStr, "CRITICAL") || !strings.Contains(profileStr, "Anaphylaxis (Peanut Allergy)") {
		t.Fatalf("expected CRITICAL Anaphylaxis alert in profile, got: %s", profileStr)
	}
	if !strings.Contains(profileStr, "Administer EpiPen immediately and call 999") {
		t.Fatalf("expected emergency response plan in profile, got: %s", profileStr)
	}
	if !strings.Contains(profileStr, "EpiPen 0.3mg IM Auto-Injector") || !strings.Contains(profileStr, "Medical Room Red Cabinet #2") {
		t.Fatalf("expected clinical medication_name_and_dose and medication_location in profile, got: %s", profileStr)
	}
	if !strings.Contains(profileStr, "Anne Byron") || !strings.Contains(profileStr, "AUTHORISED_TO_COLLECT") {
		t.Fatalf("expected unrestricted mother Anne Byron in emergency contacts, got: %s", profileStr)
	}
	if !strings.Contains(profileStr, "Judith Milbanke") || !strings.Contains(profileStr, "CONFIDENTIAL_ADDRESS_REDACTED") {
		t.Fatalf("expected refuge grandparent with CONFIDENTIAL_ADDRESS_REDACTED in emergency contacts, got: %s", profileStr)
	}
	if strings.Contains(profileStr, "George Byron") || strings.Contains(profileStr, m1.RestrictedFatherPersonID) || strings.Contains(profileStr, "Prohibited Steps Order") {
		t.Fatalf("security violation: court-order-restricted father leaked in emergency contacts: %s", profileStr)
	}
	if strings.Contains(profileStr, "10 Safe Haven Way") || strings.Contains(profileStr, "Confidential Refuge") || strings.Contains(profileStr, "SW1A 1AA") {
		t.Fatalf("security violation: confidential refuge address leaked in emergency contacts: %s", profileStr)
	}

	// 2. Log a First Aid treatment and verify post-commit custom event emission + worker execution
	logRes, err := h.ExecuteRPC(ctx, "uk.org.optimis.medical.firstaid", "log_first_aid_treatment", firstAiderCaller, map[string]any{
		"pupil_id":          m1.ActivePupilID,
		"academic_year_id":  m1.AcademicYearID,
		"reference_date":    m1.ReferenceDate,
		"incident_location": "Sports Hall",
		"injury_summary":    "Sprained left ankle during PE badminton",
		"treatment_given":   "RICE protocol applied, ice pack for 15 mins",
		"outcome":           "SENT_HOME_WITH_GUARDIAN",
	})
	if err != nil {
		t.Fatalf("log_first_aid_treatment failed: %v", err)
	}
	if !logRes.OK {
		t.Fatalf("expected OK=true from log_first_aid_treatment, got error: %s", logRes.ErrorCode)
	}

	deadline := time.Now().Add(3 * time.Second)
	var afterLogStr string
	for time.Now().Before(deadline) {
		afterLogProfile, err := h.ExecuteRPC(ctx, "uk.org.optimis.medical.firstaid", "get_pupil_first_aid_profile", firstAiderCaller, map[string]any{
			"pupil_id":         m1.ActivePupilID,
			"academic_year_id": m1.AcademicYearID,
			"reference_date":   m1.ReferenceDate,
		})
		if err != nil {
			t.Fatalf("get_pupil_first_aid_profile after log failed: %v", err)
		}
		afterLogBytes, _ := json.Marshal(afterLogProfile.Data)
		afterLogStr = string(afterLogBytes)
		if strings.Contains(afterLogStr, "plugin.uk.org.optimis.medical.firstaid.logged") && strings.Contains(afterLogStr, "DISPATCHED_POST_COMMIT") {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if !strings.Contains(afterLogStr, "plugin.uk.org.optimis.medical.firstaid.logged") || !strings.Contains(afterLogStr, "DISPATCHED_POST_COMMIT") {
		t.Fatalf("expected post-commit event log in profile after treatment, got: %s", afterLogStr)
	}

	// 3. Verify GDPR SAR export_pupil_data and purge_pupil_data hooks
	adminCaller := host.CallerContext{
		Type:        "USER",
		UserID:      "usr_" + personas.AdminStaffID,
		StaffID:     personas.AdminStaffID,
		ActiveRoles: []string{"ADMIN"},
		RequestID:   "req_fa_002",
		Scopes: []string{
			"dal.pupils.read:all",
			"dal.pupils.write:all",
		},
	}

	sarExport, err := h.ExecuteRPC(ctx, "uk.org.optimis.medical.firstaid", "export_pupil_data", adminCaller, map[string]any{
		"pupil_id": m1.ActivePupilID,
	})
	if err != nil {
		t.Fatalf("export_pupil_data failed: %v", err)
	}
	exportBytes, _ := json.Marshal(sarExport.Data)
	if !strings.Contains(string(exportBytes), "Sprained left ankle during PE badminton") {
		t.Fatalf("expected treatment in SAR export, got: %s", string(exportBytes))
	}

	sarPurge, err := h.ExecuteRPC(ctx, "uk.org.optimis.medical.firstaid", "purge_pupil_data", adminCaller, map[string]any{
		"pupil_id": m1.ActivePupilID,
	})
	if err != nil {
		t.Fatalf("purge_pupil_data failed: %v", err)
	}
	if !sarPurge.OK {
		t.Fatalf("expected OK=true from purge_pupil_data")
	}

	sarAfterPurge, err := h.ExecuteRPC(ctx, "uk.org.optimis.medical.firstaid", "export_pupil_data", adminCaller, map[string]any{
		"pupil_id": m1.ActivePupilID,
	})
	if err != nil {
		t.Fatalf("export_pupil_data after purge failed: %v", err)
	}
	afterPurgeBytes, _ := json.Marshal(sarAfterPurge.Data)
	if strings.Contains(string(afterPurgeBytes), "Sprained left ankle during PE badminton") {
		t.Fatalf("expected 0 treatment records after SAR purge, got: %s", string(afterPurgeBytes))
	}
}
