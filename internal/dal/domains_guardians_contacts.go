package dal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// ============================================================================
// Guardian Contact Mutation Types & Inputs
// ============================================================================

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

// ============================================================================
// Internal Helpers
// ============================================================================

func upsertGuardianHouseholdTx(
	ctx context.Context,
	tx *sql.Tx,
	uid, pupilID, guardianPersonID, effFrom string,
	in UpsertGuardianContactInput,
	lastName string,
) (hhID, hhName, addr1, town, postcode string, err error) {
	hhID = strings.TrimSpace(in.HouseholdID)
	addr1 = strings.TrimSpace(in.AddressLine1)
	town = strings.TrimSpace(in.TownCity)
	postcode = strings.TrimSpace(in.Postcode)
	hhName = strings.TrimSpace(in.HouseholdName)
	if hhName == "" {
		hhName = lastName + " Household"
	}

	if hhID == "" && (addr1 == "" || town == "" || postcode == "") {
		return "", "", "", "", "", nil
	}

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
		return "", "", "", "", "", fmt.Errorf("failed to upsert household: %w", err)
	}

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO household_memberships (id, household_id, person_id, member_role, residency_type, effective_from)
		 VALUES (?, ?, ?, 'ADULT', 'PRIMARY', ?)
		 ON CONFLICT(household_id, person_id, effective_from) DO NOTHING`,
		"hm_"+uid, hhID, guardianPersonID, effFrom,
	); err != nil {
		return "", "", "", "", "", fmt.Errorf("failed to upsert household membership: %w", err)
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

	return hhID, hhName, addr1, town, postcode, nil
}

// ============================================================================
// Scoped Guardian Contact Upsert
// ============================================================================

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

	hhID, hhName, addr1, town, postcode, hhErr := upsertGuardianHouseholdTx(
		ctx, tx, uid, pupilID, guardianPersonID, effFrom, in, lastName,
	)
	if hhErr != nil {
		return nil, hhErr
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
