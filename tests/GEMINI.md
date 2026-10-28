# Verification & Test Suites (`tests`)

This directory houses the end-to-end verification suites for the Optimis School MIS.

## Subdirectories

* [`tests/e2e`](file:///home/kieren/Documents/code/optimis/tests/e2e/GEMINI.md): Sequential Playwright E2E test suites (`--workers=1`) verifying the React Host UI Shell, Keep-Alive LRU Pool & Editable $N \times M$ Widget Grid (`workspace_lru_and_widget_grid.spec.ts`), Sandboxed `<iframe sandbox="allow-scripts">` Plugin Bridge & UI SDK (`mis-bridge.js` / `mis-theme.css`), Real & Mock `ctx.http` Egress Engine (`http_egress_engine.spec.ts`), Native In-Memory Event Bus & Worker Subscriptions (`event_bus_worker_subscriptions.spec.ts`), Core DAL domain scoping, Backend Session/Persona Resolution & Topbar Persona Switcher (`topbar_persona_switcher.spec.ts`), Safeguarding Vault Envelope Encryption, UK GDPR SAR Redaction & Retention Purge, External Sync & KMS Feature Flags (`external_sync_and_kms_flags.spec.ts`), and DfE Statutory School Census Generation, XSD Validation & COLLECT Submission (`dfe_statutory_school_census.spec.ts`).
