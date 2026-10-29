package host

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"

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
	BehaviourTaxonomy    []string               `json:"behaviour_taxonomy"`
	ApprovedPlugins      []MATApprovedPluginRef `json:"approved_plugins"`
}

type MATSchoolAnalytics struct {
	SchoolURN           string `json:"school_urn"`
	AcademicYearID      string `json:"academic_year_id"`
	EnrolledPupilCount  int    `json:"enrolled_pupil_count"`
	SenPupilCount       int    `json:"sen_pupil_count"`
	PresentSessionCount int    `json:"present_session_count"`
	TotalSessionCount   int    `json:"total_session_count"`
}

type hostExternalDepsState struct {
	mu                      sync.RWMutex
	enableRealCloudKMS      bool
	enableRealParentDMZSync bool
	enableRealMATSync       bool
	dmzProjections          map[string]ParentDMZProjection
	dmzInbox                []ParentDMZSubmission
	matPolicyBundle         *MATPolicyBundle
	matAnalytics            []MATSchoolAnalytics
}

var (
	externalDepsStatesMu sync.Mutex
	externalDepsStates   = make(map[*Host]*hostExternalDepsState)
)

func (h *Host) getExternalDepsState() *hostExternalDepsState {
	externalDepsStatesMu.Lock()
	defer externalDepsStatesMu.Unlock()
	st, ok := externalDepsStates[h]
	if !ok {
		realKMS := strings.EqualFold(strings.TrimSpace(os.Getenv("ENABLE_REAL_CLOUD_KMS")), "true")
		realDMZ := strings.EqualFold(strings.TrimSpace(os.Getenv("ENABLE_REAL_PARENT_DMZ_SYNC")), "true")
		realMAT := strings.EqualFold(strings.TrimSpace(os.Getenv("ENABLE_REAL_MAT_SYNC")), "true")
		st = &hostExternalDepsState{
			enableRealCloudKMS:      realKMS,
			enableRealParentDMZSync: realDMZ,
			enableRealMATSync:       realMAT,
			dmzProjections:          make(map[string]ParentDMZProjection),
			dmzInbox:                make([]ParentDMZSubmission, 0),
			matAnalytics:            make([]MATSchoolAnalytics, 0),
		}
		externalDepsStates[h] = st
	}
	return st
}

func (h *Host) ensureCloudKMSAvailable(ctx context.Context) error {
	st := h.getExternalDepsState()
	st.mu.RLock()
	realKMS := st.enableRealCloudKMS
	st.mu.RUnlock()

	if !realKMS {
		return nil
	}

	keyGenURL, err := h.coreDAL.GetSystemConfigValue(ctx, "kms_cloud_key_generator_url")
	if err != nil {
		h.logError(fmt.Sprintf("REAL_CLOUD_KMS_NOT_CONFIGURED: failed to load kms_cloud_key_generator_url while ENABLE_REAL_CLOUD_KMS=true: %v", err))
		return fmt.Errorf("REAL_CLOUD_KMS_NOT_CONFIGURED: real Cloud KMS / OAuth Cloud Key Generator is not configured when ENABLE_REAL_CLOUD_KMS=true (%w)", err)
	}
	providerEndpoint, err := h.coreDAL.GetSystemConfigValue(ctx, "kms_provider_endpoint")
	if err != nil {
		h.logError(fmt.Sprintf("REAL_CLOUD_KMS_NOT_CONFIGURED: failed to load kms_provider_endpoint while ENABLE_REAL_CLOUD_KMS=true: %v", err))
		return fmt.Errorf("REAL_CLOUD_KMS_NOT_CONFIGURED: real Cloud KMS provider endpoint missing when ENABLE_REAL_CLOUD_KMS=true (%w)", err)
	}

	h.logError(fmt.Sprintf("REAL_CLOUD_KMS_NOT_CONFIGURED: external Cloud KMS provider (%s / %s) is not reachable in this environment", providerEndpoint, keyGenURL))
	return fmt.Errorf("REAL_CLOUD_KMS_NOT_CONFIGURED: external Cloud KMS provider (%s) is not active in this build", providerEndpoint)
}

