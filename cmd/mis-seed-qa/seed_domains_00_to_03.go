package main

import (
	"database/sql"
	"fmt"
)

type seededPupilMeta struct {
	PupilID       string
	PersonID      string
	FirstName     string
	LastName      string
	Sex           string
	DOB           string
	NCYearGroup   string
	Phase         string
	KeyStage      string
	FormGroupID   string
	FormCode      string
	HouseID       string
	Status        string
	EffectiveFrom string
	EffectiveTo   *string
	IsFSM         bool
	IsPP          bool
	IsEAL         bool
	IsLAC         bool
	SENCode       string // "N", "K", "E"
}

type seededFormMeta struct {
	GroupID     string
	Code        string
	NCYearGroup string
	YearGroupID string
	Phase       string
	KeyStage    string
	TutorID     string
	RoomID      string
	PupilIDs    []string
}

type qaSeedState struct {
	schoolURN        string
	kmsKeyHandle     string
	gen              *detGen
	staffIDs         []string
	teacherStaffIDs  []string
	taStaffIDs       []string
	roomIDs          []string
	houseIDs         []string
	forms            []*seededFormMeta
	pupils           []*seededPupilMeta
	pupilByID        map[string]*seededPupilMeta
	teachingGroupIDs []string
	slotIDsByGroup   map[string][]string
	allSlotIDs       []string
}

