local function trim(s)
  if type(s) ~= "string" then
    return ""
  end
  return (s:gsub("^%s+", ""):gsub("%s+$", ""))
end

local function build_scope_opts(ctx, args)
  local school = (ctx and ctx.school) or {}
  local opts = {
    strict_own_groups = true
  }
  if type(args) == "table" then
    if trim(args.academic_year_id) ~= "" then
      opts.academic_year_id = trim(args.academic_year_id)
    elseif trim(school.academic_year_id) ~= "" then
      opts.academic_year_id = trim(school.academic_year_id)
    end
    if trim(args.reference_date) ~= "" then
      opts.reference_date = trim(args.reference_date)
    elseif trim(school.reference_date) ~= "" then
      opts.reference_date = trim(school.reference_date)
    end
  end
  return opts
end

function get_widget_gallery_summary(ctx, args)
  if type(ctx) ~= "table" or type(ctx.dal) ~= "table" then
    error("missing mandatory ctx.dal table")
  end
  if type(args) ~= "table" then
    error("missing mandatory arguments table for get_widget_gallery_summary")
  end

  local group_id = trim(args.group_id)
  if group_id == "" then
    error("missing mandatory parameter: group_id")
  end

  local opts = build_scope_opts(ctx, args)
  ctx.dal.groups.assert_caller_assigned(group_id, opts)

  local group_code = group_id
  local group_name = group_id
  if ctx.caller and trim(ctx.caller.staff_id) ~= "" and ctx.dal.groups.list_for_staff then
    local assigned = ctx.dal.groups.list_for_staff(ctx.caller.staff_id, opts)
    if type(assigned) == "table" then
      for i = 1, #assigned do
        local g = assigned[i]
        if g.id == group_id then
          group_code = g.code or group_id
          group_name = g.name or group_id
          break
        end
      end
    end
  end

  local roster = ctx.dal.pupils.list_by_group(group_id, opts)
  local pupil_ids = {}
  local pupils_summary = {}

  for i = 1, #roster do
    local p = roster[i]
    pupil_ids[#pupil_ids + 1] = p.id
    pupils_summary[#pupils_summary + 1] = {
      id = p.id,
      upn = p.upn,
      first_name = p.first_name,
      last_name = p.last_name
    }
  end

  local sen_count = 0
  local medical_count = 0
  if #pupil_ids > 0 and ctx.dal.pupils.list_sen_and_medical_batch then
    local flags = ctx.dal.pupils.list_sen_and_medical_batch(pupil_ids, opts)
    if type(flags) == "table" then
      for _, f in pairs(flags) do
        if type(f) == "table" then
          if f.sen_provision_code and f.sen_provision_code ~= "" and f.sen_provision_code ~= "N" then
            sen_count = sen_count + 1
          end
          if f.has_medical_condition or (f.medical_conditions_summary and f.medical_conditions_summary ~= "") or (type(f.medical_conditions) == "table" and #f.medical_conditions > 0) then
            medical_count = medical_count + 1
          end
        end
      end
    end
  end

  return {
    ok = true,
    data = {
      group_id = group_id,
      group_code = group_code,
      group_name = group_name,
      active_pupil_count = #pupils_summary,
      sen_alert_count = sen_count,
      medical_alert_count = medical_count,
      pupils = pupils_summary
    }
  }
end

function get_pupil_quick_card(ctx, args)
  if type(ctx) ~= "table" or type(ctx.dal) ~= "table" then
    error("missing mandatory ctx.dal table")
  end
  if type(args) ~= "table" then
    error("missing mandatory arguments table for get_pupil_quick_card")
  end

  local pupil_id = trim(args.pupil_id)
  if pupil_id == "" then
    error("missing mandatory parameter: pupil_id")
  end

  local opts = build_scope_opts(ctx, args)
  local pupil = ctx.dal.pupils.get_by_id(pupil_id, opts)
  if not pupil or not pupil.id then
    error("pupil record not found or not in caller's own_groups")
  end

  return {
    ok = true,
    data = {
      id = pupil.id,
      upn = pupil.upn,
      first_name = pupil.first_name,
      last_name = pupil.last_name
    }
  }
end
