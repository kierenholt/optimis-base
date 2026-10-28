function generate_and_validate_census_return(ctx, args)
  local academic_year_id = args.academic_year_id or "ay_2026_2027"
  local census_date = args.census_date or "2026-10-01"
  local term_code = args.term_code or "AUTUMN"
  local attendance_start_date = args.attendance_start_date or "2026-09-01"
  local attendance_end_date = args.attendance_end_date or census_date
  local tamper_invalid_upn = args.tamper_invalid_upn

  local rollup = ctx.dal.census.get_statutory_rollup({
    academic_year_id = academic_year_id,
    census_date = census_date,
    term_code = term_code,
    attendance_start_date = attendance_start_date,
    attendance_end_date = attendance_end_date
  })

  local pupil_xml_nodes = {}
  for idx, p in ipairs(rollup.pupils) do
    local upn_val = p.upn
    if tamper_invalid_upn ~= nil and tamper_invalid_upn ~= "" and idx == 1 then
      upn_val = tostring(tamper_invalid_upn)
    end

    local id_children = {
      { tag = "UPN", text = upn_val }
    }
    if p.uln ~= nil and p.uln ~= "" then
      table.insert(id_children, { tag = "ULN", text = p.uln })
    end
    table.insert(id_children, { tag = "Surname", text = p.last_name })
    table.insert(id_children, { tag = "Forename", text = p.first_name })
    table.insert(id_children, { tag = "DOB", text = p.dob })
    table.insert(id_children, { tag = "Sex", text = p.sex })

    local char_children = {
      { tag = "Ethnicity", text = p.ethnicity_code },
      { tag = "Language", text = p.first_language_code },
      { tag = "SENProvision", text = p.sen_provision_code }
    }
    if p.primary_sen_need_code ~= nil and p.primary_sen_need_code ~= "" then
      table.insert(char_children, { tag = "SENNeed", text = p.primary_sen_need_code })
    end
    table.insert(char_children, {
      tag = "LookedAfter",
      text = p.looked_after_flag and "1" or "0"
    })

    if p.fsm_periods ~= nil and #p.fsm_periods > 0 then
      local fsm_period_nodes = {}
      for _, fp in ipairs(p.fsm_periods) do
        local fp_children = {
          { tag = "FSMStartDate", text = fp.effective_from }
        }
        if fp.effective_to ~= nil and fp.effective_to ~= "" then
          table.insert(fp_children, { tag = "FSMEndDate", text = fp.effective_to })
        end
        table.insert(fp_children, { tag = "Country", text = fp.country_code or "ENG" })
        table.insert(fsm_period_nodes, {
          tag = "FSMPeriod",
          children = fp_children
        })
      end
      table.insert(char_children, {
        tag = "FSMPeriods",
        children = fsm_period_nodes
      })
    end

    local att = p.attendance_sessions
    table.insert(pupil_xml_nodes, {
      tag = "Pupil",
      children = {
        {
          tag = "PupilIdentifiers",
          children = id_children
        },
        {
          tag = "PupilCharacteristics",
          children = char_children
        },
        {
          tag = "Enrolment",
          children = {
            { tag = "NCYearActual", text = p.nc_year_group },
            { tag = "EnrolmentStatus", text = p.enrolment_status },
            { tag = "EntryDate", text = p.effective_from }
          }
        },
        {
          tag = "Attendance",
          children = {
            { tag = "PossibleSessions", text = tostring(att.possible_sessions) },
            { tag = "SessionsPresent", text = tostring(att.present_sessions) },
            { tag = "AuthorisedAbsenceSessions", text = tostring(att.authorised_absence_sessions) },
            { tag = "UnauthorisedAbsenceSessions", text = tostring(att.unauthorised_absence_sessions) }
          }
        }
      }
    })
  end

  local xml_str = ctx.xml.build({
    tag = "Message",
    attrs = { schema = "CBDS", version = "1.0" },
    children = {
      {
        tag = "Header",
        children = {
          { tag = "Collection", text = "SchoolCensus" },
          { tag = "Term", text = rollup.term_code },
          { tag = "CensusDate", text = rollup.census_date },
          {
            tag = "Source",
            children = {
              { tag = "LEA", text = "201" },
              { tag = "Estab", text = "4001" },
              { tag = "URN", text = rollup.school_urn },
              { tag = "SchoolName", text = rollup.school_name }
            }
          }
        }
      },
      {
        tag = "Pupils",
        children = pupil_xml_nodes
      }
    }
  })

  local xsd_res = ctx.xml.validate_xsd(xml_str, "dfe_school_census_v1.xsd")
  if not xsd_res.valid then
    local err_msg = "XSD_VALIDATION_FAILED: "
    if xsd_res.errors and #xsd_res.errors > 0 then
      err_msg = err_msg .. table.concat(xsd_res.errors, "; ")
    else
      err_msg = err_msg .. "XML failed DfE School Census XSD validation"
    end
    error(err_msg)
  end

  local return_id = "cen_ret_" .. rollup.academic_year_id .. "_" .. rollup.term_code .. "_" .. rollup.census_date
  local sen_counts = rollup.summary.sen_counts or {}
  local sen_k = sen_counts["K"] or 0
  local sen_e = sen_counts["E"] or 0
  local sen_n = sen_counts["N"] or 0

  ctx.plugin_db.exec("DELETE FROM plugin_dfe_census.census_pupil_snapshots WHERE return_id = ?", { return_id })
  ctx.plugin_db.exec("DELETE FROM plugin_dfe_census.census_returns WHERE id = ?", { return_id })

  ctx.plugin_db.exec(
    [[INSERT INTO plugin_dfe_census.census_returns (
        id, academic_year_id, term_code, census_date, school_urn, school_name,
        on_roll_total, sen_k_count, sen_e_count, sen_n_count, fsm_eligible_count, lac_count,
        total_possible_sessions, total_present_sessions, total_authorised_absence_sessions, total_unauthorised_absence_sessions,
        xsd_valid, status, xml_payload, generated_by_staff_id
      ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, 'VALIDATED', ?, ?)]],
    {
      return_id,
      rollup.academic_year_id,
      rollup.term_code,
      rollup.census_date,
      rollup.school_urn,
      rollup.school_name,
      rollup.summary.on_roll_total,
      sen_k,
      sen_e,
      sen_n,
      rollup.summary.fsm_eligible_count,
      rollup.summary.lac_count,
      rollup.summary.total_possible_sessions,
      rollup.summary.total_present_sessions,
      rollup.summary.total_authorised_absence_sessions,
      rollup.summary.total_unauthorised_absence_sessions,
      xml_str,
      ctx.caller.staff_id or ""
    }
  )

  for _, p in ipairs(rollup.pupils) do
    local snap_id = "csnap_" .. return_id .. "_" .. p.pupil_id
    local att = p.attendance_sessions
    local note_text = "Statutory School Census snapshot (" .. rollup.term_code .. " " .. rollup.census_date .. ") SEN=" .. p.sen_provision_code
    ctx.plugin_db.exec(
      [[INSERT INTO plugin_dfe_census.census_pupil_snapshots (
          id, return_id, pupil_id, academic_year_id, term_code, census_date,
          upn, nc_year_group, enrolment_status, sen_provision_code, primary_sen_need_code,
          fsm_eligible, lac_episode_status, possible_sessions, present_sessions,
          authorised_absence_sessions, unauthorised_absence_sessions, notes
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)]],
      {
        snap_id,
        return_id,
        p.pupil_id,
        rollup.academic_year_id,
        rollup.term_code,
        rollup.census_date,
        p.upn,
        p.nc_year_group,
        p.enrolment_status,
        p.sen_provision_code,
        p.primary_sen_need_code or "",
        p.fsm_eligible and 1 or 0,
        p.lac_episode_status or "NONE",
        att.possible_sessions,
        att.present_sessions,
        att.authorised_absence_sessions,
        att.unauthorised_absence_sessions,
        note_text
      }
    )
  end

  return {
    ok = true,
    data = {
      return_id = return_id,
      school_urn = rollup.school_urn,
      school_name = rollup.school_name,
      academic_year_id = rollup.academic_year_id,
      term_code = rollup.term_code,
      census_date = rollup.census_date,
      xsd_valid = true,
      xsd_status = "XSD_VALID",
      summary = rollup.summary,
      xml_payload = xml_str
    }
  }
