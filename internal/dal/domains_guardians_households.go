package dal

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// ============================================================================
// SQL Projections & Shared Field Constants
// ============================================================================

const siblingSummaryFieldsSQL = `
	DISTINCT
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
`

const householdSummaryFieldsSQL = `
	DISTINCT h.id, h.name, h.address_line_1, COALESCE(h.address_line_2, ''),
	         h.town_city, COALESCE(h.county, ''), h.postcode, h.is_confidential_address
`

// ============================================================================
// Internal Helpers & Row Scanners
// ============================================================================

func scanSiblingSummary(s rowScanner) (SiblingSummary, error) {
	var r SiblingSummary
	if err := s.Scan(
		&r.PupilID, &r.UPN, &r.FirstName, &r.LastName, &r.NCYearGroup,
		&r.SharedHouseholdID, &r.SharedGuardianPersonID,
	); err != nil {
		return SiblingSummary{}, err
	}
	return r, nil
}

func scanHouseholdSummary(s rowScanner) (HouseholdSummary, error) {
	var r HouseholdSummary
	var confInt int
	if err := s.Scan(
		&r.ID, &r.Name, &r.AddressLine1, &r.AddressLine2,
		&r.TownCity, &r.County, &r.Postcode, &confInt,
	); err != nil {
		return HouseholdSummary{}, err
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
	return r, nil
}

// ============================================================================
// Siblings Domain Types & Scoped Query Methods
// ============================================================================

type SiblingSummary struct {
	PupilID                string `json:"pupil_id"`
	UPN                    string `json:"upn"`
	FirstName              string `json:"first_name"`
	LastName               string `json:"last_name"`
	NCYearGroup            string `json:"nc_year_group,omitempty"`
	SharedHouseholdID      string `json:"shared_household_id,omitempty"`
	SharedGuardianPersonID string `json:"shared_guardian_person_id,omitempty"`
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

	query := fmt.Sprintf(`
		SELECT %s
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
		siblingSummaryFieldsSQL,
	)

	rows, err := d.readDB.QueryContext(
		ctx,
		query,
		scope.AcademicYearID, scope.ReferenceDate, scope.ReferenceDate, trimmedPupil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list siblings for pupil %s: %w", trimmedPupil, err)
	}
	defer rows.Close()

	result := make([]SiblingSummary, 0)
	for rows.Next() {
		r, err := scanSiblingSummary(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan sibling row: %w", err)
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// ============================================================================
// Households Domain Types & Scoped Query Methods
// ============================================================================

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

	query := fmt.Sprintf(`
		SELECT %s
		FROM households h WHERE 1=1`,
		householdSummaryFieldsSQL,
	)
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
		r, err := scanHouseholdSummary(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan household row: %w", err)
		}
		result = append(result, r)
	}
	return result, rows.Err()
}
