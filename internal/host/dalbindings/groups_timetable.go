package dalbindings

import (
	"context"
	"strings"

	lua "github.com/yuin/gopher-lua"

	"optimis/internal/dal"
	"optimis/internal/host/types"
)

// ============================================================================
// Timetable Lua Table Entity Mappers
// ============================================================================

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

func timetablePeriodToLuaTable(L *lua.LState, p dal.TimetablePeriodSummary) *lua.LTable {
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
	return pTbl
}

func upsertTimetableSlotResultToLuaTable(L *lua.LState, res *dal.UpsertTimetableSlotResult) *lua.LTable {
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
	return resTbl
}

// ============================================================================
// Timetable Argument Extraction Helpers
// ============================================================================

func extractPupilTimetableSlotsArgs(L *lua.LState) (string, int, int, *lua.LTable) {
	var pupilID string
	var cycleWeek, dayOfWeekID int
	var optsTbl *lua.LTable
	if L.GetTop() >= 1 {
		if tbl, isTbl := L.Get(1).(*lua.LTable); isTbl {
			optsTbl = tbl
			pupilID = luaTableStringOr(tbl, "pupil_id", "id")
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
	return pupilID, cycleWeek, dayOfWeekID, optsTbl
}

func extractFreeStaffPeriodArgs(L *lua.LState) (int, int, string, *lua.LTable) {
	var cycleWeek, dayOfWeekID int
	var periodIdentifier string
	var optsTbl *lua.LTable
	if firstTbl, isTbl := L.Get(1).(*lua.LTable); isTbl {
		cycleWeek = luaTableInt(firstTbl, "cycle_week", 1)
		dayOfWeekID = luaTableInt(firstTbl, "day_of_week_id", 1)
		periodIdentifier = luaTableStringOr(firstTbl, "period_id", "period_code")
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
	return cycleWeek, dayOfWeekID, periodIdentifier, optsTbl
}

func extractTimetablePeriodsArgs(L *lua.LState) (int, int, *lua.LTable) {
	var optsTbl *lua.LTable
	var cycleWeek, dayOfWeekID int
	if L.GetTop() >= 1 {
		if tbl, isTbl := L.Get(1).(*lua.LTable); isTbl {
			optsTbl = tbl
			cycleWeek = luaTableInt(tbl, "cycle_week", 0)
			dayOfWeekID = luaTableInt(tbl, "day_of_week_id", 0)
		} else {
			cycleWeek = L.OptInt(1, 0)
			dayOfWeekID = L.OptInt(2, 0)
		}
	}
	return cycleWeek, dayOfWeekID, optsTbl
}

// ============================================================================
// Timetable Read Bindings
// ============================================================================

func (b *Binder) bindLuaDALGroupsTimetableRead(
	L *lua.LState,
	execCtx context.Context,
	groupsMode dal.ScopeMode,
	caller types.CallerContext,
	groupsTbl *lua.LTable,
) {
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
		scope, sErr := resolveStrictOwnGroupsScope(execCtx, b.coreDAL, groupsMode, caller.StaffID, optsTbl)
		if sErr != nil {
			L.RaiseError("dal.groups.get_staff_timetable_slots scope error: %s", sErr.Error())
			return 0
		}
		slots, err := b.coreDAL.GetStaffTimetableSlotsScoped(execCtx, scope, staffID, cycleWeek, dayOfWeekID)
		if err != nil {
			L.RaiseError("dal.groups.get_staff_timetable_slots failed: %s", err.Error())
			return 0
		}
		return pushMappedSlice(L, slots, timetableSlotToLuaTable)
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
		scope, sErr := resolveStrictOwnGroupsScope(execCtx, b.coreDAL, groupsMode, caller.StaffID, optsTbl)
		if sErr != nil {
			L.RaiseError("dal.groups.get_staff_timetable_slots_batch scope error: %s", sErr.Error())
			return 0
		}
		batchSlots, err := b.coreDAL.GetStaffTimetableSlotsBatchScoped(execCtx, scope, staffIDs, cycleWeek, dayOfWeekID)
		if err != nil {
			L.RaiseError("dal.groups.get_staff_timetable_slots_batch failed: %s", err.Error())
			return 0
		}
		resTbl := L.NewTable()
		for sid, slots := range batchSlots {
			L.SetField(resTbl, sid, mapSliceToLuaTable(L, slots, timetableSlotToLuaTable))
		}
		L.Push(resTbl)
		return 1
	}))

	L.SetField(groupsTbl, "list_free_staff_for_period", L.NewFunction(func(L *lua.LState) int {
		cycleWeek, dayOfWeekID, periodIdentifier, optsTbl := extractFreeStaffPeriodArgs(L)
		scope, sErr := extractScopeFromLuaOpts(execCtx, b.coreDAL, dal.ScopeAll, caller.StaffID, optsTbl)
		if sErr != nil {
			L.RaiseError("dal.groups.list_free_staff_for_period scope error: %s", sErr.Error())
			return 0
		}
		freeStaff, err := b.coreDAL.ListFreeStaffForPeriodScoped(execCtx, scope, cycleWeek, dayOfWeekID, periodIdentifier)
		if err != nil {
			L.RaiseError("dal.groups.list_free_staff_for_period failed: %s", err.Error())
			return 0
		}
		return pushMappedSlice(L, freeStaff, staffToLuaTable)
	}))

	L.SetField(groupsTbl, "list_timetable_periods", L.NewFunction(func(L *lua.LState) int {
		cycleWeek, dayOfWeekID, optsTbl := extractTimetablePeriodsArgs(L)
		scope, sErr := extractScopeFromLuaOpts(execCtx, b.coreDAL, dal.ScopeAll, caller.StaffID, optsTbl)
		if sErr != nil {
			L.RaiseError("dal.groups.list_timetable_periods scope error: %s", sErr.Error())
			return 0
		}
		periods, err := b.coreDAL.ListTimetablePeriodsScoped(execCtx, scope, cycleWeek, dayOfWeekID)
		if err != nil {
			L.RaiseError("dal.groups.list_timetable_periods failed: %s", err.Error())
			return 0
		}
		return pushMappedSlice(L, periods, timetablePeriodToLuaTable)
	}))

	L.SetField(groupsTbl, "list_timetable_slots", L.NewFunction(func(L *lua.LState) int {
		var optsTbl *lua.LTable
		if L.GetTop() >= 1 {
			optsTbl, _ = L.Get(1).(*lua.LTable)
		}
		var filter dal.TimetableSlotFilter
		if optsTbl != nil {
			filter.GroupID = luaTableTrimmedString(optsTbl, "group_id")
			filter.StaffID = luaTableTrimmedString(optsTbl, "staff_id")
			filter.RoomID = luaTableTrimmedString(optsTbl, "room_id")
			filter.CycleWeek = luaTableInt(optsTbl, "cycle_week", 0)
			filter.DayOfWeekID = luaTableInt(optsTbl, "day_of_week_id", 0)
		}
		scope, sErr := resolveStrictOwnGroupsScope(execCtx, b.coreDAL, groupsMode, caller.StaffID, optsTbl)
		if sErr != nil {
			L.RaiseError("dal.groups.list_timetable_slots scope error: %s", sErr.Error())
			return 0
		}
		slots, err := b.coreDAL.ListTimetableSlotsScoped(execCtx, scope, filter)
		if err != nil {
			L.RaiseError("dal.groups.list_timetable_slots failed: %s", err.Error())
			return 0
		}
		return pushMappedSlice(L, slots, timetableSlotToLuaTable)
	}))

	L.SetField(groupsTbl, "get_pupil_timetable_slots", L.NewFunction(func(L *lua.LState) int {
		pupilID, cycleWeek, dayOfWeekID, optsTbl := extractPupilTimetableSlotsArgs(L)
		if pupilID == "" {
			L.RaiseError("missing mandatory parameter: pupil_id")
			return 0
		}
		scope, sErr := resolveStrictOwnGroupsScope(execCtx, b.coreDAL, groupsMode, caller.StaffID, optsTbl)
		if sErr != nil {
			L.RaiseError("dal.groups.get_pupil_timetable_slots scope error: %s", sErr.Error())
			return 0
		}
		slots, err := b.coreDAL.GetPupilTimetableSlotsScoped(execCtx, scope, pupilID, cycleWeek, dayOfWeekID)
		if err != nil {
			L.RaiseError("dal.groups.get_pupil_timetable_slots failed: %s", err.Error())
			return 0
		}
		return pushMappedSlice(L, slots, timetableSlotToLuaTable)
	}))
}

