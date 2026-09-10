**Header Framing & Target Scope**

* **Explicit Quantity Counter:** Name the exact scope in the modal header (e.g., Update Status for 42 Selected Students), never a generic title like Edit Records.
* **Selection Breakdown Pills:** When selections span distinct categories, show a compact breakdown chip row directly under the title (e.g., 28 Year 10 • 14 Year 11).
* **Collapsible Recipient/Entity Drawer:** Include a subtle View selected items (42) accordion or link that opens a scrollable list inside the modal to let users deselect individual outliers without closing the modal and losing their selections.

**Field-Level Intent & Partial Overwrites**

* **Explicit "Apply to Field" Checkboxes:** When updating multi-attribute records (e.g., updating Tutor Group and SEN Tier simultaneously), place an unchecked toggle beside each input. Only fields explicitly checked by the user get overwritten; all unchecked fields retain their existing disparate values.
* **"No Change" / Placeholder Distinction:** Never pre-fill bulk inputs with default values. Keep inputs blank with an explicit watermark placeholder: [Multiple Values — Leave blank to keep unchanged].
* **Append vs. Replace Toggles:** For multi-value fields (e.g., Tags, Guardian Notifications, Medical Alerts), provide an explicit segmented control:
  + Replace Existing: Clears previous tags and sets the new selection.
  + Append / Add: Merges new tags with each record's existing tags without overwriting past data.
  + Remove Selected: Strips specified tags from all target records if present.

**Conflict Pre-Flight & Eligibility Warnings**

* **Split Eligibility Alert Banner:** Before the user submits, run a client-side pre-flight check and partition the target pool:
  + *Eligible (39):* Will receive update.
  + *Ineligible / Conflicts (3):* Will be skipped.
* **Conflict Reason Inspection:** Show an expandable warning table listing reasons for skipped entities (e.g., Smith, Liam: Locked by ongoing exam entry, Taylor, Ava: Missing primary guardian phone).
* **One-Click Remediation Filter:** Provide an inline action: Exclude 3 invalid records and continue with 39.

**Destructive Guards & Confirmation Rigor**

* **Typed Keyword Verification:** For irreversible bulk actions (e.g., permanent deletion, cohort-wide archiving, timetable resets), require typing a specific confirmation keyword (e.g., CONFIRM or ARCHIVE 42) before the primary button activates.
* **Downstream Impact Disclosure:** Explicitly itemize secondary cascading effects in a warning box (e.g., *"This will also detach 126 parent portal links and cancel 14 scheduled invoices"*).
* **Two-Stage Action Button Hierarchy:** Color destructive bulk actions in high-contrast danger red (Delete 42 Records); style the secondary dismissal (Cancel / Keep Records) with high visual prominence so it is easy to abort.

**Execution State, Progress & Rollback**

* **Deterministic In-Modal Progress Bar:** When executing large batches (>50 records), do not use an ambiguous infinite spinner. Render a linear progress bar showing real-time execution (e.g., Updating record 84 of 210... (40%)).
* **Non-Blocking Background Transition:** For very large operations (>500 records), offer a Run in background button so the user can dismiss the modal and continue working while the batch processes via a toaster notification.
* **Batch Execution Error Summary:** If some operations fail during processing, do not fail silently. Display a post-execution error matrix: 198 succeeded, 12 failed with an instant Download Error Log (CSV) button and a Retry Failed Only button.
* **Granular Bulk Undo Trigger:** Surface a sticky bottom toast notification immediately upon completion: Updated 42 students. [Undo Bulk Action], active for 10–15 seconds via an atomic database rollback transaction.
