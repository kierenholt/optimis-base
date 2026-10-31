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

	h.registerHostLogAndFixtureRoutes(mux)
	h.RegisterAuthAndPersonaRoutes(mux)
	h.RegisterSafeguardingRoutes(mux)
	h.RegisterGDPRRoutes(mux)
	h.RegisterDiagnosticRoutes(mux)
	h.RegisterEventBusRoutes(mux)
	h.RegisterEgressTestRoutes(mux)
	h.RegisterCensusTestRoutes(mux)
	h.RegisterUISDKAndConfigRoutes(mux)
	h.RegisterExternalSyncAndTestRoutes(mux)
	h.RegisterPluginLifecycleRoutes(mux)

	serveHostIndexHTMLWithStatus := func(w http.ResponseWriter, r *http.Request, statusCode int) {
		if stat, statErr := os.Stat("internal/host/ui_dist/index.html"); statErr == nil && !stat.IsDir() {
			content, readErr := os.ReadFile("internal/host/ui_dist/index.html")
			if readErr == nil {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(statusCode)
				_, _ = w.Write(content)
				return
			}
		}
		content, readErr := fs.ReadFile(uiSubFS, "index.html")
		if readErr != nil {
			http.Error(w, "host index.html not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(statusCode)
		_, _ = w.Write(content)
	}

	respondPluginViewRouteError := func(w http.ResponseWriter, r *http.Request, status int, errCode string) {
		if strings.Contains(strings.ToLower(r.Header.Get("Accept")), "text/html") {
			w.Header().Set("X-MIS-Route-Error", errCode)
			serveHostIndexHTMLWithStatus(w, r, status)
			return
		}
		writeJSON(w, status, map[string]any{
			"ok":         false,
			"error_code": errCode,
		})
	}

	servePluginUIAsset := func(w http.ResponseWriter, r *http.Request, plugID string, assetPath string) {
		if plugID == "" || assetPath == "" {
			http.Error(w, "missing plug_id or asset_path", http.StatusBadRequest)
			return
		}
		plugin, ok := h.GetLoadedPluginByPlugID(plugID)
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
	}

	handlePluginViewRoute := func(w http.ResponseWriter, r *http.Request, plugID string, subpath string) {
		if plugID == "" {
			respondPluginViewRouteError(
				w,
				r,
				http.StatusBadRequest,
				"MISSING_PLUG_ID: missing mandatory plug_id path parameter",
			)
			return
		}
		plugin, ok := h.GetLoadedPluginByPlugID(plugID)
		if !ok {
			errCode := fmt.Sprintf("PLUGIN_NOT_FOUND: unknown plug_id %s", plugID)
			h.logError(fmt.Sprintf("Blocked unknown plugin route /p/%s/%s: %s", plugID, subpath, errCode))
			respondPluginViewRouteError(w, r, http.StatusNotFound, errCode)
			return
		}
		if plugin.State == PluginStateUninstalledDormant {
			errCode := "PLUGIN_DORMANT_QUARANTINE: plugin UI disabled in dormant quarantine"
			h.logError(fmt.Sprintf("Blocked dormant plugin route /p/%s/%s: %s", plugID, subpath, errCode))
			respondPluginViewRouteError(w, r, http.StatusForbidden, errCode)
			return
		}

		_, status, errCode := MatchPluginViewRoute(plugin, subpath, r.URL.Query())
		if status != http.StatusOK {
			h.logError(fmt.Sprintf("Blocked undeclared plugin route /p/%s/%s: %s", plugID, subpath, errCode))
			respondPluginViewRouteError(w, r, status, errCode)
			return
		}

		serveHostIndexHTMLWithStatus(w, r, http.StatusOK)
	}

	mux.HandleFunc("POST /api/p/{plug_id}/rpc/{method}", h.handlePluginRPCRequest)

	mux.HandleFunc("GET /_p_assets/{plug_id}/{asset_path...}", func(w http.ResponseWriter, r *http.Request) {
		plugID := strings.TrimSpace(r.PathValue("plug_id"))
		assetPath := strings.TrimSpace(r.PathValue("asset_path"))
		servePluginUIAsset(w, r, plugID, assetPath)
	})

	mux.HandleFunc("GET /p/{plug_id}/ui/{asset_path...}", func(w http.ResponseWriter, r *http.Request) {
		plugID := strings.TrimSpace(r.PathValue("plug_id"))
		assetPath := strings.TrimSpace(r.PathValue("asset_path"))
		servePluginUIAsset(w, r, plugID, assetPath)
	})

	mux.HandleFunc("GET /p/{plug_id}", func(w http.ResponseWriter, r *http.Request) {
		plugID := strings.TrimSpace(r.PathValue("plug_id"))
		handlePluginViewRoute(w, r, plugID, "/")
	})

	mux.HandleFunc("GET /p/{plug_id}/{subpath...}", func(w http.ResponseWriter, r *http.Request) {
		plugID := strings.TrimSpace(r.PathValue("plug_id"))
		subpath := strings.TrimSpace(r.PathValue("subpath"))
		if subpath == "ui" || strings.HasPrefix(subpath, "ui/") {
			servePluginUIAsset(w, r, plugID, strings.TrimSpace(strings.TrimPrefix(subpath, "ui/")))
			return
		}
		handlePluginViewRoute(w, r, plugID, subpath)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/plugins/") || strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		fileServer.ServeHTTP(w, r)
	})

	return mux, nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
