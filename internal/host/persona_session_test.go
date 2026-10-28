package host_test

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

	var foundAdmin, foundDSL, foundTeacher, foundTerminated bool
	for _, p := range personas {
		switch p.StaffID {
		case seeded.AdminStaffID:
			foundAdmin = true
		case seeded.DSLStaffID:
			foundDSL = true
			if !p.IsDSL {
				t.Errorf("expected DSL persona IsDSL == true")
			}
		case seeded.TeacherStaffID:
			foundTeacher = true
		case seeded.TerminatedStaffID:
			foundTerminated = true
		}
	}
	if !foundAdmin || !foundDSL || !foundTeacher {
		t.Fatalf("expected Admin, DSL, and Teacher in active personas list, got: %+v", personas)
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
	rpcPayload, _ := json.Marshal(map[string]any{"args": map[string]any{"pupil_id": seeded.PupilID}})
	reqRPC := httptest.NewRequest(http.MethodPost, "/api/plugins/uk.org.optimis.test/rpc/get_pupil_record", bytes.NewReader(rpcPayload))
	reqRPC.AddCookie(cookies[0])
	recRPC := httptest.NewRecorder()
	handler.ServeHTTP(recRPC, reqRPC)

	if recRPC.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 BadRequest when Teacher lacks dal.pupils.read:all, got %d (%s)", recRPC.Code, recRPC.Body.String())
	}
}
