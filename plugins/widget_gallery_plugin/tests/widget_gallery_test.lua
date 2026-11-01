function test_getWidgetGallerySummary_withOwnGroup_returnsActivePupilRosterAndCounts_test(t)
  t.set_persona("stf_tch_7a")
  local res = t.rpc("get_widget_gallery_summary", {
    group_id = "grp_reg_7a",
    academic_year_id = "ay_2026_2027",
    reference_date = "2026-10-27"
  })
  assert(res.ok == true, "expected get_widget_gallery_summary to succeed for own group")
  assert(type(res.data) == "table", "expected data table")
  assert(res.data.group_id == "grp_reg_7a", "expected group_id grp_reg_7a")
  assert(tonumber(res.data.active_pupil_count) >= 1, "expected at least 1 active pupil in grp_reg_7a")
end

function test_getWidgetGallerySummary_withUnassignedGroup_failsFastWithOwnGroupsIsolation_test(t)
  t.set_persona("stf_tch_7a")
  local res = t.rpc("get_widget_gallery_summary", {
    group_id = "grp_reg_7b",
    academic_year_id = "ay_2026_2027",
    reference_date = "2026-10-27"
  })
  assert(res.ok == false, "expected get_widget_gallery_summary to fail for unassigned group grp_reg_7b")
end

function test_getPupilQuickCard_withOwnGroupPupil_returnsPupilSummary_test(t)
  t.set_persona("stf_tch_7a")
  local res = t.rpc("get_pupil_quick_card", {
    pupil_id = "pup_yr7_alice",
    academic_year_id = "ay_2026_2027",
    reference_date = "2026-10-27"
  })
  assert(res.ok == true, "expected get_pupil_quick_card to succeed for pup_yr7_alice")
  assert(res.data.id == "pup_yr7_alice", "expected id pup_yr7_alice")
  assert(res.data.first_name == "Alice", "expected first_name Alice")
end
