package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"optimis/internal/host"
)

type uiConfigResponse struct {
	OK                 bool                         `json:"ok"`
	SchoolURN          string                       `json:"school_urn"`
	SchoolName         string                       `json:"school_name"`
	AcademicYearID     string                       `json:"academic_year_id"`
	Themes             map[string]map[string]string `json:"themes"`
	EntityLinkHandlers map[string]string            `json:"entity_link_handlers"`
	PinnedRecents      []host.PinnedRecentItem      `json:"pinned_recents"`
	UserSettings       struct {
		StaffID      string `json:"staff_id"`
		BaseFontSize string `json:"base_font_size"`
		ThemeMode    string `json:"theme_mode"`
	} `json:"user_settings"`
	Plugins []struct {
		ID               string   `json:"id"`
		PlugID           string   `json:"plug_id"`
		Title            string   `json:"title"`
		EntryURL         string   `json:"entry_url"`
		PermittedMethods []string `json:"permitted_methods"`
		Widgets          []struct {
			ID       string `json:"id"`
			Title    string `json:"title"`
			Entry    string `json:"entry"`
			EntryURL string `json:"entry_url"`
			Width    int    `json:"width"`
			Height   int    `json:"height"`
		} `json:"widgets"`
	} `json:"plugins"`
}

func setupUITestHost(t *testing.T, enablePersonaSwitcher bool, pluginDirs ...string) (*host.Host, http.Handler) {
	t.Helper()
	pluginsDir, err := filepath.Abs("../../../plugins")
	if err != nil {
		t.Fatalf("failed to resolve plugins directory: %v", err)
	}
	if enablePersonaSwitcher {
		t.Setenv("ENABLE_DEV_PERSONA_SWITCHER", "true")
	}
	h, err := host.NewHost(host.Config{DataDirPath: t.TempDir(), SchoolURN: "100001", HostKMSKeyHandle: "local-dev-kms-key-01", PluginsDir: pluginsDir})
	if err != nil {
		t.Fatalf("NewHost failed: %v", err)
	}
	logCursor := h.LogCursor()
	t.Cleanup(func() {
		if entries := h.LogsSince(logCursor); t.Failed() && len(entries) > 0 {
			t.Logf("--- TAPPED HOST SERVICE LOGS ---\n%s", strings.Join(entries, "\n"))
		}
		_ = h.Close()
	})
	for _, p := range pluginDirs {
		if err := h.LoadPlugin(p); err != nil {
			t.Fatalf("LoadPlugin(%s) failed: %v", p, err)
		}
	}
	handler, err := h.NewHTTPHandler()
	if err != nil {
		t.Fatalf("NewHTTPHandler failed: %v", err)
	}
	return h, handler
}

func loginPersona(t *testing.T, handler http.Handler, staffID string) *http.Cookie {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"staff_id": staffID})
	rec := executeRequest(handler, http.MethodPost, "/api/auth/persona", body, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /api/auth/persona (%s) expected 200, got %d: %s", staffID, rec.Code, rec.Body.String())
	}
	for _, c := range rec.Result().Cookies() {
		if c.Name == host.SessionCookieName || c.Name == "mis_session" {
			return c
		}
	}
	t.Fatalf("expected session cookie from POST /api/auth/persona for %s", staffID)
	return nil
}

