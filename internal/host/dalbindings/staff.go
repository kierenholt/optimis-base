package dalbindings

import (
	"context"
	"strings"

	lua "github.com/yuin/gopher-lua"

	"optimis/internal/dal"
	"optimis/internal/host/types"
)

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

func staffDetailToLuaTable(L *lua.LState, st dal.StaffMemberDetail) *lua.LTable {
	sTbl := L.NewTable()
	L.SetField(sTbl, "id", lua.LString(st.StaffID))
	L.SetField(sTbl, "staff_id", lua.LString(st.StaffID))
	L.SetField(sTbl, "person_id", lua.LString(st.PersonID))
	L.SetField(sTbl, "staff_code", lua.LString(st.StaffCode))
	L.SetField(sTbl, "display_name", lua.LString(st.DisplayName))
	L.SetField(sTbl, "title", lua.LString(st.Title))
	L.SetField(sTbl, "first_name", lua.LString(st.FirstName))
	L.SetField(sTbl, "last_name", lua.LString(st.LastName))
	L.SetField(sTbl, "preferred_first_name", lua.LString(st.PreferredFirstName))
	L.SetField(sTbl, "preferred_last_name", lua.LString(st.PreferredLastName))
	L.SetField(sTbl, "email", lua.LString(st.Email))
	L.SetField(sTbl, "phone_primary", lua.LString(st.PhonePrimary))
	L.SetField(sTbl, "sex", lua.LString(st.Sex))
	L.SetField(sTbl, "dob", lua.LString(st.DOB))
	L.SetField(sTbl, "status", lua.LString(st.Status))
	L.SetField(sTbl, "job_title", lua.LString(st.JobTitle))
	L.SetField(sTbl, "staff_type", lua.LString(st.StaffType))
	L.SetField(sTbl, "trn", lua.LString(st.TRN))
	L.SetField(sTbl, "qts_flag", lua.LBool(st.QTSFlag))
	L.SetField(sTbl, "eyts_flag", lua.LBool(st.EYTSFlag))
	L.SetField(sTbl, "hlta_flag", lua.LBool(st.HLTAFlag))
	L.SetField(sTbl, "is_dsl", lua.LBool(st.IsDSL))
	L.SetField(sTbl, "exclude_from_cover", lua.LBool(st.ExcludeFromCover))
	L.SetField(sTbl, "cover_priority", lua.LNumber(st.CoverPriority))
	L.SetField(sTbl, "hire_date", lua.LString(st.HireDate))
	L.SetField(sTbl, "termination_date", lua.LString(st.TerminationDate))
	return sTbl
}

func staffRoleToLuaTable(L *lua.LState, r dal.RoleSummary) *lua.LTable {
	rTbl := L.NewTable()
	L.SetField(rTbl, "id", lua.LString(r.ID))
	L.SetField(rTbl, "role_id", lua.LString(r.ID))
	L.SetField(rTbl, "code", lua.LString(r.ID))
	L.SetField(rTbl, "category", lua.LString(r.Category))
	L.SetField(rTbl, "name", lua.LString(r.Name))
	L.SetField(rTbl, "name_short", lua.LString(r.NameShort))
	L.SetField(rTbl, "description", lua.LString(r.Description))
	L.SetField(rTbl, "default_scopes_json", lua.LString(r.DefaultScopesJSON))
	return rTbl
}

