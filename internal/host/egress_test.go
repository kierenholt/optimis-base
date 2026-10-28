package host_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"optimis/internal/host"
)

func setupEgressTestHost(t *testing.T, manifestJSON string, luaSource string) *host.Host {
	t.Helper()

	tempDir := t.TempDir()
	pluginsDir := filepath.Join(tempDir, "plugins")
	pluginDir := filepath.Join(pluginsDir, "egress_plugin")
	if err := os.MkdirAll(filepath.Join(pluginDir, "src"), 0o755); err != nil {
		t.Fatalf("failed to create plugin src dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(pluginDir, "migrations"), 0o755); err != nil {
		t.Fatalf("failed to create plugin migrations dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(manifestJSON), 0o644); err != nil {
		t.Fatalf("failed to write manifest.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "src", "init.lua"), []byte(luaSource), 0o644); err != nil {
		t.Fatalf("failed to write init.lua: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(pluginDir, "migrations", "001_init.sql"),
		[]byte(`CREATE TABLE IF NOT EXISTS sync_outbox (id TEXT PRIMARY KEY, pupil_id TEXT NOT NULL, status TEXT NOT NULL);`),
		0o644,
	); err != nil {
		t.Fatalf("failed to write 001_init.sql: %v", err)
	}

	h, err := host.NewHost(host.Config{
		DataDirPath:      filepath.Join(tempDir, "data"),
		SchoolURN:        "100001",
		HostKMSKeyHandle: "kms-egress-test-key-001",
		PluginsDir:       pluginsDir,
	})
	if err != nil {
		t.Fatalf("failed to create host: %v", err)
	}

	if err := h.LoadPlugin("egress_plugin"); err != nil {
		_ = h.Close()
		t.Fatalf("failed to load egress_plugin: %v", err)
	}

	logCursor := h.LogCursor()
	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("--- TAPPED SERVICE LOGS ---\n%s", strings.Join(h.LogsSince(logCursor), "\n"))
			t.Logf("--- TAPPED DIAGNOSTIC RING BUFFER ---\n%s", strings.Join(h.Diagnostics(), "\n"))
		}
		_ = h.Close()
	})

	return h
}

func TestExecuteRPC_RealHttpClientEgressModeEnabled_ExecutesGetPostAndFollowsApprovedRedirects_test(t *testing.T) {
	ctx := context.Background()

	var receivedAuthHeader string
	var receivedPostBody string

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/redirect-start", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/v1/redirect-target", http.StatusFound)
	})
	mux.HandleFunc("/v1/redirect-target", func(w http.ResponseWriter, r *http.Request) {
		receivedAuthHeader = r.Header.Get("X-Vendor-Token")
		w.Header().Set("X-Upstream-Node", "sparx-edge-1")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"synced":true,"hop":"approved"}`))
	})
	mux.HandleFunc("/v1/push", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		b, _ := io.ReadAll(r.Body)
		receivedPostBody = string(b)
		w.Header().Set("X-Receipt-ID", "rcpt_live_77")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"accepted":true}`))
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	manifest := fmt.Sprintf(`{
  "id": "uk.org.optimis.egress.real",
  "version": "0.1.0",
  "core_dal_version": "~1.0",
  "egress": {
    "allowed_domains": [%q],
    "timeout_ms": 5000
  },
  "endpoints": {
    "run_get_and_post": {
      "scopes": [],
      "mutates": false,
      "timeout_ms": 5000
    }
  }
}`, srv.URL)

	luaSrc := `
function run_get_and_post(ctx, args)
  local get_res = ctx.http.get(args.base_url .. "/v1/redirect-start", {
    ["X-Vendor-Token"] = "secret_tok_123"
  })
  local post_res = ctx.http.post(
    args.base_url .. "/v1/push",
    { ["Content-Type"] = "application/json" },
    args.post_body
  )
  return {
    ok = true,
    data = {
      get_status = get_res.status,
      get_body = get_res.body,
      get_final_url = get_res.final_url,
      get_header_node = get_res.headers["x-upstream-node"],
      post_status = post_res.status,
      post_body = post_res.body,
      post_header_receipt = post_res.headers["x-receipt-id"]
    }
  }
end
`
	h := setupEgressTestHost(t, manifest, luaSrc)
	h.ConfigureEgressMode(true)

	caller := host.CallerContext{
		Type:      "USER",
		UserID:    "usr_teacher_1",
		StaffID:   "stf_001",
		Scopes:    []string{},
		RequestID: "req_real_egress_1",
	}

	res, err := h.ExecuteRPC(ctx, "uk.org.optimis.egress.real", "run_get_and_post", caller, map[string]any{
		"base_url":  srv.URL,
		"post_body": `{"cohort":"10Ma1"}`,
	})
	if err != nil {
		t.Fatalf("ExecuteRPC failed: %v", err)
	}
	if !res.OK {
		t.Fatalf("expected res.OK == true, got error_code=%s", res.ErrorCode)
	}

	data := res.Data.(map[string]any)
	if data["get_status"] != float64(200) {
		t.Errorf("expected get_status 200, got %v", data["get_status"])
	}
	if data["get_body"] != `{"synced":true,"hop":"approved"}` {
		t.Errorf("unexpected get_body: %v", data["get_body"])
	}
	if data["get_final_url"] != srv.URL+"/v1/redirect-target" {
		t.Errorf("expected get_final_url %s, got %v", srv.URL+"/v1/redirect-target", data["get_final_url"])
	}
	if data["get_header_node"] != "sparx-edge-1" {
		t.Errorf("expected x-upstream-node header 'sparx-edge-1', got %v", data["get_header_node"])
	}
	if receivedAuthHeader != "secret_tok_123" {
		t.Errorf("expected server to receive X-Vendor-Token header 'secret_tok_123', got %q", receivedAuthHeader)
	}
	if data["post_status"] != float64(201) {
		t.Errorf("expected post_status 201, got %v", data["post_status"])
	}
	if receivedPostBody != `{"cohort":"10Ma1"}` {
		t.Errorf("expected server to receive POST body '{\"cohort\":\"10Ma1\"}', got %q", receivedPostBody)
	}
	if data["post_header_receipt"] != "rcpt_live_77" {
		t.Errorf("expected x-receipt-id header 'rcpt_live_77', got %v", data["post_header_receipt"])
	}
}

