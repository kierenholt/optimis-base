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

	if err := verifyDSLCloudKeyToken(in.CloudKeyToken, in.DSLStaffID, in.HostKMSKeyHandle); err != nil {
		return nil, fmt.Errorf("unauthorized DSL cloud key token: %w", err)
	}

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