func staffCoverAssignmentToLuaTable(L *lua.LState, c dal.StaffCoverAssignmentRecord) *lua.LTable {
	cTbl := L.NewTable()
	L.SetField(cTbl, "id", lua.LString(c.ID))
	L.SetField(cTbl, "cover_id", lua.LString(c.CoverID))
	L.SetField(cTbl, "staff_absence_id", lua.LString(c.StaffAbsenceID))
	L.SetField(cTbl, "absence_id", lua.LString(c.AbsenceID))
	L.SetField(cTbl, "date", lua.LString(c.Date))
	L.SetField(cTbl, "timetable_slot_id", lua.LString(c.TimetableSlotID))
	L.SetField(cTbl, "absent_staff_id", lua.LString(c.AbsentStaffID))
	L.SetField(cTbl, "covering_staff_id", lua.LString(c.CoveringStaffID))
	L.SetField(cTbl, "cover_room_id", lua.LString(c.CoverRoomID))
	L.SetField(cTbl, "room_override_id", lua.LString(c.CoverRoomID))
	L.SetField(cTbl, "status", lua.LString(c.Status))
	L.SetField(cTbl, "cover_notes", lua.LString(c.CoverNotes))
	L.SetField(cTbl, "notes", lua.LString(c.Notes))
	return cTbl
}

func staffSCRToLuaTable(L *lua.LState, rec dal.StaffSCRRecord) *lua.LTable {
	rTbl := L.NewTable()
	L.SetField(rTbl, "id", lua.LString(rec.StaffID))
	L.SetField(rTbl, "staff_id", lua.LString(rec.StaffID))
	L.SetField(rTbl, "person_id", lua.LString(rec.PersonID))
	L.SetField(rTbl, "staff_code", lua.LString(rec.StaffCode))
	L.SetField(rTbl, "display_name", lua.LString(rec.DisplayName))
	L.SetField(rTbl, "first_name", lua.LString(rec.FirstName))
	L.SetField(rTbl, "last_name", lua.LString(rec.LastName))
	L.SetField(rTbl, "job_title", lua.LString(rec.JobTitle))
	L.SetField(rTbl, "staff_type", lua.LString(rec.StaffType))
	L.SetField(rTbl, "is_dsl", lua.LBool(rec.IsDSL))
	L.SetField(rTbl, "trn", lua.LString(rec.TRN))
	L.SetField(rTbl, "qts_flag", lua.LBool(rec.QTSFlag))
	L.SetField(rTbl, "qts_status", lua.LBool(rec.QTSFlag))
	L.SetField(rTbl, "eyts_flag", lua.LBool(rec.EYTSFlag))
	L.SetField(rTbl, "hlta_flag", lua.LBool(rec.HLTAFlag))
	L.SetField(rTbl, "hlta_status", lua.LBool(rec.HLTAFlag))
	L.SetField(rTbl, "dbs_certificate_number", lua.LString(rec.DBSCertificateNumber))
	L.SetField(rTbl, "dbs_check_date", lua.LString(rec.DBSCheckDate))
	L.SetField(rTbl, "prohibition_check_date", lua.LString(rec.ProhibitionCheckDate))
	L.SetField(rTbl, "right_to_work_check_date", lua.LString(rec.RightToWorkCheckDate))
	L.SetField(rTbl, "first_aid_qualified", lua.LBool(rec.FirstAidQualified))
	L.SetField(rTbl, "first_aid_expiry_date", lua.LString(rec.FirstAidExpiryDate))
	L.SetField(rTbl, "hire_date", lua.LString(rec.HireDate))
	L.SetField(rTbl, "employment_start_date", lua.LString(rec.HireDate))
	L.SetField(rTbl, "termination_date", lua.LString(rec.TerminationDate))
	L.SetField(rTbl, "employment_end_date", lua.LString(rec.TerminationDate))
	return rTbl
}

