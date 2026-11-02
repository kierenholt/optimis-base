package dalbindings

import (
	"context"

	lua "github.com/yuin/gopher-lua"

	"optimis/internal/dal"
	"optimis/internal/host/types"
)

func fsmPeriodToLuaTable(L *lua.LState, fp dal.PupilFSMPeriodSummary) *lua.LTable {
	fTbl := L.NewTable()
	L.SetField(fTbl, "id", lua.LString(fp.ID))
	L.SetField(fTbl, "fsm_period_id", lua.LString(fp.ID))
	L.SetField(fTbl, "pupil_id", lua.LString(fp.PupilID))
	L.SetField(fTbl, "fsm_type", lua.LString(fp.FSMType))
	L.SetField(fTbl, "ever6_fsm_flag", lua.LBool(fp.Ever6FSMFlag))
	L.SetField(fTbl, "ecs_reference", lua.LString(fp.ECSReference))
	L.SetField(fTbl, "effective_from", lua.LString(fp.EffectiveFrom))
	L.SetField(fTbl, "effective_to", lua.LString(fp.EffectiveTo))
	L.SetField(fTbl, "country_code", lua.LString(fp.CountryCode))
	L.SetField(fTbl, "is_active", lua.LBool(fp.IsActive))
	return fTbl
}

func pupilExclusionToLuaTable(L *lua.LState, ex dal.PupilExclusionSummary) *lua.LTable {
	eTbl := L.NewTable()
	L.SetField(eTbl, "id", lua.LString(ex.ID))
	L.SetField(eTbl, "exclusion_id", lua.LString(ex.ID))
	L.SetField(eTbl, "pupil_id", lua.LString(ex.PupilID))
	L.SetField(eTbl, "academic_year_id", lua.LString(ex.AcademicYearID))
	L.SetField(eTbl, "category", lua.LString(ex.Category))
	L.SetField(eTbl, "exclusion_category", lua.LString(ex.Category))
	L.SetField(eTbl, "dfe_reason_code", lua.LString(ex.DfEReasonCode))
	L.SetField(eTbl, "reason_code", lua.LString(ex.DfEReasonCode))
	L.SetField(eTbl, "reason_code_1", lua.LString(ex.ReasonCode1))
	L.SetField(eTbl, "reason_code_2", lua.LString(ex.ReasonCode2))
	L.SetField(eTbl, "reason_code_3", lua.LString(ex.ReasonCode3))
	L.SetField(eTbl, "start_date", lua.LString(ex.StartDate))
	L.SetField(eTbl, "end_date", lua.LString(ex.EndDate))
	L.SetField(eTbl, "sessions_lost", lua.LNumber(ex.SessionsLost))
	L.SetField(eTbl, "sessions_missed", lua.LNumber(ex.SessionsMissed))
	L.SetField(eTbl, "sen_provision_at_exclusion", lua.LString(ex.SENProvisionAtExclusion))
	L.SetField(eTbl, "review_date", lua.LString(ex.ReviewDate))
	L.SetField(eTbl, "governor_review_date", lua.LString(ex.ReviewDate))
	L.SetField(eTbl, "review_outcome", lua.LString(ex.ReviewOutcome))
	L.SetField(eTbl, "governor_review_result_code", lua.LString(ex.ReviewOutcome))
	L.SetField(eTbl, "authorising_staff_id", lua.LString(ex.AuthorisingStaffID))
	L.SetField(eTbl, "notes", lua.LString(ex.Notes))
	return eTbl
}

