package dalbindings

import (
	"context"
	"strings"

	lua "github.com/yuin/gopher-lua"

	"optimis/internal/dal"
	"optimis/internal/host/types"
)

// ============================================================================
// Shared Generic & Lua Conversion Helpers
// ============================================================================

func resolveStrictOwnGroupsScope(
	ctx context.Context,
	coreDAL *dal.CoreDAL,
	baseMode dal.ScopeMode,
	staffID string,
	optsTbl *lua.LTable,
) (dal.DALQueryScope, error) {
	effectiveMode := dal.ScopeAll
	if optsTbl != nil && optsTbl.RawGetString("strict_own_groups") == lua.LTrue {
		effectiveMode = baseMode
	}
	return extractScopeFromLuaOpts(ctx, coreDAL, effectiveMode, staffID, optsTbl)
}

func parseGroupWriteArgs(
	L *lua.LState,
	execCtx context.Context,
	coreDAL *dal.CoreDAL,
	endpointSpec types.EndpointSpec,
	caller types.CallerContext,
	groupsWriteMode dal.ScopeMode,
	methodName string,
) (*lua.LTable, dal.DALQueryScope, bool) {
	if !endpointSpec.Mutates {
		L.RaiseError("dal.groups.%s is forbidden when endpoint mutates=false", methodName)
		return nil, dal.DALQueryScope{}, false
	}
	inTbl, ok := L.Get(1).(*lua.LTable)
	if !ok {
		L.RaiseError("dal.groups.%s requires an input table", methodName)
		return nil, dal.DALQueryScope{}, false
	}
	var optsTbl *lua.LTable
	if L.GetTop() >= 2 {
		optsTbl, _ = L.Get(2).(*lua.LTable)
	}
	scope, sErr := extractScopeFromLuaOpts(execCtx, coreDAL, groupsWriteMode, caller.StaffID, optsTbl)
	if sErr != nil {
		L.RaiseError("dal.groups.%s scope error: %s", methodName, sErr.Error())
		return nil, dal.DALQueryScope{}, false
	}
	return inTbl, scope, true
}

// ============================================================================
// Group & Roster Lua Table Entity Mappers
// ============================================================================

func groupToLuaTable(L *lua.LState, g dal.GroupSummary) *lua.LTable {
	gTbl := L.NewTable()
	L.SetField(gTbl, "id", lua.LString(g.ID))
	L.SetField(gTbl, "group_id", lua.LString(g.ID))
	L.SetField(gTbl, "academic_year_id", lua.LString(g.AcademicYearID))
	L.SetField(gTbl, "group_type", lua.LString(g.GroupType))
	L.SetField(gTbl, "code", lua.LString(g.Code))
	L.SetField(gTbl, "name", lua.LString(g.Name))
	L.SetField(gTbl, "default_room_id", lua.LString(g.DefaultRoomID))
	L.SetField(gTbl, "takes_attendance", lua.LBool(g.TakesAttendance))
	L.SetField(gTbl, "effective_from", lua.LString(g.EffectiveFrom))
	if g.EffectiveTo != nil {
		L.SetField(gTbl, "effective_to", lua.LString(*g.EffectiveTo))
	}
	return gTbl
}

func groupPupilMembershipToLuaTable(L *lua.LState, m dal.GroupPupilMembershipSummary) *lua.LTable {
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
	return mTbl
}

func upsertGroupPupilMembershipResultToLuaTable(L *lua.LState, res *dal.UpsertGroupPupilMembershipResult) *lua.LTable {
	resTbl := L.NewTable()
	L.SetField(resTbl, "membership_id", lua.LString(res.MembershipID))
	L.SetField(resTbl, "group_id", lua.LString(res.GroupID))
	L.SetField(resTbl, "pupil_id", lua.LString(res.PupilID))
	L.SetField(resTbl, "academic_year_id", lua.LString(res.AcademicYearID))
	L.SetField(resTbl, "effective_from", lua.LString(res.EffectiveFrom))
	L.SetField(resTbl, "effective_to", lua.LString(res.EffectiveTo))
	return resTbl
}

