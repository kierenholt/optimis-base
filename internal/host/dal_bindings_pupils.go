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
	L.SetField(pTbl, "person_id", lua.LString(pupil.PersonID))
	L.SetField(pTbl, "upn", lua.LString(pupil.UPN))
	L.SetField(pTbl, "uln", lua.LString(pupil.ULN))
	L.SetField(pTbl, "former_upn", lua.LString(pupil.FormerUPN))
	L.SetField(pTbl, "nhs_number", lua.LString(pupil.NHSNumber))
	L.SetField(pTbl, "admission_number", lua.LString(pupil.AdmissionNumber))
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
	L.SetField(pTbl, "legal_middle_names", lua.LString(pupil.LegalMiddleNames))
	L.SetField(pTbl, "legal_last_name", lua.LString(legLast))
	L.SetField(pTbl, "preferred_first_name", lua.LString(prefFirst))
	L.SetField(pTbl, "preferred_last_name", lua.LString(prefLast))
	L.SetField(pTbl, "former_last_name", lua.LString(pupil.FormerLastName))
	L.SetField(pTbl, "dob", lua.LString(pupil.DOB))
	L.SetField(pTbl, "sex", lua.LString(pupil.Sex))
	L.SetField(pTbl, "gender_identity", lua.LString(pupil.GenderIdentity))
	L.SetField(pTbl, "email", lua.LString(pupil.Email))
	L.SetField(pTbl, "phone_primary", lua.LString(pupil.PhonePrimary))
	L.SetField(pTbl, "photo_path", lua.LString(pupil.PhotoPath))
	L.SetField(pTbl, "status", lua.LString(pupil.Status))
	L.SetField(pTbl, "ethnicity_code", lua.LString(pupil.EthnicityCode))
	L.SetField(pTbl, "first_language_code", lua.LString(pupil.FirstLanguageCode))
	L.SetField(pTbl, "eal_flag", lua.LBool(pupil.EALFlag))
	L.SetField(pTbl, "country_of_birth", lua.LString(pupil.CountryOfBirth))
	L.SetField(pTbl, "religion_code", lua.LString(pupil.ReligionCode))
	L.SetField(pTbl, "pupil_premium_flag", lua.LBool(pupil.PupilPremiumFlag))
	L.SetField(pTbl, "early_years_pp_flag", lua.LBool(pupil.EarlyYearsPPFlag))
	L.SetField(pTbl, "service_child_flag", lua.LBool(pupil.ServiceChildFlag))
	L.SetField(pTbl, "looked_after_flag", lua.LBool(pupil.LookedAfterFlag))
	L.SetField(pTbl, "post_looked_after_code", lua.LString(pupil.PostLookedAfterCode))
	L.SetField(pTbl, "young_carer_code", lua.LString(pupil.YoungCarerCode))
	L.SetField(pTbl, "nc_year_group", lua.LString(pupil.NCYearGroup))
	enrStatus := pupil.EnrolmentStatus
	if enrStatus == "" {
		enrStatus = "C"
	}
	L.SetField(pTbl, "enrolment_status", lua.LString(enrStatus))
	L.SetField(pTbl, "registration_group_id", lua.LString(pupil.RegistrationGroupID))
	L.SetField(pTbl, "house_id", lua.LString(pupil.HouseID))
	L.SetField(pTbl, "boarder_status", lua.LString(pupil.BoarderStatus))
	L.SetField(pTbl, "part_time_flag", lua.LBool(pupil.PartTimeFlag))
	L.SetField(pTbl, "ey_funded_hours", lua.LNumber(pupil.EYFundedHours))
	L.SetField(pTbl, "ey_extended_funded_hours", lua.LNumber(pupil.EYExtendedFundedHours))
	L.SetField(pTbl, "ey_thirty_hour_code", lua.LString(pupil.EYThirtyHourCode))
	L.SetField(pTbl, "ey_daf_indicator", lua.LBool(pupil.EYDAFIndicator))
	L.SetField(pTbl, "previous_school_urn", lua.LString(pupil.PreviousSchoolURN))
	L.SetField(pTbl, "destination_school_urn", lua.LString(pupil.DestinationSchoolURN))
	L.SetField(pTbl, "leaving_reason_code", lua.LString(pupil.LeavingReasonCode))
	L.SetField(pTbl, "academic_year_id", lua.LString(pupil.AcademicYearID))
	L.SetField(pTbl, "effective_from", lua.LString(pupil.EffectiveFrom))
	if pupil.EffectiveTo != nil {
		L.SetField(pTbl, "effective_to", lua.LString(*pupil.EffectiveTo))
	}
	return pTbl
}

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
	L.SetField(eTbl, "exclusion_id", lua.LString(ex.ExclusionID))
	L.SetField(eTbl, "pupil_id", lua.LString(ex.PupilID))
	L.SetField(eTbl, "academic_year_id", lua.LString(ex.AcademicYearID))
	L.SetField(eTbl, "category", lua.LString(ex.Category))
	L.SetField(eTbl, "exclusion_category", lua.LString(ex.ExclusionCategory))
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
	L.SetField(aTbl, "provision_id", lua.LString(ap.ProvisionID))
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
	L.SetField(lTbl, "aim_id", lua.LString(la.AimID))
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
	L.SetField(nTbl, "note_id", lua.LString(n.NoteID))
	L.SetField(nTbl, "pupil_id", lua.LString(n.PupilID))
	L.SetField(nTbl, "recorded_by_staff_id", lua.LString(n.RecordedByStaffID))
	L.SetField(nTbl, "category", lua.LString(n.Category))
	L.SetField(nTbl, "alert_level_id", lua.LNumber(n.AlertLevelID))
	L.SetField(nTbl, "title", lua.LString(n.Title))
	L.SetField(nTbl, "summary", lua.LString(n.Summary))
	L.SetField(nTbl, "note_text", lua.LString(n.Summary))
	L.SetField(nTbl, "created_at", lua.LString(n.CreatedAt))
	return nTbl
}

