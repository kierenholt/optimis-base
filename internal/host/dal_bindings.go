package host

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	lua "github.com/yuin/gopher-lua"

	"optimis/internal/dal"
)

var piiScrubPattern = regexp.MustCompile(`(?i)\b(pup_[a-z0-9_]+|per_[a-z0-9_]+|[A-Z][0-9]{12})\b`)

func scrubDiagnosticToken(raw string) string {
	return piiScrubPattern.ReplaceAllString(raw, "[REDACTED]")
}

func callerHasAllRole(activeRoles []string) bool {
	for _, r := range activeRoles {
		switch strings.ToUpper(strings.TrimSpace(r)) {
		case "ADMIN", "HOD", "DSL", "SUPPORT", "LEADERSHIP", "DPO", "SYSTEM_ADMIN":
			return true
		}
	}
	return false
}

func resolveEffectiveDomainScopeMode(endpointScopes []string, caller CallerContext, domainPrefix string) dal.ScopeMode {
	for _, s := range endpointScopes {
		if s == domainPrefix+":all" {
			return dal.ScopeAll
		}
	}
	if caller.Type == "USER" {
		hasAll := false
		hasOwn := false
		for _, s := range caller.Scopes {
			if s == domainPrefix+":all" {
				hasAll = true
			}
			if s == domainPrefix+":own_groups" {
				hasOwn = true
			}
		}
		if hasAll && (!hasOwn || callerHasAllRole(caller.ActiveRoles)) {
			return dal.ScopeAll
		}
	}
	return dal.ScopeOwnGroups
}

func resolveDomainScopeMode(scopes []string, domainPrefix string) dal.ScopeMode {
	for _, s := range scopes {
		if s == domainPrefix+":all" {
			return dal.ScopeAll
		}
	}
	return dal.ScopeOwnGroups
}

func extractScopeFromLuaOpts(
	ctx context.Context,
	d *dal.CoreDAL,
	mode dal.ScopeMode,
	staffID string,
	optsTbl *lua.LTable,
) (dal.DALQueryScope, error) {
	scope := dal.DALQueryScope{
		Mode:    mode,
		StaffID: staffID,
	}
	if optsTbl != nil {
		if ay := optsTbl.RawGetString("academic_year_id"); ay != lua.LNil {
			scope.AcademicYearID = strings.TrimSpace(ay.String())
		}
		if rd := optsTbl.RawGetString("reference_date"); rd != lua.LNil {
			scope.ReferenceDate = strings.TrimSpace(rd.String())
		}
	}
	return d.ResolveScopeDefaults(ctx, scope)
}

func extractPupilIDsAndOptsFromLua(L *lua.LState) ([]string, *lua.LTable, error) {
	firstTbl, ok := L.Get(1).(*lua.LTable)
	if !ok {
		return nil, nil, errors.New("requires a table of pupil_ids")
	}

	var optsTbl *lua.LTable
	if L.GetTop() >= 2 {
		optsTbl, _ = L.Get(2).(*lua.LTable)
	}

	idsSourceTbl := firstTbl
	if nested, isTbl := firstTbl.RawGetString("pupil_ids").(*lua.LTable); isTbl {
		idsSourceTbl = nested
		if optsTbl == nil {
			optsTbl = firstTbl
		}
	}

	var pupilIDs []string
	maxn := idsSourceTbl.MaxN()
	for i := 1; i <= maxn; i++ {
		val := strings.TrimSpace(idsSourceTbl.RawGetInt(i).String())
		if val != "" {
			pupilIDs = append(pupilIDs, val)
		}
	}
	return pupilIDs, optsTbl, nil
}

func pupilToLuaTable(L *lua.LState, pupil dal.Pupil) *lua.LTable {
	pTbl := L.NewTable()
	L.SetField(pTbl, "id", lua.LString(pupil.ID))
	L.SetField(pTbl, "pupil_id", lua.LString(pupil.ID))
	L.SetField(pTbl, "upn", lua.LString(pupil.UPN))
	L.SetField(pTbl, "first_name", lua.LString(pupil.FirstName))
	L.SetField(pTbl, "last_name", lua.LString(pupil.LastName))
	legFirst := pupil.LegalFirstName
	if legFirst == "" {
		legFirst = pupil.FirstName
	}
	legLast := pupil.LegalLastName
	if legLast == "" {
		legLast = pupil.LastName
	}
	prefFirst := pupil.PreferredFirstName
	if prefFirst == "" {
		prefFirst = legFirst
	}
	prefLast := pupil.PreferredLastName
	if prefLast == "" {
		prefLast = legLast
	}
	L.SetField(pTbl, "legal_first_name", lua.LString(legFirst))
	L.SetField(pTbl, "legal_last_name", lua.LString(legLast))
	L.SetField(pTbl, "preferred_first_name", lua.LString(prefFirst))
	L.SetField(pTbl, "preferred_last_name", lua.LString(prefLast))
	L.SetField(pTbl, "nc_year_group", lua.LString(pupil.NCYearGroup))
	enrStatus := pupil.EnrolmentStatus
	if enrStatus == "" {
		enrStatus = "C"
	}
	L.SetField(pTbl, "enrolment_status", lua.LString(enrStatus))
	L.SetField(pTbl, "academic_year_id", lua.LString(pupil.AcademicYearID))
	L.SetField(pTbl, "effective_from", lua.LString(pupil.EffectiveFrom))
	if pupil.EffectiveTo != nil {
		L.SetField(pTbl, "effective_to", lua.LString(*pupil.EffectiveTo))
	}
	return pTbl
}

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

