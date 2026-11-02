package dal

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// ============================================================================
// Timetable Periods
// ============================================================================

const timetablePeriodFieldsSQL = `
	id, academic_year_id, cycle_week, day_of_week_id, period_code, name,
	start_time, end_time, period_type, COALESCE(maps_to_statutory_session, ''), sequence_number
`

type TimetablePeriodSummary struct {
	ID                     string `json:"id"`
	AcademicYearID         string `json:"academic_year_id"`
	CycleWeek              int    `json:"cycle_week"`
	DayOfWeekID            int    `json:"day_of_week_id"`
	PeriodCode             string `json:"period_code"`
	Name                   string `json:"name"`
	StartTime              string `json:"start_time"`
	EndTime                string `json:"end_time"`
	PeriodType             string `json:"period_type"`
	MapsToStatutorySession string `json:"maps_to_statutory_session"`
	SequenceNumber         int    `json:"sequence_number"`
}

func scanTimetablePeriod(s rowScanner) (TimetablePeriodSummary, error) {
	var p TimetablePeriodSummary
	if err := s.Scan(
		&p.ID, &p.AcademicYearID, &p.CycleWeek, &p.DayOfWeekID, &p.PeriodCode, &p.Name,
		&p.StartTime, &p.EndTime, &p.PeriodType, &p.MapsToStatutorySession, &p.SequenceNumber,
	); err != nil {
		return TimetablePeriodSummary{}, err
	}
	return p, nil
}

func (d *CoreDAL) ListTimetablePeriodsScoped(
	ctx context.Context,
	scope DALQueryScope,
	cycleWeek int,
	dayOfWeekID int,
) ([]TimetablePeriodSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`SELECT %s
	          FROM timetable_periods
	          WHERE academic_year_id = ?`, timetablePeriodFieldsSQL)
	args := []any{scope.AcademicYearID}
	if cycleWeek > 0 {
		query += ` AND cycle_week = ?`
		args = append(args, cycleWeek)
	}
	if dayOfWeekID > 0 {
		query += ` AND day_of_week_id = ?`
		args = append(args, dayOfWeekID)
	}
	query += ` ORDER BY cycle_week ASC, day_of_week_id ASC, sequence_number ASC, id ASC`

	rows, err := d.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list timetable_periods: %w", err)
	}
	defer rows.Close()

	var result []TimetablePeriodSummary
	for rows.Next() {
		p, err := scanTimetablePeriod(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan timetable_period row: %w", err)
		}
		result = append(result, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating timetable_periods: %w", err)
	}
	return result, nil
}

type UpsertTimetablePeriodInput struct {
	PeriodID               string `json:"period_id,omitempty"`
	AcademicYearID         string `json:"academic_year_id"`
	CycleWeek              int    `json:"cycle_week"`
	DayOfWeekID            int    `json:"day_of_week_id"`
	PeriodCode             string `json:"period_code"`
	Name                   string `json:"name"`
	StartTime              string `json:"start_time"`
	EndTime                string `json:"end_time"`
	PeriodType             string `json:"period_type"`
	MapsToStatutorySession string `json:"maps_to_statutory_session,omitempty"`
	SequenceNumber         int    `json:"sequence_number"`
}

