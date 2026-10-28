package host

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

const pluginSandboxCSP = "default-src 'none'; script-src 'self' 'unsafe-inline' http: https:; style-src 'self' 'unsafe-inline' http: https:; img-src 'self' data:; font-src 'self' data:; connect-src 'none';"

type PluginUINavDescriptor struct {
	ID               string   `json:"id"`
	Title            string   `json:"title"`
	Version          string   `json:"version"`
	EntryURL         string   `json:"entry_url"`
	PermittedMethods []string `json:"permitted_methods"`
}

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

func (h *Host) RegisterUISDKAndConfigRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /sdk/{asset_name}", func(w http.ResponseWriter, r *http.Request) {
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
	})

	mux.HandleFunc("GET /api/system/ui-config", func(w http.ResponseWriter, r *http.Request) {
		schoolURN := h.cfg.SchoolURN
		schoolName := "Oakridge All-Through Academy"
		academicYearID := "ay_2026_2027"

		entityHandlers := map[string]string{
			"student": "uk.org.optimis.test",
			"group":   "uk.org.optimis.test",
			"staff":   "uk.org.optimis.test",
		}

		rows, err := h.coreDAL.ReadDB().QueryContext(r.Context(), `SELECT key, value FROM __system_config`)
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
					case "entity_handler.student":
						if strings.TrimSpace(v) != "" {
							entityHandlers["student"] = strings.TrimSpace(v)
						}
					case "entity_handler.group":
						if strings.TrimSpace(v) != "" {
							entityHandlers["group"] = strings.TrimSpace(v)
						}
					case "entity_handler.staff":
						if strings.TrimSpace(v) != "" {
							entityHandlers["staff"] = strings.TrimSpace(v)
						}
					}
				}
			}
		}

		h.mu.RLock()
		pluginDescriptors := make([]PluginUINavDescriptor, 0, len(h.plugins))
		for _, p := range h.plugins {
			if p.State != PluginStateActive {
				continue
			}
			methods := make([]string, 0, len(p.Manifest.Endpoints))
			for m := range p.Manifest.Endpoints {
				methods = append(methods, m)
			}
			sort.Strings(methods)

			title := p.Manifest.ID
			if p.Manifest.ID == "uk.org.optimis.test" {
				title = "Pupil & Classroom Workspace"
			}

			pluginDescriptors = append(pluginDescriptors, PluginUINavDescriptor{
				ID:               p.Manifest.ID,
				Title:            title,
				Version:          p.Manifest.Version,
				EntryURL:         fmt.Sprintf("/plugins/%s/ui/index.html", p.Manifest.ID),
				PermittedMethods: methods,
			})
		}
		h.mu.RUnlock()

		sort.Slice(pluginDescriptors, func(i, j int) bool {
			return pluginDescriptors[i].ID < pluginDescriptors[j].ID
		})

		writeJSON(w, http.StatusOK, map[string]any{
			"ok":                   true,
			"school_urn":           schoolURN,
			"school_name":          schoolName,
			"academic_year_id":     academicYearID,
			"themes":               DefaultThemeTokens(),
			"entity_link_handlers": entityHandlers,
			"plugins":              pluginDescriptors,
		})
	})
}
