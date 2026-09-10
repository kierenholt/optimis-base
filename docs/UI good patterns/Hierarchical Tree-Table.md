**Visual Structure, Indentation & Hierarchy Cues**

* **Strict Leftmost Expander Column:** Restrict the expansion control (caret/chevron ▸/▾) strictly to column 1, left-aligned alongside the entity name, rather than scattering toggles across columns.
* **Proportional Indentation (16–24px Steps):** Indent child rows consistently per depth level. Avoid excessive indentation (>32px) that compresses right-side data on deep trees.
* **Vertical Branch / Guide Lines:** Render subtle, light grey vertical connecting lines (tree-lines) running down from the parent node to each child row to maintain visual lineage across wide viewports.
* **Depth-Based Row Shading:** Apply subtle, progressive background tints or distinct left-border color strips (e.g., Level 0: Pure White/Neutral; Level 1: #F9FAFB; Level 2: #F3F4F6) to prevent child rows from blending into neighboring parent rows.
* **Distinct Node Type Icons:** Prefix rows with unambiguous icons denoting the entity type (e.g., School Folder icon for Departments, Group icon for Class Sets, User icon for Individual Students).
* **Column Alignment Consistency:** Ensure parent and child rows share identical horizontal column widths so numerical, financial, and date values line up vertically regardless of depth.

**Expansion State & Viewport Management**

* **Global "Expand All / Collapse All" Toggle:** Provide a split button or toolbar toggle in the table header to expand or collapse the entire tree to specific depth tiers (e.g., Expand to Level 1, Expand to Level 2, Collapse All).
* **Child Count Pills on Collapsed Nodes:** Display a compact, muted badge adjacent to collapsed parent labels showing total nested descendants (e.g., ▸ Mathematics (14 classes, 320 students)).
* **Single-Branch Auto-Accordion Mode (Optional):** Offer an optional preference that automatically collapses sibling branches when a new branch is expanded to avoid vertical scroll bloat.
* **Sticky Parent Headers on Deep Scroll:** When scrolling vertically through a lengthy expanded branch, pin the parent row to the top of the grid directly beneath the column headers until the last child scrolls past.
* **Expand State Persistence (Session Memory):** Preserve the user's expanded/collapsed node IDs across page refreshes, sorting actions, and route changes using local storage.

**Keyboard Navigation & Accessibility (W3C TreeGrid Spec)**

* **Arrow-Key Expansion Traversal:**
  + Right Arrow on a closed parent expands it; a second Right Arrow jumps focus to the first child row.
  + Left Arrow on an open parent collapses it; Left Arrow on a child moves focus directly up to its immediate parent.
* **Vertical Traversal Across Depths:** Down Arrow and Up Arrow navigate through all visible rows sequentially, respecting currently expanded/collapsed states.
* **Home / End Level Traversal:** Home jumps to the first node at the current hierarchy depth; End jumps to the last node at the current hierarchy depth.
* **Typeahead Find-as-You-Type:** Typing alphanumeric characters moves active focus directly to the next matching node within the current branch without triggering browser search.
* **Accessible ARIA Semantics:** Implement role="treegrid", role="row" with aria-level="[depth]", and aria-expanded="true|false" to ensure assistive technology clearly announces branch depths and hierarchy changes.

**Selection, Bulk Actions & Inheritance**

* **Tri-State Parent Checkboxes:**
  + *Unchecked:* No child records selected.
  + *Indeterminate ([-]):* A subset of child records selected.
  + *Checked ([✓]):* All child records across all nested tiers selected.
* **Cascading Downward Selection:** Ticking a parent checkbox automatically selects all visible and hidden descendants within that sub-tree.
* **Selection Scope Indicators:** When selecting rows across a tree, display an explicit counter in the floating batch toolbar: *"12 items selected (2 Departments, 10 Classes)"*.
* **Bulk Hierarchy Operations:** Provide batch commands for selected nodes, such as *Bulk Reassign Parent*, *Bulk Archive*, or *Bulk Export Subtree*.

**Inline Calculations, Rollups & Overrides**

* **Live Cascading Summary Rollups:** Display automated parent-level aggregates (Sum, Average, Count) calculated in real time from child values (e.g., Department Budget = sum of Class allocations).
* **Inline Calculation Visual Styling:** Distinguish calculated parent cells from editable child cells using subtle italics, a different font weight, or a distinct locked indicator (🔒).
* **Inherited vs. Overridden Badges:** When child rows inherit properties from their parent (e.g., Department Default Grading Scale or Guardian Billing Address), tag them with an "Inherited" indicator; allow users to toggle an override, switching the badge to "Custom" with a "Revert to Parent" action.

**Data Manipulation & Structural Reorganization**

* **Inline "Add Sibling" vs. "Add Child":** Provide dedicated contextual actions on row hover:
  + + Sibling: Inserts an empty editable row at the exact same hierarchy tier.
  + + Child: Automatically expands the parent and inserts an empty editable row nested directly underneath.
* **Drag-and-Drop Reparenting with Drop Zones:**
  + Support dragging rows to restructure the hierarchy.
  + Clearly differentiate drop target states: a horizontal line between rows indicates *Reorder Sibling*; highlighting the entire row indicates *Nest as Child of this Node*.
* **Reparenting Circular Dependency Prevention:** Disable or block drag-and-drop targeting that attempts to move a parent node into its own descendant branch.
* **Unlink vs. Hard Delete Warnings:**
  + Clarify whether deleting a node detaches it from the parent or permanently destroys it.
  + If a parent with nested children is deleted, prompt with an explicit selection: *"Reassign child records to a new parent"* vs. *"Permanently delete parent and all nested children"*.

**Search, Filter & Sparse Matching**

* **Filter Matching with Ancestor Retention:** When filtering (e.g., searching for a student name), keep the parent and grandparent rows visible in the tree (in a slightly dimmed state) so the user does not lose structural context.
* **Auto-Expand on Search:** Automatically expand collapsed branches that contain matching query results, highlighting the matching text strings within child rows.
* **"Flatten Tree" Toggle:** Allow users to toggle between the hierarchical tree view and a flat list view during global searches to sort strictly by attributes (e.g., sort all students by mark regardless of class/department).
