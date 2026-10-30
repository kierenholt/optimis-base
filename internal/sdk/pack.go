package sdk

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var lucideIconCatalog = map[string]string{
	"award":        `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="8" r="6"/><path d="M15.477 12.89 17 22l-5-3-5 3 1.523-9.11"/></svg>`,
	"calendar":     `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="18" height="18" x="3" y="4" rx="2" ry="2"/><line x1="16" x2="16" y1="2" y2="6"/><line x1="8" x2="8" y1="2" y2="6"/><line x1="3" x2="21" y1="10" y2="10"/></svg>`,
	"book-open":    `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M2 3h6a4 4 0 0 1 4 4v14a3 3 0 0 0-3-3H2z"/><path d="M22 3h-6a4 4 0 0 0-4 4v14a3 3 0 0 1 3-3h7z"/></svg>`,
	"shield-alert": `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10"/><path d="M12 8v4"/><path d="M12 16h.01"/></svg>`,
	"heart-pulse":  `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M19 14c1.49-1.46 3-3.21 3-5.5A5.5 5.5 0 0 0 16.5 3c-1.76 0-3 .5-4.5 2-1.5-1.5-2.74-2-4.5-2A5.5 5.5 0 0 0 2 8.5c0 2.3 1.5 4.05 3 5.5l7 7Z"/><path d="M3.22 12H9.5l.5-1 2 4.5 2-7 1.5 3.5h5.27"/></svg>`,
	"bus":          `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M8 6v6"/><path d="M15 6v6"/><path d="M2 12h19.6"/><path d="M18 18h3s.5-1.7.8-2.8c.1-.4.2-.8.2-1.2 0-.4-.1-.8-.2-1.2l-1.4-5C20.1 6.8 19.1 6 18 6H4a2 2 0 0 0-2 2v10h3"/><circle cx="7" cy="18" r="2"/><path d="M9 18h5"/><circle cx="16" cy="18" r="2"/></svg>`,
	"users":        `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M22 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>`,
	"calculator":   `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="16" height="20" x="4" y="2" rx="2"/><line x1="8" x2="16" y1="6" y2="6"/><line x1="16" x2="16" y1="14" y2="18"/><path d="M16 10h.01"/><path d="M12 10h.01"/><path d="M8 10h.01"/><path d="M12 14h.01"/><path d="M8 14h.01"/><path d="M12 18h.01"/><path d="M8 18h.01"/></svg>`,
	"file-check":   `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M15 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7Z"/><path d="M14 2v4a2 2 0 0 0 2 2h4"/><path d="m9 15 2 2 4-4"/></svg>`,
	"box":          `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16Z"/><path d="m3.3 7 8.7 5 8.7-5"/><path d="M12 22V12"/></svg>`,
}

type MispkgSignature struct {
	PluginID     string            `json:"plugin_id"`
	Version      string            `json:"version"`
	Icon         string            `json:"icon"`
	Files        map[string]string `json:"files"`
	BundleSHA256 string            `json:"bundle_sha256"`
}

func ReorderFlagsFirst(args []string, boolFlags map[string]bool) []string {
	var flags []string
	var positionals []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
			trimmed := strings.TrimLeft(arg, "-")
			if !strings.Contains(trimmed, "=") && !boolFlags[trimmed] && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				flags = append(flags, args[i+1])
				i++
			}
		} else {
			positionals = append(positionals, arg)
		}
	}
	return append(flags, positionals...)
}

func ExtractLucideSVG(iconName string) (string, string) {
	normalized := strings.ToLower(strings.TrimSpace(iconName))
	if normalized == "" {
		normalized = "box"
	}
	if svg, ok := lucideIconCatalog[normalized]; ok {
		return normalized, svg
	}
	return normalized, fmt.Sprintf(
		`<svg xmlns="http://www.w3.org/2000/svg" data-lucide-icon=%q width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="18" height="18" x="3" y="3" rx="2"/><path d="M9 12h6"/><path d="M12 9v6"/></svg>`,
		normalized,
	)
}

