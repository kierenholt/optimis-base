package dal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type GuardianFilterOptions struct {
	RequireParentalResponsibility bool `json:"require_parental_responsibility"`
}

type GuardianContactSummary struct {
	RelationshipID                string `json:"relationship_id"`
	PupilID                       string `json:"pupil_id"`
	GuardianPersonID              string `json:"guardian_person_id"`
	RelationshipCode              string `json:"relationship_code"`
	FirstName                     string `json:"first_name"`
	LastName                      string `json:"last_name"`
	Email                         string `json:"email"`
	PhonePrimary                  string `json:"phone_primary"`
	HasParentalResponsibility     bool   `json:"has_parental_responsibility"`
	ContactPriority               int    `json:"contact_priority"`
	IsEmergencyContact            bool   `json:"is_emergency_contact"`
	AuthorisedToCollect           bool   `json:"authorised_to_collect"`
	ReceiveReports                bool   `json:"receive_reports"`
	HouseholdID                   string `json:"household_id"`
	HouseholdName                 string `json:"household_name"`
	AddressLine1                  string `json:"address_line_1"`
	TownCity                      string `json:"town_city"`
	Postcode                      string `json:"postcode"`
	IsConfidentialAddressRedacted bool   `json:"is_confidential_address_redacted"`
}

func (d *CoreDAL) ListGuardiansForPupilScoped(
	ctx context.Context,
	scope DALQueryScope,
	pupilID string,
	opts GuardianFilterOptions,
) ([]GuardianContactSummary, error) {
	if strings.TrimSpace(pupilID) == "" {
		return nil, errors.New("missing mandatory parameter: pupil_id")
	}
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	if err := d.ensurePupilInOwnGroups(ctx, scope, pupilID); err != nil {
		return nil, err
	}

	query := `
		SELECT gr.id, gr.pupil_id, gr.guardian_person_id, gr.relationship_code,
		       per.legal_first_name, per.legal_last_name, COALESCE(per.email, ''), per.phone_primary,
		       gr.has_parental_responsibility, gr.contact_priority,
		       gr.is_emergency_contact, gr.authorised_to_collect, gr.receive_reports,
		       COALESCE(h.id, ''), COALESCE(h.name, ''), COALESCE(h.address_line_1, ''),
		       COALESCE(h.town_city, ''), COALESCE(h.postcode, ''), COALESCE(h.is_confidential_address, 0)
		FROM guardian_relationships gr
		JOIN persons per ON per.id = gr.guardian_person_id
		LEFT JOIN household_memberships hm ON hm.person_id = gr.guardian_person_id
		  AND hm.effective_from <= ?
		  AND (hm.effective_to IS NULL OR hm.effective_to >= ?)
		LEFT JOIN households h ON h.id = hm.household_id
		WHERE gr.pupil_id = ?
		  AND gr.court_order_restricted = 0
		  AND gr.effective_from <= ?
		  AND (gr.effective_to IS NULL OR gr.effective_to >= ?)
	`
	args := []any{
		scope.ReferenceDate, scope.ReferenceDate,
		pupilID,
		scope.ReferenceDate, scope.ReferenceDate,
	}

	if opts.RequireParentalResponsibility {
		query += ` AND gr.has_parental_responsibility = 1`
	}
	query += ` ORDER BY gr.contact_priority ASC, gr.id ASC`

	rows, err := d.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query scoped guardians: %w", err)
	}
	defer rows.Close()

	var result []GuardianContactSummary
	for rows.Next() {
		var g GuardianContactSummary
		var hasPRInt, isEmergInt, authCollectInt, recReportsInt, isConfAddrInt int
		if err := rows.Scan(
			&g.RelationshipID,
			&g.PupilID,
			&g.GuardianPersonID,
			&g.RelationshipCode,
			&g.FirstName,
			&g.LastName,
			&g.Email,
			&g.PhonePrimary,
			&hasPRInt,
			&g.ContactPriority,
			&isEmergInt,
			&authCollectInt,
			&recReportsInt,
			&g.HouseholdID,
			&g.HouseholdName,
			&g.AddressLine1,
			&g.TownCity,
			&g.Postcode,
			&isConfAddrInt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan guardian contact summary: %w", err)
		}
		g.HasParentalResponsibility = hasPRInt == 1
		g.IsEmergencyContact = isEmergInt == 1
		g.AuthorisedToCollect = authCollectInt == 1
		g.ReceiveReports = recReportsInt == 1
		if isConfAddrInt == 1 {
			g.HouseholdName = ""
			g.AddressLine1 = ""
			g.TownCity = ""
			g.Postcode = ""
			g.IsConfidentialAddressRedacted = true
		}
		result = append(result, g)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating guardian contacts: %w", err)
	}
	return result, nil
}

