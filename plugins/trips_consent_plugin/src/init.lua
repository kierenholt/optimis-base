local function resolve_temporal_opts(ctx, args, fallback_ay)
  local school_ctx = ctx.school or {}
  local academic_year_id = (type(args) == "table" and args.academic_year_id) or fallback_ay or school_ctx.academic_year_id
  local reference_date = (type(args) == "table" and args.reference_date) or school_ctx.reference_date
  if type(academic_year_id) ~= "string" or academic_year_id == "" then
    error("missing mandatory parameter: academic_year_id")
  end
  if type(reference_date) ~= "string" or reference_date == "" then
    error("missing mandatory parameter: reference_date")
  end
  return {
    academic_year_id = academic_year_id,
    reference_date = reference_date
  }
end

function create_trip(ctx, args)
  if type(args) ~= "table" then
    error("missing mandatory arguments table")
  end
  if type(args.group_id) ~= "string" or args.group_id == "" then
    error("missing mandatory parameter: group_id")
  end
  if type(args.trip_name) ~= "string" or args.trip_name == "" then
    error("missing mandatory parameter: trip_name")
  end
  if type(args.destination) ~= "string" or args.destination == "" then
    error("missing mandatory parameter: destination")
  end
  if type(args.trip_date) ~= "string" or args.trip_date == "" then
    error("missing mandatory parameter: trip_date")
  end

  local opts = resolve_temporal_opts(ctx, args, nil)

  -- Enforce :own_groups scoping and resolve active on-roll group roster via ctx.dal.pupils.list_by_group
  local roster = ctx.dal.pupils.list_by_group(args.group_id, opts)

  local trip_id = args.trip_id
  if type(trip_id) ~= "string" or trip_id == "" then
    trip_id = "trp_" .. ctx.crypto.uuid()
  end

  local created_at = ctx.clock.now_utc()
  local lead_staff_id = (ctx.caller and ctx.caller.staff_id) or ""

  ctx.plugin_db.exec(
    [[INSERT INTO plugin_trips.trips (
        id, group_id, academic_year_id, trip_name, destination, trip_date, lead_staff_id, created_at
      ) VALUES (?, ?, ?, ?, ?, ?, ?, ?)]],
    {
      trip_id,
      args.group_id,
      opts.academic_year_id,
      args.trip_name,
      args.destination,
      args.trip_date,
      lead_staff_id,
      created_at
    }
  )

  ctx.events.emit("trips.trip_created", {
    trip_id = trip_id,
    group_id = args.group_id,
    trip_name = args.trip_name,
    destination = args.destination,
    trip_date = args.trip_date
  })

  return {
    ok = true,
    data = {
      trip_id = trip_id,
      group_id = args.group_id,
      academic_year_id = opts.academic_year_id,
      trip_name = args.trip_name,
      destination = args.destination,
      trip_date = args.trip_date,
      lead_staff_id = lead_staff_id,
      created_at = created_at,
      eligible_pupil_count = #roster,
      roster = roster
    }
  }
end

