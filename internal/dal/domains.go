package dal

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
)

type ScopeMode string

const (
	ScopeAll       ScopeMode = "all"
	ScopeOwnGroups ScopeMode = "own_groups"
)

type DALQueryScope struct {
	Mode           ScopeMode `json:"mode"`
	StaffID        string    `json:"staff_id"`
	AcademicYearID string    `json:"academic_year_id"`
	ReferenceDate  string    `json:"reference_date"`
}

type PupilMedicalConditionAlert struct {
	ID                    string `json:"id"`
	ConditionName         string `json:"condition_name"`
	ConditionCategory     string `json:"condition_category"`
	AlertLevelID          string `json:"alert_level_id"`
	HasIHCP               bool   `json:"has_ihcp"`
	EmergencyResponsePlan string `json:"emergency_response_plan"`
}

type PupilSenAndMedicalSummary struct {
	PupilID            string                       `json:"pupil_id"`
	SenProvisionCode   string                       `json:"sen_provision_code"`
	PrimarySenNeedCode string                       `json:"primary_sen_need_code"`
	TeachingStrategies string                       `json:"teaching_strategies"`
	MedicalConditions  []PupilMedicalConditionAlert `json:"medical_conditions"`
}

type GroupSummary struct {
	ID              string  `json:"id"`
	AcademicYearID  string  `json:"academic_year_id"`
	GroupType       string  `json:"group_type"`
	Code            string  `json:"code"`
	Name            string  `json:"name"`
	DefaultRoomID   string  `json:"default_room_id"`
	TakesAttendance bool    `json:"takes_attendance"`
	EffectiveFrom   string  `json:"effective_from"`
	EffectiveTo     *string `json:"effective_to,omitempty"`
}

type TimetableSlotSummary struct {
	SlotID                 string `json:"slot_id"`
	PeriodID               string `json:"period_id"`
	PeriodCode             string `json:"period_code"`
	PeriodName             string `json:"period_name"`
	CycleWeek              int    `json:"cycle_week"`
	DayOfWeekID            int    `json:"day_of_week_id"`
	StartTime              string `json:"start_time"`
	EndTime                string `json:"end_time"`
	PeriodType             string `json:"period_type"`
	MapsToStatutorySession string `json:"maps_to_statutory_session"`
	GroupID                string `json:"group_id"`
	GroupCode              string `json:"group_code"`
	StaffID                string `json:"staff_id"`
	RoomID                 string `json:"room_id"`
}

type SessionAttendanceMarkInput struct {
	PupilID        string `json:"pupil_id"`
	AttendanceCode string `json:"attendance_code"`
	MinutesLate    int    `json:"minutes_late"`
	SubCode        string `json:"sub_code"`
	Comment        string `json:"comment"`
}

type UpsertSessionAttendanceBatchInput struct {
	GroupID string                       `json:"group_id"`
	Date    string                       `json:"date"`
	Session string                       `json:"session"`
	Marks   []SessionAttendanceMarkInput `json:"marks"`
}

type UpsertSessionAttendanceBatchResult struct {
	SubmissionID  string `json:"submission_id"`
	UpsertedCount int    `json:"upserted_count"`
	GroupID       string `json:"group_id"`
	Date          string `json:"date"`
	Session       string `json:"session"`
}

type SessionAttendanceMark struct {
	ID                string `json:"id"`
	PupilID           string `json:"pupil_id"`
	AcademicYearID    string `json:"academic_year_id"`
	Date              string `json:"date"`
	Session           string `json:"session"`
	AttendanceCode    string `json:"attendance_code"`
	MinutesLate       int    `json:"minutes_late"`
	SubCode           string `json:"sub_code"`
	Comment           string `json:"comment"`
	SourceGroupID     string `json:"source_group_id"`
	RecordedByStaffID string `json:"recorded_by_staff_id"`
}

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

type CreateSafeguardingPostboxInput struct {
	PupilID            string `json:"pupil_id"`
	ReportedByStaffID  string `json:"reported_by_staff_id"`
	AssignedDSLStaffID string `json:"assigned_dsl_staff_id"`
	ConcernCategory    string `json:"concern_category"`
	IncidentDate       string `json:"incident_date"`
	Summary            string `json:"summary"`
	BodyMap            string `json:"body_map"`
	VerbatimAccount    string `json:"verbatim_account"`
	ActionTaken        string `json:"action_taken"`
	RequestID          string `json:"request_id"`
	ClientIP           string `json:"client_ip"`
	HostKMSKeyHandle   string `json:"host_kms_key_handle"`
}

type SafeguardingPostboxReceipt struct {
	NoteID string `json:"note_id"`
	Status string `json:"status"`
}

type ReadSafeguardingNoteDSLInput struct {
	NoteID           string `json:"note_id"`
	DSLStaffID       string `json:"dsl_staff_id"`
	CloudKeyToken    string `json:"cloud_key_token"`
	HostKMSKeyHandle string `json:"host_kms_key_handle"`
	RequestID        string `json:"request_id"`
	ClientIP         string `json:"client_ip"`
}

type SafeguardingAuditLogEntry struct {
	AuditID       string `json:"audit_id"`
	NoteID        string `json:"note_id"`
	PupilID       string `json:"pupil_id"`
	ActorPersonID string `json:"actor_person_id"`
	ActionType    string `json:"action_type"`
	RequestID     string `json:"request_id"`
	ClientIP      string `json:"client_ip"`
	Timestamp     string `json:"timestamp"`
}

type SafeguardingNoteEnvelopeMetadata struct {
	NoteID             string `json:"note_id"`
	PupilID            string `json:"pupil_id"`
	ReportedByStaffID  string `json:"reported_by_staff_id"`
	AssignedDSLStaffID string `json:"assigned_dsl_staff_id"`
	ConcernCategory    string `json:"concern_category"`
	Status             string `json:"status"`
	IncidentDate       string `json:"incident_date"`
	KMSKeyID           string `json:"kms_key_id"`
	CreatedAt          string `json:"created_at"`
}

type DecryptedSafeguardingNote struct {
	NoteID             string                      `json:"note_id"`
	PupilID            string                      `json:"pupil_id"`
	ReportedByStaffID  string                      `json:"reported_by_staff_id"`
	AssignedDSLStaffID string                      `json:"assigned_dsl_staff_id"`
	ConcernCategory    string                      `json:"concern_category"`
	Status             string                      `json:"status"`
	IncidentDate       string                      `json:"incident_date"`
	Summary            string                      `json:"summary"`
	BodyMap            string                      `json:"body_map"`
	VerbatimAccount    string                      `json:"verbatim_account"`
	ActionTaken        string                      `json:"action_taken"`
	AuditLogs          []SafeguardingAuditLogEntry `json:"audit_logs,omitempty"`
}

type safeguardingEncryptedPayload struct {
	Summary         string `json:"summary"`
	BodyMap         string `json:"body_map"`
	VerbatimAccount string `json:"verbatim_account"`
	ActionTaken     string `json:"action_taken"`
}

type dslCloudKeyClaims struct {
	StaffID           string `json:"staff_id"`
	ExpiresAtUnixNano int64  `json:"expires_at_unix_nano"`
}

func validateScope(scope DALQueryScope) error {
	if scope.Mode != ScopeAll && scope.Mode != ScopeOwnGroups {
		return fmt.Errorf("invalid or missing DALQueryScope.Mode %q", scope.Mode)
	}
	if scope.Mode == ScopeOwnGroups && strings.TrimSpace(scope.StaffID) == "" {
		return errors.New("missing mandatory StaffID for :own_groups scope")
	}
	if strings.TrimSpace(scope.AcademicYearID) == "" {
		return errors.New("missing mandatory AcademicYearID in DALQueryScope")
	}
	if strings.TrimSpace(scope.ReferenceDate) == "" {
		return errors.New("missing mandatory ReferenceDate in DALQueryScope")
	}
	return nil
}

func (d *CoreDAL) ResolveScopeDefaults(ctx context.Context, scope DALQueryScope) (DALQueryScope, error) {
	out := scope
	if strings.TrimSpace(out.ReferenceDate) == "" {
		var today string
		if err := d.readDB.QueryRowContext(ctx, `SELECT date('now')`).Scan(&today); err != nil {
			return out, fmt.Errorf("failed to resolve current reference date: %w", err)
		}
		out.ReferenceDate = today
	}
	if strings.TrimSpace(out.AcademicYearID) == "" {
		var ayID string
		err := d.readDB.QueryRowContext(
			ctx,
			`SELECT id FROM academic_years WHERE status = 'CURRENT' ORDER BY sequence_number DESC LIMIT 1`,
		).Scan(&ayID)
		if err != nil {
			return out, fmt.Errorf("missing mandatory AcademicYearID and no CURRENT academic_year found: %w", err)
		}
		out.AcademicYearID = ayID
	}
	return out, nil
}

