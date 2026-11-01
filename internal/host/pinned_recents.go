package host

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type PinnedRecentItem struct {
	Route  string `json:"route"`
	Title  string `json:"title"`
	PlugID string `json:"plug_id"`
	ViewID string `json:"view_id,omitempty"`
	Icon   string `json:"icon,omitempty"`
}

func resolvePinnedRouteIcon(plugin *LoadedPlugin, matchedView *MatchedPluginView) string {
	if matchedView != nil {
		if vIcon := strings.ToLower(strings.TrimSpace(matchedView.View.Icon)); vIcon != "" {
			return vIcon
		}
	}
	if plugin != nil {
		if pIcon := strings.ToLower(strings.TrimSpace(plugin.Manifest.Icon)); pIcon != "" {
			return pIcon
		}
	}
	return "pin"
}

func pinnedRecentsConfigKey(staffID string) (string, error) {
	trimmed := strings.TrimSpace(staffID)
	if trimmed == "" {
		return "", fmt.Errorf("MISSING_MANDATORY_PARAMETER: staff_id is required for pinned_recents config key")
	}
	return "user_prefs.pinned_recents." + trimmed, nil
}

func parseAndValidatePinnedRoute(rawRoute string) (string, string, string, url.Values, error) {
	trimmed := strings.TrimSpace(rawRoute)
	if trimmed == "" {
		return "", "", "", nil, fmt.Errorf("MISSING_MANDATORY_PARAMETER: route is required")
	}
	if !strings.HasPrefix(trimmed, "/p/") {
		return "", "", "", nil, fmt.Errorf("INVALID_PINNED_RECENT_ROUTE: route %q must start with /p/<plugid>", trimmed)
	}
	pathPart, rawQuery, hasQuery := strings.Cut(trimmed, "?")
	queryValues := url.Values{}
	if hasQuery {
		parsedQuery, err := url.ParseQuery(rawQuery)
		if err != nil {
			return "", "", "", nil, fmt.Errorf("INVALID_PINNED_RECENT_ROUTE: invalid query string in route %q: %w", trimmed, err)
		}
		queryValues = parsedQuery
	}

	rest := strings.TrimPrefix(pathPart, "/p/")
	var plugID, subpath string
	slashIdx := strings.Index(rest, "/")
	if slashIdx == -1 {
		plugID = strings.TrimSpace(rest)
		subpath = "/"
	} else {
		plugID = strings.TrimSpace(rest[:slashIdx])
		subpath = rest[slashIdx:]
	}
	if plugID == "" || !opaquePlugIDPattern.MatchString(plugID) {
		return "", "", "", nil, fmt.Errorf("INVALID_PINNED_RECENT_ROUTE: route %q must contain a valid 8-char opaque plug_id", trimmed)
	}
	if subpath == "/ui" || strings.HasPrefix(subpath, "/ui/") {
		return "", "", "", nil, fmt.Errorf("RESERVED_PLUGIN_VIEW_PATH: route %q cannot target reserved static asset prefix /ui", trimmed)
	}
	return trimmed, plugID, subpath, queryValues, nil
}

func (h *Host) isPluginPermittedForCaller(plugin *LoadedPlugin, caller CallerContext) bool {
	if plugin == nil || plugin.State != PluginStateActive {
		return false
	}
	if len(plugin.Manifest.Endpoints) == 0 || isCallerAdminOrDPO(caller, false) {
		return true
	}
	for _, spec := range plugin.Manifest.Endpoints {
		if hasAllScopes(caller.Scopes, spec.Scopes) && hasRequiredEndpointRoles(caller, spec.RequireRoles) {
			return true
		}
	}
	return false
}

func (h *Host) validatePinnedRouteForCaller(rawRoute string, caller CallerContext) (string, string, *LoadedPlugin, *MatchedPluginView, int, error) {
	cleanRoute, plugID, subpath, queryValues, err := parseAndValidatePinnedRoute(rawRoute)
	if err != nil {
		return "", "", nil, nil, http.StatusBadRequest, err
	}
	plugin, ok := h.GetLoadedPluginByPlugID(plugID)
	if !ok || plugin == nil {
		return "", "", nil, nil, http.StatusNotFound, fmt.Errorf("PLUGIN_NOT_FOUND: unknown plug_id %q", plugID)
	}
	if plugin.State != PluginStateActive {
		return "", "", nil, nil, http.StatusForbidden, fmt.Errorf("PLUGIN_DORMANT_QUARANTINE: plugin %q is not active", plugID)
	}
	if !h.isPluginPermittedForCaller(plugin, caller) {
		return "", "", nil, nil, http.StatusForbidden, fmt.Errorf("FORBIDDEN_PLUGIN_ACCESS: caller %q lacks required scopes or roles for plugin %q", caller.StaffID, plugID)
	}
	matchedView, status, errCode := MatchPluginViewRoute(plugin, subpath, queryValues)
	if status != http.StatusOK || matchedView == nil {
		return "", "", nil, nil, status, fmt.Errorf("%s: route %q does not match any declared view in plugin %q", errCode, cleanRoute, plugID)
	}
	return cleanRoute, plugID, plugin, matchedView, http.StatusOK, nil
}

