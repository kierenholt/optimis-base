function get_pupil_record(ctx, args)
  if type(ctx) ~= "table" then
    error("missing mandatory ctx table")
  end
  if type(ctx.caller) ~= "table" or type(ctx.dal) ~= "table" or type(ctx.plugin_db) ~= "table" or type(ctx.events) ~= "table" or type(ctx.http) ~= "table" then
    error("missing one or more mandatory ctx namespaces (caller, dal, plugin_db, events, http)")
  end

  if type(args) ~= "table" or type(args.pupil_id) ~= "string" or args.pupil_id == "" then
    error("missing mandatory parameter: pupil_id")
  end

  local pupil = ctx.dal.pupils.get_by_id(args.pupil_id)
  if not pupil or not pupil.id then
    error("pupil record not found")
  end

  ctx.events.emit("test.pupil_fetched", {
    pupil_id = pupil.id
  })

  return {
    ok = true,
    data = {
      id = pupil.id,
      upn = pupil.upn,
      first_name = pupil.first_name,
      last_name = pupil.last_name,
      academic_year_id = pupil.academic_year_id,
      effective_from = pupil.effective_from,
      ctx_modules_verified = true
    }
  }
end

function save_pupil_note(ctx, args)
  if type(args) ~= "table" or type(args.pupil_id) ~= "string" or args.pupil_id == "" then
    error("missing mandatory parameter: pupil_id")
  end
  if type(args.note) ~= "string" or args.note == "" then
    error("missing mandatory parameter: note")
  end

  local note_id = args.note_id
  if type(note_id) ~= "string" or note_id == "" then
    error("missing mandatory parameter: note_id")
  end

  local exec_res = ctx.plugin_db.exec(
    "INSERT INTO pupil_notes (id, pupil_id, note, created_by) VALUES (?, ?, ?, ?) ON CONFLICT(id) DO UPDATE SET note = excluded.note",
    { note_id, args.pupil_id, args.note, ctx.caller.user_id }
  )

  local rows = ctx.plugin_db.query(
    "SELECT id, pupil_id, note, created_by FROM pupil_notes WHERE pupil_id = ? ORDER BY id ASC",
    { args.pupil_id }
  )

  return {
    ok = true,
    data = {
      rows_affected = exec_res.rows_affected,
      count = #rows,
      first_note = rows[1] and rows[1].note or "",
      notes = rows
    }
  }
end

function list_pupil_notes(ctx, args)
  if type(args) ~= "table" or type(args.pupil_id) ~= "string" or args.pupil_id == "" then
    error("missing mandatory parameter: pupil_id")
  end

  local rows = ctx.plugin_db.query(
    "SELECT id, pupil_id, note, created_by FROM pupil_notes WHERE pupil_id = ? ORDER BY id ASC",
    { args.pupil_id }
  )
  local pragma_rows = ctx.plugin_db.query("PRAGMA plugin_test.max_page_count")
  local max_pages = 0
  if pragma_rows and pragma_rows[1] and pragma_rows[1].max_page_count then
    max_pages = pragma_rows[1].max_page_count
  end

  return {
    ok = true,
    data = {
      pupil_id = args.pupil_id,
      count = #rows,
      first_note = rows[1] and rows[1].note or "",
      db_alias = "plugin_test",
      max_page_count = max_pages,
      notes = rows
    }
  }
end

function export_pupil_data(ctx, args)
  local pupil_id = type(args) == "table" and args.pupil_id or args
  if type(pupil_id) ~= "string" or pupil_id == "" then
    error("missing mandatory parameter: pupil_id")
  end

  local rows = ctx.plugin_db.query(
    "SELECT id, pupil_id, note, created_by FROM pupil_notes WHERE pupil_id = ? ORDER BY id ASC",
    { pupil_id }
  )

  local sections = {
    {
      title = "Plugin Pupil Notes",
      records = rows,
      redactable_fields = { "note" }
    }
  }

  local worker_rows = ctx.plugin_db.query(
    "SELECT id, pupil_id, event_name, hop, caller_type, caller_staff_id FROM worker_event_logs WHERE pupil_id = ? ORDER BY id ASC",
    { pupil_id }
  )
  if worker_rows and #worker_rows > 0 then
    sections[#sections + 1] = {
      title = "Plugin Worker Event Logs",
      records = worker_rows,
      redactable_fields = {}
    }
  end

  return {
    ok = true,
    data = sections
  }
end

function purge_pupil_data(ctx, args)
  local pupil_id = type(args) == "table" and args.pupil_id or args
  if type(pupil_id) ~= "string" or pupil_id == "" then
    error("missing mandatory parameter: pupil_id")
  end

  local res = ctx.plugin_db.exec(
    "DELETE FROM pupil_notes WHERE pupil_id = ?",
    { pupil_id }
  )
  local res_workers = ctx.plugin_db.exec(
    "DELETE FROM worker_event_logs WHERE pupil_id = ?",
    { pupil_id }
  )

  return {
    ok = true,
    data = {
      purged_rows = res.rows_affected + res_workers.rows_affected
    }
  }
