local function ensure_mat_schema_synced(ctx)
  local schema_ver = (ctx.school and ctx.school.mat_grading_schema_version) or ""
  local schema_tbl = ctx.school and ctx.school.mat_grading_schema
  if schema_ver == "" or type(schema_tbl) ~= "table" or type(schema_tbl.scales) ~= "table" then
    error("MAT_GRADING_SCHEMA_NOT_SYNCED: pull MAT policies into __system_config before using the markbook")
  end
  local mat_id = (ctx.school and ctx.school.mat_id) or ""
  return mat_id, schema_ver, schema_tbl
end

local function validate_and_score_mat_grade(schema_tbl, grading_scale, raw_grade)
  if type(raw_grade) ~= "string" or raw_grade == "" then
    error("INVALID_MAT_GRADE: missing mandatory grade value")
  end
  local scales = type(schema_tbl) == "table" and schema_tbl.scales or nil
  local scale_spec = type(scales) == "table" and scales[grading_scale] or nil
  if type(scale_spec) ~= "table" or type(scale_spec.points) ~= "table" then
    error("INVALID_MAT_SCALE: unsupported MAT grading scale '" .. tostring(grading_scale) .. "'")
  end
  local pts = scale_spec.points[raw_grade]
  if pts == nil then
    error("INVALID_MAT_GRADE: grade '" .. tostring(raw_grade) .. "' is not valid for MAT grading scale '" .. tostring(grading_scale) .. "'")
  end
  return tonumber(pts) or 0
end

local function build_group_markbook_snapshot(ctx, group_id, ay_id, ref_date, active_column_id)
  local pupils = ctx.dal.pupils.list_by_group(group_id, {
    academic_year_id = ay_id,
    reference_date = ref_date
  })

  local columns = ctx.plugin_db.query(
    [[SELECT id, group_id, academic_year_id, title, grading_scale,
             mat_schema_version, assessment_date, created_by_staff_id, created_at
      FROM plugin_markbook.markbook_columns
      WHERE group_id = ? AND academic_year_id = ?
      ORDER BY created_at DESC, id DESC]],
    { group_id, ay_id }
  )

  local resolved_col = nil
  if type(active_column_id) == "string" and active_column_id ~= "" then
    for i = 1, #columns do
      if columns[i].id == active_column_id then
        resolved_col = columns[i]
        break
      end
    end
  end
  if not resolved_col and #columns > 0 then
    resolved_col = columns[1]
  end

  local grades = {}
  if resolved_col then
    grades = ctx.plugin_db.query(
      [[SELECT id, column_id, group_id, pupil_id, academic_year_id,
               grade, grade_points, teacher_comment, recorded_by_staff_id, updated_at
        FROM plugin_markbook.markbook_grades
        WHERE group_id = ? AND column_id = ?
        ORDER BY pupil_id ASC]],
      { group_id, resolved_col.id }
    )
  end

  return {
    group_id = group_id,
    pupils = pupils,
    columns = columns,
    active_column = resolved_col,
    grades = grades
  }
end

function get_markbook_workspace(ctx, args)
  args = type(args) == "table" and args or {}
  local ay_id = args.academic_year_id or (ctx.school and ctx.school.academic_year_id) or "ay_2026_2027"
  local ref_date = args.reference_date or (ctx.school and ctx.school.reference_date) or "2026-10-28"

  local mat_id = (ctx.school and ctx.school.mat_id) or ""
  local schema_ver = (ctx.school and ctx.school.mat_grading_schema_version) or ""

  local groups = ctx.dal.groups.list_groups({
    academic_year_id = ay_id,
    reference_date = ref_date
  })

  local selected_group_id = args.group_id
  if (type(selected_group_id) ~= "string" or selected_group_id == "") and #groups > 0 then
    selected_group_id = groups[1].id
  end

  local snapshot = {
    group_id = selected_group_id or "",
    pupils = {},
    columns = {},
    active_column = nil,
    grades = {}
  }

  if type(selected_group_id) == "string" and selected_group_id ~= "" then
    snapshot = build_group_markbook_snapshot(ctx, selected_group_id, ay_id, ref_date, args.column_id)
  end

  return {
    ok = true,
    data = {
      mat_id = mat_id,
      grading_schema_version = schema_ver ~= "" and schema_ver or "UNSYNCED",
      mat_synced = schema_ver ~= "",
      groups = groups,
      selected_group_id = snapshot.group_id,
      pupils = snapshot.pupils,
      columns = snapshot.columns,
      column = snapshot.active_column,
      grades = snapshot.grades
    }
  }
end

