package host

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"optimis/internal/dal"
)

// ============================================================================
// System Search Helpers & Handler
// ============================================================================

type searchPermissions struct {
	hasPupilAll bool
	hasPupilOwn bool
	hasGroupAll bool
	hasGroupOwn bool
}

func resolveSearchPermissions(caller CallerContext) searchPermissions {
	perms := searchPermissions{
		hasPupilAll: isCallerAdminOrDPO(caller, false),
		hasGroupAll: isCallerAdminOrDPO(caller, false),
	}

	for _, s := range caller.Scopes {
		switch s {
		case "dal.pupils.read:all":
			perms.hasPupilAll = true
		case "dal.pupils.read:own_groups":
			perms.hasPupilOwn = true
		case "dal.groups.read:all":
			perms.hasGroupAll = true
		case "dal.groups.read:own_groups":
			perms.hasGroupOwn = true
		}
	}

	if perms.hasPupilAll && perms.hasPupilOwn && !callerHasAllRole(caller.ActiveRoles) {
		perms.hasPupilAll = false
	}
	if perms.hasGroupAll && perms.hasGroupOwn && !callerHasAllRole(caller.ActiveRoles) {
		perms.hasGroupAll = false
	}
	return perms
}

func (h *Host) searchGroups(ctx context.Context, perms searchPermissions, baseScope dal.DALQueryScope, qLower string) ([]dal.GroupSummary, []dal.GroupSummary) {
	matchedGroups := make([]dal.GroupSummary, 0)
	var callerOwnGroups []dal.GroupSummary

	if !perms.hasGroupAll && !perms.hasGroupOwn && !perms.hasPupilOwn {
		return matchedGroups, callerOwnGroups
	}

	gMode := dal.ScopeOwnGroups
	if perms.hasGroupAll {
		gMode = dal.ScopeAll
	}
	gScope := baseScope
	gScope.Mode = gMode

	groups, err := h.coreDAL.ListGroupsScoped(ctx, gScope, "")
	if err != nil {
		return matchedGroups, callerOwnGroups
	}

	if gMode == dal.ScopeOwnGroups {
		callerOwnGroups = groups
	}
	if perms.hasGroupAll || perms.hasGroupOwn {
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
	return matchedGroups, callerOwnGroups
}

func (h *Host) searchPupils(ctx context.Context, perms searchPermissions, baseScope dal.DALQueryScope, callerOwnGroups []dal.GroupSummary, staffID, qLower string) []dal.Pupil {
	matchedPupils := make([]dal.Pupil, 0)

	if perms.hasPupilAll {
		pScope := baseScope
		pScope.Mode = dal.ScopeAll
		if pupils, err := h.coreDAL.ListAllPupilsScoped(ctx, pScope); err == nil {
			for _, p := range pupils {
				if matchesPupilSearchQuery(p, qLower) {
					matchedPupils = append(matchedPupils, p)
				}
			}
		}
		return matchedPupils
	}

	if perms.hasPupilOwn && strings.TrimSpace(staffID) != "" {
		if callerOwnGroups == nil {
			ownGScope := baseScope
			ownGScope.Mode = dal.ScopeOwnGroups
			callerOwnGroups, _ = h.coreDAL.ListGroupsScoped(ctx, ownGScope, "")
		}
		pScope := baseScope
		pScope.Mode = dal.ScopeOwnGroups
		seenPupils := make(map[string]bool)
		for _, g := range callerOwnGroups {
			if gPupils, err := h.coreDAL.ListPupilsByGroupScoped(ctx, pScope, g.ID); err == nil {
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

	return matchedPupils
}

func (h *Host) searchPluginViewsAndWidgets(caller CallerContext, qLower string) ([]PluginUIViewDescriptor, []PluginUIWidgetDescriptor) {
	matchedViews := make([]PluginUIViewDescriptor, 0)
	matchedWidgets := make([]PluginUIWidgetDescriptor, 0)

	h.mu.RLock()
	defer h.mu.RUnlock()

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

	return matchedViews, matchedWidgets
}

func (h *Host) handleSystemSearch(w http.ResponseWriter, r *http.Request) {
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

	perms := resolveSearchPermissions(caller)
	matchedGroups, callerOwnGroups := h.searchGroups(r.Context(), perms, baseScope, qLower)
	matchedPupils := h.searchPupils(r.Context(), perms, baseScope, callerOwnGroups, caller.StaffID, qLower)
	matchedViews, matchedWidgets := h.searchPluginViewsAndWidgets(caller, qLower)

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"query":   rawQuery,
		"pupils":  matchedPupils,
		"groups":  matchedGroups,
		"views":   matchedViews,
		"widgets": matchedWidgets,
	})
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