func pupilAlternativeProvisionToLuaTable(L *lua.LState, ap dal.PupilAlternativeProvisionSummary) *lua.LTable {
	aTbl := L.NewTable()
	L.SetField(aTbl, "id", lua.LString(ap.ID))
	L.SetField(aTbl, "provision_id", lua.LString(ap.ID))
	L.SetField(aTbl, "pupil_id", lua.LString(ap.PupilID))
	L.SetField(aTbl, "academic_year_id", lua.LString(ap.AcademicYearID))
	L.SetField(aTbl, "provider_urn", lua.LString(ap.ProviderURN))
	L.SetField(aTbl, "provider_ukprn", lua.LString(ap.ProviderUKPRN))
	L.SetField(aTbl, "companies_house_number", lua.LString(ap.CompaniesHouseNumber))
	L.SetField(aTbl, "ap_setting_type_code", lua.LString(ap.APSettingTypeCode))
	L.SetField(aTbl, "reason_code", lua.LString(ap.ReasonCode))
	L.SetField(aTbl, "sen_provision", lua.LString(ap.SENProvision))
	L.SetField(aTbl, "sen_provision_on_entry_code", lua.LString(ap.SENProvision))
	L.SetField(aTbl, "sessions_per_week", lua.LNumber(ap.SessionsPerWeek))
	L.SetField(aTbl, "effective_from", lua.LString(ap.EffectiveFrom))
	L.SetField(aTbl, "placement_start_date", lua.LString(ap.EffectiveFrom))
	L.SetField(aTbl, "effective_to", lua.LString(ap.EffectiveTo))
	L.SetField(aTbl, "placement_end_date", lua.LString(ap.EffectiveTo))
	return aTbl
}

func pupilPost16LearningAimToLuaTable(L *lua.LState, la dal.PupilPost16LearningAimSummary) *lua.LTable {
	lTbl := L.NewTable()
	L.SetField(lTbl, "id", lua.LString(la.ID))
	L.SetField(lTbl, "aim_id", lua.LString(la.ID))
	L.SetField(lTbl, "pupil_id", lua.LString(la.PupilID))
	L.SetField(lTbl, "academic_year_id", lua.LString(la.AcademicYearID))
	L.SetField(lTbl, "qan_code", lua.LString(la.QANCode))
	L.SetField(lTbl, "qn_code", lua.LString(la.QANCode))
	L.SetField(lTbl, "discount_code", lua.LString(la.DiscountCode))
	L.SetField(lTbl, "subject_classification_code", lua.LString(la.DiscountCode))
	L.SetField(lTbl, "subject_title", lua.LString(la.SubjectTitle))
	L.SetField(lTbl, "course_id", lua.LString(la.CourseID))
	L.SetField(lTbl, "planned_glh", lua.LNumber(la.PlannedGLH))
	L.SetField(lTbl, "planned_learning_hours", lua.LNumber(la.PlannedGLH))
	L.SetField(lTbl, "planned_eep_hours", lua.LNumber(la.PlannedEEPHours))
	L.SetField(lTbl, "start_date", lua.LString(la.StartDate))
	L.SetField(lTbl, "planned_start_date", lua.LString(la.StartDate))
	L.SetField(lTbl, "planned_end_date", lua.LString(la.PlannedEndDate))
	L.SetField(lTbl, "actual_end_date", lua.LString(la.ActualEndDate))
	L.SetField(lTbl, "completion_status", lua.LString(la.CompletionStatus))
	L.SetField(lTbl, "completion_status_code", lua.LString(la.CompletionStatus))
	L.SetField(lTbl, "outcome", lua.LString(la.Outcome))
	return lTbl
}

func pupilPastoralNoteToLuaTable(L *lua.LState, n dal.PupilPastoralNoteSummary) *lua.LTable {
	nTbl := L.NewTable()
	L.SetField(nTbl, "id", lua.LString(n.ID))
	L.SetField(nTbl, "note_id", lua.LString(n.ID))
	L.SetField(nTbl, "pupil_id", lua.LString(n.PupilID))
	L.SetField(nTbl, "recorded_by_staff_id", lua.LString(n.RecordedByStaffID))
	L.SetField(nTbl, "recorded_at", lua.LString(n.CreatedAt))
	L.SetField(nTbl, "created_at", lua.LString(n.CreatedAt))
	L.SetField(nTbl, "category", lua.LString(n.Category))
	L.SetField(nTbl, "alert_level_id", lua.LNumber(n.AlertLevelID))
	L.SetField(nTbl, "title", lua.LString(n.Title))
	L.SetField(nTbl, "summary", lua.LString(n.Summary))
	L.SetField(nTbl, "note_text", lua.LString(n.Summary))
	return nTbl
}

