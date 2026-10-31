package dal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

type TimetableSlotSummary struct {
	SlotID                 string `json:"slot_id"`
	PeriodID               string `json:"period_id"`
	PeriodCode             string `json:"period_code"`
	PeriodName             string `json:"period_name"`
	CycleWeek              int    `json:"cycle_week"`
	DayOfWeekID            int    `json:"day_of_week_id"`
	StartTime              string `json:"start_time"`
	EndTime                string `json:"end_time"`
	PeriodType             string `json:"period_type"`
	MapsToStatutorySession string `json:"maps_to_statutory_session"`
	GroupID                string `json:"group_id"`
	GroupCode              string `json:"group_code"`
	StaffID                string `json:"staff_id"`
	RoomID                 string `json:"room_id"`
}

func (d *CoreDAL) GetStaffTimetableSlotsScoped(
	ctx context.Context,
	scope DALQueryScope,
	staffID string,
	cycleWeek int,
	dayOfWeekID int,
) ([]TimetableSlotSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	targetStaffID := strings.TrimSpace(staffID)
	if targetStaffID == "" {
		return nil, errors.New("missing mandatory parameter: staff_id")
	}
	if scope.Mode == ScopeOwnGroups && targetStaffID != scope.StaffID {
		return nil, fmt.Errorf("access denied under own_groups scope: staff %s cannot read timetable for staff %s", scope.StaffID, targetStaffID)
	}
	if cycleWeek != 1 && cycleWeek != 2 {
		return nil, fmt.Errorf("invalid cycleWeek %d: must be 1 or 2", cycleWeek)
	}

	rows, err := d.readDB.QueryContext(
		ctx,
		`SELECT ts.id, tp.id, tp.period_code, tp.name, tp.cycle_week, tp.day_of_week_id,
		        tp.start_time, tp.end_time, tp.period_type, COALESCE(tp.maps_to_statutory_session, ''),
		        ts.group_id, g.code, COALESCE(ts.staff_id, ''), COALESCE(ts.room_id, '')
		 FROM timetable_slots ts
		 JOIN timetable_periods tp ON tp.id = ts.timetable_period_id
		 JOIN groups g ON g.id = ts.group_id
		 WHERE ts.staff_id = ?
		   AND tp.academic_year_id = ?
		   AND tp.cycle_week = ?
		   AND tp.day_of_week_id = ?
		   AND ts.effective_from <= ?
		   AND (ts.effective_to IS NULL OR ts.effective_to >= ?)
		 ORDER BY tp.sequence_number ASC, ts.id ASC`,
		targetStaffID, scope.AcademicYearID, cycleWeek, dayOfWeekID, scope.ReferenceDate, scope.ReferenceDate,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query staff timetable slots: %w", err)
	}
	defer rows.Close()

	var result []TimetableSlotSummary
	for rows.Next() {
		var slot TimetableSlotSummary
		if err := rows.Scan(
			&slot.SlotID,
			&slot.PeriodID,
			&slot.PeriodCode,
			&slot.PeriodName,
			&slot.CycleWeek,
			&slot.DayOfWeekID,
			&slot.StartTime,
			&slot.EndTime,
			&slot.PeriodType,
			&slot.MapsToStatutorySession,
			&slot.GroupID,
			&slot.GroupCode,
			&slot.StaffID,
			&slot.RoomID,
		); err != nil {
			return nil, fmt.Errorf("failed to scan timetable slot: %w", err)
		}
		result = append(result, slot)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating timetable slots: %w", err)
	}
	return result, nil
}

