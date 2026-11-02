package dal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
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

type StaffSCRRecord struct {
	StaffID                string `json:"staff_id"`
	PersonID               string `json:"person_id"`
	StaffCode              string `json:"staff_code"`
	TRN                    string `json:"trn,omitempty"`
	DisplayName            string `json:"display_name"`
	FirstName              string `json:"first_name"`
	LastName               string `json:"last_name"`
	StaffType              string `json:"staff_type"`
	JobTitle               string `json:"job_title"`
	QTSFlag                bool   `json:"qts_flag"`
	EYTSFlag               bool   `json:"eyts_flag"`
	HLTAFlag               bool   `json:"hlta_flag"`
	IsDSL                  bool   `json:"is_dsl"`
	FirstAidQualified      bool   `json:"first_aid_qualified"`
	FirstAidExpiryDate     string `json:"first_aid_expiry_date,omitempty"`
	DBSCheckDate           string `json:"dbs_check_date,omitempty"`
	DBSCertificateNumber   string `json:"dbs_certificate_number,omitempty"`
	ProhibitionCheckDate   string `json:"prohibition_check_date,omitempty"`
	RightToWorkCheckDate   string `json:"right_to_work_check_date,omitempty"`
	HireDate               string `json:"hire_date"`
	TerminationDate        string `json:"termination_date,omitempty"`
}

