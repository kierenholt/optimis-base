# LuaLS Type Definitions (`.mis-types`)

This directory contains LuaCATS / EmmyLua type definition stubs pre-bound via [`.vscode/settings.json`](file:///home/kieren/Documents/code/optimis/.vscode/settings.json) (`10B`) so plugin authors get instant IDE autocomplete and inline type-checking for the capability-sandboxed `ctx` and `ctx.dal` surfaces in Lua 5.1. Strictly maintained under 500 lines per file.

## Files

* [`dal.d.lua`](file:///home/kieren/Documents/code/optimis/.mis-types/dal.d.lua):
  * Kept byte-identical to [`sdk/lua/ctx_types.lua`](file:///home/kieren/Documents/code/optimis/sdk/lua/ctx_types.lua) (< 500 lines, 218 lines). Defines `OptimisCallerContext`, `OptimisSchoolContext`, `OptimisPluginContext`, `OptimisDAL` (`academic`, `pupils`, `groups`, `guardians`, `attendance`, `safeguarding`, `sen`, `medical`, `staff`, `census`, `dashboard`), `OptimisPluginDB`, `OptimisHTTP`, `OptimisEvents`, `OptimisXML`, `OptimisCrypto`, `OptimisClock`, and `OptimisJSON` across all 27 production plugin stories.
* [`records.d.lua`](file:///home/kieren/Documents/code/optimis/.mis-types/records.d.lua):
  * Kept byte-identical to [`sdk/lua/records.d.lua`](file:///home/kieren/Documents/code/optimis/sdk/lua/records.d.lua) (< 500 lines, 465 lines). Defines entity record structures across all 11 Core DAL domains (pupils, staff, groups, timetable slots, guardians, SEN, medical, attendance codes, and dashboard widget placements).