func (d *CoreDAL) GetStaffTimetableSlotsBatchScoped(
	ctx context.Context,
	scope DALQueryScope,
	staffIDs []string,
	cycleWeek int,
	dayOfWeekID int,
) (map[string][]TimetableSlotSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	if len(staffIDs) == 0 {
		return nil, errors.New("missing mandatory parameter: staff_ids")
	}
	if cycleWeek != 1 && cycleWeek != 2 {
		return nil, fmt.Errorf("invalid cycleWeek %d: must be 1 or 2", cycleWeek)
	}

	result := make(map[string][]TimetableSlotSummary, len(staffIDs))
	uniqueStaffIDs := make([]string, 0, len(staffIDs))
	seen := make(map[string]struct{}, len(staffIDs))
	for _, rawID := range staffIDs {
		sid := strings.TrimSpace(rawID)
		if sid == "" {
			return nil, errors.New("staff_ids batch contains an empty staff_id")
		}
		if scope.Mode == ScopeOwnGroups && sid != scope.StaffID {
			return nil, fmt.Errorf("access denied under own_groups scope: staff %s cannot read timetable for staff %s", scope.StaffID, sid)
		}
		if _, exists := seen[sid]; !exists {
			seen[sid] = struct{}{}
			uniqueStaffIDs = append(uniqueStaffIDs, sid)
			result[sid] = []TimetableSlotSummary{}
		}
	}

	placeholders := strings.Repeat("?,", len(uniqueStaffIDs))
	placeholders = strings.TrimSuffix(placeholders, ",")
	args := make([]any, 0, len(uniqueStaffIDs)+5)
	for _, sid := range uniqueStaffIDs {
		args = append(args, sid)
	}
	args = append(args, scope.AcademicYearID, cycleWeek, dayOfWeekID, scope.ReferenceDate, scope.ReferenceDate)

	rows, err := d.readDB.QueryContext(
		ctx,
		fmt.Sprintf(
			`SELECT ts.id, tp.id, tp.period_code, tp.name, tp.cycle_week, tp.day_of_week_id,
			        tp.start_time, tp.end_time, tp.period_type, COALESCE(tp.maps_to_statutory_session, ''),
			        ts.group_id, g.code, COALESCE(ts.staff_id, ''), COALESCE(ts.room_id, '')
			 FROM timetable_slots ts
			 JOIN timetable_periods tp ON tp.id = ts.timetable_period_id
			 JOIN groups g ON g.id = ts.group_id
			 WHERE ts.staff_id IN (%s)
			   AND tp.academic_year_id = ?
			   AND tp.cycle_week = ?
			   AND tp.day_of_week_id = ?
			   AND ts.effective_from <= ?
			   AND (ts.effective_to IS NULL OR ts.effective_to >= ?)
			 ORDER BY ts.staff_id ASC, tp.sequence_number ASC, ts.id ASC`,
			placeholders,
		),
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query staff timetable slots batch: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var slot TimetableSlotSummary
		if err := rows.Scan(
			&slot.SlotID,
			&slot.PeriodID,
			&slot.PeriodCode,
			&slot.PeriodName,
			&slot.CycleWeek,
			&slot.DayOfWeekID,
			&slot.StartTime,
			&slot.EndTime,
			&slot.PeriodType,
			&slot.MapsToStatutorySession,
			&slot.GroupID,
			&slot.GroupCode,
			&slot.StaffID,
			&slot.RoomID,
		); err != nil {
			return nil, fmt.Errorf("failed to scan batch timetable slot: %w", err)
		}
		result[slot.StaffID] = append(result[slot.StaffID], slot)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating batch timetable slots: %w", err)
	}
	return result, nil
}

func (d *CoreDAL) ListFreeStaffForPeriodScoped(
	ctx context.Context,
	scope DALQueryScope,
	cycleWeek int,
	dayOfWeekID int,
	periodIdentifier string,
) ([]StaffSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	trimmedPeriod := strings.TrimSpace(periodIdentifier)
	if trimmedPeriod == "" {
		return nil, errors.New("missing mandatory parameter: period_id or period_code")
	}
	if cycleWeek != 1 && cycleWeek != 2 {
		return nil, fmt.Errorf("invalid cycleWeek %d: must be 1 or 2", cycleWeek)
	}

	rows, err := d.readDB.QueryContext(
		ctx,
		`SELECT s.id, s.person_id, s.staff_code,
		        COALESCE(p.title, ''),
		        COALESCE(NULLIF(TRIM(p.preferred_first_name), ''), p.legal_first_name),
		        COALESCE(NULLIF(TRIM(p.preferred_last_name), ''), p.legal_last_name),
		        p.legal_first_name, p.legal_last_name,
		        COALESCE(s.job_title, ''), COALESCE(s.staff_type, ''), s.is_dsl
		 FROM staff s
		 JOIN persons p ON p.id = s.person_id
		 WHERE p.status = 'ACTIVE'
		   AND (s.termination_date IS NULL OR TRIM(s.termination_date) = '')
		   AND NOT EXISTS (
		       SELECT 1
		       FROM timetable_slots ts
		       JOIN timetable_periods tp ON tp.id = ts.timetable_period_id
		       WHERE ts.staff_id = s.id
		         AND tp.academic_year_id = ?
		         AND tp.cycle_week = ?
		         AND tp.day_of_week_id = ?
		         AND (tp.id = ? OR tp.period_code = ?)
		         AND ts.effective_from <= ?
		         AND (ts.effective_to IS NULL OR ts.effective_to >= ?)
		   )
		 ORDER BY p.legal_last_name ASC, p.legal_first_name ASC, s.id ASC`,
		scope.AcademicYearID, cycleWeek, dayOfWeekID, trimmedPeriod, trimmedPeriod, scope.ReferenceDate, scope.ReferenceDate,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list free staff for period: %w", err)
	}
	defer rows.Close()

	var result []StaffSummary
	for rows.Next() {
		var st StaffSummary
		var title, prefFirst, prefLast, legFirst, legLast string
		var isDSLInt int
		if err := rows.Scan(
			&st.StaffID, &st.PersonID, &st.StaffCode,
			&title, &prefFirst, &prefLast, &legFirst, &legLast,
			&st.JobTitle, &st.StaffType, &isDSLInt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan free staff row: %w", err)
		}
		st.FirstName = prefFirst
		st.LastName = prefLast
		st.DisplayName = formatPersonaDisplayName(title, prefFirst, legFirst, prefLast, legLast)
		st.IsDSL = isDSLInt == 1
		result = append(result, st)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating free staff rows: %w", err)
	}
	return result, nil
}