func (d *CoreDAL) ListGuardiansForPupilsBatchScoped(
	ctx context.Context,
	scope DALQueryScope,
	pupilIDs []string,
	opts GuardianFilterOptions,
) (map[string][]GuardianContactSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	if len(pupilIDs) == 0 {
		return map[string][]GuardianContactSummary{}, nil
	}

	out := make(map[string][]GuardianContactSummary, len(pupilIDs))
	placeholders := make([]string, 0, len(pupilIDs))
	args := make([]any, 0, len(pupilIDs)+4)
	args = append(args, scope.ReferenceDate, scope.ReferenceDate)

	for _, rawID := range pupilIDs {
		pid := strings.TrimSpace(rawID)
		if pid == "" {
			return nil, errors.New("pupilIDs contains an empty pupil_id")
		}
		if scope.Mode == ScopeOwnGroups {
			if err := d.ensurePupilInOwnGroups(ctx, scope, pid); err != nil {
				return nil, err
			}
		}
		if _, exists := out[pid]; !exists {
			out[pid] = []GuardianContactSummary{}
			placeholders = append(placeholders, "?")
			args = append(args, pid)
		}
	}

	args = append(args, scope.ReferenceDate, scope.ReferenceDate)

	query := fmt.Sprintf(`
		SELECT gr.id, gr.pupil_id, gr.guardian_person_id, gr.relationship_code,
		       per.legal_first_name, per.legal_last_name, COALESCE(per.email, ''), per.phone_primary,
		       gr.has_parental_responsibility, gr.contact_priority,
		       gr.is_emergency_contact, gr.authorised_to_collect, gr.receive_reports,
		       COALESCE(h.id, ''), COALESCE(h.name, ''), COALESCE(h.address_line_1, ''),
		       COALESCE(h.town_city, ''), COALESCE(h.postcode, ''), COALESCE(h.is_confidential_address, 0)
		FROM guardian_relationships gr
		JOIN persons per ON per.id = gr.guardian_person_id
		LEFT JOIN household_memberships hm ON hm.person_id = gr.guardian_person_id
		  AND hm.effective_from <= ?
		  AND (hm.effective_to IS NULL OR hm.effective_to >= ?)
		LEFT JOIN households h ON h.id = hm.household_id
		WHERE gr.pupil_id IN (%s)
		  AND gr.court_order_restricted = 0
		  AND gr.effective_from <= ?
		  AND (gr.effective_to IS NULL OR gr.effective_to >= ?)
	`, strings.Join(placeholders, ","))

	if opts.RequireParentalResponsibility {
		query += ` AND gr.has_parental_responsibility = 1`
	}
	query += ` ORDER BY gr.pupil_id ASC, gr.contact_priority ASC, gr.id ASC`

	rows, err := d.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to batch query scoped guardians: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var g GuardianContactSummary
		var hasPRInt, isEmergInt, authCollectInt, recReportsInt, isConfAddrInt int
		if err := rows.Scan(
			&g.RelationshipID,
			&g.PupilID,
			&g.GuardianPersonID,
			&g.RelationshipCode,
			&g.FirstName,
			&g.LastName,
			&g.Email,
			&g.PhonePrimary,
			&hasPRInt,
			&g.ContactPriority,
			&isEmergInt,
			&authCollectInt,
			&recReportsInt,
			&g.HouseholdID,
			&g.HouseholdName,
			&g.AddressLine1,
			&g.TownCity,
			&g.Postcode,
			&isConfAddrInt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan batch guardian contact summary: %w", err)
		}
		g.HasParentalResponsibility = hasPRInt == 1
		g.IsEmergencyContact = isEmergInt == 1
		g.AuthorisedToCollect = authCollectInt == 1
		g.ReceiveReports = recReportsInt == 1
		if isConfAddrInt == 1 {
			g.HouseholdName = ""
			g.AddressLine1 = ""
			g.TownCity = ""
			g.Postcode = ""
			g.IsConfidentialAddressRedacted = true
		}
		out[g.PupilID] = append(out[g.PupilID], g)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating batch guardian contacts: %w", err)
	}
	return out, nil
}