func PackPlugin(pluginDir string, outputPath string, out io.Writer) (*MispkgSignature, error) {
	trimmedDir := strings.TrimSpace(pluginDir)
	if trimmedDir == "" {
		return nil, errors.New("missing mandatory plugin directory path")
	}
	trimmedOut := strings.TrimSpace(outputPath)
	if trimmedOut == "" {
		return nil, errors.New("missing mandatory output .mispkg path")
	}
	if out == nil {
		out = io.Discard
	}

	// Pre-flight gate: run all 4 mis-verify checks first
	if _, err := VerifyPluginDir(trimmedDir, out); err != nil {
		return nil, fmt.Errorf("pre-flight mis-verify check failed before packaging: %w", err)
	}

	absDir, err := filepath.Abs(trimmedDir)
	if err != nil {
		return nil, err
	}
	manifest, err := ReadAndValidateManifest(absDir)
	if err != nil {
		return nil, err
	}

	resolvedIcon, iconSVG := ExtractLucideSVG(manifest.Icon)

	archiveFiles := make(map[string][]byte)
	manifestBytes, err := os.ReadFile(filepath.Join(absDir, "manifest.json"))
	if err != nil {
		return nil, err
	}
	archiveFiles["manifest.json"] = manifestBytes
	archiveFiles["icon.svg"] = []byte(iconSVG)

	for _, sub := range []string{"migrations", "src", "ui/dist"} {
		subPath := filepath.Join(absDir, filepath.FromSlash(sub))
		info, statErr := os.Stat(subPath)
		if statErr != nil || !info.IsDir() {
			continue
		}
		err := filepath.WalkDir(subPath, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				return nil
			}
			rel, relErr := filepath.Rel(absDir, path)
			if relErr != nil {
				return relErr
			}
			normRel := filepath.ToSlash(rel)
			content, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			archiveFiles[normRel] = content
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	var sortedPaths []string
	for relPath := range archiveFiles {
		sortedPaths = append(sortedPaths, relPath)
	}
	sort.Strings(sortedPaths)

	fileDigests := make(map[string]string, len(sortedPaths))
	overallHasher := sha256.New()
	for _, relPath := range sortedPaths {
		sum := sha256.Sum256(archiveFiles[relPath])
		hexDigest := hex.EncodeToString(sum[:])
		fileDigests[relPath] = hexDigest
		_, _ = overallHasher.Write([]byte(relPath + ":" + hexDigest + "\n"))
	}

	sig := &MispkgSignature{
		PluginID:     manifest.ID,
		Version:      manifest.Version,
		Icon:         resolvedIcon,
		Files:        fileDigests,
		BundleSHA256: hex.EncodeToString(overallHasher.Sum(nil)),
	}
	sigBytes, err := json.MarshalIndent(sig, "", "  ")
	if err != nil {
		return nil, err
	}
	archiveFiles["SIGNATURE.json"] = sigBytes
	sortedPaths = append(sortedPaths, "SIGNATURE.json")
	sort.Strings(sortedPaths)

	if err := os.MkdirAll(filepath.Dir(trimmedOut), 0o755); err != nil {
		return nil, err
	}

	var zipBuf bytes.Buffer
	zw := zip.NewWriter(&zipBuf)
	fixedModTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	for _, relPath := range sortedPaths {
		hdr := &zip.FileHeader{
			Name:     relPath,
			Method:   zip.Deflate,
			Modified: fixedModTime,
		}
		w, err := zw.CreateHeader(hdr)
		if err != nil {
			_ = zw.Close()
			return nil, err
		}
		if _, err := w.Write(archiveFiles[relPath]); err != nil {
			_ = zw.Close()
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}

	if err := os.WriteFile(trimmedOut, zipBuf.Bytes(), 0o644); err != nil {
		return nil, err
	}

	_, _ = fmt.Fprintf(
		out,
		"type=pack status=MISPKG_PACKED plugin=%s version=%s icon=%s output=%s files=%s sha256=%s\n",
		manifest.ID,
		manifest.Version,
		resolvedIcon,
		trimmedOut,
		strings.Join(sortedPaths, ","),
		sig.BundleSHA256,
	)
	return sig, nil
}
