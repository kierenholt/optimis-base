package host

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"optimis/internal/dal"
)

func readPluginFromDisk(dirPath string, folder string) (*LoadedPlugin, error) {
	manifestPath := filepath.Join(dirPath, "manifest.json")
	manifestStat, err := os.Stat(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat manifest.json: %w", err)
	}
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest.json: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest.json: %w", err)
	}
	if strings.TrimSpace(manifest.ID) == "" {
		return nil, errors.New("manifest.json missing mandatory id")
	}
	if strings.TrimSpace(manifest.Version) == "" {
		return nil, errors.New("manifest.json missing mandatory version")
	}
	if strings.TrimSpace(manifest.CoreDALVersion) == "" {
		return nil, errors.New("manifest.json missing mandatory core_dal_version")
	}
	if len(manifest.Endpoints) == 0 {
		return nil, errors.New("manifest.json must declare at least one endpoint")
	}
	if manifest.Database != nil {
		if err := dal.ValidatePluginDatabaseAlias(manifest.Database.Alias); err != nil {
			return nil, err
		}
		if manifest.GDPRHooks == nil ||
			strings.TrimSpace(manifest.GDPRHooks.ExportPupilData) == "" ||
			strings.TrimSpace(manifest.GDPRHooks.PurgePupilData) == "" {
			return nil, fmt.Errorf(
				"manifest.json for plugin %s declares database.alias (%s) and must declare mandatory gdpr_hooks (export_pupil_data and purge_pupil_data)",
				manifest.ID,
				manifest.Database.Alias,
			)
		}
	}

	if len(manifest.Views) == 0 {
		manifest.Views = defaultPluginViews()
	} else {
		seenViewIDs := make(map[string]struct{}, len(manifest.Views))
		for idx, v := range manifest.Views {
			vid := strings.TrimSpace(v.ID)
			vpath := strings.TrimSpace(v.Path)
			ventry := strings.TrimSpace(v.Entry)
			if vid == "" || vpath == "" || ventry == "" {
				return nil, fmt.Errorf("manifest.json views[%d] for plugin %s is missing mandatory id, path, or entry", idx, manifest.ID)
			}
			if !strings.HasPrefix(vpath, "/") {
				return nil, fmt.Errorf("manifest.json view %q for plugin %s has invalid path %q: must start with '/'", vid, manifest.ID, vpath)
			}
			cleanVPath := "/" + strings.Trim(vpath, "/")
			if cleanVPath == "/ui" || strings.HasPrefix(cleanVPath, "/ui/") {
				return nil, fmt.Errorf("RESERVED_PLUGIN_VIEW_PATH: manifest.json view %q for plugin %s has path %q which collides with reserved /ui asset namespace", vid, manifest.ID, vpath)
			}
			if _, dup := seenViewIDs[vid]; dup {
				return nil, fmt.Errorf("manifest.json for plugin %s declares duplicate view id %q", manifest.ID, vid)
			}
			seenViewIDs[vid] = struct{}{}

			if len(v.QueryParams) == 0 && len(v.AllowedQueryParams) > 0 {
				manifest.Views[idx].QueryParams = append([]string(nil), v.AllowedQueryParams...)
			}
			cleanedKeywords := make([]string, 0, len(v.Keywords))
			for _, kw := range v.Keywords {
				trimmedKW := strings.TrimSpace(kw)
				if trimmedKW == "" {
					return nil, fmt.Errorf("manifest.json view %q for plugin %s contains an empty keyword", vid, manifest.ID)
				}
				cleanedKeywords = append(cleanedKeywords, trimmedKW)
			}
			manifest.Views[idx].Keywords = cleanedKeywords

			relEntry, err := normalizeWidgetEntryRelPath(ventry)
			if err != nil {
				return nil, fmt.Errorf("manifest.json view %q for plugin %s has invalid entry %q: %w", vid, manifest.ID, ventry, err)
			}
			uiDistDir := filepath.Join(dirPath, "ui", "dist")
			if distInfo, distErr := os.Stat(uiDistDir); distErr == nil && distInfo.IsDir() {
				viewFullPath := filepath.Join(uiDistDir, relEntry)
				if st, statErr := os.Stat(viewFullPath); statErr != nil || st.IsDir() {
					return nil, fmt.Errorf("manifest.json view %q entry file not found at %s", vid, viewFullPath)
				}
			}
		}
	}

	if len(manifest.Widgets) > 0 {
		seenWidgetIDs := make(map[string]struct{}, len(manifest.Widgets))
		for idx, w := range manifest.Widgets {
			wid := strings.TrimSpace(w.ID)
			wtitle := strings.TrimSpace(w.Title)
			wentry := strings.TrimSpace(w.Entry)
			if wid == "" || wtitle == "" || wentry == "" {
				return nil, fmt.Errorf("manifest.json widget[%d] for plugin %s is missing mandatory id, title, or entry", idx, manifest.ID)
			}
			if w.Width != 1 && w.Width != 2 {
				return nil, fmt.Errorf("manifest.json widget %q for plugin %s has invalid preset width %d: must be 1 or 2", wid, manifest.ID, w.Width)
			}
			if w.Height != 1 && w.Height != 2 {
				return nil, fmt.Errorf("manifest.json widget %q for plugin %s has invalid preset height %d: must be 1 or 2", wid, manifest.ID, w.Height)
			}
			wdesc := strings.TrimSpace(w.Description)
			if wdesc == "" {
				return nil, fmt.Errorf("MISSING_WIDGET_DESCRIPTION: manifest.json widget %q for plugin %s is missing mandatory description", wid, manifest.ID)
			}
			manifest.Widgets[idx].Description = wdesc
			if _, dup := seenWidgetIDs[wid]; dup {
				return nil, fmt.Errorf("manifest.json for plugin %s declares duplicate widget id %q", manifest.ID, wid)
			}
			seenWidgetIDs[wid] = struct{}{}

			cleanedKeywords := make([]string, 0, len(w.Keywords))
			for _, kw := range w.Keywords {
				trimmedKW := strings.TrimSpace(kw)
				if trimmedKW == "" {
					return nil, fmt.Errorf("manifest.json widget %q for plugin %s contains an empty keyword", wid, manifest.ID)
				}
				cleanedKeywords = append(cleanedKeywords, trimmedKW)
			}
			manifest.Widgets[idx].Keywords = cleanedKeywords

			relEntry, err := normalizeWidgetEntryRelPath(wentry)
			if err != nil {
				return nil, fmt.Errorf("manifest.json widget %q for plugin %s has invalid entry %q: %w", wid, manifest.ID, wentry, err)
			}
			uiDistDir := filepath.Join(dirPath, "ui", "dist")
			if distInfo, distErr := os.Stat(uiDistDir); distErr == nil && distInfo.IsDir() {
				widgetFullPath := filepath.Join(uiDistDir, relEntry)
				if st, statErr := os.Stat(widgetFullPath); statErr != nil || st.IsDir() {
					return nil, fmt.Errorf("manifest.json widget %q entry file not found at %s", wid, widgetFullPath)
				}
			}
		}
	}

	luaPath := filepath.Join(dirPath, "src", "init.lua")
	luaStat, err := os.Stat(luaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat src/init.lua: %w", err)
	}
	luaBytes, err := os.ReadFile(luaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read src/init.lua: %w", err)
	}

	luaSource := string(luaBytes)
	if err := validateManifestWorkersAndLuaSource(manifest, luaSource); err != nil {
		return nil, err
	}

	return &LoadedPlugin{
		Manifest:        manifest,
		LuaSource:       luaSource,
		DirPath:         dirPath,
		Folder:          folder,
		State:           PluginStateActive,
		ManifestModTime: manifestStat.ModTime(),
		LuaModTime:      luaStat.ModTime(),
	}, nil
}

