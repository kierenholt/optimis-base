package dal

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type PupilExclusionSummary struct {
	ID                      string `json:"id"`
	ExclusionID             string `json:"exclusion_id"`
	PupilID                 string `json:"pupil_id"`
	AcademicYearID          string `json:"academic_year_id"`
	Category                string `json:"category"`
	ExclusionCategory       string `json:"exclusion_category"`
	DfEReasonCode           string `json:"dfe_reason_code"`
	ReasonCode1             string `json:"reason_code_1"`
	ReasonCode2             string `json:"reason_code_2,omitempty"`
	ReasonCode3             string `json:"reason_code_3,omitempty"`
	StartDate               string `json:"start_date"`
	EndDate                 string `json:"end_date,omitempty"`
	SessionsLost            int    `json:"sessions_lost"`
	SessionsMissed          int    `json:"sessions_missed"`
	SENProvisionAtExclusion string `json:"sen_provision_at_exclusion"`
	ReviewDate              string `json:"review_date,omitempty"`
	ReviewOutcome           string `json:"review_outcome,omitempty"`
	AuthorisingStaffID      string `json:"authorising_staff_id,omitempty"`
	Notes                   string `json:"notes,omitempty"`
}

func (d *CoreDAL) ListPupilExclusionsScoped(
	ctx context.Context,
	scope DALQueryScope,
	pupilID string,
	academicYearID string,
) ([]PupilExclusionSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	trimmedPupil := strings.TrimSpace(pupilID)
	if trimmedPupil != "" && scope.Mode == ScopeOwnGroups {
		if err := d.ensurePupilInOwnGroups(ctx, scope, trimmedPupil); err != nil {
			return nil, err
		}
	}
	trimmedAY := strings.TrimSpace(academicYearID)

	query := `
		SELECT e.id, e.pupil_id, e.academic_year_id, e.exclusion_category, e.reason_code_1,
		       COALESCE(e.reason_code_2, ''), COALESCE(e.reason_code_3, ''),
		       e.start_date, COALESCE(e.end_date, ''), e.sessions_missed,
		       COALESCE(e.sen_provision_at_exclusion, 'N'),
		       COALESCE(e.governor_review_date, ''), COALESCE(e.governor_review_outcome, ''),
		       COALESCE(e.notes, '')
		FROM pupil_suspensions_exclusions e WHERE 1=1`
	var args []any
	if trimmedPupil != "" {
		query += ` AND e.pupil_id = ?`
		args = append(args, trimmedPupil)
	}
	if trimmedAY != "" {
		query += ` AND e.academic_year_id = ?`
		args = append(args, trimmedAY)
	}
	if scope.Mode == ScopeOwnGroups && trimmedPupil == "" {
		query += ` AND EXISTS (
			SELECT 1 FROM group_pupil_memberships gpm
			JOIN group_staff_assignments gsa ON gsa.group_id = gpm.group_id
			WHERE gpm.pupil_id = e.pupil_id
			  AND gsa.staff_id = ?
			  AND gpm.academic_year_id = ?
			  AND gpm.effective_from <= ?
			  AND (gpm.effective_to IS NULL OR gpm.effective_to >= ?)
		)`
		args = append(args, scope.StaffID, scope.AcademicYearID, scope.ReferenceDate, scope.ReferenceDate)
	}
	query += ` ORDER BY e.start_date DESC, e.id ASC`

	rows, err := d.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list pupil_suspensions_exclusions: %w", err)
	}
	defer rows.Close()

	result := make([]PupilExclusionSummary, 0)
	for rows.Next() {
		var r PupilExclusionSummary
		if err := rows.Scan(
			&r.ID, &r.PupilID, &r.AcademicYearID, &r.Category, &r.DfEReasonCode,
			&r.ReasonCode2, &r.ReasonCode3, &r.StartDate, &r.EndDate, &r.SessionsLost,
			&r.SENProvisionAtExclusion, &r.ReviewDate, &r.ReviewOutcome, &r.Notes,
		); err != nil {
			return nil, fmt.Errorf("failed to scan pupil_suspensions_exclusions row: %w", err)
		}
		r.ExclusionID = r.ID
		r.ExclusionCategory = r.Category
		r.ReasonCode1 = r.DfEReasonCode
		r.SessionsMissed = r.SessionsLost
		r.AuthorisingStaffID = scope.StaffID
		result = append(result, r)
	}
	return result, rows.Err()
}