type UpsertGuardianContactInput struct {
	RelationshipID            string `json:"relationship_id,omitempty"`
	PupilID                   string `json:"pupil_id"`
	GuardianPersonID          string `json:"guardian_person_id,omitempty"`
	RelationshipCode          string `json:"relationship_code"`
	FirstName                 string `json:"first_name"`
	LastName                  string `json:"last_name"`
	Email                     string `json:"email,omitempty"`
	PhonePrimary              string `json:"phone_primary,omitempty"`
	HasParentalResponsibility bool   `json:"has_parental_responsibility"`
	CourtOrderRestricted      bool   `json:"court_order_restricted,omitempty"`
	ContactPriority           int    `json:"contact_priority,omitempty"`
	IsEmergencyContact        *bool  `json:"is_emergency_contact,omitempty"`
	AuthorisedToCollect       *bool  `json:"authorised_to_collect,omitempty"`
	ReceiveReports            *bool  `json:"receive_reports,omitempty"`
	HouseholdID               string `json:"household_id,omitempty"`
	HouseholdName             string `json:"household_name,omitempty"`
	AddressLine1              string `json:"address_line_1,omitempty"`
	TownCity                  string `json:"town_city,omitempty"`
	Postcode                  string `json:"postcode,omitempty"`
	IsConfidentialAddress     bool   `json:"is_confidential_address,omitempty"`
	EffectiveFrom             string `json:"effective_from"`
}

