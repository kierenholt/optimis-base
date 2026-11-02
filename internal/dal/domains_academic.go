package dal

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type AcademicYearSummary struct {
	ID             string `json:"id"`
	AcademicYearID string `json:"academic_year_id"`
	Name           string `json:"name"`
	Status         string `json:"status"`
	SequenceNumber int    `json:"sequence_number"`
	FirstDay       string `json:"first_day"`
	LastDay        string `json:"last_day"`
}

func (d *CoreDAL) ListAcademicYears(ctx context.Context) ([]AcademicYearSummary, error) {
	rows, err := d.readDB.QueryContext(
		ctx,
		`SELECT id, name, status, sequence_number, first_day, last_day
		 FROM academic_years
		 ORDER BY sequence_number ASC, first_day ASC, id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list academic_years: %w", err)
	}
	defer rows.Close()

	result := make([]AcademicYearSummary, 0)
	for rows.Next() {
		var ay AcademicYearSummary
		if err := rows.Scan(&ay.ID, &ay.Name, &ay.Status, &ay.SequenceNumber, &ay.FirstDay, &ay.LastDay); err != nil {
			return nil, fmt.Errorf("failed to scan academic_years row: %w", err)
		}
		ay.AcademicYearID = ay.ID
		result = append(result, ay)
	}
	return result, rows.Err()
}

func (d *CoreDAL) UpsertAcademicYear(ctx context.Context, in AcademicYearSummary) (*AcademicYearSummary, error) {
	ayID := strings.TrimSpace(in.AcademicYearID)
	if ayID == "" {
		ayID = strings.TrimSpace(in.ID)
	}
	name := strings.TrimSpace(in.Name)
	firstDay := strings.TrimSpace(in.FirstDay)
	lastDay := strings.TrimSpace(in.LastDay)
	if name == "" || firstDay == "" || lastDay == "" {
		return nil, errors.New("missing mandatory parameter: name, first_day, and last_day are required")
	}
	if ayID == "" {
		ayID = "ay_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:8]
	}
	status := strings.ToUpper(strings.TrimSpace(in.Status))
	if status == "" || status == "FUTURE" {
		status = "UPCOMING"
	}
	switch status {
	case "PAST", "CURRENT", "UPCOMING":
	default:
		return nil, fmt.Errorf("invalid academic_years status %q: must be PAST, CURRENT, or UPCOMING", in.Status)
	}

	tx, err := d.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin upsert academic_year tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	seqNum := in.SequenceNumber
	if seqNum <= 0 {
		_ = tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(sequence_number), 0) + 1 FROM academic_years`).Scan(&seqNum)
		if seqNum <= 0 {
			seqNum = 1
		}
	}

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO academic_years (id, name, status, sequence_number, first_day, last_day)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		     name = excluded.name,
		     status = excluded.status,
		     sequence_number = excluded.sequence_number,
		     first_day = excluded.first_day,
		     last_day = excluded.last_day`,
		ayID, name, status, seqNum, firstDay, lastDay,
	); err != nil {
		return nil, fmt.Errorf("failed to upsert academic_years row: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit upsert academic_year tx: %w", err)
	}

	return &AcademicYearSummary{
		ID:             ayID,
		AcademicYearID: ayID,
		Name:           name,
		Status:         status,
		SequenceNumber: seqNum,
		FirstDay:       firstDay,
		LastDay:        lastDay,
	}, nil
}

type AcademicTermSummary struct {
	ID             string `json:"id"`
	TermID         string `json:"term_id"`
	AcademicYearID string `json:"academic_year_id"`
	Code           string `json:"code"`
	TermCode       string `json:"term_code"`
	Name           string `json:"name"`
	NameShort      string `json:"name_short"`
	SequenceNumber int    `json:"sequence_number"`
	FirstDay       string `json:"first_day"`
	LastDay        string `json:"last_day"`
}

func (d *CoreDAL) ListAcademicTerms(ctx context.Context, academicYearID string) ([]AcademicTermSummary, error) {
	ayID := strings.TrimSpace(academicYearID)
	query := `
		SELECT id, academic_year_id, code, name, name_short, sequence_number, first_day, last_day
		FROM academic_terms`
	var args []any
	if ayID != "" {
		query += ` WHERE academic_year_id = ?`
		args = append(args, ayID)
	}
	query += ` ORDER BY academic_year_id ASC, sequence_number ASC, id ASC`

	rows, err := d.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list academic_terms: %w", err)
	}
	defer rows.Close()

	result := make([]AcademicTermSummary, 0)
	for rows.Next() {
		var t AcademicTermSummary
		if err := rows.Scan(
			&t.ID, &t.AcademicYearID, &t.Code, &t.Name, &t.NameShort, &t.SequenceNumber,
			&t.FirstDay, &t.LastDay,
		); err != nil {
			return nil, fmt.Errorf("failed to scan academic_terms row: %w", err)
		}
		t.TermID = t.ID
		t.TermCode = t.Code
		result = append(result, t)
	}
	return result, rows.Err()
}

func (d *CoreDAL) UpsertAcademicTerm(ctx context.Context, in AcademicTermSummary) (*AcademicTermSummary, error) {
	termID := strings.TrimSpace(in.TermID)
	if termID == "" {
		termID = strings.TrimSpace(in.ID)
	}
	ayID := strings.TrimSpace(in.AcademicYearID)
	code := strings.ToUpper(strings.TrimSpace(in.Code))
	if code == "" {
		code = strings.ToUpper(strings.TrimSpace(in.TermCode))
	}
	switch code {
	case "AUT":
		code = "AUTUMN"
	case "SPR":
		code = "SPRING"
	case "SUM":
		code = "SUMMER"
	}
	name := strings.TrimSpace(in.Name)
	nameShort := strings.TrimSpace(in.NameShort)
	if nameShort == "" && len(code) >= 3 {
		nameShort = code[:3]
	}
	firstDay := strings.TrimSpace(in.FirstDay)
	lastDay := strings.TrimSpace(in.LastDay)
	if ayID == "" || code == "" || name == "" || firstDay == "" || lastDay == "" {
		return nil, errors.New("missing mandatory parameter: academic_year_id, code/term_code, name, first_day, and last_day are required")
	}
	if termID == "" {
		termID = "trm_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	}
	seqNum := in.SequenceNumber
	if seqNum <= 0 {
		seqNum = 1
	}

	if _, err := d.writeDB.ExecContext(
		ctx,
		`INSERT INTO academic_terms (
			id, academic_year_id, code, name, name_short, sequence_number, first_day, last_day
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(academic_year_id, sequence_number) DO UPDATE SET
			code = excluded.code,
			name = excluded.name,
			name_short = excluded.name_short,
			first_day = excluded.first_day,
			last_day = excluded.last_day`,
		termID, ayID, code, name, nameShort, seqNum, firstDay, lastDay,
	); err != nil {
		return nil, fmt.Errorf("failed to upsert academic_terms row: %w", err)
	}

	var persistedID string
	_ = d.readDB.QueryRowContext(
		ctx,
		`SELECT id FROM academic_terms WHERE academic_year_id = ? AND sequence_number = ?`,
		ayID, seqNum,
	).Scan(&persistedID)
	if strings.TrimSpace(persistedID) != "" {
		termID = persistedID
	}

	return &AcademicTermSummary{
		ID:             termID,
		TermID:         termID,
		AcademicYearID: ayID,
		Code:           code,
		TermCode:       code,
		Name:           name,
		NameShort:      nameShort,
		SequenceNumber: seqNum,
		FirstDay:       firstDay,
		LastDay:        lastDay,
	}, nil
}

type DayOfWeekSummary struct {
	ID              int    `json:"id"`
	Name            string `json:"name"`
	NameShort       string `json:"name_short"`
	IsSchoolDay     bool   `json:"is_school_day"`
	SchoolOpenTime  string `json:"school_open_time,omitempty"`
	AMRegisterTime  string `json:"am_register_time,omitempty"`
	PMRegisterTime  string `json:"pm_register_time,omitempty"`
	SchoolCloseTime string `json:"school_close_time,omitempty"`
}

func (d *CoreDAL) ListDaysOfWeek(ctx context.Context) ([]DayOfWeekSummary, error) {
	rows, err := d.readDB.QueryContext(
		ctx,
		`SELECT id, name, name_short, is_school_day,
		        COALESCE(school_open_time, ''), COALESCE(am_register_time, ''),
		        COALESCE(pm_register_time, ''), COALESCE(school_close_time, '')
		 FROM days_of_week
		 ORDER BY id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list days_of_week: %w", err)
	}
	defer rows.Close()

	result := make([]DayOfWeekSummary, 0)
	for rows.Next() {
		var day DayOfWeekSummary
		var isSchoolInt int
		if err := rows.Scan(
			&day.ID, &day.Name, &day.NameShort, &isSchoolInt,
			&day.SchoolOpenTime, &day.AMRegisterTime, &day.PMRegisterTime, &day.SchoolCloseTime,
		); err != nil {
			return nil, fmt.Errorf("failed to scan days_of_week row: %w", err)
		}
		day.IsSchoolDay = isSchoolInt == 1
		result = append(result, day)
	}
	return result, rows.Err()
}