func seedDomain00SystemAndCalendar(tx *sql.Tx, st *qaSeedState) error {
	configs := [][2]string{
		{"school_urn", st.schoolURN},
		{"school_name", "Oakridge All-Through Academy"},
		{"school_phase", "ALL_THROUGH"},
		{"dfe_la_code", "931"},
		{"dfe_estab_number", "4001"},
		{"academic_year_current_id", "ay_2026_2027"},
		{"core_dal_version", "1.4.0"},
	}
	for _, kv := range configs {
		if _, err := tx.Exec(`INSERT INTO __system_config (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`, kv[0], kv[1]); err != nil {
			return fmt.Errorf("__system_config insert failed: %w", err)
		}
	}

	years := []struct {
		id, name, status, first, last string
		seq                           int
	}{
		{"ay_2025_2026", "2025/2026", "PAST", "2025-09-01", "2026-07-24", 1},
		{"ay_2026_2027", "2026/2027", "CURRENT", "2026-09-01", "2027-07-23", 2},
		{"ay_2027_2028", "2027/2028", "UPCOMING", "2027-09-01", "2028-07-21", 3},
	}
	for _, y := range years {
		if _, err := tx.Exec(`INSERT INTO academic_years (id, name, status, sequence_number, first_day, last_day) VALUES (?, ?, ?, ?, ?, ?)`,
			y.id, y.name, y.status, y.seq, y.first, y.last); err != nil {
			return fmt.Errorf("academic_years insert failed: %w", err)
		}
	}

	terms := []struct {
		id, yearID, code, name, short, first, last string
		seq                                        int
	}{
		{"trm_2526_aut", "ay_2025_2026", "AUTUMN", "Autumn Term 2025", "Aut 25", "2025-09-01", "2025-12-19", 1},
		{"trm_2526_spr", "ay_2025_2026", "SPRING", "Spring Term 2026", "Spr 26", "2026-01-05", "2026-03-27", 2},
		{"trm_2526_sum", "ay_2025_2026", "SUMMER", "Summer Term 2026", "Sum 26", "2026-04-13", "2026-07-24", 3},
		{"trm_2627_aut", "ay_2026_2027", "AUTUMN", "Autumn Term 2026", "Aut 26", "2026-09-01", "2026-12-18", 1},
		{"trm_2627_spr", "ay_2026_2027", "SPRING", "Spring Term 2027", "Spr 27", "2027-01-04", "2027-03-25", 2},
		{"trm_2627_sum", "ay_2026_2027", "SUMMER", "Summer Term 2027", "Sum 27", "2027-04-12", "2027-07-23", 3},
		{"trm_2728_aut", "ay_2027_2028", "AUTUMN", "Autumn Term 2027", "Aut 27", "2027-09-01", "2027-12-17", 1},
		{"trm_2728_spr", "ay_2027_2028", "SPRING", "Spring Term 2028", "Spr 28", "2028-01-04", "2028-03-31", 2},
		{"trm_2728_sum", "ay_2027_2028", "SUMMER", "Summer Term 2028", "Sum 28", "2028-04-17", "2028-07-21", 3},
	}
	for _, t := range terms {
		if _, err := tx.Exec(`INSERT INTO academic_terms (id, academic_year_id, code, name, name_short, sequence_number, first_day, last_day) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			t.id, t.yearID, t.code, t.name, t.short, t.seq, t.first, t.last); err != nil {
			return fmt.Errorf("academic_terms insert failed: %w", err)
		}
	}

	days := []struct {
		id                        int
		name, short               string
		isSchool                  int
		open, amReg, pmReg, close any
	}{
		{1, "Monday", "Mon", 1, "08:00", "08:45", "13:30", "16:30"},
		{2, "Tuesday", "Tue", 1, "08:00", "08:45", "13:30", "16:30"},
		{3, "Wednesday", "Wed", 1, "08:00", "08:45", "13:30", "16:30"},
		{4, "Thursday", "Thu", 1, "08:00", "08:45", "13:30", "16:30"},
		{5, "Friday", "Fri", 1, "08:00", "08:45", "13:30", "16:00"},
		{6, "Saturday", "Sat", 0, nil, nil, nil, nil},
		{7, "Sunday", "Sun", 0, nil, nil, nil, nil},
	}
	for _, d := range days {
		if _, err := tx.Exec(`INSERT INTO days_of_week (id, name, name_short, is_school_day, school_open_time, am_register_time, pm_register_time, school_close_time) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			d.id, d.name, d.short, d.isSchool, d.open, d.amReg, d.pmReg, d.close); err != nil {
			return fmt.Errorf("days_of_week insert failed: %w", err)
		}
	}

	specialDays := []struct {
		id, termID, date, typ, name, desc string
	}{
		{"spd_inset_1", "trm_2627_aut", "2026-09-01", "INSET_DAY", "Autumn Staff INSET Day 1", "Whole-school statutory safeguarding and curriculum training"},
		{"spd_open_eve", "trm_2627_aut", "2026-10-08", "TIMING_CHANGE", "Year 6 Open Evening Early Finish", "Early school close at 14:15 for prospective family tours"},
		{"spd_inset_2", "trm_2627_aut", "2026-11-02", "INSET_DAY", "Autumn Assessment Moderation INSET", "Cross-MAT assessment moderation"},
		{"spd_y11_mocks", "trm_2627_aut", "2026-11-16", "OFF_TIMETABLE", "Year 11 GCSE Mock Immersion Day", "Core Maths & English exam preparation"},
		{"spd_spr_inset", "trm_2627_spr", "2027-01-04", "INSET_DAY", "Spring Term Staff INSET", "Departmental planning day"},
		{"spd_may_bank", "trm_2627_sum", "2027-05-03", "SCHOOL_CLOSURE", "Early May Bank Holiday", "Statutory bank holiday closure (DfE Code #)"},
	}
	for _, s := range specialDays {
		if _, err := tx.Exec(`INSERT INTO academic_special_days (id, academic_term_id, date, type, name, description) VALUES (?, ?, ?, ?, ?, ?)`,
			s.id, s.termID, s.date, s.typ, s.name, s.desc); err != nil {
			return fmt.Errorf("academic_special_days insert failed: %w", err)
		}
	}

	alerts := []struct {
		id, name, short, fg, bg, desc string
		seq                           int
	}{
		{"LOW", "Low Alert", "LOW", "#1E4620", "#E6F4EA", "Informational classroom support note", 1},
		{"MEDIUM", "Medium Alert", "MED", "#7A4F01", "#FEF7E0", "Moderate pastoral, SEN, or dietary awareness required", 2},
		{"HIGH", "High Alert", "HIGH", "#A50E0E", "#FCE8E6", "High-priority SEN EHCP or medical management plan", 3},
		{"CRITICAL", "Critical Life-Safety / Statutory Alert", "CRIT", "#FFFFFF", "#B31412", "Immediate life-safety medical condition (IHCP/Anaphylaxis) or statutory restriction", 4},
	}
	for _, a := range alerts {
		if _, err := tx.Exec(`INSERT INTO alert_levels (id, name, name_short, color_hex, bg_color_hex, description, sequence_number) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			a.id, a.name, a.short, a.fg, a.bg, a.desc, a.seq); err != nil {
			return fmt.Errorf("alert_levels insert failed: %w", err)
		}
	}

	retentions := []struct {
		key, desc, rule, basis string
		years                  int
	}{
		{"SAFEGUARDING", "Child Protection & KCSIE Vault Records", "DOB_PLUS_25_YEARS", "KCSIE & Limitation Act 1980", 25},
		{"SEN", "Special Educational Needs & EHCP Records", "DOB_PLUS_31_YEARS", "SEND Code of Practice & Children and Families Act 2014", 31},
		{"PUPIL_ADMISSIONS", "Statutory Admission Register & Core Pupil File", "DOB_PLUS_25_YEARS", "Education (Pupil Registration) (England) Regulations", 25},
		{"ATTENDANCE", "Statutory AM/PM Attendance Registers", "ENTRY_PLUS_3_YEARS", "School Attendance (Pupil Registration) (England) Regulations 2024", 3},
		{"STAFF_SCR", "Single Central Record & Workforce HR Files", "TERMINATION_PLUS_6_YEARS", "KCSIE Single Central Record Guidance", 6},
	}
	for _, r := range retentions {
		if _, err := tx.Exec(`INSERT INTO data_retention_policies (domain_key, description, retention_rule, retention_years, statutory_basis) VALUES (?, ?, ?, ?, ?)`,
			r.key, r.desc, r.rule, r.years, r.basis); err != nil {
			return fmt.Errorf("data_retention_policies insert failed: %w", err)
		}
	}

	return nil
}

func seedDomain01And03StaffGroupsAndPupils(tx *sql.Tx, st *qaSeedState) error {
	// 1. Seed Roles
	roles := []struct {
		id, cat, name, short, desc, scopes string
	}{
		{"ADMIN", "STAFF", "System Administrator / Headteacher", "ADM", "Full school administrative access", `["dal.pupils.read:all","dal.pupils.write:all","dal.staff.read:admin","dal.staff.write:admin","dal.groups.read:all","dal.groups.write:all","dal.attendance.read:all","dal.attendance.write:all","dal.guardians.read:all","dal.guardians.write:all","dal.sen.read:all","dal.medical.read:all"]`},
		{"DSL", "STAFF", "Designated Safeguarding Lead", "DSL", "KCSIE Safeguarding Vault read & triage authority", `["dal.safeguarding.create","dal.safeguarding.read","dal.pupils.read:all","dal.guardians.read:all","dal.attendance.read:all","dal.sen.read:all","dal.medical.read:all"]`},
		{"SENCO", "STAFF", "SEN Coordinator", "SENCO", "Whole-school SEND and EHCP coordination", `["dal.sen.read:all","dal.sen.write:all","dal.pupils.read:all","dal.medical.read:all","dal.groups.read:all","dal.safeguarding.create"]`},
		{"HOD", "STAFF", "Head of Department", "HOD", "Departmental curriculum and cohort lead", `["dal.pupils.read:all","dal.groups.read:all","dal.groups.write:all","dal.attendance.read:all","dal.guardians.read:all","dal.sen.read:all","dal.medical.read:all","dal.safeguarding.create"]`},
		{"TEACHER", "STAFF", "Classroom Teacher", "TCH", "Scoped access to assigned teaching and registration groups", `["dal.pupils.read:own_groups","dal.groups.read:own_groups","dal.attendance.read:own_groups","dal.attendance.write:own_groups","dal.guardians.read:own_groups","dal.sen.read:own_groups","dal.medical.read:own_groups","dal.safeguarding.create"]`},
		{"TA", "STAFF", "Teaching Assistant / HLTA", "TA", "Classroom and SEN intervention support", `["dal.pupils.read:own_groups","dal.groups.read:own_groups","dal.sen.read:own_groups","dal.medical.read:own_groups","dal.safeguarding.create"]`},
		{"SUPPORT", "STAFF", "School Office & Pastoral Support", "SUP", "Attendance officer, First Aider, and administrative support", `["dal.pupils.read:all","dal.attendance.read:all","dal.attendance.write:all","dal.guardians.read:all","dal.medical.read:all","dal.safeguarding.create"]`},
		{"PUPIL", "PUPIL", "Enrolled School Pupil", "PUP", "Student identity role", `[]`},
		{"GUARDIAN", "GUARDIAN", "Parent / Legal Guardian", "GDN", "Household guardian role", `[]`},
		{"SYSTEM", "SYSTEM", "Host Service Principal", "SYS", "Background worker execution role", `[]`},
	}
	for _, r := range roles {
		if _, err := tx.Exec(`INSERT INTO roles (id, category, name, name_short, description, default_scopes_json) VALUES (?, ?, ?, ?, ?, ?)`,
			r.id, r.cat, r.name, r.short, r.desc, r.scopes); err != nil {
			return fmt.Errorf("roles insert failed: %w", err)
		}
	}

	// 2. Seed Pinned Anchor Staff Personas + 87 PRNG Staff (Total = 95 Staff)
	type anchorStaffSpec struct {
		staffID, personID, code, title, first, last, sex, dob, email, ssoSub string
		staffType, jobTitle, roleID, swcPost, payScale, spinePoint           string
		qts, hlta, isDSL, firstAid                                           int
		fte                                                                  float64
	}
	anchorStaff := []anchorStaffSpec{
		{"stf_admin_01", "per_stf_admin_01", "HDT", "Dame", "Eleanor", "Vance", "F", "1974-03-14", "e.vance@oakridge.sch.uk", "sso_admin_01", "LEADERSHIP", "Executive Headteacher", "ADMIN", "EHT", "L", "L28", 1, 0, 1, 0, 1.0},
		{"stf_dsl_01", "per_stf_dsl_01", "DSL", "Mr", "Marcus", "Thorne", "M", "1980-11-02", "m.thorne@oakridge.sch.uk", "sso_dsl_01", "LEADERSHIP", "Deputy Head & Designated Safeguarding Lead", "DSL", "DHD", "L", "L18", 1, 0, 1, 1, 1.0},
		{"stf_senco_01", "per_stf_senco_01", "SNC", "Dr", "Priya", "Nair", "F", "1983-06-21", "p.nair@oakridge.sch.uk", "sso_senco_01", "LEADERSHIP", "Whole-School SENCO", "SENCO", "AHD", "L", "L12", 1, 0, 1, 1, 1.0},
		{"stf_hod_maths", "per_stf_hod_maths", "HMA", "Mr", "David", "Sterling", "M", "1985-09-10", "d.sterling@oakridge.sch.uk", "sso_hod_maths", "TEACHING", "Head of Mathematics", "HOD", "TCH", "UPS", "U3", 1, 0, 0, 0, 1.0},
		{"stf_tch_7a", "per_stf_tch_7a", "T7A", "Ms", "Hannah", "Abbott", "F", "1992-04-18", "h.abbott@oakridge.sch.uk", "sso_tch_7a", "TEACHING", "Teacher of Mathematics & 7A Form Tutor", "TEACHER", "TCH", "MPS", "M5", 1, 0, 0, 1, 1.0},
		{"stf_tch_7b", "per_stf_tch_7b", "T7B", "Mr", "Simon", "Barker", "M", "1990-01-25", "s.barker@oakridge.sch.uk", "sso_tch_7b", "TEACHING", "Teacher of English & 7B Form Tutor", "TEACHER", "TCH", "MPS", "M6", 1, 0, 0, 0, 1.0},
		{"stf_ta_01", "per_stf_ta_01", "TA1", "Mrs", "Sarah", "Jenkins", "F", "1988-07-30", "s.jenkins@oakridge.sch.uk", "sso_ta_01", "SUPPORT", "Higher Level Teaching Assistant (SEND)", "TA", "SUP", "SUPPORT", "S14", 0, 1, 0, 1, 1.0},
		{"stf_supply_01", "per_stf_supply_01", "SUP1", "Mr", "Gareth", "Cole", "M", "1986-12-09", "g.cole@supply.oakridge.sch.uk", "sso_supply_01", "SUPPLY_AGENCY", "Agency Cover Teacher", "TEACHER", "TCH", "MPS", "M3", 1, 0, 0, 0, 0.8},
	}

	for idx, a := range anchorStaff {
		trn := fmt.Sprintf("70%05d", 1000+idx)
		dbsNo := fmt.Sprintf("00189230%04d", idx+1)
		if _, err := tx.Exec(`INSERT INTO persons (id, external_sso_sub, email, title, legal_first_name, legal_last_name, preferred_first_name, preferred_last_name, sex, dob, phone_primary, phone_primary_type, status)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'MOBILE', 'ACTIVE')`,
			a.personID, a.ssoSub, a.email, a.title, a.first, a.last, a.first, a.last, a.sex, a.dob, fmt.Sprintf("07700900%03d", idx+1)); err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT INTO person_roles (person_id, role_id, is_primary, effective_from) VALUES (?, ?, 1, '2022-09-01')`,
			a.personID, a.roleID); err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT INTO staff (id, person_id, staff_code, trn, ni_number_encrypted, staff_type, job_title, qts_flag, eyts_flag, hlta_flag, is_dsl, first_aid_qualified, first_aid_expiry_date, dbs_check_date, dbs_certificate_number, prohibition_check_date, right_to_work_check_date, exclude_from_cover, cover_priority, hire_date)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0, ?, ?, ?, '2028-09-01', '2022-08-15', ?, '2022-08-15', '2022-08-15', ?, ?, '2022-09-01')`,
			a.staffID, a.personID, a.code, trn, "ENC_NI_"+a.staffID, a.staffType, a.jobTitle, a.qts, a.hlta, a.isDSL, a.firstAid, dbsNo, a.isDSL, idx+1); err != nil {
			return err
		}
		contractType := "PERMANENT"
		if a.staffType == "SUPPLY_AGENCY" {
			contractType = "AGENCY"
		}
		if _, err := tx.Exec(`INSERT INTO staff_contracts (id, staff_id, contract_type, swc_post_code, job_title, pay_scale, pay_spine_point, fte, hours_per_week, weeks_per_year, effective_from)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, 32.5, 52.0, '2022-09-01')`,
			"cnt_"+a.staffID, a.staffID, contractType, a.swcPost, a.jobTitle, a.payScale, a.spinePoint, a.fte); err != nil {
			return err
		}
		st.staffIDs = append(st.staffIDs, a.staffID)
		if a.staffType == "TEACHING" || a.staffType == "LEADERSHIP" {
			st.teacherStaffIDs = append(st.teacherStaffIDs, a.staffID)
		}
		if a.hlta == 1 {
			st.taStaffIDs = append(st.taStaffIDs, a.staffID)
		}
	}

	// Generate 87 additional staff (total 95 staff)
	for i := 9; i <= 95; i++ {
		staffID := fmt.Sprintf("stf_%03d", i)
		personID := fmt.Sprintf("per_stf_%03d", i)
		code := fmt.Sprintf("S%02d", i)
		sex := "F"
		first := st.gen.pick(ukFemaleFirstNames)
		title := "Ms"
		if st.gen.chance(0.45) {
			sex = "M"
			first = st.gen.pick(ukMaleFirstNames)
			title = "Mr"
		}
		last := st.gen.pick(ukLastNames)
		dob := fmt.Sprintf("%04d-%02d-%02d", 1968+st.gen.intN(32), 1+st.gen.intN(12), 1+st.gen.intN(28))
		email := fmt.Sprintf("%s.%s.%d@oakridge.sch.uk", first, last, i)
		ssoSub := fmt.Sprintf("sso_stf_%03d", i)

		staffType := "TEACHING"
		roleID := "TEACHER"
		jobTitle := "Classroom Teacher"
		swcPost := "TCH"
		payScale := "MPS"
		spinePoint := fmt.Sprintf("M%d", 1+st.gen.intN(6))
		qts := 1
		hlta := 0
		if i > 72 && i <= 86 {
			staffType = "SUPPORT"
			roleID = "TA"
			jobTitle = "SEND Teaching Assistant"
			swcPost = "SUP"
			payScale = "SUPPORT"
			spinePoint = "S10"
			qts = 0
			hlta = 1
		} else if i > 86 {
			staffType = "SUPPORT"
			roleID = "SUPPORT"
			jobTitle = "Pastoral & Attendance Officer"
			swcPost = "SUP"
			payScale = "SUPPORT"
			spinePoint = "S12"
			qts = 0
		}

		if _, err := tx.Exec(`INSERT INTO persons (id, external_sso_sub, email, title, legal_first_name, legal_last_name, preferred_first_name, preferred_last_name, sex, dob, phone_primary, phone_primary_type, status)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'MOBILE', 'ACTIVE')`,
			personID, ssoSub, email, title, first, last, first, last, sex, dob, fmt.Sprintf("07700910%03d", i)); err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT INTO person_roles (person_id, role_id, is_primary, effective_from) VALUES (?, ?, 1, '2023-09-01')`,
			personID, roleID); err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT INTO staff (id, person_id, staff_code, trn, ni_number_encrypted, staff_type, job_title, qts_flag, eyts_flag, hlta_flag, is_dsl, first_aid_qualified, dbs_check_date, dbs_certificate_number, prohibition_check_date, right_to_work_check_date, exclude_from_cover, cover_priority, hire_date)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0, ?, 0, 1, '2023-08-10', ?, '2023-08-10', '2023-08-10', 0, 5, '2023-09-01')`,
			staffID, personID, code, fmt.Sprintf("71%05d", i), "ENC_NI_"+staffID, staffType, jobTitle, qts, hlta, fmt.Sprintf("00189240%04d", i)); err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT INTO staff_contracts (id, staff_id, contract_type, swc_post_code, job_title, pay_scale, pay_spine_point, fte, hours_per_week, weeks_per_year, effective_from)
			VALUES (?, ?, 'PERMANENT', ?, ?, ?, ?, 1.0, 32.5, 52.0, '2023-09-01')`,
			"cnt_"+staffID, staffID, swcPost, jobTitle, payScale, spinePoint); err != nil {
			return err
		}
		st.staffIDs = append(st.staffIDs, staffID)
		if staffType == "TEACHING" {
			st.teacherStaffIDs = append(st.teacherStaffIDs, staffID)
		}
		if hlta == 1 {
			st.taStaffIDs = append(st.taStaffIDs, staffID)
		}
	}

	// 3. Seed Houses (4 Houses)
	houses := []struct {
		id, name, short, hex string
	}{
		{"hse_turing", "Turing House", "TUR", "#1A73E8"},
		{"hse_lovelace", "Lovelace House", "LOV", "#D93025"},
		{"hse_franklin", "Franklin House", "FRA", "#188038"},
		{"hse_hawking", "Hawking House", "HAW", "#F29900"},
	}
	for _, h := range houses {
		if _, err := tx.Exec(`INSERT INTO houses (id, name, name_short, color_hex) VALUES (?, ?, ?, ?)`, h.id, h.name, h.short, h.hex); err != nil {
			return err
		}
		st.houseIDs = append(st.houseIDs, h.id)
	}

	// 4. Seed Departments (12 Departments)
	depts := []struct {
		id, name, short, typ, hodID string
	}{
		{"dept_maths", "Mathematics", "MAT", "LEARNING_AREA", "stf_hod_maths"},
		{"dept_english", "English & Drama", "ENG", "LEARNING_AREA", "stf_tch_7b"},
		{"dept_science", "Sciences", "SCI", "LEARNING_AREA", "stf_009"},
		{"dept_humanities", "Humanities (History, Geography & RE)", "HUM", "LEARNING_AREA", "stf_010"},
		{"dept_mfl", "Modern Foreign Languages", "MFL", "LEARNING_AREA", "stf_011"},
		{"dept_computing", "Computing & Business", "CMP", "LEARNING_AREA", "stf_012"},
		{"dept_arts", "Creative & Performing Arts", "ART", "LEARNING_AREA", "stf_013"},
		{"dept_pe", "Physical Education & Sport", "PE", "LEARNING_AREA", "stf_014"},
		{"dept_primary", "Early Years & Primary Phase", "PRI", "LEARNING_AREA", "stf_015"},
		{"dept_sixth", "Sixth Form & Post-16 Studies", "SIX", "LEARNING_AREA", "stf_016"},
		{"dept_send", "Inclusion & SEND", "SEN", "PASTORAL", "stf_senco_01"},
		{"dept_pastoral", "Safeguarding, Pastoral & Administration", "PAS", "ADMINISTRATION", "stf_dsl_01"},
	}
	for _, d := range depts {
		if _, err := tx.Exec(`INSERT INTO departments (id, name, name_short, type, head_of_department_staff_id) VALUES (?, ?, ?, ?, ?)`,
			d.id, d.name, d.short, d.typ, d.hodID); err != nil {
			return err
		}
	}

	// 5. Seed Rooms (60 Rooms)
	for i := 1; i <= 60; i++ {
		rmID := fmt.Sprintf("rm_%03d", i)
		code := fmt.Sprintf("R%02d", i)
		name := fmt.Sprintf("Room %02d", i)
		rmType := "CLASSROOM"
		cap := 32
		if i > 40 && i <= 50 {
			code = fmt.Sprintf("SCI%d", i-40)
			name = fmt.Sprintf("Science Lab %d", i-40)
			rmType = "LAB"
		} else if i > 50 && i <= 56 {
			code = fmt.Sprintf("IT%d", i-50)
			name = fmt.Sprintf("Computing Suite %d", i-50)
			rmType = "LAB"
		} else if i > 56 {
			code = fmt.Sprintf("HAL%d", i-56)
			name = fmt.Sprintf("Sports & Assembly Hall %d", i-56)
			rmType = "HALL"
			cap = 180
		}
		if _, err := tx.Exec(`INSERT INTO rooms (id, code, name, room_type, capacity, is_active, phone_extension) VALUES (?, ?, ?, ?, ?, 1, ?)`,
			rmID, code, name, rmType, cap, fmt.Sprintf("2%03d", i)); err != nil {
			return err
		}
		st.roomIDs = append(st.roomIDs, rmID)
	}

	// 6. Seed Year Groups (16 Year Groups: N1..13) & Registration Forms (42 Forms)
	type ygSpec struct {
		id, ncCode, name, phase, ks string
		seq, numForms, formSize     int
		formSuffixes                []string
	}
	yearGroupSpecs := []ygSpec{
		{"yg_n1", "N1", "Nursery N1 (Age 3)", "NURSERY", "EYFS", 1, 1, 20, []string{"A"}},
		{"yg_n2", "N2", "Nursery N2 (Age 4)", "NURSERY", "EYFS", 2, 1, 20, []string{"A"}},
		{"yg_r", "R", "Reception", "PRIMARY", "EYFS", 3, 2, 26, []string{"A", "B"}},
		{"yg_1", "1", "Year 1", "PRIMARY", "KS1", 4, 2, 26, []string{"A", "B"}},
		{"yg_2", "2", "Year 2", "PRIMARY", "KS1", 5, 2, 26, []string{"A", "B"}},
		{"yg_3", "3", "Year 3", "PRIMARY", "KS2", 6, 2, 26, []string{"A", "B"}},
		{"yg_4", "4", "Year 4", "PRIMARY", "KS2", 7, 2, 26, []string{"A", "B"}},
		{"yg_5", "5", "Year 5", "PRIMARY", "KS2", 8, 2, 26, []string{"A", "B"}},
		{"yg_6", "6", "Year 6", "PRIMARY", "KS2", 9, 2, 26, []string{"A", "B"}},
		{"yg_7", "7", "Year 7", "SECONDARY", "KS3", 10, 4, 28, []string{"A", "B", "C", "D"}},
		{"yg_8", "8", "Year 8", "SECONDARY", "KS3", 11, 4, 28, []string{"A", "B", "C", "D"}},
		{"yg_9", "9", "Year 9", "SECONDARY", "KS3", 12, 4, 28, []string{"A", "B", "C", "D"}},
		{"yg_10", "10", "Year 10", "SECONDARY", "KS4", 13, 4, 28, []string{"A", "B", "C", "D"}},
		{"yg_11", "11", "Year 11", "SECONDARY", "KS4", 14, 4, 28, []string{"A", "B", "C", "D"}},
		{"yg_12", "12", "Year 12 (Lower Sixth)", "SIXTH_FORM", "KS5", 15, 3, 24, []string{"A", "B", "C"}},
		{"yg_13", "13", "Year 13 (Upper Sixth)", "SIXTH_FORM", "KS5", 16, 3, 24, []string{"A", "B", "C"}},
	}

	formIdx := 0
	for ygIdx, yg := range yearGroupSpecs {
		hoyID := st.teacherStaffIDs[(ygIdx+3)%len(st.teacherStaffIDs)]
		if _, err := tx.Exec(`INSERT INTO year_groups (id, nc_code, name, phase, key_stage, sequence_number, head_of_year_staff_id) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			yg.id, yg.ncCode, yg.name, yg.phase, yg.ks, yg.seq, hoyID); err != nil {
			return err
		}

		for _, suffix := range yg.formSuffixes {
			formCode := yg.ncCode + suffix
			formGroupID := "grp_form_" + formCode
			roomID := st.roomIDs[formIdx%len(st.roomIDs)]
			tutorID := st.teacherStaffIDs[(formIdx+6)%len(st.teacherStaffIDs)]
			if formCode == "7A" {
				tutorID = "stf_tch_7a"
			} else if formCode == "7B" {
				tutorID = "stf_tch_7b"
			}

			if _, err := tx.Exec(`INSERT INTO groups (id, academic_year_id, group_type, code, name, year_group_id, default_room_id, takes_attendance, effective_from)
				VALUES (?, 'ay_2026_2027', 'REGISTRATION_FORM', ?, ?, ?, ?, 1, '2026-09-01')`,
				formGroupID, formCode, "Form "+formCode, yg.id, roomID); err != nil {
				return err
			}
			if _, err := tx.Exec(`INSERT INTO group_staff_assignments (id, group_id, staff_id, role, academic_year_id, effective_from)
				VALUES (?, ?, ?, 'FORM_TUTOR', 'ay_2026_2027', '2026-09-01')`,
				"gsa_form_"+formCode, formGroupID, tutorID); err != nil {
				return err
			}

			st.forms = append(st.forms, &seededFormMeta{
				GroupID:     formGroupID,
				Code:        formCode,
				NCYearGroup: yg.ncCode,
				YearGroupID: yg.id,
				Phase:       yg.phase,
				KeyStage:    yg.ks,
				TutorID:     tutorID,
				RoomID:      roomID,
			})
			formIdx++
		}
	}

	// 7. Seed Pupils Across All 42 Forms (Pinned Anchor Pupils + Deterministic PRNG Cohort = 1,096 Pupils)
	return seedPupilsCoursesAndTimetable(tx, st, yearGroupSpecs)
}
