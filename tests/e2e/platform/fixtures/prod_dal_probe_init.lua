function exercise_all_new_dal_domains(ctx, args)
  -- 0. Academic Calendar & Lookups DAL
  local ayUpserted = ctx.dal.academic.upsert_year({
    academic_year_id = "ay_2027_2028",
    name = "2027/2028",
    status = "FUTURE",
    sequence_number = 99,
    first_day = "2027-09-01",
    last_day = "2028-07-21"
  })
  local years = ctx.dal.academic.list_years()
  local termUpserted = ctx.dal.academic.upsert_term({
    academic_year_id = "ay_2026_2027",
    term_code = "AUT",
    name = "Autumn Term",
    sequence_number = 1,
    first_day = "2026-09-01",
    last_day = "2026-12-18",
    half_term_start = "2026-10-26",
    half_term_end = "2026-10-30"
  })
  local terms = ctx.dal.academic.list_terms({ academic_year_id = "ay_2026_2027" })
  local dayUpserted = ctx.dal.academic.upsert_day_of_week({
    id = 1,
    name = "Monday",
    name_short = "Mon",
    is_school_day = true,
    school_open_time = "08:00",
    am_register_time = "08:45",
    pm_register_time = "13:30",
    school_close_time = "16:00"
  })
  local daysOfWeek = ctx.dal.academic.list_days_of_week()
  local specDay = ctx.dal.academic.upsert_special_day({
    academic_year_id = "ay_2026_2027",
    date = "2026-09-02",
    day_type = "INSET_STAFF_ONLY",
    StatutoryAttendanceCode = "#",
    name = "Staff Training Day",
    applies_to_phase = "ALL"
  })
  local specialDays = ctx.dal.academic.list_special_days({ academic_year_id = "ay_2026_2027" })
  local alertLevels = ctx.dal.academic.list_alert_levels()

  -- 1. Dashboard DAL (Admin Dashboard Editor Plugin)
  local widgets = ctx.dal.dashboard.list_available_widgets()
  local savedRole = ctx.dal.dashboard.save_role_default({
    role_id = "TEACHER",
    layout = { { plug_id = "uk.org.optimis.test.prod_dal_probe", widget_id = "w1", col = 1, row = 1 } },
    append_to_existing_staff = true
  })
  local roleDef = ctx.dal.dashboard.get_role_default("TEACHER")
  local savedStaff = ctx.dal.dashboard.save_staff_layout({
    staff_id = "stf_tch_7a",
    column_count = 5,
    layout = { { plug_id = "uk.org.optimis.test.prod_dal_probe", widget_id = "w1", col = 2, row = 1 } }
  })
  local staffLayout = ctx.dal.dashboard.get_staff_layout("stf_tch_7a")
  local resetStaff = ctx.dal.dashboard.reset_staff_layout("stf_tch_7a")

  -- 2. Pupils DAL (Full CBDS, Admissions, Year Promotion, FSM, Exclusions, AP, Post-16 & Pastoral)
  local admitted = ctx.dal.pupils.upsert_pupil_admission({
    upn = "U100001202699",
    uln = "9876543210",
    first_name = " Clara ",
    last_name = "Barton",
    dob = "2014-05-12",
    sex = "F",
    ethnicity_code = "WBRI",
    first_language_code = "ENG",
    eal_flag = false,
    pupil_premium_flag = true,
    service_child_flag = true,
    looked_after_flag = false,
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
    ey_funded_hours = 15.0,
    effective_from = "2026-09-01"
  })
  local fetchedPupil = ctx.dal.pupils.get_by_id(admitted.pupil_id)
  local fsmUpserted = ctx.dal.pupils.upsert_fsm_period({
    pupil_id = admitted.pupil_id,
    fsm_type = "STATUTORY_FSM",
    ever6_fsm_flag = true,
    effective_from = "2026-09-01"
  })
  local fsmBatch = ctx.dal.pupils.list_fsm_periods_batch({ admitted.pupil_id })
  local excl = ctx.dal.pupils.upsert_exclusion({
    pupil_id = admitted.pupil_id,
    academic_year_id = "ay_2026_2027",
    category = "SUSPENSION",
    dfe_reason_code = "DB",
    start_date = "2026-10-05",
    end_date = "2026-10-06",
    sessions_lost = 4,
    authorising_staff_id = "stf_admin_01"
  })
  local exclusions = ctx.dal.pupils.list_exclusions({ pupil_id = admitted.pupil_id })
  local ap = ctx.dal.pupils.upsert_alternative_provision({
    pupil_id = admitted.pupil_id,
    academic_year_id = "ay_2026_2027",
    provider_urn = "139999",
    reason_code = "B",
    sessions_per_week = 4,
    effective_from = "2026-09-15"
  })
  local aps = ctx.dal.pupils.list_alternative_provisions({ pupil_id = admitted.pupil_id })
  local aim = ctx.dal.pupils.upsert_post16_learning_aim({
    pupil_id = admitted.pupil_id,
    academic_year_id = "ay_2026_2027",
    qan_code = "6031164X",
    discount_code = "FK2B",
    subject_title = "A Level Mathematics",
    planned_glh = 360,
    start_date = "2026-09-01",
    planned_end_date = "2028-06-30",
    completion_status = "1",
    outcome = "8"
  })
  local aims = ctx.dal.pupils.list_post16_learning_aims({ pupil_id = admitted.pupil_id })
  local pNote = ctx.dal.pupils.upsert_pastoral_note({
    pupil_id = admitted.pupil_id,
    recorded_by_staff_id = "stf_tch_7a",
    category = "ACHIEVEMENT",
    alert_level_id = 4,
    summary = "Excellent settling-in week"
  })
  local pNotes = ctx.dal.pupils.list_pastoral_notes(admitted.pupil_id)

  -- 3. SEN & Medical Write DAL
  local needType = ctx.dal.sen.upsert_need_type({
    code = "SPLD",
    name = "Specific Learning Difficulty",
    sequence_number = 1
  })
  local needTypes = ctx.dal.sen.list_need_types()
  local senRec = ctx.dal.sen.upsert_sen_record({
    pupil_id = admitted.pupil_id,
    sen_status_code = "K",
    key_worker_staff_id = "stf_senco_01",
    exam_access_extra_time_pct = 25,
    effective_from = "2026-09-01"
  })
  local senNeed = ctx.dal.sen.upsert_sen_need({
    pupil_id = admitted.pupil_id,
    sen_need_type_code = "SPLD",
    ranking = 1,
    effective_from = "2026-09-01"
  })
  local senAsst = ctx.dal.sen.upsert_sen_assistant({
    pupil_id = admitted.pupil_id,
    staff_id = "stf_support_01",
    support_hours_per_week = 10.5,
    effective_from = "2026-09-01"
  })
  local medCond = ctx.dal.medical.upsert_condition({
    pupil_id = admitted.pupil_id,
    condition_type = "ASTHMA",
    condition_name = "Mild Asthma",
    alert_level_id = 3,
    emergency_protocol = "Blue inhaler 2 puffs",
    medication_held_in_school = true,
    medication_location = "Medical Room Cabinet A",
    medication_expiry_date = "2027-09-01",
    ihp_review_date = "2027-03-01"
  })
  local medConds = ctx.dal.medical.list_conditions({ pupil_id = admitted.pupil_id })

  -- 4. Guardians, Households & Siblings DAL
  local sibling = ctx.dal.pupils.upsert_pupil_admission({
    upn = "U100001202698",
    first_name = "Leo",
    last_name = "Barton",
    dob = "2016-03-20",
    sex = "M",
    nc_year_group = "5",
    academic_year_id = "ay_2026_2027",
    effective_from = "2026-09-01"
  })
  local upsertedGuardian = ctx.dal.guardians.upsert_guardian_contact({
    pupil_id = admitted.pupil_id,
    guardian_person_id = "per_gdn_stephen_barton",
    household_id = "hh_barton_family",
    first_name = "Stephen",
    last_name = "Barton",
    email = "s.barton@household.uk",
    phone_primary = "07700900888",
    relationship_code = "FATHER",
    has_parental_responsibility = true,
    contact_priority = 1,
    address_line_1 = "42 High Street",
    town_city = "London",
    postcode = "SW1A 1AA",
    effective_from = "2026-09-01"
  })
  ctx.dal.guardians.upsert_guardian_contact({
    pupil_id = sibling.pupil_id,
    guardian_person_id = "per_gdn_stephen_barton",
    household_id = "hh_barton_family",
    first_name = "Stephen",
    last_name = "Barton",
    email = "s.barton@household.uk",
    phone_primary = "07700900888",
    relationship_code = "FATHER",
    has_parental_responsibility = true,
    contact_priority = 1,
    address_line_1 = "42 High Street",
    town_city = "London",
    postcode = "SW1A 1AA",
    effective_from = "2026-09-01"
  })
  local guardians = ctx.dal.guardians.list_for_pupil(admitted.pupil_id, { require_parental_responsibility = true })
  local siblings = ctx.dal.guardians.list_siblings_for_pupil(admitted.pupil_id)
  local households = ctx.dal.guardians.list_households({ pupil_id = admitted.pupil_id })

  -- 5. Staff HR / SCR / SWC / Cover Assignments DAL
  local supplyStaff = ctx.dal.staff.upsert_staff_member({
    staff_id = "stf_supply_probe",
    staff_code = "SUP99",
    first_name = "Arthur",
    last_name = "Pendelton",
    email = "a.pendelton@supply.uk",
    staff_type = "SUPPLY_AGENCY",
    job_title = "Supply Teacher",
    role_id = "TEACHER",
    qts_flag = true,
    hire_date = "2026-09-01"
  })
  local allStaff = ctx.dal.staff.list_all({ include_leavers = true })
  local roles = ctx.dal.staff.list_roles()
  local scrUpdated = ctx.dal.staff.upsert_scr_record({
    staff_id = "stf_tch_7a",
    trn = "7912345",
    dbs_check_date = "2026-08-01",
    dbs_certificate_number = "001999888777",
    prohibition_check_date = "2026-08-01",
    right_to_work_check_date = "2026-08-01",
    first_aid_qualified = true,
    first_aid_expiry_date = "2029-08-01"
  })
  local scrList = ctx.dal.staff.list_scr_register()
  local contract = ctx.dal.staff.upsert_contract({
    staff_id = "stf_tch_7a",
    contract_type = "PERMANENT",
    swc_post_code = "TCH",
    job_title = "Teacher of Maths",
    pay_scale = "UPS",
    pay_spine_point = "U1",
    fte = 1.0,
    hours_per_week = 32.5,
    weeks_per_year = 52.0,
    effective_from = "2026-09-01"
  })
  local contracts = ctx.dal.staff.list_contracts({ staff_id = "stf_tch_7a" })
  local absence = ctx.dal.staff.upsert_absence({
    staff_id = "stf_tch_7a",
    academic_year_id = "ay_2026_2027",
    absence_category_code = "TRN",
    reason_detail = "NPQ Leadership Day",
    status = "APPROVED",
    cover_required = true,
    start_date = "2026-10-15",
    end_date = "2026-10-15",
    working_days_lost = 1.0
  })
  local absences = ctx.dal.staff.list_absences({ staff_id = "stf_tch_7a" })

  -- 6. Groups, Houses, Rooms, Courses & Timetable Builder DAL
  local house = ctx.dal.groups.upsert_house({
    code = "DAR",
    name = "Darwin House",
    colour_hex = "#16A34A",
    head_of_house_staff_id = "stf_tch_7a"
  })
  local houses = ctx.dal.groups.list_houses()
  local dept = ctx.dal.groups.upsert_department({
    name = "Computer Science",
    name_short = "CSC",
    type = "LEARNING_AREA",
    head_of_department_staff_id = "stf_hod_maths"
  })
  local room = ctx.dal.groups.upsert_room({
    code = "IT01",
    name = "IT Suite 1",
    room_type = "IT_SUITE",
    capacity = 30,
    is_active = true,
    phone_extension = "2010"
  })
  local yg = ctx.dal.groups.upsert_year_group({
    nc_code = "12",
    name = "Year 12",
    phase = "SIXTH_FORM",
    key_stage = "KS5",
    sequence_number = 12,
    head_of_year_staff_id = "stf_hod_maths"
  })
  local course = ctx.dal.groups.upsert_course({
    academic_year_id = "ay_2026_2027",
    department_id = dept.id,
    code = "KS3-CSC",
    name = "KS3 Computer Science",
    dfe_subject_code = "I100"
  })
  local createdGroup = ctx.dal.groups.upsert_group({
    academic_year_id = "ay_2026_2027",
    group_type = "TEACHING_CLASS",
    code = "8Sc/Z1",
    name = "Year 8 Science Set 1",
    course_id = course.id,
    default_room_id = room.id,
    takes_attendance = true,
    effective_from = "2026-09-01"
  })
  ctx.dal.groups.upsert_pupil_membership({
    group_id = createdGroup.group_id,
    pupil_id = admitted.pupil_id,
    academic_year_id = "ay_2026_2027",
    effective_from = "2026-09-01"
  })
  local memberships = ctx.dal.groups.list_pupil_memberships({ group_id = createdGroup.group_id })
  local createdPeriod = ctx.dal.groups.upsert_timetable_period({
    academic_year_id = "ay_2026_2027",
    cycle_week = 1,
    day_of_week_id = 1,
    period_code = "P1_PROBE",
    name = "Period 1 Probe",
    start_time = "09:00",
    end_time = "10:00",
    period_type = "LESSON",
    maps_to_statutory_session = "AM",
    sequence_number = 10
  })
  local createdSlot = ctx.dal.groups.upsert_timetable_slot({
    timetable_period_id = createdPeriod.period_id,
    group_id = createdGroup.group_id,
    staff_id = "stf_tch_7a",
    room_id = room.id,
    effective_from = "2026-09-01"
  })
  local cover = ctx.dal.staff.upsert_cover_assignment({
    timetable_slot_id = createdSlot.slot_id,
    date = "2026-10-15",
    absent_staff_id = "stf_tch_7a",
    staff_absence_id = absence.absence_id,
    covering_staff_id = supplyStaff.staff_id,
    room_id = room.id,
    cover_notes = "Complete worksheet 4B",
    assigned_by_staff_id = "stf_admin_01"
  })
  local covers = ctx.dal.staff.list_cover_assignments({ date = "2026-10-15" })
  local pupilSlots = ctx.dal.groups.get_pupil_timetable_slots(admitted.pupil_id, { cycle_week = 1, day_of_week_id = 1 })
  local periods = ctx.dal.groups.list_timetable_periods({ cycle_week = 1, day_of_week_id = 1 })
  local slots = ctx.dal.groups.list_timetable_slots({ group_id = createdGroup.group_id })
  local rooms = ctx.dal.groups.list_rooms()
  local courses = ctx.dal.groups.list_courses()
  local depts = ctx.dal.groups.list_departments()
  local yearGroups = ctx.dal.groups.list_year_groups()

  -- 7. Attendance Codes, Lesson Marks, Session Marks & Register Submissions DAL
  local attCode = ctx.dal.attendance.upsert_code({
    code = "/",
    dfe_code = "/",
    description = "Present (AM)",
    statistical_meaning = "PRESENT",
    physical_presence = "ON_SITE",
    is_active = true
  })
  local attCodes = ctx.dal.attendance.list_codes()
  local lessonUpsert = ctx.dal.attendance.upsert_lesson_batch({
    group_id = createdGroup.group_id,
    timetable_slot_id = createdSlot.slot_id,
    date = "2026-10-12",
    marks = {
      {
        pupil_id = admitted.pupil_id,
        attendance_code = "/",
        minutes_late = 0,
        comment = "On time"
      }
    }
  })
  local lessonMarks = ctx.dal.attendance.get_group_lesson(createdGroup.group_id, createdSlot.slot_id, "2026-10-12")
  local sessionMarks = ctx.dal.attendance.list_session_marks({ pupil_id = admitted.pupil_id })
  local regSubs = ctx.dal.attendance.list_register_submissions({ group_id = createdGroup.group_id })

  -- 8. Expanded Statutory Census Rollup
  local rollup = ctx.dal.census.get_statutory_rollup({
    academic_year_id = "ay_2026_2027",
    reference_date = "2026-10-15"
  })

  return {
    ok = true,
    data = {
      years_count = #years,
      terms_count = #terms,
      days_count = #daysOfWeek,
      special_days_count = #specialDays,
      alert_levels_count = #alertLevels,
      widgets_type = type(widgets),
      role_def_count = #roleDef,
      staff_override_before_reset = staffLayout.has_override,
      reset_ok = resetStaff.reset,
      admitted_pupil_id = admitted.pupil_id,
      pupil_uln = fetchedPupil.uln,
      pupil_pp = fetchedPupil.pupil_premium_flag,
      promoted_year = promoted.nc_year_group,
      fsm_count = #(fsmBatch[admitted.pupil_id] or {}),
      exclusions_count = #exclusions,
      aps_count = #aps,
      aims_count = #aims,
      pastoral_count = #pNotes,
      need_types_count = #needTypes,
      med_conds_count = #medConds,
      guardian_count = #guardians,
      siblings_count = #siblings,
      households_count = #households,
      all_staff_count = #allStaff,
      roles_count = #roles,
      scr_count = #scrList,
      contract_id = contract.contract_id,
      contracts_count = #contracts,
      absence_id = absence.absence_id,
      absences_count = #absences,
      covers_count = #covers,
      houses_count = #houses,
      memberships_count = #memberships,
      pupil_slots_count = #pupilSlots,
      group_id = createdGroup.group_id,
      period_id = createdPeriod.period_id,
      slot_id = createdSlot.slot_id,
      periods_count = #periods,
      slots_count = #slots,
      att_codes_count = #attCodes,
      lesson_marks_count = #lessonMarks,
      reg_subs_count = #regSubs,
      rollup_exclusions_count = rollup.summary.exclusions_count
    }
  }
end
