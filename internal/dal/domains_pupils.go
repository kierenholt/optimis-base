package dal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type Pupil struct {
	ID                 string  `json:"id"`
	PupilID            string  `json:"pupil_id,omitempty"`
	UPN                string  `json:"upn"`
	FirstName          string  `json:"first_name"`
	LastName           string  `json:"last_name"`
	LegalFirstName     string  `json:"legal_first_name,omitempty"`
	LegalLastName      string  `json:"legal_last_name,omitempty"`
	PreferredFirstName string  `json:"preferred_first_name,omitempty"`
	PreferredLastName  string  `json:"preferred_last_name,omitempty"`
	NCYearGroup        string  `json:"nc_year_group,omitempty"`
	EnrolmentStatus    string  `json:"enrolment_status,omitempty"`
	AcademicYearID     string  `json:"academic_year_id"`
	EffectiveFrom      string  `json:"effective_from"`
	EffectiveTo        *string `json:"effective_to,omitempty"`
}

type SeedPupilInput struct {
	UPN            string `json:"upn"`
	FirstName      string `json:"first_name"`
	LastName       string `json:"last_name"`
	AcademicYearID string `json:"academic_year_id"`
	EffectiveFrom  string `json:"effective_from"`
}

func (d *CoreDAL) SeedPupil(ctx context.Context, in SeedPupilInput) (*Pupil, error) {
	if strings.TrimSpace(in.UPN) == "" {
		return nil, errors.New("missing mandatory field: upn")
	}
	if strings.TrimSpace(in.FirstName) == "" {
		return nil, errors.New("missing mandatory field: first_name")
	}
	if strings.TrimSpace(in.LastName) == "" {
		return nil, errors.New("missing mandatory field: last_name")
	}
	if strings.TrimSpace(in.AcademicYearID) == "" {
		return nil, errors.New("missing mandatory field: academic_year_id")
	}
	if strings.TrimSpace(in.EffectiveFrom) == "" {
		return nil, errors.New("missing mandatory field: effective_from")
	}

	uid := strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	personID := "per_" + uid
	pupilID := "pup_" + uid
	enrolmentID := "enr_" + uid

	tx, err := d.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin SeedPupil transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO academic_years (id, name, status, sequence_number, first_day, last_day)
		 VALUES (?, ?, 'CURRENT', (SELECT COALESCE(MAX(sequence_number), 0) + 1 FROM academic_years), ?, '2099-08-31')
		 ON CONFLICT(id) DO NOTHING`,
		in.AcademicYearID, in.AcademicYearID, in.EffectiveFrom,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure academic_year %s: %w", in.AcademicYearID, err)
	}

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO persons (id, legal_first_name, legal_last_name, sex, status)
		 VALUES (?, ?, ?, 'U', 'ACTIVE')`,
		personID, in.FirstName, in.LastName,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert person for pupil: %w", err)
	}

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO pupils (id, person_id, upn)
		 VALUES (?, ?, ?)`,
		pupilID, personID, in.UPN,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert pupil: %w", err)
	}

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO pupil_year_enrolments (id, pupil_id, academic_year_id, nc_year_group, effective_from, effective_to)
		 VALUES (?, ?, ?, '7', ?, NULL)`,
		enrolmentID, pupilID, in.AcademicYearID, in.EffectiveFrom,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert pupil_year_enrolment: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit SeedPupil transaction: %w", err)
	}

	d.emitPostCommitEvent(ctx, EventCorePupilEnrolled, map[string]any{
		"pupil_id":         pupilID,
		"upn":              in.UPN,
		"academic_year_id": in.AcademicYearID,
		"effective_from":   in.EffectiveFrom,
	})

	return &Pupil{
		ID:                 pupilID,
		PupilID:            pupilID,
		UPN:                in.UPN,
		FirstName:          in.FirstName,
		LastName:           in.LastName,
		LegalFirstName:     in.FirstName,
		LegalLastName:      in.LastName,
		PreferredFirstName: in.FirstName,
		PreferredLastName:  in.LastName,
		NCYearGroup:        "7",
		EnrolmentStatus:    "C",
		AcademicYearID:     in.AcademicYearID,
		EffectiveFrom:      in.EffectiveFrom,
	}, nil
}