func staffContractToLuaTable(L *lua.LState, c dal.StaffContractRecord) *lua.LTable {
	cTbl := L.NewTable()
	L.SetField(cTbl, "id", lua.LString(c.ContractID))
	L.SetField(cTbl, "contract_id", lua.LString(c.ContractID))
	L.SetField(cTbl, "staff_id", lua.LString(c.StaffID))
	L.SetField(cTbl, "contract_type", lua.LString(c.ContractType))
	L.SetField(cTbl, "swc_post_code", lua.LString(c.SWCPostCode))
	L.SetField(cTbl, "job_title", lua.LString(c.JobTitle))
	L.SetField(cTbl, "pay_scale", lua.LString(c.PayScale))
	L.SetField(cTbl, "pay_spine_point", lua.LString(c.PaySpinePoint))
	L.SetField(cTbl, "fte", lua.LNumber(c.FTE))
	L.SetField(cTbl, "hours_per_week", lua.LNumber(c.HoursPerWeek))
	L.SetField(cTbl, "weeks_per_year", lua.LNumber(c.WeeksPerYear))
	L.SetField(cTbl, "effective_from", lua.LString(c.EffectiveFrom))
	L.SetField(cTbl, "effective_to", lua.LString(c.EffectiveTo))
	return cTbl
}

func staffAbsenceToLuaTable(L *lua.LState, a dal.StaffAbsenceRecord) *lua.LTable {
	aTbl := L.NewTable()
	L.SetField(aTbl, "id", lua.LString(a.AbsenceID))
	L.SetField(aTbl, "absence_id", lua.LString(a.AbsenceID))
	L.SetField(aTbl, "staff_id", lua.LString(a.StaffID))
	L.SetField(aTbl, "academic_year_id", lua.LString(a.AcademicYearID))
	L.SetField(aTbl, "absence_category_code", lua.LString(a.AbsenceCategoryCode))
	L.SetField(aTbl, "reason_detail", lua.LString(a.ReasonDetail))
	L.SetField(aTbl, "status", lua.LString(a.Status))
	L.SetField(aTbl, "cover_required", lua.LBool(a.CoverRequired))
	L.SetField(aTbl, "start_date", lua.LString(a.StartDate))
	L.SetField(aTbl, "end_date", lua.LString(a.EndDate))
	L.SetField(aTbl, "working_days_lost", lua.LNumber(a.WorkingDaysLost))
	return aTbl
}