func upsertGroupStaffAssignmentResultToLuaTable(L *lua.LState, res *dal.UpsertGroupStaffAssignmentResult) *lua.LTable {
	resTbl := L.NewTable()
	L.SetField(resTbl, "assignment_id", lua.LString(res.AssignmentID))
	L.SetField(resTbl, "group_id", lua.LString(res.GroupID))
	L.SetField(resTbl, "staff_id", lua.LString(res.StaffID))
	L.SetField(resTbl, "role", lua.LString(res.Role))
	L.SetField(resTbl, "academic_year_id", lua.LString(res.AcademicYearID))
	L.SetField(resTbl, "effective_from", lua.LString(res.EffectiveFrom))
	L.SetField(resTbl, "effective_to", lua.LString(res.EffectiveTo))
	return resTbl
}

// ============================================================================
// Group & Roster Argument Extraction Helpers
// ============================================================================

func extractGroupIDAndOptsFromLua(L *lua.LState) (string, *lua.LTable) {
	if firstTbl, isTbl := L.Get(1).(*lua.LTable); isTbl {
		return luaTableStringOr(firstTbl, "group_id", "id"), firstTbl
	}
	groupID := strings.TrimSpace(L.OptString(1, ""))
	var optsTbl *lua.LTable
	if L.GetTop() >= 2 {
		optsTbl, _ = L.Get(2).(*lua.LTable)
	}
	return groupID, optsTbl
}

func extractPupilMembershipsArgs(L *lua.LState) (string, string, *lua.LTable) {
	var optsTbl *lua.LTable
	var pupilID, groupID string
	if L.GetTop() >= 1 {
		if tbl, isTbl := L.Get(1).(*lua.LTable); isTbl {
			optsTbl = tbl
			pupilID = luaTableStringOr(tbl, "pupil_id", "id")
			groupID = luaTableTrimmedString(tbl, "group_id")
		} else {
			pupilID = strings.TrimSpace(L.OptString(1, ""))
			if L.GetTop() >= 2 {
				optsTbl, _ = L.Get(2).(*lua.LTable)
			}
		}
	}
	return pupilID, groupID, optsTbl
}

// ============================================================================
// Main DAL Groups Registration
// ============================================================================

func (b *Binder) bindLuaDALGroups(
	L *lua.LState,
	execCtx context.Context,
	endpointSpec types.EndpointSpec,
	caller types.CallerContext,
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

	b.bindLuaDALGroupsGuards(L, execCtx, callerGroupGuardMode, caller, groupsTbl)

	if hasGroupsRead {
		groupsMode := resolveEffectiveDomainScopeMode(endpointSpec.Scopes, caller, "dal.groups.read")
		b.bindLuaDALGroupsRead(L, execCtx, groupsMode, caller, groupsTbl)
	}

	if hasGroupsWrite {
		groupsWriteMode := resolveEffectiveDomainScopeMode(endpointSpec.Scopes, caller, "dal.groups.write")
		b.bindLuaDALGroupsWrite(L, execCtx, endpointSpec, caller, groupsWriteMode, groupsTbl)
	}

	L.SetField(dalTbl, "groups", groupsTbl)
}

// ============================================================================
// Access Guard Bindings
// ============================================================================

func (b *Binder) bindLuaDALGroupsGuards(
	L *lua.LState,
	execCtx context.Context,
	callerGroupGuardMode dal.ScopeMode,
	caller types.CallerContext,
	groupsTbl *lua.LTable,
) {
	assertAssignedFn := L.NewFunction(func(L *lua.LState) int {
		groupID, optsTbl := extractGroupIDAndOptsFromLua(L)
		if groupID == "" {
			L.RaiseError("missing mandatory parameter: group_id")
			return 0
		}
		scope, sErr := extractScopeFromLuaOpts(execCtx, b.coreDAL, callerGroupGuardMode, caller.StaffID, optsTbl)
		if sErr != nil {
			L.RaiseError("dal.groups.assert_caller_assigned scope error: %s", sErr.Error())
			return 0
		}
		if err := b.coreDAL.EnsureCallerAssignedToGroupScoped(execCtx, scope, groupID); err != nil {
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
		scope, sErr := extractScopeFromLuaOpts(execCtx, b.coreDAL, callerGroupGuardMode, caller.StaffID, optsTbl)
		if sErr != nil {
			L.RaiseError("dal.groups.is_caller_assigned scope error: %s", sErr.Error())
			return 0
		}
		assigned, err := b.coreDAL.IsCallerAssignedToGroupScoped(execCtx, scope, groupID)
		if err != nil {
			L.RaiseError("dal.groups.is_caller_assigned failed: %s", err.Error())
			return 0
		}
		L.Push(lua.LBool(assigned))
		return 1
	}))
}