// ============================================================================
// Timetable Write Bindings
// ============================================================================

func (b *Binder) bindLuaDALGroupsTimetableWrite(
	L *lua.LState,
	execCtx context.Context,
	endpointSpec types.EndpointSpec,
	caller types.CallerContext,
	groupsWriteMode dal.ScopeMode,
	groupsTbl *lua.LTable,
) {
	L.SetField(groupsTbl, "upsert_timetable_period", L.NewFunction(func(L *lua.LState) int {
		inTbl, scope, ok := parseGroupWriteArgs(L, execCtx, b.coreDAL, endpointSpec, caller, groupsWriteMode, "upsert_timetable_period")
		if !ok {
			return 0
		}
		res, err := b.coreDAL.UpsertTimetablePeriodScoped(execCtx, scope, dal.UpsertTimetablePeriodInput{
			PeriodID:               luaTableStringOr(inTbl, "period_id", "id"),
			AcademicYearID:         luaTableTrimmedString(inTbl, "academic_year_id"),
			CycleWeek:              luaTableInt(inTbl, "cycle_week", 0),
			DayOfWeekID:            luaTableInt(inTbl, "day_of_week_id", 0),
			PeriodCode:             luaTableTrimmedString(inTbl, "period_code"),
			Name:                   luaTableTrimmedString(inTbl, "name"),
			StartTime:              luaTableTrimmedString(inTbl, "start_time"),
			EndTime:                luaTableTrimmedString(inTbl, "end_time"),
			PeriodType:             luaTableTrimmedString(inTbl, "period_type"),
			MapsToStatutorySession: luaTableTrimmedString(inTbl, "maps_to_statutory_session"),
			SequenceNumber:         luaTableInt(inTbl, "sequence_number", 0),
		})
		if err != nil {
			L.RaiseError("dal.groups.upsert_timetable_period failed: %s", err.Error())
			return 0
		}
		L.Push(timetablePeriodToLuaTable(L, *res))
		return 1
	}))

	L.SetField(groupsTbl, "upsert_timetable_slot", L.NewFunction(func(L *lua.LState) int {
		inTbl, scope, ok := parseGroupWriteArgs(L, execCtx, b.coreDAL, endpointSpec, caller, groupsWriteMode, "upsert_timetable_slot")
		if !ok {
			return 0
		}
		res, err := b.coreDAL.UpsertTimetableSlotScoped(execCtx, scope, dal.UpsertTimetableSlotInput{
			SlotID:            luaTableStringOr(inTbl, "slot_id", "id"),
			TimetablePeriodID: luaTableStringOr(inTbl, "timetable_period_id", "period_id"),
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
		L.Push(upsertTimetableSlotResultToLuaTable(L, res))
		return 1
	}))
}