end

function submit_census_return(ctx, args)
  local return_id = args.return_id or ""
  local rows
  if return_id ~= "" then
    rows = ctx.plugin_db.query(
      "SELECT id, school_urn, term_code, census_date, xsd_valid, xml_payload FROM plugin_dfe_census.census_returns WHERE id = ?",
      { return_id }
    )
  else
    rows = ctx.plugin_db.query(
      "SELECT id, school_urn, term_code, census_date, xsd_valid, xml_payload FROM plugin_dfe_census.census_returns ORDER BY created_at DESC, id DESC LIMIT 1",
      {}
    )
  end

  if #rows == 0 then
    error("CENSUS_RETURN_NOT_FOUND: no validated census return found for submission")
  end

  local ret = rows[1]
  if ret.xsd_valid ~= 1 then
    error("XSD_VALIDATION_REQUIRED: census return must pass XSD validation before COLLECT submission")
  end

  -- Structural Safety Invariant (§9.2 & §10.1): Outbound HTTP executes BEFORE opening any WriteDB transaction
  local resp = ctx.http.post(
    "https://collect.dfe.gov.uk/v1/school-census/submit",
    {
      ["Content-Type"] = "application/xml",
      ["X-Optimis-School-URN"] = tostring(ret.school_urn)
    },
    ret.xml_payload
  )

  if resp.status < 200 or resp.status >= 300 then
    error("COLLECT_SUBMISSION_REJECTED: status " .. tostring(resp.status) .. " body=" .. tostring(resp.body))
  end

  local submission_id = string.match(resp.body or "", '"submission_id"%s*:%s*"([^"]+)"') or "collect_sub_accepted"

  ctx.plugin_db.exec(
    "UPDATE plugin_dfe_census.census_returns SET status = 'COLLECT_ACCEPTED', collect_submission_id = ? WHERE id = ?",
    { submission_id, ret.id }
  )

  return {
    ok = true,
    data = {
      return_id = ret.id,
      submission_id = submission_id,
      status = "COLLECT_ACCEPTED",
      term_code = ret.term_code,
      census_date = ret.census_date
    }
  }
