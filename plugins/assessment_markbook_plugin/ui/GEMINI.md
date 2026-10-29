# Assessment & Markbook Sandboxed UI (`plugins/assessment_markbook_plugin/ui`)

This directory contains the offline-bundled micro-frontend for `uk.org.optimis.assessment.markbook`, rendered inside `<iframe sandbox="allow-scripts">` with a strict `connect-src 'none'` Content Security Policy.

## Files

* [`dist/index.html`](file:///home/kieren/Documents/code/optimis/plugins/assessment_markbook_plugin/ui/dist/index.html):
  * Uses `/sdk/mis-theme.css` (`--mis-*` design tokens) and `/sdk/mis-bridge.js` (`window.mis.onInit`, `window.mis.rpc()`, and declarative `data-mis-rpc` attributes).
  * Displays the synced MAT grading schema version badge (`markbook-mat-schema-badge`) and caller role badge (`markbook-caller-role-badge`).
  * Binds `create-markbook-column-btn` (`data-mis-rpc="create_markbook_column"`), `save-pupil-grades-btn` (`data-mis-rpc="save_pupil_grades"`), and `load-department-rollup-btn` (`data-mis-rpc="get_department_attainment_rollup"`).
  * Renders the pupil grade & comment entry table (`markbook-grade-select-<pupil_id>`, `markbook-comment-input-<pupil_id>`, `markbook-saved-grade-<pupil_id>`), HoD cross-set attainment comparison panel (`markbook-department-rollup-panel`, `rollup-set-avg-<group_id>`, `rollup-highest-set-name`), and fail-fast error banner (`markbook-error-banner`).