// ============================================================================
// Read Domain Router & Roster Bindings
// ============================================================================

func (b *Binder) bindLuaDALGroupsRead(
	L *lua.LState,
	execCtx context.Context,
	groupsMode dal.ScopeMode,
	caller types.CallerContext,
	groupsTbl *lua.LTable,
) {
	b.bindLuaDALGroupsRosterRead(L, execCtx, groupsMode, caller, groupsTbl)
	b.bindLuaDALGroupsTimetableRead(L, execCtx, groupsMode, caller, groupsTbl)
	b.bindLuaDALGroupsCatalogRead(L, execCtx, groupsTbl)
}

func (b *Binder) bindLuaDALGroupsRosterRead(
	L *lua.LState,
	execCtx context.Context,
	groupsMode dal.ScopeMode,
	caller types.CallerContext,
	groupsTbl *lua.LTable,
) {
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
				groupType = luaTableTrimmedString(optsTbl, "group_type")
				if forcedMode == nil && (optsTbl.RawGetString("include_all") == lua.LTrue || luaTableTrimmedString(optsTbl, "scope") == "all") {
					effectiveMode = dal.ScopeAll
				}
			}
			scope, sErr := extractScopeFromLuaOpts(execCtx, b.coreDAL, effectiveMode, caller.StaffID, optsTbl)
			if sErr != nil {
				L.RaiseError("dal.groups.list_groups scope error: %s", sErr.Error())
				return 0
			}
			groups, err := b.coreDAL.ListGroupsScoped(execCtx, scope, groupType)
			if err != nil {
				L.RaiseError("dal.groups.list_groups failed: %s", err.Error())
				return 0
			}
			return pushMappedSlice(L, groups, groupToLuaTable)
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
		scope, sErr := extractScopeFromLuaOpts(execCtx, b.coreDAL, dal.ScopeAll, caller.StaffID, optsTbl)
		if sErr != nil {
			L.RaiseError("dal.groups.list_group_pupils scope error: %s", sErr.Error())
			return 0
		}
		pupils, err := b.coreDAL.ListPupilsByGroupScoped(execCtx, scope, groupID)
		if err != nil {
			L.RaiseError("dal.groups.list_group_pupils failed: %s", err.Error())
			return 0
		}
		return pushMappedSlice(L, pupils, pupilToLuaTable)
	}))

	L.SetField(groupsTbl, "list_staff", L.NewFunction(func(L *lua.LState) int {
		staffList, err := b.coreDAL.ListActiveStaff(execCtx)
		if err != nil {
			L.RaiseError("dal.groups.list_staff failed: %s", err.Error())
			return 0
		}
		return pushMappedSlice(L, staffList, staffToLuaTable)
	}))

	L.SetField(groupsTbl, "list_pupil_memberships", L.NewFunction(func(L *lua.LState) int {
		pupilID, groupID, optsTbl := extractPupilMembershipsArgs(L)
		scope, sErr := resolveStrictOwnGroupsScope(execCtx, b.coreDAL, groupsMode, caller.StaffID, optsTbl)
		if sErr != nil {
			L.RaiseError("dal.groups.list_pupil_memberships scope error: %s", sErr.Error())
			return 0
		}
		items, err := b.coreDAL.ListGroupPupilMembershipsScoped(execCtx, scope, pupilID, groupID)
		if err != nil {
			L.RaiseError("dal.groups.list_pupil_memberships failed: %s", err.Error())
			return 0
		}
		return pushMappedSlice(L, items, groupPupilMembershipToLuaTable)
	}))
}

// ============================================================================
// Write Domain Router & Roster Bindings
// ============================================================================

