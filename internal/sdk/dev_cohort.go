package sdk

import (
	"context"

	"optimis/internal/dal"
)

func SeedSyntheticDevCohort(ctx context.Context, coreDAL *dal.CoreDAL) error {
	tx, err := coreDAL.WriteDB().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	const ayID = "ay_2026_2027"
	const effFrom = "2026-09-01"

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO academic_years (id, name, status, sequence_number, first_day, last_day)
		 VALUES (?, '2026/2027', 'CURRENT', (SELECT COALESCE(MAX(sequence_number), 0) + 1 FROM academic_years), ?, '2027-07-23')
		 ON CONFLICT(id) DO NOTHING`,
		ayID, effFrom,
	); err != nil {
		return err
	}

	roles := []struct {
		id, cat, name, short, desc, scopes string
	}{
		{
			"ADMIN", "STAFF", "System Administrator / Headteacher", "ADM", "Full school administrative access",
			`["dal.academic.read:all","dal.academic.write:all","dal.census.read:all","dal.dashboard.read:all","dal.dashboard.write:all","dal.pupils.read:all","dal.pupils.write:all","dal.staff.read:admin","dal.staff.write:admin","dal.groups.read:all","dal.groups.write:all","dal.attendance.read:all","dal.attendance.write:all","dal.guardians.read:all","dal.guardians.write:all","dal.sen.read:all","dal.sen.write:all","dal.medical.read:all","dal.medical.write:all","plugin.config.write"]`,
		},
		{
			"DSL", "STAFF", "Designated Safeguarding Lead", "DSL", "KCSIE Safeguarding Vault read & triage authority",
			`["dal.academic.read:all","dal.safeguarding.create","dal.safeguarding.read","dal.pupils.read:all","dal.guardians.read:all","dal.attendance.read:all","dal.sen.read:all","dal.sen.write:all","dal.medical.read:all","dal.medical.write:all"]`,
		},
		{
			"HOD", "STAFF", "Head of Department", "HOD", "Departmental curriculum and cohort lead",
			`["dal.academic.read:all","dal.pupils.read:all","dal.groups.read:all","dal.groups.write:all","dal.attendance.read:all","dal.sen.read:all","dal.sen.write:all","dal.medical.read:all","dal.safeguarding.create","plugin.config.write"]`,
		},
		{
			"TEACHER", "STAFF", "Classroom Teacher", "TCH", "Scoped access to assigned teaching and registration groups",
			`["dal.academic.read:all","dal.pupils.read:own_groups","dal.groups.read:own_groups","dal.attendance.read:own_groups","dal.attendance.write:own_groups","dal.guardians.read:own_groups","dal.sen.read:own_groups","dal.medical.read:own_groups","dal.safeguarding.create"]`,
		},
		{
			"SUPPORT", "STAFF", "School Office & Pastoral Support", "SUP", "Attendance officer, First Aider, and administrative support",
			`["dal.academic.read:all","dal.pupils.read:all","dal.attendance.read:all","dal.attendance.write:all","dal.guardians.read:all","dal.medical.read:all","dal.medical.write:all","dal.safeguarding.create"]`,
		},
	}
	for _, r := range roles {
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO roles (id, category, name, name_short, description, default_scopes_json)
			 VALUES (?, ?, ?, ?, ?, ?)
			 ON CONFLICT(id) DO UPDATE SET default_scopes_json = excluded.default_scopes_json`,
			r.id, r.cat, r.name, r.short, r.desc, r.scopes,
		); err != nil {
			return err
		}
	}

	staffRows := []struct {
		staffID, personID, code, title, first, last, staffType, jobTitle, roleID string
		isDSL                                                                    int
	}{
		{"stf_admin_01", "per_sdk_adm01", "ADM01", "Dame", "Eleanor", "Vance", "LEADERSHIP", "Executive Headteacher", "ADMIN", 1},
		{"stf_dsl_01", "per_sdk_dsl01", "DSL01", "Mr", "Marcus", "Thorne", "LEADERSHIP", "Designated Safeguarding Lead", "DSL", 1},
		{"stf_senco_01", "per_sdk_sen01", "SEN01", "Dr", "Amina", "Patel", "LEADERSHIP", "SENCO", "HOD", 1},
		{"stf_hod_maths", "per_sdk_hod01", "HOD01", "Mr", "David", "Sterling", "TEACHING", "Head of Mathematics", "HOD", 0},
		{"stf_tch_7a", "per_sdk_tch7a", "TCH7A", "Ms", "Hannah", "Abbott", "TEACHING", "Form Tutor 7A", "TEACHER", 0},
		{"stf_tch_7b", "per_sdk_tch7b", "TCH7B", "Mr", "Neville", "Longbottom", "TEACHING", "Form Tutor 7B", "TEACHER", 0},
		{"stf_support_01", "per_sdk_sup01", "SUP01", "Mrs", "Clara", "Oswald", "SUPPORT", "Attendance & Medical Officer", "SUPPORT", 0},
	}

	for _, s := range staffRows {
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO persons (id, external_sso_sub, title, legal_first_name, legal_last_name, preferred_first_name, preferred_last_name, sex, status)
			 VALUES (?, ?, ?, ?, ?, ?, ?, 'U', 'ACTIVE')
			 ON CONFLICT(id) DO NOTHING`,
			s.personID, "sso_"+s.staffID, s.title, s.first, s.last, s.first, s.last,
		); err != nil {
			return err
		}
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO person_roles (person_id, role_id, is_primary, effective_from)
			 VALUES (?, ?, 1, ?)
			 ON CONFLICT(person_id, role_id, effective_from) DO NOTHING`,
			s.personID, s.roleID, effFrom,
		); err != nil {
			return err
		}
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO staff (id, person_id, staff_code, staff_type, job_title, qts_flag, is_dsl, hire_date)
			 VALUES (?, ?, ?, ?, ?, 1, ?, ?)
			 ON CONFLICT(id) DO NOTHING`,
			s.staffID, s.personID, s.code, s.staffType, s.jobTitle, s.isDSL, effFrom,
		); err != nil {
			return err
		}
	}

	groups := []struct {
		id, groupType, code, name, leadStaffID string
	}{
		{"grp_reg_7a", "REGISTRATION_FORM", "7A", "Year 7 Form 7A", "stf_tch_7a"},
		{"grp_reg_7b", "REGISTRATION_FORM", "7B", "Year 7 Form 7B", "stf_tch_7b"},
		{"grp_mat_7x1", "TEACHING_CLASS", "7Ma/X1", "Year 7 Maths Set 1", "stf_tch_7a"},
	}
	for _, g := range groups {
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO groups (id, academic_year_id, group_type, code, name, takes_attendance, effective_from, effective_to)
			 VALUES (?, ?, ?, ?, ?, 1, ?, NULL)
			 ON CONFLICT(id) DO NOTHING`,
			g.id, ayID, g.groupType, g.code, g.name, effFrom,
		); err != nil {
			return err
		}
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO group_staff_assignments (id, group_id, staff_id, role, academic_year_id, effective_from, effective_to)
			 VALUES (?, ?, ?, 'LEAD_TEACHER', ?, ?, NULL)
			 ON CONFLICT(id) DO NOTHING`,
			"gsa_"+g.id, g.id, g.leadStaffID, ayID, effFrom,
		); err != nil {
			return err
		}
	}

	pupils := []struct {
		pupilID, personID, upn, first, last, yearGroup, regGroupID string
	}{
		{"pup_yr7_alice", "per_sdk_alice", "U100001202601", "Alice", "iddell", "7", "grp_reg_7a"},
		{"pup_yr7_noah", "per_sdk_noah", "U100001202602", "Noah", "Patel", "7", "grp_reg_7a"},
		{"pup_yr7_zara", "per_sdk_zara", "U100001202603", "Zara", "Khan", "7", "grp_reg_7b"},
		{"pup_pri_maya", "per_sdk_maya", "U100001202604", "Maya", "Lin", "4", "grp_reg_7a"},
	}
	for _, p := range pupils {
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO persons (id, legal_first_name, legal_last_name, preferred_first_name, preferred_last_name, sex, status)
			 VALUES (?, ?, ?, ?, ?, 'F', 'ACTIVE')
			 ON CONFLICT(id) DO NOTHING`,
			p.personID, p.first, p.last, p.first, p.last,
		); err != nil {
			return err
		}
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO pupils (id, person_id, upn, ethnicity_code, first_language_code)
			 VALUES (?, ?, ?, 'WBRI', 'ENG')
			 ON CONFLICT(id) DO NOTHING`,
			p.pupilID, p.personID, p.upn,
		); err != nil {
			return err
		}
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO pupil_year_enrolments (id, pupil_id, academic_year_id, nc_year_group, enrolment_status, effective_from, effective_to)
			 VALUES (?, ?, ?, ?, 'C', ?, NULL)
			 ON CONFLICT(id) DO NOTHING`,
			"enr_"+p.pupilID, p.pupilID, ayID, p.yearGroup, effFrom,
		); err != nil {
			return err
		}
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO group_pupil_memberships (id, group_id, pupil_id, academic_year_id, effective_from, effective_to)
			 VALUES (?, ?, ?, ?, ?, NULL)
			 ON CONFLICT(id) DO NOTHING`,
			"gpm_"+p.pupilID+"_"+p.regGroupID, p.regGroupID, p.pupilID, ayID, effFrom,
		); err != nil {
			return err
		}
	}

	days := []struct {
		id                        int
		name, short               string
		isSchool                  int
		open, amReg, pmReg, close any
	}{
		{1, "Monday", "Mon", 1, "08:00", "08:45", "13:30", "16:00"},
		{2, "Tuesday", "Tue", 1, "08:00", "08:45", "13:30", "16:00"},
		{3, "Wednesday", "Wed", 1, "08:00", "08:45", "13:30", "16:00"},
		{4, "Thursday", "Thu", 1, "08:00", "08:45", "13:30", "16:00"},
		{5, "Friday", "Fri", 1, "08:00", "08:45", "13:30", "15:30"},
		{6, "Saturday", "Sat", 0, nil, nil, nil, nil},
		{7, "Sunday", "Sun", 0, nil, nil, nil, nil},
	}
	for _, d := range days {
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO days_of_week (id, name, name_short, is_school_day, school_open_time, am_register_time, pm_register_time, school_close_time)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			 ON CONFLICT(id) DO NOTHING`,
			d.id, d.name, d.short, d.isSchool, d.open, d.amReg, d.pmReg, d.close,
		); err != nil {
			return err
		}
	}

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO academic_terms (id, academic_year_id, code, name, name_short, sequence_number, first_day, last_day)
		 VALUES ('trm_2627_aut', ?, 'AUTUMN', 'Autumn Term 2026', 'AUT', 1, '2026-09-01', '2026-12-18'),
		        ('trm_2627_spr', ?, 'SPRING', 'Spring Term 2027', 'SPR', 2, '2027-01-04', '2027-03-26'),
		        ('trm_2627_sum', ?, 'SUMMER', 'Summer Term 2027', 'SUM', 3, '2027-04-12', '2027-07-23')
		 ON CONFLICT(id) DO NOTHING`,
		ayID, ayID, ayID,
	); err != nil {
		return err
	}

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO alert_levels (id, name, name_short, color_hex, bg_color_hex, description, sequence_number)
		 VALUES ('LOW', 'Low Alert', 'LOW', '#1E4620', '#E6F4EA', 'Informational classroom support note', 1),
		        ('MEDIUM', 'Medium Alert', 'MED', '#7A4F01', '#FEF7E0', 'Moderate pastoral, SEN, or dietary awareness required', 2),
		        ('HIGH', 'High Alert', 'HIGH', '#A50E0E', '#FCE8E6', 'High-priority SEN EHCP or medical management plan', 3),
		        ('CRITICAL', 'Critical Life-Safety / Statutory Alert', 'CRIT', '#FFFFFF', '#B31412', 'Immediate life-safety medical condition or statutory restriction', 4)
		 ON CONFLICT(id) DO NOTHING`,
	); err != nil {
		return err
	}

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO sen_need_types (code, name, broad_area, description, sequence_number)
		 VALUES ('SPLD', 'Specific Learning Difficulty', 'COGNITION_AND_LEARNING', 'Dyslexia, dyscalculia, dyspraxia', 1),
		        ('SEMH', 'Social, Emotional and Mental Health', 'SOCIAL_EMOTIONAL_AND_MENTAL_HEALTH', 'Emotional regulation and wellbeing needs', 2),
		        ('ASD', 'Autistic Spectrum Disorder', 'COMMUNICATION_AND_INTERACTION', 'Autism spectrum communication and sensory needs', 3),
		        ('SLCN', 'Speech, Language and Communication Needs', 'COMMUNICATION_AND_INTERACTION', 'Receptive or expressive language needs', 4)
		 ON CONFLICT(code) DO NOTHING`,
	); err != nil {
		return err
	}

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO attendance_codes (code, name, statistical_category, physical_status, is_statutory_code, is_active, restricted_to_admin, sequence_number)
		 VALUES ('/', 'Present (AM)', 'PRESENT', 'ONSITE', 1, 1, 0, 1),
		        ('\', 'Present (PM)', 'PRESENT', 'ONSITE', 1, 1, 0, 2),
		        ('L', 'Late before registers closed', 'PRESENT', 'ONSITE_LATE', 1, 1, 0, 3),
		        ('I', 'Illness', 'AUTHORISED_ABSENCE', 'ABSENT', 1, 1, 0, 4),
		        ('N', 'No reason yet provided for absence', 'UNAUTHORISED_ABSENCE', 'ABSENT', 1, 1, 0, 5)
		 ON CONFLICT(code) DO NOTHING`,
	); err != nil {
		return err
	}

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO houses (id, name, name_short, color_hex)
		 VALUES ('hse_curie', 'Curie House', 'CUR', '#2563EB'),
		        ('hse_turing', 'Turing House', 'TUR', '#059669')
		 ON CONFLICT(id) DO NOTHING`,
	); err != nil {
		return err
	}

	depts := []struct {
		id, name, short, typ, hodID string
	}{
		{"dept_maths", "Mathematics", "MAT", "LEARNING_AREA", "stf_hod_maths"},
		{"dept_english", "English & Drama", "ENG", "LEARNING_AREA", "stf_tch_7b"},
		{"dept_science", "Sciences", "SCI", "LEARNING_AREA", "stf_tch_7a"},
	}
	for _, d := range depts {
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO departments (id, name, name_short, type, head_of_department_staff_id)
			 VALUES (?, ?, ?, ?, ?)
			 ON CONFLICT(id) DO NOTHING`,
			d.id, d.name, d.short, d.typ, d.hodID,
		); err != nil {
			return err
		}
	}

	rooms := []struct {
		id, code, name, rmType, ext string
		cap                         int
	}{
		{"rm_001", "R01", "Room 01 (Maths)", "CLASSROOM", "2001", 32},
		{"rm_002", "R02", "Room 02 (English)", "CLASSROOM", "2002", 32},
		{"rm_003", "SCI1", "Science Lab 1", "LAB", "2003", 30},
	}
	for _, r := range rooms {
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO rooms (id, code, name, room_type, capacity, is_active, phone_extension)
			 VALUES (?, ?, ?, ?, ?, 1, ?)
			 ON CONFLICT(id) DO NOTHING`,
			r.id, r.code, r.name, r.rmType, r.cap, r.ext,
		); err != nil {
			return err
		}
	}

	yearGroups := []struct {
		id, ncCode, name, phase, ks, hoyID string
		seq                                int
	}{
		{"yg_4", "4", "Year 4", "PRIMARY", "KS2", "stf_tch_7a", 4},
		{"yg_7", "7", "Year 7", "SECONDARY", "KS3", "stf_tch_7a", 7},
		{"yg_8", "8", "Year 8", "SECONDARY", "KS3", "stf_tch_7b", 8},
		{"yg_9", "9", "Year 9", "SECONDARY", "KS3", "stf_hod_maths", 9},
		{"yg_10", "10", "Year 10", "SECONDARY", "KS4", "stf_hod_maths", 10},
		{"yg_11", "11", "Year 11", "SECONDARY", "KS4", "stf_hod_maths", 11},
	}
	for _, yg := range yearGroups {
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO year_groups (id, nc_code, name, phase, key_stage, sequence_number, head_of_year_staff_id)
			 VALUES (?, ?, ?, ?, ?, ?, ?)
			 ON CONFLICT(id) DO NOTHING`,
			yg.id, yg.ncCode, yg.name, yg.phase, yg.ks, yg.seq, yg.hoyID,
		); err != nil {
			return err
		}
	}

	courses := []struct {
		id, deptID, code, name, dfeCode string
	}{
		{"crs_maths_ks3", "dept_maths", "KS3-MAT", "KS3 Mathematics", "G100"},
		{"crs_english_ks3", "dept_english", "KS3-ENG", "KS3 English", "Q300"},
		{"crs_science_ks3", "dept_science", "KS3-SCI", "KS3 Combined Science", "F000"},
	}
	for _, c := range courses {
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO courses (id, academic_year_id, department_id, code, name, dfe_subject_code)
			 VALUES (?, ?, ?, ?, ?, ?)
			 ON CONFLICT(id) DO NOTHING`,
			c.id, ayID, c.deptID, c.code, c.name, c.dfeCode,
		); err != nil {
			return err
		}
	}

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO timetable_periods (id, academic_year_id, cycle_week, day_of_week_id, period_code, name, start_time, end_time, period_type, maps_to_statutory_session, sequence_number)
		 VALUES ('ttp_wk1_p1_am', ?, 1, 1, 'P1_AM', 'Period 1 (AM)', '09:00', '10:00', 'LESSON', 'AM', 1),
		        ('ttp_wk1_p4_pm', ?, 1, 1, 'P4_PM', 'Period 4 (PM)', '13:30', '14:30', 'LESSON', 'PM', 4)
		 ON CONFLICT(id) DO NOTHING`,
		ayID, ayID,
	); err != nil {
		return err
	}

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO timetable_slots (id, timetable_period_id, group_id, staff_id, room_id, effective_from, effective_to)
		 VALUES ('slot_wk1_p1_7x1', 'ttp_wk1_p1_am', 'grp_mat_7x1', 'stf_tch_7a', 'rm_001', ?, NULL)
		 ON CONFLICT(id) DO NOTHING`,
		effFrom,
	); err != nil {
		return err
	}

	return tx.Commit()
}
