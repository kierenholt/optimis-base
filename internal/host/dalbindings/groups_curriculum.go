package dalbindings

import (
	"context"
	"strings"

	lua "github.com/yuin/gopher-lua"

	"optimis/internal/dal"
	"optimis/internal/host/types"
)

// ============================================================================
// Curriculum & Catalog Lua Table Entity Mappers
// ============================================================================

func roomToLuaTable(L *lua.LState, r dal.RoomSummary) *lua.LTable {
	rTbl := L.NewTable()
	L.SetField(rTbl, "id", lua.LString(r.ID))
	L.SetField(rTbl, "room_id", lua.LString(r.ID))
	L.SetField(rTbl, "code", lua.LString(r.Code))
	L.SetField(rTbl, "name", lua.LString(r.Name))
	L.SetField(rTbl, "room_type", lua.LString(r.RoomType))
	L.SetField(rTbl, "capacity", lua.LNumber(r.Capacity))
	L.SetField(rTbl, "is_active", lua.LBool(r.IsActive))
	L.SetField(rTbl, "phone_extension", lua.LString(r.PhoneExtension))
	return rTbl
}

func courseToLuaTable(L *lua.LState, c dal.CourseSummary) *lua.LTable {
	cTbl := L.NewTable()
	L.SetField(cTbl, "id", lua.LString(c.ID))
	L.SetField(cTbl, "course_id", lua.LString(c.ID))
	L.SetField(cTbl, "academic_year_id", lua.LString(c.AcademicYearID))
	L.SetField(cTbl, "department_id", lua.LString(c.DepartmentID))
	L.SetField(cTbl, "code", lua.LString(c.Code))
	L.SetField(cTbl, "name", lua.LString(c.Name))
	L.SetField(cTbl, "dfe_subject_code", lua.LString(c.DfESubjectCode))
	return cTbl
}

func departmentToLuaTable(L *lua.LState, dep dal.DepartmentSummary) *lua.LTable {
	dTbl := L.NewTable()
	L.SetField(dTbl, "id", lua.LString(dep.ID))
	L.SetField(dTbl, "department_id", lua.LString(dep.ID))
	L.SetField(dTbl, "name", lua.LString(dep.Name))
	L.SetField(dTbl, "name_short", lua.LString(dep.NameShort))
	L.SetField(dTbl, "type", lua.LString(dep.Type))
	L.SetField(dTbl, "head_of_department_staff_id", lua.LString(dep.HeadOfDepartmentStaffID))
	return dTbl
}

func yearGroupToLuaTable(L *lua.LState, yg dal.YearGroupSummary) *lua.LTable {
	yTbl := L.NewTable()
	L.SetField(yTbl, "id", lua.LString(yg.ID))
	L.SetField(yTbl, "year_group_id", lua.LString(yg.ID))
	L.SetField(yTbl, "nc_code", lua.LString(yg.NCCode))
	L.SetField(yTbl, "name", lua.LString(yg.Name))
	L.SetField(yTbl, "phase", lua.LString(yg.Phase))
	L.SetField(yTbl, "key_stage", lua.LString(yg.KeyStage))
	L.SetField(yTbl, "sequence_number", lua.LNumber(yg.SequenceNumber))
	L.SetField(yTbl, "head_of_year_staff_id", lua.LString(yg.HeadOfYearStaffID))
	return yTbl
}

func houseToLuaTable(L *lua.LState, hs dal.HouseSummary) *lua.LTable {
	hTbl := L.NewTable()
	L.SetField(hTbl, "id", lua.LString(hs.ID))
	L.SetField(hTbl, "house_id", lua.LString(hs.HouseID))
	L.SetField(hTbl, "code", lua.LString(hs.Code))
	L.SetField(hTbl, "name_short", lua.LString(hs.NameShort))
	L.SetField(hTbl, "name", lua.LString(hs.Name))
	L.SetField(hTbl, "color_hex", lua.LString(hs.ColorHex))
	L.SetField(hTbl, "colour_hex", lua.LString(hs.ColorHex))
	L.SetField(hTbl, "sequence_number", lua.LNumber(hs.SequenceNumber))
	return hTbl
}

// ============================================================================
// Curriculum & Catalog Read Bindings
// ============================================================================

func (b *Binder) bindLuaDALGroupsCatalogRead(
	L *lua.LState,
	execCtx context.Context,
	groupsTbl *lua.LTable,
) {
	L.SetField(groupsTbl, "list_rooms", L.NewFunction(func(L *lua.LState) int {
		rooms, err := b.coreDAL.ListRooms(execCtx)
		if err != nil {
			L.RaiseError("dal.groups.list_rooms failed: %s", err.Error())
			return 0
		}
		return pushMappedSlice(L, rooms, roomToLuaTable)
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
		courses, err := b.coreDAL.ListCourses(execCtx, ayID)
		if err != nil {
			L.RaiseError("dal.groups.list_courses failed: %s", err.Error())
			return 0
		}
		return pushMappedSlice(L, courses, courseToLuaTable)
	}))

	L.SetField(groupsTbl, "list_departments", L.NewFunction(func(L *lua.LState) int {
		deps, err := b.coreDAL.ListDepartments(execCtx)
		if err != nil {
			L.RaiseError("dal.groups.list_departments failed: %s", err.Error())
			return 0
		}
		return pushMappedSlice(L, deps, departmentToLuaTable)
	}))

	L.SetField(groupsTbl, "list_year_groups", L.NewFunction(func(L *lua.LState) int {
		ygs, err := b.coreDAL.ListYearGroups(execCtx)
		if err != nil {
			L.RaiseError("dal.groups.list_year_groups failed: %s", err.Error())
			return 0
		}
		return pushMappedSlice(L, ygs, yearGroupToLuaTable)
	}))

	L.SetField(groupsTbl, "list_houses", L.NewFunction(func(L *lua.LState) int {
		houses, err := b.coreDAL.ListHouses(execCtx)
		if err != nil {
			L.RaiseError("dal.groups.list_houses failed: %s", err.Error())
			return 0
		}
		return pushMappedSlice(L, houses, houseToLuaTable)
	}))
}

