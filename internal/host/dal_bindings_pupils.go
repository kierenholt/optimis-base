package host

import (
	"context"
	"strings"

	lua "github.com/yuin/gopher-lua"

	"optimis/internal/dal"
)

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

func (h *Host) bindLuaDALPupils(
	L *lua.LState,
	execCtx context.Context,
	endpointSpec EndpointSpec,
	caller CallerContext,
	dalTbl *lua.LTable,
) {
	hasPupilsRead := hasDomainScope(endpointSpec.Scopes, "dal.pupils.read")
	hasSenRead := hasDomainScope(endpointSpec.Scopes, "dal.sen.read")
	hasMedicalRead := hasDomainScope(endpointSpec.Scopes, "dal.medical.read")

	if !hasPupilsRead && !hasSenRead && !hasMedicalRead {
		return
	}

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

	senMode := resolveEffectiveDomainScopeMode(endpointSpec.Scopes, caller, "dal.sen.read")
	medMode := resolveEffectiveDomainScopeMode(endpointSpec.Scopes, caller, "dal.medical.read")
	batchStrictMode := dal.ScopeOwnGroups
	if (hasPupilsRead && pupilsMode == dal.ScopeAll) ||
		(hasSenRead && senMode == dal.ScopeAll) ||
		(hasMedicalRead && medMode == dal.ScopeAll) {
		batchStrictMode = dal.ScopeAll
	}

	senMedicalBatchFn := L.NewFunction(func(L *lua.LState) int {
		pupilIDs, optsTbl, argErr := extractPupilIDsAndOptsFromLua(L)
		if argErr != nil {
			L.RaiseError("dal.pupils.list_sen_and_medical_batch %s", argErr.Error())
			return 0
		}
		effectiveMode := dal.ScopeAll
		if optsTbl != nil && optsTbl.RawGetString("strict_own_groups") == lua.LTrue {
			effectiveMode = batchStrictMode
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
				L.SetField(sTbl, "secondary_sen_need_code", lua.LString(summary.SecondarySenNeedCode))
				L.SetField(sTbl, "teaching_strategies", lua.LString(summary.TeachingStrategies))
				L.SetField(sTbl, "pupil_targets", lua.LString(summary.PupilTargets))
				L.SetField(sTbl, "senco_notes", lua.LString(summary.SencoNotes))
				L.SetField(sTbl, "ehcp_start_date", lua.LString(summary.EHCPStartDate))
				L.SetField(sTbl, "ehcp_review_due_date", lua.LString(summary.EHCPReviewDueDate))
				L.SetField(sTbl, "next_annual_review_date", lua.LString(summary.EHCPReviewDueDate))

				needsArr := L.NewTable()
				markLuaTableAsArray(L, needsArr)
				for idx, sn := range summary.SenNeeds {
					nTbl := L.NewTable()
					L.SetField(nTbl, "id", lua.LString(sn.ID))
					L.SetField(nTbl, "sen_need_code", lua.LString(sn.SenNeedCode))
					L.SetField(nTbl, "ranking", lua.LNumber(sn.Ranking))
					L.SetField(nTbl, "alert_level_id", lua.LString(sn.AlertLevelID))
					L.SetField(nTbl, "effective_from", lua.LString(sn.EffectiveFrom))
					needsArr.RawSetInt(idx+1, nTbl)
				}
				L.SetField(sTbl, "sen_needs", needsArr)

				taArr := L.NewTable()
				markLuaTableAsArray(L, taArr)
				for idx, ta := range summary.SenAssistants {
					tTbl := L.NewTable()
					L.SetField(tTbl, "id", lua.LString(ta.ID))
					L.SetField(tTbl, "staff_id", lua.LString(ta.StaffID))
					L.SetField(tTbl, "display_name", lua.LString(ta.DisplayName))
					L.SetField(tTbl, "allocation_notes", lua.LString(ta.AllocationNotes))
					L.SetField(tTbl, "effective_from", lua.LString(ta.EffectiveFrom))
					taArr.RawSetInt(idx+1, tTbl)
				}
				L.SetField(sTbl, "sen_assistants", taArr)
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
					L.SetField(mTbl, "triggers", lua.LString(mc.Triggers))
					L.SetField(mTbl, "reaction_symptoms", lua.LString(mc.ReactionSymptoms))
					L.SetField(mTbl, "emergency_response_plan", lua.LString(mc.EmergencyResponsePlan))
					L.SetField(mTbl, "medication_name_and_dose", lua.LString(mc.MedicationNameAndDose))
					L.SetField(mTbl, "medication_location", lua.LString(mc.MedicationLocation))
					L.SetField(mTbl, "parental_consent_given", lua.LBool(mc.ParentalConsentGiven))
					L.SetField(mTbl, "effective_from", lua.LString(mc.EffectiveFrom))
					L.SetField(mTbl, "effective_to", lua.LString(mc.EffectiveTo))
					L.SetField(mTbl, "medication_expiry_date", lua.LString(mc.EffectiveTo))
					medsArr.RawSetInt(idx+1, mTbl)
				}
				L.SetField(sTbl, "medical_conditions", medsArr)
			}
			L.SetField(resTbl, pid, sTbl)
		}
		L.Push(resTbl)
		return 1
	})

	senRegisterFn := L.NewFunction(func(L *lua.LState) int {
		var optsTbl *lua.LTable
		if L.GetTop() >= 1 {
			optsTbl, _ = L.Get(1).(*lua.LTable)
		}
		effectiveMode := dal.ScopeAll
		if optsTbl != nil && optsTbl.RawGetString("strict_own_groups") == lua.LTrue {
			effectiveMode = batchStrictMode
		}
		scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, effectiveMode, caller.StaffID, optsTbl)
		if sErr != nil {
			L.RaiseError("dal.sen.list_register scope error: %s", sErr.Error())
			return 0
		}
		var filterOpts dal.SenRegisterFilterOptions
		if optsTbl != nil {
			if gVal := optsTbl.RawGetString("group_id"); gVal != lua.LNil {
				filterOpts.GroupID = strings.TrimSpace(gVal.String())
			}
			if yVal := optsTbl.RawGetString("nc_year_group"); yVal != lua.LNil {
				filterOpts.NCYearGroup = strings.TrimSpace(yVal.String())
			}
			if pVal := optsTbl.RawGetString("provision_code"); pVal != lua.LNil && strings.TrimSpace(pVal.String()) != "" {
				filterOpts.ProvisionCodes = []string{strings.TrimSpace(pVal.String())}
			} else if pArr, isTbl := optsTbl.RawGetString("provision_codes").(*lua.LTable); isTbl {
				for i := 1; i <= pArr.MaxN(); i++ {
					if item := pArr.RawGetInt(i); item != lua.LNil {
						filterOpts.ProvisionCodes = append(filterOpts.ProvisionCodes, strings.TrimSpace(item.String()))
					}
				}
			}
		}

		entries, err := h.coreDAL.ListSenRegisterScoped(execCtx, scope, filterOpts)
		if err != nil {
			L.RaiseError("dal.sen.list_register failed: %s", err.Error())
			return 0
		}

		arrTbl := L.NewTable()
		markLuaTableAsArray(L, arrTbl)
		for i, entry := range entries {
			eTbl := pupilToLuaTable(L, entry.Pupil)
			L.SetField(eTbl, "sen_provision_code", lua.LString(entry.SenProvisionCode))
			L.SetField(eTbl, "primary_sen_need_code", lua.LString(entry.PrimarySenNeedCode))
			L.SetField(eTbl, "secondary_sen_need_code", lua.LString(entry.SecondarySenNeedCode))
			L.SetField(eTbl, "teaching_strategies", lua.LString(entry.TeachingStrategies))
			L.SetField(eTbl, "pupil_targets", lua.LString(entry.PupilTargets))
			L.SetField(eTbl, "senco_notes", lua.LString(entry.SencoNotes))
			L.SetField(eTbl, "ehcp_start_date", lua.LString(entry.EHCPStartDate))
			L.SetField(eTbl, "ehcp_review_due_date", lua.LString(entry.EHCPReviewDueDate))
			L.SetField(eTbl, "next_annual_review_date", lua.LString(entry.EHCPReviewDueDate))

			needsArr := L.NewTable()
			markLuaTableAsArray(L, needsArr)
			for idx, sn := range entry.SenNeeds {
				nTbl := L.NewTable()
				L.SetField(nTbl, "id", lua.LString(sn.ID))
				L.SetField(nTbl, "sen_need_code", lua.LString(sn.SenNeedCode))
				L.SetField(nTbl, "ranking", lua.LNumber(sn.Ranking))
				L.SetField(nTbl, "alert_level_id", lua.LString(sn.AlertLevelID))
				L.SetField(nTbl, "effective_from", lua.LString(sn.EffectiveFrom))
				needsArr.RawSetInt(idx+1, nTbl)
			}
			L.SetField(eTbl, "sen_needs", needsArr)

			taArr := L.NewTable()
			markLuaTableAsArray(L, taArr)
			for idx, ta := range entry.SenAssistants {
				tTbl := L.NewTable()
				L.SetField(tTbl, "id", lua.LString(ta.ID))
				L.SetField(tTbl, "staff_id", lua.LString(ta.StaffID))
				L.SetField(tTbl, "display_name", lua.LString(ta.DisplayName))
				L.SetField(tTbl, "allocation_notes", lua.LString(ta.AllocationNotes))
				L.SetField(tTbl, "effective_from", lua.LString(ta.EffectiveFrom))
				taArr.RawSetInt(idx+1, tTbl)
			}
			L.SetField(eTbl, "sen_assistants", taArr)

			arrTbl.RawSetInt(i+1, eTbl)
		}
		L.Push(arrTbl)
		return 1
	})

	L.SetField(pupilsTbl, "list_sen_and_medical_batch", senMedicalBatchFn)
	if hasPupilsRead || hasSenRead {
		L.SetField(pupilsTbl, "list_sen_register", senRegisterFn)
	}
	L.SetField(dalTbl, "pupils", pupilsTbl)

	if hasSenRead {
		senTbl := L.NewTable()
		L.SetField(senTbl, "list_batch", senMedicalBatchFn)
		L.SetField(senTbl, "list_register", senRegisterFn)
		L.SetField(dalTbl, "sen", senTbl)
	}
	if hasMedicalRead {
		medTbl := L.NewTable()
		L.SetField(medTbl, "list_batch", senMedicalBatchFn)
		L.SetField(dalTbl, "medical", medTbl)
	}
}
