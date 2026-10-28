package main

import (
	"database/sql"
	"fmt"
)

func seedDomain02HouseholdsAndGuardians(tx *sql.Tx, st *qaSeedState) error {
	// 1. Pinned Anchor Case: Alice Evans (`pup_yr7_alice`) — Confidential Refuge Household + Court-Order Restricted Father
	if _, err := tx.Exec(`INSERT INTO households (id, name, formal_salutation, uprn, address_line_1, locality, town_city, county, postcode, is_confidential_address)
		VALUES ('hh_evans_refuge', 'Evans Household (Confidential)', 'Ms C. Evans', '100020990001', 'Flat 4, Haven House, 12 St Giles Way', 'Jericho', 'Oxford', 'Oxfordshire', 'OX2 6HT', 1)`); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO households (id, name, formal_salutation, uprn, address_line_1, locality, town_city, county, postcode, is_confidential_address)
		VALUES ('hh_evans_father_sep', 'Mr R. Evans Household', 'Mr R. Evans', '100020990002', '84 Cowley Road', 'Cowley', 'Oxford', 'Oxfordshire', 'OX4 1JB', 0)`); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO persons (id, email, title, legal_first_name, legal_last_name, preferred_first_name, preferred_last_name, sex, dob, phone_primary, phone_primary_type, status)
		VALUES ('per_gdn_evans_mother', 'claire.evans@parent.oakridge.sch.uk', 'Ms', 'Claire', 'Evans', 'Claire', 'Evans', 'F', '1986-08-12', '07700950001', 'MOBILE', 'ACTIVE')`); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO persons (id, email, title, legal_first_name, legal_last_name, preferred_first_name, preferred_last_name, sex, dob, phone_primary, phone_primary_type, status)
		VALUES ('per_gdn_evans_father', 'richard.evans.restricted@parent.oakridge.sch.uk', 'Mr', 'Richard', 'Evans', 'Richard', 'Evans', 'M', '1984-02-09', '07700950002', 'MOBILE', 'ACTIVE')`); err != nil {
		return err
	}
	for _, gid := range []string{"per_gdn_evans_mother", "per_gdn_evans_father"} {
		if _, err := tx.Exec(`INSERT INTO person_roles (person_id, role_id, is_primary, effective_from) VALUES (?, 'GUARDIAN', 1, '2026-09-01')`, gid); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(`INSERT INTO household_memberships (id, household_id, person_id, member_role, residency_type, effective_from)
		VALUES ('hhm_alice_refuge', 'hh_evans_refuge', 'per_pup_yr7_alice', 'CHILD', 'PRIMARY', '2026-09-01'),
		       ('hhm_mother_refuge', 'hh_evans_refuge', 'per_gdn_evans_mother', 'ADULT', 'PRIMARY', '2026-09-01'),
		       ('hhm_father_sep', 'hh_evans_father_sep', 'per_gdn_evans_father', 'ADULT', 'PRIMARY', '2026-09-01')`); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO guardian_relationships (id, pupil_id, guardian_person_id, relationship_code, has_parental_responsibility, court_order_restricted, court_order_notes, contact_priority, is_emergency_contact, authorised_to_collect, portal_data_access, receive_reports, contact_by_phone, contact_by_sms, contact_by_email, contact_by_post, effective_from)
		VALUES ('grel_alice_mother', 'pup_yr7_alice', 'per_gdn_evans_mother', 'MOTHER', 1, 0, '', 1, 1, 1, 1, 1, 1, 1, 1, 1, '2026-09-01'),
		       ('grel_evans_father_restricted', 'pup_yr7_alice', 'per_gdn_evans_father', 'FATHER', 0, 1, 'Family Court Prohibited Steps Order (Ref OX-2026-8841): Strictly zero contact, collection, portal, or address disclosure permitted.', 10, 0, 0, 0, 0, 0, 0, 0, 0, '2026-09-01')`); err != nil {
		return err
	}

	// 2. Pinned Anchor Case: Ryan Mitchell (`pup_yr10_ryan`) — Looked After Child in Foster Care + Local Authority Social Worker
	if _, err := tx.Exec(`INSERT INTO households (id, name, formal_salutation, uprn, address_line_1, locality, town_city, county, postcode, is_confidential_address)
		VALUES ('hh_foster_taylor', 'Taylor Foster Household', 'Mr & Mrs K. Taylor', '100020990003', '29 Banbury Road', 'Summertown', 'Oxford', 'Oxfordshire', 'OX2 7JQ', 1)`); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO persons (id, email, title, legal_first_name, legal_last_name, preferred_first_name, preferred_last_name, sex, dob, phone_primary, phone_primary_type, status)
		VALUES ('per_gdn_foster_taylor', 'k.taylor@fostercare.oxfordshire.gov.uk', 'Mrs', 'Karen', 'Taylor', 'Karen', 'Taylor', 'F', '1976-05-14', '07700950003', 'MOBILE', 'ACTIVE'),
		       ('per_gdn_social_worker', 'j.okafor@children.oxfordshire.gov.uk', 'Ms', 'Joanne', 'Okafor', 'Joanne', 'Okafor', 'F', '1985-10-20', '07700950004', 'WORK', 'ACTIVE')`); err != nil {
		return err
	}
	for _, gid := range []string{"per_gdn_foster_taylor", "per_gdn_social_worker"} {
		if _, err := tx.Exec(`INSERT INTO person_roles (person_id, role_id, is_primary, effective_from) VALUES (?, 'GUARDIAN', 1, '2026-09-01')`, gid); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(`INSERT INTO household_memberships (id, household_id, person_id, member_role, residency_type, effective_from)
		VALUES ('hhm_ryan_foster', 'hh_foster_taylor', 'per_pup_yr10_ryan', 'CHILD', 'PRIMARY', '2026-09-01'),
		       ('hhm_karen_foster', 'hh_foster_taylor', 'per_gdn_foster_taylor', 'ADULT', 'PRIMARY', '2026-09-01')`); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO guardian_relationships (id, pupil_id, guardian_person_id, relationship_code, has_parental_responsibility, court_order_restricted, contact_priority, is_emergency_contact, authorised_to_collect, portal_data_access, receive_reports, contact_by_phone, contact_by_sms, contact_by_email, contact_by_post, effective_from)
		VALUES ('grel_ryan_foster', 'pup_yr10_ryan', 'per_gdn_foster_taylor', 'FOSTER_CARER', 1, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, '2026-09-01'),
		       ('grel_ryan_social', 'pup_yr10_ryan', 'per_gdn_social_worker', 'SOCIAL_WORKER', 1, 0, 2, 1, 0, 1, 1, 1, 0, 1, 0, '2026-09-01')`); err != nil {
		return err
	}

	// 3. Generate Households & Guardians for Remaining 1,094 Pupils (~880 Households total, with sibling sharing)
	hhCounter := 1
	gdnCounter := 1
	var prevHouseholdID string
	var prevMotherID, prevFatherID string
	var prevLastName string

	for idx, p := range st.pupils {
		if p.PupilID == "pup_yr7_alice" || p.PupilID == "pup_yr10_ryan" {
			continue
		}

		// ~22% of pupils share a household as siblings with the preceding family
		shareSiblingHousehold := prevHouseholdID != "" && idx%5 == 0
		if shareSiblingHousehold {
			if _, err := tx.Exec(`INSERT INTO household_memberships (id, household_id, person_id, member_role, residency_type, effective_from)
				VALUES (?, ?, ?, 'CHILD', 'PRIMARY', '2026-09-01')`,
				fmt.Sprintf("hhm_sib_%s", p.PupilID), prevHouseholdID, p.PersonID); err != nil {
				return err
			}
			if _, err := tx.Exec(`INSERT INTO guardian_relationships (id, pupil_id, guardian_person_id, relationship_code, has_parental_responsibility, court_order_restricted, contact_priority, is_emergency_contact, authorised_to_collect, portal_data_access, receive_reports, contact_by_phone, contact_by_sms, contact_by_email, contact_by_post, effective_from)
				VALUES (?, ?, ?, 'MOTHER', 1, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, '2026-09-01')`,
				fmt.Sprintf("grel_%s_m", p.PupilID), p.PupilID, prevMotherID); err != nil {
				return err
			}
			if prevFatherID != "" {
				if _, err := tx.Exec(`INSERT INTO guardian_relationships (id, pupil_id, guardian_person_id, relationship_code, has_parental_responsibility, court_order_restricted, contact_priority, is_emergency_contact, authorised_to_collect, portal_data_access, receive_reports, contact_by_phone, contact_by_sms, contact_by_email, contact_by_post, effective_from)
					VALUES (?, ?, ?, 'FATHER', 1, 0, 2, 1, 1, 1, 1, 1, 1, 1, 1, '2026-09-01')`,
					fmt.Sprintf("grel_%s_f", p.PupilID), p.PupilID, prevFatherID); err != nil {
					return err
				}
			}
			prevHouseholdID = ""
			continue
		}

		hhID := fmt.Sprintf("hh_%04d", hhCounter)
		street := st.gen.pick(ukStreets)
		loc := st.gen.pick(ukLocalities)
		outward := st.gen.pick(ukPostcodeOutwards)
		postcode := fmt.Sprintf("%s %d%c%c", outward, 1+st.gen.intN(9), 'A'+byte(st.gen.intN(20)), 'A'+byte(st.gen.intN(20)))
		uprn := fmt.Sprintf("100021%06d", hhCounter)
		isConf := 0
		if hhCounter%60 == 0 {
			isConf = 1
		}

		if _, err := tx.Exec(`INSERT INTO households (id, name, formal_salutation, uprn, address_line_1, locality, town_city, county, postcode, is_confidential_address)
			VALUES (?, ?, ?, ?, ?, ?, 'Oxford', 'Oxfordshire', ?, ?)`,
			hhID, p.LastName+" Household", "Mr & Mrs "+p.LastName, uprn, fmt.Sprintf("%d %s", 1+st.gen.intN(160), street), loc, postcode, isConf); err != nil {
			return err
		}

		if _, err := tx.Exec(`INSERT INTO household_memberships (id, household_id, person_id, member_role, residency_type, effective_from)
			VALUES (?, ?, ?, 'CHILD', 'PRIMARY', '2026-09-01')`,
			fmt.Sprintf("hhm_ch_%s", p.PupilID), hhID, p.PersonID); err != nil {
			return err
		}

		// Primary Guardian (Mother)
		motherID := fmt.Sprintf("per_gdn_%04d", gdnCounter)
		gdnCounter++
		mFirst := st.gen.pick(ukFemaleFirstNames)
		if _, err := tx.Exec(`INSERT INTO persons (id, email, title, legal_first_name, legal_last_name, preferred_first_name, preferred_last_name, sex, dob, phone_primary, phone_primary_type, status)
			VALUES (?, ?, 'Mrs', ?, ?, ?, ?, 'F', '1983-06-15', ?, 'MOBILE', 'ACTIVE')`,
			motherID, fmt.Sprintf("gdn.%s@parent.oakridge.sch.uk", motherID), mFirst, p.LastName, mFirst, p.LastName, fmt.Sprintf("077008%05d", gdnCounter)); err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT INTO person_roles (person_id, role_id, is_primary, effective_from) VALUES (?, 'GUARDIAN', 1, '2026-09-01')`, motherID); err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT INTO household_memberships (id, household_id, person_id, member_role, residency_type, effective_from)
			VALUES (?, ?, ?, 'ADULT', 'PRIMARY', '2026-09-01')`,
			fmt.Sprintf("hhm_ad_%s", motherID), hhID, motherID); err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT INTO guardian_relationships (id, pupil_id, guardian_person_id, relationship_code, has_parental_responsibility, court_order_restricted, contact_priority, is_emergency_contact, authorised_to_collect, portal_data_access, receive_reports, contact_by_phone, contact_by_sms, contact_by_email, contact_by_post, effective_from)
			VALUES (?, ?, ?, 'MOTHER', 1, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, '2026-09-01')`,
			fmt.Sprintf("grel_%s_m", p.PupilID), p.PupilID, motherID); err != nil {
			return err
		}

		// Secondary Guardian (~75% of households have a 2nd guardian; 10 of those are court-order restricted)
		fatherID := ""
		if st.gen.chance(0.75) {
			fatherID = fmt.Sprintf("per_gdn_%04d", gdnCounter)
			gdnCounter++
			fFirst := st.gen.pick(ukMaleFirstNames)
			if _, err := tx.Exec(`INSERT INTO persons (id, email, title, legal_first_name, legal_last_name, preferred_first_name, preferred_last_name, sex, dob, phone_primary, phone_primary_type, status)
				VALUES (?, ?, 'Mr', ?, ?, ?, ?, 'M', '1981-09-20', ?, 'MOBILE', 'ACTIVE')`,
				fatherID, fmt.Sprintf("gdn.%s@parent.oakridge.sch.uk", fatherID), fFirst, p.LastName, fFirst, p.LastName, fmt.Sprintf("077008%05d", gdnCounter)); err != nil {
				return err
			}
			if _, err := tx.Exec(`INSERT INTO person_roles (person_id, role_id, is_primary, effective_from) VALUES (?, 'GUARDIAN', 1, '2026-09-01')`, fatherID); err != nil {
				return err
			}
			if _, err := tx.Exec(`INSERT INTO household_memberships (id, household_id, person_id, member_role, residency_type, effective_from)
				VALUES (?, ?, ?, 'ADULT', 'PRIMARY', '2026-09-01')`,
				fmt.Sprintf("hhm_ad_%s", fatherID), hhID, fatherID); err != nil {
				return err
			}

			if hhCounter <= 10 {
				// Court-order restricted secondary guardian (all 8 contact/portal flags strictly 0)
				if _, err := tx.Exec(`INSERT INTO guardian_relationships (id, pupil_id, guardian_person_id, relationship_code, has_parental_responsibility, court_order_restricted, court_order_notes, contact_priority, is_emergency_contact, authorised_to_collect, portal_data_access, receive_reports, contact_by_phone, contact_by_sms, contact_by_email, contact_by_post, effective_from)
					VALUES (?, ?, ?, 'FATHER', 0, 1, 'Court order restriction on file', 10, 0, 0, 0, 0, 0, 0, 0, 0, '2026-09-01')`,
					fmt.Sprintf("grel_%s_f", p.PupilID), p.PupilID, fatherID); err != nil {
					return err
				}
				fatherID = ""
			} else {
				if _, err := tx.Exec(`INSERT INTO guardian_relationships (id, pupil_id, guardian_person_id, relationship_code, has_parental_responsibility, court_order_restricted, contact_priority, is_emergency_contact, authorised_to_collect, portal_data_access, receive_reports, contact_by_phone, contact_by_sms, contact_by_email, contact_by_post, effective_from)
					VALUES (?, ?, ?, 'FATHER', 1, 0, 2, 1, 1, 1, 1, 1, 1, 1, 1, '2026-09-01')`,
					fmt.Sprintf("grel_%s_f", p.PupilID), p.PupilID, fatherID); err != nil {
					return err
				}
			}
		}

		prevHouseholdID = hhID
		prevMotherID = motherID
		prevFatherID = fatherID
		prevLastName = p.LastName
		_ = prevLastName
		hhCounter++
	}

	return nil
}

func seedDomain04StatutoryAttendance(tx *sql.Tx, st *qaSeedState) error {
	// 1. Seed Official DfE Attendance Codes (August 2024+ Statutory Guidance)
	codes := []struct {
		code, name, statCat, phys string
		isStat, adminOnly, seq    int
	}{
		{"/", "Present (AM)", "PRESENT", "ONSITE", 1, 0, 1},
		{`\`, "Present (PM)", "PRESENT", "ONSITE", 1, 0, 2},
		{"L", "Late Arrival (Before Register Closed)", "PRESENT", "ONSITE_LATE", 1, 0, 3},
		{"K", "Attending Education Provision Arranged by Local Authority", "APPROVED_EDUCATIONAL_ACTIVITY", "OFFSITE_EDUCATIONAL", 1, 1, 4},
		{"V", "Educational Visit or Trip", "APPROVED_EDUCATIONAL_ACTIVITY", "OFFSITE_EDUCATIONAL", 1, 0, 5},
		{"P", "Approved Sporting Activity", "APPROVED_EDUCATIONAL_ACTIVITY", "OFFSITE_EDUCATIONAL", 1, 0, 6},
		{"W", "Work Experience", "APPROVED_EDUCATIONAL_ACTIVITY", "OFFSITE_EDUCATIONAL", 1, 0, 7},
		{"B", "Educated Off-Site (Approved Alternative Provision)", "APPROVED_EDUCATIONAL_ACTIVITY", "OFFSITE_EDUCATIONAL", 1, 1, 8},
		{"J1", "Interview with Prospective Employer or Educational Establishment", "APPROVED_EDUCATIONAL_ACTIVITY", "OFFSITE_EDUCATIONAL", 1, 0, 9},
		{"I", "Illness (Not Medical or Dental Appointment)", "AUTHORISED_ABSENCE", "ABSENT", 1, 0, 10},
		{"M", "Medical or Dental Appointment", "AUTHORISED_ABSENCE", "ABSENT", 1, 0, 11},
		{"C", "Leave of Absence for Exceptional Circumstances", "AUTHORISED_ABSENCE", "ABSENT", 1, 1, 12},
		{"C1", "Leave of Absence for Regulated Performance", "AUTHORISED_ABSENCE", "ABSENT", 1, 1, 13},
		{"E", "Suspended or Permanently Excluded (No Alternative Provision)", "AUTHORISED_ABSENCE", "ABSENT", 1, 1, 14},
		{"R", "Religious Observance", "AUTHORISED_ABSENCE", "ABSENT", 1, 0, 15},
		{"T", "Parent Travelling for Occupational Purposes", "AUTHORISED_ABSENCE", "ABSENT", 1, 1, 16},
		{"G", "Holiday Not Granted by School", "UNAUTHORISED_ABSENCE", "ABSENT", 1, 1, 17},
		{"N", "Reason for Absence Not Yet Provided", "UNAUTHORISED_ABSENCE", "ABSENT", 1, 0, 18},
		{"O", "Absent in Other or Unknown Circumstances", "UNAUTHORISED_ABSENCE", "ABSENT", 1, 0, 19},
		{"U", "Arrived in School After Registration Closed", "UNAUTHORISED_ABSENCE", "ONSITE_LATE", 1, 0, 20},
		{"X", "Non-Compulsory School Age Pupil Not Required to Attend", "NOT_REQUIRED_TO_ATTEND", "ABSENT", 1, 0, 21},
		{"Y1", "Unable to Attend Due to Transport Normally Provided Not Available", "UNABLE_TO_ATTEND_EXCEPTIONAL", "ABSENT", 1, 1, 22},
		{"#", "Planned Whole or Partial School Closure", "NOT_REQUIRED_TO_ATTEND", "CLOSURE", 1, 1, 23},
	}
	for _, c := range codes {
		if _, err := tx.Exec(`INSERT INTO attendance_codes (code, name, statistical_category, physical_status, is_statutory_code, is_active, restricted_to_admin, sequence_number)
			VALUES (?, ?, ?, ?, ?, 1, ?, ?)`,
			c.code, c.name, c.statCat, c.phys, c.isStat, c.adminOnly, c.seq); err != nil {
			return err
		}
	}

	// 2. Seed 20 School Days of Statutory AM/PM Marks (`attendance_session_marks`) using a prepared statement
	sessionStmt, err := tx.Prepare(`INSERT INTO attendance_session_marks (id, pupil_id, academic_year_id, date, session, attendance_code, minutes_late, sub_code, comment, source_group_id, recorded_by_staff_id)
		VALUES (?, ?, 'ay_2026_2027', ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer sessionStmt.Close()

	regSubStmt, err := tx.Prepare(`INSERT INTO attendance_register_submissions (id, group_id, date, session_or_slot_key, taken_by_staff_id)
		VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer regSubStmt.Close()

	markSeq := 1
	subSeq := 1
	for dayIdx, dateStr := range autumn2026SchoolDays {
		for _, form := range st.forms {
			// Record AM & PM register submissions for this form
			for _, sess := range []string{"AM", "PM"} {
				subID := fmt.Sprintf("ars_%06d", subSeq)
				subSeq++
				if _, err := regSubStmt.Exec(subID, form.GroupID, dateStr, sess, form.TutorID); err != nil {
					return err
				}
			}

			for _, pid := range form.PupilIDs {
				// Pinned suspension scenario for Ryan Mitchell on days 10 & 11
				if pid == "pup_yr10_ryan" && (dayIdx == 10 || dayIdx == 11) {
					for _, sess := range []string{"AM", "PM"} {
						id := fmt.Sprintf("asm_%06d", markSeq)
						markSeq++
						if _, err := sessionStmt.Exec(id, pid, dateStr, sess, "E", 0, "", "Fixed-term suspension", form.GroupID, form.TutorID); err != nil {
							return err
						}
					}
					continue
				}

				for _, sess := range []string{"AM", "PM"} {
					id := fmt.Sprintf("asm_%06d", markSeq)
					markSeq++
					code := "/"
					if sess == "PM" {
						code = `\`
					}
					minsLate := 0
					subCode := ""
					comment := ""

					roll := st.gen.rng.Float64()
					if roll < 0.024 {
						code = "L"
						minsLate = 4 + st.gen.intN(22)
						comment = "Late bus / traffic"
					} else if roll < 0.052 {
						code = "I"
						subCode = "I01"
						comment = "Parent reported viral illness"
					} else if roll < 0.058 {
						code = "M"
						comment = "Orthodontist / GP appointment"
					} else if roll < 0.066 {
						code = "O"
						comment = "Unauthorised absence"
					} else if roll < 0.070 {
						code = "U"
						minsLate = 35 + st.gen.intN(25)
						comment = "Arrived after register closed"
					}

					if _, err := sessionStmt.Exec(id, pid, dateStr, sess, code, minsLate, subCode, comment, form.GroupID, form.TutorID); err != nil {
						return err
					}
				}
			}
		}
	}

	// 3. Seed Period-by-Period Lesson Attendance Marks (`attendance_lesson_marks`) for Secondary/Sixth Form classes across 3 days
	lessonStmt, err := tx.Prepare(`INSERT INTO attendance_lesson_marks (id, pupil_id, timetable_slot_id, group_id, date, attendance_code, minutes_late, comment, recorded_by_staff_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer lessonStmt.Close()

	lMarkSeq := 1
	for _, dateStr := range autumn2026SchoolDays[:3] {
		for _, form := range st.forms {
			if form.Phase != "SECONDARY" && form.Phase != "SIXTH_FORM" {
				continue
			}
			var classGroupID string
			if form.KeyStage == "KS3" || form.KeyStage == "KS4" || form.KeyStage == "KS5" {
				classGroupID = "grp_class_" + form.Code + "Ma"
			}
			slots := st.slotIDsByGroup[classGroupID]
			if len(slots) == 0 {
				continue
			}
			slotID := slots[0]

			subID := fmt.Sprintf("ars_%06d", subSeq)
			subSeq++
			if _, err := regSubStmt.Exec(subID, classGroupID, dateStr, slotID, form.TutorID); err != nil {
				return err
			}

			for _, pid := range form.PupilIDs {
				id := fmt.Sprintf("alm_%06d", lMarkSeq)
				lMarkSeq++
				code := "/"
				minsLate := 0
				if st.gen.chance(0.03) {
					code = "L"
					minsLate = 5
				}
				if _, err := lessonStmt.Exec(id, pid, slotID, classGroupID, dateStr, code, minsLate, "", form.TutorID); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func seedDomain05And06SenMedicalCensusAndSafeguarding(tx *sql.Tx, st *qaSeedState) error {
	// 1. Seed Official DfE SEN Need Types (`sen_need_types`)
	senTypes := []struct {
		code, name, area, desc string
		seq                    int
	}{
		{"ASD", "Autistic Spectrum Disorder", "COMMUNICATION_AND_INTERACTION", "Diagnosis of Autistic Spectrum Condition", 1},
		{"SLCN", "Speech, Language and Communication Needs", "COMMUNICATION_AND_INTERACTION", "Receptive or expressive language needs", 2},
		{"SEMH", "Social, Emotional and Mental Health", "SOCIAL_EMOTIONAL_AND_MENTAL_HEALTH", "Emotional regulation, anxiety, or ADHD support", 3},
		{"SPLD", "Specific Learning Difficulty (Dyslexia/Dyscalculia/Dyspraxia)", "COGNITION_AND_LEARNING", "Specific literacy or numeracy processing difficulty", 4},
		{"MLD", "Moderate Learning Difficulty", "COGNITION_AND_LEARNING", "General cognition and learning support", 5},
		{"SLD", "Severe Learning Difficulty", "COGNITION_AND_LEARNING", "Significant global learning difficulty", 6},
		{"PMLD", "Profound and Multiple Learning Difficulty", "COGNITION_AND_LEARNING", "Complex multiple learning and physical needs", 7},
		{"HI", "Hearing Impairment", "SENSORY_AND_PHYSICAL", "Audiological support and radio aid usage", 8},
		{"VI", "Visual Impairment", "SENSORY_AND_PHYSICAL", "Modified print and visual access support", 9},
		{"MSI", "Multi-Sensory Impairment", "SENSORY_AND_PHYSICAL", "Combined hearing and vision support", 10},
		{"PD", "Physical Disability", "SENSORY_AND_PHYSICAL", "Mobility and physical access adjustments", 11},
		{"OTH", "Other Difficulty / Disability", "OTHER", "Other diagnosed SEN need", 12},
		{"NSA", "SEN Support But No Specialist Assessment of Type of Need", "OTHER", "Awaiting EP/specialist assessment", 13},
	}
	senCodes := []string{"SPLD", "SLCN", "SEMH", "ASD", "MLD", "HI", "VI", "PD", "OTH"}
	for _, s := range senTypes {
		if _, err := tx.Exec(`INSERT INTO sen_need_types (code, name, broad_area, description, sequence_number) VALUES (?, ?, ?, ?, ?)`,
			s.code, s.name, s.area, s.desc, s.seq); err != nil {
			return err
		}
	}

	// 2. Seed FSM Periods, SEN Records, SEN Needs, SEN Assistants, Post-16 Learning Aims, Medical Conditions, Pastoral Notes
	fsmSeq := 1
	senSeq := 1
	aimSeq := 1
	medSeq := 1
	pasSeq := 1

	for idx, p := range st.pupils {
		// FSM Eligibility
		if p.IsFSM {
			if _, err := tx.Exec(`INSERT INTO pupil_fsm_periods (id, pupil_id, effective_from, country_code, ecs_reference)
				VALUES (?, ?, '2025-04-01', 'ENG', ?)`,
				fmt.Sprintf("fsm_%04d", fsmSeq), p.PupilID, fmt.Sprintf("ECS-931-%05d", fsmSeq)); err != nil {
				return err
			}
			fsmSeq++
		}

		// SEN Records (`K` and `E`)
		if p.SENCode == "K" || p.SENCode == "E" {
			var ehcpStart, ehcpReview any
			alertID := "MEDIUM"
			if p.SENCode == "E" {
				ehcpStart = "2024-05-15"
				ehcpReview = "2027-05-15"
				alertID = "HIGH"
			}
			if _, err := tx.Exec(`INSERT INTO pupil_sen_records (id, pupil_id, provision_code, ehcp_start_date, ehcp_review_due_date, teaching_strategies, pupil_targets, senco_notes, effective_from)
				VALUES (?, ?, ?, ?, ?, 'Chunked instructions, visual timetable, pre-teaching key vocabulary, extra processing time.', 'Termly literacy and self-regulation targets agreed with parents.', 'Reviewed by Dr P. Nair (SENCO).', '2026-09-01')`,
				fmt.Sprintf("senr_%04d", senSeq), p.PupilID, p.SENCode, ehcpStart, ehcpReview); err != nil {
				return err
			}

			primaryNeed := st.gen.pick(senCodes)
			if p.PupilID == "pup_yr7_noah" {
				primaryNeed = "ASD"
			} else if p.PupilID == "pup_yr7_zara" {
				primaryNeed = "SPLD"
			}
			if _, err := tx.Exec(`INSERT INTO pupil_sen_needs (id, pupil_id, sen_need_code, ranking, alert_level_id, effective_from)
				VALUES (?, ?, ?, 1, ?, '2026-09-01')`,
				fmt.Sprintf("senn_%04d_1", senSeq), p.PupilID, primaryNeed, alertID); err != nil {
				return err
			}

			// Secondary need for EHCP pupils and Noah
			if p.SENCode == "E" || p.PupilID == "pup_yr7_noah" {
				secNeed := "SEMH"
				if primaryNeed == "SEMH" {
					secNeed = "SLCN"
				}
				if _, err := tx.Exec(`INSERT INTO pupil_sen_needs (id, pupil_id, sen_need_code, ranking, alert_level_id, effective_from)
					VALUES (?, ?, ?, 2, 'MEDIUM', '2026-09-01')`,
					fmt.Sprintf("senn_%04d_2", senSeq), p.PupilID, secNeed); err != nil {
					return err
				}
				taID := st.taStaffIDs[senSeq%len(st.taStaffIDs)]
				if p.PupilID == "pup_yr7_noah" {
					taID = "stf_ta_01"
				}
				if _, err := tx.Exec(`INSERT INTO pupil_sen_assistants (id, pupil_id, staff_id, allocation_notes, effective_from)
					VALUES (?, ?, ?, '15 hours/week in-class and small-group EHCP support', '2026-09-01')`,
					fmt.Sprintf("sena_%04d", senSeq), p.PupilID, taID); err != nil {
					return err
				}
			}
			senSeq++
		}

		// Post-16 Learning Aims for all KS5 (Year 12 & Year 13) pupils
		if p.KeyStage == "KS5" {
			aims := []struct {
				qn, ldcs, crs, name string
			}{
				{"6031333X", "G100", "crs_ks5_mat", "A-Level Mathematics"},
				{"60314953", "G100", "crs_ks5_fma", "A-Level Further Mathematics"},
				{"60147477", "F300", "crs_ks5_phy", "A-Level Physics"},
			}
			for _, a := range aims {
				if _, err := tx.Exec(`INSERT INTO pupil_post16_learning_aims (id, pupil_id, academic_year_id, qn_code, subject_classification_code, course_id, planned_learning_hours, planned_eep_hours, completion_status, outcome_code, effective_from, planned_end_date)
					VALUES (?, ?, 'ay_2026_2027', ?, ?, ?, 180, 30, '1', '8', '2026-09-01', '2028-06-30')`,
					fmt.Sprintf("p16_%04d", aimSeq), p.PupilID, a.qn, a.ldcs, a.crs); err != nil {
					return err
				}
				aimSeq++
			}
		}

		// Medical Conditions (~8.5% of cohort + Maya Patel Critical Peanut Anaphylaxis)
		if p.PupilID == "pup_pri_maya" {
			if _, err := tx.Exec(`INSERT INTO pupil_medical_conditions (id, pupil_id, condition_name, condition_category, alert_level_id, has_ihcp, triggers, reaction_symptoms, emergency_response_plan, medication_name_and_dose, medication_location, parental_consent_given, effective_from)
				VALUES ('med_maya_anaphylaxis', 'pup_pri_maya', 'Severe Peanut & Tree Nut Anaphylaxis', 'ALLERGY', 'CRITICAL', 1, 'Ingestion or trace airborne/surface contact with peanuts or tree nuts', 'Airway constriction, facial swelling, hives, rapid drop in blood pressure', 'Administer Jext 150mcg adrenaline auto-injector immediately into outer thigh and dial 999.', 'Jext 150mcg Auto-Injector (x2)', 'Classroom red medical pouch + Medical Room Wall Cabinet B', 1, '2026-09-01')`); err != nil {
				return err
			}
		} else if idx%12 == 0 {
			medSeq++
			if _, err := tx.Exec(`INSERT INTO pupil_medical_conditions (id, pupil_id, condition_name, condition_category, alert_level_id, has_ihcp, triggers, reaction_symptoms, emergency_response_plan, medication_name_and_dose, medication_location, parental_consent_given, effective_from)
				VALUES (?, ?, 'Moderate Asthma', 'MEDICAL', 'MEDIUM', 0, 'Cold air, vigorous exercise, pollen', 'Wheezing, shortness of breath, persistent cough', 'Two puffs of Salbutamol reliever inhaler via spacer; sit upright and monitor.', 'Salbutamol 100mcg CFC-free Inhaler', 'Carried by pupil + Spare in Main Office', 1, '2026-09-01')`,
				fmt.Sprintf("med_%04d", medSeq), p.PupilID); err != nil {
				return err
			}
		}

		// Pastoral Notes (Alice Evans GDPR SAR Redaction fixture + ~120 across the school)
		if p.PupilID == "pup_yr7_alice" {
			if _, err := tx.Exec(`INSERT INTO pupil_pastoral_notes (id, pupil_id, category, title, note_text, created_by_staff_id)
				VALUES ('pas_alice_sar_redact', 'pup_yr7_alice', 'ACADEMIC_WELFARE', 'Lunchtime playground disagreement (SAR Redaction Fixture)', 'Alice Evans reported a disagreement at lunchtime involving Noah Williams (7A) and Zara Khan (7B) near the library. Ms H. Abbott spoke with all three pupils and resolved the matter amicably.', 'stf_tch_7a')`); err != nil {
				return err
			}
		} else if idx%9 == 0 {
			pasSeq++
			if _, err := tx.Exec(`INSERT INTO pupil_pastoral_notes (id, pupil_id, category, title, note_text, created_by_staff_id)
				VALUES (?, ?, 'GENERAL', 'Positive Settling-In Check', 'Pupil has settled well into autumn routines and contributed positively in tutor time.', 'stf_tch_7a')`,
				fmt.Sprintf("pas_%04d", pasSeq), p.PupilID); err != nil {
				return err
			}
		}
	}

	// 3. Seed Suspensions & Exclusions (`pupil_suspensions_exclusions`: Ryan Mitchell + 15 secondary pupils)
	if _, err := tx.Exec(`INSERT INTO pupil_suspensions_exclusions (id, pupil_id, academic_year_id, exclusion_category, reason_code_1, reason_code_2, start_date, end_date, sessions_missed, sen_provision_at_exclusion, governor_review_date, governor_review_outcome, notes)
		VALUES ('excl_ryan_01', 'pup_yr10_ryan', 'ay_2026_2027', 'SUSPENSION', 'DB', 'VA', '2026-09-16', '2026-09-17', 4, 'K', '2026-09-24', 'UPHELD', '2-day fixed-term suspension; reintegration meeting held with foster carer and social worker.')`); err != nil {
		return err
	}
	for i := 1; i <= 15; i++ {
		p := st.pupils[(i*45)%len(st.pupils)]
		if _, err := tx.Exec(`INSERT INTO pupil_suspensions_exclusions (id, pupil_id, academic_year_id, exclusion_category, reason_code_1, start_date, end_date, sessions_missed, sen_provision_at_exclusion, governor_review_outcome, notes)
			VALUES (?, ?, 'ay_2026_2027', 'SUSPENSION', 'DB', '2026-09-21', '2026-09-21', 2, ?, 'UPHELD', '1-day suspension for persistent disruptive behaviour.')`,
			fmt.Sprintf("excl_%03d", i), p.PupilID, p.SENCode); err != nil {
			return err
		}
	}

	// 4. Seed Alternative Provisions (`pupil_alternative_provisions`: Ryan Mitchell + 7 pupils)
	if _, err := tx.Exec(`INSERT INTO pupil_alternative_provisions (id, pupil_id, academic_year_id, provider_urn, provider_ukprn, ap_setting_type_code, ap_reason_code, sen_provision_during_ap, sessions_per_week, effective_from)
		VALUES ('ap_ryan_01', 'pup_yr10_ryan', 'ay_2026_2027', '139801', '10045678', 'FE', 'BHV', 'K', 4, '2026-09-07')`); err != nil {
		return err
	}
	for i := 1; i <= 7; i++ {
		p := st.pupils[(i*80+15)%len(st.pupils)]
		if _, err := tx.Exec(`INSERT INTO pupil_alternative_provisions (id, pupil_id, academic_year_id, provider_urn, ap_setting_type_code, ap_reason_code, sen_provision_during_ap, sessions_per_week, effective_from)
			VALUES (?, ?, 'ay_2026_2027', ?, 'OTH', 'MH', ?, 2, '2026-09-14')`,
			fmt.Sprintf("ap_%03d", i), p.PupilID, fmt.Sprintf("1398%02d", i+10), p.SENCode); err != nil {
			return err
		}
	}

	// 5. Seed Domain 06: KCSIE Envelope-Encrypted Safeguarding Vault (`safeguarding_notes` & `safeguarding_audit_logs`)
	sfgConcerns := []string{"EMOTIONAL", "NEGLECT", "ONLINE_SAFETY", "PEER_ON_PEER", "PHYSICAL", "CCE_CSE"}
	sfgStatuses := []string{"OPEN_TRIAGE", "UNDER_DSL_REVIEW", "REFERRED_MASH_LADO", "MONITORING", "CLOSED"}

	for i := 1; i <= 20; i++ {
		noteID := fmt.Sprintf("sfg_%03d", i)
		pupilID := st.pupils[(i*35)%len(st.pupils)].PupilID
		reporterID := st.teacherStaffIDs[i%len(st.teacherStaffIDs)]
		concern := sfgConcerns[(i-1)%len(sfgConcerns)]
		status := sfgStatuses[(i-1)%len(sfgStatuses)]
		incDate := autumn2026SchoolDays[(i-1)%len(autumn2026SchoolDays)]

		if i == 1 {
			noteID = "sfg_alice_001"
			pupilID = "pup_yr7_alice"
			reporterID = "stf_tch_7a"
			concern = "EMOTIONAL"
			status = "UNDER_DSL_REVIEW"
			incDate = "2026-09-10"
		}

		enc, err := encryptSafeguardingPayload(st.kmsKeyHandle, noteID, map[string]any{
			"note_id":          noteID,
			"pupil_id":         pupilID,
			"concern_category": concern,
			"summary":          "Confidential KCSIE safeguarding referral logged via write-only postbox and encrypted for DSL review.",
			"verbatim_account": "Pupil disclosed distress regarding an attempt by a restricted non-resident party to make contact.",
			"action_taken":     "DSL Mr M. Thorne notified immediately; refuge address lockdown verified.",
		})
		if err != nil {
			return err
		}

		if _, err := tx.Exec(`INSERT INTO safeguarding_notes (id, pupil_id, reported_by_staff_id, assigned_dsl_staff_id, concern_category, status, incident_date, kms_key_id, wrapped_dek, nonce, ciphertext_payload, retention_purge_eligible)
			VALUES (?, ?, ?, 'stf_dsl_01', ?, ?, ?, ?, ?, ?, ?, 0)`,
			noteID, pupilID, reporterID, concern, status, incDate, enc.KMSKeyID, enc.WrappedDEK, enc.Nonce, enc.CiphertextPayload); err != nil {
			return err
		}

		// Append-only audit logs for creation and DSL decryption view
		if _, err := tx.Exec(`INSERT INTO safeguarding_audit_logs (audit_id, note_id, pupil_id, actor_person_id, action_type, request_id, client_ip)
			VALUES (?, ?, ?, 'per_stf_tch_7a', 'CREATE_POSTBOX', ?, '10.24.0.15'),
			       (?, ?, ?, 'per_stf_dsl_01', 'DSL_DECRYPT_VIEW', ?, '10.24.0.10')`,
			fmt.Sprintf("sfga_%03d_create", i), noteID, pupilID, fmt.Sprintf("req_sfg_create_%03d", i),
			fmt.Sprintf("sfga_%03d_view", i), noteID, pupilID, fmt.Sprintf("req_sfg_view_%03d", i)); err != nil {
			return err
		}
	}

	return nil
}
