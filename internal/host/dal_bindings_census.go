package host

import (
	"context"

	lua "github.com/yuin/gopher-lua"

	"optimis/internal/dal"
)

func (h *Host) bindLuaDALCensus(
	L *lua.LState,
	execCtx context.Context,
	endpointSpec EndpointSpec,
	dalTbl *lua.LTable,
) {
	// ctx.dal.census (Statutory DfE School Census Rollup; strictly requires dal.census.read:all)
	if !hasDomainScope(endpointSpec.Scopes, "dal.census.read") {
		return
	}

	censusMode := resolveDomainScopeMode(endpointSpec.Scopes, "dal.census.read")
	censusTbl := L.NewTable()

	L.SetField(censusTbl, "get_statutory_rollup", L.NewFunction(func(L *lua.LState) int {
		inTbl, ok := L.Get(1).(*lua.LTable)
		if !ok {
			L.RaiseError("dal.census.get_statutory_rollup requires an input options table")
			return 0
		}

		censusDate := luaTableTrimmedString(inTbl, "census_date")
		if censusDate == "" {
			censusDate = luaTableTrimmedString(inTbl, "reference_date")
		}
		rollup, err := h.coreDAL.GetStatutoryCensusRollup(execCtx, dal.StatutoryCensusRollupInput{
			AcademicYearID:      luaTableTrimmedString(inTbl, "academic_year_id"),
			CensusDate:          censusDate,
			TermCode:            luaTableTrimmedString(inTbl, "term_code"),
			AttendanceStartDate: luaTableTrimmedString(inTbl, "attendance_start_date"),
			AttendanceEndDate:   luaTableTrimmedString(inTbl, "attendance_end_date"),
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
		L.SetField(outTbl, "reference_date", lua.LString(rollup.ReferenceDate))

		sumTbl := L.NewTable()
		L.SetField(sumTbl, "on_roll_total", lua.LNumber(rollup.Summary.OnRollTotal))
		L.SetField(sumTbl, "total_pupils_on_roll", lua.LNumber(rollup.Summary.TotalPupilsOnRoll))
		senCountsTbl := L.NewTable()
		for k, v := range rollup.Summary.SENCounts {
			L.SetField(senCountsTbl, k, lua.LNumber(v))
		}
		L.SetField(sumTbl, "sen_counts", senCountsTbl)
		L.SetField(sumTbl, "fsm_eligible_count", lua.LNumber(rollup.Summary.FSMEligibleCount))
		L.SetField(sumTbl, "fsm_pupils_count", lua.LNumber(rollup.Summary.FSMPupilsCount))
		L.SetField(sumTbl, "lac_count", lua.LNumber(rollup.Summary.LACCount))
		L.SetField(sumTbl, "exclusions_count", lua.LNumber(rollup.Summary.ExclusionsCount))
		L.SetField(sumTbl, "alternative_provision_count", lua.LNumber(rollup.Summary.AlternativeProvisionCount))
		L.SetField(sumTbl, "post16_learning_aims_count", lua.LNumber(rollup.Summary.Post16LearningAimsCount))
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
			L.SetField(pTbl, "eal_flag", lua.LBool(p.EALFlag))
			L.SetField(pTbl, "pupil_premium_flag", lua.LBool(p.PupilPremiumFlag))
			L.SetField(pTbl, "service_child_flag", lua.LBool(p.ServiceChildFlag))
			L.SetField(pTbl, "nc_year_group", lua.LString(p.NCYearGroup))
			L.SetField(pTbl, "enrolment_status", lua.LString(p.EnrolmentStatus))
			L.SetField(pTbl, "ey_funded_hours", lua.LNumber(p.EYFundedHours))
			L.SetField(pTbl, "ey_extended_funded_hours", lua.LNumber(p.EYExtendedFundedHours))
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

			exclArr := L.NewTable()
			markLuaTableAsArray(L, exclArr)
			for eIdx, ex := range p.Exclusions {
				exTbl := L.NewTable()
				L.SetField(exTbl, "id", lua.LString(ex.ID))
				L.SetField(exTbl, "exclusion_id", lua.LString(ex.ExclusionID))
				L.SetField(exTbl, "category", lua.LString(ex.Category))
				L.SetField(exTbl, "dfe_reason_code", lua.LString(ex.DfEReasonCode))
				L.SetField(exTbl, "start_date", lua.LString(ex.StartDate))
				L.SetField(exTbl, "end_date", lua.LString(ex.EndDate))
				L.SetField(exTbl, "sessions_lost", lua.LNumber(ex.SessionsLost))
				exclArr.RawSetInt(eIdx+1, exTbl)
			}
			L.SetField(pTbl, "exclusions", exclArr)

			apArr := L.NewTable()
			markLuaTableAsArray(L, apArr)
			for aIdx, ap := range p.AlternativeProvisions {
				apTbl := L.NewTable()
				L.SetField(apTbl, "id", lua.LString(ap.ID))
				L.SetField(apTbl, "provision_id", lua.LString(ap.ProvisionID))
				L.SetField(apTbl, "provider_urn", lua.LString(ap.ProviderURN))
				L.SetField(apTbl, "reason_code", lua.LString(ap.ReasonCode))
				L.SetField(apTbl, "sessions_per_week", lua.LNumber(ap.SessionsPerWeek))
				L.SetField(apTbl, "effective_from", lua.LString(ap.EffectiveFrom))
				apArr.RawSetInt(aIdx+1, apTbl)
			}
			L.SetField(pTbl, "alternative_provisions", apArr)

			aimsArr := L.NewTable()
			markLuaTableAsArray(L, aimsArr)
			for lIdx, aim := range p.Post16LearningAims {
				aimTbl := L.NewTable()
				L.SetField(aimTbl, "id", lua.LString(aim.ID))
				L.SetField(aimTbl, "aim_id", lua.LString(aim.AimID))
				L.SetField(aimTbl, "qan_code", lua.LString(aim.QANCode))
				L.SetField(aimTbl, "subject_title", lua.LString(aim.SubjectTitle))
				L.SetField(aimTbl, "planned_glh", lua.LNumber(aim.PlannedGLH))
				L.SetField(aimTbl, "completion_status", lua.LString(aim.CompletionStatus))
				L.SetField(aimTbl, "outcome", lua.LString(aim.Outcome))
				aimsArr.RawSetInt(lIdx+1, aimTbl)
			}
			L.SetField(pTbl, "post16_learning_aims", aimsArr)

			pupilsArr.RawSetInt(idx+1, pTbl)
		}
		L.SetField(outTbl, "pupils", pupilsArr)

		L.Push(outTbl)
		return 1
	}))

	L.SetField(dalTbl, "census", censusTbl)
}