func (b *Binder) bindLuaDALGroupsWrite(
	L *lua.LState,
	execCtx context.Context,
	endpointSpec types.EndpointSpec,
	caller types.CallerContext,
	groupsWriteMode dal.ScopeMode,
	groupsTbl *lua.LTable,
) {
	b.bindLuaDALGroupsRosterWrite(L, execCtx, endpointSpec, caller, groupsWriteMode, groupsTbl)
	b.bindLuaDALGroupsTimetableWrite(L, execCtx, endpointSpec, caller, groupsWriteMode, groupsTbl)
	b.bindLuaDALGroupsCatalogWrite(L, execCtx, endpointSpec, caller, groupsWriteMode, groupsTbl)
}

func (b *Binder) bindLuaDALGroupsRosterWrite(
	L *lua.LState,
	execCtx context.Context,
	endpointSpec types.EndpointSpec,
	caller types.CallerContext,
	groupsWriteMode dal.ScopeMode,
	groupsTbl *lua.LTable,
) {
	L.SetField(groupsTbl, "upsert_pupil_membership", L.NewFunction(func(L *lua.LState) int {
		inTbl, scope, ok := parseGroupWriteArgs(L, execCtx, b.coreDAL, endpointSpec, caller, groupsWriteMode, "upsert_pupil_membership")
		if !ok {
			return 0
		}
		res, err := b.coreDAL.UpsertGroupPupilMembershipScoped(execCtx, scope, dal.UpsertGroupPupilMembershipInput{
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
		L.Push(upsertGroupPupilMembershipResultToLuaTable(L, res))
		return 1
	}))

	L.SetField(groupsTbl, "upsert_staff_assignment", L.NewFunction(func(L *lua.LState) int {
		inTbl, scope, ok := parseGroupWriteArgs(L, execCtx, b.coreDAL, endpointSpec, caller, groupsWriteMode, "upsert_staff_assignment")
		if !ok {
			return 0
		}
		res, err := b.coreDAL.UpsertGroupStaffAssignmentScoped(execCtx, scope, dal.UpsertGroupStaffAssignmentInput{
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
		L.Push(upsertGroupStaffAssignmentResultToLuaTable(L, res))
		return 1
	}))

	revokeStaffAssignFn := L.NewFunction(func(L *lua.LState) int {
		inTbl, scope, ok := parseGroupWriteArgs(L, execCtx, b.coreDAL, endpointSpec, caller, groupsWriteMode, "revoke_staff_assignment")
		if !ok {
			return 0
		}
		deletedCount, err := b.coreDAL.RevokeGroupStaffAssignmentScoped(execCtx, scope, dal.RevokeGroupStaffAssignmentInput{
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
		inTbl, scope, ok := parseGroupWriteArgs(L, execCtx, b.coreDAL, endpointSpec, caller, groupsWriteMode, "upsert_group")
		if !ok {
			return 0
		}
		res, err := b.coreDAL.UpsertGroupScoped(execCtx, scope, dal.UpsertGroupInput{
			GroupID:         luaTableStringOr(inTbl, "group_id", "id"),
			AcademicYearID:  luaTableTrimmedString(inTbl, "academic_year_id"),
			GroupType:       luaTableTrimmedString(inTbl, "group_type"),
			Code:            luaTableTrimmedString(inTbl, "code"),
			Name:            luaTableTrimmedString(inTbl, "name"),
			YearGroupID:     luaTableTrimmedString(inTbl, "year_group_id"),
			CourseID:        luaTableTrimmedString(inTbl, "course_id"),
			DefaultRoomID:   luaTableTrimmedString(inTbl, "default_room_id"),
			TakesAttendance: luaTableOptBoolPtr(inTbl, "takes_attendance"),
			EffectiveFrom:   luaTableTrimmedString(inTbl, "effective_from"),
			EffectiveTo:     luaTableTrimmedString(inTbl, "effective_to"),
		})
		if err != nil {
			L.RaiseError("dal.groups.upsert_group failed: %s", err.Error())
			return 0
		}
		L.Push(groupToLuaTable(L, *res))
		return 1
	}))
}
