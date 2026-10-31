package dal

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
