---@type OptimisCtx
local _ = nil

local DEFAULT_SPARX_SYNC_URL = "https://sync.sparxmaths.uk/v1/homework/sync"

local function mask_api_key(raw_key)
  local k = tostring(raw_key or "")
  if #k <= 8 then
    return "spx_...masked"
  end
  local prefix = string.sub(k, 1, 4)
  local suffix = string.match(k, "(%d%d%d%d)%D*$") or string.sub(k, -4)
  return prefix .. "..." .. suffix
end

function configure_vendor_credentials(ctx, args)
  local api_key = tostring(args.api_key or "")
  local sync_endpoint_url = tostring(args.sync_endpoint_url or "")

  if api_key == "" then
    error("missing mandatory parameter: api_key")
  end
  if sync_endpoint_url == "" then
    sync_endpoint_url = DEFAULT_SPARX_SYNC_URL
  end

  -- 1C: Encrypt vendor API key at rest using Host-managed AES-256-GCM envelope (HOST_KMS_KEY_HANDLE + plugin_id)
  local sealed_api_key = ctx.crypto.seal_secret(api_key)
  local key_preview = mask_api_key(api_key)
  local staff_id = tostring(ctx.caller.staff_id or "")
  local updated_at = ctx.clock.now_utc()

  ctx.plugin_db.exec(
    [[INSERT INTO plugin_sparx.sparx_vendor_config (
        config_key, api_key, key_preview, sync_endpoint_url, updated_by_staff_id, updated_at
      ) VALUES ('primary', ?, ?, ?, ?, ?)
      ON CONFLICT(config_key) DO UPDATE SET
        api_key = excluded.api_key,
        key_preview = excluded.key_preview,
        sync_endpoint_url = excluded.sync_endpoint_url,
        updated_by_staff_id = excluded.updated_by_staff_id,
        updated_at = excluded.updated_at]],
    {
      sealed_api_key,
      key_preview,
      sync_endpoint_url,
      staff_id,
      updated_at
    }
  )

  return {
    ok = true,
    data = {
      status = "CONFIGURED",
      is_sealed_envelope = (string.sub(sealed_api_key, 1, 7) == "enc:v1:"),
      key_preview = key_preview,
      sync_endpoint_url = sync_endpoint_url,
      updated_by_staff_id = staff_id,
      updated_at = updated_at
    }
  }
end

function get_vendor_config(ctx, args)
  local rows = ctx.plugin_db.query(
    [[SELECT config_key, api_key, key_preview, sync_endpoint_url, updated_by_staff_id, updated_at
      FROM plugin_sparx.sparx_vendor_config
      WHERE config_key = 'primary']],
    {}
  )

  if #rows == 0 then
    return {
      ok = true,
      data = {
        status = "UNCONFIGURED",
        is_sealed_envelope = false,
        key_preview = "",
        sync_endpoint_url = DEFAULT_SPARX_SYNC_URL
      }
    }
  end

  local cfg = rows[1]
  local stored_cipher = tostring(cfg.api_key or "")
  return {
    ok = true,
    data = {
      status = "CONFIGURED",
      is_sealed_envelope = (string.sub(stored_cipher, 1, 7) == "enc:v1:"),
      key_preview = tostring(cfg.key_preview or ""),
      sync_endpoint_url = tostring(cfg.sync_endpoint_url or DEFAULT_SPARX_SYNC_URL),
      updated_by_staff_id = tostring(cfg.updated_by_staff_id or ""),
      updated_at = tostring(cfg.updated_at or "")
    }
  }
end