func executeRequest(handler http.Handler, method, target string, body any, cookie *http.Cookie, headers ...string) *httptest.ResponseRecorder {
	var bodyReader io.Reader
	var isJSON bool
	if body != nil {
		switch v := body.(type) {
		case string:
			bodyReader = strings.NewReader(v)
		case []byte:
			bodyReader = bytes.NewReader(v)
		case io.Reader:
			bodyReader = v
		default:
			data, _ := json.Marshal(v)
			bodyReader = bytes.NewReader(data)
			isJSON = true
		}
	}
	req := httptest.NewRequest(method, target, bodyReader)
	if isJSON {
		req.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	for i := 0; i < len(headers)-1; i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func decodeJSON[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("failed to decode JSON from %s: %v", rec.Body.String(), err)
	}
	return out
}

func postPinnedRecent(handler http.Handler, cookie *http.Cookie, payload map[string]any) *httptest.ResponseRecorder {
	return executeRequest(handler, http.MethodPost, "/api/system/pinned-recents", payload, cookie)
}

func createTempPlugin(t *testing.T, folderName, manifestJSON string) string {
	t.Helper()
	pluginDir := filepath.Join(t.TempDir(), folderName)
	if err := os.MkdirAll(filepath.Join(pluginDir, "src"), 0o755); err != nil {
		t.Fatalf("MkdirAll failed for %s: %v", pluginDir, err)
	}
	_ = os.WriteFile(filepath.Join(pluginDir, "src", "init.lua"), []byte(`function ping(ctx, args) return { ok = true } end`), 0o644)
	_ = os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(manifestJSON), 0o644)
	return pluginDir
}

func assertRouteStatus(t *testing.T, handler http.Handler, method, url string, wantCode int, wantSubstr string, headers ...string) {
	t.Helper()
	rec := executeRequest(handler, method, url, nil, nil, headers...)
	if rec.Code != wantCode || (wantSubstr != "" && !strings.Contains(strings.ToLower(rec.Body.String()), strings.ToLower(wantSubstr))) {
		t.Fatalf("%s %s expected %d containing %q, got %d: %s", method, url, wantCode, wantSubstr, rec.Code, rec.Body.String())
	}
}

func TestServePluginUISDKAssets_MisBridgeJsAndMisThemeCss_ReturnsAssetsWithConnectSrcNoneCSP_test(t *testing.T) {
	h, handler := setupUITestHost(t, true, "test_plugin")
	testPlugin, ok := h.GetLoadedPlugin("uk.org.optimis.test")
	if !ok || testPlugin.PlugID == "" {
		t.Fatalf("expected uk.org.optimis.test to be loaded with non-empty PlugID")
	}

	for _, tc := range []struct{ url, contentType, mustContain string }{
		{"/sdk/mis-bridge.js", "application/javascript", "window.mis"},
		{"/sdk/mis-theme.css", "text/css", "--mis-brand-primary"},
		{"/p/" + testPlugin.PlugID + "/ui/mis-bridge.js", "application/javascript", "mis:rpc:request"},
		{"/p/" + testPlugin.PlugID + "/ui/mis-theme.css", "text/css", "high-contrast"},
	} {
		rec := executeRequest(handler, http.MethodGet, tc.url, nil, nil)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Header().Get("Content-Type"), tc.contentType) ||
			!strings.Contains(rec.Header().Get("Content-Security-Policy"), "connect-src 'none'") ||
			!strings.Contains(rec.Body.String(), tc.mustContain) {
			t.Fatalf("GET %s failed: code=%d, type=%q, csp=%q, body=%s", tc.url, rec.Code, rec.Header().Get("Content-Type"), rec.Header().Get("Content-Security-Policy"), rec.Body.String())
		}
	}
}

func TestGetSystemUIConfig_DefaultAndSeededSystemConfig_ReturnsSchoolThemeTokensPluginsAndEntityLinkHandlers_test(t *testing.T) {
	h, handler := setupUITestHost(t, true, "test_plugin")
	testPlugin, ok := h.GetLoadedPlugin("uk.org.optimis.test")
	if !ok || testPlugin.PlugID == "" {
		t.Fatalf("expected uk.org.optimis.test to be loaded with non-empty PlugID")
	}

	rec := executeRequest(handler, http.MethodGet, "/api/system/ui-config", nil, nil)
	if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), "uk.org.optimis.test") {
		t.Fatalf("GET /api/system/ui-config unexpected response (code %d, leaked manifest id): %s", rec.Code, rec.Body.String())
	}

	payload := decodeJSON[uiConfigResponse](t, rec)
	if !payload.OK || payload.SchoolURN != "100001" {
		t.Fatalf("expected ok=true and school_urn=100001, got %+v", payload)
	}
	if payload.Themes["light"]["--mis-brand-primary"] != "#1d4ed8" || payload.Themes["high-contrast"]["--mis-brand-primary"] != "#facc15" {
		t.Fatalf("unexpected theme tokens: %+v", payload.Themes)
	}
	if payload.EntityLinkHandlers["student"] != testPlugin.PlugID {
		t.Fatalf("expected student entity handler %s, got %q", testPlugin.PlugID, payload.EntityLinkHandlers["student"])
	}
	if len(payload.Plugins) == 0 || payload.Plugins[0].ID != testPlugin.PlugID || payload.Plugins[0].PlugID != testPlugin.PlugID ||
		payload.Plugins[0].EntryURL != "/p/"+testPlugin.PlugID+"/ui/index.html" {
		t.Fatalf("unexpected active plugins list: %+v", payload.Plugins)
	}
}

