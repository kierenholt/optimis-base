# Plugins Workspace (`plugins`)

This directory contains unpacked plugin bundles mounted by the Go Host Container during local development and automated test verification (`PLUGINS_DIR`).

## Subdirectories

* [`plugins/parent_dmz_plugin`](file:///home/kieren/Documents/code/optimis/plugins/parent_dmz_plugin/GEMINI.md): First-party **Parent Portal DMZ Sync** plugin (`uk.org.optimis.parent.dmz`) per Architecture Blueprint Sections 1, 3, 5, 6 & 9:
  * [`manifest.json`](file:///home/kieren/Documents/code/optimis/plugins/parent_dmz_plugin/manifest.json): Default-deny routing, `database.alias` (`plugin_parent_dmz`), `gdpr_hooks`, `egress.allowed_domains` (`["https://dmz.parent.optimis.org.uk"]`), and `dal.pupils.read:all`, `dal.guardians.read:all`, `dal.attendance.read:all` scoped endpoints.
  * [`src/init.lua`](file:///home/kieren/Documents/code/optimis/plugins/parent_dmz_plugin/src/GEMINI.md): Outbound DMZ write-inbox pull (`pull_dmz_inbox` enforcing Parental Responsibility & court-order blocks with `REJECTED_SAFEGUARDING_BLOCK`), outbound sanitized read-projection push (`push_sanitized_projection` stripping statutory `UPN` and confidential household addresses), and UK GDPR SAR/retention purge hooks.
  * [`migrations/`](file:///home/kieren/Documents/code/optimis/plugins/parent_dmz_plugin/migrations/GEMINI.md): Non-destructive SQLite schema migrations (`001_init.sql` creating `dmz_inbox_submissions` and `dmz_projection_snapshots`).
  * [`ui/`](file:///home/kieren/Documents/code/optimis/plugins/parent_dmz_plugin/ui/GEMINI.md): Offline-bundled micro-frontend (`ui/dist/index.html`) running inside `<iframe sandbox="allow-scripts">` with `connect-src 'none'` CSP.
* [`plugins/dfe_census_plugin`](file:///home/kieren/Documents/code/optimis/plugins/dfe_census_plugin/GEMINI.md): First-party hot-swappable **DfE Statutory School Census** plugin (`uk.gov.dfe.census`) per Architecture Blueprint Section 4 & Section 9:
  * [`manifest.json`](file:///home/kieren/Documents/code/optimis/plugins/dfe_census_plugin/manifest.json): Default-deny routing, `database.alias` (`plugin_dfe_census`), `gdpr_hooks`, `egress.allowed_domains` (`["https://collect.dfe.gov.uk"]`), and `dal.census.read:all` scoped endpoints.
  * [`src/init.lua`](file:///home/kieren/Documents/code/optimis/plugins/dfe_census_plugin/src/GEMINI.md): Sandboxed Lua 5.1 census return generation (`ctx.dal.census.get_statutory_rollup`), deterministic XML/XSD validation (`ctx.xml.build` & `ctx.xml.validate_xsd`), COLLECT submission (`ctx.http.post`), and UK GDPR SAR/retention purge hooks.
  * [`migrations/`](file:///home/kieren/Documents/code/optimis/plugins/dfe_census_plugin/migrations/GEMINI.md): Non-destructive SQLite schema migrations (`001_init.sql` creating `census_returns` and `census_pupil_snapshots`).
  * [`ui/`](file:///home/kieren/Documents/code/optimis/plugins/dfe_census_plugin/ui/GEMINI.md): Offline-bundled micro-frontend (`ui/dist/index.html`) running inside `<iframe sandbox="allow-scripts">` with `connect-src 'none'` CSP.
* [`plugins/test_plugin`](file:///home/kieren/Documents/code/optimis/plugins/test_plugin/GEMINI.md): Reference synthetic fixture plugin (`uk.org.optimis.test`) exercising the full Plugin Contract (Section 9):
  * [`manifest.json`](file:///home/kieren/Documents/code/optimis/plugins/test_plugin/manifest.json): Default-deny routing, `database.alias` (`plugin_test`), `gdpr_hooks`, `egress.allowed_domains` & `egress.timeout_ms`, and scoped endpoints.
  * [`src/init.lua`](file:///home/kieren/Documents/code/optimis/plugins/test_plugin/src/GEMINI.md): Sandboxed Lua 5.1 endpoint and GDPR hook implementations (`ctx.dal`, `ctx.plugin_db`, `ctx.events`, `ctx.http`).
  * [`migrations/`](file:///home/kieren/Documents/code/optimis/plugins/test_plugin/migrations/GEMINI.md): Non-destructive SQLite schema migrations (`001_init.sql`, `002_worker_event_logs.sql`).
  * [`ui/`](file:///home/kieren/Documents/code/optimis/plugins/test_plugin/ui/GEMINI.md): Offline-bundled micro-frontend (`ui/dist/index.html` & `ui/dist/widget.html`) running inside `<iframe sandbox="allow-scripts">` with `connect-src 'none'` CSP.
