# Optimis School MIS

UK All-Through (Primary, Secondary, K–12) open-source Management Information System built on an in-process **Go Host Container**, **SQLite WAL** (`core.db` + per-plugin `plugin_<alias>.db`), capability-sandboxed **Pure-Go Lua 5.1 (`gopher-lua`)** plugins, and an embedded **React Host UI Shell** (`<iframe sandbox="allow-scripts">` micro-frontends).

## Architecture & Documentation

* **[`GEMINI.md`](file:///home/kieren/Documents/code/optimis/GEMINI.md):** Master Architecture Blueprint (Tenancy, Core vs. Plugin Boundary, RBAC, Envelope-Encrypted DSL Safeguarding Vault, UK GDPR SAR & Retention Purge, Sandboxed UI & RPC Bridge, and Module Index).
* **[`schema/core/SCHEMA_MAP.md`](file:///home/kieren/Documents/code/optimis/schema/core/SCHEMA_MAP.md):** 45-table `core.db` SQLite schema map across 7 domain SQL files (`00`–`06`).
* **[`.agents/AGENTS.md`](file:///home/kieren/Documents/code/optimis/.agents/AGENTS.md):** Repository rules and available `npm` build, seed, and test commands.
