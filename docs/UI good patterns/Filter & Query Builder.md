**Active Filter Anatomy & Removable Pills (Filter Chips)**

* **Structured Pill Format:** Render active filters using an explicit [Field] : [Operator] [Value] format (e.g., Attendance : < 85%, Year Group : 10 or 11). Never hide the applied criteria behind generic buttons.
* **Direct Click-to-Edit Popover:** Clicking the body of a filter pill opens an inline popover allowing immediate adjustments to values or operators without deleting and re-creating the filter.
* **One-Click Individual Removal (✕):** Position a distinct remove icon on the right edge of each pill with sufficient hit padding to prevent accidental edits.
* **Global Clear Affordance:** Include an unambiguous Clear All Filters text button adjacent to the pill strip that appears only when at least one active filter is applied.
* **Filter Bar Overflow Wrap:** Wrap active filter chips onto a secondary line or collapse them into a compact counter pill (+3 more filters...) with a dropdown preview to avoid breaking horizontal page layouts.

**Filter Dropdown / Popover Anatomy**

* **Instant In-Menu Search (Combobox):** Include a search input at the top of multi-select filter lists (e.g., filtering through 40+ subjects or 200+ staff members) that highlights matching text.
* **Faceted Live Counts:** Display dynamic matching count badges beside each checkbox in the filter dropdown (e.g., Year 7 (142), Year 8 (138)) so users know the expected data yield before toggling.
* **"Select All" & "Deselect All" Controls:** Provide sticky utility buttons above long option lists to avoid forcing users to manually click dozens of checkboxes.
* **Explicit vs. Instant Apply:**
  + *Instant Apply (Live Refilter):* Best for simple single-select toggles (e.g., sorting by Status) with fast, local client-side data.
  + *Explicit Apply Button:* Best for multi-select checklists and complex date ranges; keep selections staged inside the popover until the user clicks Apply, preventing unnecessary network roundtrips.

**Advanced Query Builders (Rule-Group & Boolean Logic)**

* **Visual Clause Stacking:** Represent complex logic using nested visual cards or indented rule blocks, avoiding raw SQL-like text syntax.
* **Explicit AND / OR Segmented Toggles:** Provide a toggle switch at the head of every rule group: Match ALL of the following (AND) vs. Match ANY of the following (OR).
* **Field-Driven Operator Adaptability:** Dynamically adjust the available condition operators based on the data type of the selected field:
  + *Numeric (e.g., Marks, Age):* is equal to, is greater than, is between, is empty.
  + *Text/Strings (e.g., Name, Notes):* contains, starts with, exact match, does not contain.
  + *Categorical/Enums (e.g., House, Tutor Group):* is any of, is none of.
  + *Dates (e.g., Enrollment Date):* in the past N days, between [Date] and [Date], before, after.
* **Add Rule vs. Add Group Buttons:** Include clear secondary triggers: + Add Condition (appends a single line within the current block) and + Add Condition Group (creates a nested sub-group for parenthetical logic like (A OR B) AND (C OR D)).
* **Relative Date Shorthands:** Offer rolling, dynamic date options (e.g., Today, This Academic Term, Last 30 Days) alongside static calendar pickers so automated reports never expire.

**View Presets, Persistence & Sharing**

* **Save View Workflow:** Allow users to name, save, and bookmark complex filter sets into a persistent "Views" dropdown (e.g., Persistent Absentees (<90%), Year 10 SEN Cohort).
* **System vs. Custom Views:** Partition saved views into:
  + *Standard System Views:* Pre-built institutional presets (e.g., All Active Students, Graduating Class).
  + *My Views:* Personal custom queries created by the logged-in user.
* **URL Parameter Serialization:** Encode all active filters, boolean operators, and sorting orders cleanly into the URL query string (e.g., ?status=active&attendance\_lt=90&sort=last\_name\_asc) so exact search states can be bookmarked or shared via email/Slack.
* **Default View Setting:** Allow users to mark any custom query as their personal landing default whenever they visit that data table.

**Empty States, Pre-Flight & Performance Feedback**

* **Zero-Results Remediation:** When filters yield zero records, avoid showing a blank screen. Display an informative empty state: *"No students match your active filters"* accompanied by an action button: *"Reset all filters"* or *"Broaden search criteria"*.
* **Live Result Pre-Flight Counter:** In complex query builders, display a live estimated match counter beside the Apply button (e.g., Matches ~48 records) before running the search, preventing queries that accidentally yield zero or millions of results.
* **Async Query Throttling / Debouncing:** Debounce text search inputs by 300–400ms to prevent firing an API request on every individual keystroke.
* **Non-Matching Field Warnings:** If a filter references a conflicting field (e.g., searching for Year 7 AND Enrolled in Advanced Calculus), display an inline hint warning the user of potential mutually exclusive criteria.
