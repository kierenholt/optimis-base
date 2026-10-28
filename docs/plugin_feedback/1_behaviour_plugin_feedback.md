# Plugin Contract Evaluation — Vertical 3: Behaviour & Rewards (`uk.org.optimis.behaviour`)

**Plugin Under Review:** [`plugins/behaviour_plugin`](file:///home/kieren/Documents/code/optimis/plugins/behaviour_plugin/GEMINI.md) (`uk.org.optimis.behaviour`)  
**E2E Verification Suite:** [`tests/e2e/behaviour_plugin_vertical.spec.ts`](file:///home/kieren/Documents/code/optimis/tests/e2e/behaviour_plugin_vertical.spec.ts)  
**Contract Surfaces Exercised:**
* Attached Plugin SQLite DB (`database.alias: "plugin_behaviour"`)
* Interactive Endpoints (`log_behaviour_event` with `mutates: true`, `get_pupil_behaviour_summary` with `mutates: false`)
* Method-Level Scoped RBAC (`dal.pupils.read:own_groups`)
* Home Dashboard Mini-Iframe Widget (`widgets: [{ "id": "behaviour_summary", "entry": "ui/dist/widget.html" }]`)
* Entity Deep-Linking (`<a data-mis-link="student" data-pupil-id="...">`) into the Keep-Alive LRU Pool
* UK GDPR SAR Export & Retention Purge Hooks (`export_pupil_data` with `redactable_fields: ["incident_notes"]` & `purge_pupil_data`)

---

## 1. Contract Surfaces That Work Well (Keep As-Is)

1. **`mutates: true/false` Read/Write SQLite Pool Routing (`ctx.plugin_db`)**
   * Declaring `mutates: true` on `log_behaviour_event` and `mutates: false` on `get_pupil_behaviour_summary` in [`manifest.json`](file:///home/kieren/Documents/code/optimis/plugins/behaviour_plugin/manifest.json) cleanly separated concurrent WAL reads (`ReadDB`) from serialized writes (`WriteDB`) in [`internal/host/server.go`](file:///home/kieren/Documents/code/optimis/internal/host/server.go) without requiring manual transaction boilerplate in Lua.
2. **UK GDPR SAR Hook Shape (`export_pupil_data` & `redactable_fields`)**
   * Returning `{ title = "Behaviour Merits & Incidents", redactable_fields = { "incident_notes" }, records = ... }` from [`src/init.lua`](file:///home/kieren/Documents/code/optimis/plugins/behaviour_plugin/src/init.lua#L145-L185) plugged directly into [`internal/host/gdpr.go`](file:///home/kieren/Documents/code/optimis/internal/host/gdpr.go) and [`ui/src/GdprSarPanel.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/GdprSarPanel.tsx), enabling DPO/Admin third-party name redaction with zero custom UI per plugin.
3. **Declarative `data-mis-rpc` & `data-mis-link` inside `<iframe sandbox="allow-scripts">`**
   * Zero-network (`connect-src 'none'`) sandboxed iframes worked seamlessly across both the full-page plugin view ([`ui/dist/index.html`](file:///home/kieren/Documents/code/optimis/plugins/behaviour_plugin/ui/dist/index.html)) and the Home Dashboard mini-widget ([`ui/dist/widget.html`](file:///home/kieren/Documents/code/optimis/plugins/behaviour_plugin/ui/dist/widget.html)).

---

## 2. Where the Contract Is Too Restrictive or Needs Adjustment

### 2.1 Scope Subsumption (`:all` Caller Invoking an `:own_groups` Endpoint)
* **Observed Friction:**
  In [`internal/host/server.go`](file:///home/kieren/Documents/code/optimis/internal/host/server.go#L1142-L1150) (`resolveDomainScopeMode`), the Host determines whether `ctx.dal.pupils` executes in `ScopeOwnGroups` or `ScopeAll` by inspecting the **endpoint's static `manifest.json` scope list** rather than the **caller's effective scope**.
  * When `log_behaviour_event` declares `["dal.pupils.read:own_groups"]`, a Head of Year, SLT member, or Admin (who holds `dal.pupils.read:all` in their role, and may not have personal timetable teaching groups) is still forced through `ScopeOwnGroups` (`ensurePupilInOwnGroups`) and blocked from logging a corridor or lunchtime behaviour incident.
  * Conversely, if the plugin author declares `["dal.pupils.read:all"]` in `manifest.json`, regular Classroom Teachers (who only hold `:own_groups`) are rejected at the HTTP gateway with `403 INSUFFICIENT_USER_SCOPES`.
* **Recommended Contract Adjustment:**
  Treat the manifest scope modifier (`:own_groups`) as the **minimum required scope**, and resolve the runtime DAL scope mode from the **caller's effective permissions**:
  * Caller holds `:own_groups` only $\rightarrow$ `ctx.dal.<domain>` enforces `ScopeOwnGroups`.
  * Caller holds `:all` $\rightarrow$ `ctx.dal.<domain>` runs in `ScopeAll` even when the endpoint minimum in `manifest.json` is `:own_groups`.

### 2.2 Cross-Boundary Batch Hydration (`ctx.plugin_db` $\leftrightarrow$ `ctx.dal.pupils`)
* **Observed Friction:**
  Because `plugin_behaviour.db` stores only `pupil_id` (preserving `core.db` as the single source of truth for pupil PII), a dashboard widget querying recent behaviour events from `ctx.plugin_db.query("SELECT pupil_id, ... FROM behaviour_events")` cannot perform a SQL `JOIN` against `core.pupils`. Currently, `ctx.dal.pupils` exposes `get_by_id(pupil_id)` and `list_by_group(group_id)`, but no batch pupil lookup.
  * Hydrating a widget summary across 10 pupils requires an $N+1$ `get_by_id` loop in Lua.
  * Worse, if even one row in `plugin_behaviour.db` belongs to a pupil outside the teacher's active groups, `get_by_id` raises a hard Lua error (`pupil does not belong to caller's active groups`), aborting the entire widget RPC.
* **Recommended Contract Adjustment:**
  * Add **`ctx.dal.pupils.get_batch({ pupil_ids = {...} })`** returning a table of pupils accessible to `ctx.caller` (filtering out inaccessible pupil IDs rather than throwing a fatal error on the whole batch).
  * Add **`ctx.dal.groups.list_my_groups()`** so standalone plugins and dashboard widgets can render a group/pupil picker when opened directly from the sidebar without a deep-linked `pupil_id`.

### 2.3 Stripped `os` Library Leaves No Server Timestamp or UUID Helper in Lua
* **Observed Friction:**
  Stripping `os` from `gopher-lua` (Section 9.2) is important for sandboxing, but it leaves Lua plugins without `os.time()`, `os.date()`, or a cryptographic UUID generator. In [`src/init.lua`](file:///home/kieren/Documents/code/optimis/plugins/behaviour_plugin/src/init.lua#L89-L108), the plugin had to accept `recorded_at` from client args / SQLite `strftime` and construct row IDs using `COUNT(*) + 1` (which is brittle under concurrent writes or row deletions).
* **Recommended Contract Adjustment:**
  Inject two capability-safe, deterministic helpers on `ctx`:
  * **`ctx.clock.now_utc()`**: Returns the Host's current UTC timestamp in ISO-8601 (`YYYY-MM-DDTHH:MM:SSZ`).
  * **`ctx.crypto.uuid()`**: Returns a cryptographically random v4/v7 UUID generated by the Go Host.

### 2.4 Caller Identity Propagation Between Host Shell and Child Iframes
* **Observed Friction:**
  When a session uses a persona override (`?staff_id=...` or the Topbar Persona Switcher), the React Host Shell sends `context.staff_id` into the iframe via `mis:init`. However, to prevent [`ResolveRequestCaller`](file:///home/kieren/Documents/code/optimis/internal/host/server.go#L1063-L1094) from falling back to a stale global dev persona when no cookie is set for a query-param override, [`ui/dist/index.html`](file:///home/kieren/Documents/code/optimis/plugins/behaviour_plugin/ui/dist/index.html) and [`ui/dist/widget.html`](file:///home/kieren/Documents/code/optimis/plugins/behaviour_plugin/ui/dist/widget.html) had to pass `staff_id` inside RPC `args`.
* **Recommended Contract Adjustment:**
  Plugin UIs should never pass `staff_id` in RPC `args`. The parent React Host Shell ([`ui/src/App.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/App.tsx)) should attach the active session/persona header on the `POST /api/plugins/:id/rpc/:method` fetch request itself so `args` remains strictly domain payload data and `ctx.caller.staff_id` is 100% host-derived.

### 2.5 Keep-Alive LRU Pool & Dashboard Widget Invalidation Notification
* **Observed Friction:**
  Because both full-page plugins (via the Section 9A Keep-Alive LRU Pool) and Home Dashboard widgets (Section 9B) remain mounted in the DOM across tab switches, mutating state in `index.html` (`mutates: true`) does not automatically notify an already-mounted `widget.html` iframe when the user switches back to the Home Dashboard (requiring an explicit "Refresh" button or manual re-fetch).
* **Recommended Contract Adjustment:**
  When an RPC call with `mutates: true` succeeds for `plugin_id` (or when a cached workspace/dashboard tab becomes visible again), the Host UI Shell should broadcast a lightweight `mis:plugin:invalidated` / `mis:visibility:shown` `postMessage` event to mounted iframes of that `plugin_id` so widgets and views can refresh automatically without polling.