func medicalConditionAlertToLuaTable(L *lua.LState, mc dal.PupilMedicalConditionAlert) *lua.LTable {
	mTbl := L.NewTable()
	L.SetField(mTbl, "id", lua.LString(mc.ID))
	L.SetField(mTbl, "condition_id", lua.LString(mc.ID))
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
	return mTbl
}

func medicalConditionInputToLuaTable(L *lua.LState, mc dal.UpsertPupilMedicalConditionInput) *lua.LTable {
	mTbl := L.NewTable()
	L.SetField(mTbl, "id", lua.LString(mc.ID))
	L.SetField(mTbl, "condition_id", lua.LString(mc.ID))
	L.SetField(mTbl, "pupil_id", lua.LString(mc.PupilID))
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
	return mTbl
}

func (h *Host) bindLuaDALPupils(
	L *lua.LState,
	execCtx context.Context,
	endpointSpec EndpointSpec,
	caller CallerContext,
	dalTbl *lua.LTable,
) {
	hasPupilsRead := hasDomainScope(endpointSpec.Scopes, "dal.pupils.read")
	hasPupilsWrite := hasDomainScope(endpointSpec.Scopes, "dal.pupils.write")
	hasSenRead := hasDomainScope(endpointSpec.Scopes, "dal.sen.read")
	hasSenWrite := hasDomainScope(endpointSpec.Scopes, "dal.sen.write")
	hasMedicalRead := hasDomainScope(endpointSpec.Scopes, "dal.medical.read")
	hasMedicalWrite := hasDomainScope(endpointSpec.Scopes, "dal.medical.write")

	if !hasPupilsRead && !hasPupilsWrite && !hasSenRead && !hasSenWrite && !hasMedicalRead && !hasMedicalWrite {
		return
	}

	pupilsMode := resolveEffectiveDomainScopeMode(endpointSpec.Scopes, caller, "dal.pupils.read")
	pupilsWriteMode := resolveEffectiveDomainScopeMode(endpointSpec.Scopes, caller, "dal.pupils.write")
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
			includeLeavers := optsTbl != nil && optsTbl.RawGetString("include_leavers") == lua.LTrue
			pupils, err := h.coreDAL.ListAllPupilsWithOptionsScoped(execCtx, scope, dal.ListPupilsOptions{
				IncludeLeavers: includeLeavers,
			})
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
			scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, effectiveMode, caller.StaffID, optsTbl)
			if sErr != nil {
				L.RaiseError("dal.pupils.list_fsm_periods_batch scope error: %s", sErr.Error())
				return 0
			}
			batch, err := h.coreDAL.ListPupilFSMPeriodsBatchScoped(execCtx, scope, pupilIDs)
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
			var optsTbl *lua.LTable
			var pupilID, ayID string
			if L.GetTop() >= 1 {
				if tbl, isTbl := L.Get(1).(*lua.LTable); isTbl {
					optsTbl = tbl
					pupilID = luaTableTrimmedString(tbl, "pupil_id")
					ayID = luaTableTrimmedString(tbl, "academic_year_id")
				} else {
					pupilID = strings.TrimSpace(L.OptString(1, ""))
					if L.GetTop() >= 2 {
						optsTbl, _ = L.Get(2).(*lua.LTable)
					}
				}
			}
			scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, pupilsMode, caller.StaffID, optsTbl)
			if sErr != nil {
				L.RaiseError("dal.pupils.list_exclusions scope error: %s", sErr.Error())
				return 0
			}
			items, err := h.coreDAL.ListPupilExclusionsScoped(execCtx, scope, pupilID, ayID)
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
			var optsTbl *lua.LTable
			var pupilID, ayID string
			if L.GetTop() >= 1 {
				if tbl, isTbl := L.Get(1).(*lua.LTable); isTbl {
					optsTbl = tbl
					pupilID = luaTableTrimmedString(tbl, "pupil_id")
					ayID = luaTableTrimmedString(tbl, "academic_year_id")
				} else {
					pupilID = strings.TrimSpace(L.OptString(1, ""))
					if L.GetTop() >= 2 {
						optsTbl, _ = L.Get(2).(*lua.LTable)
					}
				}
			}
			scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, pupilsMode, caller.StaffID, optsTbl)
			if sErr != nil {
				L.RaiseError("dal.pupils.list_alternative_provisions scope error: %s", sErr.Error())
				return 0
			}
			items, err := h.coreDAL.ListPupilAlternativeProvisionsScoped(execCtx, scope, pupilID, ayID)
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
			var optsTbl *lua.LTable
			var pupilID, ayID string
			if L.GetTop() >= 1 {
				if tbl, isTbl := L.Get(1).(*lua.LTable); isTbl {
					optsTbl = tbl
					pupilID = luaTableTrimmedString(tbl, "pupil_id")
					ayID = luaTableTrimmedString(tbl, "academic_year_id")
				} else {
					pupilID = strings.TrimSpace(L.OptString(1, ""))
					if L.GetTop() >= 2 {
						optsTbl, _ = L.Get(2).(*lua.LTable)
					}
				}
			}
			scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, pupilsMode, caller.StaffID, optsTbl)
			if sErr != nil {
				L.RaiseError("dal.pupils.list_post16_learning_aims scope error: %s", sErr.Error())
				return 0
			}
			items, err := h.coreDAL.ListPupilPost16LearningAimsScoped(execCtx, scope, pupilID, ayID)
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
			scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, pupilsMode, caller.StaffID, optsTbl)
			if sErr != nil {
				L.RaiseError("dal.pupils.list_pastoral_notes scope error: %s", sErr.Error())
				return 0
			}
			items, err := h.coreDAL.ListPupilPastoralNotesScoped(execCtx, scope, pupilID)
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

	if hasPupilsWrite {
		L.SetField(pupilsTbl, "upsert_pupil_admission", L.NewFunction(func(L *lua.LState) int {
			if !endpointSpec.Mutates {
				L.RaiseError("dal.pupils.upsert_pupil_admission is forbidden when endpoint mutates=false")
				return 0
			}
			inTbl, ok := L.Get(1).(*lua.LTable)
			if !ok {
				L.RaiseError("dal.pupils.upsert_pupil_admission requires an input table")
				return 0
			}
			var optsTbl *lua.LTable
			if L.GetTop() >= 2 {
				optsTbl, _ = L.Get(2).(*lua.LTable)
			}
			scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, pupilsWriteMode, caller.StaffID, optsTbl)
			if sErr != nil {
				L.RaiseError("dal.pupils.upsert_pupil_admission scope error: %s", sErr.Error())
				return 0
			}
			pupilID := luaTableTrimmedString(inTbl, "pupil_id")
			if pupilID == "" {
				pupilID = luaTableTrimmedString(inTbl, "id")
			}
			firstName := luaTableTrimmedString(inTbl, "first_name")
			if firstName == "" {
				firstName = luaTableTrimmedString(inTbl, "legal_first_name")
			}
			lastName := luaTableTrimmedString(inTbl, "last_name")
			if lastName == "" {
				lastName = luaTableTrimmedString(inTbl, "legal_last_name")
			}
			pupil, err := h.coreDAL.UpsertPupilAdmissionScoped(execCtx, scope, dal.UpsertPupilAdmissionInput{
				PupilID:               pupilID,
				PersonID:              luaTableTrimmedString(inTbl, "person_id"),
				UPN:                   luaTableTrimmedString(inTbl, "upn"),
				ULN:                   luaTableTrimmedString(inTbl, "uln"),
				FormerUPN:             luaTableTrimmedString(inTbl, "former_upn"),
				NHSNumber:             luaTableTrimmedString(inTbl, "nhs_number"),
				AdmissionNumber:       luaTableTrimmedString(inTbl, "admission_number"),
				FirstName:             firstName,
				LegalMiddleNames:      luaTableTrimmedString(inTbl, "legal_middle_names"),
				LastName:              lastName,
				PreferredFirstName:    luaTableTrimmedString(inTbl, "preferred_first_name"),
				PreferredLastName:     luaTableTrimmedString(inTbl, "preferred_last_name"),
				FormerLastName:        luaTableTrimmedString(inTbl, "former_last_name"),
				DOB:                   luaTableTrimmedString(inTbl, "dob"),
				Sex:                   luaTableTrimmedString(inTbl, "sex"),
				GenderIdentity:        luaTableTrimmedString(inTbl, "gender_identity"),
				Email:                 luaTableTrimmedString(inTbl, "email"),
				PhonePrimary:          luaTableTrimmedString(inTbl, "phone_primary"),
				PhotoPath:             luaTableTrimmedString(inTbl, "photo_path"),
				EthnicityCode:         luaTableTrimmedString(inTbl, "ethnicity_code"),
				FirstLanguageCode:     luaTableTrimmedString(inTbl, "first_language_code"),
				EALFlag:               luaTableOptBoolPtr(inTbl, "eal_flag"),
				CountryOfBirth:        luaTableTrimmedString(inTbl, "country_of_birth"),
				ReligionCode:          luaTableTrimmedString(inTbl, "religion_code"),
				PupilPremiumFlag:      luaTableOptBoolPtr(inTbl, "pupil_premium_flag"),
				EarlyYearsPPFlag:      luaTableOptBoolPtr(inTbl, "early_years_pp_flag"),
				ServiceChildFlag:      luaTableOptBoolPtr(inTbl, "service_child_flag"),
				LookedAfterFlag:       luaTableOptBoolPtr(inTbl, "looked_after_flag"),
				PostLookedAfterCode:   luaTableTrimmedString(inTbl, "post_looked_after_code"),
				YoungCarerCode:        luaTableTrimmedString(inTbl, "young_carer_code"),
				NCYearGroup:           luaTableTrimmedString(inTbl, "nc_year_group"),
				EnrolmentStatus:       luaTableTrimmedString(inTbl, "enrolment_status"),
				AcademicYearID:        luaTableTrimmedString(inTbl, "academic_year_id"),
				RegistrationGroupID:   luaTableTrimmedString(inTbl, "registration_group_id"),
				HouseID:               luaTableTrimmedString(inTbl, "house_id"),
				BoarderStatus:         luaTableTrimmedString(inTbl, "boarder_status"),
				PartTimeFlag:          luaTableOptBoolPtr(inTbl, "part_time_flag"),
				EYFundedHours:         luaTableFloat(inTbl, "ey_funded_hours", 0),
				EYExtendedFundedHours: luaTableFloat(inTbl, "ey_extended_funded_hours", 0),
				EYThirtyHourCode:      luaTableTrimmedString(inTbl, "ey_thirty_hour_code"),
				EYDAFIndicator:        luaTableOptBoolPtr(inTbl, "ey_daf_indicator"),
				PreviousSchoolURN:     luaTableTrimmedString(inTbl, "previous_school_urn"),
				EffectiveFrom:         luaTableTrimmedString(inTbl, "effective_from"),
				FSMEligible:           inTbl.RawGetString("fsm_eligible") == lua.LTrue,
				FSMEffectiveFrom:      luaTableTrimmedString(inTbl, "fsm_effective_from"),
			})
			if err != nil {
				L.RaiseError("dal.pupils.upsert_pupil_admission failed: %s", err.Error())
				return 0
			}
			L.Push(pupilToLuaTable(L, *pupil))
			return 1
		}))

		upsertEnrolmentFn := L.NewFunction(func(L *lua.LState) int {
			if !endpointSpec.Mutates {
				L.RaiseError("dal.pupils.upsert_year_enrolment is forbidden when endpoint mutates=false")
				return 0
			}
			inTbl, ok := L.Get(1).(*lua.LTable)
			if !ok {
				L.RaiseError("dal.pupils.upsert_year_enrolment requires an input table")
				return 0
			}
			var optsTbl *lua.LTable
			if L.GetTop() >= 2 {
				optsTbl, _ = L.Get(2).(*lua.LTable)
			}
			scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, pupilsWriteMode, caller.StaffID, optsTbl)
			if sErr != nil {
				L.RaiseError("dal.pupils.upsert_year_enrolment scope error: %s", sErr.Error())
				return 0
			}
			enrolmentID := luaTableTrimmedString(inTbl, "enrolment_id")
			if enrolmentID == "" {
				enrolmentID = luaTableTrimmedString(inTbl, "id")
			}
			pupil, err := h.coreDAL.UpsertPupilYearEnrolmentScoped(execCtx, scope, dal.UpsertPupilYearEnrolmentInput{
				EnrolmentID:           enrolmentID,
				PupilID:               luaTableTrimmedString(inTbl, "pupil_id"),
				AcademicYearID:        luaTableTrimmedString(inTbl, "academic_year_id"),
				NCYearGroup:           luaTableTrimmedString(inTbl, "nc_year_group"),
				EnrolmentStatus:       luaTableTrimmedString(inTbl, "enrolment_status"),
				RegistrationGroupID:   luaTableTrimmedString(inTbl, "registration_group_id"),
				HouseID:               luaTableTrimmedString(inTbl, "house_id"),
				BoarderStatus:         luaTableTrimmedString(inTbl, "boarder_status"),
				PartTimeFlag:          luaTableOptBoolPtr(inTbl, "part_time_flag"),
				EYFundedHours:         luaTableFloat(inTbl, "ey_funded_hours", 0),
				EYExtendedFundedHours: luaTableFloat(inTbl, "ey_extended_funded_hours", 0),
				EYThirtyHourCode:      luaTableTrimmedString(inTbl, "ey_thirty_hour_code"),
				EYDAFIndicator:        luaTableOptBoolPtr(inTbl, "ey_daf_indicator"),
				PreviousSchoolURN:     luaTableTrimmedString(inTbl, "previous_school_urn"),
				DestinationSchoolURN:  luaTableTrimmedString(inTbl, "destination_school_urn"),
				LeavingReasonCode:     luaTableTrimmedString(inTbl, "leaving_reason_code"),
				EffectiveFrom:         luaTableTrimmedString(inTbl, "effective_from"),
				EffectiveTo:           luaTableTrimmedString(inTbl, "effective_to"),
			})
			if err != nil {
				L.RaiseError("dal.pupils.upsert_year_enrolment failed: %s", err.Error())
				return 0
			}
			L.Push(pupilToLuaTable(L, *pupil))
			return 1
		})
		L.SetField(pupilsTbl, "upsert_year_enrolment", upsertEnrolmentFn)
		L.SetField(pupilsTbl, "promote_pupil_enrolment", upsertEnrolmentFn)

		L.SetField(pupilsTbl, "upsert_fsm_period", L.NewFunction(func(L *lua.LState) int {
			if !endpointSpec.Mutates {
				L.RaiseError("dal.pupils.upsert_fsm_period is forbidden when endpoint mutates=false")
				return 0
			}
			inTbl, ok := L.Get(1).(*lua.LTable)
			if !ok {
				L.RaiseError("dal.pupils.upsert_fsm_period requires an input table")
				return 0
			}
			var optsTbl *lua.LTable
			if L.GetTop() >= 2 {
				optsTbl, _ = L.Get(2).(*lua.LTable)
			}
			scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, pupilsWriteMode, caller.StaffID, optsTbl)
			if sErr != nil {
				L.RaiseError("dal.pupils.upsert_fsm_period scope error: %s", sErr.Error())
				return 0
			}
			periodID := luaTableTrimmedString(inTbl, "fsm_period_id")
			if periodID == "" {
				periodID = luaTableTrimmedString(inTbl, "id")
			}
			fp, err := h.coreDAL.UpsertPupilFSMPeriodScoped(execCtx, scope, dal.UpsertPupilFSMPeriodInput{
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
			if !endpointSpec.Mutates {
				L.RaiseError("dal.pupils.upsert_exclusion is forbidden when endpoint mutates=false")
				return 0
			}
			inTbl, ok := L.Get(1).(*lua.LTable)
			if !ok {
				L.RaiseError("dal.pupils.upsert_exclusion requires an input table")
				return 0
			}
			var optsTbl *lua.LTable
			if L.GetTop() >= 2 {
				optsTbl, _ = L.Get(2).(*lua.LTable)
			}
			scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, pupilsWriteMode, caller.StaffID, optsTbl)
			if sErr != nil {
				L.RaiseError("dal.pupils.upsert_exclusion scope error: %s", sErr.Error())
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
			ex, err := h.coreDAL.UpsertPupilExclusionScoped(execCtx, scope, dal.PupilExclusionSummary{
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
			if !endpointSpec.Mutates {
				L.RaiseError("dal.pupils.upsert_alternative_provision is forbidden when endpoint mutates=false")
				return 0
			}
			inTbl, ok := L.Get(1).(*lua.LTable)
			if !ok {
				L.RaiseError("dal.pupils.upsert_alternative_provision requires an input table")
				return 0
			}
			var optsTbl *lua.LTable
			if L.GetTop() >= 2 {
				optsTbl, _ = L.Get(2).(*lua.LTable)
			}
			scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, pupilsWriteMode, caller.StaffID, optsTbl)
			if sErr != nil {
				L.RaiseError("dal.pupils.upsert_alternative_provision scope error: %s", sErr.Error())
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
			ap, err := h.coreDAL.UpsertPupilAlternativeProvisionScoped(execCtx, scope, dal.PupilAlternativeProvisionSummary{
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
			if !endpointSpec.Mutates {
				L.RaiseError("dal.pupils.upsert_post16_learning_aim is forbidden when endpoint mutates=false")
				return 0
			}
			inTbl, ok := L.Get(1).(*lua.LTable)
			if !ok {
				L.RaiseError("dal.pupils.upsert_post16_learning_aim requires an input table")
				return 0
			}
			var optsTbl *lua.LTable
			if L.GetTop() >= 2 {
				optsTbl, _ = L.Get(2).(*lua.LTable)
			}
			scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, pupilsWriteMode, caller.StaffID, optsTbl)
			if sErr != nil {
				L.RaiseError("dal.pupils.upsert_post16_learning_aim scope error: %s", sErr.Error())
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
			la, err := h.coreDAL.UpsertPupilPost16LearningAimScoped(execCtx, scope, dal.PupilPost16LearningAimSummary{
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
			if !endpointSpec.Mutates {
				L.RaiseError("dal.pupils.upsert_pastoral_note is forbidden when endpoint mutates=false")
				return 0
			}
			inTbl, ok := L.Get(1).(*lua.LTable)
			if !ok {
				L.RaiseError("dal.pupils.upsert_pastoral_note requires an input table")
				return 0
			}
			var optsTbl *lua.LTable
			if L.GetTop() >= 2 {
				optsTbl, _ = L.Get(2).(*lua.LTable)
			}
			scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, pupilsWriteMode, caller.StaffID, optsTbl)
			if sErr != nil {
				L.RaiseError("dal.pupils.upsert_pastoral_note scope error: %s", sErr.Error())
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
			n, err := h.coreDAL.UpsertPupilPastoralNoteScoped(execCtx, scope, dal.PupilPastoralNoteSummary{
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

	senMode := resolveEffectiveDomainScopeMode(endpointSpec.Scopes, caller, "dal.sen.read")
	senWriteMode := resolveEffectiveDomainScopeMode(endpointSpec.Scopes, caller, "dal.sen.write")
	medMode := resolveEffectiveDomainScopeMode(endpointSpec.Scopes, caller, "dal.medical.read")
	medWriteMode := resolveEffectiveDomainScopeMode(endpointSpec.Scopes, caller, "dal.medical.write")
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
					medsArr.RawSetInt(idx+1, medicalConditionAlertToLuaTable(L, mc))
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

	if hasSenRead || hasSenWrite {
		senTbl := L.NewTable()
		if hasSenRead {
			L.SetField(senTbl, "list_batch", senMedicalBatchFn)
			L.SetField(senTbl, "list_register", senRegisterFn)
			L.SetField(senTbl, "list_need_types", L.NewFunction(func(L *lua.LState) int {
				items, err := h.coreDAL.ListSenNeedTypes(execCtx)
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
				if !endpointSpec.Mutates {
					L.RaiseError("dal.sen.upsert_need_type is forbidden when endpoint mutates=false")
					return 0
				}
				inTbl, ok := L.Get(1).(*lua.LTable)
				if !ok {
					L.RaiseError("dal.sen.upsert_need_type requires an input table")
					return 0
				}
				var optsTbl *lua.LTable
				if L.GetTop() >= 2 {
					optsTbl, _ = L.Get(2).(*lua.LTable)
				}
				scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, senWriteMode, caller.StaffID, optsTbl)
				if sErr != nil {
					L.RaiseError("dal.sen.upsert_need_type scope error: %s", sErr.Error())
					return 0
				}
				nt, err := h.coreDAL.UpsertSenNeedType(execCtx, scope, dal.SenNeedTypeSummary{
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
				if !endpointSpec.Mutates {
					L.RaiseError("dal.sen.upsert_sen_record is forbidden when endpoint mutates=false")
					return 0
				}
				inTbl, ok := L.Get(1).(*lua.LTable)
				if !ok {
					L.RaiseError("dal.sen.upsert_sen_record requires an input table")
					return 0
				}
				var optsTbl *lua.LTable
				if L.GetTop() >= 2 {
					optsTbl, _ = L.Get(2).(*lua.LTable)
				}
				scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, senWriteMode, caller.StaffID, optsTbl)
				if sErr != nil {
					L.RaiseError("dal.sen.upsert_sen_record scope error: %s", sErr.Error())
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
				rec, err := h.coreDAL.UpsertPupilSenRecordScoped(execCtx, scope, dal.UpsertPupilSenRecordInput{
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
				if !endpointSpec.Mutates {
					L.RaiseError("dal.sen.upsert_sen_need is forbidden when endpoint mutates=false")
					return 0
				}
				inTbl, ok := L.Get(1).(*lua.LTable)
				if !ok {
					L.RaiseError("dal.sen.upsert_sen_need requires an input table")
					return 0
				}
				var optsTbl *lua.LTable
				if L.GetTop() >= 2 {
					optsTbl, _ = L.Get(2).(*lua.LTable)
				}
				scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, senWriteMode, caller.StaffID, optsTbl)
				if sErr != nil {
					L.RaiseError("dal.sen.upsert_sen_need scope error: %s", sErr.Error())
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
				sn, err := h.coreDAL.UpsertPupilSenNeedScoped(execCtx, scope, dal.UpsertPupilSenNeedInput{
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
				if !endpointSpec.Mutates {
					L.RaiseError("dal.sen.upsert_sen_assistant is forbidden when endpoint mutates=false")
					return 0
				}
				inTbl, ok := L.Get(1).(*lua.LTable)
				if !ok {
					L.RaiseError("dal.sen.upsert_sen_assistant requires an input table")
					return 0
				}
				var optsTbl *lua.LTable
				if L.GetTop() >= 2 {
					optsTbl, _ = L.Get(2).(*lua.LTable)
				}
				scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, senWriteMode, caller.StaffID, optsTbl)
				if sErr != nil {
					L.RaiseError("dal.sen.upsert_sen_assistant scope error: %s", sErr.Error())
					return 0
				}
				taID := luaTableTrimmedString(inTbl, "sen_assistant_id")
				if taID == "" {
					taID = luaTableTrimmedString(inTbl, "id")
				}
				ta, err := h.coreDAL.UpsertPupilSenAssistantScoped(execCtx, scope, dal.UpsertPupilSenAssistantInput{
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

	if hasMedicalRead || hasMedicalWrite {
		medTbl := L.NewTable()
		if hasMedicalRead {
			L.SetField(medTbl, "list_batch", senMedicalBatchFn)
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
				scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, medMode, caller.StaffID, optsTbl)
				if sErr != nil {
					L.RaiseError("dal.medical.list_conditions scope error: %s", sErr.Error())
					return 0
				}
				items, err := h.coreDAL.ListPupilMedicalConditionsScoped(execCtx, scope, pupilID, condCat)
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
				if !endpointSpec.Mutates {
					L.RaiseError("dal.medical.upsert_condition is forbidden when endpoint mutates=false")
					return 0
				}
				inTbl, ok := L.Get(1).(*lua.LTable)
				if !ok {
					L.RaiseError("dal.medical.upsert_condition requires an input table")
					return 0
				}
				var optsTbl *lua.LTable
				if L.GetTop() >= 2 {
					optsTbl, _ = L.Get(2).(*lua.LTable)
				}
				scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, medWriteMode, caller.StaffID, optsTbl)
				if sErr != nil {
					L.RaiseError("dal.medical.upsert_condition scope error: %s", sErr.Error())
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
				mc, err := h.coreDAL.UpsertPupilMedicalConditionScoped(execCtx, scope, dal.UpsertPupilMedicalConditionInput{
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
}
