-- ============================================================================
-- Optimis Core Schema: 06_safeguarding_vault.sql
-- Domain: KCSIE Child Protection Vault (Envelope Encrypted) & Immutable Audit Log
-- Engine: SQLite 3 (WAL mode, Foreign Keys ON)
-- Mandatory Optimis Security Invariants (GEMINI.md §5B & compliance.md §4):
--   1. Write-Only Postbox for General Staff (`dal.safeguarding.create`).
--   2. Read/Decrypt Restricted Strictly to DSLs (`dal.safeguarding.read`) using
--      Envelope Encryption (Data Encryption Key wrapped by Cloud Key Generator).
--   3. Zero Plugin Exposure: Never exposed on `ctx.dal` to third-party plugins.
--   4. Un-bypassable SQLite Engine Triggers enforcing append-only immutability
--      on `safeguarding_audit_logs` AND blocking accidental deletion of
--      `safeguarding_notes` prior to statutory retention expiry (DOB + 25/35y).
-- ============================================================================

PRAGMA foreign_keys = ON;

-- @domain: safeguarding
-- @dal_scope: dal.safeguarding.create (write-only postbox) | dal.safeguarding.read (DSL only)
-- @dfe_cbds: KCSIE Statutory Child Protection Record (Retained until DOB + 25/35 years)
CREATE TABLE IF NOT EXISTS safeguarding_notes (
    id TEXT PRIMARY KEY,                              -- e.g. 'sfg_...'
    pupil_id TEXT NOT NULL REFERENCES pupils(id) ON DELETE RESTRICT,
    reported_by_staff_id TEXT NOT NULL REFERENCES staff(id) ON DELETE RESTRICT,
    assigned_dsl_staff_id TEXT REFERENCES staff(id) ON DELETE SET NULL,
    concern_category TEXT NOT NULL,                   -- e.g. 'PHYSICAL', 'EMOTIONAL', 'NEGLECT', 'SEXUAL', 'CCE_CSE', 'PREVENT', 'ONLINE_SAFETY', 'PEER_ON_PEER'
    status TEXT NOT NULL DEFAULT 'OPEN_TRIAGE' CHECK (status IN ('OPEN_TRIAGE', 'UNDER_DSL_REVIEW', 'REFERRED_MASH_LADO', 'MONITORING', 'CLOSED')),
    incident_date TEXT NOT NULL,                      -- ISO-8601 Date (YYYY-MM-DD)
    -- ENVELOPE ENCRYPTION PAYLOAD (GEMINI.md §5B):
    kms_key_id TEXT NOT NULL,                         -- Identifier of the DSL Master Key used to wrap the DEK
    wrapped_dek BLOB NOT NULL,                        -- Data Encryption Key (AES-256) encrypted by DSL Vault Master Key
    nonce BLOB NOT NULL,                              -- 12-byte AES-GCM IV/nonce
    ciphertext_payload BLOB NOT NULL,                 -- AES-256-GCM encrypted JSON: { summary, body_map, verbatim_account, action_taken }
    retention_purge_eligible INTEGER NOT NULL DEFAULT 0 CHECK (retention_purge_eligible IN (0, 1)), -- Must be explicitly flipped to 1 by statutory retention saga after DOB+25/35y before deletion
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);
CREATE INDEX IF NOT EXISTS idx_safeguarding_pupil ON safeguarding_notes(pupil_id, created_at);
CREATE INDEX IF NOT EXISTS idx_safeguarding_status ON safeguarding_notes(status, assigned_dsl_staff_id);
CREATE INDEX IF NOT EXISTS idx_safeguarding_reporter ON safeguarding_notes(reported_by_staff_id);
CREATE INDEX IF NOT EXISTS idx_safeguarding_dsl ON safeguarding_notes(assigned_dsl_staff_id);

-- SQLite Engine Trigger blocking deletion of safeguarding_notes unless statutory retention expiry is explicitly unlocked
CREATE TRIGGER IF NOT EXISTS trg_safeguarding_notes_protect_delete
BEFORE DELETE ON safeguarding_notes
WHEN OLD.retention_purge_eligible = 0
BEGIN
    SELECT RAISE(ABORT, 'Security Violation: KCSIE safeguarding records cannot be deleted before statutory retention expiry (DOB + 25/35 years).');
END;

-- @domain: safeguarding
-- @dal_scope: dal.safeguarding.read (DSL / System Auditor)
-- @description: Append-only KCSIE audit trail recording every creation, view, decryption, or export
CREATE TABLE IF NOT EXISTS safeguarding_audit_logs (
    audit_id TEXT PRIMARY KEY,
    note_id TEXT NOT NULL REFERENCES safeguarding_notes(id) ON DELETE RESTRICT,
    pupil_id TEXT NOT NULL REFERENCES pupils(id) ON DELETE RESTRICT,
    actor_person_id TEXT NOT NULL REFERENCES persons(id) ON DELETE RESTRICT,
    action_type TEXT NOT NULL CHECK (action_type IN ('CREATE_POSTBOX', 'DSL_DECRYPT_VIEW', 'DSL_UPDATE_STATUS', 'DSL_EXPORT_CTF_TRANSFER', 'SAR_REVIEW')),
    request_id TEXT NOT NULL,
    client_ip TEXT NOT NULL,
    timestamp TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);
CREATE INDEX IF NOT EXISTS idx_safeguarding_audit_note ON safeguarding_audit_logs(note_id, timestamp);
CREATE INDEX IF NOT EXISTS idx_safeguarding_audit_pupil ON safeguarding_audit_logs(pupil_id, timestamp);
CREATE INDEX IF NOT EXISTS idx_safeguarding_audit_actor ON safeguarding_audit_logs(actor_person_id, timestamp);

-- SQLite Engine Triggers enforcing append-only immutability on safeguarding_audit_logs
CREATE TRIGGER IF NOT EXISTS trg_safeguarding_audit_no_update
BEFORE UPDATE ON safeguarding_audit_logs
BEGIN
    SELECT RAISE(ABORT, 'Security Violation: Statutory safeguarding audit logs are immutable and cannot be updated.');
END;

CREATE TRIGGER IF NOT EXISTS trg_safeguarding_audit_no_delete
BEFORE DELETE ON safeguarding_audit_logs
BEGIN
    SELECT RAISE(ABORT, 'Security Violation: Statutory safeguarding audit logs are immutable and cannot be deleted.');
END;
