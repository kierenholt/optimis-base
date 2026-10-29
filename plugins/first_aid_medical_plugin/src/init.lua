local function resolve_pupil_id(args)
  if type(args) == "string" then
    if args == "" then
      error("pupil_id is required")
    end
    return args
  end
  if type(args) == "table" and type(args.pupil_id) == "string" and args.pupil_id ~= "" then
    return args.pupil_id
  end
  error("pupil_id is required")
end

local function require_non_empty_string(args, field_name)
  if type(args) ~= "table" then
    error(field_name .. " is required")
  end
  local val = args[field_name]
  if type(val) ~= "string" or val == "" then
    error(field_name .. " is required")
  end
  return val
end

local function load_pupil_first_aid_bundle(ctx, pupil_id, academic_year_id, reference_date)
  local opts = {
    academic_year_id = academic_year_id,
    reference_date = reference_date
  }

  local all_pupils = ctx.dal.pupils.list_all(opts)
  local selected_pupil = ctx.dal.pupils.get_by_id(pupil_id, opts)

  local sen_med_map = ctx.dal.pupils.list_sen_and_medical_batch({ pupil_id }, opts)
  local pupil_sen_med = sen_med_map[pupil_id] or {
    pupil_id = pupil_id,
    sen_provision_code = "N",
    has_ehcp = false,
    sen_needs = {},
    medical_conditions = {}
  }

  local raw_guardians = ctx.dal.guardians.list_for_pupil(pupil_id, opts)
  local safe_contacts = {}
  for i = 1, #raw_guardians do
    local g = raw_guardians[i]
    local can_collect = (g.authorised_to_collect == true) or (g.can_collect_pupil == true) or (g.can_collect_child == true)
    local collection_status = "RESTRICTED_COLLECTION"
    if can_collect then
      collection_status = "AUTHORISED_TO_COLLECT"
    end
    local address_summary = "No Household Address"
    if g.is_confidential_address_redacted then
      address_summary = "CONFIDENTIAL_ADDRESS_REDACTED"
    elseif type(g.address_line_1) == "string" and g.address_line_1 ~= "" then
      address_summary = g.address_line_1
      if type(g.town_city) == "string" and g.town_city ~= "" then
        address_summary = address_summary .. ", " .. g.town_city
      end
      if type(g.postcode) == "string" and g.postcode ~= "" then
        address_summary = address_summary .. " " .. g.postcode
      end
    end

    safe_contacts[#safe_contacts + 1] = {
      relationship_id = g.relationship_id,
      guardian_person_id = g.guardian_person_id,
      full_name = (g.first_name or "") .. " " .. (g.last_name or ""),
      relationship_code = g.relationship_code or "",
      has_parental_responsibility = g.has_parental_responsibility == true,
      can_collect_child = can_collect,
      collection_status = collection_status,
      emergency_contact_priority = g.contact_priority or g.emergency_priority or g.emergency_contact_priority or 99,
      phone_primary = g.phone_primary or "",
      address_summary = address_summary,
      is_confidential_address_redacted = g.is_confidential_address_redacted == true
    }
  end

  local treatments = ctx.plugin_db.query(
    "SELECT id, pupil_id, pupil_name, nc_year_group, incident_date, incident_location, " ..
    "injury_summary, treatment_given, outcome, critical_alert_snapshot, " ..
    "notified_guardian_id, notified_guardian_name, recorded_by_staff_id, created_at " ..
    "FROM plugin_first_aid.first_aid_treatments WHERE pupil_id = ? ORDER BY id DESC",
    { pupil_id }
  )

  local event_logs = ctx.plugin_db.query(
    "SELECT id, treatment_id, pupil_id, event_name, hop_count, outcome, dispatch_status, dispatched_at " ..
    "FROM plugin_first_aid.first_aid_event_logs WHERE pupil_id = ? ORDER BY id DESC",
    { pupil_id }
  )

  return {
    pupils = all_pupils,
    selected_pupil = selected_pupil,
    medical_conditions = pupil_sen_med.medical_conditions or {},
    sen_provision_code = pupil_sen_med.sen_provision_code or "N",
    has_ehcp = pupil_sen_med.has_ehcp == true,
    emergency_contacts = safe_contacts,
    treatments = treatments,
    event_logs = event_logs
  }
end

function get_pupil_first_aid_profile(ctx, args)
  local pupil_id = require_non_empty_string(args, "pupil_id")
  local academic_year_id = require_non_empty_string(args, "academic_year_id")
  local reference_date = require_non_empty_string(args, "reference_date")

  local bundle = load_pupil_first_aid_bundle(ctx, pupil_id, academic_year_id, reference_date)
  return {
    ok = true,
    data = bundle
  }
end

