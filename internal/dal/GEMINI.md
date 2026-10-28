# Core Data Access Layer & Identity/Persona Resolution (`internal/dal`)

This package implements the **Optimis Core Data Access Layer (`CoreDAL`)** over `core.db` and attached `plugin_<alias>.db` SQLite databases (Sections 2, 3, 5, 7, and 9.3 of the Architecture Blueprint).

---

## 1. Architectural Invariants Enforced by `CoreDAL`

1. **Separated SQLite WAL Connection Pools ([`dal.go`](file:///home/kieren/Documents/code/optimis/internal/dal/dal.go))**:
   * `WriteDB` (`MaxOpenConns(1)`): Serializes all database writes (`BEGIN IMMEDIATE` / transactions) without `SQLITE_BUSY` contention.
   * `ReadDB` (`MaxOpenConns(NumCPU)`): Serves concurrent WAL reads across CPU cores, using a driver-level [`ConnectHook`](file:///home/kieren/Documents/code/optimis/internal/dal/dal.go#L47) so dynamically attached `plugin_<alias>.db` files and `PRAGMA <alias>.max_page_count` limits stay synchronized across every pooled connection.
2. **Database-Backed Staff Persona & RBAC Scope Resolution ([`persona_auth.go`](file:///home/kieren/Documents/code/optimis/internal/dal/persona_auth.go))**:
   * [`ListActiveStaffPersonas`](file:///home/kieren/Documents/code/optimis/internal/dal/persona_auth.go#L127-L198) and [`ResolveStaffPersonaByID`](file:///home/kieren/Documents/code/optimis/internal/dal/persona_auth.go#L200-L263) resolve staff identities against `staff`, `persons`, temporal `person_roles` (`effective_to IS NULL OR effective_to >= date('now')`), and `roles.default_scopes_json`.
   * **Fail-Fast Guards**: Rejects missing `staff_id`, non-`ACTIVE` `persons.status`, staff with a non-empty `termination_date`, or staff with zero active roles.
   * [`GetSystemConfigValue`](file:///home/kieren/Documents/code/optimis/internal/dal/persona_auth.go#L35-L57) reads required runtime configuration keys (such as `sso_oidc_issuer_url` and `sso_oidc_client_id`) from `__system_config` with zero silent fallbacks.
3. **Temporal (`:own_groups` vs `:all`), Statutory Session & Safeguarding Invariants ([`domains.go`](file:///home/kieren/Documents/code/optimis/internal/dal/domains.go))**:
   * **Row-Level Temporal Filtering**: When a query runs with `ScopeOwnGroups`, SQL results are restricted to groups and pupils where the calling `staff_id` holds an active `group_staff_assignments` row (`effective_from <= reference_date AND (effective_to IS NULL OR effective_to >= reference_date)`).
   * **Guardian PR & Court-Order Blocks**: [`ListGuardiansForPupilScoped`](file:///home/kieren/Documents/code/optimis/internal/dal/domains.go) strictly excludes any guardian with `court_order_restricted = 1`, enforces `has_parental_responsibility = 1` when requested, and redacts address fields whenever `households.is_confidential_address = 1`.
   * **Statutory Attendance (`AM` / `PM`)**: [`UpsertSessionAttendanceBatchScoped`](file:///home/kieren/Documents/code/optimis/internal/dal/domains.go) validates statutory `AM`/`PM` sessions and writes both `attendance_session_marks` and `attendance_register_submissions` audit rows atomically on `WriteDB`.
   * **Safeguarding Envelope Encryption**: [`CreateSafeguardingPostboxNote`](file:///home/kieren/Documents/code/optimis/internal/dal/domains.go) encrypts notes using per-record AES-256-GCM DEKs wrapped under `HOST_KMS_KEY_HANDLE` and records an immutable `CREATE_POSTBOX` audit entry. [`ListSafeguardingNoteEnvelopesDSL`](file:///home/kieren/Documents/code/optimis/internal/dal/domains.go) returns encrypted envelope metadata strictly to `is_dsl = 1` staff without decrypting ciphertext. [`ReadSafeguardingNoteDSL`](file:///home/kieren/Documents/code/optimis/internal/dal/domains.go) verifies `staff.is_dsl = 1` and a valid signed short-lived DSL Cloud Key token ([`IssueDSLCloudKeyToken`](file:///home/kieren/Documents/code/optimis/internal/dal/domains.go)) before unwrapping the DEK, decrypting the payload, and recording an append-only `DSL_DECRYPT_VIEW` audit log entry.
4. **UK GDPR Subject Access Request (SAR) Export & Statutory Retention Purge ([`gdpr.go`](file:///home/kieren/Documents/code/optimis/internal/dal/gdpr.go))**:
   * [`ExportCorePupilSAR`](file:///home/kieren/Documents/code/optimis/internal/dal/gdpr.go#L41): Exports structured `[]SARSection` tables (`title`, `records`, `redactable_fields`) across Core domains (`Core Pupil Profile & Enrolments`, `Core Guardians & Household Contacts` with `court_order_restricted = 0` & confidential refuge redaction, `Core Statutory Session Attendance`, `Core SEN & Medical Records`, and `Core Pastoral Notes`).
   * [`CheckPupilRetentionPurgeEligible`](file:///home/kieren/Documents/code/optimis/internal/dal/gdpr.go#L285) & [`PurgeCorePupilData`](file:///home/kieren/Documents/code/optimis/internal/dal/gdpr.go#L326): Enforces KCSIE safeguarding retention locks (`KCSIE_RETENTION_LOCK` when `safeguarding_notes.retention_purge_eligible = 0` or `safeguarding_audit_logs` exist) before atomically purging all pupil records across `core.db`.

---

## 2. Key Files

* [`dal.go`](file:///home/kieren/Documents/code/optimis/internal/dal/dal.go): SQLite WAL pool initialization, `ConnectHook` `ATTACH`/`DETACH` synchronization, 45-table `schema/core/*.sql` bootstrap, and basic pupil seed helpers.
* [`persona_auth.go`](file:///home/kieren/Documents/code/optimis/internal/dal/persona_auth.go): Staff persona listing, single-staff role/scope resolution, `__system_config` lookups, and isolated persona test fixture seeding/cleanup.
* [`domains.go`](file:///home/kieren/Documents/code/optimis/internal/dal/domains.go): Coarse-grained scoped Core DAL methods across `pupils`, `sen`, `medical`, `groups`, `attendance`, `guardians`, and `safeguarding`.
* [`gdpr.go`](file:///home/kieren/Documents/code/optimis/internal/dal/gdpr.go): Core UK GDPR SAR export section builder, KCSIE retention pre-flight check, atomic Core pupil retention purge, and isolated GDPR test fixture seeding/teardown.
* [`core_schema_functional_test.go`](file:///home/kieren/Documents/code/optimis/internal/dal/core_schema_functional_test.go) & [`core_dal_domains_test.go`](file:///home/kieren/Documents/code/optimis/internal/dal/core_dal_domains_test.go): Isolated SQLite WAL unit and functional tests covering all 45 core tables and domain scoping invariants.
