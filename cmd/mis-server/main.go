package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"optimis/internal/host"
)

func loadDotEnv(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("mandatory .env file missing at %s: %w", path, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, val)
		}
	}
	return scanner.Err()
}

func requireEnv(key string) (string, error) {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return "", fmt.Errorf("missing mandatory environment variable: %s", key)
	}
	return val, nil
}

func parseStrictBoolEnv(key string) (bool, error) {
	raw, err := requireEnv(key)
	if err != nil {
		return false, err
	}
	switch strings.ToLower(raw) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean environment variable %s=%q: must be 'true' or 'false'", key, raw)
	}
}

func discoverPluginFolders(dir string, includeTestPlugin bool) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read plugin directory %s: %w", dir, err)
	}
	var folders []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !includeTestPlugin && name == "test_plugin" {
			continue
		}
		manifestPath := filepath.Join(dir, name, "manifest.json")
		if st, err := os.Stat(manifestPath); err == nil && !st.IsDir() {
			folders = append(folders, name)
		}
	}
	sort.Strings(folders)
	return folders, nil
}

func resolvePreloadPluginFolders(pluginsDir string, rawSpec string, includeTestPlugin bool) ([]string, error) {
	trimmedSpec := strings.TrimSpace(rawSpec)
	if trimmedSpec == "" || strings.EqualFold(trimmedSpec, "none") || strings.EqualFold(trimmedSpec, "false") || trimmedSpec == "0" {
		return nil, nil
	}

	seen := make(map[string]struct{})
	var ordered []string
	addFolder := func(name string) {
		if _, exists := seen[name]; !exists {
			seen[name] = struct{}{}
			ordered = append(ordered, name)
		}
	}

	for _, part := range strings.Split(trimmedSpec, ",") {
		token := strings.TrimSpace(part)
		if token == "" || strings.EqualFold(token, "none") {
			continue
		}
		if strings.EqualFold(token, "all") || token == "*" {
			discovered, err := discoverPluginFolders(pluginsDir, includeTestPlugin)
			if err != nil {
				return nil, err
			}
			for _, f := range discovered {
				addFolder(f)
			}
			continue
		}

		// Case 1: token is a folder name directly inside PLUGINS_DIR (e.g. "timetable_attendance_plugin")
		candidateInPluginsDir := filepath.Join(pluginsDir, token)
		if st, err := os.Stat(filepath.Join(candidateInPluginsDir, "manifest.json")); err == nil && !st.IsDir() {
			addFolder(filepath.Base(filepath.Clean(token)))
			continue
		}

		// Case 2: token is a relative or absolute path on disk
		cleanPath := filepath.Clean(token)
		if dirStat, err := os.Stat(cleanPath); err == nil && dirStat.IsDir() {
			// Check if cleanPath itself is a single plugin directory containing manifest.json
			if mfStat, mfErr := os.Stat(filepath.Join(cleanPath, "manifest.json")); mfErr == nil && !mfStat.IsDir() {
				addFolder(filepath.Base(cleanPath))
				continue
			}
			// Otherwise check if cleanPath is a parent directory containing multiple plugin folders (e.g. "./plugins")
			discovered, dErr := discoverPluginFolders(cleanPath, includeTestPlugin)
			if dErr != nil {
				return nil, dErr
			}
			if len(discovered) > 0 {
				for _, f := range discovered {
					addFolder(f)
				}
				continue
			}
		}

		return nil, fmt.Errorf("PRELOAD_PLUGINS target %q is neither a valid plugin folder nor a directory of plugins in %s", token, pluginsDir)
	}

	return ordered, nil
}

func resolveDatabaseConfig(customFlag string) (string, string, error) {
	raw := strings.TrimSpace(customFlag)
	if raw == "" {
		raw = strings.TrimSpace(os.Getenv("DATABASE_PATH"))
	}
	if raw == "" {
		if legacyDataDir := strings.TrimSpace(os.Getenv("DATA_DIR_PATH")); legacyDataDir != "" {
			raw = filepath.Join(legacyDataDir, "core.db")
		} else {
			return "", "", errors.New("missing mandatory environment variable: DATABASE_PATH")
		}
	}

	cleanDB := filepath.Clean(raw)
	dataDir := strings.TrimSpace(os.Getenv("DATA_DIR_PATH"))
	if dataDir == "" {
		dataDir = filepath.Dir(cleanDB)
		if dataDir == "" || dataDir == "." {
			dataDir = ".data"
		}
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return "", "", fmt.Errorf("failed to create data directory %s: %w", dataDir, err)
	}
	return cleanDB, dataDir, nil
}

