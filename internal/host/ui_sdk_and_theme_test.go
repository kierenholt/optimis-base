package host

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServePluginUISDKAssets_MisBridgeJsAndMisThemeCss_ReturnsAssetsWithConnectSrcNoneCSP_test(t *testing.T) {
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
	defer h.Close()

	if err := h.LoadPlugin("test_plugin"); err != nil {
		t.Fatalf("LoadPlugin failed: %v", err)
	}
	testPlugin, ok := h.GetLoadedPlugin("uk.org.optimis.test")
	if !ok || testPlugin.PlugID == "" {
		t.Fatalf("expected uk.org.optimis.test to be loaded with non-empty PlugID")
	}

	handler, err := h.NewHTTPHandler()
	if err != nil {
		t.Fatalf("NewHTTPHandler failed: %v", err)
	}

	paths := []struct {
		url         string
		contentType string
		mustContain string
	}{
		{
			url:         "/sdk/mis-bridge.js",
			contentType: "application/javascript",
			mustContain: "window.mis",
		},
		{
			url:         "/sdk/mis-theme.css",
			contentType: "text/css",
			mustContain: "--mis-brand-primary",
		},
		{
			url:         "/p/" + testPlugin.PlugID + "/ui/mis-bridge.js",
			contentType: "application/javascript",
			mustContain: "mis:rpc:request",
		},
		{
			url:         "/p/" + testPlugin.PlugID + "/ui/mis-theme.css",
			contentType: "text/css",
			mustContain: "high-contrast",
		},
	}

	for _, tc := range paths {
		req := httptest.NewRequest(http.MethodGet, tc.url, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s expected 200 OK, got %d: %s", tc.url, rec.Code, rec.Body.String())
		}
		ct := rec.Header().Get("Content-Type")
		if !strings.Contains(ct, tc.contentType) {
			t.Fatalf("GET %s expected Content-Type containing %q, got %q", tc.url, tc.contentType, ct)
		}
		csp := rec.Header().Get("Content-Security-Policy")
		if !strings.Contains(csp, "connect-src 'none'") {
			t.Fatalf("GET %s expected CSP with connect-src 'none', got %q", tc.url, csp)
		}
		if !strings.Contains(rec.Body.String(), tc.mustContain) {
			t.Fatalf("GET %s expected body to contain %q", tc.url, tc.mustContain)
		}
	}
}

