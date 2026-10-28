local CLASSROOM_SYNC_URL = "https://classroom.googleapis.com/v1/courses/roster:sync"

function upsert_pupil_membership(ctx, args)
  local group_id = args.group_id or ""
  local pupil_id = args.pupil_id or ""
  local academic_year_id = args.academic_year_id or ""
  local effective_from = args.effective_from or ""
  local effective_to = args.effective_to or ""
  local membership_id = args.membership_id or ""

  if group_id == "" then
    error("missing mandatory parameter: group_id")
  end
  if pupil_id == "" then
    error("missing mandatory parameter: pupil_id")
  end
  if academic_year_id == "" then
    error("missing mandatory parameter: academic_year_id")
  end
  if effective_from == "" then
    error("missing mandatory parameter: effective_from")
  end

  local res = ctx.dal.groups.upsert_pupil_membership(
    {
      membership_id = membership_id,
      group_id = group_id,
      pupil_id = pupil_id,
      academic_year_id = academic_year_id,
      effective_from = effective_from,
      effective_to = effective_to
    },
    {
      academic_year_id = academic_year_id,
      reference_date = effective_from
    }
  )

  return {
    ok = true,
    data = {
      status = "MEMBERSHIP_UPSERTED",
      membership_id = res.membership_id,
      group_id = res.group_id,
      pupil_id = res.pupil_id,
      academic_year_id = res.academic_year_id,
      effective_from = res.effective_from,
      effective_to = res.effective_to or ""
    }
  }
end

function get_group_roster(ctx, args)
  local group_id = args.group_id or ""
  local academic_year_id = args.academic_year_id or ""
  local reference_date = args.reference_date or ""

  if group_id == "" then
    error("missing mandatory parameter: group_id")
  end
  if academic_year_id == "" then
    error("missing mandatory parameter: academic_year_id")
  end
  if reference_date == "" then
    error("missing mandatory parameter: reference_date")
  end

  local pupils = ctx.dal.pupils.list_by_group(group_id, {
    academic_year_id = academic_year_id,
    reference_date = reference_date
  })

  return {
    ok = true,
    data = {
      group_id = group_id,
      academic_year_id = academic_year_id,
      reference_date = reference_date,
      pupil_count = #pupils,
      pupils = pupils
    }
  }
end

function on_membership_changed(ctx, args)
  local event_name = args.event_name or "core.group.membership_changed"
  local hop = tonumber(args.hop) or 1
  local membership_id = tostring(args.membership_id or "")
  local group_id = tostring(args.group_id or "")
  local pupil_id = tostring(args.pupil_id or "")
  local academic_year_id = tostring(args.academic_year_id or "")
  local effective_from = tostring(args.effective_from or "")
  local effective_to = tostring(args.effective_to or "")

  if group_id == "" then
    error("on_membership_changed missing mandatory event field: group_id")
  end
  if pupil_id == "" then
    error("on_membership_changed missing mandatory event field: pupil_id")
  end
  if academic_year_id == "" then
    error("on_membership_changed missing mandatory event field: academic_year_id")
  end
  if effective_from == "" then
    error("on_membership_changed missing mandatory event field: effective_from")
  end

  -- 1. Query current group roster from Core DAL (ReadDB)
  local roster = ctx.dal.pupils.list_by_group(group_id, {
    academic_year_id = academic_year_id,
    reference_date = effective_from
  })

  local pupil_ids = {}
  local roster_pupils = {}
  for _, p in ipairs(roster) do
    table.insert(pupil_ids, tostring(p.id))
    table.insert(roster_pupils, {
      pupil_id = tostring(p.id),
      preferred_first_name = tostring(p.preferred_first_name or p.first_name or ""),
      preferred_last_name = tostring(p.preferred_last_name or p.last_name or ""),
      nc_year_group = tostring(p.nc_year_group or "")
    })
  end

  local outbound_body = ctx.json.encode({
    course_group_id = group_id,
    academic_year_id = academic_year_id,
    triggered_by_pupil_id = pupil_id,
    membership_id = membership_id,
    effective_from = effective_from,
    roster_count = #roster,
    pupil_ids = pupil_ids,
    pupils = roster_pupils
  })

  -- 2. Structural Safety Invariant (§9.2 & §10.1): Execute outbound HTTP BEFORE opening any SQLite write transaction
  local resp = ctx.http.post(
    CLASSROOM_SYNC_URL,
    {
      ["Content-Type"] = "application/json",
      ["X-Optimis-Plugin"] = "uk.org.optimis.lms.classroom",
      ["X-Optimis-Group-ID"] = group_id
    },
    outbound_body
  )

  local http_status = tonumber(resp.status) or 0
  local sync_status = "FAILED"
  local external_sync_id = ""
  local sync_notes = ""

  if resp.ok and http_status >= 200 and http_status < 300 then
    sync_status = "SYNCED"
    if resp.body and resp.body ~= "" then
      local decoded = ctx.json.decode(resp.body)
      if type(decoded) == "table" and decoded.sync_receipt_id then
        external_sync_id = tostring(decoded.sync_receipt_id)
      end
    end
    if external_sync_id == "" then
      external_sync_id = "gclass_sync_ok"
    end
    sync_notes = string.format(
      "Google Classroom roster sync succeeded for group %s (pupil %s, roster_count=%d, receipt=%s)",
      group_id,
      pupil_id,
      #roster,
      external_sync_id
    )
  else
    sync_status = "FAILED"
    external_sync_id = ""
    local err_code = tostring(resp.error_code or "")
    if err_code ~= "" then
      sync_notes = string.format(
        "Google Classroom roster sync failed for group %s (pupil %s, http_status=%d, error_code=%s)",
        group_id,
        pupil_id,
        http_status,
        err_code
      )
    else
      sync_notes = string.format(
        "Google Classroom roster sync failed for group %s (pupil %s, http_status=%d)",
        group_id,
        pupil_id,
        http_status
      )
    end
  end

  local log_id = "lms_log_" .. ctx.crypto.uuid()
  local synced_at = ctx.clock.now_utc()

  -- 3. Persist sync status audit record into plugin_lms_sync.db
  ctx.plugin_db.exec(
    [[INSERT INTO plugin_lms_sync.lms_sync_logs (
        id, event_name, hop, membership_id, group_id, pupil_id,
        academic_year_id, effective_from, effective_to, roster_count,
        target_url, http_status, status, external_sync_id,
        caller_type, caller_staff_id, sync_notes, synced_at
      ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)]],
    {
      log_id,
      event_name,
      hop,
      membership_id,
      group_id,
      pupil_id,
      academic_year_id,
      effective_from,
      effective_to,
      #roster,
      CLASSROOM_SYNC_URL,
      http_status,
      sync_status,
      external_sync_id,
      ctx.caller.type or "SYSTEM",
      ctx.caller.staff_id or "",
      sync_notes,
      synced_at
    }
  )

  return {
    ok = true,
    data = {
      log_id = log_id,
      status = sync_status,
      http_status = http_status,
      external_sync_id = external_sync_id,
      group_id = group_id,
      pupil_id = pupil_id,
      roster_count = #roster,
      synced_at = synced_at
    }
  }
