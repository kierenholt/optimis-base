package dalbindings

import (
	"context"
	"strings"

	lua "github.com/yuin/gopher-lua"

	"optimis/internal/dal"
	"optimis/internal/host/types"
)

func medicalConditionAlertToLuaTable(L *lua.LState, mc dal.PupilMedicalConditionAlert) *lua.LTable {
	mTbl := L.NewTable()
	L.SetField(mTbl, "id", lua.LString(mc.ID))
	L.SetField(mTbl, "condition_id", lua.LString(mc.ID))
	L.SetField(mTbl, "condition_name", lua.LString(mc.ConditionName))
	L.SetField(mTbl, "condition_category", lua.LString(mc.ConditionCategory))
	L.SetField(mTbl, "condition_type", lua.LString(mc.ConditionCategory))
	L.SetField(mTbl, "alert_level_id", lua.LString(mc.AlertLevelID))
	L.SetField(mTbl, "has_ihcp", lua.LBool(mc.HasIHCP))
	L.SetField(mTbl, "triggers", lua.LString(mc.Triggers))
	L.SetField(mTbl, "reaction_symptoms", lua.LString(mc.ReactionSymptoms))
	L.SetField(mTbl, "emergency_response_plan", lua.LString(mc.EmergencyResponsePlan))
	L.SetField(mTbl, "emergency_protocol", lua.LString(mc.EmergencyResponsePlan))
	L.SetField(mTbl, "medication_name_and_dose", lua.LString(mc.MedicationNameAndDose))
	L.SetField(mTbl, "medication_location", lua.LString(mc.MedicationLocation))
	L.SetField(mTbl, "parental_consent_given", lua.LBool(mc.ParentalConsentGiven))
	L.SetField(mTbl, "effective_from", lua.LString(mc.EffectiveFrom))
	return mTbl
}

func medicalConditionInputToLuaTable(L *lua.LState, mc dal.UpsertPupilMedicalConditionInput) *lua.LTable {
	mTbl := L.NewTable()
	L.SetField(mTbl, "id", lua.LString(mc.ID))
	L.SetField(mTbl, "condition_id", lua.LString(mc.ID))
	L.SetField(mTbl, "pupil_id", lua.LString(mc.PupilID))
	L.SetField(mTbl, "condition_name", lua.LString(mc.ConditionName))
	L.SetField(mTbl, "condition_category", lua.LString(mc.ConditionCategory))
	L.SetField(mTbl, "condition_type", lua.LString(mc.ConditionCategory))
	L.SetField(mTbl, "alert_level_id", lua.LString(mc.AlertLevelID))
	L.SetField(mTbl, "has_ihcp", lua.LBool(mc.HasIHCP))
	L.SetField(mTbl, "triggers", lua.LString(mc.Triggers))
	L.SetField(mTbl, "reaction_symptoms", lua.LString(mc.ReactionSymptoms))
	L.SetField(mTbl, "emergency_response_plan", lua.LString(mc.EmergencyResponsePlan))
	L.SetField(mTbl, "emergency_protocol", lua.LString(mc.EmergencyResponsePlan))
	L.SetField(mTbl, "medication_name_and_dose", lua.LString(mc.MedicationNameAndDose))
	L.SetField(mTbl, "medication_location", lua.LString(mc.MedicationLocation))
	L.SetField(mTbl, "parental_consent_given", lua.LBool(mc.ParentalConsentGiven))
	L.SetField(mTbl, "effective_from", lua.LString(mc.EffectiveFrom))
	L.SetField(mTbl, "effective_to", lua.LString(mc.EffectiveTo))
	return mTbl
}

