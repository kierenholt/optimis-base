package host

import (
	"context"
	"strings"

	lua "github.com/yuin/gopher-lua"

	"optimis/internal/dal"
)

func (h *Host) bindLuaDALAttendance(
	L *lua.LState,
	execCtx context.Context,
	endpointSpec EndpointSpec,
	caller CallerContext,
	dalTbl *lua.LTable,
) {
	hasAttRead := hasDomainScope(endpointSpec.Scopes, "dal.attendance.read")
	hasAttWrite := hasDomainScope(endpointSpec.Scopes, "dal.attendance.write")
	if !hasAttRead && !hasAttWrite {
		return
	}

	attTbl := L.NewTable()

	if hasAttRead {
		attReadMode := resolveEffectiveDomainScopeMode(endpointSpec.Scopes, caller, "dal.attendance.read")
		L.SetField(attTbl, "get_group_session", L.NewFunction(func(L *lua.LState) int {
			groupID := strings.TrimSpace(L.OptString(1, ""))
			date := strings.TrimSpace(L.OptString(2, ""))
			session := strings.TrimSpace(L.OptString(3, ""))
			var optsTbl *lua.LTable
			if L.GetTop() >= 4 {
				optsTbl, _ = L.Get(4).(*lua.LTable)
			}
			scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, attReadMode, caller.StaffID, optsTbl)
			if sErr != nil {
				L.RaiseError("dal.attendance.get_group_session scope error: %s", sErr.Error())
				return 0
			}
			marks, err := h.coreDAL.GetGroupSessionAttendanceScoped(execCtx, scope, groupID, date, session)
			if err != nil {
				L.RaiseError("dal.attendance.get_group_session failed: %s", err.Error())
				return 0
			}
			arrTbl := L.NewTable()
			markLuaTableAsArray(L, arrTbl)
			for i, m := range marks {
				mTbl := L.NewTable()
				L.SetField(mTbl, "id", lua.LString(m.ID))
				L.SetField(mTbl, "pupil_id", lua.LString(m.PupilID))
				L.SetField(mTbl, "academic_year_id", lua.LString(m.AcademicYearID))
				L.SetField(mTbl, "date", lua.LString(m.Date))
				L.SetField(mTbl, "session", lua.LString(m.Session))
				L.SetField(mTbl, "attendance_code", lua.LString(m.AttendanceCode))
				L.SetField(mTbl, "minutes_late", lua.LNumber(m.MinutesLate))
				L.SetField(mTbl, "sub_code", lua.LString(m.SubCode))
				L.SetField(mTbl, "comment", lua.LString(m.Comment))
				L.SetField(mTbl, "source_group_id", lua.LString(m.SourceGroupID))
				L.SetField(mTbl, "recorded_by_staff_id", lua.LString(m.RecordedByStaffID))
				arrTbl.RawSetInt(i+1, mTbl)
			}
			L.Push(arrTbl)
			return 1
		}))

		L.SetField(attTbl, "get_pupil_summary", L.NewFunction(func(L *lua.LState) int {
			pupilID := strings.TrimSpace(L.OptString(1, ""))
			if pupilID == "" {
				L.RaiseError("missing mandatory parameter: pupil_id")
				return 0
			}
			var optsTbl *lua.LTable
			if L.GetTop() >= 2 {
				optsTbl, _ = L.Get(2).(*lua.LTable)
			}
			scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, attReadMode, caller.StaffID, optsTbl)
			if sErr != nil {
				L.RaiseError("dal.attendance.get_pupil_summary scope error: %s", sErr.Error())
				return 0
			}
			summary, err := h.coreDAL.GetPupilAttendanceSummaryScoped(execCtx, scope, pupilID)
			if err != nil {
				L.RaiseError("dal.attendance.get_pupil_summary failed: %s", err.Error())
				return 0
			}
			sTbl := L.NewTable()
			L.SetField(sTbl, "pupil_id", lua.LString(summary.PupilID))
			L.SetField(sTbl, "academic_year_id", lua.LString(summary.AcademicYearID))
			L.SetField(sTbl, "reference_date", lua.LString(summary.ReferenceDate))
			L.SetField(sTbl, "total_sessions", lua.LNumber(summary.TotalSessions))
			L.SetField(sTbl, "present_sessions", lua.LNumber(summary.PresentSessions))
			L.SetField(sTbl, "am_present_sessions", lua.LNumber(summary.AMPresentSessions))
			L.SetField(sTbl, "pm_present_sessions", lua.LNumber(summary.PMPresentSessions))
			L.Push(sTbl)
			return 1
		}))
	}

	if hasAttWrite {
		attWriteMode := resolveEffectiveDomainScopeMode(endpointSpec.Scopes, caller, "dal.attendance.write")
		upsertSessionBatchFn := L.NewFunction(func(L *lua.LState) int {
			if !endpointSpec.Mutates {
				L.RaiseError("dal.attendance.upsert_session_batch is forbidden when endpoint mutates=false")
				return 0
			}
			var groupID, date, session string
			var marksTbl, optsTbl *lua.LTable
			if firstTbl, isTbl := L.Get(1).(*lua.LTable); isTbl {
				groupID = luaTableTrimmedString(firstTbl, "group_id")
				date = luaTableTrimmedString(firstTbl, "date")
				session = luaTableTrimmedString(firstTbl, "session")
				if m, ok := firstTbl.RawGetString("marks").(*lua.LTable); ok {
					marksTbl = m
				}
				if L.GetTop() >= 2 {
					optsTbl, _ = L.Get(2).(*lua.LTable)
				} else {
					optsTbl = firstTbl
				}
			} else {
				groupID = strings.TrimSpace(L.OptString(1, ""))
				date = strings.TrimSpace(L.OptString(2, ""))
				session = strings.TrimSpace(L.OptString(3, ""))
				var ok bool
				marksTbl, ok = L.Get(4).(*lua.LTable)
				if !ok {
					L.RaiseError("dal.attendance.upsert_session_batch requires a marks table as 4th argument")
					return 0
				}
				if L.GetTop() >= 5 {
					optsTbl, _ = L.Get(5).(*lua.LTable)
				}
			}
			if marksTbl == nil {
				L.RaiseError("dal.attendance.upsert_session_batch requires a marks table")
				return 0
			}
			scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, attWriteMode, caller.StaffID, optsTbl)
			if sErr != nil {
				L.RaiseError("dal.attendance.upsert_session_batch scope error: %s", sErr.Error())
				return 0
			}
			if strings.TrimSpace(date) != "" && (optsTbl == nil || optsTbl.RawGetString("reference_date") == lua.LNil) {
				scope.ReferenceDate = date
			}

			maxn := marksTbl.MaxN()
			marks := make([]dal.SessionAttendanceMarkInput, 0, maxn)
			for i := 1; i <= maxn; i++ {
				itemTbl, ok := marksTbl.RawGetInt(i).(*lua.LTable)
				if !ok {
					L.RaiseError("invalid mark entry at index %d", i)
					return 0
				}
				var minLate int
				if ml := itemTbl.RawGetString("minutes_late"); ml != lua.LNil {
					if num, ok := ml.(lua.LNumber); ok {
						minLate = int(num)
					}
				}
				var subCode, comment string
				if sc := itemTbl.RawGetString("sub_code"); sc != lua.LNil {
					subCode = sc.String()
				}
				if cm := itemTbl.RawGetString("comment"); cm != lua.LNil {
					comment = cm.String()
				}
				marks = append(marks, dal.SessionAttendanceMarkInput{
					PupilID:        luaTableTrimmedString(itemTbl, "pupil_id"),
					AttendanceCode: luaTableTrimmedString(itemTbl, "attendance_code"),
					MinutesLate:    minLate,
					SubCode:        subCode,
					Comment:        comment,
				})
			}

			res, err := h.coreDAL.UpsertSessionAttendanceBatchScoped(execCtx, scope, dal.UpsertSessionAttendanceBatchInput{
				GroupID: groupID,
				Date:    date,
				Session: session,
				Marks:   marks,
			})
			if err != nil {
				L.RaiseError("dal.attendance.upsert_session_batch failed: %s", err.Error())
				return 0
			}

			resTbl := L.NewTable()
			L.SetField(resTbl, "submission_id", lua.LString(res.SubmissionID))
			L.SetField(resTbl, "upserted_count", lua.LNumber(res.UpsertedCount))
			L.SetField(resTbl, "group_id", lua.LString(res.GroupID))
			L.SetField(resTbl, "date", lua.LString(res.Date))
			L.SetField(resTbl, "session", lua.LString(res.Session))
			L.Push(resTbl)
			return 1
		})
		L.SetField(attTbl, "record_session_batch", upsertSessionBatchFn)
		L.SetField(attTbl, "upsert_session_batch", upsertSessionBatchFn)
	}

	L.SetField(dalTbl, "attendance", attTbl)
}
