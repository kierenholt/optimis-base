# Official Plugin Lua Runtime Type Stubs (`sdk/lua`)

This directory contains EmmyLua / LuaLS static type annotations for the injected Optimis Lua 5.1 sandbox runtime (`ctx`), implementing Section 9.2 of [`GEMINI.md`](file:///home/kieren/Documents/code/optimis/GEMINI.md). Strictly maintained under 500 lines per file.

## Key Files

* [`ctx_types.lua`](file:///home/kieren/Documents/code/optimis/sdk/lua/ctx_types.lua): EmmyLua / LuaLS `@class` and `@field` type definitions (synchronized with [`.mis-types/dal.d.lua`](file:///home/kieren/Documents/code/optimis/.mis-types/dal.d.lua), < 500 lines, 218 lines) covering `ctx.caller`, `ctx.school`, `ctx.dal.*` across all 11 Core DAL domains (`academic`, `pupils`, `groups`, `staff`, `guardians`, `attendance`, `safeguarding`, `sen`, `medical`, `census`, `dashboard`), `ctx.plugin_db` (`query`, `exec`, `exec_batch`), `ctx.crypto` (`uuid`, `seal_secret`, `open_secret`), `ctx.clock`, `ctx.json`, `ctx.http`, `ctx.events`, and `ctx.xml`.
* [`records.d.lua`](file:///home/kieren/Documents/code/optimis/sdk/lua/records.d.lua): EmmyLua / LuaLS `@class` and `@field` entity record structures (synchronized with [`.mis-types/records.d.lua`](file:///home/kieren/Documents/code/optimis/.mis-types/records.d.lua), < 500 lines, 465 lines) defining all DAL domain return and parameter tables.
