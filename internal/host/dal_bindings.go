package host

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	lua "github.com/yuin/gopher-lua"

	"optimis/internal/dal"
)

var piiScrubPattern = regexp.MustCompile(`(?i)\b(pup_[a-z0-9_]+|per_[a-z0-9_]+|[A-Z][0-9]{12})\b`)

func scrubDiagnosticToken(raw string) string {
	return piiScrubPattern.ReplaceAllString(raw, "[REDACTED]")
}

func resolveDomainScopeMode(scopes []string, domainPrefix string) dal.ScopeMode {
	for _, s := range scopes {
		if s == domainPrefix+":all" {
			return dal.ScopeAll
		}
	}
	return dal.ScopeOwnGroups
}

func extractScopeFromLuaOpts(
	ctx context.Context,
	d *dal.CoreDAL,
	mode dal.ScopeMode,
	staffID string,
	optsTbl *lua.LTable,
) (dal.DALQueryScope, error) {
	scope := dal.DALQueryScope{
		Mode:    mode,
		StaffID: staffID,
	}
	if optsTbl != nil {
		if ay := optsTbl.RawGetString("academic_year_id"); ay != lua.LNil {
			scope.AcademicYearID = strings.TrimSpace(ay.String())
		}
		if rd := optsTbl.RawGetString("reference_date"); rd != lua.LNil {
			scope.ReferenceDate = strings.TrimSpace(rd.String())
		}
	}
	return d.ResolveScopeDefaults(ctx, scope)
}

func pupilToLuaTable(L *lua.LState, pupil dal.Pupil) *lua.LTable {
	pTbl := L.NewTable()
	L.SetField(pTbl, "id", lua.LString(pupil.ID))
	L.SetField(pTbl, "upn", lua.LString(pupil.UPN))
	L.SetField(pTbl, "first_name", lua.LString(pupil.FirstName))
	L.SetField(pTbl, "last_name", lua.LString(pupil.LastName))
	L.SetField(pTbl, "academic_year_id", lua.LString(pupil.AcademicYearID))
	L.SetField(pTbl, "effective_from", lua.LString(pupil.EffectiveFrom))
	if pupil.EffectiveTo != nil {
		L.SetField(pTbl, "effective_to", lua.LString(*pupil.EffectiveTo))
	}
	return pTbl
}