func (d *CoreDAL) DeletePupil(ctx context.Context, pupilID string) error {
	if strings.TrimSpace(pupilID) == "" {
		return errors.New("missing mandatory parameter: pupil_id")
	}

	tx, err := d.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin DeletePupil transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var personID string
	err = tx.QueryRowContext(ctx, `SELECT person_id FROM pupils WHERE id = ?`, pupilID).Scan(&personID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to look up person_id for pupil %s: %w", pupilID, err)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM pupil_year_enrolments WHERE pupil_id = ?`, pupilID); err != nil {
		return fmt.Errorf("failed to delete pupil_year_enrolments: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM pupils WHERE id = ?`, pupilID); err != nil {
		return fmt.Errorf("failed to delete pupil: %w", err)
	}
	if personID != "" {
		if _, err := tx.ExecContext(ctx, `DELETE FROM persons WHERE id = ?`, personID); err != nil {
			return fmt.Errorf("failed to delete person: %w", err)
		}
	}

	return tx.Commit()
}

func (d *CoreDAL) GetPupilByID(ctx context.Context, pupilID string) (*Pupil, error) {
	if strings.TrimSpace(pupilID) == "" {
		return nil, errors.New("missing mandatory parameter: pupil_id")
	}

	row := d.readDB.QueryRowContext(
		ctx,
		`SELECT p.id, p.upn, per.legal_first_name, per.legal_last_name,
		        COALESCE(NULLIF(TRIM(per.preferred_first_name), ''), per.legal_first_name),
		        COALESCE(NULLIF(TRIM(per.preferred_last_name), ''), per.legal_last_name),
		        COALESCE(enr.nc_year_group, ''), COALESCE(enr.enrolment_status, 'C'),
		        enr.academic_year_id, enr.effective_from, enr.effective_to
		 FROM pupils p
		 JOIN persons per ON per.id = p.person_id
		 JOIN pupil_year_enrolments enr ON enr.pupil_id = p.id
		 WHERE p.id = ? AND (enr.effective_to IS NULL OR enr.effective_to >= date('now'))
		 ORDER BY enr.effective_from DESC
		 LIMIT 1`,
		pupilID,
	)

	var p Pupil
	var effTo sql.NullString
	if err := row.Scan(
		&p.ID, &p.UPN, &p.FirstName, &p.LastName,
		&p.PreferredFirstName, &p.PreferredLastName,
		&p.NCYearGroup, &p.EnrolmentStatus,
		&p.AcademicYearID, &p.EffectiveFrom, &effTo,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("pupil not found: %s", pupilID)
		}
		return nil, fmt.Errorf("failed to query pupil: %w", err)
	}
	p.PupilID = p.ID
	p.LegalFirstName = p.FirstName
	p.LegalLastName = p.LastName
	if effTo.Valid {
		val := effTo.String
		p.EffectiveTo = &val
	}
	return &p, nil
}

func (d *CoreDAL) GetPupilByIDScoped(ctx context.Context, scope DALQueryScope, pupilID string) (*Pupil, error) {
	if strings.TrimSpace(pupilID) == "" {
		return nil, errors.New("missing mandatory parameter: pupil_id")
	}
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	if err := d.ensurePupilInOwnGroups(ctx, scope, pupilID); err != nil {
		return nil, err
	}

	row := d.readDB.QueryRowContext(
		ctx,
		`SELECT p.id, p.upn, per.legal_first_name, per.legal_last_name,
		        COALESCE(NULLIF(TRIM(per.preferred_first_name), ''), per.legal_first_name),
		        COALESCE(NULLIF(TRIM(per.preferred_last_name), ''), per.legal_last_name),
		        COALESCE(enr.nc_year_group, ''), COALESCE(enr.enrolment_status, 'C'),
		        enr.academic_year_id, enr.effective_from, enr.effective_to
		 FROM pupils p
		 JOIN persons per ON per.id = p.person_id
		 JOIN pupil_year_enrolments enr ON enr.pupil_id = p.id
		 WHERE p.id = ?
		   AND enr.academic_year_id = ?
		   AND enr.effective_from <= ?
		   AND (enr.effective_to IS NULL OR enr.effective_to >= ?)
		 ORDER BY enr.effective_from DESC
		 LIMIT 1`,
		pupilID, scope.AcademicYearID, scope.ReferenceDate, scope.ReferenceDate,
	)

	var p Pupil
	var effTo sql.NullString
	if err := row.Scan(
		&p.ID, &p.UPN, &p.FirstName, &p.LastName,
		&p.PreferredFirstName, &p.PreferredLastName,
		&p.NCYearGroup, &p.EnrolmentStatus,
		&p.AcademicYearID, &p.EffectiveFrom, &effTo,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("pupil not found or not enrolled on %s", scope.ReferenceDate)
		}
		return nil, fmt.Errorf("failed to query scoped pupil: %w", err)
	}
	p.PupilID = p.ID
	p.LegalFirstName = p.FirstName
	p.LegalLastName = p.LastName
	if effTo.Valid {
		val := effTo.String
		p.EffectiveTo = &val
	}
	return &p, nil
}

