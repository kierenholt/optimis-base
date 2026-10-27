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

	"github.com/google/uuid"

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

		caller := CallerContext{
			Type:        "USER",
			UserID:      "usr_teacher_1",
			StaffID:     "stf_001",
			ActiveRoles: []string{"teacher"},
			Scopes:      []string{"dal.pupils.read:all"},
			RequestID:   "req_" + uuid.NewString(),
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

		cleanRel := filepath.Clean(assetPath)
		if strings.HasPrefix(cleanRel, "..") {
			http.Error(w, "invalid asset path", http.StatusBadRequest)
			return
		}
		fullPath := filepath.Join(plugin.DirPath, "ui", "dist", cleanRel)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			http.Error(w, "asset not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; connect-src 'none';")
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
