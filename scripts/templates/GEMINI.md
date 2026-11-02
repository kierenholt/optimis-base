# Plugin Template File Generators (`scripts/templates`)

This directory contains modular template definitions used by [`scripts/populate_plugin_template.mjs`](../populate_plugin_template.mjs) to scaffold standalone Optimis plugin repositories.

## Files

* [`config_templates.mjs`](./config_templates.mjs): Generators for `.vscode/extensions.json`, `package.json`, `.gitignore`, and the cross-platform CLI runner `scripts/run-cli.mjs`.
* [`starter_code.mjs`](./starter_code.mjs): Generators for SQLite schema migrations (`migrations/001_init.sql`), sandboxed Lua 5.1 runtime (`src/init.lua`), sandboxed offline micro-frontend assets (`ui/dist/index.html`, `ui/dist/widget.html`), and headless test suite (`tests/init_test.lua`).
* [`documentation.mjs`](./documentation.mjs): Generators for `README.md`, repository `GEMINI.md`, subfolder `GEMINI.md` files, and `.agents/` guidelines (`AGENTS.md`, `test-author`, `code-implementer`).
