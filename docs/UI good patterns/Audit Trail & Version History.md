**Event Anatomy & Metadata Scaffolding**

* **Five-W Semantic Sentence:** Format every log entry header using the standard sentence structure: [Who] [Did What Action] to [Which Target Entity] on [Exact Timestamp] via [Channel/Method] (e.g., Sarah Jenkins (Registrar) changed SEN Tier for Liam Taylor on 08 Sep 2026, 10:42:15 BST via Web Portal).
* **Actor Attribution & Impersonation Flags:** Clearly distinguish between direct user actions, automated background system workers (System Scheduler), and admin-impersonated sessions (Admin John Smith acting as Sarah Jenkins).
* **Action Severity Badging:** Categorize log entries using color-coded badge pills:
  + *Informational / Read (Blue/Slate):* Exported CSV, Viewed Safeguarding Note.
  + *State Mutation (Amber):* Updated Record, Reassigned Set.
  + *Destructive / Security (Red):* Deleted Student Profile, Permissions Elevated, Failed Login Attempt (3x).
* **Network & Device Fingerprinting:** Include a collapsible technical metadata drawer per entry showing: IP Address, User Agent / OS, Session ID, and API Request ID for compliance and legal forensics.

**Visual Diff Views & Change Inspection**

* **Side-by-Side vs. Unified Diff Toggle:** Provide a toggle switch between:
  + *Unified Inline Diff:* Red strikethrough for removed values, green background for additions within a single block.
  + *Two-Column Split Matrix:* Left column = Previous Value; Right column = New Value.
* **Field-Level Granularity:** Isolate and display only the specific fields that changed rather than showing the entire unmodified payload.
* **Character-Level Highlighting for Rich Text:** In narrative fields (e.g., behavior notes, legal IEP logs), highlight the exact edited words within the paragraph rather than flagging the whole paragraph as replaced.
* **Complex Data Diff Parsing:** Format tabular and array mutations clearly (e.g., for attendance codes, show: Session 2: [A (Absent) ➔ P (Present)]).

**Timeline Navigation, Snapshots & Milestones**

* **Chronological Rail with Date Grouping:** Render a vertical connecting spine (tree-line) linking events, grouped into sticky day blocks (Today, Yesterday, 02 Sep 2026).
* **Milestone / Major Version Tagging:** Allow privileged users or automated triggers to flag key milestones (e.g., v2.0: Official End-of-Term Snapshot, Signed Off by Designated Safeguarding Lead), filtering out mundane auto-saves.
* **Point-in-Time State Reconstruction ("Time Travel"):** Provide a slider or dropdown allowing users to view the entire entity as it existed at any historical date/time (e.g., *"Viewing Liam Taylor's profile as of 15 March 2025"* with a persistent amber warning header).
* **Relative vs. Absolute Timestamp Toggle:** Display human-readable relative time by default (12 minutes ago, 3 days ago), but reveal the exact ISO timestamp (2026-09-08 10:42:15 BST) instantly on hover or global toggle.

**Rollback, Restore & Cherry-Picking**

* **Two-Step Granular Rollback:** Allow reverting a single specific field edit (e.g., *"Revert Emergency Phone Number to previous value"*) via an inline Revert Field action, without rolling back unrelated edits made in the same session.
* **Full-Snapshot Rollback Preview:** Clicking *"Restore to this version"* must never apply immediately. It opens a modal preview displaying a side-by-side comparison of current live data versus the selected historical snapshot, listing the exact fields that will be overwritten.
* **Rollbacks Logged as New Events:** Never overwrite or erase log history when rolling back. Restoring a past state must append a brand-new audit entry: Restored record to state from 12 Aug 2026.
* **Secondary Confirmation for Cascading Restores:** If restoring an older version impacts child records (e.g., restoring an older curriculum set that unlinks newly enrolled students), list all downstream records requiring re-verification.

**Search, Filtering & Forensic Scoping**

* **Multi-Facet Filter Toolbar:** Include top-level filter chips for:
  + *Actor:* Filter by specific staff member or system automated service.
  + *Action Type:* Filter by Created, Updated, Deleted, Exported, Viewed.
  + *Target Field:* Filter strictly to changes affecting a sensitive field (e.g., Show only Medical / Safeguarding edits).
  + *Date Range:* Absolute calendar and relative pickers.
* **Subject Access Request (SAR) / GDPR Export:** A dedicated one-click export macro (Export Audit History as Tamper-Proof PDF / CSV) formatted specifically for legal evidence, data subject requests, or regulatory audits.
* **Free-Text Change Search:** Support searching within the changed values themselves (e.g., searching for a phone number string +44 7700 finds all historical entries where that number was added, modified, or removed).

**Compliance, Immutability & Security Guards**

* **Write-Once, Read-Many (WORM) Immutability:** Audit records must be architecturally append-only. Visually communicate this by completely omitting any "Edit" or "Delete" controls on log items.
* **Tamper-Evident Verification Badges:** For statutory or legal records, include a cryptographic checksum or verification tag (Hash Verified) reassuring auditors that log entries have not been altered in the database.
* **Restricted View Access (RBAC):** Restrict access to sensitive change categories (e.g., salary adjustments, child protection log changes) behind elevated administrative roles or secondary password/MFA prompts.