func (d *CoreDAL) ensureGroupInOwnGroups(ctx context.Context, scope DALQueryScope, groupID string) error {
	if err := validateScope(scope); err != nil {
		return err
	}
	if strings.TrimSpace(groupID) == "" {
		return errors.New("missing mandatory parameter: group_id")
	}

	if scope.Mode == ScopeAll {
		var count int
		err := d.readDB.QueryRowContext(
			ctx,
			`SELECT COUNT(*) FROM groups
			 WHERE id = ?
			   AND academic_year_id = ?
			   AND effective_from <= ?
			   AND (effective_to IS NULL OR effective_to >= ?)`,
			groupID, scope.AcademicYearID, scope.ReferenceDate, scope.ReferenceDate,
		).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to verify group %s: %w", groupID, err)
		}
		if count == 0 {
			return fmt.Errorf("group %s not found or inactive on %s", groupID, scope.ReferenceDate)
		}
		return nil
	}

	var count int
	err := d.readDB.QueryRowContext(
		ctx,
		`SELECT COUNT(*)
		 FROM groups g
		 JOIN group_staff_assignments gsa ON gsa.group_id = g.id
		 WHERE g.id = ?
		   AND g.academic_year_id = ?
		   AND g.effective_from <= ?
		   AND (g.effective_to IS NULL OR g.effective_to >= ?)
		   AND gsa.staff_id = ?
		   AND gsa.academic_year_id = ?
		   AND gsa.effective_from <= ?
		   AND (gsa.effective_to IS NULL OR gsa.effective_to >= ?)`,
		groupID,
		scope.AcademicYearID, scope.ReferenceDate, scope.ReferenceDate,
		scope.StaffID,
		scope.AcademicYearID, scope.ReferenceDate, scope.ReferenceDate,
	).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to verify :own_groups assignment for group %s: %w", groupID, err)
	}
	if count == 0 {
		return fmt.Errorf("access denied under own_groups scope: staff %s is not assigned to active group %s on %s", scope.StaffID, groupID, scope.ReferenceDate)
	}
	return nil
}

func (d *CoreDAL) ensurePupilInOwnGroups(ctx context.Context, scope DALQueryScope, pupilID string) error {
	if err := validateScope(scope); err != nil {
		return err
	}
	if strings.TrimSpace(pupilID) == "" {
		return errors.New("missing mandatory parameter: pupil_id")
	}
	if scope.Mode == ScopeAll {
		return nil
	}

	var count int
	err := d.readDB.QueryRowContext(
		ctx,
		`SELECT COUNT(*)
		 FROM group_pupil_memberships gpm
		 JOIN groups g ON g.id = gpm.group_id
		 JOIN group_staff_assignments gsa ON gsa.group_id = g.id
		 WHERE gpm.pupil_id = ?
		   AND gpm.academic_year_id = ?
		   AND gpm.effective_from <= ?
		   AND (gpm.effective_to IS NULL OR gpm.effective_to >= ?)
		   AND g.academic_year_id = ?
		   AND g.effective_from <= ?
		   AND (g.effective_to IS NULL OR g.effective_to >= ?)
		   AND gsa.staff_id = ?
		   AND gsa.academic_year_id = ?
		   AND gsa.effective_from <= ?
		   AND (gsa.effective_to IS NULL OR gsa.effective_to >= ?)`,
		pupilID,
		scope.AcademicYearID, scope.ReferenceDate, scope.ReferenceDate,
		scope.AcademicYearID, scope.ReferenceDate, scope.ReferenceDate,
		scope.StaffID,
		scope.AcademicYearID, scope.ReferenceDate, scope.ReferenceDate,
	).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to verify own_groups pupil membership: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("access denied under own_groups scope: pupil is not in any active group assigned to staff %s on %s", scope.StaffID, scope.ReferenceDate)
	}
	return nil
}

