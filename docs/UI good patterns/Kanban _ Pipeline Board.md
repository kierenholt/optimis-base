**Column Header & Capacity Management**

* **Persistent Counter Badges:** Display total card counts prominently in every column header (e.g., Applied (12), Under Review (4)).
* **Work-In-Progress (WIP) Limits with Visual Warnings:** Allow administrators to set minimum and maximum capacity caps per column. When a column exceeds its limit (e.g., 8/5 Reviewers), highlight the header counter in amber/red without completely locking the board.
* **Aggregated Financial / Metric Rollups:** For pipeline workflows (e.g., tuition billing, admissions grants), show dynamic currency or value sums in the column header alongside the card count.
* **Collapsible Columns:** Include an icon or toggle to collapse low-priority, empty, or terminal columns (e.g., Rejected, Enrolled - Archived) into a narrow vertical spine to free up horizontal screen space.
* **Sticky Column Headers:** Lock column headers to the top of the viewport when vertical scrolling occurs inside deep columns.

**Card Anatomy & Density Control**

* **Three-Tier Typographic Hierarchy:**
  + *Top:* Identification tag or ID code (e.g., #APP-1042) and priority indicator.
  + *Middle:* Primary subject/entity title (bold, 14–15px).
  + *Bottom:* Assignee avatar, due date badge, and nested item counts.
* **Left-Border Category Color Strips:** Use a 3–4px solid color accent on the card’s left edge to represent entity categories (e.g., Year Group, Urgency Level, Funding Source) rather than flooding the entire card background.
* **SLA & Due Date Badging:** Display date badges that shift colors dynamically based on urgency: neutral for future dates, amber for due within 24 hours, and solid red for overdue items.
* **Micro-Progress Indicators:** For cards with subtasks or checklists, display a compact progress pill (e.g., 3/5 steps completed) or a 2px micro-progress bar across the card bottom.
* **Density Mode Toggles:** Provide "Compact" (title + avatar only) and "Expanded" (full tags, checklist summaries, custom fields) view settings for different operator preferences.

**Direct Manipulation, Drag-and-Drop & Ghosting**

* **Translucent Ghost Drag Preview:** Render a semi-transparent, tilted ghost card beneath the user’s cursor during drag, keeping destination columns legible.
* **Magnetic Drop Zones & Insertion Lines:** As a card moves between other cards in a column, render an animated horizontal placeholder line (2–3px colored bar) indicating the precise placement slot.
* **Auto-Scrolling on Column Edges:** Hovering a dragged card near the top/bottom of a column or near the left/right edges of the board automatically scrolls the viewport smoothly.
* **Pre-Drop Target Validation:** If business logic forbids moving a card directly into a specific stage (e.g., skipping from Intake straight to Enrolled), tint the invalid column red and show a "No Entry" cursor state before drop release.
* **Transition Reason & Validation Prompts:** If moving a card to a stage requires metadata (e.g., dropping into Rejected requires a rejection reason, or Accepted requires assigning a start date), drop the card immediately and pop a non-blocking modal requiring the specific mandatory fields.

**Keyboard-Driven Pipeline Management & Accessibility**

* **Grid Coordinate Navigation:** Use standard arrow keys (←, → between columns; ↑, ↓ between cards in the active column).
* **Keyboard-Driven Stage Moves:** Support dedicated hotkeys (e.g., [ and ] or Ctrl/Cmd + Arrow Keys) to move the focused card left or right between pipeline columns without using a mouse.
* **One-Key Quick Actions:** Pressing E opens quick-edit fields, Space opens preview drawers, A opens the assignee picker, and Delete/Backspace prompts for card archiving.
* **Focus Restoration:** After moving a card with the keyboard, retain active keyboard focus on that same card in its new position rather than resetting focus to the top of the board.

**Inline Creation, Quick Editing & Drawers**

* **Persistent Top/Bottom "Quick Add":** Place a sticky + Add Card button at the top or bottom of every column that reveals a single-input inline box for typing a title and hitting Enter to create records rapidly.
* **Hover-Activated Ghost Controls:** Show quick-utility icons (e.g., edit pencil, context menu ..., archive icon) only when the card is hovered or focused to eliminate visual clutter.
* **Side-Drawer Detail Inspection:** Clicking a card should slide open a peek drawer (master-detail pattern) over the right side of the board to view full history, comments, and file attachments without navigating away from the pipeline.
* **Inline Status Dropdown Fallback:** Inside the card itself, provide a quick status dropdown so users can reassign stages via a single click menu instead of dragging across wide monitors.

**Grouping, Swimlanes & Filtering**

* **Horizontal Swimlanes:** Allow grouping cards horizontally across the vertical columns by secondary dimensions (e.g., Swimlanes by Assignee, by Priority Tier, or by Department).
* **Collapsible Swimlanes with Independent Counts:** Allow users to collapse individual swimlane rows with a single caret click, displaying sub-aggregates per lane.
* **Interactive Quick-Filter Ribbons:** Provide one-click filter pills above the board (e.g., Assigned to Me, Overdue Only, Unassigned, High Priority) that filter the cards in place with zero reload delay.
* **Assignee Face-Pile Filtering:** Display an avatar row of all team members in the top toolbar; clicking an avatar filters the board strictly to cards owned by that person.

**Batch & Pipeline Operations**

* **Multi-Card Dragging:** Allow Shift + Click or checkbox selection of multiple cards within a column to drag the entire batch into a new stage simultaneously.
* **Column-Level Bulk Actions:** Include a contextual ... menu on column headers with macros such as: *Mark all cards as Complete*, *Reassign all cards in this stage*, or *Export column to CSV*.
* **Automated Stage Rules Indicators:** Show a tiny lightning-bolt icon on columns that have background automations configured (e.g., auto-sending an email when a card lands in Awaiting Documents).
