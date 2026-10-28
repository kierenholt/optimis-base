package host

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"optimis/internal/dal"
)

func hasExplicitScope(scopes []string, required string) bool {
	for _, s := range scopes {
		if strings.TrimSpace(s) == required {
			return true
		}
	}
	return false
}

func (h *Host) RegisterSafeguardingRoutes(mux *http.ServeMux) {
	// 1. Write-Only Safeguarding Postbox (Requires dal.safeguarding.create — held by all staff)
	mux.HandleFunc("POST /api/safeguarding/postbox", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			PupilID            string `json:"pupil_id"`
			AssignedDSLStaffID string `json:"assigned_dsl_staff_id"`
			ConcernCategory    string `json:"concern_category"`
			IncidentDate       string `json:"incident_date"`
			Summary            string `json:"summary"`
			BodyMap            string `json:"body_map"`
			VerbatimAccount    string `json:"verbatim_account"`
			ActionTaken        string `json:"action_taken"`
			StaffID            string `json:"staff_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": fmt.Sprintf("invalid JSON body: %v", err),
			})
			return
		}

		caller, err := h.ResolveRequestCaller(r, body.StaffID)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]any{
				"ok":         false,
				"error_code": err.Error(),
			})
			return
		}

		if !hasExplicitScope(caller.Scopes, "dal.safeguarding.create") {
			h.logError(fmt.Sprintf("Unauthorized safeguarding postbox submission by %s (missing dal.safeguarding.create)", caller.StaffID))
			writeJSON(w, http.StatusForbidden, map[string]any{
				"ok":         false,
				"error_code": "UNAUTHORIZED_SCOPE: caller lacks required scope dal.safeguarding.create",
			})
			return
		}

		receipt, createErr := h.coreDAL.CreateSafeguardingPostboxNote(r.Context(), dal.CreateSafeguardingPostboxInput{
			PupilID:            body.PupilID,
			ReportedByStaffID:  caller.StaffID,
			AssignedDSLStaffID: body.AssignedDSLStaffID,
			ConcernCategory:    body.ConcernCategory,
			IncidentDate:       body.IncidentDate,
			Summary:            body.Summary,
			BodyMap:            body.BodyMap,
			VerbatimAccount:    body.VerbatimAccount,
			ActionTaken:        body.ActionTaken,
			RequestID:          caller.RequestID,
			ClientIP:           "127.0.0.1",
			HostKMSKeyHandle:   h.cfg.HostKMSKeyHandle,
		})
		if createErr != nil {
			h.logError(fmt.Sprintf("Failed to create safeguarding postbox note: %v", createErr))
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": createErr.Error(),
			})
			return
		}

		h.logInfo(fmt.Sprintf("Created safeguarding postbox note %s by staff %s", receipt.NoteID, caller.StaffID))
		writeJSON(w, http.StatusCreated, map[string]any{
			"ok":      true,
			"receipt": receipt,
		})
	})

	// 2. DSL Vault Envelope Metadata Listing (Requires dal.safeguarding.read; returns ZERO decrypted payload)
	mux.HandleFunc("GET /api/safeguarding/notes", func(w http.ResponseWriter, r *http.Request) {
		caller, err := h.ResolveRequestCaller(r, r.URL.Query().Get("staff_id"))
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]any{
				"ok":         false,
				"error_code": err.Error(),
			})
			return
		}

		if !hasExplicitScope(caller.Scopes, "dal.safeguarding.read") {
			h.logError(fmt.Sprintf("Unauthorized DSL vault list attempt by %s (missing dal.safeguarding.read)", caller.StaffID))
			writeJSON(w, http.StatusForbidden, map[string]any{
				"ok":         false,
				"error_code": "UNAUTHORIZED_SCOPE: caller lacks required scope dal.safeguarding.read",
			})
			return
		}

		pupilID := strings.TrimSpace(r.URL.Query().Get("pupil_id"))
		notes, listErr := h.coreDAL.ListSafeguardingNoteEnvelopesDSL(r.Context(), caller.StaffID, pupilID)
		if listErr != nil {
			status := http.StatusBadRequest
			if strings.Contains(listErr.Error(), "unauthorized") {
				status = http.StatusForbidden
			}
			h.logError(fmt.Sprintf("Failed to list safeguarding note envelopes for DSL %s: %v", caller.StaffID, listErr))
			writeJSON(w, status, map[string]any{
				"ok":         false,
				"error_code": listErr.Error(),
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"ok":    true,
			"notes": notes,
		})
	})

	// 3. DSL OAuth Cloud Key Renewal Endpoint (Requires dal.safeguarding.read & is_dsl = 1)
	mux.HandleFunc("POST /api/safeguarding/cloud-key/renew", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			TTLSeconds *int64 `json:"ttl_seconds"`
			StaffID    string `json:"staff_id"`
		}
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&body)
		}

		caller, err := h.ResolveRequestCaller(r, body.StaffID)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]any{
				"ok":         false,
				"error_code": err.Error(),
			})
			return
		}

		if !hasExplicitScope(caller.Scopes, "dal.safeguarding.read") {
			h.logError(fmt.Sprintf("Unauthorized DSL cloud key renewal attempt by %s (missing dal.safeguarding.read)", caller.StaffID))
			writeJSON(w, http.StatusForbidden, map[string]any{
				"ok":         false,
				"error_code": "UNAUTHORIZED_SCOPE: caller lacks required scope dal.safeguarding.read",
			})
			return
		}

		// Verify caller is an active DSL in core.db
		if _, listErr := h.coreDAL.ListSafeguardingNoteEnvelopesDSL(r.Context(), caller.StaffID, ""); listErr != nil {
			writeJSON(w, http.StatusForbidden, map[string]any{
				"ok":         false,
				"error_code": listErr.Error(),
			})
			return
		}

		ttl := 24 * time.Hour
		if body.TTLSeconds != nil {
			ttl = time.Duration(*body.TTLSeconds) * time.Second
		}

		token, tokenErr := h.coreDAL.IssueDSLCloudKeyToken(caller.StaffID, h.cfg.HostKMSKeyHandle, ttl)
		if tokenErr != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": tokenErr.Error(),
			})
			return
		}

		h.logInfo(fmt.Sprintf("Issued signed DSL OAuth Cloud Key token for DSL %s", caller.StaffID))
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":              true,
			"staff_id":        caller.StaffID,
			"cloud_key_token": token,
		})
	})

	// 4. DSL Envelope Decryption Endpoint (Requires dal.safeguarding.read, is_dsl = 1, and valid signed CloudKeyToken)
	mux.HandleFunc("POST /api/safeguarding/notes/{note_id}/decrypt", func(w http.ResponseWriter, r *http.Request) {
		noteID := strings.TrimSpace(r.PathValue("note_id"))
		if noteID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": "missing mandatory path parameter: note_id",
			})
			return
		}

		var body struct {
			CloudKeyToken string `json:"cloud_key_token"`
			StaffID       string `json:"staff_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": fmt.Sprintf("invalid JSON body: %v", err),
			})
			return
		}

		caller, err := h.ResolveRequestCaller(r, body.StaffID)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]any{
				"ok":         false,
				"error_code": err.Error(),
			})
			return
		}

		if !hasExplicitScope(caller.Scopes, "dal.safeguarding.read") {
			h.logError(fmt.Sprintf("Unauthorized DSL vault decrypt attempt by %s on note %s", caller.StaffID, noteID))
			writeJSON(w, http.StatusForbidden, map[string]any{
				"ok":         false,
				"error_code": "UNAUTHORIZED_SCOPE: caller lacks required scope dal.safeguarding.read",
			})
			return
		}

		decrypted, decErr := h.coreDAL.ReadSafeguardingNoteDSL(r.Context(), dal.ReadSafeguardingNoteDSLInput{
			NoteID:           noteID,
			DSLStaffID:       caller.StaffID,
			CloudKeyToken:    body.CloudKeyToken,
			HostKMSKeyHandle: h.cfg.HostKMSKeyHandle,
			RequestID:        caller.RequestID,
			ClientIP:         "127.0.0.1",
		})
		if decErr != nil {
			status := http.StatusBadRequest
			if strings.Contains(decErr.Error(), "unauthorized") {
				status = http.StatusForbidden
			}
			h.logError(fmt.Sprintf("DSL vault decrypt rejected for %s on note %s: %v", caller.StaffID, noteID, decErr))
			writeJSON(w, status, map[string]any{
				"ok":         false,
				"error_code": decErr.Error(),
			})
			return
		}

		h.logInfo(fmt.Sprintf("DSL %s unlocked and decrypted safeguarding note %s", caller.StaffID, noteID))
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":   true,
			"note": decrypted,
		})
	})
}
