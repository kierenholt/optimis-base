package dal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

type StatutoryCensusRollupInput struct {
	AcademicYearID      string       `json:"academic_year_id"`
	CensusDate          string       `json:"census_date"`
	TermCode            string       `json:"term_code"`
	AttendanceStartDate string       `json:"attendance_start_date"`
	AttendanceEndDate   string       `json:"attendance_end_date"`
	Scope               ScopeMode `json:"scope"`
}

type CensusPupilAttendanceSessions struct {
	PossibleSessions            int `json:"possible_sessions"`
	PresentSessions             int `json:"present_sessions"`
	AuthorisedAbsenceSessions   int `json:"authorised_absence_sessions"`
	UnauthorisedAbsenceSessions int `json:"unauthorised_absence_sessions"`
	AMPresent                   int `json:"am_present"`
	PMPresent                   int `json:"pm_present"`
}

type CensusPupilRecord struct {
	PupilID                 string                             `json:"pupil_id"`
	UPN                     string                             `json:"upn"`
	ULN                     string                             `json:"uln,omitempty"`
	FirstName               string                             `json:"first_name"`
	LastName                string                             `json:"last_name"`
	DOB                     string                             `json:"dob"`
	Sex                     string                             `json:"sex"`
	EthnicityCode           string                             `json:"ethnicity_code"`
	FirstLanguageCode       string                             `json:"first_language_code"`
	EALFlag                 bool                               `json:"eal_flag"`
	PupilPremiumFlag        bool                               `json:"pupil_premium_flag"`
	ServiceChildFlag        bool                               `json:"service_child_flag"`
	LookedAfterFlag         bool                               `json:"looked_after_flag"`
	PostLookedAfterCode     string                             `json:"post_looked_after_code"`
	LACEpisodeStatus        string                             `json:"lac_episode_status"`
	NCYearGroup             string                             `json:"nc_year_group"`
	EnrolmentStatus         string                             `json:"enrolment_status"`
	EYFundedHours           float64                            `json:"ey_funded_hours,omitempty"`
	EYExtendedFundedHours   float64                            `json:"ey_extended_funded_hours,omitempty"`
	EffectiveFrom           string                             `json:"effective_from"`
	EffectiveTo             string                             `json:"effective_to,omitempty"`
	FSMEligible             bool                               `json:"fsm_eligible"`
	FSMPeriods              []PupilFSMPeriodSummary            `json:"fsm_periods"`
	SENProvisionCode        string                             `json:"sen_provision_code"`
	SENStatusCode           string                             `json:"sen_status_code"`
	PrimarySENNeedCode      string                             `json:"primary_sen_need_code"`
	PrimaryNeedCode         string                             `json:"primary_need_code"`
	AttendanceSessions      CensusPupilAttendanceSessions      `json:"attendance_sessions"`
	Exclusions              []PupilExclusionSummary            `json:"exclusions,omitempty"`
	AlternativeProvisions   []PupilAlternativeProvisionSummary `json:"alternative_provisions,omitempty"`
	Post16LearningAims      []PupilPost16LearningAimSummary    `json:"post16_learning_aims,omitempty"`
}

type StatutoryCensusSummary struct {
	OnRollTotal                      int            `json:"on_roll_total"`
	TotalPupilsOnRoll                int            `json:"total_pupils_on_roll"`
	SENCounts                        map[string]int `json:"sen_counts"`
	FSMEligibleCount                 int            `json:"fsm_eligible_count"`
	FSMPupilsCount                   int            `json:"fsm_pupils_count"`
	SENSupportCount                  int            `json:"sen_support_count"`
	EHCPCount                        int            `json:"ehcp_count"`
	LACCount                         int            `json:"lac_count"`
	ExclusionsCount                  int            `json:"exclusions_count"`
	AlternativeProvisionCount        int            `json:"alternative_provision_count"`
	Post16LearningAimsCount          int            `json:"post16_learning_aims_count"`
	TotalPossibleSessions            int            `json:"total_possible_sessions"`
	TotalPresentSessions             int            `json:"total_present_sessions"`
	TotalAuthorisedAbsenceSessions   int            `json:"total_authorised_absence_sessions"`
	TotalUnauthorisedAbsenceSessions int            `json:"total_unauthorised_absence_sessions"`
}

