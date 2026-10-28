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

func TestCoreEventCatalog_CoreDALMutationsCommitAndRollback_EmitsCatalogEventsOnlyAfterCommit_test(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()
	pluginsDir := filepath.Join(tempDir, "plugins")
	pluginDir := filepath.Join(pluginsDir, "noop_plugin")
	if err := os.MkdirAll(filepath.Join(pluginDir, "src"), 0o755); err != nil {
		t.Fatalf("failed to create plugin dir: %v", err)
	}
	manifest := `{
  "id": "uk.org.optimis.noop",
  "version": "0.1.0",
  "core_dal_version": "~1.0",
  "egress": { "allowed_domains": [] },
  "endpoints": {
    "ping": { "scopes": [], "mutates": false }
  }
}`
	if err := os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("failed to write manifest.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "src", "init.lua"), []byte(`function ping(ctx, args) return { ok = true, data = {} } end`), 0o644); err != nil {
		t.Fatalf("failed to write init.lua: %v", err)
	}

	h, err := host.NewHost(host.Config{
		DataDirPath:      filepath.Join(tempDir, "data"),
		SchoolURN:        "100001",
		HostKMSKeyHandle: "kms-test-key-001",
		PluginsDir:       pluginsDir,
	})
	if err != nil {
		t.Fatalf("failed to create host: %v", err)
	}
	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("--- TAPPED SERVICE LOGS ---\n%s", strings.Join(h.LogsSince("0"), "\n"))
		}
		_ = h.Close()
	})

	// Verify Core Event Catalog contains all required standard Core domain events
	catalog := dal.CoreEventCatalog()
	expectedEvents := []string{
		dal.EventCorePupilEnrolled,
		dal.EventCoreGroupMembershipChanged,
		dal.EventCoreAttendanceSessionRecorded,
		dal.EventCoreSafeguardingPostboxCreated,
		dal.EventCoreGDPRPupilPurged,
	}
	for _, exp := range expectedEvents {
		if !dal.IsValidCoreEventName(exp) {
			t.Fatalf("expected %q to be a valid Core Event Catalog name (catalog=%v)", exp, catalog)
		}
	}
	if dal.IsValidCoreEventName("core.nonexistent.event") {
		t.Fatalf("expected core.nonexistent.event to be rejected by IsValidCoreEventName")
	}

	enrolledCh := h.SubscribeEvents(dal.EventCorePupilEnrolled)
	membershipCh := h.SubscribeEvents(dal.EventCoreGroupMembershipChanged)
	attendanceCh := h.SubscribeEvents(dal.EventCoreAttendanceSessionRecorded)
	safeguardingCh := h.SubscribeEvents(dal.EventCoreSafeguardingPostboxCreated)
	purgedCh := h.SubscribeEvents(dal.EventCoreGDPRPupilPurged)

	// 1. SeedPupil -> emits core.pupil.enrolled post-commit
	seededPupil, err := h.DAL().SeedPupil(ctx, dal.SeedPupilInput{
		UPN:            "E123456789012",
		FirstName:      "Grace",
		LastName:       "Hopper",
		AcademicYearID: "ay_2026_2027",
		EffectiveFrom:  "2026-09-01",
	})
	if err != nil {
		t.Fatalf("SeedPupil failed: %v", err)
	}

	select {
	case evt := <-enrolledCh:
		if evt.Hop != 1 {
			t.Errorf("expected Hop == 1 on core.pupil.enrolled, got %d", evt.Hop)
		}
		if evt.Payload["pupil_id"] != seededPupil.ID {
			t.Errorf("expected pupil_id %s, got %v", seededPupil.ID, evt.Payload["pupil_id"])
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for %s", dal.EventCorePupilEnrolled)
	}

	// Seed Milestone 1 fixtures so we can test group membership, attendance, and safeguarding postbox commits
	m1Seeded, err := h.DAL().SeedMilestone1Fixtures(ctx, dal.Milestone1SeedInput{
		AcademicYearID: "ay_2026_2027",
		ReferenceDate:  "2026-10-27",
	})
	if err != nil {
		t.Fatalf("SeedMilestone1Fixtures failed: %v", err)
	}
	t.Cleanup(func() {
		_ = h.DAL().DeleteMilestone1Fixtures(context.Background(), m1Seeded.FixtureID)
	})

	scope := dal.DALQueryScope{
		Mode:           dal.ScopeOwnGroups,
		StaffID:        m1Seeded.TeacherStaffID,
		AcademicYearID: m1Seeded.AcademicYearID,
		ReferenceDate:  m1Seeded.ReferenceDate,
	}

	// 2. UpsertGroupPupilMembershipScoped -> emits core.group.membership_changed post-commit
	_, err = h.DAL().UpsertGroupPupilMembershipScoped(ctx, scope, dal.UpsertGroupPupilMembershipInput{
		GroupID:        m1Seeded.OwnGroupID,
		PupilID:        seededPupil.ID,
		AcademicYearID: m1Seeded.AcademicYearID,
		EffectiveFrom:  "2026-09-01",
	})
	if err != nil {
		t.Fatalf("UpsertGroupPupilMembershipScoped failed: %v", err)
	}

	select {
	case evt := <-membershipCh:
		if evt.Hop != 1 {
			t.Errorf("expected Hop == 1 on core.group.membership_changed, got %d", evt.Hop)
		}
		if evt.Payload["group_id"] != m1Seeded.OwnGroupID || evt.Payload["pupil_id"] != seededPupil.ID {
			t.Errorf("unexpected membership event payload: %+v", evt.Payload)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for %s", dal.EventCoreGroupMembershipChanged)
	}

	// 3. Verify failed/rolled-back attendance batch does NOT emit core.attendance.session_recorded
	_, err = h.DAL().UpsertSessionAttendanceBatchScoped(ctx, scope, dal.UpsertSessionAttendanceBatchInput{
		GroupID: m1Seeded.OwnGroupID,
		Date:    m1Seeded.ReferenceDate,
		Session: "AM",
		Marks: []dal.SessionAttendanceMarkInput{
			{
				PupilID:        "pup_not_in_group_999",
				AttendanceCode: "/",
			},
		},
	})
	if err == nil {
		t.Fatalf("expected UpsertSessionAttendanceBatchScoped to fail for non-member pupil")
	}
	select {
	case unexpected := <-attendanceCh:
		t.Fatalf("expected ZERO events on rolled-back transaction, got %+v", unexpected)
	default:
	}

	// 4. Valid UpsertSessionAttendanceBatchScoped -> emits core.attendance.session_recorded post-commit
	attRes, err := h.DAL().UpsertSessionAttendanceBatchScoped(ctx, scope, dal.UpsertSessionAttendanceBatchInput{
		GroupID: m1Seeded.OwnGroupID,
		Date:    m1Seeded.ReferenceDate,
		Session: "AM",
		Marks: []dal.SessionAttendanceMarkInput{
			{
				PupilID:        m1Seeded.ActivePupilID,
				AttendanceCode: "/",
				MinutesLate:    0,
			},
		},
	})
	if err != nil {
		t.Fatalf("UpsertSessionAttendanceBatchScoped failed: %v", err)
	}

	select {
	case evt := <-attendanceCh:
		if evt.Hop != 1 {
			t.Errorf("expected Hop == 1, got %d", evt.Hop)
		}
		if evt.Payload["submission_id"] != attRes.SubmissionID || evt.Payload["group_id"] != m1Seeded.OwnGroupID {
			t.Errorf("unexpected attendance event payload: %+v", evt.Payload)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for %s", dal.EventCoreAttendanceSessionRecorded)
	}

	// 5. CreateSafeguardingPostboxNote -> emits core.safeguarding.postbox_created post-commit (envelope metadata only)
	sfgReceipt, err := h.DAL().CreateSafeguardingPostboxNote(ctx, dal.CreateSafeguardingPostboxInput{
		PupilID:           m1Seeded.ActivePupilID,
		ReportedByStaffID: m1Seeded.TeacherStaffID,
		ConcernCategory:   "PHYSICAL",
		IncidentDate:      m1Seeded.ReferenceDate,
		Summary:           "CONFIDENTIAL_SUMMARY_MUST_NOT_LEAK",
		VerbatimAccount:   "CONFIDENTIAL_VERBATIM_MUST_NOT_LEAK",
		RequestID:         "req_evt_sfg_1",
		HostKMSKeyHandle:  "kms-test-key-001",
	})
	if err != nil {
		t.Fatalf("CreateSafeguardingPostboxNote failed: %v", err)
	}

	select {
	case evt := <-safeguardingCh:
		if evt.Hop != 1 {
			t.Errorf("expected Hop == 1, got %d", evt.Hop)
		}
		if evt.Payload["note_id"] != sfgReceipt.NoteID || evt.Payload["pupil_id"] != m1Seeded.ActivePupilID {
			t.Errorf("unexpected safeguarding event payload: %+v", evt.Payload)
		}
		if _, hasSummary := evt.Payload["summary"]; hasSummary {
			t.Fatalf("SECURITY VIOLATION: safeguarding postbox event leaked plaintext summary")
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for %s", dal.EventCoreSafeguardingPostboxCreated)
	}

	// 6. PurgeCorePupilData on eligible seededPupil -> emits core.gdpr.pupil_purged post-commit
	if err := h.DAL().PurgeCorePupilData(ctx, seededPupil.ID); err != nil {
		t.Fatalf("PurgeCorePupilData failed: %v", err)
	}

	select {
	case evt := <-purgedCh:
		if evt.Hop != 1 {
			t.Errorf("expected Hop == 1, got %d", evt.Hop)
		}
		if evt.Payload["pupil_id"] != seededPupil.ID {
			t.Errorf("expected purged pupil_id %s, got %v", seededPupil.ID, evt.Payload["pupil_id"])
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for %s", dal.EventCoreGDPRPupilPurged)
	}
}

func TestEventBusRouter_ManifestWorkerSubscriptionsAndHopLimit_ExecutesWorkersAndDropsEventsAboveHop3_test(t *testing.T) {
	ctx := context.Background()
	h, cleanup := setupIsolatedTestHost(t)
	t.Cleanup(cleanup)

	m1Seeded, err := h.DAL().SeedMilestone1Fixtures(ctx, dal.Milestone1SeedInput{
		AcademicYearID: "ay_2026_2027",
		ReferenceDate:  "2026-10-27",
	})
	if err != nil {
		t.Fatalf("SeedMilestone1Fixtures failed: %v", err)
	}
	t.Cleanup(func() {
		_ = h.DAL().DeleteMilestone1Fixtures(context.Background(), m1Seeded.FixtureID)
	})

	teacherCaller := host.CallerContext{
		Type:        "USER",
		UserID:      "usr_teacher_1",
		StaffID:     m1Seeded.TeacherStaffID,
		ActiveRoles: []string{"TEACHER"},
		Scopes: []string{
			"dal.pupils.read:all",
			"dal.attendance.read:own_groups",
			"dal.attendance.write:own_groups",
		},
		RequestID: "req_hop_chain_1",
	}

	// 1. Trigger statutory AM attendance via RPC -> CoreDAL commits -> emits core.attendance.session_recorded (Hop=1)
	// -> test_plugin worker `on_core_event_audit` receives event and writes row to plugin_test.worker_event_logs
	_, err = h.ExecuteRPC(ctx, "uk.org.optimis.test", "record_group_am_attendance", teacherCaller, map[string]any{
		"group_id":         m1Seeded.OwnGroupID,
		"pupil_id":         m1Seeded.ActivePupilID,
		"reference_date":   m1Seeded.ReferenceDate,
		"academic_year_id": m1Seeded.AcademicYearID,
	})
	if err != nil {
		t.Fatalf("ExecuteRPC record_group_am_attendance failed: %v", err)
	}

	// 2. Trigger recursive loop relay via RPC -> emits test.loop_relay (Hop=1) -> worker re-emits Hop=2 -> Hop=3 -> Hop=4 (dropped!)
	_, err = h.ExecuteRPC(ctx, "uk.org.optimis.test", "trigger_loop_relay", teacherCaller, map[string]any{
		"pupil_id": m1Seeded.ActivePupilID,
	})
	if err != nil {
		t.Fatalf("ExecuteRPC trigger_loop_relay failed: %v", err)
	}

	// Poll worker_event_logs until core.attendance.session_recorded (1) + test.loop_relay (3 hops) = 4 rows are recorded
	deadline := time.Now().Add(3 * time.Second)
	var logs []any
	for time.Now().Before(deadline) {
		res, qErr := h.ExecuteRPC(ctx, "uk.org.optimis.test", "list_worker_event_logs", teacherCaller, map[string]any{
			"pupil_id": m1Seeded.ActivePupilID,
		})
		if qErr == nil && res.OK {
			dataMap := res.Data.(map[string]any)
			if rawLogs, ok := dataMap["logs"].([]any); ok && len(rawLogs) >= 4 {
				logs = rawLogs
				break
			}
		}
		time.Sleep(25 * time.Millisecond)
	}

	if len(logs) != 4 {
		t.Fatalf("expected exactly 4 worker_event_logs entries (1 core.attendance.session_recorded + 3 test.loop_relay hops), got %d: %+v", len(logs), logs)
	}

	var loopHops []int
	var foundCoreAtt bool
	for _, item := range logs {
		row := item.(map[string]any)
		evtName, _ := row["event_name"].(string)
		hopFloat, _ := row["hop"].(float64)
		callerType, _ := row["caller_type"].(string)
		callerStaffID, _ := row["caller_staff_id"].(string)

		if callerType != "SYSTEM" || callerStaffID != "" {
			t.Errorf("expected worker execution to have caller_type='SYSTEM' and caller_staff_id='', got type=%q staff_id=%q", callerType, callerStaffID)
		}
		if evtName == dal.EventCoreAttendanceSessionRecorded {
			foundCoreAtt = true
			if int(hopFloat) != 1 {
				t.Errorf("expected core.attendance.session_recorded hop == 1, got %v", hopFloat)
			}
		}
		if evtName == "test.loop_relay" {
			loopHops = append(loopHops, int(hopFloat))
		}
	}

	if !foundCoreAtt {
		t.Errorf("expected worker to receive %s", dal.EventCoreAttendanceSessionRecorded)
	}
	if len(loopHops) != 3 || loopHops[0] != 1 || loopHops[1] != 2 || loopHops[2] != 3 {
		t.Fatalf("expected test.loop_relay to execute for exactly hops [1, 2, 3] and drop hop 4, got %v", loopHops)
	}

	metrics := h.EventBusMetrics()
	if metrics.EventsDroppedTotal < 1 {
		t.Errorf("expected EventsDroppedTotal >= 1 after Hop > 3 drop, got %d", metrics.EventsDroppedTotal)
	}
}

func TestEventBusRouter_SlowPluginSubscriber_BuffersInPerPluginChannelWithoutBlockingFastPlugin_test(t *testing.T) {
	tempDir := t.TempDir()
	pluginsDir := filepath.Join(tempDir, "plugins")

	// Plugin A (slow worker: blocks for 400ms per event, max_concurrency = 1)
	slowDir := filepath.Join(pluginsDir, "slow_plugin")
	if err := os.MkdirAll(filepath.Join(slowDir, "src"), 0o755); err != nil {
		t.Fatalf("failed to create slow_plugin dir: %v", err)
	}
	slowManifest := `{
  "id": "uk.org.optimis.slow",
  "version": "0.1.0",
  "core_dal_version": "~1.0",
  "max_concurrency": 1,
  "egress": { "allowed_domains": [] },
  "endpoints": {
    "emit_shared": { "scopes": [], "mutates": false }
  },
  "workers": {
    "on_shared_slow": {
      "events": ["shared.burst_event"],
      "scopes": [],
      "mutates": false
    }
  }
}`
	slowLua := `
function emit_shared(ctx, args)
  ctx.events.emit("shared.burst_event", { seq = args.seq })
  return { ok = true, data = { emitted = args.seq } }
end

function on_shared_slow(ctx, args)
  -- Busy loop ~150ms to simulate slow plugin worker processing
  local s = 0
  for i = 1, 2500000 do
    s = s + i
  end
  return { ok = true, data = { sum = s } }
end
`
	if err := os.WriteFile(filepath.Join(slowDir, "manifest.json"), []byte(slowManifest), 0o644); err != nil {
		t.Fatalf("failed to write slow manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(slowDir, "src", "init.lua"), []byte(slowLua), 0o644); err != nil {
		t.Fatalf("failed to write slow lua: %v", err)
	}

	// Plugin B (fast worker: immediately re-emits fast.ack_done)
	fastDir := filepath.Join(pluginsDir, "fast_plugin")
	if err := os.MkdirAll(filepath.Join(fastDir, "src"), 0o755); err != nil {
		t.Fatalf("failed to create fast_plugin dir: %v", err)
	}
	fastManifest := `{
  "id": "uk.org.optimis.fast",
  "version": "0.1.0",
  "core_dal_version": "~1.0",
  "max_concurrency": 2,
  "egress": { "allowed_domains": [] },
  "endpoints": {
    "noop": { "scopes": [], "mutates": false }
  },
  "workers": {
    "on_shared_fast": {
      "events": ["shared.burst_event"],
      "scopes": [],
      "mutates": false
    }
  }
}`
	fastLua := `
function noop(ctx, args)
  return { ok = true, data = {} }
end

function on_shared_fast(ctx, args)
  ctx.events.emit("fast.ack_done", { seq = args.seq })
  return { ok = true, data = {} }
end
`
	if err := os.WriteFile(filepath.Join(fastDir, "manifest.json"), []byte(fastManifest), 0o644); err != nil {
		t.Fatalf("failed to write fast manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(fastDir, "src", "init.lua"), []byte(fastLua), 0o644); err != nil {
		t.Fatalf("failed to write fast lua: %v", err)
	}

	h, err := host.NewHost(host.Config{
		DataDirPath:      filepath.Join(tempDir, "data"),
		SchoolURN:        "100001",
		HostKMSKeyHandle: "kms-test-key-001",
		PluginsDir:       pluginsDir,
	})
	if err != nil {
		t.Fatalf("failed to create host: %v", err)
	}
	t.Cleanup(func() {
		_ = h.Close()
	})

	if err := h.LoadPlugin("slow_plugin"); err != nil {
		t.Fatalf("LoadPlugin slow_plugin failed: %v", err)
	}
	if err := h.LoadPlugin("fast_plugin"); err != nil {
		t.Fatalf("LoadPlugin fast_plugin failed: %v", err)
	}

	fastAckCh := h.SubscribeEvents("fast.ack_done")
	caller := host.CallerContext{
		Type:      "USER",
		UserID:    "usr_1",
		StaffID:   "stf_1",
		RequestID: "req_burst",
	}

	start := time.Now()
	const burstCount = 5
	for i := 1; i <= burstCount; i++ {
		_, err := h.ExecuteRPC(context.Background(), "uk.org.optimis.slow", "emit_shared", caller, map[string]any{
			"seq": i,
		})
		if err != nil {
			t.Fatalf("emit_shared %d failed: %v", i, err)
		}
	}
	dispatchElapsed := time.Since(start)
	if dispatchElapsed > 250*time.Millisecond {
		t.Errorf("expected non-blocking dispatch for %d events to complete quickly (<250ms), took %v", burstCount, dispatchElapsed)
	}

	for i := 1; i <= burstCount; i++ {
		select {
		case evt := <-fastAckCh:
			if evt.Hop != 2 {
				t.Errorf("expected fast.ack_done Hop == 2, got %d", evt.Hop)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for fast_plugin ack %d/%d", i, burstCount)
		}
	}
}

func TestLoadPlugin_WorkerSubscribesToUnknownCoreEventOrMissingLuaFunction_FailsFast_test(t *testing.T) {
	tempDir := t.TempDir()
	pluginsDir := filepath.Join(tempDir, "plugins")

	// 1. Plugin subscribing to an unknown `core.*` event not in CoreEventCatalog MUST fail fast
	badEventDir := filepath.Join(pluginsDir, "bad_core_event_plugin")
	if err := os.MkdirAll(filepath.Join(badEventDir, "src"), 0o755); err != nil {
		t.Fatalf("failed to create bad_core_event_plugin dir: %v", err)
	}
	badEventManifest := `{
  "id": "uk.org.optimis.badevent",
  "version": "0.1.0",
  "core_dal_version": "~1.0",
  "egress": { "allowed_domains": [] },
  "endpoints": {
    "ping": { "scopes": [], "mutates": false }
  },
  "workers": {
    "on_fake_core_event": {
      "events": ["core.imaginary.unknown_event"],
      "scopes": [],
      "mutates": false
    }
  }
}`
	if err := os.WriteFile(filepath.Join(badEventDir, "manifest.json"), []byte(badEventManifest), 0o644); err != nil {
		t.Fatalf("failed to write manifest.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(badEventDir, "src", "init.lua"), []byte(`
function ping(ctx, args) return { ok = true, data = {} } end
function on_fake_core_event(ctx, args) return { ok = true, data = {} } end
`), 0o644); err != nil {
		t.Fatalf("failed to write init.lua: %v", err)
	}

	// 2. Plugin declaring a worker in manifest.json that does not exist in init.lua MUST fail fast
	missingFnDir := filepath.Join(pluginsDir, "missing_worker_fn_plugin")
	if err := os.MkdirAll(filepath.Join(missingFnDir, "src"), 0o755); err != nil {
		t.Fatalf("failed to create missing_worker_fn_plugin dir: %v", err)
	}
	missingFnManifest := `{
  "id": "uk.org.optimis.missingworker",
  "version": "0.1.0",
  "core_dal_version": "~1.0",
  "egress": { "allowed_domains": [] },
  "endpoints": {
    "ping": { "scopes": [], "mutates": false }
  },
  "workers": {
    "undeclared_in_lua_worker": {
      "events": ["core.attendance.session_recorded"],
      "scopes": [],
      "mutates": false
    }
  }
}`
	if err := os.WriteFile(filepath.Join(missingFnDir, "manifest.json"), []byte(missingFnManifest), 0o644); err != nil {
		t.Fatalf("failed to write manifest.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(missingFnDir, "src", "init.lua"), []byte(`
function ping(ctx, args) return { ok = true, data = {} } end
`), 0o644); err != nil {
		t.Fatalf("failed to write init.lua: %v", err)
	}

	h, err := host.NewHost(host.Config{
		DataDirPath:      filepath.Join(tempDir, "data"),
		SchoolURN:        "100001",
		HostKMSKeyHandle: "kms-test-key-001",
		PluginsDir:       pluginsDir,
	})
	if err != nil {
		t.Fatalf("failed to create host: %v", err)
	}
	t.Cleanup(func() {
		_ = h.Close()
	})

	if err := h.LoadPlugin("bad_core_event_plugin"); err == nil {
		t.Fatalf("expected LoadPlugin to fail fast when worker subscribes to unknown core.* event")
	}

	if err := h.LoadPlugin("missing_worker_fn_plugin"); err == nil {
		t.Fatalf("expected LoadPlugin to fail fast when manifest worker function is missing in init.lua")
	}
}