func (h *Host) buildLuaDALTable(
	L *lua.LState,
	execCtx context.Context,
	endpointSpec EndpointSpec,
	caller CallerContext,
) *lua.LTable {
	dalTbl := L.NewTable()

	// 1. ctx.dal.pupils
	if hasDomainScope(endpointSpec.Scopes, "dal.pupils.read") {
		pupilsMode := resolveDomainScopeMode(endpointSpec.Scopes, "dal.pupils.read")
		pupilsTbl := L.NewTable()

		L.SetField(pupilsTbl, "get_by_id", L.NewFunction(func(L *lua.LState) int {
			pupilID := strings.TrimSpace(L.OptString(1, ""))
			if pupilID == "" {
				L.RaiseError("missing mandatory parameter: pupil_id")
				return 0
			}
			var optsTbl *lua.LTable
			if L.GetTop() >= 2 {
				optsTbl, _ = L.Get(2).(*lua.LTable)
			}

			var pupil *dal.Pupil
			var err error
			if pupilsMode == dal.ScopeAll && optsTbl == nil {
				pupil, err = h.coreDAL.GetPupilByID(execCtx, pupilID)
			} else {
				scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, pupilsMode, caller.StaffID, optsTbl)
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
			scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, pupilsMode, caller.StaffID, optsTbl)
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

		L.SetField(pupilsTbl, "list_sen_and_medical_batch", L.NewFunction(func(L *lua.LState) int {
			idsTbl, ok := L.Get(1).(*lua.LTable)
			if !ok {
				L.RaiseError("dal.pupils.list_sen_and_medical_batch requires a table of pupil_ids")
				return 0
			}
			var pupilIDs []string
			maxn := idsTbl.MaxN()
			for i := 1; i <= maxn; i++ {
				val := strings.TrimSpace(idsTbl.RawGetInt(i).String())
				if val != "" {
					pupilIDs = append(pupilIDs, val)
				}
			}
			var optsTbl *lua.LTable
			if L.GetTop() >= 2 {
				optsTbl, _ = L.Get(2).(*lua.LTable)
			}
			scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, pupilsMode, caller.StaffID, optsTbl)
			if sErr != nil {
				L.RaiseError("dal.pupils.list_sen_and_medical_batch scope error: %s", sErr.Error())
				return 0
			}
			batch, err := h.coreDAL.ListPupilSenAndMedicalBatch(execCtx, scope, pupilIDs)
			if err != nil {
				L.RaiseError("dal.pupils.list_sen_and_medical_batch failed: %s", err.Error())
				return 0
			}
			resTbl := L.NewTable()
			for pid, summary := range batch {
				sTbl := L.NewTable()
				L.SetField(sTbl, "pupil_id", lua.LString(summary.PupilID))
				L.SetField(sTbl, "sen_provision_code", lua.LString(summary.SenProvisionCode))
				L.SetField(sTbl, "primary_sen_need_code", lua.LString(summary.PrimarySenNeedCode))
				L.SetField(sTbl, "teaching_strategies", lua.LString(summary.TeachingStrategies))

				medsArr := L.NewTable()
				markLuaTableAsArray(L, medsArr)
				for idx, mc := range summary.MedicalConditions {
					mTbl := L.NewTable()
					L.SetField(mTbl, "id", lua.LString(mc.ID))
					L.SetField(mTbl, "condition_name", lua.LString(mc.ConditionName))
					L.SetField(mTbl, "condition_category", lua.LString(mc.ConditionCategory))
					L.SetField(mTbl, "alert_level_id", lua.LString(mc.AlertLevelID))
					L.SetField(mTbl, "has_ihcp", lua.LBool(mc.HasIHCP))
					L.SetField(mTbl, "emergency_response_plan", lua.LString(mc.EmergencyResponsePlan))
					medsArr.RawSetInt(idx+1, mTbl)
				}
				L.SetField(sTbl, "medical_conditions", medsArr)
				L.SetField(resTbl, pid, sTbl)
			}
			L.Push(resTbl)
			return 1
		}))

		L.SetField(dalTbl, "pupils", pupilsTbl)
	}

	// 2. ctx.dal.groups
	if hasDomainScope(endpointSpec.Scopes, "dal.groups.read") {
		groupsMode := resolveDomainScopeMode(endpointSpec.Scopes, "dal.groups.read")
		groupsTbl := L.NewTable()

		L.SetField(groupsTbl, "list_groups", L.NewFunction(func(L *lua.LState) int {
			var optsTbl *lua.LTable
			if L.GetTop() >= 1 {
				optsTbl, _ = L.Get(1).(*lua.LTable)
			}
			var groupType string
			if optsTbl != nil {
				if gt := optsTbl.RawGetString("group_type"); gt != lua.LNil {
					groupType = strings.TrimSpace(gt.String())
				}
			}
			scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, groupsMode, caller.StaffID, optsTbl)
			if sErr != nil {
				L.RaiseError("dal.groups.list_groups scope error: %s", sErr.Error())
				return 0
			}
			groups, err := h.coreDAL.ListGroupsScoped(execCtx, scope, groupType)
			if err != nil {
				L.RaiseError("dal.groups.list_groups failed: %s", err.Error())
				return 0
			}
			arrTbl := L.NewTable()
			markLuaTableAsArray(L, arrTbl)
			for i, g := range groups {
				gTbl := L.NewTable()
				L.SetField(gTbl, "id", lua.LString(g.ID))
				L.SetField(gTbl, "academic_year_id", lua.LString(g.AcademicYearID))
				L.SetField(gTbl, "group_type", lua.LString(g.GroupType))
				L.SetField(gTbl, "code", lua.LString(g.Code))
				L.SetField(gTbl, "name", lua.LString(g.Name))
				L.SetField(gTbl, "default_room_id", lua.LString(g.DefaultRoomID))
				L.SetField(gTbl, "takes_attendance", lua.LBool(g.TakesAttendance))
				L.SetField(gTbl, "effective_from", lua.LString(g.EffectiveFrom))
				arrTbl.RawSetInt(i+1, gTbl)
			}
			L.Push(arrTbl)
			return 1
		}))

		L.SetField(groupsTbl, "get_staff_timetable_slots", L.NewFunction(func(L *lua.LState) int {
			staffID := strings.TrimSpace(L.OptString(1, ""))
			cycleWeek := L.OptInt(2, 1)
			dayOfWeekID := L.OptInt(3, 1)
			var optsTbl *lua.LTable
			if L.GetTop() >= 4 {
				optsTbl, _ = L.Get(4).(*lua.LTable)
			}
			scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, groupsMode, caller.StaffID, optsTbl)
			if sErr != nil {
				L.RaiseError("dal.groups.get_staff_timetable_slots scope error: %s", sErr.Error())
				return 0
			}
			slots, err := h.coreDAL.GetStaffTimetableSlotsScoped(execCtx, scope, staffID, cycleWeek, dayOfWeekID)
			if err != nil {
				L.RaiseError("dal.groups.get_staff_timetable_slots failed: %s", err.Error())
				return 0
			}
			arrTbl := L.NewTable()
			markLuaTableAsArray(L, arrTbl)
			for i, s := range slots {
				sTbl := L.NewTable()
				L.SetField(sTbl, "slot_id", lua.LString(s.SlotID))
				L.SetField(sTbl, "period_id", lua.LString(s.PeriodID))
				L.SetField(sTbl, "period_code", lua.LString(s.PeriodCode))
				L.SetField(sTbl, "period_name", lua.LString(s.PeriodName))
				L.SetField(sTbl, "cycle_week", lua.LNumber(s.CycleWeek))
				L.SetField(sTbl, "day_of_week_id", lua.LNumber(s.DayOfWeekID))
				L.SetField(sTbl, "start_time", lua.LString(s.StartTime))
				L.SetField(sTbl, "end_time", lua.LString(s.EndTime))
				L.SetField(sTbl, "period_type", lua.LString(s.PeriodType))
				L.SetField(sTbl, "maps_to_statutory_session", lua.LString(s.MapsToStatutorySession))
				L.SetField(sTbl, "group_id", lua.LString(s.GroupID))
				L.SetField(sTbl, "group_code", lua.LString(s.GroupCode))
				L.SetField(sTbl, "staff_id", lua.LString(s.StaffID))
				L.SetField(sTbl, "room_id", lua.LString(s.RoomID))
				arrTbl.RawSetInt(i+1, sTbl)
			}
			L.Push(arrTbl)
			return 1
		}))

		L.SetField(dalTbl, "groups", groupsTbl)
	}

	// 3. ctx.dal.attendance
	hasAttRead := hasDomainScope(endpointSpec.Scopes, "dal.attendance.read")
	hasAttWrite := hasDomainScope(endpointSpec.Scopes, "dal.attendance.write")
	if hasAttRead || hasAttWrite {
		attTbl := L.NewTable()

		if hasAttRead {
			attReadMode := resolveDomainScopeMode(endpointSpec.Scopes, "dal.attendance.read")
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
		}

		if hasAttWrite {
			attWriteMode := resolveDomainScopeMode(endpointSpec.Scopes, "dal.attendance.write")
			L.SetField(attTbl, "record_session_batch", L.NewFunction(func(L *lua.LState) int {
				if !endpointSpec.Mutates {
					L.RaiseError("dal.attendance.record_session_batch is forbidden when endpoint mutates=false")
					return 0
				}
				groupID := strings.TrimSpace(L.OptString(1, ""))
				date := strings.TrimSpace(L.OptString(2, ""))
				session := strings.TrimSpace(L.OptString(3, ""))
				marksTbl, ok := L.Get(4).(*lua.LTable)
				if !ok {
					L.RaiseError("dal.attendance.record_session_batch requires a marks table as 4th argument")
					return 0
				}
				var optsTbl *lua.LTable
				if L.GetTop() >= 5 {
					optsTbl, _ = L.Get(5).(*lua.LTable)
				}
				scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, attWriteMode, caller.StaffID, optsTbl)
				if sErr != nil {
					L.RaiseError("dal.attendance.record_session_batch scope error: %s", sErr.Error())
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
						PupilID:        strings.TrimSpace(itemTbl.RawGetString("pupil_id").String()),
						AttendanceCode: strings.TrimSpace(itemTbl.RawGetString("attendance_code").String()),
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
					L.RaiseError("dal.attendance.record_session_batch failed: %s", err.Error())
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
			}))
		}

		L.SetField(dalTbl, "attendance", attTbl)
	}

	// 4. ctx.dal.guardians
	if hasDomainScope(endpointSpec.Scopes, "dal.guardians.read") {
		guardiansMode := resolveDomainScopeMode(endpointSpec.Scopes, "dal.guardians.read")
		guardiansTbl := L.NewTable()

		L.SetField(guardiansTbl, "list_for_pupil", L.NewFunction(func(L *lua.LState) int {
			pupilID := strings.TrimSpace(L.OptString(1, ""))
			if pupilID == "" {
				L.RaiseError("missing mandatory parameter: pupil_id")
				return 0
			}
			var optsTbl *lua.LTable
			if L.GetTop() >= 2 {
				optsTbl, _ = L.Get(2).(*lua.LTable)
			}
			var reqPR bool
			if optsTbl != nil {
				if prVal := optsTbl.RawGetString("require_parental_responsibility"); prVal == lua.LTrue {
					reqPR = true
				}
			}
			scope, sErr := extractScopeFromLuaOpts(execCtx, h.coreDAL, guardiansMode, caller.StaffID, optsTbl)
			if sErr != nil {
				L.RaiseError("dal.guardians.list_for_pupil scope error: %s", sErr.Error())
				return 0
			}
			guardians, err := h.coreDAL.ListGuardiansForPupilScoped(execCtx, scope, pupilID, dal.GuardianFilterOptions{
				RequireParentalResponsibility: reqPR,
			})
			if err != nil {
				L.RaiseError("dal.guardians.list_for_pupil failed: %s", err.Error())
				return 0
			}
			arrTbl := L.NewTable()
			markLuaTableAsArray(L, arrTbl)
			for i, g := range guardians {
				gTbl := L.NewTable()
				L.SetField(gTbl, "relationship_id", lua.LString(g.RelationshipID))
				L.SetField(gTbl, "pupil_id", lua.LString(g.PupilID))
				L.SetField(gTbl, "guardian_person_id", lua.LString(g.GuardianPersonID))
				L.SetField(gTbl, "relationship_code", lua.LString(g.RelationshipCode))
				L.SetField(gTbl, "first_name", lua.LString(g.FirstName))
				L.SetField(gTbl, "last_name", lua.LString(g.LastName))
				L.SetField(gTbl, "email", lua.LString(g.Email))
				L.SetField(gTbl, "phone_primary", lua.LString(g.PhonePrimary))
				L.SetField(gTbl, "has_parental_responsibility", lua.LBool(g.HasParentalResponsibility))
				L.SetField(gTbl, "contact_priority", lua.LNumber(g.ContactPriority))
				L.SetField(gTbl, "is_emergency_contact", lua.LBool(g.IsEmergencyContact))
				L.SetField(gTbl, "authorised_to_collect", lua.LBool(g.AuthorisedToCollect))
				L.SetField(gTbl, "receive_reports", lua.LBool(g.ReceiveReports))
				L.SetField(gTbl, "household_id", lua.LString(g.HouseholdID))
				L.SetField(gTbl, "household_name", lua.LString(g.HouseholdName))
				L.SetField(gTbl, "address_line_1", lua.LString(g.AddressLine1))
				L.SetField(gTbl, "town_city", lua.LString(g.TownCity))
				L.SetField(gTbl, "postcode", lua.LString(g.Postcode))
				L.SetField(gTbl, "is_confidential_address_redacted", lua.LBool(g.IsConfidentialAddressRedacted))
				arrTbl.RawSetInt(i+1, gTbl)
			}
			L.Push(arrTbl)
			return 1
		}))

		L.SetField(dalTbl, "guardians", guardiansTbl)
	}

	// 5. ctx.dal.safeguarding (Strictly write-only postbox; ZERO read/decrypt exposed to Lua)
	if hasDomainScope(endpointSpec.Scopes, "dal.safeguarding.create") {
		sfgTbl := L.NewTable()

		L.SetField(sfgTbl, "create_postbox", L.NewFunction(func(L *lua.LState) int {
			if !endpointSpec.Mutates {
				L.RaiseError("dal.safeguarding.create_postbox is forbidden when endpoint mutates=false")
				return 0
			}
			inTbl, ok := L.Get(1).(*lua.LTable)
			if !ok {
				L.RaiseError("dal.safeguarding.create_postbox requires an input table")
				return 0
			}
			getStr := func(key string) string {
				v := inTbl.RawGetString(key)
				if v == lua.LNil {
					return ""
				}
				return strings.TrimSpace(v.String())
			}

			receipt, err := h.coreDAL.CreateSafeguardingPostboxNote(execCtx, dal.CreateSafeguardingPostboxInput{
				PupilID:            getStr("pupil_id"),
				ReportedByStaffID:  caller.StaffID,
				AssignedDSLStaffID: getStr("assigned_dsl_staff_id"),
				ConcernCategory:    getStr("concern_category"),
				IncidentDate:       getStr("incident_date"),
				Summary:            getStr("summary"),
				BodyMap:            getStr("body_map"),
				VerbatimAccount:    getStr("verbatim_account"),
				ActionTaken:        getStr("action_taken"),
				RequestID:          caller.RequestID,
				ClientIP:           "127.0.0.1",
				HostKMSKeyHandle:   h.cfg.HostKMSKeyHandle,
			})
			if err != nil {
				L.RaiseError("dal.safeguarding.create_postbox failed: %s", err.Error())
				return 0
			}

			resTbl := L.NewTable()
			L.SetField(resTbl, "note_id", lua.LString(receipt.NoteID))
			L.SetField(resTbl, "status", lua.LString(receipt.Status))
			L.Push(resTbl)
			return 1
		}))

		L.SetField(dalTbl, "safeguarding", sfgTbl)
	}

	return dalTbl
}