func (b *Binder) bindPupilsPastoralReadMethods(
	L *lua.LState,
	execCtx context.Context,
	endpointSpec types.EndpointSpec,
	caller types.CallerContext,
	pupilsMode dal.ScopeMode,
	pupilsTbl *lua.LTable,
) {
	L.SetField(pupilsTbl, "list_fsm_periods_batch", L.NewFunction(func(L *lua.LState) int {
		pupilIDs, optsTbl, argErr := extractPupilIDsAndOptsFromLua(L)
		if argErr != nil {
			L.RaiseError("dal.pupils.list_fsm_periods_batch %s", argErr.Error())
			return 0
		}
		effectiveMode := dal.ScopeAll
		if optsTbl != nil && optsTbl.RawGetString("strict_own_groups") == lua.LTrue {
			effectiveMode = pupilsMode
		}
		scope, sErr := extractScopeFromLuaOpts(execCtx, b.coreDAL, effectiveMode, caller.StaffID, optsTbl)
		if sErr != nil {
			L.RaiseError("dal.pupils.list_fsm_periods_batch scope error: %s", sErr.Error())
			return 0
		}
		batch, err := b.coreDAL.ListPupilFSMPeriodsBatchScoped(execCtx, scope, pupilIDs)
		if err != nil {
			L.RaiseError("dal.pupils.list_fsm_periods_batch failed: %s", err.Error())
			return 0
		}
		resTbl := L.NewTable()
		for pid, periods := range batch {
			arrTbl := L.NewTable()
			markLuaTableAsArray(L, arrTbl)
			for idx, fp := range periods {
				arrTbl.RawSetInt(idx+1, fsmPeriodToLuaTable(L, fp))
			}
			L.SetField(resTbl, pid, arrTbl)
		}
		L.Push(resTbl)
		return 1
	}))

	L.SetField(pupilsTbl, "list_exclusions", L.NewFunction(func(L *lua.LState) int {
		pupilID, ayID, optsTbl := extractPupilQueryOpts(L)
		scope, sErr := extractScopeFromLuaOpts(execCtx, b.coreDAL, pupilsMode, caller.StaffID, optsTbl)
		if sErr != nil {
			L.RaiseError("dal.pupils.list_exclusions scope error: %s", sErr.Error())
			return 0
		}
		items, err := b.coreDAL.ListPupilExclusionsScoped(execCtx, scope, pupilID, ayID)
		if err != nil {
			L.RaiseError("dal.pupils.list_exclusions failed: %s", err.Error())
			return 0
		}
		arrTbl := L.NewTable()
		markLuaTableAsArray(L, arrTbl)
		for i, ex := range items {
			arrTbl.RawSetInt(i+1, pupilExclusionToLuaTable(L, ex))
		}
		L.Push(arrTbl)
		return 1
	}))

	L.SetField(pupilsTbl, "list_alternative_provisions", L.NewFunction(func(L *lua.LState) int {
		pupilID, ayID, optsTbl := extractPupilQueryOpts(L)
		scope, sErr := extractScopeFromLuaOpts(execCtx, b.coreDAL, pupilsMode, caller.StaffID, optsTbl)
		if sErr != nil {
			L.RaiseError("dal.pupils.list_alternative_provisions scope error: %s", sErr.Error())
			return 0
		}
		items, err := b.coreDAL.ListPupilAlternativeProvisionsScoped(execCtx, scope, pupilID, ayID)
		if err != nil {
			L.RaiseError("dal.pupils.list_alternative_provisions failed: %s", err.Error())
			return 0
		}
		arrTbl := L.NewTable()
		markLuaTableAsArray(L, arrTbl)
		for i, ap := range items {
			arrTbl.RawSetInt(i+1, pupilAlternativeProvisionToLuaTable(L, ap))
		}
		L.Push(arrTbl)
		return 1
	}))

	L.SetField(pupilsTbl, "list_post16_learning_aims", L.NewFunction(func(L *lua.LState) int {
		pupilID, ayID, optsTbl := extractPupilQueryOpts(L)
		scope, sErr := extractScopeFromLuaOpts(execCtx, b.coreDAL, pupilsMode, caller.StaffID, optsTbl)
		if sErr != nil {
			L.RaiseError("dal.pupils.list_post16_learning_aims scope error: %s", sErr.Error())
			return 0
		}
		items, err := b.coreDAL.ListPupilPost16LearningAimsScoped(execCtx, scope, pupilID, ayID)
		if err != nil {
			L.RaiseError("dal.pupils.list_post16_learning_aims failed: %s", err.Error())
			return 0
		}
		arrTbl := L.NewTable()
		markLuaTableAsArray(L, arrTbl)
		for i, la := range items {
			arrTbl.RawSetInt(i+1, pupilPost16LearningAimToLuaTable(L, la))
		}
		L.Push(arrTbl)
		return 1
	}))

	L.SetField(pupilsTbl, "list_pastoral_notes", L.NewFunction(func(L *lua.LState) int {
		pupilID, _, optsTbl := extractPupilQueryOpts(L)
		scope, sErr := extractScopeFromLuaOpts(execCtx, b.coreDAL, pupilsMode, caller.StaffID, optsTbl)
		if sErr != nil {
			L.RaiseError("dal.pupils.list_pastoral_notes scope error: %s", sErr.Error())
			return 0
		}
		items, err := b.coreDAL.ListPupilPastoralNotesScoped(execCtx, scope, pupilID)
		if err != nil {
			L.RaiseError("dal.pupils.list_pastoral_notes failed: %s", err.Error())
			return 0
		}
		arrTbl := L.NewTable()
		markLuaTableAsArray(L, arrTbl)
		for i, n := range items {
			arrTbl.RawSetInt(i+1, pupilPastoralNoteToLuaTable(L, n))
		}
		L.Push(arrTbl)
		return 1
	}))
}

