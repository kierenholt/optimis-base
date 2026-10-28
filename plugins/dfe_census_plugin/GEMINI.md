# DfE Statutory School Census Plugin (`plugins/dfe_census_plugin`)

Implements the hot-swappable termly **DfE Statutory School Census** plugin (`uk.gov.dfe.census`) per Architecture Blueprint Section 4 & Section 9.

## Key Files

* [`manifest.json`](file:///home/kieren/Documents/code/optimis/plugins/dfe_census_plugin/manifest.json): Default-deny permission contract declaring `plugin_dfe_census` attached SQLite storage, `gdpr_hooks` (`export_pupil_data`, `purge_pupil_data`), `https://collect.dfe.gov.uk` outbound HTTPS egress whitelist, and `dal.census.read:all` scoped endpoints.
* [`migrations/`](file:///home/kieren/Documents/code/optimis/plugins/dfe_census_plugin/migrations/GEMINI.md) ([`001_init.sql`](file:///home/kieren/Documents/code/optimis/plugins/dfe_census_plugin/migrations/001_init.sql)): Expand-and-contract DDL creating `census_returns` and `census_pupil_snapshots` in `plugin_dfe_census.db`.
* [`src/`](file:///home/kieren/Documents/code/optimis/plugins/dfe_census_plugin/src/GEMINI.md) ([`src/init.lua`](file:///home/kieren/Documents/code/optimis/plugins/dfe_census_plugin/src/init.lua)): Lua 5.1 implementation of `generate_and_validate_census_return` (calling `ctx.dal.census.get_statutory_rollup`, `ctx.xml.build`, and `ctx.xml.validate_xsd`), `submit_census_return` (executing outbound `ctx.http.post` to DfE COLLECT prior to any SQLite write transaction), `list_census_returns`, and UK GDPR SAR/retention hooks (`export_pupil_data`, `purge_pupil_data`).
* [`ui/`](file:///home/kieren/Documents/code/optimis/plugins/dfe_census_plugin/ui/GEMINI.md) ([`ui/dist/index.html`](file:///home/kieren/Documents/code/optimis/plugins/dfe_census_plugin/ui/dist/index.html)): Sandboxed micro-frontend (`connect-src 'none'`) wired via `/sdk/mis-bridge.js` and `data-mis-rpc` attributes for termly census rollup inspection, CBDS XML preview, XSD status badge, and COLLECT return submission.
