package host_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"optimis/internal/dal"
)

func TestConfigureCloudKMSMode_RealModeEnabledWithoutConfig_FailsFastAndBlocksSafeguardingVault_test(t *testing.T) {
	ctx := context.Background()
	h, cleanup := setupIsolatedTestHost(t)
	t.Cleanup(cleanup)

	if err := h.ConfigureAuthMode(ctx, true); err != nil {
		t.Fatalf("ConfigureAuthMode(true) failed: %v", err)
	}

	seeded, err := h.DAL().SeedPersonaTestFixtures(ctx, "ay_2026_2027", "2026-09-01")
	if err != nil {
		t.Fatalf("SeedPersonaTestFixtures failed: %v", err)
	}
	t.Cleanup(func() {
		_ = h.DAL().DeletePersonaTestFixtures(context.Background(), seeded.FixtureID)
	})

	// 1. Enabling real Cloud KMS mode without provider configuration in __system_config must fail fast
	err = h.ConfigureCloudKMSMode(ctx, true)
	if err == nil {
		t.Fatalf("expected fail-fast error when ENABLE_REAL_CLOUD_KMS=true and kms_cloud_key_generator_url is not configured, got nil")
	}
	if !strings.Contains(err.Error(), "REAL_CLOUD_KMS_NOT_CONFIGURED") {
		t.Errorf("expected error to contain REAL_CLOUD_KMS_NOT_CONFIGURED, got: %v", err)
	}

	handler, err := h.NewHTTPHandler()
	if err != nil {
		t.Fatalf("NewHTTPHandler failed: %v", err)
	}

	// 2. Verify Cloud Key renewal endpoint fails fast with 503 REAL_CLOUD_KMS_NOT_CONFIGURED while real KMS is enabled without config
	renewPayload, _ := json.Marshal(map[string]any{
		"staff_id":    seeded.DSLStaffID,
		"ttl_seconds": 3600,
	})
	reqRenew := httptest.NewRequest(http.MethodPost, "/api/safeguarding/cloud-key/renew", bytes.NewReader(renewPayload))
	recRenew := httptest.NewRecorder()
	handler.ServeHTTP(recRenew, reqRenew)
	if recRenew.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 ServiceUnavailable when REAL_CLOUD_KMS_NOT_CONFIGURED, got %d (%s)", recRenew.Code, recRenew.Body.String())
	}
	if !strings.Contains(recRenew.Body.String(), "REAL_CLOUD_KMS_NOT_CONFIGURED") {
		t.Errorf("expected response body to contain REAL_CLOUD_KMS_NOT_CONFIGURED, got: %s", recRenew.Body.String())
	}

	// 3. Restore mock Cloud KMS mode (ENABLE_REAL_CLOUD_KMS=false) and verify Cloud Key renewal succeeds
	if err := h.ConfigureCloudKMSMode(ctx, false); err != nil {
		t.Fatalf("ConfigureCloudKMSMode(false) failed: %v", err)
	}

	reqRenewMock := httptest.NewRequest(http.MethodPost, "/api/safeguarding/cloud-key/renew", bytes.NewReader(renewPayload))
	recRenewMock := httptest.NewRecorder()
	handler.ServeHTTP(recRenewMock, reqRenewMock)
	if recRenewMock.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from POST /api/safeguarding/cloud-key/renew in mock KMS mode, got %d (%s)", recRenewMock.Code, recRenewMock.Body.String())
	}
}