func staffToLuaTable(L *lua.LState, st dal.StaffSummary) *lua.LTable {
	sTbl := L.NewTable()
	L.SetField(sTbl, "id", lua.LString(st.StaffID))
	L.SetField(sTbl, "staff_id", lua.LString(st.StaffID))
	L.SetField(sTbl, "person_id", lua.LString(st.PersonID))
	L.SetField(sTbl, "staff_code", lua.LString(st.StaffCode))
	L.SetField(sTbl, "display_name", lua.LString(st.DisplayName))
	L.SetField(sTbl, "first_name", lua.LString(st.FirstName))
	L.SetField(sTbl, "last_name", lua.LString(st.LastName))
	L.SetField(sTbl, "job_title", lua.LString(st.JobTitle))
	L.SetField(sTbl, "staff_type", lua.LString(st.StaffType))
	L.SetField(sTbl, "is_dsl", lua.LBool(st.IsDSL))
	return sTbl
}

func (h *Host) buildLuaDALTable(
	L *lua.LState,
	execCtx context.Context,
	endpointSpec EndpointSpec,
	caller CallerContext,
) *lua.LTable {
	dalTbl := L.NewTable()

	hasPupilsRead := hasDomainScope(endpointSpec.Scopes, "dal.pupils.read")
	hasSenRead := hasDomainScope(endpointSpec.Scopes, "dal.sen.read")
	hasMedicalRead := hasDomainScope(endpointSpec.Scopes, "dal.medical.read")

	// 1. ctx.dal.pupils (and SEN/Medical batch helper)
	if hasPupilsRead || hasSenRead || hasMedicalRead {
		pupilsMode := resolveEffectiveDomainScopeMode(endpointSpec.Scopes, caller, "dal.pupils.read")
		pupilsTbl := L.NewTable()

		if hasPupilsRead {
			L.SetField(pupilsTbl, "get_by_id", L.NewFunction(func(L *lua.LState) int {
				var pupilID string
				var optsTbl *lua.LTable
				if firstTbl, isTbl := L.Get(1).(*lua.LTable); isTbl {
					if v := firstTbl.RawGetString("pupil_id"); v != lua.LNil {
						pupilID = strings.TrimSpace(v.String())
					} else if v := firstTbl.RawGetString("id"); v != lua.LNil {
						pupilID = strings.TrimSpace(v.String())
					}
					optsTbl = firstTbl
				} else {
					pupilID = strings.TrimSpace(L.OptString(1, ""))
					if L.GetTop() >= 2 {
						optsTbl, _ = L.Get(2).(*lua.LTable)
					}
				}
				if pupilID == "" {
					L.RaiseError("missing mandatory parameter: pupil_id")
					return 0
				}

				// Relaxed student core profile visibility (1C): any teacher holding dal.pupils.read
				// can look up any student's core profile across the school unless strict_own_groups=true.
				effectiveMode := dal.ScopeAll
				if optsTbl != nil && optsTbl.RawGetString("strict_own_groups") == lua.LTrue {
					effectiveMode = pupilsMode
				}

				var pupil *dal.Pupil
				var err error
				if effectiveMode == dal.ScopeAll && optsTbl == nil {
					pupil, err = h.coreDAL.GetPupilByID(execCtx, pupilID)
				} else {
					scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, effectiveMode, caller.StaffID, optsTbl)
					if sErr != nil {
						L.RaiseError("dal.pupils.get_by_id scope error: %s", sErr.Error())
						return 0
					}
					pupil, err = h.coreDAL.GetPupilByIDScoped(execCtx, scope, pupilID)
				}
				if err != nil {
					L.RaiseError("dal.pupils.get_by_id failed: %s", err.Error())
					return 0
				}
				L.Push(pupilToLuaTable(L, *pupil))
				return 1
			}))

			getByIDsBatchFn := L.NewFunction(func(L *lua.LState) int {
				pupilIDs, optsTbl, argErr := extractPupilIDsAndOptsFromLua(L)
				if argErr != nil {
					L.RaiseError("dal.pupils.get_by_ids_batch %s", argErr.Error())
					return 0
				}
				effectiveMode := dal.ScopeAll
				if optsTbl != nil && optsTbl.RawGetString("strict_own_groups") == lua.LTrue {
					effectiveMode = pupilsMode
				}
				scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, effectiveMode, caller.StaffID, optsTbl)
				if sErr != nil {
					L.RaiseError("dal.pupils.get_by_ids_batch scope error: %s", sErr.Error())
					return 0
				}
				pupils, err := h.coreDAL.GetPupilsByIDsBatchScoped(execCtx, scope, pupilIDs)
				if err != nil {
					L.RaiseError("dal.pupils.get_by_ids_batch failed: %s", err.Error())
					return 0
				}
				arrTbl := L.NewTable()
				markLuaTableAsArray(L, arrTbl)
				for i, p := range pupils {
					arrTbl.RawSetInt(i+1, pupilToLuaTable(L, p))
				}
				L.Push(arrTbl)
				return 1
			})
			L.SetField(pupilsTbl, "get_by_ids_batch", getByIDsBatchFn)
			L.SetField(pupilsTbl, "get_batch", getByIDsBatchFn)

			L.SetField(pupilsTbl, "list_all", L.NewFunction(func(L *lua.LState) int {
				var optsTbl *lua.LTable
				if L.GetTop() >= 1 {
					optsTbl, _ = L.Get(1).(*lua.LTable)
				}
				scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, dal.ScopeAll, caller.StaffID, optsTbl)
				if sErr != nil {
					L.RaiseError("dal.pupils.list_all scope error: %s", sErr.Error())
					return 0
				}
				pupils, err := h.coreDAL.ListAllPupilsScoped(execCtx, scope)
				if err != nil {
					L.RaiseError("dal.pupils.list_all failed: %s", err.Error())
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

			L.SetField(pupilsTbl, "list_by_group", L.NewFunction(func(L *lua.LState) int {
				groupID := strings.TrimSpace(L.OptString(1, ""))
				if groupID == "" {
					L.RaiseError("missing mandatory parameter: group_id")
					return 0
				}
				var optsTbl *lua.LTable
				if L.GetTop() >= 2 {
					optsTbl, _ = L.Get(2).(*lua.LTable)
				}
				effectiveMode := pupilsMode
				if optsTbl != nil && (optsTbl.RawGetString("include_all_classes") == lua.LTrue || optsTbl.RawGetString("scope").String() == "all") {
					effectiveMode = dal.ScopeAll
				}
				scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, effectiveMode, caller.StaffID, optsTbl)
				if sErr != nil {
					L.RaiseError("dal.pupils.list_by_group scope error: %s", sErr.Error())
					return 0
				}
				pupils, err := h.coreDAL.ListPupilsByGroupScoped(execCtx, scope, groupID)
				if err != nil {
					L.RaiseError("dal.pupils.list_by_group failed: %s", err.Error())
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
		}

		senMedicalBatchFn := L.NewFunction(func(L *lua.LState) int {
			pupilIDs, optsTbl, argErr := extractPupilIDsAndOptsFromLua(L)
			if argErr != nil {
				L.RaiseError("dal.pupils.list_sen_and_medical_batch %s", argErr.Error())
				return 0
			}
			effectiveMode := dal.ScopeAll
			if optsTbl != nil && optsTbl.RawGetString("strict_own_groups") == lua.LTrue {
				effectiveMode = pupilsMode
			}
			scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, effectiveMode, caller.StaffID, optsTbl)
			if sErr != nil {
				L.RaiseError("dal.pupils.list_sen_and_medical_batch scope error: %s", sErr.Error())
				return 0
			}
			batch, err := h.coreDAL.ListPupilSenAndMedicalBatch(execCtx, scope, pupilIDs)
			if err != nil {
				L.RaiseError("dal.pupils.list_sen_and_medical_batch failed: %s", err.Error())
				return 0
			}
			includeSen := hasPupilsRead || hasSenRead
			includeMedical := hasPupilsRead || hasMedicalRead

			resTbl := L.NewTable()
			for pid, summary := range batch {
				sTbl := L.NewTable()
				L.SetField(sTbl, "pupil_id", lua.LString(summary.PupilID))
				if includeSen {
					L.SetField(sTbl, "sen_provision_code", lua.LString(summary.SenProvisionCode))
					L.SetField(sTbl, "primary_sen_need_code", lua.LString(summary.PrimarySenNeedCode))
					L.SetField(sTbl, "teaching_strategies", lua.LString(summary.TeachingStrategies))
				}
				if includeMedical {
					medsArr := L.NewTable()
					markLuaTableAsArray(L, medsArr)
					for idx, mc := range summary.MedicalConditions {
						mTbl := L.NewTable()
						L.SetField(mTbl, "id", lua.LString(mc.ID))
						L.SetField(mTbl, "condition_name", lua.LString(mc.ConditionName))
						L.SetField(mTbl, "condition_category", lua.LString(mc.ConditionCategory))
						L.SetField(mTbl, "alert_level_id", lua.LString(mc.AlertLevelID))
						L.SetField(mTbl, "has_ihcp", lua.LBool(mc.HasIHCP))
						L.SetField(mTbl, "emergency_response_plan", lua.LString(mc.EmergencyResponsePlan))
						medsArr.RawSetInt(idx+1, mTbl)
					}
					L.SetField(sTbl, "medical_conditions", medsArr)
				}
				L.SetField(resTbl, pid, sTbl)
			}
			L.Push(resTbl)
			return 1
		})

		L.SetField(pupilsTbl, "list_sen_and_medical_batch", senMedicalBatchFn)
		L.SetField(dalTbl, "pupils", pupilsTbl)

		if hasSenRead {
			senTbl := L.NewTable()
			L.SetField(senTbl, "list_batch", senMedicalBatchFn)
			L.SetField(dalTbl, "sen", senTbl)
		}
		if hasMedicalRead {
			medTbl := L.NewTable()
			L.SetField(medTbl, "list_batch", senMedicalBatchFn)
			L.SetField(dalTbl, "medical", medTbl)
		}
	}

	// 2. ctx.dal.groups
	hasGroupsRead := hasDomainScope(endpointSpec.Scopes, "dal.groups.read")
	hasGroupsWrite := hasDomainScope(endpointSpec.Scopes, "dal.groups.write")
	if hasGroupsRead || hasGroupsWrite {
		groupsTbl := L.NewTable()

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
				var groupID string
				var optsTbl *lua.LTable
				if firstTbl, isTbl := L.Get(1).(*lua.LTable); isTbl {
					if v := firstTbl.RawGetString("group_id"); v != lua.LNil {
						groupID = strings.TrimSpace(v.String())
					} else if v := firstTbl.RawGetString("id"); v != lua.LNil {
						groupID = strings.TrimSpace(v.String())
					}
					optsTbl = firstTbl
				} else {
					groupID = strings.TrimSpace(L.OptString(1, ""))
					if L.GetTop() >= 2 {
						optsTbl, _ = L.Get(2).(*lua.LTable)
					}
				}
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
					arrTbl.RawSetInt(i+1, sTbl)
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
				getStr := func(key string) string {
					v := inTbl.RawGetString(key)
					if v == lua.LNil {
						return ""
					}
					return strings.TrimSpace(v.String())
				}
				res, err := h.coreDAL.UpsertGroupPupilMembershipScoped(execCtx, scope, dal.UpsertGroupPupilMembershipInput{
					MembershipID:   getStr("membership_id"),
					GroupID:        getStr("group_id"),
					PupilID:        getStr("pupil_id"),
					AcademicYearID: getStr("academic_year_id"),
					EffectiveFrom:  getStr("effective_from"),
					EffectiveTo:    getStr("effective_to"),
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
		}

		L.SetField(dalTbl, "groups", groupsTbl)
	}

	// 2b. ctx.dal.staff (Staff directory read)
	if hasDomainScope(endpointSpec.Scopes, "dal.staff.read") {
		staffTbl := L.NewTable()
		L.SetField(staffTbl, "list_active", L.NewFunction(func(L *lua.LState) int {
			staffList, err := h.coreDAL.ListActiveStaff(execCtx)
			if err != nil {
				L.RaiseError("dal.staff.list_active failed: %s", err.Error())
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
		L.SetField(staffTbl, "get_by_id", L.NewFunction(func(L *lua.LState) int {
			staffID := strings.TrimSpace(L.OptString(1, ""))
			if staffID == "" {
				L.RaiseError("missing mandatory parameter: staff_id")
				return 0
			}
			st, err := h.coreDAL.GetStaffByID(execCtx, staffID)
			if err != nil {
				L.RaiseError("dal.staff.get_by_id failed: %s", err.Error())
				return 0
			}
			L.Push(staffToLuaTable(L, *st))
			return 1
		}))
		L.SetField(dalTbl, "staff", staffTbl)
	}

	// 3. ctx.dal.attendance
	hasAttRead := hasDomainScope(endpointSpec.Scopes, "dal.attendance.read")
	hasAttWrite := hasDomainScope(endpointSpec.Scopes, "dal.attendance.write")
	if hasAttRead || hasAttWrite {
		attTbl := L.NewTable()

		if hasAttRead {
			attReadMode := resolveEffectiveDomainScopeMode(endpointSpec.Scopes, caller, "dal.attendance.read")
			L.SetField(attTbl, "get_group_session", L.NewFunction(func(L *lua.LState) int {
				groupID := strings.TrimSpace(L.OptString(1, ""))
				date := strings.TrimSpace(L.OptString(2, ""))
				session := strings.TrimSpace(L.OptString(3, ""))
				var optsTbl *lua.LTable
				if L.GetTop() >= 4 {
					optsTbl, _ = L.Get(4).(*lua.LTable)
				}
				scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, attReadMode, caller.StaffID, optsTbl)
				if sErr != nil {
					L.RaiseError("dal.attendance.get_group_session scope error: %s", sErr.Error())
					return 0
				}
				marks, err := h.coreDAL.GetGroupSessionAttendanceScoped(execCtx, scope, groupID, date, session)
				if err != nil {
					L.RaiseError("dal.attendance.get_group_session failed: %s", err.Error())
					return 0
				}
				arrTbl := L.NewTable()
				markLuaTableAsArray(L, arrTbl)
				for i, m := range marks {
					mTbl := L.NewTable()
					L.SetField(mTbl, "id", lua.LString(m.ID))
					L.SetField(mTbl, "pupil_id", lua.LString(m.PupilID))
					L.SetField(mTbl, "academic_year_id", lua.LString(m.AcademicYearID))
					L.SetField(mTbl, "date", lua.LString(m.Date))
					L.SetField(mTbl, "session", lua.LString(m.Session))
					L.SetField(mTbl, "attendance_code", lua.LString(m.AttendanceCode))
					L.SetField(mTbl, "minutes_late", lua.LNumber(m.MinutesLate))
					L.SetField(mTbl, "sub_code", lua.LString(m.SubCode))
					L.SetField(mTbl, "comment", lua.LString(m.Comment))
					L.SetField(mTbl, "source_group_id", lua.LString(m.SourceGroupID))
					L.SetField(mTbl, "recorded_by_staff_id", lua.LString(m.RecordedByStaffID))
					arrTbl.RawSetInt(i+1, mTbl)
				}
				L.Push(arrTbl)
				return 1
			}))

			L.SetField(attTbl, "get_pupil_summary", L.NewFunction(func(L *lua.LState) int {
				pupilID := strings.TrimSpace(L.OptString(1, ""))
				if pupilID == "" {
					L.RaiseError("missing mandatory parameter: pupil_id")
					return 0
				}
				var optsTbl *lua.LTable
				if L.GetTop() >= 2 {
					optsTbl, _ = L.Get(2).(*lua.LTable)
				}
				scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, attReadMode, caller.StaffID, optsTbl)
				if sErr != nil {
					L.RaiseError("dal.attendance.get_pupil_summary scope error: %s", sErr.Error())
					return 0
				}
				summary, err := h.coreDAL.GetPupilAttendanceSummaryScoped(execCtx, scope, pupilID)
				if err != nil {
					L.RaiseError("dal.attendance.get_pupil_summary failed: %s", err.Error())
					return 0
				}
				sTbl := L.NewTable()
				L.SetField(sTbl, "pupil_id", lua.LString(summary.PupilID))
				L.SetField(sTbl, "academic_year_id", lua.LString(summary.AcademicYearID))
				L.SetField(sTbl, "reference_date", lua.LString(summary.ReferenceDate))
				L.SetField(sTbl, "total_sessions", lua.LNumber(summary.TotalSessions))
				L.SetField(sTbl, "present_sessions", lua.LNumber(summary.PresentSessions))
				L.SetField(sTbl, "am_present_sessions", lua.LNumber(summary.AMPresentSessions))
				L.SetField(sTbl, "pm_present_sessions", lua.LNumber(summary.PMPresentSessions))
				L.Push(sTbl)
				return 1
			}))
		}

		if hasAttWrite {
			attWriteMode := resolveEffectiveDomainScopeMode(endpointSpec.Scopes, caller, "dal.attendance.write")
			upsertSessionBatchFn := L.NewFunction(func(L *lua.LState) int {
				if !endpointSpec.Mutates {
					L.RaiseError("dal.attendance.upsert_session_batch is forbidden when endpoint mutates=false")
					return 0
				}
				var groupID, date, session string
				var marksTbl, optsTbl *lua.LTable
				if firstTbl, isTbl := L.Get(1).(*lua.LTable); isTbl {
					if v := firstTbl.RawGetString("group_id"); v != lua.LNil {
						groupID = strings.TrimSpace(v.String())
					}
					if v := firstTbl.RawGetString("date"); v != lua.LNil {
						date = strings.TrimSpace(v.String())
					}
					if v := firstTbl.RawGetString("session"); v != lua.LNil {
						session = strings.TrimSpace(v.String())
					}
					if m, ok := firstTbl.RawGetString("marks").(*lua.LTable); ok {
						marksTbl = m
					}
					if L.GetTop() >= 2 {
						optsTbl, _ = L.Get(2).(*lua.LTable)
					} else {
						optsTbl = firstTbl
					}
				} else {
					groupID = strings.TrimSpace(L.OptString(1, ""))
					date = strings.TrimSpace(L.OptString(2, ""))
					session = strings.TrimSpace(L.OptString(3, ""))
					var ok bool
					marksTbl, ok = L.Get(4).(*lua.LTable)
					if !ok {
						L.RaiseError("dal.attendance.upsert_session_batch requires a marks table as 4th argument")
						return 0
					}
					if L.GetTop() >= 5 {
						optsTbl, _ = L.Get(5).(*lua.LTable)
					}
				}
				if marksTbl == nil {
					L.RaiseError("dal.attendance.upsert_session_batch requires a marks table")
					return 0
				}
				scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, attWriteMode, caller.StaffID, optsTbl)
				if sErr != nil {
					L.RaiseError("dal.attendance.upsert_session_batch scope error: %s", sErr.Error())
					return 0
				}
				if strings.TrimSpace(date) != "" && (optsTbl == nil || optsTbl.RawGetString("reference_date") == lua.LNil) {
					scope.ReferenceDate = date
				}

				maxn := marksTbl.MaxN()
				marks := make([]dal.SessionAttendanceMarkInput, 0, maxn)
				for i := 1; i <= maxn; i++ {
					itemTbl, ok := marksTbl.RawGetInt(i).(*lua.LTable)
					if !ok {
						L.RaiseError("invalid mark entry at index %d", i)
						return 0
					}
					var minLate int
					if ml := itemTbl.RawGetString("minutes_late"); ml != lua.LNil {
						if num, ok := ml.(lua.LNumber); ok {
							minLate = int(num)
						}
					}
					var subCode, comment string
					if sc := itemTbl.RawGetString("sub_code"); sc != lua.LNil {
						subCode = sc.String()
					}
					if cm := itemTbl.RawGetString("comment"); cm != lua.LNil {
						comment = cm.String()
					}
					marks = append(marks, dal.SessionAttendanceMarkInput{
						PupilID:        strings.TrimSpace(itemTbl.RawGetString("pupil_id").String()),
						AttendanceCode: strings.TrimSpace(itemTbl.RawGetString("attendance_code").String()),
						MinutesLate:    minLate,
						SubCode:        subCode,
						Comment:        comment,
					})
				}

				res, err := h.coreDAL.UpsertSessionAttendanceBatchScoped(execCtx, scope, dal.UpsertSessionAttendanceBatchInput{
					GroupID: groupID,
					Date:    date,
					Session: session,
					Marks:   marks,
				})
				if err != nil {
					L.RaiseError("dal.attendance.upsert_session_batch failed: %s", err.Error())
					return 0
				}

				resTbl := L.NewTable()
				L.SetField(resTbl, "submission_id", lua.LString(res.SubmissionID))
				L.SetField(resTbl, "upserted_count", lua.LNumber(res.UpsertedCount))
				L.SetField(resTbl, "group_id", lua.LString(res.GroupID))
				L.SetField(resTbl, "date", lua.LString(res.Date))
				L.SetField(resTbl, "session", lua.LString(res.Session))
				L.Push(resTbl)
				return 1
			})
			L.SetField(attTbl, "record_session_batch", upsertSessionBatchFn)
			L.SetField(attTbl, "upsert_session_batch", upsertSessionBatchFn)
		}

		L.SetField(dalTbl, "attendance", attTbl)
	}

	// 4. ctx.dal.guardians
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
			if optsTbl != nil {
				if prVal := optsTbl.RawGetString("require_parental_responsibility"); prVal == lua.LTrue {
					reqPR = true
				}
			}
			scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, guardiansMode, caller.StaffID, optsTbl)
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
			if optsTbl != nil {
				if prVal := optsTbl.RawGetString("require_parental_responsibility"); prVal == lua.LTrue {
					reqPR = true
				}
			}
			scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, guardiansMode, caller.StaffID, optsTbl)
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

	// 5. ctx.dal.safeguarding (Strictly write-only postbox; ZERO read/decrypt exposed to Lua)
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
			getStr := func(key string) string {
				v := inTbl.RawGetString(key)
				if v == lua.LNil {
					return ""
				}
				return strings.TrimSpace(v.String())
			}

			if kmsErr := h.ensureCloudKMSAvailable(execCtx); kmsErr != nil {
				L.RaiseError("dal.safeguarding.create_postbox failed: %s", kmsErr.Error())
				return 0
			}

			receipt, err := h.coreDAL.CreateSafeguardingPostboxNote(execCtx, dal.CreateSafeguardingPostboxInput{
				PupilID:            getStr("pupil_id"),
				ReportedByStaffID:  caller.StaffID,
				AssignedDSLStaffID: getStr("assigned_dsl_staff_id"),
				ConcernCategory:    getStr("concern_category"),
				IncidentDate:       getStr("incident_date"),
				Summary:            getStr("summary"),
				BodyMap:            getStr("body_map"),
				VerbatimAccount:    getStr("verbatim_account"),
				ActionTaken:        getStr("action_taken"),
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

	// 6. ctx.dal.census (Statutory DfE School Census Rollup; strictly requires dal.census.read:all)
	if hasDomainScope(endpointSpec.Scopes, "dal.census.read") {
		censusMode := resolveDomainScopeMode(endpointSpec.Scopes, "dal.census.read")
		censusTbl := L.NewTable()

		L.SetField(censusTbl, "get_statutory_rollup", L.NewFunction(func(L *lua.LState) int {
			inTbl, ok := L.Get(1).(*lua.LTable)
			if !ok {
				L.RaiseError("dal.census.get_statutory_rollup requires an input options table")
				return 0
			}
			getStr := func(key string) string {
				v := inTbl.RawGetString(key)
				if v == lua.LNil {
					return ""
				}
				return strings.TrimSpace(v.String())
			}

			rollup, err := h.coreDAL.GetStatutoryCensusRollup(execCtx, dal.StatutoryCensusRollupInput{
				AcademicYearID:      getStr("academic_year_id"),
				CensusDate:          getStr("census_date"),
				TermCode:            getStr("term_code"),
				AttendanceStartDate: getStr("attendance_start_date"),
				AttendanceEndDate:   getStr("attendance_end_date"),
				Scope:               censusMode,
			})
			if err != nil {
				L.RaiseError("dal.census.get_statutory_rollup failed: %s", err.Error())
				return 0
			}

			outTbl := L.NewTable()
			L.SetField(outTbl, "school_urn", lua.LString(rollup.SchoolURN))
			L.SetField(outTbl, "school_name", lua.LString(rollup.SchoolName))
			L.SetField(outTbl, "academic_year_id", lua.LString(rollup.AcademicYearID))
			L.SetField(outTbl, "term_code", lua.LString(rollup.TermCode))
			L.SetField(outTbl, "census_date", lua.LString(rollup.CensusDate))

			sumTbl := L.NewTable()
			L.SetField(sumTbl, "on_roll_total", lua.LNumber(rollup.Summary.OnRollTotal))
			senCountsTbl := L.NewTable()
			for k, v := range rollup.Summary.SENCounts {
				L.SetField(senCountsTbl, k, lua.LNumber(v))
			}
			L.SetField(sumTbl, "sen_counts", senCountsTbl)
			L.SetField(sumTbl, "fsm_eligible_count", lua.LNumber(rollup.Summary.FSMEligibleCount))
			L.SetField(sumTbl, "lac_count", lua.LNumber(rollup.Summary.LACCount))
			L.SetField(sumTbl, "total_possible_sessions", lua.LNumber(rollup.Summary.TotalPossibleSessions))
			L.SetField(sumTbl, "total_present_sessions", lua.LNumber(rollup.Summary.TotalPresentSessions))
			L.SetField(sumTbl, "total_authorised_absence_sessions", lua.LNumber(rollup.Summary.TotalAuthorisedAbsenceSessions))
			L.SetField(sumTbl, "total_unauthorised_absence_sessions", lua.LNumber(rollup.Summary.TotalUnauthorisedAbsenceSessions))
			L.SetField(outTbl, "summary", sumTbl)

			pupilsArr := L.NewTable()
			markLuaTableAsArray(L, pupilsArr)
			for idx, p := range rollup.Pupils {
				pTbl := L.NewTable()
				L.SetField(pTbl, "pupil_id", lua.LString(p.PupilID))
				L.SetField(pTbl, "upn", lua.LString(p.UPN))
				L.SetField(pTbl, "uln", lua.LString(p.ULN))
				L.SetField(pTbl, "first_name", lua.LString(p.FirstName))
				L.SetField(pTbl, "last_name", lua.LString(p.LastName))
				L.SetField(pTbl, "dob", lua.LString(p.DOB))
				L.SetField(pTbl, "sex", lua.LString(p.Sex))
				L.SetField(pTbl, "ethnicity_code", lua.LString(p.EthnicityCode))
				L.SetField(pTbl, "first_language_code", lua.LString(p.FirstLanguageCode))
				L.SetField(pTbl, "nc_year_group", lua.LString(p.NCYearGroup))
				L.SetField(pTbl, "enrolment_status", lua.LString(p.EnrolmentStatus))
				L.SetField(pTbl, "effective_from", lua.LString(p.EffectiveFrom))
				L.SetField(pTbl, "effective_to", lua.LString(p.EffectiveTo))
				L.SetField(pTbl, "sen_provision_code", lua.LString(p.SENProvisionCode))
				L.SetField(pTbl, "primary_sen_need_code", lua.LString(p.PrimarySENNeedCode))
				L.SetField(pTbl, "fsm_eligible", lua.LBool(p.FSMEligible))
				L.SetField(pTbl, "looked_after_flag", lua.LBool(p.LookedAfterFlag))
				L.SetField(pTbl, "post_looked_after_code", lua.LString(p.PostLookedAfterCode))
				L.SetField(pTbl, "lac_episode_status", lua.LString(p.LACEpisodeStatus))

				fsmArr := L.NewTable()
				markLuaTableAsArray(L, fsmArr)
				for fIdx, fp := range p.FSMPeriods {
					fpTbl := L.NewTable()
					L.SetField(fpTbl, "id", lua.LString(fp.ID))
					L.SetField(fpTbl, "effective_from", lua.LString(fp.EffectiveFrom))
					L.SetField(fpTbl, "effective_to", lua.LString(fp.EffectiveTo))
					L.SetField(fpTbl, "country_code", lua.LString(fp.CountryCode))
					fsmArr.RawSetInt(fIdx+1, fpTbl)
				}
				L.SetField(pTbl, "fsm_periods", fsmArr)

				attSessionsTbl := L.NewTable()
				L.SetField(attSessionsTbl, "possible_sessions", lua.LNumber(p.AttendanceSessions.PossibleSessions))
				L.SetField(attSessionsTbl, "present_sessions", lua.LNumber(p.AttendanceSessions.PresentSessions))
				L.SetField(attSessionsTbl, "authorised_absence_sessions", lua.LNumber(p.AttendanceSessions.AuthorisedAbsenceSessions))
				L.SetField(attSessionsTbl, "unauthorised_absence_sessions", lua.LNumber(p.AttendanceSessions.UnauthorisedAbsenceSessions))
				L.SetField(attSessionsTbl, "am_present", lua.LNumber(p.AttendanceSessions.AMPresent))
				L.SetField(attSessionsTbl, "pm_present", lua.LNumber(p.AttendanceSessions.PMPresent))
				L.SetField(pTbl, "attendance_sessions", attSessionsTbl)

				pupilsArr.RawSetInt(idx+1, pTbl)
			}
			L.SetField(outTbl, "pupils", pupilsArr)

			L.Push(outTbl)
			return 1
		}))

		L.SetField(dalTbl, "census", censusTbl)
	}

	return dalTbl
}

func (h *Host) ExecuteWorker(
	ctx context.Context,
	pluginID string,
	workerName string,
	args map[string]any,
) (*RPCResponse, error) {
	start := time.Now()

	if strings.TrimSpace(pluginID) == "" {
		return nil, errors.New("missing mandatory parameter: pluginID")
	}
	if strings.TrimSpace(workerName) == "" {
		return nil, errors.New("missing mandatory parameter: workerName")
	}

	plugin, exists := h.GetLoadedPlugin(pluginID)
	if !exists {
		h.logError(fmt.Sprintf("Plugin not found for worker execution: %s", pluginID))
		return nil, fmt.Errorf("plugin not loaded: %s", pluginID)
	}

	workerSpec, declared := plugin.Manifest.Workers[workerName]
	if plugin.State == PluginStateUninstalledDormant {
		h.recordDiagnostic(pluginID, workerName, "PLUGIN_DORMANT_QUARANTINE", time.Since(start))
		return nil, fmt.Errorf("PLUGIN_DORMANT_QUARANTINE: plugin %s is uninstalled and in dormant GDPR retention quarantine", pluginID)
	}
	if !declared {
		h.recordDiagnostic(pluginID, workerName, "UNDECLARED_WORKER", time.Since(start))
		return nil, fmt.Errorf("worker %q is not declared in manifest.json for plugin %s", workerName, pluginID)
	}

	timeoutMS := workerSpec.TimeoutMS
	if timeoutMS <= 0 {
		timeoutMS = 30000
	}
	execCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMS)*time.Millisecond)
	defer cancel()

	caller := CallerContext{
		Type:        "SYSTEM",
		UserID:      "system:" + pluginID,
		StaffID:     "",
		ActiveRoles: []string{"system_worker"},
		Scopes:      workerSpec.Scopes,
		RequestID:   "wrk_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12],
	}

	L := lua.NewState(lua.Options{SkipOpenLibs: true})
	defer L.Close()
	for _, lib := range []struct {
		name string
		fn   lua.LGFunction
	}{
		{lua.LoadLibName, lua.OpenPackage},
		{lua.BaseLibName, lua.OpenBase},
		{lua.TabLibName, lua.OpenTable},
		{lua.StringLibName, lua.OpenString},
		{lua.MathLibName, lua.OpenMath},
	} {
		L.Push(L.NewFunction(lib.fn))
		L.Push(lua.LString(lib.name))
		L.Call(1, 0)
	}
	L.SetGlobal("dofile", lua.LNil)
	L.SetGlobal("loadfile", lua.LNil)
	L.SetGlobal("package", lua.LNil)
	L.SetGlobal("require", lua.LNil)
	L.SetGlobal("os", lua.LNil)
	L.SetGlobal("io", lua.LNil)
	L.SetGlobal("debug", lua.LNil)
	L.SetContext(execCtx)

	var bufferedEvents []Event
	var writeTx *sql.Tx
	lockTracker := &sqlLockTracker{}
	argSecrets := collectArgSecretStrings(args)
	defer func() {
		if writeTx != nil {
			_ = writeTx.Rollback()
			lockTracker.finishHold()
		}
	}()

	ctxTable := h.buildLuaCtxTable(L, execCtx, plugin, workerSpec, caller, &bufferedEvents, &writeTx, lockTracker)
	argsTable := goMapToLuaTable(L, args)

	if err := L.DoString(plugin.LuaSource); err != nil {
		h.recordDiagnosticDetailed(pluginID, workerName, "LUA_LOAD_ERROR", err, time.Since(start), lockTracker, argSecrets)
		return nil, fmt.Errorf("failed to load plugin Lua script: %w", err)
	}

	fn := L.GetGlobal(workerName)
	if fn.Type() != lua.LTFunction {
		h.recordDiagnosticDetailed(pluginID, workerName, "MISSING_LUA_FUNCTION", nil, time.Since(start), lockTracker, argSecrets)
		return nil, fmt.Errorf("declared worker %q is not a function in init.lua", workerName)
	}

	if err := L.CallByParam(lua.P{
		Fn:      fn,
		NRet:    1,
		Protect: true,
	}, ctxTable, argsTable); err != nil {
		h.recordDiagnosticDetailed(pluginID, workerName, "LUA_RUNTIME_ERROR", err, time.Since(start), lockTracker, argSecrets)
		return nil, fmt.Errorf("lua worker execution error in %s.%s: %w", pluginID, workerName, err)
	}

	ret := L.Get(-1)
	L.Pop(1)

	retTbl, ok := ret.(*lua.LTable)
	if !ok {
		h.recordDiagnosticDetailed(pluginID, workerName, "INVALID_RETURN_TYPE", nil, time.Since(start), lockTracker, argSecrets)
		return nil, fmt.Errorf("worker %s.%s must return a table", pluginID, workerName)
	}

	okVal := L.GetField(retTbl, "ok")
	if okVal != lua.LTrue {
		if writeTx != nil {
			_ = writeTx.Rollback()
			writeTx = nil
			lockTracker.finishHold()
		}
		errCode := L.GetField(retTbl, "error_code").String()
		return &RPCResponse{
			OK:        false,
			ErrorCode: errCode,
		}, nil
	}

	if writeTx != nil {
		commitErr := writeTx.Commit()
		writeTx = nil
		lockTracker.finishHold()
		if commitErr != nil {
			h.recordDiagnosticDetailed(pluginID, workerName, "SQLITE_COMMIT_ERROR", commitErr, time.Since(start), lockTracker, argSecrets)
			return nil, fmt.Errorf("failed to commit plugin_db write transaction in worker: %w", commitErr)
		}
	}

	dataVal := luaValueToGo(L.GetField(retTbl, "data"))
	if len(bufferedEvents) > 0 {
		h.dispatchEvents(bufferedEvents)
	}

	return &RPCResponse{
		OK:   true,
		Data: dataVal,
	}, nil
}