func (d *CoreDAL) UpsertDayOfWeek(ctx context.Context, in DayOfWeekSummary) (*DayOfWeekSummary, error) {
	if in.ID < 1 || in.ID > 7 {
		return nil, fmt.Errorf("invalid day_of_week id %d: must be 1..7", in.ID)
	}
	name := strings.TrimSpace(in.Name)
	short := strings.TrimSpace(in.NameShort)
	if name == "" || short == "" {
		return nil, errors.New("missing mandatory parameter: name and name_short are required")
	}
	isSchoolInt := 0
	if in.IsSchoolDay {
		isSchoolInt = 1
	}

	var openArg, amArg, pmArg, closeArg any
	if strings.TrimSpace(in.SchoolOpenTime) != "" {
		openArg = strings.TrimSpace(in.SchoolOpenTime)
	}
	if strings.TrimSpace(in.AMRegisterTime) != "" {
		amArg = strings.TrimSpace(in.AMRegisterTime)
	}
	if strings.TrimSpace(in.PMRegisterTime) != "" {
		pmArg = strings.TrimSpace(in.PMRegisterTime)
	}
	if strings.TrimSpace(in.SchoolCloseTime) != "" {
		closeArg = strings.TrimSpace(in.SchoolCloseTime)
	}

	if _, err := d.writeDB.ExecContext(
		ctx,
		`INSERT INTO days_of_week (
			id, name, name_short, is_school_day,
			school_open_time, am_register_time, pm_register_time, school_close_time
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			name_short = excluded.name_short,
			is_school_day = excluded.is_school_day,
			school_open_time = excluded.school_open_time,
			am_register_time = excluded.am_register_time,
			pm_register_time = excluded.pm_register_time,
			school_close_time = excluded.school_close_time`,
		in.ID, name, short, isSchoolInt, openArg, amArg, pmArg, closeArg,
	); err != nil {
		return nil, fmt.Errorf("failed to upsert days_of_week row: %w", err)
	}

	out := in
	out.Name = name
	out.NameShort = short
	return &out, nil
}