// ============================================================================
// 1. Pupils, SEN & Medical Domain
// ============================================================================

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
		`SELECT p.id, p.upn, per.legal_first_name, per.legal_last_name, enr.academic_year_id, enr.effective_from, enr.effective_to
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
	if err := row.Scan(&p.ID, &p.UPN, &p.FirstName, &p.LastName, &p.AcademicYearID, &p.EffectiveFrom, &effTo); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("pupil not found or not enrolled on %s", scope.ReferenceDate)
		}
		return nil, fmt.Errorf("failed to query scoped pupil: %w", err)
	}
	if effTo.Valid {
		val := effTo.String
		p.EffectiveTo = &val
	}
	return &p, nil
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
		`SELECT p.id, p.upn, per.legal_first_name, per.legal_last_name, gpm.academic_year_id, gpm.effective_from, gpm.effective_to
		 FROM group_pupil_memberships gpm
		 JOIN pupils p ON p.id = gpm.pupil_id
		 JOIN persons per ON per.id = p.person_id
		 WHERE gpm.group_id = ?
		   AND gpm.academic_year_id = ?
		   AND gpm.effective_from <= ?
		   AND (gpm.effective_to IS NULL OR gpm.effective_to >= ?)
		 ORDER BY per.legal_last_name ASC, per.legal_first_name ASC, p.id ASC`,
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
		if err := rows.Scan(&p.ID, &p.UPN, &p.FirstName, &p.LastName, &p.AcademicYearID, &p.EffectiveFrom, &effTo); err != nil {
			return nil, fmt.Errorf("failed to scan group pupil row: %w", err)
		}
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

func (d *CoreDAL) ListPupilSenAndMedicalBatch(
	ctx context.Context,
	scope DALQueryScope,
	pupilIDs []string,
) (map[string]PupilSenAndMedicalSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	if len(pupilIDs) == 0 {
		return map[string]PupilSenAndMedicalSummary{}, nil
	}

	result := make(map[string]PupilSenAndMedicalSummary, len(pupilIDs))
	for _, rawID := range pupilIDs {
		pupilID := strings.TrimSpace(rawID)
		if pupilID == "" {
			return nil, errors.New("pupilIDs contains an empty pupil_id")
		}
		if err := d.ensurePupilInOwnGroups(ctx, scope, pupilID); err != nil {
			return nil, err
		}

		summary := PupilSenAndMedicalSummary{
			PupilID:           pupilID,
			SenProvisionCode:  "N",
			MedicalConditions: []PupilMedicalConditionAlert{},
		}

		var provCode, strategies string
		err := d.readDB.QueryRowContext(
			ctx,
			`SELECT provision_code, teaching_strategies
			 FROM pupil_sen_records
			 WHERE pupil_id = ?
			   AND effective_from <= ?
			   AND (effective_to IS NULL OR effective_to >= ?)
			 ORDER BY effective_from DESC
			 LIMIT 1`,
			pupilID, scope.ReferenceDate, scope.ReferenceDate,
		).Scan(&provCode, &strategies)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("failed to query SEN record: %w", err)
		}
		if err == nil {
			summary.SenProvisionCode = provCode
			summary.TeachingStrategies = strategies
		}

		var primaryNeed string
		err = d.readDB.QueryRowContext(
			ctx,
			`SELECT sen_need_code
			 FROM pupil_sen_needs
			 WHERE pupil_id = ?
			   AND ranking = 1
			   AND effective_from <= ?
			   AND (effective_to IS NULL OR effective_to >= ?)
			 ORDER BY effective_from DESC
			 LIMIT 1`,
			pupilID, scope.ReferenceDate, scope.ReferenceDate,
		).Scan(&primaryNeed)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("failed to query primary SEN need: %w", err)
		}
		if err == nil {
			summary.PrimarySenNeedCode = primaryNeed
		}

		medRows, err := d.readDB.QueryContext(
			ctx,
			`SELECT id, condition_name, condition_category, COALESCE(alert_level_id, ''), has_ihcp, emergency_response_plan
			 FROM pupil_medical_conditions
			 WHERE pupil_id = ?
			   AND effective_from <= ?
			   AND (effective_to IS NULL OR effective_to >= ?)
			 ORDER BY id ASC`,
			pupilID, scope.ReferenceDate, scope.ReferenceDate,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to query medical conditions: %w", err)
		}
		for medRows.Next() {
			var cond PupilMedicalConditionAlert
			var hasIHCPInt int
			if scanErr := medRows.Scan(
				&cond.ID,
				&cond.ConditionName,
				&cond.ConditionCategory,
				&cond.AlertLevelID,
				&hasIHCPInt,
				&cond.EmergencyResponsePlan,
			); scanErr != nil {
				_ = medRows.Close()
				return nil, fmt.Errorf("failed to scan medical condition: %w", scanErr)
			}
			cond.HasIHCP = hasIHCPInt == 1
			summary.MedicalConditions = append(summary.MedicalConditions, cond)
		}
		if err := medRows.Err(); err != nil {
			_ = medRows.Close()
			return nil, fmt.Errorf("error iterating medical conditions: %w", err)
		}
		_ = medRows.Close()

		result[pupilID] = summary
	}

	return result, nil
}

// ============================================================================
// 2. Groups & Timetable Spine
// ============================================================================

func (d *CoreDAL) ListGroupsScoped(ctx context.Context, scope DALQueryScope, groupType string) ([]GroupSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}

	var rows *sql.Rows
	var err error
	trimmedType := strings.TrimSpace(groupType)

	if scope.Mode == ScopeOwnGroups {
		if trimmedType != "" {
			rows, err = d.readDB.QueryContext(
				ctx,
				`SELECT DISTINCT g.id, g.academic_year_id, g.group_type, g.code, g.name, COALESCE(g.default_room_id, ''), g.takes_attendance, g.effective_from, g.effective_to
				 FROM groups g
				 JOIN group_staff_assignments gsa ON gsa.group_id = g.id
				 WHERE g.academic_year_id = ?
				   AND g.group_type = ?
				   AND g.effective_from <= ?
				   AND (g.effective_to IS NULL OR g.effective_to >= ?)
				   AND gsa.staff_id = ?
				   AND gsa.academic_year_id = ?
				   AND gsa.effective_from <= ?
				   AND (gsa.effective_to IS NULL OR gsa.effective_to >= ?)
				 ORDER BY g.code ASC, g.id ASC`,
				scope.AcademicYearID, trimmedType, scope.ReferenceDate, scope.ReferenceDate,
				scope.StaffID, scope.AcademicYearID, scope.ReferenceDate, scope.ReferenceDate,
			)
		} else {
			rows, err = d.readDB.QueryContext(
				ctx,
				`SELECT DISTINCT g.id, g.academic_year_id, g.group_type, g.code, g.name, COALESCE(g.default_room_id, ''), g.takes_attendance, g.effective_from, g.effective_to
				 FROM groups g
				 JOIN group_staff_assignments gsa ON gsa.group_id = g.id
				 WHERE g.academic_year_id = ?
				   AND g.effective_from <= ?
				   AND (g.effective_to IS NULL OR g.effective_to >= ?)
				   AND gsa.staff_id = ?
				   AND gsa.academic_year_id = ?
				   AND gsa.effective_from <= ?
				   AND (gsa.effective_to IS NULL OR gsa.effective_to >= ?)
				 ORDER BY g.code ASC, g.id ASC`,
				scope.AcademicYearID, scope.ReferenceDate, scope.ReferenceDate,
				scope.StaffID, scope.AcademicYearID, scope.ReferenceDate, scope.ReferenceDate,
			)
		}
	} else {
		if trimmedType != "" {
			rows, err = d.readDB.QueryContext(
				ctx,
				`SELECT g.id, g.academic_year_id, g.group_type, g.code, g.name, COALESCE(g.default_room_id, ''), g.takes_attendance, g.effective_from, g.effective_to
				 FROM groups g
				 WHERE g.academic_year_id = ?
				   AND g.group_type = ?
				   AND g.effective_from <= ?
				   AND (g.effective_to IS NULL OR g.effective_to >= ?)
				 ORDER BY g.code ASC, g.id ASC`,
				scope.AcademicYearID, trimmedType, scope.ReferenceDate, scope.ReferenceDate,
			)
		} else {
			rows, err = d.readDB.QueryContext(
				ctx,
				`SELECT g.id, g.academic_year_id, g.group_type, g.code, g.name, COALESCE(g.default_room_id, ''), g.takes_attendance, g.effective_from, g.effective_to
				 FROM groups g
				 WHERE g.academic_year_id = ?
				   AND g.effective_from <= ?
				   AND (g.effective_to IS NULL OR g.effective_to >= ?)
				 ORDER BY g.code ASC, g.id ASC`,
				scope.AcademicYearID, scope.ReferenceDate, scope.ReferenceDate,
			)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to list scoped groups: %w", err)
	}
	defer rows.Close()

	var result []GroupSummary
	for rows.Next() {
		var g GroupSummary
		var takesAttInt int
		var effTo sql.NullString
		if err := rows.Scan(
			&g.ID,
			&g.AcademicYearID,
			&g.GroupType,
			&g.Code,
			&g.Name,
			&g.DefaultRoomID,
			&takesAttInt,
			&g.EffectiveFrom,
			&effTo,
		); err != nil {
			return nil, fmt.Errorf("failed to scan group row: %w", err)
		}
		g.TakesAttendance = takesAttInt == 1
		if effTo.Valid {
			val := effTo.String
			g.EffectiveTo = &val
		}
		result = append(result, g)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating groups: %w", err)
	}
	return result, nil
}

func (d *CoreDAL) GetStaffTimetableSlotsScoped(
	ctx context.Context,
	scope DALQueryScope,
	staffID string,
	cycleWeek int,
	dayOfWeekID int,
) ([]TimetableSlotSummary, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	targetStaffID := strings.TrimSpace(staffID)
	if targetStaffID == "" {
		return nil, errors.New("missing mandatory parameter: staff_id")
	}
	if scope.Mode == ScopeOwnGroups && targetStaffID != scope.StaffID {
		return nil, fmt.Errorf("access denied under own_groups scope: staff %s cannot read timetable for staff %s", scope.StaffID, targetStaffID)
	}
	if cycleWeek != 1 && cycleWeek != 2 {
		return nil, fmt.Errorf("invalid cycleWeek %d: must be 1 or 2", cycleWeek)
	}

	rows, err := d.readDB.QueryContext(
		ctx,
		`SELECT ts.id, tp.id, tp.period_code, tp.name, tp.cycle_week, tp.day_of_week_id,
		        tp.start_time, tp.end_time, tp.period_type, COALESCE(tp.maps_to_statutory_session, ''),
		        ts.group_id, g.code, COALESCE(ts.staff_id, ''), COALESCE(ts.room_id, '')
		 FROM timetable_slots ts
		 JOIN timetable_periods tp ON tp.id = ts.timetable_period_id
		 JOIN groups g ON g.id = ts.group_id
		 WHERE ts.staff_id = ?
		   AND tp.academic_year_id = ?
		   AND tp.cycle_week = ?
		   AND tp.day_of_week_id = ?
		   AND ts.effective_from <= ?
		   AND (ts.effective_to IS NULL OR ts.effective_to >= ?)
		 ORDER BY tp.sequence_number ASC, ts.id ASC`,
		targetStaffID, scope.AcademicYearID, cycleWeek, dayOfWeekID, scope.ReferenceDate, scope.ReferenceDate,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query staff timetable slots: %w", err)
	}
	defer rows.Close()

	var result []TimetableSlotSummary
	for rows.Next() {
		var slot TimetableSlotSummary
		if err := rows.Scan(
			&slot.SlotID,
			&slot.PeriodID,
			&slot.PeriodCode,
			&slot.PeriodName,
			&slot.CycleWeek,
			&slot.DayOfWeekID,
			&slot.StartTime,
			&slot.EndTime,
			&slot.PeriodType,
			&slot.MapsToStatutorySession,
			&slot.GroupID,
			&slot.GroupCode,
			&slot.StaffID,
			&slot.RoomID,
		); err != nil {
			return nil, fmt.Errorf("failed to scan timetable slot: %w", err)
		}
		result = append(result, slot)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating timetable slots: %w", err)
	}
	return result, nil
}

type UpsertGroupPupilMembershipInput struct {
	MembershipID   string `json:"membership_id,omitempty"`
	GroupID        string `json:"group_id"`
	PupilID        string `json:"pupil_id"`
	AcademicYearID string `json:"academic_year_id"`
	EffectiveFrom  string `json:"effective_from"`
	EffectiveTo    string `json:"effective_to,omitempty"`
}

type UpsertGroupPupilMembershipResult struct {
	MembershipID   string `json:"membership_id"`
	GroupID        string `json:"group_id"`
	PupilID        string `json:"pupil_id"`
	AcademicYearID string `json:"academic_year_id"`
	EffectiveFrom  string `json:"effective_from"`
	EffectiveTo    string `json:"effective_to,omitempty"`
}

func (d *CoreDAL) UpsertGroupPupilMembershipScoped(
	ctx context.Context,
	scope DALQueryScope,
	in UpsertGroupPupilMembershipInput,
) (*UpsertGroupPupilMembershipResult, error) {
	groupID := strings.TrimSpace(in.GroupID)
	pupilID := strings.TrimSpace(in.PupilID)
	ayID := strings.TrimSpace(in.AcademicYearID)
	if ayID == "" {
		ayID = strings.TrimSpace(scope.AcademicYearID)
	}
	effFrom := strings.TrimSpace(in.EffectiveFrom)
	if effFrom == "" {
		effFrom = strings.TrimSpace(scope.ReferenceDate)
	}
	if groupID == "" {
		return nil, errors.New("missing mandatory parameter: group_id")
	}
	if pupilID == "" {
		return nil, errors.New("missing mandatory parameter: pupil_id")
	}
	if ayID == "" {
		return nil, errors.New("missing mandatory parameter: academic_year_id")
	}
	if effFrom == "" {
		return nil, errors.New("missing mandatory parameter: effective_from")
	}
	if err := d.ensureGroupInOwnGroups(ctx, scope, groupID); err != nil {
		return nil, err
	}

	membershipID := strings.TrimSpace(in.MembershipID)
	if membershipID == "" {
		membershipID = "gpm_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	}
	var effToArg any
	trimmedEffTo := strings.TrimSpace(in.EffectiveTo)
	if trimmedEffTo != "" {
		effToArg = trimmedEffTo
	}

	tx, err := d.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin group membership transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO group_pupil_memberships (
			id, group_id, pupil_id, academic_year_id, effective_from, effective_to
		) VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			effective_from = excluded.effective_from,
			effective_to = excluded.effective_to`,
		membershipID, groupID, pupilID, ayID, effFrom, effToArg,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert group_pupil_memberships row: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit group membership transaction: %w", err)
	}

	d.emitPostCommitEvent(ctx, EventCoreGroupMembershipChanged, map[string]any{
		"membership_id":    membershipID,
		"group_id":         groupID,
		"pupil_id":         pupilID,
		"academic_year_id": ayID,
		"effective_from":   effFrom,
		"effective_to":     trimmedEffTo,
	})

	return &UpsertGroupPupilMembershipResult{
		MembershipID:   membershipID,
		GroupID:        groupID,
		PupilID:        pupilID,
		AcademicYearID: ayID,
		EffectiveFrom:  effFrom,
		EffectiveTo:    trimmedEffTo,
	}, nil
}

