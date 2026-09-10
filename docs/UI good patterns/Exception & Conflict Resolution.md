**Collision Identification & Visual Triage**

* **Dual-Entity Conflict Framing:** Present collisions by showing both conflicting records side by side or stacked (e.g., Class 10A (Rm 204) vs. Staff Training (Rm 204)), rather than reporting an isolated error on just one record.
* **Semantic Collision Severity Badges:**
  + *Hard Blocks (Red):* Physical impossibilities (e.g., two lessons in the same room at the same time, staff assigned during approved medical leave).
  + *Soft Warnings / Policy Outliers (Amber):* Suboptimal states (e.g., student exceeds maximum recommended weekly study hours, teacher has 4 consecutive periods without a break).
* **Direct "Jump to Collision" Links:** Inside the conflict banner, provide a one-click anchor link ([View Conflicting Entry]) that scrolls the viewport directly to the other involved cell, card, or row.
* **Dedicated Conflict Triage Queue:** On complex scheduling or bulk operations, surface a persistent drawer or tab: Exceptions to Resolve (6). Clicking items steps the user through each conflict sequentially.

**Resolution Workflows & Decision Cards**

* **Guided Remediation Paths (Actionable Choices):** Avoid generic *"Please fix this error"* text. Offer mutually exclusive, one-click resolution buttons:
  + [Swap Bookings]: Transposes the two assignments in a single transaction.
  + [Move to Unassigned Pool]: Detaches the conflicting record to a staging dock for later placement.
  + [Reassign Room/Staff]: Opens an inline replacement picker.
  + [Cancel Entry]: Aborts the incoming operation.
* **Proactive Alternative Recommendations:** The system queries available inventory and suggests verified conflict-free options (e.g., *"Room 204 is busy, but Room 208 and Science Lab 2 are free at this period. [Assign Room 208]"*).
* **In-Context Field Disambiguation (Merge vs. Overwrite):** In data-sync collisions (e.g., a teacher edited student records offline while an admin updated them online), present a three-way diff picker:
  + Column A: Local Offline Value
  + Column B: Server Database Value
  + Action: Field-by-field radio buttons (Keep Local, Keep Server, or Custom Edit).

**Administrative Overrides & Policy Exceptions**

* **Explicit Override Checkboxes:** For soft policy conflicts, provide an explicit, un-prechecked toggle: [ ] Acknowledge and Override Policy.
* **Mandatory Justification Log:** Gating the override behind a required text field: *"Reason for exception (e.g., Authorized by Principal)"*, which is permanently stamped into the record’s audit trail.
* **Elevated Credential Gate:** If an override exceeds standard user authority (e.g., exceeding statutory student-to-teacher classroom ratios), prompt for secondary manager authentication (PIN, password, or biometric signoff) inline without kicking the user out of the flow.

**Pre-Flight Prevention & Simulation**

* **Real-Time Collision Previews (Ghosting):** When dragging or selecting an entity, highlight potential conflict slots with red striped backgrounds before the user drops or commits the record.
* **Impact Radius Calculation (Blast Radius Warning):** If fixing a conflict cascades into other records, preview the consequences before executing (e.g., *"Moving English 10B to Period 4 will cause 3 student timetable clashes"*).
* **"Dry Run" Conflict Check:** On bulk imports, roster shifts, or timetable publishing, provide a Validate & Simulate button that runs conflict-detection algorithms in memory and generates an exception ledger before any database write occurs.

**Safety, Undo & Resolution History**

* **Atomic Transaction Safety:** Ensure multi-record fixes (like swaps or cascaded moves) occur in an all-or-nothing database transaction—if one step fails, the entire state rolls back to prevent half-resolved states.
* **Resolution Undo Banner:** Upon resolving an exception, present a sticky toast: Resolved room collision between 10A and 11B. [Undo Resolution], active for 15 seconds.
* **Resolution History Stamp:** Display a discreet informational tag on resolved records: Manual Override Applied by S. Jenkins on 08/09/2026: "Temporary exam overflow".
