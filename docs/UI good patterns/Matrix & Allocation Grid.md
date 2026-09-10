**Cell Anatomy & Micro-Information Architecture**

* **Two-Tier Typographic Hierarchy:** The primary subject (e.g., Class 10-A, Shift: Night) is bold and top-aligned; secondary metadata (Rm 204, Smith, J., Ratio 1:8) sits below in muted, tabular text.
* **Category Color-Coded Accent Strips:** Use a 3–4px solid color bar on the left edge of each card for categorization (e.g., Department, Subject, Shift Type) instead of full-card background fills, preserving text contrast.
* **Ghost Plus Affordance on Empty Cells:** Empty allocation blocks display an ultra-faint outline and a ghost + icon on hover/focus to indicate click-to-add availability.
* **Compact Overflow Counter Badges:** If a single time slot contains multiple events, render the first 1–2 cards plus a compact pill (e.g., +3 more) that opens an unconstrained popover on click.
* **Capacity & Utilization Micro-Meters:** Show a subtle fraction or mini progress track inside the cell (e.g., 28/30 desks or 4/5 staff assigned) that shifts to amber/red when at or near threshold.
* **Status Pill Badges:** Display tiny, pill-shaped tags inside the block for operational states: Unconfirmed, Substitute Required, Published, or Overtime.

**Viewport, Axes & Layout Control**

* **Dual-Axis Pinning (Bi-Directional Freeze):** Lock both the horizontal time header (e.g., 09:00, Period 1) and the vertical entity column (e.g., Staff Member, Room Number) during scrolling.
* **One-Click Axis Inversion (Pivot Toggle):** Allow users to flip rows and columns with a single button:
  + *Resource View:* Rows = Staff/Rooms, Columns = Time Slots.
  + *Schedule View:* Rows = Time Slots, Columns = Groups/Classes.
* **Granularity Scale Switcher:** Instant toggles for grid scale (e.g., 15 min, 30 min, 1 hour, Full Day / Period) to quickly switch between granular staffing and bird’s-eye planning.
* **Current-Time Indicator ("Now" Ruler):** A distinct vertical line (typically bright red or blue) cutting across the active day's column indicating the exact current time, updating every 60 seconds.
* **"Jump to Today / Now" Button:** A persistent toolbar control that snaps horizontal and vertical scroll positions directly to the current date and active time block.
* **Non-Working Hours / Weekend Dimming:** Apply a muted cross-hatch or shaded background to non-operational hours, weekends, or scheduled breaks (e.g., Lunch, Recess) to focus user attention on active slots.

**Direct Manipulation, Drag-and-Drop & Mouse UX**

* **Translucent Ghost Drag Previews:** When dragging an assignment, display a semi-transparent replica of the card under the cursor, keeping target cells visible underneath.
* **Slot Magnetic Snap-to-Grid:** Cards automatically snap to valid slot boundaries (e.g., 15-minute or period increments) rather than free-floating across arbitrary pixel coordinates.
* **Pre-Drop Target Highlighting:** As a dragged card hovers across cells, tint valid destination slots green and invalid destination slots red before mouse release.
* **Edge-Triggered Auto-Scroll:** Dragging a card near the boundary of the viewport (top, bottom, left, right) automatically scrolls the grid smoothly in that direction.
* **Occupied Slot Conflict Dialog:** Dropping a card onto an already occupied slot triggers an explicit contextual popover with actions: *Swap Slots*, *Double-Book*, or *Cancel*.
* **Multi-Slot Selection Drag:** Allow users to click and drag across an empty time range (e.g., from 09:00 to 12:00) to create an extended, multi-period allocation block in one gesture.
* **Edge-Resize Handles:** Provide draggable left/right or top/bottom handles on cards to adjust duration directly on the grid without opening an edit modal.

**Keyboard-Driven Allocation & Accessibility**

* **Grid-Coordinate Focus Trap:** Full directional arrow navigation (↑, ↓, ←, →) across time-slots and entity rows.
* **Direct-Key Quick Allocation:** Hitting Enter on an empty slot opens an inline search combobox; typing a shorthand code (e.g., ENG10) and hitting Enter commits the allocation instantly without mouse clicks.
* **Quick-Clone Modifiers:** Holding Alt (or Option) while dragging an existing card duplicates it to the destination slot instead of moving it.
* **One-Key Quick Delete / Clear:** Hitting Backspace or Delete on a focused allocation card removes the assignment (or prompts for confirmation if it impacts downstream records).
* **Keyboard Cut/Paste Reallocation:** Support standard clipboard keys (Ctrl/Cmd + X to cut an assignment card, navigating to another slot, and pressing Ctrl/Cmd + V to paste).

**Collision Detection, Constraints & Validation**

* **Non-Blocking Visual Collision Icons:** Mark conflicting allocations with an amber/red warning badge directly on the card rather than throwing a blocking modal during entry.
* **Conflict Inspection Popover:** Clicking a collision badge reveals the specific rule violation (e.g., *"John Smith is double-booked in Room 102"* or *"Exceeds 4 hours consecutive teaching"*).
* **Direct Conflict-Resolution Link:** Inside the conflict popover, include a one-click action: *"Jump to conflicting slot"* or *"Auto-suggest free alternative staff/room"*.
* **Differentiated Constraint Severities:**
  + *Hard Constraints (Blocking):* Physical impossibilities (e.g., room occupied, staff on approved medical leave).
  + *Soft Constraints (Advisory):* Policy preferences (e.g., uneven shift distribution, back-to-back classes across distant buildings).
* **Proactive Candidate Highlighting:** When selecting an unallocated lesson/shift, highlight all rows (teachers/rooms) that are verified free, qualified, and non-conflicting during that target slot.

**Batch Operations, Templates & Staging**

* **Staging Dock (Unassigned Pool):** A collapsible bottom or right drawer containing unplaced lessons, unassigned shifts, or waitlisted events ready to be dragged onto the grid.
* **"Stamp" / Paint-Bucket Allocation Mode:** Activate a "Stamp" tool holding a specific class or shift configuration, allowing the user to click multiple empty slots in rapid succession to place identical assignments across the week.
* **Period / Week Pattern Replication:** A dedicated toolbar macro allowing users to copy an entire day's layout or full week's rota to a future date range with an automatic conflict check.
* **Draft vs. Published Working State:** Allow entire timetables to be edited in a "Draft" layer (indicated by a persistent header banner) with an explicit "Publish & Notify" workflow that pushes changes live in bulk.
* **Global Clear by Filter:** A utility to clear specific sub-allocations in bulk (e.g., "Clear all Room 101 bookings for Friday" or "Wipe all unconfirmed cover shifts").
