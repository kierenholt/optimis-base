package host_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"optimis/internal/dal"
	"optimis/internal/host"
)

func setupCensusPluginTestHost(t *testing.T) (*host.Host, func()) {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "optimis_census_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	pluginsDir, err := filepath.Abs(filepath.Join("..", "..", "plugins"))
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("failed to resolve plugins dir path: %v", err)
	}

	cfg := host.Config{
		DataDirPath:      tempDir,
		SchoolURN:        "100001",
		HostKMSKeyHandle: "kms-census-test-key-001",
		PluginsDir:       pluginsDir,
	}

	h, err := host.NewHost(cfg)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("failed to initialize host: %v", err)
	}

	if err := h.LoadPlugin("test_plugin"); err != nil {
		_ = h.Close()
		os.RemoveAll(tempDir)
		t.Fatalf("failed to load test_plugin: %v", err)
	}
	if err := h.LoadPlugin("dfe_census_plugin"); err != nil {
		_ = h.Close()
		os.RemoveAll(tempDir)
		t.Fatalf("failed to load dfe_census_plugin: %v", err)
	}

	logCursor := h.LogCursor()

	cleanup := func() {
		if t.Failed() {
			logs := h.LogsSince(logCursor)
			t.Logf("--- TAPPED SERVICE LOGS ---\n%s", strings.Join(logs, "\n"))
			t.Logf("--- TAPPED DIAGNOSTIC RING BUFFER ---\n%s", strings.Join(h.Diagnostics(), "\n"))
		}
		_ = h.Close()
		_ = os.RemoveAll(tempDir)
	}

	return h, cleanup
}

