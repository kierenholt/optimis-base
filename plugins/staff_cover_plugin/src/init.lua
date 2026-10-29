local function trim(s)
  if type(s) ~= "string" then
    return ""
  end
  return (s:gsub("^%s+", ""):gsub("%s+$", ""))
end

local function resolve_school_defaults(ctx)
  local school = (ctx and ctx.school) or {}
  return {
    urn = trim(school.urn),
    academic_year_id = trim(school.academic_year_id),
    reference_date = trim(school.reference_date)
  }
end

local function build_scope_opts(ctx, args, fallback_date, strict_own_groups)
  local school_defaults = resolve_school_defaults(ctx)
  local opts = {
    strict_own_groups = strict_own_groups == true
  }
  if type(args) == "table" then
    if trim(args.academic_year_id) ~= "" then
      opts.academic_year_id = trim(args.academic_year_id)
    elseif school_defaults.academic_year_id ~= "" then
      opts.academic_year_id = school_defaults.academic_year_id
    end

    if trim(args.reference_date) ~= "" then
      opts.reference_date = trim(args.reference_date)
    elseif trim(args.date) ~= "" then
      opts.reference_date = trim(args.date)
    elseif trim(fallback_date) ~= "" then
      opts.reference_date = trim(fallback_date)
    elseif school_defaults.reference_date ~= "" then
      opts.reference_date = school_defaults.reference_date
    end
  end
  return opts
end

local function validate_statutory_session(session)
  local s = trim(session)
  if s ~= "AM" and s ~= "PM" then
    error("invalid statutory session: must be AM or PM (got '" .. tostring(session) .. "')")
  end
  return s
end

function load_absent_teacher_timetable(ctx, args)
  if type(args) ~= "table" then
    error("missing mandatory arguments table for load_absent_teacher_timetable")
  end

  local absent_staff_id = trim(args.absent_staff_id)
  if absent_staff_id == "" then
    absent_staff_id = trim(args.staff_id)
  end
  if absent_staff_id == "" then
    error("missing mandatory parameter: absent_staff_id")
  end

  local cycle_week = tonumber(args.cycle_week) or 1
  local day_of_week_id = tonumber(args.day_of_week_id) or 1

  local opts = build_scope_opts(ctx, args, args.date, false)
  local slots = ctx.dal.groups.get_staff_timetable_slots(
    absent_staff_id,
    cycle_week,
    day_of_week_id,
    opts
  )

  return {
    ok = true,
    data = {
      absent_staff_id = absent_staff_id,
      cycle_week = cycle_week,
      day_of_week_id = day_of_week_id,
      academic_year_id = opts.academic_year_id or "",
      reference_date = opts.reference_date or "",
      slots = slots
    }
  }
end

