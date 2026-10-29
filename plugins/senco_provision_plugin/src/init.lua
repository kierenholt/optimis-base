local function build_pupil_sen_rollup(ctx, pupil, sen_summary)
  local rows = ctx.plugin_db.query(
    [[SELECT id, pupil_id, academic_year_id, sen_provision_code, primary_sen_need_code,
             iep_target, classroom_adjustment, ta_allocation_hours, assigned_ta_name,
             review_notes, logged_by_staff_id, effective_date, created_at
      FROM plugin_sen_provision.sen_provisions
      WHERE pupil_id = ?
      ORDER BY created_at ASC, id ASC]],
    { pupil.id }
  )

  local total_ta_hours = 0
  local latest_iep_target = ""
  local latest_classroom_adjustment = ""
  local latest_assigned_ta_name = ""

  for i = 1, #rows do
    local hrs = tonumber(rows[i].ta_allocation_hours) or 0
    total_ta_hours = total_ta_hours + hrs
    latest_iep_target = rows[i].iep_target or ""
    latest_classroom_adjustment = rows[i].classroom_adjustment or ""
    latest_assigned_ta_name = rows[i].assigned_ta_name or ""
  end

  local first_name = pupil.preferred_first_name or pupil.first_name or ""
  local last_name = pupil.preferred_last_name or pupil.last_name or ""
  local src = sen_summary or pupil
  local prov_code = (src and src.sen_provision_code) or "N"
  local need_code = (src and src.primary_sen_need_code) or ""
  local sec_need_code = (src and src.secondary_sen_need_code) or ""
  local strategies = (src and src.teaching_strategies) or ""
  local pupil_targets = (src and src.pupil_targets) or ""
  local senco_notes = (src and src.senco_notes) or ""
  local ehcp_start_date = (src and src.ehcp_start_date) or ""
  local ehcp_review_due_date = (src and src.ehcp_review_due_date) or ""
  local next_annual_review_date = (src and src.next_annual_review_date) or ehcp_review_due_date
  local sen_needs = (src and src.sen_needs) or {}
  local sen_assistants = (src and src.sen_assistants) or {}
  local med_conditions = (src and src.medical_conditions) or {}

  return {
    pupil_id = pupil.id,
    pupil_name = first_name .. " " .. last_name,
    preferred_first_name = first_name,
    preferred_last_name = last_name,
    nc_year_group = pupil.nc_year_group or "",
    academic_year_id = pupil.academic_year_id or (ctx.school and ctx.school.academic_year_id) or "",
    sen_provision_code = prov_code,
    primary_sen_need_code = need_code,
    secondary_sen_need_code = sec_need_code,
    teaching_strategies = strategies,
    pupil_targets = pupil_targets,
    senco_notes = senco_notes,
    ehcp_start_date = ehcp_start_date,
    ehcp_review_due_date = ehcp_review_due_date,
    next_annual_review_date = next_annual_review_date,
    sen_needs = sen_needs,
    sen_assistants = sen_assistants,
    medical_conditions = med_conditions,
    total_ta_hours = total_ta_hours,
    provision_count = #rows,
    latest_iep_target = latest_iep_target,
    latest_classroom_adjustment = latest_classroom_adjustment,
    latest_assigned_ta_name = latest_assigned_ta_name,
    provisions = rows
  }
end

local function build_sen_register_pupils(ctx, group_id, opts)
  local reg_opts = {
    academic_year_id = opts.academic_year_id,
    reference_date = opts.reference_date,
    strict_own_groups = opts.strict_own_groups,
    group_id = group_id or ""
  }
  local entries = {}
  if ctx.dal.sen and ctx.dal.sen.list_register then
    entries = ctx.dal.sen.list_register(reg_opts)
  elseif ctx.dal.pupils and ctx.dal.pupils.list_sen_register then
    entries = ctx.dal.pupils.list_sen_register(reg_opts)
  end

  local sen_pupils = {}
  for i = 1, #entries do
    local entry = entries[i]
    sen_pupils[i] = build_pupil_sen_rollup(ctx, entry, entry)
  end
  return sen_pupils
end