func TestCtxXmlBuilderAndXsdValidator_ValidAndMalformedCensusXml_EnforcesDeterministicXmlAndXsdRules_test(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()
	pluginsDir := filepath.Join(tempDir, "plugins")
	pluginDir := filepath.Join(pluginsDir, "xml_test_plugin")
	if err := os.MkdirAll(filepath.Join(pluginDir, "src"), 0o755); err != nil {
		t.Fatalf("failed to create plugin src dir: %v", err)
	}

	manifestJSON := `{
  "id": "uk.org.optimis.xml.test",
  "version": "0.1.0",
  "core_dal_version": "~1.0",
  "egress": { "allowed_domains": [] },
  "endpoints": {
    "build_and_validate": { "scopes": [], "mutates": false },
    "validate_unknown_schema": { "scopes": [], "mutates": false }
  }
}`
	luaSrc := `
function build_and_validate(ctx, args)
  local upn_val = args.upn or "H823400026001"
  local sen_val = args.sen_provision or "K"
  local xml_str = ctx.xml.build({
    tag = "Message",
    attrs = { version = "1.0", schema = "CBDS" },
    children = {
      {
        tag = "Header",
        children = {
          { tag = "Collection", text = "SchoolCensus" },
          { tag = "Term", text = "AUTUMN" },
          { tag = "CensusDate", text = "2026-10-01" },
          {
            tag = "Source",
            children = {
              { tag = "LEA", text = "201" },
              { tag = "Estab", text = "4001" },
              { tag = "URN", text = "100001" },
              { tag = "SchoolName", text = "St Mary's &amp; All Saints <Academy>" }
            }
          }
        }
      },
      {
        tag = "Pupils",
        children = {
          {
            tag = "Pupil",
            children = {
              {
                tag = "PupilIdentifiers",
                children = {
                  { tag = "UPN", text = upn_val },
                  { tag = "ULN", text = "1234567890" },
                  { tag = "Surname", text = "Lovelace" },
                  { tag = "Forename", text = "Ada" },
                  { tag = "DOB", text = "2012-12-10" },
                  { tag = "Sex", text = "F" }
                }
              },
              {
                tag = "PupilCharacteristics",
                children = {
                  { tag = "Ethnicity", text = "WBRI" },
                  { tag = "Language", text = "ENG" },
                  { tag = "SENProvision", text = sen_val },
                  { tag = "LookedAfter", text = "0" },
                  {
                    tag = "FSMPeriods",
                    children = {
                      {
                        tag = "FSMPeriod",
                        children = {
                          { tag = "FSMStartDate", text = "2025-09-01" }
                        }
                      }
                    }
                  }
                }
              },
              {
                tag = "Enrolment",
                children = {
                  { tag = "NCYearActual", text = "10" },
                  { tag = "EnrolmentStatus", text = "C" },
                  { tag = "EntryDate", text = "2026-09-01" }
                }
              },
              {
                tag = "Attendance",
                children = {
                  { tag = "PossibleSessions", text = "40" },
                  { tag = "SessionsPresent", text = "36" },
                  { tag = "AuthorisedAbsenceSessions", text = "2" },
                  { tag = "UnauthorisedAbsenceSessions", text = "2" }
                }
              }
            }
          }
        }
      }
    }
  })

  local res = ctx.xml.validate_xsd(xml_str, "dfe_school_census_v1.xsd")
  return {
    ok = true,
    data = {
      xml = xml_str,
      valid = res.valid,
      errors = res.errors
    }
  }
end

function validate_unknown_schema(ctx, args)
  local res = ctx.xml.validate_xsd("<Message></Message>", "non_existent_schema.xsd")
  return { ok = true, data = res }
end
`
	if err := os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(manifestJSON), 0o644); err != nil {
		t.Fatalf("failed to write manifest.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "src", "init.lua"), []byte(luaSrc), 0o644); err != nil {
		t.Fatalf("failed to write init.lua: %v", err)
	}

	h, err := host.NewHost(host.Config{
		DataDirPath:      filepath.Join(tempDir, "data"),
		SchoolURN:        "100001",
		HostKMSKeyHandle: "kms-xml-test-key-001",
		PluginsDir:       pluginsDir,
	})
	if err != nil {
		t.Fatalf("failed to create host: %v", err)
	}
	if err := h.LoadPlugin("xml_test_plugin"); err != nil {
		_ = h.Close()
		t.Fatalf("failed to load xml_test_plugin: %v", err)
	}

	logCursor := h.LogCursor()
	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("--- TAPPED SERVICE LOGS ---\n%s", strings.Join(h.LogsSince(logCursor), "\n"))
			t.Logf("--- TAPPED DIAGNOSTIC RING BUFFER ---\n%s", strings.Join(h.Diagnostics(), "\n"))
		}
		_ = h.Close()
	})

	caller := host.CallerContext{
		Type:      "USER",
		UserID:    "usr_xml_1",
		StaffID:   "stf_xml_1",
		Scopes:    []string{},
		RequestID: "req_xml_1",
	}

	// 1. Valid DfE CBDS XML passes XSD validation and escapes XML entities deterministically
	validRes, err := h.ExecuteRPC(ctx, "uk.org.optimis.xml.test", "build_and_validate", caller, map[string]any{
		"upn":           "H823400026001",
		"sen_provision": "K",
	})
	if err != nil {
		t.Fatalf("ExecuteRPC build_and_validate failed: %v", err)
	}
	data := validRes.Data.(map[string]any)
	if data["valid"] != true {
		t.Fatalf("expected valid == true, got %v (errors=%v)", data["valid"], data["errors"])
	}
	xmlStr, _ := data["xml"].(string)
	if !strings.HasPrefix(xmlStr, `<?xml version="1.0" encoding="UTF-8"?>`) {
		t.Errorf("expected XML declaration header, got: %s", xmlStr)
	}
	if !strings.Contains(xmlStr, `<Message schema="CBDS" version="1.0">`) {
		t.Errorf("expected sorted attributes on Message tag, got: %s", xmlStr)
	}
	if !strings.Contains(xmlStr, `&lt;Academy&gt;`) {
		t.Errorf("expected escaped XML entities in SchoolName, got: %s", xmlStr)
	}

	// 2. Malformed UPN (not 13-char ^[A-Z][0-9]{12}$) fails XSD validation
	badUpnRes, err := h.ExecuteRPC(ctx, "uk.org.optimis.xml.test", "build_and_validate", caller, map[string]any{
		"upn":           "INVALID_UPN",
		"sen_provision": "K",
	})
	if err != nil {
		t.Fatalf("ExecuteRPC build_and_validate with bad UPN failed: %v", err)
	}
	badUpnData := badUpnRes.Data.(map[string]any)
	if badUpnData["valid"] != false {
		t.Fatalf("expected valid == false for malformed UPN, got true")
	}

	// 3. Invalid SENProvision code (must be N, K, or E) fails XSD validation
	badSenRes, err := h.ExecuteRPC(ctx, "uk.org.optimis.xml.test", "build_and_validate", caller, map[string]any{
		"upn":           "H823400026001",
		"sen_provision": "INVALID_SEN",
	})
	if err != nil {
		t.Fatalf("ExecuteRPC build_and_validate with bad SEN failed: %v", err)
	}
	badSenData := badSenRes.Data.(map[string]any)
	if badSenData["valid"] != false {
		t.Fatalf("expected valid == false for invalid SENProvision, got true")
	}

	// 4. Unknown XSD schema name fails fast
	if _, err := h.ExecuteRPC(ctx, "uk.org.optimis.xml.test", "validate_unknown_schema", caller, map[string]any{}); err == nil {
		t.Fatalf("expected validate_unknown_schema to fail fast for unknown XSD schema name, got nil")
	}
}

