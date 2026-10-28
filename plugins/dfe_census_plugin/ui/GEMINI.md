# DfE Census Plugin Sandboxed Micro-Frontend UI (`plugins/dfe_census_plugin/ui`)

This directory contains [`dist/index.html`](file:///home/kieren/Documents/code/optimis/plugins/dfe_census_plugin/ui/dist/index.html), the offline-bundled micro-frontend for `uk.gov.dfe.census` served inside `<iframe sandbox="allow-scripts">` with a strict `connect-src 'none'` Content Security Policy.

---

## Key UI Elements & Bridge Bindings

* Loads [`/sdk/mis-theme.css`](file:///home/kieren/Documents/code/optimis/sdk/ui/mis-theme.css) and [`/sdk/mis-bridge.js`](file:///home/kieren/Documents/code/optimis/sdk/ui/mis-bridge.js).
* **Controls**:
  * `data-testid="census-term-select"`: Selects statutory term (`AUTUMN`, `SPRING`, `SUMMER`).
  * `data-testid="census-date-input"`: Selects reference census date (`YYYY-MM-DD`).
  * `data-testid="generate-census-btn"` (`data-mis-rpc="generate_and_validate_census_return"`): Generates statutory rollups, builds CBDS XML, and runs XSD validation.
  * `data-testid="submit-census-collect-btn"` (`data-mis-rpc="submit_census_return"`): Submits the validated XML return outbound to `https://collect.dfe.gov.uk/v1/school-census/submit`.
* **Rollup & Diagnostic Panels**:
  * `data-testid="census-summary-cards"`: Renders `census-on-roll-count`, `census-sen-breakdown` (`N`/`K`/`E`), `census-fsm-count`, `census-lac-count`, `census-attendance-summary`, and `census-xsd-status` (`XSD_VALID`).
  * `data-testid="census-xml-preview"`: Displays the generated `<Message>` DfE CBDS XML payload.
  * `data-testid="census-collect-receipt"`: Displays the DfE COLLECT receipt (`COLLECT_ACCEPTED` + `submission_id`).
  * `data-testid="census-error-banner"`: Renders fail-fast XSD validation, RBAC scope, or `MOCK_DFE_CENSUS_SERVICE_DISABLED` errors.
