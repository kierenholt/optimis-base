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
