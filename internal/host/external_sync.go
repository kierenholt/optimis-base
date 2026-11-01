package host

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"optimis/internal/dal"
)

type ParentDMZProjection struct {
	PupilID                          string   `json:"pupil_id"`
	FirstName                        string   `json:"first_name"`
	LastName                         string   `json:"last_name"`
	AcademicYearID                   string   `json:"academic_year_id"`
	ReferenceDate                    string   `json:"reference_date"`
	PresentSessions                  int      `json:"present_sessions"`
	TotalSessions                    int      `json:"total_sessions"`
	AuthorizedGuardianIDs            []string `json:"authorized_guardian_ids"`
	RedactedConfidentialAddressCount int      `json:"redacted_confidential_address_count"`
}

type ParentDMZSubmission struct {
	SubmissionID                string `json:"submission_id"`
	PupilID                     string `json:"pupil_id"`
	SubmittedByGuardianPersonID string `json:"submitted_by_guardian_person_id"`
	SubmissionType              string `json:"submission_type"`
	Date                        string `json:"date"`
	Session                     string `json:"session,omitempty"`
	Reason                      string `json:"reason"`
	Status                      string `json:"status"`
}

type MATApprovedPluginRef struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

type MATPolicyBundle struct {
	MATID                string                 `json:"mat_id"`
	GradingSchemaVersion string                 `json:"grading_schema_version"`
	GradingSchema        map[string]any         `json:"grading_schema,omitempty"`
	BehaviourTaxonomy    []string               `json:"behaviour_taxonomy"`
	ApprovedPlugins      []MATApprovedPluginRef `json:"approved_plugins"`
}

func defaultMATGradingSchemaForVersion(version string) map[string]any {
	v := strings.TrimSpace(version)
	if v == "" {
		return nil
	}
	return map[string]any{
		"version": v,
		"scales": map[string]any{
			"9-1": map[string]any{
				"scale_code":  "9-1",
				"description": "GCSE 9-1 Numerical Scale",
				"grades":      []any{"9", "8", "7", "6", "5", "4", "3", "2", "1", "U"},
				"points":      map[string]any{"9": 9, "8": 8, "7": 7, "6": 6, "5": 5, "4": 4, "3": 3, "2": 2, "1": 1, "U": 0},
			},
			"WT,EXS,GDS": map[string]any{
				"scale_code":  "WT,EXS,GDS",
				"description": "Primary / KS2-KS3 Attainment Standard Scale",
				"grades":      []any{"GDS", "EXS", "WT"},
				"points":      map[string]any{"GDS": 3, "EXS": 2, "WT": 1},
			},
		},
	}
}

type MATSchoolAnalytics struct {
	SchoolURN           string `json:"school_urn"`
	AcademicYearID      string `json:"academic_year_id"`
	EnrolledPupilCount  int    `json:"enrolled_pupil_count"`
	SenPupilCount       int    `json:"sen_pupil_count"`
	PresentSessionCount int    `json:"present_session_count"`
	TotalSessionCount   int    `json:"total_session_count"`
}