func (h *Host) loadPinnedRecentsForStaff(ctx context.Context, staffID string) []PinnedRecentItem {
	settings, err := h.coreDAL.GetStaffSettings(ctx, staffID)
	if err != nil {
		return []PinnedRecentItem{}
	}
	rawVal := strings.TrimSpace(settings.PinnedRecentsJSON)
	if rawVal == "" || rawVal == "[]" {
		return []PinnedRecentItem{}
	}
	var items []PinnedRecentItem
	if jsonErr := json.Unmarshal([]byte(rawVal), &items); jsonErr != nil || len(items) == 0 {
		return []PinnedRecentItem{}
	}

	targetCaller := h.resolveCallerForStaffID(ctx, staffID)
	validItems := make([]PinnedRecentItem, 0, len(items))
	for _, item := range items {
		cleanRoute, plugID, plugin, matchedView, status, vErr := h.validatePinnedRouteForCaller(item.Route, targetCaller)
		if vErr != nil || status != http.StatusOK || matchedView == nil {
			continue
		}
		viewID := strings.TrimSpace(item.ViewID)
		if viewID == "" {
			viewID = matchedView.View.ID
		}
		iconName := strings.ToLower(strings.TrimSpace(item.Icon))
		if iconName == "" || iconName == "pin" {
			iconName = resolvePinnedRouteIcon(plugin, matchedView)
		}
		validItems = append(validItems, PinnedRecentItem{
			Route:  cleanRoute,
			Title:  item.Title,
			PlugID: plugID,
			ViewID: viewID,
			Icon:   iconName,
		})
	}
	return validItems
}

func (h *Host) savePinnedRecentsForStaff(ctx context.Context, staffID string, items []PinnedRecentItem) error {
	if len(items) == 0 {
		return h.coreDAL.SaveStaffPinnedRecentsJSON(ctx, staffID, "[]")
	}
	encoded, err := json.Marshal(items)
	if err != nil {
		return fmt.Errorf("failed to marshal pinned_recents: %w", err)
	}
	return h.coreDAL.SaveStaffPinnedRecentsJSON(ctx, staffID, string(encoded))
}

