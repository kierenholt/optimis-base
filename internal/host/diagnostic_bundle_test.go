package host_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"optimis/internal/dal"
	"optimis/internal/host"
)

func TestExportDiagnosticBundle_LuaRuntimeErrorAndWriteLockHalt_CapturesSanitizedStackTraceWithZeroPIIOrLocals_test(t *testing.T) {
	ctx := context.Background()
	h, cleanup := setupIsolatedTestHost(t)
	t.Cleanup(cleanup)

	h.ResetDiagnostics(100)

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

	teacherCaller := host.CallerContext{
		Type:        "USER",
		UserID:      "usr_teacher_diag",
		StaffID:     seeded.TeacherStaffID,
		ActiveRoles: []string{"TEACHER"},
		Scopes:      []string{"dal.pupils.read:all", "dal.pupils.read:own_groups"},
		RequestID:   "req_diag_1",
	}

	// 1. Trigger a Lua runtime error with pupil ID, UPN, ULN, and local secret in args
	_, err = h.ExecuteRPC(ctx, "uk.org.optimis.test", "get_pupil_record", teacherCaller, map[string]any{
		"pupil_id":     "pup_secret_unit_999",
		"upn":          "A123456789012",
		"uln":          "9876543210",
		"local_secret": "Confidential local variable Oliver Twist",
	})
	if err == nil {
		t.Fatalf("expected error from get_pupil_record for non-existent pupil, got nil")
	}

	// 2. Trigger a mutation + illegal egress inside open WriteDB transaction
	_, err = h.ExecuteRPC(ctx, "uk.org.optimis.test", "mutate_then_illegal_egress", teacherCaller, map[string]any{
		"pupil_id":   seeded.PupilID,
		"note_id":    "pnote_secret_unit_1",
		"note":       "Confidential safeguarding note for Harry Potter UPN B987654321012",
		"target_url": "https://sync.sparxmaths.uk/v1/sync",
	})
	if err == nil {
		t.Fatalf("expected error from mutate_then_illegal_egress, got nil")
	}

	// 3. Verify non-Admin caller is rejected by ExportDiagnosticBundle
	_, err = h.ExportDiagnosticBundle(teacherCaller, "")
	if err == nil || !strings.Contains(err.Error(), "UNAUTHORIZED_ADMIN_DIAGNOSTICS_SCOPE") {
		t.Fatalf("expected UNAUTHORIZED_ADMIN_DIAGNOSTICS_SCOPE for teacher caller, got %v", err)
	}

	// 4. Export bundle as Admin caller
	adminCaller := host.CallerContext{
		Type:        "USER",
		UserID:      "usr_admin_diag",
		StaffID:     seeded.AdminStaffID,
		ActiveRoles: []string{"ADMIN", "DPO"},
		Scopes:      []string{"dal.pupils.read:all", "dal.pupils.write:all", "dal.staff.write:admin"},
		RequestID:   "req_diag_admin",
	}

	bundle, err := h.ExportDiagnosticBundle(adminCaller, "")
	if err != nil {
		t.Fatalf("ExportDiagnosticBundle failed for Admin caller: %v", err)
	}

	if bundle.SchoolURN != "100001" {
		t.Errorf("expected SchoolURN 100001, got %s", bundle.SchoolURN)
	}
	if !bundle.PIIScrubbed {
		t.Errorf("expected PIIScrubbed == true")
	}
	if bundle.EntryCount < 2 || len(bundle.Entries) != bundle.EntryCount {
		t.Fatalf("expected at least 2 diagnostic entries, got %d", bundle.EntryCount)
	}

	var foundGetPupil, foundMutateEgress bool
	for _, entry := range bundle.Entries {
		if entry.PluginID == "uk.org.optimis.test" && entry.Method == "get_pupil_record" && entry.ErrorClass == "LUA_RUNTIME_ERROR" {
			foundGetPupil = true
			if entry.File != "src/init.lua" || entry.Line != 13 {
				t.Errorf("expected src/init.lua:13 for get_pupil_record fault, got %s:%d", entry.File, entry.Line)
			}
			if !strings.Contains(entry.StackTrace, "src/init.lua:13") {
				t.Errorf("expected stack_trace to contain src/init.lua:13, got %s", entry.StackTrace)
			}
		}
		if entry.PluginID == "uk.org.optimis.test" && entry.Method == "mutate_then_illegal_egress" {
			foundMutateEgress = true
			if entry.File != "src/init.lua" || entry.Line != 365 {
				t.Errorf("expected src/init.lua:365 for mutate_then_illegal_egress fault, got %s:%d", entry.File, entry.Line)
			}
			if entry.SQLiteLockWaitMS < 0 || entry.SQLiteWriteHoldMS < 0 {
				t.Errorf("expected non-negative lock wait/hold durations, got wait=%d hold=%d", entry.SQLiteLockWaitMS, entry.SQLiteWriteHoldMS)
			}
		}
	}
	if !foundGetPupil {
		t.Errorf("did not find get_pupil_record LUA_RUNTIME_ERROR entry in bundle: %+v", bundle.Entries)
	}
	if !foundMutateEgress {
		t.Errorf("did not find mutate_then_illegal_egress entry in bundle: %+v", bundle.Entries)
	}

	rawJSONBytes, err := json.Marshal(bundle)
	if err != nil {
		t.Fatalf("failed to marshal diagnostic bundle: %v", err)
	}
	rawJSON := string(rawJSONBytes)

	for _, forbidden := range []string{
		"pup_secret_unit_999",
		seeded.PupilID,
		"A123456789012",
		"B987654321012",
		"9876543210",
		"Oliver Twist",
		"Harry Potter",
		"pnote_secret_unit_1",
	} {
		if strings.Contains(rawJSON, forbidden) {
			t.Errorf("expected diagnostic JSON bundle to scrub %q, but found it in: %s", forbidden, rawJSON)
		}
		if strings.Contains(bundle.LogArchive, forbidden) {
			t.Errorf("expected diagnostic log_archive to scrub %q, but found it in: %s", forbidden, bundle.LogArchive)
		}
	}
}