func (b *Binder) bindLuaDALStaff(
	L *lua.LState,
	execCtx context.Context,
	endpointSpec types.EndpointSpec,
	dalTbl *lua.LTable,
) {
	hasStaffRead := hasDomainScope(endpointSpec.Scopes, "dal.staff.read")
	hasStaffWrite := hasDomainScope(endpointSpec.Scopes, "dal.staff.write")
	if !hasStaffRead && !hasStaffWrite {
		return
	}
	staffTbl := L.NewTable()

	if hasStaffRead {
		L.SetField(staffTbl, "list_active", L.NewFunction(func(L *lua.LState) int {
			staffList, err := b.coreDAL.ListActiveStaff(execCtx)
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
		L.SetField(staffTbl, "list_all", L.NewFunction(func(L *lua.LState) int {
			var opts dal.ListStaffOptions
			if L.GetTop() >= 1 {
				if optsTbl, ok := L.Get(1).(*lua.LTable); ok {
					opts.IncludeLeavers = optsTbl.RawGetString("include_leavers") == lua.LTrue
					opts.StaffType = luaTableTrimmedString(optsTbl, "staff_type")
				}
			}
			staffList, err := b.coreDAL.ListAllStaff(execCtx, opts)
			if err != nil {
				L.RaiseError("dal.staff.list_all failed: %s", err.Error())
				return 0
			}
			arrTbl := L.NewTable()
			markLuaTableAsArray(L, arrTbl)
			for i, st := range staffList {
				arrTbl.RawSetInt(i+1, staffDetailToLuaTable(L, st))
			}
			L.Push(arrTbl)
			return 1
		}))
		L.SetField(staffTbl, "list_roles", L.NewFunction(func(L *lua.LState) int {
			roles, err := b.coreDAL.ListRoles(execCtx)
			if err != nil {
				L.RaiseError("dal.staff.list_roles failed: %s", err.Error())
				return 0
			}
			arrTbl := L.NewTable()
			markLuaTableAsArray(L, arrTbl)
			for i, r := range roles {
				arrTbl.RawSetInt(i+1, staffRoleToLuaTable(L, r))
			}
			L.Push(arrTbl)
			return 1
		}))
		L.SetField(staffTbl, "list_cover_assignments", L.NewFunction(func(L *lua.LState) int {
			var filter dal.ListStaffCoverAssignmentsOptions
			if L.GetTop() >= 1 {
				if optsTbl, ok := L.Get(1).(*lua.LTable); ok {
					filter.Date = luaTableTrimmedString(optsTbl, "date")
					filter.AbsentStaffID = luaTableTrimmedString(optsTbl, "absent_staff_id")
					filter.CoveringStaffID = luaTableTrimmedString(optsTbl, "covering_staff_id")
					filter.AbsenceID = luaTableTrimmedString(optsTbl, "absence_id")
				}
			}
			items, err := b.coreDAL.ListStaffCoverAssignments(execCtx, filter)
			if err != nil {
				L.RaiseError("dal.staff.list_cover_assignments failed: %s", err.Error())
				return 0
			}
			arrTbl := L.NewTable()
			markLuaTableAsArray(L, arrTbl)
			for i, c := range items {
				arrTbl.RawSetInt(i+1, staffCoverAssignmentToLuaTable(L, c))
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
			st, err := b.coreDAL.GetStaffByID(execCtx, staffID)
			if err != nil {
				L.RaiseError("dal.staff.get_by_id failed: %s", err.Error())
				return 0
			}
			L.Push(staffToLuaTable(L, *st))
			return 1
		}))
		L.SetField(staffTbl, "list_scr_register", L.NewFunction(func(L *lua.LState) int {
			records, err := b.coreDAL.ListStaffSCRRegister(execCtx)
			if err != nil {
				L.RaiseError("dal.staff.list_scr_register failed: %s", err.Error())
				return 0
			}
			arrTbl := L.NewTable()
			markLuaTableAsArray(L, arrTbl)
			for i, rec := range records {
				arrTbl.RawSetInt(i+1, staffSCRToLuaTable(L, rec))
			}
			L.Push(arrTbl)
			return 1
		}))
		L.SetField(staffTbl, "list_contracts", L.NewFunction(func(L *lua.LState) int {
			var staffID string
			if L.GetTop() >= 1 {
				if optsTbl, isTbl := L.Get(1).(*lua.LTable); isTbl {
					staffID = luaTableTrimmedString(optsTbl, "staff_id")
				} else {
					staffID = strings.TrimSpace(L.OptString(1, ""))
				}
			}
			contracts, err := b.coreDAL.ListStaffContracts(execCtx, staffID)
			if err != nil {
				L.RaiseError("dal.staff.list_contracts failed: %s", err.Error())
				return 0
			}
			arrTbl := L.NewTable()
			markLuaTableAsArray(L, arrTbl)
			for i, c := range contracts {
				arrTbl.RawSetInt(i+1, staffContractToLuaTable(L, c))
			}
			L.Push(arrTbl)
			return 1
		}))
		L.SetField(staffTbl, "list_absences", L.NewFunction(func(L *lua.LState) int {
			var staffID, ayID string
			if L.GetTop() >= 1 {
				if optsTbl, isTbl := L.Get(1).(*lua.LTable); isTbl {
					staffID = luaTableTrimmedString(optsTbl, "staff_id")
					ayID = luaTableTrimmedString(optsTbl, "academic_year_id")
				} else {
					staffID = strings.TrimSpace(L.OptString(1, ""))
					ayID = strings.TrimSpace(L.OptString(2, ""))
				}
			}
			absences, err := b.coreDAL.ListStaffAbsences(execCtx, staffID, ayID)
			if err != nil {
				L.RaiseError("dal.staff.list_absences failed: %s", err.Error())
				return 0
			}
			arrTbl := L.NewTable()
			markLuaTableAsArray(L, arrTbl)
			for i, a := range absences {
				arrTbl.RawSetInt(i+1, staffAbsenceToLuaTable(L, a))
			}
			L.Push(arrTbl)
			return 1
		}))
	}

	if hasStaffWrite {
		L.SetField(staffTbl, "upsert_staff_member", L.NewFunction(func(L *lua.LState) int {
			if !endpointSpec.Mutates {
				L.RaiseError("dal.staff.upsert_staff_member is forbidden when endpoint mutates=false")
				return 0
			}
			inTbl, ok := L.Get(1).(*lua.LTable)
			if !ok {
				L.RaiseError("dal.staff.upsert_staff_member requires an input table")
				return 0
			}
			var roles []string
			if rTbl, isTbl := inTbl.RawGetString("roles").(*lua.LTable); isTbl {
				for i := 1; i <= rTbl.MaxN(); i++ {
					if v := strings.TrimSpace(rTbl.RawGetInt(i).String()); v != "" {
						roles = append(roles, v)
					}
				}
			}
			staffID := luaTableTrimmedString(inTbl, "staff_id")
			if staffID == "" {
				staffID = luaTableTrimmedString(inTbl, "id")
			}
			firstName := luaTableTrimmedString(inTbl, "first_name")
			if firstName == "" {
				firstName = luaTableTrimmedString(inTbl, "legal_first_name")
			}
			lastName := luaTableTrimmedString(inTbl, "last_name")
			if lastName == "" {
				lastName = luaTableTrimmedString(inTbl, "legal_last_name")
			}
			st, err := b.coreDAL.UpsertStaffMember(execCtx, dal.UpsertStaffMemberInput{
				StaffID:            staffID,
				PersonID:           luaTableTrimmedString(inTbl, "person_id"),
				StaffCode:          luaTableTrimmedString(inTbl, "staff_code"),
				FirstName:          firstName,
				LastName:           lastName,
				PreferredFirstName: luaTableTrimmedString(inTbl, "preferred_first_name"),
				PreferredLastName:  luaTableTrimmedString(inTbl, "preferred_last_name"),
				Title:              luaTableTrimmedString(inTbl, "title"),
				DOB:                luaTableTrimmedString(inTbl, "dob"),
				Sex:                luaTableTrimmedString(inTbl, "sex"),
				Email:              luaTableTrimmedString(inTbl, "email"),
				PhonePrimary:       luaTableTrimmedString(inTbl, "phone_primary"),
				Status:             luaTableTrimmedString(inTbl, "status"),
				StaffType:          luaTableTrimmedString(inTbl, "staff_type"),
				JobTitle:           luaTableTrimmedString(inTbl, "job_title"),
				TRN:                luaTableTrimmedString(inTbl, "trn"),
				QTSFlag:            inTbl.RawGetString("qts_flag") == lua.LTrue,
				EYTSFlag:           inTbl.RawGetString("eyts_flag") == lua.LTrue,
				HLTAFlag:           inTbl.RawGetString("hlta_flag") == lua.LTrue,
				IsDSL:              inTbl.RawGetString("is_dsl") == lua.LTrue,
				ExcludeFromCover:   inTbl.RawGetString("exclude_from_cover") == lua.LTrue,
				CoverPriority:      luaTableInt(inTbl, "cover_priority", 0),
				HireDate:           luaTableTrimmedString(inTbl, "hire_date"),
				TerminationDate:    luaTableTrimmedString(inTbl, "termination_date"),
				Roles:              roles,
			})
			if err != nil {
				L.RaiseError("dal.staff.upsert_staff_member failed: %s", err.Error())
				return 0
			}
			L.Push(staffDetailToLuaTable(L, *st))
			return 1
		}))

		L.SetField(staffTbl, "upsert_cover_assignment", L.NewFunction(func(L *lua.LState) int {
			if !endpointSpec.Mutates {
				L.RaiseError("dal.staff.upsert_cover_assignment is forbidden when endpoint mutates=false")
				return 0
			}
			inTbl, ok := L.Get(1).(*lua.LTable)
			if !ok {
				L.RaiseError("dal.staff.upsert_cover_assignment requires an input table")
				return 0
			}
			coverID := luaTableTrimmedString(inTbl, "cover_id")
			if coverID == "" {
				coverID = luaTableTrimmedString(inTbl, "id")
			}
			coverRoomID := luaTableTrimmedString(inTbl, "cover_room_id")
			if coverRoomID == "" {
				coverRoomID = luaTableTrimmedString(inTbl, "room_override_id")
			}
			coverNotes := luaTableTrimmedString(inTbl, "cover_notes")
			if coverNotes == "" {
				coverNotes = luaTableTrimmedString(inTbl, "notes")
			}
			rec, err := b.coreDAL.UpsertStaffCoverAssignment(execCtx, dal.StaffCoverAssignmentRecord{
				ID:              coverID,
				CoverID:         coverID,
				StaffAbsenceID:  luaTableTrimmedString(inTbl, "staff_absence_id"),
				AbsenceID:       luaTableTrimmedString(inTbl, "absence_id"),
				Date:            luaTableTrimmedString(inTbl, "date"),
				TimetableSlotID: luaTableTrimmedString(inTbl, "timetable_slot_id"),
				AbsentStaffID:   luaTableTrimmedString(inTbl, "absent_staff_id"),
				CoveringStaffID: luaTableTrimmedString(inTbl, "covering_staff_id"),
				CoverRoomID:     coverRoomID,
				Status:          luaTableTrimmedString(inTbl, "status"),
				CoverNotes:      coverNotes,
				Notes:           coverNotes,
			})
			if err != nil {
				L.RaiseError("dal.staff.upsert_cover_assignment failed: %s", err.Error())
				return 0
			}
			L.Push(staffCoverAssignmentToLuaTable(L, *rec))
			return 1
		}))

		L.SetField(staffTbl, "upsert_scr_record", L.NewFunction(func(L *lua.LState) int {
			if !endpointSpec.Mutates {
				L.RaiseError("dal.staff.upsert_scr_record is forbidden when endpoint mutates=false")
				return 0
			}
			inTbl, ok := L.Get(1).(*lua.LTable)
			if !ok {
				L.RaiseError("dal.staff.upsert_scr_record requires an input table")
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
			staffID := luaTableTrimmedString(inTbl, "staff_id")
			if staffID == "" {
				staffID = luaTableTrimmedString(inTbl, "id")
			}
			qtsPtr := optBoolPtr("qts_flag")
			if qtsPtr == nil {
				qtsPtr = optBoolPtr("qts_status")
			}
			hltaPtr := optBoolPtr("hlta_flag")
			if hltaPtr == nil {
				hltaPtr = optBoolPtr("hlta_status")
			}
			rec, err := b.coreDAL.UpsertStaffSCRRecord(execCtx, dal.UpsertStaffSCRInput{
				StaffID:              staffID,
				TRN:                  luaTableTrimmedString(inTbl, "trn"),
				DBSCheckDate:         luaTableTrimmedString(inTbl, "dbs_check_date"),
				DBSCertificateNumber: luaTableTrimmedString(inTbl, "dbs_certificate_number"),
				ProhibitionCheckDate: luaTableTrimmedString(inTbl, "prohibition_check_date"),
				RightToWorkCheckDate: luaTableTrimmedString(inTbl, "right_to_work_check_date"),
				FirstAidQualified:    optBoolPtr("first_aid_qualified"),
				FirstAidExpiryDate:   luaTableTrimmedString(inTbl, "first_aid_expiry_date"),
				QTSFlag:              qtsPtr,
				HLTAFlag:             hltaPtr,
			})
			if err != nil {
				L.RaiseError("dal.staff.upsert_scr_record failed: %s", err.Error())
				return 0
			}
			L.Push(staffSCRToLuaTable(L, *rec))
			return 1
		}))

		L.SetField(staffTbl, "upsert_contract", L.NewFunction(func(L *lua.LState) int {
			if !endpointSpec.Mutates {
				L.RaiseError("dal.staff.upsert_contract is forbidden when endpoint mutates=false")
				return 0
			}
			inTbl, ok := L.Get(1).(*lua.LTable)
			if !ok {
				L.RaiseError("dal.staff.upsert_contract requires an input table")
				return 0
			}
			optFloat := func(key string) float64 {
				v := inTbl.RawGetString(key)
				if n, isNum := v.(lua.LNumber); isNum {
					return float64(n)
				}
				return 0
			}
			contractID := luaTableTrimmedString(inTbl, "contract_id")
			if contractID == "" {
				contractID = luaTableTrimmedString(inTbl, "id")
			}
			rec, err := b.coreDAL.UpsertStaffContract(execCtx, dal.StaffContractRecord{
				ContractID:    contractID,
				StaffID:       luaTableTrimmedString(inTbl, "staff_id"),
				ContractType:  luaTableTrimmedString(inTbl, "contract_type"),
				SWCPostCode:   luaTableTrimmedString(inTbl, "swc_post_code"),
				JobTitle:      luaTableTrimmedString(inTbl, "job_title"),
				PayScale:      luaTableTrimmedString(inTbl, "pay_scale"),
				PaySpinePoint: luaTableTrimmedString(inTbl, "pay_spine_point"),
				FTE:           optFloat("fte"),
				HoursPerWeek:  optFloat("hours_per_week"),
				WeeksPerYear:  optFloat("weeks_per_year"),
				EffectiveFrom: luaTableTrimmedString(inTbl, "effective_from"),
				EffectiveTo:   luaTableTrimmedString(inTbl, "effective_to"),
			})
			if err != nil {
				L.RaiseError("dal.staff.upsert_contract failed: %s", err.Error())
				return 0
			}
			L.Push(staffContractToLuaTable(L, *rec))
			return 1
		}))

		L.SetField(staffTbl, "upsert_absence", L.NewFunction(func(L *lua.LState) int {
			if !endpointSpec.Mutates {
				L.RaiseError("dal.staff.upsert_absence is forbidden when endpoint mutates=false")
				return 0
			}
			inTbl, ok := L.Get(1).(*lua.LTable)
			if !ok {
				L.RaiseError("dal.staff.upsert_absence requires an input table")
				return 0
			}
			var daysLost float64
			if v := inTbl.RawGetString("working_days_lost"); v != lua.LNil {
				if n, isNum := v.(lua.LNumber); isNum {
					daysLost = float64(n)
				}
			}
			absenceID := luaTableTrimmedString(inTbl, "absence_id")
			if absenceID == "" {
				absenceID = luaTableTrimmedString(inTbl, "id")
			}
			rec, err := b.coreDAL.UpsertStaffAbsence(execCtx, dal.StaffAbsenceRecord{
				AbsenceID:           absenceID,
				StaffID:             luaTableTrimmedString(inTbl, "staff_id"),
				AcademicYearID:      luaTableTrimmedString(inTbl, "academic_year_id"),
				AbsenceCategoryCode: luaTableTrimmedString(inTbl, "absence_category_code"),
				ReasonDetail:        luaTableTrimmedString(inTbl, "reason_detail"),
				Status:              luaTableTrimmedString(inTbl, "status"),
				CoverRequired:       inTbl.RawGetString("cover_required") == lua.LTrue,
				StartDate:           luaTableTrimmedString(inTbl, "start_date"),
				EndDate:             luaTableTrimmedString(inTbl, "end_date"),
				WorkingDaysLost:     daysLost,
			})
			if err != nil {
				L.RaiseError("dal.staff.upsert_absence failed: %s", err.Error())
				return 0
			}
			L.Push(staffAbsenceToLuaTable(L, *rec))
			return 1
		}))
	}

	L.SetField(dalTbl, "staff", staffTbl)
}