func TestParentDMZSync_MockModePushProjectionAndPullInbox_EnforcesGuardianSafeguardingAndFailsFastInRealMode_test(t *testing.T) {
	ctx := context.Background()
	h, cleanup := setupIsolatedTestHost(t)
	t.Cleanup(cleanup)

	if err := h.ConfigureAuthMode(ctx, true); err != nil {
		t.Fatalf("ConfigureAuthMode(true) failed: %v", err)
	}
	if err := h.ConfigureParentDMZMode(ctx, false); err != nil {
		t.Fatalf("ConfigureParentDMZMode(false) failed: %v", err)
	}

	m1Seeded, err := h.DAL().SeedMilestone1Fixtures(ctx, dal.Milestone1SeedInput{
		AcademicYearID: "ay_2026_2027",
		ReferenceDate:  "2026-10-28",
	})
	if err != nil {
		t.Fatalf("SeedMilestone1Fixtures failed: %v", err)
	}
	t.Cleanup(func() {
		_ = h.DAL().DeleteMilestone1Fixtures(context.Background(), m1Seeded.FixtureID)
	})

	personas, err := h.DAL().SeedPersonaTestFixtures(ctx, "ay_2026_2027", "2026-09-01")
	if err != nil {
		t.Fatalf("SeedPersonaTestFixtures failed: %v", err)
	}
	t.Cleanup(func() {
		_ = h.DAL().DeletePersonaTestFixtures(context.Background(), personas.FixtureID)
	})

	handler, err := h.NewHTTPHandler()
	if err != nil {
		t.Fatalf("NewHTTPHandler failed: %v", err)
	}

	// 1. Push sanitized parent read-projection to mock DMZ
	pushBodyBytes, _ := json.Marshal(map[string]any{
		"pupil_id":         m1Seeded.ActivePupilID,
		"reference_date":   m1Seeded.ReferenceDate,
		"academic_year_id": m1Seeded.AcademicYearID,
		"staff_id":         personas.AdminStaffID,
	})
	reqPush := httptest.NewRequest(http.MethodPost, "/api/sync/parent-dmz/push-projection", bytes.NewReader(pushBodyBytes))
	recPush := httptest.NewRecorder()
	handler.ServeHTTP(recPush, reqPush)
	if recPush.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from POST /api/sync/parent-dmz/push-projection, got %d (%s)", recPush.Code, recPush.Body.String())
	}

	var pushResp struct {
		OK         bool `json:"ok"`
		Projection struct {
			PupilID                          string   `json:"pupil_id"`
			FirstName                        string   `json:"first_name"`
			LastName                         string   `json:"last_name"`
			AcademicYearID                   string   `json:"academic_year_id"`
			AuthorizedGuardianIDs            []string `json:"authorized_guardian_ids"`
			RedactedConfidentialAddressCount int      `json:"redacted_confidential_address_count"`
		} `json:"projection"`
	}
	if err := json.Unmarshal(recPush.Body.Bytes(), &pushResp); err != nil {
		t.Fatalf("failed to decode push-projection response: %v", err)
	}
	if !pushResp.OK || pushResp.Projection.PupilID != m1Seeded.ActivePupilID {
		t.Fatalf("unexpected projection response: %+v", pushResp)
	}

	foundMother := false
	for _, gID := range pushResp.Projection.AuthorizedGuardianIDs {
		if gID == m1Seeded.UnrestrictedMotherPersonID {
			foundMother = true
		}
		if gID == m1Seeded.RestrictedFatherPersonID {
			t.Fatalf("SAFEGUARDING VIOLATION: court-order-restricted father %s included in parent DMZ projection", gID)
		}
	}
	if !foundMother {
		t.Fatalf("expected unrestricted mother %s in authorized_guardian_ids, got %v", m1Seeded.UnrestrictedMotherPersonID, pushResp.Projection.AuthorizedGuardianIDs)
	}
	if pushResp.Projection.RedactedConfidentialAddressCount < 1 {
		t.Errorf("expected redacted_confidential_address_count >= 1, got %d", pushResp.Projection.RedactedConfidentialAddressCount)
	}

	// 2. Enqueue untrusted submissions (one from unrestricted mother, one from court-order-restricted father) and pull into School Container
	enqMom, _ := json.Marshal(map[string]any{
		"pupil_id":                        m1Seeded.ActivePupilID,
		"submitted_by_guardian_person_id": m1Seeded.UnrestrictedMotherPersonID,
		"submission_type":                 "ABSENCE_NOTICE",
		"date":                            m1Seeded.ReferenceDate,
		"session":                         "AM",
		"reason":                          "GP appointment",
	})
	recEnqMom := httptest.NewRecorder()
	handler.ServeHTTP(recEnqMom, httptest.NewRequest(http.MethodPost, "/api/test/parent-dmz/enqueue-inbox", bytes.NewReader(enqMom)))
	if recEnqMom.Code != http.StatusOK && recEnqMom.Code != http.StatusCreated {
		t.Fatalf("enqueue mother submission failed: %d (%s)", recEnqMom.Code, recEnqMom.Body.String())
	}

	enqDad, _ := json.Marshal(map[string]any{
		"pupil_id":                        m1Seeded.ActivePupilID,
		"submitted_by_guardian_person_id": m1Seeded.RestrictedFatherPersonID,
		"submission_type":                 "CONTACT_UPDATE",
		"date":                            m1Seeded.ReferenceDate,
		"reason":                          "Blocked contact attempt",
	})
	recEnqDad := httptest.NewRecorder()
	handler.ServeHTTP(recEnqDad, httptest.NewRequest(http.MethodPost, "/api/test/parent-dmz/enqueue-inbox", bytes.NewReader(enqDad)))
	if recEnqDad.Code != http.StatusOK && recEnqDad.Code != http.StatusCreated {
		t.Fatalf("enqueue father submission failed: %d (%s)", recEnqDad.Code, recEnqDad.Body.String())
	}

	pullPayload, _ := json.Marshal(map[string]any{"staff_id": personas.AdminStaffID})
	recPull := httptest.NewRecorder()
	handler.ServeHTTP(recPull, httptest.NewRequest(http.MethodPost, "/api/sync/parent-dmz/pull-inbox", bytes.NewReader(pullPayload)))
	if recPull.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from POST /api/sync/parent-dmz/pull-inbox, got %d (%s)", recPull.Code, recPull.Body.String())
	}

	var pullResp struct {
		OK            bool `json:"ok"`
		AcceptedCount int  `json:"accepted_count"`
		RejectedCount int  `json:"rejected_count"`
	}
	if err := json.Unmarshal(recPull.Body.Bytes(), &pullResp); err != nil {
		t.Fatalf("failed to decode pull-inbox response: %v", err)
	}
	if pullResp.AcceptedCount != 1 || pullResp.RejectedCount != 1 {
		t.Fatalf("expected accepted_count=1 and rejected_count=1, got %+v", pullResp)
	}

	// 3. Enabling real Parent DMZ mode without parent_dmz_endpoint_url in __system_config must fail fast
	err = h.ConfigureParentDMZMode(ctx, true)
	if err == nil || !strings.Contains(err.Error(), "REAL_PARENT_DMZ_NOT_CONFIGURED") {
		t.Fatalf("expected REAL_PARENT_DMZ_NOT_CONFIGURED error when enabling real DMZ sync without config, got: %v", err)
	}
}

