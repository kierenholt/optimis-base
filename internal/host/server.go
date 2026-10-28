package host

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"optimis/internal/dal"
)

//go:embed all:ui_dist
var embeddedUIFS embed.FS

func (h *Host) NewHTTPHandler() (http.Handler, error) {
	uiSubFS, err := fs.Sub(embeddedUIFS, "ui_dist")
	if err != nil {
		return nil, fmt.Errorf("failed to mount embedded UI filesystem: %w", err)
	}
	var fileServer http.Handler
	if stat, statErr := os.Stat("internal/host/ui_dist"); statErr == nil && stat.IsDir() {
		fileServer = http.FileServer(http.Dir("internal/host/ui_dist"))
	} else {
		fileServer = http.FileServer(http.FS(uiSubFS))
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/test/logs/cursor", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"cursor": h.LogCursor(),
		})
	})

	mux.HandleFunc("GET /api/test/logs", func(w http.ResponseWriter, r *http.Request) {
		since := strings.TrimSpace(r.URL.Query().Get("since"))
		if since == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error": "missing mandatory query parameter: since",
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"entries": h.LogsSince(since),
		})
	})

	mux.HandleFunc("POST /api/test/seed-pupil", func(w http.ResponseWriter, r *http.Request) {
		var in dal.SeedPupilInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error": fmt.Sprintf("invalid JSON body: %v", err),
			})
			return
		}
		pupil, err := h.coreDAL.SeedPupil(r.Context(), in)
		if err != nil {
			h.logError(fmt.Sprintf("Failed to seed pupil: %v", err))
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error": err.Error(),
			})
			return
		}
		h.logInfo(fmt.Sprintf("Seeded pupil record %s", pupil.ID))
		writeJSON(w, http.StatusCreated, map[string]any{
			"pupil_id": pupil.ID,
			"pupil":    pupil,
		})
	})

	mux.HandleFunc("DELETE /api/test/seed-pupil/{pupil_id}", func(w http.ResponseWriter, r *http.Request) {
		pupilID := strings.TrimSpace(r.PathValue("pupil_id"))
		if pupilID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error": "missing mandatory path parameter: pupil_id",
			})
			return
		}
		if err := h.coreDAL.DeletePupil(r.Context(), pupilID); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error": err.Error(),
			})
			return
		}
		h.logInfo(fmt.Sprintf("Deleted seeded pupil record %s", pupilID))
		writeJSON(w, http.StatusOK, map[string]any{
			"deleted": true,
		})
	})

	mux.HandleFunc("POST /api/test/seed-milestone1-fixtures", func(w http.ResponseWriter, r *http.Request) {
		var in dal.Milestone1SeedInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error": fmt.Sprintf("invalid JSON body: %v", err),
			})
			return
		}
		seeded, err := h.coreDAL.SeedMilestone1Fixtures(r.Context(), in)
		if err != nil {
			h.logError(fmt.Sprintf("Failed to seed Milestone 1 fixtures: %v", err))
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error": err.Error(),
			})
			return
		}
		h.logInfo(fmt.Sprintf("Seeded Milestone 1 fixtures %s", seeded.FixtureID))
		writeJSON(w, http.StatusCreated, seeded)
	})

	mux.HandleFunc("DELETE /api/test/seed-milestone1-fixtures/{fixture_id}", func(w http.ResponseWriter, r *http.Request) {
		fixtureID := strings.TrimSpace(r.PathValue("fixture_id"))
		if fixtureID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error": "missing mandatory path parameter: fixture_id",
			})
			return
		}
		if err := h.coreDAL.DeleteMilestone1Fixtures(r.Context(), fixtureID); err != nil {
			h.logError(fmt.Sprintf("Failed to delete Milestone 1 fixtures %s: %v", fixtureID, err))
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error": err.Error(),
			})
			return
		}
		h.logInfo(fmt.Sprintf("Deleted Milestone 1 fixtures %s", fixtureID))
		writeJSON(w, http.StatusOK, map[string]any{
			"deleted": true,
		})
	})

	h.RegisterAuthAndPersonaRoutes(mux)
	h.RegisterSafeguardingRoutes(mux)
	h.RegisterGDPRRoutes(mux)
	h.RegisterEventBusRoutes(mux)
	h.RegisterEgressTestRoutes(mux)
	h.RegisterUISDKAndConfigRoutes(mux)

	mux.HandleFunc("POST /api/plugins/{plugin_id}/rpc/{method}", func(w http.ResponseWriter, r *http.Request) {
		pluginID := strings.TrimSpace(r.PathValue("plugin_id"))
		method := strings.TrimSpace(r.PathValue("method"))
		if pluginID == "" || method == "" {
			writeJSON(w, http.StatusBadRequest, RPCResponse{
				OK:        false,
				ErrorCode: "MISSING_PLUGIN_OR_METHOD",
			})
			return
		}

		var body struct {
			Args map[string]any `json:"args"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, RPCResponse{
				OK:        false,
				ErrorCode: "INVALID_JSON_PAYLOAD",
			})
			return
		}
		if body.Args == nil {
			body.Args = map[string]any{}
		}

		explicitStaffID := ""
		if s, ok := body.Args["staff_id"].(string); ok && strings.TrimSpace(s) != "" {
			explicitStaffID = strings.TrimSpace(s)
		} else if s, ok := body.Args["__caller_staff_id"].(string); ok && strings.TrimSpace(s) != "" {
			explicitStaffID = strings.TrimSpace(s)
		} else if qStaff := strings.TrimSpace(r.URL.Query().Get("staff_id")); qStaff != "" {
			explicitStaffID = qStaff
		}

		caller, callerErr := h.ResolveRequestCaller(r, explicitStaffID)
		if callerErr != nil {
			writeJSON(w, http.StatusBadRequest, RPCResponse{
				OK:        false,
				ErrorCode: callerErr.Error(),
			})
			return
		}

		res, err := h.ExecuteRPC(r.Context(), pluginID, method, caller, body.Args)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, RPCResponse{
				OK:        false,
				ErrorCode: err.Error(),
			})
			return
		}
		writeJSON(w, http.StatusOK, res)
	})

	mux.HandleFunc("GET /plugins/{plugin_id}/ui/{asset_path...}", func(w http.ResponseWriter, r *http.Request) {
		pluginID := strings.TrimSpace(r.PathValue("plugin_id"))
		assetPath := strings.TrimSpace(r.PathValue("asset_path"))
		if pluginID == "" || assetPath == "" {
			http.Error(w, "missing plugin_id or asset_path", http.StatusBadRequest)
			return
		}
		plugin, ok := h.GetLoadedPlugin(pluginID)
		if !ok {
			http.Error(w, "plugin not found", http.StatusNotFound)
			return
		}
		if plugin.State == PluginStateUninstalledDormant {
			http.Error(w, "PLUGIN_DORMANT_QUARANTINE: plugin UI disabled in dormant quarantine", http.StatusForbidden)
			return
		}

		cleanRel := filepath.Clean(assetPath)
		if strings.HasPrefix(cleanRel, "..") {
			http.Error(w, "invalid asset path", http.StatusBadRequest)
			return
		}
		fullPath := filepath.Join(plugin.DirPath, "ui", "dist", cleanRel)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			if cleanRel == "mis-bridge.js" || cleanRel == "mis-theme.css" {
				sdkBytes, sdkCT, sdkErr := readSDKUIAsset(cleanRel)
				if sdkErr == nil {
					w.Header().Set("Content-Security-Policy", pluginSandboxCSP)
					w.Header().Set("Access-Control-Allow-Origin", "*")
					w.Header().Set("Content-Type", sdkCT)
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write(sdkBytes)
					return
				}
			}
			http.Error(w, "asset not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Security-Policy", pluginSandboxCSP)
		w.Header().Set("Access-Control-Allow-Origin", "*")
		if strings.HasSuffix(cleanRel, ".html") {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		} else if strings.HasSuffix(cleanRel, ".js") {
			w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		} else if strings.HasSuffix(cleanRel, ".css") {
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(content)
	})

	mux.Handle("/", fileServer)

	return mux, nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