func TestExecuteRPC_RealHttpClientRedirectToUnapprovedDomain_RejectsRedirectViaCheckRedirectAndFailsFast_test(t *testing.T) {
	ctx := context.Background()

	var unapprovedHits atomic.Int32
	unapprovedSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		unapprovedHits.Add(1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"stolen":true}`))
	}))
	t.Cleanup(unapprovedSrv.Close)

	approvedSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, unapprovedSrv.URL+"/exfiltrate", http.StatusFound)
	}))
	t.Cleanup(approvedSrv.Close)

	manifest := fmt.Sprintf(`{
  "id": "uk.org.optimis.egress.redirect",
  "version": "0.1.0",
  "core_dal_version": "~1.0",
  "egress": {
    "allowed_domains": [%q]
  },
  "endpoints": {
    "attempt_cross_domain_redirect": {
      "scopes": [],
      "mutates": false
    }
  }
}`, approvedSrv.URL)

	luaSrc := `
function attempt_cross_domain_redirect(ctx, args)
  local res = ctx.http.get(args.url)
  return { ok = true, data = res }
end
`
	h := setupEgressTestHost(t, manifest, luaSrc)
	h.ConfigureEgressMode(true)

	caller := host.CallerContext{
		Type:      "USER",
		UserID:    "usr_teacher_1",
		StaffID:   "stf_001",
		Scopes:    []string{},
		RequestID: "req_redirect_guard",
	}

	_, err := h.ExecuteRPC(ctx, "uk.org.optimis.egress.redirect", "attempt_cross_domain_redirect", caller, map[string]any{
		"url": approvedSrv.URL + "/start",
	})
	if err == nil {
		t.Fatalf("expected fail-fast error when CheckRedirect encounters redirect to unapproved domain %s, got nil", unapprovedSrv.URL)
	}
	if !strings.Contains(err.Error(), "egress redirect target domain not permitted by manifest.json") {
		t.Errorf("expected redirect domain rejection message, got: %v", err)
	}
	if unapprovedHits.Load() != 0 {
		t.Fatalf("SECURITY VIOLATION: unapproved redirect target server was contacted (%d hits)", unapprovedHits.Load())
	}
}

func TestExecuteRPC_EgressTimeoutMsAndDefault8000msFallback_EnforcesClientTimeoutAndAbortsSlowRequest_test(t *testing.T) {
	ctx := context.Background()

	slowSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(600 * time.Millisecond):
			w.WriteHeader(http.StatusOK)
		case <-r.Context().Done():
			return
		}
	}))
	t.Cleanup(slowSrv.Close)

	manifest := fmt.Sprintf(`{
  "id": "uk.org.optimis.egress.timeout",
  "version": "0.1.0",
  "core_dal_version": "~1.0",
  "egress": {
    "allowed_domains": [%q],
    "timeout_ms": 100
  },
  "endpoints": {
    "fetch_slow": {
      "scopes": [],
      "mutates": false,
      "timeout_ms": 2500
    }
  }
}`, slowSrv.URL)

	luaSrc := `
function fetch_slow(ctx, args)
  local res = ctx.http.get(args.url)
  return { ok = true, data = res }
end
`
	h := setupEgressTestHost(t, manifest, luaSrc)
	h.ConfigureEgressMode(true)

	if gotFallback := host.ResolveEgressTimeoutMS(0); gotFallback != 8000 {
		t.Fatalf("expected fallback egress.timeout_ms to be 8000ms when omitted/0, got %dms", gotFallback)
	}

	caller := host.CallerContext{
		Type:      "USER",
		UserID:    "usr_teacher_1",
		StaffID:   "stf_001",
		Scopes:    []string{},
		RequestID: "req_timeout_1",
	}

	start := time.Now()
	_, err := h.ExecuteRPC(ctx, "uk.org.optimis.egress.timeout", "fetch_slow", caller, map[string]any{
		"url": slowSrv.URL + "/slow",
	})
	elapsed := time.Since(start)
	if err == nil {
		t.Fatalf("expected fail-fast timeout error when upstream exceeds egress.timeout_ms (100ms), got nil")
	}
	if elapsed > 1500*time.Millisecond {
		t.Errorf("expected request to abort near 100ms egress.timeout_ms, took %v", elapsed)
	}
}

func TestExecuteRPC_MockEgressEngineMode_EnforcesOpenWriteTxHaltAndUnconfiguredUrlFailFast_test(t *testing.T) {
	ctx := context.Background()

	manifest := `{
  "id": "uk.org.optimis.egress.mock",
  "version": "0.1.0",
  "core_dal_version": "~1.0",
  "database": {
    "alias": "plugin_egress_mock"
  },
  "gdpr_hooks": {
    "export_pupil_data": "export_pupil_data",
    "purge_pupil_data": "purge_pupil_data"
  },
  "egress": {
    "allowed_domains": ["https://sync.sparxmaths.uk"]
  },
  "endpoints": {
    "fetch_mock": {
      "scopes": [],
      "mutates": false
    },
    "write_then_post": {
      "scopes": [],
      "mutates": true
    },
    "count_outbox": {
      "scopes": [],
      "mutates": false
    }
  }
}`
	luaSrc := `
function fetch_mock(ctx, args)
  local res = ctx.http.get(args.url)
  return { ok = true, data = res }
end

function write_then_post(ctx, args)
  ctx.plugin_db.exec("INSERT INTO sync_outbox (id, pupil_id, status) VALUES ('ob_1', 'pup_1', 'PENDING')")
  ctx.http.post("https://sync.sparxmaths.uk/v1/push", { ["Content-Type"] = "application/json" }, '{"id":"ob_1"}')
  return { ok = true, data = {} }
end

function count_outbox(ctx, args)
  local rows = ctx.plugin_db.query("SELECT COUNT(*) AS cnt FROM sync_outbox")
  return { ok = true, data = { cnt = rows[1].cnt } }
end

function export_pupil_data(ctx, args)
  return { ok = true, data = {} }
end

function purge_pupil_data(ctx, args)
  return { ok = true, data = { purged_rows = 0 } }
end
`
	h := setupEgressTestHost(t, manifest, luaSrc)
	h.ConfigureEgressMode(false)
	h.ClearMockEgressRules()

	caller := host.CallerContext{
		Type:      "USER",
		UserID:    "usr_teacher_1",
		StaffID:   "stf_001",
		Scopes:    []string{},
		RequestID: "req_mock_egress",
	}

	// 1. Calling an allowed domain URL in mock mode without a registered mock rule MUST fail fast (no silent 200 stub)
	_, err := h.ExecuteRPC(ctx, "uk.org.optimis.egress.mock", "fetch_mock", caller, map[string]any{
		"url": "https://sync.sparxmaths.uk/v1/unconfigured",
	})
	if err == nil || !strings.Contains(err.Error(), "MOCK_EGRESS_UNCONFIGURED_URL") {
		t.Fatalf("expected fail-fast MOCK_EGRESS_UNCONFIGURED_URL when no mock rule is registered, got: %v", err)
	}

	// 2. Register a mock rule and verify round-trip
	h.RegisterMockEgressRule(host.MockEgressRule{
		Method: "GET",
		URL:    "https://sync.sparxmaths.uk/v1/status",
		Status: 200,
		Headers: map[string]string{
			"X-Mock-Source": "local-testbed",
		},
		Body: `{"healthy":true}`,
	})

	okRes, err := h.ExecuteRPC(ctx, "uk.org.optimis.egress.mock", "fetch_mock", caller, map[string]any{
		"url": "https://sync.sparxmaths.uk/v1/status",
	})
	if err != nil {
		t.Fatalf("fetch_mock with registered rule failed: %v", err)
	}
	okData, _ := okRes.Data.(map[string]any)
	if okData["body"] != `{"healthy":true}` {
		t.Errorf("expected response body '{\"healthy\":true}', got %v", okData["body"])
	}

	// 3. Calling ctx.http.post inside an open SQLite write transaction must halt/rollback WriteDB tx and fail fast
	_, err = h.ExecuteRPC(ctx, "uk.org.optimis.egress.mock", "write_then_post", caller, map[string]any{})
	if err == nil || !strings.Contains(err.Error(), "disallowed inside an open SQLite write transaction") {
		t.Fatalf("expected fail-fast open-write-transaction halt error on ctx.http.post, got: %v", err)
	}

	cntRes, err := h.ExecuteRPC(ctx, "uk.org.optimis.egress.mock", "count_outbox", caller, map[string]any{})
	if err != nil {
		t.Fatalf("count_outbox failed: %v", err)
	}
	if cntRes.Data.(map[string]any)["cnt"] != float64(0) {
		t.Fatalf("expected uncommitted row in sync_outbox to be rolled back when ctx.http.post halted write transaction, got %v", cntRes.Data)
	}
}