function record_absence_and_assign_cover(ctx, args)
  if type(args) ~= "table" then
    error("missing mandatory arguments table for record_absence_and_assign_cover")
  end

  local absent_staff_id = trim(args.absent_staff_id)
  if absent_staff_id == "" then
    error("missing mandatory parameter: absent_staff_id")
  end

  local cover_staff_id = trim(args.cover_staff_id)
  if cover_staff_id == "" then
    error("missing mandatory parameter: cover_staff_id")
  end

  local group_id = trim(args.group_id)
  if group_id == "" then
    error("missing mandatory parameter: group_id")
  end

  local slot_id = trim(args.slot_id)
  if slot_id == "" then
    error("missing mandatory parameter: slot_id")
  end

  local school_defaults = resolve_school_defaults(ctx)
  local cover_date = trim(args.date)
  if cover_date == "" then
    cover_date = trim(args.reference_date)
  end
  if cover_date == "" then
    cover_date = school_defaults.reference_date
  end
  if cover_date == "" then
    error("missing mandatory parameter: date")
  end

  local academic_year_id = trim(args.academic_year_id)
  if academic_year_id == "" then
    academic_year_id = school_defaults.academic_year_id
  end
  if academic_year_id == "" then
    error("missing mandatory parameter: academic_year_id")
  end

  local absence_reason = trim(args.absence_reason)
  local cover_notes = trim(args.cover_notes)
  local assigned_role = trim(args.role)
  if assigned_role == "" then
    assigned_role = "CO_TEACHER"
  end

  local session = trim(args.session)
  if session == "" then
    session = "AM"
  end
  session = validate_statutory_session(session)

  -- Grant temporal :own_groups access to Cover Teacher via Core DAL upsert_staff_assignment
  local gsa_res = ctx.dal.groups.upsert_staff_assignment(
    {
      group_id = group_id,
      staff_id = cover_staff_id,
      role = assigned_role,
      academic_year_id = academic_year_id,
      effective_from = cover_date,
      effective_to = cover_date
    },
    {
      academic_year_id = academic_year_id,
      reference_date = cover_date
    }
  )

  local cover_id = "cov_" .. ctx.crypto.uuid()
  local created_at = ctx.clock.now_utc()
  local caller_staff_id = (ctx.caller and trim(ctx.caller.staff_id)) or ""

  ctx.plugin_db.exec(
    [[INSERT INTO plugin_cover.cover_absences_and_assignments (
        id, absent_staff_id, cover_staff_id, group_id, slot_id,
        academic_year_id, cover_date, session, assigned_role,
        staff_assignment_id, absence_reason, cover_notes,
        assigned_by_staff_id, created_at
      ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)]],
    {
      cover_id,
      absent_staff_id,
      cover_staff_id,
      group_id,
      slot_id,
      academic_year_id,
      cover_date,
      session,
      gsa_res.role or assigned_role,
      gsa_res.assignment_id or "",
      absence_reason,
      cover_notes,
      caller_staff_id,
      created_at
    }
  )

  return {
    ok = true,
    data = {
      status = "COVER_ASSIGNED",
      cover_id = cover_id,
      absent_staff_id = absent_staff_id,
      cover_staff_id = cover_staff_id,
      group_id = group_id,
      slot_id = slot_id,
      academic_year_id = academic_year_id,
      date = cover_date,
      session = session,
      assigned_role = gsa_res.role or assigned_role,
      staff_assignment_id = gsa_res.assignment_id or "",
      absence_reason = absence_reason,
      cover_notes = cover_notes,
      created_at = created_at
    }
  }
end