func (d *CoreDAL) ListStaffSCRRegister(ctx context.Context) ([]StaffSCRRecord, error) {
	rows, err := d.readDB.QueryContext(
		ctx,
		`SELECT s.id, s.person_id, COALESCE(s.staff_code, ''), COALESCE(s.trn, ''),
		        COALESCE(per.title, ''),
		        COALESCE(NULLIF(TRIM(per.preferred_first_name), ''), per.legal_first_name),
		        COALESCE(NULLIF(TRIM(per.preferred_last_name), ''), per.legal_last_name),
		        per.legal_first_name, per.legal_last_name,
		        COALESCE(s.staff_type, ''), COALESCE(s.job_title, ''),
		        s.qts_flag, s.eyts_flag, s.hlta_flag, s.is_dsl,
		        s.first_aid_qualified, COALESCE(s.first_aid_expiry_date, ''),
		        COALESCE(s.dbs_check_date, ''), COALESCE(s.dbs_certificate_number, ''),
		        COALESCE(s.prohibition_check_date, ''), COALESCE(s.right_to_work_check_date, ''),
		        COALESCE(s.hire_date, ''), COALESCE(s.termination_date, '')
		 FROM staff s
		 JOIN persons per ON per.id = s.person_id
		 ORDER BY per.legal_last_name ASC, per.legal_first_name ASC, s.id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list staff SCR register: %w", err)
	}
	defer rows.Close()

	result := make([]StaffSCRRecord, 0)
	for rows.Next() {
		var r StaffSCRRecord
		var title, prefFirst, prefLast, legFirst, legLast string
		var qtsInt, eytsInt, hltaInt, dslInt, faInt int
		if err := rows.Scan(
			&r.StaffID, &r.PersonID, &r.StaffCode, &r.TRN,
			&title, &prefFirst, &prefLast, &legFirst, &legLast,
			&r.StaffType, &r.JobTitle,
			&qtsInt, &eytsInt, &hltaInt, &dslInt,
			&faInt, &r.FirstAidExpiryDate,
			&r.DBSCheckDate, &r.DBSCertificateNumber,
			&r.ProhibitionCheckDate, &r.RightToWorkCheckDate,
			&r.HireDate, &r.TerminationDate,
		); err != nil {
			return nil, fmt.Errorf("failed to scan staff SCR row: %w", err)
		}
		r.FirstName = prefFirst
		r.LastName = prefLast
		r.DisplayName = formatPersonaDisplayName(title, prefFirst, legFirst, prefLast, legLast)
		r.QTSFlag = qtsInt == 1
		r.EYTSFlag = eytsInt == 1
		r.HLTAFlag = hltaInt == 1
		r.IsDSL = dslInt == 1
		r.FirstAidQualified = faInt == 1
		result = append(result, r)
	}
	return result, rows.Err()
}

type UpsertStaffSCRInput struct {
	StaffID              string `json:"staff_id"`
	TRN                  string `json:"trn,omitempty"`
	DBSCheckDate         string `json:"dbs_check_date,omitempty"`
	DBSCertificateNumber string `json:"dbs_certificate_number,omitempty"`
	ProhibitionCheckDate string `json:"prohibition_check_date,omitempty"`
	RightToWorkCheckDate string `json:"right_to_work_check_date,omitempty"`
	FirstAidQualified    *bool  `json:"first_aid_qualified,omitempty"`
	FirstAidExpiryDate   string `json:"first_aid_expiry_date,omitempty"`
	QTSFlag              *bool  `json:"qts_flag,omitempty"`
	HLTAFlag             *bool  `json:"hlta_flag,omitempty"`
}

func (d *CoreDAL) UpsertStaffSCRRecord(ctx context.Context, in UpsertStaffSCRInput) (*StaffSCRRecord, error) {
	staffID := strings.TrimSpace(in.StaffID)
	if staffID == "" {
		return nil, errors.New("missing mandatory parameter: staff_id")
	}

	var trnArg, dbsDateArg, dbsNoArg, prohibArg, rtwArg, faExpArg any
	if strings.TrimSpace(in.TRN) != "" {
		trnArg = strings.TrimSpace(in.TRN)
	}
	if strings.TrimSpace(in.DBSCheckDate) != "" {
		dbsDateArg = strings.TrimSpace(in.DBSCheckDate)
	}
	if strings.TrimSpace(in.DBSCertificateNumber) != "" {
		dbsNoArg = strings.TrimSpace(in.DBSCertificateNumber)
	}
	if strings.TrimSpace(in.ProhibitionCheckDate) != "" {
		prohibArg = strings.TrimSpace(in.ProhibitionCheckDate)
	}
	if strings.TrimSpace(in.RightToWorkCheckDate) != "" {
		rtwArg = strings.TrimSpace(in.RightToWorkCheckDate)
	}
	if strings.TrimSpace(in.FirstAidExpiryDate) != "" {
		faExpArg = strings.TrimSpace(in.FirstAidExpiryDate)
	}
	var faFlagArg, qtsFlagArg, hltaFlagArg any
	if in.FirstAidQualified != nil {
		if *in.FirstAidQualified {
			faFlagArg = 1
		} else {
			faFlagArg = 0
		}
	}
	if in.QTSFlag != nil {
		if *in.QTSFlag {
			qtsFlagArg = 1
		} else {
			qtsFlagArg = 0
		}
	}
	if in.HLTAFlag != nil {
		if *in.HLTAFlag {
			hltaFlagArg = 1
		} else {
			hltaFlagArg = 0
		}
	}

	res, err := d.writeDB.ExecContext(
		ctx,
		`UPDATE staff
		 SET trn = COALESCE(?, trn),
		     dbs_check_date = COALESCE(?, dbs_check_date),
		     dbs_certificate_number = COALESCE(?, dbs_certificate_number),
		     prohibition_check_date = COALESCE(?, prohibition_check_date),
		     right_to_work_check_date = COALESCE(?, right_to_work_check_date),
		     first_aid_qualified = COALESCE(?, first_aid_qualified),
		     first_aid_expiry_date = COALESCE(?, first_aid_expiry_date),
		     qts_flag = COALESCE(?, qts_flag),
		     hlta_flag = COALESCE(?, hlta_flag)
		 WHERE id = ?`,
		trnArg, dbsDateArg, dbsNoArg, prohibArg, rtwArg, faFlagArg, faExpArg, qtsFlagArg, hltaFlagArg, staffID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update staff SCR record: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, fmt.Errorf("staff not found: %s", staffID)
	}

	all, err := d.ListStaffSCRRegister(ctx)
	if err != nil {
		return nil, err
	}
	for _, r := range all {
		if r.StaffID == staffID {
			return &r, nil
		}
	}
	return nil, fmt.Errorf("staff not found after SCR update: %s", staffID)
}

type StaffContractRecord struct {
	ID            string  `json:"id"`
	ContractID    string  `json:"contract_id"`
	StaffID       string  `json:"staff_id"`
	ContractType  string  `json:"contract_type"`
	SWCPostCode   string  `json:"swc_post_code"`
	JobTitle      string  `json:"job_title"`
	PayScale      string  `json:"pay_scale,omitempty"`
	PaySpinePoint string  `json:"pay_spine_point,omitempty"`
	FTE           float64 `json:"fte"`
	HoursPerWeek  float64 `json:"hours_per_week,omitempty"`
	WeeksPerYear  float64 `json:"weeks_per_year,omitempty"`
	EffectiveFrom string  `json:"effective_from"`
	EffectiveTo   string  `json:"effective_to,omitempty"`
}

func (d *CoreDAL) ListStaffContracts(ctx context.Context, staffID string) ([]StaffContractRecord, error) {
	trimmed := strings.TrimSpace(staffID)
	query := `
		SELECT id, staff_id, contract_type, swc_post_code, job_title,
		       COALESCE(pay_scale, ''), COALESCE(pay_spine_point, ''),
		       fte, COALESCE(hours_per_week, 0), COALESCE(weeks_per_year, 0),
		       effective_from, COALESCE(effective_to, '')
		FROM staff_contracts`
	var args []any
	if trimmed != "" {
		query += ` WHERE staff_id = ?`
		args = append(args, trimmed)
	}
	query += ` ORDER BY staff_id ASC, effective_from DESC, id ASC`

	rows, err := d.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list staff_contracts: %w", err)
	}
	defer rows.Close()

	result := make([]StaffContractRecord, 0)
	for rows.Next() {
		var r StaffContractRecord
		if err := rows.Scan(
			&r.ID, &r.StaffID, &r.ContractType, &r.SWCPostCode, &r.JobTitle,
			&r.PayScale, &r.PaySpinePoint,
			&r.FTE, &r.HoursPerWeek, &r.WeeksPerYear,
			&r.EffectiveFrom, &r.EffectiveTo,
		); err != nil {
			return nil, fmt.Errorf("failed to scan staff_contracts row: %w", err)
		}
		r.ContractID = r.ID
		result = append(result, r)
	}
	return result, rows.Err()
}

func (d *CoreDAL) UpsertStaffContract(ctx context.Context, in StaffContractRecord) (*StaffContractRecord, error) {
	staffID := strings.TrimSpace(in.StaffID)
	jobTitle := strings.TrimSpace(in.JobTitle)
	effFrom := strings.TrimSpace(in.EffectiveFrom)
	if staffID == "" || jobTitle == "" || effFrom == "" {
		return nil, errors.New("missing mandatory parameter: staff_id, job_title, and effective_from")
	}
	contractID := strings.TrimSpace(in.ContractID)
	if contractID == "" {
		contractID = strings.TrimSpace(in.ID)
	}
	if contractID == "" {
		contractID = "cnt_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	}
	cType := strings.ToUpper(strings.TrimSpace(in.ContractType))
	if cType == "" {
		cType = "PERMANENT"
	}
	swcPost := strings.ToUpper(strings.TrimSpace(in.SWCPostCode))
	if swcPost == "" {
		swcPost = "TCH"
	}
	fte := in.FTE
	if fte <= 0 {
		fte = 1.0
	}
	var effToArg any
	if strings.TrimSpace(in.EffectiveTo) != "" {
		effToArg = strings.TrimSpace(in.EffectiveTo)
	}

	if _, err := d.writeDB.ExecContext(
		ctx,
		`INSERT INTO staff_contracts (
			id, staff_id, contract_type, swc_post_code, job_title, pay_scale, pay_spine_point,
			fte, hours_per_week, weeks_per_year, effective_from, effective_to
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			contract_type = excluded.contract_type,
			swc_post_code = excluded.swc_post_code,
			job_title = excluded.job_title,
			pay_scale = excluded.pay_scale,
			pay_spine_point = excluded.pay_spine_point,
			fte = excluded.fte,
			hours_per_week = excluded.hours_per_week,
			weeks_per_year = excluded.weeks_per_year,
			effective_from = excluded.effective_from,
			effective_to = excluded.effective_to`,
		contractID, staffID, cType, swcPost, jobTitle, strings.TrimSpace(in.PayScale), strings.TrimSpace(in.PaySpinePoint),
		fte, in.HoursPerWeek, in.WeeksPerYear, effFrom, effToArg,
	); err != nil {
		return nil, fmt.Errorf("failed to upsert staff_contracts row: %w", err)
	}

	out := in
	out.ID = contractID
	out.ContractID = contractID
	out.StaffID = staffID
	out.ContractType = cType
	out.SWCPostCode = swcPost
	out.JobTitle = jobTitle
	out.FTE = fte
	out.EffectiveFrom = effFrom
	return &out, nil
}

type StaffAbsenceRecord struct {
	ID                  string  `json:"id"`
	AbsenceID           string  `json:"absence_id"`
	StaffID             string  `json:"staff_id"`
	AcademicYearID      string  `json:"academic_year_id"`
	AbsenceCategoryCode string  `json:"absence_category_code"`
	ReasonDetail        string  `json:"reason_detail"`
	Status              string  `json:"status"`
	CoverRequired       bool    `json:"cover_required"`
	StartDate           string  `json:"start_date"`
	EndDate             string  `json:"end_date"`
	WorkingDaysLost     float64 `json:"working_days_lost"`
	ApprovedByStaffID   string  `json:"approved_by_staff_id,omitempty"`
}

func (d *CoreDAL) ListStaffAbsences(ctx context.Context, staffID string, academicYearID string) ([]StaffAbsenceRecord, error) {
	trimmedStaff := strings.TrimSpace(staffID)
	trimmedAY := strings.TrimSpace(academicYearID)
	query := `
		SELECT id, staff_id, academic_year_id, absence_category_code, reason_detail,
		       status, cover_required, start_date, end_date, working_days_lost, COALESCE(approved_by_staff_id, '')
		FROM staff_absences WHERE 1=1`
	var args []any
	if trimmedStaff != "" {
		query += ` AND staff_id = ?`
		args = append(args, trimmedStaff)
	}
	if trimmedAY != "" {
		query += ` AND academic_year_id = ?`
		args = append(args, trimmedAY)
	}
	query += ` ORDER BY start_date DESC, id ASC`

	rows, err := d.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list staff_absences: %w", err)
	}
	defer rows.Close()

	result := make([]StaffAbsenceRecord, 0)
	for rows.Next() {
		var r StaffAbsenceRecord
		var covInt int
		if err := rows.Scan(
			&r.ID, &r.StaffID, &r.AcademicYearID, &r.AbsenceCategoryCode, &r.ReasonDetail,
			&r.Status, &covInt, &r.StartDate, &r.EndDate, &r.WorkingDaysLost, &r.ApprovedByStaffID,
		); err != nil {
			return nil, fmt.Errorf("failed to scan staff_absences row: %w", err)
		}
		r.AbsenceID = r.ID
		r.CoverRequired = covInt == 1
		result = append(result, r)
	}
	return result, rows.Err()
}

func (d *CoreDAL) UpsertStaffAbsence(ctx context.Context, in StaffAbsenceRecord) (*StaffAbsenceRecord, error) {
	staffID := strings.TrimSpace(in.StaffID)
	ayID := strings.TrimSpace(in.AcademicYearID)
	catCode := strings.ToUpper(strings.TrimSpace(in.AbsenceCategoryCode))
	startDate := strings.TrimSpace(in.StartDate)
	endDate := strings.TrimSpace(in.EndDate)
	if endDate == "" {
		endDate = startDate
	}
	if staffID == "" || ayID == "" || catCode == "" || startDate == "" {
		return nil, errors.New("missing mandatory parameter: staff_id, academic_year_id, absence_category_code, and start_date")
	}
	absID := strings.TrimSpace(in.AbsenceID)
	if absID == "" {
		absID = strings.TrimSpace(in.ID)
	}
	if absID == "" {
		absID = "sabs_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	}
	status := strings.ToUpper(strings.TrimSpace(in.Status))
	if status == "" {
		status = "APPROVED"
	}
	covInt := 0
	if in.CoverRequired {
		covInt = 1
	}
	daysLost := in.WorkingDaysLost
	if daysLost <= 0 {
		daysLost = 1.0
	}
	var appByArg any
	if strings.TrimSpace(in.ApprovedByStaffID) != "" {
		appByArg = strings.TrimSpace(in.ApprovedByStaffID)
	}

	if _, err := d.writeDB.ExecContext(
		ctx,
		`INSERT INTO staff_absences (
			id, staff_id, academic_year_id, absence_category_code, reason_detail,
			status, cover_required, start_date, end_date, working_days_lost, approved_by_staff_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			absence_category_code = excluded.absence_category_code,
			reason_detail = excluded.reason_detail,
			status = excluded.status,
			cover_required = excluded.cover_required,
			start_date = excluded.start_date,
			end_date = excluded.end_date,
			working_days_lost = excluded.working_days_lost,
			approved_by_staff_id = COALESCE(excluded.approved_by_staff_id, staff_absences.approved_by_staff_id)`,
		absID, staffID, ayID, catCode, strings.TrimSpace(in.ReasonDetail),
		status, covInt, startDate, endDate, daysLost, appByArg,
	); err != nil {
		return nil, fmt.Errorf("failed to upsert staff_absences row: %w", err)
	}

	out := in
	out.ID = absID
	out.AbsenceID = absID
	out.StaffID = staffID
	out.AcademicYearID = ayID
	out.AbsenceCategoryCode = catCode
	out.Status = status
	out.StartDate = startDate
	out.EndDate = endDate
	out.WorkingDaysLost = daysLost
	return &out, nil
}

type ListStaffOptions struct {
	IncludeLeavers bool   `json:"include_leavers"`
	StaffType      string `json:"staff_type,omitempty"`
}

type StaffMemberDetail struct {
	StaffID            string `json:"staff_id"`
	PersonID           string `json:"person_id"`
	StaffCode          string `json:"staff_code"`
	DisplayName        string `json:"display_name"`
	Title              string `json:"title,omitempty"`
	FirstName          string `json:"first_name"`
	LastName           string `json:"last_name"`
	PreferredFirstName string `json:"preferred_first_name,omitempty"`
	PreferredLastName  string `json:"preferred_last_name,omitempty"`
	Email              string `json:"email,omitempty"`
	PhonePrimary       string `json:"phone_primary,omitempty"`
	Sex                string `json:"sex,omitempty"`
	DOB                string `json:"dob,omitempty"`
	Status             string `json:"status,omitempty"`
	JobTitle           string `json:"job_title"`
	StaffType          string `json:"staff_type"`
	TRN                string `json:"trn,omitempty"`
	QTSFlag            bool   `json:"qts_flag"`
	EYTSFlag           bool   `json:"eyts_flag"`
	HLTAFlag           bool   `json:"hlta_flag"`
	IsDSL              bool   `json:"is_dsl"`
	ExcludeFromCover   bool   `json:"exclude_from_cover"`
	CoverPriority      int    `json:"cover_priority"`
	HireDate           string `json:"hire_date,omitempty"`
	TerminationDate    string `json:"termination_date,omitempty"`
}

func (d *CoreDAL) ListAllStaff(ctx context.Context, opts ListStaffOptions) ([]StaffMemberDetail, error) {
	query := `
		SELECT s.id, s.person_id, s.staff_code,
		       COALESCE(per.title, ''),
		       COALESCE(NULLIF(TRIM(per.preferred_first_name), ''), per.legal_first_name),
		       COALESCE(NULLIF(TRIM(per.preferred_last_name), ''), per.legal_last_name),
		       per.legal_first_name, per.legal_last_name,
		       COALESCE(per.email, ''), COALESCE(per.phone_primary, ''),
		       COALESCE(per.sex, 'U'), COALESCE(per.dob, ''), COALESCE(per.status, 'ACTIVE'),
		       COALESCE(s.job_title, ''), COALESCE(s.staff_type, ''), COALESCE(s.trn, ''),
		       s.qts_flag, s.eyts_flag, s.hlta_flag, s.is_dsl,
		       s.exclude_from_cover, s.cover_priority,
		       COALESCE(s.hire_date, ''), COALESCE(s.termination_date, '')
		FROM staff s
		JOIN persons per ON per.id = s.person_id
		WHERE 1=1`
	var args []any
	if !opts.IncludeLeavers {
		query += ` AND per.status = 'ACTIVE' AND (s.termination_date IS NULL OR TRIM(s.termination_date) = '')`
	}
	if strings.TrimSpace(opts.StaffType) != "" {
		query += ` AND s.staff_type = ?`
		args = append(args, strings.ToUpper(strings.TrimSpace(opts.StaffType)))
	}
	query += ` ORDER BY per.legal_last_name ASC, per.legal_first_name ASC, s.id ASC`

	rows, err := d.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list staff: %w", err)
	}
	defer rows.Close()

	result := make([]StaffMemberDetail, 0)
	for rows.Next() {
		var st StaffMemberDetail
		var prefFirst, prefLast, legFirst, legLast string
		var qtsInt, eytsInt, hltaInt, dslInt, exclInt int
		if err := rows.Scan(
			&st.StaffID, &st.PersonID, &st.StaffCode,
			&st.Title, &prefFirst, &prefLast, &legFirst, &legLast,
			&st.Email, &st.PhonePrimary,
			&st.Sex, &st.DOB, &st.Status,
			&st.JobTitle, &st.StaffType, &st.TRN,
			&qtsInt, &eytsInt, &hltaInt, &dslInt,
			&exclInt, &st.CoverPriority,
			&st.HireDate, &st.TerminationDate,
		); err != nil {
			return nil, fmt.Errorf("failed to scan staff row: %w", err)
		}
		st.FirstName = legFirst
		st.LastName = legLast
		st.PreferredFirstName = prefFirst
		st.PreferredLastName = prefLast
		st.DisplayName = formatPersonaDisplayName(st.Title, prefFirst, legFirst, prefLast, legLast)
		st.QTSFlag = qtsInt == 1
		st.EYTSFlag = eytsInt == 1
		st.HLTAFlag = hltaInt == 1
		st.IsDSL = dslInt == 1
		st.ExcludeFromCover = exclInt == 1
		result = append(result, st)
	}
	return result, rows.Err()
}

type UpsertStaffMemberInput struct {
	StaffID            string   `json:"staff_id,omitempty"`
	PersonID           string   `json:"person_id,omitempty"`
	StaffCode          string   `json:"staff_code"`
	Title              string   `json:"title,omitempty"`
	FirstName          string   `json:"first_name"`
	LastName           string   `json:"last_name"`
	PreferredFirstName string   `json:"preferred_first_name,omitempty"`
	PreferredLastName  string   `json:"preferred_last_name,omitempty"`
	Email              string   `json:"email,omitempty"`
	PhonePrimary       string   `json:"phone_primary,omitempty"`
	Sex                string   `json:"sex,omitempty"`
	DOB                string   `json:"dob,omitempty"`
	Status             string   `json:"status,omitempty"`
	StaffType          string   `json:"staff_type"`
	JobTitle           string   `json:"job_title"`
	TRN                string   `json:"trn,omitempty"`
	QTSFlag            bool     `json:"qts_flag"`
	EYTSFlag           bool     `json:"eyts_flag"`
	HLTAFlag           bool     `json:"hlta_flag"`
	IsDSL              bool     `json:"is_dsl"`
	ExcludeFromCover   bool     `json:"exclude_from_cover"`
	CoverPriority      int      `json:"cover_priority"`
	HireDate           string   `json:"hire_date"`
	TerminationDate    string   `json:"termination_date,omitempty"`
	Roles              []string `json:"roles,omitempty"`
}

func (d *CoreDAL) UpsertStaffMember(ctx context.Context, in UpsertStaffMemberInput) (*StaffMemberDetail, error) {
	staffCode := strings.ToUpper(strings.TrimSpace(in.StaffCode))
	firstName := strings.TrimSpace(in.FirstName)
	lastName := strings.TrimSpace(in.LastName)
	staffType := strings.ToUpper(strings.TrimSpace(in.StaffType))
	if staffType == "" {
		staffType = "TEACHING"
	}
	hireDate := strings.TrimSpace(in.HireDate)
	if hireDate == "" {
		hireDate = "2026-09-01"
	}
	if staffCode == "" || firstName == "" || lastName == "" {
		return nil, errors.New("missing mandatory parameter: staff_code, first_name, and last_name are required")
	}
	switch staffType {
	case "TEACHING", "SUPPORT", "LEADERSHIP", "SUPPLY_INTERNAL", "SUPPLY_AGENCY":
	default:
		return nil, fmt.Errorf("invalid staff_type %q", in.StaffType)
	}

	uid := strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	staffID := strings.TrimSpace(in.StaffID)
	if staffID == "" {
		staffID = "stf_" + uid
	}
	personID := strings.TrimSpace(in.PersonID)
	if personID == "" {
		personID = "per_" + uid
	}

	tx, err := d.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin UpsertStaffMember tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var existingStaffID, existingPersonID string
	if err := tx.QueryRowContext(ctx, `SELECT id, person_id FROM staff WHERE staff_code = ? OR id = ? LIMIT 1`, staffCode, staffID).Scan(&existingStaffID, &existingPersonID); err == nil && existingStaffID != "" {
		staffID = existingStaffID
		personID = existingPersonID
	}

	prefFirst := strings.TrimSpace(in.PreferredFirstName)
	if prefFirst == "" {
		prefFirst = firstName
	}
	prefLast := strings.TrimSpace(in.PreferredLastName)
	if prefLast == "" {
		prefLast = lastName
	}
	sex := strings.ToUpper(strings.TrimSpace(in.Sex))
	if sex == "" {
		sex = "U"
	}
	status := strings.ToUpper(strings.TrimSpace(in.Status))
	if status == "" {
		if strings.TrimSpace(in.TerminationDate) != "" {
			status = "LEFT"
		} else {
			status = "ACTIVE"
		}
	}
	optStr := func(s string) any {
		if strings.TrimSpace(s) == "" {
			return nil
		}
		return strings.TrimSpace(s)
	}
	boolInt := func(b bool) int {
		if b {
			return 1
		}
		return 0
	}

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO persons (
			id, email, title, legal_first_name, legal_last_name,
			preferred_first_name, preferred_last_name, sex, dob, phone_primary, status
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			email = COALESCE(excluded.email, persons.email),
			title = excluded.title,
			legal_first_name = excluded.legal_first_name,
			legal_last_name = excluded.legal_last_name,
			preferred_first_name = excluded.preferred_first_name,
			preferred_last_name = excluded.preferred_last_name,
			sex = excluded.sex,
			dob = COALESCE(excluded.dob, persons.dob),
			phone_primary = excluded.phone_primary,
			status = excluded.status`,
		personID, optStr(in.Email), strings.TrimSpace(in.Title), firstName, lastName,
		prefFirst, prefLast, sex, optStr(in.DOB), strings.TrimSpace(in.PhonePrimary), status,
	); err != nil {
		return nil, fmt.Errorf("failed to upsert person for staff: %w", err)
	}

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO staff (
			id, person_id, staff_code, trn, staff_type, job_title,
			qts_flag, eyts_flag, hlta_flag, is_dsl,
			exclude_from_cover, cover_priority, hire_date, termination_date
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			staff_code = excluded.staff_code,
			trn = COALESCE(excluded.trn, staff.trn),
			staff_type = excluded.staff_type,
			job_title = excluded.job_title,
			qts_flag = excluded.qts_flag,
			eyts_flag = excluded.eyts_flag,
			hlta_flag = excluded.hlta_flag,
			is_dsl = excluded.is_dsl,
			exclude_from_cover = excluded.exclude_from_cover,
			cover_priority = excluded.cover_priority,
			hire_date = excluded.hire_date,
			termination_date = excluded.termination_date`,
		staffID, personID, staffCode, optStr(in.TRN), staffType, strings.TrimSpace(in.JobTitle),
		boolInt(in.QTSFlag), boolInt(in.EYTSFlag), boolInt(in.HLTAFlag), boolInt(in.IsDSL),
		boolInt(in.ExcludeFromCover), in.CoverPriority, hireDate, optStr(in.TerminationDate),
	); err != nil {
		return nil, fmt.Errorf("failed to upsert staff row: %w", err)
	}

	for idx, rawRole := range in.Roles {
		roleID := strings.ToUpper(strings.TrimSpace(rawRole))
		if roleID == "" {
			continue
		}
		isPrim := 0
		if idx == 0 {
			isPrim = 1
		}
		_, _ = tx.ExecContext(
			ctx,
			`INSERT INTO person_roles (person_id, role_id, is_primary, effective_from)
			 VALUES (?, ?, ?, ?)
			 ON CONFLICT(person_id, role_id, effective_from) DO NOTHING`,
			personID, roleID, isPrim, hireDate,
		)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit UpsertStaffMember tx: %w", err)
	}

	return &StaffMemberDetail{
		StaffID:            staffID,
		PersonID:           personID,
		StaffCode:          staffCode,
		DisplayName:        formatPersonaDisplayName(in.Title, prefFirst, firstName, prefLast, lastName),
		Title:              strings.TrimSpace(in.Title),
		FirstName:          firstName,
		LastName:           lastName,
		PreferredFirstName: prefFirst,
		PreferredLastName:  prefLast,
		Email:              strings.TrimSpace(in.Email),
		PhonePrimary:       strings.TrimSpace(in.PhonePrimary),
		Sex:                sex,
		DOB:                strings.TrimSpace(in.DOB),
		Status:             status,
		JobTitle:           strings.TrimSpace(in.JobTitle),
		StaffType:          staffType,
		TRN:                strings.TrimSpace(in.TRN),
		QTSFlag:            in.QTSFlag,
		EYTSFlag:           in.EYTSFlag,
		HLTAFlag:           in.HLTAFlag,
		IsDSL:              in.IsDSL,
		ExcludeFromCover:   in.ExcludeFromCover,
		CoverPriority:      in.CoverPriority,
		HireDate:           hireDate,
		TerminationDate:    strings.TrimSpace(in.TerminationDate),
	}, nil
}

type RoleSummary struct {
	ID                string `json:"id"`
	Category          string `json:"category"`
	Name              string `json:"name"`
	NameShort         string `json:"name_short"`
	Description       string `json:"description,omitempty"`
	DefaultScopesJSON string `json:"default_scopes_json"`
}

func (d *CoreDAL) ListRoles(ctx context.Context) ([]RoleSummary, error) {
	rows, err := d.readDB.QueryContext(
		ctx,
		`SELECT id, category, name, name_short, COALESCE(description, ''), COALESCE(default_scopes_json, '[]')
		 FROM roles
		 ORDER BY category ASC, id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}
	defer rows.Close()

	result := make([]RoleSummary, 0)
	for rows.Next() {
		var r RoleSummary
		if err := rows.Scan(&r.ID, &r.Category, &r.Name, &r.NameShort, &r.Description, &r.DefaultScopesJSON); err != nil {
			return nil, fmt.Errorf("failed to scan roles row: %w", err)
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

type StaffCoverAssignmentRecord struct {
	ID              string `json:"id"`
	CoverID         string `json:"cover_id"`
	StaffAbsenceID  string `json:"staff_absence_id,omitempty"`
	AbsenceID       string `json:"absence_id,omitempty"`
	TimetableSlotID string `json:"timetable_slot_id,omitempty"`
	Date            string `json:"date"`
	AbsentStaffID   string `json:"absent_staff_id"`
	CoveringStaffID string `json:"covering_staff_id,omitempty"`
	CoverRoomID     string `json:"cover_room_id,omitempty"`
	Status          string `json:"status"`
	CoverNotes      string `json:"cover_notes,omitempty"`
	Notes           string `json:"notes,omitempty"`
}

type ListStaffCoverAssignmentsOptions struct {
	Date            string `json:"date,omitempty"`
	AbsentStaffID   string `json:"absent_staff_id,omitempty"`
	CoveringStaffID string `json:"covering_staff_id,omitempty"`
	AbsenceID       string `json:"absence_id,omitempty"`
}

func (d *CoreDAL) ListStaffCoverAssignments(
	ctx context.Context,
	opts ListStaffCoverAssignmentsOptions,
) ([]StaffCoverAssignmentRecord, error) {
	query := `
		SELECT id, COALESCE(staff_absence_id, ''), COALESCE(timetable_slot_id, ''),
		       date, absent_staff_id, COALESCE(covering_staff_id, ''),
		       status, COALESCE(cover_notes, '')
		FROM staff_cover_assignments
		WHERE 1=1`
	var args []any
	if strings.TrimSpace(opts.Date) != "" {
		query += ` AND date = ?`
		args = append(args, strings.TrimSpace(opts.Date))
	}
	if strings.TrimSpace(opts.AbsentStaffID) != "" {
		query += ` AND absent_staff_id = ?`
		args = append(args, strings.TrimSpace(opts.AbsentStaffID))
	}
	if strings.TrimSpace(opts.CoveringStaffID) != "" {
		query += ` AND covering_staff_id = ?`
		args = append(args, strings.TrimSpace(opts.CoveringStaffID))
	}
	if strings.TrimSpace(opts.AbsenceID) != "" {
		query += ` AND staff_absence_id = ?`
		args = append(args, strings.TrimSpace(opts.AbsenceID))
	}
	query += ` ORDER BY date DESC, id ASC`

	rows, err := d.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list staff_cover_assignments: %w", err)
	}
	defer rows.Close()

	result := make([]StaffCoverAssignmentRecord, 0)
	for rows.Next() {
		var r StaffCoverAssignmentRecord
		if err := rows.Scan(
			&r.ID, &r.StaffAbsenceID, &r.TimetableSlotID,
			&r.Date, &r.AbsentStaffID, &r.CoveringStaffID,
			&r.Status, &r.CoverNotes,
		); err != nil {
			return nil, fmt.Errorf("failed to scan staff_cover_assignments row: %w", err)
		}
		r.CoverID = r.ID
		r.AbsenceID = r.StaffAbsenceID
		r.Notes = r.CoverNotes
		result = append(result, r)
	}
	return result, rows.Err()
}

func (d *CoreDAL) UpsertStaffCoverAssignment(
	ctx context.Context,
	in StaffCoverAssignmentRecord,
) (*StaffCoverAssignmentRecord, error) {
	date := strings.TrimSpace(in.Date)
	absentStaffID := strings.TrimSpace(in.AbsentStaffID)
	if date == "" || absentStaffID == "" {
		return nil, errors.New("missing mandatory parameter: date and absent_staff_id are required")
	}
	coverID := strings.TrimSpace(in.CoverID)
	if coverID == "" {
		coverID = strings.TrimSpace(in.ID)
	}
	if coverID == "" {
		coverID = "cov_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	}
	rawStatus := strings.ToUpper(strings.TrimSpace(in.Status))
	dbStatus := rawStatus
	switch dbStatus {
	case "ASSIGNED", "CONFIRMED":
		dbStatus = "ACCEPTED"
	case "":
		dbStatus = "REQUESTED"
	}
	switch dbStatus {
	case "REQUESTED", "ACCEPTED", "DECLINED", "CANCELLED":
	default:
		return nil, fmt.Errorf("invalid cover assignment status %q", in.Status)
	}
	absID := strings.TrimSpace(in.StaffAbsenceID)
	if absID == "" {
		absID = strings.TrimSpace(in.AbsenceID)
	}
	notes := strings.TrimSpace(in.CoverNotes)
	if notes == "" {
		notes = strings.TrimSpace(in.Notes)
	}
	optStr := func(s string) any {
		if strings.TrimSpace(s) == "" {
			return nil
		}
		return strings.TrimSpace(s)
	}

	if _, err := d.writeDB.ExecContext(
		ctx,
		`INSERT INTO staff_cover_assignments (
			id, staff_absence_id, timetable_slot_id, date,
			absent_staff_id, covering_staff_id, status, cover_notes
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			staff_absence_id = excluded.staff_absence_id,
			timetable_slot_id = excluded.timetable_slot_id,
			date = excluded.date,
			absent_staff_id = excluded.absent_staff_id,
			covering_staff_id = excluded.covering_staff_id,
			status = excluded.status,
			cover_notes = excluded.cover_notes`,
		coverID, optStr(absID), optStr(in.TimetableSlotID), date,
		absentStaffID, optStr(in.CoveringStaffID), dbStatus, notes,
	); err != nil {
		return nil, fmt.Errorf("failed to upsert staff_cover_assignments row: %w", err)
	}

	out := in
	out.ID = coverID
	out.CoverID = coverID
	out.StaffAbsenceID = absID
	out.AbsenceID = absID
	out.Date = date
	out.AbsentStaffID = absentStaffID
	if rawStatus != "" {
		out.Status = rawStatus
	} else {
		out.Status = dbStatus
	}
	out.CoverNotes = notes
	out.Notes = notes
	return &out, nil
}