function log_iep_provision(ctx, args)
  if type(args) ~= "table" or type(args.pupil_id) ~= "string" or args.pupil_id == "" then
    error("missing mandatory parameter: pupil_id")
  end
  if type(args.iep_target) ~= "string" or args.iep_target == "" then
    error("missing mandatory parameter: iep_target")
  end
  if type(args.classroom_adjustment) ~= "string" or args.classroom_adjustment == "" then
    error("missing mandatory parameter: classroom_adjustment")
  end
  local ta_hours = tonumber(args.ta_allocation_hours)
  if ta_hours == nil or ta_hours < 0 then
    error("missing or invalid mandatory numeric parameter: ta_allocation_hours")
  end
  if type(args.assigned_ta_name) ~= "string" or args.assigned_ta_name == "" then
    error("missing mandatory parameter: assigned_ta_name")
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
    error("pupil not found or inaccessible")
  end

  local sen_batch = ctx.dal.pupils.list_sen_and_medical_batch({ pupil.id }, opts)
  local sen_summary = sen_batch and sen_batch[pupil.id] or nil
  local prov_code = sen_summary and sen_summary.sen_provision_code or "N"
  if prov_code ~= "K" and prov_code ~= "E" then
    error("pupil does not have active K or E SEN provision")
  end

  local provision_id = args.provision_id
  if type(provision_id) ~= "string" or provision_id == "" then
    provision_id = "iep_" .. ctx.crypto.uuid()
  end

  local effective_date = args.reference_date
  if type(effective_date) ~= "string" or effective_date == "" then
    effective_date = school_ref_date ~= "" and school_ref_date or (pupil.effective_from or "")
  end

  local ay_id = args.academic_year_id
  if type(ay_id) ~= "string" or ay_id == "" then
    ay_id = pupil.academic_year_id or school_ay
  end

  local review_notes = type(args.review_notes) == "string" and args.review_notes or ""
  local created_at = ctx.clock.now_utc()

  ctx.plugin_db.exec(
    [[INSERT INTO plugin_sen_provision.sen_provisions (
        id, pupil_id, academic_year_id, sen_provision_code, primary_sen_need_code,
        iep_target, classroom_adjustment, ta_allocation_hours, assigned_ta_name,
        review_notes, logged_by_staff_id, effective_date, created_at
      ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)]],
    {
      provision_id,
      pupil.id,
      ay_id,
      prov_code,
      sen_summary.primary_sen_need_code or "",
      args.iep_target,
      args.classroom_adjustment,
      ta_hours,
      args.assigned_ta_name,
      review_notes,
      ctx.caller.staff_id or "",
      effective_date,
      created_at
    }
  )

  local rollup = build_pupil_sen_rollup(ctx, pupil, sen_summary)
  rollup.logged_provision_id = provision_id
  if type(args.group_id) == "string" and args.group_id ~= "" then
    rollup.sen_pupils = build_sen_register_pupils(ctx, args.group_id, opts)
  end

  ctx.events.emit("sen.provision_logged", {
    provision_id = provision_id,
    pupil_id = pupil.id,
    sen_provision_code = prov_code,
    ta_allocation_hours = ta_hours
  })

  return {
    ok = true,
    data = rollup
  }
end

function get_sen_differentiation_summary(ctx, args)
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

  if type(args.pupil_ids) == "table" and #args.pupil_ids > 0 then
    local pupils = ctx.dal.pupils.get_by_ids_batch(args.pupil_ids, opts)
    local sen_batch = ctx.dal.pupils.list_sen_and_medical_batch(args.pupil_ids, opts)
    local sen_pupils = {}
    local idx = 1
    for i = 1, #pupils do
      local p = pupils[i]
      local s = sen_batch and sen_batch[p.id] or nil
      local code = s and s.sen_provision_code or "N"
      if code == "K" or code == "E" then
        sen_pupils[idx] = build_pupil_sen_rollup(ctx, p, s)
        idx = idx + 1
      end
    end
    return {
      ok = true,
      data = {
        sen_pupils = sen_pupils
      }
    }
  end

  local sen_pupils = {}
  if type(args.group_id) == "string" and args.group_id ~= "" then
    sen_pupils = build_sen_register_pupils(ctx, args.group_id, opts)
  end

  if type(args.pupil_id) == "string" and args.pupil_id ~= "" then
    local pupil = ctx.dal.pupils.get_by_id(args.pupil_id, opts)
    if not pupil or not pupil.id then
      error("pupil not found or inaccessible under own_groups scope")
    end
    local sen_batch = ctx.dal.pupils.list_sen_and_medical_batch({ pupil.id }, opts)
    local sen_summary = sen_batch and sen_batch[pupil.id] or nil
    local rollup = build_pupil_sen_rollup(ctx, pupil, sen_summary)
    if #sen_pupils == 0 and (rollup.sen_provision_code == "K" or rollup.sen_provision_code == "E") then
      sen_pupils = { rollup }
    end
    rollup.sen_pupils = sen_pupils
    return {
      ok = true,
      data = rollup
    }
  end

  if type(args.group_id) == "string" and args.group_id ~= "" then
    return {
      ok = true,
      data = {
        group_id = args.group_id,
        sen_pupils = sen_pupils
      }
    }
  end

  -- 1G: When neither pupil_id nor group_id is provided, query ctx.dal.sen.list_register across the caller's scoped SEN register
  local register_pupils = build_sen_register_pupils(ctx, "", opts)
  return {
    ok = true,
    data = {
      sen_pupils = register_pupils
    }
  }
end

function export_pupil_data(ctx, args)
  local pupil_id = type(args) == "table" and args.pupil_id or args
  if type(pupil_id) ~= "string" or pupil_id == "" then
    error("missing mandatory parameter: pupil_id")
  end

  local rows = ctx.plugin_db.query(
    [[SELECT id, pupil_id, academic_year_id, sen_provision_code, primary_sen_need_code,
             iep_target, classroom_adjustment, ta_allocation_hours, assigned_ta_name,
             review_notes, logged_by_staff_id, effective_date, created_at
      FROM plugin_sen_provision.sen_provisions
      WHERE pupil_id = ?
      ORDER BY created_at ASC, id ASC]],
    { pupil_id }
  )

  return {
    ok = true,
    data = {
      {
        title = "Plugin SEN IEP & TA Provision Records",
        records = rows,
        redactable_fields = { "iep_target", "classroom_adjustment", "review_notes" }
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
    "DELETE FROM plugin_sen_provision.sen_provisions WHERE pupil_id = ?",
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