func TestMATCentralSync_MockModePullPoliciesAndPushAnalytics_PersistsPoliciesExcludesPIIAndFailsFastInRealMode_test(t *testing.T) {
	ctx := context.Background()
	h, cleanup := setupIsolatedTestHost(t)
	t.Cleanup(cleanup)

	if err := h.ConfigureAuthMode(ctx, true); err != nil {
		t.Fatalf("ConfigureAuthMode(true) failed: %v", err)
	}
	if err := h.ConfigureMATSyncMode(ctx, false); err != nil {
		t.Fatalf("ConfigureMATSyncMode(false) failed: %v", err)
	}

	m1Seeded, err := h.DAL().SeedMilestone1Fixtures(ctx, dal.Milestone1SeedInput{
		AcademicYearID: "ay_2026_2027",
		ReferenceDate:  "2026-10-28",
	})
	if err != nil {
		t.Fatalf("SeedMilestone1Fixtures failed: %v", err)
	}
	t.Cleanup(func() {
		_ = h.DAL().DeleteMilestone1Fixtures(context.Background(), m1Seeded.FixtureID)
	})

	personas, err := h.DAL().SeedPersonaTestFixtures(ctx, "ay_2026_2027", "2026-09-01")
	if err != nil {
		t.Fatalf("SeedPersonaTestFixtures failed: %v", err)
	}
	t.Cleanup(func() {
		_ = h.DAL().DeletePersonaTestFixtures(context.Background(), personas.FixtureID)
	})

	handler, err := h.NewHTTPHandler()
	if err != nil {
		t.Fatalf("NewHTTPHandler failed: %v", err)
	}

	// 1. Seed mock MAT feed and pull policies outbound
	seedMatBytes, _ := json.Marshal(map[string]any{
		"mat_id":                 "MAT_UNIT_TEST_01",
		"grading_schema_version": "KS3_MASTERY_V2",
		"behaviour_taxonomy":     []string{"PRAISE_GOLD", "SANCTION_C1"},
		"approved_plugins": []map[string]any{
			{"id": "uk.gov.dfe.census", "version": "1.0.0"},
		},
	})
	recSeedMat := httptest.NewRecorder()
	handler.ServeHTTP(recSeedMat, httptest.NewRequest(http.MethodPost, "/api/test/mat/seed-feed", bytes.NewReader(seedMatBytes)))
	if recSeedMat.Code != http.StatusOK && recSeedMat.Code != http.StatusCreated {
		t.Fatalf("seed MAT feed failed: %d (%s)", recSeedMat.Code, recSeedMat.Body.String())
	}

	pullMatBytes, _ := json.Marshal(map[string]any{"staff_id": personas.AdminStaffID})
	recPullMat := httptest.NewRecorder()
	handler.ServeHTTP(recPullMat, httptest.NewRequest(http.MethodPost, "/api/sync/mat/pull-policies", bytes.NewReader(pullMatBytes)))
	if recPullMat.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from POST /api/sync/mat/pull-policies, got %d (%s)", recPullMat.Code, recPullMat.Body.String())
	}

	matIDVal, err := h.DAL().GetSystemConfigValue(ctx, "mat_id")
	if err != nil || matIDVal != "MAT_UNIT_TEST_01" {
		t.Fatalf("expected mat_id MAT_UNIT_TEST_01 persisted in __system_config, got %q (err: %v)", matIDVal, err)
	}

	// 2. Push aggregated school-level MAT analytics and verify zero pupil PII
	pushAnalyticsBytes, _ := json.Marshal(map[string]any{
		"academic_year_id": m1Seeded.AcademicYearID,
		"staff_id":         personas.AdminStaffID,
	})
	recPushAnalytics := httptest.NewRecorder()
	handler.ServeHTTP(recPushAnalytics, httptest.NewRequest(http.MethodPost, "/api/sync/mat/push-analytics", bytes.NewReader(pushAnalyticsBytes)))
	if recPushAnalytics.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from POST /api/sync/mat/push-analytics, got %d (%s)", recPushAnalytics.Code, recPushAnalytics.Body.String())
	}
	rawAnalytics := recPushAnalytics.Body.String()
	if strings.Contains(rawAnalytics, m1Seeded.ActivePupilID) || strings.Contains(rawAnalytics, "Lovelace") {
		t.Fatalf("PII LEAK IN MAT ANALYTICS: found pupil identifier or name in %s", rawAnalytics)
	}

	// 3. Enabling real MAT sync mode without mat_sync_feed_url in __system_config must fail fast
	err = h.ConfigureMATSyncMode(ctx, true)
	if err == nil || !strings.Contains(err.Error(), "REAL_MAT_SYNC_NOT_CONFIGURED") {
		t.Fatalf("expected REAL_MAT_SYNC_NOT_CONFIGURED error when enabling real MAT sync without config, got: %v", err)
	}
}