func (d *CoreDAL) UpsertGuardianContactScoped(
	ctx context.Context,
	scope DALQueryScope,
	in UpsertGuardianContactInput,
) (*GuardianContactSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	pupilID := strings.TrimSpace(in.PupilID)
	firstName := strings.TrimSpace(in.FirstName)
	lastName := strings.TrimSpace(in.LastName)
	relCode := strings.ToUpper(strings.TrimSpace(in.RelationshipCode))
	if relCode == "" {
		relCode = "OTHER"
	}
	effFrom := strings.TrimSpace(in.EffectiveFrom)
	if effFrom == "" {
		effFrom = scope.ReferenceDate
	}
	if pupilID == "" {
		return nil, errors.New("missing mandatory parameter: pupil_id")
	}
	if firstName == "" || lastName == "" {
		return nil, errors.New("missing mandatory parameter: first_name and last_name")
	}
	if err := d.ensurePupilInOwnGroups(ctx, scope, pupilID); err != nil {
		return nil, err
	}

	uid := strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	guardianPersonID := strings.TrimSpace(in.GuardianPersonID)
	if guardianPersonID == "" {
		guardianPersonID = "per_gdn_" + uid
	}
	relID := strings.TrimSpace(in.RelationshipID)
	if relID == "" {
		relID = "grel_" + uid
	}

	priority := in.ContactPriority
	if priority < 1 || priority > 10 {
		priority = 1
	}

	tx, err := d.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin UpsertGuardianContactScoped tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// Check if an existing relationship is court-order restricted
	var existingRestricted int
	err = tx.QueryRowContext(
		ctx,
		`SELECT court_order_restricted FROM guardian_relationships WHERE id = ? OR (pupil_id = ? AND guardian_person_id = ?) LIMIT 1`,
		relID, pupilID, guardianPersonID,
	).Scan(&existingRestricted)
	isRestricted := in.CourtOrderRestricted || (err == nil && existingRestricted == 1)

	var emailArg any
	if strings.TrimSpace(in.Email) != "" {
		emailArg = strings.TrimSpace(in.Email)
	}
	phone := strings.TrimSpace(in.PhonePrimary)

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO persons (id, email, legal_first_name, legal_last_name, preferred_first_name, preferred_last_name, phone_primary, status)
		 VALUES (?, ?, ?, ?, ?, ?, ?, 'ACTIVE')
		 ON CONFLICT(id) DO UPDATE SET
		     email = COALESCE(excluded.email, persons.email),
		     legal_first_name = excluded.legal_first_name,
		     legal_last_name = excluded.legal_last_name,
		     preferred_first_name = excluded.preferred_first_name,
		     preferred_last_name = excluded.preferred_last_name,
		     phone_primary = excluded.phone_primary`,
		guardianPersonID, emailArg, firstName, lastName, firstName, lastName, phone,
	); err != nil {
		return nil, fmt.Errorf("failed to upsert guardian person: %w", err)
	}

	hhID := strings.TrimSpace(in.HouseholdID)
	addr1 := strings.TrimSpace(in.AddressLine1)
	town := strings.TrimSpace(in.TownCity)
	postcode := strings.TrimSpace(in.Postcode)
	hhName := strings.TrimSpace(in.HouseholdName)
	if hhName == "" {
		hhName = lastName + " Household"
	}

	if hhID != "" || (addr1 != "" && town != "" && postcode != "") {
		if hhID == "" {
			hhID = "hh_" + uid
		}
		confInt := 0
		if in.IsConfidentialAddress {
			confInt = 1
		}
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO households (id, name, address_line_1, town_city, postcode, is_confidential_address)
			 VALUES (?, ?, ?, ?, ?, ?)
			 ON CONFLICT(id) DO UPDATE SET
			     name = excluded.name,
			     address_line_1 = excluded.address_line_1,
			     town_city = excluded.town_city,
			     postcode = excluded.postcode,
			     is_confidential_address = excluded.is_confidential_address`,
			hhID, hhName, addr1, town, postcode, confInt,
		); err != nil {
			return nil, fmt.Errorf("failed to upsert household: %w", err)
		}
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO household_memberships (id, household_id, person_id, member_role, residency_type, effective_from)
			 VALUES (?, ?, ?, 'ADULT', 'PRIMARY', ?)
			 ON CONFLICT(household_id, person_id, effective_from) DO NOTHING`,
			"hm_"+uid, hhID, guardianPersonID, effFrom,
		); err != nil {
			return nil, fmt.Errorf("failed to upsert household membership: %w", err)
		}
		var pupilPersonID string
		if err := tx.QueryRowContext(ctx, `SELECT person_id FROM pupils WHERE id = ?`, pupilID).Scan(&pupilPersonID); err == nil && pupilPersonID != "" {
			_, _ = tx.ExecContext(
				ctx,
				`INSERT INTO household_memberships (id, household_id, person_id, member_role, residency_type, effective_from)
				 VALUES (?, ?, ?, 'CHILD', 'PRIMARY', ?)
				 ON CONFLICT(household_id, person_id, effective_from) DO NOTHING`,
				"hmc_"+uid, hhID, pupilPersonID, effFrom,
			)
		}
	}

	hasPRInt := 0
	if in.HasParentalResponsibility {
		hasPRInt = 1
	}
	restrictedInt := 0
	isEmergInt := 1
	if in.IsEmergencyContact != nil && !*in.IsEmergencyContact {
		isEmergInt = 0
	}
	authCollectInt := 1
	if in.AuthorisedToCollect != nil && !*in.AuthorisedToCollect {
		authCollectInt = 0
	}
	recReportsInt := 1
	if in.ReceiveReports != nil && !*in.ReceiveReports {
		recReportsInt = 0
	}
	portalInt, phoneInt, smsInt, emailInt, postInt := 1, 1, 1, 1, 1
	if isRestricted {
		restrictedInt = 1
		isEmergInt = 0
		authCollectInt = 0
		recReportsInt = 0
		portalInt, phoneInt, smsInt, emailInt, postInt = 0, 0, 0, 0, 0
	}

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO guardian_relationships (
			id, pupil_id, guardian_person_id, relationship_code,
			has_parental_responsibility, court_order_restricted, contact_priority,
			is_emergency_contact, authorised_to_collect, portal_data_access, receive_reports,
			contact_by_phone, contact_by_sms, contact_by_email, contact_by_post,
			effective_from, effective_to
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL)
		ON CONFLICT(pupil_id, guardian_person_id, effective_from) DO UPDATE SET
			relationship_code = excluded.relationship_code,
			has_parental_responsibility = excluded.has_parental_responsibility,
			court_order_restricted = excluded.court_order_restricted,
			contact_priority = excluded.contact_priority,
			is_emergency_contact = excluded.is_emergency_contact,
			authorised_to_collect = excluded.authorised_to_collect,
			portal_data_access = excluded.portal_data_access,
			receive_reports = excluded.receive_reports,
			contact_by_phone = excluded.contact_by_phone,
			contact_by_sms = excluded.contact_by_sms,
			contact_by_email = excluded.contact_by_email,
			contact_by_post = excluded.contact_by_post`,
		relID, pupilID, guardianPersonID, relCode,
		hasPRInt, restrictedInt, priority,
		isEmergInt, authCollectInt, portalInt, recReportsInt,
		phoneInt, smsInt, emailInt, postInt,
		effFrom,
	); err != nil {
		return nil, fmt.Errorf("failed to upsert guardian_relationships row: %w", err)
	}

	var persistedRelID string
	if err := tx.QueryRowContext(
		ctx,
		`SELECT id FROM guardian_relationships WHERE pupil_id = ? AND guardian_person_id = ? AND effective_from = ?`,
		pupilID, guardianPersonID, effFrom,
	).Scan(&persistedRelID); err == nil && persistedRelID != "" {
		relID = persistedRelID
	} else if !errors.Is(err, sql.ErrNoRows) && err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit UpsertGuardianContactScoped tx: %w", err)
	}

	outAddr1 := addr1
	outTown := town
	outPostcode := postcode
	outHHName := hhName
	if in.IsConfidentialAddress {
		outAddr1 = ""
		outTown = ""
		outPostcode = ""
		outHHName = ""
	}

	return &GuardianContactSummary{
		RelationshipID:                relID,
		PupilID:                       pupilID,
		GuardianPersonID:              guardianPersonID,
		RelationshipCode:              relCode,
		FirstName:                     firstName,
		LastName:                      lastName,
		Email:                         strings.TrimSpace(in.Email),
		PhonePrimary:                  phone,
		HasParentalResponsibility:     in.HasParentalResponsibility,
		ContactPriority:               priority,
		IsEmergencyContact:            isEmergInt == 1,
		AuthorisedToCollect:           authCollectInt == 1,
		ReceiveReports:                recReportsInt == 1,
		HouseholdID:                   hhID,
		HouseholdName:                 outHHName,
		AddressLine1:                  outAddr1,
		TownCity:                      outTown,
		Postcode:                      outPostcode,
		IsConfidentialAddressRedacted: in.IsConfidentialAddress,
	}, nil
}

