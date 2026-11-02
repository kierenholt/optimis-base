package dal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// ============================================================================
// SQL Projections & Shared Field Constants
// ============================================================================

const staffSCRFieldsSQL = `
	s.id, s.person_id, COALESCE(s.staff_code, ''), COALESCE(s.trn, ''),
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
`

const staffContractFieldsSQL = `
	id, staff_id, contract_type, swc_post_code, job_title,
	COALESCE(pay_scale, ''), COALESCE(pay_spine_point, ''),
	fte, COALESCE(hours_per_week, 0), COALESCE(weeks_per_year, 0),
	effective_from, COALESCE(effective_to, '')
`

const staffAbsenceFieldsSQL = `
	id, staff_id, academic_year_id, absence_category_code, reason_detail,
	status, cover_required, start_date, end_date, working_days_lost, COALESCE(approved_by_staff_id, '')
`

const staffCoverAssignmentFieldsSQL = `
	id, COALESCE(staff_absence_id, ''), COALESCE(timetable_slot_id, ''),
	date, absent_staff_id, COALESCE(covering_staff_id, ''),
	status, COALESCE(cover_notes, '')
`

// ============================================================================
// Internal Helpers & Row Scanners
// ============================================================================

func scanStaffSCRRecord(s rowScanner) (StaffSCRRecord, error) {
	var r StaffSCRRecord
	var title, prefFirst, prefLast, legFirst, legLast string
	var qtsInt, eytsInt, hltaInt, dslInt, faInt int
	if err := s.Scan(
		&r.StaffID, &r.PersonID, &r.StaffCode, &r.TRN,
		&title, &prefFirst, &prefLast, &legFirst, &legLast,
		&r.StaffType, &r.JobTitle,
		&qtsInt, &eytsInt, &hltaInt, &dslInt,
		&faInt, &r.FirstAidExpiryDate,
		&r.DBSCheckDate, &r.DBSCertificateNumber,
		&r.ProhibitionCheckDate, &r.RightToWorkCheckDate,
		&r.HireDate, &r.TerminationDate,
	); err != nil {
		return StaffSCRRecord{}, err
	}
	r.FirstName = prefFirst
	r.LastName = prefLast
	r.DisplayName = formatPersonaDisplayName(title, prefFirst, legFirst, prefLast, legLast)
	r.QTSFlag = qtsInt == 1
	r.EYTSFlag = eytsInt == 1
	r.HLTAFlag = hltaInt == 1
	r.IsDSL = dslInt == 1
	r.FirstAidQualified = faInt == 1
	return r, nil
}

func scanStaffContract(s rowScanner) (StaffContractRecord, error) {
	var r StaffContractRecord
	if err := s.Scan(
		&r.ID, &r.StaffID, &r.ContractType, &r.SWCPostCode, &r.JobTitle,
		&r.PayScale, &r.PaySpinePoint,
		&r.FTE, &r.HoursPerWeek, &r.WeeksPerYear,
		&r.EffectiveFrom, &r.EffectiveTo,
	); err != nil {
		return StaffContractRecord{}, err
	}
	r.ContractID = r.ID
	return r, nil
}

func scanStaffAbsence(s rowScanner) (StaffAbsenceRecord, error) {
	var r StaffAbsenceRecord
	var covInt int
	if err := s.Scan(
		&r.ID, &r.StaffID, &r.AcademicYearID, &r.AbsenceCategoryCode, &r.ReasonDetail,
		&r.Status, &covInt, &r.StartDate, &r.EndDate, &r.WorkingDaysLost, &r.ApprovedByStaffID,
	); err != nil {
		return StaffAbsenceRecord{}, err
	}
	r.AbsenceID = r.ID
	r.CoverRequired = covInt == 1
	return r, nil
}

func scanStaffCoverAssignment(s rowScanner) (StaffCoverAssignmentRecord, error) {
	var r StaffCoverAssignmentRecord
	if err := s.Scan(
		&r.ID, &r.StaffAbsenceID, &r.TimetableSlotID,
		&r.Date, &r.AbsentStaffID, &r.CoveringStaffID,
		&r.Status, &r.CoverNotes,
	); err != nil {
		return StaffCoverAssignmentRecord{}, err
	}
	r.CoverID = r.ID
	r.AbsenceID = r.StaffAbsenceID
	r.Notes = r.CoverNotes
	return r, nil
}