func (h *Host) ConfigureCloudKMSMode(ctx context.Context, enableRealCloudKMS bool) error {
	st := h.getExternalDepsState()
	st.mu.Lock()
	st.enableRealCloudKMS = enableRealCloudKMS
	st.mu.Unlock()

	if enableRealCloudKMS {
		return h.ensureCloudKMSAvailable(ctx)
	}

	h.logInfo("Configured Cloud KMS mode: ENABLE_REAL_CLOUD_KMS=false (Local Mock Cloud Key Generator active)")
	return nil
}

func (h *Host) ensureParentDMZAvailable(ctx context.Context) error {
	st := h.getExternalDepsState()
	st.mu.RLock()
	realDMZ := st.enableRealParentDMZSync
	st.mu.RUnlock()

	if !realDMZ {
		return nil
	}

	dmzURL, err := h.coreDAL.GetSystemConfigValue(ctx, "parent_dmz_endpoint_url")
	if err != nil {
		h.logError(fmt.Sprintf("REAL_PARENT_DMZ_NOT_CONFIGURED: failed to load parent_dmz_endpoint_url while ENABLE_REAL_PARENT_DMZ_SYNC=true: %v", err))
		return fmt.Errorf("REAL_PARENT_DMZ_NOT_CONFIGURED: real Parent DMZ endpoint is not configured when ENABLE_REAL_PARENT_DMZ_SYNC=true (%w)", err)
	}
	if _, err := h.coreDAL.GetSystemConfigValue(ctx, "parent_dmz_api_key"); err != nil {
		h.logError(fmt.Sprintf("REAL_PARENT_DMZ_NOT_CONFIGURED: failed to load parent_dmz_api_key while ENABLE_REAL_PARENT_DMZ_SYNC=true: %v", err))
		return fmt.Errorf("REAL_PARENT_DMZ_NOT_CONFIGURED: real Parent DMZ API key missing when ENABLE_REAL_PARENT_DMZ_SYNC=true (%w)", err)
	}

	h.logError(fmt.Sprintf("REAL_PARENT_DMZ_NOT_CONFIGURED: external Public Cloud DMZ (%s) is not reachable in this environment", dmzURL))
	return fmt.Errorf("REAL_PARENT_DMZ_NOT_CONFIGURED: external Public Cloud DMZ (%s) is not active in this build", dmzURL)
}

func (h *Host) ConfigureParentDMZMode(ctx context.Context, enableRealParentDMZ bool) error {
	st := h.getExternalDepsState()
	st.mu.Lock()
	st.enableRealParentDMZSync = enableRealParentDMZ
	if !enableRealParentDMZ {
		st.dmzProjections = make(map[string]ParentDMZProjection)
		st.dmzInbox = make([]ParentDMZSubmission, 0)
	}
	st.mu.Unlock()

	if enableRealParentDMZ {
		return h.ensureParentDMZAvailable(ctx)
	}

	h.logInfo("Configured Parent DMZ sync mode: ENABLE_REAL_PARENT_DMZ_SYNC=false (Local In-Memory Mock DMZ active)")
	return nil
}

func (h *Host) ensureMATSyncAvailable(ctx context.Context) error {
	st := h.getExternalDepsState()
	st.mu.RLock()
	realMAT := st.enableRealMATSync
	st.mu.RUnlock()

	if !realMAT {
		return nil
	}

	matURL, err := h.coreDAL.GetSystemConfigValue(ctx, "mat_sync_feed_url")
	if err != nil {
		h.logError(fmt.Sprintf("REAL_MAT_SYNC_NOT_CONFIGURED: failed to load mat_sync_feed_url while ENABLE_REAL_MAT_SYNC=true: %v", err))
		return fmt.Errorf("REAL_MAT_SYNC_NOT_CONFIGURED: real MAT Central Sync feed URL is not configured when ENABLE_REAL_MAT_SYNC=true (%w)", err)
	}
	if _, err := h.coreDAL.GetSystemConfigValue(ctx, "mat_sync_api_token"); err != nil {
		h.logError(fmt.Sprintf("REAL_MAT_SYNC_NOT_CONFIGURED: failed to load mat_sync_api_token while ENABLE_REAL_MAT_SYNC=true: %v", err))
		return fmt.Errorf("REAL_MAT_SYNC_NOT_CONFIGURED: real MAT Central Sync API token missing when ENABLE_REAL_MAT_SYNC=true (%w)", err)
	}

	h.logError(fmt.Sprintf("REAL_MAT_SYNC_NOT_CONFIGURED: external MAT Central Sync endpoint (%s) is not reachable in this environment", matURL))
	return fmt.Errorf("REAL_MAT_SYNC_NOT_CONFIGURED: external MAT Central Sync endpoint (%s) is not active in this build", matURL)
}

