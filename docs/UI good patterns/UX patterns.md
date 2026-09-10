**Core Enterprise & Data-Heavy UI Types & Flows**

**1. Data Ingestion & Manipulation Types**

* **Editable Data Grid / Spreadsheet Table:** High-density, multi-cell tabular entry (keyboard-driven, batch pastes, inline formulas).
* **Matrix & Allocation Grid (Timetable / Rota):** 2D scheduling surfaces (time vs. resource/location, drag-and-drop, conflict collision).
* **Master-Detail / Split View:** High-level list paired with a side drawer, split-pane, or peek panel for reading and editing child records.
* **Hierarchical Tree-Table:** Collapsible parent-child nested rows within a single table layout.
* **Kanban / Pipeline Board:** Column-based card workflows representing sequential stages (intake, triage, review, enrolled).
* **Document-Centric / Rich-Text Canvas:** Mixed-format logging (e.g., student case notes, IEPs, behavior incident narratives with embedded structured tags).

**2. Form & Creation Types**

* **Single-Record Long Form:** Dense, stacked data collection (top-aligned, tab-indexed, inline validated).
* **Multi-Step Wizard:** Sequenced, linear data entry for multi-department intake, onboarding, or application workflows.
* **Bulk / Batch Modals:** Parameter application across dozens or hundreds of selected records simultaneously.

**3. Search, Discovery & Analysis Types**

* **Filter & Query Builder:** Faceted search sidebars, pill/chip filter bars, and boolean (AND/OR) rule-group query builders.
* **Aggregated Dashboard / Summary Cards:** High-level metric drill-downs that filter or deep-link directly into underlying transactional grids.
* **Audit Trail & Version History:** Diff viewers, timeline activity streams, and point-in-time record rollback views.

**4. Core Operational Flows**

* **Lookup & Link:** Searching an external/master directory to attach an existing entity (or inline-create if missing).
* **Staging & Review (Import/Draft Flow):** Uploading CSV/XLSX data into a temporary, editable scratchpad to fix errors before committing to the database.
* **Approval & State Transition:** Requesting, verifying, or rejecting state changes (e.g., grade submissions, timetable publishing, record archiving).
* **Exception & Conflict Resolution:** Highlighting collisions (double-booked rooms, invalid grade bounds) and routing the user through a guided fix.
