package dalbindings

import (
	"context"
	"strings"

	lua "github.com/yuin/gopher-lua"

	"optimis/internal/dal"
	"optimis/internal/host/types"
)

func attendanceCodeToLuaTable(L *lua.LState, c dal.AttendanceCodeSummary) *lua.LTable {
	cTbl := L.NewTable()
	L.SetField(cTbl, "code", lua.LString(c.Code))
	L.SetField(cTbl, "dfe_code", lua.LString(c.Code))
	L.SetField(cTbl, "name", lua.LString(c.Name))
	L.SetField(cTbl, "description", lua.LString(c.Description))
	L.SetField(cTbl, "statistical_category", lua.LString(c.StatisticalCategory))
	L.SetField(cTbl, "statistical_meaning", lua.LString(c.StatisticalMeaning))
	L.SetField(cTbl, "physical_status", lua.LString(c.PhysicalStatus))
	L.SetField(cTbl, "is_statutory_code", lua.LBool(c.IsStatutoryCode))
	L.SetField(cTbl, "is_present_for_fire_drill", lua.LBool(c.PhysicalStatus == "ONSITE"))
	L.SetField(cTbl, "is_active", lua.LBool(c.IsActive))
	L.SetField(cTbl, "restricted_to_admin", lua.LBool(c.RestrictedToAdmin))
	L.SetField(cTbl, "sequence_number", lua.LNumber(c.SequenceNumber))
	return cTbl
}

func sessionAttendanceMarkToLuaTable(L *lua.LState, m dal.SessionAttendanceMark) *lua.LTable {
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
	return mTbl
}

func lessonAttendanceMarkToLuaTable(L *lua.LState, m dal.LessonAttendanceMark) *lua.LTable {
	mTbl := L.NewTable()
	L.SetField(mTbl, "id", lua.LString(m.ID))
	L.SetField(mTbl, "pupil_id", lua.LString(m.PupilID))
	L.SetField(mTbl, "group_id", lua.LString(m.GroupID))
	L.SetField(mTbl, "timetable_slot_id", lua.LString(m.TimetableSlotID))
	L.SetField(mTbl, "date", lua.LString(m.Date))
	L.SetField(mTbl, "attendance_code", lua.LString(m.AttendanceCode))
	L.SetField(mTbl, "minutes_late", lua.LNumber(m.MinutesLate))
	L.SetField(mTbl, "comment", lua.LString(m.Comment))
	L.SetField(mTbl, "recorded_by_staff_id", lua.LString(m.RecordedByStaffID))
	return mTbl
}

func registerSubmissionToLuaTable(L *lua.LState, s dal.AttendanceRegisterSubmissionSummary) *lua.LTable {
	sTbl := L.NewTable()
	L.SetField(sTbl, "id", lua.LString(s.ID))
	L.SetField(sTbl, "submission_id", lua.LString(s.SubmissionID))
	L.SetField(sTbl, "group_id", lua.LString(s.GroupID))
	L.SetField(sTbl, "date", lua.LString(s.Date))
	L.SetField(sTbl, "session_or_slot_key", lua.LString(s.SessionOrSlotKey))
	L.SetField(sTbl, "session", lua.LString(s.SessionOrSlotKey))
	L.SetField(sTbl, "taken_by_staff_id", lua.LString(s.TakenByStaffID))
	L.SetField(sTbl, "submitted_by_staff_id", lua.LString(s.TakenByStaffID))
	L.SetField(sTbl, "taken_at", lua.LString(s.TakenAt))
	L.SetField(sTbl, "submitted_at", lua.LString(s.TakenAt))
	return sTbl
}

