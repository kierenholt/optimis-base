package dal

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type AttendanceCodeSummary struct {
	Code                string `json:"code"`
	Name                string `json:"name"`
	Description         string `json:"description"`
	StatisticalCategory string `json:"statistical_category"`
	StatisticalMeaning  string `json:"statistical_meaning"`
	PhysicalStatus      string `json:"physical_status"`
	IsStatutoryCode     bool   `json:"is_statutory_code"`
	IsActive            bool   `json:"is_active"`
	RestrictedToAdmin   bool   `json:"restricted_to_admin"`
	SequenceNumber      int    `json:"sequence_number"`
}

func (d *CoreDAL) ListAttendanceCodes(ctx context.Context) ([]AttendanceCodeSummary, error) {
	rows, err := d.readDB.QueryContext(
		ctx,
		`SELECT code, name, statistical_category, physical_status, is_statutory_code, is_active, restricted_to_admin, sequence_number
		 FROM attendance_codes
		 ORDER BY sequence_number ASC, code ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list attendance_codes: %w", err)
	}
	defer rows.Close()

	result := make([]AttendanceCodeSummary, 0)
	for rows.Next() {
		var r AttendanceCodeSummary
		var statInt, actInt, restInt int
		if err := rows.Scan(
			&r.Code, &r.Name, &r.StatisticalCategory, &r.PhysicalStatus,
			&statInt, &actInt, &restInt, &r.SequenceNumber,
		); err != nil {
			return nil, fmt.Errorf("failed to scan attendance_codes row: %w", err)
		}
		r.Description = r.Name
		r.StatisticalMeaning = r.StatisticalCategory
		r.IsStatutoryCode = statInt == 1
		r.IsActive = actInt == 1
		r.RestrictedToAdmin = restInt == 1
		result = append(result, r)
	}
	return result, rows.Err()
}

func (d *CoreDAL) UpsertAttendanceCodeScoped(
	ctx context.Context,
	scope DALQueryScope,
	in AttendanceCodeSummary,
) (*AttendanceCodeSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	if scope.Mode != ScopeAll {
		return nil, errors.New("access denied: upserting attendance codes requires dal.attendance.write:all scope")
	}

	code := strings.TrimSpace(in.Code)
	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = strings.TrimSpace(in.Description)
	}
	statCat := strings.ToUpper(strings.TrimSpace(in.StatisticalCategory))
	if statCat == "" {
		statCat = strings.ToUpper(strings.TrimSpace(in.StatisticalMeaning))
	}
	if code == "" || name == "" || statCat == "" {
		return nil, errors.New("missing mandatory parameter: code, name/description, and statistical_category/statistical_meaning are required")
	}
	switch statCat {
	case "PRESENT", "APPROVED_EDUCATIONAL_ACTIVITY", "AUTHORISED_ABSENCE", "UNAUTHORISED_ABSENCE", "UNABLE_TO_ATTEND_EXCEPTIONAL", "NOT_REQUIRED_TO_ATTEND":
	default:
		return nil, fmt.Errorf("invalid statistical_category %q", statCat)
	}
	physStatus := strings.ToUpper(strings.TrimSpace(in.PhysicalStatus))
	if physStatus == "" {
		switch statCat {
		case "PRESENT":
			physStatus = "ONSITE"
		case "APPROVED_EDUCATIONAL_ACTIVITY":
			physStatus = "OFFSITE_EDUCATIONAL"
		case "NOT_REQUIRED_TO_ATTEND", "UNABLE_TO_ATTEND_EXCEPTIONAL":
			physStatus = "CLOSURE"
		default:
			physStatus = "ABSENT"
		}
	}
	seq := in.SequenceNumber
	if seq <= 0 {
		_ = d.readDB.QueryRowContext(ctx, `SELECT COALESCE(MAX(sequence_number), 0) + 1 FROM attendance_codes`).Scan(&seq)
		if seq <= 0 {
			seq = 1
		}
	}
	statInt := 1
	actInt := 1
	if !in.IsActive && in.SequenceNumber < 0 {
		actInt = 0
	}
	restInt := 0
	if in.RestrictedToAdmin {
		restInt = 1
	}

	if _, err := d.writeDB.ExecContext(
		ctx,
		`INSERT INTO attendance_codes (
			code, name, statistical_category, physical_status,
			is_statutory_code, is_active, restricted_to_admin, sequence_number
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(code) DO UPDATE SET
			name = excluded.name,
			statistical_category = excluded.statistical_category,
			physical_status = excluded.physical_status,
			is_active = excluded.is_active,
			restricted_to_admin = excluded.restricted_to_admin,
			sequence_number = excluded.sequence_number`,
		code, name, statCat, physStatus, statInt, actInt, restInt, seq,
	); err != nil {
		return nil, fmt.Errorf("failed to upsert attendance_codes row: %w", err)
	}

	return &AttendanceCodeSummary{
		Code:                code,
		Name:                name,
		Description:         name,
		StatisticalCategory: statCat,
		StatisticalMeaning:  statCat,
		PhysicalStatus:      physStatus,
		IsStatutoryCode:     statInt == 1,
		IsActive:            actInt == 1,
		RestrictedToAdmin:   restInt == 1,
		SequenceNumber:      seq,
	}, nil
}
