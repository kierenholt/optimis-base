package host_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"optimis/internal/dal"
	"optimis/internal/host"
)

func setupIsolatedTestHost(t *testing.T) (*host.Host, func()) {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "optimis_mvp_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	pluginDir := filepath.Join("..", "..", "plugins", "test_plugin")
	absPluginDir, err := filepath.Abs(pluginDir)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("failed to resolve plugin dir path: %v", err)
	}

	cfg := host.Config{
		DataDirPath:      tempDir,
		SchoolURN:        "100001",
		HostKMSKeyHandle: "kms-test-key-001",
		PluginsDir:       filepath.Dir(absPluginDir),
	}

	h, err := host.NewHost(cfg)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("failed to initialize host: %v", err)
	}

	if err := h.LoadPlugin("test_plugin"); err != nil {
		h.Close()
		os.RemoveAll(tempDir)
		t.Fatalf("failed to load test_plugin: %v", err)
	}

	logCursor := h.LogCursor()

	cleanup := func() {
		if t.Failed() {
			logs := h.LogsSince(logCursor)
			t.Logf("--- TAPPED SERVICE LOGS ---\n%s", strings.Join(logs, "\n"))
			t.Logf("--- TAPPED DIAGNOSTIC RING BUFFER ---\n%s", strings.Join(h.Diagnostics(), "\n"))
		}
		h.Close()
		os.RemoveAll(tempDir)
	}

	return h, cleanup
}

func TestExecuteRPC_GetPupilRecordWithValidScope_ReturnsSeededPupilAndEmitsEvent_test(t *testing.T) {
	ctx := context.Background()
	h, cleanup := setupIsolatedTestHost(t)
	t.Cleanup(cleanup)

	seededPupil, err := h.DAL().SeedPupil(ctx, dal.SeedPupilInput{
		UPN:            "A123456789012",
		FirstName:      "Ada",
		LastName:       "Lovelace",
		AcademicYearID: "2026-2027",
		EffectiveFrom:  "2026-09-01",
	})
	if err != nil {
		t.Fatalf("failed to seed pupil: %v", err)
	}
	t.Cleanup(func() {
		_ = h.DAL().DeletePupil(context.Background(), seededPupil.ID)
	})

	eventCh := h.SubscribeEvents("test.pupil_fetched")

	caller := host.CallerContext{
		Type:        "USER",
		UserID:      "usr_teacher_1",
		StaffID:     "stf_001",
		ActiveRoles: []string{"teacher"},
		Scopes:      []string{"dal.pupils.read:all"},
		RequestID:   "req_test_001",
	}

	res, err := h.ExecuteRPC(ctx, "uk.org.optimis.test", "get_pupil_record", caller, map[string]any{
		"pupil_id": seededPupil.ID,
	})
	if err != nil {
		t.Fatalf("ExecuteRPC returned unexpected error: %v", err)
	}
	if !res.OK {
		t.Fatalf("expected res.OK == true, got false (error: %s)", res.ErrorCode)
	}

	dataMap, ok := res.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected res.Data to be map[string]any, got %T", res.Data)
	}

	if dataMap["id"] != seededPupil.ID {
		t.Errorf("expected id %q, got %v", seededPupil.ID, dataMap["id"])
	}
	if dataMap["upn"] != "A123456789012" {
		t.Errorf("expected upn %q, got %v", "A123456789012", dataMap["upn"])
	}
	if dataMap["first_name"] != "Ada" {
		t.Errorf("expected first_name %q, got %v", "Ada", dataMap["first_name"])
	}
	if dataMap["last_name"] != "Lovelace" {
		t.Errorf("expected last_name %q, got %v", "Lovelace", dataMap["last_name"])
	}
	if dataMap["ctx_modules_verified"] != true {
		t.Errorf("expected Lua runtime to verify all 5 injected ctx namespaces (caller, dal, plugin_db, events, http), got %v", dataMap["ctx_modules_verified"])
	}

	select {
	case evt := <-eventCh:
		if evt.Name != "test.pupil_fetched" {
			t.Errorf("expected event name test.pupil_fetched, got %s", evt.Name)
		}
		if evt.Payload["pupil_id"] != seededPupil.ID {
			t.Errorf("expected event payload pupil_id %q, got %v", seededPupil.ID, evt.Payload["pupil_id"])
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for post-commit event test.pupil_fetched on Event Bus")
	}
}

func TestExecuteRPC_UndeclaredMethodOrMissingScope_FailsFast_test(t *testing.T) {
	ctx := context.Background()
	h, cleanup := setupIsolatedTestHost(t)
	t.Cleanup(cleanup)

	unauthorizedCaller := host.CallerContext{
		Type:        "USER",
		UserID:      "usr_guest",
		StaffID:     "stf_guest",
		ActiveRoles: []string{"guest"},
		Scopes:      []string{},
		RequestID:   "req_test_unauth",
	}

	_, err := h.ExecuteRPC(ctx, "uk.org.optimis.test", "get_pupil_record", unauthorizedCaller, map[string]any{
		"pupil_id": "pup_123",
	})
	if err == nil {
		t.Fatalf("expected fail-fast error when caller lacks required scope dal.pupils.read:all, got nil")
	}

	authorizedCaller := host.CallerContext{
		Type:        "USER",
		UserID:      "usr_teacher_1",
		StaffID:     "stf_001",
		ActiveRoles: []string{"teacher"},
		Scopes:      []string{"dal.pupils.read:all"},
		RequestID:   "req_test_undeclared",
	}

	_, err = h.ExecuteRPC(ctx, "uk.org.optimis.test", "undeclared_endpoint", authorizedCaller, map[string]any{})
	if err == nil {
		t.Fatalf("expected fail-fast default-deny error for undeclared endpoint, got nil")
	}
}