func TestGetSystemUIConfig_DefaultAndSeededSystemConfig_ReturnsSchoolThemeTokensPluginsAndEntityLinkHandlers_test(t *testing.T) {
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
	defer h.Close()

	if err := h.LoadPlugin("test_plugin"); err != nil {
		t.Fatalf("LoadPlugin failed: %v", err)
	}
	testPlugin, ok := h.GetLoadedPlugin("uk.org.optimis.test")
	if !ok || testPlugin.PlugID == "" {
		t.Fatalf("expected uk.org.optimis.test to be loaded with non-empty PlugID")
	}

	handler, err := h.NewHTTPHandler()
	if err != nil {
		t.Fatalf("NewHTTPHandler failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/system/ui-config", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/system/ui-config expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "uk.org.optimis.test") {
		t.Fatalf("GET /api/system/ui-config must not leak global reverse-DNS manifest ID uk.org.optimis.test: %s", rec.Body.String())
	}

	var payload struct {
		OK                 bool                         `json:"ok"`
		SchoolURN          string                       `json:"school_urn"`
		SchoolName         string                       `json:"school_name"`
		AcademicYearID     string                       `json:"academic_year_id"`
		Themes             map[string]map[string]string `json:"themes"`
		EntityLinkHandlers map[string]string            `json:"entity_link_handlers"`
		Plugins            []struct {
			ID               string   `json:"id"`
			PlugID           string   `json:"plug_id"`
			Title            string   `json:"title"`
			EntryURL         string   `json:"entry_url"`
			PermittedMethods []string `json:"permitted_methods"`
		} `json:"plugins"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to unmarshal ui-config response: %v", err)
	}

	if !payload.OK {
		t.Fatalf("expected payload.OK to be true")
	}
	if payload.SchoolURN != "100001" {
		t.Fatalf("expected school_urn 100001, got %q", payload.SchoolURN)
	}
	if payload.Themes["light"]["--mis-brand-primary"] != "#1d4ed8" {
		t.Fatalf("expected light --mis-brand-primary #1d4ed8, got %q", payload.Themes["light"]["--mis-brand-primary"])
	}
	if payload.Themes["high-contrast"]["--mis-brand-primary"] != "#facc15" {
		t.Fatalf("expected high-contrast --mis-brand-primary #facc15, got %q", payload.Themes["high-contrast"]["--mis-brand-primary"])
	}
	if payload.EntityLinkHandlers["student"] != testPlugin.PlugID {
		t.Fatalf("expected student entity handler %s, got %q", testPlugin.PlugID, payload.EntityLinkHandlers["student"])
	}
	if len(payload.Plugins) == 0 || payload.Plugins[0].ID != testPlugin.PlugID || payload.Plugins[0].PlugID != testPlugin.PlugID {
		t.Fatalf("expected active plugin %s in ui-config plugins list, got %+v", testPlugin.PlugID, payload.Plugins)
	}
	if payload.Plugins[0].EntryURL != "/p/"+testPlugin.PlugID+"/ui/index.html" {
		t.Fatalf("expected entry_url /p/%s/ui/index.html, got %q", testPlugin.PlugID, payload.Plugins[0].EntryURL)
	}
}

func TestGetSystemUIConfig_CallerWithRestrictedPersonaScopes_FiltersPermittedMethodsAndHidesUnauthorizedPlugins_test(t *testing.T) {
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
		t.Fatalf("LoadPlugin failed: %v", err)
	}
	testPlugin, ok := h.GetLoadedPlugin("uk.org.optimis.test")
	if !ok || testPlugin.PlugID == "" {
		t.Fatalf("expected uk.org.optimis.test to be loaded with non-empty PlugID")
	}

	// Seed two restricted staff personas in core.db:
	// 1) stf_postbox_only holding ONLY ["dal.safeguarding.create"]
	// 2) stf_no_plugin_scopes holding ONLY ["dal.sen.read:own_groups"] (matches zero endpoints in uk.org.optimis.test)
	writeDB := h.DAL().WriteDB()
	seedSQL := []string{
		`INSERT INTO roles (id, category, name, name_short, description, default_scopes_json)
		 VALUES ('ROLE_POSTBOX_ONLY', 'STAFF', 'Safeguarding Postbox Only', 'PBO', 'Write-only safeguarding postbox role', '["dal.safeguarding.create"]')`,
		`INSERT INTO roles (id, category, name, name_short, description, default_scopes_json)
		 VALUES ('ROLE_NO_PLUGIN_SCOPES', 'STAFF', 'Unmatched Scope Role', 'UMS', 'Holds scope not used by test_plugin', '["dal.sen.read:own_groups"]')`,
		`INSERT INTO persons (id, external_sso_sub, title, legal_first_name, legal_last_name, preferred_first_name, preferred_last_name, sex, status)
		 VALUES ('per_postbox_only', 'sso_postbox_only', 'Ms', 'Clara', 'Oswald', 'Clara', 'Oswald', 'F', 'ACTIVE')`,
		`INSERT INTO person_roles (person_id, role_id, is_primary, effective_from)
		 VALUES ('per_postbox_only', 'ROLE_POSTBOX_ONLY', 1, '2026-09-01')`,
		`INSERT INTO staff (id, person_id, staff_code, staff_type, job_title, qts_flag, is_dsl, hire_date, termination_date)
		 VALUES ('stf_postbox_only', 'per_postbox_only', 'PBO01', 'SUPPORT', 'Pastoral Assistant', 0, 0, '2026-09-01', NULL)`,
		`INSERT INTO persons (id, external_sso_sub, title, legal_first_name, legal_last_name, preferred_first_name, preferred_last_name, sex, status)
		 VALUES ('per_no_plugin_scopes', 'sso_no_plugin_scopes', 'Mr', 'Rory', 'Williams', 'Rory', 'Williams', 'M', 'ACTIVE')`,
		`INSERT INTO person_roles (person_id, role_id, is_primary, effective_from)
		 VALUES ('per_no_plugin_scopes', 'ROLE_NO_PLUGIN_SCOPES', 1, '2026-09-01')`,
		`INSERT INTO staff (id, person_id, staff_code, staff_type, job_title, qts_flag, is_dsl, hire_date, termination_date)
		 VALUES ('stf_no_plugin_scopes', 'per_no_plugin_scopes', 'UMS01', 'SUPPORT', 'SEN Support Assistant', 0, 0, '2026-09-01', NULL)`,
	}
	for _, stmt := range seedSQL {
		if _, err := writeDB.ExecContext(ctx, stmt); err != nil {
			t.Fatalf("failed to seed restricted persona SQL (%s): %v", stmt, err)
		}
	}
	t.Cleanup(func() {
		cleanupSQL := []string{
			`DELETE FROM staff WHERE id IN ('stf_postbox_only', 'stf_no_plugin_scopes')`,
			`DELETE FROM person_roles WHERE person_id IN ('per_postbox_only', 'per_no_plugin_scopes')`,
			`DELETE FROM persons WHERE id IN ('per_postbox_only', 'per_no_plugin_scopes')`,
			`DELETE FROM roles WHERE id IN ('ROLE_POSTBOX_ONLY', 'ROLE_NO_PLUGIN_SCOPES')`,
		}
		for _, stmt := range cleanupSQL {
			_, _ = writeDB.ExecContext(context.Background(), stmt)
		}
		h.ResetDevPersonaSessions()
	})

	handler, err := h.NewHTTPHandler()
	if err != nil {
		t.Fatalf("NewHTTPHandler failed: %v", err)
	}

	// Case 1: Switch to stf_postbox_only (only holds dal.safeguarding.create)
	switchBody1, _ := json.Marshal(map[string]any{"staff_id": "stf_postbox_only"})
	reqSwitch1 := httptest.NewRequest(http.MethodPost, "/api/auth/persona", bytes.NewReader(switchBody1))
	recSwitch1 := httptest.NewRecorder()
	handler.ServeHTTP(recSwitch1, reqSwitch1)
	if recSwitch1.Code != http.StatusOK {
		t.Fatalf("POST /api/auth/persona (stf_postbox_only) expected 200 OK, got %d: %s", recSwitch1.Code, recSwitch1.Body.String())
	}
	cookies1 := recSwitch1.Result().Cookies()
	if len(cookies1) == 0 {
		t.Fatalf("expected mis_session cookie from POST /api/auth/persona")
	}

	reqConfig1 := httptest.NewRequest(http.MethodGet, "/api/system/ui-config", nil)
	reqConfig1.AddCookie(cookies1[0])
	recConfig1 := httptest.NewRecorder()
	handler.ServeHTTP(recConfig1, reqConfig1)
	if recConfig1.Code != http.StatusOK {
		t.Fatalf("GET /api/system/ui-config expected 200 OK, got %d: %s", recConfig1.Code, recConfig1.Body.String())
	}

	var payload1 struct {
		OK      bool `json:"ok"`
		Plugins []struct {
			ID               string   `json:"id"`
			PermittedMethods []string `json:"permitted_methods"`
		} `json:"plugins"`
	}
	if err := json.Unmarshal(recConfig1.Body.Bytes(), &payload1); err != nil {
		t.Fatalf("failed to unmarshal ui-config response: %v", err)
	}
	if len(payload1.Plugins) != 1 || payload1.Plugins[0].ID != testPlugin.PlugID {
		t.Fatalf("expected 1 plugin (%s) for stf_postbox_only, got %+v", testPlugin.PlugID, payload1.Plugins)
	}
	if len(payload1.Plugins[0].PermittedMethods) != 1 || payload1.Plugins[0].PermittedMethods[0] != "submit_safeguarding_postbox" {
		t.Fatalf("expected PermittedMethods to be strictly [submit_safeguarding_postbox], got %v", payload1.Plugins[0].PermittedMethods)
	}

	// Case 2: Switch to stf_no_plugin_scopes (matches zero endpoints in uk.org.optimis.test) -> plugin must be omitted from navigation list
	switchBody2, _ := json.Marshal(map[string]any{"staff_id": "stf_no_plugin_scopes"})
	reqSwitch2 := httptest.NewRequest(http.MethodPost, "/api/auth/persona", bytes.NewReader(switchBody2))
	recSwitch2 := httptest.NewRecorder()
	handler.ServeHTTP(recSwitch2, reqSwitch2)
	if recSwitch2.Code != http.StatusOK {
		t.Fatalf("POST /api/auth/persona (stf_no_plugin_scopes) expected 200 OK, got %d: %s", recSwitch2.Code, recSwitch2.Body.String())
	}
	cookies2 := recSwitch2.Result().Cookies()
	if len(cookies2) == 0 {
		t.Fatalf("expected mis_session cookie from POST /api/auth/persona")
	}

	reqConfig2 := httptest.NewRequest(http.MethodGet, "/api/system/ui-config", nil)
	reqConfig2.AddCookie(cookies2[0])
	recConfig2 := httptest.NewRecorder()
	handler.ServeHTTP(recConfig2, reqConfig2)
	if recConfig2.Code != http.StatusOK {
		t.Fatalf("GET /api/system/ui-config expected 200 OK, got %d: %s", recConfig2.Code, recConfig2.Body.String())
	}

	var payload2 struct {
		OK      bool `json:"ok"`
		Plugins []struct {
			ID               string   `json:"id"`
			PermittedMethods []string `json:"permitted_methods"`
		} `json:"plugins"`
	}
	if err := json.Unmarshal(recConfig2.Body.Bytes(), &payload2); err != nil {
		t.Fatalf("failed to unmarshal ui-config response: %v", err)
	}
	if len(payload2.Plugins) != 0 {
		t.Fatalf("expected 0 plugins when caller holds zero matching endpoint scopes, got %+v", payload2.Plugins)
	}
}

func TestManifestWidgetsAndSeedLruPlugins_ValidAndInvalidWidgetDeclarations_ExposesWidgetsInUIConfigAndFailsFastOnInvalidEntry_test(t *testing.T) {
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
		t.Fatalf("LoadPlugin failed: %v", err)
	}
	testPlugin, ok := h.GetLoadedPlugin("uk.org.optimis.test")
	if !ok || testPlugin.PlugID == "" {
		t.Fatalf("expected uk.org.optimis.test to be loaded with non-empty PlugID")
	}

	handler, err := h.NewHTTPHandler()
	if err != nil {
		t.Fatalf("NewHTTPHandler failed: %v", err)
	}

	// 1. Verify GET /p/{plug_id}/ui/widget.html serves widget HTML with connect-src 'none' CSP
	reqWidget := httptest.NewRequest(http.MethodGet, "/p/"+testPlugin.PlugID+"/ui/widget.html", nil)
	recWidget := httptest.NewRecorder()
	handler.ServeHTTP(recWidget, reqWidget)
	if recWidget.Code != http.StatusOK {
		t.Fatalf("GET /p/%s/ui/widget.html expected 200 OK, got %d: %s", testPlugin.PlugID, recWidget.Code, recWidget.Body.String())
	}
	if !strings.Contains(recWidget.Header().Get("Content-Security-Policy"), "connect-src 'none'") {
		t.Fatalf("expected connect-src 'none' CSP on widget.html, got %q", recWidget.Header().Get("Content-Security-Policy"))
	}

	// 2. Seed synthetic LRU plugins via POST /api/test/seed-lru-plugins and verify GET /api/system/ui-config
	seedBody, _ := json.Marshal(map[string]any{
		"plugin_ids": []string{"uk.org.optimis.lru1", "uk.org.optimis.lru2"},
	})
	reqSeed := httptest.NewRequest(http.MethodPost, "/api/test/seed-lru-plugins", bytes.NewReader(seedBody))
	recSeed := httptest.NewRecorder()
	handler.ServeHTTP(recSeed, reqSeed)
	if recSeed.Code != http.StatusCreated {
		t.Fatalf("POST /api/test/seed-lru-plugins expected 201 Created, got %d: %s", recSeed.Code, recSeed.Body.String())
	}

	reqConfig := httptest.NewRequest(http.MethodGet, "/api/system/ui-config", nil)
	recConfig := httptest.NewRecorder()
	handler.ServeHTTP(recConfig, reqConfig)
	if recConfig.Code != http.StatusOK {
		t.Fatalf("GET /api/system/ui-config expected 200 OK, got %d", recConfig.Code)
	}

	var configPayload struct {
		OK      bool `json:"ok"`
		Plugins []struct {
			ID      string `json:"id"`
			Widgets []struct {
				ID       string `json:"id"`
				Title    string `json:"title"`
				Entry    string `json:"entry"`
				EntryURL string `json:"entry_url"`
				Width    int    `json:"width"`
				Height   int    `json:"height"`
			} `json:"widgets"`
		} `json:"plugins"`
	}
	if err := json.Unmarshal(recConfig.Body.Bytes(), &configPayload); err != nil {
		t.Fatalf("failed to unmarshal ui-config response: %v", err)
	}
	if len(configPayload.Plugins) != 3 {
		t.Fatalf("expected 3 active plugins (test + lru1 + lru2), got %d", len(configPayload.Plugins))
	}

	foundTestPlugin := false
	expectedWidgetEntryURL := "/p/" + testPlugin.PlugID + "/ui/widget.html"
	for _, p := range configPayload.Plugins {
		if p.ID == testPlugin.PlugID {
			foundTestPlugin = true
			if len(p.Widgets) < 2 {
				t.Fatalf("expected at least 2 widgets on %s, got %+v", testPlugin.PlugID, p.Widgets)
			}
			if p.Widgets[0].EntryURL != expectedWidgetEntryURL {
				t.Fatalf("expected widget entry_url %s, got %q", expectedWidgetEntryURL, p.Widgets[0].EntryURL)
			}
			if p.Widgets[0].Width != 2 || p.Widgets[0].Height != 1 {
				t.Fatalf("expected attendance_pulse widget preset size 2x1, got %dx%d", p.Widgets[0].Width, p.Widgets[0].Height)
			}
			if p.Widgets[1].Width != 1 || p.Widgets[1].Height != 1 {
				t.Fatalf("expected pastoral_watch widget preset size 1x1, got %dx%d", p.Widgets[1].Width, p.Widgets[1].Height)
			}
		}
	}
	if !foundTestPlugin {
		t.Fatalf("expected %s in ui-config plugins", testPlugin.PlugID)
	}

	// 3. Verify readPluginFromDisk fails fast when a manifest widget omits width/height or specifies values outside {1, 2}
	invalidWidgetPluginDir := filepath.Join(tempDir, "invalid_widget_size_plugin")
	if err := os.MkdirAll(filepath.Join(invalidWidgetPluginDir, "src"), 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(invalidWidgetPluginDir, "src", "init.lua"), []byte(`function ping(ctx, args) return { ok = true } end`), 0o644); err != nil {
		t.Fatalf("WriteFile init.lua failed: %v", err)
	}
	invalidWidgetManifestJSON := `{
		"id": "uk.org.optimis.invalid.widget.size",
		"version": "1.0.0",
		"core_dal_version": "~1.0",
		"views": [
			{ "id": "home", "path": "/", "entry": "ui/dist/index.html" }
		],
		"widgets": [
			{ "id": "bad_size", "title": "Bad Size Widget", "entry": "ui/dist/widget.html", "width": 3, "height": 1 }
		],
		"endpoints": {
			"ping": { "scopes": ["dal.pupils.read:own_groups"], "mutates": false }
		}
	}`
	if err := os.WriteFile(filepath.Join(invalidWidgetPluginDir, "manifest.json"), []byte(invalidWidgetManifestJSON), 0o644); err != nil {
		t.Fatalf("WriteFile manifest.json failed: %v", err)
	}
	if _, errBadSize := readPluginFromDisk(invalidWidgetPluginDir, "invalid_widget_size_plugin"); errBadSize == nil || !strings.Contains(errBadSize.Error(), "width") {
		t.Fatalf("expected readPluginFromDisk to fail fast on widget width=3, got: %v", errBadSize)
	}

	// Clean up synthetic LRU plugins via DELETE /api/test/seed-lru-plugins
	reqDel := httptest.NewRequest(http.MethodDelete, "/api/test/seed-lru-plugins", nil)
	recDel := httptest.NewRecorder()
	handler.ServeHTTP(recDel, reqDel)
	if recDel.Code != http.StatusOK {
		t.Fatalf("DELETE /api/test/seed-lru-plugins expected 200 OK, got %d", recDel.Code)
	}
}

func TestOpaquePluginViewRouting_DeclaredAndUndeclaredViewsAndLegacyRoutes_EnforcesDefaultDenyAndServesHostSPA_test(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	pluginsDir, err := filepath.Abs("../../plugins")
	if err != nil {
		t.Fatalf("failed to resolve plugins directory: %v", err)
	}

	h, err := NewHost(Config{
		DataDirPath:      tempDir,
		SchoolURN:        "100001",
		HostKMSKeyHandle: "local-dev-kms-key-01",
		PluginsDir:       pluginsDir,
	})
	if err != nil {
		t.Fatalf("NewHost failed: %v", err)
	}
	defer h.Close()

	if err := h.LoadPlugin("senco_provision_plugin"); err != nil {
		t.Fatalf("LoadPlugin(senco_provision_plugin) failed: %v", err)
	}
	senPlugin, ok := h.GetLoadedPlugin("uk.org.optimis.sen.provision")
	if !ok || !opaquePlugIDPattern.MatchString(senPlugin.PlugID) {
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
	reloaded, ok := h.GetLoadedPlugin("uk.org.optimis.sen.provision")
	if !ok || reloaded.PlugID != initialPlugID {
		t.Fatalf("expected persisted PlugID %q across reinstall, got %q", initialPlugID, reloaded.PlugID)
	}

	handler, err := h.NewHTTPHandler()
	if err != nil {
		t.Fatalf("NewHTTPHandler failed: %v", err)
	}

	// 1. Declared routes GET /p/{plug_id} and GET /p/{plug_id}/student/4092?tab=iep return 200 OK
	for _, validURL := range []string{
		"/p/" + initialPlugID,
		"/p/" + initialPlugID + "/student/4092?tab=iep",
		"/p/" + initialPlugID + "/group/grp_101?tab=overview",
	} {
		req := httptest.NewRequest(http.MethodGet, validURL, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s expected 200 OK, got %d: %s", validURL, rec.Code, rec.Body.String())
		}
	}

	// 2. Undeclared view path fails fast with 404 UNDECLARED_PLUGIN_VIEW_PATH (JSON when no Accept: text/html)
	reqUndeclaredPath := httptest.NewRequest(http.MethodGet, "/p/"+initialPlugID+"/undeclared/subpath", nil)
	recUndeclaredPath := httptest.NewRecorder()
	handler.ServeHTTP(recUndeclaredPath, reqUndeclaredPath)
	if recUndeclaredPath.Code != http.StatusNotFound || !strings.Contains(recUndeclaredPath.Body.String(), "UNDECLARED_PLUGIN_VIEW_PATH") {
		t.Fatalf("expected 404 UNDECLARED_PLUGIN_VIEW_PATH, got %d: %s", recUndeclaredPath.Code, recUndeclaredPath.Body.String())
	}

	// 2b. Browser navigation (Accept: text/html) to undeclared view path returns 404 AND serves Host SPA index.html
	reqUndeclaredPathHTML := httptest.NewRequest(http.MethodGet, "/p/"+initialPlugID+"/undeclared/subpath", nil)
	reqUndeclaredPathHTML.Header.Set("Accept", "text/html,application/xhtml+xml")
	recUndeclaredPathHTML := httptest.NewRecorder()
	handler.ServeHTTP(recUndeclaredPathHTML, reqUndeclaredPathHTML)
	if recUndeclaredPathHTML.Code != http.StatusNotFound || !strings.Contains(strings.ToLower(recUndeclaredPathHTML.Body.String()), "<!doctype html>") {
		t.Fatalf("expected 404 with Host SPA HTML on Accept: text/html, got %d: %s", recUndeclaredPathHTML.Code, recUndeclaredPathHTML.Body.String())
	}

	// 3. Undeclared query param fails fast with 400 UNDECLARED_PLUGIN_VIEW_QUERY_PARAM (JSON when no Accept: text/html)
	reqUndeclaredQuery := httptest.NewRequest(http.MethodGet, "/p/"+initialPlugID+"/student/4092?tab=iep&bogus_param=1", nil)
	recUndeclaredQuery := httptest.NewRecorder()
	handler.ServeHTTP(recUndeclaredQuery, reqUndeclaredQuery)
	if recUndeclaredQuery.Code != http.StatusBadRequest || !strings.Contains(recUndeclaredQuery.Body.String(), "UNDECLARED_PLUGIN_VIEW_QUERY_PARAM") {
		t.Fatalf("expected 400 UNDECLARED_PLUGIN_VIEW_QUERY_PARAM, got %d: %s", recUndeclaredQuery.Code, recUndeclaredQuery.Body.String())
	}

	// 3b. Browser navigation (Accept: text/html) with undeclared query param returns 400 AND serves Host SPA index.html
	reqUndeclaredQueryHTML := httptest.NewRequest(http.MethodGet, "/p/"+initialPlugID+"/student/4092?tab=iep&bogus_param=1", nil)
	reqUndeclaredQueryHTML.Header.Set("Accept", "text/html")
	recUndeclaredQueryHTML := httptest.NewRecorder()
	handler.ServeHTTP(recUndeclaredQueryHTML, reqUndeclaredQueryHTML)
	if recUndeclaredQueryHTML.Code != http.StatusBadRequest || !strings.Contains(strings.ToLower(recUndeclaredQueryHTML.Body.String()), "<!doctype html>") {
		t.Fatalf("expected 400 with Host SPA HTML on Accept: text/html, got %d: %s", recUndeclaredQueryHTML.Code, recUndeclaredQueryHTML.Body.String())
	}

	// 4. Issue 1: Static UI assets under /p/{plug_id}/ui/... and /_p_assets/{plug_id}/... never collide with views[]
	for _, assetURL := range []string{
		"/p/" + initialPlugID + "/ui/index.html",
		"/_p_assets/" + initialPlugID + "/index.html",
	} {
		reqAsset := httptest.NewRequest(http.MethodGet, assetURL, nil)
		recAsset := httptest.NewRecorder()
		handler.ServeHTTP(recAsset, reqAsset)
		if recAsset.Code != http.StatusOK {
			t.Fatalf("GET %s expected 200 OK without colliding with views[], got %d: %s", assetURL, recAsset.Code, recAsset.Body.String())
		}
	}

	// 5. Issue 1: readPluginFromDisk rejects any views[].path starting with /ui (RESERVED_PLUGIN_VIEW_PATH)
	reservedPluginDir := filepath.Join(tempDir, "reserved_ui_plugin")
	if err := os.MkdirAll(filepath.Join(reservedPluginDir, "src"), 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(reservedPluginDir, "src", "init.lua"), []byte(`function ping(ctx, args) return { ok = true } end`), 0o644); err != nil {
		t.Fatalf("WriteFile init.lua failed: %v", err)
	}
	reservedManifestJSON := `{
		"id": "uk.org.optimis.reserved.ui",
		"version": "1.0.0",
		"core_dal_version": "~1.0",
		"views": [
			{ "id": "home", "path": "/", "entry": "ui/dist/index.html" },
			{ "id": "shadow_ui", "path": "/ui/custom", "entry": "ui/dist/index.html" }
		],
		"endpoints": {
			"ping": { "scopes": ["dal.pupils.read:own_groups"], "mutates": false }
		}
	}`
	if err := os.WriteFile(filepath.Join(reservedPluginDir, "manifest.json"), []byte(reservedManifestJSON), 0o644); err != nil {
		t.Fatalf("WriteFile manifest.json failed: %v", err)
	}
	_, errReserved := readPluginFromDisk(reservedPluginDir, "reserved_ui_plugin")
	if errReserved == nil || !strings.Contains(errReserved.Error(), "RESERVED_PLUGIN_VIEW_PATH") {
		t.Fatalf("expected RESERVED_PLUGIN_VIEW_PATH error for views[].path starting with /ui, got %v", errReserved)
	}

	// 6. Legacy /plugins/... and /api/plugins/... routes are removed (404)
	reqLegacyUI := httptest.NewRequest(http.MethodGet, "/plugins/uk.org.optimis.sen.provision/ui/index.html", nil)
	recLegacyUI := httptest.NewRecorder()
	handler.ServeHTTP(recLegacyUI, reqLegacyUI)
	if recLegacyUI.Code != http.StatusNotFound {
		t.Fatalf("expected 404 on legacy /plugins/... UI route, got %d", recLegacyUI.Code)
	}

	reqLegacyRPC := httptest.NewRequest(http.MethodPost, "/api/plugins/uk.org.optimis.sen.provision/rpc/get_sen_differentiation_summary", strings.NewReader(`{"args":{}}`))
	recLegacyRPC := httptest.NewRecorder()
	handler.ServeHTTP(recLegacyRPC, reqLegacyRPC)
	if recLegacyRPC.Code != http.StatusNotFound {
		t.Fatalf("expected 404 on legacy /api/plugins/... RPC route, got %d", recLegacyRPC.Code)
	}
}

func TestPinnedRecentsSecurityAndValidation_CrossStaffSpoofingUnvalidatedRoutesDormantPluginAndNoDefaultFallback_EnforcesAllInvariants_test(t *testing.T) {
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

	if err := h.LoadPlugin("senco_provision_plugin"); err != nil {
		t.Fatalf("LoadPlugin(senco_provision_plugin) failed: %v", err)
	}
	if err := h.LoadPlugin("dfe_census_plugin"); err != nil {
		t.Fatalf("LoadPlugin(dfe_census_plugin) failed: %v", err)
	}
	senPlug, ok := h.GetLoadedPlugin("uk.org.optimis.sen.provision")
	if !ok {
		t.Fatalf("expected uk.org.optimis.sen.provision to be loaded")
	}
	censusPlug, ok := h.GetLoadedPlugin("uk.gov.dfe.census")
	if !ok {
		t.Fatalf("expected uk.gov.dfe.census to be loaded")
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

	// Obtain a valid mis_session cookie for Teacher
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
		t.Fatalf("expected mis_session cookie for Teacher")
	}

	// 1. Cross-staff spoofing: Teacher session attempting to pin a route for HoD fails fast with 403 FORBIDDEN_STAFF_ID_MISMATCH
	spoofPinBody, _ := json.Marshal(map[string]any{
		"staff_id": fx.HoDStaffID,
		"action":   "pin",
		"route":    "/p/" + senPlug.PlugID + "/",
		"title":    "Spoofed Pin",
	})
	reqSpoofPin := httptest.NewRequest(http.MethodPost, "/api/system/pinned-recents", bytes.NewReader(spoofPinBody))
	reqSpoofPin.AddCookie(teacherCookie)
	recSpoofPin := httptest.NewRecorder()
	handler.ServeHTTP(recSpoofPin, reqSpoofPin)
	if recSpoofPin.Code != http.StatusForbidden || !strings.Contains(recSpoofPin.Body.String(), "FORBIDDEN_STAFF_ID_MISMATCH") {
		t.Fatalf("expected 403 FORBIDDEN_STAFF_ID_MISMATCH on cross-staff pin attempt, got %d: %s", recSpoofPin.Code, recSpoofPin.Body.String())
	}

	// 2. Unvalidated sub-routes & unpermitted plugins are rejected on pin:
	// 2a. Unknown plug_id -> 404 PLUGIN_NOT_FOUND
	unknownPlugBody, _ := json.Marshal(map[string]any{
		"staff_id": fx.TeacherStaffID,
		"action":   "pin",
		"route":    "/p/zzzz9999/",
		"title":    "Unknown Plugin",
	})
	reqUnknown := httptest.NewRequest(http.MethodPost, "/api/system/pinned-recents", bytes.NewReader(unknownPlugBody))
	reqUnknown.AddCookie(teacherCookie)
	recUnknown := httptest.NewRecorder()
	handler.ServeHTTP(recUnknown, reqUnknown)
	if recUnknown.Code != http.StatusNotFound || !strings.Contains(recUnknown.Body.String(), "PLUGIN_NOT_FOUND") {
		t.Fatalf("expected 404 PLUGIN_NOT_FOUND for unknown plug_id, got %d: %s", recUnknown.Code, recUnknown.Body.String())
	}

	// 2b. Undeclared sub-path -> 404 UNDECLARED_PLUGIN_VIEW_PATH
	undeclaredSubpathBody, _ := json.Marshal(map[string]any{
		"staff_id": fx.TeacherStaffID,
		"action":   "pin",
		"route":    "/p/" + senPlug.PlugID + "/undeclared/deep/route",
		"title":    "Undeclared Subpath",
	})
	reqUndeclaredPath := httptest.NewRequest(http.MethodPost, "/api/system/pinned-recents", bytes.NewReader(undeclaredSubpathBody))
	reqUndeclaredPath.AddCookie(teacherCookie)
	recUndeclaredPath := httptest.NewRecorder()
	handler.ServeHTTP(recUndeclaredPath, reqUndeclaredPath)
	if recUndeclaredPath.Code != http.StatusNotFound || !strings.Contains(recUndeclaredPath.Body.String(), "UNDECLARED_PLUGIN_VIEW_PATH") {
		t.Fatalf("expected 404 UNDECLARED_PLUGIN_VIEW_PATH for undeclared subpath, got %d: %s", recUndeclaredPath.Code, recUndeclaredPath.Body.String())
	}

	// 2c. Undeclared query param -> 400 UNDECLARED_PLUGIN_VIEW_QUERY_PARAM
	undeclaredQueryBody, _ := json.Marshal(map[string]any{
		"staff_id": fx.TeacherStaffID,
		"action":   "pin",
		"route":    "/p/" + senPlug.PlugID + "/?evil_param=1",
		"title":    "Undeclared Query",
	})
	reqUndeclaredQuery := httptest.NewRequest(http.MethodPost, "/api/system/pinned-recents", bytes.NewReader(undeclaredQueryBody))
	reqUndeclaredQuery.AddCookie(teacherCookie)
	recUndeclaredQuery := httptest.NewRecorder()
	handler.ServeHTTP(recUndeclaredQuery, reqUndeclaredQuery)
	if recUndeclaredQuery.Code != http.StatusBadRequest || !strings.Contains(recUndeclaredQuery.Body.String(), "UNDECLARED_PLUGIN_VIEW_QUERY_PARAM") {
		t.Fatalf("expected 400 UNDECLARED_PLUGIN_VIEW_QUERY_PARAM for undeclared query param, got %d: %s", recUndeclaredQuery.Code, recUndeclaredQuery.Body.String())
	}

	// 2d. Unpermitted plugin (DfE Census requires dal.census.read:all which Teacher lacks) -> 403 FORBIDDEN_PLUGIN_ACCESS
	unpermittedBody, _ := json.Marshal(map[string]any{
		"staff_id": fx.TeacherStaffID,
		"action":   "pin",
		"route":    "/p/" + censusPlug.PlugID + "/",
		"title":    "DfE Census",
	})
	reqUnpermitted := httptest.NewRequest(http.MethodPost, "/api/system/pinned-recents", bytes.NewReader(unpermittedBody))
	reqUnpermitted.AddCookie(teacherCookie)
	recUnpermitted := httptest.NewRecorder()
	handler.ServeHTTP(recUnpermitted, reqUnpermitted)
	if recUnpermitted.Code != http.StatusForbidden || !strings.Contains(recUnpermitted.Body.String(), "FORBIDDEN_PLUGIN_ACCESS") {
		t.Fatalf("expected 403 FORBIDDEN_PLUGIN_ACCESS when Teacher pins Admin-only census plugin, got %d: %s", recUnpermitted.Code, recUnpermitted.Body.String())
	}

	// 3. Valid pin for Teacher succeeds, and user_prefs.pinned_recents.default poisoning never occurs on unpin
	if _, err := h.DAL().SetSystemConfigValue(ctx, "user_prefs.pinned_recents.default", `[{"route":"/p/`+senPlug.PlugID+`/","plug_id":"`+senPlug.PlugID+`","title":"Poisoned Default Pin"}]`); err != nil {
		t.Fatalf("SetSystemConfigValue(user_prefs.pinned_recents.default) failed: %v", err)
	}

	validRoute := "/p/" + senPlug.PlugID + "/"
	validPinBody, _ := json.Marshal(map[string]any{
		"staff_id": fx.TeacherStaffID,
		"action":   "pin",
		"route":    validRoute,
		"title":    "SEN Provision Workspace",
	})
	reqValidPin := httptest.NewRequest(http.MethodPost, "/api/system/pinned-recents", bytes.NewReader(validPinBody))
	reqValidPin.AddCookie(teacherCookie)
	recValidPin := httptest.NewRecorder()
	handler.ServeHTTP(recValidPin, reqValidPin)
	if recValidPin.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on valid Teacher pin, got %d: %s", recValidPin.Code, recValidPin.Body.String())
	}

	// Unpin the last route -> must return 0 items and never fall back to user_prefs.pinned_recents.default
	unpinBody, _ := json.Marshal(map[string]any{
		"staff_id": fx.TeacherStaffID,
		"action":   "unpin",
		"route":    validRoute,
	})
	reqUnpin := httptest.NewRequest(http.MethodPost, "/api/system/pinned-recents", bytes.NewReader(unpinBody))
	reqUnpin.AddCookie(teacherCookie)
	recUnpin := httptest.NewRecorder()
	handler.ServeHTTP(recUnpin, reqUnpin)
	if recUnpin.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on unpin, got %d: %s", recUnpin.Code, recUnpin.Body.String())
	}
	var unpinResp struct {
		PinnedRecents []PinnedRecentItem `json:"pinned_recents"`
	}
	if err := json.Unmarshal(recUnpin.Body.Bytes(), &unpinResp); err != nil {
		t.Fatalf("failed to decode unpin response: %v", err)
	}
	if len(unpinResp.PinnedRecents) != 0 {
		t.Fatalf("expected 0 pinned_recents after unpinning last item (no .default fallback), got %+v", unpinResp.PinnedRecents)
	}

	// Re-pin validRoute, then transition sen_provision_plugin to UNINSTALLED_DORMANT -> loadPinnedRecentsForStaff filters out dormant plugin pins
	reqRePin := httptest.NewRequest(http.MethodPost, "/api/system/pinned-recents", bytes.NewReader(validPinBody))
	reqRePin.AddCookie(teacherCookie)
	recRePin := httptest.NewRecorder()
	handler.ServeHTTP(recRePin, reqRePin)
	if recRePin.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on re-pin, got %d", recRePin.Code)
	}

	if err := h.UninstallPlugin("uk.org.optimis.sen.provision"); err != nil {
		t.Fatalf("UninstallPlugin failed: %v", err)
	}

	reqUIConfig := httptest.NewRequest(http.MethodGet, "/api/system/ui-config", nil)
	reqUIConfig.AddCookie(teacherCookie)
	recUIConfig := httptest.NewRecorder()
	handler.ServeHTTP(recUIConfig, reqUIConfig)
	if recUIConfig.Code != http.StatusOK {
		t.Fatalf("GET /api/system/ui-config expected 200 OK, got %d", recUIConfig.Code)
	}
	var uiCfgResp struct {
		PinnedRecents []PinnedRecentItem `json:"pinned_recents"`
	}
	if err := json.Unmarshal(recUIConfig.Body.Bytes(), &uiCfgResp); err != nil {
		t.Fatalf("failed to decode ui-config response: %v", err)
	}
	if len(uiCfgResp.PinnedRecents) != 0 {
		t.Fatalf("expected dormant plugin pins to be filtered out from /api/system/ui-config, got %+v", uiCfgResp.PinnedRecents)
	}
}