end

function get_group_roster_summary(ctx, args)
  if type(args) ~= "table" or type(args.group_id) ~= "string" or args.group_id == "" then
    error("missing mandatory parameter: group_id")
  end

  local opts = {
    reference_date = args.reference_date,
    academic_year_id = args.academic_year_id
  }
  local pupils = ctx.dal.pupils.list_by_group(args.group_id, opts)
  local pupil_ids = {}
  for i = 1, #pupils do
    pupil_ids[i] = pupils[i].id
  end

  local sen_med_map = ctx.dal.pupils.list_sen_and_medical_batch(pupil_ids, opts)
  local roster = {}
  for i = 1, #pupils do
    local p = pupils[i]
    local sm = sen_med_map[p.id] or {}
    local first_med = ""
    if sm.medical_conditions and sm.medical_conditions[1] then
      first_med = sm.medical_conditions[1].condition_name
    end
    roster[i] = {
      id = p.id,
      full_name = p.first_name .. " " .. p.last_name,
      sen_provision_code = sm.sen_provision_code or "N",
      primary_sen_need_code = sm.primary_sen_need_code or "",
      medical_alert = first_med
    }
  end

  return {
    ok = true,
    data = {
      group_id = args.group_id,
      roster = roster
    }
  }
end

function get_pupil_guardians(ctx, args)
  if type(args) ~= "table" or type(args.pupil_id) ~= "string" or args.pupil_id == "" then
    error("missing mandatory parameter: pupil_id")
  end

  local guardians = ctx.dal.guardians.list_for_pupil(args.pupil_id, {
    reference_date = args.reference_date,
    academic_year_id = args.academic_year_id,
    require_parental_responsibility = false
  })
  for i = 1, #guardians do
    local g = guardians[i]
    g.full_name = (g.first_name or "") .. " " .. (g.last_name or "")
  end

  return {
    ok = true,
    data = {
      pupil_id = args.pupil_id,
      guardians = guardians
    }
  }
end

function record_group_am_attendance(ctx, args)
  if type(args) ~= "table" or type(args.group_id) ~= "string" or args.group_id == "" then
    error("missing mandatory parameter: group_id")
  end
  if type(args.pupil_id) ~= "string" or args.pupil_id == "" then
    error("missing mandatory parameter: pupil_id")
  end

  local res = ctx.dal.attendance.record_session_batch(
    args.group_id,
    args.reference_date,
    "AM",
    {
      {
        pupil_id = args.pupil_id,
        attendance_code = "/",
        minutes_late = 0,
        comment = "Present on time"
      }
    },
    {
      reference_date = args.reference_date,
      academic_year_id = args.academic_year_id
    }
  )

  ctx.events.emit("attendance.session_marks_recorded", {
    group_id = args.group_id,
    session = "AM",
    pupil_id = args.pupil_id,
    submission_id = res.submission_id
  })

  return {
    ok = true,
    data = {
      status = "SESSION_SAVED:AM",
      pupil_id = args.pupil_id,
      submission_id = res.submission_id,
      emitted_event = "attendance.session_marks_recorded"
    }
  }
end

function submit_safeguarding_postbox(ctx, args)
  if type(args) ~= "table" or type(args.pupil_id) ~= "string" or args.pupil_id == "" then
    error("missing mandatory parameter: pupil_id")
  end

  local read_back_blocked = (ctx.dal.safeguarding.read == nil and ctx.dal.safeguarding.get_by_id == nil)
  local receipt = ctx.dal.safeguarding.create_postbox({
    pupil_id = args.pupil_id,
    concern_category = "PHYSICAL",
    incident_date = args.reference_date,
    summary = "Morning registration welfare concern",
    body_map = "NONE",
    verbatim_account = "Pupil reported concern during registration.",
    action_taken = "Logged in safeguarding postbox for DSL triage"
  })

  return {
    ok = true,
    data = {
      status = "POSTBOX_ACCEPTED",
      note_id = receipt.note_id,
      read_back_blocked = read_back_blocked
    }
  }
end

function sync_external_sparx_get(ctx, args)
  if type(args) ~= "table" or type(args.pupil_id) ~= "string" or args.pupil_id == "" then
    error("missing mandatory parameter: pupil_id")
  end
  if type(args.target_url) ~= "string" or args.target_url == "" then
    error("missing mandatory parameter: target_url")
  end

  local res = ctx.http.get(args.target_url, {
    ["X-School-URN"] = "100001",
    ["X-Pupil-Ref"] = args.pupil_id
  })

  return {
    ok = true,
    data = {
      status = res.status,
      body = res.body,
      final_url = res.final_url,
      headers = res.headers
    }
  }
end

