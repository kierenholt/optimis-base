package dalbindings

import (
	"context"
	"strings"

	lua "github.com/yuin/gopher-lua"

	"optimis/internal/dal"
	"optimis/internal/host/types"
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

func extractMutationInputAndScope(
	L *lua.LState,
	execCtx context.Context,
	coreDAL *dal.CoreDAL,
	endpointSpec types.EndpointSpec,
	methodName string,
	writeMode dal.ScopeMode,
	staffID string,
) (*lua.LTable, dal.DALQueryScope, bool) {
	if !endpointSpec.Mutates {
		L.RaiseError("%s is forbidden when endpoint mutates=false", methodName)
		return nil, dal.DALQueryScope{}, false
	}
	inTbl, ok := L.Get(1).(*lua.LTable)
	if !ok {
		L.RaiseError("%s requires an input table", methodName)
		return nil, dal.DALQueryScope{}, false
	}
	var optsTbl *lua.LTable
	if L.GetTop() >= 2 {
		optsTbl, _ = L.Get(2).(*lua.LTable)
	}
	scope, sErr := extractScopeFromLuaOpts(execCtx, coreDAL, writeMode, staffID, optsTbl)
	if sErr != nil {
		L.RaiseError("%s scope error: %s", methodName, sErr.Error())
		return nil, dal.DALQueryScope{}, false
	}
	return inTbl, scope, true
}

func extractPupilQueryOpts(L *lua.LState) (pupilID string, academicYearID string, optsTbl *lua.LTable) {
	if L.GetTop() >= 1 {
		if tbl, isTbl := L.Get(1).(*lua.LTable); isTbl {
			optsTbl = tbl
			pupilID = luaTableTrimmedString(tbl, "pupil_id")
			academicYearID = luaTableTrimmedString(tbl, "academic_year_id")
		} else {
			pupilID = strings.TrimSpace(L.OptString(1, ""))
			if L.GetTop() >= 2 {
				optsTbl, _ = L.Get(2).(*lua.LTable)
				if optsTbl != nil && academicYearID == "" {
					academicYearID = luaTableTrimmedString(optsTbl, "academic_year_id")
				}
			}
		}
	}
	return pupilID, academicYearID, optsTbl
}

func (b *Binder) bindLuaDALPupils(
	L *lua.LState,
	execCtx context.Context,
	endpointSpec types.EndpointSpec,
	caller types.CallerContext,
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
				pupil, err = b.coreDAL.GetPupilByID(execCtx, pupilID)
			} else {
				scope, sErr := extractScopeFromLuaOpts(execCtx, b.coreDAL, effectiveMode, caller.StaffID, optsTbl)
				if sErr != nil {
					L.RaiseError("dal.pupils.get_by_id scope error: %s", sErr.Error())
					return 0
				}
				pupil, err = b.coreDAL.GetPupilByIDScoped(execCtx, scope, pupilID)
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
			scope, sErr := extractScopeFromLuaOpts(execCtx, b.coreDAL, effectiveMode, caller.StaffID, optsTbl)
			if sErr != nil {
				L.RaiseError("dal.pupils.get_by_ids_batch scope error: %s", sErr.Error())
				return 0
			}
			pupils, err := b.coreDAL.GetPupilsByIDsBatchScoped(execCtx, scope, pupilIDs)
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
			scope, sErr := extractScopeFromLuaOpts(execCtx, b.coreDAL, dal.ScopeAll, caller.StaffID, optsTbl)
			if sErr != nil {
				L.RaiseError("dal.pupils.list_all scope error: %s", sErr.Error())
				return 0
			}
			includeLeavers := optsTbl != nil && optsTbl.RawGetString("include_leavers") == lua.LTrue
			pupils, err := b.coreDAL.ListAllPupilsWithOptionsScoped(execCtx, scope, dal.ListPupilsOptions{
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
			scope, sErr := extractScopeFromLuaOpts(execCtx, b.coreDAL, effectiveMode, caller.StaffID, optsTbl)
			if sErr != nil {
				L.RaiseError("dal.pupils.list_by_group scope error: %s", sErr.Error())
				return 0
			}
			pupils, err := b.coreDAL.ListPupilsByGroupScoped(execCtx, scope, groupID)
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

		b.bindPupilsPastoralReadMethods(L, execCtx, endpointSpec, caller, pupilsMode, pupilsTbl)
	}

	if hasPupilsWrite {
		L.SetField(pupilsTbl, "upsert_pupil_admission", L.NewFunction(func(L *lua.LState) int {
			inTbl, scope, ok := extractMutationInputAndScope(L, execCtx, b.coreDAL, endpointSpec, "dal.pupils.upsert_pupil_admission", pupilsWriteMode, caller.StaffID)
			if !ok {
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
			pupil, err := b.coreDAL.UpsertPupilAdmissionScoped(execCtx, scope, dal.UpsertPupilAdmissionInput{
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
			inTbl, scope, ok := extractMutationInputAndScope(L, execCtx, b.coreDAL, endpointSpec, "dal.pupils.upsert_year_enrolment", pupilsWriteMode, caller.StaffID)
			if !ok {
				return 0
			}
			enrolmentID := luaTableTrimmedString(inTbl, "enrolment_id")
			if enrolmentID == "" {
				enrolmentID = luaTableTrimmedString(inTbl, "id")
			}
			pupil, err := b.coreDAL.UpsertPupilYearEnrolmentScoped(execCtx, scope, dal.UpsertPupilYearEnrolmentInput{
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

		b.bindPupilsPastoralWriteMethods(L, execCtx, endpointSpec, caller, pupilsWriteMode, pupilsTbl)
	}

	L.SetField(pupilsTbl, "list_sen_and_medical_batch", b.createSenMedicalBatchFn(L, execCtx, endpointSpec, caller))
	if hasPupilsRead || hasSenRead {
		L.SetField(pupilsTbl, "list_sen_register", b.createSenRegisterFn(L, execCtx, endpointSpec, caller))
	}
	L.SetField(dalTbl, "pupils", pupilsTbl)
}
