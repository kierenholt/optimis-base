package tests

import (
	"context"
	"testing"

	"optimis/internal/host"
)

func TestHostDALBindings_ProductionDomainsReadiness_ExecutesDashboardPupilsGuardiansStaffAndGroups_Test(t *testing.T) {
	ctx := context.Background()

	manifest := `{
  "id": "uk.org.optimis.m1.prod_dal",
  "version": "1.0.0",
  "core_dal_version": "~1.4",
  "endpoints": {
    "probe_production_dal": {
      "scopes": [
        "dal.academic.read:all",
        "dal.academic.write:all",
        "dal.dashboard.read:all",
        "dal.dashboard.write:all",
        "dal.pupils.read:all",
        "dal.pupils.write:all",
        "dal.sen.read:all",
        "dal.sen.write:all",
        "dal.medical.read:all",
        "dal.medical.write:all",
        "dal.guardians.read:all",
        "dal.guardians.write:all",
        "dal.staff.read:admin",
        "dal.staff.write:admin",
        "dal.groups.read:all",
        "dal.groups.write:all",
        "dal.attendance.read:all",
        "dal.attendance.write:all",
        "dal.census.read:all"
      ],
      "mutates": true
    }
  }
}`
	luaSrc := `
function probe_production_dal(ctx, args)
  local ay = ctx.dal.academic.upsert_year({
    academic_year_id = "ay_2027_2028",
    name = "2027/2028",
    status = "FUTURE",
    sequence_number = 99,
    first_day = "2027-09-01",
    last_day = "2028-07-21"
  })
  local years = ctx.dal.academic.list_years()

  local savedRole = ctx.dal.dashboard.save_role_default({
    role_id = "TEACHER",
    layout = { { plug_id = "uk.org.optimis.m1.prod_dal", widget_id = "w1", col = 1, row = 1 } },
    append_to_existing_staff = true
  })
  local roleDef = ctx.dal.dashboard.get_role_default("TEACHER")

  local admitted = ctx.dal.pupils.upsert_pupil_admission({
    upn = "U900001202699",
    uln = "1234567890",
    first_name = "Grace",
    last_name = "Hopper",
    dob = "2014-01-01",
    sex = "F",
    pupil_premium_flag = true,
    nc_year_group = "7",
    academic_year_id = "ay_2026_2027",
    effective_from = "2026-09-01",
    fsm_eligible = true,
    fsm_effective_from = "2026-09-01"
  })
  local promoted = ctx.dal.pupils.upsert_year_enrolment({
    pupil_id = admitted.pupil_id,
    academic_year_id = "ay_2026_2027",
    nc_year_group = "8",
    enrolment_status = "C",
    effective_from = "2026-09-01"
  })
  local fsmBatch = ctx.dal.pupils.list_fsm_periods_batch({ admitted.pupil_id })
  local excl = ctx.dal.pupils.upsert_exclusion({
    pupil_id = admitted.pupil_id,
    academic_year_id = "ay_2026_2027",
    category = "SUSPENSION",
    dfe_reason_code = "PP",
    start_date = "2026-10-01",
    end_date = "2026-10-02",
    sessions_lost = 4,
    authorising_staff_id = args.staff_id
  })
  local exclusions = ctx.dal.pupils.list_exclusions({ pupil_id = admitted.pupil_id })

  ctx.dal.sen.upsert_need_type({ code = "ASD", name = "Autistic Spectrum Disorder", sequence_number = 1 })
  ctx.dal.sen.upsert_sen_record({
    pupil_id = admitted.pupil_id,
    sen_status_code = "K",
    effective_from = "2026-09-01"
  })
  ctx.dal.medical.upsert_condition({
    pupil_id = admitted.pupil_id,
    condition_type = "ALLERGY",
    condition_name = "Peanut Allergy",
    alert_level_id = 1,
    emergency_protocol = "EpiPen immediately"
  })
  local medConds = ctx.dal.medical.list_conditions({ pupil_id = admitted.pupil_id })

  local guardian = ctx.dal.guardians.upsert_guardian_contact({
    pupil_id = admitted.pupil_id,
    first_name = "Walter",
    last_name = "Hopper",
    email = "w.hopper@household.uk",
    phone_primary = "07700900111",
    relationship_code = "FATHER",
    has_parental_responsibility = true,
    contact_priority = 1,
    address_line_1 = "10 Fleet Street",
    town_city = "London",
    postcode = "EC4Y 1AA",
    effective_from = "2026-09-01"
  })
  local households = ctx.dal.guardians.list_households({ pupil_id = admitted.pupil_id })

  local scr = ctx.dal.staff.upsert_scr_record({
    staff_id = args.staff_id,
    trn = "8811223",
    dbs_check_date = "2026-08-15",
    dbs_certificate_number = "001122334455",
    first_aid_qualified = true
  })
  local scrList = ctx.dal.staff.list_scr_register()

  local grp = ctx.dal.groups.upsert_group({
    academic_year_id = "ay_2026_2027",
    group_type = "TEACHING_CLASS",
    code = "8Cs/1",
    name = "Year 8 Computer Science",
    takes_attendance = true,
    effective_from = "2026-09-01"
  })
  local prd = ctx.dal.groups.upsert_timetable_period({
    academic_year_id = "ay_2026_2027",
    cycle_week = 1,
    day_of_week_id = 1,
    period_code = "P2_CS",
    name = "Period 2 CS",
    start_time = "10:00",
    end_time = "11:00",
    period_type = "LESSON",
    maps_to_statutory_session = "AM",
    sequence_number = 2
  })
  local slot = ctx.dal.groups.upsert_timetable_slot({
    timetable_period_id = prd.period_id,
    group_id = grp.group_id,
    staff_id = args.staff_id,
    effective_from = "2026-09-01"
  })
  local slots = ctx.dal.groups.list_timetable_slots({ group_id = grp.group_id })

  ctx.dal.attendance.upsert_code({
    code = "/",
    dfe_code = "/",
    description = "Present (AM)",
    statistical_meaning = "PRESENT",
    physical_presence = "ON_SITE",
    is_active = true
  })
  ctx.dal.attendance.upsert_lesson_batch({
    group_id = grp.group_id,
    timetable_slot_id = slot.slot_id,
    date = "2026-10-12",
    marks = {
      { pupil_id = admitted.pupil_id, attendance_code = "/", minutes_late = 0 }
    }
  })
  local lessonMarks = ctx.dal.attendance.get_group_lesson(grp.group_id, slot.slot_id, "2026-10-12")

  return {
    ok = true,
    data = {
      years_count = #years,
      role_def_count = #roleDef,
      promoted_year = promoted.nc_year_group,
      fsm_count = #(fsmBatch[admitted.pupil_id] or {}),
      exclusions_count = #exclusions,
      med_conds_count = #medConds,
      guardian_rel_id = guardian.relationship_id,
      households_count = #households,
      scr_trn = scr.trn,
      scr_count = #scrList,
      slot_id = slot.slot_id,
      slots_count = #slots,
      lesson_marks_count = #lessonMarks
    }
  }
end
`
	fx := setupMilestone1HostTest(t, manifest, luaSrc)

	caller := host.CallerContext{
		Type:        "USER",
		UserID:      "usr_admin_1",
		StaffID:     fx.TeacherStaffID,
		ActiveRoles: []string{"ADMIN"},
		Scopes: []string{
			"dal.academic.read:all",
			"dal.academic.write:all",
			"dal.dashboard.read:all",
			"dal.dashboard.write:all",
			"dal.pupils.read:all",
			"dal.pupils.write:all",
			"dal.sen.read:all",
			"dal.sen.write:all",
			"dal.medical.read:all",
			"dal.medical.write:all",
			"dal.guardians.read:all",
			"dal.guardians.write:all",
			"dal.staff.read:admin",
			"dal.staff.write:admin",
			"dal.groups.read:all",
			"dal.groups.write:all",
			"dal.attendance.read:all",
			"dal.attendance.write:all",
			"dal.census.read:all",
		},
		RequestID: "req_prod_dal_probe",
	}

	resp, err := fx.Host.ExecuteRPC(ctx, "uk.org.optimis.m1.prod_dal", "probe_production_dal", caller, map[string]any{
		"staff_id": fx.TeacherStaffID,
	})
	if err != nil {
		t.Fatalf("ExecuteRPC probe_production_dal failed: %v", err)
	}
	if !resp.OK {
		t.Fatalf("expected resp.OK=true, got error_code=%s error=%s", resp.ErrorCode, resp.Error)
	}
	dataMap, ok := resp.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected map data, got %T", resp.Data)
	}
	if dataMap["promoted_year"] != "8" {
		t.Fatalf("expected promoted_year=8, got %v", dataMap["promoted_year"])
	}
	if dataMap["exclusions_count"] != float64(1) {
		t.Fatalf("expected exclusions_count=1, got %v", dataMap["exclusions_count"])
	}
	if dataMap["med_conds_count"] != float64(1) {
		t.Fatalf("expected med_conds_count=1, got %v", dataMap["med_conds_count"])
	}
	if dataMap["scr_trn"] != "8811223" {
		t.Fatalf("expected scr_trn=8811223, got %v", dataMap["scr_trn"])
	}
	if dataMap["slots_count"] != float64(1) {
		t.Fatalf("expected slots_count=1, got %v", dataMap["slots_count"])
	}
	if dataMap["lesson_marks_count"] != float64(1) {
		t.Fatalf("expected lesson_marks_count=1, got %v", dataMap["lesson_marks_count"])
	}
}
