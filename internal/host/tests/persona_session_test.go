package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInitAuth_DevPersonaSwitcherFalseAndRealSSONotConfigured_FailsFast_test(t *testing.T) {
	ctx := context.Background()
	h, cleanup := setupIsolatedTestHost(t)
	t.Cleanup(cleanup)

	err := h.ConfigureAuthMode(ctx, false)
	if err == nil {
		t.Fatalf("expected fail-fast error when ENABLE_DEV_PERSONA_SWITCHER=false and real SSO is not configured, got nil")
	}
	if !strings.Contains(err.Error(), "REAL_SSO_NOT_CONFIGURED") {
		t.Errorf("expected error to contain REAL_SSO_NOT_CONFIGURED, got: %v", err)
	}
}

func TestResolveStaffPersona_SeededPersonasInCoreDB_ReturnsActiveRolesAndMergedScopesAndRejectsTerminatedStaff_test(t *testing.T) {
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

	personas, err := h.DAL().ListActiveStaffPersonas(ctx)
	if err != nil {
		t.Fatalf("ListActiveStaffPersonas failed: %v", err)
	}

	var foundAdmin, foundDSL, foundHoD, foundTeacher, foundTerminated bool
	for _, p := range personas {
		switch p.StaffID {
		case seeded.AdminStaffID:
			foundAdmin = true
		case seeded.DSLStaffID:
			foundDSL = true
			if !p.IsDSL {
				t.Errorf("expected DSL persona IsDSL == true")
			}
		case seeded.HoDStaffID:
			foundHoD = true
			if len(p.ActiveRoles) == 0 || p.ActiveRoles[0] != "HOD" {
				t.Errorf("expected HoD persona active_roles to include HOD, got %v", p.ActiveRoles)
			}
		case seeded.TeacherStaffID:
			foundTeacher = true
		case seeded.TerminatedStaffID:
			foundTerminated = true
		}
	}
	if !foundAdmin || !foundDSL || !foundHoD || !foundTeacher {
		t.Fatalf("expected Admin, DSL, HoD, and Teacher in active personas list, got: %+v", personas)
	}
	if foundTerminated {
		t.Fatalf("SECURITY VIOLATION: terminated staff member %s was returned in active personas list", seeded.TerminatedStaffID)
	}

	// Resolving terminated staff member by ID must fail fast
	if _, err := h.DAL().ResolveStaffPersonaByID(ctx, seeded.TerminatedStaffID); err == nil {
		t.Fatalf("expected ResolveStaffPersonaByID to fail fast for terminated staff %s", seeded.TerminatedStaffID)
	}
}

func TestSessionCookieResolution_SwitchPersonaAndExecuteRPC_EnforcesSessionCallerContext_test(t *testing.T) {
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

	handler, err := h.NewHTTPHandler()
	if err != nil {
		t.Fatalf("NewHTTPHandler failed: %v", err)
	}

	// 1. Switch persona to Teacher (who only has :own_groups scopes)
	switchPayload, _ := json.Marshal(map[string]any{"staff_id": seeded.TeacherStaffID})
	reqSwitch := httptest.NewRequest(http.MethodPost, "/api/auth/persona", bytes.NewReader(switchPayload))
	recSwitch := httptest.NewRecorder()
	handler.ServeHTTP(recSwitch, reqSwitch)

	if recSwitch.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from POST /api/auth/persona, got %d (%s)", recSwitch.Code, recSwitch.Body.String())
	}
	cookies := recSwitch.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatalf("expected mis_session cookie to be set")
	}

	// 2. Execute RPC get_pupil_record (requires dal.pupils.read:all) with Teacher session cookie -> must fail fast
	testPlugin, ok := h.GetLoadedPlugin("uk.org.optimis.test")
	if !ok || testPlugin.PlugID == "" {
		t.Fatalf("expected uk.org.optimis.test to be loaded with non-empty PlugID")
	}
	rpcPayload, _ := json.Marshal(map[string]any{"args": map[string]any{"pupil_id": seeded.PupilID}})
	reqRPC := httptest.NewRequest(http.MethodPost, "/api/p/"+testPlugin.PlugID+"/rpc/get_pupil_record", bytes.NewReader(rpcPayload))
	reqRPC.AddCookie(cookies[0])
	recRPC := httptest.NewRecorder()
	handler.ServeHTTP(recRPC, reqRPC)

	if recRPC.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 BadRequest when Teacher lacks dal.pupils.read:all, got %d (%s)", recRPC.Code, recRPC.Body.String())
	}
}