func (d *CoreDAL) GetPupilsByIDsBatchScoped(ctx context.Context, scope DALQueryScope, pupilIDs []string) ([]Pupil, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	if len(pupilIDs) == 0 {
		return []Pupil{}, nil
	}

	cleaned := make([]string, 0, len(pupilIDs))
	placeholders := make([]string, 0, len(pupilIDs))
	args := make([]any, 0, len(pupilIDs)+4)
	args = append(args, scope.AcademicYearID, scope.ReferenceDate, scope.ReferenceDate)

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
		cleaned = append(cleaned, pid)
		placeholders = append(placeholders, "?")
		args = append(args, pid)
	}

	query := fmt.Sprintf(`
		SELECT p.id, p.upn, per.legal_first_name, per.legal_last_name,
		       COALESCE(NULLIF(TRIM(per.preferred_first_name), ''), per.legal_first_name),
		       COALESCE(NULLIF(TRIM(per.preferred_last_name), ''), per.legal_last_name),
		       COALESCE(enr.nc_year_group, ''), COALESCE(enr.enrolment_status, 'C'),
		       enr.academic_year_id, enr.effective_from, enr.effective_to
		FROM pupils p
		JOIN persons per ON per.id = p.person_id
		JOIN pupil_year_enrolments enr ON enr.pupil_id = p.id
		  AND enr.academic_year_id = ?
		  AND enr.effective_from <= ?
		  AND (enr.effective_to IS NULL OR enr.effective_to >= ?)
		WHERE p.id IN (%s)
		ORDER BY per.legal_last_name ASC, per.legal_first_name ASC, p.id ASC`,
		strings.Join(placeholders, ","),
	)

	rows, err := d.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to batch query pupils: %w", err)
	}
	defer rows.Close()

	byID := make(map[string]Pupil, len(cleaned))
	for rows.Next() {
		var p Pupil
		var effTo sql.NullString
		if err := rows.Scan(
			&p.ID, &p.UPN, &p.FirstName, &p.LastName,
			&p.PreferredFirstName, &p.PreferredLastName,
			&p.NCYearGroup, &p.EnrolmentStatus,
			&p.AcademicYearID, &p.EffectiveFrom, &effTo,
		); err != nil {
			return nil, fmt.Errorf("failed to scan batch pupil row: %w", err)
		}
		p.PupilID = p.ID
		p.LegalFirstName = p.FirstName
		p.LegalLastName = p.LastName
		if effTo.Valid {
			val := effTo.String
			p.EffectiveTo = &val
		}
		byID[p.ID] = p
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating batch pupils: %w", err)
	}

	result := make([]Pupil, 0, len(cleaned))
	seen := make(map[string]bool, len(cleaned))
	for _, pid := range cleaned {
		if seen[pid] {
			continue
		}
		if p, ok := byID[pid]; ok {
			seen[pid] = true
			result = append(result, p)
		}
	}
	return result, nil
}

