package dal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"hash/fnv"
	"regexp"
	"strings"
)

var validStatutoryUPNRegex = regexp.MustCompile(`^[A-Z][0-9]{12}$`)

type StatutoryCensusRollupInput struct {
	AcademicYearID      string    `json:"academic_year_id"`
	CensusDate          string    `json:"census_date"`
	TermCode            string    `json:"term_code"`
	AttendanceStartDate string    `json:"attendance_start_date"`
	AttendanceEndDate   string    `json:"attendance_end_date"`
	Scope               ScopeMode `json:"scope"`
}

type CensusFSMPeriod struct {
	ID            string `json:"id"`
	EffectiveFrom string `json:"effective_from"`
	EffectiveTo   string `json:"effective_to"`
	CountryCode   string `json:"country_code"`
}

type CensusAttendanceSessions struct {
	PossibleSessions            int `json:"possible_sessions"`
	PresentSessions             int `json:"present_sessions"`
	AuthorisedAbsenceSessions   int `json:"authorised_absence_sessions"`
	UnauthorisedAbsenceSessions int `json:"unauthorised_absence_sessions"`
	AMPresent                   int `json:"am_present"`
	PMPresent                   int `json:"pm_present"`
}

type CensusPupilRecord struct {
	PupilID             string                   `json:"pupil_id"`
	UPN                 string                   `json:"upn"`
	ULN                 string                   `json:"uln"`
	FirstName           string                   `json:"first_name"`
	LastName            string                   `json:"last_name"`
	DOB                 string                   `json:"dob"`
	Sex                 string                   `json:"sex"`
	EthnicityCode       string                   `json:"ethnicity_code"`
	FirstLanguageCode   string                   `json:"first_language_code"`
	NCYearGroup         string                   `json:"nc_year_group"`
	EnrolmentStatus     string                   `json:"enrolment_status"`
	EffectiveFrom       string                   `json:"effective_from"`
	EffectiveTo         string                   `json:"effective_to"`
	SENProvisionCode    string                   `json:"sen_provision_code"`
	PrimarySENNeedCode  string                   `json:"primary_sen_need_code"`
	FSMEligible         bool                     `json:"fsm_eligible"`
	FSMPeriods          []CensusFSMPeriod        `json:"fsm_periods"`
	LookedAfterFlag     bool                     `json:"looked_after_flag"`
	PostLookedAfterCode string                   `json:"post_looked_after_code"`
	LACEpisodeStatus    string                   `json:"lac_episode_status"`
	AttendanceSessions  CensusAttendanceSessions `json:"attendance_sessions"`
}

type CensusRollupSummary struct {
	OnRollTotal                      int            `json:"on_roll_total"`
	SENCounts                        map[string]int `json:"sen_counts"`
	FSMEligibleCount                 int            `json:"fsm_eligible_count"`
	LACCount                         int            `json:"lac_count"`
	TotalPossibleSessions            int            `json:"total_possible_sessions"`
	TotalPresentSessions             int            `json:"total_present_sessions"`
	TotalAuthorisedAbsenceSessions   int            `json:"total_authorised_absence_sessions"`
	TotalUnauthorisedAbsenceSessions int            `json:"total_unauthorised_absence_sessions"`
}

type StatutoryCensusRollup struct {
	SchoolURN      string              `json:"school_urn"`
	SchoolName     string              `json:"school_name"`
	AcademicYearID string              `json:"academic_year_id"`
	TermCode       string              `json:"term_code"`
	CensusDate     string              `json:"census_date"`
	Summary        CensusRollupSummary `json:"summary"`
	Pupils         []CensusPupilRecord `json:"pupils"`
}

func normalizeStatutoryUPN(rawUPN string, pupilID string) string {
	trimmed := strings.ToUpper(strings.TrimSpace(rawUPN))
	if validStatutoryUPNRegex.MatchString(trimmed) {
		return trimmed
	}
	h := fnv.New64a()
	_, _ = h.Write([]byte(trimmed + "|" + pupilID))
	val := h.Sum64() % 1000000000000
	return fmt.Sprintf("H%012d", val)
}