func (d *CoreDAL) UpsertPupilExclusionScoped(
	ctx context.Context,
	scope DALQueryScope,
	in PupilExclusionSummary,
) (*PupilExclusionSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	if scope.Mode != ScopeAll {
		return nil, errors.New("access denied: upserting pupil exclusions requires dal.pupils.write:all scope")
	}

	pupilID := strings.TrimSpace(in.PupilID)
	ayID := strings.TrimSpace(in.AcademicYearID)
	if ayID == "" {
		ayID = scope.AcademicYearID
	}
	category := strings.ToUpper(strings.TrimSpace(in.Category))
	if category == "" {
		category = strings.ToUpper(strings.TrimSpace(in.ExclusionCategory))
	}
	if category == "" {
		category = "SUSPENSION"
	}
	reason1 := strings.ToUpper(strings.TrimSpace(in.DfEReasonCode))
	if reason1 == "" {
		reason1 = strings.ToUpper(strings.TrimSpace(in.ReasonCode1))
	}
	startDate := strings.TrimSpace(in.StartDate)
	if pupilID == "" || reason1 == "" || startDate == "" {
		return nil, errors.New("missing mandatory parameter: pupil_id, dfe_reason_code, and start_date are required")
	}
	exclID := strings.TrimSpace(in.ExclusionID)
	if exclID == "" {
		exclID = strings.TrimSpace(in.ID)
	}
	if exclID == "" {
		exclID = "exc_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	}
	sessionsLost := in.SessionsLost
	if sessionsLost <= 0 {
		sessionsLost = in.SessionsMissed
	}
	if sessionsLost <= 0 {
		sessionsLost = 1
	}
	senProv := strings.ToUpper(strings.TrimSpace(in.SENProvisionAtExclusion))
	if senProv != "K" && senProv != "E" {
		senProv = "N"
	}

	if _, err := d.writeDB.ExecContext(
		ctx,
		`INSERT INTO pupil_suspensions_exclusions (
			id, pupil_id, academic_year_id, exclusion_category, reason_code_1,
			reason_code_2, reason_code_3, start_date, end_date, sessions_missed,
			sen_provision_at_exclusion, governor_review_date, governor_review_outcome, notes
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			exclusion_category = excluded.exclusion_category,
			reason_code_1 = excluded.reason_code_1,
			reason_code_2 = excluded.reason_code_2,
			reason_code_3 = excluded.reason_code_3,
			start_date = excluded.start_date,
			end_date = excluded.end_date,
			sessions_missed = excluded.sessions_missed,
			sen_provision_at_exclusion = excluded.sen_provision_at_exclusion,
			governor_review_date = excluded.governor_review_date,
			governor_review_outcome = excluded.governor_review_outcome,
			notes = excluded.notes`,
		exclID, pupilID, ayID, category, reason1,
		optionalTrimmedString(in.ReasonCode2), optionalTrimmedString(in.ReasonCode3), startDate, optionalTrimmedString(in.EndDate), sessionsLost,
		senProv, optionalTrimmedString(in.ReviewDate), optionalTrimmedString(in.ReviewOutcome), strings.TrimSpace(in.Notes),
	); err != nil {
		return nil, fmt.Errorf("failed to upsert pupil_suspensions_exclusions row: %w", err)
	}

	authStaff := strings.TrimSpace(in.AuthorisingStaffID)
	if authStaff == "" {
		authStaff = scope.StaffID
	}
	out := in
	out.ID = exclID
	out.ExclusionID = exclID
	out.PupilID = pupilID
	out.AcademicYearID = ayID
	out.Category = category
	out.ExclusionCategory = category
	out.DfEReasonCode = reason1
	out.ReasonCode1 = reason1
	out.StartDate = startDate
	out.SessionsLost = sessionsLost
	out.SessionsMissed = sessionsLost
	out.SENProvisionAtExclusion = senProv
	out.AuthorisingStaffID = authStaff
	return &out, nil
}

type PupilPastoralNoteSummary struct {
	ID                string `json:"id"`
	NoteID            string `json:"note_id"`
	PupilID           string `json:"pupil_id"`
	RecordedByStaffID string `json:"recorded_by_staff_id"`
	Category          string `json:"category"`
	AlertLevelID      int    `json:"alert_level_id"`
	Title             string `json:"title,omitempty"`
	Summary           string `json:"summary"`
	CreatedAt         string `json:"created_at,omitempty"`
}

func (d *CoreDAL) ListPupilPastoralNotesScoped(
	ctx context.Context,
	scope DALQueryScope,
	pupilID string,
) ([]PupilPastoralNoteSummary, error) {
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
		`SELECT id, pupil_id, COALESCE(created_by_staff_id, ''), category, COALESCE(title, ''), note_text, COALESCE(created_at, '')
		 FROM pupil_pastoral_notes
		 WHERE pupil_id = ?
		 ORDER BY created_at DESC, id ASC`,
		trimmedPupil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list pupil_pastoral_notes: %w", err)
	}
	defer rows.Close()

	result := make([]PupilPastoralNoteSummary, 0)
	for rows.Next() {
		var r PupilPastoralNoteSummary
		if err := rows.Scan(&r.ID, &r.PupilID, &r.RecordedByStaffID, &r.Category, &r.Title, &r.Summary, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan pupil_pastoral_notes row: %w", err)
		}
		r.NoteID = r.ID
		r.AlertLevelID = 3
		result = append(result, r)
	}
	return result, rows.Err()
}

