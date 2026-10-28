local function json_escape(str)
  local s = tostring(str or "")
  s = string.gsub(s, '\\', '\\\\')
  s = string.gsub(s, '"', '\\"')
  s = string.gsub(s, '\n', '\\n')
  s = string.gsub(s, '\r', '\\r')
  s = string.gsub(s, '\t', '\\t')
  return '"' .. s .. '"'
end

local function parse_inbox_submissions_json(body, fallback_date)
  local submissions = {}
  local arr_str = string.match(body or "", '"submissions"%s*:%s*%[(.*)%]') or ""
  for obj_str in string.gmatch(arr_str, "{([^{}]+)}") do
    local sub_id = string.match(obj_str, '"submission_id"%s*:%s*"([^"]*)"') or ""
    local pupil_id = string.match(obj_str, '"pupil_id"%s*:%s*"([^"]*)"') or ""
    local guardian_id = string.match(obj_str, '"submitted_by_guardian_person_id"%s*:%s*"([^"]*)"') or ""
    local sub_type = string.match(obj_str, '"submission_type"%s*:%s*"([^"]*)"') or "ABSENCE_NOTICE"
    local sub_date = string.match(obj_str, '"date"%s*:%s*"([^"]*)"') or fallback_date
    local session = string.match(obj_str, '"session"%s*:%s*"([^"]*)"') or "AM"
    local reason = string.match(obj_str, '"reason"%s*:%s*"([^"]*)"') or ""
    if sub_id ~= "" and pupil_id ~= "" then
      table.insert(submissions, {
        submission_id = sub_id,
        pupil_id = pupil_id,
        submitted_by_guardian_person_id = guardian_id,
        submission_type = sub_type,
        date = sub_date,
        session = session,
        reason = reason
      })
    end
  end
  return submissions
end

function pull_dmz_inbox(ctx, args)
  local academic_year_id = args.academic_year_id or "ay_2026_2027"
  local reference_date = args.reference_date or "2026-10-28"

  -- Structural Safety Invariant (§9.2 & §10.1): Outbound HTTP executes BEFORE opening any WriteDB transaction
  local resp = ctx.http.post(
    "https://dmz.parent.optimis.org.uk/v1/inbox/pull",
    { ["Content-Type"] = "application/json" },
    "{}"
  )

  if resp.status < 200 or resp.status >= 300 then
    error("DMZ_INBOX_PULL_FAILED: status " .. tostring(resp.status) .. " body=" .. tostring(resp.body))
  end

  local raw_submissions = parse_inbox_submissions_json(resp.body, reference_date)
  local processed = {}
  local accepted_count = 0
  local rejected_count = 0

  for _, sub in ipairs(raw_submissions) do
    local ok_pupil, _ = pcall(function()
      return ctx.dal.pupils.get_by_id(sub.pupil_id, {
        reference_date = sub.date,
        academic_year_id = academic_year_id
      })
    end)

    local status = "REJECTED_SAFEGUARDING_BLOCK"
    local guardian_name = ""

    if ok_pupil then
      local pr_guardians = ctx.dal.guardians.list_for_pupil(sub.pupil_id, {
        reference_date = sub.date,
        academic_year_id = academic_year_id,
        require_parental_responsibility = true
      })

      for _, g in ipairs(pr_guardians) do
        if g.guardian_person_id == sub.submitted_by_guardian_person_id and g.has_parental_responsibility then
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
      submission_id = sub.submission_id,
      pupil_id = sub.pupil_id,
      submitted_by_guardian_person_id = sub.submitted_by_guardian_person_id,
      guardian_name = guardian_name,
      submission_type = sub.submission_type,
      date = sub.date,
      session = sub.session,
      reason = sub.reason,
      status = status
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
          submission_type, date, session, reason, status, processed_by_staff_id
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)]],
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
        ctx.caller.staff_id or ""
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
  local academic_year_id = args.academic_year_id or "ay_2026_2027"
  local reference_date = args.reference_date or "2026-10-28"

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
  local quoted_guardian_ids = {}
  local redacted_count = 0

  for _, g in ipairs(all_guardians) do
    if g.is_confidential_address_redacted then
      redacted_count = redacted_count + 1
    end
    if g.has_parental_responsibility then
      table.insert(authorized_ids, g.guardian_person_id)
      table.insert(quoted_guardian_ids, json_escape(g.guardian_person_id))
    end
  end

  local att = ctx.dal.attendance.get_pupil_summary(pupil_id, {
    reference_date = reference_date,
    academic_year_id = academic_year_id
  })

  local present_sessions = att.present_sessions or 0
  local total_sessions = att.total_sessions or 0
  local authorized_guardians_json = "[" .. table.concat(quoted_guardian_ids, ",") .. "]"

  -- Explicitly strip statutory UPN and confidential household addresses from the outbound DMZ read-projection
  local sanitized_json = string.format(
    '{"pupil_id":%s,"first_name":%s,"last_name":%s,"academic_year_id":%s,"reference_date":%s,"present_sessions":%d,"total_sessions":%d,"authorized_guardian_ids":%s,"redacted_confidential_address_count":%d}',
    json_escape(pupil.id),
    json_escape(pupil.first_name),
    json_escape(pupil.last_name),
    json_escape(academic_year_id),
    json_escape(reference_date),
    present_sessions,
    total_sessions,
    authorized_guardians_json,
    redacted_count
  )

  -- Structural Safety Invariant (§9.2 & §10.1): Outbound HTTP executes BEFORE opening any WriteDB transaction
  local resp = ctx.http.post(
    "https://dmz.parent.optimis.org.uk/v1/projections/push",
    { ["Content-Type"] = "application/json" },
    sanitized_json
  )

  if resp.status < 200 or resp.status >= 300 then
    error("DMZ_PROJECTION_PUSH_FAILED: status " .. tostring(resp.status) .. " body=" .. tostring(resp.body))
  end

  local snap_id = "dmz_proj_" .. pupil.id .. "_" .. reference_date
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
        notes, pushed_by_staff_id
      ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, ?, ?, ?)]],
    {
      snap_id,
      pupil.id,
      pupil.first_name,
      pupil.last_name,
      academic_year_id,
      reference_date,
      present_sessions,
      total_sessions,
      authorized_guardians_json,
      redacted_count,
      sanitized_json,
      note_text,
      ctx.caller.staff_id or ""
    }
  )

  return {
    ok = true,
    data = {
      projection = {
        pupil_id = pupil.id,
        first_name = pupil.first_name,
        last_name = pupil.last_name,
        academic_year_id = academic_year_id,
        reference_date = reference_date,
        present_sessions = present_sessions,
        total_sessions = total_sessions,
        authorized_guardian_ids = authorized_ids,
        redacted_confidential_address_count = redacted_count
      },
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