// ============================================================================
// 3. Statutory Attendance
// ============================================================================

func (d *CoreDAL) GetGroupSessionAttendanceScoped(
	ctx context.Context,
	scope DALQueryScope,
	groupID string,
	date string,
	session string,
) ([]SessionAttendanceMark, error) {
	if strings.TrimSpace(groupID) == "" {
		return nil, errors.New("missing mandatory parameter: group_id")
	}
	if strings.TrimSpace(date) == "" {
		return nil, errors.New("missing mandatory parameter: date")
	}
	if session != "AM" && session != "PM" {
		return nil, fmt.Errorf("invalid statutory session %q: must be 'AM' or 'PM'", session)
	}
	if err := d.ensureGroupInOwnGroups(ctx, scope, groupID); err != nil {
		return nil, err
	}

	rows, err := d.readDB.QueryContext(
		ctx,
		`SELECT asm.id, asm.pupil_id, asm.academic_year_id, asm.date, asm.session,
		        asm.attendance_code, asm.minutes_late, asm.sub_code, asm.comment,
		        COALESCE(asm.source_group_id, ''), COALESCE(asm.recorded_by_staff_id, '')
		 FROM attendance_session_marks asm
		 JOIN group_pupil_memberships gpm ON gpm.pupil_id = asm.pupil_id AND gpm.group_id = ?
		 WHERE asm.date = ?
		   AND asm.session = ?
		   AND gpm.academic_year_id = ?
		   AND gpm.effective_from <= ?
		   AND (gpm.effective_to IS NULL OR gpm.effective_to >= ?)
		 ORDER BY asm.pupil_id ASC`,
		groupID, date, session, scope.AcademicYearID, date, date,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query group session attendance: %w", err)
	}
	defer rows.Close()

	var result []SessionAttendanceMark
	for rows.Next() {
		var m SessionAttendanceMark
		if err := rows.Scan(
			&m.ID,
			&m.PupilID,
			&m.AcademicYearID,
			&m.Date,
			&m.Session,
			&m.AttendanceCode,
			&m.MinutesLate,
			&m.SubCode,
			&m.Comment,
			&m.SourceGroupID,
			&m.RecordedByStaffID,
		); err != nil {
			return nil, fmt.Errorf("failed to scan attendance session mark: %w", err)
		}
		result = append(result, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating attendance session marks: %w", err)
	}
	return result, nil
}

func (d *CoreDAL) UpsertSessionAttendanceBatchScoped(
	ctx context.Context,
	scope DALQueryScope,
	in UpsertSessionAttendanceBatchInput,
) (*UpsertSessionAttendanceBatchResult, error) {
	if strings.TrimSpace(in.GroupID) == "" {
		return nil, errors.New("missing mandatory parameter: group_id")
	}
	if strings.TrimSpace(in.Date) == "" {
		return nil, errors.New("missing mandatory parameter: date")
	}
	if in.Session != "AM" && in.Session != "PM" {
		return nil, fmt.Errorf("invalid statutory session %q: must be 'AM' or 'PM'", in.Session)
	}
	if len(in.Marks) == 0 {
		return nil, errors.New("marks batch must contain at least one attendance mark")
	}
	if strings.TrimSpace(scope.StaffID) == "" {
		return nil, errors.New("missing mandatory StaffID for recording attendance submission")
	}
	if err := d.ensureGroupInOwnGroups(ctx, scope, in.GroupID); err != nil {
		return nil, err
	}

	tx, err := d.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin attendance batch transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	recordedPupilIDs := make([]string, 0, len(in.Marks))
	for _, mark := range in.Marks {
		pupilID := strings.TrimSpace(mark.PupilID)
		code := strings.TrimSpace(mark.AttendanceCode)
		if pupilID == "" || code == "" {
			return nil, errors.New("each attendance mark requires pupil_id and attendance_code")
		}
		if mark.MinutesLate < 0 {
			return nil, errors.New("minutes_late cannot be negative")
		}

		// Verify pupil belongs to the target group on in.Date
		var memberCount int
		if err := tx.QueryRowContext(
			ctx,
			`SELECT COUNT(*)
			 FROM group_pupil_memberships
			 WHERE group_id = ?
			   AND pupil_id = ?
			   AND academic_year_id = ?
			   AND effective_from <= ?
			   AND (effective_to IS NULL OR effective_to >= ?)`,
			in.GroupID, pupilID, scope.AcademicYearID, in.Date, in.Date,
		).Scan(&memberCount); err != nil {
			return nil, fmt.Errorf("failed to verify group membership for attendance mark: %w", err)
		}
		if memberCount == 0 {
			return nil, fmt.Errorf("pupil is not an active member of group %s on %s", in.GroupID, in.Date)
		}

		markID := "asm_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
		_, err := tx.ExecContext(
			ctx,
			`INSERT INTO attendance_session_marks (
				id, pupil_id, academic_year_id, date, session,
				attendance_code, minutes_late, sub_code, comment,
				source_group_id, recorded_by_staff_id
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(pupil_id, date, session) DO UPDATE SET
				attendance_code = excluded.attendance_code,
				minutes_late = excluded.minutes_late,
				sub_code = excluded.sub_code,
				comment = excluded.comment,
				source_group_id = excluded.source_group_id,
				recorded_by_staff_id = excluded.recorded_by_staff_id,
				recorded_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')`,
			markID, pupilID, scope.AcademicYearID, in.Date, in.Session,
			code, mark.MinutesLate, mark.SubCode, mark.Comment,
			in.GroupID, scope.StaffID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to upsert attendance session mark: %w", err)
		}
		recordedPupilIDs = append(recordedPupilIDs, pupilID)
	}

	newSubID := "ars_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO attendance_register_submissions (
			id, group_id, date, session_or_slot_key, taken_by_staff_id
		) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(group_id, date, session_or_slot_key) DO UPDATE SET
			taken_by_staff_id = excluded.taken_by_staff_id,
			taken_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')`,
		newSubID, in.GroupID, in.Date, in.Session, scope.StaffID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert attendance_register_submissions row: %w", err)
	}

	var persistedSubID string
	if err := tx.QueryRowContext(
		ctx,
		`SELECT id FROM attendance_register_submissions WHERE group_id = ? AND date = ? AND session_or_slot_key = ?`,
		in.GroupID, in.Date, in.Session,
	).Scan(&persistedSubID); err != nil {
		return nil, fmt.Errorf("failed to read attendance register submission ID: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit attendance batch transaction: %w", err)
	}

	firstPupilID := ""
	if len(recordedPupilIDs) > 0 {
		firstPupilID = recordedPupilIDs[0]
	}
	d.emitPostCommitEvent(ctx, EventCoreAttendanceSessionRecorded, map[string]any{
		"submission_id":        persistedSubID,
		"group_id":             in.GroupID,
		"date":                 in.Date,
		"session":              in.Session,
		"academic_year_id":     scope.AcademicYearID,
		"upserted_count":       len(in.Marks),
		"recorded_by_staff_id": scope.StaffID,
		"pupil_id":             firstPupilID,
		"pupil_ids":            recordedPupilIDs,
	})

	return &UpsertSessionAttendanceBatchResult{
		SubmissionID:  persistedSubID,
		UpsertedCount: len(in.Marks),
		GroupID:       in.GroupID,
		Date:          in.Date,
		Session:       in.Session,
	}, nil
}

// ============================================================================
// 4. Guardians & Households (Mandatory PR & Court-Order Invariant)
// ============================================================================

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

// ============================================================================
// 5. Safeguarding Vault (Envelope Encryption + DSL Cloud Key Token + Audit Log)
// ============================================================================

func deriveDSLVaultMasterKey(hostKMSKeyHandle string) [32]byte {
	return sha256.Sum256([]byte("optimis-dsl-vault-master-key-v1:" + hostKMSKeyHandle))
}

