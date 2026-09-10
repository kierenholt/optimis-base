**Navigation & Focus Architecture**

* **Strict Tab & Enter Directionality:** Tab moves right, Shift+Tab moves left; Enter commits and moves down, Shift+Enter commits and moves up.
* **Two-Stage Cell Focus vs. Edit Mode:** Single click or arrow navigation places an outer focus ring on the cell without activating an input; double-clicking, pressing F2, or typing any alphanumeric character enters edit mode.
* **Escape-Key Reversion:** Pressing Esc in edit mode immediately discards uncommitted cell edits and returns the cell to passive focus mode without shifting focus.
* **Arrow-Key Free-Roam:** Arrow keys traverse adjacent cells freely when not in edit mode; in edit mode, arrow keys navigate text cursor positions inside the cell.
* **Page-Level Traversal Keys:** Home jumps to the first cell of the current row; End jumps to the last populated cell; Ctrl/Cmd + Home jumps to cell A1; Ctrl/Cmd + End jumps to the bottom-right active cell.
* **Auto-Advancement on Fixed Input:** Commit the value and shift active focus downward automatically upon receiving a single-character keypress (e.g., grading scales 1–9 or attendance P/A/L).
* **Focus Retention Across Refreshes:** Retain row/column index focus even after server re-validations, sorting triggers, or pagination changes.

**Viewport & Display Density**

* **Bi-Directional Sticky Headers (Freezing):** Lock column headers to the top on vertical scroll; pin identity columns (e.g., ID, Photo, Student Name) to the left on horizontal scroll.
* **Configurable Density Toggles:** Provide three row-height settings (Compact: 28–32px, Normal: 40px, Comfortable: 48px) so power users can maximize visible screen data.
* **Tabular (Monospaced) Number Alignment:** Use font-variant-numeric: tabular-nums to ensure all numeric digits have identical widths, aligning decimal points vertically.
* **Strict Left vs. Right Alignment:** Left-align alphanumeric text; right-align numbers, currencies, and percentages; center status badges, icons, and dates.
* **Single-Line Truncation with Zero-Delay Tooltips:** Prevent text from wrapping and breaking consistent row heights; truncate with ellipses (...) and reveal full text via an instant tooltip on hover/focus.
* **Persistent Bottom Aggregation Row:** Lock a summary row at the bottom displaying dynamic calculations (Sum, Average, Count, Median) that update instantly as numbers are keyed in.
* **Visual Striping / High-Contrast Borders:** Subtle alternating zebra striping (#FFFFFF / #F9FAFB) or distinct row borders to prevent horizontal eye drift across wide monitors.

**Data Ingestion & Batch Editing**

* **Excel-Style Fill Handle (Drag-to-Fill):** A small square handle on the bottom-right corner of the active cell to drag values across rows or columns, duplicating content or incrementing sequences.
* **Multi-Cell Range Selection:** Enable Shift + Click or mouse-drag across contiguous blocks of cells for bulk copying, clearing, or formatting.
* **Native Clipboard Matrix Parsing:** Allow pasting multi-column, multi-row clipboard data directly from Excel or Google Sheets, automatically mapping values into target cells.
* **One-Click Default Filling:** Contextual header macros to fill remaining blank cells with a standard default (e.g., "Mark all remaining unmarked as Present").
* **Find and Replace (Ctrl/Cmd + F & H):** In-grid search overlay that highlights matching terms across off-screen cells, with support for batch replacements.
* **Floating Batch Toolbar:** Selecting multiple row checkboxes triggers a floating bottom action dock (e.g., "Assign Set", "Export Selection", "Delete Selected") rather than cluttering header controls.

**Validation, Error Prevention & Feedback**

* **Non-Blocking Inline Errors:** Never interrupt the typing flow with blocking modal alerts; flag errors with an amber/red cell border or corner ribbon, keeping entry fluid.
* **Input Masks:** Format phone numbers, postal codes, and structured IDs automatically as the user types without requiring manual entry of symbols.
* **Soft Outlier Warnings:** Differentiate hard schema errors (blocking save) from unusual values (e.g., exam mark of 98 in a 40-average cohort) by using an advisory badge that permits saving.
* **"Show Errors Only" Filter Toggle:** A dedicated switch in the table toolbar that instantly collapses the view to only rows containing validation errors.
* **Cell-Level Dirty Indicators:** Display a subtle visual indicator (e.g., an amber triangle in the top-left cell corner) indicating that data has changed but is not yet synced to the database.
* **Ambient Auto-Save Status:** Save edits in the background via debounced network requests, displaying unobtrusive status text in the header (Saving... → All changes saved).
* **Local Crash Protection (IndexedDB/Storage):** Cache unsaved cell inputs locally to protect against browser crashes or network dropouts mid-entry.
* **Multi-Level Undo/Redo (Ctrl/Cmd + Z & Ctrl/Cmd + Y):** Maintain a full stack of cell-level changes, allowing users to reverse single edits, bulk pastes, or accidental deletions.

**Cell Editor Types & Inline Controls**

* **Keyboard-First Combobox / Typeahead:** Typing in a categorical cell filters matching options immediately; hitting Enter or Tab selects the top match without mouse interaction.
* **Direct-Entry Date Parsing:** Allow direct typing of shorthand dates (080926 or 8/9) with automatic parsing into standard formats, alongside a secondary calendar icon.
* **Single-Click Quick Toggles:** Provide instant toggles for binary states (e.g., booleans, checkboxes, attendance tags) that flip state on a single click without opening a nested input element.
* **Ghost Actions on Hover:** Display row-level utility icons (e.g., row delete, duplicate, open detail drawer) only when a user hovers over the specific row to reduce visual noise.
* **Bulk Multi-Select Tags:** Allow inline multi-select pill inputs within a single cell, expandable into a mini-popover if selections exceed cell width.