func TestHandleGetDiagnosticBundle_RingBufferCapAndHTTPRoute_EnforcesRBACAndEvictsOldest_test(t *testing.T) {
	ctx := context.Background()
	h, cleanup := setupIsolatedTestHost(t)
	t.Cleanup(cleanup)

	h.ResetDiagnostics(2)

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

	caller := host.CallerContext{
		Type:        "USER",
		UserID:      "usr_teacher",
		StaffID:     seeded.TeacherStaffID,
		ActiveRoles: []string{"TEACHER"},
		Scopes:      []string{"dal.pupils.read:all"},
		RequestID:   "req_ring",
	}

	for _, method := range []string{"m_oldest_1", "m_middle_2", "m_newest_3"} {
		_, _ = h.ExecuteRPC(ctx, "uk.org.optimis.test", method, caller, map[string]any{
			"pupil_id": seeded.PupilID,
		})
	}

	handler, err := h.NewHTTPHandler()
	if err != nil {
		t.Fatalf("NewHTTPHandler failed: %v", err)
	}

	// Non-admin HTTP request -> 403
	reqUnauth := httptest.NewRequest(http.MethodGet, "/api/admin/diagnostics/bundle?staff_id="+seeded.TeacherStaffID, nil)
	recUnauth := httptest.NewRecorder()
	handler.ServeHTTP(recUnauth, reqUnauth)
	if recUnauth.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for teacher, got %d (%s)", recUnauth.Code, recUnauth.Body.String())
	}

	// Admin HTTP request -> 200 with capacity=2 and only m_middle_2, m_newest_3
	reqAdmin := httptest.NewRequest(http.MethodGet, "/api/admin/diagnostics/bundle?staff_id="+seeded.AdminStaffID, nil)
	recAdmin := httptest.NewRecorder()
	handler.ServeHTTP(recAdmin, reqAdmin)
	if recAdmin.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for admin, got %d (%s)", recAdmin.Code, recAdmin.Body.String())
	}

	var resp struct {
		OK     bool                  `json:"ok"`
		Bundle host.DiagnosticBundle `json:"bundle"`
	}
	if err := json.Unmarshal(recAdmin.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response JSON: %v", err)
	}
	if !resp.OK {
		t.Fatalf("expected resp.OK == true")
	}
	if resp.Bundle.RingBufferCapacity != 2 || resp.Bundle.EntryCount != 2 {
		t.Fatalf("expected ring buffer capacity=2 and entry_count=2, got cap=%d count=%d", resp.Bundle.RingBufferCapacity, resp.Bundle.EntryCount)
	}
	if resp.Bundle.Entries[0].Method != "m_middle_2" || resp.Bundle.Entries[1].Method != "m_newest_3" {
		t.Errorf("expected retained methods [m_middle_2, m_newest_3], got [%s, %s]", resp.Bundle.Entries[0].Method, resp.Bundle.Entries[1].Method)
	}
}