func TestCoreDalCensusStatutoryRollup_TemporalEnrolmentsSenFsmLacAndAmPmSessions_ExcludesOffRollAndComputesRollups_test(t *testing.T) {
	ctx := context.Background()
	h, cleanup := setupCensusPluginTestHost(t)
	t.Cleanup(cleanup)

	seeded, err := h.DAL().SeedCensusTestFixtures(ctx, dal.CensusTestFixtureInput{
		AcademicYearID:      "ay_2026_2027",
		CensusDate:          "2026-10-01",
		TermCode:            "AUTUMN",
		AttendanceStartDate: "2026-09-01",
		AttendanceEndDate:   "2026-10-01",
	})
	if err != nil {
		t.Fatalf("SeedCensusTestFixtures failed: %v", err)
	}
	t.Cleanup(func() {
		_ = h.DAL().DeleteCensusTestFixtures(context.Background(), seeded.FixtureID)
	})

	// 1. ScopeOwnGroups must be rejected for statutory school-wide census rollups
	_, err = h.DAL().GetStatutoryCensusRollup(ctx, dal.StatutoryCensusRollupInput{
		AcademicYearID:      seeded.AcademicYearID,
		CensusDate:          seeded.CensusDate,
		TermCode:            seeded.TermCode,
		AttendanceStartDate: "2026-09-01",
		AttendanceEndDate:   "2026-10-01",
		Scope:               dal.ScopeOwnGroups,
	})
	if err == nil {
		t.Fatalf("expected GetStatutoryCensusRollup with ScopeOwnGroups to fail fast, got nil")
	}

	// 2. ScopeAll computes statutory school-wide rollups and excludes off-roll pupils
	rollup, err := h.DAL().GetStatutoryCensusRollup(ctx, dal.StatutoryCensusRollupInput{
		AcademicYearID:      seeded.AcademicYearID,
		CensusDate:          seeded.CensusDate,
		TermCode:            seeded.TermCode,
		AttendanceStartDate: "2026-09-01",
		AttendanceEndDate:   "2026-10-01",
		Scope:               dal.ScopeAll,
	})
	if err != nil {
		t.Fatalf("GetStatutoryCensusRollup failed: %v", err)
	}

	if rollup.SchoolURN != "100001" {
		t.Errorf("expected SchoolURN 100001, got %s", rollup.SchoolURN)
	}
	if rollup.TermCode != "AUTUMN" {
		t.Errorf("expected TermCode AUTUMN, got %s", rollup.TermCode)
	}

	foundSenK := false
	foundEhcpLac := false
	foundNoSen := false
	for _, p := range rollup.Pupils {
		if p.PupilID == seeded.OffRollPupilID {
			t.Fatalf("off-roll pupil %s (effective_to < census_date) must be excluded from census rollup", seeded.OffRollPupilID)
		}
		if p.PupilID == seeded.PupilSenKFsmID {
			foundSenK = true
			if p.SENProvisionCode != "K" || p.PrimarySENNeedCode != "SPLD" {
				t.Errorf("expected SEN K / SPLD for %s, got %s / %s", p.PupilID, p.SENProvisionCode, p.PrimarySENNeedCode)
			}
			if !p.FSMEligible || len(p.FSMPeriods) == 0 {
				t.Errorf("expected FSM eligible with FSMPeriods for %s", p.PupilID)
			}
			if p.AttendanceSessions.PossibleSessions <= 0 || p.AttendanceSessions.PresentSessions <= 0 {
				t.Errorf("expected positive AM/PM attendance session counts for %s, got %+v", p.PupilID, p.AttendanceSessions)
			}
		}
		if p.PupilID == seeded.PupilEhcpLacID {
			foundEhcpLac = true
			if p.SENProvisionCode != "E" || p.PrimarySENNeedCode != "ASD" {
				t.Errorf("expected SEN E / ASD for %s, got %s / %s", p.PupilID, p.SENProvisionCode, p.PrimarySENNeedCode)
			}
			if !p.LookedAfterFlag || p.PostLookedAfterCode != "A" {
				t.Errorf("expected LookedAfterFlag=true and PostLookedAfterCode='A' for %s, got %v / %s", p.PupilID, p.LookedAfterFlag, p.PostLookedAfterCode)
			}
		}
		if p.PupilID == seeded.PupilNoSenID {
			foundNoSen = true
			if p.SENProvisionCode != "N" {
				t.Errorf("expected SEN N for %s, got %s", p.PupilID, p.SENProvisionCode)
			}
		}
	}

	if !foundSenK || !foundEhcpLac || !foundNoSen {
		t.Fatalf("missing seeded on-roll pupils in rollup: senK=%v ehcpLac=%v noSen=%v", foundSenK, foundEhcpLac, foundNoSen)
	}
	if rollup.Summary.OnRollTotal < 3 {
		t.Errorf("expected OnRollTotal >= 3, got %d", rollup.Summary.OnRollTotal)
	}
	if rollup.Summary.SENCounts["K"] < 1 || rollup.Summary.SENCounts["E"] < 1 || rollup.Summary.SENCounts["N"] < 1 {
		t.Errorf("unexpected SENCounts breakdown: %+v", rollup.Summary.SENCounts)
	}
	if rollup.Summary.FSMEligibleCount < 2 {
		t.Errorf("expected FSMEligibleCount >= 2, got %d", rollup.Summary.FSMEligibleCount)
	}
	if rollup.Summary.LACCount < 1 {
		t.Errorf("expected LACCount >= 1, got %d", rollup.Summary.LACCount)
	}
}

