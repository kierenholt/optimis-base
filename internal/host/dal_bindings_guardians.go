package host

import (
	"context"
	"strings"

	lua "github.com/yuin/gopher-lua"

	"optimis/internal/dal"
)

func guardianToLuaTable(L *lua.LState, g dal.GuardianContactSummary) *lua.LTable {
	gTbl := L.NewTable()
	L.SetField(gTbl, "relationship_id", lua.LString(g.RelationshipID))
	L.SetField(gTbl, "pupil_id", lua.LString(g.PupilID))
	L.SetField(gTbl, "guardian_person_id", lua.LString(g.GuardianPersonID))
	L.SetField(gTbl, "relationship_code", lua.LString(g.RelationshipCode))
	L.SetField(gTbl, "first_name", lua.LString(g.FirstName))
	L.SetField(gTbl, "last_name", lua.LString(g.LastName))
	L.SetField(gTbl, "email", lua.LString(g.Email))
	L.SetField(gTbl, "phone_primary", lua.LString(g.PhonePrimary))
	L.SetField(gTbl, "has_parental_responsibility", lua.LBool(g.HasParentalResponsibility))
	L.SetField(gTbl, "contact_priority", lua.LNumber(g.ContactPriority))
	L.SetField(gTbl, "is_emergency_contact", lua.LBool(g.IsEmergencyContact))
	L.SetField(gTbl, "authorised_to_collect", lua.LBool(g.AuthorisedToCollect))
	L.SetField(gTbl, "receive_reports", lua.LBool(g.ReceiveReports))
	L.SetField(gTbl, "household_id", lua.LString(g.HouseholdID))
	L.SetField(gTbl, "household_name", lua.LString(g.HouseholdName))
	L.SetField(gTbl, "address_line_1", lua.LString(g.AddressLine1))
	L.SetField(gTbl, "town_city", lua.LString(g.TownCity))
	L.SetField(gTbl, "postcode", lua.LString(g.Postcode))
	L.SetField(gTbl, "is_confidential_address_redacted", lua.LBool(g.IsConfidentialAddressRedacted))
	return gTbl
}