function log_first_aid_treatment(ctx, args)
  local pupil_id = require_non_empty_string(args, "pupil_id")
  local academic_year_id = require_non_empty_string(args, "academic_year_id")
  local reference_date = require_non_empty_string(args, "reference_date")
  local incident_location = require_non_empty_string(args, "incident_location")
  local injury_summary = require_non_empty_string(args, "injury_summary")
  local treatment_given = require_non_empty_string(args, "treatment_given")
  local outcome = require_non_empty_string(args, "outcome")

  local opts = {
    academic_year_id = academic_year_id,
    reference_date = reference_date
  }

  local pupil = ctx.dal.pupils.get_by_id(pupil_id, opts)
  local pupil_name = (pupil.preferred_first_name or pupil.legal_first_name or "") .. " " ..
                     (pupil.preferred_last_name or pupil.legal_last_name or "")

  local sen_med_map = ctx.dal.pupils.list_sen_and_medical_batch({ pupil_id }, opts)
  local pupil_sen_med = sen_med_map[pupil_id] or { medical_conditions = {} }
  local med_list = pupil_sen_med.medical_conditions or {}

  local alert_summary = "NONE"
  for i = 1, #med_list do
    local m = med_list[i]
    local ihcp_tag = m.has_ihcp and " [IHCP]" or ""
    alert_summary = (m.alert_level_id or "ALERT") .. ": " .. (m.condition_name or "") .. ihcp_tag
    break
  end

  local guardians = ctx.dal.guardians.list_for_pupil(pupil_id, opts)
  local notified_guardian_id = ""
  local notified_guardian_name = "None"
  if #guardians > 0 then
    local g = guardians[1]
    notified_guardian_id = g.guardian_person_id or ""
    notified_guardian_name = (g.first_name or "") .. " " .. (g.last_name or "")
  end

  local staff_id = (ctx.caller and ctx.caller.staff_id and ctx.caller.staff_id ~= "") and ctx.caller.staff_id or "SYSTEM"

  local exec_res = ctx.plugin_db.exec(
    "INSERT INTO plugin_first_aid.first_aid_treatments (" ..
    "pupil_id, pupil_name, nc_year_group, incident_date, incident_location, " ..
    "injury_summary, treatment_given, outcome, critical_alert_snapshot, " ..
    "notified_guardian_id, notified_guardian_name, recorded_by_staff_id" ..
    ") VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
    {
      pupil_id,
      pupil_name,
      pupil.nc_year_group or "",
      reference_date,
      incident_location,
      injury_summary,
      treatment_given,
      outcome,
      alert_summary,
      notified_guardian_id,
      notified_guardian_name,
      staff_id
    }
  )

  local treatment_id = exec_res.last_insert_id or 1
  local custom_event_name = "plugin.uk.org.optimis.medical.firstaid.logged"

  ctx.events.emit(custom_event_name, {
    treatment_id = treatment_id,
    pupil_id = pupil_id,
    pupil_name = pupil_name,
    incident_date = reference_date,
    incident_location = incident_location,
    outcome = outcome,
    critical_alert_snapshot = alert_summary,
    notified_guardian_id = notified_guardian_id
  })

  local bundle = load_pupil_first_aid_bundle(ctx, pupil_id, academic_year_id, reference_date)
  bundle.logged_treatment_id = treatment_id
  bundle.emitted_event = custom_event_name
  bundle.notified_guardian_name = notified_guardian_name

  return {
    ok = true,
    data = bundle
  }
end

function on_first_aid_logged(ctx, event)
  if type(event) ~= "table" then
    error("event envelope is required")
  end
  local payload = event.payload or {}
  local pupil_id = payload.pupil_id
  if type(pupil_id) ~= "string" or pupil_id == "" then
    error("event payload pupil_id is required")
  end
  local treatment_id = tonumber(payload.treatment_id) or 0
  local event_name = event.event_name or "plugin.uk.org.optimis.medical.firstaid.logged"
  local hop_count = tonumber(event.hop_count) or 1
  local outcome = payload.outcome or "RECORDED"

  ctx.plugin_db.exec(
    "INSERT INTO plugin_first_aid.first_aid_event_logs (" ..
    "treatment_id, pupil_id, event_name, hop_count, outcome, dispatch_status" ..
    ") VALUES (?, ?, ?, ?, ?, ?)",
    {
      treatment_id,
      pupil_id,
      event_name,
      hop_count,
      outcome,
      "DISPATCHED_POST_COMMIT"
    }
  )

  return {
    ok = true,
    data = {
      treatment_id = treatment_id,
      pupil_id = pupil_id,
      event_name = event_name,
      dispatch_status = "DISPATCHED_POST_COMMIT"
    }
  }
end

function export_pupil_data(ctx, args)
  local pupil_id = resolve_pupil_id(args)

  local treatments = ctx.plugin_db.query(
    "SELECT id, pupil_id, pupil_name, nc_year_group, incident_date, incident_location, " ..
    "injury_summary, treatment_given, outcome, critical_alert_snapshot, " ..
    "notified_guardian_id, notified_guardian_name, recorded_by_staff_id, created_at " ..
    "FROM plugin_first_aid.first_aid_treatments WHERE pupil_id = ? ORDER BY id ASC",
    { pupil_id }
  )

  local event_logs = ctx.plugin_db.query(
    "SELECT id, treatment_id, pupil_id, event_name, hop_count, outcome, dispatch_status, dispatched_at " ..
    "FROM plugin_first_aid.first_aid_event_logs WHERE pupil_id = ? ORDER BY id ASC",
    { pupil_id }
  )

  return {
    ok = true,
    data = {
      pupil_id = pupil_id,
      sections = {
        {
          title = "Medical Room & First Aid Treatments",
          records = treatments,
          redactable_fields = { "injury_summary", "treatment_given" }
        },
        {
          title = "First Aid Post-Commit Event Audit Logs",
          records = event_logs,
          redactable_fields = { "outcome" }
        }
      }
    }
  }
end

function purge_pupil_data(ctx, args)
  local pupil_id = resolve_pupil_id(args)

  local del_events = ctx.plugin_db.exec(
    "DELETE FROM plugin_first_aid.first_aid_event_logs WHERE pupil_id = ?",
    { pupil_id }
  )
  local del_treatments = ctx.plugin_db.exec(
    "DELETE FROM plugin_first_aid.first_aid_treatments WHERE pupil_id = ?",
    { pupil_id }
  )

  local purged_rows = (del_events.rows_affected or 0) + (del_treatments.rows_affected or 0)
  return {
    ok = true,
    data = {
      pupil_id = pupil_id,
      purged = true,
      purged_rows = purged_rows
    }
  }
end