func TestResolveRequestCaller_TestDefaultPersonaEnvSet_ResolvesConfiguredPersonaAndRejectsUnknownStaff_test(t *testing.T) {
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

	// Clear any in-memory devPersonaStaffID so TEST_DEFAULT_PERSONA is evaluated
	h.ResetDevPersonaSessions()

	t.Setenv("TEST_DEFAULT_PERSONA", seeded.HoDStaffID)
	req := httptest.NewRequest(http.MethodGet, "/api/auth/session", nil)
	caller, err := h.ResolveRequestCaller(req, "")
	if err != nil {
		t.Fatalf("expected ResolveRequestCaller to resolve TEST_DEFAULT_PERSONA=%s, got error: %v", seeded.HoDStaffID, err)
	}
	if caller.StaffID != seeded.HoDStaffID {
		t.Fatalf("expected caller.StaffID=%s from TEST_DEFAULT_PERSONA, got %s", seeded.HoDStaffID, caller.StaffID)
	}

	// Setting TEST_DEFAULT_PERSONA to a nonexistent staff ID must fail fast
	h.ResetDevPersonaSessions()
	t.Setenv("TEST_DEFAULT_PERSONA", "stf_nonexistent_999")
	if _, err := h.ResolveRequestCaller(req, ""); err == nil {
		t.Fatalf("expected ResolveRequestCaller to fail fast when TEST_DEFAULT_PERSONA is unknown")
	}
}

func TestResolveRequestCaller_IgnoresQueryAndHeaderOverridesAndEnforcesSessionInProdMode_test(t *testing.T) {
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

	h.ResetDevPersonaSessions()
	h.SetActiveDevPersona(seeded.TeacherStaffID)

	// 1. Query and HTTP header overrides (?__persona, ?staff_id, X-MIS-Persona, X-MIS-Staff-ID) must be completely ignored
	reqSpoof := httptest.NewRequest(
		http.MethodGet,
		"/api/auth/session?__persona="+seeded.AdminStaffID+"&staff_id="+seeded.AdminStaffID,
		nil,
	)
	reqSpoof.Header.Set("X-MIS-Persona", seeded.AdminStaffID)
	reqSpoof.Header.Set("X-MIS-Staff-ID", seeded.AdminStaffID)

	caller, err := h.ResolveRequestCaller(reqSpoof, seeded.AdminStaffID)
	if err != nil {
		t.Fatalf("ResolveRequestCaller failed: %v", err)
	}
	if caller.StaffID != seeded.TeacherStaffID {
		t.Fatalf("SECURITY VIOLATION: request query/header/explicit staff override leaked caller.StaffID=%s (expected %s)", caller.StaffID, seeded.TeacherStaffID)
	}

	// 2. When ENABLE_DEV_PERSONA_SWITCHER=false (production mode), unauthenticated requests without mis_session must fail fast with UNAUTHORIZED_SESSION_REQUIRED even if SSO is configured and TEST_DEFAULT_PERSONA is set
	if _, err := h.DAL().SetSystemConfigValue(ctx, "sso_provider", "ENTRA_ID"); err != nil {
		t.Fatalf("SetSystemConfigValue(sso_provider) failed: %v", err)
	}
	if _, err := h.DAL().SetSystemConfigValue(ctx, "sso_oidc_issuer_url", "https://login.microsoftonline.com/common/v2.0"); err != nil {
		t.Fatalf("SetSystemConfigValue(sso_oidc_issuer_url) failed: %v", err)
	}
	if _, err := h.DAL().SetSystemConfigValue(ctx, "sso_oidc_client_id", "optimis-prod-client-id"); err != nil {
		t.Fatalf("SetSystemConfigValue(sso_oidc_client_id) failed: %v", err)
	}
	if err := h.ConfigureAuthMode(ctx, false); err != nil {
		t.Fatalf("ConfigureAuthMode(false) failed: %v", err)
	}

	t.Setenv("TEST_DEFAULT_PERSONA", seeded.AdminStaffID)
	if _, err := h.ResolveRequestCaller(reqSpoof, seeded.AdminStaffID); err == nil || !strings.Contains(err.Error(), "UNAUTHORIZED_SESSION_REQUIRED") {
		t.Fatalf("expected UNAUTHORIZED_SESSION_REQUIRED in production mode without mis_session cookie, got: %v", err)
	}
}


