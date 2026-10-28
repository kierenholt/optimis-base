package main

import (
	"database/sql"
	"fmt"
)

func birthYearForNC(nc string) int {
	switch nc {
	case "N1":
		return 2022
	case "N2":
		return 2021
	case "R":
		return 2020
	case "1":
		return 2019
	case "2":
		return 2018
	case "3":
		return 2017
	case "4":
		return 2016
	case "5":
		return 2015
	case "6":
		return 2014
	case "7":
		return 2013
	case "8":
		return 2012
	case "9":
		return 2011
	case "10":
		return 2010
	case "11":
		return 2009
	case "12":
		return 2008
	case "13":
		return 2007
	default:
		return 2013
	}
}

func targetFormSize(phase string) int {
	switch phase {
	case "NURSERY":
		return 20
	case "PRIMARY":
		return 26
	case "SECONDARY":
		return 28
	case "SIXTH_FORM":
		return 24
	default:
		return 26
	}
}

func seedPupilsCoursesAndTimetable(tx *sql.Tx, st *qaSeedState, _ any) error {
	type anchorPupilSpec struct {
		pupilID, personID, first, last, sex, dob, formCode string
		eth, lang, enrStatus, senCode, personStatus        string
		eal, pp, eyPP, lac, daf                            int
		eyFunded, eyExt                                    float64
		eyCode, leaveReason, effFrom                       string
		effTo                                              *string
		transferFromForm7A                                 bool
	}

	leftDate := "2026-10-02"
	anchorsByForm := map[string][]anchorPupilSpec{
		"N2A": {
			{
				pupilID: "pup_eyfs_leo", personID: "per_pup_eyfs_leo",
				first: "Leo", last: "Carter", sex: "M", dob: "2022-02-14", formCode: "N2A",
				eth: "WBRI", lang: "ENG", enrStatus: "C", senCode: "N", personStatus: "ACTIVE",
				eyPP: 1, daf: 1, eyFunded: 15.0, eyExt: 15.0, eyCode: "50012345678", effFrom: "2026-09-01",
			},
		},
		"4A": {
			{
				pupilID: "pup_pri_maya", personID: "per_pup_pri_maya",
				first: "Maya", last: "Patel", sex: "F", dob: "2017-05-19", formCode: "4A",
				eth: "INDN", lang: "GUJ", enrStatus: "C", senCode: "N", personStatus: "ACTIVE",
				eal: 1, pp: 1, effFrom: "2026-09-01",
			},
		},
		"7A": {
			{
				pupilID: "pup_yr7_alice", personID: "per_pup_yr7_alice",
				first: "Alice", last: "Evans", sex: "F", dob: "2014-01-11", formCode: "7A",
				eth: "WBRI", lang: "ENG", enrStatus: "C", senCode: "N", personStatus: "ACTIVE",
				pp: 1, effFrom: "2026-09-01",
			},
			{
				pupilID: "pup_yr7_noah", personID: "per_pup_yr7_noah",
				first: "Noah", last: "Williams", sex: "M", dob: "2013-11-23", formCode: "7A",
				eth: "WBRI", lang: "ENG", enrStatus: "C", senCode: "E", personStatus: "ACTIVE",
				pp: 1, effFrom: "2026-09-01",
			},
		},
		"7B": {
			{
				pupilID: "pup_yr7_zara", personID: "per_pup_yr7_zara",
				first: "Zara", last: "Khan", sex: "F", dob: "2014-04-08", formCode: "7B",
				eth: "PKST", lang: "URD", enrStatus: "C", senCode: "K", personStatus: "ACTIVE",
				eal: 1, effFrom: "2026-09-01",
			},
			{
				pupilID: "pup_yr7_liam", personID: "per_pup_yr7_liam",
				first: "Liam", last: "O'Connor", sex: "M", dob: "2013-10-03", formCode: "7B",
				eth: "WIRI", lang: "ENG", enrStatus: "C", senCode: "N", personStatus: "ACTIVE",
				effFrom: "2026-09-01", transferFromForm7A: true,
			},
		},
		"10A": {
			{
				pupilID: "pup_yr10_ryan", personID: "per_pup_yr10_ryan",
				first: "Ryan", last: "Mitchell", sex: "M", dob: "2010-12-17", formCode: "10A",
				eth: "MWBC", lang: "ENG", enrStatus: "S", senCode: "K", personStatus: "ACTIVE",
				pp: 1, lac: 1, effFrom: "2026-09-01",
			},
		},
		"10B": {
			{
				pupilID: "pup_left_oliver", personID: "per_pup_left_oliver",
				first: "Oliver", last: "Grant", sex: "M", dob: "2011-03-29", formCode: "10B",
				eth: "WBRI", lang: "ENG", enrStatus: "C", senCode: "N", personStatus: "LEFT",
				leaveReason: "OT", effFrom: "2026-09-01", effTo: &leftDate,
			},
		},
		"12A": {
			{
				pupilID: "pup_yr12_chloe", personID: "per_pup_yr12_chloe",
				first: "Chloe", last: "Sterling", sex: "F", dob: "2009-02-04", formCode: "12A",
				eth: "WBRI", lang: "ENG", enrStatus: "C", senCode: "N", personStatus: "ACTIVE",
				effFrom: "2026-09-01",
			},
		},
	}

	serial := 1
	for _, form := range st.forms {
		size := targetFormSize(form.Phase)
		anchors := anchorsByForm[form.Code]

		for slot := 0; slot < size; slot++ {
			var meta *seededPupilMeta
			houseID := st.houseIDs[(serial-1)%len(st.houseIDs)]
			upn := computeUPN("9314001", 26, serial)
			nhs := fmt.Sprintf("943%07d", serial)
			admNo := fmt.Sprintf("ADM%04d", serial)
			var uln any
			if form.Phase == "SECONDARY" || form.Phase == "SIXTH_FORM" {
				uln = fmt.Sprintf("10%08d", serial)
			}

			if slot < len(anchors) {
				a := anchors[slot]
				var eyCode any
				if a.eyCode != "" {
					eyCode = a.eyCode
				}
				var leaveReason any
				if a.leaveReason != "" {
					leaveReason = a.leaveReason
				}

				if _, err := tx.Exec(`INSERT INTO persons (id, email, legal_first_name, legal_last_name, preferred_first_name, preferred_last_name, sex, dob, status)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					a.personID, fmt.Sprintf("%s@pupil.oakridge.sch.uk", a.pupilID), a.first, a.last, a.first, a.last, a.sex, a.dob, a.personStatus); err != nil {
					return err
				}
				if _, err := tx.Exec(`INSERT INTO person_roles (person_id, role_id, is_primary, effective_from, effective_to) VALUES (?, 'PUPIL', 1, ?, ?)`,
					a.personID, a.effFrom, a.effTo); err != nil {
					return err
				}
				if _, err := tx.Exec(`INSERT INTO pupils (id, person_id, upn, uln, nhs_number, admission_number, ethnicity_code, first_language_code, eal_flag, pupil_premium_flag, early_years_pp_flag, looked_after_flag)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					a.pupilID, a.personID, upn, uln, nhs, admNo, a.eth, a.lang, a.eal, a.pp, a.eyPP, a.lac); err != nil {
					return err
				}
				if _, err := tx.Exec(`INSERT INTO pupil_year_enrolments (id, pupil_id, academic_year_id, nc_year_group, enrolment_status, registration_group_id, house_id, roll_order, ey_funded_hours, ey_extended_funded_hours, ey_thirty_hour_code, ey_daf_indicator, leaving_reason_code, effective_from, effective_to)
					VALUES (?, ?, 'ay_2026_2027', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					"enr_"+a.pupilID, a.pupilID, form.NCYearGroup, a.enrStatus, form.GroupID, houseID, slot+1, a.eyFunded, a.eyExt, eyCode, a.daf, leaveReason, a.effFrom, a.effTo); err != nil {
					return err
				}

				if a.transferFromForm7A {
					// Temporal membership in 7A from Sept 1 to Sept 30, then 7B from Oct 1 onwards
					if _, err := tx.Exec(`INSERT INTO group_pupil_memberships (id, group_id, pupil_id, academic_year_id, effective_from, effective_to)
						VALUES ('gpm_liam_7a_hist', 'grp_form_7A', ?, 'ay_2026_2027', '2026-09-01', '2026-09-30')`, a.pupilID); err != nil {
						return err
					}
					if _, err := tx.Exec(`INSERT INTO group_pupil_memberships (id, group_id, pupil_id, academic_year_id, effective_from, effective_to)
						VALUES ('gpm_liam_7b_curr', 'grp_form_7B', ?, 'ay_2026_2027', '2026-10-01', NULL)`, a.pupilID); err != nil {
						return err
					}
				} else {
					if _, err := tx.Exec(`INSERT INTO group_pupil_memberships (id, group_id, pupil_id, academic_year_id, effective_from, effective_to)
						VALUES (?, ?, ?, 'ay_2026_2027', ?, ?)`,
						"gpm_form_"+a.pupilID, form.GroupID, a.pupilID, a.effFrom, a.effTo); err != nil {
						return err
					}
				}

				meta = &seededPupilMeta{
					PupilID:       a.pupilID,
					PersonID:      a.personID,
					FirstName:     a.first,
					LastName:      a.last,
					Sex:           a.sex,
					DOB:           a.dob,
					NCYearGroup:   form.NCYearGroup,
					Phase:         form.Phase,
					KeyStage:      form.KeyStage,
					FormGroupID:   form.GroupID,
					FormCode:      form.Code,
					HouseID:       houseID,
					Status:        a.personStatus,
					EffectiveFrom: a.effFrom,
					EffectiveTo:   a.effTo,
					IsFSM:         a.pp == 1,
					IsPP:          a.pp == 1,
					IsEAL:         a.eal == 1,
					IsLAC:         a.lac == 1,
					SENCode:       a.senCode,
				}
			} else {
				pupilID := fmt.Sprintf("pup_%04d", serial)
				personID := fmt.Sprintf("per_pup_%04d", serial)
				sex := "F"
				first := st.gen.pick(ukFemaleFirstNames)
				if st.gen.chance(0.50) {
					sex = "M"
					first = st.gen.pick(ukMaleFirstNames)
				}
				last := st.gen.pick(ukLastNames)
				by := birthYearForNC(form.NCYearGroup)
				dob := fmt.Sprintf("%04d-%02d-%02d", by, 1+st.gen.intN(12), 1+st.gen.intN(28))
				eth := st.gen.pick(dfeEthnicities)
				isEAL := st.gen.chance(0.20)
				lang := "ENG"
				ealInt := 0
				if isEAL {
					lang = st.gen.pick(dfeEALLanguages)
					ealInt = 1
				}
				isFSM := st.gen.chance(0.24)
				isPP := isFSM || st.gen.chance(0.05)
				ppInt := 0
				if isPP {
					ppInt = 1
				}
				isLAC := st.gen.chance(0.012)
				lacInt := 0
				if isLAC {
					lacInt = 1
				}
				svcFlag := "N"
				if st.gen.chance(0.02) {
					svcFlag = "Y"
				}
				senCode := "N"
				r := st.gen.rng.Float64()
				if r < 0.035 {
					senCode = "E"
				} else if r < 0.165 {
					senCode = "K"
				}

				eyFunded := 0.0
				eyExt := 0.0
				var eyCode any
				eyPP := 0
				if form.Phase == "NURSERY" {
					eyFunded = 15.0
					if st.gen.chance(0.60) {
						eyExt = 15.0
						eyCode = fmt.Sprintf("500%08d", serial)
					}
					if isPP {
						eyPP = 1
					}
				}

				if _, err := tx.Exec(`INSERT INTO persons (id, email, legal_first_name, legal_last_name, preferred_first_name, preferred_last_name, sex, dob, status)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'ACTIVE')`,
					personID, fmt.Sprintf("%s@pupil.oakridge.sch.uk", pupilID), first, last, first, last, sex, dob); err != nil {
					return err
				}
				if _, err := tx.Exec(`INSERT INTO person_roles (person_id, role_id, is_primary, effective_from) VALUES (?, 'PUPIL', 1, '2026-09-01')`,
					personID); err != nil {
					return err
				}
				if _, err := tx.Exec(`INSERT INTO pupils (id, person_id, upn, uln, nhs_number, admission_number, ethnicity_code, first_language_code, eal_flag, pupil_premium_flag, early_years_pp_flag, service_child_flag, looked_after_flag)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					pupilID, personID, upn, uln, nhs, admNo, eth, lang, ealInt, ppInt, eyPP, svcFlag, lacInt); err != nil {
					return err
				}
				if _, err := tx.Exec(`INSERT INTO pupil_year_enrolments (id, pupil_id, academic_year_id, nc_year_group, enrolment_status, registration_group_id, house_id, roll_order, ey_funded_hours, ey_extended_funded_hours, ey_thirty_hour_code, effective_from)
					VALUES (?, ?, 'ay_2026_2027', ?, 'C', ?, ?, ?, ?, ?, ?, '2026-09-01')`,
					"enr_"+pupilID, pupilID, form.NCYearGroup, form.GroupID, houseID, slot+1, eyFunded, eyExt, eyCode); err != nil {
					return err
				}
				if _, err := tx.Exec(`INSERT INTO group_pupil_memberships (id, group_id, pupil_id, academic_year_id, effective_from)
					VALUES (?, ?, ?, 'ay_2026_2027', '2026-09-01')`,
					"gpm_form_"+pupilID, form.GroupID, pupilID); err != nil {
					return err
				}

				meta = &seededPupilMeta{
					PupilID:       pupilID,
					PersonID:      personID,
					FirstName:     first,
					LastName:      last,
					Sex:           sex,
					DOB:           dob,
					NCYearGroup:   form.NCYearGroup,
					Phase:         form.Phase,
					KeyStage:      form.KeyStage,
					FormGroupID:   form.GroupID,
					FormCode:      form.Code,
					HouseID:       houseID,
					Status:        "ACTIVE",
					EffectiveFrom: "2026-09-01",
					IsFSM:         isFSM,
					IsPP:          isPP,
					IsEAL:         isEAL,
					IsLAC:         isLAC,
					SENCode:       senCode,
				}
			}

			st.pupils = append(st.pupils, meta)
			st.pupilByID[meta.PupilID] = meta
			form.PupilIDs = append(form.PupilIDs, meta.PupilID)
			serial++
		}
	}

	// 8. Seed Courses & Teaching Classes + Intervention Groups
	courses := []struct {
		id, deptID, code, name, dfeCode string
	}{
		{"crs_pri_core", "dept_primary", "PRI-CORE", "Primary Integrated Curriculum", "PRI01"},
		{"crs_ks3_mat", "dept_maths", "MAT-KS3", "Key Stage 3 Mathematics", "G100"},
		{"crs_ks3_eng", "dept_english", "ENG-KS3", "Key Stage 3 English Language & Literature", "Q300"},
		{"crs_ks3_sci", "dept_science", "SCI-KS3", "Key Stage 3 Combined Science", "F000"},
		{"crs_ks3_hum", "dept_humanities", "HUM-KS3", "Key Stage 3 History & Geography", "V100"},
		{"crs_ks3_mfl", "dept_mfl", "MFL-KS3", "Key Stage 3 Modern Languages", "R100"},
		{"crs_ks3_cmp", "dept_computing", "CMP-KS3", "Key Stage 3 Computer Science", "I100"},
		{"crs_ks4_mat", "dept_maths", "MAT-GCSE", "GCSE Mathematics (9-1)", "2210"},
		{"crs_ks4_eng", "dept_english", "ENG-GCSE", "GCSE English Language & Literature", "5030"},
		{"crs_ks4_sci", "dept_science", "SCI-GCSE", "GCSE Double Award Combined Science", "1370"},
		{"crs_ks4_his", "dept_humanities", "HIS-GCSE", "GCSE History", "4010"},
		{"crs_ks4_cmp", "dept_computing", "CMP-GCSE", "GCSE Computer Science", "2610"},
		{"crs_ks5_mat", "dept_maths", "MAT-ALEV", "A-Level Mathematics", "2330"},
		{"crs_ks5_fma", "dept_maths", "FMA-ALEV", "A-Level Further Mathematics", "2335"},
		{"crs_ks5_phy", "dept_science", "PHY-ALEV", "A-Level Physics", "1210"},
		{"crs_ks5_eng", "dept_english", "ENG-ALEV", "A-Level English Literature", "5010"},
	}
	for _, c := range courses {
		if _, err := tx.Exec(`INSERT INTO courses (id, academic_year_id, department_id, code, name, dfe_subject_code) VALUES (?, 'ay_2026_2027', ?, ?, ?, ?)`,
			c.id, c.deptID, c.code, c.name, c.dfeCode); err != nil {
			return err
		}
	}

	// Create Teaching Classes per Secondary/Sixth Form Form & Primary Classes
	classCounter := 0
	for _, form := range st.forms {
		switch form.KeyStage {
		case "EYFS", "KS1", "KS2":
			grpID := "grp_class_" + form.Code + "_core"
			code := form.Code + "-CORE"
			if _, err := tx.Exec(`INSERT INTO groups (id, academic_year_id, group_type, code, name, year_group_id, course_id, default_room_id, takes_attendance, effective_from)
				VALUES (?, 'ay_2026_2027', 'TEACHING_CLASS', ?, ?, ?, 'crs_pri_core', ?, 1, '2026-09-01')`,
				grpID, code, form.Code+" Primary Class", form.YearGroupID, form.RoomID); err != nil {
				return err
			}
			if _, err := tx.Exec(`INSERT INTO group_staff_assignments (id, group_id, staff_id, role, academic_year_id, effective_from)
				VALUES (?, ?, ?, 'LEAD_TEACHER', 'ay_2026_2027', '2026-09-01')`,
				"gsa_cls_"+code, grpID, form.TutorID); err != nil {
				return err
			}
			for _, pid := range form.PupilIDs {
				if _, err := tx.Exec(`INSERT INTO group_pupil_memberships (id, group_id, pupil_id, academic_year_id, effective_from)
					VALUES (?, ?, ?, 'ay_2026_2027', '2026-09-01')`,
					fmt.Sprintf("gpm_%s_%s", code, pid), grpID, pid); err != nil {
					return err
				}
			}
			st.teachingGroupIDs = append(st.teachingGroupIDs, grpID)

		case "KS3":
			ks3Subs := []struct{ suf, crs, name string }{
				{"Ma", "crs_ks3_mat", "Mathematics"},
				{"En", "crs_ks3_eng", "English"},
				{"Sc", "crs_ks3_sci", "Science"},
				{"Hu", "crs_ks3_hum", "Humanities"},
			}
			for _, sub := range ks3Subs {
				code := form.Code + sub.suf
				grpID := "grp_class_" + code
				roomID := st.roomIDs[(classCounter+10)%len(st.roomIDs)]
				tchID := st.teacherStaffIDs[(classCounter+4)%len(st.teacherStaffIDs)]
				if code == "7AMa" {
					tchID = "stf_tch_7a"
				} else if code == "7BEn" {
					tchID = "stf_tch_7b"
				}
				if _, err := tx.Exec(`INSERT INTO groups (id, academic_year_id, group_type, code, name, year_group_id, course_id, default_room_id, takes_attendance, effective_from)
					VALUES (?, 'ay_2026_2027', 'TEACHING_CLASS', ?, ?, ?, ?, ?, 1, '2026-09-01')`,
					grpID, code, form.Code+" "+sub.name, form.YearGroupID, sub.crs, roomID); err != nil {
					return err
				}
				if _, err := tx.Exec(`INSERT INTO group_staff_assignments (id, group_id, staff_id, role, academic_year_id, effective_from)
					VALUES (?, ?, ?, 'LEAD_TEACHER', 'ay_2026_2027', '2026-09-01')`,
					"gsa_cls_"+code, grpID, tchID); err != nil {
					return err
				}
				for _, pid := range form.PupilIDs {
					if _, err := tx.Exec(`INSERT INTO group_pupil_memberships (id, group_id, pupil_id, academic_year_id, effective_from)
						VALUES (?, ?, ?, 'ay_2026_2027', '2026-09-01')`,
						fmt.Sprintf("gpm_%s_%s", code, pid), grpID, pid); err != nil {
						return err
					}
				}
				st.teachingGroupIDs = append(st.teachingGroupIDs, grpID)
				classCounter++
			}

		case "KS4":
			ks4Subs := []struct{ suf, crs, name string }{
				{"Ma", "crs_ks4_mat", "GCSE Mathematics"},
				{"En", "crs_ks4_eng", "GCSE English"},
				{"Sc", "crs_ks4_sci", "GCSE Science"},
			}
			for _, sub := range ks4Subs {
				code := form.Code + sub.suf
				grpID := "grp_class_" + code
				roomID := st.roomIDs[(classCounter+20)%len(st.roomIDs)]
				tchID := st.teacherStaffIDs[(classCounter+8)%len(st.teacherStaffIDs)]
				if code == "10AMa" {
					tchID = "stf_hod_maths"
				}
				if _, err := tx.Exec(`INSERT INTO groups (id, academic_year_id, group_type, code, name, year_group_id, course_id, default_room_id, takes_attendance, effective_from)
					VALUES (?, 'ay_2026_2027', 'TEACHING_CLASS', ?, ?, ?, ?, ?, 1, '2026-09-01')`,
					grpID, code, form.Code+" "+sub.name, form.YearGroupID, sub.crs, roomID); err != nil {
					return err
				}
				if _, err := tx.Exec(`INSERT INTO group_staff_assignments (id, group_id, staff_id, role, academic_year_id, effective_from)
					VALUES (?, ?, ?, 'LEAD_TEACHER', 'ay_2026_2027', '2026-09-01')`,
					"gsa_cls_"+code, grpID, tchID); err != nil {
					return err
				}
				for _, pid := range form.PupilIDs {
					p := st.pupilByID[pid]
					if _, err := tx.Exec(`INSERT INTO group_pupil_memberships (id, group_id, pupil_id, academic_year_id, effective_from, effective_to)
						VALUES (?, ?, ?, 'ay_2026_2027', '2026-09-01', ?)`,
						fmt.Sprintf("gpm_%s_%s", code, pid), grpID, pid, p.EffectiveTo); err != nil {
						return err
					}
				}
				st.teachingGroupIDs = append(st.teachingGroupIDs, grpID)
				classCounter++
			}

		case "KS5":
			code := form.Code + "Ma"
			grpID := "grp_class_" + code
			roomID := st.roomIDs[(classCounter+30)%len(st.roomIDs)]
			tchID := "stf_hod_maths"
			if _, err := tx.Exec(`INSERT INTO groups (id, academic_year_id, group_type, code, name, year_group_id, course_id, default_room_id, takes_attendance, effective_from)
				VALUES (?, 'ay_2026_2027', 'TEACHING_CLASS', ?, ?, ?, 'crs_ks5_mat', ?, 1, '2026-09-01')`,
				grpID, code, form.Code+" A-Level Mathematics", form.YearGroupID, roomID); err != nil {
				return err
			}
			if _, err := tx.Exec(`INSERT INTO group_staff_assignments (id, group_id, staff_id, role, academic_year_id, effective_from)
				VALUES (?, ?, ?, 'LEAD_TEACHER', 'ay_2026_2027', '2026-09-01')`,
				"gsa_cls_"+code, grpID, tchID); err != nil {
				return err
			}
			for _, pid := range form.PupilIDs {
				if _, err := tx.Exec(`INSERT INTO group_pupil_memberships (id, group_id, pupil_id, academic_year_id, effective_from)
					VALUES (?, ?, ?, 'ay_2026_2027', '2026-09-01')`,
					fmt.Sprintf("gpm_%s_%s", code, pid), grpID, pid); err != nil {
					return err
				}
			}
			st.teachingGroupIDs = append(st.teachingGroupIDs, grpID)
			classCounter++
		}
	}

	// Seed SEN Intervention Group (`grp_int_lit7`)
	if _, err := tx.Exec(`INSERT INTO groups (id, academic_year_id, group_type, code, name, year_group_id, default_room_id, takes_attendance, effective_from)
		VALUES ('grp_int_lit7', 'ay_2026_2027', 'INTERVENTION', 'INT-LIT-7', 'Year 7 Literacy & Social Communication Intervention', 'yg_7', 'rm_001', 1, '2026-09-01')`); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO group_staff_assignments (id, group_id, staff_id, role, academic_year_id, effective_from)
		VALUES ('gsa_int_lit7_ta', 'grp_int_lit7', 'stf_ta_01', 'TEACHING_ASSISTANT', 'ay_2026_2027', '2026-09-01')`); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO group_pupil_memberships (id, group_id, pupil_id, academic_year_id, effective_from)
		VALUES ('gpm_int_noah', 'grp_int_lit7', 'pup_yr7_noah', 'ay_2026_2027', '2026-09-01')`); err != nil {
		return err
	}

	// 9. Seed 2-Week Rolling Timetable Periods (`timetable_periods`: 90 periods across Week 1 & Week 2, Mon-Fri)
	periodTemplate := []struct {
		code, name, start, end, pType string
		statSession                   any
		seq                           int
	}{
		{"AM_REG", "Morning Registration & Tutor Time", "08:45", "09:05", "STATUTORY_AM_REG", "AM", 1},
		{"P1", "Period 1", "09:05", "10:05", "LESSON", nil, 2},
		{"P2", "Period 2", "10:05", "11:05", "LESSON", nil, 3},
		{"BREAK", "Morning Break", "11:05", "11:25", "BREAK", nil, 4},
		{"P3", "Period 3", "11:25", "12:25", "LESSON", nil, 5},
		{"LUNCH", "Lunch", "12:25", "13:25", "BREAK", nil, 6},
		{"PM_REG", "Afternoon Statutory Roll Call", "13:25", "13:35", "STATUTORY_PM_REG", "PM", 7},
		{"P4", "Period 4", "13:35", "14:35", "LESSON", nil, 8},
		{"P5", "Period 5", "14:35", "15:35", "LESSON", nil, 9},
	}

	var lessonPeriodIDs []string
	var amRegPeriodIDs []string
	for wk := 1; wk <= 2; wk++ {
		for day := 1; day <= 5; day++ {
			for _, pt := range periodTemplate {
				pid := fmt.Sprintf("ttp_w%d_d%d_%s", wk, day, pt.code)
				if _, err := tx.Exec(`INSERT INTO timetable_periods (id, academic_year_id, cycle_week, day_of_week_id, period_code, name, start_time, end_time, period_type, maps_to_statutory_session, sequence_number)
					VALUES (?, 'ay_2026_2027', ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					pid, wk, day, pt.code, pt.name, pt.start, pt.end, pt.pType, pt.statSession, pt.seq); err != nil {
					return err
				}
				if pt.pType == "LESSON" && wk == 1 {
					lessonPeriodIDs = append(lessonPeriodIDs, pid)
				}
				if pt.pType == "STATUTORY_AM_REG" && wk == 1 {
					amRegPeriodIDs = append(amRegPeriodIDs, pid)
				}
			}
		}
	}

	// Seed Timetable Slots (`timetable_slots`) for Registration Forms and Teaching Classes
	slotCounter := 0
	for _, form := range st.forms {
		for _, amPID := range amRegPeriodIDs {
			slotCounter++
			slotID := fmt.Sprintf("tts_reg_%04d", slotCounter)
			if _, err := tx.Exec(`INSERT INTO timetable_slots (id, timetable_period_id, group_id, staff_id, room_id, effective_from)
				VALUES (?, ?, ?, ?, ?, '2026-09-01')`,
				slotID, amPID, form.GroupID, form.TutorID, form.RoomID); err != nil {
				return err
			}
		}
	}

	for idx, tgID := range st.teachingGroupIDs {
		for p := 0; p < 4; p++ {
			periodID := lessonPeriodIDs[(idx*4+p)%len(lessonPeriodIDs)]
			slotCounter++
			slotID := fmt.Sprintf("tts_cls_%04d", slotCounter)
			tchID := st.teacherStaffIDs[(idx+p)%len(st.teacherStaffIDs)]
			roomID := st.roomIDs[(idx+p)%len(st.roomIDs)]
			if _, err := tx.Exec(`INSERT INTO timetable_slots (id, timetable_period_id, group_id, staff_id, room_id, effective_from)
				VALUES (?, ?, ?, ?, ?, '2026-09-01')`,
				slotID, periodID, tgID, tchID, roomID); err != nil {
				return err
			}
			st.slotIDsByGroup[tgID] = append(st.slotIDsByGroup[tgID], slotID)
			st.allSlotIDs = append(st.allSlotIDs, slotID)
		}
	}

	// 10. Seed Staff Absences & Cover Assignments (`staff_absences`, `staff_cover_assignments`)
	absCats := []string{"SIC", "TRN", "OTH", "MAT"}
	for i := 1; i <= 20; i++ {
		absID := fmt.Sprintf("sabs_%03d", i)
		absentStaffID := st.teacherStaffIDs[(i+2)%len(st.teacherStaffIDs)]
		if i == 1 {
			absentStaffID = "stf_tch_7b"
		}
		cat := absCats[(i-1)%len(absCats)]
		dayDate := autumn2026SchoolDays[(i-1)%len(autumn2026SchoolDays)]
		if _, err := tx.Exec(`INSERT INTO staff_absences (id, staff_id, academic_year_id, absence_category_code, reason_detail, confidential_notes, status, cover_required, start_date, end_date, working_days_lost, approved_by_staff_id)
			VALUES (?, ?, 'ay_2026_2027', ?, 'Scheduled or reported staff absence', 'Logged by HR Officer', 'APPROVED', 1, ?, ?, 1.0, 'stf_admin_01')`,
			absID, absentStaffID, cat, dayDate, dayDate); err != nil {
			return err
		}

		for c := 1; c <= 2; c++ {
			covID := fmt.Sprintf("scov_%03d_%d", i, c)
			slotID := st.allSlotIDs[(i*2+c)%len(st.allSlotIDs)]
			coveringID := "stf_supply_01"
			if c == 2 {
				coveringID = st.teacherStaffIDs[(i+5)%len(st.teacherStaffIDs)]
			}
			if _, err := tx.Exec(`INSERT INTO staff_cover_assignments (id, staff_absence_id, timetable_slot_id, date, absent_staff_id, covering_staff_id, status, cover_notes)
				VALUES (?, ?, ?, ?, ?, ?, 'ACCEPTED', 'Cover work set on virtual learning workspace')`,
				covID, absID, slotID, dayDate, absentStaffID, coveringID); err != nil {
				return err
			}
		}
	}

	return nil
}
