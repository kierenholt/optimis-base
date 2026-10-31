package host

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

const pluginSandboxCSP = "default-src 'none'; script-src 'self' 'unsafe-inline' http: https:; style-src 'self' 'unsafe-inline' http: https:; img-src 'self' data:; font-src 'self' data:; connect-src 'none';"

type PluginUIWidgetDescriptor struct {
	ID       string `json:"id"`
	PluginID string `json:"plugin_id"`
	Title    string `json:"title"`
	Entry    string `json:"entry"`
	EntryURL string `json:"entry_url"`
}

type PluginUIViewDescriptor struct {
	ID          string   `json:"id"`
	PluginID    string   `json:"plugin_id"`
	Title       string   `json:"title"`
	Path        string   `json:"path"`
	Entry       string   `json:"entry"`
	EntryURL    string   `json:"entry_url"`
	QueryParams []string `json:"query_params"`
}

type PluginUINavDescriptor struct {
	ID               string                     `json:"id"`
	PlugID           string                     `json:"plug_id"`
	Title            string                     `json:"title"`
	Version          string                     `json:"version"`
	EntryURL         string                     `json:"entry_url"`
	PermittedMethods []string                   `json:"permitted_methods"`
	Views            []PluginUIViewDescriptor   `json:"views"`
	Widgets          []PluginUIWidgetDescriptor `json:"widgets"`
	 isDefaultTest    bool
}

type MatchedPluginView struct {
	View        ViewSpec          `json:"view"`
	EntryRel    string            `json:"entry_rel"`
	PathParams  map[string]string `json:"path_params"`
	QueryParams map[string]string `json:"query_params"`
}

func MatchPluginViewRoute(plugin *LoadedPlugin, rawSubpath string, query url.Values) (*MatchedPluginView, int, string) {
	if plugin == nil {
		return nil, http.StatusNotFound, "PLUGIN_NOT_FOUND: plugin is nil"
	}
	trimmedSub := strings.Trim(strings.TrimSpace(rawSubpath), "/")
	normalizedPath := "/"
	var targetSegs []string
	if trimmedSub != "" {
		normalizedPath = "/" + trimmedSub
		targetSegs = strings.Split(trimmedSub, "/")
	}

	views := plugin.Manifest.Views
	if len(views) == 0 {
		views = defaultPluginViews()
	}

	var matchedView *ViewSpec
	var extractedParams map[string]string

	for i := range views {
		v := &views[i]
		patTrimmed := strings.Trim(strings.TrimSpace(v.Path), "/")
		var patSegs []string
		if patTrimmed != "" {
			patSegs = strings.Split(patTrimmed, "/")
		}
		if len(patSegs) != len(targetSegs) {
			continue
		}
		params := make(map[string]string)
		matched := true
		for idx, pSeg := range patSegs {
			tSeg := strings.TrimSpace(targetSegs[idx])
			if tSeg == "" {
				matched = false
				break
			}
			if strings.HasPrefix(pSeg, ":") && len(pSeg) > 1 {
				params[pSeg[1:]] = tSeg
			} else if pSeg != tSeg {
				matched = false
				break
			}
		}
		if matched {
			matchedView = v
			extractedParams = params
			break
		}
	}

	if matchedView == nil {
		return nil, http.StatusNotFound, fmt.Sprintf("UNDECLARED_PLUGIN_VIEW_PATH: path %q is not declared in manifest.json views", normalizedPath)
	}

	allowedQuery := map[string]bool{
		"__persona": true,
		"staff_id":  true,
	}
	for _, qp := range matchedView.QueryParams {
		trimmedQP := strings.TrimSpace(qp)
		if trimmedQP != "" {
			allowedQuery[trimmedQP] = true
		}
	}

	validatedQuery := make(map[string]string, len(query))
	for k, vals := range query {
		if !allowedQuery[k] {
			return nil, http.StatusBadRequest, fmt.Sprintf(
				"UNDECLARED_PLUGIN_VIEW_QUERY_PARAM: query parameter %q is not declared in manifest.json views for path %q",
				k,
				matchedView.Path,
			)
		}
		if len(vals) > 0 {
			validatedQuery[k] = vals[0]
		}
	}

	relEntry, err := normalizeWidgetEntryRelPath(matchedView.Entry)
	if err != nil {
		relEntry = "index.html"
	}

	return &MatchedPluginView{
		View:        *matchedView,
		EntryRel:    relEntry,
		PathParams:  extractedParams,
		QueryParams: validatedQuery,
	}, http.StatusOK, ""
}

