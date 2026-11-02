package host

import (
	"context"

	lua "github.com/yuin/gopher-lua"

	"optimis/internal/dal"
)

func (h *Host) bindLuaDALAcademic(
	L *lua.LState,
	execCtx context.Context,
	endpointSpec EndpointSpec,
	caller CallerContext,
	dalTbl *lua.LTable,
) {
	hasAcadRead := hasDomainScope(endpointSpec.Scopes, "dal.academic.read")
	hasAcadWrite := hasDomainScope(endpointSpec.Scopes, "dal.academic.write")
	if !hasAcadRead && !hasAcadWrite {
		return
	}

	acadTbl := L.NewTable()

	if hasAcadRead {
		L.SetField(acadTbl, "list_years", L.NewFunction(func(L *lua.LState) int {
			years, err := h.coreDAL.ListAcademicYears(execCtx)
			if err != nil {
				L.RaiseError("dal.academic.list_years failed: %s", err.Error())
				return 0
			}
			arrTbl := L.NewTable()
			markLuaTableAsArray(L, arrTbl)
			for idx, y := range years {
				yTbl := L.NewTable()
				L.SetField(yTbl, "id", lua.LString(y.ID))
				L.SetField(yTbl, "academic_year_id", lua.LString(y.AcademicYearID))
				L.SetField(yTbl, "name", lua.LString(y.Name))
				L.SetField(yTbl, "status", lua.LString(y.Status))
				L.SetField(yTbl, "sequence_number", lua.LNumber(y.SequenceNumber))
				L.SetField(yTbl, "first_day", lua.LString(y.FirstDay))
				L.SetField(yTbl, "last_day", lua.LString(y.LastDay))
				arrTbl.RawSetInt(idx+1, yTbl)
			}
			L.Push(arrTbl)
			return 1
		}))

		L.SetField(acadTbl, "list_terms", L.NewFunction(func(L *lua.LState) int {
			var ayID string
			if L.GetTop() >= 1 {
				if inTbl, ok := L.Get(1).(*lua.LTable); ok {
					ayID = luaTableTrimmedString(inTbl, "academic_year_id")
				} else {
					ayID = L.OptString(1, "")
				}
			}
			terms, err := h.coreDAL.ListAcademicTerms(execCtx, ayID)
			if err != nil {
				L.RaiseError("dal.academic.list_terms failed: %s", err.Error())
				return 0
			}
			arrTbl := L.NewTable()
			markLuaTableAsArray(L, arrTbl)
			for idx, t := range terms {
				tTbl := L.NewTable()
				L.SetField(tTbl, "id", lua.LString(t.ID))
				L.SetField(tTbl, "term_id", lua.LString(t.TermID))
				L.SetField(tTbl, "academic_year_id", lua.LString(t.AcademicYearID))
				L.SetField(tTbl, "code", lua.LString(t.Code))
				L.SetField(tTbl, "term_code", lua.LString(t.TermCode))
				L.SetField(tTbl, "dfe_term_code", lua.LString(t.Code))
				L.SetField(tTbl, "name", lua.LString(t.Name))
				L.SetField(tTbl, "name_short", lua.LString(t.NameShort))
				L.SetField(tTbl, "first_day", lua.LString(t.FirstDay))
				L.SetField(tTbl, "last_day", lua.LString(t.LastDay))
				L.SetField(tTbl, "sequence_number", lua.LNumber(t.SequenceNumber))
				arrTbl.RawSetInt(idx+1, tTbl)
			}
			L.Push(arrTbl)
			return 1
		}))

		L.SetField(acadTbl, "list_days_of_week", L.NewFunction(func(L *lua.LState) int {
			days, err := h.coreDAL.ListDaysOfWeek(execCtx)
			if err != nil {
				L.RaiseError("dal.academic.list_days_of_week failed: %s", err.Error())
				return 0
			}
			arrTbl := L.NewTable()
			markLuaTableAsArray(L, arrTbl)
			for idx, d := range days {
				dTbl := L.NewTable()
				L.SetField(dTbl, "id", lua.LNumber(d.ID))
				L.SetField(dTbl, "name", lua.LString(d.Name))
				L.SetField(dTbl, "name_short", lua.LString(d.NameShort))
				L.SetField(dTbl, "sequence_number", lua.LNumber(d.ID))
				L.SetField(dTbl, "is_school_day", lua.LBool(d.IsSchoolDay))
				L.SetField(dTbl, "school_open_time", lua.LString(d.SchoolOpenTime))
				L.SetField(dTbl, "am_register_time", lua.LString(d.AMRegisterTime))
				L.SetField(dTbl, "school_start_time", lua.LString(d.AMRegisterTime))
				L.SetField(dTbl, "pm_register_time", lua.LString(d.PMRegisterTime))
				L.SetField(dTbl, "school_end_time", lua.LString(d.PMRegisterTime))
				L.SetField(dTbl, "school_close_time", lua.LString(d.SchoolCloseTime))
				arrTbl.RawSetInt(idx+1, dTbl)
			}
			L.Push(arrTbl)
			return 1
		}))

		L.SetField(acadTbl, "list_special_days", L.NewFunction(func(L *lua.LState) int {
			var ayID string
			if L.GetTop() >= 1 {
				if inTbl, ok := L.Get(1).(*lua.LTable); ok {
					ayID = luaTableTrimmedString(inTbl, "academic_year_id")
				} else {
					ayID = L.OptString(1, "")
				}
			}
			days, err := h.coreDAL.ListAcademicSpecialDays(execCtx, ayID)
			if err != nil {
				L.RaiseError("dal.academic.list_special_days failed: %s", err.Error())
				return 0
			}
			arrTbl := L.NewTable()
			markLuaTableAsArray(L, arrTbl)
			for idx, sd := range days {
				sdTbl := L.NewTable()
				L.SetField(sdTbl, "id", lua.LString(sd.ID))
				L.SetField(sdTbl, "special_day_id", lua.LString(sd.SpecialDayID))
				L.SetField(sdTbl, "academic_year_id", lua.LString(sd.AcademicYearID))
				L.SetField(sdTbl, "academic_term_id", lua.LString(sd.AcademicTermID))
				L.SetField(sdTbl, "date", lua.LString(sd.Date))
				L.SetField(sdTbl, "type", lua.LString(sd.Type))
				L.SetField(sdTbl, "day_type", lua.LString(sd.DayType))
				L.SetField(sdTbl, "name", lua.LString(sd.Name))
				L.SetField(sdTbl, "description", lua.LString(sd.Description))
				arrTbl.RawSetInt(idx+1, sdTbl)
			}
			L.Push(arrTbl)
			return 1
		}))

		L.SetField(acadTbl, "list_alert_levels", L.NewFunction(func(L *lua.LState) int {
			levels, err := h.coreDAL.ListAlertLevels(execCtx)
			if err != nil {
				L.RaiseError("dal.academic.list_alert_levels failed: %s", err.Error())
				return 0
			}
			arrTbl := L.NewTable()
			markLuaTableAsArray(L, arrTbl)
			for idx, al := range levels {
				alTbl := L.NewTable()
				L.SetField(alTbl, "id", lua.LString(al.ID))
				L.SetField(alTbl, "code", lua.LString(al.Code))
				L.SetField(alTbl, "name", lua.LString(al.Name))
				L.SetField(alTbl, "name_short", lua.LString(al.NameShort))
				L.SetField(alTbl, "color_hex", lua.LString(al.ColorHex))
				L.SetField(alTbl, "colour_hex", lua.LString(al.ColourHex))
				L.SetField(alTbl, "bg_color_hex", lua.LString(al.BgColorHex))
				L.SetField(alTbl, "description", lua.LString(al.Description))
				L.SetField(alTbl, "sequence_number", lua.LNumber(al.SequenceNumber))
				arrTbl.RawSetInt(idx+1, alTbl)
			}
			L.Push(arrTbl)
			return 1
		}))
	}

	if hasAcadWrite {
		L.SetField(acadTbl, "upsert_year", L.NewFunction(func(L *lua.LState) int {
			if !endpointSpec.Mutates {
				L.RaiseError("dal.academic.upsert_year is forbidden when endpoint mutates=false")
				return 0
			}
			inTbl, ok := L.Get(1).(*lua.LTable)
			if !ok {
				L.RaiseError("dal.academic.upsert_year requires an input table")
				return 0
			}
			res, err := h.coreDAL.UpsertAcademicYear(execCtx, dal.AcademicYearSummary{
				ID:             luaTableTrimmedString(inTbl, "id"),
				AcademicYearID: luaTableTrimmedString(inTbl, "academic_year_id"),
				Name:           luaTableTrimmedString(inTbl, "name"),
				Status:         luaTableTrimmedString(inTbl, "status"),
				SequenceNumber: luaTableInt(inTbl, "sequence_number", 0),
				FirstDay:       luaTableTrimmedString(inTbl, "first_day"),
				LastDay:        luaTableTrimmedString(inTbl, "last_day"),
			})
			if err != nil {
				L.RaiseError("dal.academic.upsert_year failed: %s", err.Error())
				return 0
			}
			yTbl := L.NewTable()
			L.SetField(yTbl, "id", lua.LString(res.ID))
			L.SetField(yTbl, "academic_year_id", lua.LString(res.AcademicYearID))
			L.SetField(yTbl, "name", lua.LString(res.Name))
			L.SetField(yTbl, "status", lua.LString(res.Status))
			L.SetField(yTbl, "sequence_number", lua.LNumber(res.SequenceNumber))
			L.SetField(yTbl, "first_day", lua.LString(res.FirstDay))
			L.SetField(yTbl, "last_day", lua.LString(res.LastDay))
			L.Push(yTbl)
			return 1
		}))

		L.SetField(acadTbl, "upsert_term", L.NewFunction(func(L *lua.LState) int {
			if !endpointSpec.Mutates {
				L.RaiseError("dal.academic.upsert_term is forbidden when endpoint mutates=false")
				return 0
			}
			inTbl, ok := L.Get(1).(*lua.LTable)
			if !ok {
				L.RaiseError("dal.academic.upsert_term requires an input table")
				return 0
			}
			code := luaTableTrimmedString(inTbl, "code")
			if code == "" {
				code = luaTableTrimmedString(inTbl, "term_code")
			}
			if code == "" {
				code = luaTableTrimmedString(inTbl, "dfe_term_code")
			}
			res, err := h.coreDAL.UpsertAcademicTerm(execCtx, dal.AcademicTermSummary{
				ID:             luaTableTrimmedString(inTbl, "id"),
				TermID:         luaTableTrimmedString(inTbl, "term_id"),
				AcademicYearID: luaTableTrimmedString(inTbl, "academic_year_id"),
				Code:           code,
				TermCode:       code,
				Name:           luaTableTrimmedString(inTbl, "name"),
				NameShort:      luaTableTrimmedString(inTbl, "name_short"),
				FirstDay:       luaTableTrimmedString(inTbl, "first_day"),
				LastDay:        luaTableTrimmedString(inTbl, "last_day"),
				SequenceNumber: luaTableInt(inTbl, "sequence_number", 0),
			})
			if err != nil {
				L.RaiseError("dal.academic.upsert_term failed: %s", err.Error())
				return 0
			}
			tTbl := L.NewTable()
			L.SetField(tTbl, "id", lua.LString(res.ID))
			L.SetField(tTbl, "term_id", lua.LString(res.TermID))
			L.SetField(tTbl, "academic_year_id", lua.LString(res.AcademicYearID))
			L.SetField(tTbl, "code", lua.LString(res.Code))
			L.SetField(tTbl, "term_code", lua.LString(res.TermCode))
			L.SetField(tTbl, "dfe_term_code", lua.LString(res.Code))
			L.SetField(tTbl, "name", lua.LString(res.Name))
			L.SetField(tTbl, "name_short", lua.LString(res.NameShort))
			L.SetField(tTbl, "first_day", lua.LString(res.FirstDay))
			L.SetField(tTbl, "last_day", lua.LString(res.LastDay))
			L.SetField(tTbl, "sequence_number", lua.LNumber(res.SequenceNumber))
			L.Push(tTbl)
			return 1
		}))

		L.SetField(acadTbl, "upsert_day_of_week", L.NewFunction(func(L *lua.LState) int {
			if !endpointSpec.Mutates {
				L.RaiseError("dal.academic.upsert_day_of_week is forbidden when endpoint mutates=false")
				return 0
			}
			inTbl, ok := L.Get(1).(*lua.LTable)
			if !ok {
				L.RaiseError("dal.academic.upsert_day_of_week requires an input table")
				return 0
			}
			amReg := luaTableTrimmedString(inTbl, "am_register_time")
			if amReg == "" {
				amReg = luaTableTrimmedString(inTbl, "school_start_time")
			}
			pmReg := luaTableTrimmedString(inTbl, "pm_register_time")
			if pmReg == "" {
				pmReg = luaTableTrimmedString(inTbl, "school_end_time")
			}
			res, err := h.coreDAL.UpsertDayOfWeek(execCtx, dal.DayOfWeekSummary{
				ID:              luaTableInt(inTbl, "id", 0),
				Name:            luaTableTrimmedString(inTbl, "name"),
				NameShort:       luaTableTrimmedString(inTbl, "name_short"),
				IsSchoolDay:     inTbl.RawGetString("is_school_day") == lua.LTrue,
				SchoolOpenTime:  luaTableTrimmedString(inTbl, "school_open_time"),
				AMRegisterTime:  amReg,
				PMRegisterTime:  pmReg,
				SchoolCloseTime: luaTableTrimmedString(inTbl, "school_close_time"),
			})
			if err != nil {
				L.RaiseError("dal.academic.upsert_day_of_week failed: %s", err.Error())
				return 0
			}
			dTbl := L.NewTable()
			L.SetField(dTbl, "id", lua.LNumber(res.ID))
			L.SetField(dTbl, "name", lua.LString(res.Name))
			L.SetField(dTbl, "name_short", lua.LString(res.NameShort))
			L.SetField(dTbl, "sequence_number", lua.LNumber(res.ID))
			L.SetField(dTbl, "is_school_day", lua.LBool(res.IsSchoolDay))
			L.SetField(dTbl, "school_open_time", lua.LString(res.SchoolOpenTime))
			L.SetField(dTbl, "am_register_time", lua.LString(res.AMRegisterTime))
			L.SetField(dTbl, "school_start_time", lua.LString(res.AMRegisterTime))
			L.SetField(dTbl, "pm_register_time", lua.LString(res.PMRegisterTime))
			L.SetField(dTbl, "school_end_time", lua.LString(res.PMRegisterTime))
			L.SetField(dTbl, "school_close_time", lua.LString(res.SchoolCloseTime))
			L.Push(dTbl)
			return 1
		}))

		L.SetField(acadTbl, "upsert_special_day", L.NewFunction(func(L *lua.LState) int {
			if !endpointSpec.Mutates {
				L.RaiseError("dal.academic.upsert_special_day is forbidden when endpoint mutates=false")
				return 0
			}
			inTbl, ok := L.Get(1).(*lua.LTable)
			if !ok {
				L.RaiseError("dal.academic.upsert_special_day requires an input table")
				return 0
			}
			res, err := h.coreDAL.UpsertAcademicSpecialDay(execCtx, dal.AcademicSpecialDaySummary{
				ID:             luaTableTrimmedString(inTbl, "id"),
				SpecialDayID:   luaTableTrimmedString(inTbl, "special_day_id"),
				AcademicYearID: luaTableTrimmedString(inTbl, "academic_year_id"),
				AcademicTermID: luaTableTrimmedString(inTbl, "academic_term_id"),
				Date:           luaTableTrimmedString(inTbl, "date"),
				Type:           luaTableTrimmedString(inTbl, "type"),
				DayType:        luaTableTrimmedString(inTbl, "day_type"),
				Name:           luaTableTrimmedString(inTbl, "name"),
				Description:    luaTableTrimmedString(inTbl, "description"),
			})
			if err != nil {
				L.RaiseError("dal.academic.upsert_special_day failed: %s", err.Error())
				return 0
			}
			sdTbl := L.NewTable()
			L.SetField(sdTbl, "id", lua.LString(res.ID))
			L.SetField(sdTbl, "special_day_id", lua.LString(res.SpecialDayID))
			L.SetField(sdTbl, "academic_year_id", lua.LString(res.AcademicYearID))
			L.SetField(sdTbl, "academic_term_id", lua.LString(res.AcademicTermID))
			L.SetField(sdTbl, "date", lua.LString(res.Date))
			L.SetField(sdTbl, "type", lua.LString(res.Type))
			L.SetField(sdTbl, "day_type", lua.LString(res.DayType))
			L.SetField(sdTbl, "name", lua.LString(res.Name))
			L.SetField(sdTbl, "description", lua.LString(res.Description))
			L.Push(sdTbl)
			return 1
		}))
	}

	L.SetField(dalTbl, "academic", acadTbl)
}
