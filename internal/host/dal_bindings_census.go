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

		rollup, err := h.coreDAL.GetStatutoryCensusRollup(execCtx, dal.StatutoryCensusRollupInput{
			AcademicYearID:      luaTableTrimmedString(inTbl, "academic_year_id"),
			CensusDate:          luaTableTrimmedString(inTbl, "census_date"),
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