func defaultPluginViews() []ViewSpec {
	commonParams := []string{
		"tab",
		"view",
		"pupil_id",
		"second_pupil_id",
		"group_id",
		"other_group_id",
		"absent_staff_id",
		"cover_staff_id",
		"slot_id",
		"reference_date",
		"academic_year_id",
	}
	return []ViewSpec{
		{
			ID:          "home",
			Title:       "Workspace Overview",
			Path:        "/",
			Entry:       "ui/dist/index.html",
			QueryParams: commonParams,
			Keywords:    []string{"workspace", "overview", "home", "dashboard"},
		},
		{
			ID:          "student",
			Title:       "Student View",
			Path:        "/student/:pupil_id",
			Entry:       "ui/dist/index.html",
			QueryParams: commonParams,
			Keywords:    []string{"student", "pupil", "learner", "profile"},
		},
		{
			ID:          "group",
			Title:       "Group View",
			Path:        "/group/:group_id",
			Entry:       "ui/dist/index.html",
			QueryParams: commonParams,
			Keywords:    []string{"group", "class", "set", "roster"},
		},
		{
			ID:          "staff",
			Title:       "Staff View",
			Path:        "/staff/:staff_id",
			Entry:       "ui/dist/index.html",
			QueryParams: commonParams,
			Keywords:    []string{"staff", "teacher", "colleague"},
		},
	}
}

var opaquePlugIDPattern = regexp.MustCompile(`^[a-z0-9]{8}$`)

func generateRandomPlugID() (string, error) {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	buf := make([]byte, 8)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return "", fmt.Errorf("failed to generate opaque plug_id: %w", err)
	}
	out := make([]byte, 8)
	for i, b := range buf {
		out[i] = chars[int(b)%len(chars)]
	}
	return string(out), nil
}

