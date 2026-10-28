package host_test

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

	"optimis/internal/dal"
	"optimis/internal/host"
)

func setupIsolatedEmptyTestHost(t *testing.T) (*host.Host, func()) {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "optimis_lifecycle_test_*")
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

func TestInstallActivateAndRemovePlugin_RuntimeLifecycle_ProvisionsAttachesActivatesWorkersAndDetachesOnRemove_test(t *testing.T) {
	ctx := context.Background()
	h, cleanup := setupIsolatedEmptyTestHost(t)
	t.Cleanup(cleanup)

	// 1. Initially zero plugins are installed and plugin_test is not attached
	if got := h.ListPlugins(); len(got) != 0 {
		t.Fatalf("expected 0 plugins before install, got %d", len(got))
	}
	if h.DAL().IsPluginDBAttached("plugin_test") {
		t.Fatalf("expected plugin_test to NOT be attached before install")
	}

	// 2. Install test_plugin -> provisions & attaches plugin_test.db, starts in UNINSTALLED_DORMANT state
	loaded, err := h.InstallPlugin(ctx, "test_plugin")
	if err != nil {
		t.Fatalf("InstallPlugin(test_plugin) failed: %v", err)
	}
	if loaded.Manifest.ID != "uk.org.optimis.test" {
		t.Fatalf("expected plugin ID uk.org.optimis.test, got %s", loaded.Manifest.ID)
	}
	if loaded.State != host.PluginStateUninstalledDormant {
		t.Fatalf("expected initial installed state UNINSTALLED_DORMANT, got %s", loaded.State)
	}
	if !h.DAL().IsPluginDBAttached("plugin_test") {
		t.Fatalf("expected plugin_test to be attached after InstallPlugin")
	}

	seededPupil, err := h.DAL().SeedPupil(ctx, dal.SeedPupilInput{
		UPN:            "L123456789012",
		FirstName:      "Grace",
		LastName:       "Hopper",
		AcademicYearID: "ay_2026_2027",
		EffectiveFrom:  "2026-09-01",
	})
	if err != nil {
		t.Fatalf("failed to seed pupil: %v", err)
	}
	t.Cleanup(func() {
		_ = h.DAL().DeletePupil(context.Background(), seededPupil.ID)
	})

	adminCaller := host.CallerContext{
		Type:        "USER",
		UserID:      "usr_admin_1",
		StaffID:     "stf_admin_1",
		ActiveRoles: []string{"ADMIN"},
		Scopes:      []string{"dal.pupils.read:all", "dal.pupils.write:all"},
		RequestID:   "req_lifecycle_1",
	}

	// 3. While in UNINSTALLED_DORMANT, ExecuteRPC must fail fast with PLUGIN_DORMANT_QUARANTINE
	_, err = h.ExecuteRPC(ctx, "uk.org.optimis.test", "get_pupil_record", adminCaller, map[string]any{
		"pupil_id": seededPupil.ID,
	})
	if err == nil || !strings.Contains(err.Error(), "PLUGIN_DORMANT_QUARANTINE") {
		t.Fatalf("expected PLUGIN_DORMANT_QUARANTINE while installed but not activated, got: %v", err)
	}

	// 4. Activate plugin -> transitions to ACTIVE, allows ExecuteRPC and attached DB operations
	if err := h.ActivatePlugin("uk.org.optimis.test"); err != nil {
		t.Fatalf("ActivatePlugin(uk.org.optimis.test) failed: %v", err)
	}

	plugins := h.ListPlugins()
	if len(plugins) != 1 || plugins[0].ID != "uk.org.optimis.test" || plugins[0].State != string(host.PluginStateActive) {
		t.Fatalf("expected 1 ACTIVE plugin in ListPlugins(), got %+v", plugins)
	}

	rpcRes, err := h.ExecuteRPC(ctx, "uk.org.optimis.test", "get_pupil_record", adminCaller, map[string]any{
		"pupil_id": seededPupil.ID,
	})
	if err != nil || !rpcRes.OK {
		t.Fatalf("expected ExecuteRPC to succeed after ActivatePlugin, got err=%v res=%+v", err, rpcRes)
	}

	// 5. Remove plugin -> stops workers, removes from ListPlugins(), and detaches plugin_test.db
	if err := h.RemovePlugin(ctx, "uk.org.optimis.test"); err != nil {
		t.Fatalf("RemovePlugin(uk.org.optimis.test) failed: %v", err)
	}
	if got := h.ListPlugins(); len(got) != 0 {
		t.Fatalf("expected 0 plugins after RemovePlugin, got %+v", got)
	}
	if h.DAL().IsPluginDBAttached("plugin_test") {
		t.Fatalf("expected plugin_test to be detached after RemovePlugin")
	}

	_, err = h.ExecuteRPC(ctx, "uk.org.optimis.test", "get_pupil_record", adminCaller, map[string]any{
		"pupil_id": seededPupil.ID,
	})
	if err == nil || !strings.Contains(err.Error(), "plugin not loaded") {
		t.Fatalf("expected ExecuteRPC after RemovePlugin to fail with 'plugin not loaded', got: %v", err)
	}
}

