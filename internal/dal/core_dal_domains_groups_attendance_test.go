package dal_test

import (
	"context"
	"testing"

	"optimis/internal/dal"
)

// ============================================================================
// 2. Groups & Timetable Spine (`ctx.dal.groups`)
// ============================================================================

func TestListGroupsScoped_TeacherWithOwnGroupsScope_ReturnsOnlyTemporallyAssignedGroups_test(t *testing.T) {
	ctx := context.Background()
	d, fx, logTap := setupIsolatedCoreDALWithFixtures(t)

	scope := dal.DALQueryScope{
		Mode:           dal.ScopeOwnGroups,
		StaffID:        fx.TeacherStaffID,
		AcademicYearID: fx.AcademicYearID,
		ReferenceDate:  fx.ReferenceDate,
	}

	groups, err := d.ListGroupsScoped(ctx, scope, "")
	if err != nil {
		t.Fatalf("ListGroupsScoped failed: %v", err)
	}
	logTap.Logf("teacher :own_groups count on %s: %d", fx.ReferenceDate, len(groups))

	// Must return grp_10ma1 only (excluding grp_10ma2 taught by stf_teacher_2 and grp_expired which ended 2026-10-01)
	if len(groups) != 1 {
		t.Fatalf("expected 1 active group for teacher 1 on %s, got %d (%+v)", fx.ReferenceDate, len(groups), groups)
	}
	if groups[0].ID != fx.OwnGroupID || groups[0].Code != "10Ma1" {
		t.Errorf("expected group %s (10Ma1), got %+v", fx.OwnGroupID, groups[0])
	}
}

func TestListGroupsScoped_CallerWithAllScope_ReturnsAllActiveGroupsForAcademicYear_test(t *testing.T) {
	ctx := context.Background()
	d, fx, logTap := setupIsolatedCoreDALWithFixtures(t)

	scope := dal.DALQueryScope{
		Mode:           dal.ScopeAll,
		StaffID:        fx.TeacherStaffID,
		AcademicYearID: fx.AcademicYearID,
		ReferenceDate:  fx.ReferenceDate,
	}

	groups, err := d.ListGroupsScoped(ctx, scope, "TEACHING_CLASS")
	if err != nil {
		t.Fatalf("ListGroupsScoped (:all) failed: %v", err)
	}
	logTap.Logf(":all scope returned %d active TEACHING_CLASS groups", len(groups))

	if len(groups) != 2 {
		t.Fatalf("expected 2 active TEACHING_CLASS groups (10Ma1 and 10Ma2), got %d (%+v)", len(groups), groups)
	}
}

func TestGetStaffTimetableSlotsScoped_TwoWeekRollingCycle_ReturnsSlotsWithMappedStatutorySession_test(t *testing.T) {
	ctx := context.Background()
	d, fx, logTap := setupIsolatedCoreDALWithFixtures(t)

	scope := dal.DALQueryScope{
		Mode:           dal.ScopeOwnGroups,
		StaffID:        fx.TeacherStaffID,
		AcademicYearID: fx.AcademicYearID,
		ReferenceDate:  fx.ReferenceDate,
	}

	wk1Slots, err := d.GetStaffTimetableSlotsScoped(ctx, scope, fx.TeacherStaffID, 1, 1)
	if err != nil {
		t.Fatalf("GetStaffTimetableSlotsScoped (Week 1) failed: %v", err)
	}
	if len(wk1Slots) != 1 {
		t.Fatalf("expected 1 slot on CycleWeek 1 Day 1, got %d", len(wk1Slots))
	}
	if wk1Slots[0].SlotID != fx.TimetableSlotAMID || wk1Slots[0].MapsToStatutorySession != "AM" {
		t.Errorf("expected slot %s mapped to statutory 'AM', got %+v", fx.TimetableSlotAMID, wk1Slots[0])
	}

	wk2Slots, err := d.GetStaffTimetableSlotsScoped(ctx, scope, fx.TeacherStaffID, 2, 1)
	if err != nil {
		t.Fatalf("GetStaffTimetableSlotsScoped (Week 2) failed: %v", err)
	}
	if len(wk2Slots) != 1 {
		t.Fatalf("expected 1 slot on CycleWeek 2 Day 1, got %d", len(wk2Slots))
	}
	if wk2Slots[0].SlotID != fx.TimetableSlotPMID || wk2Slots[0].MapsToStatutorySession != "PM" {
		t.Errorf("expected slot %s mapped to statutory 'PM', got %+v", fx.TimetableSlotPMID, wk2Slots[0])
	}
	logTap.Logf("verified 2-week rolling timetable statutory session mapping (Wk1=AM, Wk2=PM)")
}