type SiblingSummary struct {
	PupilID                 string `json:"pupil_id"`
	UPN                     string `json:"upn"`
	FirstName               string `json:"first_name"`
	LastName                string `json:"last_name"`
	NCYearGroup             string `json:"nc_year_group,omitempty"`
	SharedHouseholdID       string `json:"shared_household_id,omitempty"`
	SharedGuardianPersonID  string `json:"shared_guardian_person_id,omitempty"`
}

func (d *CoreDAL) ListSiblingsForPupilScoped(
	ctx context.Context,
	scope DALQueryScope,
	pupilID string,
) ([]SiblingSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	trimmedPupil := strings.TrimSpace(pupilID)
	if trimmedPupil == "" {
		return nil, errors.New("missing mandatory parameter: pupil_id")
	}
	if scope.Mode == ScopeOwnGroups {
		if err := d.ensurePupilInOwnGroups(ctx, scope, trimmedPupil); err != nil {
			return nil, err
		}
	}

	rows, err := d.readDB.QueryContext(
		ctx,
		`SELECT DISTINCT
			sib.id,
			sib.upn,
			COALESCE(NULLIF(TRIM(per.preferred_first_name), ''), per.legal_first_name),
			COALESCE(NULLIF(TRIM(per.preferred_last_name), ''), per.legal_last_name),
			COALESCE(enr.nc_year_group, ''),
			COALESCE((
				SELECT hm1.household_id
				FROM household_memberships hm1
				JOIN household_memberships hm2 ON hm2.household_id = hm1.household_id
				WHERE (hm1.person_id = p_src.person_id OR hm1.person_id IN (SELECT guardian_person_id FROM guardian_relationships WHERE pupil_id = p_src.id AND court_order_restricted = 0))
				  AND (hm2.person_id = sib.person_id OR hm2.person_id IN (SELECT guardian_person_id FROM guardian_relationships WHERE pupil_id = sib.id AND court_order_restricted = 0))
				LIMIT 1
			), ''),
			COALESCE((
				SELECT gr1.guardian_person_id
				FROM guardian_relationships gr1
				JOIN guardian_relationships gr2 ON gr2.guardian_person_id = gr1.guardian_person_id
				WHERE gr1.pupil_id = p_src.id
				  AND gr2.pupil_id = sib.id
				  AND gr1.court_order_restricted = 0
				  AND gr2.court_order_restricted = 0
				LIMIT 1
			), '')
		 FROM pupils p_src
		 JOIN pupils sib ON sib.id != p_src.id
		 JOIN persons per ON per.id = sib.person_id
		 LEFT JOIN pupil_year_enrolments enr ON enr.pupil_id = sib.id
		   AND enr.academic_year_id = ?
		   AND enr.effective_from <= ?
		   AND (enr.effective_to IS NULL OR enr.effective_to >= ?)
		 WHERE p_src.id = ?
		   AND (
		       EXISTS (
		           SELECT 1 FROM guardian_relationships gr1
		           JOIN guardian_relationships gr2 ON gr2.guardian_person_id = gr1.guardian_person_id
		           WHERE gr1.pupil_id = p_src.id
		             AND gr2.pupil_id = sib.id
		             AND gr1.court_order_restricted = 0
		             AND gr2.court_order_restricted = 0
		       )
		       OR EXISTS (
		           SELECT 1 FROM household_memberships hm1
		           JOIN household_memberships hm2 ON hm2.household_id = hm1.household_id
		           WHERE hm1.person_id = p_src.person_id
		             AND hm2.person_id = sib.person_id
		       )
		   )
		 ORDER BY per.legal_last_name ASC, per.legal_first_name ASC, sib.id ASC`,
		scope.AcademicYearID, scope.ReferenceDate, scope.ReferenceDate, trimmedPupil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list siblings for pupil %s: %w", trimmedPupil, err)
	}
	defer rows.Close()

	result := make([]SiblingSummary, 0)
	for rows.Next() {
		var r SiblingSummary
		if err := rows.Scan(
			&r.PupilID, &r.UPN, &r.FirstName, &r.LastName, &r.NCYearGroup,
			&r.SharedHouseholdID, &r.SharedGuardianPersonID,
		); err != nil {
			return nil, fmt.Errorf("failed to scan sibling row: %w", err)
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

type HouseholdSummary struct {
	ID                            string `json:"id"`
	HouseholdID                   string `json:"household_id"`
	Name                          string `json:"name"`
	HouseholdName                 string `json:"household_name"`
	AddressLine1                  string `json:"address_line_1"`
	AddressLine2                  string `json:"address_line_2,omitempty"`
	TownCity                      string `json:"town_city"`
	County                        string `json:"county,omitempty"`
	Postcode                      string `json:"postcode"`
	IsConfidentialAddressRedacted bool   `json:"is_confidential_address_redacted"`
}

func (d *CoreDAL) ListHouseholdsScoped(
	ctx context.Context,
	scope DALQueryScope,
	pupilID string,
) ([]HouseholdSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	trimmedPupil := strings.TrimSpace(pupilID)
	if trimmedPupil != "" && scope.Mode == ScopeOwnGroups {
		if err := d.ensurePupilInOwnGroups(ctx, scope, trimmedPupil); err != nil {
			return nil, err
		}
	}

	query := `
		SELECT DISTINCT h.id, h.name, h.address_line_1, COALESCE(h.address_line_2, ''),
		                h.town_city, COALESCE(h.county, ''), h.postcode, h.is_confidential_address
		FROM households h WHERE 1=1`
	var args []any
	if trimmedPupil != "" {
		query += ` AND EXISTS (
			SELECT 1 FROM household_memberships hm
			WHERE hm.household_id = h.id
			  AND (
			      hm.person_id = (SELECT person_id FROM pupils WHERE id = ?)
			      OR hm.person_id IN (SELECT guardian_person_id FROM guardian_relationships WHERE pupil_id = ? AND court_order_restricted = 0)
			  )
		)`
		args = append(args, trimmedPupil, trimmedPupil)
	}
	query += ` ORDER BY h.id ASC`

	rows, err := d.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list households: %w", err)
	}
	defer rows.Close()

	result := make([]HouseholdSummary, 0)
	for rows.Next() {
		var r HouseholdSummary
		var confInt int
		if err := rows.Scan(
			&r.ID, &r.Name, &r.AddressLine1, &r.AddressLine2,
			&r.TownCity, &r.County, &r.Postcode, &confInt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan household row: %w", err)
		}
		r.HouseholdID = r.ID
		r.HouseholdName = r.Name
		if confInt == 1 {
			r.IsConfidentialAddressRedacted = true
			r.Name = ""
			r.HouseholdName = ""
			r.AddressLine1 = ""
			r.AddressLine2 = ""
			r.TownCity = ""
			r.County = ""
			r.Postcode = ""
		}
		result = append(result, r)
	}
	return result, rows.Err()
}


