package dalbindings

import (
	"context"
	"strings"

	lua "github.com/yuin/gopher-lua"

	"optimis/internal/dal"
	"optimis/internal/host/types"
)

func (b *Binder) createSenMedicalBatchFn(
	L *lua.LState,
	execCtx context.Context,
	endpointSpec types.EndpointSpec,
	caller types.CallerContext,
) *lua.LFunction {
	hasPupilsRead := hasDomainScope(endpointSpec.Scopes, "dal.pupils.read")
	hasSenRead := hasDomainScope(endpointSpec.Scopes, "dal.sen.read")
	hasMedicalRead := hasDomainScope(endpointSpec.Scopes, "dal.medical.read")
	pupilsMode := resolveEffectiveDomainScopeMode(endpointSpec.Scopes, caller, "dal.pupils.read")
	senMode := resolveEffectiveDomainScopeMode(endpointSpec.Scopes, caller, "dal.sen.read")
	medMode := resolveEffectiveDomainScopeMode(endpointSpec.Scopes, caller, "dal.medical.read")

	batchStrictMode := dal.ScopeOwnGroups
	if (hasPupilsRead && pupilsMode == dal.ScopeAll) ||
		(hasSenRead && senMode == dal.ScopeAll) ||
		(hasMedicalRead && medMode == dal.ScopeAll) {
		batchStrictMode = dal.ScopeAll
	}

	return L.NewFunction(func(L *lua.LState) int {
		pupilIDs, optsTbl, argErr := extractPupilIDsAndOptsFromLua(L)
		if argErr != nil {
			L.RaiseError("dal.pupils.list_sen_and_medical_batch %s", argErr.Error())
			return 0
		}
		effectiveMode := dal.ScopeAll
		if optsTbl != nil && optsTbl.RawGetString("strict_own_groups") == lua.LTrue {
			effectiveMode = batchStrictMode
		}
		scope, sErr := extractScopeFromLuaOpts(execCtx, b.coreDAL, effectiveMode, caller.StaffID, optsTbl)
		if sErr != nil {
			L.RaiseError("dal.pupils.list_sen_and_medical_batch scope error: %s", sErr.Error())
			return 0
		}
		batch, err := b.coreDAL.ListPupilSenAndMedicalBatch(execCtx, scope, pupilIDs)
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
					medsArr.RawSetInt(idx+1, medicalConditionAlertToLuaTable(L, mc))
				}
				L.SetField(sTbl, "medical_conditions", medsArr)
			}
			L.SetField(resTbl, pid, sTbl)
		}
		L.Push(resTbl)
		return 1
	})
}

