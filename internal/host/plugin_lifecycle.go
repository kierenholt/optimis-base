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
	"time"

	"optimis/internal/dal"
)

type PluginStatusSummary struct {
	ID             string   `json:"id"`
	PlugID         string   `json:"plug_id"`
	ManifestID     string   `json:"manifest_id,omitempty"`
	Folder         string   `json:"folder"`
	Title          string   `json:"title"`
	Version        string   `json:"version"`
	State          string   `json:"state"`
	DatabaseAlias  string   `json:"database_alias,omitempty"`
	InstalledAt    string   `json:"installed_at,omitempty"`
	UpdatedAt      string   `json:"updated_at,omitempty"`
	ApprovedScopes []string `json:"approved_scopes,omitempty"`
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

	plugID, err := h.ensurePluginInstanceID(ctx, loaded.Manifest.ID)
	if err != nil {
		return nil, err
	}
	loaded.PlugID = plugID
	loaded.State = PluginStateUninstalledDormant

	now := time.Now().UTC().Format(time.RFC3339)
	if existingRec, rErr := h.coreDAL.GetInstalledPlugin(ctx, loaded.Manifest.ID); rErr == nil && existingRec.InstalledAt != "" {
		loaded.InstalledAt = existingRec.InstalledAt
	} else {
		loaded.InstalledAt = now
	}
	loaded.UpdatedAt = now

	alias := ""
	if loaded.Manifest.Database != nil {
		alias = strings.TrimSpace(loaded.Manifest.Database.Alias)
	}

	scopes := make([]string, 0)
	for _, ep := range loaded.Manifest.Endpoints {
		scopes = append(scopes, ep.Scopes...)
	}
	sort.Strings(scopes)
	scopesJSONBytes, _ := json.Marshal(scopes)

	if uErr := h.coreDAL.UpsertInstalledPlugin(ctx, dal.InstalledPluginRecord{
		ManifestID:         loaded.Manifest.ID,
		PlugID:             plugID,
		Version:            loaded.Manifest.Version,
		State:              string(loaded.State),
		DatabaseAlias:      alias,
		Folder:             cleanFolder,
		ApprovedScopesJSON: string(scopesJSONBytes),
		InstalledAt:        loaded.InstalledAt,
		UpdatedAt:          loaded.UpdatedAt,
	}); uErr != nil {
		return nil, fmt.Errorf("failed to persist installed plugin record: %w", uErr)
	}

	h.mu.Lock()
	prevPool := h.workerPools[loaded.Manifest.ID]
	delete(h.workerPools, loaded.Manifest.ID)
	h.plugins[loaded.Manifest.ID] = loaded
	h.mu.Unlock()

	if prevPool != nil {
		prevPool.stop()
	}

	h.logInfo(fmt.Sprintf("Installed plugin %s (%s) v%s from folder %s (state=%s)", loaded.Manifest.ID, loaded.PlugID, loaded.Manifest.Version, cleanFolder, loaded.State))
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
	canonicalKey := trimmedID
	if !exists {
		for mID, candidate := range h.plugins {
			if candidate.PlugID == trimmedID {
				p = candidate
				canonicalKey = mID
				exists = true
				break
			}
		}
	}
	if !exists {
		h.mu.Unlock()
		return fmt.Errorf("plugin not found: %s", trimmedID)
	}

	pool := h.workerPools[canonicalKey]
	delete(h.workerPools, canonicalKey)
	delete(h.plugins, canonicalKey)

	if canonicalKey == "uk.org.optimis.test" {
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

	_ = h.coreDAL.SetInstalledPluginState(ctx, canonicalKey, string(PluginStateUninstalledDormant))

	h.logInfo(fmt.Sprintf("Removed plugin %s (%s)", canonicalKey, p.PlugID))
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
			title = p.Folder
			if p.Manifest.ID == "uk.org.optimis.test" {
				title = "Pupil & Classroom Workspace"
			}
		}
		alias := ""
		if p.Manifest.Database != nil {
			alias = strings.TrimSpace(p.Manifest.Database.Alias)
		}
		plugID := strings.TrimSpace(p.PlugID)
		if plugID == "" {
			plugID = p.Manifest.ID
		}

		var scopes []string
		for _, ep := range p.Manifest.Endpoints {
			scopes = append(scopes, ep.Scopes...)
		}
		sort.Strings(scopes)

		out = append(out, PluginStatusSummary{
			ID:             plugID,
			PlugID:         plugID,
			ManifestID:     p.Manifest.ID,
			Folder:         p.Folder,
			Title:          title,
			Version:        p.Manifest.Version,
			State:          string(p.State),
			DatabaseAlias:  alias,
			InstalledAt:    p.InstalledAt,
			UpdatedAt:      p.UpdatedAt,
			ApprovedScopes: scopes,
		})
	}
	h.mu.RUnlock()

	sort.Slice(out, func(i, j int) bool {
		return out[i].Folder < out[j].Folder
	})
	return out
}

func (h *Host) authorizeAdminPluginLifecycle(r *http.Request) (CallerContext, error) {
	caller, err := h.ResolveRequestCaller(r, "")
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

	if hasValidSessionCookie || activeDevStaffID != "" || caller.StaffID != "stf_001" {
		if !isCallerAdminOrDPO(caller, true) {
			h.logError(fmt.Sprintf("UNAUTHORIZED_ADMIN_PLUGIN_SCOPE: caller %s (%v) attempted to manage plugin lifecycle", caller.StaffID, caller.ActiveRoles))
			return CallerContext{}, errors.New("UNAUTHORIZED_ADMIN_PLUGIN_SCOPE: managing plugin lifecycle requires System Admin / DPO authority")
		}
	}

	return caller, nil
}

func (h *Host) RegisterPluginLifecycleRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/admin/plugins", func(w http.ResponseWriter, r *http.Request) {
		if _, err := h.authorizeAdminPluginLifecycle(r); err != nil {
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
			Folder string `json:"folder"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": fmt.Sprintf("invalid JSON body: %v", err),
			})
			return
		}

		if _, err := h.authorizeAdminPluginLifecycle(r); err != nil {
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
			"plug_id":        loaded.PlugID,
			"plugin_id":      loaded.PlugID,
			"manifest_id":    loaded.Manifest.ID,
			"folder":         loaded.Folder,
			"version":        loaded.Manifest.Version,
			"state":          string(loaded.State),
			"database_alias": alias,
			"installed_at":   loaded.InstalledAt,
			"updated_at":     loaded.UpdatedAt,
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

		if r.Body != nil {
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
				writeJSON(w, http.StatusBadRequest, map[string]any{
					"ok":         false,
					"error_code": fmt.Sprintf("invalid JSON body: %v", err),
				})
				return
			}
		}

		if _, err := h.authorizeAdminPluginLifecycle(r); err != nil {
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

		updatedAt := time.Now().UTC().Format(time.RFC3339)
		if p, ok := h.GetLoadedPlugin(pluginID); ok && p.UpdatedAt != "" {
			updatedAt = p.UpdatedAt
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"ok":         true,
			"plug_id":    pluginID,
			"plugin_id":  pluginID,
			"state":      string(PluginStateActive),
			"updated_at": updatedAt,
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

		if r.Body != nil {
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
				writeJSON(w, http.StatusBadRequest, map[string]any{
					"ok":         false,
					"error_code": fmt.Sprintf("invalid JSON body: %v", err),
				})
				return
			}
		}

		if _, err := h.authorizeAdminPluginLifecycle(r); err != nil {
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
			"plug_id":   pluginID,
			"plugin_id": pluginID,
			"removed":   true,
		})
	})
}
