package dal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"hash/fnv"
	"regexp"
	"strings"

	"github.com/google/uuid"
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

type CensusTestFixtureInput struct {
	AcademicYearID      string `json:"academic_year_id"`
	CensusDate          string `json:"census_date"`
	TermCode            string `json:"term_code"`
	AttendanceStartDate string `json:"attendance_start_date"`
	AttendanceEndDate   string `json:"attendance_end_date"`
}

type CensusTestFixtureResult struct {
	FixtureID           string `json:"fixture_id"`
	AcademicYearID      string `json:"academic_year_id"`
	CensusDate          string `json:"census_date"`
	TermCode            string `json:"term_code"`
	AttendanceStartDate string `json:"attendance_start_date"`
	AttendanceEndDate   string `json:"attendance_end_date"`
	AdminStaffID        string `json:"admin_staff_id"`
	TeacherStaffID      string `json:"teacher_staff_id"`
	PupilSenKFsmID      string `json:"pupil_sen_k_fsm_id"`
	PupilEhcpLacID      string `json:"pupil_ehcp_lac_id"`
	PupilNoSenID        string `json:"pupil_no_sen_id"`
	OffRollPupilID      string `json:"off_roll_pupil_id"`
}

func (d *CoreDAL) SeedCensusTestFixtures(ctx context.Context, in CensusTestFixtureInput) (*CensusTestFixtureResult, error) {
	ayID := strings.TrimSpace(in.AcademicYearID)
	if ayID == "" {
		return nil, errors.New("missing mandatory field: academic_year_id")
	}
	censusDate := strings.TrimSpace(in.CensusDate)
	if censusDate == "" {
		return nil, errors.New("missing mandatory field: census_date")
	}
	termCode := strings.ToUpper(strings.TrimSpace(in.TermCode))
	if termCode == "" {
		termCode = "AUTUMN"
	}
	attStart := strings.TrimSpace(in.AttendanceStartDate)
	if attStart == "" {
		attStart = "2026-09-01"
	}
	attEnd := strings.TrimSpace(in.AttendanceEndDate)
	if attEnd == "" {
		attEnd = censusDate
	}

	u := uuid.New()
	sfx := strings.ReplaceAll(u.String(), "-", "")[:8]
	numSeed := int(u.ID()%800000) + 100000

	res := &CensusTestFixtureResult{
		FixtureID:           sfx,
		AcademicYearID:      ayID,
		CensusDate:          censusDate,
		TermCode:            termCode,
		AttendanceStartDate: attStart,
		AttendanceEndDate:   attEnd,
		AdminStaffID:        "stf_cadm_" + sfx,
		TeacherStaffID:      "stf_ctch_" + sfx,
		PupilSenKFsmID:      "pup_csenk_" + sfx,
		PupilEhcpLacID:      "pup_cehcp_" + sfx,
		PupilNoSenID:        "pup_cnosen_" + sfx,
		OffRollPupilID:      "pup_coff_" + sfx,
	}

	upn1 := fmt.Sprintf("H8234%06d01", numSeed)
	upn2 := fmt.Sprintf("H8234%06d02", numSeed)
	upn3 := fmt.Sprintf("H8234%06d03", numSeed)
	upn4 := fmt.Sprintf("H8234%06d04", numSeed)

	uln1 := fmt.Sprintf("10%06d01", numSeed)
	uln2 := fmt.Sprintf("10%06d02", numSeed)
	uln3 := fmt.Sprintf("10%06d03", numSeed)
	uln4 := fmt.Sprintf("10%06d04", numSeed)

	tx, err := d.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin SeedCensusTestFixtures transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	stmts := []struct {
		sql  string
		args []any
	}{
		{
			sql: `INSERT INTO academic_years (id, name, status, sequence_number, first_day, last_day)
			      VALUES (?, '2026/2027', 'CURRENT', (SELECT COALESCE(MAX(sequence_number), 0) + 1 FROM academic_years), '2026-09-01', '2027-07-23')
			      ON CONFLICT(id) DO NOTHING`,
			args: []any{ayID},
		},
		{
			sql: `INSERT INTO alert_levels (id, name, name_short, color_hex, bg_color_hex, description, sequence_number)
			      VALUES ('HIGH', 'High Priority', 'HI', '#FFFFFF', '#D32F2F', 'Critical alert', 1),
			             ('MEDIUM', 'Medium Priority', 'MD', '#FFFFFF', '#F57C00', 'Medium alert', 2)
			      ON CONFLICT(id) DO NOTHING`,
		},
		{
			sql: `INSERT INTO sen_need_types (code, name, broad_area, sequence_number)
			      VALUES ('ASD', 'Autistic Spectrum Disorder', 'COMMUNICATION_AND_INTERACTION', 1),
			             ('SPLD', 'Specific Learning Difficulty', 'COGNITION_AND_LEARNING', 2)
			      ON CONFLICT(code) DO NOTHING`,
		},
		{
			sql: `INSERT INTO attendance_codes (code, name, statistical_category, physical_status, sequence_number)
			      VALUES ('/', 'Present (AM)', 'PRESENT', 'ONSITE', 1),
			             ('\', 'Present (PM)', 'PRESENT', 'ONSITE', 2),
			             ('I', 'Illness', 'AUTHORISED_ABSENCE', 'ABSENT', 3),
			             ('O', 'Unauthorised Absence', 'UNAUTHORISED_ABSENCE', 'ABSENT', 4)
			      ON CONFLICT(code) DO NOTHING`,
		},
		{
			sql: `INSERT INTO roles (id, category, name, name_short, description, default_scopes_json)
			      VALUES
			        ('ADMIN', 'STAFF', 'System Administrator / Headteacher', 'ADM', 'Full administrative & statutory census authority', '["dal.census.read:all","dal.pupils.read:all","dal.pupils.write:all","dal.staff.read:admin","dal.staff.write:admin","dal.groups.read:all","dal.groups.write:all","dal.attendance.read:all","dal.attendance.write:all","dal.guardians.read:all","dal.guardians.write:all","dal.sen.read:all","dal.medical.read:all"]'),
			        ('TEACHER', 'STAFF', 'Classroom Teacher', 'TCH', 'Scoped classroom teacher access', '["dal.pupils.read:own_groups","dal.groups.read:own_groups","dal.attendance.read:own_groups","dal.attendance.write:own_groups","dal.sen.read:own_groups","dal.medical.read:own_groups","dal.safeguarding.create"]')
			      ON CONFLICT(id) DO UPDATE SET default_scopes_json = excluded.default_scopes_json`,
		},
		{
			sql: `INSERT INTO persons (id, external_sso_sub, title, legal_first_name, legal_last_name, preferred_first_name, preferred_last_name, dob, sex, status)
			      VALUES
			        (?, ?, 'Dame', 'Eleanor', 'Vance', 'Eleanor', 'Vance', '1975-04-12', 'F', 'ACTIVE'),
			        (?, ?, 'Ms', 'Hannah', 'Abbott', 'Hannah', 'Abbott', '1990-08-19', 'F', 'ACTIVE'),
			        (?, NULL, '', 'Maya', 'Patel', 'Maya', 'Patel', '2011-05-19', 'F', 'ACTIVE'),
			        (?, NULL, '', 'Noah', 'Williams', 'Noah', 'Williams', '2011-11-23', 'M', 'ACTIVE'),
			        (?, NULL, '', 'Chloe', 'Sterling', 'Chloe', 'Sterling', '2011-02-04', 'F', 'ACTIVE'),
			        (?, NULL, '', 'Oliver', 'Grant', 'Oliver', 'Grant', '2011-03-29', 'M', 'LEFT')`,
			args: []any{
				"per_cadm_" + sfx, "sso_cadm_" + sfx,
				"per_ctch_" + sfx, "sso_ctch_" + sfx,
				"per_cp1_" + sfx,
				"per_cp2_" + sfx,
				"per_cp3_" + sfx,
				"per_cp4_" + sfx,
			},
		},
		{
			sql: `INSERT INTO person_roles (person_id, role_id, is_primary, effective_from)
			      VALUES
			        (?, 'ADMIN', 1, '2026-09-01'),
			        (?, 'TEACHER', 1, '2026-09-01')`,
			args: []any{
				"per_cadm_" + sfx,
				"per_ctch_" + sfx,
			},
		},
		{
			sql: `INSERT INTO staff (id, person_id, staff_code, staff_type, job_title, qts_flag, is_dsl, hire_date)
			      VALUES
			        (?, ?, ?, 'LEADERSHIP', 'Executive Headteacher', 1, 1, '2018-09-01'),
			        (?, ?, ?, 'TEACHING', 'Teacher of Mathematics', 1, 0, '2021-09-01')`,
			args: []any{
				res.AdminStaffID, "per_cadm_" + sfx, "CA_" + sfx[:4],
				res.TeacherStaffID, "per_ctch_" + sfx, "CT_" + sfx[:4],
			},
		},
		{
			sql: `INSERT INTO pupils (id, person_id, upn, uln, ethnicity_code, first_language_code, pupil_premium_flag, looked_after_flag, post_looked_after_code)
			      VALUES
			        (?, ?, ?, ?, 'INDN', 'GUJ', 1, 0, 'N'),
			        (?, ?, ?, ?, 'WBRI', 'ENG', 1, 1, 'A'),
			        (?, ?, ?, ?, 'WBRI', 'ENG', 0, 0, 'N'),
			        (?, ?, ?, ?, 'WBRI', 'ENG', 0, 0, 'N')`,
			args: []any{
				res.PupilSenKFsmID, "per_cp1_" + sfx, upn1, uln1,
				res.PupilEhcpLacID, "per_cp2_" + sfx, upn2, uln2,
				res.PupilNoSenID, "per_cp3_" + sfx, upn3, uln3,
				res.OffRollPupilID, "per_cp4_" + sfx, upn4, uln4,
			},
		},
		{
			sql: `INSERT INTO pupil_year_enrolments (id, pupil_id, academic_year_id, nc_year_group, enrolment_status, effective_from, effective_to)
			      VALUES
			        (?, ?, ?, '10', 'C', '2026-09-01', NULL),
			        (?, ?, ?, '10', 'C', '2026-09-01', NULL),
			        (?, ?, ?, '10', 'C', '2026-09-01', NULL),
			        (?, ?, ?, '10', 'C', '2026-09-01', '2026-09-15')`,
			args: []any{
				"cenr_1_" + sfx, res.PupilSenKFsmID, ayID,
				"cenr_2_" + sfx, res.PupilEhcpLacID, ayID,
				"cenr_3_" + sfx, res.PupilNoSenID, ayID,
				"cenr_4_" + sfx, res.OffRollPupilID, ayID,
			},
		},
		{
			sql: `INSERT INTO pupil_sen_records (id, pupil_id, provision_code, ehcp_start_date, teaching_strategies, effective_from, effective_to)
			      VALUES
			        (?, ?, 'K', NULL, 'Extra time and dyslexia overlays', '2026-09-01', NULL),
			        (?, ?, 'E', '2025-05-10', '1:1 TA support and structured visual schedule', '2026-09-01', NULL),
			        (?, ?, 'N', NULL, '', '2026-09-01', NULL)`,
			args: []any{
				"csen_1_" + sfx, res.PupilSenKFsmID,
				"csen_2_" + sfx, res.PupilEhcpLacID,
				"csen_3_" + sfx, res.PupilNoSenID,
			},
		},
		{
			sql: `INSERT INTO pupil_sen_needs (id, pupil_id, sen_need_code, ranking, alert_level_id, effective_from, effective_to)
			      VALUES
			        (?, ?, 'SPLD', 1, 'MEDIUM', '2026-09-01', NULL),
			        (?, ?, 'ASD', 1, 'HIGH', '2026-09-01', NULL)`,
			args: []any{
				"csneed_1_" + sfx, res.PupilSenKFsmID,
				"csneed_2_" + sfx, res.PupilEhcpLacID,
			},
		},
		{
			sql: `INSERT INTO pupil_fsm_periods (id, pupil_id, effective_from, effective_to, country_code, ecs_reference)
			      VALUES
			        (?, ?, '2025-09-01', NULL, 'ENG', 'ECS-001'),
			        (?, ?, '2025-09-01', NULL, 'ENG', 'ECS-002')`,
			args: []any{
				"cfsm_1_" + sfx, res.PupilSenKFsmID,
				"cfsm_2_" + sfx, res.PupilEhcpLacID,
			},
		},
		{
			sql: `INSERT INTO attendance_session_marks (id, pupil_id, academic_year_id, date, session, attendance_code, recorded_by_staff_id)
			      VALUES
			        (?, ?, ?, '2026-09-10', 'AM', '/', ?),
			        (?, ?, ?, '2026-09-10', 'PM', '\', ?),
			        (?, ?, ?, '2026-09-11', 'AM', 'I', ?),
			        (?, ?, ?, '2026-09-11', 'PM', 'O', ?),
			        (?, ?, ?, '2026-09-10', 'AM', '/', ?),
			        (?, ?, ?, '2026-09-10', 'PM', '\', ?),
			        (?, ?, ?, '2026-09-10', 'AM', '/', ?),
			        (?, ?, ?, '2026-09-10', 'PM', '\', ?)`,
			args: []any{
				"casm_1_" + sfx, res.PupilSenKFsmID, ayID, res.TeacherStaffID,
				"casm_2_" + sfx, res.PupilSenKFsmID, ayID, res.TeacherStaffID,
				"casm_3_" + sfx, res.PupilSenKFsmID, ayID, res.TeacherStaffID,
				"casm_4_" + sfx, res.PupilSenKFsmID, ayID, res.TeacherStaffID,
				"casm_5_" + sfx, res.PupilEhcpLacID, ayID, res.TeacherStaffID,
				"casm_6_" + sfx, res.PupilEhcpLacID, ayID, res.TeacherStaffID,
				"casm_7_" + sfx, res.PupilNoSenID, ayID, res.TeacherStaffID,
				"casm_8_" + sfx, res.PupilNoSenID, ayID, res.TeacherStaffID,
			},
		},
	}

	for _, stmt := range stmts {
		if _, err := tx.ExecContext(ctx, stmt.sql, stmt.args...); err != nil {
			return nil, fmt.Errorf("failed to execute census fixture seed statement: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit SeedCensusTestFixtures transaction: %w", err)
	}
	return res, nil
}

func (d *CoreDAL) DeleteCensusTestFixtures(ctx context.Context, fixtureID string) error {
	sfx := strings.TrimSpace(fixtureID)
	if sfx == "" {
		return errors.New("missing mandatory parameter: fixture_id")
	}

	p1 := "pup_csenk_" + sfx
	p2 := "pup_cehcp_" + sfx
	p3 := "pup_cnosen_" + sfx
	p4 := "pup_coff_" + sfx

	tx, err := d.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin DeleteCensusTestFixtures transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	cleanupStmts := []struct {
		sql  string
		args []any
	}{
		{
			sql:  `DELETE FROM attendance_session_marks WHERE pupil_id IN (?, ?, ?, ?)`,
			args: []any{p1, p2, p3, p4},
		},
		{
			sql:  `DELETE FROM pupil_fsm_periods WHERE pupil_id IN (?, ?, ?, ?)`,
			args: []any{p1, p2, p3, p4},
		},
		{
			sql:  `DELETE FROM pupil_sen_needs WHERE pupil_id IN (?, ?, ?, ?)`,
			args: []any{p1, p2, p3, p4},
		},
		{
			sql:  `DELETE FROM pupil_sen_records WHERE pupil_id IN (?, ?, ?, ?)`,
			args: []any{p1, p2, p3, p4},
		},
		{
			sql:  `DELETE FROM pupil_year_enrolments WHERE pupil_id IN (?, ?, ?, ?)`,
			args: []any{p1, p2, p3, p4},
		},
		{
			sql:  `DELETE FROM pupils WHERE id IN (?, ?, ?, ?)`,
			args: []any{p1, p2, p3, p4},
		},
		{
			sql:  `DELETE FROM staff WHERE id IN (?, ?)`,
			args: []any{"stf_cadm_" + sfx, "stf_ctch_" + sfx},
		},
		{
			sql:  `DELETE FROM person_roles WHERE person_id IN (?, ?)`,
			args: []any{"per_cadm_" + sfx, "per_ctch_" + sfx},
		},
		{
			sql: `DELETE FROM persons WHERE id IN (?, ?, ?, ?, ?, ?)`,
			args: []any{
				"per_cadm_" + sfx,
				"per_ctch_" + sfx,
				"per_cp1_" + sfx,
				"per_cp2_" + sfx,
				"per_cp3_" + sfx,
				"per_cp4_" + sfx,
			},
		},
	}

	for _, q := range cleanupStmts {
		if _, err := tx.ExecContext(ctx, q.sql, q.args...); err != nil {
			return fmt.Errorf("failed to execute census cleanup query (%s): %w", q.sql, err)
		}
	}

	return tx.Commit()
}