function sync_group_homework(ctx, args)
  local group_id = tostring(args.group_id or "")
  local academic_year_id = tostring(args.academic_year_id or "")
  local reference_date = tostring(args.reference_date or "")
  local homework_week = tostring(args.homework_week or "2026-W44")
  local simulate_fault = tostring(args.simulate_fault or "")

  if group_id == "" then
    error("missing mandatory parameter: group_id")
  end
  if academic_year_id == "" then
    academic_year_id = tostring(ctx.school.academic_year_id or "ay_2026_2027")
  end
  if reference_date == "" then
    reference_date = tostring(ctx.school.reference_date or "2026-10-28")
  end

  -- =========================================================================
  -- PHASE 1: READ & EGRESS (Zero open SQLite WriteDB transaction)
  -- =========================================================================

  -- 1a. Read committed sealed vendor API key from plugin_sparx.db via ReadDB and decrypt via ctx.crypto.open_secret (1C)
  local cfg_rows = ctx.plugin_db.query(
    [[SELECT api_key, key_preview, sync_endpoint_url
      FROM plugin_sparx.sparx_vendor_config
      WHERE config_key = 'primary']],
    {}
  )
  if #cfg_rows == 0 or tostring(cfg_rows[1].api_key or "") == "" then
    error("SPARX_API_KEY_NOT_CONFIGURED: configure vendor API key in plugin_sparx.db before triggering sync")
  end

  local sealed_api_key = tostring(cfg_rows[1].api_key)
  local vendor_api_key = ctx.crypto.open_secret(sealed_api_key)
  local target_url = tostring(cfg_rows[1].sync_endpoint_url or DEFAULT_SPARX_SYNC_URL)
  if target_url == "" then
    target_url = DEFAULT_SPARX_SYNC_URL
  end

  -- 1b. Enforce :own_groups assignment guard and fetch teaching group roster via Core DAL (ReadDB)
  ctx.dal.groups.assert_caller_assigned(group_id, {
    academic_year_id = academic_year_id,
    reference_date = reference_date
  })

  local roster = ctx.dal.pupils.list_by_group(group_id, {
    academic_year_id = academic_year_id,
    reference_date = reference_date
  })

  local pupil_map = {}
  local outbound_roster = {}
  for _, p in ipairs(roster) do
    local pid = tostring(p.id)
    local full_name = tostring(p.preferred_first_name or p.first_name or "") .. " " .. tostring(p.preferred_last_name or p.last_name or "")
    pupil_map[pid] = full_name
    table.insert(outbound_roster, {
      pupil_id = pid,
      first_name = tostring(p.preferred_first_name or p.first_name or ""),
      last_name = tostring(p.preferred_last_name or p.last_name or ""),
      nc_year_group = tostring(p.nc_year_group or "")
    })
  end

  -- Optional simulated fault 1: Verify diagnostic ring buffer deterministically strips opened API keys (1C) and pupil IDs
  if simulate_fault == "VENDOR_AUTH_FAULT_WITH_STORED_KEY" then
    local sample_pupil = ""
    if #outbound_roster > 0 then
      sample_pupil = outbound_roster[1].pupil_id
    end
    error(string.format(
      "SPARX_PARTNER_SYNC_FAULT: handshake failed with Authorization: Bearer %s (x-sparx-api-key=%s) for pupil %s",
      vendor_api_key,
      vendor_api_key,
      sample_pupil
    ))
  end

  -- Optional simulated fault 2: Verify Host halts open WriteDB transaction if write occurs before ctx.http.post
  if simulate_fault == "WRITE_TX_BEFORE_EGRESS" then
    ctx.plugin_db.exec(
      [[INSERT INTO plugin_sparx.sparx_sync_runs (
          id, sync_batch_id, group_id, academic_year_id, reference_date, homework_week,
          roster_count, persisted_scores_count, target_url, http_status, status,
          two_phase_pattern, synced_by_staff_id, synced_at
        ) VALUES (?, 'illegal_tx_batch', ?, ?, ?, ?, ?, 0, ?, 0, 'PENDING_ILLEGAL_TX', 'VIOLATION', ?, ?)]],
      {
        "spx_illegal_" .. ctx.crypto.uuid(),
        group_id,
        academic_year_id,
        reference_date,
        homework_week,
        #outbound_roster,
        target_url,
        tostring(ctx.caller.staff_id or ""),
        ctx.clock.now_utc()
      }
    )
  end

  -- 1c. Outbound HTTP POST to https://sync.sparxmaths.uk (with NO open WriteDB transaction)
  local req_payload = ctx.json.encode({
    school_urn = tostring(ctx.school.urn or ""),
    group_id = group_id,
    academic_year_id = academic_year_id,
    reference_date = reference_date,
    homework_week = homework_week,
    roster_count = #outbound_roster,
    pupils = outbound_roster
  })

  local resp = ctx.http.post(
    target_url,
    {
      ["Content-Type"] = "application/json",
      ["Authorization"] = "Bearer " .. vendor_api_key,
      ["X-Sparx-Partner-SDK"] = "uk.co.sparx.maths"
    },
    req_payload
  )

  local http_status = tonumber(resp.status) or 0
  if not resp.ok or http_status < 200 or http_status >= 300 then
    error(string.format("SPARX_EGRESS_ERROR: partner endpoint returned HTTP %d", http_status))
  end

  local decoded = {}
  if resp.body and resp.body ~= "" then
    decoded = ctx.json.decode(resp.body) or {}
  end

  local sync_batch_id = tostring(decoded.sync_batch_id or ("sparx_batch_" .. ctx.crypto.uuid()))
  local returned_week = tostring(decoded.homework_week or homework_week)
  local raw_scores = decoded.scores or {}

  -- =========================================================================
  -- PHASE 2: WRITEDB PERSISTENCE (Opened only AFTER ctx.http.post completes)
  -- =========================================================================
  local synced_at = ctx.clock.now_utc()
  local staff_id = tostring(ctx.caller.staff_id or "")
  local persisted_scores = {}
  local batch_rows = {}

  for _, item in ipairs(raw_scores) do
    local pid = tostring(item.pupil_id or "")
    -- Only persist scores for pupils verified to be in the caller's :own_groups roster
    if pid ~= "" and pupil_map[pid] ~= nil then
      local score_id = "spx_score_" .. ctx.crypto.uuid()
      local p_name = pupil_map[pid]
      local w_label = tostring(item.homework_week or returned_week)
      local completion_rate = tonumber(item.completion_rate) or 0
      local xp_earned = tonumber(item.xp_earned) or 0
      local bookwork_accuracy = tonumber(item.bookwork_accuracy) or 0
      local teacher_note = tostring(item.teacher_note or "")

      table.insert(batch_rows, {
        score_id,
        sync_batch_id,
        group_id,
        pid,
        p_name,
        academic_year_id,
        w_label,
        completion_rate,
        xp_earned,
        bookwork_accuracy,
        teacher_note,
        staff_id,
        synced_at
      })

      table.insert(persisted_scores, {
        id = score_id,
        sync_batch_id = sync_batch_id,
        group_id = group_id,
        pupil_id = pid,
        pupil_name = p_name,
        academic_year_id = academic_year_id,
        homework_week = w_label,
        completion_rate = completion_rate,
        xp_earned = xp_earned,
        bookwork_accuracy = bookwork_accuracy,
        teacher_note = teacher_note,
        synced_by_staff_id = staff_id,
        synced_at = synced_at
      })
    end
  end

  if #batch_rows > 0 then
    ctx.plugin_db.exec_batch(
      [[INSERT INTO plugin_sparx.sparx_homework_scores (
          id, sync_batch_id, group_id, pupil_id, pupil_name, academic_year_id,
          homework_week, completion_rate, xp_earned, bookwork_accuracy,
          teacher_note, synced_by_staff_id, synced_at
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)]],
      batch_rows
    )
  end

  local run_id = "spx_run_" .. ctx.crypto.uuid()
  ctx.plugin_db.exec(
    [[INSERT INTO plugin_sparx.sparx_sync_runs (
        id, sync_batch_id, group_id, academic_year_id, reference_date, homework_week,
        roster_count, persisted_scores_count, target_url, http_status, status,
        two_phase_pattern, synced_by_staff_id, synced_at
      ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'SYNCED', 'READ_EGRESS_THEN_WRITEDB_OK', ?, ?)]],
    {
      run_id,
      sync_batch_id,
      group_id,
      academic_year_id,
      reference_date,
      returned_week,
      #outbound_roster,
      #persisted_scores,
      target_url,
      http_status,
      staff_id,
      synced_at
    }
  )

  return {
    ok = true,
    data = {
      run_id = run_id,
      status = "SYNCED",
      two_phase_pattern = "READ_EGRESS_THEN_WRITEDB_OK",
      sync_batch_id = sync_batch_id,
      group_id = group_id,
      academic_year_id = academic_year_id,
      reference_date = reference_date,
      homework_week = returned_week,
      roster_count = #outbound_roster,
      persisted_scores_count = #persisted_scores,
      scores = persisted_scores,
      synced_at = synced_at
    }
  }