func TestAdminPluginLifecycleHTTP_NonAdminCallerRejectedAndAdminSucceeds_test(t *testing.T) {
	ctx := context.Background()
	h, cleanup := setupIsolatedEmptyTestHost(t)
	t.Cleanup(cleanup)

	seeded, err := h.DAL().SeedGDPRTestFixtures(ctx, dal.GDPRTestFixtureInput{
		AcademicYearID:   "ay_2026_2027",
		ReferenceDate:    "2026-10-20",
		HostKMSKeyHandle: "kms-test-key-001",
	})
	if err != nil {
		t.Fatalf("failed to seed GDPR fixtures: %v", err)
	}
	t.Cleanup(func() {
		_ = h.DAL().DeleteGDPRTestFixtures(context.Background(), seeded.FixtureID)
	})

	handler, err := h.NewHTTPHandler()
	if err != nil {
		t.Fatalf("NewHTTPHandler failed: %v", err)
	}

	// 1. Non-Admin (TEACHER) rejected with 403 UNAUTHORIZED_ADMIN_PLUGIN_SCOPE on install
	installTeacherBody, _ := json.Marshal(map[string]any{
		"folder":   "test_plugin",
		"staff_id": seeded.TeacherStaffID,
	})
	reqUnauthInstall := httptest.NewRequest(http.MethodPost, "/api/admin/plugins/install", bytes.NewReader(installTeacherBody))
	recUnauthInstall := httptest.NewRecorder()
	handler.ServeHTTP(recUnauthInstall, reqUnauthInstall)
	if recUnauthInstall.Code != http.StatusForbidden || !strings.Contains(recUnauthInstall.Body.String(), "UNAUTHORIZED_ADMIN_PLUGIN_SCOPE") {
		t.Fatalf("expected 403 UNAUTHORIZED_ADMIN_PLUGIN_SCOPE for teacher on install, got %d (%s)", recUnauthInstall.Code, recUnauthInstall.Body.String())
	}

	// 2. Admin installs test_plugin -> 201 Created, state: UNINSTALLED_DORMANT
	installAdminBody, _ := json.Marshal(map[string]any{
		"folder":   "test_plugin",
		"staff_id": seeded.AdminStaffID,
	})
	reqInstall := httptest.NewRequest(http.MethodPost, "/api/admin/plugins/install", bytes.NewReader(installAdminBody))
	recInstall := httptest.NewRecorder()
	handler.ServeHTTP(recInstall, reqInstall)
	if recInstall.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for admin install, got %d (%s)", recInstall.Code, recInstall.Body.String())
	}
	var installResp struct {
		OK       bool   `json:"ok"`
		PluginID string `json:"plugin_id"`
		Folder   string `json:"folder"`
		State    string `json:"state"`
	}
	if err := json.Unmarshal(recInstall.Body.Bytes(), &installResp); err != nil {
		t.Fatalf("failed to decode install response: %v", err)
	}
	if !installResp.OK || installResp.PluginID != "uk.org.optimis.test" || installResp.State != "UNINSTALLED_DORMANT" {
		t.Fatalf("unexpected install response: %+v", installResp)
	}

	// 3. Non-Admin (TEACHER) rejected with 403 on GET /api/admin/plugins and POST /api/admin/plugins/{id}/activate
	reqUnauthList := httptest.NewRequest(http.MethodGet, "/api/admin/plugins?staff_id="+seeded.TeacherStaffID, nil)
	recUnauthList := httptest.NewRecorder()
	handler.ServeHTTP(recUnauthList, reqUnauthList)
	if recUnauthList.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for teacher on GET /api/admin/plugins, got %d", recUnauthList.Code)
	}

	activateTeacherBody, _ := json.Marshal(map[string]any{
		"staff_id": seeded.TeacherStaffID,
	})
	reqUnauthActivate := httptest.NewRequest(http.MethodPost, "/api/admin/plugins/uk.org.optimis.test/activate", bytes.NewReader(activateTeacherBody))
	recUnauthActivate := httptest.NewRecorder()
	handler.ServeHTTP(recUnauthActivate, reqUnauthActivate)
	if recUnauthActivate.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for teacher on activate, got %d", recUnauthActivate.Code)
	}

	// 4. Admin activates uk.org.optimis.test -> 200 OK, state: ACTIVE
	activateAdminBody, _ := json.Marshal(map[string]any{
		"staff_id": seeded.AdminStaffID,
	})
	reqActivate := httptest.NewRequest(http.MethodPost, "/api/admin/plugins/uk.org.optimis.test/activate", bytes.NewReader(activateAdminBody))
	recActivate := httptest.NewRecorder()
	handler.ServeHTTP(recActivate, reqActivate)
	if recActivate.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for admin activate, got %d (%s)", recActivate.Code, recActivate.Body.String())
	}

	// 5. Non-Admin (TEACHER) rejected with 403 on DELETE /api/admin/plugins/{id}
	reqUnauthDelete := httptest.NewRequest(http.MethodDelete, "/api/admin/plugins/uk.org.optimis.test?staff_id="+seeded.TeacherStaffID, nil)
	recUnauthDelete := httptest.NewRecorder()
	handler.ServeHTTP(recUnauthDelete, reqUnauthDelete)
	if recUnauthDelete.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for teacher on delete, got %d", recUnauthDelete.Code)
	}

	// 6. Admin removes uk.org.optimis.test -> 200 OK, removed: true
	reqDelete := httptest.NewRequest(http.MethodDelete, "/api/admin/plugins/uk.org.optimis.test?staff_id="+seeded.AdminStaffID, nil)
	recDelete := httptest.NewRecorder()
	handler.ServeHTTP(recDelete, reqDelete)
	if recDelete.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for admin delete, got %d (%s)", recDelete.Code, recDelete.Body.String())
	}
	var deleteResp struct {
		OK       bool   `json:"ok"`
		PluginID string `json:"plugin_id"`
		Removed  bool   `json:"removed"`
	}
	if err := json.Unmarshal(recDelete.Body.Bytes(), &deleteResp); err != nil {
		t.Fatalf("failed to decode delete response: %v", err)
	}
	if !deleteResp.OK || !deleteResp.Removed || deleteResp.PluginID != "uk.org.optimis.test" {
		t.Fatalf("unexpected delete response: %+v", deleteResp)
	}
}
