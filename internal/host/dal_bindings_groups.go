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
	}

	L.SetField(dalTbl, "groups", groupsTbl)
}
