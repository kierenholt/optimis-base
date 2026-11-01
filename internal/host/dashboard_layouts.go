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

type ResolvedDashboardWidgetSlot struct {
	PlugID     string `json:"plug_id"`
	WidgetID   string `json:"widget_id"`
	Permitted  bool   `json:"permitted"`
	ErrorCode  string `json:"error_code,omitempty"`
	ErrorTitle string `json:"error_title,omitempty"`
	Title      string `json:"title,omitempty"`
	Entry      string `json:"entry,omitempty"`
	EntryURL   string `json:"entry_url,omitempty"`
	Width      int    `json:"width,omitempty"`
	Height     int    `json:"height,omitempty"`
}

func (h *Host) resolveCallerDashboardState(
	ctx context.Context,
	caller CallerContext,
) ([]dal.DashboardWidgetItem, string, []ResolvedDashboardWidgetSlot, error) {
	type permWidgetMeta struct {
		title    string
		entry    string
		entryURL string
		width    int
		height   int
	}

	h.mu.RLock()
	permittedWidgets := make(map[string]permWidgetMeta)
	allPluginStates := make(map[string]PluginState, len(h.plugins))
	allPluginHasWidget := make(map[string]bool)

	type candidateFirst struct {
		isDefaultTest bool
		pluginTitle   string
		plugID        string
		widgetID      string
	}
	var candidates []candidateFirst

	for _, p := range h.plugins {
		plugID := strings.TrimSpace(p.PlugID)
		if plugID == "" {
			plugID = p.Manifest.ID
		}
		allPluginStates[plugID] = p.State
		for _, wSpec := range p.Manifest.Widgets {
			allPluginHasWidget[plugID+":"+wSpec.ID] = true
		}

		if p.State != PluginStateActive {
			continue
		}

		methods := 0
		for _, spec := range p.Manifest.Endpoints {
			if (hasAllScopes(caller.Scopes, spec.Scopes) && hasRequiredEndpointRoles(caller, spec.RequireRoles)) || isCallerAdminOrDPO(caller, false) {
				methods++
			}
		}
		if len(p.Manifest.Endpoints) > 0 && methods == 0 {
			continue
		}

		pTitle := strings.TrimSpace(p.Manifest.Title)
		if pTitle == "" {
			pTitle = p.Folder
		}

		for idx, wSpec := range p.Manifest.Widgets {
			relEntry, err := normalizeWidgetEntryRelPath(wSpec.Entry)
			if err != nil {
				continue
			}
			permittedWidgets[plugID+":"+wSpec.ID] = permWidgetMeta{
				title:    wSpec.Title,
				entry:    wSpec.Entry,
				entryURL: fmt.Sprintf("/p/%s/ui/%s", plugID, relEntry),
				width:    wSpec.Width,
				height:   wSpec.Height,
			}
			if idx == 0 {
				candidates = append(candidates, candidateFirst{
					isDefaultTest: p.Manifest.ID == "uk.org.optimis.test",
					pluginTitle:   pTitle,
					plugID:        plugID,
					widgetID:      wSpec.ID,
				})
			}
		}
	}
	h.mu.RUnlock()

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].isDefaultTest {
			return true
		}
		if candidates[j].isDefaultTest {
			return false
		}
		return candidates[i].pluginTitle < candidates[j].pluginTitle
	})

	var fallbackDefault []dal.DashboardWidgetItem
	if len(candidates) > 0 {
		fallbackDefault = []dal.DashboardWidgetItem{
			{
				PlugID:   candidates[0].plugID,
				WidgetID: candidates[0].widgetID,
			},
		}
	}

	layout, source, err := h.coreDAL.ResolveEffectiveDashboardLayout(
		ctx,
		caller.StaffID,
		caller.ActiveRoles,
		fallbackDefault,
	)
	if err != nil {
		return nil, "", nil, err
	}

	resolved := make([]ResolvedDashboardWidgetSlot, 0, len(layout))
	for _, item := range layout {
		key := item.PlugID + ":" + item.WidgetID
		if meta, ok := permittedWidgets[key]; ok {
			resolved = append(resolved, ResolvedDashboardWidgetSlot{
				PlugID:    item.PlugID,
				WidgetID:  item.WidgetID,
				Permitted: true,
				Title:     meta.title,
				Entry:     meta.entry,
				EntryURL:  meta.entryURL,
				Width:     meta.width,
				Height:    meta.height,
			})
			continue
		}

		errCode := "WIDGET_PERMISSION_DENIED"
		errTitle := "Widget Access Restricted for Active Role"
		if st, exists := allPluginStates[item.PlugID]; exists {
			if st == PluginStateUninstalledDormant {
				errCode = "WIDGET_PLUGIN_REVOKED_DORMANT"
				errTitle = "Plugin Uninstalled / Dormant Quarantine"
			} else if !allPluginHasWidget[key] {
				errCode = "WIDGET_UNAVAILABLE_OR_REVOKED"
				errTitle = "Widget Revoked or Unavailable"
			}
		} else {
			errCode = "WIDGET_PLUGIN_REVOKED"
			errTitle = "Plugin Revoked or Unpermitted"
		}

		resolved = append(resolved, ResolvedDashboardWidgetSlot{
			PlugID:     item.PlugID,
			WidgetID:   item.WidgetID,
			Permitted:  false,
			ErrorCode:  errCode,
			ErrorTitle: errTitle,
			Title:      item.WidgetID,
			Width:      2,
			Height:     1,
		})
	}

	return layout, source, resolved, nil
}

func (h *Host) RegisterDashboardLayoutRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/dashboard/layout", func(w http.ResponseWriter, r *http.Request) {
		caller, callerErr := h.ResolveRequestCaller(r, strings.TrimSpace(r.URL.Query().Get("staff_id")))
		if callerErr != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]any{
				"ok":         false,
				"error_code": callerErr.Error(),
			})
			return
		}

		layout, source, resolved, err := h.resolveCallerDashboardState(r.Context(), caller)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"ok":         false,
				"error_code": err.Error(),
			})
			return
		}

		roleDefaults, _ := h.coreDAL.ListAllRoleDefaultDashboardLayouts(r.Context())

		writeJSON(w, http.StatusOK, map[string]any{
			"ok":               true,
			"staff_id":         caller.StaffID,
			"active_roles":     caller.ActiveRoles,
			"is_admin":         isCallerAdminOrDPO(caller, true),
			"layout":           layout,
			"layout_source":    source,
			"resolved_widgets": resolved,
			"role_defaults":    roleDefaults,
		})
	})

	handleSaveStaffLayout := func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			StaffID string                    `json:"staff_id"`
			Layout  []dal.DashboardWidgetItem `json:"layout"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": fmt.Sprintf("invalid JSON body: %v", err),
			})
			return
		}
		if body.Layout == nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": "missing mandatory field: layout",
			})
			return
		}

		caller, targetStaffID, authStatus, authErr := h.ResolveAuthenticatedCallerAndTargetStaff(r, body.StaffID)
		if authErr != nil {
			h.logError(fmt.Sprintf("Blocked dashboard layout save for target %q by caller %q: %v", body.StaffID, caller.StaffID, authErr))
			writeJSON(w, authStatus, map[string]any{
				"ok":         false,
				"error_code": authErr.Error(),
			})
			return
		}

		saved, err := h.coreDAL.SaveStaffDashboardLayout(r.Context(), targetStaffID, body.Layout, caller.StaffID)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": err.Error(),
			})
			return
		}

		targetCaller := caller
		if targetStaffID != caller.StaffID {
			targetCaller = h.resolveCallerForStaffID(r.Context(), targetStaffID)
		}
		_, _, resolved, _ := h.resolveCallerDashboardState(r.Context(), targetCaller)

		h.logInfo(fmt.Sprintf("Saved 1D dashboard layout for staff %s (%d widgets)", targetStaffID, len(saved)))
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":               true,
			"staff_id":         targetStaffID,
			"layout":           saved,
			"layout_source":    "STAFF",
			"resolved_widgets": resolved,
		})
	}

	mux.HandleFunc("PUT /api/dashboard/layout", handleSaveStaffLayout)
	mux.HandleFunc("POST /api/dashboard/layout", handleSaveStaffLayout)

	handleSaveRoleDefault := func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			StaffID               string                    `json:"staff_id"`
			RoleID                string                    `json:"role_id"`
			Layout                []dal.DashboardWidgetItem `json:"layout"`
			AppendToExistingStaff *bool                     `json:"append_to_existing_staff"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": fmt.Sprintf("invalid JSON body: %v", err),
			})
			return
		}
		if strings.TrimSpace(body.RoleID) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": "missing mandatory field: role_id",
			})
			return
		}
		if body.Layout == nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": "missing mandatory field: layout",
			})
			return
		}

		caller, callerErr := h.ResolveRequestCaller(r, strings.TrimSpace(body.StaffID))
		if callerErr != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]any{
				"ok":         false,
				"error_code": callerErr.Error(),
			})
			return
		}
		if !isCallerAdminOrDPO(caller, true) {
			h.logError(fmt.Sprintf("Blocked non-admin %s from updating role default dashboard layout for %s", caller.StaffID, body.RoleID))
			writeJSON(w, http.StatusForbidden, map[string]any{
				"ok":         false,
				"error_code": "FORBIDDEN_ADMIN_ROLE_REQUIRED",
			})
			return
		}

		appendToStaff := true
		if body.AppendToExistingStaff != nil {
			appendToStaff = *body.AppendToExistingStaff
		}

		saved, appendedCount, err := h.coreDAL.SaveRoleDefaultDashboardLayout(
			r.Context(),
			body.RoleID,
			body.Layout,
			appendToStaff,
			caller.StaffID,
		)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": err.Error(),
			})
			return
		}

		roleDefaults, _ := h.coreDAL.ListAllRoleDefaultDashboardLayouts(r.Context())

		h.logInfo(fmt.Sprintf("Admin %s saved role default dashboard layout for %s (%d widgets, appended to %d staff)", caller.StaffID, strings.ToUpper(strings.TrimSpace(body.RoleID)), len(saved), appendedCount))
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":                   true,
			"role_id":              strings.ToUpper(strings.TrimSpace(body.RoleID)),
			"layout":               saved,
			"appended_staff_count": appendedCount,
			"role_defaults":        roleDefaults,
		})
	}

	mux.HandleFunc("PUT /api/admin/dashboard/role-default", handleSaveRoleDefault)
	mux.HandleFunc("POST /api/admin/dashboard/role-default", handleSaveRoleDefault)

	mux.HandleFunc("POST /api/admin/dashboard/role-default/append", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			StaffID  string `json:"staff_id"`
			RoleID   string `json:"role_id"`
			PlugID   string `json:"plug_id"`
			WidgetID string `json:"widget_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": fmt.Sprintf("invalid JSON body: %v", err),
			})
			return
		}
		if strings.TrimSpace(body.RoleID) == "" || strings.TrimSpace(body.PlugID) == "" || strings.TrimSpace(body.WidgetID) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": "missing mandatory field: role_id, plug_id, or widget_id",
			})
			return
		}

		caller, callerErr := h.ResolveRequestCaller(r, strings.TrimSpace(body.StaffID))
		if callerErr != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]any{
				"ok":         false,
				"error_code": callerErr.Error(),
			})
			return
		}
		if !isCallerAdminOrDPO(caller, true) {
			h.logError(fmt.Sprintf("Blocked non-admin %s from appending widget %s:%s to role %s", caller.StaffID, body.PlugID, body.WidgetID, body.RoleID))
			writeJSON(w, http.StatusForbidden, map[string]any{
				"ok":         false,
				"error_code": "FORBIDDEN_ADMIN_ROLE_REQUIRED",
			})
			return
		}

		updated, appendedCount, err := h.coreDAL.AppendWidgetToRoleDashboardLayout(
			r.Context(),
			body.RoleID,
			dal.DashboardWidgetItem{
				PlugID:   body.PlugID,
				WidgetID: body.WidgetID,
			},
			caller.StaffID,
		)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": err.Error(),
			})
			return
		}

		roleDefaults, _ := h.coreDAL.ListAllRoleDefaultDashboardLayouts(r.Context())

		h.logInfo(fmt.Sprintf("Admin %s appended widget %s:%s to role %s (appended to %d staff)", caller.StaffID, body.PlugID, body.WidgetID, strings.ToUpper(strings.TrimSpace(body.RoleID)), appendedCount))
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":                   true,
			"role_id":              strings.ToUpper(strings.TrimSpace(body.RoleID)),
			"layout":               updated,
			"appended_staff_count": appendedCount,
			"role_defaults":        roleDefaults,
		})
	})

	mux.HandleFunc("DELETE /api/test/dashboard-layouts", func(w http.ResponseWriter, r *http.Request) {
		if err := h.coreDAL.ResetAllDashboardLayoutsForTest(r.Context()); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"ok":    false,
				"error": err.Error(),
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":      true,
			"deleted": true,
		})
	})
}