func (b *Binder) bindLuaDALAttendance(
	L *lua.LState,
	execCtx context.Context,
	endpointSpec types.EndpointSpec,
	caller types.CallerContext,
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

		L.SetField(attTbl, "list_codes", L.NewFunction(func(L *lua.LState) int {
			codes, err := b.coreDAL.ListAttendanceCodes(execCtx)
			if err != nil {
				L.RaiseError("dal.attendance.list_codes failed: %s", err.Error())
				return 0
			}
			arrTbl := L.NewTable()
			markLuaTableAsArray(L, arrTbl)
			for i, c := range codes {
				arrTbl.RawSetInt(i+1, attendanceCodeToLuaTable(L, c))
			}
			L.Push(arrTbl)
			return 1
		}))

		L.SetField(attTbl, "get_group_session", L.NewFunction(func(L *lua.LState) int {
			groupID := strings.TrimSpace(L.OptString(1, ""))
			date := strings.TrimSpace(L.OptString(2, ""))
			session := strings.TrimSpace(L.OptString(3, ""))
			var optsTbl *lua.LTable
			if L.GetTop() >= 4 {
				optsTbl, _ = L.Get(4).(*lua.LTable)
			}
			scope, sErr := extractScopeFromLuaOpts(execCtx, b.coreDAL, attReadMode, caller.StaffID, optsTbl)
			if sErr != nil {
				L.RaiseError("dal.attendance.get_group_session scope error: %s", sErr.Error())
				return 0
			}
			marks, err := b.coreDAL.GetGroupSessionAttendanceScoped(execCtx, scope, groupID, date, session)
			if err != nil {
				L.RaiseError("dal.attendance.get_group_session failed: %s", err.Error())
				return 0
			}
			arrTbl := L.NewTable()
			markLuaTableAsArray(L, arrTbl)
			for i, m := range marks {
				arrTbl.RawSetInt(i+1, sessionAttendanceMarkToLuaTable(L, m))
			}
			L.Push(arrTbl)
			return 1
		}))

		L.SetField(attTbl, "get_group_lesson", L.NewFunction(func(L *lua.LState) int {
			var groupID, slotID, date string
			var optsTbl *lua.LTable
			if firstTbl, isTbl := L.Get(1).(*lua.LTable); isTbl {
				groupID = luaTableTrimmedString(firstTbl, "group_id")
				slotID = luaTableTrimmedString(firstTbl, "timetable_slot_id")
				date = luaTableTrimmedString(firstTbl, "date")
				if L.GetTop() >= 2 {
					optsTbl, _ = L.Get(2).(*lua.LTable)
				} else {
					optsTbl = firstTbl
				}
			} else {
				groupID = strings.TrimSpace(L.OptString(1, ""))
				slotID = strings.TrimSpace(L.OptString(2, ""))
				date = strings.TrimSpace(L.OptString(3, ""))
				if L.GetTop() >= 4 {
					optsTbl, _ = L.Get(4).(*lua.LTable)
				}
			}
			scope, sErr := extractScopeFromLuaOpts(execCtx, b.coreDAL, attReadMode, caller.StaffID, optsTbl)
			if sErr != nil {
				L.RaiseError("dal.attendance.get_group_lesson scope error: %s", sErr.Error())
				return 0
			}
			marks, err := b.coreDAL.GetGroupLessonAttendanceScoped(execCtx, scope, groupID, slotID, date)
			if err != nil {
				L.RaiseError("dal.attendance.get_group_lesson failed: %s", err.Error())
				return 0
			}
			arrTbl := L.NewTable()
			markLuaTableAsArray(L, arrTbl)
			for i, m := range marks {
				arrTbl.RawSetInt(i+1, lessonAttendanceMarkToLuaTable(L, m))
			}
			L.Push(arrTbl)
			return 1
		}))

		L.SetField(attTbl, "list_session_marks", L.NewFunction(func(L *lua.LState) int {
			var optsTbl *lua.LTable
			var filter dal.ListSessionMarksFilter
			if L.GetTop() >= 1 {
				if tbl, isTbl := L.Get(1).(*lua.LTable); isTbl {
					optsTbl = tbl
					filter.PupilID = luaTableTrimmedString(tbl, "pupil_id")
					filter.GroupID = luaTableTrimmedString(tbl, "group_id")
					filter.Date = luaTableTrimmedString(tbl, "date")
					filter.Session = luaTableTrimmedString(tbl, "session")
				}
			}
			scope, sErr := extractScopeFromLuaOpts(execCtx, b.coreDAL, attReadMode, caller.StaffID, optsTbl)
			if sErr != nil {
				L.RaiseError("dal.attendance.list_session_marks scope error: %s", sErr.Error())
				return 0
			}
			marks, err := b.coreDAL.ListSessionAttendanceMarksScoped(execCtx, scope, filter)
			if err != nil {
				L.RaiseError("dal.attendance.list_session_marks failed: %s", err.Error())
				return 0
			}
			arrTbl := L.NewTable()
			markLuaTableAsArray(L, arrTbl)
			for i, m := range marks {
				arrTbl.RawSetInt(i+1, sessionAttendanceMarkToLuaTable(L, m))
			}
			L.Push(arrTbl)
			return 1
		}))

		L.SetField(attTbl, "list_register_submissions", L.NewFunction(func(L *lua.LState) int {
			var optsTbl *lua.LTable
			var date, groupID string
			if L.GetTop() >= 1 {
				if tbl, isTbl := L.Get(1).(*lua.LTable); isTbl {
					optsTbl = tbl
					date = luaTableTrimmedString(tbl, "date")
					groupID = luaTableTrimmedString(tbl, "group_id")
				}
			}
			scope, sErr := extractScopeFromLuaOpts(execCtx, b.coreDAL, attReadMode, caller.StaffID, optsTbl)
			if sErr != nil {
				L.RaiseError("dal.attendance.list_register_submissions scope error: %s", sErr.Error())
				return 0
			}
			subs, err := b.coreDAL.ListAttendanceRegisterSubmissionsScoped(execCtx, scope, date, groupID)
			if err != nil {
				L.RaiseError("dal.attendance.list_register_submissions failed: %s", err.Error())
				return 0
			}
			arrTbl := L.NewTable()
			markLuaTableAsArray(L, arrTbl)
			for i, s := range subs {
				arrTbl.RawSetInt(i+1, registerSubmissionToLuaTable(L, s))
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
			scope, sErr := extractScopeFromLuaOpts(execCtx, b.coreDAL, attReadMode, caller.StaffID, optsTbl)
			if sErr != nil {
				L.RaiseError("dal.attendance.get_pupil_summary scope error: %s", sErr.Error())
				return 0
			}
			summary, err := b.coreDAL.GetPupilAttendanceSummaryScoped(execCtx, scope, pupilID)
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

		L.SetField(attTbl, "upsert_code", L.NewFunction(func(L *lua.LState) int {
			if !endpointSpec.Mutates {
				L.RaiseError("dal.attendance.upsert_code is forbidden when endpoint mutates=false")
				return 0
			}
			inTbl, ok := L.Get(1).(*lua.LTable)
			if !ok {
				L.RaiseError("dal.attendance.upsert_code requires an input table")
				return 0
			}
			var optsTbl *lua.LTable
			if L.GetTop() >= 2 {
				optsTbl, _ = L.Get(2).(*lua.LTable)
			}
			scope, sErr := extractScopeFromLuaOpts(execCtx, b.coreDAL, attWriteMode, caller.StaffID, optsTbl)
			if sErr != nil {
				L.RaiseError("dal.attendance.upsert_code scope error: %s", sErr.Error())
				return 0
			}
			isActive := true
			if v := inTbl.RawGetString("is_active"); v != lua.LNil {
				isActive = v == lua.LTrue
			}
			code := luaTableTrimmedString(inTbl, "code")
			if code == "" {
				code = luaTableTrimmedString(inTbl, "dfe_code")
			}
			c, err := b.coreDAL.UpsertAttendanceCodeScoped(execCtx, scope, dal.AttendanceCodeSummary{
				Code:                code,
				Name:                luaTableTrimmedString(inTbl, "name"),
				Description:         luaTableTrimmedString(inTbl, "description"),
				StatisticalCategory: luaTableTrimmedString(inTbl, "statistical_category"),
				StatisticalMeaning:  luaTableTrimmedString(inTbl, "statistical_meaning"),
				PhysicalStatus:      luaTableTrimmedString(inTbl, "physical_status"),
				IsStatutoryCode:     inTbl.RawGetString("is_statutory_code") == lua.LTrue,
				IsActive:            isActive,
				RestrictedToAdmin:   inTbl.RawGetString("restricted_to_admin") == lua.LTrue,
				SequenceNumber:      luaTableInt(inTbl, "sequence_number", 0),
			})
			if err != nil {
				L.RaiseError("dal.attendance.upsert_code failed: %s", err.Error())
				return 0
			}
			L.Push(attendanceCodeToLuaTable(L, *c))
			return 1
		}))

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
			scope, sErr := extractScopeFromLuaOpts(execCtx, b.coreDAL, attWriteMode, caller.StaffID, optsTbl)
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

			res, err := b.coreDAL.UpsertSessionAttendanceBatchScoped(execCtx, scope, dal.UpsertSessionAttendanceBatchInput{
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

		upsertLessonBatchFn := L.NewFunction(func(L *lua.LState) int {
			if !endpointSpec.Mutates {
				L.RaiseError("dal.attendance.upsert_lesson_batch is forbidden when endpoint mutates=false")
				return 0
			}
			var groupID, slotID, date string
			var marksTbl, optsTbl *lua.LTable
			if firstTbl, isTbl := L.Get(1).(*lua.LTable); isTbl {
				groupID = luaTableTrimmedString(firstTbl, "group_id")
				slotID = luaTableTrimmedString(firstTbl, "timetable_slot_id")
				date = luaTableTrimmedString(firstTbl, "date")
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
				slotID = strings.TrimSpace(L.OptString(2, ""))
				date = strings.TrimSpace(L.OptString(3, ""))
				var ok bool
				marksTbl, ok = L.Get(4).(*lua.LTable)
				if !ok {
					L.RaiseError("dal.attendance.upsert_lesson_batch requires a marks table as 4th argument")
					return 0
				}
				if L.GetTop() >= 5 {
					optsTbl, _ = L.Get(5).(*lua.LTable)
				}
			}
			if marksTbl == nil {
				L.RaiseError("dal.attendance.upsert_lesson_batch requires a marks table")
				return 0
			}
			scope, sErr := extractScopeFromLuaOpts(execCtx, b.coreDAL, attWriteMode, caller.StaffID, optsTbl)
			if sErr != nil {
				L.RaiseError("dal.attendance.upsert_lesson_batch scope error: %s", sErr.Error())
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
					L.RaiseError("invalid lesson mark entry at index %d", i)
					return 0
				}
				marks = append(marks, dal.SessionAttendanceMarkInput{
					PupilID:        luaTableTrimmedString(itemTbl, "pupil_id"),
					AttendanceCode: luaTableTrimmedString(itemTbl, "attendance_code"),
					MinutesLate:    luaTableInt(itemTbl, "minutes_late", 0),
					Comment:        luaTableTrimmedString(itemTbl, "comment"),
				})
			}

			res, err := b.coreDAL.UpsertLessonAttendanceBatchScoped(execCtx, scope, dal.UpsertLessonAttendanceBatchInput{
				GroupID:         groupID,
				TimetableSlotID: slotID,
				Date:            date,
				Marks:           marks,
			})
			if err != nil {
				L.RaiseError("dal.attendance.upsert_lesson_batch failed: %s", err.Error())
				return 0
			}

			resTbl := L.NewTable()
			L.SetField(resTbl, "submission_id", lua.LString(res.SubmissionID))
			L.SetField(resTbl, "upserted_count", lua.LNumber(res.UpsertedCount))
			L.SetField(resTbl, "group_id", lua.LString(res.GroupID))
			L.SetField(resTbl, "timetable_slot_id", lua.LString(res.TimetableSlotID))
			L.SetField(resTbl, "date", lua.LString(res.Date))
			L.Push(resTbl)
			return 1
		})
		L.SetField(attTbl, "upsert_lesson_batch", upsertLessonBatchFn)
		L.SetField(attTbl, "record_lesson_batch", upsertLessonBatchFn)
	}

	L.SetField(dalTbl, "attendance", attTbl)
}