type AcademicSpecialDaySummary struct {
	ID             string `json:"id"`
	SpecialDayID   string `json:"special_day_id"`
	AcademicTermID string `json:"academic_term_id"`
	AcademicYearID string `json:"academic_year_id,omitempty"`
	Date           string `json:"date"`
	Type           string `json:"type"`
	DayType        string `json:"day_type"`
	Name           string `json:"name"`
	Description    string `json:"description,omitempty"`
}

func (d *CoreDAL) ListAcademicSpecialDays(ctx context.Context, academicYearID string) ([]AcademicSpecialDaySummary, error) {
	ayID := strings.TrimSpace(academicYearID)
	query := `
		SELECT sd.id, sd.academic_term_id, COALESCE(t.academic_year_id, ''), sd.date, sd.type, sd.name, COALESCE(sd.description, '')
		FROM academic_special_days sd
		LEFT JOIN academic_terms t ON t.id = sd.academic_term_id`
	var args []any
	if ayID != "" {
		query += ` WHERE t.academic_year_id = ?`
		args = append(args, ayID)
	}
	query += ` ORDER BY sd.date ASC, sd.id ASC`

	rows, err := d.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list academic_special_days: %w", err)
	}
	defer rows.Close()

	result := make([]AcademicSpecialDaySummary, 0)
	for rows.Next() {
		var sd AcademicSpecialDaySummary
		if err := rows.Scan(
			&sd.ID, &sd.AcademicTermID, &sd.AcademicYearID, &sd.Date, &sd.Type,
			&sd.Name, &sd.Description,
		); err != nil {
			return nil, fmt.Errorf("failed to scan academic_special_days row: %w", err)
		}
		sd.SpecialDayID = sd.ID
		sd.DayType = sd.Type
		result = append(result, sd)
	}
	return result, rows.Err()
}