function create_markbook_column(ctx, args)
  if type(args) ~= "table" then
    error("missing mandatory arguments table")
  end
  local mat_id, schema_ver, schema_tbl = ensure_mat_schema_synced(ctx)

  local group_id = args.group_id
  if type(group_id) ~= "string" or group_id == "" then
    error("missing mandatory parameter: group_id")
  end
  local title = args.title
  if type(title) ~= "string" or title == "" then
    error("missing mandatory parameter: title")
  end
  local grading_scale = args.grading_scale
  if type(grading_scale) ~= "string" or type(schema_tbl.scales[grading_scale]) ~= "table" then
    error("INVALID_MAT_SCALE: grading_scale must be '9-1' or 'WT,EXS,GDS'")
  end

  local ay_id = args.academic_year_id or (ctx.school and ctx.school.academic_year_id) or "ay_2026_2027"
  local ref_date = args.reference_date or (ctx.school and ctx.school.reference_date) or "2026-10-28"

  -- Enforce caller scope on group_id via first-class DAL write guard (:own_groups blocks other sets for Teachers; :all allows HoD)
  ctx.dal.groups.assert_caller_assigned(group_id, {
    academic_year_id = ay_id,
    reference_date = ref_date
  })

  local pupils = ctx.dal.pupils.list_by_group(group_id, {
    academic_year_id = ay_id,
    reference_date = ref_date
  })

  local col_id = "col_" .. ctx.crypto.uuid()
  local created_at = ctx.clock.now_utc()

  ctx.plugin_db.exec(
    [[INSERT INTO plugin_markbook.markbook_columns (
        id, group_id, academic_year_id, title, grading_scale,
        mat_schema_version, assessment_date, created_by_staff_id, created_at
      ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)]],
    {
      col_id,
      group_id,
      ay_id,
      title,
      grading_scale,
      schema_ver,
      ref_date,
      ctx.caller.staff_id or "",
      created_at
    }
  )

  local created_col = {
    id = col_id,
    group_id = group_id,
    academic_year_id = ay_id,
    title = title,
    grading_scale = grading_scale,
    mat_schema_version = schema_ver,
    assessment_date = ref_date,
    created_by_staff_id = ctx.caller.staff_id or "",
    created_at = created_at
  }

  return {
    ok = true,
    data = {
      mat_id = mat_id,
      grading_schema_version = schema_ver,
      selected_group_id = group_id,
      column = created_col,
      pupils = pupils,
      grades = {}
    }
  }
end

