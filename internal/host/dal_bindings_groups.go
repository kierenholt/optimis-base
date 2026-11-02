package host

import (
	"context"
	"strings"

	lua "github.com/yuin/gopher-lua"

	"optimis/internal/dal"
)

func timetableSlotToLuaTable(L *lua.LState, s dal.TimetableSlotSummary) *lua.LTable {
	sTbl := L.NewTable()
	L.SetField(sTbl, "slot_id", lua.LString(s.SlotID))
	L.SetField(sTbl, "period_id", lua.LString(s.PeriodID))
	L.SetField(sTbl, "period_code", lua.LString(s.PeriodCode))
	L.SetField(sTbl, "period_name", lua.LString(s.PeriodName))
	L.SetField(sTbl, "cycle_week", lua.LNumber(s.CycleWeek))
	L.SetField(sTbl, "day_of_week_id", lua.LNumber(s.DayOfWeekID))
	L.SetField(sTbl, "start_time", lua.LString(s.StartTime))
	L.SetField(sTbl, "end_time", lua.LString(s.EndTime))
	L.SetField(sTbl, "period_type", lua.LString(s.PeriodType))
	L.SetField(sTbl, "maps_to_statutory_session", lua.LString(s.MapsToStatutorySession))
	L.SetField(sTbl, "group_id", lua.LString(s.GroupID))
	L.SetField(sTbl, "group_code", lua.LString(s.GroupCode))
	L.SetField(sTbl, "staff_id", lua.LString(s.StaffID))
	L.SetField(sTbl, "room_id", lua.LString(s.RoomID))
	return sTbl
}

func extractGroupIDAndOptsFromLua(L *lua.LState) (string, *lua.LTable) {
	if firstTbl, isTbl := L.Get(1).(*lua.LTable); isTbl {
		var groupID string
		if v := firstTbl.RawGetString("group_id"); v != lua.LNil {
			groupID = strings.TrimSpace(v.String())
		} else if v := firstTbl.RawGetString("id"); v != lua.LNil {
			groupID = strings.TrimSpace(v.String())
		}
		return groupID, firstTbl
	}
	groupID := strings.TrimSpace(L.OptString(1, ""))
	var optsTbl *lua.LTable
	if L.GetTop() >= 2 {
		optsTbl, _ = L.Get(2).(*lua.LTable)
	}
	return groupID, optsTbl
}