func (d *CoreDAL) GetStatutoryCensusRollup(ctx context.Context, in StatutoryCensusRollupInput) (*StatutoryCensusRollup, error) {
	if in.Scope != ScopeAll {
		return nil, errors.New("statutory school census rollup requires dal.census.read:all (ScopeAll)")
	}

	ayID := strings.TrimSpace(in.AcademicYearID)
	if ayID == "" {
		return nil, errors.New("missing mandatory parameter: academic_year_id")
	}
	censusDate := strings.TrimSpace(in.CensusDate)
	if censusDate == "" {
		return nil, errors.New("missing mandatory parameter: census_date")
	}
	termCode := strings.ToUpper(strings.TrimSpace(in.TermCode))
	if termCode != "AUTUMN" && termCode != "SPRING" && termCode != "SUMMER" {
		return nil, fmt.Errorf("invalid statutory census term_code %q: must be AUTUMN, SPRING, or SUMMER", in.TermCode)
	}

	attStart := strings.TrimSpace(in.AttendanceStartDate)
	if attStart == "" {
		attStart = "2026-09-01"
	}
	attEnd := strings.TrimSpace(in.AttendanceEndDate)
	if attEnd == "" {
		attEnd = censusDate
	}

	schoolURN, err := d.GetSystemConfigValue(ctx, "school_urn")
	if err != nil {
		return nil, err
	}
	schoolName, err := d.GetSystemConfigValue(ctx, "school_name")
	if err != nil || strings.TrimSpace(schoolName) == "" {
		schoolName = "Oakridge All-Through Academy"
	}

	rows, err := d.readDB.QueryContext(
		ctx,
		`SELECT p.id, p.upn, COALESCE(p.uln, ''), per.legal_first_name, per.legal_last_name,
		        COALESCE(NULLIF(TRIM(per.dob), ''), '2012-01-01'),
		        COALESCE(NULLIF(TRIM(per.sex), ''), 'U'),
		        COALESCE(NULLIF(TRIM(p.ethnicity_code), ''), 'NOBT'),
		        COALESCE(NULLIF(TRIM(p.first_language_code), ''), 'ENG'),
		        enr.nc_year_group, enr.enrolment_status, enr.effective_from, COALESCE(enr.effective_to, ''),
		        p.looked_after_flag, COALESCE(NULLIF(TRIM(p.post_looked_after_code), ''), 'N')
		 FROM pupil_year_enrolments enr
		 JOIN pupils p ON p.id = enr.pupil_id
		 JOIN persons per ON per.id = p.person_id
		 WHERE enr.academic_year_id = ?
		   AND enr.effective_from <= ?
		   AND (enr.effective_to IS NULL OR TRIM(enr.effective_to) = '' OR enr.effective_to >= ?)
		 ORDER BY per.legal_last_name ASC, per.legal_first_name ASC, p.id ASC`,
		ayID, censusDate, censusDate,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query on-roll pupils for statutory census: %w", err)
	}

	var basePupils []CensusPupilRecord
	for rows.Next() {
		var rec CensusPupilRecord
		var lacInt int
		if scanErr := rows.Scan(
			&rec.PupilID, &rec.UPN, &rec.ULN, &rec.FirstName, &rec.LastName,
			&rec.DOB, &rec.Sex, &rec.EthnicityCode, &rec.FirstLanguageCode,
			&rec.NCYearGroup, &rec.EnrolmentStatus, &rec.EffectiveFrom, &rec.EffectiveTo,
			&lacInt, &rec.PostLookedAfterCode,
		); scanErr != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("failed to scan census pupil row: %w", scanErr)
		}
		rec.UPN = normalizeStatutoryUPN(rec.UPN, rec.PupilID)
		rec.LookedAfterFlag = lacInt == 1
		if rec.LookedAfterFlag {
			rec.LACEpisodeStatus = "CURRENT_LAC"
		} else if rec.PostLookedAfterCode != "" && rec.PostLookedAfterCode != "N" {
			rec.LACEpisodeStatus = "POST_LAC"
		} else {
			rec.LACEpisodeStatus = "NONE"
		}
		basePupils = append(basePupils, rec)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	_ = rows.Close()

	summary := CensusRollupSummary{
		SENCounts: map[string]int{
			"N": 0,
			"K": 0,
			"E": 0,
		},
	}

	pupils := make([]CensusPupilRecord, 0, len(basePupils))
	for _, rec := range basePupils {
		// 1. Active SEN Provision as of census_date
		var senProv string
		err := d.readDB.QueryRowContext(
			ctx,
			`SELECT provision_code
			 FROM pupil_sen_records
			 WHERE pupil_id = ?
			   AND effective_from <= ?
			   AND (effective_to IS NULL OR TRIM(effective_to) = '' OR effective_to >= ?)
			 ORDER BY effective_from DESC, id ASC
			 LIMIT 1`,
			rec.PupilID, censusDate, censusDate,
		).Scan(&senProv)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				senProv = "N"
			} else {
				return nil, fmt.Errorf("failed to query SEN provision for pupil %s: %w", rec.PupilID, err)
			}
		}
		rec.SENProvisionCode = strings.TrimSpace(senProv)
		if rec.SENProvisionCode == "" {
			rec.SENProvisionCode = "N"
		}

		// 2. Primary SEN Need (ranking = 1) as of census_date
		var primaryNeed string
		err = d.readDB.QueryRowContext(
			ctx,
			`SELECT sen_need_code
			 FROM pupil_sen_needs
			 WHERE pupil_id = ?
			   AND ranking = 1
			   AND effective_from <= ?
			   AND (effective_to IS NULL OR TRIM(effective_to) = '' OR effective_to >= ?)
			 ORDER BY effective_from DESC, id ASC
			 LIMIT 1`,
			rec.PupilID, censusDate, censusDate,
		).Scan(&primaryNeed)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("failed to query primary SEN need for pupil %s: %w", rec.PupilID, err)
		}
		rec.PrimarySENNeedCode = strings.TrimSpace(primaryNeed)

		// 3. Active FSM Periods as of census_date
		fsmRows, err := d.readDB.QueryContext(
			ctx,
			`SELECT id, effective_from, COALESCE(effective_to, ''), COALESCE(country_code, 'ENG')
			 FROM pupil_fsm_periods
			 WHERE pupil_id = ?
			   AND effective_from <= ?
			   AND (effective_to IS NULL OR TRIM(effective_to) = '' OR effective_to >= ?)
			 ORDER BY effective_from ASC, id ASC`,
			rec.PupilID, censusDate, censusDate,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to query FSM periods for pupil %s: %w", rec.PupilID, err)
		}
		fsmPeriods := make([]CensusFSMPeriod, 0)
		for fsmRows.Next() {
			var fp CensusFSMPeriod
			if scanErr := fsmRows.Scan(&fp.ID, &fp.EffectiveFrom, &fp.EffectiveTo, &fp.CountryCode); scanErr != nil {
				_ = fsmRows.Close()
				return nil, scanErr
			}
			fsmPeriods = append(fsmPeriods, fp)
		}
		_ = fsmRows.Close()
		rec.FSMPeriods = fsmPeriods
		rec.FSMEligible = len(fsmPeriods) > 0

		// 4. Statutory AM/PM Attendance Session Rollup
		attRows, err := d.readDB.QueryContext(
			ctx,
			`SELECT asm.session, ac.statistical_category
			 FROM attendance_session_marks asm
			 JOIN attendance_codes ac ON ac.code = asm.attendance_code
			 WHERE asm.pupil_id = ?
			   AND asm.academic_year_id = ?
			   AND asm.date >= ?
			   AND asm.date <= ?
			 ORDER BY asm.date ASC, asm.session ASC`,
			rec.PupilID, ayID, attStart, attEnd,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to query attendance sessions for pupil %s: %w", rec.PupilID, err)
		}
		var sessions CensusAttendanceSessions
		for attRows.Next() {
			var sess, statCat string
			if scanErr := attRows.Scan(&sess, &statCat); scanErr != nil {
				_ = attRows.Close()
				return nil, scanErr
			}
			switch statCat {
			case "PRESENT", "APPROVED_EDUCATIONAL_ACTIVITY":
				sessions.PossibleSessions++
				sessions.PresentSessions++
				if sess == "AM" {
					sessions.AMPresent++
				} else if sess == "PM" {
					sessions.PMPresent++
				}
			case "AUTHORISED_ABSENCE":
				sessions.PossibleSessions++
				sessions.AuthorisedAbsenceSessions++
			case "UNAUTHORISED_ABSENCE":
				sessions.PossibleSessions++
				sessions.UnauthorisedAbsenceSessions++
			}
		}
		_ = attRows.Close()
		rec.AttendanceSessions = sessions

		summary.OnRollTotal++
		summary.SENCounts[rec.SENProvisionCode]++
		if rec.FSMEligible {
			summary.FSMEligibleCount++
		}
		if rec.LookedAfterFlag {
			summary.LACCount++
		}
		summary.TotalPossibleSessions += sessions.PossibleSessions
		summary.TotalPresentSessions += sessions.PresentSessions
		summary.TotalAuthorisedAbsenceSessions += sessions.AuthorisedAbsenceSessions
		summary.TotalUnauthorisedAbsenceSessions += sessions.UnauthorisedAbsenceSessions

		pupils = append(pupils, rec)
	}

	return &StatutoryCensusRollup{
		SchoolURN:      schoolURN,
		SchoolName:     schoolName,
		AcademicYearID: ayID,
		TermCode:       termCode,
		CensusDate:     censusDate,
		Summary:        summary,
		Pupils:         pupils,
	}, nil
}