function push_external_sparx_post(ctx, args)
  if type(args) ~= "table" or type(args.pupil_id) ~= "string" or args.pupil_id == "" then
    error("missing mandatory parameter: pupil_id")
  end
  if type(args.target_url) ~= "string" or args.target_url == "" then
    error("missing mandatory parameter: target_url")
  end
  if type(args.payload) ~= "string" or args.payload == "" then
    error("missing mandatory parameter: payload")
  end

  local res = ctx.http.post(
    args.target_url,
    { ["Content-Type"] = "application/json" },
    args.payload
  )

  return {
    ok = true,
    data = {
      status = res.status,
      body = res.body,
      final_url = res.final_url,
      headers = res.headers
    }
  }
end

function mutate_then_illegal_egress(ctx, args)
  if type(args) ~= "table" or type(args.pupil_id) ~= "string" or args.pupil_id == "" then
    error("missing mandatory parameter: pupil_id")
  end
  if type(args.note_id) ~= "string" or args.note_id == "" then
    error("missing mandatory parameter: note_id")
  end
  if type(args.note) ~= "string" or args.note == "" then
    error("missing mandatory parameter: note")
  end
  if type(args.target_url) ~= "string" or args.target_url == "" then
    error("missing mandatory parameter: target_url")
  end

  ctx.plugin_db.exec(
    "INSERT INTO pupil_notes (id, pupil_id, note, created_by) VALUES (?, ?, ?, ?)",
    { args.note_id, args.pupil_id, args.note, ctx.caller.user_id }
  )

  local res = ctx.http.post(
    args.target_url,
    { ["Content-Type"] = "application/json" },
    '{"note_id":"' .. args.note_id .. '"}'
  )

  return {
    ok = true,
    data = {
      status = res.status
    }
  }
end

function trigger_loop_relay(ctx, args)
  if type(args) ~= "table" or type(args.pupil_id) ~= "string" or args.pupil_id == "" then
    error("missing mandatory parameter: pupil_id")
  end

  ctx.events.emit("test.loop_relay", {
    pupil_id = args.pupil_id
  })

  return {
    ok = true,
    data = {
      status = "LOOP_RELAY_EMITTED",
      pupil_id = args.pupil_id
    }
  }
end

function list_worker_event_logs(ctx, args)
  if type(args) ~= "table" or type(args.pupil_id) ~= "string" or args.pupil_id == "" then
    error("missing mandatory parameter: pupil_id")
  end

  local rows = ctx.plugin_db.query(
    "SELECT id, pupil_id, event_name, hop, caller_type, caller_staff_id FROM worker_event_logs WHERE pupil_id = ? ORDER BY hop ASC, id ASC",
    { args.pupil_id }
  )

  return {
    ok = true,
    data = {
      pupil_id = args.pupil_id,
      count = #rows,
      logs = rows
    }
  }
end

function on_core_event_audit(ctx, args)
  if type(args) ~= "table" then
    error("missing mandatory worker args table")
  end

  local pupil_id = args.pupil_id
  if (type(pupil_id) ~= "string" or pupil_id == "") and type(args.payload) == "table" then
    pupil_id = args.payload.pupil_id
  end
  if type(pupil_id) ~= "string" or pupil_id == "" then
    return { ok = true, data = { skipped = true } }
  end

  local event_name = args.event_name or "unknown"
  local hop = tonumber(args.hop) or 1
  local log_id = "wlog_" .. event_name .. "_" .. pupil_id .. "_h" .. tostring(hop)

  ctx.plugin_db.exec(
    "INSERT INTO worker_event_logs (id, pupil_id, event_name, hop, caller_type, caller_staff_id) VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO UPDATE SET hop = excluded.hop",
    { log_id, pupil_id, event_name, hop, ctx.caller.type, ctx.caller.staff_id }
  )

  return {
    ok = true,
    data = {
      logged_id = log_id,
      event_name = event_name,
      hop = hop
    }
  }
end

function on_loop_relay(ctx, args)
  if type(args) ~= "table" then
    error("missing mandatory worker args table")
  end

  local pupil_id = args.pupil_id
  if (type(pupil_id) ~= "string" or pupil_id == "") and type(args.payload) == "table" then
    pupil_id = args.payload.pupil_id
  end
  if type(pupil_id) ~= "string" or pupil_id == "" then
    error("missing mandatory parameter: pupil_id in loop relay")
  end

  local hop = tonumber(args.hop) or 1
  local log_id = "wlog_test.loop_relay_" .. pupil_id .. "_h" .. tostring(hop)

  ctx.plugin_db.exec(
    "INSERT INTO worker_event_logs (id, pupil_id, event_name, hop, caller_type, caller_staff_id) VALUES (?, ?, 'test.loop_relay', ?, ?, ?) ON CONFLICT(id) DO UPDATE SET hop = excluded.hop",
    { log_id, pupil_id, hop, ctx.caller.type, ctx.caller.staff_id }
  )

  -- Re-emit unconditionally; the Host Event Bus router must drop when Hop > 3 (i.e. Hop == 4)
  ctx.events.emit("test.loop_relay", {
    pupil_id = pupil_id
  })

  return {
    ok = true,
    data = {
      logged_id = log_id,
      hop = hop
    }
  }
end