function load_cover_class_roster(ctx, args)
  if type(args) ~= "table" then
    error("missing mandatory arguments table for load_cover_class_roster")
  end

  local group_id = trim(args.group_id)
  if group_id == "" then
    error("missing mandatory parameter: group_id")
  end

  local school_defaults = resolve_school_defaults(ctx)
  local date = trim(args.date)
  if date == "" then
    date = trim(args.reference_date)
  end
  if date == "" then
    date = school_defaults.reference_date
  end
  if date == "" then
    error("missing mandatory parameter: date")
  end

  local session = trim(args.session)
  if session == "" then
    session = "AM"
  end
  session = validate_statutory_session(session)

  local opts = build_scope_opts(ctx, args, date, true)

  -- Enforce strict :own_groups access on the target group via list_by_group and get_group_session
  local pupils = ctx.dal.pupils.list_by_group(group_id, opts)
  local pupil_ids = {}
  if type(pupils) == "table" then
    for i = 1, #pupils do
      pupil_ids[i] = pupils[i].pupil_id or pupils[i].id
    end
  end

  local flags_by_pupil = {}
  if #pupil_ids > 0 then
    flags_by_pupil = ctx.dal.pupils.list_sen_and_medical_batch(pupil_ids, opts)
  end

  local existing_marks = ctx.dal.attendance.get_group_session(group_id, date, session, opts)
  local marks_by_pupil = {}
  if type(existing_marks) == "table" then
    for i = 1, #existing_marks do
      local m = existing_marks[i]
      if m and m.pupil_id then
        marks_by_pupil[m.pupil_id] = m
      end
    end
  end

  local cover_rows = ctx.plugin_db.query(
    [[SELECT id, absent_staff_id, cover_staff_id, group_id, slot_id,
             academic_year_id, cover_date, session, assigned_role,
             staff_assignment_id, absence_reason, cover_notes, created_at
      FROM plugin_cover.cover_absences_and_assignments
      WHERE group_id = ? AND cover_date = ?
      ORDER BY created_at DESC, id DESC]],
    { group_id, date }
  )

  local roster = {}
  if type(pupils) == "table" then
    for i = 1, #pupils do
      local p = pupils[i]
      local pid = p.pupil_id or p.id
      local flag_info = flags_by_pupil[pid] or {}
      local med_list = flag_info.medical_conditions or {}
      local existing = marks_by_pupil[pid]

      local legal_first = p.legal_first_name or p.first_name or ""
      local legal_last = p.legal_last_name or p.last_name or ""
      local pref_first = p.preferred_first_name or legal_first
      local pref_last = p.preferred_last_name or legal_last

      local row = {
        pupil_id = pid,
        upn = p.upn or "",
        first_name = legal_first,
        last_name = legal_last,
        legal_first_name = legal_first,
        legal_last_name = legal_last,
        preferred_first_name = pref_first,
        preferred_last_name = pref_last,
        nc_year_group = p.nc_year_group or "",
        enrolment_status = p.enrolment_status or "C",
        sen_provision_code = flag_info.sen_provision_code or "N",
        primary_sen_need_code = flag_info.primary_sen_need_code or "",
        teaching_strategies = flag_info.teaching_strategies or "",
        medical_conditions = med_list
      }
      if existing ~= nil then
        row.persisted_mark = {
          attendance_code = existing.attendance_code,
          minutes_late = existing.minutes_late,
          sub_code = existing.sub_code,
          comment = existing.comment
        }
      end
      roster[i] = row
    end
  end

  return {
    ok = true,
    data = {
      group_id = group_id,
      date = date,
      session = session,
      academic_year_id = opts.academic_year_id or "",
      cover_assignments = cover_rows,
      roster = roster
    }
  }
end

function submit_cover_attendance_batch(ctx, args)
  if type(args) ~= "table" then
    error("missing mandatory arguments table for submit_cover_attendance_batch")
  end

  local group_id = trim(args.group_id)
  if group_id == "" then
    error("missing mandatory parameter: group_id")
  end

  local school_defaults = resolve_school_defaults(ctx)
  local date = trim(args.date)
  if date == "" then
    date = trim(args.reference_date)
  end
  if date == "" then
    date = school_defaults.reference_date
  end
  if date == "" then
    error("missing mandatory parameter: date")
  end

  local session = trim(args.session)
  if session == "" then
    session = "AM"
  end
  session = validate_statutory_session(session)

  if type(args.marks) ~= "table" or #args.marks == 0 then
    error("missing mandatory parameter: marks must be a non-empty array")
  end

  local opts = build_scope_opts(ctx, args, date, true)
  local upsert_res = ctx.dal.attendance.upsert_session_batch(
    group_id,
    date,
    session,
    args.marks,
    opts
  )

  local cover_assignment_id = ""
  local cover_rows = ctx.plugin_db.query(
    [[SELECT id FROM plugin_cover.cover_absences_and_assignments
      WHERE group_id = ? AND cover_date = ?
      ORDER BY created_at DESC LIMIT 1]],
    { group_id, date }
  )
  if type(cover_rows) == "table" and cover_rows[1] and cover_rows[1].id then
    cover_assignment_id = tostring(cover_rows[1].id)
  end

  local created_at = ctx.clock.now_utc()
  local caller_staff_id = (ctx.caller and trim(ctx.caller.staff_id)) or ""
  local ay_id = opts.academic_year_id or school_defaults.academic_year_id or ""

  for i = 1, #args.marks do
    local m = args.marks[i]
    if type(m) == "table" and trim(m.pupil_id) ~= "" then
      local log_id = "cpr_" .. ctx.crypto.uuid()
      ctx.plugin_db.exec(
        [[INSERT INTO plugin_cover.cover_pupil_register_logs (
            id, cover_assignment_id, pupil_id, group_id, academic_year_id,
            cover_date, session, attendance_code, minutes_late,
            register_comment, taken_by_staff_id, submission_id, created_at
          ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)]],
        {
          log_id,
          cover_assignment_id,
          trim(m.pupil_id),
          group_id,
          ay_id,
          date,
          session,
          trim(m.attendance_code),
          tonumber(m.minutes_late) or 0,
          trim(m.comment),
          caller_staff_id,
          upsert_res.submission_id or "",
          created_at
        }
      )
    end
  end

  local persisted_marks = ctx.dal.attendance.get_group_session(group_id, date, session, opts)

  return {
    ok = true,
    data = {
      status = "COVER_REGISTER_SAVED:" .. session,
      submission_id = upsert_res.submission_id,
      upserted_count = upsert_res.upserted_count,
      group_id = upsert_res.group_id,
      date = upsert_res.date,
      session = upsert_res.session,
      submitted_at_utc = created_at,
      persisted_marks = persisted_marks
    }
  }