func run() error {
	if err := loadDotEnv(".env"); err != nil {
		return err
	}

	var allPluginsFlag bool
	var noPluginsFlag bool
	var includeTestPluginFlag bool
	var pluginsListFlag string
	var customDBPathFlag string

	flag.BoolVar(&allPluginsFlag, "all-plugins", false, "Install and activate all production plugins in PLUGINS_DIR at startup")
	flag.BoolVar(&noPluginsFlag, "no-plugins", false, "Boot with zero plugins loaded (overrides PRELOAD_PLUGINS)")
	flag.BoolVar(&includeTestPluginFlag, "include-test-plugin", false, "Include test_plugin when preloading all plugins from a directory")
	flag.StringVar(&pluginsListFlag, "plugins", "", "Single plugin folder, parent directory, or comma-separated list of plugin folders to preload at startup")
	flag.StringVar(&customDBPathFlag, "db", "", "Explicit SQLite database path (overrides DATABASE_PATH from .env)")
	flag.Parse()

	coreDBPath, dataDirPath, err := resolveDatabaseConfig(customDBPathFlag)
	if err != nil {
		return err
	}

	schoolURN, err := requireEnv("SCHOOL_URN")
	if err != nil {
		return err
	}
	kmsKeyHandle, err := requireEnv("HOST_KMS_KEY_HANDLE")
	if err != nil {
		return err
	}

	// Section 7C Local Break-Glass CLI (`./mis-server config ...`)
	if flag.Arg(0) == "config" || (len(os.Args) > 1 && strings.TrimSpace(os.Args[1]) == "config") {
		subArgs := os.Args[2:]
		if flag.Arg(0) == "config" {
			subArgs = flag.Args()[1:]
		}
		return host.RunConfigCLIWithPath(context.Background(), dataDirPath, schoolURN, coreDBPath, subArgs, os.Stdout, os.Stderr)
	}

	port, err := requireEnv("PORT")
	if err != nil {
		return err
	}
	pluginsDir, err := requireEnv("PLUGINS_DIR")
	if err != nil {
		return err
	}
	enableDevPersonaSwitcher, err := parseStrictBoolEnv("ENABLE_DEV_PERSONA_SWITCHER")
	if err != nil {
		return err
	}
	enableRealHTTPEgress, err := parseStrictBoolEnv("ENABLE_REAL_HTTP_EGRESS")
	if err != nil {
		return err
	}
	enableRealCloudKMS, err := parseStrictBoolEnv("ENABLE_REAL_CLOUD_KMS")
	if err != nil {
		return err
	}
	enableRealParentDMZSync, err := parseStrictBoolEnv("ENABLE_REAL_PARENT_DMZ_SYNC")
	if err != nil {
		return err
	}
	enableRealMATSync, err := parseStrictBoolEnv("ENABLE_REAL_MAT_SYNC")
	if err != nil {
		return err
	}
	enableMockDfeCensusService, err := parseStrictBoolEnv("ENABLE_MOCK_DFE_CENSUS_SERVICE")
	if err != nil {
		return err
	}

	h, err := host.NewHost(host.Config{
		DataDirPath:      dataDirPath,
		SchoolURN:        schoolURN,
		HostKMSKeyHandle: kmsKeyHandle,
		PluginsDir:       pluginsDir,
		CoreDBPath:       coreDBPath,
	})
	if err != nil {
		return err
	}
	defer h.Close()

	if err := h.ConfigureAuthMode(context.Background(), enableDevPersonaSwitcher); err != nil {
		return err
	}
	h.ConfigureEgressMode(enableRealHTTPEgress)
	// Configure mock DfE COLLECT census service feature flag.
	h.ConfigureMockDfeCensusService(enableMockDfeCensusService)
	if err := h.ConfigureCloudKMSMode(context.Background(), enableRealCloudKMS); err != nil {
		return err
	}
	if err := h.ConfigureParentDMZMode(context.Background(), enableRealParentDMZSync); err != nil {
		return err
	}
	if err := h.ConfigureMATSyncMode(context.Background(), enableRealMATSync); err != nil {
		return err
	}

	preloadSpec := strings.TrimSpace(os.Getenv("PRELOAD_PLUGINS"))
	if preloadSpec == "" {
		preloadSpec = strings.TrimSpace(os.Getenv("LOAD_PLUGINS"))
	}
	if strings.TrimSpace(pluginsListFlag) != "" {
		preloadSpec = strings.TrimSpace(pluginsListFlag)
	}
	if allPluginsFlag {
		preloadSpec = "all"
	}

	var foldersToLoad []string
	if !noPluginsFlag && preloadSpec != "" {
		resolved, rErr := resolvePreloadPluginFolders(pluginsDir, preloadSpec, includeTestPluginFlag)
		if rErr != nil {
			return rErr
		}
		foldersToLoad = resolved
	}

	for _, folder := range foldersToLoad {
		if err := h.LoadPlugin(folder); err != nil {
			return fmt.Errorf("failed to preload plugin folder %q: %w", folder, err)
		}
		log.Printf("Preloaded plugin folder: %s", folder)
	}

	handler, err := h.NewHTTPHandler()
	if err != nil {
		return err
	}

	addr := "127.0.0.1:" + port
	log.Printf("Optimis MIS Host Server listening on http://%s (preloaded_plugins=%d, core_db=%s, log_file=%s)", addr, len(foldersToLoad), coreDBPath, filepath.Join(dataDirPath, "mis-server.log"))
	if err := http.ListenAndServe(addr, handler); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: %v\n", err)
		os.Exit(1)
	}
}
