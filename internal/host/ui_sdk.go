package host

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

// ============================================================================
// Route Pattern Matching & Query Parameter Validation
// ============================================================================

func matchViewPattern(views []ViewSpec, targetSegs []string) (*ViewSpec, map[string]string) {
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
			return v, params
		}
	}
	return nil, nil
}

func validateViewQueryParams(view *ViewSpec, query url.Values) (map[string]string, error) {
	allowedQuery := make(map[string]bool)
	for _, qp := range view.QueryParams {
		if trimmedQP := strings.TrimSpace(qp); trimmedQP != "" {
			allowedQuery[trimmedQP] = true
		}
	}
	for _, qp := range view.AllowedQueryParams {
		if trimmedQP := strings.TrimSpace(qp); trimmedQP != "" {
			allowedQuery[trimmedQP] = true
		}
	}

	validatedQuery := make(map[string]string, len(query))
	for k, vals := range query {
		if !allowedQuery[k] {
			return nil, fmt.Errorf(
				"UNDECLARED_PLUGIN_VIEW_QUERY_PARAM: query parameter %q is not declared in manifest.json views for path %q",
				k,
				view.Path,
			)
		}
		if len(vals) > 0 {
			validatedQuery[k] = vals[0]
		}
	}
	return validatedQuery, nil
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

	matchedView, extractedParams := matchViewPattern(views, targetSegs)
	if matchedView == nil {
		return nil, http.StatusNotFound, fmt.Sprintf("UNDECLARED_PLUGIN_VIEW_PATH: path %q is not declared in manifest.json views", normalizedPath)
	}

	validatedQuery, err := validateViewQueryParams(matchedView, query)
	if err != nil {
		return nil, http.StatusBadRequest, err.Error()
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

// ============================================================================
// Theming & SDK UI Static Assets
// ============================================================================

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

// ============================================================================
// Router Registration
// ============================================================================

func (h *Host) RegisterUISDKAndConfigRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /sdk/{asset_name}", h.handleGetSDKAsset)
	mux.HandleFunc("POST /api/test/seed-lru-plugins", h.handleSeedLRUPlugins)
	mux.HandleFunc("DELETE /api/test/seed-lru-plugins", h.handleDeleteLRUPlugins)
	mux.HandleFunc("GET /api/system/ui-config", h.handleGetSystemUIConfig)
	mux.HandleFunc("GET /api/system/search", h.handleSystemSearch)

	h.RegisterPinnedRecentsAndLogoutRoutes(mux)
}

// ============================================================================
// SDK Asset Handler
// ============================================================================

func (h *Host) handleGetSDKAsset(w http.ResponseWriter, r *http.Request) {
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
}
