# Verification & Test Suites (`tests`)

This directory houses the end-to-end verification suites for the Optimis School MIS.

## Subdirectories

* [`tests/e2e/`](file:///home/kieren/Documents/code/optimis/tests/e2e/GEMINI.md): Sequential Playwright E2E test suites (`--workers=1`) structured across 5 domain-aligned subdirectories:
  * [`tests/e2e/core/`](file:///home/kieren/Documents/code/optimis/tests/e2e/core/GEMINI.md): Core SQLite schema invariants, DAL domain scoping, Event Bus, and HTTP Egress engine.
  * [`tests/e2e/platform/`](file:///home/kieren/Documents/code/optimis/tests/e2e/platform/GEMINI.md): Plugin runtime sandbox, attached DBs, lifecycle management, UI SDK, and SDK Toolchain (`10A`–`10D`).
  * [`tests/e2e/security/`](file:///home/kieren/Documents/code/optimis/tests/e2e/security/GEMINI.md): Auth/personas, safeguarding vault envelope encryption, UK GDPR SAR/purge, diagnostics, and break-glass CLI.
  * [`tests/e2e/shell/`](file:///home/kieren/Documents/code/optimis/tests/e2e/shell/GEMINI.md): React Host UI Shell, narrow icon rail, search, command palette (`Ctrl+K`), Keep-Alive LRU, 2D widget grid, and font scaling.
  * [`tests/e2e/verticals/`](file:///home/kieren/Documents/code/optimis/tests/e2e/verticals/GEMINI.md): Full-stack integration suites for all 11 production plugin verticals.