func TestGetSystemUIConfig_CallerWithRestrictedPersonaScopes_FiltersPermittedMethodsAndHidesUnauthorizedPlugins_test(t *testing.T) {
	ctx := context.Background()
	h, handler := setupUITestHost(t, true, "test_plugin")
	testPlugin, ok := h.GetLoadedPlugin("uk.org.optimis.test")
	if !ok || testPlugin.PlugID == "" {
		t.Fatalf("expected uk.org.optimis.test to be loaded with non-empty PlugID")
	}

	writeDB := h.DAL().WriteDB()
	seedSQL := []string{
		`INSERT INTO roles (id, category, name, name_short, description, default_scopes_json) VALUES ('ROLE_POSTBOX_ONLY', 'STAFF', 'Safeguarding Postbox Only', 'PBO', 'Write-only safeguarding postbox role', '["dal.safeguarding.create"]')`,
		`INSERT INTO roles (id, category, name, name_short, description, default_scopes_json) VALUES ('ROLE_NO_PLUGIN_SCOPES', 'STAFF', 'Unmatched Scope Role', 'UMS', 'Holds scope not used by test_plugin', '["dal.sen.read:own_groups"]')`,
		`INSERT INTO persons (id, external_sso_sub, title, legal_first_name, legal_last_name, preferred_first_name, preferred_last_name, sex, status) VALUES ('per_postbox_only', 'sso_postbox_only', 'Ms', 'Clara', 'Oswald', 'Clara', 'Oswald', 'F', 'ACTIVE')`,
		`INSERT INTO person_roles (person_id, role_id, is_primary, effective_from) VALUES ('per_postbox_only', 'ROLE_POSTBOX_ONLY', 1, '2026-09-01')`,
		`INSERT INTO staff (id, person_id, staff_code, staff_type, job_title, qts_flag, is_dsl, hire_date, termination_date) VALUES ('stf_postbox_only', 'per_postbox_only', 'PBO01', 'SUPPORT', 'Pastoral Assistant', 0, 0, '2026-09-01', NULL)`,
		`INSERT INTO persons (id, external_sso_sub, title, legal_first_name, legal_last_name, preferred_first_name, preferred_last_name, sex, status) VALUES ('per_no_plugin_scopes', 'sso_no_plugin_scopes', 'Mr', 'Rory', 'Williams', 'Rory', 'Williams', 'M', 'ACTIVE')`,
		`INSERT INTO person_roles (person_id, role_id, is_primary, effective_from) VALUES ('per_no_plugin_scopes', 'ROLE_NO_PLUGIN_SCOPES', 1, '2026-09-01')`,
		`INSERT INTO staff (id, person_id, staff_code, staff_type, job_title, qts_flag, is_dsl, hire_date, termination_date) VALUES ('stf_no_plugin_scopes', 'per_no_plugin_scopes', 'UMS01', 'SUPPORT', 'SEN Support Assistant', 0, 0, '2026-09-01', NULL)`,
	}
	for _, stmt := range seedSQL {
		if _, err := writeDB.ExecContext(ctx, stmt); err != nil {
			t.Fatalf("failed to seed restricted persona SQL: %v", err)
		}
	}
	t.Cleanup(func() {
		for _, stmt := range []string{
			`DELETE FROM staff WHERE id IN ('stf_postbox_only', 'stf_no_plugin_scopes')`,
			`DELETE FROM person_roles WHERE person_id IN ('per_postbox_only', 'per_no_plugin_scopes')`,
			`DELETE FROM persons WHERE id IN ('per_postbox_only', 'per_no_plugin_scopes')`,
			`DELETE FROM roles WHERE id IN ('ROLE_POSTBOX_ONLY', 'ROLE_NO_PLUGIN_SCOPES')`,
		} {
			_, _ = writeDB.ExecContext(context.Background(), stmt)
		}
		h.ResetDevPersonaSessions()
	})

	// Case 1: stf_postbox_only holds only dal.safeguarding.create
	cookie1 := loginPersona(t, handler, "stf_postbox_only")
	rec1 := executeRequest(handler, http.MethodGet, "/api/system/ui-config", nil, cookie1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("GET /api/system/ui-config expected 200, got %d: %s", rec1.Code, rec1.Body.String())
	}
	payload1 := decodeJSON[uiConfigResponse](t, rec1)
	if len(payload1.Plugins) != 1 || payload1.Plugins[0].ID != testPlugin.PlugID ||
		len(payload1.Plugins[0].PermittedMethods) != 1 || payload1.Plugins[0].PermittedMethods[0] != "submit_safeguarding_postbox" {
		t.Fatalf("expected strictly submit_safeguarding_postbox, got %+v", payload1.Plugins)
	}

	// Case 2: stf_no_plugin_scopes holds 0 matching scopes -> plugin omitted
	cookie2 := loginPersona(t, handler, "stf_no_plugin_scopes")
	rec2 := executeRequest(handler, http.MethodGet, "/api/system/ui-config", nil, cookie2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("GET /api/system/ui-config expected 200, got %d: %s", rec2.Code, rec2.Body.String())
	}
	payload2 := decodeJSON[uiConfigResponse](t, rec2)
	if len(payload2.Plugins) != 0 {
		t.Fatalf("expected 0 plugins for stf_no_plugin_scopes, got %+v", payload2.Plugins)
	}
}