func (h *Host) RegisterExternalSyncAndTestRoutes(mux *http.ServeMux) {
	h.registerExternalDepsModeRoute(mux)

	// 2. Outbound Parent DMZ Sync: Push sanitized read-projection (Section 1)
	mux.HandleFunc("POST /api/sync/parent-dmz/push-projection", func(w http.ResponseWriter, r *http.Request) {
		if err := h.ensureParentDMZAvailable(r.Context()); err != nil {
			writeSyncError(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		var body struct {
			PupilID        string `json:"pupil_id"`
			ReferenceDate  string `json:"reference_date"`
			AcademicYearID string `json:"academic_year_id"`
			StaffID        string `json:"staff_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeSyncError(w, http.StatusBadRequest, fmt.Sprintf("invalid JSON body: %v", err))
			return
		}
		pupilID := strings.TrimSpace(body.PupilID)
		if pupilID == "" {
			writeSyncError(w, http.StatusBadRequest, "missing mandatory field: pupil_id")
			return
		}
		caller, err := h.ResolveRequestCaller(r, "")
		if err != nil {
			writeSyncError(w, http.StatusUnauthorized, err.Error())
			return
		}
		scope, err := h.coreDAL.ResolveScopeDefaults(r.Context(), dal.DALQueryScope{
			Mode:           dal.ScopeAll,
			StaffID:        caller.StaffID,
			AcademicYearID: strings.TrimSpace(body.AcademicYearID),
			ReferenceDate:  strings.TrimSpace(body.ReferenceDate),
		})
		if err != nil {
			writeSyncError(w, http.StatusBadRequest, err.Error())
			return
		}
		pupil, err := h.coreDAL.GetPupilByIDScoped(r.Context(), scope, pupilID)
		if err != nil {
			writeSyncError(w, http.StatusBadRequest, err.Error())
			return
		}
		allUnrestrictedGuardians, err := h.coreDAL.ListGuardiansForPupilScoped(r.Context(), scope, pupilID, dal.GuardianFilterOptions{RequireParentalResponsibility: false})
		if err != nil {
			writeSyncError(w, http.StatusBadRequest, err.Error())
			return
		}
		authorizedGuardianIDs := make([]string, 0)
		redactedConfidentialCount := 0
		for _, g := range allUnrestrictedGuardians {
			if g.IsConfidentialAddressRedacted {
				redactedConfidentialCount++
			}
			if g.HasParentalResponsibility {
				authorizedGuardianIDs = append(authorizedGuardianIDs, g.GuardianPersonID)
			}
		}
		var totalSessions, presentSessions int
		if err := h.coreDAL.ReadDB().QueryRowContext(
			r.Context(),
			`SELECT COUNT(*), COALESCE(SUM(CASE WHEN ac.statistical_category IN ('PRESENT', 'APPROVED_EDUCATIONAL_ACTIVITY') OR asm.attendance_code IN ('/', '\', 'L', 'B', 'P', 'V', 'W') THEN 1 ELSE 0 END), 0)
			 FROM attendance_session_marks asm LEFT JOIN attendance_codes ac ON ac.code = asm.attendance_code
			 WHERE asm.pupil_id = ? AND asm.academic_year_id = ?`,
			pupilID, scope.AcademicYearID,
		).Scan(&totalSessions, &presentSessions); err != nil {
			writeSyncError(w, http.StatusInternalServerError, fmt.Sprintf("failed to aggregate attendance sessions for DMZ projection: %v", err))
			return
		}
		projection := ParentDMZProjection{
			PupilID: pupil.ID, FirstName: pupil.FirstName, LastName: pupil.LastName,
			AcademicYearID: pupil.AcademicYearID, ReferenceDate: scope.ReferenceDate,
			PresentSessions: presentSessions, TotalSessions: totalSessions,
			AuthorizedGuardianIDs:            authorizedGuardianIDs,
			RedactedConfidentialAddressCount: redactedConfidentialCount,
		}
		st := h.getExternalDepsState()
		st.mu.Lock()
		st.dmzProjections[pupil.ID] = projection
		st.mu.Unlock()

		h.logInfo(fmt.Sprintf("Pushed sanitized parent DMZ read-projection for pupil (authorized_guardians=%d, redacted_addresses=%d)", len(authorizedGuardianIDs), redactedConfidentialCount))
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "projection": projection})
	})

	// 3. Test helper: Enqueue untrusted parent submission in Mock DMZ Write-Inbox
	mux.HandleFunc("POST /api/test/parent-dmz/enqueue-inbox", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			PupilID                     string `json:"pupil_id"`
			SubmittedByGuardianPersonID string `json:"submitted_by_guardian_person_id"`
			SubmissionType              string `json:"submission_type"`
			Date                        string `json:"date"`
			Session                     string `json:"session"`
			Reason                      string `json:"reason"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeSyncError(w, http.StatusBadRequest, fmt.Sprintf("invalid JSON body: %v", err))
			return
		}
		if strings.TrimSpace(body.PupilID) == "" || strings.TrimSpace(body.SubmittedByGuardianPersonID) == "" || strings.TrimSpace(body.SubmissionType) == "" {
			writeSyncError(w, http.StatusBadRequest, "missing mandatory fields: pupil_id, submitted_by_guardian_person_id, submission_type")
			return
		}
		sub := ParentDMZSubmission{
			SubmissionID:                "dmz_sub_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12],
			PupilID:                     strings.TrimSpace(body.PupilID),
			SubmittedByGuardianPersonID: strings.TrimSpace(body.SubmittedByGuardianPersonID),
			SubmissionType:              strings.TrimSpace(body.SubmissionType),
			Date:                        strings.TrimSpace(body.Date),
			Session:                     strings.TrimSpace(body.Session),
			Reason:                      strings.TrimSpace(body.Reason),
			Status:                      "QUEUED",
		}
		st := h.getExternalDepsState()
		st.mu.Lock()
		st.dmzInbox = append(st.dmzInbox, sub)
		st.mu.Unlock()
		writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "submission": sub})
	})

	// 4. Outbound Parent DMZ Sync: Pull queued submissions from DMZ Write-Inbox (Section 1)
	mux.HandleFunc("POST /api/sync/parent-dmz/pull-inbox", func(w http.ResponseWriter, r *http.Request) {
		if err := h.ensureParentDMZAvailable(r.Context()); err != nil {
			writeSyncError(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		var body struct {
			StaffID string `json:"staff_id"`
		}
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&body)
		}
		caller, err := h.ResolveRequestCaller(r, "")
		if err != nil {
			writeSyncError(w, http.StatusUnauthorized, err.Error())
			return
		}
		st := h.getExternalDepsState()
		st.mu.Lock()
		queued := make([]ParentDMZSubmission, len(st.dmzInbox))
		copy(queued, st.dmzInbox)
		st.dmzInbox = make([]ParentDMZSubmission, 0)
		st.mu.Unlock()

		processed := make([]ParentDMZSubmission, 0, len(queued))
		acceptedCount, rejectedCount := 0, 0
		for _, sub := range queued {
			scope, sErr := h.coreDAL.ResolveScopeDefaults(r.Context(), dal.DALQueryScope{
				Mode: dal.ScopeAll, StaffID: caller.StaffID, ReferenceDate: sub.Date,
			})
			if sErr != nil {
				sub.Status = "REJECTED_SAFEGUARDING_BLOCK"
				rejectedCount++
				processed = append(processed, sub)
				continue
			}
			prGuardians, gErr := h.coreDAL.ListGuardiansForPupilScoped(r.Context(), scope, sub.PupilID, dal.GuardianFilterOptions{RequireParentalResponsibility: true})
			authorized := false
			if gErr == nil {
				for _, g := range prGuardians {
					if g.GuardianPersonID == sub.SubmittedByGuardianPersonID {
						authorized = true
						break
					}
				}
			}
			if authorized {
				sub.Status = "ACCEPTED"
				acceptedCount++
				h.dispatchEvents([]Event{{
					Name: dal.EventCoreParentDMZSubmissionReceived,
					Payload: map[string]any{
						"submission_id": sub.SubmissionID, "pupil_id": sub.PupilID,
						"submitted_by_guardian_person_id": sub.SubmittedByGuardianPersonID,
						"submission_type": sub.SubmissionType, "date": sub.Date,
						"session": sub.Session, "reason": sub.Reason, "status": sub.Status,
					},
					Hop: 1,
				}})
			} else {
				sub.Status = "REJECTED_SAFEGUARDING_BLOCK"
				rejectedCount++
			}
			processed = append(processed, sub)
		}
		h.logInfo(fmt.Sprintf("Pulled %d parent DMZ write-inbox submissions (accepted=%d, rejected=%d)", len(processed), acceptedCount, rejectedCount))
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "accepted_count": acceptedCount, "rejected_count": rejectedCount, "submissions": processed})
	})

	// 5. Test helper: Seed Mock MAT Central Sync Feed
	mux.HandleFunc("POST /api/test/mat/seed-feed", func(w http.ResponseWriter, r *http.Request) {
		var bundle MATPolicyBundle
		if err := json.NewDecoder(r.Body).Decode(&bundle); err != nil {
			writeSyncError(w, http.StatusBadRequest, fmt.Sprintf("invalid JSON body: %v", err))
			return
		}
		if strings.TrimSpace(bundle.MATID) == "" || strings.TrimSpace(bundle.GradingSchemaVersion) == "" {
			writeSyncError(w, http.StatusBadRequest, "missing mandatory fields: mat_id, grading_schema_version")
			return
		}
		if len(bundle.GradingSchema) == 0 {
			bundle.GradingSchema = defaultMATGradingSchemaForVersion(bundle.GradingSchemaVersion)
		}
		st := h.getExternalDepsState()
		st.mu.Lock()
		st.matPolicyBundle = &bundle
		st.mu.Unlock()
		writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "policy_bundle": bundle})
	})

	// 6. Outbound MAT Central Sync: Pull MAT policies & approved plugin catalog (Sections 1 & 6A)
	mux.HandleFunc("POST /api/sync/mat/pull-policies", func(w http.ResponseWriter, r *http.Request) {
		if err := h.ensureMATSyncAvailable(r.Context()); err != nil {
			writeSyncError(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		var body struct {
			StaffID string `json:"staff_id"`
		}
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&body)
		}
		if _, err := h.ResolveRequestCaller(r, ""); err != nil {
			writeSyncError(w, http.StatusUnauthorized, err.Error())
			return
		}
		st := h.getExternalDepsState()
		st.mu.RLock()
		bundle := st.matPolicyBundle
		st.mu.RUnlock()
		if bundle == nil {
			writeSyncError(w, http.StatusBadRequest, "MAT_FEED_EMPTY: no MAT policy bundle available in mock feed")
			return
		}
		gradingSchemaMap := bundle.GradingSchema
		if len(gradingSchemaMap) == 0 {
			gradingSchemaMap = defaultMATGradingSchemaForVersion(bundle.GradingSchemaVersion)
		}
		gradingSchemaJSON, _ := json.Marshal(gradingSchemaMap)
		taxonomyJSON, _ := json.Marshal(bundle.BehaviourTaxonomy)
		pluginsJSON, _ := json.Marshal(bundle.ApprovedPlugins)

		configEntries := []struct{ key, val string }{
			{"mat_id", bundle.MATID},
			{"mat_grading_schema_version", bundle.GradingSchemaVersion},
			{"mat_grading_schema_json", string(gradingSchemaJSON)},
			{"mat_behaviour_taxonomy_json", string(taxonomyJSON)},
			{"mat_approved_plugins_json", string(pluginsJSON)},
		}
		for _, entry := range configEntries {
			if _, err := h.coreDAL.WriteDB().ExecContext(
				r.Context(),
				`INSERT INTO __system_config (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
				entry.key, entry.val,
			); err != nil {
				writeSyncError(w, http.StatusInternalServerError, fmt.Sprintf("failed to persist MAT policy key %s: %v", entry.key, err))
				return
			}
		}
		h.logInfo(fmt.Sprintf("Pulled MAT policy bundle from %s (schema=%s)", bundle.MATID, bundle.GradingSchemaVersion))
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "policy_bundle": bundle})
	})

	// 7. Outbound MAT Central Sync: Push aggregated school-level analytics (Section 1)
	mux.HandleFunc("POST /api/sync/mat/push-analytics", func(w http.ResponseWriter, r *http.Request) {
		if err := h.ensureMATSyncAvailable(r.Context()); err != nil {
			writeSyncError(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		var body struct {
			AcademicYearID string `json:"academic_year_id"`
			StaffID        string `json:"staff_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeSyncError(w, http.StatusBadRequest, fmt.Sprintf("invalid JSON body: %v", err))
			return
		}
		ayID := strings.TrimSpace(body.AcademicYearID)
		if ayID == "" {
			writeSyncError(w, http.StatusBadRequest, "missing mandatory field: academic_year_id")
			return
		}
		if _, err := h.ResolveRequestCaller(r, ""); err != nil {
			writeSyncError(w, http.StatusUnauthorized, err.Error())
			return
		}
		var enrolledCount, senCount, totalSessions, presentSessions int
		if err := h.coreDAL.ReadDB().QueryRowContext(
			r.Context(),
			`SELECT COUNT(DISTINCT pupil_id) FROM pupil_year_enrolments WHERE academic_year_id = ? AND (effective_to IS NULL OR effective_to >= date('now'))`,
			ayID,
		).Scan(&enrolledCount); err != nil {
			writeSyncError(w, http.StatusInternalServerError, fmt.Sprintf("failed to count enrolled pupils: %v", err))
			return
		}
		if err := h.coreDAL.ReadDB().QueryRowContext(
			r.Context(),
			`SELECT COUNT(DISTINCT psr.pupil_id) FROM pupil_sen_records psr JOIN pupil_year_enrolments enr ON enr.pupil_id = psr.pupil_id
			 WHERE enr.academic_year_id = ? AND psr.provision_code IN ('K', 'E') AND (psr.effective_to IS NULL OR psr.effective_to >= date('now'))`,
			ayID,
		).Scan(&senCount); err != nil {
			writeSyncError(w, http.StatusInternalServerError, fmt.Sprintf("failed to count SEN pupils: %v", err))
			return
		}
		if err := h.coreDAL.ReadDB().QueryRowContext(
			r.Context(),
			`SELECT COUNT(*), COALESCE(SUM(CASE WHEN ac.statistical_category IN ('PRESENT', 'APPROVED_EDUCATIONAL_ACTIVITY') OR asm.attendance_code IN ('/', '\', 'L', 'B', 'P', 'V', 'W') THEN 1 ELSE 0 END), 0)
			 FROM attendance_session_marks asm LEFT JOIN attendance_codes ac ON ac.code = asm.attendance_code WHERE asm.academic_year_id = ?`,
			ayID,
		).Scan(&totalSessions, &presentSessions); err != nil {
			writeSyncError(w, http.StatusInternalServerError, fmt.Sprintf("failed to aggregate MAT attendance analytics: %v", err))
			return
		}
		analytics := MATSchoolAnalytics{
			SchoolURN: h.cfg.SchoolURN, AcademicYearID: ayID,
			EnrolledPupilCount: enrolledCount, SenPupilCount: senCount,
			PresentSessionCount: presentSessions, TotalSessionCount: totalSessions,
		}
		st := h.getExternalDepsState()
		st.mu.Lock()
		st.matAnalytics = append(st.matAnalytics, analytics)
		st.mu.Unlock()

		h.logInfo(fmt.Sprintf("Pushed aggregated MAT school analytics for %s (enrolled=%d, sen=%d)", ayID, enrolledCount, senCount))
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "analytics": analytics})
	})

	mux.HandleFunc("GET /api/test/parent-dmz/projections", func(w http.ResponseWriter, r *http.Request) {
		st := h.getExternalDepsState()
		st.mu.RLock()
		defer st.mu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "projections": st.dmzProjections})
	})

	mux.HandleFunc("GET /api/test/mat/analytics", func(w http.ResponseWriter, r *http.Request) {
		st := h.getExternalDepsState()
		st.mu.RLock()
		defer st.mu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "analytics": st.matAnalytics})
	})
}