func TestExecuteRPC_MissingMandatoryParameter_FailsFast_test(t *testing.T) {
	ctx := context.Background()
	h, cleanup := setupIsolatedTestHost(t)
	t.Cleanup(cleanup)

	caller := host.CallerContext{
		Type:        "USER",
		UserID:      "usr_teacher_1",
		StaffID:     "stf_001",
		ActiveRoles: []string{"teacher"},
		Scopes:      []string{"dal.pupils.read:all"},
		RequestID:   "req_test_missing_param",
	}

	_, err := h.ExecuteRPC(ctx, "uk.org.optimis.test", "get_pupil_record", caller, map[string]any{})
	if err == nil {
		t.Fatalf("expected fail-fast error when mandatory pupil_id parameter is missing, got nil")
	}
}

func TestExecuteRPC_PluginLuaModifiedOnDisk_HotReloadsWithoutRestart_test(t *testing.T) {
	ctx := context.Background()

	tempDir, err := os.MkdirTemp("", "optimis_hotreload_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(tempDir)
	})

	tempPluginsDir := filepath.Join(tempDir, "plugins")
	tempPluginDir := filepath.Join(tempPluginsDir, "test_plugin")
	if err := os.MkdirAll(filepath.Join(tempPluginDir, "src"), 0o755); err != nil {
		t.Fatalf("failed to create temp plugin src dir: %v", err)
	}

	origPluginDir, err := filepath.Abs(filepath.Join("..", "..", "plugins", "test_plugin"))
	if err != nil {
		t.Fatalf("failed to resolve orig plugin dir: %v", err)
	}
	manifestBytes, err := os.ReadFile(filepath.Join(origPluginDir, "manifest.json"))
	if err != nil {
		t.Fatalf("failed to read orig manifest.json: %v", err)
	}
	luaBytes, err := os.ReadFile(filepath.Join(origPluginDir, "src", "init.lua"))
	if err != nil {
		t.Fatalf("failed to read orig init.lua: %v", err)
	}

	manifestPath := filepath.Join(tempPluginDir, "manifest.json")
	luaPath := filepath.Join(tempPluginDir, "src", "init.lua")
	if err := os.WriteFile(manifestPath, manifestBytes, 0o644); err != nil {
		t.Fatalf("failed to write temp manifest.json: %v", err)
	}
	if err := os.WriteFile(luaPath, luaBytes, 0o644); err != nil {
		t.Fatalf("failed to write temp init.lua: %v", err)
	}

	h, err := host.NewHost(host.Config{
		DataDirPath:      filepath.Join(tempDir, "data"),
		SchoolURN:        "100001",
		HostKMSKeyHandle: "kms-test-key-001",
		PluginsDir:       tempPluginsDir,
	})
	if err != nil {
		t.Fatalf("failed to create host: %v", err)
	}
	logCursor := h.LogCursor()
	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("--- TAPPED SERVICE LOGS ---\n%s", strings.Join(h.LogsSince(logCursor), "\n"))
		}
		h.Close()
	})

	if err := h.LoadPlugin("test_plugin"); err != nil {
		t.Fatalf("failed to load plugin: %v", err)
	}

	seeded, err := h.DAL().SeedPupil(ctx, dal.SeedPupilInput{
		UPN:            "A123456789012",
		FirstName:      "Ada",
		LastName:       "Lovelace",
		AcademicYearID: "2026-2027",
		EffectiveFrom:  "2026-09-01",
	})
	if err != nil {
		t.Fatalf("failed to seed pupil: %v", err)
	}
	t.Cleanup(func() {
		_ = h.DAL().DeletePupil(context.Background(), seeded.ID)
	})

	caller := host.CallerContext{
		Type:        "USER",
		UserID:      "usr_teacher_1",
		StaffID:     "stf_001",
		ActiveRoles: []string{"teacher"},
		Scopes:      []string{"dal.pupils.read:all"},
		RequestID:   "req_hotreload_1",
	}

	updatedLua := strings.Replace(
		string(luaBytes),
		"ctx_modules_verified = true",
		"ctx_modules_verified = true,\n      hot_reloaded = true",
		1,
	)
	futureTime := time.Now().Add(2 * time.Second)
	if err := os.WriteFile(luaPath, []byte(updatedLua), 0o644); err != nil {
		t.Fatalf("failed to update init.lua on disk: %v", err)
	}
	if err := os.Chtimes(luaPath, futureTime, futureTime); err != nil {
		t.Fatalf("failed to update init.lua mtime: %v", err)
	}

	res, err := h.ExecuteRPC(ctx, "uk.org.optimis.test", "get_pupil_record", caller, map[string]any{
		"pupil_id": seeded.ID,
	})
	if err != nil {
		t.Fatalf("ExecuteRPC after hot reload failed: %v", err)
	}
	dataMap, ok := res.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any data, got %T", res.Data)
	}
	if dataMap["hot_reloaded"] != true {
		t.Fatalf("expected hot_reloaded == true after editing init.lua on disk without restarting host, got %v", dataMap["hot_reloaded"])
	}
}

