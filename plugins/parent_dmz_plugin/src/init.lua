function pull_dmz_inbox(ctx, args)
  local school_ctx = ctx.school or {}
  local academic_year_id = args.academic_year_id or school_ctx.academic_year_id
  local reference_date = args.reference_date or school_ctx.reference_date
  if not academic_year_id or academic_year_id == "" then
    error("missing mandatory parameter: academic_year_id")
  end
  if not reference_date or reference_date == "" then
    error("missing mandatory parameter: reference_date")
  end

  -- Structural Safety Invariant (§9.2 & §10.1): Outbound HTTP executes BEFORE opening any WriteDB transaction
  local resp = ctx.http.post(
    "https://dmz.parent.optimis.org.uk/v1/inbox/pull",
    { ["Content-Type"] = "application/json" },
    ctx.json.encode({})
  )

  if not resp.ok then
    error(
      "DMZ_INBOX_PULL_FAILED: status " ..
        tostring(resp.status) ..
        " error=" ..
        tostring(resp.error_code or resp.error or resp.body)
    )
  end

  local decoded = ctx.json.decode(resp.body)
  local raw_submissions = decoded.submissions or {}

  -- Collect unique pupil IDs across the pulled inbox batch for coarse-grained O(1) DAL lookups (Item 1E)
  local distinct_pupil_ids = {}
  local seen_pupil_ids = {}
  for _, sub in ipairs(raw_submissions) do
    local pid = sub.pupil_id or ""
    if pid ~= "" and not seen_pupil_ids[pid] then
      seen_pupil_ids[pid] = true
      table.insert(distinct_pupil_ids, pid)
    end
  end

  local active_pupils_by_id = {}
  local pr_guardians_by_pupil = {}

  if #distinct_pupil_ids > 0 then
    local pupils_batch = ctx.dal.pupils.get_by_ids_batch(distinct_pupil_ids, {
      reference_date = reference_date,
      academic_year_id = academic_year_id
    })
    for _, p in ipairs(pupils_batch) do
      active_pupils_by_id[p.id] = p
    end

    pr_guardians_by_pupil = ctx.dal.guardians.list_for_pupils_batch(distinct_pupil_ids, {
      reference_date = reference_date,
      academic_year_id = academic_year_id,
      require_parental_responsibility = true
    })
  end

  local processed = {}
  local accepted_count = 0
  local rejected_count = 0
  local now_utc = ctx.clock.now_utc()

  for _, sub in ipairs(raw_submissions) do
    local sub_id = sub.submission_id or ctx.crypto.uuid()
    local pupil_id = sub.pupil_id or ""
    local guardian_id = sub.submitted_by_guardian_person_id or ""
    local sub_type = sub.submission_type or "ABSENCE_NOTICE"
    local sub_date = sub.date or reference_date
    local session = sub.session or "AM"
    local reason = sub.reason or ""

    local status = "REJECTED_SAFEGUARDING_BLOCK"
    local guardian_name = ""

    if active_pupils_by_id[pupil_id] ~= nil then
      local pr_guardians = pr_guardians_by_pupil[pupil_id] or {}
      for _, g in ipairs(pr_guardians) do
        if g.guardian_person_id == guardian_id and g.has_parental_responsibility then
          status = "ACCEPTED"
          guardian_name = (g.first_name or "") .. " " .. (g.last_name or "")
          break
        end
      end
    end

    if status == "ACCEPTED" then
      accepted_count = accepted_count + 1
    else
      rejected_count = rejected_count + 1
    end

    table.insert(processed, {
      submission_id = sub_id,
      pupil_id = pupil_id,
      submitted_by_guardian_person_id = guardian_id,
      guardian_name = guardian_name,
      submission_type = sub_type,
      date = sub_date,
      session = session,
      reason = reason,
      status = status,
      processed_at = now_utc
    })
  end

  for _, item in ipairs(processed) do
    ctx.plugin_db.exec(
      "DELETE FROM plugin_parent_dmz.dmz_inbox_submissions WHERE submission_id = ?",
      { item.submission_id }
    )
    ctx.plugin_db.exec(
      [[INSERT INTO plugin_parent_dmz.dmz_inbox_submissions (
          submission_id, pupil_id, submitted_by_guardian_person_id, guardian_name,
          submission_type, date, session, reason, status, processed_by_staff_id, processed_at
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)]],
      {
        item.submission_id,
        item.pupil_id,
        item.submitted_by_guardian_person_id,
        item.guardian_name,
        item.submission_type,
        item.date,
        item.session,
        item.reason,
        item.status,
        ctx.caller.staff_id or "",
        item.processed_at
      }
    )
  end

  return {
    ok = true,
    data = {
      accepted_count = accepted_count,
      rejected_count = rejected_count,
      submissions = processed
    }
  }