func TestDfeCensusPlugin_GenerateSubmitWithFeatureFlagAndGdprHooks_EnforcesMockFlagScopesAndZeroOrphanPurge_test(t *testing.T) {
	ctx := context.Background()
	h, cleanup := setupCensusPluginTestHost(t)
	t.Cleanup(cleanup)

	seeded, err := h.DAL().SeedCensusTestFixtures(ctx, dal.CensusTestFixtureInput{
		AcademicYearID:      "ay_2026_2027",
		CensusDate:          "2026-10-01",
		TermCode:            "AUTUMN",
		AttendanceStartDate: "2026-09-01",
		AttendanceEndDate:   "2026-10-01",
	})
	if err != nil {
		t.Fatalf("SeedCensusTestFixtures failed: %v", err)
	}
	t.Cleanup(func() {
		_ = h.DAL().DeleteCensusTestFixtures(context.Background(), seeded.FixtureID)
	})

	teacherCaller := host.CallerContext{
		Type:        "USER",
		UserID:      "usr_teacher_census",
		StaffID:     seeded.TeacherStaffID,
		ActiveRoles: []string{"TEACHER"},
		Scopes:      []string{"dal.pupils.read:own_groups"},
		RequestID:   "req_census_teacher",
	}

	adminCaller := host.CallerContext{
		Type:        "USER",
		UserID:      "usr_admin_census",
		StaffID:     seeded.AdminStaffID,
		ActiveRoles: []string{"ADMIN"},
		Scopes: []string{
			"dal.census.read:all",
			"dal.pupils.read:all",
			"dal.pupils.write:all",
			"dal.staff.read:admin",
			"dal.staff.write:admin",
		},
		RequestID: "req_census_admin",
	}

	// 1. Teacher without dal.census.read:all must be rejected
	if _, err := h.ExecuteRPC(ctx, "uk.gov.dfe.census", "generate_and_validate_census_return", teacherCaller, map[string]any{
		"academic_year_id": seeded.AcademicYearID,
		"census_date":      seeded.CensusDate,
		"term_code":        seeded.TermCode,
	}); err == nil || !strings.Contains(err.Error(), "caller lacks required DAL scopes") {
		t.Fatalf("expected teacher caller to fail with missing DAL scopes, got: %v", err)
	}

	// 2. Enable Mock DfE COLLECT service flag and generate + submit census return
	h.ConfigureMockDfeCensusService(true)
	h.ClearMockDfeCensusSubmissions()

	genRes, err := h.ExecuteRPC(ctx, "uk.gov.dfe.census", "generate_and_validate_census_return", adminCaller, map[string]any{
		"academic_year_id":      seeded.AcademicYearID,
		"census_date":           seeded.CensusDate,
		"term_code":             seeded.TermCode,
		"attendance_start_date": "2026-09-01",
		"attendance_end_date":   "2026-10-01",
	})
	if err != nil {
		t.Fatalf("generate_and_validate_census_return failed: %v", err)
	}
	if !genRes.OK {
		t.Fatalf("expected genRes.OK == true, got error_code=%s", genRes.ErrorCode)
	}
	genData := genRes.Data.(map[string]any)
	returnID, _ := genData["return_id"].(string)
	if returnID == "" {
		t.Fatalf("expected non-empty return_id")
	}

	submitRes, err := h.ExecuteRPC(ctx, "uk.gov.dfe.census", "submit_census_return", adminCaller, map[string]any{
		"return_id": returnID,
	})
	if err != nil {
		t.Fatalf("submit_census_return with mock service enabled failed: %v", err)
	}
	if !submitRes.OK {
		t.Fatalf("expected submitRes.OK == true, got error_code=%s", submitRes.ErrorCode)
	}
	submissions := h.ListMockDfeCensusSubmissions()
	if len(submissions) != 1 {
		t.Fatalf("expected 1 mock COLLECT submission, got %d", len(submissions))
	}
	if submissions[0].Status != "COLLECT_ACCEPTED" {
		t.Errorf("expected status COLLECT_ACCEPTED, got %s", submissions[0].Status)
	}

	// 3. Disable Mock DfE COLLECT service flag and verify fail-fast rejection
	h.ConfigureMockDfeCensusService(false)
	_, err = h.ExecuteRPC(ctx, "uk.gov.dfe.census", "submit_census_return", adminCaller, map[string]any{
		"return_id": returnID,
	})
	if err == nil || !strings.Contains(err.Error(), "MOCK_DFE_CENSUS_SERVICE_DISABLED") {
		t.Fatalf("expected submit_census_return with mock service disabled to fail with MOCK_DFE_CENSUS_SERVICE_DISABLED, got: %v", err)
	}

	// 4. Verify UK GDPR SAR export and retention purge hooks on uk.gov.dfe.census
	sarBundle, err := h.ExportPupilSAR(ctx, adminCaller, seeded.PupilSenKFsmID)
	if err != nil {
		t.Fatalf("ExportPupilSAR failed: %v", err)
	}
	foundCensusSection := false
	for _, sec := range sarBundle.Sections {
		if sec.SourceID == "uk.gov.dfe.census" && sec.Title == "DfE Statutory School Census Snapshots" {
			foundCensusSection = true
			if len(sec.Records) == 0 {
				t.Errorf("expected census snapshot records in SAR export for %s", seeded.PupilSenKFsmID)
			}
		}
	}
	if !foundCensusSection {
		t.Fatalf("expected SAR bundle to include section from uk.gov.dfe.census")
	}

	purgeRes, err := h.ExecuteRPC(ctx, "uk.gov.dfe.census", "purge_pupil_data", adminCaller, map[string]any{
		"pupil_id": seeded.PupilSenKFsmID,
	})
	if err != nil || !purgeRes.OK {
		t.Fatalf("purge_pupil_data on uk.gov.dfe.census failed: err=%v res=%+v", err, purgeRes)
	}

	var remainingSnapshots int
	if err := h.DAL().ReadDB().QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM plugin_dfe_census.census_pupil_snapshots WHERE pupil_id = ?`,
		seeded.PupilSenKFsmID,
	).Scan(&remainingSnapshots); err != nil {
		t.Fatalf("failed querying plugin_dfe_census.census_pupil_snapshots: %v", err)
	}
	if remainingSnapshots != 0 {
		t.Errorf("expected 0 remaining census_pupil_snapshots for %s after purge, got %d", seeded.PupilSenKFsmID, remainingSnapshots)
	}
}