function save_pupil_grades(ctx, args)
  if type(args) ~= "table" then
    error("missing mandatory arguments table")
  end
  local mat_id, schema_ver, schema_tbl = ensure_mat_schema_synced(ctx)

  local column_id = args.column_id
  if type(column_id) ~= "string" or column_id == "" then
    error("missing mandatory parameter: column_id")
  end
  local group_id = args.group_id
  if type(group_id) ~= "string" or group_id == "" then
    error("missing mandatory parameter: group_id")
  end
  if type(args.entries) ~= "table" or #args.entries == 0 then
    error("missing mandatory parameter: entries (non-empty array)")
  end

  local ay_id = args.academic_year_id or (ctx.school and ctx.school.academic_year_id) or "ay_2026_2027"
  local ref_date = args.reference_date or (ctx.school and ctx.school.reference_date) or "2026-10-28"

  -- Enforce caller scope on group_id via first-class DAL write guard and verify every graded pupil is an active member of group_id on ref_date
  ctx.dal.groups.assert_caller_assigned(group_id, {
    academic_year_id = ay_id,
    reference_date = ref_date
  })

  local group_pupils = ctx.dal.pupils.list_by_group(group_id, {
    academic_year_id = ay_id,
    reference_date = ref_date
  })
  local active_pupil_map = {}
  for i = 1, #group_pupils do
    active_pupil_map[group_pupils[i].id] = group_pupils[i]
  end

  local col_rows = ctx.plugin_db.query(
    [[SELECT id, group_id, academic_year_id, title, grading_scale, mat_schema_version
      FROM plugin_markbook.markbook_columns
      WHERE id = ? AND group_id = ?]],
    { column_id, group_id }
  )
  if #col_rows == 0 then
    error("markbook column not found for group_id: " .. tostring(column_id))
  end
  local col = col_rows[1]

  local updated_at = ctx.clock.now_utc()
  local batch_rows = {}

  for i = 1, #args.entries do
    local entry = args.entries[i]
    local pid = type(entry) == "table" and entry.pupil_id or ""
    local grade_val = type(entry) == "table" and entry.grade or ""
    local comment_val = (type(entry) == "table" and type(entry.teacher_comment) == "string") and entry.teacher_comment or ""

    if type(pid) ~= "string" or pid == "" then
      error("missing mandatory pupil_id in entries[" .. tostring(i) .. "]")
    end
    if not active_pupil_map[pid] then
      error("access denied under own_groups scope: pupil " .. tostring(pid) .. " is not an active member of group " .. tostring(group_id))
    end

    local pts = validate_and_score_mat_grade(schema_tbl, col.grading_scale, grade_val)
    local grade_id = "mkg_" .. ctx.crypto.uuid()

    batch_rows[#batch_rows + 1] = {
      grade_id,
      column_id,
      group_id,
      pid,
      ay_id,
      grade_val,
      pts,
      comment_val,
      ctx.caller.staff_id or "",
      updated_at
    }
  end

  ctx.plugin_db.exec_batch(
    [[INSERT INTO plugin_markbook.markbook_grades (
        id, column_id, group_id, pupil_id, academic_year_id,
        grade, grade_points, teacher_comment, recorded_by_staff_id, updated_at
      ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
      ON CONFLICT(column_id, pupil_id) DO UPDATE SET
        grade = excluded.grade,
        grade_points = excluded.grade_points,
        teacher_comment = excluded.teacher_comment,
        recorded_by_staff_id = excluded.recorded_by_staff_id,
        updated_at = excluded.updated_at]],
    batch_rows
  )

  local saved_count = #batch_rows

  local saved_grades = ctx.plugin_db.query(
    [[SELECT id, column_id, group_id, pupil_id, academic_year_id,
             grade, grade_points, teacher_comment, recorded_by_staff_id, updated_at
      FROM plugin_markbook.markbook_grades
      WHERE group_id = ? AND column_id = ?
      ORDER BY pupil_id ASC]],
    { group_id, column_id }
  )

  return {
    ok = true,
    data = {
      mat_id = mat_id,
      grading_schema_version = schema_ver,
      selected_group_id = group_id,
      column = col,
      saved_count = saved_count,
      pupils = group_pupils,
      grades = saved_grades
    }
  }
end

function get_department_attainment_rollup(ctx, args)
  args = type(args) == "table" and args or {}
  local mat_id, schema_ver = ensure_mat_schema_synced(ctx)

  local ay_id = args.academic_year_id or (ctx.school and ctx.school.academic_year_id) or "ay_2026_2027"
  local ref_date = args.reference_date or (ctx.school and ctx.school.reference_date) or "2026-10-28"

  local all_groups = ctx.dal.groups.list_all_groups({
    academic_year_id = ay_id,
    reference_date = ref_date,
    group_type = "TEACHING_CLASS"
  })

  local filter_ids = nil
  if type(args.group_ids) == "table" and #args.group_ids > 0 then
    filter_ids = {}
    for i = 1, #args.group_ids do
      filter_ids[args.group_ids[i]] = true
    end
  end

  local sets = {}
  local highest_avg = -1
  local highest_group_id = ""
  local highest_group_name = ""

  for i = 1, #all_groups do
    local grp = all_groups[i]
    if not filter_ids or filter_ids[grp.id] then
      local grade_rows = ctx.plugin_db.query(
        [[SELECT grade, grade_points
          FROM plugin_markbook.markbook_grades
          WHERE group_id = ? AND academic_year_id = ?]],
        { grp.id, ay_id }
      )

      local graded_count = #grade_rows
      local total_pts = 0
      for j = 1, graded_count do
        total_pts = total_pts + (tonumber(grade_rows[j].grade_points) or 0)
      end

      local avg_num = 0
      local avg_str = "0.0"
      if graded_count > 0 then
        avg_num = total_pts / graded_count
        avg_str = string.format("%.1f", avg_num)
        if avg_num > highest_avg then
          highest_avg = avg_num
          highest_group_id = grp.id
          highest_group_name = grp.name
        end
      end

      if not filter_ids and graded_count > 0 or filter_ids then
        sets[#sets + 1] = {
          group_id = grp.id,
          group_code = grp.code,
          group_name = grp.name,
          graded_count = graded_count,
          average_points = avg_str,
          average_points_num = avg_num
        }
      end
    end
  end

  return {
    ok = true,
    data = {
      mat_id = mat_id,
      grading_schema_version = schema_ver,
      sets = sets,
      highest_attaining_group_id = highest_group_id,
      highest_attaining_group_name = highest_group_name
    }
  }
end

function export_pupil_data(ctx, args)
  local pupil_id = type(args) == "table" and args.pupil_id or args
  if type(pupil_id) ~= "string" or pupil_id == "" then
    error("missing mandatory parameter: pupil_id")
  end

  local rows = ctx.plugin_db.query(
    [[SELECT g.id, g.column_id, g.group_id, g.pupil_id, g.academic_year_id,
             c.title AS assessment_title, c.grading_scale, c.mat_schema_version,
             g.grade, g.grade_points, g.teacher_comment, g.recorded_by_staff_id, g.updated_at
      FROM plugin_markbook.markbook_grades g
      JOIN plugin_markbook.markbook_columns c ON c.id = g.column_id
      WHERE g.pupil_id = ?
      ORDER BY g.updated_at ASC, g.id ASC]],
    { pupil_id }
  )

  return {
    ok = true,
    data = {
      {
        title = "Assessment Markbook Grades & Teacher Comments",
        records = rows,
        redactable_fields = { "teacher_comment" }
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
    "DELETE FROM plugin_markbook.markbook_grades WHERE pupil_id = ?",
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