func DefaultThemeTokens() map[string]map[string]string {
	return map[string]map[string]string{
		"light": {
			"--mis-brand-primary":       "#1d4ed8",
			"--mis-brand-primary-hover": "#1e40af",
			"--mis-brand-primary-fg":    "#ffffff",
			"--mis-brand-accent":        "#0f766e",
			"--mis-bg-canvas":           "#f1f5f9",
			"--mis-bg-surface":          "#ffffff",
			"--mis-bg-elevated":         "#f8fafc",
			"--mis-bg-sidebar":          "#0f172a",
			"--mis-text-sidebar":        "#f8fafc",
			"--mis-text-primary":        "#0f172a",
			"--mis-text-secondary":      "#475569",
			"--mis-text-muted":          "#64748b",
			"--mis-text-inverse":        "#ffffff",
			"--mis-border-subtle":       "#e2e8f0",
			"--mis-border-strong":       "#cbd5e1",
			"--mis-status-success":      "#15803d",
			"--mis-status-warning":      "#b45309",
			"--mis-status-danger":       "#b91c1c",
			"--mis-status-danger-bg":    "#fef2f2",
		},
		"high-contrast": {
			"--mis-brand-primary":       "#facc15",
			"--mis-brand-primary-hover": "#fde047",
			"--mis-brand-primary-fg":    "#000000",
			"--mis-brand-accent":        "#22d3ee",
			"--mis-bg-canvas":           "#000000",
			"--mis-bg-surface":          "#0a0a0a",
			"--mis-bg-elevated":         "#171717",
			"--mis-bg-sidebar":          "#000000",
			"--mis-text-sidebar":        "#ffffff",
			"--mis-text-primary":        "#ffffff",
			"--mis-text-secondary":      "#fef08a",
			"--mis-text-muted":          "#e2e8f0",
			"--mis-text-inverse":        "#000000",
			"--mis-border-subtle":       "#ffffff",
			"--mis-border-strong":       "#facc15",
			"--mis-status-success":      "#4ade80",
			"--mis-status-warning":      "#facc15",
			"--mis-status-danger":       "#ff6b6b",
			"--mis-status-danger-bg":    "#2a0000",
		},
	}
}

func resolveSDKUIDir() (string, error) {
	if stat, err := os.Stat("sdk/ui"); err == nil && stat.IsDir() {
		return filepath.Abs("sdk/ui")
	}
	_, callerFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("failed to resolve caller path for sdk/ui directory")
	}
	sdkDir := filepath.Clean(filepath.Join(filepath.Dir(callerFile), "..", "..", "sdk", "ui"))
	info, err := os.Stat(sdkDir)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("mandatory sdk/ui directory not found at %s: %w", sdkDir, err)
	}
	return sdkDir, nil
}

func readSDKUIAsset(assetName string) ([]byte, string, error) {
	clean := filepath.Base(strings.TrimSpace(assetName))
	if clean != "mis-bridge.js" && clean != "mis-theme.css" {
		return nil, "", fmt.Errorf("unsupported SDK UI asset: %s", assetName)
	}
	sdkDir, err := resolveSDKUIDir()
	if err != nil {
		return nil, "", err
	}
	fullPath := filepath.Join(sdkDir, clean)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read SDK UI asset %s: %w", clean, err)
	}
	contentType := "application/javascript; charset=utf-8"
	if strings.HasSuffix(clean, ".css") {
		contentType = "text/css; charset=utf-8"
	}
	return content, contentType, nil
}

func (h *Host) RegisterUISDKAndConfigRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /sdk/{asset_name}", func(w http.ResponseWriter, r *http.Request) {
		assetName := strings.TrimSpace(r.PathValue("asset_name"))
		content, contentType, err := readSDKUIAsset(assetName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Security-Policy", pluginSandboxCSP)
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(content)
	})

	mux.HandleFunc("POST /api/test/seed-lru-plugins", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			PluginIDs []string `json:"plugin_ids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":    false,
				"error": fmt.Sprintf("invalid JSON body: %v", err),
			})
			return
		}
		if len(in.PluginIDs) == 0 {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":    false,
				"error": "missing mandatory parameter: plugin_ids",
			})
			return
		}

		h.mu.RLock()
		basePlugin, ok := h.plugins["uk.org.optimis.test"]
		h.mu.RUnlock()
		if !ok {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":    false,
				"error": "base plugin uk.org.optimis.test is not loaded",
			})
			return
		}

		seededIDs := make([]string, 0, len(in.PluginIDs))
		for _, rawID := range in.PluginIDs {
			pid := strings.TrimSpace(rawID)
			if pid == "" || pid == "uk.org.optimis.test" || pid == basePlugin.PlugID {
				continue
			}
			plugID := pid
			if !opaquePlugIDPattern.MatchString(plugID) {
				var genErr error
				plugID, genErr = h.ensurePluginInstanceID(r.Context(), pid)
				if genErr != nil {
					writeJSON(w, http.StatusInternalServerError, map[string]any{
						"ok":    false,
						"error": genErr.Error(),
					})
					return
				}
			}

			cloneManifest := basePlugin.Manifest
			cloneManifest.ID = pid
			cloneManifest.Title = fmt.Sprintf("Synthetic Workspace (%s)", plugID)
			cloneManifest.Widgets = nil

			h.mu.Lock()
			h.plugins[pid] = &LoadedPlugin{
				PlugID:           plugID,
				Manifest:         cloneManifest,
				LuaSource:        basePlugin.LuaSource,
				DirPath:          basePlugin.DirPath,
				Folder:           basePlugin.Folder,
				State:            PluginStateActive,
				ManifestModTime:  basePlugin.ManifestModTime,
				LuaModTime:       basePlugin.LuaModTime,
				IsSyntheticClone: true,
			}
			h.mu.Unlock()
			seededIDs = append(seededIDs, plugID)
		}

		h.logInfo(fmt.Sprintf("Seeded %d synthetic LRU plugin(s)", len(seededIDs)))
		writeJSON(w, http.StatusCreated, map[string]any{
			"ok":                true,
			"seeded_plugin_ids": seededIDs,
		})
	})

	mux.HandleFunc("DELETE /api/test/seed-lru-plugins", func(w http.ResponseWriter, r *http.Request) {
		h.mu.Lock()
		removed := 0
		for pid, p := range h.plugins {
			if p.IsSyntheticClone {
				delete(h.plugins, pid)
				removed++
			}
		}
		h.mu.Unlock()

		h.logInfo(fmt.Sprintf("Deleted %d synthetic LRU plugin(s)", removed))
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":      true,
			"deleted": true,
			"removed": removed,
		})
	})

	mux.HandleFunc("GET /api/system/ui-config", func(w http.ResponseWriter, r *http.Request) {
		caller, callerErr := h.ResolveRequestCaller(r, strings.TrimSpace(r.URL.Query().Get("staff_id")))
		if callerErr != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]any{
				"ok":         false,
				"error_code": callerErr.Error(),
			})
			return
		}

		schoolURN := h.cfg.SchoolURN
		schoolName := "Oakridge All-Through Academy"
		academicYearID := "ay_2026_2027"

		rawEntityHandlers := map[string]string{
			"student": "uk.org.optimis.test",
			"group":   "uk.org.optimis.test",
			"staff":   "uk.org.optimis.test",
		}

		rows, err := h.coreDAL.ReadDB().QueryContext(r.Context(), `SELECT key, value FROM __system_config`)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var k, v string
				if scanErr := rows.Scan(&k, &v); scanErr == nil {
					switch k {
					case "school_urn":
						if strings.TrimSpace(v) != "" {
							schoolURN = strings.TrimSpace(v)
						}
					case "school_name":
						if strings.TrimSpace(v) != "" {
							schoolName = strings.TrimSpace(v)
						}
					case "academic_year_current_id":
						if strings.TrimSpace(v) != "" {
							academicYearID = strings.TrimSpace(v)
						}
					default:
						if strings.HasPrefix(k, "entity_handler.") {
							entityKey := strings.TrimSpace(strings.TrimPrefix(k, "entity_handler."))
							if entityKey != "" && strings.TrimSpace(v) != "" {
								rawEntityHandlers[entityKey] = strings.TrimSpace(v)
							}
						}
					}
				}
			}
		}

		h.mu.RLock()
		activeManifestToPlugID := make(map[string]string, len(h.plugins))
		activePlugIDs := make(map[string]bool, len(h.plugins))
		var firstActivePlugID string
		pluginDescriptors := make([]PluginUINavDescriptor, 0, len(h.plugins))
		for _, p := range h.plugins {
			if p.State != PluginStateActive {
				continue
			}
			plugID := strings.TrimSpace(p.PlugID)
			if plugID == "" {
				plugID = p.Manifest.ID
			}

			methods := make([]string, 0, len(p.Manifest.Endpoints))
			for m, spec := range p.Manifest.Endpoints {
				if (hasAllScopes(caller.Scopes, spec.Scopes) && hasRequiredEndpointRoles(caller, spec.RequireRoles)) || isCallerAdminOrDPO(caller, false) {
					methods = append(methods, m)
				}
			}
			if len(p.Manifest.Endpoints) > 0 && len(methods) == 0 {
				continue
			}
			sort.Strings(methods)

			activeManifestToPlugID[p.Manifest.ID] = plugID
			activePlugIDs[plugID] = true

			title := strings.TrimSpace(p.Manifest.Title)
			if title == "" {
				title = p.Folder
				if p.Manifest.ID == "uk.org.optimis.test" {
					title = "Pupil & Classroom Workspace"
				}
			}

			viewsList := p.Manifest.Views
			if len(viewsList) == 0 {
				viewsList = defaultPluginViews()
			}
			views := make([]PluginUIViewDescriptor, 0, len(viewsList))
			for _, vSpec := range viewsList {
				relEntry, err := normalizeWidgetEntryRelPath(vSpec.Entry)
				if err != nil {
					continue
				}
				qParams := make([]string, 0, len(vSpec.QueryParams))
				for _, qp := range vSpec.QueryParams {
					if trimmedQP := strings.TrimSpace(qp); trimmedQP != "" {
						qParams = append(qParams, trimmedQP)
					}
				}
				vTitle := strings.TrimSpace(vSpec.Title)
				if vTitle == "" {
					vTitle = title
				}
				views = append(views, PluginUIViewDescriptor{
					ID:          vSpec.ID,
					PluginID:    plugID,
					Title:       vTitle,
					Path:        vSpec.Path,
					Entry:       vSpec.Entry,
					EntryURL:    fmt.Sprintf("/p/%s/ui/%s", plugID, relEntry),
					QueryParams: qParams,
				})
			}

			widgets := make([]PluginUIWidgetDescriptor, 0, len(p.Manifest.Widgets))
			for _, wSpec := range p.Manifest.Widgets {
				relEntry, err := normalizeWidgetEntryRelPath(wSpec.Entry)
				if err != nil {
					continue
				}
				widgets = append(widgets, PluginUIWidgetDescriptor{
					ID:       wSpec.ID,
					PluginID: plugID,
					Title:    wSpec.Title,
					Entry:    wSpec.Entry,
					EntryURL: fmt.Sprintf("/p/%s/ui/%s", plugID, relEntry),
				})
			}

			pluginDescriptors = append(pluginDescriptors, PluginUINavDescriptor{
				ID:               plugID,
				PlugID:           plugID,
				Title:            title,
				Version:          p.Manifest.Version,
				EntryURL:         fmt.Sprintf("/p/%s/ui/index.html", plugID),
				PermittedMethods: methods,
				Views:            views,
				Widgets:          widgets,
				isDefaultTest:    p.Manifest.ID == "uk.org.optimis.test",
			})
		}
		h.mu.RUnlock()

		sort.Slice(pluginDescriptors, func(i, j int) bool {
			if pluginDescriptors[i].isDefaultTest {
				return true
			}
			if pluginDescriptors[j].isDefaultTest {
				return false
			}
			return pluginDescriptors[i].Title < pluginDescriptors[j].Title
		})
		if len(pluginDescriptors) > 0 {
			firstActivePlugID = pluginDescriptors[0].PlugID
		}

		entityHandlers := make(map[string]string, len(rawEntityHandlers))
		for entityType, rawTarget := range rawEntityHandlers {
			if mappedPlugID, ok := activeManifestToPlugID[rawTarget]; ok {
				entityHandlers[entityType] = mappedPlugID
			} else if activePlugIDs[rawTarget] {
				entityHandlers[entityType] = rawTarget
			} else if firstActivePlugID != "" {
				entityHandlers[entityType] = firstActivePlugID
			} else {
				entityHandlers[entityType] = ""
			}
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"ok":                   true,
			"school_urn":           schoolURN,
			"school_name":          schoolName,
			"academic_year_id":     academicYearID,
			"themes":               DefaultThemeTokens(),
			"entity_link_handlers": entityHandlers,
			"plugins":              pluginDescriptors,
		})
	})
}