function record_trip_consent(ctx, args)
  if type(args) ~= "table" then
    error("missing mandatory arguments table")
  end
  if type(args.trip_id) ~= "string" or args.trip_id == "" then
    error("missing mandatory parameter: trip_id")
  end
  if type(args.pupil_id) ~= "string" or args.pupil_id == "" then
    error("missing mandatory parameter: pupil_id")
  end
  if type(args.guardian_person_id) ~= "string" or args.guardian_person_id == "" then
    error("missing mandatory parameter: guardian_person_id")
  end

  local trip_rows = ctx.plugin_db.query(
    [[SELECT id, group_id, academic_year_id, trip_name, destination, trip_date, lead_staff_id, created_at
      FROM plugin_trips.trips
      WHERE id = ?]],
    { args.trip_id }
  )
  if #trip_rows == 0 then
    error("trip not found: " .. tostring(args.trip_id))
  end
  local trip = trip_rows[1]

  local opts = resolve_temporal_opts(ctx, args, trip.academic_year_id)

  -- Verify pupil is an active member of the trip's group under caller's :own_groups scope
  local group_pupils = ctx.dal.pupils.list_by_group(trip.group_id, opts)
  local matched_pupil = nil
  for i = 1, #group_pupils do
    if group_pupils[i].id == args.pupil_id then
      matched_pupil = group_pupils[i]
      break
    end
  end
  if not matched_pupil then
    error("pupil is not an active member of trip group " .. tostring(trip.group_id))
  end

  -- Query strictly guardians holding Parental Responsibility (PR) and not restricted by a court order
  local pr_guardians = ctx.dal.guardians.list_for_pupil(args.pupil_id, {
    academic_year_id = opts.academic_year_id,
    reference_date = opts.reference_date,
    require_parental_responsibility = true
  })

  local authorized_guardian = nil
  for i = 1, #pr_guardians do
    local g = pr_guardians[i]
    if g.guardian_person_id == args.guardian_person_id and g.has_parental_responsibility == true then
      authorized_guardian = g
      break
    end
  end

  if not authorized_guardian then
    error("GUARDIAN_NOT_AUTHORIZED_FOR_CONSENT: guardian does not hold active unrestricted Parental Responsibility (PR) for this pupil")
  end

  local consent_id = "tcon_" .. ctx.crypto.uuid()
  local guardian_name = (authorized_guardian.first_name or "") .. " " .. (authorized_guardian.last_name or "")
  local relationship_code = authorized_guardian.relationship_code or ""
  local consent_status = args.consent_status or "CONSENTED"
  if consent_status ~= "CONSENTED" and consent_status ~= "DECLINED" then
    error("invalid consent_status: must be CONSENTED or DECLINED")
  end
  local dietary_notes = args.dietary_notes or ""
  local medical_notes = args.medical_notes or ""
  local emergency_phone = authorized_guardian.phone_primary or ""
  local recorded_by_staff_id = (ctx.caller and ctx.caller.staff_id) or ""
  local recorded_at = ctx.clock.now_utc()

  ctx.plugin_db.exec(
    "DELETE FROM plugin_trips.trip_consents WHERE trip_id = ? AND pupil_id = ?",
    { trip.id, args.pupil_id }
  )
  ctx.plugin_db.exec(
    [[INSERT INTO plugin_trips.trip_consents (
        id, trip_id, pupil_id, guardian_person_id, guardian_name,
        relationship_code, has_parental_responsibility, consent_status,
        dietary_notes, medical_notes, emergency_phone,
        recorded_by_staff_id, recorded_at
      ) VALUES (?, ?, ?, ?, ?, ?, 1, ?, ?, ?, ?, ?, ?)]],
    {
      consent_id,
      trip.id,
      args.pupil_id,
      authorized_guardian.guardian_person_id,
      guardian_name,
      relationship_code,
      consent_status,
      dietary_notes,
      medical_notes,
      emergency_phone,
      recorded_by_staff_id,
      recorded_at
    }
  )

  return {
    ok = true,
    data = {
      consent_id = consent_id,
      trip_id = trip.id,
      pupil_id = args.pupil_id,
      guardian_person_id = authorized_guardian.guardian_person_id,
      guardian_name = guardian_name,
      relationship_code = relationship_code,
      has_parental_responsibility = true,
      consent_status = consent_status,
      dietary_notes = dietary_notes,
      medical_notes = medical_notes,
      emergency_phone = emergency_phone,
      recorded_at = recorded_at
    }
  }
end

function generate_trip_emergency_pack(ctx, args)
  if type(args) ~= "table" or type(args.trip_id) ~= "string" or args.trip_id == "" then
    error("missing mandatory parameter: trip_id")
  end

  local trip_rows = ctx.plugin_db.query(
    [[SELECT id, group_id, academic_year_id, trip_name, destination, trip_date, lead_staff_id, created_at
      FROM plugin_trips.trips
      WHERE id = ?]],
    { args.trip_id }
  )
  if #trip_rows == 0 then
    error("trip not found: " .. tostring(args.trip_id))
  end
  local trip = trip_rows[1]

  local opts = resolve_temporal_opts(ctx, args, trip.academic_year_id)

  -- 1. Fetch active on-roll group roster (enforcing :own_groups for the Trip Leader)
  local group_pupils = ctx.dal.pupils.list_by_group(trip.group_id, opts)
  local pupil_by_id = {}
  for i = 1, #group_pupils do
    local p = group_pupils[i]
    pupil_by_id[p.id] = p
  end

  -- 2. Query consented pupils for this trip
  local consent_rows = ctx.plugin_db.query(
    [[SELECT id, trip_id, pupil_id, guardian_person_id, guardian_name,
             relationship_code, has_parental_responsibility, consent_status,
             dietary_notes, medical_notes, emergency_phone,
             recorded_by_staff_id, recorded_at
      FROM plugin_trips.trip_consents
      WHERE trip_id = ?
        AND consent_status = 'CONSENTED'
        AND has_parental_responsibility = 1
      ORDER BY recorded_at ASC, id ASC]],
    { trip.id }
  )

  local consented_pupil_ids = {}
  local active_consents = {}
  for i = 1, #consent_rows do
    local c = consent_rows[i]
    if pupil_by_id[c.pupil_id] ~= nil then
      table.insert(consented_pupil_ids, c.pupil_id)
      table.insert(active_consents, c)
    end
  end

  -- 3. Batch fetch SEN & Medical / IHCP flags for all consented pupils via ctx.dal.pupils.list_sen_and_medical_batch
  local sen_medical_map = {}
  if #consented_pupil_ids > 0 then
    sen_medical_map = ctx.dal.pupils.list_sen_and_medical_batch(consented_pupil_ids, opts)
  end

  local emergency_manifest = {}
  for i = 1, #active_consents do
    local c = active_consents[i]
    local p = pupil_by_id[c.pupil_id]
    local sm = sen_medical_map[c.pupil_id] or {}
    local first_name = p.preferred_first_name or p.first_name or ""
    local last_name = p.preferred_last_name or p.last_name or ""

    table.insert(emergency_manifest, {
      pupil_id = p.id,
      pupil_name = first_name .. " " .. last_name,
      nc_year_group = p.nc_year_group or "",
      consent_id = c.id,
      consent_status = c.consent_status,
      guardian_person_id = c.guardian_person_id,
      guardian_name = c.guardian_name,
      relationship_code = c.relationship_code,
      has_parental_responsibility = true,
      emergency_phone = c.emergency_phone or "",
      dietary_notes = c.dietary_notes or "",
      medical_notes = c.medical_notes or "",
      sen_provision_code = sm.sen_provision_code or "N",
      primary_sen_need_code = sm.primary_sen_need_code or "",
      teaching_strategies = sm.teaching_strategies or "",
      medical_conditions = sm.medical_conditions or {}
    })
  end

  return {
    ok = true,
    data = {
      trip = trip,
      consented_count = #emergency_manifest,
      total_group_pupils = #group_pupils,
      emergency_manifest = emergency_manifest,
      generated_at = ctx.clock.now_utc()
    }
  }