func (h *Host) handleMockParentDMZRoundTrip(req *http.Request) (*http.Response, error) {
	if err := h.ensureParentDMZAvailable(req.Context()); err != nil {
		return nil, err
	}
	path := ""
	if req.URL != nil {
		path = req.URL.Path
	}
	hdr := make(http.Header)
	hdr.Set("Content-Type", "application/json; charset=utf-8")

	switch path {
	case "/v1/inbox/pull":
		st := h.getExternalDepsState()
		st.mu.Lock()
		queued := make([]ParentDMZSubmission, len(st.dmzInbox))
		copy(queued, st.dmzInbox)
		st.dmzInbox = make([]ParentDMZSubmission, 0)
		st.mu.Unlock()

		respBody, _ := json.Marshal(map[string]any{"ok": true, "submissions": queued})
		return &http.Response{
			StatusCode: http.StatusOK, Status: "200 OK", Header: hdr,
			Body: io.NopCloser(strings.NewReader(string(respBody))), Request: req,
		}, nil

	case "/v1/projections/push":
		var bodyBytes []byte
		if req.Body != nil {
			var err error
			bodyBytes, err = io.ReadAll(io.LimitReader(req.Body, maxEgressBodyBytes))
			if err != nil {
				return nil, fmt.Errorf("failed reading DMZ projection payload: %w", err)
			}
		}
		var rawMap map[string]any
		if err := json.Unmarshal(bodyBytes, &rawMap); err != nil {
			return nil, fmt.Errorf("invalid DMZ projection JSON: %w", err)
		}
		if _, hasUPN := rawMap["upn"]; hasUPN {
			return nil, fmt.Errorf("DMZ_SECURITY_VIOLATION: statutory UPN must be stripped before pushing to Parent DMZ")
		}
		var proj ParentDMZProjection
		if err := json.Unmarshal(bodyBytes, &proj); err != nil {
			return nil, fmt.Errorf("failed decoding ParentDMZProjection: %w", err)
		}
		if strings.TrimSpace(proj.PupilID) == "" {
			return nil, fmt.Errorf("missing mandatory pupil_id in ParentDMZProjection")
		}
		if proj.AuthorizedGuardianIDs == nil {
			proj.AuthorizedGuardianIDs = []string{}
		}
		st := h.getExternalDepsState()
		st.mu.Lock()
		st.dmzProjections[proj.PupilID] = proj
		st.mu.Unlock()

		respBody, _ := json.Marshal(map[string]any{"ok": true, "projection": proj})
		return &http.Response{
			StatusCode: http.StatusOK, Status: "200 OK", Header: hdr,
			Body: io.NopCloser(strings.NewReader(string(respBody))), Request: req,
		}, nil

	default:
		return nil, fmt.Errorf("MOCK_PARENT_DMZ_UNKNOWN_PATH: unsupported path %s", path)
	}
}
