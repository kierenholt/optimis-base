package host

import (
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
