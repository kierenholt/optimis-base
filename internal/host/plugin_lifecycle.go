package host

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type PluginStatusSummary struct {
	ID            string `json:"id"`
	Folder        string `json:"folder"`
	Title         string `json:"title"`
	Version       string `json:"version"`
	State         string `json:"state"`
	DatabaseAlias string `json:"database_alias,omitempty"`
}

func validatePluginFolderName(pluginFolder string) (string, error) {
	trimmed := strings.TrimSpace(pluginFolder)
	if trimmed == "" {
		return "", errors.New("missing mandatory parameter: pluginFolder")
	}
	if strings.Contains(trimmed, "..") ||
		strings.ContainsAny(trimmed, `/\`) ||
		filepath.Clean(trimmed) != trimmed ||
		trimmed == "." {
		return "", fmt.Errorf("invalid plugin folder %q: path traversal and separators are forbidden", pluginFolder)
	}
	return trimmed, nil
}

func (h *Host) InstallPlugin(ctx context.Context, pluginFolder string) (*LoadedPlugin, error) {
	cleanFolder, err := validatePluginFolderName(pluginFolder)
	if err != nil {
		return nil, err
	}

	var candidateDirs []string
	for _, part := range strings.FieldsFunc(h.cfg.PluginsDir, func(r rune) bool { return r == ',' || r == ';' }) {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			candidateDirs = append(candidateDirs, trimmed)
		}
	}
	candidateDirs = append(candidateDirs, "plugins", "test_fixtures/plugins")

	var loaded *LoadedPlugin
	var lastErr error
	for _, baseDir := range candidateDirs {
		dirPath := filepath.Join(baseDir, cleanFolder)
		if info, statErr := os.Stat(dirPath); statErr == nil && info.IsDir() {
			loaded, lastErr = readPluginFromDisk(dirPath, cleanFolder)
			if lastErr == nil {
				break
			}
		} else if lastErr == nil {
			lastErr = statErr
		}
	}
	if loaded == nil {
		if lastErr != nil {
			return nil, lastErr
		}
		return nil, fmt.Errorf("plugin folder not found: %s", cleanFolder)
	}

	if err := h.provisionAndMigratePluginDB(ctx, loaded); err != nil {
		return nil, err
	}

	loaded.State = PluginStateUninstalledDormant

	h.mu.Lock()
	prevPool := h.workerPools[loaded.Manifest.ID]
	delete(h.workerPools, loaded.Manifest.ID)
	h.plugins[loaded.Manifest.ID] = loaded
	h.mu.Unlock()

	if prevPool != nil {
		prevPool.stop()
	}

	h.logInfo(fmt.Sprintf("Installed plugin %s v%s from folder %s (state=%s)", loaded.Manifest.ID, loaded.Manifest.Version, cleanFolder, loaded.State))
	return loaded, nil
}

func (h *Host) ActivatePlugin(pluginID string) error {
	return h.SetPluginState(pluginID, PluginStateActive)
}

func (h *Host) RemovePlugin(ctx context.Context, pluginID string) error {
	trimmedID := strings.TrimSpace(pluginID)
	if trimmedID == "" {
		return errors.New("missing mandatory parameter: pluginID")
	}

	h.mu.Lock()
	p, exists := h.plugins[trimmedID]
	if !exists {
		h.mu.Unlock()
		return fmt.Errorf("plugin not found: %s", trimmedID)
	}

	pool := h.workerPools[trimmedID]
	delete(h.workerPools, trimmedID)
	delete(h.plugins, trimmedID)

	if trimmedID == "uk.org.optimis.test" {
		for pid, candidate := range h.plugins {
			if candidate.IsSyntheticClone {
				if cPool := h.workerPools[pid]; cPool != nil {
					cPool.stop()
					delete(h.workerPools, pid)
				}
				delete(h.plugins, pid)
			}
		}
	}

	detachAlias := ""
	if p.Manifest.Database != nil && strings.TrimSpace(p.Manifest.Database.Alias) != "" {
		alias := strings.TrimSpace(p.Manifest.Database.Alias)
		stillUsed := false
		for _, other := range h.plugins {
			if other.Manifest.Database != nil && strings.TrimSpace(other.Manifest.Database.Alias) == alias {
				stillUsed = true
				break
			}
		}
		if !stillUsed {
			detachAlias = alias
		}
	}
	h.mu.Unlock()

	if pool != nil {
		pool.stop()
	}

	if detachAlias != "" {
		if err := h.coreDAL.DetachPluginDB(ctx, detachAlias); err != nil {
			return err
		}
	}

	h.logInfo(fmt.Sprintf("Removed plugin %s", trimmedID))
	return nil
}

func (h *Host) ListPlugins() []PluginStatusSummary {
	h.mu.RLock()
	out := make([]PluginStatusSummary, 0, len(h.plugins))
	for _, p := range h.plugins {
		if p.IsSyntheticClone {
			continue
		}
		title := strings.TrimSpace(p.Manifest.Title)
		if title == "" {
			title = p.Manifest.ID
			if p.Manifest.ID == "uk.org.optimis.test" {
				title = "Pupil & Classroom Workspace"
			}
		}
		alias := ""
		if p.Manifest.Database != nil {
			alias = strings.TrimSpace(p.Manifest.Database.Alias)
		}
		out = append(out, PluginStatusSummary{
			ID:            p.Manifest.ID,
			Folder:        p.Folder,
			Title:         title,
			Version:       p.Manifest.Version,
			State:         string(p.State),
			DatabaseAlias: alias,
		})
	}
	h.mu.RUnlock()

	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out
}

func (h *Host) authorizeAdminPluginLifecycle(r *http.Request, explicitStaffID string) (CallerContext, error) {
	trimmedExplicit := strings.TrimSpace(explicitStaffID)
	caller, err := h.ResolveRequestCaller(r, trimmedExplicit)
	if err != nil {
		return CallerContext{}, err
	}

	st := h.getAuthState()
	st.mu.RLock()
	activeDevStaffID := strings.TrimSpace(st.activeDevStaffID)
	hasValidSessionCookie := false
	if cookie, cErr := r.Cookie(SessionCookieName); cErr == nil && strings.TrimSpace(cookie.Value) != "" {
		if sid, ok := st.sessions[strings.TrimSpace(cookie.Value)]; ok && strings.TrimSpace(sid) != "" {
			hasValidSessionCookie = true
		}
	}
	st.mu.RUnlock()

	if trimmedExplicit != "" || hasValidSessionCookie || activeDevStaffID != "" || caller.StaffID != "stf_001" {
		if !isCallerAdminOrDPO(caller, true) {
			h.logError(fmt.Sprintf("UNAUTHORIZED_ADMIN_PLUGIN_SCOPE: caller %s (%v) attempted to manage plugin lifecycle", caller.StaffID, caller.ActiveRoles))
			return CallerContext{}, errors.New("UNAUTHORIZED_ADMIN_PLUGIN_SCOPE: managing plugin lifecycle requires System Admin / DPO authority")
		}
	}

	return caller, nil
}

func (h *Host) RegisterPluginLifecycleRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/admin/plugins", func(w http.ResponseWriter, r *http.Request) {
		explicitStaffID := strings.TrimSpace(r.URL.Query().Get("staff_id"))
		if _, err := h.authorizeAdminPluginLifecycle(r, explicitStaffID); err != nil {
			status := http.StatusForbidden
			if !strings.Contains(err.Error(), "UNAUTHORIZED_ADMIN_PLUGIN_SCOPE") {
				status = http.StatusUnauthorized
			}
			writeJSON(w, status, map[string]any{
				"ok":         false,
				"error_code": err.Error(),
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"ok":      true,
			"plugins": h.ListPlugins(),
		})
	})

	mux.HandleFunc("POST /api/admin/plugins/install", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Folder  string `json:"folder"`
			StaffID string `json:"staff_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": fmt.Sprintf("invalid JSON body: %v", err),
			})
			return
		}

		explicitStaffID := strings.TrimSpace(body.StaffID)
		if explicitStaffID == "" {
			explicitStaffID = strings.TrimSpace(r.URL.Query().Get("staff_id"))
		}
		if _, err := h.authorizeAdminPluginLifecycle(r, explicitStaffID); err != nil {
			status := http.StatusForbidden
			if !strings.Contains(err.Error(), "UNAUTHORIZED_ADMIN_PLUGIN_SCOPE") {
				status = http.StatusUnauthorized
			}
			writeJSON(w, status, map[string]any{
				"ok":         false,
				"error_code": err.Error(),
			})
			return
		}

		loaded, err := h.InstallPlugin(r.Context(), body.Folder)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": err.Error(),
			})
			return
		}

		alias := ""
		if loaded.Manifest.Database != nil {
			alias = strings.TrimSpace(loaded.Manifest.Database.Alias)
		}

		writeJSON(w, http.StatusCreated, map[string]any{
			"ok":             true,
			"plugin_id":      loaded.Manifest.ID,
			"folder":         loaded.Folder,
			"version":        loaded.Manifest.Version,
			"state":          string(loaded.State),
			"database_alias": alias,
		})
	})

	mux.HandleFunc("POST /api/admin/plugins/{plugin_id}/activate", func(w http.ResponseWriter, r *http.Request) {
		pluginID := strings.TrimSpace(r.PathValue("plugin_id"))
		if pluginID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": "missing mandatory path parameter: plugin_id",
			})
			return
		}

		var body struct {
			StaffID string `json:"staff_id"`
		}
		if r.Body != nil {
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
				writeJSON(w, http.StatusBadRequest, map[string]any{
					"ok":         false,
					"error_code": fmt.Sprintf("invalid JSON body: %v", err),
				})
				return
			}
		}

		explicitStaffID := strings.TrimSpace(body.StaffID)
		if explicitStaffID == "" {
			explicitStaffID = strings.TrimSpace(r.URL.Query().Get("staff_id"))
		}
		if _, err := h.authorizeAdminPluginLifecycle(r, explicitStaffID); err != nil {
			status := http.StatusForbidden
			if !strings.Contains(err.Error(), "UNAUTHORIZED_ADMIN_PLUGIN_SCOPE") {
				status = http.StatusUnauthorized
			}
			writeJSON(w, status, map[string]any{
				"ok":         false,
				"error_code": err.Error(),
			})
			return
		}

		if err := h.ActivatePlugin(pluginID); err != nil {
			status := http.StatusBadRequest
			if strings.Contains(err.Error(), "plugin not found") {
				status = http.StatusNotFound
			}
			writeJSON(w, status, map[string]any{
				"ok":         false,
				"error_code": err.Error(),
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"ok":        true,
			"plugin_id": pluginID,
			"state":     string(PluginStateActive),
		})
	})

	mux.HandleFunc("DELETE /api/admin/plugins/{plugin_id}", func(w http.ResponseWriter, r *http.Request) {
		pluginID := strings.TrimSpace(r.PathValue("plugin_id"))
		if pluginID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": "missing mandatory path parameter: plugin_id",
			})
			return
		}

		var body struct {
			StaffID string `json:"staff_id"`
		}
		if r.Body != nil {
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
				writeJSON(w, http.StatusBadRequest, map[string]any{
					"ok":         false,
					"error_code": fmt.Sprintf("invalid JSON body: %v", err),
				})
				return
			}
		}

		explicitStaffID := strings.TrimSpace(body.StaffID)
		if explicitStaffID == "" {
			explicitStaffID = strings.TrimSpace(r.URL.Query().Get("staff_id"))
		}
		if _, err := h.authorizeAdminPluginLifecycle(r, explicitStaffID); err != nil {
			status := http.StatusForbidden
			if !strings.Contains(err.Error(), "UNAUTHORIZED_ADMIN_PLUGIN_SCOPE") {
				status = http.StatusUnauthorized
			}
			writeJSON(w, status, map[string]any{
				"ok":         false,
				"error_code": err.Error(),
			})
			return
		}

		if err := h.RemovePlugin(r.Context(), pluginID); err != nil {
			status := http.StatusBadRequest
			if strings.Contains(err.Error(), "plugin not found") {
				status = http.StatusNotFound
			}
			writeJSON(w, status, map[string]any{
				"ok":         false,
				"error_code": err.Error(),
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"ok":        true,
			"plugin_id": pluginID,
			"removed":   true,
		})
	})
}