type StaffSummary struct {
	StaffID     string `json:"staff_id"`
	PersonID    string `json:"person_id"`
	StaffCode   string `json:"staff_code"`
	DisplayName string `json:"display_name"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	JobTitle    string `json:"job_title"`
	StaffType   string `json:"staff_type"`
	IsDSL       bool   `json:"is_dsl"`
}

func (d *CoreDAL) ListActiveStaff(ctx context.Context) ([]StaffSummary, error) {
	rows, err := d.readDB.QueryContext(
		ctx,
		`SELECT s.id, s.person_id, s.staff_code,
		        COALESCE(per.title, ''),
		        COALESCE(NULLIF(TRIM(per.preferred_first_name), ''), per.legal_first_name),
		        COALESCE(NULLIF(TRIM(per.preferred_last_name), ''), per.legal_last_name),
		        per.legal_first_name, per.legal_last_name,
		        COALESCE(s.job_title, ''), COALESCE(s.staff_type, ''), s.is_dsl
		 FROM staff s
		 JOIN persons per ON per.id = s.person_id
		 WHERE per.status = 'ACTIVE'
		   AND (s.termination_date IS NULL OR TRIM(s.termination_date) = '')
		 ORDER BY per.legal_last_name ASC, per.legal_first_name ASC, s.id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list active staff: %w", err)
	}
	defer rows.Close()

	result := make([]StaffSummary, 0)
	for rows.Next() {
		var st StaffSummary
		var title, prefFirst, prefLast, legFirst, legLast string
		var isDSLInt int
		if err := rows.Scan(
			&st.StaffID, &st.PersonID, &st.StaffCode,
			&title, &prefFirst, &prefLast, &legFirst, &legLast,
			&st.JobTitle, &st.StaffType, &isDSLInt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan active staff row: %w", err)
		}
		st.FirstName = prefFirst
		st.LastName = prefLast
		st.DisplayName = formatPersonaDisplayName(title, prefFirst, legFirst, prefLast, legLast)
		st.IsDSL = isDSLInt == 1
		result = append(result, st)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating active staff: %w", err)
	}
	return result, nil
}

func (d *CoreDAL) GetStaffByID(ctx context.Context, staffID string) (*StaffSummary, error) {
	trimmed := strings.TrimSpace(staffID)
	if trimmed == "" {
		return nil, errors.New("missing mandatory parameter: staff_id")
	}

	var st StaffSummary
	var title, prefFirst, prefLast, legFirst, legLast string
	var isDSLInt int
	err := d.readDB.QueryRowContext(
		ctx,
		`SELECT s.id, s.person_id, s.staff_code,
		        COALESCE(per.title, ''),
		        COALESCE(NULLIF(TRIM(per.preferred_first_name), ''), per.legal_first_name),
		        COALESCE(NULLIF(TRIM(per.preferred_last_name), ''), per.legal_last_name),
		        per.legal_first_name, per.legal_last_name,
		        COALESCE(s.job_title, ''), COALESCE(s.staff_type, ''), s.is_dsl
		 FROM staff s
		 JOIN persons per ON per.id = s.person_id
		 WHERE s.id = ?
		   AND per.status = 'ACTIVE'
		   AND (s.termination_date IS NULL OR TRIM(s.termination_date) = '')`,
		trimmed,
	).Scan(
		&st.StaffID, &st.PersonID, &st.StaffCode,
		&title, &prefFirst, &prefLast, &legFirst, &legLast,
		&st.JobTitle, &st.StaffType, &isDSLInt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("active staff not found: %s", trimmed)
		}
		return nil, fmt.Errorf("failed to query staff %s: %w", trimmed, err)
	}
	st.FirstName = prefFirst
	st.LastName = prefLast
	st.DisplayName = formatPersonaDisplayName(title, prefFirst, legFirst, prefLast, legLast)
	st.IsDSL = isDSLInt == 1
	return &st, nil
}
