local function compute_pupil_behaviour_rollup(ctx, pupil)
  local rows = ctx.plugin_db.query(
    [[SELECT id, pupil_id, academic_year_id, event_date, event_type, category,
             points, incident_notes, logged_by_staff_id, created_at
      FROM plugin_behaviour.behaviour_events
      WHERE pupil_id = ?
      ORDER BY created_at ASC, id ASC]],
    { pupil.id }
  )

  local total_merits = 0
  local total_demerits = 0
  local net_points = 0

  for i = 1, #rows do
    local pts = tonumber(rows[i].points) or 0
    local ev_type = rows[i].event_type or ""
    if ev_type == "MERIT" or pts > 0 then
      total_merits = total_merits + pts
    else
      total_demerits = total_demerits + pts
    end
    net_points = net_points + pts
  end

  local first_name = pupil.preferred_first_name or pupil.first_name or ""
  local last_name = pupil.preferred_last_name or pupil.last_name or ""

  return {
    pupil_id = pupil.id,
    pupil_name = first_name .. " " .. last_name,
    preferred_first_name = first_name,
    preferred_last_name = last_name,
    nc_year_group = pupil.nc_year_group or "",
    upn = pupil.upn or "",
    academic_year_id = pupil.academic_year_id or (ctx.school and ctx.school.academic_year_id) or "",
    total_merits = total_merits,
    total_demerits = total_demerits,
    net_points = net_points,
    event_count = #rows,
    events = rows
  }
end

function log_behaviour_event(ctx, args)
  if type(args) ~= "table" or type(args.pupil_id) ~= "string" or args.pupil_id == "" then
    error("missing mandatory parameter: pupil_id")
  end
  if type(args.event_type) ~= "string" or (args.event_type ~= "MERIT" and args.event_type ~= "INCIDENT") then
    error("invalid or missing mandatory parameter: event_type (must be MERIT or INCIDENT)")
  end
  if type(args.category) ~= "string" or args.category == "" then
    error("missing mandatory parameter: category")
  end
  local points = tonumber(args.points)
  if points == nil then
    error("missing mandatory numeric parameter: points")
  end
  if type(args.incident_notes) ~= "string" or args.incident_notes == "" then
    error("missing mandatory parameter: incident_notes")
  end

  local school_ay = (ctx.school and ctx.school.academic_year_id) or ""
  local school_ref_date = (ctx.school and ctx.school.reference_date) or ""

  local opts = {
    academic_year_id = args.academic_year_id or school_ay,
    reference_date = args.reference_date or school_ref_date,
    strict_own_groups = args.strict_own_groups == true
  }
  local pupil = ctx.dal.pupils.get_by_id(args.pupil_id, opts)
  if not pupil or not pupil.id then
    error("pupil not found or inaccessible under own_groups scope")
  end

  local event_id = args.event_id
  if type(event_id) ~= "string" or event_id == "" then
    event_id = "bev_" .. ctx.crypto.uuid()
  end

  local event_date = args.reference_date
  if type(event_date) ~= "string" or event_date == "" then
    event_date = school_ref_date ~= "" and school_ref_date or (pupil.effective_from or "")
  end

  local ay_id = args.academic_year_id
  if type(ay_id) ~= "string" or ay_id == "" then
    ay_id = pupil.academic_year_id or school_ay
  end

  local created_at = ctx.clock.now_utc()

  ctx.plugin_db.exec(
    [[INSERT INTO plugin_behaviour.behaviour_events (
        id, pupil_id, academic_year_id, event_date, event_type,
        category, points, incident_notes, logged_by_staff_id, created_at
      ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)]],
    {
      event_id,
      pupil.id,
      ay_id,
      event_date,
      args.event_type,
      args.category,
      points,
      args.incident_notes,
      ctx.caller.staff_id or "",
      created_at
    }
  )

  local rollup = compute_pupil_behaviour_rollup(ctx, pupil)
  rollup.logged_event_id = event_id

  ctx.events.emit("behaviour.event_logged", {
    event_id = event_id,
    pupil_id = pupil.id,
    event_type = args.event_type,
    points = points
  })

  return {
    ok = true,
    data = rollup
  }
end

function get_pupil_behaviour_summary(ctx, args)
  if type(args) ~= "table" then
    error("missing mandatory arguments table")
  end

  local school_ay = (ctx.school and ctx.school.academic_year_id) or ""
  local school_ref_date = (ctx.school and ctx.school.reference_date) or ""

  local opts = {
    academic_year_id = args.academic_year_id or school_ay,
    reference_date = args.reference_date or school_ref_date,
    strict_own_groups = args.strict_own_groups == true
  }

  -- 1E: Support batch pupil lookup via ctx.dal.pupils.get_by_ids_batch when args.pupil_ids is provided
  if type(args.pupil_ids) == "table" then
    local pupils = ctx.dal.pupils.get_by_ids_batch(args.pupil_ids, opts)
    local rollups = {}
    for i = 1, #pupils do
      rollups[i] = compute_pupil_behaviour_rollup(ctx, pupils[i])
    end
    return {
      ok = true,
      data = {
        pupils = rollups
      }
    }
  end

  if type(args.pupil_id) ~= "string" or args.pupil_id == "" then
    error("missing mandatory parameter: pupil_id")
  end

  local pupil = ctx.dal.pupils.get_by_id(args.pupil_id, opts)
  if not pupil or not pupil.id then
    error("pupil not found or inaccessible under own_groups scope")
  end

  local rollup = compute_pupil_behaviour_rollup(ctx, pupil)
  return {
    ok = true,
    data = rollup
  }
end

function export_pupil_data(ctx, args)
  local pupil_id = type(args) == "table" and args.pupil_id or args
  if type(pupil_id) ~= "string" or pupil_id == "" then
    error("missing mandatory parameter: pupil_id")
  end

  local rows = ctx.plugin_db.query(
    [[SELECT id, pupil_id, academic_year_id, event_date, event_type,
             category, points, incident_notes, logged_by_staff_id, created_at
      FROM plugin_behaviour.behaviour_events
      WHERE pupil_id = ?
      ORDER BY created_at ASC, id ASC]],
    { pupil_id }
  )

  return {
    ok = true,
    data = {
      {
        title = "Plugin Behaviour & Incident Records",
        records = rows,
        redactable_fields = { "incident_notes" }
      }
    }
  }
end

function purge_pupil_data(ctx, args)
  local pupil_id = type(args) == "table" and args.pupil_id or args
  if type(pupil_id) ~= "string" or pupil_id == "" then
    error("missing mandatory parameter: pupil_id")
  end

  local res = ctx.plugin_db.exec(
    "DELETE FROM plugin_behaviour.behaviour_events WHERE pupil_id = ?",
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
