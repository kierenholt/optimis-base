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
	PlugID      string `json:"plug_id"`
	PluginTitle string `json:"plugin_title,omitempty"`
	WidgetID    string `json:"widget_id"`
	Col         int    `json:"col"`
	Row         int    `json:"row"`
	Permitted   bool   `json:"permitted"`
	ErrorCode   string `json:"error_code,omitempty"`
	ErrorTitle  string `json:"error_title,omitempty"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Icon        string `json:"icon,omitempty"`
	Entry       string `json:"entry,omitempty"`
	EntryURL    string `json:"entry_url,omitempty"`
	Width       int    `json:"width,omitempty"`
	Height      int    `json:"height,omitempty"`
}

func is2DFootprintFree(occupied map[[2]int]bool, col, row, width, height, colCount int) bool {
	if col < 1 || row < 1 || col+width-1 > colCount {
		return false
	}
	for r := row; r < row+height; r++ {
		for c := col; c < col+width; c++ {
			if occupied[[2]int{c, r}] {
				return false
			}
		}
	}
	return true
}

func mark2DFootprintOccupied(occupied map[[2]int]bool, col, row, width, height int) {
	for r := row; r < row+height; r++ {
		for c := col; c < col+width; c++ {
			occupied[[2]int{c, r}] = true
		}
	}
}

func findFirstFree2DCoordinate(occupied map[[2]int]bool, width, height, colCount int) (int, int) {
	effectiveWidth := width
	if effectiveWidth > colCount {
		effectiveWidth = colCount
	}
	if effectiveWidth < 1 {
		effectiveWidth = 1
	}
	effectiveHeight := height
	if effectiveHeight < 1 {
		effectiveHeight = 1
	}
	for r := 1; r <= 200; r++ {
		for c := 1; c <= colCount-effectiveWidth+1; c++ {
			if is2DFootprintFree(occupied, c, r, effectiveWidth, effectiveHeight, colCount) {
				return c, r
			}
		}
	}
	return 1, 1
}

func (h *Host) resolveCallerDashboardState(
	ctx context.Context,
	caller CallerContext,
) ([]dal.DashboardWidgetItem, int, string, []ResolvedDashboardWidgetSlot, error) {
	type permWidgetMeta struct {
		pluginTitle string
		title       string
		description string
		icon        string
		entry       string
		entryURL    string
		width       int
		height      int
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
			wIcon := strings.TrimSpace(wSpec.Icon)
			if wIcon == "" {
				wIcon = strings.TrimSpace(p.Manifest.Icon)
			}
			permittedWidgets[plugID+":"+wSpec.ID] = permWidgetMeta{
				pluginTitle: pTitle,
				title:       wSpec.Title,
				description: wSpec.Description,
				icon:        wIcon,
				entry:       wSpec.Entry,
				entryURL:    fmt.Sprintf("/p/%s/ui/%s", plugID, relEntry),
				width:       wSpec.Width,
				height:      wSpec.Height,
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
				Col:      1,
				Row:      1,
			},
		}
	}

	layout, colCount, source, err := h.coreDAL.ResolveEffectiveDashboardLayoutWithColumns(
		ctx,
		caller.StaffID,
		caller.ActiveRoles,
		fallbackDefault,
	)
	if err != nil {
		return nil, dal.DefaultDashboardColumnCount, "", nil, err
	}

	occupied := make(map[[2]int]bool)
	resolved := make([]ResolvedDashboardWidgetSlot, 0, len(layout))
	for i, item := range layout {
		key := item.PlugID + ":" + item.WidgetID
		if meta, ok := permittedWidgets[key]; ok {
			w := meta.width
			if w != 1 && w != 2 {
				w = 2
			}
			hgt := meta.height
			if hgt != 1 && hgt != 2 {
				hgt = 1
			}
			col, row := item.Col, item.Row
			if !is2DFootprintFree(occupied, col, row, w, hgt, colCount) {
				col, row = findFirstFree2DCoordinate(occupied, w, hgt, colCount)
			}
			mark2DFootprintOccupied(occupied, col, row, w, hgt)
			layout[i].Col = col
			layout[i].Row = row

			resolved = append(resolved, ResolvedDashboardWidgetSlot{
				PlugID:      item.PlugID,
				PluginTitle: meta.pluginTitle,
				WidgetID:    item.WidgetID,
				Col:         col,
				Row:         row,
				Permitted:   true,
				Title:       meta.title,
				Description: meta.description,
				Icon:        meta.icon,
				Entry:       meta.entry,
				EntryURL:    meta.entryURL,
				Width:       w,
				Height:      hgt,
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

		col, row := item.Col, item.Row
		if !is2DFootprintFree(occupied, col, row, 2, 1, colCount) {
			col, row = findFirstFree2DCoordinate(occupied, 2, 1, colCount)
		}
		mark2DFootprintOccupied(occupied, col, row, 2, 1)
		layout[i].Col = col
		layout[i].Row = row

		resolved = append(resolved, ResolvedDashboardWidgetSlot{
			PlugID:     item.PlugID,
			WidgetID:   item.WidgetID,
			Col:        col,
			Row:        row,
			Permitted:  false,
			ErrorCode:  errCode,
			ErrorTitle: errTitle,
			Title:      item.WidgetID,
			Width:      2,
			Height:     1,
		})
	}

	return layout, colCount, source, resolved, nil
}

func (h *Host) RegisterDashboardLayoutRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/dashboard/layout", func(w http.ResponseWriter, r *http.Request) {
		caller, callerErr := h.ResolveRequestCaller(r, "")
		if callerErr != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]any{
				"ok":         false,
				"error_code": callerErr.Error(),
			})
			return
		}

		layout, colCount, source, resolved, err := h.resolveCallerDashboardState(r.Context(), caller)
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
			"column_count":     colCount,
			"layout_source":    source,
			"resolved_widgets": resolved,
			"role_defaults":    roleDefaults,
		})
	})

	handleSaveStaffLayout := func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			StaffID     string                    `json:"staff_id"`
			Layout      []dal.DashboardWidgetItem `json:"layout"`
			ColumnCount *int                      `json:"column_count"`
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

		reqColCount := 0
		if body.ColumnCount != nil {
			reqColCount = *body.ColumnCount
		}

		saved, savedColCount, err := h.coreDAL.SaveStaffDashboardLayoutWithColumns(
			r.Context(),
			targetStaffID,
			body.Layout,
			reqColCount,
			caller.StaffID,
		)
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
		_, _, _, resolved, _ := h.resolveCallerDashboardState(r.Context(), targetCaller)

		h.logInfo(fmt.Sprintf("Saved 1D dashboard layout for staff %s (%d widgets, %d columns)", targetStaffID, len(saved), savedColCount))
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":               true,
			"staff_id":         targetStaffID,
			"layout":           saved,
			"column_count":     savedColCount,
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

		caller, callerErr := h.ResolveRequestCaller(r, "")
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

		caller, callerErr := h.ResolveRequestCaller(r, "")
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
