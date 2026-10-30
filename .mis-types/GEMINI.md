# LuaLS Type Definitions (`.mis-types`)

This directory contains LuaCATS / EmmyLua type definition stubs pre-bound via [`.vscode/settings.json`](file:///home/kieren/Documents/code/optimis/.vscode/settings.json) (`10B`) so plugin authors get instant IDE autocomplete and inline type-checking for the capability-sandboxed `ctx` and `ctx.dal` surfaces in Lua 5.1.

## Files

* [`dal.d.lua`](file:///home/kieren/Documents/code/optimis/.mis-types/dal.d.lua):
  * Defines `OptimisPluginContext`, `OptimisDAL` (`pupils`, `groups`, `guardians`, `attendance`, `safeguarding`, `sen`, `medical`, `staff`, `census`), `OptimisPluginDB`, `OptimisHTTP`, `OptimisEvents`, `OptimisXML`, `OptimisCrypto`, `OptimisClock`, and `OptimisJSON`.