func (h *Host) ConfigureMATSyncMode(ctx context.Context, enableRealMATSync bool) error {
	st := h.getExternalDepsState()
	st.mu.Lock()
	st.enableRealMATSync = enableRealMATSync
	if !enableRealMATSync {
		st.matPolicyBundle = nil
		st.matAnalytics = make([]MATSchoolAnalytics, 0)
	}
	st.mu.Unlock()

	if !enableRealMATSync {
		for _, k := range []string{
			"mat_id",
			"mat_grading_schema_version",
			"mat_behaviour_taxonomy_json",
			"mat_approved_plugins_json",
		} {
			_ = h.coreDAL.DeleteSystemConfigKey(ctx, k)
		}
	}

	if enableRealMATSync {
		return h.ensureMATSyncAvailable(ctx)
	}

	h.logInfo("Configured MAT Central sync mode: ENABLE_REAL_MAT_SYNC=false (Local In-Memory Mock MAT Feed active)")
	return nil
}

func (h *Host) RegisterExternalSyncAndTestRoutes(mux *http.ServeMux) {
	// 1. Test control endpoint to toggle external dependency feature flags at runtime
	mux.HandleFunc("POST /api/test/external-deps-mode", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			EnableRealCloudKMS      *bool `json:"enable_real_cloud_kms"`
			EnableRealParentDMZSync *bool `json:"enable_real_parent_dmz_sync"`
			EnableRealMATSync       *bool `json:"enable_real_mat_sync"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": fmt.Sprintf("invalid JSON body: %v", err),
			})
			return
		}

		if body.EnableRealCloudKMS != nil {
			if err := h.ConfigureCloudKMSMode(r.Context(), *body.EnableRealCloudKMS); err != nil {
				writeJSON(w, http.StatusServiceUnavailable, map[string]any{
					"ok":         false,
					"error_code": err.Error(),
				})
				return
			}
		}
		if body.EnableRealParentDMZSync != nil {
			if err := h.ConfigureParentDMZMode(r.Context(), *body.EnableRealParentDMZSync); err != nil {
				writeJSON(w, http.StatusServiceUnavailable, map[string]any{
					"ok":         false,
					"error_code": err.Error(),
				})
				return
			}
		}
		if body.EnableRealMATSync != nil {
			if err := h.ConfigureMATSyncMode(r.Context(), *body.EnableRealMATSync); err != nil {
				writeJSON(w, http.StatusServiceUnavailable, map[string]any{
					"ok":         false,
					"error_code": err.Error(),
				})
				return
			}
		}

		st := h.getExternalDepsState()
		st.mu.RLock()
		resp := map[string]any{
			"ok":                          true,
			"enable_real_cloud_kms":       st.enableRealCloudKMS,
			"enable_real_parent_dmz_sync": st.enableRealParentDMZSync,
			"enable_real_mat_sync":        st.enableRealMATSync,
		}
		st.mu.RUnlock()
		writeJSON(w, http.StatusOK, resp)
	})

	// 2. Outbound Parent DMZ Sync: Push sanitized read-projection (Section 1)
	mux.HandleFunc("POST /api/sync/parent-dmz/push-projection", func(w http.ResponseWriter, r *http.Request) {
		if err := h.ensureParentDMZAvailable(r.Context()); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{
				"ok":         false,
				"error_code": err.Error(),
			})
			return
		}

		var body struct {
			PupilID        string `json:"pupil_id"`
			ReferenceDate  string `json:"reference_date"`
			AcademicYearID string `json:"academic_year_id"`
			StaffID        string `json:"staff_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": fmt.Sprintf("invalid JSON body: %v", err),
			})
			return
		}
		pupilID := strings.TrimSpace(body.PupilID)
		if pupilID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": "missing mandatory field: pupil_id",
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

		scope, err := h.coreDAL.ResolveScopeDefaults(r.Context(), dal.DALQueryScope{
			Mode:           dal.ScopeAll,
			StaffID:        caller.StaffID,
			AcademicYearID: strings.TrimSpace(body.AcademicYearID),
			ReferenceDate:  strings.TrimSpace(body.ReferenceDate),
		})
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": err.Error(),
			})
			return
		}

		pupil, err := h.coreDAL.GetPupilByIDScoped(r.Context(), scope, pupilID)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": err.Error(),
			})
			return
		}

		// List all non-court-order-restricted guardians to count confidential address redactions
		// and collect authorized Parental Responsibility (PR) guardian IDs.
		allUnrestrictedGuardians, err := h.coreDAL.ListGuardiansForPupilScoped(
			r.Context(),
			scope,
			pupilID,
			dal.GuardianFilterOptions{RequireParentalResponsibility: false},
		)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": err.Error(),
			})
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
			`SELECT
				COUNT(*),
				COALESCE(SUM(CASE WHEN ac.statistical_category IN ('PRESENT', 'APPROVED_EDUCATIONAL_ACTIVITY') OR asm.attendance_code IN ('/', '\', 'L', 'B', 'P', 'V', 'W') THEN 1 ELSE 0 END), 0)
			 FROM attendance_session_marks asm
			 LEFT JOIN attendance_codes ac ON ac.code = asm.attendance_code
			 WHERE asm.pupil_id = ? AND asm.academic_year_id = ?`,
			pupilID, scope.AcademicYearID,
		).Scan(&totalSessions, &presentSessions); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"ok":         false,
				"error_code": fmt.Sprintf("failed to aggregate attendance sessions for DMZ projection: %v", err),
			})
			return
		}

		projection := ParentDMZProjection{
			PupilID:                          pupil.ID,
			FirstName:                        pupil.FirstName,
			LastName:                         pupil.LastName,
			AcademicYearID:                   pupil.AcademicYearID,
			ReferenceDate:                    scope.ReferenceDate,
			PresentSessions:                  presentSessions,
			TotalSessions:                    totalSessions,
			AuthorizedGuardianIDs:            authorizedGuardianIDs,
			RedactedConfidentialAddressCount: redactedConfidentialCount,
		}

		st := h.getExternalDepsState()
		st.mu.Lock()
		st.dmzProjections[pupil.ID] = projection
		st.mu.Unlock()

		h.logInfo(fmt.Sprintf("Pushed sanitized parent DMZ read-projection for pupil (authorized_guardians=%d, redacted_addresses=%d)", len(authorizedGuardianIDs), redactedConfidentialCount))
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":         true,
			"projection": projection,
		})
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
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": fmt.Sprintf("invalid JSON body: %v", err),
			})
			return
		}
		if strings.TrimSpace(body.PupilID) == "" ||
			strings.TrimSpace(body.SubmittedByGuardianPersonID) == "" ||
			strings.TrimSpace(body.SubmissionType) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": "missing mandatory fields: pupil_id, submitted_by_guardian_person_id, submission_type",
			})
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

		writeJSON(w, http.StatusCreated, map[string]any{
			"ok":         true,
			"submission": sub,
		})
	})

	// 4. Outbound Parent DMZ Sync: Pull queued submissions from DMZ Write-Inbox (Section 1)
	mux.HandleFunc("POST /api/sync/parent-dmz/pull-inbox", func(w http.ResponseWriter, r *http.Request) {
		if err := h.ensureParentDMZAvailable(r.Context()); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{
				"ok":         false,
				"error_code": err.Error(),
			})
			return
		}

		var body struct {
			StaffID string `json:"staff_id"`
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

		st := h.getExternalDepsState()
		st.mu.Lock()
		queued := make([]ParentDMZSubmission, len(st.dmzInbox))
		copy(queued, st.dmzInbox)
		st.dmzInbox = make([]ParentDMZSubmission, 0)
		st.mu.Unlock()

		processed := make([]ParentDMZSubmission, 0, len(queued))
		acceptedCount := 0
		rejectedCount := 0

		for _, sub := range queued {
			scope, sErr := h.coreDAL.ResolveScopeDefaults(r.Context(), dal.DALQueryScope{
				Mode:          dal.ScopeAll,
				StaffID:       caller.StaffID,
				ReferenceDate: sub.Date,
			})
			if sErr != nil {
				sub.Status = "REJECTED_SAFEGUARDING_BLOCK"
				rejectedCount++
				processed = append(processed, sub)
				continue
			}

			prGuardians, gErr := h.coreDAL.ListGuardiansForPupilScoped(
				r.Context(),
				scope,
				sub.PupilID,
				dal.GuardianFilterOptions{RequireParentalResponsibility: true},
			)
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
			} else {
				sub.Status = "REJECTED_SAFEGUARDING_BLOCK"
				rejectedCount++
			}
			processed = append(processed, sub)
		}

		h.logInfo(fmt.Sprintf("Pulled %d parent DMZ write-inbox submissions (accepted=%d, rejected=%d)", len(processed), acceptedCount, rejectedCount))
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":             true,
			"accepted_count": acceptedCount,
			"rejected_count": rejectedCount,
			"submissions":    processed,
		})
	})

	// 5. Test helper: Seed Mock MAT Central Sync Feed
	mux.HandleFunc("POST /api/test/mat/seed-feed", func(w http.ResponseWriter, r *http.Request) {
		var bundle MATPolicyBundle
		if err := json.NewDecoder(r.Body).Decode(&bundle); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": fmt.Sprintf("invalid JSON body: %v", err),
			})
			return
		}
		if strings.TrimSpace(bundle.MATID) == "" || strings.TrimSpace(bundle.GradingSchemaVersion) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": "missing mandatory fields: mat_id, grading_schema_version",
			})
			return
		}

		st := h.getExternalDepsState()
		st.mu.Lock()
		st.matPolicyBundle = &bundle
		st.mu.Unlock()

		writeJSON(w, http.StatusCreated, map[string]any{
			"ok":            true,
			"policy_bundle": bundle,
		})
	})

	// 6. Outbound MAT Central Sync: Pull MAT policies & approved plugin catalog (Sections 1 & 6A)
	mux.HandleFunc("POST /api/sync/mat/pull-policies", func(w http.ResponseWriter, r *http.Request) {
		if err := h.ensureMATSyncAvailable(r.Context()); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{
				"ok":         false,
				"error_code": err.Error(),
			})
			return
		}

		var body struct {
			StaffID string `json:"staff_id"`
		}
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&body)
		}

		if _, err := h.ResolveRequestCaller(r, body.StaffID); err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]any{
				"ok":         false,
				"error_code": err.Error(),
			})
			return
		}

		st := h.getExternalDepsState()
		st.mu.RLock()
		bundle := st.matPolicyBundle
		st.mu.RUnlock()

		if bundle == nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": "MAT_FEED_EMPTY: no MAT policy bundle available in mock feed",
			})
			return
		}

		taxonomyJSON, _ := json.Marshal(bundle.BehaviourTaxonomy)
		pluginsJSON, _ := json.Marshal(bundle.ApprovedPlugins)

		configEntries := []struct {
			key string
			val string
		}{
			{"mat_id", bundle.MATID},
			{"mat_grading_schema_version", bundle.GradingSchemaVersion},
			{"mat_behaviour_taxonomy_json", string(taxonomyJSON)},
			{"mat_approved_plugins_json", string(pluginsJSON)},
		}
		for _, entry := range configEntries {
			if _, err := h.coreDAL.WriteDB().ExecContext(
				r.Context(),
				`INSERT INTO __system_config (key, value) VALUES (?, ?)
				 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
				entry.key, entry.val,
			); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{
					"ok":         false,
					"error_code": fmt.Sprintf("failed to persist MAT policy key %s: %v", entry.key, err),
				})
				return
			}
		}

		h.logInfo(fmt.Sprintf("Pulled MAT policy bundle from %s (schema=%s)", bundle.MATID, bundle.GradingSchemaVersion))
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":            true,
			"policy_bundle": bundle,
		})
	})

	// 7. Outbound MAT Central Sync: Push aggregated school-level analytics (Section 1)
	mux.HandleFunc("POST /api/sync/mat/push-analytics", func(w http.ResponseWriter, r *http.Request) {
		if err := h.ensureMATSyncAvailable(r.Context()); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{
				"ok":         false,
				"error_code": err.Error(),
			})
			return
		}

		var body struct {
			AcademicYearID string `json:"academic_year_id"`
			StaffID        string `json:"staff_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": fmt.Sprintf("invalid JSON body: %v", err),
			})
			return
		}
		ayID := strings.TrimSpace(body.AcademicYearID)
		if ayID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": "missing mandatory field: academic_year_id",
			})
			return
		}

		if _, err := h.ResolveRequestCaller(r, body.StaffID); err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]any{
				"ok":         false,
				"error_code": err.Error(),
			})
			return
		}

		var enrolledCount int
		if err := h.coreDAL.ReadDB().QueryRowContext(
			r.Context(),
			`SELECT COUNT(DISTINCT pupil_id)
			 FROM pupil_year_enrolments
			 WHERE academic_year_id = ?
			   AND (effective_to IS NULL OR effective_to >= date('now'))`,
			ayID,
		).Scan(&enrolledCount); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"ok":         false,
				"error_code": fmt.Sprintf("failed to count enrolled pupils: %v", err),
			})
			return
		}

		var senCount int
		if err := h.coreDAL.ReadDB().QueryRowContext(
			r.Context(),
			`SELECT COUNT(DISTINCT psr.pupil_id)
			 FROM pupil_sen_records psr
			 JOIN pupil_year_enrolments enr ON enr.pupil_id = psr.pupil_id
			 WHERE enr.academic_year_id = ?
			   AND psr.provision_code IN ('K', 'E')
			   AND (psr.effective_to IS NULL OR psr.effective_to >= date('now'))`,
			ayID,
		).Scan(&senCount); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"ok":         false,
				"error_code": fmt.Sprintf("failed to count SEN pupils: %v", err),
			})
			return
		}

		var totalSessions, presentSessions int
		if err := h.coreDAL.ReadDB().QueryRowContext(
			r.Context(),
			`SELECT
				COUNT(*),
				COALESCE(SUM(CASE WHEN ac.statistical_category IN ('PRESENT', 'APPROVED_EDUCATIONAL_ACTIVITY') OR asm.attendance_code IN ('/', '\', 'L', 'B', 'P', 'V', 'W') THEN 1 ELSE 0 END), 0)
			 FROM attendance_session_marks asm
			 LEFT JOIN attendance_codes ac ON ac.code = asm.attendance_code
			 WHERE asm.academic_year_id = ?`,
			ayID,
		).Scan(&totalSessions, &presentSessions); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"ok":         false,
				"error_code": fmt.Sprintf("failed to aggregate MAT attendance analytics: %v", err),
			})
			return
		}

		analytics := MATSchoolAnalytics{
			SchoolURN:           h.cfg.SchoolURN,
			AcademicYearID:      ayID,
			EnrolledPupilCount:  enrolledCount,
			SenPupilCount:       senCount,
			PresentSessionCount: presentSessions,
			TotalSessionCount:   totalSessions,
		}

		st := h.getExternalDepsState()
		st.mu.Lock()
		st.matAnalytics = append(st.matAnalytics, analytics)
		st.mu.Unlock()

		h.logInfo(fmt.Sprintf("Pushed aggregated MAT school analytics for %s (enrolled=%d, sen=%d)", ayID, enrolledCount, senCount))
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":        true,
			"analytics": analytics,
		})
	})

	mux.HandleFunc("GET /api/test/parent-dmz/projections", func(w http.ResponseWriter, r *http.Request) {
		st := h.getExternalDepsState()
		st.mu.RLock()
		defer st.mu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":          true,
			"projections": st.dmzProjections,
		})
	})

	mux.HandleFunc("GET /api/test/mat/analytics", func(w http.ResponseWriter, r *http.Request) {
		st := h.getExternalDepsState()
		st.mu.RLock()
		defer st.mu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":        true,
			"analytics": st.matAnalytics,
		})
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

		respBody, _ := json.Marshal(map[string]any{
			"ok":          true,
			"submissions": queued,
		})
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     hdr,
			Body:       io.NopCloser(strings.NewReader(string(respBody))),
			Request:    req,
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

		respBody, _ := json.Marshal(map[string]any{
			"ok":         true,
			"projection": proj,
		})
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     hdr,
			Body:       io.NopCloser(strings.NewReader(string(respBody))),
			Request:    req,
		}, nil

	default:
		return nil, fmt.Errorf("MOCK_PARENT_DMZ_UNKNOWN_PATH: unsupported path %s", path)
	}
}