func (h *Host) bindLuaDALGroups(
	L *lua.LState,
	execCtx context.Context,
	endpointSpec EndpointSpec,
	caller CallerContext,
	dalTbl *lua.LTable,
) {
	hasGroupsRead := hasDomainScope(endpointSpec.Scopes, "dal.groups.read")
	hasGroupsWrite := hasDomainScope(endpointSpec.Scopes, "dal.groups.write")
	if !hasGroupsRead && !hasGroupsWrite {
		return
	}

	groupsTbl := L.NewTable()

	callerGroupGuardMode := dal.ScopeOwnGroups
	if resolveEffectiveDomainScopeMode(endpointSpec.Scopes, caller, "dal.groups.read") == dal.ScopeAll ||
		resolveEffectiveDomainScopeMode(endpointSpec.Scopes, caller, "dal.groups.write") == dal.ScopeAll ||
		resolveEffectiveDomainScopeMode(endpointSpec.Scopes, caller, "dal.pupils.read") == dal.ScopeAll {
		callerGroupGuardMode = dal.ScopeAll
	}

	assertAssignedFn := L.NewFunction(func(L *lua.LState) int {
		groupID, optsTbl := extractGroupIDAndOptsFromLua(L)
		if groupID == "" {
			L.RaiseError("missing mandatory parameter: group_id")
			return 0
		}
		scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, callerGroupGuardMode, caller.StaffID, optsTbl)
		if sErr != nil {
			L.RaiseError("dal.groups.assert_caller_assigned scope error: %s", sErr.Error())
			return 0
		}
		if err := h.coreDAL.EnsureCallerAssignedToGroupScoped(execCtx, scope, groupID); err != nil {
			L.RaiseError("%s", err.Error())
			return 0
		}
		L.Push(lua.LTrue)
		return 1
	})
	L.SetField(groupsTbl, "assert_caller_assigned", assertAssignedFn)
	L.SetField(groupsTbl, "can_mutate_group", assertAssignedFn)

	L.SetField(groupsTbl, "is_caller_assigned", L.NewFunction(func(L *lua.LState) int {
		groupID, optsTbl := extractGroupIDAndOptsFromLua(L)
		if groupID == "" {
			L.RaiseError("missing mandatory parameter: group_id")
			return 0
		}
		scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, callerGroupGuardMode, caller.StaffID, optsTbl)
		if sErr != nil {
			L.RaiseError("dal.groups.is_caller_assigned scope error: %s", sErr.Error())
			return 0
		}
		assigned, err := h.coreDAL.IsCallerAssignedToGroupScoped(execCtx, scope, groupID)
		if err != nil {
			L.RaiseError("dal.groups.is_caller_assigned failed: %s", err.Error())
			return 0
		}
		L.Push(lua.LBool(assigned))
		return 1
	}))

	if hasGroupsRead {
		groupsMode := resolveEffectiveDomainScopeMode(endpointSpec.Scopes, caller, "dal.groups.read")
		listGroupsWithMode := func(forcedMode *dal.ScopeMode) *lua.LFunction {
			return L.NewFunction(func(L *lua.LState) int {
				var optsTbl *lua.LTable
				if L.GetTop() >= 1 {
					optsTbl, _ = L.Get(1).(*lua.LTable)
				}
				var groupType string
				effectiveMode := groupsMode
				if forcedMode != nil {
					effectiveMode = *forcedMode
				}
				if optsTbl != nil {
					if gt := optsTbl.RawGetString("group_type"); gt != lua.LNil {
						groupType = strings.TrimSpace(gt.String())
					}
					if forcedMode == nil && (optsTbl.RawGetString("include_all") == lua.LTrue || optsTbl.RawGetString("scope").String() == "all") {
						effectiveMode = dal.ScopeAll
					}
				}
				scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, effectiveMode, caller.StaffID, optsTbl)
				if sErr != nil {
					L.RaiseError("dal.groups.list_groups scope error: %s", sErr.Error())
					return 0
				}
				groups, err := h.coreDAL.ListGroupsScoped(execCtx, scope, groupType)
				if err != nil {
					L.RaiseError("dal.groups.list_groups failed: %s", err.Error())
					return 0
				}
				arrTbl := L.NewTable()
				markLuaTableAsArray(L, arrTbl)
				for i, g := range groups {
					gTbl := L.NewTable()
					L.SetField(gTbl, "id", lua.LString(g.ID))
					L.SetField(gTbl, "academic_year_id", lua.LString(g.AcademicYearID))
					L.SetField(gTbl, "group_type", lua.LString(g.GroupType))
					L.SetField(gTbl, "code", lua.LString(g.Code))
					L.SetField(gTbl, "name", lua.LString(g.Name))
					L.SetField(gTbl, "default_room_id", lua.LString(g.DefaultRoomID))
					L.SetField(gTbl, "takes_attendance", lua.LBool(g.TakesAttendance))
					L.SetField(gTbl, "effective_from", lua.LString(g.EffectiveFrom))
					arrTbl.RawSetInt(i+1, gTbl)
				}
				L.Push(arrTbl)
				return 1
			})
		}

		ownMode := dal.ScopeOwnGroups
		allMode := dal.ScopeAll
		L.SetField(groupsTbl, "list_groups", listGroupsWithMode(nil))
		L.SetField(groupsTbl, "list_my_groups", listGroupsWithMode(&ownMode))
		L.SetField(groupsTbl, "list_all_groups", listGroupsWithMode(&allMode))

		L.SetField(groupsTbl, "list_group_pupils", L.NewFunction(func(L *lua.LState) int {
			groupID, optsTbl := extractGroupIDAndOptsFromLua(L)
			if groupID == "" {
				L.RaiseError("missing mandatory parameter: group_id")
				return 0
			}
			scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, dal.ScopeAll, caller.StaffID, optsTbl)
			if sErr != nil {
				L.RaiseError("dal.groups.list_group_pupils scope error: %s", sErr.Error())
				return 0
			}
			pupils, err := h.coreDAL.ListPupilsByGroupScoped(execCtx, scope, groupID)
			if err != nil {
				L.RaiseError("dal.groups.list_group_pupils failed: %s", err.Error())
				return 0
			}
			arrTbl := L.NewTable()
			markLuaTableAsArray(L, arrTbl)
			for i, p := range pupils {
				arrTbl.RawSetInt(i+1, pupilToLuaTable(L, p))
			}
			L.Push(arrTbl)
			return 1
		}))

		L.SetField(groupsTbl, "list_staff", L.NewFunction(func(L *lua.LState) int {
			staffList, err := h.coreDAL.ListActiveStaff(execCtx)
			if err != nil {
				L.RaiseError("dal.groups.list_staff failed: %s", err.Error())
				return 0
			}
			arrTbl := L.NewTable()
			markLuaTableAsArray(L, arrTbl)
			for i, st := range staffList {
				arrTbl.RawSetInt(i+1, staffToLuaTable(L, st))
			}
			L.Push(arrTbl)
			return 1
		}))

		L.SetField(groupsTbl, "get_staff_timetable_slots", L.NewFunction(func(L *lua.LState) int {
			staffID := strings.TrimSpace(L.OptString(1, ""))
			if staffID == "" {
				staffID = caller.StaffID
			}
			cycleWeek := L.OptInt(2, 1)
			dayOfWeekID := L.OptInt(3, 1)
			var optsTbl *lua.LTable
			if L.GetTop() >= 4 {
				optsTbl, _ = L.Get(4).(*lua.LTable)
			}
			effectiveMode := dal.ScopeAll
			if optsTbl != nil && optsTbl.RawGetString("strict_own_groups") == lua.LTrue {
				effectiveMode = groupsMode
			}
			scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, effectiveMode, caller.StaffID, optsTbl)
			if sErr != nil {
				L.RaiseError("dal.groups.get_staff_timetable_slots scope error: %s", sErr.Error())
				return 0
			}
			slots, err := h.coreDAL.GetStaffTimetableSlotsScoped(execCtx, scope, staffID, cycleWeek, dayOfWeekID)
			if err != nil {
				L.RaiseError("dal.groups.get_staff_timetable_slots failed: %s", err.Error())
				return 0
			}
			arrTbl := L.NewTable()
			markLuaTableAsArray(L, arrTbl)
			for i, s := range slots {
				arrTbl.RawSetInt(i+1, timetableSlotToLuaTable(L, s))
			}
			L.Push(arrTbl)
			return 1
		}))

		L.SetField(groupsTbl, "get_staff_timetable_slots_batch", L.NewFunction(func(L *lua.LState) int {
			idsTbl, ok := L.Get(1).(*lua.LTable)
			if !ok {
				L.RaiseError("dal.groups.get_staff_timetable_slots_batch requires a table of staff_ids")
				return 0
			}
			var staffIDs []string
			for i := 1; i <= idsTbl.MaxN(); i++ {
				if val := idsTbl.RawGetInt(i); val != lua.LNil {
					staffIDs = append(staffIDs, strings.TrimSpace(val.String()))
				}
			}
			cycleWeek := L.OptInt(2, 1)
			dayOfWeekID := L.OptInt(3, 1)
			var optsTbl *lua.LTable
			if L.GetTop() >= 4 {
				optsTbl, _ = L.Get(4).(*lua.LTable)
			}
			effectiveMode := dal.ScopeAll
			if optsTbl != nil && optsTbl.RawGetString("strict_own_groups") == lua.LTrue {
				effectiveMode = groupsMode
			}
			scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, effectiveMode, caller.StaffID, optsTbl)
			if sErr != nil {
				L.RaiseError("dal.groups.get_staff_timetable_slots_batch scope error: %s", sErr.Error())
				return 0
			}
			batchSlots, err := h.coreDAL.GetStaffTimetableSlotsBatchScoped(execCtx, scope, staffIDs, cycleWeek, dayOfWeekID)
			if err != nil {
				L.RaiseError("dal.groups.get_staff_timetable_slots_batch failed: %s", err.Error())
				return 0
			}
			resTbl := L.NewTable()
			for sid, slots := range batchSlots {
				arrTbl := L.NewTable()
				markLuaTableAsArray(L, arrTbl)
				for i, s := range slots {
					arrTbl.RawSetInt(i+1, timetableSlotToLuaTable(L, s))
				}
				L.SetField(resTbl, sid, arrTbl)
			}
			L.Push(resTbl)
			return 1
		}))

		L.SetField(groupsTbl, "list_free_staff_for_period", L.NewFunction(func(L *lua.LState) int {
			var cycleWeek int
			var dayOfWeekID int
			var periodIdentifier string
			var optsTbl *lua.LTable
			if firstTbl, isTbl := L.Get(1).(*lua.LTable); isTbl {
				if v := firstTbl.RawGetString("cycle_week"); v != lua.LNil {
					if n, ok := v.(lua.LNumber); ok {
						cycleWeek = int(n)
					}
				}
				if v := firstTbl.RawGetString("day_of_week_id"); v != lua.LNil {
					if n, ok := v.(lua.LNumber); ok {
						dayOfWeekID = int(n)
					}
				}
				if v := firstTbl.RawGetString("period_id"); v != lua.LNil && strings.TrimSpace(v.String()) != "" {
					periodIdentifier = strings.TrimSpace(v.String())
				} else if v := firstTbl.RawGetString("period_code"); v != lua.LNil {
					periodIdentifier = strings.TrimSpace(v.String())
				}
				optsTbl = firstTbl
			} else {
				cycleWeek = L.OptInt(1, 1)
				dayOfWeekID = L.OptInt(2, 1)
				periodIdentifier = strings.TrimSpace(L.OptString(3, ""))
				if L.GetTop() >= 4 {
					optsTbl, _ = L.Get(4).(*lua.LTable)
				}
			}
			if cycleWeek == 0 {
				cycleWeek = 1
			}
			if dayOfWeekID == 0 {
				dayOfWeekID = 1
			}
			scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, dal.ScopeAll, caller.StaffID, optsTbl)
			if sErr != nil {
				L.RaiseError("dal.groups.list_free_staff_for_period scope error: %s", sErr.Error())
				return 0
			}
			freeStaff, err := h.coreDAL.ListFreeStaffForPeriodScoped(execCtx, scope, cycleWeek, dayOfWeekID, periodIdentifier)
			if err != nil {
				L.RaiseError("dal.groups.list_free_staff_for_period failed: %s", err.Error())
				return 0
			}
			arrTbl := L.NewTable()
			markLuaTableAsArray(L, arrTbl)
			for i, st := range freeStaff {
				arrTbl.RawSetInt(i+1, staffToLuaTable(L, st))
			}
			L.Push(arrTbl)
			return 1
		}))

		L.SetField(groupsTbl, "list_rooms", L.NewFunction(func(L *lua.LState) int {
			rooms, err := h.coreDAL.ListRooms(execCtx)
			if err != nil {
				L.RaiseError("dal.groups.list_rooms failed: %s", err.Error())
				return 0
			}
			arrTbl := L.NewTable()
			markLuaTableAsArray(L, arrTbl)
			for i, r := range rooms {
				rTbl := L.NewTable()
				L.SetField(rTbl, "id", lua.LString(r.ID))
				L.SetField(rTbl, "room_id", lua.LString(r.ID))
				L.SetField(rTbl, "code", lua.LString(r.Code))
				L.SetField(rTbl, "name", lua.LString(r.Name))
				L.SetField(rTbl, "room_type", lua.LString(r.RoomType))
				L.SetField(rTbl, "capacity", lua.LNumber(r.Capacity))
				L.SetField(rTbl, "is_active", lua.LBool(r.IsActive))
				L.SetField(rTbl, "phone_extension", lua.LString(r.PhoneExtension))
				arrTbl.RawSetInt(i+1, rTbl)
			}
			L.Push(arrTbl)
			return 1
		}))

		L.SetField(groupsTbl, "list_courses", L.NewFunction(func(L *lua.LState) int {
			var ayID string
			if L.GetTop() >= 1 {
				if optsTbl, isTbl := L.Get(1).(*lua.LTable); isTbl {
					ayID = luaTableTrimmedString(optsTbl, "academic_year_id")
				} else {
					ayID = strings.TrimSpace(L.OptString(1, ""))
				}
			}
			courses, err := h.coreDAL.ListCourses(execCtx, ayID)
			if err != nil {
				L.RaiseError("dal.groups.list_courses failed: %s", err.Error())
				return 0
			}
			arrTbl := L.NewTable()
			markLuaTableAsArray(L, arrTbl)
			for i, c := range courses {
				cTbl := L.NewTable()
				L.SetField(cTbl, "id", lua.LString(c.ID))
				L.SetField(cTbl, "course_id", lua.LString(c.ID))
				L.SetField(cTbl, "academic_year_id", lua.LString(c.AcademicYearID))
				L.SetField(cTbl, "department_id", lua.LString(c.DepartmentID))
				L.SetField(cTbl, "code", lua.LString(c.Code))
				L.SetField(cTbl, "name", lua.LString(c.Name))
				L.SetField(cTbl, "dfe_subject_code", lua.LString(c.DfESubjectCode))
				arrTbl.RawSetInt(i+1, cTbl)
			}
			L.Push(arrTbl)
			return 1
		}))

		L.SetField(groupsTbl, "list_departments", L.NewFunction(func(L *lua.LState) int {
			deps, err := h.coreDAL.ListDepartments(execCtx)
			if err != nil {
				L.RaiseError("dal.groups.list_departments failed: %s", err.Error())
				return 0
			}
			arrTbl := L.NewTable()
			markLuaTableAsArray(L, arrTbl)
			for i, dep := range deps {
				dTbl := L.NewTable()
				L.SetField(dTbl, "id", lua.LString(dep.ID))
				L.SetField(dTbl, "department_id", lua.LString(dep.ID))
				L.SetField(dTbl, "name", lua.LString(dep.Name))
				L.SetField(dTbl, "name_short", lua.LString(dep.NameShort))
				L.SetField(dTbl, "type", lua.LString(dep.Type))
				L.SetField(dTbl, "head_of_department_staff_id", lua.LString(dep.HeadOfDepartmentStaffID))
				arrTbl.RawSetInt(i+1, dTbl)
			}
			L.Push(arrTbl)
			return 1
		}))

		L.SetField(groupsTbl, "list_year_groups", L.NewFunction(func(L *lua.LState) int {
			ygs, err := h.coreDAL.ListYearGroups(execCtx)
			if err != nil {
				L.RaiseError("dal.groups.list_year_groups failed: %s", err.Error())
				return 0
			}
			arrTbl := L.NewTable()
			markLuaTableAsArray(L, arrTbl)
			for i, yg := range ygs {
				yTbl := L.NewTable()
				L.SetField(yTbl, "id", lua.LString(yg.ID))
				L.SetField(yTbl, "year_group_id", lua.LString(yg.ID))
				L.SetField(yTbl, "nc_code", lua.LString(yg.NCCode))
				L.SetField(yTbl, "name", lua.LString(yg.Name))
				L.SetField(yTbl, "phase", lua.LString(yg.Phase))
				L.SetField(yTbl, "key_stage", lua.LString(yg.KeyStage))
				L.SetField(yTbl, "sequence_number", lua.LNumber(yg.SequenceNumber))
				L.SetField(yTbl, "head_of_year_staff_id", lua.LString(yg.HeadOfYearStaffID))
				arrTbl.RawSetInt(i+1, yTbl)
			}
			L.Push(arrTbl)
			return 1
		}))

		L.SetField(groupsTbl, "list_timetable_periods", L.NewFunction(func(L *lua.LState) int {
			var optsTbl *lua.LTable
			var cycleWeek, dayOfWeekID int
			if L.GetTop() >= 1 {
				if tbl, isTbl := L.Get(1).(*lua.LTable); isTbl {
					optsTbl = tbl
					if v := tbl.RawGetString("cycle_week"); v != lua.LNil {
						if n, isNum := v.(lua.LNumber); isNum {
							cycleWeek = int(n)
						}
					}
					if v := tbl.RawGetString("day_of_week_id"); v != lua.LNil {
						if n, isNum := v.(lua.LNumber); isNum {
							dayOfWeekID = int(n)
						}
					}
				} else {
					cycleWeek = L.OptInt(1, 0)
					dayOfWeekID = L.OptInt(2, 0)
				}
			}
			scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, dal.ScopeAll, caller.StaffID, optsTbl)
			if sErr != nil {
				L.RaiseError("dal.groups.list_timetable_periods scope error: %s", sErr.Error())
				return 0
			}
			periods, err := h.coreDAL.ListTimetablePeriodsScoped(execCtx, scope, cycleWeek, dayOfWeekID)
			if err != nil {
				L.RaiseError("dal.groups.list_timetable_periods failed: %s", err.Error())
				return 0
			}
			arrTbl := L.NewTable()
			markLuaTableAsArray(L, arrTbl)
			for i, p := range periods {
				pTbl := L.NewTable()
				L.SetField(pTbl, "id", lua.LString(p.ID))
				L.SetField(pTbl, "period_id", lua.LString(p.ID))
				L.SetField(pTbl, "academic_year_id", lua.LString(p.AcademicYearID))
				L.SetField(pTbl, "cycle_week", lua.LNumber(p.CycleWeek))
				L.SetField(pTbl, "day_of_week_id", lua.LNumber(p.DayOfWeekID))
				L.SetField(pTbl, "period_code", lua.LString(p.PeriodCode))
				L.SetField(pTbl, "name", lua.LString(p.Name))
				L.SetField(pTbl, "start_time", lua.LString(p.StartTime))
				L.SetField(pTbl, "end_time", lua.LString(p.EndTime))
				L.SetField(pTbl, "period_type", lua.LString(p.PeriodType))
				L.SetField(pTbl, "maps_to_statutory_session", lua.LString(p.MapsToStatutorySession))
				L.SetField(pTbl, "sequence_number", lua.LNumber(p.SequenceNumber))
				arrTbl.RawSetInt(i+1, pTbl)
			}
			L.Push(arrTbl)
			return 1
		}))

		L.SetField(groupsTbl, "list_timetable_slots", L.NewFunction(func(L *lua.LState) int {
			var optsTbl *lua.LTable
			if L.GetTop() >= 1 {
				optsTbl, _ = L.Get(1).(*lua.LTable)
			}
			var filter dal.TimetableSlotFilter
			effectiveMode := dal.ScopeAll
			if optsTbl != nil {
				filter.GroupID = luaTableTrimmedString(optsTbl, "group_id")
				filter.StaffID = luaTableTrimmedString(optsTbl, "staff_id")
				filter.RoomID = luaTableTrimmedString(optsTbl, "room_id")
				if v := optsTbl.RawGetString("cycle_week"); v != lua.LNil {
					if n, isNum := v.(lua.LNumber); isNum {
						filter.CycleWeek = int(n)
					}
				}
				if v := optsTbl.RawGetString("day_of_week_id"); v != lua.LNil {
					if n, isNum := v.(lua.LNumber); isNum {
						filter.DayOfWeekID = int(n)
					}
				}
				if optsTbl.RawGetString("strict_own_groups") == lua.LTrue {
					effectiveMode = groupsMode
				}
			}
			scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, effectiveMode, caller.StaffID, optsTbl)
			if sErr != nil {
				L.RaiseError("dal.groups.list_timetable_slots scope error: %s", sErr.Error())
				return 0
			}
			slots, err := h.coreDAL.ListTimetableSlotsScoped(execCtx, scope, filter)
			if err != nil {
				L.RaiseError("dal.groups.list_timetable_slots failed: %s", err.Error())
				return 0
			}
			arrTbl := L.NewTable()
			markLuaTableAsArray(L, arrTbl)
			for i, s := range slots {
				arrTbl.RawSetInt(i+1, timetableSlotToLuaTable(L, s))
			}
			L.Push(arrTbl)
			return 1
		}))

		L.SetField(groupsTbl, "list_houses", L.NewFunction(func(L *lua.LState) int {
			houses, err := h.coreDAL.ListHouses(execCtx)
			if err != nil {
				L.RaiseError("dal.groups.list_houses failed: %s", err.Error())
				return 0
			}
			arrTbl := L.NewTable()
			markLuaTableAsArray(L, arrTbl)
			for i, hs := range houses {
				hTbl := L.NewTable()
				L.SetField(hTbl, "id", lua.LString(hs.ID))
				L.SetField(hTbl, "house_id", lua.LString(hs.HouseID))
				L.SetField(hTbl, "code", lua.LString(hs.Code))
				L.SetField(hTbl, "name_short", lua.LString(hs.NameShort))
				L.SetField(hTbl, "name", lua.LString(hs.Name))
				L.SetField(hTbl, "color_hex", lua.LString(hs.ColorHex))
				L.SetField(hTbl, "colour_hex", lua.LString(hs.ColorHex))
				L.SetField(hTbl, "sequence_number", lua.LNumber(hs.SequenceNumber))
				arrTbl.RawSetInt(i+1, hTbl)
			}
			L.Push(arrTbl)
			return 1
		}))

		L.SetField(groupsTbl, "get_pupil_timetable_slots", L.NewFunction(func(L *lua.LState) int {
			var pupilID string
			var cycleWeek, dayOfWeekID int
			var optsTbl *lua.LTable
			if L.GetTop() >= 1 {
				if tbl, isTbl := L.Get(1).(*lua.LTable); isTbl {
					optsTbl = tbl
					pupilID = luaTableTrimmedString(tbl, "pupil_id")
					cycleWeek = luaTableInt(tbl, "cycle_week", 0)
					dayOfWeekID = luaTableInt(tbl, "day_of_week_id", 0)
				} else {
					pupilID = strings.TrimSpace(L.OptString(1, ""))
					if L.GetTop() >= 2 {
						if tbl, isTbl := L.Get(2).(*lua.LTable); isTbl {
							optsTbl = tbl
							cycleWeek = luaTableInt(tbl, "cycle_week", 0)
							dayOfWeekID = luaTableInt(tbl, "day_of_week_id", 0)
						} else {
							cycleWeek = L.OptInt(2, 0)
							dayOfWeekID = L.OptInt(3, 0)
							if L.GetTop() >= 4 {
								optsTbl, _ = L.Get(4).(*lua.LTable)
							}
						}
					}
				}
			}
			if pupilID == "" {
				L.RaiseError("missing mandatory parameter: pupil_id")
				return 0
			}
			effectiveMode := dal.ScopeAll
			if optsTbl != nil && optsTbl.RawGetString("strict_own_groups") == lua.LTrue {
				effectiveMode = groupsMode
			}
			scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, effectiveMode, caller.StaffID, optsTbl)
			if sErr != nil {
				L.RaiseError("dal.groups.get_pupil_timetable_slots scope error: %s", sErr.Error())
				return 0
			}
			slots, err := h.coreDAL.GetPupilTimetableSlotsScoped(execCtx, scope, pupilID, cycleWeek, dayOfWeekID)
			if err != nil {
				L.RaiseError("dal.groups.get_pupil_timetable_slots failed: %s", err.Error())
				return 0
			}
			arrTbl := L.NewTable()
			markLuaTableAsArray(L, arrTbl)
			for i, s := range slots {
				arrTbl.RawSetInt(i+1, timetableSlotToLuaTable(L, s))
			}
			L.Push(arrTbl)
			return 1
		}))

		L.SetField(groupsTbl, "list_pupil_memberships", L.NewFunction(func(L *lua.LState) int {
			var optsTbl *lua.LTable
			var pupilID, groupID string
			if L.GetTop() >= 1 {
				if tbl, isTbl := L.Get(1).(*lua.LTable); isTbl {
					optsTbl = tbl
					pupilID = luaTableTrimmedString(tbl, "pupil_id")
					groupID = luaTableTrimmedString(tbl, "group_id")
				} else {
					pupilID = strings.TrimSpace(L.OptString(1, ""))
					if L.GetTop() >= 2 {
						optsTbl, _ = L.Get(2).(*lua.LTable)
					}
				}
			}
			effectiveMode := dal.ScopeAll
			if optsTbl != nil && optsTbl.RawGetString("strict_own_groups") == lua.LTrue {
				effectiveMode = groupsMode
			}
			scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, effectiveMode, caller.StaffID, optsTbl)
			if sErr != nil {
				L.RaiseError("dal.groups.list_pupil_memberships scope error: %s", sErr.Error())
				return 0
			}
			items, err := h.coreDAL.ListGroupPupilMembershipsScoped(execCtx, scope, pupilID, groupID)
			if err != nil {
				L.RaiseError("dal.groups.list_pupil_memberships failed: %s", err.Error())
				return 0
			}
			arrTbl := L.NewTable()
			markLuaTableAsArray(L, arrTbl)
			for i, m := range items {
				mTbl := L.NewTable()
				L.SetField(mTbl, "id", lua.LString(m.MembershipID))
				L.SetField(mTbl, "membership_id", lua.LString(m.MembershipID))
				L.SetField(mTbl, "group_id", lua.LString(m.GroupID))
				L.SetField(mTbl, "group_code", lua.LString(m.GroupCode))
				L.SetField(mTbl, "group_name", lua.LString(m.GroupName))
				L.SetField(mTbl, "group_type", lua.LString(m.GroupType))
				L.SetField(mTbl, "pupil_id", lua.LString(m.PupilID))
				L.SetField(mTbl, "academic_year_id", lua.LString(m.AcademicYearID))
				L.SetField(mTbl, "effective_from", lua.LString(m.EffectiveFrom))
				L.SetField(mTbl, "effective_to", lua.LString(m.EffectiveTo))
				arrTbl.RawSetInt(i+1, mTbl)
			}
			L.Push(arrTbl)
			return 1
		}))
	}

	if hasGroupsWrite {
		groupsWriteMode := resolveEffectiveDomainScopeMode(endpointSpec.Scopes, caller, "dal.groups.write")
		L.SetField(groupsTbl, "upsert_pupil_membership", L.NewFunction(func(L *lua.LState) int {
			if !endpointSpec.Mutates {
				L.RaiseError("dal.groups.upsert_pupil_membership is forbidden when endpoint mutates=false")
				return 0
			}
			inTbl, ok := L.Get(1).(*lua.LTable)
			if !ok {
				L.RaiseError("dal.groups.upsert_pupil_membership requires an input table")
				return 0
			}
			var optsTbl *lua.LTable
			if L.GetTop() >= 2 {
				optsTbl, _ = L.Get(2).(*lua.LTable)
			}
			scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, groupsWriteMode, caller.StaffID, optsTbl)
			if sErr != nil {
				L.RaiseError("dal.groups.upsert_pupil_membership scope error: %s", sErr.Error())
				return 0
			}
			res, err := h.coreDAL.UpsertGroupPupilMembershipScoped(execCtx, scope, dal.UpsertGroupPupilMembershipInput{
				MembershipID:   luaTableTrimmedString(inTbl, "membership_id"),
				GroupID:        luaTableTrimmedString(inTbl, "group_id"),
				PupilID:        luaTableTrimmedString(inTbl, "pupil_id"),
				AcademicYearID: luaTableTrimmedString(inTbl, "academic_year_id"),
				EffectiveFrom:  luaTableTrimmedString(inTbl, "effective_from"),
				EffectiveTo:    luaTableTrimmedString(inTbl, "effective_to"),
			})
			if err != nil {
				L.RaiseError("dal.groups.upsert_pupil_membership failed: %s", err.Error())
				return 0
			}
			resTbl := L.NewTable()
			L.SetField(resTbl, "membership_id", lua.LString(res.MembershipID))
			L.SetField(resTbl, "group_id", lua.LString(res.GroupID))
			L.SetField(resTbl, "pupil_id", lua.LString(res.PupilID))
			L.SetField(resTbl, "academic_year_id", lua.LString(res.AcademicYearID))
			L.SetField(resTbl, "effective_from", lua.LString(res.EffectiveFrom))
			L.SetField(resTbl, "effective_to", lua.LString(res.EffectiveTo))
			L.Push(resTbl)
			return 1
		}))

		L.SetField(groupsTbl, "upsert_staff_assignment", L.NewFunction(func(L *lua.LState) int {
			if !endpointSpec.Mutates {
				L.RaiseError("dal.groups.upsert_staff_assignment is forbidden when endpoint mutates=false")
				return 0
			}
			inTbl, ok := L.Get(1).(*lua.LTable)
			if !ok {
				L.RaiseError("dal.groups.upsert_staff_assignment requires an input table")
				return 0
			}
			var optsTbl *lua.LTable
			if L.GetTop() >= 2 {
				optsTbl, _ = L.Get(2).(*lua.LTable)
			}
			scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, groupsWriteMode, caller.StaffID, optsTbl)
			if sErr != nil {
				L.RaiseError("dal.groups.upsert_staff_assignment scope error: %s", sErr.Error())
				return 0
			}
			res, err := h.coreDAL.UpsertGroupStaffAssignmentScoped(execCtx, scope, dal.UpsertGroupStaffAssignmentInput{
				AssignmentID:   luaTableTrimmedString(inTbl, "assignment_id"),
				GroupID:        luaTableTrimmedString(inTbl, "group_id"),
				StaffID:        luaTableTrimmedString(inTbl, "staff_id"),
				Role:           luaTableTrimmedString(inTbl, "role"),
				AcademicYearID: luaTableTrimmedString(inTbl, "academic_year_id"),
				EffectiveFrom:  luaTableTrimmedString(inTbl, "effective_from"),
				EffectiveTo:    luaTableTrimmedString(inTbl, "effective_to"),
			})
			if err != nil {
				L.RaiseError("dal.groups.upsert_staff_assignment failed: %s", err.Error())
				return 0
			}
			resTbl := L.NewTable()
			L.SetField(resTbl, "assignment_id", lua.LString(res.AssignmentID))
			L.SetField(resTbl, "group_id", lua.LString(res.GroupID))
			L.SetField(resTbl, "staff_id", lua.LString(res.StaffID))
			L.SetField(resTbl, "role", lua.LString(res.Role))
			L.SetField(resTbl, "academic_year_id", lua.LString(res.AcademicYearID))
			L.SetField(resTbl, "effective_from", lua.LString(res.EffectiveFrom))
			L.SetField(resTbl, "effective_to", lua.LString(res.EffectiveTo))
			L.Push(resTbl)
			return 1
		}))

		revokeStaffAssignFn := L.NewFunction(func(L *lua.LState) int {
			if !endpointSpec.Mutates {
				L.RaiseError("dal.groups.revoke_staff_assignment is forbidden when endpoint mutates=false")
				return 0
			}
			inTbl, ok := L.Get(1).(*lua.LTable)
			if !ok {
				L.RaiseError("dal.groups.revoke_staff_assignment requires an input table")
				return 0
			}
			var optsTbl *lua.LTable
			if L.GetTop() >= 2 {
				optsTbl, _ = L.Get(2).(*lua.LTable)
			}
			scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, groupsWriteMode, caller.StaffID, optsTbl)
			if sErr != nil {
				L.RaiseError("dal.groups.revoke_staff_assignment scope error: %s", sErr.Error())
				return 0
			}
			deletedCount, err := h.coreDAL.RevokeGroupStaffAssignmentScoped(execCtx, scope, dal.RevokeGroupStaffAssignmentInput{
				AssignmentID:  luaTableTrimmedString(inTbl, "assignment_id"),
				GroupID:       luaTableTrimmedString(inTbl, "group_id"),
				StaffID:       luaTableTrimmedString(inTbl, "staff_id"),
				Role:          luaTableTrimmedString(inTbl, "role"),
				EffectiveFrom: luaTableTrimmedString(inTbl, "effective_from"),
			})
			if err != nil {
				L.RaiseError("dal.groups.revoke_staff_assignment failed: %s", err.Error())
				return 0
			}
			resTbl := L.NewTable()
			L.SetField(resTbl, "revoked", lua.LBool(deletedCount > 0))
			L.SetField(resTbl, "rows_affected", lua.LNumber(deletedCount))
			L.Push(resTbl)
			return 1
		})
		L.SetField(groupsTbl, "revoke_staff_assignment", revokeStaffAssignFn)
		L.SetField(groupsTbl, "delete_staff_assignment", revokeStaffAssignFn)

		L.SetField(groupsTbl, "upsert_group", L.NewFunction(func(L *lua.LState) int {
			if !endpointSpec.Mutates {
				L.RaiseError("dal.groups.upsert_group is forbidden when endpoint mutates=false")
				return 0
			}
			inTbl, ok := L.Get(1).(*lua.LTable)
			if !ok {
				L.RaiseError("dal.groups.upsert_group requires an input table")
				return 0
			}
			var optsTbl *lua.LTable
			if L.GetTop() >= 2 {
				optsTbl, _ = L.Get(2).(*lua.LTable)
			}
			scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, groupsWriteMode, caller.StaffID, optsTbl)
			if sErr != nil {
				L.RaiseError("dal.groups.upsert_group scope error: %s", sErr.Error())
				return 0
			}
			groupID := luaTableTrimmedString(inTbl, "group_id")
			if groupID == "" {
				groupID = luaTableTrimmedString(inTbl, "id")
			}
			var takesAttPtr *bool
			if v := inTbl.RawGetString("takes_attendance"); v != lua.LNil {
				b := v == lua.LTrue
				takesAttPtr = &b
			}
			res, err := h.coreDAL.UpsertGroupScoped(execCtx, scope, dal.UpsertGroupInput{
				GroupID:         groupID,
				AcademicYearID:  luaTableTrimmedString(inTbl, "academic_year_id"),
				GroupType:       luaTableTrimmedString(inTbl, "group_type"),
				Code:            luaTableTrimmedString(inTbl, "code"),
				Name:            luaTableTrimmedString(inTbl, "name"),
				YearGroupID:     luaTableTrimmedString(inTbl, "year_group_id"),
				CourseID:        luaTableTrimmedString(inTbl, "course_id"),
				DefaultRoomID:   luaTableTrimmedString(inTbl, "default_room_id"),
				TakesAttendance: takesAttPtr,
				EffectiveFrom:   luaTableTrimmedString(inTbl, "effective_from"),
				EffectiveTo:     luaTableTrimmedString(inTbl, "effective_to"),
			})
			if err != nil {
				L.RaiseError("dal.groups.upsert_group failed: %s", err.Error())
				return 0
			}
			resTbl := L.NewTable()
			L.SetField(resTbl, "id", lua.LString(res.ID))
			L.SetField(resTbl, "group_id", lua.LString(res.ID))
			L.SetField(resTbl, "academic_year_id", lua.LString(res.AcademicYearID))
			L.SetField(resTbl, "group_type", lua.LString(res.GroupType))
			L.SetField(resTbl, "code", lua.LString(res.Code))
			L.SetField(resTbl, "name", lua.LString(res.Name))
			L.SetField(resTbl, "default_room_id", lua.LString(res.DefaultRoomID))
			L.SetField(resTbl, "takes_attendance", lua.LBool(res.TakesAttendance))
			L.SetField(resTbl, "effective_from", lua.LString(res.EffectiveFrom))
			if res.EffectiveTo != nil {
				L.SetField(resTbl, "effective_to", lua.LString(*res.EffectiveTo))
			}
			L.Push(resTbl)
			return 1
		}))

		L.SetField(groupsTbl, "upsert_timetable_period", L.NewFunction(func(L *lua.LState) int {
			if !endpointSpec.Mutates {
				L.RaiseError("dal.groups.upsert_timetable_period is forbidden when endpoint mutates=false")
				return 0
			}
			inTbl, ok := L.Get(1).(*lua.LTable)
			if !ok {
				L.RaiseError("dal.groups.upsert_timetable_period requires an input table")
				return 0
			}
			var optsTbl *lua.LTable
			if L.GetTop() >= 2 {
				optsTbl, _ = L.Get(2).(*lua.LTable)
			}
			scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, groupsWriteMode, caller.StaffID, optsTbl)
			if sErr != nil {
				L.RaiseError("dal.groups.upsert_timetable_period scope error: %s", sErr.Error())
				return 0
			}
			periodID := luaTableTrimmedString(inTbl, "period_id")
			if periodID == "" {
				periodID = luaTableTrimmedString(inTbl, "id")
			}
			var cycleWeek, dayOfWeekID, seqNum int
			if v := inTbl.RawGetString("cycle_week"); v != lua.LNil {
				if n, isNum := v.(lua.LNumber); isNum {
					cycleWeek = int(n)
				}
			}
			if v := inTbl.RawGetString("day_of_week_id"); v != lua.LNil {
				if n, isNum := v.(lua.LNumber); isNum {
					dayOfWeekID = int(n)
				}
			}
			if v := inTbl.RawGetString("sequence_number"); v != lua.LNil {
				if n, isNum := v.(lua.LNumber); isNum {
					seqNum = int(n)
				}
			}
			res, err := h.coreDAL.UpsertTimetablePeriodScoped(execCtx, scope, dal.UpsertTimetablePeriodInput{
				PeriodID:               periodID,
				AcademicYearID:         luaTableTrimmedString(inTbl, "academic_year_id"),
				CycleWeek:              cycleWeek,
				DayOfWeekID:            dayOfWeekID,
				PeriodCode:             luaTableTrimmedString(inTbl, "period_code"),
				Name:                   luaTableTrimmedString(inTbl, "name"),
				StartTime:              luaTableTrimmedString(inTbl, "start_time"),
				EndTime:                luaTableTrimmedString(inTbl, "end_time"),
				PeriodType:             luaTableTrimmedString(inTbl, "period_type"),
				MapsToStatutorySession: luaTableTrimmedString(inTbl, "maps_to_statutory_session"),
				SequenceNumber:         seqNum,
			})
			if err != nil {
				L.RaiseError("dal.groups.upsert_timetable_period failed: %s", err.Error())
				return 0
			}
			resTbl := L.NewTable()
			L.SetField(resTbl, "id", lua.LString(res.ID))
			L.SetField(resTbl, "period_id", lua.LString(res.ID))
			L.SetField(resTbl, "academic_year_id", lua.LString(res.AcademicYearID))
			L.SetField(resTbl, "cycle_week", lua.LNumber(res.CycleWeek))
			L.SetField(resTbl, "day_of_week_id", lua.LNumber(res.DayOfWeekID))
			L.SetField(resTbl, "period_code", lua.LString(res.PeriodCode))
			L.SetField(resTbl, "name", lua.LString(res.Name))
			L.SetField(resTbl, "start_time", lua.LString(res.StartTime))
			L.SetField(resTbl, "end_time", lua.LString(res.EndTime))
			L.SetField(resTbl, "period_type", lua.LString(res.PeriodType))
			L.SetField(resTbl, "maps_to_statutory_session", lua.LString(res.MapsToStatutorySession))
			L.SetField(resTbl, "sequence_number", lua.LNumber(res.SequenceNumber))
			L.Push(resTbl)
			return 1
		}))

		L.SetField(groupsTbl, "upsert_timetable_slot", L.NewFunction(func(L *lua.LState) int {
			if !endpointSpec.Mutates {
				L.RaiseError("dal.groups.upsert_timetable_slot is forbidden when endpoint mutates=false")
				return 0
			}
			inTbl, ok := L.Get(1).(*lua.LTable)
			if !ok {
				L.RaiseError("dal.groups.upsert_timetable_slot requires an input table")
				return 0
			}
			var optsTbl *lua.LTable
			if L.GetTop() >= 2 {
				optsTbl, _ = L.Get(2).(*lua.LTable)
			}
			scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, groupsWriteMode, caller.StaffID, optsTbl)
			if sErr != nil {
				L.RaiseError("dal.groups.upsert_timetable_slot scope error: %s", sErr.Error())
				return 0
			}
			slotID := luaTableTrimmedString(inTbl, "slot_id")
			if slotID == "" {
				slotID = luaTableTrimmedString(inTbl, "id")
			}
			periodID := luaTableTrimmedString(inTbl, "timetable_period_id")
			if periodID == "" {
				periodID = luaTableTrimmedString(inTbl, "period_id")
			}
			res, err := h.coreDAL.UpsertTimetableSlotScoped(execCtx, scope, dal.UpsertTimetableSlotInput{
				SlotID:            slotID,
				TimetablePeriodID: periodID,
				GroupID:           luaTableTrimmedString(inTbl, "group_id"),
				StaffID:           luaTableTrimmedString(inTbl, "staff_id"),
				RoomID:            luaTableTrimmedString(inTbl, "room_id"),
				EffectiveFrom:     luaTableTrimmedString(inTbl, "effective_from"),
				EffectiveTo:       luaTableTrimmedString(inTbl, "effective_to"),
			})
			if err != nil {
				L.RaiseError("dal.groups.upsert_timetable_slot failed: %s", err.Error())
				return 0
			}
			resTbl := L.NewTable()
			L.SetField(resTbl, "id", lua.LString(res.SlotID))
			L.SetField(resTbl, "slot_id", lua.LString(res.SlotID))
			L.SetField(resTbl, "timetable_period_id", lua.LString(res.TimetablePeriodID))
			L.SetField(resTbl, "period_id", lua.LString(res.TimetablePeriodID))
			L.SetField(resTbl, "group_id", lua.LString(res.GroupID))
			L.SetField(resTbl, "staff_id", lua.LString(res.StaffID))
			L.SetField(resTbl, "room_id", lua.LString(res.RoomID))
			L.SetField(resTbl, "effective_from", lua.LString(res.EffectiveFrom))
			L.SetField(resTbl, "effective_to", lua.LString(res.EffectiveTo))
			L.Push(resTbl)
			return 1
		}))

		L.SetField(groupsTbl, "upsert_house", L.NewFunction(func(L *lua.LState) int {
			if !endpointSpec.Mutates {
				L.RaiseError("dal.groups.upsert_house is forbidden when endpoint mutates=false")
				return 0
			}
			inTbl, ok := L.Get(1).(*lua.LTable)
			if !ok {
				L.RaiseError("dal.groups.upsert_house requires an input table")
				return 0
			}
			var optsTbl *lua.LTable
			if L.GetTop() >= 2 {
				optsTbl, _ = L.Get(2).(*lua.LTable)
			}
			scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, groupsWriteMode, caller.StaffID, optsTbl)
			if sErr != nil {
				L.RaiseError("dal.groups.upsert_house scope error: %s", sErr.Error())
				return 0
			}
			houseID := luaTableTrimmedString(inTbl, "house_id")
			if houseID == "" {
				houseID = luaTableTrimmedString(inTbl, "id")
			}
			colorHex := luaTableTrimmedString(inTbl, "color_hex")
			if colorHex == "" {
				colorHex = luaTableTrimmedString(inTbl, "colour_hex")
			}
			hs, err := h.coreDAL.UpsertHouseScoped(execCtx, scope, dal.HouseSummary{
				ID:             houseID,
				HouseID:        houseID,
				Code:           luaTableTrimmedString(inTbl, "code"),
				NameShort:      luaTableTrimmedString(inTbl, "name_short"),
				Name:           luaTableTrimmedString(inTbl, "name"),
				ColorHex:       colorHex,
				SequenceNumber: luaTableInt(inTbl, "sequence_number", 0),
			})
			if err != nil {
				L.RaiseError("dal.groups.upsert_house failed: %s", err.Error())
				return 0
			}
			hTbl := L.NewTable()
			L.SetField(hTbl, "id", lua.LString(hs.ID))
			L.SetField(hTbl, "house_id", lua.LString(hs.HouseID))
			L.SetField(hTbl, "code", lua.LString(hs.Code))
			L.SetField(hTbl, "name_short", lua.LString(hs.NameShort))
			L.SetField(hTbl, "name", lua.LString(hs.Name))
			L.SetField(hTbl, "color_hex", lua.LString(hs.ColorHex))
			L.SetField(hTbl, "colour_hex", lua.LString(hs.ColorHex))
			L.SetField(hTbl, "sequence_number", lua.LNumber(hs.SequenceNumber))
			L.Push(hTbl)
			return 1
		}))

		L.SetField(groupsTbl, "upsert_room", L.NewFunction(func(L *lua.LState) int {
			if !endpointSpec.Mutates {
				L.RaiseError("dal.groups.upsert_room is forbidden when endpoint mutates=false")
				return 0
			}
			inTbl, ok := L.Get(1).(*lua.LTable)
			if !ok {
				L.RaiseError("dal.groups.upsert_room requires an input table")
				return 0
			}
			var optsTbl *lua.LTable
			if L.GetTop() >= 2 {
				optsTbl, _ = L.Get(2).(*lua.LTable)
			}
			scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, groupsWriteMode, caller.StaffID, optsTbl)
			if sErr != nil {
				L.RaiseError("dal.groups.upsert_room scope error: %s", sErr.Error())
				return 0
			}
			roomID := luaTableTrimmedString(inTbl, "room_id")
			if roomID == "" {
				roomID = luaTableTrimmedString(inTbl, "id")
			}
			isActive := true
			if v := inTbl.RawGetString("is_active"); v != lua.LNil {
				isActive = v == lua.LTrue
			}
			rm, err := h.coreDAL.UpsertRoomScoped(execCtx, scope, dal.RoomSummary{
				ID:             roomID,
				Code:           luaTableTrimmedString(inTbl, "code"),
				Name:           luaTableTrimmedString(inTbl, "name"),
				RoomType:       luaTableTrimmedString(inTbl, "room_type"),
				Capacity:       luaTableInt(inTbl, "capacity", 30),
				IsActive:       isActive,
				PhoneExtension: luaTableTrimmedString(inTbl, "phone_extension"),
			})
			if err != nil {
				L.RaiseError("dal.groups.upsert_room failed: %s", err.Error())
				return 0
			}
			rTbl := L.NewTable()
			L.SetField(rTbl, "id", lua.LString(rm.ID))
			L.SetField(rTbl, "room_id", lua.LString(rm.ID))
			L.SetField(rTbl, "code", lua.LString(rm.Code))
			L.SetField(rTbl, "name", lua.LString(rm.Name))
			L.SetField(rTbl, "room_type", lua.LString(rm.RoomType))
			L.SetField(rTbl, "capacity", lua.LNumber(rm.Capacity))
			L.SetField(rTbl, "is_active", lua.LBool(rm.IsActive))
			L.SetField(rTbl, "phone_extension", lua.LString(rm.PhoneExtension))
			L.Push(rTbl)
			return 1
		}))

		L.SetField(groupsTbl, "upsert_department", L.NewFunction(func(L *lua.LState) int {
			if !endpointSpec.Mutates {
				L.RaiseError("dal.groups.upsert_department is forbidden when endpoint mutates=false")
				return 0
			}
			inTbl, ok := L.Get(1).(*lua.LTable)
			if !ok {
				L.RaiseError("dal.groups.upsert_department requires an input table")
				return 0
			}
			var optsTbl *lua.LTable
			if L.GetTop() >= 2 {
				optsTbl, _ = L.Get(2).(*lua.LTable)
			}
			scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, groupsWriteMode, caller.StaffID, optsTbl)
			if sErr != nil {
				L.RaiseError("dal.groups.upsert_department scope error: %s", sErr.Error())
				return 0
			}
			depID := luaTableTrimmedString(inTbl, "department_id")
			if depID == "" {
				depID = luaTableTrimmedString(inTbl, "id")
			}
			dep, err := h.coreDAL.UpsertDepartmentScoped(execCtx, scope, dal.DepartmentSummary{
				ID:                      depID,
				Name:                    luaTableTrimmedString(inTbl, "name"),
				NameShort:               luaTableTrimmedString(inTbl, "name_short"),
				Type:                    luaTableTrimmedString(inTbl, "type"),
				HeadOfDepartmentStaffID: luaTableTrimmedString(inTbl, "head_of_department_staff_id"),
			})
			if err != nil {
				L.RaiseError("dal.groups.upsert_department failed: %s", err.Error())
				return 0
			}
			dTbl := L.NewTable()
			L.SetField(dTbl, "id", lua.LString(dep.ID))
			L.SetField(dTbl, "department_id", lua.LString(dep.ID))
			L.SetField(dTbl, "name", lua.LString(dep.Name))
			L.SetField(dTbl, "name_short", lua.LString(dep.NameShort))
			L.SetField(dTbl, "type", lua.LString(dep.Type))
			L.SetField(dTbl, "head_of_department_staff_id", lua.LString(dep.HeadOfDepartmentStaffID))
			L.Push(dTbl)
			return 1
		}))

		L.SetField(groupsTbl, "upsert_course", L.NewFunction(func(L *lua.LState) int {
			if !endpointSpec.Mutates {
				L.RaiseError("dal.groups.upsert_course is forbidden when endpoint mutates=false")
				return 0
			}
			inTbl, ok := L.Get(1).(*lua.LTable)
			if !ok {
				L.RaiseError("dal.groups.upsert_course requires an input table")
				return 0
			}
			var optsTbl *lua.LTable
			if L.GetTop() >= 2 {
				optsTbl, _ = L.Get(2).(*lua.LTable)
			}
			scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, groupsWriteMode, caller.StaffID, optsTbl)
			if sErr != nil {
				L.RaiseError("dal.groups.upsert_course scope error: %s", sErr.Error())
				return 0
			}
			courseID := luaTableTrimmedString(inTbl, "course_id")
			if courseID == "" {
				courseID = luaTableTrimmedString(inTbl, "id")
			}
			c, err := h.coreDAL.UpsertCourseScoped(execCtx, scope, dal.CourseSummary{
				ID:             courseID,
				AcademicYearID: luaTableTrimmedString(inTbl, "academic_year_id"),
				DepartmentID:   luaTableTrimmedString(inTbl, "department_id"),
				Code:           luaTableTrimmedString(inTbl, "code"),
				Name:           luaTableTrimmedString(inTbl, "name"),
				DfESubjectCode: luaTableTrimmedString(inTbl, "dfe_subject_code"),
			})
			if err != nil {
				L.RaiseError("dal.groups.upsert_course failed: %s", err.Error())
				return 0
			}
			cTbl := L.NewTable()
			L.SetField(cTbl, "id", lua.LString(c.ID))
			L.SetField(cTbl, "course_id", lua.LString(c.ID))
			L.SetField(cTbl, "academic_year_id", lua.LString(c.AcademicYearID))
			L.SetField(cTbl, "department_id", lua.LString(c.DepartmentID))
			L.SetField(cTbl, "code", lua.LString(c.Code))
			L.SetField(cTbl, "name", lua.LString(c.Name))
			L.SetField(cTbl, "dfe_subject_code", lua.LString(c.DfESubjectCode))
			L.Push(cTbl)
			return 1
		}))

		L.SetField(groupsTbl, "upsert_year_group", L.NewFunction(func(L *lua.LState) int {
			if !endpointSpec.Mutates {
				L.RaiseError("dal.groups.upsert_year_group is forbidden when endpoint mutates=false")
				return 0
			}
			inTbl, ok := L.Get(1).(*lua.LTable)
			if !ok {
				L.RaiseError("dal.groups.upsert_year_group requires an input table")
				return 0
			}
			var optsTbl *lua.LTable
			if L.GetTop() >= 2 {
				optsTbl, _ = L.Get(2).(*lua.LTable)
			}
			scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, groupsWriteMode, caller.StaffID, optsTbl)
			if sErr != nil {
				L.RaiseError("dal.groups.upsert_year_group scope error: %s", sErr.Error())
				return 0
			}
			ygID := luaTableTrimmedString(inTbl, "year_group_id")
			if ygID == "" {
				ygID = luaTableTrimmedString(inTbl, "id")
			}
			yg, err := h.coreDAL.UpsertYearGroupScoped(execCtx, scope, dal.YearGroupSummary{
				ID:                ygID,
				NCCode:            luaTableTrimmedString(inTbl, "nc_code"),
				Name:              luaTableTrimmedString(inTbl, "name"),
				Phase:             luaTableTrimmedString(inTbl, "phase"),
				KeyStage:          luaTableTrimmedString(inTbl, "key_stage"),
				SequenceNumber:    luaTableInt(inTbl, "sequence_number", 0),
				HeadOfYearStaffID: luaTableTrimmedString(inTbl, "head_of_year_staff_id"),
			})
			if err != nil {
				L.RaiseError("dal.groups.upsert_year_group failed: %s", err.Error())
				return 0
			}
			yTbl := L.NewTable()
			L.SetField(yTbl, "id", lua.LString(yg.ID))
			L.SetField(yTbl, "year_group_id", lua.LString(yg.ID))
			L.SetField(yTbl, "nc_code", lua.LString(yg.NCCode))
			L.SetField(yTbl, "name", lua.LString(yg.Name))
			L.SetField(yTbl, "phase", lua.LString(yg.Phase))
			L.SetField(yTbl, "key_stage", lua.LString(yg.KeyStage))
			L.SetField(yTbl, "sequence_number", lua.LNumber(yg.SequenceNumber))
			L.SetField(yTbl, "head_of_year_staff_id", lua.LString(yg.HeadOfYearStaffID))
			L.Push(yTbl)
			return 1
		}))
	}

	L.SetField(dalTbl, "groups", groupsTbl)
}