func (b *Binder) createSenRegisterFn(
	L *lua.LState,
	execCtx context.Context,
	endpointSpec types.EndpointSpec,
	caller types.CallerContext,
) *lua.LFunction {
	hasPupilsRead := hasDomainScope(endpointSpec.Scopes, "dal.pupils.read")
	hasSenRead := hasDomainScope(endpointSpec.Scopes, "dal.sen.read")
	hasMedicalRead := hasDomainScope(endpointSpec.Scopes, "dal.medical.read")
	pupilsMode := resolveEffectiveDomainScopeMode(endpointSpec.Scopes, caller, "dal.pupils.read")
	senMode := resolveEffectiveDomainScopeMode(endpointSpec.Scopes, caller, "dal.sen.read")
	medMode := resolveEffectiveDomainScopeMode(endpointSpec.Scopes, caller, "dal.medical.read")

	batchStrictMode := dal.ScopeOwnGroups
	if (hasPupilsRead && pupilsMode == dal.ScopeAll) ||
		(hasSenRead && senMode == dal.ScopeAll) ||
		(hasMedicalRead && medMode == dal.ScopeAll) {
		batchStrictMode = dal.ScopeAll
	}

	return L.NewFunction(func(L *lua.LState) int {
		var optsTbl *lua.LTable
		if L.GetTop() >= 1 {
			optsTbl, _ = L.Get(1).(*lua.LTable)
		}
		effectiveMode := dal.ScopeAll
		if optsTbl != nil && optsTbl.RawGetString("strict_own_groups") == lua.LTrue {
			effectiveMode = batchStrictMode
		}
		scope, sErr := extractScopeFromLuaOpts(execCtx, b.coreDAL, effectiveMode, caller.StaffID, optsTbl)
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

		entries, err := b.coreDAL.ListSenRegisterScoped(execCtx, scope, filterOpts)
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
}

func (b *Binder) bindLuaDALSen(
	L *lua.LState,
	execCtx context.Context,
	endpointSpec types.EndpointSpec,
	caller types.CallerContext,
	dalTbl *lua.LTable,
) {
	hasSenRead := hasDomainScope(endpointSpec.Scopes, "dal.sen.read")
	hasSenWrite := hasDomainScope(endpointSpec.Scopes, "dal.sen.write")
	if !hasSenRead && !hasSenWrite {
		return
	}

	senWriteMode := resolveEffectiveDomainScopeMode(endpointSpec.Scopes, caller, "dal.sen.write")
	senTbl := L.NewTable()

	if hasSenRead {
		L.SetField(senTbl, "list_batch", b.createSenMedicalBatchFn(L, execCtx, endpointSpec, caller))
		L.SetField(senTbl, "list_register", b.createSenRegisterFn(L, execCtx, endpointSpec, caller))
		L.SetField(senTbl, "list_need_types", L.NewFunction(func(L *lua.LState) int {
			items, err := b.coreDAL.ListSenNeedTypes(execCtx)
			if err != nil {
				L.RaiseError("dal.sen.list_need_types failed: %s", err.Error())
				return 0
			}
			arrTbl := L.NewTable()
			markLuaTableAsArray(L, arrTbl)
			for i, nt := range items {
				nTbl := L.NewTable()
				L.SetField(nTbl, "code", lua.LString(nt.Code))
				L.SetField(nTbl, "name", lua.LString(nt.Name))
				L.SetField(nTbl, "broad_area", lua.LString(nt.BroadArea))
				L.SetField(nTbl, "description", lua.LString(nt.Description))
				L.SetField(nTbl, "sequence_number", lua.LNumber(nt.SequenceNumber))
				arrTbl.RawSetInt(i+1, nTbl)
			}
			L.Push(arrTbl)
			return 1
		}))
	}

	if hasSenWrite {
		L.SetField(senTbl, "upsert_need_type", L.NewFunction(func(L *lua.LState) int {
			inTbl, scope, ok := extractMutationInputAndScope(L, execCtx, b.coreDAL, endpointSpec, "dal.sen.upsert_need_type", senWriteMode, caller.StaffID)
			if !ok {
				return 0
			}
			nt, err := b.coreDAL.UpsertSenNeedType(execCtx, scope, dal.SenNeedTypeSummary{
				Code:           luaTableTrimmedString(inTbl, "code"),
				Name:           luaTableTrimmedString(inTbl, "name"),
				BroadArea:      luaTableTrimmedString(inTbl, "broad_area"),
				Description:    luaTableTrimmedString(inTbl, "description"),
				SequenceNumber: luaTableInt(inTbl, "sequence_number", 0),
			})
			if err != nil {
				L.RaiseError("dal.sen.upsert_need_type failed: %s", err.Error())
				return 0
			}
			nTbl := L.NewTable()
			L.SetField(nTbl, "code", lua.LString(nt.Code))
			L.SetField(nTbl, "name", lua.LString(nt.Name))
			L.SetField(nTbl, "broad_area", lua.LString(nt.BroadArea))
			L.SetField(nTbl, "description", lua.LString(nt.Description))
			L.SetField(nTbl, "sequence_number", lua.LNumber(nt.SequenceNumber))
			L.Push(nTbl)
			return 1
		}))

		L.SetField(senTbl, "upsert_sen_record", L.NewFunction(func(L *lua.LState) int {
			inTbl, scope, ok := extractMutationInputAndScope(L, execCtx, b.coreDAL, endpointSpec, "dal.sen.upsert_sen_record", senWriteMode, caller.StaffID)
			if !ok {
				return 0
			}
			recID := luaTableTrimmedString(inTbl, "sen_record_id")
			if recID == "" {
				recID = luaTableTrimmedString(inTbl, "id")
			}
			provCode := luaTableTrimmedString(inTbl, "provision_code")
			if provCode == "" {
				provCode = luaTableTrimmedString(inTbl, "sen_provision_code")
			}
			if provCode == "" {
				provCode = luaTableTrimmedString(inTbl, "sen_status_code")
			}
			rec, err := b.coreDAL.UpsertPupilSenRecordScoped(execCtx, scope, dal.UpsertPupilSenRecordInput{
				ID:                          recID,
				PupilID:                     luaTableTrimmedString(inTbl, "pupil_id"),
				ProvisionCode:               provCode,
				SENStatusCode:               provCode,
				EHCPStartDate:               luaTableTrimmedString(inTbl, "ehcp_start_date"),
				EHCPReviewDueDate:           luaTableTrimmedString(inTbl, "ehcp_review_due_date"),
				SENUnitIndicator:            inTbl.RawGetString("sen_unit_indicator") == lua.LTrue,
				ResourcedProvisionIndicator: inTbl.RawGetString("resourced_provision_indicator") == lua.LTrue,
				SencoNotes:                  luaTableTrimmedString(inTbl, "senco_notes"),
				TeachingStrategies:          luaTableTrimmedString(inTbl, "teaching_strategies"),
				PupilTargets:                luaTableTrimmedString(inTbl, "pupil_targets"),
				EffectiveFrom:               luaTableTrimmedString(inTbl, "effective_from"),
				EffectiveTo:                 luaTableTrimmedString(inTbl, "effective_to"),
			})
			if err != nil {
				L.RaiseError("dal.sen.upsert_sen_record failed: %s", err.Error())
				return 0
			}
			rTbl := L.NewTable()
			L.SetField(rTbl, "id", lua.LString(rec.ID))
			L.SetField(rTbl, "sen_record_id", lua.LString(rec.ID))
			L.SetField(rTbl, "pupil_id", lua.LString(rec.PupilID))
			L.SetField(rTbl, "provision_code", lua.LString(rec.ProvisionCode))
			L.SetField(rTbl, "sen_provision_code", lua.LString(rec.ProvisionCode))
			L.SetField(rTbl, "sen_status_code", lua.LString(rec.SENStatusCode))
			L.SetField(rTbl, "ehcp_start_date", lua.LString(rec.EHCPStartDate))
			L.SetField(rTbl, "ehcp_review_due_date", lua.LString(rec.EHCPReviewDueDate))
			L.SetField(rTbl, "senco_notes", lua.LString(rec.SencoNotes))
			L.SetField(rTbl, "teaching_strategies", lua.LString(rec.TeachingStrategies))
			L.SetField(rTbl, "pupil_targets", lua.LString(rec.PupilTargets))
			L.SetField(rTbl, "effective_from", lua.LString(rec.EffectiveFrom))
			L.SetField(rTbl, "effective_to", lua.LString(rec.EffectiveTo))
			L.Push(rTbl)
			return 1
		}))

		L.SetField(senTbl, "upsert_sen_need", L.NewFunction(func(L *lua.LState) int {
			inTbl, scope, ok := extractMutationInputAndScope(L, execCtx, b.coreDAL, endpointSpec, "dal.sen.upsert_sen_need", senWriteMode, caller.StaffID)
			if !ok {
				return 0
			}
			needID := luaTableTrimmedString(inTbl, "sen_need_id")
			if needID == "" {
				needID = luaTableTrimmedString(inTbl, "id")
			}
			senNeedCode := luaTableTrimmedString(inTbl, "sen_need_code")
			if senNeedCode == "" {
				senNeedCode = luaTableTrimmedString(inTbl, "sen_need_type_code")
			}
			if senNeedCode == "" {
				senNeedCode = luaTableTrimmedString(inTbl, "code")
			}
			sn, err := b.coreDAL.UpsertPupilSenNeedScoped(execCtx, scope, dal.UpsertPupilSenNeedInput{
				ID:            needID,
				PupilID:       luaTableTrimmedString(inTbl, "pupil_id"),
				SenNeedCode:   senNeedCode,
				Ranking:       luaTableInt(inTbl, "ranking", 1),
				AlertLevelID:  luaTableTrimmedString(inTbl, "alert_level_id"),
				EffectiveFrom: luaTableTrimmedString(inTbl, "effective_from"),
				EffectiveTo:   luaTableTrimmedString(inTbl, "effective_to"),
			})
			if err != nil {
				L.RaiseError("dal.sen.upsert_sen_need failed: %s", err.Error())
				return 0
			}
			nTbl := L.NewTable()
			L.SetField(nTbl, "id", lua.LString(sn.ID))
			L.SetField(nTbl, "sen_need_id", lua.LString(sn.ID))
			L.SetField(nTbl, "sen_need_code", lua.LString(sn.SenNeedCode))
			L.SetField(nTbl, "sen_need_type_code", lua.LString(sn.SenNeedCode))
			L.SetField(nTbl, "ranking", lua.LNumber(sn.Ranking))
			L.SetField(nTbl, "alert_level_id", lua.LString(sn.AlertLevelID))
			L.SetField(nTbl, "effective_from", lua.LString(sn.EffectiveFrom))
			L.Push(nTbl)
			return 1
		}))

		L.SetField(senTbl, "upsert_sen_assistant", L.NewFunction(func(L *lua.LState) int {
			inTbl, scope, ok := extractMutationInputAndScope(L, execCtx, b.coreDAL, endpointSpec, "dal.sen.upsert_sen_assistant", senWriteMode, caller.StaffID)
			if !ok {
				return 0
			}
			taID := luaTableTrimmedString(inTbl, "sen_assistant_id")
			if taID == "" {
				taID = luaTableTrimmedString(inTbl, "id")
			}
			ta, err := b.coreDAL.UpsertPupilSenAssistantScoped(execCtx, scope, dal.UpsertPupilSenAssistantInput{
				ID:              taID,
				PupilID:         luaTableTrimmedString(inTbl, "pupil_id"),
				StaffID:         luaTableTrimmedString(inTbl, "staff_id"),
				AllocationNotes: luaTableTrimmedString(inTbl, "allocation_notes"),
				EffectiveFrom:   luaTableTrimmedString(inTbl, "effective_from"),
				EffectiveTo:     luaTableTrimmedString(inTbl, "effective_to"),
			})
			if err != nil {
				L.RaiseError("dal.sen.upsert_sen_assistant failed: %s", err.Error())
				return 0
			}
			tTbl := L.NewTable()
			L.SetField(tTbl, "id", lua.LString(ta.ID))
			L.SetField(tTbl, "sen_assistant_id", lua.LString(ta.ID))
			L.SetField(tTbl, "staff_id", lua.LString(ta.StaffID))
			L.SetField(tTbl, "display_name", lua.LString(ta.DisplayName))
			L.SetField(tTbl, "allocation_notes", lua.LString(ta.AllocationNotes))
			L.SetField(tTbl, "effective_from", lua.LString(ta.EffectiveFrom))
			L.Push(tTbl)
			return 1
		}))
	}

	L.SetField(dalTbl, "sen", senTbl)
}