func (h *Host) RegisterPinnedRecentsAndLogoutRoutes(mux *http.ServeMux) {
	handleUserSettings := func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			StaffID      string  `json:"staff_id"`
			BaseFontSize *string `json:"base_font_size"`
			ThemeMode    *string `json:"theme_mode"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": fmt.Sprintf("INVALID_JSON: %v", err),
			})
			return
		}

		caller, targetStaffID, authStatus, authErr := h.ResolveAuthenticatedCallerAndTargetStaff(r, in.StaffID)
		if authErr != nil {
			h.logError(fmt.Sprintf("Blocked user-settings update for target %q by caller %q: %v", in.StaffID, caller.StaffID, authErr))
			writeJSON(w, authStatus, map[string]any{
				"ok":         false,
				"error_code": authErr.Error(),
			})
			return
		}

		updatedSettings, err := h.coreDAL.UpsertStaffUISettings(r.Context(), targetStaffID, in.BaseFontSize, in.ThemeMode)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": err.Error(),
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"ok": true,
			"user_settings": map[string]any{
				"staff_id":       updatedSettings.StaffID,
				"base_font_size": updatedSettings.BaseFontSize,
				"theme_mode":     updatedSettings.ThemeMode,
			},
		})
	}
	mux.HandleFunc("PUT /api/system/user-settings", handleUserSettings)
	mux.HandleFunc("POST /api/system/user-settings", handleUserSettings)

	mux.HandleFunc("POST /api/system/pinned-recents", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Action  string `json:"action"`
			Route   string `json:"route"`
			Title   string `json:"title"`
			ViewID  string `json:"view_id"`
			Icon    string `json:"icon"`
			StaffID string `json:"staff_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": fmt.Sprintf("INVALID_JSON: %v", err),
			})
			return
		}

		caller, targetStaffID, authStatus, authErr := h.ResolveAuthenticatedCallerAndTargetStaff(r, in.StaffID)
		if authErr != nil {
			h.logError(fmt.Sprintf("Blocked pinned-recents update for target %q by caller %q: %v", in.StaffID, caller.StaffID, authErr))
			writeJSON(w, authStatus, map[string]any{
				"ok":         false,
				"error_code": authErr.Error(),
			})
			return
		}

		targetCaller := caller
		if targetStaffID != caller.StaffID {
			targetCaller = h.resolveCallerForStaffID(r.Context(), targetStaffID)
		}

		action := strings.ToLower(strings.TrimSpace(in.Action))
		if action == "" {
			action = "pin"
		}

		switch action {
		case "pin":
			cleanRoute, plugID, plugin, matchedView, routeStatus, routeErr := h.validatePinnedRouteForCaller(in.Route, targetCaller)
			if routeErr != nil {
				writeJSON(w, routeStatus, map[string]any{
					"ok":         false,
					"error_code": routeErr.Error(),
				})
				return
			}
			cleanTitle := strings.TrimSpace(in.Title)
			if cleanTitle == "" {
				writeJSON(w, http.StatusBadRequest, map[string]any{
					"ok":         false,
					"error_code": "MISSING_MANDATORY_PARAMETER: title is required when pinning a view route",
				})
				return
			}
			viewID := strings.TrimSpace(in.ViewID)
			if viewID == "" && matchedView != nil {
				viewID = matchedView.View.ID
			}
			iconName := strings.ToLower(strings.TrimSpace(in.Icon))
			if iconName == "" {
				iconName = resolvePinnedRouteIcon(plugin, matchedView)
			}

			existing := h.loadPinnedRecentsForStaff(r.Context(), targetStaffID)
			updated := make([]PinnedRecentItem, 0, len(existing)+1)
			found := false
			for _, item := range existing {
				if item.Route == cleanRoute {
					updated = append(updated, PinnedRecentItem{
						Route:  cleanRoute,
						Title:  cleanTitle,
						PlugID: plugID,
						ViewID: viewID,
						Icon:   iconName,
					})
					found = true
				} else {
					updated = append(updated, item)
				}
			}
			if !found {
				updated = append(updated, PinnedRecentItem{
					Route:  cleanRoute,
					Title:  cleanTitle,
					PlugID: plugID,
					ViewID: viewID,
					Icon:   iconName,
				})
			}

			if err := h.savePinnedRecentsForStaff(r.Context(), targetStaffID, updated); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{
					"ok":         false,
					"error_code": err.Error(),
				})
				return
			}

			writeJSON(w, http.StatusOK, map[string]any{
				"ok":             true,
				"staff_id":       targetStaffID,
				"pinned_recents": updated,
			})

		case "unpin":
			cleanRoute, _, _, _, routeErr := parseAndValidatePinnedRoute(in.Route)
			if routeErr != nil {
				writeJSON(w, http.StatusBadRequest, map[string]any{
					"ok":         false,
					"error_code": routeErr.Error(),
				})
				return
			}

			existing := h.loadPinnedRecentsForStaff(r.Context(), targetStaffID)
			updated := make([]PinnedRecentItem, 0, len(existing))
			for _, item := range existing {
				if item.Route != cleanRoute {
					updated = append(updated, item)
				}
			}

			if err := h.savePinnedRecentsForStaff(r.Context(), targetStaffID, updated); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{
					"ok":         false,
					"error_code": err.Error(),
				})
				return
			}

			writeJSON(w, http.StatusOK, map[string]any{
				"ok":             true,
				"staff_id":       targetStaffID,
				"pinned_recents": updated,
			})

		case "clear":
			if err := h.savePinnedRecentsForStaff(r.Context(), targetStaffID, nil); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{
					"ok":         false,
					"error_code": err.Error(),
				})
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{
				"ok":             true,
				"staff_id":       targetStaffID,
				"pinned_recents": []PinnedRecentItem{},
			})

		default:
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": fmt.Sprintf("INVALID_ACTION: unsupported pinned-recents action %q", in.Action),
			})
		}
	})

	mux.HandleFunc("POST /api/auth/logout", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:     "mis_session",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
		})
		h.logInfo("User session signed out via POST /api/auth/logout")
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":         true,
			"logged_out": true,
		})
	})
}
