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
	hasGuardiansRead := hasDomainScope(endpointSpec.Scopes, "dal.guardians.read")
	hasGuardiansWrite := hasDomainScope(endpointSpec.Scopes, "dal.guardians.write")
	if hasGuardiansRead || hasGuardiansWrite {
		guardiansMode := resolveEffectiveDomainScopeMode(endpointSpec.Scopes, caller, "dal.guardians.read")
		guardiansWriteMode := resolveEffectiveDomainScopeMode(endpointSpec.Scopes, caller, "dal.guardians.write")
		guardiansTbl := L.NewTable()

		if hasGuardiansRead {
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

			L.SetField(guardiansTbl, "list_siblings_for_pupil", L.NewFunction(func(L *lua.LState) int {
				var pupilID string
				var optsTbl *lua.LTable
				if L.GetTop() >= 1 {
					if tbl, isTbl := L.Get(1).(*lua.LTable); isTbl {
						optsTbl = tbl
						pupilID = luaTableTrimmedString(tbl, "pupil_id")
					} else {
						pupilID = strings.TrimSpace(L.OptString(1, ""))
						if L.GetTop() >= 2 {
							optsTbl, _ = L.Get(2).(*lua.LTable)
						}
					}
				}
				if pupilID == "" {
					L.RaiseError("missing mandatory parameter: pupil_id")
					return 0
				}
				effectiveMode := dal.ScopeAll
				if optsTbl != nil && optsTbl.RawGetString("strict_own_groups") == lua.LTrue {
					effectiveMode = guardiansMode
				}
				scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, effectiveMode, caller.StaffID, optsTbl)
				if sErr != nil {
					L.RaiseError("dal.guardians.list_siblings_for_pupil scope error: %s", sErr.Error())
					return 0
				}
				siblings, err := h.coreDAL.ListSiblingsForPupilScoped(execCtx, scope, pupilID)
				if err != nil {
					L.RaiseError("dal.guardians.list_siblings_for_pupil failed: %s", err.Error())
					return 0
				}
				arrTbl := L.NewTable()
				markLuaTableAsArray(L, arrTbl)
				for i, sib := range siblings {
					sTbl := L.NewTable()
					L.SetField(sTbl, "pupil_id", lua.LString(sib.PupilID))
					L.SetField(sTbl, "upn", lua.LString(sib.UPN))
					L.SetField(sTbl, "first_name", lua.LString(sib.FirstName))
					L.SetField(sTbl, "last_name", lua.LString(sib.LastName))
					L.SetField(sTbl, "nc_year_group", lua.LString(sib.NCYearGroup))
					L.SetField(sTbl, "shared_household_id", lua.LString(sib.SharedHouseholdID))
					L.SetField(sTbl, "household_id", lua.LString(sib.SharedHouseholdID))
					L.SetField(sTbl, "shared_guardian_person_id", lua.LString(sib.SharedGuardianPersonID))
					arrTbl.RawSetInt(i+1, sTbl)
				}
				L.Push(arrTbl)
				return 1
			}))

			L.SetField(guardiansTbl, "list_households", L.NewFunction(func(L *lua.LState) int {
				var optsTbl *lua.LTable
				var pupilID string
				if L.GetTop() >= 1 {
					if tbl, isTbl := L.Get(1).(*lua.LTable); isTbl {
						optsTbl = tbl
						pupilID = luaTableTrimmedString(tbl, "pupil_id")
					} else {
						pupilID = strings.TrimSpace(L.OptString(1, ""))
						if L.GetTop() >= 2 {
							optsTbl, _ = L.Get(2).(*lua.LTable)
						}
					}
				}
				effectiveMode := dal.ScopeAll
				if optsTbl != nil && optsTbl.RawGetString("strict_own_groups") == lua.LTrue {
					effectiveMode = guardiansMode
				}
				scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, effectiveMode, caller.StaffID, optsTbl)
				if sErr != nil {
					L.RaiseError("dal.guardians.list_households scope error: %s", sErr.Error())
					return 0
				}
				households, err := h.coreDAL.ListHouseholdsScoped(execCtx, scope, pupilID)
				if err != nil {
					L.RaiseError("dal.guardians.list_households failed: %s", err.Error())
					return 0
				}
				arrTbl := L.NewTable()
				markLuaTableAsArray(L, arrTbl)
				for i, hh := range households {
					hTbl := L.NewTable()
					L.SetField(hTbl, "id", lua.LString(hh.ID))
					L.SetField(hTbl, "household_id", lua.LString(hh.HouseholdID))
					L.SetField(hTbl, "name", lua.LString(hh.Name))
					L.SetField(hTbl, "household_name", lua.LString(hh.HouseholdName))
					L.SetField(hTbl, "address_line_1", lua.LString(hh.AddressLine1))
					L.SetField(hTbl, "address_line_2", lua.LString(hh.AddressLine2))
					L.SetField(hTbl, "town_city", lua.LString(hh.TownCity))
					L.SetField(hTbl, "county", lua.LString(hh.County))
					L.SetField(hTbl, "postcode", lua.LString(hh.Postcode))
					L.SetField(hTbl, "is_confidential_address_redacted", lua.LBool(hh.IsConfidentialAddressRedacted))
					arrTbl.RawSetInt(i+1, hTbl)
				}
				L.Push(arrTbl)
				return 1
			}))
		}

		if hasGuardiansWrite {
			L.SetField(guardiansTbl, "upsert_guardian_contact", L.NewFunction(func(L *lua.LState) int {
				if !endpointSpec.Mutates {
					L.RaiseError("dal.guardians.upsert_guardian_contact is forbidden when endpoint mutates=false")
					return 0
				}
				inTbl, ok := L.Get(1).(*lua.LTable)
				if !ok {
					L.RaiseError("dal.guardians.upsert_guardian_contact requires an input table")
					return 0
				}
				var optsTbl *lua.LTable
				if L.GetTop() >= 2 {
					optsTbl, _ = L.Get(2).(*lua.LTable)
				}
				scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, guardiansWriteMode, caller.StaffID, optsTbl)
				if sErr != nil {
					L.RaiseError("dal.guardians.upsert_guardian_contact scope error: %s", sErr.Error())
					return 0
				}
				optBoolPtr := func(key string) *bool {
					v := inTbl.RawGetString(key)
					if v == lua.LNil {
						return nil
					}
					b := v == lua.LTrue
					return &b
				}
				var contactPriority int
				if cpVal := inTbl.RawGetString("contact_priority"); cpVal != lua.LNil {
					if n, isNum := cpVal.(lua.LNumber); isNum {
						contactPriority = int(n)
					}
				}
				relID := luaTableTrimmedString(inTbl, "relationship_id")
				if relID == "" {
					relID = luaTableTrimmedString(inTbl, "id")
				}
				gSummary, err := h.coreDAL.UpsertGuardianContactScoped(execCtx, scope, dal.UpsertGuardianContactInput{
					RelationshipID:            relID,
					PupilID:                   luaTableTrimmedString(inTbl, "pupil_id"),
					GuardianPersonID:          luaTableTrimmedString(inTbl, "guardian_person_id"),
					RelationshipCode:          luaTableTrimmedString(inTbl, "relationship_code"),
					FirstName:                 luaTableTrimmedString(inTbl, "first_name"),
					LastName:                  luaTableTrimmedString(inTbl, "last_name"),
					Email:                     luaTableTrimmedString(inTbl, "email"),
					PhonePrimary:              luaTableTrimmedString(inTbl, "phone_primary"),
					HasParentalResponsibility: inTbl.RawGetString("has_parental_responsibility") == lua.LTrue,
					ContactPriority:           contactPriority,
					IsEmergencyContact:        optBoolPtr("is_emergency_contact"),
					AuthorisedToCollect:       optBoolPtr("authorised_to_collect"),
					ReceiveReports:            optBoolPtr("receive_reports"),
					CourtOrderRestricted:      inTbl.RawGetString("court_order_restricted") == lua.LTrue,
					HouseholdID:               luaTableTrimmedString(inTbl, "household_id"),
					HouseholdName:             luaTableTrimmedString(inTbl, "household_name"),
					AddressLine1:              luaTableTrimmedString(inTbl, "address_line_1"),
					TownCity:                  luaTableTrimmedString(inTbl, "town_city"),
					Postcode:                  luaTableTrimmedString(inTbl, "postcode"),
					IsConfidentialAddress:     inTbl.RawGetString("is_confidential_address") == lua.LTrue,
					EffectiveFrom:             luaTableTrimmedString(inTbl, "effective_from"),
				})
				if err != nil {
					L.RaiseError("dal.guardians.upsert_guardian_contact failed: %s", err.Error())
					return 0
				}
				L.Push(guardianToLuaTable(L, *gSummary))
				return 1
			}))
		}

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
