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
			url:         "/plugins/uk.org.optimis.test/ui/mis-bridge.js",
			contentType: "application/javascript",
			mustContain: "mis:rpc:request",
		},
		{
			url:         "/plugins/uk.org.optimis.test/ui/mis-theme.css",
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

	var payload struct {
		OK                 bool                         `json:"ok"`
		SchoolURN          string                       `json:"school_urn"`
		SchoolName         string                       `json:"school_name"`
		AcademicYearID     string                       `json:"academic_year_id"`
		Themes             map[string]map[string]string `json:"themes"`
		EntityLinkHandlers map[string]string            `json:"entity_link_handlers"`
		Plugins            []struct {
			ID               string   `json:"id"`
			Title            string   `json:"title"`
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
	if payload.EntityLinkHandlers["student"] != "uk.org.optimis.test" {
		t.Fatalf("expected student entity handler uk.org.optimis.test, got %q", payload.EntityLinkHandlers["student"])
	}
	if len(payload.Plugins) == 0 || payload.Plugins[0].ID != "uk.org.optimis.test" {
		t.Fatalf("expected active plugin uk.org.optimis.test in ui-config plugins list")
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
	if len(payload1.Plugins) != 1 || payload1.Plugins[0].ID != "uk.org.optimis.test" {
		t.Fatalf("expected 1 plugin (uk.org.optimis.test) for stf_postbox_only, got %+v", payload1.Plugins)
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

	handler, err := h.NewHTTPHandler()
	if err != nil {
		t.Fatalf("NewHTTPHandler failed: %v", err)
	}

	// 1. Verify GET /plugins/uk.org.optimis.test/ui/widget.html serves widget HTML with connect-src 'none' CSP
	reqWidget := httptest.NewRequest(http.MethodGet, "/plugins/uk.org.optimis.test/ui/widget.html", nil)
	recWidget := httptest.NewRecorder()
	handler.ServeHTTP(recWidget, reqWidget)
	if recWidget.Code != http.StatusOK {
		t.Fatalf("GET /plugins/uk.org.optimis.test/ui/widget.html expected 200 OK, got %d: %s", recWidget.Code, recWidget.Body.String())
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
	for _, p := range configPayload.Plugins {
		if p.ID == "uk.org.optimis.test" {
			foundTestPlugin = true
			if len(p.Widgets) < 2 {
				t.Fatalf("expected at least 2 widgets on uk.org.optimis.test, got %+v", p.Widgets)
			}
			if p.Widgets[0].EntryURL != "/plugins/uk.org.optimis.test/ui/widget.html" {
				t.Fatalf("expected widget entry_url /plugins/uk.org.optimis.test/ui/widget.html, got %q", p.Widgets[0].EntryURL)
			}
		}
	}
	if !foundTestPlugin {
		t.Fatalf("expected uk.org.optimis.test in ui-config plugins")
	}

	// Clean up synthetic LRU plugins via DELETE /api/test/seed-lru-plugins
	reqDel := httptest.NewRequest(http.MethodDelete, "/api/test/seed-lru-plugins", nil)
	recDel := httptest.NewRecorder()
	handler.ServeHTTP(recDel, reqDel)
	if recDel.Code != http.StatusOK {
		t.Fatalf("DELETE /api/test/seed-lru-plugins expected 200 OK, got %d", recDel.Code)
	}
}

