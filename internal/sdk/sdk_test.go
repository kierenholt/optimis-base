package sdk

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifyPluginDir_ValidAndInvalidBundles(t *testing.T) {
	// 1. Valid production plugin (behaviour_plugin) should pass all 4 CI gates
	var out bytes.Buffer
	report, err := VerifyPluginDir(filepath.Join("..", "..", "plugins", "behaviour_plugin"), &out)
	if err != nil {
		t.Fatalf("expected behaviour_plugin to pass VerifyPluginDir, got error: %v", err)
	}
	if report.PluginID != "uk.org.optimis.behaviour" {
		t.Fatalf("unexpected plugin ID: %s", report.PluginID)
	}
	stdout := out.String()
	for _, gate := range []string{"MANIFEST_LUA_UI_OK", "CSP_OFFLINE_OK", "MIGRATIONS_OK", "GDPR_ZERO_ORPHANS_OK"} {
		if !strings.Contains(stdout, gate) {
			t.Fatalf("expected VerifyPluginDir output to contain %s, got: %s", gate, stdout)
		}
	}

	// 2. Broken UI referencing undeclared data-mis-rpc method should fail fast
	tmpDir := t.TempDir()
	brokenDir := filepath.Join(tmpDir, "broken_plugin")
	if err := copyDirectory(filepath.Join("..", "..", "plugins", "behaviour_plugin"), brokenDir); err != nil {
		t.Fatalf("failed to copy plugin: %v", err)
	}
	htmlPath := filepath.Join(brokenDir, "ui", "dist", "index.html")
	if err := os.WriteFile(htmlPath, []byte(`<button data-mis-rpc="undeclared_fn" data-mis-args="{}">Test</button>`), 0o644); err != nil {
		t.Fatalf("failed to write html: %v", err)
	}
	if _, err := VerifyPluginDir(brokenDir, nil); err == nil || !strings.Contains(err.Error(), "undeclared_fn") {
		t.Fatalf("expected VerifyPluginDir to fail on undeclared data-mis-rpc, got: %v", err)
	}
}

func TestRunPluginTestsAndPackPlugin_ExecutesAndBundles(t *testing.T) {
	tmpDir := t.TempDir()
	pluginDir := filepath.Join(tmpDir, "testable_plugin")
	if err := copyDirectory(filepath.Join("..", "..", "plugins", "behaviour_plugin"), pluginDir); err != nil {
		t.Fatalf("failed to copy plugin: %v", err)
	}
	testsDir := filepath.Join(pluginDir, "tests")
	if err := os.MkdirAll(testsDir, 0o755); err != nil {
		t.Fatalf("failed to create tests dir: %v", err)
	}
	testLua := `
function test_log_merit_succeeds(t)
  local res = t.rpc("log_behaviour_event", {
    pupil_id = "pup_yr7_alice",
    event_type = "MERIT",
    category = "Teamwork",
    points = 3,
    incident_notes = "Helped peer"
  })
  assert(res.ok == true, "expected ok=true")
end
`
	if err := os.WriteFile(filepath.Join(testsDir, "behaviour_test.lua"), []byte(testLua), 0o644); err != nil {
		t.Fatalf("failed to write test lua: %v", err)
	}

	var testOut bytes.Buffer
	runRep, err := RunPluginTests(pluginDir, "stf_tch_7a", &testOut)
	if err != nil {
		t.Fatalf("expected RunPluginTests to pass, got error: %v (out=%s)", err, testOut.String())
	}
	if runRep.Passed != 1 || runRep.Failed != 0 {
		t.Fatalf("unexpected test report: %+v", runRep)
	}
	if !strings.Contains(testOut.String(), "type=rpc") || !strings.Contains(testOut.String(), "sqlite_write_hold_ms=") {
		t.Fatalf("expected structured rpc log output, got: %s", testOut.String())
	}

	outPkg := filepath.Join(tmpDir, "behaviour.mispkg")
	var packOut bytes.Buffer
	sig, err := PackPlugin(pluginDir, outPkg, &packOut)
	if err != nil {
		t.Fatalf("expected PackPlugin to succeed, got: %v", err)
	}
	if sig.PluginID != "uk.org.optimis.behaviour" {
		t.Fatalf("unexpected signature plugin_id: %s", sig.PluginID)
	}
}