end

function list_cover_assignments(ctx, args)
  local group_id = type(args) == "table" and trim(args.group_id) or ""
  local rows
  if group_id ~= "" then
    rows = ctx.plugin_db.query(
      [[SELECT id, absent_staff_id, cover_staff_id, group_id, slot_id,
               academic_year_id, cover_date, session, assigned_role,
               staff_assignment_id, absence_reason, cover_notes,
               assigned_by_staff_id, created_at
        FROM plugin_cover.cover_absences_and_assignments
        WHERE group_id = ?
        ORDER BY created_at DESC, id DESC]],
      { group_id }
    )
  else
    rows = ctx.plugin_db.query(
      [[SELECT id, absent_staff_id, cover_staff_id, group_id, slot_id,
               academic_year_id, cover_date, session, assigned_role,
               staff_assignment_id, absence_reason, cover_notes,
               assigned_by_staff_id, created_at
        FROM plugin_cover.cover_absences_and_assignments
        ORDER BY created_at DESC, id DESC]],
      {}
    )
  end

  return {
    ok = true,
    data = {
      assignments = rows
    }
  }
end

function export_pupil_data(ctx, args)
  local pupil_id = type(args) == "table" and trim(args.pupil_id) or trim(args)
  if pupil_id == "" then
    error("missing mandatory parameter: pupil_id")
  end

  local rows = ctx.plugin_db.query(
    [[SELECT id, cover_assignment_id, pupil_id, group_id, academic_year_id,
             cover_date, session, attendance_code, minutes_late,
             register_comment, taken_by_staff_id, submission_id, created_at
      FROM plugin_cover.cover_pupil_register_logs
      WHERE pupil_id = ?
      ORDER BY created_at ASC, id ASC]],
    { pupil_id }
  )

  return {
    ok = true,
    data = {
      sections = {
        {
          title = "Staff Cover Attendance Register Logs",
          records = rows,
          redactable_fields = { "register_comment" }
        }
      }
    }
  }
end

function purge_pupil_data(ctx, args)
  local pupil_id = type(args) == "table" and trim(args.pupil_id) or trim(args)
  if pupil_id == "" then
    error("missing mandatory parameter: pupil_id")
  end

  local res = ctx.plugin_db.exec(
    "DELETE FROM plugin_cover.cover_pupil_register_logs WHERE pupil_id = ?",
    { pupil_id }
  )

  return {
    ok = true,
    data = {
      pupil_id = pupil_id,
      purged_rows = res.rows_affected
    }
  }
end
