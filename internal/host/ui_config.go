package host

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"optimis/internal/dal"
)

// ============================================================================
// System UI Config Helpers & Handler
// ============================================================================

func (h *Host) loadSystemConfigMetadata(ctx context.Context) (schoolURN, schoolName, academicYearID string, rawEntityHandlers map[string]string) {
	schoolURN = h.cfg.SchoolURN
	schoolName = "Oakridge All-Through Academy"
	academicYearID = "ay_2026_2027"

	rawEntityHandlers = map[string]string{
		"student": "uk.org.optimis.test",
		"group":   "uk.org.optimis.test",
		"staff":   "uk.org.optimis.test",
	}

	rows, err := h.coreDAL.ReadDB().QueryContext(ctx, `SELECT key, value FROM __system_config`)
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
	return
}

func buildPluginUIViewDescriptors(viewsList []ViewSpec, plugID, fallbackTitle, fallbackIcon string) []PluginUIViewDescriptor {
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
			vTitle = fallbackTitle
		}
		vIcon := strings.ToLower(strings.TrimSpace(vSpec.Icon))
		if vIcon == "" {
			vIcon = fallbackIcon
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
	return views
}

func buildPluginUIWidgetDescriptors(widgetsList []WidgetSpec, plugID, fallbackTitle, fallbackIcon string) []PluginUIWidgetDescriptor {
	widgets := make([]PluginUIWidgetDescriptor, 0, len(widgetsList))
	for _, wSpec := range widgetsList {
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
			wIcon = fallbackIcon
		}
		widgets = append(widgets, PluginUIWidgetDescriptor{
			ID:          wSpec.ID,
			PluginID:    plugID,
			PluginTitle: fallbackTitle,
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
	return widgets
}

func (h *Host) buildPluginNavDescriptors(caller CallerContext) ([]PluginUINavDescriptor, map[string]string, map[string]bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	activeManifestToPlugID := make(map[string]string, len(h.plugins))
	activePlugIDs := make(map[string]bool, len(h.plugins))
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

		views := buildPluginUIViewDescriptors(p.Manifest.Views, plugID, title, pluginIcon)
		widgets := buildPluginUIWidgetDescriptors(p.Manifest.Widgets, plugID, title, pluginIcon)

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

	sort.Slice(pluginDescriptors, func(i, j int) bool {
		if pluginDescriptors[i].isDefaultTest {
			return true
		}
		if pluginDescriptors[j].isDefaultTest {
			return false
		}
		return pluginDescriptors[i].Title < pluginDescriptors[j].Title
	})

	return pluginDescriptors, activeManifestToPlugID, activePlugIDs
}

func resolveEntityHandlers(rawEntityHandlers, activeManifestToPlugID map[string]string, activePlugIDs map[string]bool, firstActivePlugID string) map[string]string {
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
	return entityHandlers
}

func (h *Host) handleGetSystemUIConfig(w http.ResponseWriter, r *http.Request) {
	caller, callerErr := h.ResolveRequestCaller(r, "")
	if callerErr != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"ok":         false,
			"error_code": callerErr.Error(),
		})
		return
	}

	schoolURN, schoolName, academicYearID, rawEntityHandlers := h.loadSystemConfigMetadata(r.Context())
	pluginDescriptors, activeManifestToPlugID, activePlugIDs := h.buildPluginNavDescriptors(caller)

	var firstActivePlugID string
	if len(pluginDescriptors) > 0 {
		firstActivePlugID = pluginDescriptors[0].PlugID
	}
	entityHandlers := resolveEntityHandlers(rawEntityHandlers, activeManifestToPlugID, activePlugIDs, firstActivePlugID)

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
}

// ============================================================================
// Synthetic LRU Testing Handlers
// ============================================================================

func (h *Host) handleSeedLRUPlugins(w http.ResponseWriter, r *http.Request) {
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
}

func (h *Host) handleDeleteLRUPlugins(w http.ResponseWriter, r *http.Request) {
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
}