func (d *CoreDAL) CreateSafeguardingPostboxNote(
	ctx context.Context,
	in CreateSafeguardingPostboxInput,
) (*SafeguardingPostboxReceipt, error) {
	if strings.TrimSpace(in.PupilID) == "" {
		return nil, errors.New("missing mandatory parameter: pupil_id")
	}
	if strings.TrimSpace(in.ReportedByStaffID) == "" {
		return nil, errors.New("missing mandatory parameter: reported_by_staff_id")
	}
	if strings.TrimSpace(in.ConcernCategory) == "" {
		return nil, errors.New("missing mandatory parameter: concern_category")
	}
	if strings.TrimSpace(in.IncidentDate) == "" {
		return nil, errors.New("missing mandatory parameter: incident_date")
	}
	if strings.TrimSpace(in.Summary) == "" {
		return nil, errors.New("missing mandatory parameter: summary")
	}
	if strings.TrimSpace(in.VerbatimAccount) == "" {
		return nil, errors.New("missing mandatory parameter: verbatim_account")
	}
	if strings.TrimSpace(in.RequestID) == "" {
		return nil, errors.New("missing mandatory parameter: request_id")
	}
	if strings.TrimSpace(in.HostKMSKeyHandle) == "" {
		return nil, errors.New("missing mandatory parameter: host_kms_key_handle")
	}
	clientIP := strings.TrimSpace(in.ClientIP)
	if clientIP == "" {
		clientIP = "127.0.0.1"
	}

	var reporterPersonID string
	if err := d.readDB.QueryRowContext(
		ctx,
		`SELECT person_id FROM staff WHERE id = ?`,
		in.ReportedByStaffID,
	).Scan(&reporterPersonID); err != nil {
		return nil, fmt.Errorf("failed to resolve reporting staff person_id: %w", err)
	}

	plaintextStruct := safeguardingEncryptedPayload{
		Summary:         in.Summary,
		BodyMap:         in.BodyMap,
		VerbatimAccount: in.VerbatimAccount,
		ActionTaken:     in.ActionTaken,
	}
	plaintextJSON, err := json.Marshal(plaintextStruct)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize safeguarding payload: %w", err)
	}

	// 1. Generate per-note random 256-bit Data Encryption Key (DEK) and 12-byte AES-GCM nonce
	dek := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, dek); err != nil {
		return nil, fmt.Errorf("failed to generate random DEK: %w", err)
	}
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate random AES-GCM nonce: %w", err)
	}

	dekBlock, err := aes.NewCipher(dek)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize DEK AES cipher: %w", err)
	}
	dekGCM, err := cipher.NewGCM(dekBlock)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize DEK GCM: %w", err)
	}
	ciphertext := dekGCM.Seal(nil, nonce, plaintextJSON, nil)

	// 2. Wrap the DEK with the DSL Vault Master Key
	masterKey := deriveDSLVaultMasterKey(in.HostKMSKeyHandle)
	masterBlock, err := aes.NewCipher(masterKey[:])
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Master Key AES cipher: %w", err)
	}
	masterGCM, err := cipher.NewGCM(masterBlock)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Master Key GCM: %w", err)
	}
	wrapNonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, wrapNonce); err != nil {
		return nil, fmt.Errorf("failed to generate DEK wrap nonce: %w", err)
	}
	wrappedDEK := append(wrapNonce, masterGCM.Seal(nil, wrapNonce, dek, nil)...)

	noteID := "sfg_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	auditID := "aud_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]

	var assignedDSL any
	if strings.TrimSpace(in.AssignedDSLStaffID) != "" {
		assignedDSL = strings.TrimSpace(in.AssignedDSLStaffID)
	}

	tx, err := d.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin safeguarding postbox transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO safeguarding_notes (
			id, pupil_id, reported_by_staff_id, assigned_dsl_staff_id,
			concern_category, status, incident_date,
			kms_key_id, wrapped_dek, nonce, ciphertext_payload
		) VALUES (?, ?, ?, ?, ?, 'OPEN_TRIAGE', ?, ?, ?, ?, ?)`,
		noteID, in.PupilID, in.ReportedByStaffID, assignedDSL,
		in.ConcernCategory, in.IncidentDate,
		in.HostKMSKeyHandle, wrappedDEK, nonce, ciphertext,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert safeguarding_notes postbox record: %w", err)
	}

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO safeguarding_audit_logs (
			audit_id, note_id, pupil_id, actor_person_id, action_type, request_id, client_ip
		) VALUES (?, ?, ?, ?, 'CREATE_POSTBOX', ?, ?)`,
		auditID, noteID, in.PupilID, reporterPersonID, in.RequestID, clientIP,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert CREATE_POSTBOX audit log: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit safeguarding postbox transaction: %w", err)
	}

	d.emitPostCommitEvent(ctx, EventCoreSafeguardingPostboxCreated, map[string]any{
		"note_id":               noteID,
		"pupil_id":              in.PupilID,
		"concern_category":      in.ConcernCategory,
		"status":                "OPEN_TRIAGE",
		"incident_date":         in.IncidentDate,
		"reported_by_staff_id":  in.ReportedByStaffID,
		"assigned_dsl_staff_id": strings.TrimSpace(in.AssignedDSLStaffID),
	})

	return &SafeguardingPostboxReceipt{
		NoteID: noteID,
		Status: "OPEN_TRIAGE",
	}, nil
}

func (d *CoreDAL) IssueDSLCloudKeyToken(dslStaffID string, hostKMSKeyHandle string, ttl time.Duration) (string, error) {
	if strings.TrimSpace(dslStaffID) == "" {
		return "", errors.New("missing mandatory parameter: dslStaffID")
	}
	if strings.TrimSpace(hostKMSKeyHandle) == "" {
		return "", errors.New("missing mandatory parameter: hostKMSKeyHandle")
	}

	claims := dslCloudKeyClaims{
		StaffID:           strings.TrimSpace(dslStaffID),
		ExpiresAtUnixNano: time.Now().Add(ttl).UnixNano(),
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	b64Claims := base64.RawURLEncoding.EncodeToString(claimsJSON)

	mac := hmac.New(sha256.New, []byte("optimis-dsl-cloud-token:"+hostKMSKeyHandle))
	mac.Write([]byte(b64Claims))
	sig := hex.EncodeToString(mac.Sum(nil))

	return b64Claims + "." + sig, nil
}

func verifyDSLCloudKeyToken(token string, expectedStaffID string, hostKMSKeyHandle string) error {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return errors.New("malformed DSL cloud key token")
	}
	b64Claims, sigHex := parts[0], parts[1]

	mac := hmac.New(sha256.New, []byte("optimis-dsl-cloud-token:"+hostKMSKeyHandle))
	mac.Write([]byte(b64Claims))
	expectedSig := mac.Sum(nil)

	providedSig, err := hex.DecodeString(sigHex)
	if err != nil || !hmac.Equal(providedSig, expectedSig) {
		return errors.New("invalid DSL cloud key token cryptographic signature")
	}

	claimsBytes, err := base64.RawURLEncoding.DecodeString(b64Claims)
	if err != nil {
		return errors.New("invalid DSL cloud key token encoding")
	}
	var claims dslCloudKeyClaims
	if err := json.Unmarshal(claimsBytes, &claims); err != nil {
		return errors.New("invalid DSL cloud key token claims")
	}
	if claims.StaffID != expectedStaffID {
		return fmt.Errorf("DSL cloud key token staff_id mismatch: token issued for %s, caller is %s", claims.StaffID, expectedStaffID)
	}
	if time.Now().UnixNano() > claims.ExpiresAtUnixNano {
		return errors.New("DSL cloud key token has expired; OAuth cloud key renewal required")
	}
	return nil
}

