# Verification & Test Suites (`tests`)

This directory houses the end-to-end verification suites for the Optimis School MIS.

## Subdirectories

* [`tests/e2e`](file:///home/kieren/Documents/code/optimis/tests/e2e/GEMINI.md): Sequential Playwright E2E test suites (`--workers=1`) verifying the React Host UI Shell, Sandboxed `<iframe sandbox="allow-scripts">` Plugin Bridge & UI SDK (`mis-bridge.js` / `mis-theme.css`), Real & Mock `ctx.http` Egress Engine (`http_egress_engine.spec.ts`), Core DAL domain scoping, Backend Session/Persona Resolution, Safeguarding Vault Envelope Encryption, and UK GDPR SAR Redaction & Retention Purge.