func TestManifestWidgetsAndSeedLruPlugins_ValidAndInvalidWidgetDeclarations_ExposesWidgetsInUIConfigAndFailsFastOnInvalidEntry_test(t *testing.T) {
	h, handler := setupUITestHost(t, true, "test_plugin")
	testPlugin, ok := h.GetLoadedPlugin("uk.org.optimis.test")
	if !ok || testPlugin.PlugID == "" {
		t.Fatalf("expected uk.org.optimis.test to be loaded with non-empty PlugID")
	}

	// 1. Verify GET /p/{plug_id}/ui/widget.html serves widget HTML with connect-src 'none' CSP
	recWidget := executeRequest(handler, http.MethodGet, "/p/"+testPlugin.PlugID+"/ui/widget.html", nil, nil)
	if recWidget.Code != http.StatusOK || !strings.Contains(recWidget.Header().Get("Content-Security-Policy"), "connect-src 'none'") {
		t.Fatalf("widget.html failed CSP check: code=%d, csp=%q", recWidget.Code, recWidget.Header().Get("Content-Security-Policy"))
	}

	// 2. Seed synthetic LRU plugins via POST /api/test/seed-lru-plugins and verify GET /api/system/ui-config
	seedBody := map[string]any{"plugin_ids": []string{"uk.org.optimis.lru1", "uk.org.optimis.lru2"}}
	if rec := executeRequest(handler, http.MethodPost, "/api/test/seed-lru-plugins", seedBody, nil); rec.Code != http.StatusCreated {
		t.Fatalf("POST /api/test/seed-lru-plugins expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
	}

	recConfig := executeRequest(handler, http.MethodGet, "/api/system/ui-config", nil, nil)
	if recConfig.Code != http.StatusOK {
		t.Fatalf("GET /api/system/ui-config expected 200 OK, got %d", recConfig.Code)
	}
	configPayload := decodeJSON[uiConfigResponse](t, recConfig)
	if len(configPayload.Plugins) != 3 {
		t.Fatalf("expected 3 active plugins, got %d", len(configPayload.Plugins))
	}

	var foundTestPlugin bool
	expectedEntryURL := "/p/" + testPlugin.PlugID + "/ui/widget.html"
	for _, p := range configPayload.Plugins {
		if p.ID == testPlugin.PlugID {
			foundTestPlugin = true
			if len(p.Widgets) < 2 || p.Widgets[0].EntryURL != expectedEntryURL ||
				p.Widgets[0].Width != 2 || p.Widgets[0].Height != 1 ||
				p.Widgets[1].Width != 1 || p.Widgets[1].Height != 1 {
				t.Fatalf("unexpected widget config for %s: %+v", testPlugin.PlugID, p.Widgets)
			}
		}
	}
	if !foundTestPlugin {
		t.Fatalf("expected %s in ui-config plugins", testPlugin.PlugID)
	}

	// 3. Verify readPluginFromDisk fails fast when a manifest widget specifies width outside {1, 2}
	badPluginDir := createTempPlugin(t, "invalid_widget_size_plugin", `{
		"id": "uk.org.optimis.invalid.widget.size", "version": "1.0.0", "core_dal_version": "~1.0",
		"views": [{ "id": "home", "path": "/", "entry": "ui/dist/index.html" }],
		"widgets": [{ "id": "bad_size", "title": "Bad Size Widget", "entry": "ui/dist/widget.html", "width": 3, "height": 1 }],
		"endpoints": { "ping": { "scopes": ["dal.pupils.read:own_groups"], "mutates": false } }
	}`)
	if _, err := host.ReadPluginFromDisk(badPluginDir, "invalid_widget_size_plugin"); err == nil || !strings.Contains(err.Error(), "width") {
		t.Fatalf("expected readPluginFromDisk to fail fast on widget width=3, got: %v", err)
	}

	if rec := executeRequest(handler, http.MethodDelete, "/api/test/seed-lru-plugins", nil, nil); rec.Code != http.StatusOK {
		t.Fatalf("DELETE /api/test/seed-lru-plugins expected 200 OK, got %d", rec.Code)
	}
}

func TestOpaquePluginViewRouting_DeclaredAndUndeclaredViewsAndLegacyRoutes_EnforcesDefaultDenyAndServesHostSPA_test(t *testing.T) {
	ctx := context.Background()
	h, handler := setupUITestHost(t, false, "senco_provision_plugin")
	senPlugin, ok := h.GetLoadedPlugin("uk.org.optimis.sen.provision")
	if !ok || !host.OpaquePlugIDPattern.MatchString(senPlugin.PlugID) {
		t.Fatalf("expected 8-char opaque PlugID for senco_provision_plugin, got %q", senPlugin.PlugID)
	}
	initialPlugID := senPlugin.PlugID

	// Verify PlugID persists across RemovePlugin + LoadPlugin via __system_config
	if err := h.RemovePlugin(ctx, initialPlugID); err != nil {
		t.Fatalf("RemovePlugin failed: %v", err)
	}
	if err := h.LoadPlugin("senco_provision_plugin"); err != nil {
		t.Fatalf("Reloading senco_provision_plugin failed: %v", err)
	}
	if reloaded, ok := h.GetLoadedPlugin("uk.org.optimis.sen.provision"); !ok || reloaded.PlugID != initialPlugID {
		t.Fatalf("expected persisted PlugID %q across reinstall", initialPlugID)
	}

	// 1. Declared routes return 200 OK
	for _, validURL := range []string{"/p/" + initialPlugID, "/p/" + initialPlugID + "/student/4092?tab=iep", "/p/" + initialPlugID + "/group/grp_101?tab=overview"} {
		assertRouteStatus(t, handler, http.MethodGet, validURL, http.StatusOK, "")
	}

	// 2. Undeclared view path: 404 with JSON or Host SPA HTML
	assertRouteStatus(t, handler, http.MethodGet, "/p/"+initialPlugID+"/undeclared/subpath", http.StatusNotFound, "UNDECLARED_PLUGIN_VIEW_PATH")
	assertRouteStatus(t, handler, http.MethodGet, "/p/"+initialPlugID+"/undeclared/subpath", http.StatusNotFound, "<!doctype html>", "Accept", "text/html,application/xhtml+xml")

	// 3. Undeclared query param: 400 with JSON or Host SPA HTML
	assertRouteStatus(t, handler, http.MethodGet, "/p/"+initialPlugID+"/student/4092?tab=iep&bogus_param=1", http.StatusBadRequest, "UNDECLARED_PLUGIN_VIEW_QUERY_PARAM")
	assertRouteStatus(t, handler, http.MethodGet, "/p/"+initialPlugID+"/student/4092?tab=iep&bogus_param=1", http.StatusBadRequest, "<!doctype html>", "Accept", "text/html")

	// 4. Static UI assets under /p/{plug_id}/ui/... and /_p_assets/{plug_id}/... never collide with views[]
	for _, assetURL := range []string{"/p/" + initialPlugID + "/ui/index.html", "/_p_assets/" + initialPlugID + "/index.html"} {
		assertRouteStatus(t, handler, http.MethodGet, assetURL, http.StatusOK, "")
	}

	// 5. readPluginFromDisk rejects views[].path starting with /ui (RESERVED_PLUGIN_VIEW_PATH)
	reservedDir := createTempPlugin(t, "reserved_ui_plugin", `{
		"id": "uk.org.optimis.reserved.ui", "version": "1.0.0", "core_dal_version": "~1.0",
		"views": [{ "id": "home", "path": "/", "entry": "ui/dist/index.html" }, { "id": "shadow_ui", "path": "/ui/custom", "entry": "ui/dist/index.html" }],
		"endpoints": { "ping": { "scopes": ["dal.pupils.read:own_groups"], "mutates": false } }
	}`)
	if _, err := host.ReadPluginFromDisk(reservedDir, "reserved_ui_plugin"); err == nil || !strings.Contains(err.Error(), "RESERVED_PLUGIN_VIEW_PATH") {
		t.Fatalf("expected RESERVED_PLUGIN_VIEW_PATH error, got %v", err)
	}

	// 6. Legacy /plugins/... and /api/plugins/... routes return 404
	assertRouteStatus(t, handler, http.MethodGet, "/plugins/uk.org.optimis.sen.provision/ui/index.html", http.StatusNotFound, "")
	assertRouteStatus(t, handler, http.MethodPost, "/api/plugins/uk.org.optimis.sen.provision/rpc/get_sen_differentiation_summary", http.StatusNotFound, "")
}

func TestPinnedRecentsSecurityAndValidation_CrossStaffSpoofingUnvalidatedRoutesDormantPluginAndNoDefaultFallback_EnforcesAllInvariants_test(t *testing.T) {
	ctx := context.Background()
	h, handler := setupUITestHost(t, true, "senco_provision_plugin", "dfe_census_plugin")
	senPlug, ok1 := h.GetLoadedPlugin("uk.org.optimis.sen.provision")
	censusPlug, ok2 := h.GetLoadedPlugin("uk.gov.dfe.census")
	if !ok1 || !ok2 {
		t.Fatalf("expected plugins to be loaded")
	}

	fx, err := h.DAL().SeedPersonaTestFixtures(ctx, "ay_2026_2027", "2026-09-01")
	if err != nil {
		t.Fatalf("SeedPersonaTestFixtures failed: %v", err)
	}
	t.Cleanup(func() { _ = h.DAL().DeletePersonaTestFixtures(context.Background(), fx.FixtureID) })

	teacherCookie := loginPersona(t, handler, fx.TeacherStaffID)

	// 1. Cross-staff spoofing fails fast with 403 FORBIDDEN_STAFF_ID_MISMATCH
	recSpoof := postPinnedRecent(handler, teacherCookie, map[string]any{"staff_id": fx.HoDStaffID, "action": "pin", "route": "/p/" + senPlug.PlugID + "/", "title": "Spoofed Pin"})
	if recSpoof.Code != http.StatusForbidden || !strings.Contains(recSpoof.Body.String(), "FORBIDDEN_STAFF_ID_MISMATCH") {
		t.Fatalf("expected 403 FORBIDDEN_STAFF_ID_MISMATCH, got %d: %s", recSpoof.Code, recSpoof.Body.String())
	}

	// 2. Unvalidated sub-routes & unpermitted plugins are rejected on pin
	for _, tc := range []struct {
		name, route string
		code        int
		substr      string
	}{
		{"unknown plug_id", "/p/zzzz9999/", http.StatusNotFound, "PLUGIN_NOT_FOUND"},
		{"undeclared subpath", "/p/" + senPlug.PlugID + "/undeclared/deep/route", http.StatusNotFound, "UNDECLARED_PLUGIN_VIEW_PATH"},
		{"undeclared query param", "/p/" + senPlug.PlugID + "/?evil_param=1", http.StatusBadRequest, "UNDECLARED_PLUGIN_VIEW_QUERY_PARAM"},
		{"unpermitted plugin", "/p/" + censusPlug.PlugID + "/", http.StatusForbidden, "FORBIDDEN_PLUGIN_ACCESS"},
	} {
		rec := postPinnedRecent(handler, teacherCookie, map[string]any{"staff_id": fx.TeacherStaffID, "action": "pin", "route": tc.route, "title": tc.name})
		if rec.Code != tc.code || !strings.Contains(rec.Body.String(), tc.substr) {
			t.Fatalf("%s expected %d %s, got %d: %s", tc.name, tc.code, tc.substr, rec.Code, rec.Body.String())
		}
	}

	// 3. Valid pin succeeds, unpin never falls back to poisoned .default
	_, _ = h.DAL().SetSystemConfigValue(ctx, "user_prefs.pinned_recents.default", `[{"route":"/p/`+senPlug.PlugID+`/","plug_id":"`+senPlug.PlugID+`","title":"Poisoned"}]`)
	validPinBody := map[string]any{"staff_id": fx.TeacherStaffID, "action": "pin", "route": "/p/" + senPlug.PlugID + "/", "title": "SEN Provision Workspace"}
	recValidPin := postPinnedRecent(handler, teacherCookie, validPinBody)
	recUnpin := postPinnedRecent(handler, teacherCookie, map[string]any{"staff_id": fx.TeacherStaffID, "action": "unpin", "route": "/p/" + senPlug.PlugID + "/"})
	if recValidPin.Code != http.StatusOK || recUnpin.Code != http.StatusOK || len(decodeJSON[uiConfigResponse](t, recUnpin).PinnedRecents) != 0 {
		t.Fatalf("pin/unpin failed: pin=%d, unpin=%d", recValidPin.Code, recUnpin.Code)
	}

	// Re-pin, uninstall plugin -> dormant plugin pins filtered out from ui-config
	_ = postPinnedRecent(handler, teacherCookie, validPinBody)
	if err := h.UninstallPlugin("uk.org.optimis.sen.provision"); err != nil {
		t.Fatalf("UninstallPlugin failed: %v", err)
	}
	recUIConfig := executeRequest(handler, http.MethodGet, "/api/system/ui-config", nil, teacherCookie)
	if recUIConfig.Code != http.StatusOK || len(decodeJSON[uiConfigResponse](t, recUIConfig).PinnedRecents) != 0 {
		t.Fatalf("expected dormant plugin pins filtered out from ui-config, got %d: %s", recUIConfig.Code, recUIConfig.Body.String())
	}
}

func TestUserSettingsAndPinnedRecents_StaffSettingsTablePersistenceAndValidation_EnforcesSchemaAndRBAC_test(t *testing.T) {
	ctx := context.Background()
	h, handler := setupUITestHost(t, true, "test_plugin")
	testPlug, ok := h.GetLoadedPlugin("uk.org.optimis.test")
	if !ok || testPlug == nil {
		t.Fatalf("expected uk.org.optimis.test to be loaded")
	}

	fx, err := h.DAL().SeedPersonaTestFixtures(ctx, "ay_2026_2027", "2026-09-01")
	if err != nil {
		t.Fatalf("SeedPersonaTestFixtures failed: %v", err)
	}
	t.Cleanup(func() { _ = h.DAL().DeletePersonaTestFixtures(ctx, fx.FixtureID) })

	teacherCookie := loginPersona(t, handler, fx.TeacherStaffID)

	// 1. Default GET /api/system/ui-config returns base_font_size="medium" and theme_mode="light"
	recDefaultCfg := executeRequest(handler, http.MethodGet, "/api/system/ui-config", nil, teacherCookie)
	defaultCfg := decodeJSON[uiConfigResponse](t, recDefaultCfg)
	if recDefaultCfg.Code != http.StatusOK || defaultCfg.UserSettings.BaseFontSize != "medium" || defaultCfg.UserSettings.ThemeMode != "light" {
		t.Fatalf("expected default user_settings (medium, light), got code=%d, %+v", recDefaultCfg.Code, defaultCfg.UserSettings)
	}

	// 2. Cross-staff spoofing on PUT /api/system/user-settings fails fast with 403 FORBIDDEN_STAFF_ID_MISMATCH
	recSpoof := executeRequest(handler, http.MethodPut, "/api/system/user-settings", map[string]any{"staff_id": fx.HoDStaffID, "base_font_size": "large"}, teacherCookie)
	if recSpoof.Code != http.StatusForbidden || !strings.Contains(recSpoof.Body.String(), "FORBIDDEN_STAFF_ID_MISMATCH") {
		t.Fatalf("expected 403 FORBIDDEN_STAFF_ID_MISMATCH, got %d: %s", recSpoof.Code, recSpoof.Body.String())
	}

	// 3. Invalid base_font_size fails fast with 400 INVALID_BASE_FONT_SIZE
	recInvalid := executeRequest(handler, http.MethodPut, "/api/system/user-settings", map[string]any{"base_font_size": "24px"}, teacherCookie)
	if recInvalid.Code != http.StatusBadRequest || !strings.Contains(recInvalid.Body.String(), "INVALID_BASE_FONT_SIZE") {
		t.Fatalf("expected 400 INVALID_BASE_FONT_SIZE, got %d: %s", recInvalid.Code, recInvalid.Body.String())
	}

	// 4. Pin route and update user-settings -> persist in staff_settings without overwriting
	recPin := postPinnedRecent(handler, teacherCookie, map[string]any{"action": "pin", "route": "/p/" + testPlug.PlugID, "title": "Test Workspace"})
	recUpdate := executeRequest(handler, http.MethodPut, "/api/system/user-settings", map[string]any{"base_font_size": "extra-large", "theme_mode": "high-contrast"}, teacherCookie)
	if recPin.Code != http.StatusOK || recUpdate.Code != http.StatusOK {
		t.Fatalf("pin/update failed: pin=%d, update=%d", recPin.Code, recUpdate.Code)
	}

	// Verify row directly in core.db staff_settings table
	var dbFontSize, dbThemeMode, dbPinnedJSON string
	if err := h.DAL().ReadDB().QueryRowContext(ctx, `SELECT base_font_size, theme_mode, pinned_recents_json FROM staff_settings WHERE staff_id = ?`, fx.TeacherStaffID).Scan(&dbFontSize, &dbThemeMode, &dbPinnedJSON); err != nil {
		t.Fatalf("expected row in staff_settings for %s: %v", fx.TeacherStaffID, err)
	}
	if dbFontSize != "extra-large" || dbThemeMode != "high-contrast" || !strings.Contains(dbPinnedJSON, testPlug.PlugID) {
		t.Fatalf("unexpected staff_settings row: (%s, %s, %s)", dbFontSize, dbThemeMode, dbPinnedJSON)
	}
}