func (d *CoreDAL) ReadSafeguardingNoteDSL(
	ctx context.Context,
	in ReadSafeguardingNoteDSLInput,
) (*DecryptedSafeguardingNote, error) {
	if strings.TrimSpace(in.NoteID) == "" {
		return nil, errors.New("missing mandatory parameter: note_id")
	}
	if strings.TrimSpace(in.DSLStaffID) == "" {
		return nil, errors.New("missing mandatory parameter: dsl_staff_id")
	}
	if strings.TrimSpace(in.CloudKeyToken) == "" {
		return nil, errors.New("missing mandatory parameter: cloud_key_token")
	}
	if strings.TrimSpace(in.HostKMSKeyHandle) == "" {
		return nil, errors.New("missing mandatory parameter: host_kms_key_handle")
	}
	if strings.TrimSpace(in.RequestID) == "" {
		return nil, errors.New("missing mandatory parameter: request_id")
	}
	clientIP := strings.TrimSpace(in.ClientIP)
	if clientIP == "" {
		clientIP = "127.0.0.1"
	}

	// 1. Verify staff member is an active Designated Safeguarding Lead (is_dsl == 1)
	var dslPersonID string
	var isDSL int
	err := d.readDB.QueryRowContext(
		ctx,
		`SELECT person_id, is_dsl FROM staff WHERE id = ?`,
		in.DSLStaffID,
	).Scan(&dslPersonID, &isDSL)
	if err != nil {
		return nil, fmt.Errorf("failed to verify DSL staff record: %w", err)
	}
	if isDSL != 1 {
		return nil, fmt.Errorf("unauthorized: staff %s is not a Designated Safeguarding Lead (is_dsl=0)", in.DSLStaffID)
	}

	// 2. Verify short-lived signed OAuth Cloud Key token
	if err := verifyDSLCloudKeyToken(in.CloudKeyToken, in.DSLStaffID, in.HostKMSKeyHandle); err != nil {
		return nil, fmt.Errorf("unauthorized DSL cloud key token: %w", err)
	}

	// 3. Read encrypted row from safeguarding_notes
	var note DecryptedSafeguardingNote
	var kmsKeyID string
	var wrappedDEK, nonce, ciphertext []byte
	err = d.readDB.QueryRowContext(
		ctx,
		`SELECT id, pupil_id, reported_by_staff_id, COALESCE(assigned_dsl_staff_id, ''),
		        concern_category, status, incident_date, kms_key_id, wrapped_dek, nonce, ciphertext_payload
		 FROM safeguarding_notes
		 WHERE id = ?`,
		in.NoteID,
	).Scan(
		&note.NoteID,
		&note.PupilID,
		&note.ReportedByStaffID,
		&note.AssignedDSLStaffID,
		&note.ConcernCategory,
		&note.Status,
		&note.IncidentDate,
		&kmsKeyID,
		&wrappedDEK,
		&nonce,
		&ciphertext,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to read safeguarding note %s: %w", in.NoteID, err)
	}
	if kmsKeyID != in.HostKMSKeyHandle {
		return nil, errors.New("KMS key handle mismatch for safeguarding note envelope")
	}
	if len(wrappedDEK) <= 12 {
		return nil, errors.New("corrupted wrapped_dek in safeguarding note")
	}

	// 4. Unwrap DEK using DSL Vault Master Key
	masterKey := deriveDSLVaultMasterKey(in.HostKMSKeyHandle)
	masterBlock, err := aes.NewCipher(masterKey[:])
	if err != nil {
		return nil, err
	}
	masterGCM, err := cipher.NewGCM(masterBlock)
	if err != nil {
		return nil, err
	}
	wrapNonce := wrappedDEK[:12]
	wrappedCipher := wrappedDEK[12:]
	dek, err := masterGCM.Open(nil, wrapNonce, wrappedCipher, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to unwrap safeguarding DEK: %w", err)
	}

	// 5. Decrypt payload using unwrapped DEK
	dekBlock, err := aes.NewCipher(dek)
	if err != nil {
		return nil, err
	}
	dekGCM, err := cipher.NewGCM(dekBlock)
	if err != nil {
		return nil, err
	}
	plaintextJSON, err := dekGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt safeguarding payload: %w", err)
	}

	var payload safeguardingEncryptedPayload
	if err := json.Unmarshal(plaintextJSON, &payload); err != nil {
		return nil, fmt.Errorf("failed to decode decrypted safeguarding JSON: %w", err)
	}
	note.Summary = payload.Summary
	note.BodyMap = payload.BodyMap
	note.VerbatimAccount = payload.VerbatimAccount
	note.ActionTaken = payload.ActionTaken

	// 6. Write immutable DSL_DECRYPT_VIEW entry to safeguarding_audit_logs
	auditID := "aud_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	if _, err := d.writeDB.ExecContext(
		ctx,
		`INSERT INTO safeguarding_audit_logs (
			audit_id, note_id, pupil_id, actor_person_id, action_type, request_id, client_ip
		) VALUES (?, ?, ?, ?, 'DSL_DECRYPT_VIEW', ?, ?)`,
		auditID, note.NoteID, note.PupilID, dslPersonID, in.RequestID, clientIP,
	); err != nil {
		return nil, fmt.Errorf("failed to write DSL_DECRYPT_VIEW audit log: %w", err)
	}

	auditRows, err := d.readDB.QueryContext(
		ctx,
		`SELECT audit_id, note_id, pupil_id, actor_person_id, action_type, request_id, client_ip, timestamp
		 FROM safeguarding_audit_logs
		 WHERE note_id = ?
		 ORDER BY timestamp ASC, audit_id ASC`,
		note.NoteID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query safeguarding_audit_logs for note %s: %w", note.NoteID, err)
	}
	defer auditRows.Close()

	for auditRows.Next() {
		var entry SafeguardingAuditLogEntry
		if scanErr := auditRows.Scan(
			&entry.AuditID,
			&entry.NoteID,
			&entry.PupilID,
			&entry.ActorPersonID,
			&entry.ActionType,
			&entry.RequestID,
			&entry.ClientIP,
			&entry.Timestamp,
		); scanErr != nil {
			return nil, fmt.Errorf("failed to scan safeguarding_audit_logs row: %w", scanErr)
		}
		note.AuditLogs = append(note.AuditLogs, entry)
	}
	if err := auditRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating safeguarding_audit_logs: %w", err)
	}

	return &note, nil
}

