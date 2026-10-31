package host

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
)

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
		for _, k := range []string{"mat_id", "mat_grading_schema_version", "mat_grading_schema_json", "mat_behaviour_taxonomy_json", "mat_approved_plugins_json"} {
			_ = h.coreDAL.DeleteSystemConfigKey(ctx, k)
		}
	}
	if enableRealMATSync {
		return h.ensureMATSyncAvailable(ctx)
	}
	h.logInfo("Configured MAT Central sync mode: ENABLE_REAL_MAT_SYNC=false (Local In-Memory Mock MAT Feed active)")
	return nil
}

func writeSyncError(w http.ResponseWriter, status int, errCode string) {
	writeJSON(w, status, map[string]any{"ok": false, "error_code": errCode})
}

func (h *Host) registerExternalDepsModeRoute(mux *http.ServeMux) {
	// 1. Test control endpoint to toggle external dependency feature flags at runtime
	mux.HandleFunc("POST /api/test/external-deps-mode", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			EnableRealCloudKMS      *bool `json:"enable_real_cloud_kms"`
			EnableRealParentDMZSync *bool `json:"enable_real_parent_dmz_sync"`
			EnableRealMATSync       *bool `json:"enable_real_mat_sync"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeSyncError(w, http.StatusBadRequest, fmt.Sprintf("invalid JSON body: %v", err))
			return
		}
		if body.EnableRealCloudKMS != nil {
			if err := h.ConfigureCloudKMSMode(r.Context(), *body.EnableRealCloudKMS); err != nil {
				writeSyncError(w, http.StatusServiceUnavailable, err.Error())
				return
			}
		}
		if body.EnableRealParentDMZSync != nil {
			if err := h.ConfigureParentDMZMode(r.Context(), *body.EnableRealParentDMZSync); err != nil {
				writeSyncError(w, http.StatusServiceUnavailable, err.Error())
				return
			}
		}
		if body.EnableRealMATSync != nil {
			if err := h.ConfigureMATSyncMode(r.Context(), *body.EnableRealMATSync); err != nil {
				writeSyncError(w, http.StatusServiceUnavailable, err.Error())
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
}