end

function push_sanitized_projection(ctx, args)
  local pupil_id = args.pupil_id
  if not pupil_id or pupil_id == "" then
    error("missing mandatory parameter: pupil_id")
  end

  local school_ctx = ctx.school or {}
  local academic_year_id = args.academic_year_id or school_ctx.academic_year_id
  local reference_date = args.reference_date or school_ctx.reference_date
  if not academic_year_id or academic_year_id == "" then
    error("missing mandatory parameter: academic_year_id")
  end
  if not reference_date or reference_date == "" then
    error("missing mandatory parameter: reference_date")
  end

  local pupil = ctx.dal.pupils.get_by_id(pupil_id, {
    reference_date = reference_date,
    academic_year_id = academic_year_id
  })

  local all_guardians = ctx.dal.guardians.list_for_pupil(pupil_id, {
    reference_date = reference_date,
    academic_year_id = academic_year_id,
    require_parental_responsibility = false
  })

  local authorized_ids = {}
  local redacted_count = 0

  for _, g in ipairs(all_guardians) do
    if g.is_confidential_address_redacted then
      redacted_count = redacted_count + 1
    end
    if g.has_parental_responsibility then
      table.insert(authorized_ids, g.guardian_person_id)
    end
  end

  local att = ctx.dal.attendance.get_pupil_summary(pupil_id, {
    reference_date = reference_date,
    academic_year_id = academic_year_id
  })

  local present_sessions = att.present_sessions or 0
  local total_sessions = att.total_sessions or 0
  local first_name = pupil.preferred_first_name or pupil.first_name
  local last_name = pupil.preferred_last_name or pupil.last_name

  -- Explicitly strip statutory UPN/ULN and confidential household addresses from the outbound DMZ read-projection
  local projection_payload = {
    pupil_id = pupil.id,
    first_name = first_name,
    last_name = last_name,
    academic_year_id = academic_year_id,
    reference_date = reference_date,
    present_sessions = present_sessions,
    total_sessions = total_sessions,
    authorized_guardian_ids = authorized_ids,
    redacted_confidential_address_count = redacted_count
  }

  local sanitized_json = ctx.json.encode(projection_payload)
  local authorized_guardians_json = (#authorized_ids > 0) and ctx.json.encode(authorized_ids) or "[]"

  -- Structural Safety Invariant (§9.2 & §10.1): Outbound HTTP executes BEFORE opening any WriteDB transaction
  local resp = ctx.http.post(
    "https://dmz.parent.optimis.org.uk/v1/projections/push",
    { ["Content-Type"] = "application/json" },
    sanitized_json
  )

  if not resp.ok then
    error(
      "DMZ_PROJECTION_PUSH_FAILED: status " ..
        tostring(resp.status) ..
        " error=" ..
        tostring(resp.error_code or resp.error or resp.body)
    )
  end

  local snap_id = "dmz_proj_" .. pupil.id .. "_" .. reference_date
  local pushed_at = ctx.clock.now_utc()
  local note_text = "Sanitized DMZ read-projection pushed (UPN stripped; redacted_confidential_addresses=" .. tostring(redacted_count) .. ")"

  ctx.plugin_db.exec(
    "DELETE FROM plugin_parent_dmz.dmz_projection_snapshots WHERE id = ?",
    { snap_id }
  )
  ctx.plugin_db.exec(
    [[INSERT INTO plugin_parent_dmz.dmz_projection_snapshots (
        id, pupil_id, first_name, last_name, academic_year_id, reference_date,
        present_sessions, total_sessions, authorized_guardian_ids_json,
        redacted_confidential_address_count, upn_stripped, sanitized_payload_json,
        notes, pushed_by_staff_id, pushed_at
      ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, ?, ?, ?, ?)]],
    {
      snap_id,
      pupil.id,
      first_name,
      last_name,
      academic_year_id,
      reference_date,
      present_sessions,
      total_sessions,
      authorized_guardians_json,
      redacted_count,
      sanitized_json,
      note_text,
      ctx.caller.staff_id or "",
      pushed_at
    }
  )

  return {
    ok = true,
    data = {
      projection = projection_payload,
      upn_stripped = true,
      sanitized_payload_json = sanitized_json
    }
  }
end

function list_dmz_sync_history(ctx, args)
  local pupil_id = args.pupil_id or ""
  local inbox_rows
  local proj_rows

  if pupil_id ~= "" then
    inbox_rows = ctx.plugin_db.query(
      [[SELECT submission_id, pupil_id, submitted_by_guardian_person_id, guardian_name,
               submission_type, date, session, reason, status, processed_by_staff_id, processed_at
        FROM plugin_parent_dmz.dmz_inbox_submissions
        WHERE pupil_id = ?
        ORDER BY processed_at DESC, submission_id DESC]],
      { pupil_id }
    )
    proj_rows = ctx.plugin_db.query(
      [[SELECT id, pupil_id, first_name, last_name, academic_year_id, reference_date,
               present_sessions, total_sessions, authorized_guardian_ids_json,
               redacted_confidential_address_count, upn_stripped, sanitized_payload_json, notes, pushed_at
        FROM plugin_parent_dmz.dmz_projection_snapshots
        WHERE pupil_id = ?
        ORDER BY pushed_at DESC, id DESC]],
      { pupil_id }
    )
  else
    inbox_rows = ctx.plugin_db.query(
      [[SELECT submission_id, pupil_id, submitted_by_guardian_person_id, guardian_name,
               submission_type, date, session, reason, status, processed_by_staff_id, processed_at
        FROM plugin_parent_dmz.dmz_inbox_submissions
        ORDER BY processed_at DESC, submission_id DESC]],
      {}
    )
    proj_rows = ctx.plugin_db.query(
      [[SELECT id, pupil_id, first_name, last_name, academic_year_id, reference_date,
               present_sessions, total_sessions, authorized_guardian_ids_json,
               redacted_confidential_address_count, upn_stripped, sanitized_payload_json, notes, pushed_at
        FROM plugin_parent_dmz.dmz_projection_snapshots
        ORDER BY pushed_at DESC, id DESC]],
      {}
    )
  end

  return {
    ok = true,
    data = {
      inbox_submissions = inbox_rows,
      projection_snapshots = proj_rows
    }
  }
end

function export_pupil_data(ctx, args)
  local pupil_id = args.pupil_id
  if not pupil_id or pupil_id == "" then
    error("missing mandatory parameter: pupil_id")
  end

  local inbox_rows = ctx.plugin_db.query(
    [[SELECT submission_id, pupil_id, submitted_by_guardian_person_id, guardian_name,
             submission_type, date, session, reason, status, processed_by_staff_id, processed_at
      FROM plugin_parent_dmz.dmz_inbox_submissions
      WHERE pupil_id = ?
      ORDER BY processed_at ASC, submission_id ASC]],
    { pupil_id }
  )

  local proj_rows = ctx.plugin_db.query(
    [[SELECT id, pupil_id, first_name, last_name, academic_year_id, reference_date,
             present_sessions, total_sessions, authorized_guardian_ids_json,
             redacted_confidential_address_count, upn_stripped, sanitized_payload_json, notes, pushed_at
      FROM plugin_parent_dmz.dmz_projection_snapshots
      WHERE pupil_id = ?
      ORDER BY pushed_at ASC, id ASC]],
    { pupil_id }
  )

  return {
    ok = true,
    data = {
      sections = {
        {
          title = "Parent DMZ Write-Inbox Submissions",
          records = inbox_rows,
          redactable_fields = { "reason" }
        },
        {
          title = "Parent DMZ Read-Projection Snapshots",
          records = proj_rows,
          redactable_fields = { "notes" }
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

  local res_inbox = ctx.plugin_db.exec(
    "DELETE FROM plugin_parent_dmz.dmz_inbox_submissions WHERE pupil_id = ?",
    { pupil_id }
  )
  local res_proj = ctx.plugin_db.exec(
    "DELETE FROM plugin_parent_dmz.dmz_projection_snapshots WHERE pupil_id = ?",
    { pupil_id }
  )

  local total_purged = (res_inbox.rows_affected or 0) + (res_proj.rows_affected or 0)

  return {
    ok = true,
    data = {
      pupil_id = pupil_id,
      purged_rows = total_purged
    }
  }
end