func (d *CoreDAL) ListSafeguardingNoteEnvelopesDSL(
	ctx context.Context,
	dslStaffID string,
	pupilID string,
) ([]SafeguardingNoteEnvelopeMetadata, error) {
	trimmedDSL := strings.TrimSpace(dslStaffID)
	if trimmedDSL == "" {
		return nil, errors.New("missing mandatory parameter: dsl_staff_id")
	}

	var isDSL int
	err := d.readDB.QueryRowContext(
		ctx,
		`SELECT is_dsl FROM staff WHERE id = ?`,
		trimmedDSL,
	).Scan(&isDSL)
	if err != nil {
		return nil, fmt.Errorf("failed to verify DSL staff record: %w", err)
	}
	if isDSL != 1 {
		return nil, fmt.Errorf("unauthorized: staff %s is not a Designated Safeguarding Lead (is_dsl=0)", trimmedDSL)
	}

	var rows *sql.Rows
	trimmedPupil := strings.TrimSpace(pupilID)
	if trimmedPupil != "" {
		rows, err = d.readDB.QueryContext(
			ctx,
			`SELECT id, pupil_id, reported_by_staff_id, COALESCE(assigned_dsl_staff_id, ''),
			        concern_category, status, incident_date, kms_key_id, created_at
			 FROM safeguarding_notes
			 WHERE pupil_id = ?
			 ORDER BY created_at DESC, id ASC`,
			trimmedPupil,
		)
	} else {
		rows, err = d.readDB.QueryContext(
			ctx,
			`SELECT id, pupil_id, reported_by_staff_id, COALESCE(assigned_dsl_staff_id, ''),
			        concern_category, status, incident_date, kms_key_id, created_at
			 FROM safeguarding_notes
			 ORDER BY created_at DESC, id ASC`,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query safeguarding note envelopes: %w", err)
	}
	defer rows.Close()

	result := make([]SafeguardingNoteEnvelopeMetadata, 0)
	for rows.Next() {
		var meta SafeguardingNoteEnvelopeMetadata
		if err := rows.Scan(
			&meta.NoteID,
			&meta.PupilID,
			&meta.ReportedByStaffID,
			&meta.AssignedDSLStaffID,
			&meta.ConcernCategory,
			&meta.Status,
			&meta.IncidentDate,
			&meta.KMSKeyID,
			&meta.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan safeguarding note envelope metadata: %w", err)
		}
		result = append(result, meta)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating safeguarding note envelopes: %w", err)
	}

	return result, nil
}

// ============================================================================
// 6. Milestone 1 Test Fixture Seeding & Teardown
// ============================================================================

type Milestone1SeedInput struct {
	AcademicYearID string `json:"academic_year_id"`
	ReferenceDate  string `json:"reference_date"`
}

type Milestone1SeedResult struct {
	FixtureID                  string `json:"fixture_id"`
	AcademicYearID             string `json:"academic_year_id"`
	ReferenceDate              string `json:"reference_date"`
	TeacherStaffID             string `json:"teacher_staff_id"`
	OtherTeacherStaffID        string `json:"other_teacher_staff_id"`
	DSLStaffID                 string `json:"dsl_staff_id"`
	OwnGroupID                 string `json:"own_group_id"`
	OtherGroupID               string `json:"other_group_id"`
	ActivePupilID              string `json:"active_pupil_id"`
	TransferredPupilID         string `json:"transferred_pupil_id"`
	OtherGroupPupilID          string `json:"other_group_pupil_id"`
	UnrestrictedMotherPersonID string `json:"unrestricted_mother_person_id"`
	RestrictedFatherPersonID   string `json:"restricted_father_person_id"`
	SafeHouseholdID            string `json:"safe_household_id"`
	ConfidentialHouseholdID    string `json:"confidential_household_id"`
}

func (d *CoreDAL) SeedMilestone1Fixtures(ctx context.Context, in Milestone1SeedInput) (*Milestone1SeedResult, error) {
	ayID := strings.TrimSpace(in.AcademicYearID)
	refDate := strings.TrimSpace(in.ReferenceDate)
	if ayID == "" {
		return nil, errors.New("missing mandatory field: academic_year_id")
	}
	if refDate == "" {
		return nil, errors.New("missing mandatory field: reference_date")
	}

	sfx := strings.ReplaceAll(uuid.NewString(), "-", "")[:8]
	res := &Milestone1SeedResult{
		FixtureID:                  sfx,
		AcademicYearID:             ayID,
		ReferenceDate:              refDate,
		TeacherStaffID:             "stf_t1_" + sfx,
		OtherTeacherStaffID:        "stf_t2_" + sfx,
		DSLStaffID:                 "stf_dsl_" + sfx,
		OwnGroupID:                 "grp_own_" + sfx,
		OtherGroupID:               "grp_oth_" + sfx,
		ActivePupilID:              "pup_act_" + sfx,
		TransferredPupilID:         "pup_trn_" + sfx,
		OtherGroupPupilID:          "pup_oth_" + sfx,
		UnrestrictedMotherPersonID: "per_mom_" + sfx,
		RestrictedFatherPersonID:   "per_dad_" + sfx,
		SafeHouseholdID:            "hh_safe_" + sfx,
		ConfidentialHouseholdID:    "hh_conf_" + sfx,
	}

	tx, err := d.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin SeedMilestone1Fixtures transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	stmts := []struct {
		sql  string
		args []any
	}{
		{
			sql: `INSERT INTO academic_years (id, name, status, sequence_number, first_day, last_day)
			      VALUES (?, '2026/2027', 'CURRENT', (SELECT COALESCE(MAX(sequence_number), 0) + 1 FROM academic_years), '2026-09-01', '2027-07-23')
			      ON CONFLICT(id) DO NOTHING`,
			args: []any{ayID},
		},
		{
			sql: `INSERT INTO alert_levels (id, name, name_short, color_hex, bg_color_hex, description, sequence_number)
			      VALUES ('HIGH', 'High Priority', 'HI', '#FFFFFF', '#D32F2F', 'Critical alert', 1)
			      ON CONFLICT(id) DO NOTHING`,
		},
		{
			sql: `INSERT INTO sen_need_types (code, name, broad_area, sequence_number)
			      VALUES ('ASD', 'Autistic Spectrum Disorder', 'COMMUNICATION_AND_INTERACTION', 1)
			      ON CONFLICT(code) DO NOTHING`,
		},
		{
			sql: `INSERT INTO attendance_codes (code, name, statistical_category, physical_status, sequence_number)
			      VALUES ('/', 'Present (AM)', 'PRESENT', 'ONSITE', 1),
			             ('\', 'Present (PM)', 'PRESENT', 'ONSITE', 2),
			             ('L', 'Late before register closed', 'PRESENT', 'ONSITE_LATE', 3)
			      ON CONFLICT(code) DO NOTHING`,
		},
		{
			sql: `INSERT INTO persons (id, legal_first_name, legal_last_name, sex, dob, phone_primary, status)
			      VALUES (?, 'Ada', 'Lovelace', 'F', '2012-12-10', '', 'ACTIVE'),
			             (?, 'Charles', 'Babbage', 'M', '2012-10-15', '', 'ACTIVE'),
			             (?, 'Alan', 'Turing', 'M', '2012-06-23', '', 'ACTIVE'),
			             (?, 'Grace', 'Hopper', 'F', '1985-05-01', '', 'ACTIVE'),
			             (?, 'Katherine', 'Johnson', 'F', '1984-08-26', '', 'ACTIVE'),
			             (?, 'Margaret', 'Hamilton', 'F', '1980-08-17', '', 'ACTIVE'),
			             (?, 'Anne', 'Byron', 'F', '1988-03-14', '07700900111', 'ACTIVE'),
			             (?, 'George', 'Byron', 'M', '1987-01-22', '07700900222', 'ACTIVE'),
			             (?, 'Judith', 'Milbanke', 'F', '1960-04-10', '07700900333', 'ACTIVE')`,
			args: []any{
				"per_p1_" + sfx,
				"per_p2_" + sfx,
				"per_p3_" + sfx,
				"per_t1_" + sfx,
				"per_t2_" + sfx,
				"per_dsl_" + sfx,
				res.UnrestrictedMotherPersonID,
				res.RestrictedFatherPersonID,
				"per_gp_" + sfx,
			},
		},
		{
			sql: `INSERT INTO pupils (id, person_id, upn, ethnicity_code, first_language_code)
			      VALUES (?, ?, ?, 'WBRI', 'ENG'),
			             (?, ?, ?, 'WBRI', 'ENG'),
			             (?, ?, ?, 'WBRI', 'ENG')`,
			args: []any{
				res.ActivePupilID, "per_p1_" + sfx, "U1" + strings.ToUpper(sfx) + "001",
				res.TransferredPupilID, "per_p2_" + sfx, "U2" + strings.ToUpper(sfx) + "002",
				res.OtherGroupPupilID, "per_p3_" + sfx, "U3" + strings.ToUpper(sfx) + "003",
			},
		},
		{
			sql: `INSERT INTO pupil_year_enrolments (id, pupil_id, academic_year_id, nc_year_group, enrolment_status, effective_from, effective_to)
			      VALUES (?, ?, ?, '10', 'C', '2026-09-01', NULL),
			             (?, ?, ?, '10', 'C', '2026-09-01', NULL),
			             (?, ?, ?, '10', 'C', '2026-09-01', NULL)`,
			args: []any{
				"enr_1_" + sfx, res.ActivePupilID, ayID,
				"enr_2_" + sfx, res.TransferredPupilID, ayID,
				"enr_3_" + sfx, res.OtherGroupPupilID, ayID,
			},
		},
		{
			sql: `INSERT INTO staff (id, person_id, staff_code, trn, staff_type, job_title, qts_flag, is_dsl, hire_date)
			      VALUES (?, ?, ?, ?, 'TEACHING', 'Teacher of Maths', 1, 0, '2020-09-01'),
			             (?, ?, ?, ?, 'TEACHING', 'Teacher of Maths', 1, 0, '2021-09-01'),
			             (?, ?, ?, ?, 'LEADERSHIP', 'DSL', 1, 1, '2018-09-01')`,
			args: []any{
				res.TeacherStaffID, "per_t1_" + sfx, "T1_" + sfx[:4], "71" + sfx[:5],
				res.OtherTeacherStaffID, "per_t2_" + sfx, "T2_" + sfx[:4], "72" + sfx[:5],
				res.DSLStaffID, "per_dsl_" + sfx, "DS_" + sfx[:4], "73" + sfx[:5],
			},
		},
		{
			sql: `INSERT INTO groups (id, academic_year_id, group_type, code, name, takes_attendance, effective_from, effective_to)
			      VALUES (?, ?, 'TEACHING_CLASS', ?, 'Year 10 Maths Set 1', 1, '2026-09-01', NULL),
			             (?, ?, 'TEACHING_CLASS', ?, 'Year 10 Maths Set 2', 1, '2026-09-01', NULL)`,
			args: []any{
				res.OwnGroupID, ayID, "10Ma1_" + sfx[:4],
				res.OtherGroupID, ayID, "10Ma2_" + sfx[:4],
			},
		},
		{
			sql: `INSERT INTO group_staff_assignments (id, group_id, staff_id, role, academic_year_id, effective_from, effective_to)
			      VALUES (?, ?, ?, 'LEAD_TEACHER', ?, '2026-09-01', NULL),
			             (?, ?, ?, 'LEAD_TEACHER', ?, '2026-09-01', NULL)`,
			args: []any{
				"gsa_1_" + sfx, res.OwnGroupID, res.TeacherStaffID, ayID,
				"gsa_2_" + sfx, res.OtherGroupID, res.OtherTeacherStaffID, ayID,
			},
		},
		{
			sql: `INSERT INTO group_pupil_memberships (id, group_id, pupil_id, academic_year_id, effective_from, effective_to)
			      VALUES (?, ?, ?, ?, '2026-09-01', NULL),
			             (?, ?, ?, ?, '2026-09-01', '2026-10-15'),
			             (?, ?, ?, ?, '2026-10-16', NULL),
			             (?, ?, ?, ?, '2026-09-01', NULL)`,
			args: []any{
				"gpm_act_" + sfx, res.OwnGroupID, res.ActivePupilID, ayID,
				"gpm_tro_" + sfx, res.OwnGroupID, res.TransferredPupilID, ayID,
				"gpm_trn_" + sfx, res.OtherGroupID, res.TransferredPupilID, ayID,
				"gpm_oth_" + sfx, res.OtherGroupID, res.OtherGroupPupilID, ayID,
			},
		},
		{
			sql: `INSERT INTO pupil_sen_records (id, pupil_id, provision_code, ehcp_start_date, teaching_strategies, effective_from, effective_to)
			      VALUES (?, ?, 'E', '2026-10-11', 'Chunk instructions and provide visual timetable', '2026-10-11', NULL)`,
			args: []any{"sen_" + sfx, res.ActivePupilID},
		},
		{
			sql: `INSERT INTO pupil_sen_needs (id, pupil_id, sen_need_code, ranking, alert_level_id, effective_from, effective_to)
			      VALUES (?, ?, 'ASD', 1, 'HIGH', '2026-09-01', NULL)`,
			args: []any{"sneed_" + sfx, res.ActivePupilID},
		},
		{
			sql: `INSERT INTO pupil_medical_conditions (
				id, pupil_id, condition_name, condition_category, alert_level_id,
				has_ihcp, emergency_response_plan, parental_consent_given, effective_from, effective_to
			) VALUES (?, ?, 'Anaphylaxis (Peanut Allergy)', 'ALLERGY', 'HIGH', 1, 'Administer EpiPen immediately and call 999', 1, '2026-09-01', NULL)`,
			args: []any{"med_" + sfx, res.ActivePupilID},
		},
		{
			sql: `INSERT INTO households (id, name, address_line_1, town_city, postcode, is_confidential_address)
			      VALUES (?, 'Byron Household', '42 High Street', 'London', 'SW1A 2BB', 0),
			             (?, 'Confidential Refuge', '10 Safe Haven Way', 'London', 'SW1A 1AA', 1)`,
			args: []any{res.SafeHouseholdID, res.ConfidentialHouseholdID},
		},
		{
			sql: `INSERT INTO household_memberships (id, household_id, person_id, member_role, residency_type, effective_from, effective_to)
			      VALUES (?, ?, ?, 'ADULT', 'PRIMARY', '2026-09-01', NULL),
			             (?, ?, ?, 'CHILD', 'PRIMARY', '2026-09-01', NULL),
			             (?, ?, ?, 'ADULT', 'SECONDARY_SHARED', '2026-09-01', NULL)`,
			args: []any{
				"hm_mom_" + sfx, res.SafeHouseholdID, res.UnrestrictedMotherPersonID,
				"hm_pup_" + sfx, res.SafeHouseholdID, "per_p1_" + sfx,
				"hm_gp_" + sfx, res.ConfidentialHouseholdID, "per_gp_" + sfx,
			},
		},
		{
			sql: `INSERT INTO guardian_relationships (
				id, pupil_id, guardian_person_id, relationship_code,
				has_parental_responsibility, court_order_restricted, court_order_notes,
				contact_priority, is_emergency_contact, authorised_to_collect,
				portal_data_access, receive_reports, contact_by_phone, contact_by_sms, contact_by_email, contact_by_post,
				effective_from, effective_to
			) VALUES
				(?, ?, ?, 'MOTHER', 1, 0, '', 1, 1, 1, 1, 1, 1, 1, 1, 1, '2026-09-01', NULL),
				(?, ?, ?, 'FATHER', 1, 1, 'Prohibited Steps Order - Do Not Contact', 2, 0, 0, 0, 0, 0, 0, 0, 0, '2026-09-01', NULL),
				(?, ?, ?, 'GRANDPARENT', 0, 0, '', 3, 1, 1, 0, 0, 1, 1, 1, 0, '2026-09-01', NULL)`,
			args: []any{
				"grel_mom_" + sfx, res.ActivePupilID, res.UnrestrictedMotherPersonID,
				"grel_dad_" + sfx, res.ActivePupilID, res.RestrictedFatherPersonID,
				"grel_gp_" + sfx, res.ActivePupilID, "per_gp_" + sfx,
			},
		},
	}

	for _, stmt := range stmts {
		if _, err := tx.ExecContext(ctx, stmt.sql, stmt.args...); err != nil {
			return nil, fmt.Errorf("failed to execute Milestone 1 seed statement: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit SeedMilestone1Fixtures transaction: %w", err)
	}
	return res, nil
}

func (d *CoreDAL) DeleteMilestone1Fixtures(ctx context.Context, fixtureID string) error {
	sfx := strings.TrimSpace(fixtureID)
	if sfx == "" {
		return errors.New("missing mandatory parameter: fixture_id")
	}

	activePupilID := "pup_act_" + sfx
	transPupilID := "pup_trn_" + sfx
	otherPupilID := "pup_oth_" + sfx
	ownGroupID := "grp_own_" + sfx
	otherGroupID := "grp_oth_" + sfx

	tx, err := d.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin DeleteMilestone1Fixtures transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// Allow test teardown to purge any safeguarding postbox notes created for this fixture
	if _, err := tx.ExecContext(
		ctx,
		`UPDATE safeguarding_notes SET retention_purge_eligible = 1 WHERE pupil_id IN (?, ?, ?)`,
		activePupilID, transPupilID, otherPupilID,
	); err != nil {
		return fmt.Errorf("failed to mark fixture safeguarding_notes purge eligible: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DROP TRIGGER IF EXISTS trg_safeguarding_audit_no_delete;`); err != nil {
		return fmt.Errorf("failed to temporarily drop audit delete trigger for fixture cleanup: %w", err)
	}
	if _, err := tx.ExecContext(
		ctx,
		`DELETE FROM safeguarding_audit_logs WHERE pupil_id IN (?, ?, ?)`,
		activePupilID, transPupilID, otherPupilID,
	); err != nil {
		return fmt.Errorf("failed to delete fixture safeguarding_audit_logs: %w", err)
	}
	if _, err := tx.ExecContext(
		ctx,
		`CREATE TRIGGER IF NOT EXISTS trg_safeguarding_audit_no_delete
		 BEFORE DELETE ON safeguarding_audit_logs
		 BEGIN
		     SELECT RAISE(ABORT, 'Security Violation: Statutory safeguarding audit logs are immutable and cannot be deleted.');
		 END;`,
	); err != nil {
		return fmt.Errorf("failed to restore trg_safeguarding_audit_no_delete: %w", err)
	}
	if _, err := tx.ExecContext(
		ctx,
		`DELETE FROM safeguarding_notes WHERE pupil_id IN (?, ?, ?)`,
		activePupilID, transPupilID, otherPupilID,
	); err != nil {
		return fmt.Errorf("failed to delete fixture safeguarding_notes: %w", err)
	}

	cleanupQueries := []struct {
		sql  string
		args []any
	}{
		{
			sql:  `DELETE FROM attendance_session_marks WHERE pupil_id IN (?, ?, ?)`,
			args: []any{activePupilID, transPupilID, otherPupilID},
		},
		{
			sql:  `DELETE FROM attendance_register_submissions WHERE group_id IN (?, ?)`,
			args: []any{ownGroupID, otherGroupID},
		},
		{
			sql:  `DELETE FROM guardian_relationships WHERE pupil_id IN (?, ?, ?)`,
			args: []any{activePupilID, transPupilID, otherPupilID},
		},
		{
			sql:  `DELETE FROM household_memberships WHERE household_id IN (?, ?)`,
			args: []any{"hh_safe_" + sfx, "hh_conf_" + sfx},
		},
		{
			sql:  `DELETE FROM households WHERE id IN (?, ?)`,
			args: []any{"hh_safe_" + sfx, "hh_conf_" + sfx},
		},
		{
			sql:  `DELETE FROM pupil_medical_conditions WHERE pupil_id IN (?, ?, ?)`,
			args: []any{activePupilID, transPupilID, otherPupilID},
		},
		{
			sql:  `DELETE FROM pupil_sen_needs WHERE pupil_id IN (?, ?, ?)`,
			args: []any{activePupilID, transPupilID, otherPupilID},
		},
		{
			sql:  `DELETE FROM pupil_sen_records WHERE pupil_id IN (?, ?, ?)`,
			args: []any{activePupilID, transPupilID, otherPupilID},
		},
		{
			sql:  `DELETE FROM group_pupil_memberships WHERE group_id IN (?, ?)`,
			args: []any{ownGroupID, otherGroupID},
		},
		{
			sql:  `DELETE FROM group_staff_assignments WHERE group_id IN (?, ?)`,
			args: []any{ownGroupID, otherGroupID},
		},
		{
			sql:  `DELETE FROM groups WHERE id IN (?, ?)`,
			args: []any{ownGroupID, otherGroupID},
		},
		{
			sql:  `DELETE FROM staff WHERE id IN (?, ?, ?)`,
			args: []any{"stf_t1_" + sfx, "stf_t2_" + sfx, "stf_dsl_" + sfx},
		},
		{
			sql:  `DELETE FROM pupil_year_enrolments WHERE pupil_id IN (?, ?, ?)`,
			args: []any{activePupilID, transPupilID, otherPupilID},
		},
		{
			sql:  `DELETE FROM pupils WHERE id IN (?, ?, ?)`,
			args: []any{activePupilID, transPupilID, otherPupilID},
		},
		{
			sql: `DELETE FROM persons WHERE id IN (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			args: []any{
				"per_p1_" + sfx,
				"per_p2_" + sfx,
				"per_p3_" + sfx,
				"per_t1_" + sfx,
				"per_t2_" + sfx,
				"per_dsl_" + sfx,
				"per_mom_" + sfx,
				"per_dad_" + sfx,
				"per_gp_" + sfx,
			},
		},
	}

	for _, q := range cleanupQueries {
		if _, err := tx.ExecContext(ctx, q.sql, q.args...); err != nil {
			return fmt.Errorf("failed to execute Milestone 1 cleanup query (%s): %w", q.sql, err)
		}
	}

	return tx.Commit()
}