end

function list_census_returns(ctx, args)
  local ay_id = args.academic_year_id or ""
  local rows
  if ay_id ~= "" then
    rows = ctx.plugin_db.query(
      "SELECT id, academic_year_id, term_code, census_date, school_urn, on_roll_total, sen_k_count, sen_e_count, sen_n_count, fsm_eligible_count, lac_count, xsd_valid, status, collect_submission_id, created_at FROM plugin_dfe_census.census_returns WHERE academic_year_id = ? ORDER BY created_at DESC, id DESC",
      { ay_id }
    )
  else
    rows = ctx.plugin_db.query(
      "SELECT id, academic_year_id, term_code, census_date, school_urn, on_roll_total, sen_k_count, sen_e_count, sen_n_count, fsm_eligible_count, lac_count, xsd_valid, status, collect_submission_id, created_at FROM plugin_dfe_census.census_returns ORDER BY created_at DESC, id DESC",
      {}
    )
  end

  return {
    ok = true,
    data = {
      returns = rows
    }
  }
end

function export_pupil_data(ctx, args)
  local pupil_id = args.pupil_id
  if not pupil_id or pupil_id == "" then
    error("missing mandatory parameter: pupil_id")
  end

  local rows = ctx.plugin_db.query(
    [[SELECT id, return_id, pupil_id, academic_year_id, term_code, census_date,
             upn, nc_year_group, enrolment_status, sen_provision_code, primary_sen_need_code,
             fsm_eligible, lac_episode_status, possible_sessions, present_sessions,
             authorised_absence_sessions, unauthorised_absence_sessions, notes, created_at
      FROM plugin_dfe_census.census_pupil_snapshots
      WHERE pupil_id = ?
      ORDER BY created_at ASC, id ASC]],
    { pupil_id }
  )

  return {
    ok = true,
    data = {
      sections = {
        {
          title = "DfE Statutory School Census Snapshots",
          records = rows,
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

  local res = ctx.plugin_db.exec(
    "DELETE FROM plugin_dfe_census.census_pupil_snapshots WHERE pupil_id = ?",
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