// ============================================================================
// Curriculum & Catalog Write Bindings
// ============================================================================

func (b *Binder) bindLuaDALGroupsCatalogWrite(
	L *lua.LState,
	execCtx context.Context,
	endpointSpec types.EndpointSpec,
	caller types.CallerContext,
	groupsWriteMode dal.ScopeMode,
	groupsTbl *lua.LTable,
) {
	L.SetField(groupsTbl, "upsert_house", L.NewFunction(func(L *lua.LState) int {
		inTbl, scope, ok := parseGroupWriteArgs(L, execCtx, b.coreDAL, endpointSpec, caller, groupsWriteMode, "upsert_house")
		if !ok {
			return 0
		}
		houseID := luaTableStringOr(inTbl, "house_id", "id")
		hs, err := b.coreDAL.UpsertHouseScoped(execCtx, scope, dal.HouseSummary{
			ID:             houseID,
			HouseID:        houseID,
			Code:           luaTableTrimmedString(inTbl, "code"),
			NameShort:      luaTableTrimmedString(inTbl, "name_short"),
			Name:           luaTableTrimmedString(inTbl, "name"),
			ColorHex:       luaTableStringOr(inTbl, "color_hex", "colour_hex"),
			SequenceNumber: luaTableInt(inTbl, "sequence_number", 0),
		})
		if err != nil {
			L.RaiseError("dal.groups.upsert_house failed: %s", err.Error())
			return 0
		}
		L.Push(houseToLuaTable(L, *hs))
		return 1
	}))

	L.SetField(groupsTbl, "upsert_room", L.NewFunction(func(L *lua.LState) int {
		inTbl, scope, ok := parseGroupWriteArgs(L, execCtx, b.coreDAL, endpointSpec, caller, groupsWriteMode, "upsert_room")
		if !ok {
			return 0
		}
		isActive := true
		if v := inTbl.RawGetString("is_active"); v != lua.LNil {
			isActive = v == lua.LTrue
		}
		rm, err := b.coreDAL.UpsertRoomScoped(execCtx, scope, dal.RoomSummary{
			ID:             luaTableStringOr(inTbl, "room_id", "id"),
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
		L.Push(roomToLuaTable(L, *rm))
		return 1
	}))

	L.SetField(groupsTbl, "upsert_department", L.NewFunction(func(L *lua.LState) int {
		inTbl, scope, ok := parseGroupWriteArgs(L, execCtx, b.coreDAL, endpointSpec, caller, groupsWriteMode, "upsert_department")
		if !ok {
			return 0
		}
		dep, err := b.coreDAL.UpsertDepartmentScoped(execCtx, scope, dal.DepartmentSummary{
			ID:                      luaTableStringOr(inTbl, "department_id", "id"),
			Name:                    luaTableTrimmedString(inTbl, "name"),
			NameShort:               luaTableTrimmedString(inTbl, "name_short"),
			Type:                    luaTableTrimmedString(inTbl, "type"),
			HeadOfDepartmentStaffID: luaTableTrimmedString(inTbl, "head_of_department_staff_id"),
		})
		if err != nil {
			L.RaiseError("dal.groups.upsert_department failed: %s", err.Error())
			return 0
		}
		L.Push(departmentToLuaTable(L, *dep))
		return 1
	}))

	L.SetField(groupsTbl, "upsert_course", L.NewFunction(func(L *lua.LState) int {
		inTbl, scope, ok := parseGroupWriteArgs(L, execCtx, b.coreDAL, endpointSpec, caller, groupsWriteMode, "upsert_course")
		if !ok {
			return 0
		}
		c, err := b.coreDAL.UpsertCourseScoped(execCtx, scope, dal.CourseSummary{
			ID:             luaTableStringOr(inTbl, "course_id", "id"),
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
		L.Push(courseToLuaTable(L, *c))
		return 1
	}))

	L.SetField(groupsTbl, "upsert_year_group", L.NewFunction(func(L *lua.LState) int {
		inTbl, scope, ok := parseGroupWriteArgs(L, execCtx, b.coreDAL, endpointSpec, caller, groupsWriteMode, "upsert_year_group")
		if !ok {
			return 0
		}
		yg, err := b.coreDAL.UpsertYearGroupScoped(execCtx, scope, dal.YearGroupSummary{
			ID:                luaTableStringOr(inTbl, "year_group_id", "id"),
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
		L.Push(yearGroupToLuaTable(L, *yg))
		return 1
	}))
}
