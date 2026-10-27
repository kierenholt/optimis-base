function get_pupil_record(ctx, args)
  if type(ctx) ~= "table" then
    error("missing mandatory ctx table")
  end
  if type(ctx.caller) ~= "table" or type(ctx.dal) ~= "table" or type(ctx.plugin_db) ~= "table" or type(ctx.events) ~= "table" or type(ctx.http) ~= "table" then
    error("missing one or more mandatory ctx namespaces (caller, dal, plugin_db, events, http)")
  end

  if type(args) ~= "table" or type(args.pupil_id) ~= "string" or args.pupil_id == "" then
    error("missing mandatory parameter: pupil_id")
  end

  local pupil = ctx.dal.pupils.get_by_id(args.pupil_id)
  if not pupil or not pupil.id then
    error("pupil record not found")
  end

  ctx.events.emit("test.pupil_fetched", {
    pupil_id = pupil.id
  })

  return {
    ok = true,
    data = {
      id = pupil.id,
      upn = pupil.upn,
      first_name = pupil.first_name,
      last_name = pupil.last_name,
      academic_year_id = pupil.academic_year_id,
      effective_from = pupil.effective_from,
      ctx_modules_verified = true
    }
  }
end
