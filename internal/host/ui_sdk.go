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

	"optimis/internal/dal"
)

const pluginSandboxCSP = "default-src 'none'; script-src 'self' 'unsafe-inline' http: https:; style-src 'self' 'unsafe-inline' http: https:; img-src 'self' data:; font-src 'self' data:; connect-src 'none';"

type PluginUIWidgetDescriptor struct {
	ID          string   `json:"id"`
	PluginID    string   `json:"plugin_id"`
	PluginTitle string   `json:"plugin_title"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Icon        string   `json:"icon"`
	Entry       string   `json:"entry"`
	EntryURL    string   `json:"entry_url"`
	Width       int      `json:"width"`
	Height      int      `json:"height"`
	Keywords    []string `json:"keywords"`
}

type PluginUIViewDescriptor struct {
	ID          string   `json:"id"`
	PluginID    string   `json:"plugin_id"`
	Title       string   `json:"title"`
	Icon        string   `json:"icon"`
	Path        string   `json:"path"`
	Entry       string   `json:"entry"`
	EntryURL    string   `json:"entry_url"`
	QueryParams []string `json:"query_params"`
	Keywords    []string `json:"keywords"`
}

type PluginUINavDescriptor struct {
	ID               string                     `json:"id"`
	PlugID           string                     `json:"plug_id"`
	Title            string                     `json:"title"`
	Icon             string                     `json:"icon"`
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

	allowedQuery := map[string]bool{}
	for _, qp := range matchedView.QueryParams {
		trimmedQP := strings.TrimSpace(qp)
		if trimmedQP != "" {
			allowedQuery[trimmedQP] = true
		}
	}
	for _, qp := range matchedView.AllowedQueryParams {
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
			"--mis-font-size-base":      "16px",
			"--mis-font-scale":          "1",
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
			"--mis-font-size-base":      "16px",
			"--mis-font-scale":          "1",
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
			PluginIDs      []string `json:"plugin_ids"`
			IncludeWidgets bool     `json:"include_widgets"`
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
			if in.IncludeWidgets && len(basePlugin.Manifest.Widgets) > 0 {
				clonedWidgets := make([]WidgetSpec, len(basePlugin.Manifest.Widgets))
				copy(clonedWidgets, basePlugin.Manifest.Widgets)
				cloneManifest.Widgets = clonedWidgets
			} else {
				cloneManifest.Widgets = nil
			}

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
		caller, callerErr := h.ResolveRequestCaller(r, "")
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

			pluginIcon := strings.ToLower(strings.TrimSpace(p.Manifest.Icon))
			if pluginIcon == "" {
				pluginIcon = "box"
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
				qParams := make([]string, 0, len(vSpec.QueryParams)+len(vSpec.AllowedQueryParams))
				seenQP := make(map[string]struct{}, len(vSpec.QueryParams)+len(vSpec.AllowedQueryParams))
				for _, qp := range append(append([]string(nil), vSpec.QueryParams...), vSpec.AllowedQueryParams...) {
					if trimmedQP := strings.TrimSpace(qp); trimmedQP != "" {
						if _, exists := seenQP[trimmedQP]; !exists {
							seenQP[trimmedQP] = struct{}{}
							qParams = append(qParams, trimmedQP)
						}
					}
				}
				vKeywords := make([]string, 0, len(vSpec.Keywords))
				for _, kw := range vSpec.Keywords {
					if trimmedKW := strings.TrimSpace(kw); trimmedKW != "" {
						vKeywords = append(vKeywords, trimmedKW)
					}
				}
				vTitle := strings.TrimSpace(vSpec.Title)
				if vTitle == "" {
					vTitle = title
				}
				vIcon := strings.ToLower(strings.TrimSpace(vSpec.Icon))
				if vIcon == "" {
					vIcon = pluginIcon
				}
				views = append(views, PluginUIViewDescriptor{
					ID:          vSpec.ID,
					PluginID:    plugID,
					Title:       vTitle,
					Icon:        vIcon,
					Path:        vSpec.Path,
					Entry:       vSpec.Entry,
					EntryURL:    fmt.Sprintf("/p/%s/ui/%s", plugID, relEntry),
					QueryParams: qParams,
					Keywords:    vKeywords,
				})
			}

			widgets := make([]PluginUIWidgetDescriptor, 0, len(p.Manifest.Widgets))
			for _, wSpec := range p.Manifest.Widgets {
				relEntry, err := normalizeWidgetEntryRelPath(wSpec.Entry)
				if err != nil {
					continue
				}
				wKeywords := make([]string, 0, len(wSpec.Keywords))
				for _, kw := range wSpec.Keywords {
					if trimmedKW := strings.TrimSpace(kw); trimmedKW != "" {
						wKeywords = append(wKeywords, trimmedKW)
					}
				}
				wIcon := strings.ToLower(strings.TrimSpace(wSpec.Icon))
				if wIcon == "" {
					wIcon = pluginIcon
				}
				widgets = append(widgets, PluginUIWidgetDescriptor{
					ID:          wSpec.ID,
					PluginID:    plugID,
					PluginTitle: title,
					Title:       wSpec.Title,
					Description: wSpec.Description,
					Icon:        wIcon,
					Entry:       wSpec.Entry,
					EntryURL:    fmt.Sprintf("/p/%s/ui/%s", plugID, relEntry),
					Width:       wSpec.Width,
					Height:      wSpec.Height,
					Keywords:    wKeywords,
				})
			}

			pluginDescriptors = append(pluginDescriptors, PluginUINavDescriptor{
				ID:               plugID,
				PlugID:           plugID,
				Title:            title,
				Icon:             pluginIcon,
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

		staffSettings, settingsErr := h.coreDAL.GetStaffSettings(r.Context(), caller.StaffID)
		if settingsErr != nil {
			staffSettings = dal.StaffSettings{
				StaffID:      caller.StaffID,
				BaseFontSize: "medium",
				ThemeMode:    "light",
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
			"pinned_recents":       h.loadPinnedRecentsForStaff(r.Context(), caller.StaffID),
			"user_settings": map[string]any{
				"staff_id":       staffSettings.StaffID,
				"base_font_size": staffSettings.BaseFontSize,
				"theme_mode":     staffSettings.ThemeMode,
			},
		})
	})

	mux.HandleFunc("GET /api/system/search", func(w http.ResponseWriter, r *http.Request) {
		caller, callerErr := h.ResolveRequestCaller(r, "")
		if callerErr != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]any{
				"ok":         false,
				"error_code": callerErr.Error(),
			})
			return
		}

		rawQuery := strings.TrimSpace(r.URL.Query().Get("q"))
		qLower := strings.ToLower(rawQuery)

		baseScope, scopeErr := h.coreDAL.ResolveScopeDefaults(r.Context(), dal.DALQueryScope{
			Mode:           dal.ScopeAll,
			StaffID:        caller.StaffID,
			AcademicYearID: strings.TrimSpace(r.URL.Query().Get("academic_year_id")),
			ReferenceDate:  strings.TrimSpace(r.URL.Query().Get("reference_date")),
		})
		if scopeErr != nil {
			baseScope.AcademicYearID = "ay_2026_2027"
			baseScope.ReferenceDate = "2026-10-27"
		}

		hasPupilAll := isCallerAdminOrDPO(caller, false)
		hasPupilOwn := false
		hasGroupAll := isCallerAdminOrDPO(caller, false)
		hasGroupOwn := false

		for _, s := range caller.Scopes {
			switch s {
			case "dal.pupils.read:all":
				hasPupilAll = true
			case "dal.pupils.read:own_groups":
				hasPupilOwn = true
			case "dal.groups.read:all":
				hasGroupAll = true
			case "dal.groups.read:own_groups":
				hasGroupOwn = true
			}
		}

		if hasPupilAll && hasPupilOwn && !callerHasAllRole(caller.ActiveRoles) {
			hasPupilAll = false
		}
		if hasGroupAll && hasGroupOwn && !callerHasAllRole(caller.ActiveRoles) {
			hasGroupAll = false
		}

		matchedGroups := make([]dal.GroupSummary, 0)
		var callerOwnGroups []dal.GroupSummary
		if hasGroupAll || hasGroupOwn || hasPupilOwn {
			gMode := dal.ScopeOwnGroups
			if hasGroupAll {
				gMode = dal.ScopeAll
			}
			gScope := baseScope
			gScope.Mode = gMode
			if groups, err := h.coreDAL.ListGroupsScoped(r.Context(), gScope, ""); err == nil {
				if gMode == dal.ScopeOwnGroups {
					callerOwnGroups = groups
				}
				if hasGroupAll || hasGroupOwn {
					for _, g := range groups {
						if qLower == "" ||
							strings.Contains(strings.ToLower(g.Code), qLower) ||
							strings.Contains(strings.ToLower(g.Name), qLower) ||
							strings.Contains(strings.ToLower(g.GroupType), qLower) ||
							strings.Contains(strings.ToLower(g.ID), qLower) {
							matchedGroups = append(matchedGroups, g)
						}
					}
				}
			}
		}

		matchedPupils := make([]dal.Pupil, 0)
		if hasPupilAll {
			pScope := baseScope
			pScope.Mode = dal.ScopeAll
			if pupils, err := h.coreDAL.ListAllPupilsScoped(r.Context(), pScope); err == nil {
				for _, p := range pupils {
					if matchesPupilSearchQuery(p, qLower) {
						matchedPupils = append(matchedPupils, p)
					}
				}
			}
		} else if hasPupilOwn && strings.TrimSpace(caller.StaffID) != "" {
			if callerOwnGroups == nil {
				ownGScope := baseScope
				ownGScope.Mode = dal.ScopeOwnGroups
				callerOwnGroups, _ = h.coreDAL.ListGroupsScoped(r.Context(), ownGScope, "")
			}
			pScope := baseScope
			pScope.Mode = dal.ScopeOwnGroups
			seenPupils := make(map[string]bool)
			for _, g := range callerOwnGroups {
				if gPupils, err := h.coreDAL.ListPupilsByGroupScoped(r.Context(), pScope, g.ID); err == nil {
					for _, p := range gPupils {
						if seenPupils[p.ID] {
							continue
						}
						seenPupils[p.ID] = true
						if matchesPupilSearchQuery(p, qLower) {
							matchedPupils = append(matchedPupils, p)
						}
					}
				}
			}
		}

		matchedViews := make([]PluginUIViewDescriptor, 0)
		matchedWidgets := make([]PluginUIWidgetDescriptor, 0)

		h.mu.RLock()
		for _, p := range h.plugins {
			if p.State != PluginStateActive {
				continue
			}
			plugID := strings.TrimSpace(p.PlugID)
			if plugID == "" {
				plugID = p.Manifest.ID
			}
			permittedCount := 0
			for _, spec := range p.Manifest.Endpoints {
				if (hasAllScopes(caller.Scopes, spec.Scopes) && hasRequiredEndpointRoles(caller, spec.RequireRoles)) || isCallerAdminOrDPO(caller, false) {
					permittedCount++
				}
			}
			if len(p.Manifest.Endpoints) > 0 && permittedCount == 0 {
				continue
			}

			pluginTitle := strings.TrimSpace(p.Manifest.Title)
			if pluginTitle == "" {
				pluginTitle = p.Folder
			}
			pluginIcon := strings.ToLower(strings.TrimSpace(p.Manifest.Icon))
			if pluginIcon == "" {
				pluginIcon = "box"
			}

			viewsList := p.Manifest.Views
			if len(viewsList) == 0 {
				viewsList = defaultPluginViews()
			}
			for _, vSpec := range viewsList {
				relEntry, err := normalizeWidgetEntryRelPath(vSpec.Entry)
				if err != nil {
					continue
				}
				vTitle := strings.TrimSpace(vSpec.Title)
				if vTitle == "" {
					vTitle = pluginTitle
				}
				vIcon := strings.ToLower(strings.TrimSpace(vSpec.Icon))
				if vIcon == "" {
					vIcon = pluginIcon
				}
				vKeywords := make([]string, 0, len(vSpec.Keywords))
				for _, kw := range vSpec.Keywords {
					if trimmedKW := strings.TrimSpace(kw); trimmedKW != "" {
						vKeywords = append(vKeywords, trimmedKW)
					}
				}
				if matchesViewOrWidgetSearchQuery(vTitle, vSpec.ID, vSpec.Path, pluginTitle, vKeywords, qLower) {
					matchedViews = append(matchedViews, PluginUIViewDescriptor{
						ID:          vSpec.ID,
						PluginID:    plugID,
						Title:       vTitle,
						Icon:        vIcon,
						Path:        vSpec.Path,
						Entry:       vSpec.Entry,
						EntryURL:    fmt.Sprintf("/p/%s/ui/%s", plugID, relEntry),
						QueryParams: vSpec.QueryParams,
						Keywords:    vKeywords,
					})
				}
			}

			for _, wSpec := range p.Manifest.Widgets {
				relEntry, err := normalizeWidgetEntryRelPath(wSpec.Entry)
				if err != nil {
					continue
				}
				wIcon := strings.ToLower(strings.TrimSpace(wSpec.Icon))
				if wIcon == "" {
					wIcon = pluginIcon
				}
				wKeywords := make([]string, 0, len(wSpec.Keywords))
				for _, kw := range wSpec.Keywords {
					if trimmedKW := strings.TrimSpace(kw); trimmedKW != "" {
						wKeywords = append(wKeywords, trimmedKW)
					}
				}
				if matchesViewOrWidgetSearchQuery(wSpec.Title, wSpec.ID, wSpec.Description, pluginTitle, wKeywords, qLower) {
					matchedWidgets = append(matchedWidgets, PluginUIWidgetDescriptor{
						ID:          wSpec.ID,
						PluginID:    plugID,
						PluginTitle: pluginTitle,
						Title:       wSpec.Title,
						Description: wSpec.Description,
						Icon:        wIcon,
						Entry:       wSpec.Entry,
						EntryURL:    fmt.Sprintf("/p/%s/ui/%s", plugID, relEntry),
						Width:       wSpec.Width,
						Height:      wSpec.Height,
						Keywords:    wKeywords,
					})
				}
			}
		}
		h.mu.RUnlock()

		writeJSON(w, http.StatusOK, map[string]any{
			"ok":      true,
			"query":   rawQuery,
			"pupils":  matchedPupils,
			"groups":  matchedGroups,
			"views":   matchedViews,
			"widgets": matchedWidgets,
		})
	})

	h.RegisterPinnedRecentsAndLogoutRoutes(mux)
}

func matchesPupilSearchQuery(p dal.Pupil, qLower string) bool {
	if qLower == "" {
		return true
	}
	fullName := strings.ToLower(strings.TrimSpace(p.FirstName + " " + p.LastName))
	prefName := strings.ToLower(strings.TrimSpace(p.PreferredFirstName + " " + p.PreferredLastName))
	return strings.Contains(fullName, qLower) ||
		strings.Contains(prefName, qLower) ||
		strings.Contains(strings.ToLower(p.FirstName), qLower) ||
		strings.Contains(strings.ToLower(p.LastName), qLower) ||
		strings.Contains(strings.ToLower(p.UPN), qLower) ||
		strings.Contains(strings.ToLower(p.ID), qLower) ||
		strings.Contains(strings.ToLower("year "+p.NCYearGroup), qLower)
}

func matchesViewOrWidgetSearchQuery(title, id, path, pluginTitle string, keywords []string, qLower string) bool {
	if qLower == "" {
		return true
	}
	if strings.Contains(strings.ToLower(title), qLower) ||
		strings.Contains(strings.ToLower(id), qLower) ||
		strings.Contains(strings.ToLower(path), qLower) ||
		strings.Contains(strings.ToLower(pluginTitle), qLower) {
		return true
	}
	for _, kw := range keywords {
		if strings.Contains(strings.ToLower(kw), qLower) {
			return true
		}
	}
	return false
}


