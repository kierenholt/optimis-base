package dal

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type StaffPersona struct {
	StaffID     string   `json:"staff_id"`
	PersonID    string   `json:"person_id"`
	UserID      string   `json:"user_id"`
	StaffCode   string   `json:"staff_code"`
	DisplayName string   `json:"display_name"`
	JobTitle    string   `json:"job_title"`
	IsDSL       bool     `json:"is_dsl"`
	ActiveRoles []string `json:"active_roles"`
	Scopes      []string `json:"scopes"`
}

func formatPersonaDisplayName(title, prefFirst, legalFirst, prefLast, legalLast string) string {
	first := strings.TrimSpace(prefFirst)
	if first == "" {
		first = strings.TrimSpace(legalFirst)
	}
	last := strings.TrimSpace(prefLast)
	if last == "" {
		last = strings.TrimSpace(legalLast)
	}
	t := strings.TrimSpace(title)
	if t != "" {
		return fmt.Sprintf("%s %s %s", t, first, last)
	}
	return strings.TrimSpace(fmt.Sprintf("%s %s", first, last))
}

func (d *CoreDAL) resolveRolesAndScopesForPerson(ctx context.Context, personID string) ([]string, []string, error) {
	rows, err := d.readDB.QueryContext(
		ctx,
		`SELECT r.id, r.default_scopes_json
		 FROM person_roles pr
		 JOIN roles r ON r.id = pr.role_id
		 WHERE pr.person_id = ?
		   AND (pr.effective_to IS NULL OR pr.effective_to >= date('now'))
		 ORDER BY pr.is_primary DESC, r.id ASC`,
		personID,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query person_roles for %s: %w", personID, err)
	}
	defer rows.Close()

	var activeRoles []string
	var scopes []string
	seenScopes := make(map[string]struct{})

	for rows.Next() {
		var roleID, scopesJSON string
		if err := rows.Scan(&roleID, &scopesJSON); err != nil {
			return nil, nil, fmt.Errorf("failed to scan role row for %s: %w", personID, err)
		}
		activeRoles = append(activeRoles, roleID)

		var parsedScopes []string
		if err := json.Unmarshal([]byte(scopesJSON), &parsedScopes); err != nil {
			return nil, nil, fmt.Errorf("invalid default_scopes_json on role %s: %w", roleID, err)
		}
		for _, s := range parsedScopes {
			trimmed := strings.TrimSpace(s)
			if trimmed == "" {
				continue
			}
			if _, exists := seenScopes[trimmed]; !exists {
				seenScopes[trimmed] = struct{}{}
				scopes = append(scopes, trimmed)
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("error iterating roles for %s: %w", personID, err)
	}

	return activeRoles, scopes, nil
}

func (d *CoreDAL) ListActiveStaffPersonas(ctx context.Context) ([]StaffPersona, error) {
	rows, err := d.readDB.QueryContext(
		ctx,
		`SELECT stf.id, stf.person_id, COALESCE(per.external_sso_sub, ''), COALESCE(stf.staff_code, ''),
		        per.title, per.preferred_first_name, per.legal_first_name,
		        per.preferred_last_name, per.legal_last_name, stf.job_title, stf.is_dsl
		 FROM staff stf
		 JOIN persons per ON per.id = stf.person_id
		 WHERE per.status = 'ACTIVE'
		   AND (stf.termination_date IS NULL OR TRIM(stf.termination_date) = '')
		 ORDER BY stf.id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list active staff personas: %w", err)
	}

	type rawStaff struct {
		staffID   string
		personID  string
		ssoSub    string
		staffCode string
		title     string
		prefFirst string
		legFirst  string
		prefLast  string
		legLast   string
		jobTitle  string
		isDSL     int
	}
	var rawList []rawStaff
	for rows.Next() {
		var r rawStaff
		if err := rows.Scan(
			&r.staffID, &r.personID, &r.ssoSub, &r.staffCode,
			&r.title, &r.prefFirst, &r.legFirst,
			&r.prefLast, &r.legLast, &r.jobTitle, &r.isDSL,
		); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("failed to scan staff persona row: %w", err)
		}
		rawList = append(rawList, r)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	_ = rows.Close()

	personas := make([]StaffPersona, 0, len(rawList))
	for _, r := range rawList {
		activeRoles, scopes, err := d.resolveRolesAndScopesForPerson(ctx, r.personID)
		if err != nil {
			return nil, err
		}
		if len(activeRoles) == 0 {
			continue
		}
		userID := strings.TrimSpace(r.ssoSub)
		if userID == "" {
			userID = r.personID
		}
		personas = append(personas, StaffPersona{
			StaffID:     r.staffID,
			PersonID:    r.personID,
			UserID:      userID,
			StaffCode:   r.staffCode,
			DisplayName: formatPersonaDisplayName(r.title, r.prefFirst, r.legFirst, r.prefLast, r.legLast),
			JobTitle:    r.jobTitle,
			IsDSL:       r.isDSL == 1,
			ActiveRoles: activeRoles,
			Scopes:      scopes,
		})
	}

	return personas, nil
}

func (d *CoreDAL) ResolveStaffPersonaByID(ctx context.Context, staffID string) (*StaffPersona, error) {
	trimmedStaffID := strings.TrimSpace(staffID)
	if trimmedStaffID == "" {
		return nil, errors.New("missing mandatory parameter: staff_id")
	}

	var personID, ssoSub, staffCode, title, prefFirst, legFirst, prefLast, legLast, jobTitle, personStatus string
	var termDate sql.NullString
	var isDSL int

	err := d.readDB.QueryRowContext(
		ctx,
		`SELECT stf.person_id, COALESCE(per.external_sso_sub, ''), COALESCE(stf.staff_code, ''),
		        per.title, per.preferred_first_name, per.legal_first_name,
		        per.preferred_last_name, per.legal_last_name, stf.job_title, stf.is_dsl,
		        per.status, stf.termination_date
		 FROM staff stf
		 JOIN persons per ON per.id = stf.person_id
		 WHERE stf.id = ?`,
		trimmedStaffID,
	).Scan(
		&personID, &ssoSub, &staffCode,
		&title, &prefFirst, &legFirst,
		&prefLast, &legLast, &jobTitle, &isDSL,
		&personStatus, &termDate,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("staff persona not found: %s", trimmedStaffID)
		}
		return nil, fmt.Errorf("failed to query staff persona %s: %w", trimmedStaffID, err)
	}

	if personStatus != "ACTIVE" {
		return nil, fmt.Errorf("staff persona %s is not ACTIVE (status=%s)", trimmedStaffID, personStatus)
	}
	if termDate.Valid && strings.TrimSpace(termDate.String) != "" {
		return nil, fmt.Errorf("staff persona %s is terminated (termination_date=%s)", trimmedStaffID, termDate.String)
	}

	activeRoles, scopes, err := d.resolveRolesAndScopesForPerson(ctx, personID)
	if err != nil {
		return nil, err
	}
	if len(activeRoles) == 0 {
		return nil, fmt.Errorf("staff persona %s has no active roles", trimmedStaffID)
	}

	userID := strings.TrimSpace(ssoSub)
	if userID == "" {
		userID = personID
	}

	return &StaffPersona{
		StaffID:     trimmedStaffID,
		PersonID:    personID,
		UserID:      userID,
		StaffCode:   staffCode,
		DisplayName: formatPersonaDisplayName(title, prefFirst, legFirst, prefLast, legLast),
		JobTitle:    jobTitle,
		IsDSL:       isDSL == 1,
		ActiveRoles: activeRoles,
		Scopes:      scopes,
	}, nil
}
