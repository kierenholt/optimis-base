package host

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"optimis/internal/dal"
)

func (h *Host) RegisterGDPRRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/admin/gdpr/sar/export", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			PupilID string `json:"pupil_id"`
			StaffID string `json:"staff_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": fmt.Sprintf("invalid JSON body: %v", err),
			})
			return
		}
		caller, err := h.ResolveRequestCaller(r, "")
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]any{
				"ok":         false,
				"error_code": err.Error(),
			})
			return
		}

		bundle, err := h.ExportPupilSAR(r.Context(), caller, body.PupilID)
		if err != nil {
			status := http.StatusBadRequest
			if strings.Contains(err.Error(), "UNAUTHORIZED_ADMIN_GDPR_SCOPE") {
				status = http.StatusForbidden
			}
			writeJSON(w, status, map[string]any{
				"ok":         false,
				"error_code": err.Error(),
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":     true,
			"bundle": bundle,
		})
	})

	mux.HandleFunc("POST /api/admin/gdpr/sar/finalize", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			PupilID  string           `json:"pupil_id"`
			StaffID  string           `json:"staff_id"`
			Sections []dal.SARSection `json:"sections"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": fmt.Sprintf("invalid JSON body: %v", err),
			})
			return
		}
		caller, err := h.ResolveRequestCaller(r, "")
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]any{
				"ok":         false,
				"error_code": err.Error(),
			})
			return
		}

		pkg, err := h.FinalizePupilSAR(r.Context(), caller, SARFinalizeInput{
			PupilID:  body.PupilID,
			Sections: body.Sections,
		})
		if err != nil {
			status := http.StatusBadRequest
			if strings.Contains(err.Error(), "UNAUTHORIZED_ADMIN_GDPR_SCOPE") {
				status = http.StatusForbidden
			}
			writeJSON(w, status, map[string]any{
				"ok":         false,
				"error_code": err.Error(),
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":      true,
			"package": pkg,
		})
	})

	mux.HandleFunc("POST /api/admin/gdpr/purge", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			PupilID string `json:"pupil_id"`
			StaffID string `json:"staff_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": fmt.Sprintf("invalid JSON body: %v", err),
			})
			return
		}
		caller, err := h.ResolveRequestCaller(r, "")
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]any{
				"ok":         false,
				"error_code": err.Error(),
			})
			return
		}

		res, err := h.PurgePupilData(r.Context(), caller, body.PupilID)
		if err != nil {
			status := http.StatusBadRequest
			if strings.Contains(err.Error(), "UNAUTHORIZED_ADMIN_GDPR_SCOPE") {
				status = http.StatusForbidden
			} else if strings.Contains(err.Error(), "KCSIE_RETENTION_LOCK") {
				status = http.StatusConflict
			}
			writeJSON(w, status, map[string]any{
				"ok":         false,
				"error_code": err.Error(),
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":     true,
			"result": res,
		})
	})

	mux.HandleFunc("POST /api/admin/plugins/{plugin_id}/state", func(w http.ResponseWriter, r *http.Request) {
		pluginID := strings.TrimSpace(r.PathValue("plugin_id"))
		if pluginID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": "missing mandatory path parameter: plugin_id",
			})
			return
		}
		var body struct {
			State   string `json:"state"`
			StaffID string `json:"staff_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": fmt.Sprintf("invalid JSON body: %v", err),
			})
			return
		}
		caller, err := h.ResolveRequestCaller(r, "")
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]any{
				"ok":         false,
				"error_code": err.Error(),
			})
			return
		}
		if !isCallerAdminOrDPO(caller, true) {
			writeJSON(w, http.StatusForbidden, map[string]any{
				"ok":         false,
				"error_code": "UNAUTHORIZED_ADMIN_GDPR_SCOPE: changing plugin lifecycle state requires System Admin authority",
			})
			return
		}
		if err := h.SetPluginState(pluginID, PluginState(strings.TrimSpace(body.State))); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": err.Error(),
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":        true,
			"plugin_id": pluginID,
			"state":     strings.TrimSpace(body.State),
		})
	})

	mux.HandleFunc("POST /api/test/seed-gdpr-fixtures", func(w http.ResponseWriter, r *http.Request) {
		var in dal.GDPRTestFixtureInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error": fmt.Sprintf("invalid JSON body: %v", err),
			})
			return
		}
		in.HostKMSKeyHandle = h.cfg.HostKMSKeyHandle
		seeded, err := h.coreDAL.SeedGDPRTestFixtures(r.Context(), in)
		if err != nil {
			h.logError(fmt.Sprintf("Failed to seed GDPR fixtures: %v", err))
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error": err.Error(),
			})
			return
		}
		h.ResetDevPersonaSessions()
		h.logInfo(fmt.Sprintf("Seeded GDPR test fixtures %s", seeded.FixtureID))
		writeJSON(w, http.StatusCreated, seeded)
	})

	mux.HandleFunc("DELETE /api/test/seed-gdpr-fixtures/{fixture_id}", func(w http.ResponseWriter, r *http.Request) {
		fixtureID := strings.TrimSpace(r.PathValue("fixture_id"))
		if fixtureID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error": "missing mandatory path parameter: fixture_id",
			})
			return
		}
		if err := h.coreDAL.DeleteGDPRTestFixtures(r.Context(), fixtureID); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error": err.Error(),
			})
			return
		}
		h.ResetDevPersonaSessions()
		h.logInfo(fmt.Sprintf("Deleted GDPR test fixtures %s", fixtureID))
		writeJSON(w, http.StatusOK, map[string]any{
			"deleted": true,
		})
	})
}
