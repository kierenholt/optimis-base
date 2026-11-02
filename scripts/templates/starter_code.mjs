/**
 * Starter Plugin Code Templates (SQL, Lua, HTML, Tests) for Optimis Plugin Starter
 */

export const CODE_TEMPLATES = {
  initSql: () => `-- Expand-and-Contract Migration 001: Classroom Praise & Notes
-- Note: Minor migrations must never use DROP COLUMN, RENAME COLUMN, or DROP TABLE.

CREATE TABLE IF NOT EXISTS pupil_praise_notes (
    id TEXT PRIMARY KEY,
    pupil_id TEXT NOT NULL,
    group_id TEXT NOT NULL,
    staff_id TEXT NOT NULL,
    category TEXT NOT NULL,
    points INTEGER NOT NULL DEFAULT 1,
    praise_note TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_pupil_praise_notes_pupil
    ON pupil_praise_notes(pupil_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_pupil_praise_notes_group
    ON pupil_praise_notes(group_id, created_at DESC);
`,

  initLua: () => `---@param ctx OptimisPluginContext
---@param args table
function load_group_praise_roster(ctx, args)
  local group_id = args and args.group_id
  if not group_id or group_id == "" then
    local groups = ctx.dal.groups.list("REGISTRATION")
    if #groups == 0 then
      groups = ctx.dal.groups.list("")
    end
    if #groups == 0 then
      error("NO_ASSIGNED_GROUPS: caller has no active groups")
    end
    group_id = groups[1].id
  end

  local pupils = ctx.dal.pupils.list_by_group(group_id)
  local notes = ctx.plugin_db.query(
    [[SELECT id, pupil_id, group_id, staff_id, category, points, praise_note, created_at
      FROM plugin_starter_notes.pupil_praise_notes
      WHERE group_id = ?
      ORDER BY created_at DESC, id DESC]],
    { group_id }
  )

  return {
    ok = true,
    data = {
      group_id = group_id,
      caller_staff_id = ctx.caller.staff_id,
      pupils = pupils,
      notes = notes,
    },
  }
end

---@param ctx OptimisPluginContext
---@param args table
function record_pupil_praise_note(ctx, args)
  if not args or not args.pupil_id or args.pupil_id == "" then
    error("MISSING_PUPIL_ID: pupil_id is required")
  end
  if not args.group_id or args.group_id == "" then
    error("MISSING_GROUP_ID: group_id is required")
  end
  if not args.praise_note or args.praise_note == "" then
    error("MISSING_PRAISE_NOTE: praise_note is required")
  end

  -- Verify pupil belongs to caller's scoped group (:own_groups enforced automatically by Core DAL)
  local pupil = ctx.dal.pupils.get_by_id(args.pupil_id)
  if not pupil or not pupil.id then
    error("PUPIL_NOT_FOUND_OR_OUT_OF_SCOPE: " .. tostring(args.pupil_id))
  end

  local category = args.category or "Academic Effort"
  local points = tonumber(args.points) or 1
  if points < 1 then
    error("INVALID_POINTS: points must be >= 1")
  end

  local existing = ctx.plugin_db.query(
    "SELECT COUNT(*) AS cnt FROM plugin_starter_notes.pupil_praise_notes",
    {}
  )
  local next_seq = 1
  if existing and existing[1] and existing[1].cnt then
    next_seq = tonumber(existing[1].cnt) + 1
  end
  local note_id = string.format("prs_%s_%d", tostring(args.pupil_id), next_seq)

  ctx.plugin_db.exec(
    [[INSERT INTO plugin_starter_notes.pupil_praise_notes
      (id, pupil_id, group_id, staff_id, category, points, praise_note)
      VALUES (?, ?, ?, ?, ?, ?, ?)]],
    {
      note_id,
      tostring(args.pupil_id),
      tostring(args.group_id),
      tostring(ctx.caller.staff_id or "SYSTEM"),
      tostring(category),
      points,
      tostring(args.praise_note),
    }
  )

  return {
    ok = true,
    data = {
      id = note_id,
      pupil_id = tostring(args.pupil_id),
      group_id = tostring(args.group_id),
      category = tostring(category),
      points = points,
      praise_note = tostring(args.praise_note),
    },
  }
end

---@param ctx OptimisPluginContext
---@param args table
function get_recent_praise_summary(ctx, args)
  local rows = ctx.plugin_db.query(
    [[SELECT id, pupil_id, group_id, category, points, praise_note, created_at
      FROM plugin_starter_notes.pupil_praise_notes
      ORDER BY created_at DESC, id DESC
      LIMIT 5]],
    {}
  )

  local total_points = 0
  for _, row in ipairs(rows) do
    total_points = total_points + (tonumber(row.points) or 0)
  end

  return {
    ok = true,
    data = {
      recent_count = #rows,
      total_points = total_points,
      items = rows,
    },
  }
end

---@param ctx OptimisPluginContext
---@param args table|string
function export_pupil_data(ctx, args)
  local pupil_id = type(args) == "table" and args.pupil_id or args
  if not pupil_id or pupil_id == "" then
    error("MISSING_PUPIL_ID: pupil_id is required for SAR export")
  end

  local records = ctx.plugin_db.query(
    [[SELECT id, pupil_id, group_id, staff_id, category, points, praise_note, created_at
      FROM plugin_starter_notes.pupil_praise_notes
      WHERE pupil_id = ?
      ORDER BY created_at DESC, id DESC]],
    { tostring(pupil_id) }
  )

  return {
    ok = true,
    data = {
      {
        title = "Classroom Praise & Recognition Notes",
        records = records,
        redactable_fields = { "praise_note" },
      },
    },
  }
end

---@param ctx OptimisPluginContext
---@param args table|string
function purge_pupil_data(ctx, args)
  local pupil_id = type(args) == "table" and args.pupil_id or args
  if not pupil_id or pupil_id == "" then
    error("MISSING_PUPIL_ID: pupil_id is required for retention purge")
  end

  local res = ctx.plugin_db.exec(
    "DELETE FROM plugin_starter_notes.pupil_praise_notes WHERE pupil_id = ?",
    { tostring(pupil_id) }
  )

  return {
    ok = true,
    data = {
      pupil_id = tostring(pupil_id),
      rows_affected = res and res.rows_affected or 0,
    },
  }
end
`,

  uiIndexHtml: () => `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>Classroom Praise & Notes Starter</title>
  <link rel="stylesheet" href="/sdk/mis-theme.css" />
  <script src="/sdk/mis-bridge.js"></script>
</head>
<body class="mis-surface" style="padding: 1.25rem; margin: 0;">
  <main style="max-width: 960px; margin: 0 auto;">
    <header style="display: flex; align-items: center; justify-content: space-between; margin-bottom: 1rem;">
      <div style="display: flex; align-items: center; gap: 0.625rem;">
        <span data-mis-icon="award"></span>
        <div>
          <h1 style="margin: 0; font-size: 1.25rem;">Classroom Praise &amp; Notes</h1>
          <p class="mis-text-muted" style="margin: 0.15rem 0 0 0; font-size: 0.875rem;">
            Starter Optimis Plugin — Scoped to your assigned teaching &amp; registration groups
          </p>
        </div>
      </div>
      <button
        type="button"
        id="refresh-roster-btn"
        data-testid="refresh-roster-btn"
        data-mis-rpc="load_group_praise_roster"
        data-mis-args='{"group_id":"grp_reg_7a"}'
        class="mis-btn-primary"
      >
        <span data-mis-icon="refresh-cw"></span>
        <span>Load Group 7A</span>
      </button>
    </header>

    <section class="mis-card" style="padding: 1rem; margin-bottom: 1rem;">
      <h2 style="margin-top: 0; font-size: 1rem;">Award Classroom Praise</h2>
      <div style="display: grid; grid-template-columns: 1fr 1fr 2fr auto; gap: 0.75rem; align-items: end;">
        <label style="display: flex; flex-direction: column; gap: 0.25rem; font-size: 0.875rem;">
          <span>Pupil ID</span>
          <input id="praise-pupil-id" data-testid="praise-pupil-id" value="pup_yr7_alice" style="padding: 0.45rem;" />
        </label>
        <label style="display: flex; flex-direction: column; gap: 0.25rem; font-size: 0.875rem;">
          <span>Category</span>
          <input id="praise-category" data-testid="praise-category" value="Academic Effort" style="padding: 0.45rem;" />
        </label>
        <label style="display: flex; flex-direction: column; gap: 0.25rem; font-size: 0.875rem;">
          <span>Praise Note (SAR Redactable)</span>
          <input id="praise-note" data-testid="praise-note" value="Helped peers solve linear equations" style="padding: 0.45rem;" />
        </label>
        <button
          type="button"
          id="submit-praise-btn"
          data-testid="submit-praise-btn"
          data-mis-rpc="record_pupil_praise_note"
          data-mis-args='{"pupil_id":"pup_yr7_alice","group_id":"grp_reg_7a","category":"Academic Effort","points":2,"praise_note":"Helped peers solve linear equations"}'
          class="mis-btn-primary"
        >
          <span data-mis-icon="clipboard-check"></span>
          <span>Save Praise</span>
        </button>
      </div>
    </section>

    <section class="mis-card" style="padding: 1rem;">
      <h2 style="margin-top: 0; font-size: 1rem;">Group Roster &amp; Recent Notes</h2>
      <pre id="plugin-output" data-testid="plugin-output" style="margin: 0; font-size: 0.8125rem; white-space: pre-wrap;">Ready.</pre>
    </section>
  </main>

  <script>
    (function () {
      var outputEl = document.getElementById('plugin-output');
      var submitBtn = document.getElementById('submit-praise-btn');

      function syncArgs() {
        var pupilId = document.getElementById('praise-pupil-id').value;
        var category = document.getElementById('praise-category').value;
        var note = document.getElementById('praise-note').value;
        submitBtn.setAttribute(
          'data-mis-args',
          JSON.stringify({
            pupil_id: pupilId,
            group_id: 'grp_reg_7a',
            category: category,
            points: 2,
            praise_note: note,
          })
        );
      }

      ['praise-pupil-id', 'praise-category', 'praise-note'].forEach(function (id) {
        document.getElementById(id).addEventListener('input', syncArgs);
      });

      document.addEventListener('mis:rpc:success', function (ev) {
        outputEl.textContent = JSON.stringify(ev.detail, null, 2);
      });

      document.addEventListener('mis:rpc:error', function (ev) {
        outputEl.textContent = 'RPC Error: ' + JSON.stringify(ev.detail, null, 2);
      });
    })();
  </script>
</body>
</html>
`,

  uiWidgetHtml: () => `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>Recent Classroom Praise Widget</title>
  <link rel="stylesheet" href="/sdk/mis-theme.css" />
  <script src="/sdk/mis-bridge.js"></script>
</head>
<body class="mis-surface" style="padding: 0.75rem; margin: 0;">
  <div style="display: flex; align-items: center; justify-content: space-between; margin-bottom: 0.5rem;">
    <div style="display: flex; align-items: center; gap: 0.4rem; font-weight: 600;">
      <span data-mis-icon="award"></span>
      <span>Recent Praise</span>
    </div>
    <button
      type="button"
      data-testid="widget-refresh-btn"
      data-mis-rpc="get_recent_praise_summary"
      data-mis-args="{}"
      class="mis-btn-primary"
      style="padding: 0.25rem 0.5rem; font-size: 0.75rem;"
    >
      <span data-mis-icon="refresh-cw"></span>
    </button>
  </div>
  <div id="widget-summary" data-testid="widget-summary" class="mis-text-muted" style="font-size: 0.8125rem;">
    Click refresh or wait for handshake...
  </div>
  <script>
    window.addEventListener('message', function (ev) {
      if (ev.data && ev.data.type === 'mis:init' && window.mis && typeof window.mis.rpc === 'function') {
        window.mis.rpc('get_recent_praise_summary', {}).then(function (res) {
          var el = document.getElementById('widget-summary');
          if (res && res.data) {
            el.textContent = 'Notes logged: ' + res.data.recent_count + ' (Total points: ' + res.data.total_points + ')';
          }
        });
      }
    });
  </script>
</body>
</html>
`,

  initTestLua: () => `function test_recordAndLoadPraise_withScopedTeacher_persistsNoteAndSatisfiesGdprHooks_test(t)
  -- 1. Load group 7A praise roster as Form Tutor 7A (stf_tch_7a)
  local roster_res = t.rpc("load_group_praise_roster", { group_id = "grp_reg_7a" }, "stf_tch_7a")
  assert(roster_res.ok == true, "expected load_group_praise_roster to succeed")
  assert(roster_res.data.group_id == "grp_reg_7a", "expected group_id grp_reg_7a")
  assert(#roster_res.data.pupils >= 1, "expected synthetic pupils in grp_reg_7a")

  -- 2. Record a praise note for Alice (pup_yr7_alice in grp_reg_7a)
  local create_res = t.rpc("record_pupil_praise_note", {
    pupil_id = "pup_yr7_alice",
    group_id = "grp_reg_7a",
    category = "Mathematics Mastery",
    points = 3,
    praise_note = "Explained simultaneous equations clearly to the class",
  }, "stf_tch_7a")
  assert(create_res.ok == true, "expected record_pupil_praise_note to succeed")
  assert(create_res.data.points == 3, "expected 3 points recorded")

  -- 3. Verify widget summary endpoint reflects the newly logged praise
  local summary_res = t.rpc("get_recent_praise_summary", {}, "stf_tch_7a")
  assert(summary_res.ok == true, "expected get_recent_praise_summary to succeed")
  assert(summary_res.data.recent_count == 1, "expected recent_count == 1")
  assert(summary_res.data.total_points == 3, "expected total_points == 3")

  -- 4. Verify UK GDPR SAR export_pupil_data hook returns redactable praise_note field
  local sar_res = t.rpc("export_pupil_data", { pupil_id = "pup_yr7_alice" }, "stf_admin_01")
  assert(sar_res.ok == true, "expected export_pupil_data to succeed")
  assert(#sar_res.data == 1, "expected 1 SAR section")
  assert(#sar_res.data[1].records == 1, "expected 1 exported praise record")

  -- 5. Verify UK GDPR purge_pupil_data hook removes all rows for pup_yr7_alice
  local purge_res = t.rpc("purge_pupil_data", { pupil_id = "pup_yr7_alice" }, "stf_admin_01")
  assert(purge_res.ok == true, "expected purge_pupil_data to succeed")

  local after_sar = t.rpc("export_pupil_data", { pupil_id = "pup_yr7_alice" }, "stf_admin_01")
  assert(#after_sar.data[1].records == 0, "expected 0 records after purge_pupil_data")
end
`,
};