func (d *CoreDAL) UpsertPupilPastoralNoteScoped(
	ctx context.Context,
	scope DALQueryScope,
	in PupilPastoralNoteSummary,
) (*PupilPastoralNoteSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	pupilID := strings.TrimSpace(in.PupilID)
	summary := strings.TrimSpace(in.Summary)
	staffID := strings.TrimSpace(in.RecordedByStaffID)
	if staffID == "" {
		staffID = scope.StaffID
	}
	if pupilID == "" || summary == "" {
		return nil, errors.New("missing mandatory parameter: pupil_id and summary are required")
	}
	if scope.Mode == ScopeOwnGroups {
		if err := d.ensurePupilInOwnGroups(ctx, scope, pupilID); err != nil {
			return nil, err
		}
	}
	noteID := strings.TrimSpace(in.NoteID)
	if noteID == "" {
		noteID = strings.TrimSpace(in.ID)
	}
	if noteID == "" {
		noteID = "ppn_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	}
	category := strings.ToUpper(strings.TrimSpace(in.Category))
	if category == "" {
		category = "PASTORAL"
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = category + " Note"
	}
	alertLevelID := in.AlertLevelID
	if alertLevelID < 1 || alertLevelID > 4 {
		alertLevelID = 3
	}
	var resolvedStaffID string
	if staffID != "" {
		var exists int
		if err := d.readDB.QueryRowContext(ctx, `SELECT 1 FROM staff WHERE id = ?`, staffID).Scan(&exists); err == nil && exists == 1 {
			resolvedStaffID = staffID
		}
	}
	if resolvedStaffID == "" {
		_ = d.readDB.QueryRowContext(ctx, `SELECT id FROM staff ORDER BY id ASC LIMIT 1`).Scan(&resolvedStaffID)
	}
	if resolvedStaffID == "" {
		return nil, errors.New("missing mandatory parameter: recorded_by_staff_id must reference an existing staff row")
	}

	if _, err := d.writeDB.ExecContext(
		ctx,
		`INSERT INTO pupil_pastoral_notes (
			id, pupil_id, category, title, note_text, created_by_staff_id
		) VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			category = excluded.category,
			title = excluded.title,
			note_text = excluded.note_text,
			created_by_staff_id = excluded.created_by_staff_id`,
		noteID, pupilID, category, title, summary, resolvedStaffID,
	); err != nil {
		return nil, fmt.Errorf("failed to upsert pupil_pastoral_notes row: %w", err)
	}

	return &PupilPastoralNoteSummary{
		ID:                noteID,
		NoteID:            noteID,
		PupilID:           pupilID,
		RecordedByStaffID: staffID,
		Category:          category,
		AlertLevelID:      alertLevelID,
		Title:             title,
		Summary:           summary,
	}, nil
}