func (d *CoreDAL) UpsertTimetablePeriodScoped(
	ctx context.Context,
	scope DALQueryScope,
	in UpsertTimetablePeriodInput,
) (*TimetablePeriodSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	if scope.Mode != ScopeAll {
		return nil, errors.New("access denied: upserting timetable periods requires dal.groups.write:all scope")
	}

	periodID := strings.TrimSpace(in.PeriodID)
	if periodID == "" {
		periodID = "ttp_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	}
	ayID := strings.TrimSpace(in.AcademicYearID)
	if ayID == "" {
		ayID = strings.TrimSpace(scope.AcademicYearID)
	}
	cycleWeek := in.CycleWeek
	if cycleWeek == 0 {
		cycleWeek = 1
	}
	if cycleWeek != 1 && cycleWeek != 2 {
		return nil, fmt.Errorf("invalid cycle_week %d: must be 1 or 2", cycleWeek)
	}
	dayOfWeekID := in.DayOfWeekID
	if dayOfWeekID < 1 || dayOfWeekID > 7 {
		return nil, fmt.Errorf("invalid day_of_week_id %d: must be 1..7", dayOfWeekID)
	}
	periodCode := strings.TrimSpace(in.PeriodCode)
	name := strings.TrimSpace(in.Name)
	startTime := strings.TrimSpace(in.StartTime)
	endTime := strings.TrimSpace(in.EndTime)
	if periodCode == "" || name == "" || startTime == "" || endTime == "" {
		return nil, errors.New("missing mandatory parameter: period_code, name, start_time, and end_time are required")
	}
	periodType := strings.ToUpper(strings.TrimSpace(in.PeriodType))
	if periodType == "" {
		periodType = "LESSON"
	}
	switch periodType {
	case "STATUTORY_AM_REG", "STATUTORY_PM_REG", "LESSON", "BREAK", "PASTORAL":
	default:
		return nil, fmt.Errorf("invalid period_type %q", in.PeriodType)
	}

	var statSessArg any
	statSess := strings.ToUpper(strings.TrimSpace(in.MapsToStatutorySession))
	if statSess != "" {
		if statSess != "AM" && statSess != "PM" {
			return nil, fmt.Errorf("invalid maps_to_statutory_session %q: must be AM or PM", in.MapsToStatutorySession)
		}
		statSessArg = statSess
	}

	seqNum := in.SequenceNumber
	if seqNum <= 0 {
		seqNum = 1
	}

	tx, err := d.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin upsert timetable_period tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO timetable_periods (
			id, academic_year_id, cycle_week, day_of_week_id, period_code, name,
			start_time, end_time, period_type, maps_to_statutory_session, sequence_number
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(academic_year_id, cycle_week, day_of_week_id, period_code) DO UPDATE SET
			name = excluded.name,
			start_time = excluded.start_time,
			end_time = excluded.end_time,
			period_type = excluded.period_type,
			maps_to_statutory_session = excluded.maps_to_statutory_session,
			sequence_number = excluded.sequence_number`,
		periodID, ayID, cycleWeek, dayOfWeekID, periodCode, name, startTime, endTime, periodType, statSessArg, seqNum,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert timetable_period row: %w", err)
	}

	var persistedID string
	if err := tx.QueryRowContext(
		ctx,
		`SELECT id FROM timetable_periods WHERE academic_year_id = ? AND cycle_week = ? AND day_of_week_id = ? AND period_code = ?`,
		ayID, cycleWeek, dayOfWeekID, periodCode,
	).Scan(&persistedID); err == nil && strings.TrimSpace(persistedID) != "" {
		periodID = persistedID
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit upsert timetable_period tx: %w", err)
	}

	return &TimetablePeriodSummary{
		ID:                     periodID,
		AcademicYearID:         ayID,
		CycleWeek:              cycleWeek,
		DayOfWeekID:            dayOfWeekID,
		PeriodCode:             periodCode,
		Name:                   name,
		StartTime:              startTime,
		EndTime:                endTime,
		PeriodType:             periodType,
		MapsToStatutorySession: statSess,
		SequenceNumber:         seqNum,
	}, nil
}

// ============================================================================
// Timetable Slots
// ============================================================================

type TimetableSlotFilter struct {
	GroupID     string
	StaffID     string
	RoomID      string
	CycleWeek   int
	DayOfWeekID int
}

func (d *CoreDAL) ListTimetableSlotsScoped(
	ctx context.Context,
	scope DALQueryScope,
	filter TimetableSlotFilter,
) ([]TimetableSlotSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`SELECT %s
	          FROM timetable_slots ts
	          JOIN timetable_periods tp ON tp.id = ts.timetable_period_id
	          JOIN groups g ON g.id = ts.group_id
	          WHERE tp.academic_year_id = ?
	            AND ts.effective_from <= ?
	            AND (ts.effective_to IS NULL OR ts.effective_to >= ?)`, timetableSlotFieldsSQL)
	args := []any{scope.AcademicYearID, scope.ReferenceDate, scope.ReferenceDate}

	if scope.Mode == ScopeOwnGroups {
		query += ` AND EXISTS (
			SELECT 1 FROM group_staff_assignments gsa
			WHERE gsa.group_id = ts.group_id
			  AND gsa.staff_id = ?
			  AND gsa.academic_year_id = ?
			  AND gsa.effective_from <= ?
			  AND (gsa.effective_to IS NULL OR gsa.effective_to >= ?)
		)`
		args = append(args, scope.StaffID, scope.AcademicYearID, scope.ReferenceDate, scope.ReferenceDate)
	}

	if gID := strings.TrimSpace(filter.GroupID); gID != "" {
		query += ` AND ts.group_id = ?`
		args = append(args, gID)
	}
	if sID := strings.TrimSpace(filter.StaffID); sID != "" {
		query += ` AND ts.staff_id = ?`
		args = append(args, sID)
	}
	if rID := strings.TrimSpace(filter.RoomID); rID != "" {
		query += ` AND ts.room_id = ?`
		args = append(args, rID)
	}
	if filter.CycleWeek > 0 {
		query += ` AND tp.cycle_week = ?`
		args = append(args, filter.CycleWeek)
	}
	if filter.DayOfWeekID > 0 {
		query += ` AND tp.day_of_week_id = ?`
		args = append(args, filter.DayOfWeekID)
	}
	query += ` ORDER BY tp.cycle_week ASC, tp.day_of_week_id ASC, tp.sequence_number ASC, ts.id ASC`

	rows, err := d.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list timetable_slots: %w", err)
	}
	defer rows.Close()

	var result []TimetableSlotSummary
	for rows.Next() {
		slot, err := scanTimetableSlot(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan timetable_slot row: %w", err)
		}
		result = append(result, slot)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating timetable_slots: %w", err)
	}
	return result, nil
}

type UpsertTimetableSlotInput struct {
	SlotID            string `json:"slot_id,omitempty"`
	TimetablePeriodID string `json:"timetable_period_id"`
	GroupID           string `json:"group_id"`
	StaffID           string `json:"staff_id,omitempty"`
	RoomID            string `json:"room_id,omitempty"`
	EffectiveFrom     string `json:"effective_from"`
	EffectiveTo       string `json:"effective_to,omitempty"`
}

type UpsertTimetableSlotResult struct {
	SlotID            string `json:"slot_id"`
	TimetablePeriodID string `json:"timetable_period_id"`
	GroupID           string `json:"group_id"`
	StaffID           string `json:"staff_id"`
	RoomID            string `json:"room_id"`
	EffectiveFrom     string `json:"effective_from"`
	EffectiveTo       string `json:"effective_to,omitempty"`
}

func (d *CoreDAL) UpsertTimetableSlotScoped(
	ctx context.Context,
	scope DALQueryScope,
	in UpsertTimetableSlotInput,
) (*UpsertTimetableSlotResult, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}

	periodID := strings.TrimSpace(in.TimetablePeriodID)
	groupID := strings.TrimSpace(in.GroupID)
	effFrom := strings.TrimSpace(in.EffectiveFrom)
	if effFrom == "" {
		effFrom = strings.TrimSpace(scope.ReferenceDate)
	}
	if periodID == "" || groupID == "" || effFrom == "" {
		return nil, errors.New("missing mandatory parameter: timetable_period_id, group_id, and effective_from are required")
	}
	if err := d.ensureGroupInOwnGroups(ctx, scope, groupID); err != nil {
		return nil, err
	}

	slotID := strings.TrimSpace(in.SlotID)
	if slotID == "" {
		slotID = "tts_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	}

	var staffArg, roomArg, effToArg any
	trimmedStaff := strings.TrimSpace(in.StaffID)
	if trimmedStaff != "" {
		staffArg = trimmedStaff
	}
	trimmedRoom := strings.TrimSpace(in.RoomID)
	if trimmedRoom != "" {
		roomArg = trimmedRoom
	}
	trimmedEffTo := strings.TrimSpace(in.EffectiveTo)
	if trimmedEffTo != "" {
		effToArg = trimmedEffTo
	}

	tx, err := d.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin upsert timetable_slot tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO timetable_slots (
			id, timetable_period_id, group_id, staff_id, room_id, effective_from, effective_to
		) VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(timetable_period_id, group_id, effective_from) DO UPDATE SET
			staff_id = excluded.staff_id,
			room_id = excluded.room_id,
			effective_to = excluded.effective_to`,
		slotID, periodID, groupID, staffArg, roomArg, effFrom, effToArg,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert timetable_slot row: %w", err)
	}

	var persistedID string
	if err := tx.QueryRowContext(
		ctx,
		`SELECT id FROM timetable_slots WHERE timetable_period_id = ? AND group_id = ? AND effective_from = ?`,
		periodID, groupID, effFrom,
	).Scan(&persistedID); err == nil && strings.TrimSpace(persistedID) != "" {
		slotID = persistedID
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit upsert timetable_slot tx: %w", err)
	}

	return &UpsertTimetableSlotResult{
		SlotID:            slotID,
		TimetablePeriodID: periodID,
		GroupID:           groupID,
		StaffID:           trimmedStaff,
		RoomID:            trimmedRoom,
		EffectiveFrom:     effFrom,
		EffectiveTo:       trimmedEffTo,
	}, nil
}

func (d *CoreDAL) GetPupilTimetableSlotsScoped(
	ctx context.Context,
	scope DALQueryScope,
	pupilID string,
	cycleWeek int,
	dayOfWeekID int,
) ([]TimetableSlotSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	trimmedPupil := strings.TrimSpace(pupilID)
	if trimmedPupil == "" {
		return nil, errors.New("missing mandatory parameter: pupil_id")
	}
	if scope.Mode == ScopeOwnGroups {
		if err := d.ensurePupilInOwnGroups(ctx, scope, trimmedPupil); err != nil {
			return nil, err
		}
	}

	query := fmt.Sprintf(`
		SELECT %s
		FROM group_pupil_memberships gpm
		JOIN timetable_slots ts ON ts.group_id = gpm.group_id
		JOIN timetable_periods tp ON tp.id = ts.timetable_period_id
		JOIN groups g ON g.id = ts.group_id
		WHERE gpm.pupil_id = ?
		  AND gpm.academic_year_id = ?
		  AND gpm.effective_from <= ?
		  AND (gpm.effective_to IS NULL OR gpm.effective_to >= ?)
		  AND tp.academic_year_id = ?
		  AND ts.effective_from <= ?
		  AND (ts.effective_to IS NULL OR ts.effective_to >= ?)`, timetableSlotFieldsSQL)
	args := []any{
		trimmedPupil,
		scope.AcademicYearID, scope.ReferenceDate, scope.ReferenceDate,
		scope.AcademicYearID, scope.ReferenceDate, scope.ReferenceDate,
	}
	if cycleWeek > 0 {
		query += ` AND tp.cycle_week = ?`
		args = append(args, cycleWeek)
	}
	if dayOfWeekID > 0 {
		query += ` AND day_of_week_id = ?`
		args = append(args, dayOfWeekID)
	}
	query += ` ORDER BY tp.cycle_week ASC, tp.day_of_week_id ASC, tp.sequence_number ASC, ts.id ASC`

	rows, err := d.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query pupil timetable slots: %w", err)
	}
	defer rows.Close()

	result := make([]TimetableSlotSummary, 0)
	for rows.Next() {
		slot, err := scanTimetableSlot(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pupil timetable slot: %w", err)
		}
		result = append(result, slot)
	}
	return result, rows.Err()
}