func (b *Binder) bindPupilsPastoralWriteMethods(
	L *lua.LState,
	execCtx context.Context,
	endpointSpec types.EndpointSpec,
	caller types.CallerContext,
	pupilsWriteMode dal.ScopeMode,
	pupilsTbl *lua.LTable,
) {
	L.SetField(pupilsTbl, "upsert_fsm_period", L.NewFunction(func(L *lua.LState) int {
		inTbl, scope, ok := extractMutationInputAndScope(L, execCtx, b.coreDAL, endpointSpec, "dal.pupils.upsert_fsm_period", pupilsWriteMode, caller.StaffID)
		if !ok {
			return 0
		}
		periodID := luaTableTrimmedString(inTbl, "fsm_period_id")
		if periodID == "" {
			periodID = luaTableTrimmedString(inTbl, "id")
		}
		fp, err := b.coreDAL.UpsertPupilFSMPeriodScoped(execCtx, scope, dal.UpsertPupilFSMPeriodInput{
			ID:            periodID,
			PupilID:       luaTableTrimmedString(inTbl, "pupil_id"),
			FSMType:       luaTableTrimmedString(inTbl, "fsm_type"),
			Ever6FSMFlag:  inTbl.RawGetString("ever6_fsm_flag") == lua.LTrue,
			ECSReference:  luaTableTrimmedString(inTbl, "ecs_reference"),
			CountryCode:   luaTableTrimmedString(inTbl, "country_code"),
			EffectiveFrom: luaTableTrimmedString(inTbl, "effective_from"),
			EffectiveTo:   luaTableTrimmedString(inTbl, "effective_to"),
		})
		if err != nil {
			L.RaiseError("dal.pupils.upsert_fsm_period failed: %s", err.Error())
			return 0
		}
		L.Push(fsmPeriodToLuaTable(L, *fp))
		return 1
	}))

	L.SetField(pupilsTbl, "upsert_exclusion", L.NewFunction(func(L *lua.LState) int {
		inTbl, scope, ok := extractMutationInputAndScope(L, execCtx, b.coreDAL, endpointSpec, "dal.pupils.upsert_exclusion", pupilsWriteMode, caller.StaffID)
		if !ok {
			return 0
		}
		exID := luaTableTrimmedString(inTbl, "exclusion_id")
		if exID == "" {
			exID = luaTableTrimmedString(inTbl, "id")
		}
		reason1 := luaTableTrimmedString(inTbl, "dfe_reason_code")
		if reason1 == "" {
			reason1 = luaTableTrimmedString(inTbl, "reason_code")
		}
		if reason1 == "" {
			reason1 = luaTableTrimmedString(inTbl, "reason_code_1")
		}
		reviewDate := luaTableTrimmedString(inTbl, "review_date")
		if reviewDate == "" {
			reviewDate = luaTableTrimmedString(inTbl, "governor_review_date")
		}
		reviewOutcome := luaTableTrimmedString(inTbl, "review_outcome")
		if reviewOutcome == "" {
			reviewOutcome = luaTableTrimmedString(inTbl, "governor_review_result_code")
		}
		ex, err := b.coreDAL.UpsertPupilExclusionScoped(execCtx, scope, dal.PupilExclusionSummary{
			ID:                      exID,
			ExclusionID:             exID,
			PupilID:                 luaTableTrimmedString(inTbl, "pupil_id"),
			AcademicYearID:          luaTableTrimmedString(inTbl, "academic_year_id"),
			Category:                luaTableTrimmedString(inTbl, "category"),
			ExclusionCategory:       luaTableTrimmedString(inTbl, "exclusion_category"),
			DfEReasonCode:           reason1,
			ReasonCode1:             reason1,
			ReasonCode2:             luaTableTrimmedString(inTbl, "reason_code_2"),
			ReasonCode3:             luaTableTrimmedString(inTbl, "reason_code_3"),
			StartDate:               luaTableTrimmedString(inTbl, "start_date"),
			EndDate:                 luaTableTrimmedString(inTbl, "end_date"),
			SessionsLost:            luaTableInt(inTbl, "sessions_lost", 0),
			SessionsMissed:          luaTableInt(inTbl, "sessions_missed", 0),
			SENProvisionAtExclusion: luaTableTrimmedString(inTbl, "sen_provision_at_exclusion"),
			ReviewDate:              reviewDate,
			ReviewOutcome:           reviewOutcome,
			AuthorisingStaffID:      luaTableTrimmedString(inTbl, "authorising_staff_id"),
			Notes:                   luaTableTrimmedString(inTbl, "notes"),
		})
		if err != nil {
			L.RaiseError("dal.pupils.upsert_exclusion failed: %s", err.Error())
			return 0
		}
		L.Push(pupilExclusionToLuaTable(L, *ex))
		return 1
	}))

	L.SetField(pupilsTbl, "upsert_alternative_provision", L.NewFunction(func(L *lua.LState) int {
		inTbl, scope, ok := extractMutationInputAndScope(L, execCtx, b.coreDAL, endpointSpec, "dal.pupils.upsert_alternative_provision", pupilsWriteMode, caller.StaffID)
		if !ok {
			return 0
		}
		apID := luaTableTrimmedString(inTbl, "provision_id")
		if apID == "" {
			apID = luaTableTrimmedString(inTbl, "id")
		}
		effFrom := luaTableTrimmedString(inTbl, "effective_from")
		if effFrom == "" {
			effFrom = luaTableTrimmedString(inTbl, "placement_start_date")
		}
		effTo := luaTableTrimmedString(inTbl, "effective_to")
		if effTo == "" {
			effTo = luaTableTrimmedString(inTbl, "placement_end_date")
		}
		senProv := luaTableTrimmedString(inTbl, "sen_provision")
		if senProv == "" {
			senProv = luaTableTrimmedString(inTbl, "sen_provision_on_entry_code")
		}
		ap, err := b.coreDAL.UpsertPupilAlternativeProvisionScoped(execCtx, scope, dal.PupilAlternativeProvisionSummary{
			ID:                   apID,
			ProvisionID:          apID,
			PupilID:              luaTableTrimmedString(inTbl, "pupil_id"),
			AcademicYearID:       luaTableTrimmedString(inTbl, "academic_year_id"),
			ProviderURN:          luaTableTrimmedString(inTbl, "provider_urn"),
			ProviderUKPRN:        luaTableTrimmedString(inTbl, "provider_ukprn"),
			CompaniesHouseNumber: luaTableTrimmedString(inTbl, "companies_house_number"),
			APSettingTypeCode:    luaTableTrimmedString(inTbl, "ap_setting_type_code"),
			ReasonCode:           luaTableTrimmedString(inTbl, "reason_code"),
			SENProvision:         senProv,
			SessionsPerWeek:      luaTableInt(inTbl, "sessions_per_week", 0),
			EffectiveFrom:        effFrom,
			EffectiveTo:          effTo,
		})
		if err != nil {
			L.RaiseError("dal.pupils.upsert_alternative_provision failed: %s", err.Error())
			return 0
		}
		L.Push(pupilAlternativeProvisionToLuaTable(L, *ap))
		return 1
	}))

	L.SetField(pupilsTbl, "upsert_post16_learning_aim", L.NewFunction(func(L *lua.LState) int {
		inTbl, scope, ok := extractMutationInputAndScope(L, execCtx, b.coreDAL, endpointSpec, "dal.pupils.upsert_post16_learning_aim", pupilsWriteMode, caller.StaffID)
		if !ok {
			return 0
		}
		aimID := luaTableTrimmedString(inTbl, "aim_id")
		if aimID == "" {
			aimID = luaTableTrimmedString(inTbl, "id")
		}
		qanCode := luaTableTrimmedString(inTbl, "qan_code")
		if qanCode == "" {
			qanCode = luaTableTrimmedString(inTbl, "qn_code")
		}
		discountCode := luaTableTrimmedString(inTbl, "discount_code")
		if discountCode == "" {
			discountCode = luaTableTrimmedString(inTbl, "subject_classification_code")
		}
		plannedGLH := luaTableInt(inTbl, "planned_glh", 0)
		if plannedGLH == 0 {
			plannedGLH = luaTableInt(inTbl, "planned_learning_hours", 0)
		}
		startDate := luaTableTrimmedString(inTbl, "start_date")
		if startDate == "" {
			startDate = luaTableTrimmedString(inTbl, "planned_start_date")
		}
		compStatus := luaTableTrimmedString(inTbl, "completion_status")
		if compStatus == "" {
			compStatus = luaTableTrimmedString(inTbl, "completion_status_code")
		}
		la, err := b.coreDAL.UpsertPupilPost16LearningAimScoped(execCtx, scope, dal.PupilPost16LearningAimSummary{
			ID:               aimID,
			AimID:            aimID,
			PupilID:          luaTableTrimmedString(inTbl, "pupil_id"),
			AcademicYearID:   luaTableTrimmedString(inTbl, "academic_year_id"),
			QANCode:          qanCode,
			DiscountCode:     discountCode,
			SubjectTitle:     luaTableTrimmedString(inTbl, "subject_title"),
			CourseID:         luaTableTrimmedString(inTbl, "course_id"),
			PlannedGLH:       plannedGLH,
			PlannedEEPHours:  luaTableInt(inTbl, "planned_eep_hours", 0),
			StartDate:        startDate,
			PlannedEndDate:   luaTableTrimmedString(inTbl, "planned_end_date"),
			ActualEndDate:    luaTableTrimmedString(inTbl, "actual_end_date"),
			CompletionStatus: compStatus,
			Outcome:          luaTableTrimmedString(inTbl, "outcome"),
		})
		if err != nil {
			L.RaiseError("dal.pupils.upsert_post16_learning_aim failed: %s", err.Error())
			return 0
		}
		L.Push(pupilPost16LearningAimToLuaTable(L, *la))
		return 1
	}))

	L.SetField(pupilsTbl, "upsert_pastoral_note", L.NewFunction(func(L *lua.LState) int {
		inTbl, scope, ok := extractMutationInputAndScope(L, execCtx, b.coreDAL, endpointSpec, "dal.pupils.upsert_pastoral_note", pupilsWriteMode, caller.StaffID)
		if !ok {
			return 0
		}
		noteID := luaTableTrimmedString(inTbl, "note_id")
		if noteID == "" {
			noteID = luaTableTrimmedString(inTbl, "id")
		}
		recBy := luaTableTrimmedString(inTbl, "recorded_by_staff_id")
		if recBy == "" {
			recBy = caller.StaffID
		}
		summary := luaTableTrimmedString(inTbl, "summary")
		if summary == "" {
			summary = luaTableTrimmedString(inTbl, "note_text")
		}
		n, err := b.coreDAL.UpsertPupilPastoralNoteScoped(execCtx, scope, dal.PupilPastoralNoteSummary{
			ID:                noteID,
			NoteID:            noteID,
			PupilID:           luaTableTrimmedString(inTbl, "pupil_id"),
			RecordedByStaffID: recBy,
			Category:          luaTableTrimmedString(inTbl, "category"),
			AlertLevelID:      luaTableInt(inTbl, "alert_level_id", 3),
			Title:             luaTableTrimmedString(inTbl, "title"),
			Summary:           summary,
		})
		if err != nil {
			L.RaiseError("dal.pupils.upsert_pastoral_note failed: %s", err.Error())
			return 0
		}
		L.Push(pupilPastoralNoteToLuaTable(L, *n))
		return 1
	}))
}