func (d *CoreDAL) ListAllPupilsScoped(ctx context.Context, scope DALQueryScope) ([]Pupil, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}

	rows, err := d.readDB.QueryContext(
		ctx,
		`SELECT p.id, p.upn, per.legal_first_name, per.legal_last_name,
		        COALESCE(NULLIF(TRIM(per.preferred_first_name), ''), per.legal_first_name),
		        COALESCE(NULLIF(TRIM(per.preferred_last_name), ''), per.legal_last_name),
		        COALESCE(enr.nc_year_group, ''), COALESCE(enr.enrolment_status, 'C'),
		        enr.academic_year_id, enr.effective_from, enr.effective_to
		 FROM pupils p
		 JOIN persons per ON per.id = p.person_id
		 JOIN pupil_year_enrolments enr ON enr.pupil_id = p.id
		 WHERE enr.academic_year_id = ?
		   AND enr.effective_from <= ?
		   AND (enr.effective_to IS NULL OR enr.effective_to >= ?)
		 ORDER BY per.legal_last_name ASC, per.legal_first_name ASC, p.id ASC`,
		scope.AcademicYearID, scope.ReferenceDate, scope.ReferenceDate,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list all active pupils: %w", err)
	}
	defer rows.Close()

	result := make([]Pupil, 0)
	for rows.Next() {
		var p Pupil
		var effTo sql.NullString
		if err := rows.Scan(
			&p.ID, &p.UPN, &p.FirstName, &p.LastName,
			&p.PreferredFirstName, &p.PreferredLastName,
			&p.NCYearGroup, &p.EnrolmentStatus,
			&p.AcademicYearID, &p.EffectiveFrom, &effTo,
		); err != nil {
			return nil, fmt.Errorf("failed to scan active pupil row: %w", err)
		}
		p.PupilID = p.ID
		p.LegalFirstName = p.FirstName
		p.LegalLastName = p.LastName
		if effTo.Valid {
			val := effTo.String
			p.EffectiveTo = &val
		}
		result = append(result, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating active pupils: %w", err)
	}
	return result, nil
}

func (d *CoreDAL) ListPupilsByGroupScoped(ctx context.Context, scope DALQueryScope, groupID string) ([]Pupil, error) {
	if strings.TrimSpace(groupID) == "" {
		return nil, errors.New("missing mandatory parameter: group_id")
	}
	if err := d.ensureGroupInOwnGroups(ctx, scope, groupID); err != nil {
		return nil, err
	}

	rows, err := d.readDB.QueryContext(
		ctx,
		`SELECT p.id, p.upn, per.legal_first_name, per.legal_last_name,
		        COALESCE(NULLIF(TRIM(per.preferred_first_name), ''), per.legal_first_name),
		        COALESCE(NULLIF(TRIM(per.preferred_last_name), ''), per.legal_last_name),
		        COALESCE((
		            SELECT enr.nc_year_group FROM pupil_year_enrolments enr
		            WHERE enr.pupil_id = p.id
		              AND enr.academic_year_id = gpm.academic_year_id
		              AND enr.effective_from <= ?
		              AND (enr.effective_to IS NULL OR enr.effective_to >= ?)
		            ORDER BY enr.effective_from DESC LIMIT 1
		        ), ''),
		        COALESCE((
		            SELECT enr.enrolment_status FROM pupil_year_enrolments enr
		            WHERE enr.pupil_id = p.id
		              AND enr.academic_year_id = gpm.academic_year_id
		              AND enr.effective_from <= ?
		              AND (enr.effective_to IS NULL OR enr.effective_to >= ?)
		            ORDER BY enr.effective_from DESC LIMIT 1
		        ), 'C'),
		        gpm.academic_year_id, gpm.effective_from, gpm.effective_to
		 FROM group_pupil_memberships gpm
		 JOIN pupils p ON p.id = gpm.pupil_id
		 JOIN persons per ON per.id = p.person_id
		 WHERE gpm.group_id = ?
		   AND gpm.academic_year_id = ?
		   AND gpm.effective_from <= ?
		   AND (gpm.effective_to IS NULL OR gpm.effective_to >= ?)
		 ORDER BY per.legal_last_name ASC, per.legal_first_name ASC, p.id ASC`,
		scope.ReferenceDate, scope.ReferenceDate,
		scope.ReferenceDate, scope.ReferenceDate,
		groupID, scope.AcademicYearID, scope.ReferenceDate, scope.ReferenceDate,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list pupils for group %s: %w", groupID, err)
	}
	defer rows.Close()

	var result []Pupil
	for rows.Next() {
		var p Pupil
		var effTo sql.NullString
		if err := rows.Scan(
			&p.ID, &p.UPN, &p.FirstName, &p.LastName,
			&p.PreferredFirstName, &p.PreferredLastName,
			&p.NCYearGroup, &p.EnrolmentStatus,
			&p.AcademicYearID, &p.EffectiveFrom, &effTo,
		); err != nil {
			return nil, fmt.Errorf("failed to scan group pupil row: %w", err)
		}
		p.PupilID = p.ID
		p.LegalFirstName = p.FirstName
		p.LegalLastName = p.LastName
		if effTo.Valid {
			val := effTo.String
			p.EffectiveTo = &val
		}
		result = append(result, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating group pupils: %w", err)
	}
	return result, nil
}