end

function list_homework_scores(ctx, args)
  local group_id = tostring(args.group_id or "")
  local pupil_id = tostring(args.pupil_id or "")

  local rows
  if group_id ~= "" and pupil_id ~= "" then
    rows = ctx.plugin_db.query(
      [[SELECT id, sync_batch_id, group_id, pupil_id, pupil_name, academic_year_id,
               homework_week, completion_rate, xp_earned, bookwork_accuracy,
               teacher_note, synced_by_staff_id, synced_at
        FROM plugin_sparx.sparx_homework_scores
        WHERE group_id = ? AND pupil_id = ?
        ORDER BY synced_at DESC, id DESC]],
      { group_id, pupil_id }
    )
  elseif group_id ~= "" then
    rows = ctx.plugin_db.query(
      [[SELECT id, sync_batch_id, group_id, pupil_id, pupil_name, academic_year_id,
               homework_week, completion_rate, xp_earned, bookwork_accuracy,
               teacher_note, synced_by_staff_id, synced_at
        FROM plugin_sparx.sparx_homework_scores
        WHERE group_id = ?
        ORDER BY synced_at DESC, id DESC]],
      { group_id }
    )
  elseif pupil_id ~= "" then
    rows = ctx.plugin_db.query(
      [[SELECT id, sync_batch_id, group_id, pupil_id, pupil_name, academic_year_id,
               homework_week, completion_rate, xp_earned, bookwork_accuracy,
               teacher_note, synced_by_staff_id, synced_at
        FROM plugin_sparx.sparx_homework_scores
        WHERE pupil_id = ?
        ORDER BY synced_at DESC, id DESC]],
      { pupil_id }
    )
  else
    rows = ctx.plugin_db.query(
      [[SELECT id, sync_batch_id, group_id, pupil_id, pupil_name, academic_year_id,
               homework_week, completion_rate, xp_earned, bookwork_accuracy,
               teacher_note, synced_by_staff_id, synced_at
        FROM plugin_sparx.sparx_homework_scores
        ORDER BY synced_at DESC, id DESC]],
      {}
    )
  end

  return {
    ok = true,
    data = {
      scores = rows
    }
  }
end

function export_pupil_data(ctx, args)
  local pupil_id = tostring(args.pupil_id or "")
  if pupil_id == "" then
    error("missing mandatory parameter: pupil_id")
  end

  local rows = ctx.plugin_db.query(
    [[SELECT id, sync_batch_id, group_id, pupil_id, pupil_name, academic_year_id,
             homework_week, completion_rate, xp_earned, bookwork_accuracy,
             teacher_note, synced_by_staff_id, synced_at
      FROM plugin_sparx.sparx_homework_scores
      WHERE pupil_id = ?
      ORDER BY synced_at ASC, id ASC]],
    { pupil_id }
  )

  return {
    ok = true,
    data = {
      sections = {
        {
          title = "Sparx Maths Homework Completion Scores",
          records = rows,
          redactable_fields = { "teacher_note" }
        }
      }
    }
  }
end

function purge_pupil_data(ctx, args)
  local pupil_id = tostring(args.pupil_id or "")
  if pupil_id == "" then
    error("missing mandatory parameter: pupil_id")
  end

  local res = ctx.plugin_db.exec(
    "DELETE FROM plugin_sparx.sparx_homework_scores WHERE pupil_id = ?",
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