func (b *Binder) bindLuaDALMedical(
	L *lua.LState,
	execCtx context.Context,
	endpointSpec types.EndpointSpec,
	caller types.CallerContext,
	dalTbl *lua.LTable,
) {
	hasMedicalRead := hasDomainScope(endpointSpec.Scopes, "dal.medical.read")
	hasMedicalWrite := hasDomainScope(endpointSpec.Scopes, "dal.medical.write")
	if !hasMedicalRead && !hasMedicalWrite {
		return
	}

	medMode := resolveEffectiveDomainScopeMode(endpointSpec.Scopes, caller, "dal.medical.read")
	medWriteMode := resolveEffectiveDomainScopeMode(endpointSpec.Scopes, caller, "dal.medical.write")
	medTbl := L.NewTable()

	if hasMedicalRead {
		L.SetField(medTbl, "list_batch", b.createSenMedicalBatchFn(L, execCtx, endpointSpec, caller))
		L.SetField(medTbl, "list_conditions", L.NewFunction(func(L *lua.LState) int {
			var optsTbl *lua.LTable
			var pupilID, condCat string
			if L.GetTop() >= 1 {
				if tbl, isTbl := L.Get(1).(*lua.LTable); isTbl {
					optsTbl = tbl
					pupilID = luaTableTrimmedString(tbl, "pupil_id")
					condCat = luaTableTrimmedString(tbl, "condition_category")
				} else {
					pupilID = strings.TrimSpace(L.OptString(1, ""))
					if L.GetTop() >= 2 {
						optsTbl, _ = L.Get(2).(*lua.LTable)
						if optsTbl != nil {
							condCat = luaTableTrimmedString(optsTbl, "condition_category")
						}
					}
				}
			}
			scope, sErr := extractScopeFromLuaOpts(execCtx, b.coreDAL, medMode, caller.StaffID, optsTbl)
			if sErr != nil {
				L.RaiseError("dal.medical.list_conditions scope error: %s", sErr.Error())
				return 0
			}
			items, err := b.coreDAL.ListPupilMedicalConditionsScoped(execCtx, scope, pupilID, condCat)
			if err != nil {
				L.RaiseError("dal.medical.list_conditions failed: %s", err.Error())
				return 0
			}
			arrTbl := L.NewTable()
			markLuaTableAsArray(L, arrTbl)
			for i, mc := range items {
				arrTbl.RawSetInt(i+1, medicalConditionInputToLuaTable(L, mc))
			}
			L.Push(arrTbl)
			return 1
		}))
	}

	if hasMedicalWrite {
		L.SetField(medTbl, "upsert_condition", L.NewFunction(func(L *lua.LState) int {
			inTbl, scope, ok := extractMutationInputAndScope(L, execCtx, b.coreDAL, endpointSpec, "dal.medical.upsert_condition", medWriteMode, caller.StaffID)
			if !ok {
				return 0
			}
			condID := luaTableTrimmedString(inTbl, "condition_id")
			if condID == "" {
				condID = luaTableTrimmedString(inTbl, "id")
			}
			condCat := luaTableTrimmedString(inTbl, "condition_category")
			if condCat == "" {
				condCat = luaTableTrimmedString(inTbl, "condition_type")
			}
			emergPlan := luaTableTrimmedString(inTbl, "emergency_response_plan")
			if emergPlan == "" {
				emergPlan = luaTableTrimmedString(inTbl, "emergency_protocol")
			}
			mc, err := b.coreDAL.UpsertPupilMedicalConditionScoped(execCtx, scope, dal.UpsertPupilMedicalConditionInput{
				ID:                    condID,
				PupilID:               luaTableTrimmedString(inTbl, "pupil_id"),
				ConditionName:         luaTableTrimmedString(inTbl, "condition_name"),
				ConditionCategory:     condCat,
				AlertLevelID:          luaTableTrimmedString(inTbl, "alert_level_id"),
				HasIHCP:               inTbl.RawGetString("has_ihcp") == lua.LTrue,
				Triggers:              luaTableTrimmedString(inTbl, "triggers"),
				ReactionSymptoms:      luaTableTrimmedString(inTbl, "reaction_symptoms"),
				EmergencyResponsePlan: emergPlan,
				MedicationNameAndDose: luaTableTrimmedString(inTbl, "medication_name_and_dose"),
				MedicationLocation:    luaTableTrimmedString(inTbl, "medication_location"),
				ParentalConsentGiven:  inTbl.RawGetString("parental_consent_given") == lua.LTrue,
				EffectiveFrom:         luaTableTrimmedString(inTbl, "effective_from"),
				EffectiveTo:           luaTableTrimmedString(inTbl, "effective_to"),
			})
			if err != nil {
				L.RaiseError("dal.medical.upsert_condition failed: %s", err.Error())
				return 0
			}
			L.Push(medicalConditionInputToLuaTable(L, *mc))
			return 1
		}))
	}

	L.SetField(dalTbl, "medical", medTbl)
}