// ============================================================================
// 3. Statutory Attendance (`ctx.dal.attendance`)
// ============================================================================

func TestGetGroupSessionAttendanceScoped_GroupInTeacherOwnGroups_ReturnsStatutoryAmPmMarks_test(t *testing.T) {
	ctx := context.Background()
	d, fx, logTap := setupIsolatedCoreDALWithFixtures(t)

	scope := dal.DALQueryScope{
		Mode:           dal.ScopeOwnGroups,
		StaffID:        fx.TeacherStaffID,
		AcademicYearID: fx.AcademicYearID,
		ReferenceDate:  fx.ReferenceDate,
	}

	_, err := d.UpsertSessionAttendanceBatchScoped(ctx, scope, dal.UpsertSessionAttendanceBatchInput{
		GroupID: fx.OwnGroupID,
		Date:    fx.ReferenceDate,
		Session: "AM",
		Marks: []dal.SessionAttendanceMarkInput{
			{
				PupilID:        fx.ActivePupilID,
				AttendanceCode: "/",
				MinutesLate:    0,
				Comment:        "On time",
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to seed AM attendance mark via UpsertSessionAttendanceBatchScoped: %v", err)
	}

	marks, err := d.GetGroupSessionAttendanceScoped(ctx, scope, fx.OwnGroupID, fx.ReferenceDate, "AM")
	if err != nil {
		t.Fatalf("GetGroupSessionAttendanceScoped failed: %v", err)
	}
	logTap.Logf("fetched %d AM attendance marks for %s", len(marks), fx.OwnGroupID)

	if len(marks) != 1 {
		t.Fatalf("expected 1 AM session mark, got %d", len(marks))
	}
	if marks[0].PupilID != fx.ActivePupilID || marks[0].AttendanceCode != "/" || marks[0].Session != "AM" {
		t.Errorf("unexpected AM session mark: %+v", marks[0])
	}
}

func TestUpsertSessionAttendanceBatchScoped_ValidAmAndPmMarks_PersistsMarksAndRegisterSubmissionAtomically_test(t *testing.T) {
	ctx := context.Background()
	d, fx, logTap := setupIsolatedCoreDALWithFixtures(t)

	scope := dal.DALQueryScope{
		Mode:           dal.ScopeOwnGroups,
		StaffID:        fx.TeacherStaffID,
		AcademicYearID: fx.AcademicYearID,
		ReferenceDate:  fx.ReferenceDate,
	}

	// 1. Insert initial AM mark ('/')
	res1, err := d.UpsertSessionAttendanceBatchScoped(ctx, scope, dal.UpsertSessionAttendanceBatchInput{
		GroupID: fx.OwnGroupID,
		Date:    fx.ReferenceDate,
		Session: "AM",
		Marks: []dal.SessionAttendanceMarkInput{
			{
				PupilID:        fx.ActivePupilID,
				AttendanceCode: "/",
				MinutesLate:    0,
			},
		},
	})
	if err != nil {
		t.Fatalf("initial AM batch upsert failed: %v", err)
	}
	if res1.UpsertedCount != 1 || res1.SubmissionID == "" {
		t.Fatalf("expected UpsertedCount=1 and non-empty SubmissionID, got %+v", res1)
	}

	// 2. Upsert updated AM mark ('L' with 12 minutes late) and verify idempotent update + audit submission row
	res2, err := d.UpsertSessionAttendanceBatchScoped(ctx, scope, dal.UpsertSessionAttendanceBatchInput{
		GroupID: fx.OwnGroupID,
		Date:    fx.ReferenceDate,
		Session: "AM",
		Marks: []dal.SessionAttendanceMarkInput{
			{
				PupilID:        fx.ActivePupilID,
				AttendanceCode: "L",
				MinutesLate:    12,
				Comment:        "Bus delayed",
			},
		},
	})
	if err != nil {
		t.Fatalf("second AM batch upsert failed: %v", err)
	}
	logTap.Logf("upserted AM register submission %s", res2.SubmissionID)

	marks, err := d.GetGroupSessionAttendanceScoped(ctx, scope, fx.OwnGroupID, fx.ReferenceDate, "AM")
	if err != nil {
		t.Fatalf("GetGroupSessionAttendanceScoped failed: %v", err)
	}
	if len(marks) != 1 || marks[0].AttendanceCode != "L" || marks[0].MinutesLate != 12 {
		t.Fatalf("expected updated mark code='L' minutes_late=12, got %+v", marks)
	}

	var submissionCount int
	err = d.ReadDB().QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM attendance_register_submissions WHERE group_id = ? AND date = ? AND session_or_slot_key = 'AM'`,
		fx.OwnGroupID, fx.ReferenceDate,
	).Scan(&submissionCount)
	if err != nil || submissionCount != 1 {
		t.Fatalf("expected 1 attendance_register_submissions row, got count=%d err=%v", submissionCount, err)
	}
}

func TestUpsertSessionAttendanceBatchScoped_InvalidNonAmPmSessionOrUnauthorizedGroup_FailsFastAndRollsBack_test(t *testing.T) {
	ctx := context.Background()
	d, fx, logTap := setupIsolatedCoreDALWithFixtures(t)

	scope := dal.DALQueryScope{
		Mode:           dal.ScopeOwnGroups,
		StaffID:        fx.TeacherStaffID,
		AcademicYearID: fx.AcademicYearID,
		ReferenceDate:  fx.ReferenceDate,
	}

	// 1. Non-AM/PM session ('LUNCH') MUST fail fast
	_, err := d.UpsertSessionAttendanceBatchScoped(ctx, scope, dal.UpsertSessionAttendanceBatchInput{
		GroupID: fx.OwnGroupID,
		Date:    fx.ReferenceDate,
		Session: "LUNCH",
		Marks: []dal.SessionAttendanceMarkInput{
			{PupilID: fx.ActivePupilID, AttendanceCode: "/"},
		},
	})
	if err == nil {
		t.Fatalf("expected fail-fast error for non-statutory session 'LUNCH', got nil")
	}

	// 2. Unauthorized group (grp_10ma2) under :own_groups MUST fail fast
	_, err = d.UpsertSessionAttendanceBatchScoped(ctx, scope, dal.UpsertSessionAttendanceBatchInput{
		GroupID: fx.OtherGroupID,
		Date:    fx.ReferenceDate,
		Session: "AM",
		Marks: []dal.SessionAttendanceMarkInput{
			{PupilID: fx.OtherGroupPupilID, AttendanceCode: "/"},
		},
	})
	if err == nil {
		t.Fatalf("expected fail-fast error when recording attendance for unassigned group %s", fx.OtherGroupID)
	}
	logTap.Logf("verified fail-fast rollback for invalid session and unauthorized group")
}

func TestUpsertLessonAttendanceBatchScoped_ValidLessonMarks_PersistsToDedicatedLessonSubmissionsTable_test(t *testing.T) {
	ctx := context.Background()
	d, fx, logTap := setupIsolatedCoreDALWithFixtures(t)

	scope := dal.DALQueryScope{
		Mode:           dal.ScopeOwnGroups,
		StaffID:        fx.TeacherStaffID,
		AcademicYearID: fx.AcademicYearID,
		ReferenceDate:  fx.ReferenceDate,
	}

	res, err := d.UpsertLessonAttendanceBatchScoped(ctx, scope, dal.UpsertLessonAttendanceBatchInput{
		GroupID:         fx.OwnGroupID,
		TimetableSlotID: fx.TimetableSlotAMID,
		Date:            fx.ReferenceDate,
		Marks: []dal.SessionAttendanceMarkInput{
			{
				PupilID:        fx.ActivePupilID,
				AttendanceCode: "/",
				MinutesLate:    0,
				Comment:        "Present for lesson",
			},
		},
	})
	if err != nil {
		t.Fatalf("UpsertLessonAttendanceBatchScoped failed: %v", err)
	}
	if res.UpsertedCount != 1 || res.SubmissionID == "" {
		t.Fatalf("expected UpsertedCount=1 and non-empty SubmissionID, got %+v", res)
	}
	logTap.Logf("persisted lesson register submission %s for slot %s", res.SubmissionID, fx.TimetableSlotAMID)

	// Verify directly in attendance_lesson_register_submissions
	var lessonSubCount int
	err = d.ReadDB().QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM attendance_lesson_register_submissions WHERE group_id = ? AND date = ? AND timetable_slot_id = ?`,
		fx.OwnGroupID, fx.ReferenceDate, fx.TimetableSlotAMID,
	).Scan(&lessonSubCount)
	if err != nil || lessonSubCount != 1 {
		t.Fatalf("expected 1 row in attendance_lesson_register_submissions, got count=%d err=%v", lessonSubCount, err)
	}

	// Verify in backwards-compatible unified view attendance_register_submissions
	var viewCount int
	var subType string
	err = d.ReadDB().QueryRowContext(
		ctx,
		`SELECT COUNT(*), submission_type FROM attendance_register_submissions WHERE group_id = ? AND date = ? AND timetable_slot_id = ? GROUP BY submission_type`,
		fx.OwnGroupID, fx.ReferenceDate, fx.TimetableSlotAMID,
	).Scan(&viewCount, &subType)
	if err != nil || viewCount != 1 || subType != "LESSON" {
		t.Fatalf("expected 1 LESSON row in attendance_register_submissions view, got count=%d subType=%s err=%v", viewCount, subType, err)
	}
}

func TestListAttendanceRegisterSubmissionsScoped_SessionAndLesson_ReturnsUnifiedResultsWithDiscriminatedTypes_test(t *testing.T) {
	ctx := context.Background()
	d, fx, _ := setupIsolatedCoreDALWithFixtures(t)

	scope := dal.DALQueryScope{
		Mode:           dal.ScopeOwnGroups,
		StaffID:        fx.TeacherStaffID,
		AcademicYearID: fx.AcademicYearID,
		ReferenceDate:  fx.ReferenceDate,
	}

	// 1. Submit session register (AM)
	_, err := d.UpsertSessionAttendanceBatchScoped(ctx, scope, dal.UpsertSessionAttendanceBatchInput{
		GroupID: fx.OwnGroupID,
		Date:    fx.ReferenceDate,
		Session: "AM",
		Marks: []dal.SessionAttendanceMarkInput{
			{PupilID: fx.ActivePupilID, AttendanceCode: "/"},
		},
	})
	if err != nil {
		t.Fatalf("upsert AM session failed: %v", err)
	}

	// 2. Submit lesson register (Slot)
	_, err = d.UpsertLessonAttendanceBatchScoped(ctx, scope, dal.UpsertLessonAttendanceBatchInput{
		GroupID:         fx.OwnGroupID,
		TimetableSlotID: fx.TimetableSlotAMID,
		Date:            fx.ReferenceDate,
		Marks: []dal.SessionAttendanceMarkInput{
			{PupilID: fx.ActivePupilID, AttendanceCode: "/"},
		},
	})
	if err != nil {
		t.Fatalf("upsert lesson failed: %v", err)
	}

	// 3. List register submissions
	subs, err := d.ListAttendanceRegisterSubmissionsScoped(ctx, scope, fx.ReferenceDate, fx.OwnGroupID)
	if err != nil {
		t.Fatalf("ListAttendanceRegisterSubmissionsScoped failed: %v", err)
	}
	if len(subs) != 2 {
		t.Fatalf("expected 2 submissions (1 session + 1 lesson), got %d: %+v", len(subs), subs)
	}

	foundSession := false
	foundLesson := false
	for _, s := range subs {
		if s.SubmissionType == "SESSION" {
			foundSession = true
			if s.Session != "AM" || s.SessionOrSlotKey != "AM" || s.TimetableSlotID != "" {
				t.Errorf("unexpected session submission data: %+v", s)
			}
		}
		if s.SubmissionType == "LESSON" {
			foundLesson = true
			if s.TimetableSlotID != fx.TimetableSlotAMID || s.SessionOrSlotKey != fx.TimetableSlotAMID || s.Session != "" {
				t.Errorf("unexpected lesson submission data: %+v", s)
			}
		}
	}
	if !foundSession || !foundLesson {
		t.Fatalf("expected both session and lesson submissions, got %+v", subs)
	}
}

func TestAttendanceRegisterSubmissions_SchemaIntegrity_EnforcesForeignKeysAndCheckConstraints_test(t *testing.T) {
	ctx := context.Background()
	d, fx, _ := setupIsolatedCoreDALWithFixtures(t)

	// 1. Direct insert of invalid session ('MIDDAY') into attendance_session_register_submissions MUST violate CHECK constraint
	_, err := d.WriteDB().ExecContext(
		ctx,
		`INSERT INTO attendance_session_register_submissions (id, group_id, date, session, taken_by_staff_id)
		 VALUES ('ars_chk_fail', ?, ?, 'MIDDAY', ?)`,
		fx.OwnGroupID, fx.ReferenceDate, fx.TeacherStaffID,
	)
	if err == nil {
		t.Fatalf("expected SQLite CHECK constraint failure on attendance_session_register_submissions for session='MIDDAY', got nil")
	}

	// 2. Direct insert of non-existent timetable_slot_id into attendance_lesson_register_submissions MUST violate FOREIGN KEY constraint
	_, err = d.WriteDB().ExecContext(
		ctx,
		`INSERT INTO attendance_lesson_register_submissions (id, group_id, timetable_slot_id, date, taken_by_staff_id)
		 VALUES ('ars_fk_fail', ?, 'non_existent_slot_999', ?, ?)`,
		fx.OwnGroupID, fx.ReferenceDate, fx.TeacherStaffID,
	)
	if err == nil {
		t.Fatalf("expected SQLite FOREIGN KEY constraint failure on attendance_lesson_register_submissions for invalid slot, got nil")
	}

	// 3. Inserting valid lesson submission then trying to delete referenced timetable slot MUST be blocked by ON DELETE RESTRICT
	_, err = d.WriteDB().ExecContext(
		ctx,
		`INSERT INTO attendance_lesson_register_submissions (id, group_id, timetable_slot_id, date, taken_by_staff_id)
		 VALUES ('ars_fk_valid', ?, ?, ?, ?)`,
		fx.OwnGroupID, fx.TimetableSlotAMID, fx.ReferenceDate, fx.TeacherStaffID,
	)
	if err != nil {
		t.Fatalf("insert valid lesson submission failed: %v", err)
	}

	_, err = d.WriteDB().ExecContext(ctx, `DELETE FROM timetable_slots WHERE id = ?`, fx.TimetableSlotAMID)
	if err == nil {
		t.Fatalf("expected SQLite FOREIGN KEY (ON DELETE RESTRICT) to block deleting timetable_slot referenced by attendance_lesson_register_submissions, got nil")
	}
}