type StatutoryCensusRollup struct {
	SchoolURN               string                 `json:"school_urn"`
	SchoolName              string                 `json:"school_name"`
	Phase                   string                 `json:"phase"`
	LAEstab                 string                 `json:"la_estab"`
	AcademicYearID          string                 `json:"academic_year_id"`
	TermCode                string                 `json:"term_code"`
	CensusDate              string                 `json:"census_date"`
	ReferenceDate           string                 `json:"reference_date"`
	AttendanceTermStartDate string                 `json:"attendance_term_start_date"`
	AttendanceTermEndDate   string                 `json:"attendance_term_end_date"`
	Pupils                  []CensusPupilRecord    `json:"pupils"`
	Summary                 StatutoryCensusSummary `json:"summary"`
}

func (d *CoreDAL) GetStatutoryCensusRollup(
	ctx context.Context,
	in StatutoryCensusRollupInput,
) (*StatutoryCensusRollup, error) {
	if in.Scope != ScopeAll {
		return nil, errors.New("access denied: dal.census.read:all scope is required for statutory census rollup")
	}

	resolvedScope := DALQueryScope{
		Mode:           ScopeAll,
		AcademicYearID: strings.TrimSpace(in.AcademicYearID),
		ReferenceDate:  strings.TrimSpace(in.CensusDate),
	}
	var err error
	resolvedScope, err = d.ResolveScopeDefaults(ctx, resolvedScope)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve census rollup scope defaults: %w", err)
	}

	termCode := strings.ToUpper(strings.TrimSpace(in.TermCode))
	if termCode == "" {
		termCode = "AUTUMN"
	}

	attStart := strings.TrimSpace(in.AttendanceStartDate)
	if attStart == "" {
		_ = d.readDB.QueryRowContext(
			ctx,
			`SELECT first_day FROM academic_years WHERE id = ?`,
			resolvedScope.AcademicYearID,
		).Scan(&attStart)
		if attStart == "" {
			attStart = "2000-01-01"
		}
	}
	attEnd := strings.TrimSpace(in.AttendanceEndDate)
	if attEnd == "" {
		attEnd = resolvedScope.ReferenceDate
	}

	schoolURN := "100000"
	schoolName := "Optimis School"
	schoolPhase := "ALL_THROUGH"
	laCode := "201"
	estabNumber := "4000"

	sysRows, err := d.readDB.QueryContext(
		ctx,
		`SELECT key, value FROM __system_config WHERE key IN ('school.urn', 'school_urn', 'school.name', 'school_name', 'school.phase', 'school.la_code', 'school.estab_number')`,
	)
	if err == nil {
		for sysRows.Next() {
			var k, v string
			if scanErr := sysRows.Scan(&k, &v); scanErr == nil && strings.TrimSpace(v) != "" {
				switch k {
				case "school.urn", "school_urn":
					schoolURN = strings.TrimSpace(v)
				case "school.name", "school_name":
					schoolName = strings.TrimSpace(v)
				case "school.phase":
					schoolPhase = strings.TrimSpace(v)
				case "school.la_code":
					laCode = strings.TrimSpace(v)
				case "school.estab_number":
					estabNumber = strings.TrimSpace(v)
				}
			}
		}
		_ = sysRows.Close()
	}

	rows, err := d.readDB.QueryContext(
		ctx,
		`SELECT p.id, p.upn, COALESCE(p.uln, ''), per.legal_first_name, per.legal_last_name,
		        COALESCE(per.dob, '2014-01-01'), COALESCE(per.sex, 'U'),
		        COALESCE(p.ethnicity_code, 'NOBT'), COALESCE(p.first_language_code, 'ENG'),
		        p.eal_flag, p.pupil_premium_flag, p.service_child_flag, p.looked_after_flag,
		        COALESCE(p.post_looked_after_code, 'N'),
		        enr.nc_year_group, enr.enrolment_status,
		        COALESCE(enr.ey_funded_hours, 0), COALESCE(enr.ey_extended_funded_hours, 0),
		        enr.effective_from, enr.effective_to,
		        CASE WHEN EXISTS (
		            SELECT 1 FROM pupil_fsm_periods fsm
		            WHERE fsm.pupil_id = p.id
		              AND fsm.effective_from <= ?
		              AND (fsm.effective_to IS NULL OR fsm.effective_to >= ?)
		        ) THEN 1 ELSE 0 END AS fsm_eligible,
		        COALESCE((
		            SELECT sr.provision_code FROM pupil_sen_records sr
		            WHERE sr.pupil_id = p.id
		              AND sr.effective_from <= ?
		              AND (sr.effective_to IS NULL OR sr.effective_to >= ?)
		            ORDER BY sr.effective_from DESC LIMIT 1
		        ), 'N') AS sen_status_code,
		        COALESCE((
		            SELECT sn.sen_need_code FROM pupil_sen_needs sn
		            WHERE sn.pupil_id = p.id
		              AND sn.effective_from <= ?
		              AND (sn.effective_to IS NULL OR sn.effective_to >= ?)
		            ORDER BY sn.ranking ASC, sn.id ASC LIMIT 1
		        ), '') AS primary_need_code
		 FROM pupils p
		 JOIN persons per ON per.id = p.person_id
		 JOIN pupil_year_enrolments enr ON enr.pupil_id = p.id
		 WHERE enr.academic_year_id = ?
		   AND enr.effective_from <= ?
		   AND (enr.effective_to IS NULL OR enr.effective_to >= ?)
		 ORDER BY per.legal_last_name ASC, per.legal_first_name ASC, p.id ASC`,
		resolvedScope.ReferenceDate, resolvedScope.ReferenceDate,
		resolvedScope.ReferenceDate, resolvedScope.ReferenceDate,
		resolvedScope.ReferenceDate, resolvedScope.ReferenceDate,
		resolvedScope.AcademicYearID, resolvedScope.ReferenceDate, resolvedScope.ReferenceDate,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query statutory census pupil rollup: %w", err)
	}
	defer rows.Close()

	var pupils []CensusPupilRecord
	pupilIndexByID := make(map[string]int)
	pupilIDs := make([]string, 0)
	summary := StatutoryCensusSummary{
		SENCounts: map[string]int{"N": 0, "K": 0, "E": 0},
	}

	for rows.Next() {
		var rec CensusPupilRecord
		var ealInt, ppInt, lacInt, fsmInt int
		var svcStr string
		var effTo sql.NullString
		if err := rows.Scan(
			&rec.PupilID,
			&rec.UPN,
			&rec.ULN,
			&rec.FirstName,
			&rec.LastName,
			&rec.DOB,
			&rec.Sex,
			&rec.EthnicityCode,
			&rec.FirstLanguageCode,
			&ealInt,
			&ppInt,
			&svcStr,
			&lacInt,
			&rec.PostLookedAfterCode,
			&rec.NCYearGroup,
			&rec.EnrolmentStatus,
			&rec.EYFundedHours,
			&rec.EYExtendedFundedHours,
			&rec.EffectiveFrom,
			&effTo,
			&fsmInt,
			&rec.SENProvisionCode,
			&rec.PrimarySENNeedCode,
		); err != nil {
			return nil, fmt.Errorf("failed to scan census pupil record: %w", err)
		}
		if effTo.Valid {
			rec.EffectiveTo = effTo.String
		}
		rec.EALFlag = ealInt == 1
		rec.PupilPremiumFlag = ppInt == 1
		rec.ServiceChildFlag = strings.EqualFold(strings.TrimSpace(svcStr), "Y") || strings.TrimSpace(svcStr) == "1"
		rec.LookedAfterFlag = lacInt == 1
		if rec.LookedAfterFlag {
			rec.LACEpisodeStatus = "IN_CARE"
		} else if rec.PostLookedAfterCode != "" && rec.PostLookedAfterCode != "N" {
			rec.LACEpisodeStatus = "POST_LAC_" + rec.PostLookedAfterCode
		} else {
			rec.LACEpisodeStatus = "NONE"
		}
		rec.FSMEligible = fsmInt == 1
		rec.SENStatusCode = rec.SENProvisionCode
		rec.PrimaryNeedCode = rec.PrimarySENNeedCode
		rec.FSMPeriods = []PupilFSMPeriodSummary{}
		rec.Exclusions = []PupilExclusionSummary{}
		rec.AlternativeProvisions = []PupilAlternativeProvisionSummary{}
		rec.Post16LearningAims = []PupilPost16LearningAimSummary{}

		summary.OnRollTotal++
		summary.TotalPupilsOnRoll++
		if rec.FSMEligible {
			summary.FSMEligibleCount++
			summary.FSMPupilsCount++
		}
		if rec.LookedAfterFlag {
			summary.LACCount++
		}
		summary.SENCounts[rec.SENProvisionCode]++
		switch rec.SENProvisionCode {
		case "K":
			summary.SENSupportCount++
		case "E":
			summary.EHCPCount++
		}

		pupilIndexByID[rec.PupilID] = len(pupils)
		pupilIDs = append(pupilIDs, rec.PupilID)
		pupils = append(pupils, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating census pupil rollup: %w", err)
	}

	if len(pupils) > 0 {
		if fsmMap, fErr := d.ListPupilFSMPeriodsBatchScoped(ctx, resolvedScope, pupilIDs); fErr == nil {
			for pid, periods := range fsmMap {
				if idx, ok := pupilIndexByID[pid]; ok {
					pupils[idx].FSMPeriods = periods
				}
			}
		}

		attRows, err := d.readDB.QueryContext(
			ctx,
			`SELECT sm.pupil_id, sm.session, ac.statistical_category, COUNT(*)
			 FROM attendance_session_marks sm
			 JOIN attendance_codes ac ON ac.code = sm.attendance_code
			 WHERE sm.academic_year_id = ?
			   AND sm.date >= ?
			   AND sm.date <= ?
			 GROUP BY sm.pupil_id, sm.session, ac.statistical_category`,
			resolvedScope.AcademicYearID, attStart, attEnd,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to aggregate statutory census attendance: %w", err)
		}
		defer attRows.Close()

		for attRows.Next() {
			var pid, session, meaning string
			var count int
			if err := attRows.Scan(&pid, &session, &meaning, &count); err != nil {
				return nil, fmt.Errorf("failed to scan census attendance row: %w", err)
			}
			idx, ok := pupilIndexByID[pid]
			if !ok {
				continue
			}
			switch meaning {
			case "PRESENT", "APPROVED_EDUCATIONAL_ACTIVITY":
				pupils[idx].AttendanceSessions.PossibleSessions += count
				pupils[idx].AttendanceSessions.PresentSessions += count
				if session == "AM" {
					pupils[idx].AttendanceSessions.AMPresent += count
				} else if session == "PM" {
					pupils[idx].AttendanceSessions.PMPresent += count
				}
				summary.TotalPossibleSessions += count
				summary.TotalPresentSessions += count
			case "AUTHORISED_ABSENCE":
				pupils[idx].AttendanceSessions.PossibleSessions += count
				pupils[idx].AttendanceSessions.AuthorisedAbsenceSessions += count
				summary.TotalPossibleSessions += count
				summary.TotalAuthorisedAbsenceSessions += count
			case "UNAUTHORISED_ABSENCE":
				pupils[idx].AttendanceSessions.PossibleSessions += count
				pupils[idx].AttendanceSessions.UnauthorisedAbsenceSessions += count
				summary.TotalPossibleSessions += count
				summary.TotalUnauthorisedAbsenceSessions += count
			}
		}
		if err := attRows.Err(); err != nil {
			return nil, fmt.Errorf("error iterating census attendance aggregation: %w", err)
		}

		exclusions, err := d.ListPupilExclusionsScoped(ctx, resolvedScope, "", resolvedScope.AcademicYearID)
		if err == nil {
			for _, ex := range exclusions {
				if idx, ok := pupilIndexByID[ex.PupilID]; ok {
					pupils[idx].Exclusions = append(pupils[idx].Exclusions, ex)
					summary.ExclusionsCount++
				}
			}
		}

		aps, err := d.ListPupilAlternativeProvisionsScoped(ctx, resolvedScope, "", resolvedScope.AcademicYearID)
		if err == nil {
			for _, ap := range aps {
				if idx, ok := pupilIndexByID[ap.PupilID]; ok {
					pupils[idx].AlternativeProvisions = append(pupils[idx].AlternativeProvisions, ap)
					summary.AlternativeProvisionCount++
				}
			}
		}

		aims, err := d.ListPupilPost16LearningAimsScoped(ctx, resolvedScope, "", resolvedScope.AcademicYearID)
		if err == nil {
			for _, aim := range aims {
				if idx, ok := pupilIndexByID[aim.PupilID]; ok {
					pupils[idx].Post16LearningAims = append(pupils[idx].Post16LearningAims, aim)
					summary.Post16LearningAimsCount++
				}
			}
		}
	}

	if pupils == nil {
		pupils = []CensusPupilRecord{}
	}

	return &StatutoryCensusRollup{
		SchoolURN:               schoolURN,
		SchoolName:              schoolName,
		Phase:                   schoolPhase,
		LAEstab:                 laCode + estabNumber,
		AcademicYearID:          resolvedScope.AcademicYearID,
		TermCode:                termCode,
		CensusDate:              resolvedScope.ReferenceDate,
		ReferenceDate:           resolvedScope.ReferenceDate,
		AttendanceTermStartDate: attStart,
		AttendanceTermEndDate:   attEnd,
		Pupils:                  pupils,
		Summary:                 summary,
	}, nil
}
