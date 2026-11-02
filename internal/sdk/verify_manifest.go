package sdk

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	lua "github.com/yuin/gopher-lua"

	"optimis/internal/host"
)

var (
	manifestIDPattern    = regexp.MustCompile(`^[a-z0-9]+(\.[a-z0-9_-]+)+$`)
	dbAliasPattern       = regexp.MustCompile(`^plugin_[a-z0-9_]+$`)
	dataMisRPCAttrRegex  = regexp.MustCompile(`data-mis-rpc\s*=\s*["']([^"']+)["']`)
	dataMisIconAttrRegex = regexp.MustCompile(`data-mis-icon\s*=\s*["']([^"']+)["']`)
)

func ReadAndValidateManifest(pluginDir string) (*host.Manifest, error) {
	manifestPath := filepath.Join(pluginDir, "manifest.json")
	rawBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("missing or unreadable manifest.json at %s: %w", manifestPath, err)
	}

	var manifest host.Manifest
	dec := json.NewDecoder(bytes.NewReader(rawBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&manifest); err != nil {
		return nil, fmt.Errorf("invalid manifest.json schema in %s: %w", manifestPath, err)
	}

	if !manifestIDPattern.MatchString(strings.TrimSpace(manifest.ID)) {
		return nil, fmt.Errorf("invalid manifest.id %q: must be reverse-DNS identifier", manifest.ID)
	}
	if strings.TrimSpace(manifest.Version) == "" {
		return nil, errors.New("manifest.json missing mandatory field: version")
	}
	if strings.TrimSpace(manifest.CoreDALVersion) == "" {
		return nil, errors.New("manifest.json missing mandatory field: core_dal_version")
	}
	if len(manifest.Endpoints) == 0 {
		return nil, errors.New("manifest.json must declare at least one endpoint in endpoints")
	}
	if !IsValidLucideIcon(manifest.Icon) {
		return nil, fmt.Errorf("UNKNOWN_LUCIDE_ICON: manifest.json for plugin %s declares unknown monochrome Lucide icon %q", manifest.ID, manifest.Icon)
	}

	if manifest.Database != nil && strings.TrimSpace(manifest.Database.Alias) != "" {
		if !dbAliasPattern.MatchString(strings.TrimSpace(manifest.Database.Alias)) {
			return nil, fmt.Errorf("invalid database.alias %q: must match ^plugin_[a-z0-9_]+$", manifest.Database.Alias)
		}
		if manifest.GDPRHooks == nil ||
			strings.TrimSpace(manifest.GDPRHooks.ExportPupilData) == "" ||
			strings.TrimSpace(manifest.GDPRHooks.PurgePupilData) == "" {
			return nil, fmt.Errorf("plugin %s attaches database %s but is missing mandatory gdpr_hooks (export_pupil_data and purge_pupil_data)", manifest.ID, manifest.Database.Alias)
		}
	}

	if len(manifest.Views) > 0 {
		seenViewIDs := make(map[string]struct{}, len(manifest.Views))
		for idx, v := range manifest.Views {
			vid := strings.TrimSpace(v.ID)
			vpath := strings.TrimSpace(v.Path)
			ventry := strings.TrimSpace(v.Entry)
			if vid == "" || vpath == "" || ventry == "" {
				return nil, fmt.Errorf("manifest.json views[%d] for plugin %s is missing mandatory id, path, or entry", idx, manifest.ID)
			}
			if !IsValidLucideIcon(v.Icon) {
				return nil, fmt.Errorf("UNKNOWN_LUCIDE_ICON: manifest.json view %q for plugin %s declares unknown monochrome Lucide icon %q", vid, manifest.ID, v.Icon)
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
			for _, kw := range v.Keywords {
				if strings.TrimSpace(kw) == "" {
					return nil, fmt.Errorf("manifest.json view %q for plugin %s contains an empty keyword", vid, manifest.ID)
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
			if !IsValidLucideIcon(w.Icon) {
				return nil, fmt.Errorf("UNKNOWN_LUCIDE_ICON: manifest.json widget %q for plugin %s declares unknown monochrome Lucide icon %q", wid, manifest.ID, w.Icon)
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
			for _, kw := range w.Keywords {
				if strings.TrimSpace(kw) == "" {
					return nil, fmt.Errorf("manifest.json widget %q for plugin %s contains an empty keyword", wid, manifest.ID)
				}
			}
		}
	}

	return &manifest, nil
}

func verifyManifestLuaAndUI(pluginDir string) (*host.Manifest, error) {
	manifest, err := ReadAndValidateManifest(pluginDir)
	if err != nil {
		return nil, err
	}

	luaPath := filepath.Join(pluginDir, "src", "init.lua")
	luaBytes, err := os.ReadFile(luaPath)
	if err != nil {
		return nil, fmt.Errorf("missing or unreadable src/init.lua at %s: %w", luaPath, err)
	}

	L := newSandboxedLuaState()
	defer L.Close()

	if err := L.DoString(string(luaBytes)); err != nil {
		return nil, fmt.Errorf("syntax or load error in src/init.lua: %w", err)
	}

	for endpointName := range manifest.Endpoints {
		if L.GetGlobal(endpointName).Type() != lua.LTFunction {
			return nil, fmt.Errorf("MANIFEST_LUA_MISMATCH: endpoint %q declared in manifest.json is not a function in src/init.lua", endpointName)
		}
	}

	for workerName := range manifest.Workers {
		if L.GetGlobal(workerName).Type() != lua.LTFunction {
			return nil, fmt.Errorf("MANIFEST_LUA_MISMATCH: worker %q declared in manifest.json is not a function in src/init.lua", workerName)
		}
	}

	if manifest.GDPRHooks != nil {
		for _, hookName := range []string{
			strings.TrimSpace(manifest.GDPRHooks.ExportPupilData),
			strings.TrimSpace(manifest.GDPRHooks.PurgePupilData),
		} {
			if hookName != "" && L.GetGlobal(hookName).Type() != lua.LTFunction {
				return nil, fmt.Errorf("MANIFEST_LUA_MISMATCH: gdpr_hook %q declared in manifest.json is not a function in src/init.lua", hookName)
			}
		}
	}

	uiDir := filepath.Join(pluginDir, "ui")
	if info, statErr := os.Stat(uiDir); statErr == nil && info.IsDir() {
		err = filepath.WalkDir(uiDir, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(path))
			if ext != ".html" && ext != ".js" {
				return nil
			}
			content, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			text := string(content)
			relPath, _ := filepath.Rel(pluginDir, path)

			matches := dataMisRPCAttrRegex.FindAllStringSubmatch(text, -1)
			for _, m := range matches {
				rpcMethod := strings.TrimSpace(m[1])
				if rpcMethod == "" {
					continue
				}
				if _, declared := manifest.Endpoints[rpcMethod]; !declared {
					return fmt.Errorf("UI_MANIFEST_MISMATCH: undeclared data-mis-rpc method %q in %s (not declared in manifest.json endpoints)", rpcMethod, relPath)
				}
			}

			iconMatches := dataMisIconAttrRegex.FindAllStringSubmatch(text, -1)
			for _, m := range iconMatches {
				iconName := strings.TrimSpace(m[1])
				if iconName != "" && !IsValidLucideIcon(iconName) {
					return fmt.Errorf("UNKNOWN_LUCIDE_ICON: undeclared or unsupported data-mis-icon %q in %s (all icons must be monochrome Lucide)", iconName, relPath)
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return manifest, nil
}

func newSandboxedLuaState() *lua.LState {
	L := lua.NewState(lua.Options{SkipOpenLibs: true})
	for _, lib := range []struct {
		name string
		fn   lua.LGFunction
	}{
		{lua.LoadLibName, lua.OpenPackage},
		{lua.BaseLibName, lua.OpenBase},
		{lua.TabLibName, lua.OpenTable},
		{lua.StringLibName, lua.OpenString},
		{lua.MathLibName, lua.OpenMath},
	} {
		L.Push(L.NewFunction(lib.fn))
		L.Push(lua.LString(lib.name))
		L.Call(1, 0)
	}
	L.SetGlobal("dofile", lua.LNil)
	L.SetGlobal("loadfile", lua.LNil)
	L.SetGlobal("package", lua.LNil)
	L.SetGlobal("require", lua.LNil)
	L.SetGlobal("os", lua.LNil)
	L.SetGlobal("io", lua.LNil)
	L.SetGlobal("debug", lua.LNil)
	return L
}
