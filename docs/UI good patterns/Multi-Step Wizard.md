**Stepper Anatomy & Progress Mental Model**

* **Explicit Step Numbering & Labels:** Format steps as Step [N] of [Total]: [Descriptive Name] (e.g., Step 2 of 5: Guardian & Emergency Contacts). Never rely on bare unlabeled dots or unnumbered progress bars.
* **Granular Step State Indicators:** Use distinct visual iconography across four lifecycle states:
  + *Completed:* Filled green circle with a white checkmark (✓).
  + *Active / Current:* Bold primary color ring with active step number.
  + *Upcoming / Unlocked:* Plain neutral ring with muted step number.
  + *Locked / Inaccessible:* Greyed-out lock icon (🔒) for steps gated by prior inputs.
* **Estimated Time / Scope Indicator:** Display a brief helper note at the top of Step 1 (e.g., *"Takes ~4 minutes • Have student NHS/ID number ready"*).
* **Vertical vs. Horizontal Orientation:** Use a vertical left-rail stepper if steps exceed 5 or have long descriptive titles; reserve horizontal top steppers strictly for short (3–5 step) workflows.

**Navigation Freedom & State Boundaries**

* **Free Backward Navigation:** Users must always be able to click previous steps in the stepper or hit the Back button without wiping previously entered data.
* **Strict Forward Gating:** Disable the Next button or prevent forward jumps until all required inputs in the *current* step pass client-side validation.
* **Direct Jump to Completed Steps:** Allow one-click jumping to any previously completed step via the stepper rail without forcing the user to hit Back five times.
* **Linear vs. Branching Visibility:** Dynamically recalculate total steps if answers dictate branching paths (e.g., selecting *"International Student"* inserts a Visa & Passports step; update the stepper from 4 Steps to 5 Steps with an animated entrance).

**Step-Level Validation & Error Containment**

* **Local Step Validation Boundary:** Never validate downstream future steps while the user is on Step 1. Focus validation checks strictly on the active viewport.
* **On-Demand "Next" Error Sweeps:** If a user clicks Next while fields remain invalid, trigger an instant smooth scroll to the first invalid field on the current step and focus it with an inline error message.
* **Persistent Error Badges on Stepper Nodes:** If an earlier step is retroactively invalidated by changes on a later step (e.g., changing birthdate on Step 2 invalidates a consent selection on Step 1), place a red warning icon (⚠) directly onto that step's node in the stepper rail.

**Persistence, Drafts & Session Safety**

* **Per-Step Server/Local Sync:** Automatically persist form state to localStorage or send a background draft payload to the backend upon every successful Next transition.
* **Explicit "Save & Exit" / "Save Draft" Trigger:** Always provide a secondary Save & Finish Later action alongside Next, generating a resume link or session token.
* **Session Resume Prompt:** When returning to an incomplete wizard, show a resume modal: *"You have an in-progress submission from yesterday. [Resume Step 3] or [Start Over]"*.
* **URL Parameter Sync:** Sync the active step to the browser URL (e.g., /enrollment/wizard?step=3) so browser back/forward buttons map to step transitions rather than abandoning the entire wizard.

**Summary, Review & Final Submission**

* **Pre-Flight Summary Screen:** The penultimate step must be a read-only review matrix showing all collected data categorized by step cards.
* **Direct "Edit" Jump-Links:** Include an Edit link on each review section card that jumps the user directly back into that specific step, with a return button labeled *"Return to Review"*.
* **Two-Stage Final Submission Button:** Transform the Next button into an unambiguous final action (e.g., Submit Application or Confirm & Pay). Once clicked, disable the button immediately and render an active loading spinner to prevent duplicate submissions.
* **Confirmation & Post-Completion Canvas:** Replace the form upon submission with an explicit confirmation view displaying:
  + Reference / Tracking Number (with a one-click Copy button).
  + Downloadable PDF summary button.
  + Clear *"What happens next"* timeline listing expected response times and point of contact.

**Keyboard-Driven Flow & Footer Controls**

* **Sticky Footer Action Strip:** Anchor Back, Save Draft, and Next / Continue buttons in a persistent bottom bar so they remain accessible regardless of content height.
* **Visual Primary Hierarchy:** Ensure the Next / Submit button uses strong solid styling, Back uses a low-contrast outline/ghost button, and Cancel / Discard is a plain text link to prevent accidental aborts.
* **Global Shortcut Traversal:** Support Alt/Opt + Right Arrow for Next and Alt/Opt + Left Arrow for Back to enable high-speed data entry without leaving the keyboard.
