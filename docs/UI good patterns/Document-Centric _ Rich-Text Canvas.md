**Canvas Layout & Document Architecture**

* **Constrained Reading Measure:** Cap the central document body at a readable character width (680–800px or 65–75 characters per line) rather than letting text stretch across wide enterprise displays.
* **Sticky Floating Outline / Mini-Map:** Provide a collapsible table-of-contents rail on the left or right that dynamically highlights the active heading as the user scrolls, with one-click smooth scrolling to sections.
* **Document Status & Sign-Off Banner:** Anchor a persistent top banner indicating legal and operational states (e.g., Draft, Pending Guardian Review, Locked / Legally Binding, Signed off by SENCO).
* **Side-by-Side Reference Pane:** Allow users to split the screen or slide out a reference rail to view supporting records (e.g., past medical assessments, attendance history, incident timestamps) side-by-side while writing.
* **Page-Break / Print-Ready Mode:** For documents requiring formal export (e.g., EHCPs, IEPs, exclusion notices), provide a toggle showing exact PDF page margins and page breaks directly within the editor.

**Embedded Structured Data Tokens ("Smart Chips")**

* **At-Mention Trigger (@):** Typing @ summons an instant lookup combobox to embed live-linked entity tokens (e.g., @John Smith, @Dr. Davis, @Room 102).
* **Slash Commands (/):** Typing / opens a fast block-selection menu for inserting structured elements (e.g., /table, /callout, /goal-tracker, /action-item, /signature-box).
* **Dynamic Database Tokens ({{variable}}):** Support variable merge tags for document templates (e.g., {{student.first\_name}}, {{current\_term.attendance\_pct}}) that resolve live database values into read-only badges.
* **Entity Hover Cards:** Hovering over an embedded smart chip opens a non-blocking popover displaying quick metadata and contact actions without leaving the typing context.
* **Interactive Inline Micro-Widgets:** Embed functional database elements directly inside the prose flow—such as a checklist with functional toggle boxes, an embedded risk-matrix selector, or a measurable goal progress slider.

**Collaborative Review, Suggesting & Commenting**

* **Side-Rail Marginalia (Google Docs Style):** Anchor comments and suggested revisions to the right margin aligned directly with the targeted highlight, preventing comments from obscuring canvas prose.
* **Explicit Mode Switching:** Provide a top-bar mode selector:
  + *Editing Mode:* Direct text mutations.
  + *Suggesting / Track-Changes Mode:* Replaced text displays in red strikethrough; additions display in green underline with individual Accept (✓) and Reject (✗) pill controls.
  + *Viewing Mode:* Read-only inspection with all editing controls hidden.
* **Section-Level Locking:** In multi-disciplinary reports (e.g., Educational Psychologist section + Speech Pathologist section), lock editing access to specific heading blocks based on user role or sign-off status.
* **Threaded Resolvable Comments:** Comments support replies, @mentions with automated email alerts, and an explicit Resolve button that collapses the thread into a review archive.

**Keyboard-Driven Flow & Formatting Toolbars**

* **Contextual Bubble / Floating Toolbar:** Highlight text to summon an immediate floating formatting toolbar (Bold, Italic, Link, Callout, Convert to H2) directly above the selection, minimizing cursor travel to the top ribbon.
* **Markdown Shorthand Auto-Formatting:** Convert plain-text syntax instantly upon spacebar entry:
  + # converts to Heading 1; ## converts to Heading 2.
  + - or \* converts to an unordered bullet list.
  + 1. converts to a numbered ordered list.
  + [] converts to an interactive checklist box.
  + > converts to a styled blockquote or callout block.
* **Block Handle Dragging (⋮⋮):** Display a six-dot hover handle to the left of any paragraph, callout, or table, allowing single-gesture vertical drag-and-drop reordering of whole sections.
* **Standard Shortcut Consistency:** Strict support for universal shortcuts (Ctrl/Cmd + B, Ctrl/Cmd + K for hyperlinks, Ctrl/Cmd + Z/Y, Shift + Enter for soft breaks).

**Revision History, Auditing & Legal Rollbacks**

* **Point-in-Time Snapshot Browser:** Provide a dedicated side-by-side version comparison panel showing timestamped checkpoints alongside who made each edit.
* **Visual Diff Engine:** Highlight changes between versions using standard diff styling (green highlight for added prose, red strikethrough for deleted text).
* **Named Milestone Snapshots:** Allow users to manually label specific versions (e.g., *"Draft submitted to Local Authority 12/10/2026"* or *"Pre-hearing Revision"*), separating them from routine 30-second autosave checkpoints.
* **One-Click Restore & Fork:** A clear header action to *"Restore this version"* or *"Save version as a new template"*, protected by a confirmation prompt.
* **Tamper-Evident Audit Logging:** For sensitive logs (e.g., child safeguarding, behavior incident narratives), record immutable logs of the exact IP, user, and timestamp whenever text is viewed, exported, or edited.

**Template Engine & Guided Writing Assistance**

* **Ghost-Text Scaffold Placeholders:** Empty template sections contain light grey instruction text (e.g., *"Detail specific physical accommodations required in classroom settings..."*) that disappears the moment typing begins.
* **Mandatory Section Validation Guards:** Prevent final submission, signing, or PDF generation if mandatory structured blocks (e.g., Medical Risk Protocol, Review Date) remain empty or below word thresholds.
* **Snippet / Boilerplate Library:** A quick-insert drawer of pre-approved institutional phrases and legal clauses that authors can search and drop directly into the text.
* **Reading Age & Accessibility Scoring:** Subtle, real-time readability indicators in the bottom status bar (e.g., Flesch-Kincaid grade level, screen-reader contrast check) to help staff write accessible reports for parents.