end

function list_sync_logs(ctx, args)
  local pupil_id = args.pupil_id or ""
  local group_id = args.group_id or ""

  local rows
  if pupil_id ~= "" and group_id ~= "" then
    rows = ctx.plugin_db.query(
      [[SELECT id, event_name, hop, membership_id, group_id, pupil_id,
               academic_year_id, effective_from, effective_to, roster_count,
               target_url, http_status, status, external_sync_id,
               caller_type, caller_staff_id, sync_notes, synced_at
        FROM plugin_lms_sync.lms_sync_logs
        WHERE pupil_id = ? AND group_id = ?
        ORDER BY synced_at DESC, id DESC]],
      { pupil_id, group_id }
    )
  elseif pupil_id ~= "" then
    rows = ctx.plugin_db.query(
      [[SELECT id, event_name, hop, membership_id, group_id, pupil_id,
               academic_year_id, effective_from, effective_to, roster_count,
               target_url, http_status, status, external_sync_id,
               caller_type, caller_staff_id, sync_notes, synced_at
        FROM plugin_lms_sync.lms_sync_logs
        WHERE pupil_id = ?
        ORDER BY synced_at DESC, id DESC]],
      { pupil_id }
    )
  elseif group_id ~= "" then
    rows = ctx.plugin_db.query(
      [[SELECT id, event_name, hop, membership_id, group_id, pupil_id,
               academic_year_id, effective_from, effective_to, roster_count,
               target_url, http_status, status, external_sync_id,
               caller_type, caller_staff_id, sync_notes, synced_at
        FROM plugin_lms_sync.lms_sync_logs
        WHERE group_id = ?
        ORDER BY synced_at DESC, id DESC]],
      { group_id }
    )
  else
    rows = ctx.plugin_db.query(
      [[SELECT id, event_name, hop, membership_id, group_id, pupil_id,
               academic_year_id, effective_from, effective_to, roster_count,
               target_url, http_status, status, external_sync_id,
               caller_type, caller_staff_id, sync_notes, synced_at
        FROM plugin_lms_sync.lms_sync_logs
        ORDER BY synced_at DESC, id DESC]],
      {}
    )
  end

  return {
    ok = true,
    data = {
      logs = rows
    }
  }
end

function export_pupil_data(ctx, args)
  local pupil_id = args.pupil_id
  if not pupil_id or pupil_id == "" then
    error("missing mandatory parameter: pupil_id")
  end

  local rows = ctx.plugin_db.query(
    [[SELECT id, event_name, hop, membership_id, group_id, pupil_id,
             academic_year_id, effective_from, effective_to, roster_count,
             target_url, http_status, status, external_sync_id,
             caller_type, caller_staff_id, sync_notes, synced_at
      FROM plugin_lms_sync.lms_sync_logs
      WHERE pupil_id = ?
      ORDER BY synced_at ASC, id ASC]],
    { pupil_id }
  )

  return {
    ok = true,
    data = {
      sections = {
        {
          title = "Google Classroom LMS Roster Sync Logs",
          records = rows,
          redactable_fields = { "sync_notes" }
        }
      }
    }
  }
end

function purge_pupil_data(ctx, args)
  local pupil_id = args.pupil_id
  if not pupil_id or pupil_id == "" then
    error("missing mandatory parameter: pupil_id")
  end

  local res = ctx.plugin_db.exec(
    "DELETE FROM plugin_lms_sync.lms_sync_logs WHERE pupil_id = ?",
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