end

function get_trip_details(ctx, args)
  if type(args) ~= "table" then
    error("missing mandatory arguments table")
  end

  local trip_rows = {}
  if type(args.trip_id) == "string" and args.trip_id ~= "" then
    trip_rows = ctx.plugin_db.query(
      [[SELECT id, group_id, academic_year_id, trip_name, destination, trip_date, lead_staff_id, created_at
        FROM plugin_trips.trips
        WHERE id = ?]],
      { args.trip_id }
    )
  elseif type(args.group_id) == "string" and args.group_id ~= "" then
    local opts = resolve_temporal_opts(ctx, args, nil)
    ctx.dal.pupils.list_by_group(args.group_id, opts)
    trip_rows = ctx.plugin_db.query(
      [[SELECT id, group_id, academic_year_id, trip_name, destination, trip_date, lead_staff_id, created_at
        FROM plugin_trips.trips
        WHERE group_id = ?
        ORDER BY created_at DESC, id DESC
        LIMIT 1]],
      { args.group_id }
    )
  end

  if #trip_rows == 0 then
    return {
      ok = true,
      data = {
        trip = nil,
        consents = {}
      }
    }
  end

  local trip = trip_rows[1]
  local consents = ctx.plugin_db.query(
    [[SELECT id, trip_id, pupil_id, guardian_person_id, guardian_name,
             relationship_code, has_parental_responsibility, consent_status,
             dietary_notes, medical_notes, emergency_phone,
             recorded_by_staff_id, recorded_at
      FROM plugin_trips.trip_consents
      WHERE trip_id = ?
      ORDER BY recorded_at ASC, id ASC]],
    { trip.id }
  )

  return {
    ok = true,
    data = {
      trip = trip,
      consents = consents
    }
  }
end

function export_pupil_data(ctx, args)
  local pupil_id = type(args) == "table" and args.pupil_id or args
  if type(pupil_id) ~= "string" or pupil_id == "" then
    error("missing mandatory parameter: pupil_id")
  end

  local rows = ctx.plugin_db.query(
    [[SELECT tc.id, tc.trip_id, t.trip_name, t.destination, t.trip_date,
             tc.pupil_id, tc.guardian_person_id, tc.guardian_name,
             tc.relationship_code, tc.has_parental_responsibility,
             tc.consent_status, tc.dietary_notes, tc.medical_notes,
             tc.emergency_phone, tc.recorded_by_staff_id, tc.recorded_at
      FROM plugin_trips.trip_consents tc
      LEFT JOIN plugin_trips.trips t ON t.id = tc.trip_id
      WHERE tc.pupil_id = ?
      ORDER BY tc.recorded_at ASC, tc.id ASC]],
    { pupil_id }
  )

  return {
    ok = true,
    data = {
      {
        title = "Educational Visits & Parental Consent Records",
        records = rows,
        redactable_fields = { "dietary_notes", "medical_notes" }
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
    "DELETE FROM plugin_trips.trip_consents WHERE pupil_id = ?",
    { pupil_id }
  )

  return {
    ok = true,
    data = {
      pupil_id = pupil_id,
      purged_rows = res.rows_affected or 0
    }
  }
end