func (h *Host) ExecuteWorker(
	ctx context.Context,
	pluginID string,
	workerName string,
	args map[string]any,
) (*RPCResponse, error) {
	start := time.Now()

	if strings.TrimSpace(pluginID) == "" {
		return nil, errors.New("missing mandatory parameter: pluginID")
	}
	if strings.TrimSpace(workerName) == "" {
		return nil, errors.New("missing mandatory parameter: workerName")
	}

	plugin, exists := h.GetLoadedPlugin(pluginID)
	if !exists {
		h.logError(fmt.Sprintf("Plugin not found for worker execution: %s", pluginID))
		return nil, fmt.Errorf("plugin not loaded: %s", pluginID)
	}

	workerSpec, declared := plugin.Manifest.Workers[workerName]
	if !declared {
		h.recordDiagnostic(pluginID, workerName, "UNDECLARED_WORKER", time.Since(start))
		return nil, fmt.Errorf("worker %q is not declared in manifest.json for plugin %s", workerName, pluginID)
	}

	timeoutMS := workerSpec.TimeoutMS
	if timeoutMS <= 0 {
		timeoutMS = 30000
	}
	execCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMS)*time.Millisecond)
	defer cancel()

	caller := CallerContext{
		Type:        "SYSTEM",
		UserID:      "system:" + pluginID,
		StaffID:     "",
		ActiveRoles: []string{"system_worker"},
		Scopes:      workerSpec.Scopes,
		RequestID:   "wrk_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12],
	}

	L := lua.NewState(lua.Options{SkipOpenLibs: true})
	defer L.Close()
	for _, lib := range []struct {
		name string
		fn   lua.LGFunction
	}{
		{lua.LoadLibName, lua.OpenPackage},
		{lua.BaseLibName, lua.OpenBase},
		{lua.TabLibName, lua.OpenTable},
		{lua.StringLibName, lua.OpenString},
		{lua.MathLibName, lua.OpenMath},
	} {
		L.Push(L.NewFunction(lib.fn))
		L.Push(lua.LString(lib.name))
		L.Call(1, 0)
	}
	L.SetGlobal("dofile", lua.LNil)
	L.SetGlobal("loadfile", lua.LNil)
	L.SetGlobal("package", lua.LNil)
	L.SetGlobal("require", lua.LNil)
	L.SetGlobal("os", lua.LNil)
	L.SetGlobal("io", lua.LNil)
	L.SetGlobal("debug", lua.LNil)
	L.SetContext(execCtx)

	var bufferedEvents []Event
	var writeTx *sql.Tx
	defer func() {
		if writeTx != nil {
			_ = writeTx.Rollback()
		}
	}()

	ctxTable := h.buildLuaCtxTable(L, execCtx, plugin, workerSpec, caller, &bufferedEvents, &writeTx)
	argsTable := goMapToLuaTable(L, args)

	if err := L.DoString(plugin.LuaSource); err != nil {
		h.recordDiagnostic(pluginID, workerName, "LUA_LOAD_ERROR", time.Since(start))
		return nil, fmt.Errorf("failed to load plugin Lua script: %w", err)
	}

	fn := L.GetGlobal(workerName)
	if fn.Type() != lua.LTFunction {
		h.recordDiagnostic(pluginID, workerName, "MISSING_LUA_FUNCTION", time.Since(start))
		return nil, fmt.Errorf("declared worker %q is not a function in init.lua", workerName)
	}

	if err := L.CallByParam(lua.P{
		Fn:      fn,
		NRet:    1,
		Protect: true,
	}, ctxTable, argsTable); err != nil {
		h.recordDiagnostic(pluginID, workerName, "LUA_RUNTIME_ERROR", time.Since(start))
		return nil, fmt.Errorf("lua worker execution error in %s.%s: %w", pluginID, workerName, err)
	}

	ret := L.Get(-1)
	L.Pop(1)

	retTbl, ok := ret.(*lua.LTable)
	if !ok {
		h.recordDiagnostic(pluginID, workerName, "INVALID_RETURN_TYPE", time.Since(start))
		return nil, fmt.Errorf("worker %s.%s must return a table", pluginID, workerName)
	}

	okVal := L.GetField(retTbl, "ok")
	if okVal != lua.LTrue {
		if writeTx != nil {
			_ = writeTx.Rollback()
			writeTx = nil
		}
		errCode := L.GetField(retTbl, "error_code").String()
		return &RPCResponse{
			OK:        false,
			ErrorCode: errCode,
		}, nil
	}

	if writeTx != nil {
		if err := writeTx.Commit(); err != nil {
			writeTx = nil
			h.recordDiagnostic(pluginID, workerName, "SQLITE_COMMIT_ERROR", time.Since(start))
			return nil, fmt.Errorf("failed to commit plugin_db write transaction in worker: %w", err)
		}
		writeTx = nil
	}

	dataVal := luaValueToGo(L.GetField(retTbl, "data"))
	if len(bufferedEvents) > 0 {
		h.dispatchEvents(bufferedEvents)
	}

	return &RPCResponse{
		OK:   true,
		Data: dataVal,
	}, nil
}
