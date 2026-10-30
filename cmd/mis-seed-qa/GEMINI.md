# QA Synthetic UK School Database Seeder (`cmd/mis-seed-qa`)

This package implements the standalone deterministic PRNG + pinned-anchor QA database generator (`npm run seed` / `go run ./cmd/mis-seed-qa`) that populates a target SQLite database (defaulting to `.data/core.db`) across all 45 tables in [`schema/core/`](file:///home/kieren/Documents/code/optimis/schema/core/SCHEMA_MAP.md).

`npm run seed` does not read from `.env`. When run without arguments, it interactively prompts the user for the target database location. Non-interactive execution is supported via `-db <path>` or a positional path argument.

## Seeded Dataset Profile

* **1,108 Pupils** across 16 UK National Curriculum year groups (`N1`, `N2`, `R`, `Y1`–`Y13`) and 42 registration forms, including pinned test anchors (sibling groups, mid-year set transfers, dual-household custody, court-order blocks, EHCPs, and medical IHCPs).
* **95 Staff** across Leadership, DSLs (`is_dsl = 1`), SENCO, Teachers, Teaching Assistants, and Support Staff with DBS checks and qualifications.
* **889 Households** & guardian relationships with Parental Responsibility (`has_parental_responsibility`) and confidential address flags (`is_confidential_address`).
* **137 Groups** & a 2-week rolling timetable (`timetable_periods`, `timetable_slots`, `timetable_room_exceptions`).
* **44,320 Statutory `AM`/`PM` Session Marks** & lesson attendance across 20 school days.
* **AES-256-GCM Envelope-Encrypted Safeguarding Vault Records** (`safeguarding_notes` & `safeguarding_audit_logs`).

## Key Files

* [`main.go`](file:///home/kieren/Documents/code/optimis/cmd/mis-seed-qa/main.go): CLI entrypoint, schema bootstrap, single-transaction seeding coordinator, and post-seed table/FK verification report.
* [`catalog_and_names.go`](file:///home/kieren/Documents/code/optimis/cmd/mis-seed-qa/catalog_and_names.go): UK name pools, DfE statutory attendance codes, SEN need codes, and curriculum subject catalogs.
* [`seed_domains_00_to_03.go`](file:///home/kieren/Documents/code/optimis/cmd/mis-seed-qa/seed_domains_00_to_03.go): Seeds academic calendar, buildings/rooms, departments, staff, qualifications, and courses.
* [`seed_pupils_and_timetable.go`](file:///home/kieren/Documents/code/optimis/cmd/mis-seed-qa/seed_pupils_and_timetable.go): Seeds 1,108 pupils, enrolments, registration/teaching groups, and 2-week rolling timetable slots.
* [`seed_domains_02_04_05_06.go`](file:///home/kieren/Documents/code/optimis/cmd/mis-seed-qa/seed_domains_02_04_05_06.go): Seeds households, guardians, court orders, statutory `AM`/`PM` & lesson attendance, SEN/Medical/FSM/LAC records, and envelope-encrypted safeguarding notes.
