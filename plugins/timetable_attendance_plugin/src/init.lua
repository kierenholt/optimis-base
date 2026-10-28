local function trim(s)
  if type(s) ~= "string" then
    return ""
  end
  return (s:gsub("^%s+", ""):gsub("%s+$", ""))
end

local function build_scope_opts(args, fallback_date)
  local opts = {}
  if type(args) == "table" then
    if trim(args.academic_year_id) ~= "" then
      opts.academic_year_id = trim(args.academic_year_id)
    end
    if trim(args.reference_date) ~= "" then
      opts.reference_date = trim(args.reference_date)
    elseif trim(fallback_date) ~= "" then
      opts.reference_date = trim(fallback_date)
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

function load_staff_timetable(ctx, args)
  if type(args) ~= "table" then
    error("missing mandatory arguments table for load_staff_timetable")
  end

  local target_staff_id = trim(args.target_staff_id)
  if target_staff_id == "" then
    target_staff_id = trim(args.staff_id)
  end
  if target_staff_id == "" and ctx and ctx.caller then
    target_staff_id = trim(ctx.caller.staff_id)
  end
  if target_staff_id == "" then
    error("missing mandatory parameter: staff_id")
  end

  local cycle_week = tonumber(args.cycle_week)
  if not cycle_week then
    error("missing mandatory parameter: cycle_week")
  end
  local day_of_week_id = tonumber(args.day_of_week_id)
  if not day_of_week_id then
    error("missing mandatory parameter: day_of_week_id")
  end

  local opts = build_scope_opts(args, args.date)
  local slots = ctx.dal.groups.get_staff_timetable_slots(
    target_staff_id,
    cycle_week,
    day_of_week_id,
    opts
  )

  local resolved_session = ""
  if type(slots) == "table" and slots[1] ~= nil then
    local candidate = trim(slots[1].maps_to_statutory_session)
    if candidate ~= "" then
      resolved_session = validate_statutory_session(candidate)
    end
  end

  return {
    ok = true,
    data = {
      staff_id = target_staff_id,
      cycle_week = cycle_week,
      day_of_week_id = day_of_week_id,
      resolved_session = resolved_session,
      slots = slots
    }
  }
end

function load_slot_roster_with_flags(ctx, args)
  if type(args) ~= "table" then
    error("missing mandatory arguments table for load_slot_roster_with_flags")
  end

  local group_id = trim(args.group_id)
  if group_id == "" then
    error("missing mandatory parameter: group_id")
  end

  local date = trim(args.date)
  if date == "" then
    date = trim(args.reference_date)
  end
  if date == "" then
    error("missing mandatory parameter: date")
  end

  local session = validate_statutory_session(args.session)
  local opts = build_scope_opts(args, date)

  local pupils = ctx.dal.pupils.list_by_group(group_id, opts)
  local pupil_ids = {}
  if type(pupils) == "table" then
    for i = 1, #pupils do
      pupil_ids[i] = pupils[i].id
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

  local roster = {}
  if type(pupils) == "table" then
    for i = 1, #pupils do
      local p = pupils[i]
      local pid = p.id
      local flag_info = flags_by_pupil[pid] or {}
      local med_list = flag_info.medical_conditions or {}
      local existing = marks_by_pupil[pid]

      local first_name = p.first_name or p.legal_first_name or ""
      local last_name = p.last_name or p.legal_last_name or ""
      local row = {
        pupil_id = pid,
        upn = p.upn or "",
        first_name = first_name,
        last_name = last_name,
        legal_first_name = first_name,
        legal_last_name = last_name,
        preferred_first_name = p.preferred_first_name or first_name,
        preferred_last_name = p.preferred_last_name or last_name,
        nc_year_group = p.nc_year_group or "",
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
      roster = roster
    }
  }
end

function submit_session_attendance_batch(ctx, args)
  if type(args) ~= "table" then
    error("missing mandatory arguments table for submit_session_attendance_batch")
  end

  local group_id = trim(args.group_id)
  if group_id == "" then
    error("missing mandatory parameter: group_id")
  end

  local date = trim(args.date)
  if date == "" then
    date = trim(args.reference_date)
  end
  if date == "" then
    error("missing mandatory parameter: date")
  end

  local session = validate_statutory_session(args.session)
  if type(args.marks) ~= "table" or #args.marks == 0 then
    error("missing mandatory parameter: marks must be a non-empty array")
  end

  local opts = build_scope_opts(args, date)
  local upsert_res = ctx.dal.attendance.upsert_session_batch(
    group_id,
    date,
    session,
    args.marks,
    opts
  )

  local persisted_marks = ctx.dal.attendance.get_group_session(group_id, date, session, opts)

  return {
    ok = true,
    data = {
      status = "BATCH_SAVED:" .. session,
      submission_id = upsert_res.submission_id,
      upserted_count = upsert_res.upserted_count,
      group_id = upsert_res.group_id,
      date = upsert_res.date,
      session = upsert_res.session,
      persisted_marks = persisted_marks
    }
  }
end