func (d *CoreDAL) UpsertAcademicSpecialDay(ctx context.Context, in AcademicSpecialDaySummary) (*AcademicSpecialDaySummary, error) {
	sdID := strings.TrimSpace(in.SpecialDayID)
	if sdID == "" {
		sdID = strings.TrimSpace(in.ID)
	}
	termID := strings.TrimSpace(in.AcademicTermID)
	ayID := strings.TrimSpace(in.AcademicYearID)
	date := strings.TrimSpace(in.Date)
	name := strings.TrimSpace(in.Name)
	if date == "" || name == "" || (termID == "" && ayID == "") {
		return nil, errors.New("missing mandatory parameter: (academic_term_id or academic_year_id), date, and name are required")
	}
	if termID == "" && ayID != "" {
		_ = d.readDB.QueryRowContext(
			ctx,
			`SELECT id FROM academic_terms WHERE academic_year_id = ? ORDER BY sequence_number ASC LIMIT 1`,
			ayID,
		).Scan(&termID)
		if termID == "" {
			createdTerm, err := d.UpsertAcademicTerm(ctx, AcademicTermSummary{
				AcademicYearID: ayID,
				Code:           "AUTUMN",
				Name:           "Autumn Term",
				NameShort:      "Aut",
				SequenceNumber: 1,
				FirstDay:       date,
				LastDay:        date,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to resolve academic_term_id for special day: %w", err)
			}
			termID = createdTerm.ID
		}
	}
	if sdID == "" {
		sdID = "asd_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	}
	dayType := strings.ToUpper(strings.TrimSpace(in.Type))
	if dayType == "" {
		dayType = strings.ToUpper(strings.TrimSpace(in.DayType))
	}
	switch dayType {
	case "", "INSET_STAFF_ONLY", "INSET_DAY":
		dayType = "INSET_DAY"
	case "BANK_HOLIDAY", "SCHOOL_HOLIDAY", "EMERGENCY_CLOSURE", "SCHOOL_CLOSURE":
		dayType = "SCHOOL_CLOSURE"
	case "TIMING_CHANGE", "OFF_TIMETABLE":
	default:
		return nil, fmt.Errorf("invalid special day type %q", in.DayType)
	}

	if _, err := d.writeDB.ExecContext(
		ctx,
		`INSERT INTO academic_special_days (
			id, academic_term_id, date, type, name, description
		) VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(date) DO UPDATE SET
			academic_term_id = excluded.academic_term_id,
			type = excluded.type,
			name = excluded.name,
			description = excluded.description`,
		sdID, termID, date, dayType, name, strings.TrimSpace(in.Description),
	); err != nil {
		return nil, fmt.Errorf("failed to upsert academic_special_days row: %w", err)
	}

	var persistedID string
	_ = d.readDB.QueryRowContext(
		ctx,
		`SELECT id FROM academic_special_days WHERE date = ?`,
		date,
	).Scan(&persistedID)
	if strings.TrimSpace(persistedID) != "" {
		sdID = persistedID
	}

	return &AcademicSpecialDaySummary{
		ID:             sdID,
		SpecialDayID:   sdID,
		AcademicTermID: termID,
		AcademicYearID: ayID,
		Date:           date,
		Type:           dayType,
		DayType:        dayType,
		Name:           name,
		Description:    strings.TrimSpace(in.Description),
	}, nil
}

type AlertLevelSummary struct {
	ID             string `json:"id"`
	Code           string `json:"code"`
	Name           string `json:"name"`
	NameShort      string `json:"name_short"`
	ColorHex       string `json:"color_hex"`
	ColourHex      string `json:"colour_hex"`
	BgColorHex     string `json:"bg_color_hex"`
	Description    string `json:"description"`
	SequenceNumber int    `json:"sequence_number"`
}

func (d *CoreDAL) ListAlertLevels(ctx context.Context) ([]AlertLevelSummary, error) {
	rows, err := d.readDB.QueryContext(
		ctx,
		`SELECT id, name, name_short, color_hex, bg_color_hex, description, sequence_number
		 FROM alert_levels
		 ORDER BY sequence_number ASC, id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list alert_levels: %w", err)
	}
	defer rows.Close()

	result := make([]AlertLevelSummary, 0)
	for rows.Next() {
		var al AlertLevelSummary
		if err := rows.Scan(&al.ID, &al.Name, &al.NameShort, &al.ColorHex, &al.BgColorHex, &al.Description, &al.SequenceNumber); err != nil {
			return nil, fmt.Errorf("failed to scan alert_levels row: %w", err)
		}
		al.Code = al.ID
		al.ColourHex = al.ColorHex
		result = append(result, al)
	}
	return result, rows.Err()
}
