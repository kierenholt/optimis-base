package host

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"optimis/internal/dal"
)

func TestDashboardLayoutsPersistence_RoleDefaultsTeacherReorderAndRevokedWidgetResolution_PersistsInCoreDB_test(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	pluginsDir, err := filepath.Abs("../../plugins")
	if err != nil {
		t.Fatalf("failed to resolve plugins directory: %v", err)
	}

	t.Setenv("ENABLE_DEV_PERSONA_SWITCHER", "true")

	h, err := NewHost(Config{
		DataDirPath:      tempDir,
		SchoolURN:        "100001",
		HostKMSKeyHandle: "local-dev-kms-key-01",
		PluginsDir:       pluginsDir,
	})
	if err != nil {
		t.Fatalf("NewHost failed: %v", err)
	}
	logCursor := h.LogCursor()
	t.Cleanup(func() {
		entries := h.LogsSince(logCursor)
		if t.Failed() && len(entries) > 0 {
			t.Logf("--- TAPPED HOST SERVICE LOGS ---\n%s", strings.Join(entries, "\n"))
		}
		_ = h.Close()
	})

	if err := h.LoadPlugin("test_plugin"); err != nil {
		t.Fatalf("LoadPlugin(test_plugin) failed: %v", err)
	}
	testPlugin, ok := h.GetLoadedPlugin("uk.org.optimis.test")
	if !ok || testPlugin.PlugID == "" {
		t.Fatalf("expected uk.org.optimis.test to be loaded with non-empty PlugID")
	}

	fx, err := h.DAL().SeedPersonaTestFixtures(ctx, "ay_2026_2027", "2026-09-01")
	if err != nil {
		t.Fatalf("SeedPersonaTestFixtures failed: %v", err)
	}
	t.Cleanup(func() {
		_ = h.DAL().DeletePersonaTestFixtures(context.Background(), fx.FixtureID)
	})

	handler, err := h.NewHTTPHandler()
	if err != nil {
		t.Fatalf("NewHTTPHandler failed: %v", err)
	}

	// 1. Teacher saves an ordered 1D array of { plug_id, widget_id } objects via PUT /api/dashboard/layout
	teacherLayout := []dal.DashboardWidgetItem{
		{PlugID: testPlugin.PlugID, WidgetID: "pastoral_watch"},
		{PlugID: testPlugin.PlugID, WidgetID: "attendance_pulse"},
	}
	putBody, _ := json.Marshal(map[string]any{
		"staff_id": fx.TeacherStaffID,
		"layout":   teacherLayout,
	})
	reqPut := httptest.NewRequest(http.MethodPut, "/api/dashboard/layout", bytes.NewReader(putBody))
	recPut := httptest.NewRecorder()
	handler.ServeHTTP(recPut, reqPut)
	if recPut.Code != http.StatusOK {
		t.Fatalf("PUT /api/dashboard/layout expected 200 OK, got %d: %s", recPut.Code, recPut.Body.String())
	}

	// Verify GET /api/dashboard/layout returns exact 1D array in order
	reqGet := httptest.NewRequest(http.MethodGet, "/api/dashboard/layout?staff_id="+fx.TeacherStaffID, nil)
	recGet := httptest.NewRecorder()
	handler.ServeHTTP(recGet, reqGet)
	if recGet.Code != http.StatusOK {
		t.Fatalf("GET /api/dashboard/layout expected 200 OK, got %d: %s", recGet.Code, recGet.Body.String())
	}
	var getResp struct {
		OK           bool                      `json:"ok"`
		LayoutSource string                    `json:"layout_source"`
		Layout       []dal.DashboardWidgetItem `json:"layout"`
	}
	if err := json.Unmarshal(recGet.Body.Bytes(), &getResp); err != nil {
		t.Fatalf("failed to decode GET /api/dashboard/layout: %v", err)
	}
	if getResp.LayoutSource != "STAFF" || len(getResp.Layout) != 2 {
		t.Fatalf("expected STAFF layout with 2 items, got source=%q layout=%+v", getResp.LayoutSource, getResp.Layout)
	}
	if getResp.Layout[0].WidgetID != "pastoral_watch" || getResp.Layout[1].WidgetID != "attendance_pulse" {
		t.Fatalf("expected [pastoral_watch, attendance_pulse], got %+v", getResp.Layout)
	}

	// 2. Teacher removes pastoral_watch leaving [attendance_pulse]
	singleLayout := []dal.DashboardWidgetItem{
		{PlugID: testPlugin.PlugID, WidgetID: "attendance_pulse"},
	}
	putBody2, _ := json.Marshal(map[string]any{
		"staff_id": fx.TeacherStaffID,
		"layout":   singleLayout,
	})
	reqPut2 := httptest.NewRequest(http.MethodPut, "/api/dashboard/layout", bytes.NewReader(putBody2))
	recPut2 := httptest.NewRecorder()
	handler.ServeHTTP(recPut2, reqPut2)
	if recPut2.Code != http.StatusOK {
		t.Fatalf("PUT /api/dashboard/layout expected 200 OK, got %d", recPut2.Code)
	}

	// 3. Non-admin attempting POST /api/admin/dashboard/role-default/append fails fast with 403 FORBIDDEN_ADMIN_ROLE_REQUIRED
	appendBodyForbidden, _ := json.Marshal(map[string]any{
		"staff_id":  fx.TeacherStaffID,
		"role_id":   "TEACHER",
		"plug_id":   testPlugin.PlugID,
		"widget_id": "pastoral_watch",
	})
	reqForbidden := httptest.NewRequest(http.MethodPost, "/api/admin/dashboard/role-default/append", bytes.NewReader(appendBodyForbidden))
	recForbidden := httptest.NewRecorder()
	handler.ServeHTTP(recForbidden, reqForbidden)
	if recForbidden.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden when non-admin appends role default widget, got %d: %s", recForbidden.Code, recForbidden.Body.String())
	}

	// 4. Admin appends pastoral_watch to TEACHER role default -> updates role default AND appends to existing TEACHER staff layout
	appendBodyAdmin, _ := json.Marshal(map[string]any{
		"staff_id":  fx.AdminStaffID,
		"role_id":   "TEACHER",
		"plug_id":   testPlugin.PlugID,
		"widget_id": "pastoral_watch",
	})
	reqAppend := httptest.NewRequest(http.MethodPost, "/api/admin/dashboard/role-default/append", bytes.NewReader(appendBodyAdmin))
	recAppend := httptest.NewRecorder()
	handler.ServeHTTP(recAppend, reqAppend)
	if recAppend.Code != http.StatusOK {
		t.Fatalf("POST /api/admin/dashboard/role-default/append expected 200 OK, got %d: %s", recAppend.Code, recAppend.Body.String())
	}

	reqGetAfterAppend := httptest.NewRequest(http.MethodGet, "/api/dashboard/layout?staff_id="+fx.TeacherStaffID, nil)
	recGetAfterAppend := httptest.NewRecorder()
	handler.ServeHTTP(recGetAfterAppend, reqGetAfterAppend)
	if err := json.Unmarshal(recGetAfterAppend.Body.Bytes(), &getResp); err != nil {
		t.Fatalf("failed to decode GET /api/dashboard/layout after append: %v", err)
	}
	if len(getResp.Layout) != 2 || getResp.Layout[0].WidgetID != "attendance_pulse" || getResp.Layout[1].WidgetID != "pastoral_watch" {
		t.Fatalf("expected [attendance_pulse, pastoral_watch] after Admin role append, got %+v", getResp.Layout)
	}

	// 5. Revoked / unpermitted widget in saved 1D array resolves with permitted=false and error_code for inline permission error card
	revokedLayout := []dal.DashboardWidgetItem{
		{PlugID: testPlugin.PlugID, WidgetID: "attendance_pulse"},
		{PlugID: "revoked99", WidgetID: "restricted_safeguarding_widget"},
	}
	putBody3, _ := json.Marshal(map[string]any{
		"staff_id": fx.TeacherStaffID,
		"layout":   revokedLayout,
	})
	reqPut3 := httptest.NewRequest(http.MethodPut, "/api/dashboard/layout", bytes.NewReader(putBody3))
	recPut3 := httptest.NewRecorder()
	handler.ServeHTTP(recPut3, reqPut3)
	if recPut3.Code != http.StatusOK {
		t.Fatalf("PUT /api/dashboard/layout with revoked widget expected 200 OK, got %d", recPut3.Code)
	}

	reqGetResolved := httptest.NewRequest(http.MethodGet, "/api/dashboard/layout?staff_id="+fx.TeacherStaffID, nil)
	recGetResolved := httptest.NewRecorder()
	handler.ServeHTTP(recGetResolved, reqGetResolved)
	var resolvedResp struct {
		OK              bool                          `json:"ok"`
		ResolvedWidgets []ResolvedDashboardWidgetSlot `json:"resolved_widgets"`
	}
	if err := json.Unmarshal(recGetResolved.Body.Bytes(), &resolvedResp); err != nil {
		t.Fatalf("failed to decode resolved_widgets: %v", err)
	}
	if len(resolvedResp.ResolvedWidgets) != 2 {
		t.Fatalf("expected 2 resolved widget slots, got %+v", resolvedResp.ResolvedWidgets)
	}
	if !resolvedResp.ResolvedWidgets[0].Permitted || resolvedResp.ResolvedWidgets[0].WidgetID != "attendance_pulse" {
		t.Fatalf("expected slot 0 to be permitted attendance_pulse, got %+v", resolvedResp.ResolvedWidgets[0])
	}
	if resolvedResp.ResolvedWidgets[1].Permitted || resolvedResp.ResolvedWidgets[1].ErrorCode == "" {
		t.Fatalf("expected slot 1 to be unpermitted with non-empty error_code, got %+v", resolvedResp.ResolvedWidgets[1])
	}

	// 6. Cross-staff spoofing protection: when Teacher has an active mis_session cookie and sends PUT /api/dashboard/layout with HoD's staff_id, Host fails fast with 403 FORBIDDEN_STAFF_ID_MISMATCH
	switchBody, _ := json.Marshal(map[string]any{"staff_id": fx.TeacherStaffID})
	reqSwitch := httptest.NewRequest(http.MethodPost, "/api/auth/persona", bytes.NewReader(switchBody))
	recSwitch := httptest.NewRecorder()
	handler.ServeHTTP(recSwitch, reqSwitch)
	if recSwitch.Code != http.StatusOK {
		t.Fatalf("POST /api/auth/persona expected 200 OK, got %d: %s", recSwitch.Code, recSwitch.Body.String())
	}
	var teacherCookie *http.Cookie
	for _, c := range recSwitch.Result().Cookies() {
		if c.Name == SessionCookieName {
			teacherCookie = c
			break
		}
	}
	if teacherCookie == nil {
		t.Fatalf("expected mis_session cookie from /api/auth/persona")
	}

	spoofPutBody, _ := json.Marshal(map[string]any{
		"staff_id": fx.HoDStaffID,
		"layout":   singleLayout,
	})
	reqSpoofPut := httptest.NewRequest(http.MethodPut, "/api/dashboard/layout", bytes.NewReader(spoofPutBody))
	reqSpoofPut.AddCookie(teacherCookie)
	recSpoofPut := httptest.NewRecorder()
	handler.ServeHTTP(recSpoofPut, reqSpoofPut)
	if recSpoofPut.Code != http.StatusForbidden || !strings.Contains(recSpoofPut.Body.String(), "FORBIDDEN_STAFF_ID_MISMATCH") {
		t.Fatalf("expected 403 FORBIDDEN_STAFF_ID_MISMATCH when Teacher session cookie attempts to overwrite HoD layout, got %d: %s", recSpoofPut.Code, recSpoofPut.Body.String())
	}
}

