# Official Plugin Lua Runtime Type Stubs (`sdk/lua`)

This directory contains EmmyLua / LuaLS static type annotations for the injected Optimis Lua 5.1 sandbox runtime (`ctx`), implementing Section 9.2 of [`GEMINI.md`](file:///home/kieren/Documents/code/optimis/GEMINI.md).

## Key Files

* [`ctx_types.lua`](file:///home/kieren/Documents/code/optimis/sdk/lua/ctx_types.lua): EmmyLua / LuaLS `@class` and `@field` type definitions covering `ctx.caller`, `ctx.school`, `ctx.dal.*` domain records (`OptimisPupilRecord`, `OptimisSenAndMedicalSummary`, `OptimisMedicalConditionAlert`, `OptimisGuardianRecord` including `authorised_to_collect`, `is_emergency_contact`, `contact_priority`, `OptimisStaffRecord`, `OptimisGroupSummary`, `OptimisTimetableSlot`), `ctx.plugin_db` (`query`, `exec`, `exec_batch`), `ctx.crypto` (`uuid`, `seal_secret`, `open_secret`), `ctx.clock`, `ctx.json`, `ctx.http`, `ctx.events`, and `ctx.xml`.