func (h *Host) bindLuaDALGuardiansAndSafeguarding(
	L *lua.LState,
	execCtx context.Context,
	endpointSpec EndpointSpec,
	caller CallerContext,
	dalTbl *lua.LTable,
) {
	if hasDomainScope(endpointSpec.Scopes, "dal.guardians.read") {
		guardiansMode := resolveEffectiveDomainScopeMode(endpointSpec.Scopes, caller, "dal.guardians.read")
		guardiansTbl := L.NewTable()

		L.SetField(guardiansTbl, "list_for_pupil", L.NewFunction(func(L *lua.LState) int {
			pupilID := strings.TrimSpace(L.OptString(1, ""))
			if pupilID == "" {
				L.RaiseError("missing mandatory parameter: pupil_id")
				return 0
			}
			var optsTbl *lua.LTable
			if L.GetTop() >= 2 {
				optsTbl, _ = L.Get(2).(*lua.LTable)
			}
			var reqPR bool
			effectiveMode := dal.ScopeAll
			if optsTbl != nil {
				if prVal := optsTbl.RawGetString("require_parental_responsibility"); prVal == lua.LTrue {
					reqPR = true
				}
				if optsTbl.RawGetString("strict_own_groups") == lua.LTrue {
					effectiveMode = guardiansMode
				}
			}
			scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, effectiveMode, caller.StaffID, optsTbl)
			if sErr != nil {
				L.RaiseError("dal.guardians.list_for_pupil scope error: %s", sErr.Error())
				return 0
			}
			guardians, err := h.coreDAL.ListGuardiansForPupilScoped(execCtx, scope, pupilID, dal.GuardianFilterOptions{
				RequireParentalResponsibility: reqPR,
			})
			if err != nil {
				L.RaiseError("dal.guardians.list_for_pupil failed: %s", err.Error())
				return 0
			}
			arrTbl := L.NewTable()
			markLuaTableAsArray(L, arrTbl)
			for i, g := range guardians {
				arrTbl.RawSetInt(i+1, guardianToLuaTable(L, g))
			}
			L.Push(arrTbl)
			return 1
		}))

		listForPupilsBatchFn := L.NewFunction(func(L *lua.LState) int {
			pupilIDs, optsTbl, argErr := extractPupilIDsAndOptsFromLua(L)
			if argErr != nil {
				L.RaiseError("dal.guardians.list_for_pupils_batch %s", argErr.Error())
				return 0
			}
			var reqPR bool
			effectiveMode := dal.ScopeAll
			if optsTbl != nil {
				if prVal := optsTbl.RawGetString("require_parental_responsibility"); prVal == lua.LTrue {
					reqPR = true
				}
				if optsTbl.RawGetString("strict_own_groups") == lua.LTrue {
					effectiveMode = guardiansMode
				}
			}
			scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, effectiveMode, caller.StaffID, optsTbl)
			if sErr != nil {
				L.RaiseError("dal.guardians.list_for_pupils_batch scope error: %s", sErr.Error())
				return 0
			}
			batch, err := h.coreDAL.ListGuardiansForPupilsBatchScoped(execCtx, scope, pupilIDs, dal.GuardianFilterOptions{
				RequireParentalResponsibility: reqPR,
			})
			if err != nil {
				L.RaiseError("dal.guardians.list_for_pupils_batch failed: %s", err.Error())
				return 0
			}
			resTbl := L.NewTable()
			for pid, gList := range batch {
				arrTbl := L.NewTable()
				markLuaTableAsArray(L, arrTbl)
				for idx, g := range gList {
					arrTbl.RawSetInt(idx+1, guardianToLuaTable(L, g))
				}
				L.SetField(resTbl, pid, arrTbl)
			}
			L.Push(resTbl)
			return 1
		})
		L.SetField(guardiansTbl, "list_for_pupils_batch", listForPupilsBatchFn)
		L.SetField(guardiansTbl, "list_batch", listForPupilsBatchFn)

		L.SetField(dalTbl, "guardians", guardiansTbl)
	}

	// ctx.dal.safeguarding (Strictly write-only postbox; ZERO read/decrypt exposed to Lua)
	if hasDomainScope(endpointSpec.Scopes, "dal.safeguarding.create") {
		sfgTbl := L.NewTable()

		L.SetField(sfgTbl, "create_postbox", L.NewFunction(func(L *lua.LState) int {
			if !endpointSpec.Mutates {
				L.RaiseError("dal.safeguarding.create_postbox is forbidden when endpoint mutates=false")
				return 0
			}
			inTbl, ok := L.Get(1).(*lua.LTable)
			if !ok {
				L.RaiseError("dal.safeguarding.create_postbox requires an input table")
				return 0
			}

			if kmsErr := h.ensureCloudKMSAvailable(execCtx); kmsErr != nil {
				L.RaiseError("dal.safeguarding.create_postbox failed: %s", kmsErr.Error())
				return 0
			}

			receipt, err := h.coreDAL.CreateSafeguardingPostboxNote(execCtx, dal.CreateSafeguardingPostboxInput{
				PupilID:            luaTableTrimmedString(inTbl, "pupil_id"),
				ReportedByStaffID:  caller.StaffID,
				AssignedDSLStaffID: luaTableTrimmedString(inTbl, "assigned_dsl_staff_id"),
				ConcernCategory:    luaTableTrimmedString(inTbl, "concern_category"),
				IncidentDate:       luaTableTrimmedString(inTbl, "incident_date"),
				Summary:            luaTableTrimmedString(inTbl, "summary"),
				BodyMap:            luaTableTrimmedString(inTbl, "body_map"),
				VerbatimAccount:    luaTableTrimmedString(inTbl, "verbatim_account"),
				ActionTaken:        luaTableTrimmedString(inTbl, "action_taken"),
				RequestID:          caller.RequestID,
				ClientIP:           "127.0.0.1",
				HostKMSKeyHandle:   h.cfg.HostKMSKeyHandle,
			})
			if err != nil {
				L.RaiseError("dal.safeguarding.create_postbox failed: %s", err.Error())
				return 0
			}

			resTbl := L.NewTable()
			L.SetField(resTbl, "note_id", lua.LString(receipt.NoteID))
			L.SetField(resTbl, "status", lua.LString(receipt.Status))
			L.Push(resTbl)
			return 1
		}))

		L.SetField(dalTbl, "safeguarding", sfgTbl)
	}
}
