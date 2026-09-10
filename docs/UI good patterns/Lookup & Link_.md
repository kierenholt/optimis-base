**Input Anatomy & Asynchronous Combobox**

* **Instant Typeahead with Debounce:** Trigger search queries after a 250–300ms pause in typing; show an inline loading spinner inside the right edge of the input while fetching matches from the directory.
* **Minimum Character Threshold:** Show a clear helper hint when empty (e.g., Type at least 2 characters to search directory...) rather than querying a database of tens of thousands of records on a single keystroke.
* **Tokenized Pill Formatting:** Once linked, convert the input text into a high-contrast entity pill containing:
  + Avatar or entity icon
  + Primary record label (e.g., Taylor, Liam (Y10))
  + Quick-action context menu (...)
  + Detach/Remove button (✕)
* **Multi-Attribute Result Display:** Do not display bare names alone in the suggestion menu. Format each dropdown result as a structured mini-card displaying disambiguating attributes:
  + *Top Line:* Primary Name (e.g., Sarah Jenkins)
  + *Bottom Line (Muted):* Disambiguators (e.g., DOB: 14/05/2011 • ID: #8402 • Form 8B) to prevent linking the wrong homonymous individual.

**Directory Search, Fallbacks & "Create New" Flow**

* **Pinned "Create New" Menu Action:** If the entity does not exist in the database, pin a persistent action to the bottom of the active search dropdown: + Create "[typed query]" as new Contact.
* **In-Context Creation Drawer/Modal:** Clicking Create New slides open a sub-drawer pre-populated with whatever text the user already typed into the combobox, allowing instant creation without navigating away or losing context.
* **Zero-Results Clarity & Broadening Tips:** If a search yields zero matches, provide explicit diagnostic guidance: *"No contacts match 'Jonathon'. Try searching by Student ID, Phone Number, or check spelling."*
* **Recent & Frequent Entities Section:** When focusing an empty lookup field, display a dropdown list of the user's 5 most recently accessed or frequently linked entities before any keystroke is entered.

**Relationship Semantics & Edge Attributes**

* **Relationship Role Dropdown:** When linking two entities (e.g., linking an adult to a student), pair the lookup combobox directly with a relationship selector:
  + Entity Lookup: [ Jane Doe ]
  + Relationship Selector: [ Mother ▾ ]
* **Contextual Edge Checkboxes (Link Properties):** Surface relationship-specific permissions immediately alongside the linked token rather than burying them in secondary screens:
  + [✓] Primary Contact
  + [✓] Parental Responsibility
  + [✓] Authorized for Pickup
  + [ ] Emergency Only
* **Bidirectional Linkage Clarity:** Explicitly indicate inverse relationships (e.g., linking Jane Doe as *Mother* to Liam Taylor automatically previews: *"Liam Taylor will be marked as 'Son' on Jane Doe's profile"*).

**Disambiguation & Duplicate Prevention**

* **Duplicate Detection Warning:** If the user attempts to link an entity that shares identifiers (e.g., matching address or phone) with an existing profile, render an advisory banner: *"Potential Duplicate: A record with this email already exists for 'Dr. J. Doe'. [Link Existing Instead]"*.
* **Highlight Matching Substrings:** Bold the exact matched characters within the dropdown list options so users immediately understand why a particular record appeared (e.g., searching 7700 bolds the matched digits in Phone: +44 \*\*7700\*\* 900123).
* **Active Conflict Gating:** If business rules forbid linking (e.g., linking a teacher to a class scheduled during an existing teaching commitment), show the result in a disabled, tinted-red state with an inline explanation: Unavailable: Conflicting schedule in Rm 102.

**Keyboard Navigation & Focus Management**

* **Arrow-Key Menu Traversal:** Pressing Down Arrow immediately shifts active focus from the text input into the dropdown suggestion list; Up Arrow moves backwards.
* **Selection Hotkeys:** Pressing Enter or Tab selects the currently focused suggestion item, creates the relationship pill, and places the cursor into the relationship role selector.
* **Backspace-to-Edit Pill:** When the cursor is placed immediately after an active entity pill, pressing Backspace selects the pill; pressing Backspace a second time detaches the entity and restores the editable text input.
* **Escape to Dismiss:** Pressing Esc immediately dismisses the suggestion flyout without modifying or removing previously selected items.

**Detachment, Unlinking & Safety Guards**

* **Explicit Unlink vs. Delete Terminology:** Use clear language: label detachment buttons Unlink from Class or Remove from Family, never use a bare red trashcan labeled Delete (which implies deleting the person from the database).
* **Cascading Relationship Warnings:** If detaching an entity breaks a core operational requirement (e.g., removing a student's sole primary guardian), block the action with an alert modal: *"Cannot unlink Jane Doe: Liam Taylor must have at least one contact with Parental Responsibility. [Assign Replacement First]"*.
* **Reversible Unlink Toast:** When a record is unlinked, render a temporary toast notification (Unlinked Jane Doe from Student. [Undo]) allowing immediate recovery of the link relationship.