func (h *Host) ensurePluginInstanceID(ctx context.Context, manifestID string) (string, error) {
	trimmedManifestID := strings.TrimSpace(manifestID)
	if trimmedManifestID == "" {
		return "", errors.New("missing mandatory manifestID for instance plug_id")
	}
	configKey := "plugin_instance_id." + trimmedManifestID

	var existing string
	err := h.coreDAL.ReadDB().QueryRowContext(
		ctx,
		`SELECT value FROM __system_config WHERE key = ?`,
		configKey,
	).Scan(&existing)
	if err == nil {
		trimmedExisting := strings.TrimSpace(existing)
		if opaquePlugIDPattern.MatchString(trimmedExisting) {
			return trimmedExisting, nil
		}
	}

	for attempt := 0; attempt < 16; attempt++ {
		candidate, genErr := generateRandomPlugID()
		if genErr != nil {
			return "", genErr
		}
		var conflictKey string
		cErr := h.coreDAL.ReadDB().QueryRowContext(
			ctx,
			`SELECT key FROM __system_config WHERE key LIKE 'plugin_instance_id.%' AND value = ? AND key != ? LIMIT 1`,
			candidate,
			configKey,
		).Scan(&conflictKey)
		if cErr != nil && !errors.Is(cErr, sql.ErrNoRows) {
			return "", fmt.Errorf("failed to check plug_id collision: %w", cErr)
		}
		if conflictKey != "" {
			continue
		}

		_, execErr := h.coreDAL.WriteDB().ExecContext(
			ctx,
			`INSERT INTO __system_config (key, value, updated_at)
			 VALUES (?, ?, CURRENT_TIMESTAMP)
			 ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = CURRENT_TIMESTAMP`,
			configKey,
			candidate,
		)
		if execErr != nil {
			return "", fmt.Errorf("failed to persist opaque plug_id in __system_config: %w", execErr)
		}
		return candidate, nil
	}
	return "", errors.New("failed to allocate unique opaque plug_id after 16 attempts")
}

func normalizeWidgetEntryRelPath(rawEntry string) (string, error) {
	trimmed := strings.TrimSpace(rawEntry)
	if trimmed == "" {
		return "", errors.New("empty widget entry path")
	}
	trimmed = strings.TrimPrefix(trimmed, "/")
	if strings.HasPrefix(trimmed, "ui/dist/") {
		trimmed = strings.TrimPrefix(trimmed, "ui/dist/")
	} else if strings.HasPrefix(trimmed, "ui/") {
		trimmed = strings.TrimPrefix(trimmed, "ui/")
	}
	clean := filepath.Clean(trimmed)
	if clean == "." || clean == "" || strings.HasPrefix(clean, "..") {
		return "", fmt.Errorf("disallowed widget entry path: %s", rawEntry)
	}
	return clean, nil
}

func (h *Host) LoadPlugin(pluginFolder string) error {
	loaded, err := h.InstallPlugin(context.Background(), pluginFolder)
	if err != nil {
		return err
	}
	if err := h.ActivatePlugin(loaded.PlugID); err != nil {
		return err
	}
	h.logInfo(fmt.Sprintf("Loaded plugin %s (%s) v%s", loaded.Manifest.ID, loaded.PlugID, loaded.Manifest.Version))
	return nil
}

func (h *Host) GetLoadedPluginByPlugID(plugID string) (*LoadedPlugin, bool) {
	trimmed := strings.TrimSpace(plugID)
	if trimmed == "" {
		return nil, false
	}
	h.mu.RLock()
	var matchedManifestID string
	for mID, candidate := range h.plugins {
		if candidate.PlugID == trimmed {
			matchedManifestID = mID
			break
		}
	}
	h.mu.RUnlock()
	if matchedManifestID == "" {
		return nil, false
	}
	return h.GetLoadedPlugin(matchedManifestID)
}

func (h *Host) GetLoadedPlugin(pluginID string) (*LoadedPlugin, bool) {
	trimmed := strings.TrimSpace(pluginID)
	if trimmed == "" {
		return nil, false
	}
	h.mu.RLock()
	p, ok := h.plugins[trimmed]
	canonicalKey := trimmed
	if !ok {
		for mID, candidate := range h.plugins {
			if candidate.PlugID == trimmed {
				p = candidate
				canonicalKey = mID
				ok = true
				break
			}
		}
	}
	h.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if p.IsSyntheticClone {
		return p, true
	}

	manifestStat, mErr := os.Stat(filepath.Join(p.DirPath, "manifest.json"))
	luaStat, lErr := os.Stat(filepath.Join(p.DirPath, "src", "init.lua"))
	if mErr == nil && lErr == nil {
		if !manifestStat.ModTime().Equal(p.ManifestModTime) || !luaStat.ModTime().Equal(p.LuaModTime) {
			if reloaded, err := readPluginFromDisk(p.DirPath, p.Folder); err == nil {
				reloaded.PlugID = p.PlugID
				reloaded.State = p.State
				if migErr := h.provisionAndMigratePluginDB(context.Background(), reloaded); migErr == nil {
					h.mu.Lock()
					h.plugins[canonicalKey] = reloaded
					h.mu.Unlock()
					if reloaded.State == PluginStateActive {
						h.startOrReplacePluginWorkerPool(reloaded)
					}
					h.logInfo(fmt.Sprintf("Hot-reloaded plugin %s v%s from disk", reloaded.Manifest.ID, reloaded.Manifest.Version))
					return reloaded, true
				}
			}
		}
	}

	return p, true
}
