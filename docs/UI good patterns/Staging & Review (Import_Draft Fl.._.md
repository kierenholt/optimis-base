**File Ingestion & Raw Preview**

* **Drag-and-Drop Zone with Instant Feedback:** Provide a dedicated upload target that accepts .csv, .tsv, and .xlsx files, instantly displaying file size, row count, and detected character encoding.
* **Direct Raw File Preview:** Show a read-only 5-row preview of the unparsed file immediately after upload so users can verify they uploaded the correct file before committing to mapping.
* **Auto-Delimiter & Encoding Detection:** Automatically detect separators (comma, semicolon, tab) and UTF-8 vs. ISO encodings; offer a fallback manual override selector if the preview appears malformed.
* **Multi-Tab Sheet Selector:** If an Excel workbook contains multiple worksheets, prompt the user to choose which sheet to stage before advancing.

**Column Mapping & Schema Alignment**

* **Fuzzy Auto-Matching:** Automatically match uploaded column headers to system fields based on semantic similarity (e.g., mapping Surname, Last Name, and Family Name to the system field last\_name).
* **Side-by-Side Mapping Matrix:** Display a clean two-column mapping table:
  + *Left Column:* Source Header from file + 3 sample values from real rows.
  + *Right Column:* System Target Field dropdown with status pills (Matched, Ignored, Required).
* **Unmapped Required Fields Banner:** Pin an alert at the top of the mapping screen detailing mandatory database fields that remain unassigned (e.g., Missing required: Date of Birth, Student ID).
* **Transformation & Format Harmonization:** Allow light inline transformations during mapping via simple toggles (e.g., auto-capitalize names, trim whitespace, or select source date format: DD/MM/YYYY vs. MM/DD/YYYY).
* **Ignore/Skip Column Controls:** Provide an explicit Do not import this column option for each field to strip extraneous spreadsheet data.

**The Staging Scratchpad (Editable Sandbox Grid)**

* **Isolated Staging Environment:** Load parsed data into a temporary, isolated workspace table where users can edit values directly without altering production records.
* **Inline Error Highlighting:** Highlight invalid cells directly within the grid with high-contrast red borders, displaying specific failure reasons on hover (e.g., Invalid email format, Grade out of bounds [0–100]).
* **Direct Cell In-Place Editing:** Allow users to double-click and correct invalid data directly inside the staging table rather than forcing them to fix the original CSV in Excel and re-upload.
* **Error-Filtering Toolbar (The Triage Switch):** Provide a segmented view controller at the top of the grid:
  + All Rows (1,240)
  + Valid Rows (1,180)
  + Errors Only (60) — allows users to isolate and work down failing records.

**Duplicate Detection & Record Matching**

* **Match Strategy Configuration:** Allow the user to specify unique identifier rules before processing (e.g., match existing students by Student ID or Email + Date of Birth).
* **Duplicate Collision Modes:** Offer an explicit segmented strategy for existing matches:
  + *Skip:* Leave existing database records untouched.
  + *Overwrite / Update:* Update existing records with incoming staged data.
  + *Create Copy / Duplicate:* Ingest as distinct new entities.
* **Side-by-Side Collision Resolver:** For flagged ambiguous duplicates, open a two-column comparison card displaying Existing Record vs. Incoming Staged Data with field-level radio buttons to select which value survives.

**Batch Fixes & Bulk Remediation**

* **Column-Level Batch Transformations:** Provide a contextual menu on staging column headers with instant fix macros:
  + *Fill Blanks:* Fill all empty cells in this column with a default value (e.g., set missing Country to United Kingdom).
  + *Trim Spaces:* Strip leading/trailing whitespace across all rows.
  + *Find and Replace:* Swap legacy codes across the entire staged dataset (e.g., replace Y with Yes).
* **Bulk Row Deletion:** Allow selecting and discarding entire rows that are irrecoverably broken or irrelevant.
* **"Export Remaining Errors" Fallback:** If hundreds of errors exist, provide an action to Download Rows with Errors as CSV (with an appended Error\_Reason column), commit the valid rows immediately, and allow re-uploading the fixed remainder later.

**Commit, Audit & Rollback**

* **Pre-Flight Validation Check:** Disable the primary Commit Import button until 0 critical schema errors remain, or provide an explicit checkbox: Import only 1,180 valid rows and discard 60 errors.
* **Summary Confirmation Screen:** Before final database write, display an impact ledger:
  + Records to Create: 940
  + Records to Update: 240
  + Records Skipped: 60
* **Deterministic Commit Progress Bar:** During database insertion, render a live progress indicator (Writing record 450 of 1,180...) to assure users the browser has not frozen.
* **One-Click Batch Rollback (The "Undo Import" Token):** Tag all newly created records with an import\_batch\_id. In the audit history, provide an explicit Undo Entire Import button for up to 24–48 hours, cleanly removing all created records and restoring overwritten values.