// ============================================================================
// Section 1: Staff SCR Register (Single Central Record)
// ============================================================================

type StaffSCRRecord struct {
	StaffID              string `json:"staff_id"`
	PersonID             string `json:"person_id"`
	StaffCode            string `json:"staff_code"`
	TRN                  string `json:"trn,omitempty"`
	DisplayName          string `json:"display_name"`
	FirstName            string `json:"first_name"`
	LastName             string `json:"last_name"`
	StaffType            string `json:"staff_type"`
	JobTitle             string `json:"job_title"`
	QTSFlag              bool   `json:"qts_flag"`
	EYTSFlag             bool   `json:"eyts_flag"`
	HLTAFlag             bool   `json:"hlta_flag"`
	IsDSL                bool   `json:"is_dsl"`
	FirstAidQualified    bool   `json:"first_aid_qualified"`
	FirstAidExpiryDate   string `json:"first_aid_expiry_date,omitempty"`
	DBSCheckDate         string `json:"dbs_check_date,omitempty"`
	DBSCertificateNumber string `json:"dbs_certificate_number,omitempty"`
	ProhibitionCheckDate string `json:"prohibition_check_date,omitempty"`
	RightToWorkCheckDate string `json:"right_to_work_check_date,omitempty"`
	HireDate             string `json:"hire_date"`
	TerminationDate      string `json:"termination_date,omitempty"`
}

func (d *CoreDAL) ListStaffSCRRegister(ctx context.Context) ([]StaffSCRRecord, error) {
	rows, err := d.readDB.QueryContext(
		ctx,
		`SELECT `+staffSCRFieldsSQL+`
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
		r, err := scanStaffSCRRecord(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan staff SCR row: %w", err)
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

func (d *CoreDAL) getStaffSCRRecordByID(ctx context.Context, staffID string) (*StaffSCRRecord, error) {
	row := d.readDB.QueryRowContext(
		ctx,
		`SELECT `+staffSCRFieldsSQL+`
		 FROM staff s
		 JOIN persons per ON per.id = s.person_id
		 WHERE s.id = ?`,
		staffID,
	)
	r, err := scanStaffSCRRecord(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("staff not found after SCR update: %s", staffID)
		}
		return nil, fmt.Errorf("failed to query staff SCR record %s: %w", staffID, err)
	}
	return &r, nil
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
		optTrimmedString(in.TRN),
		optTrimmedString(in.DBSCheckDate),
		optTrimmedString(in.DBSCertificateNumber),
		optTrimmedString(in.ProhibitionCheckDate),
		optTrimmedString(in.RightToWorkCheckDate),
		boolPtrToAny(in.FirstAidQualified),
		optTrimmedString(in.FirstAidExpiryDate),
		boolPtrToAny(in.QTSFlag),
		boolPtrToAny(in.HLTAFlag),
		staffID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update staff SCR record: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, fmt.Errorf("staff not found: %s", staffID)
	}

	return d.getStaffSCRRecordByID(ctx, staffID)
}

// ============================================================================
// Section 2: Staff Contracts
// ============================================================================

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
		SELECT ` + staffContractFieldsSQL + `
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
		r, err := scanStaffContract(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan staff_contracts row: %w", err)
		}
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
		fte, in.HoursPerWeek, in.WeeksPerYear, effFrom, optTrimmedString(in.EffectiveTo),
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

// ============================================================================
// Section 3: Staff Absences
// ============================================================================

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
		SELECT ` + staffAbsenceFieldsSQL + `
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
		r, err := scanStaffAbsence(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan staff_absences row: %w", err)
		}
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
	daysLost := in.WorkingDaysLost
	if daysLost <= 0 {
		daysLost = 1.0
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
		status, boolToInt(in.CoverRequired), startDate, endDate, daysLost, optTrimmedString(in.ApprovedByStaffID),
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

// ============================================================================
// Section 4: Staff Cover Assignments
// ============================================================================

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
		SELECT ` + staffCoverAssignmentFieldsSQL + `
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
		r, err := scanStaffCoverAssignment(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan staff_cover_assignments row: %w", err)
		}
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
		coverID, optTrimmedString(absID), optTrimmedString(in.TimetableSlotID), date,
		absentStaffID, optTrimmedString(in.CoveringStaffID), dbStatus, notes,
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
