package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
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

func run() error {
	if err := loadDotEnv(".env"); err != nil {
		return err
	}

	dataDirPath, err := requireEnv("DATA_DIR_PATH")
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
	if len(os.Args) > 1 && strings.TrimSpace(os.Args[1]) == "config" {
		return host.RunConfigCLI(context.Background(), dataDirPath, schoolURN, os.Args[2:], os.Stdout, os.Stderr)
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

	if err := h.LoadPlugin("test_plugin"); err != nil {
		return fmt.Errorf("failed to load test_plugin: %w", err)
	}
	if err := h.LoadPlugin("dfe_census_plugin"); err != nil {
		return fmt.Errorf("failed to load dfe_census_plugin: %w", err)
	}

	handler, err := h.NewHTTPHandler()
	if err != nil {
		return err
	}

	addr := "127.0.0.1:" + port
	log.Printf("Optimis MIS Host Server listening on http://%s", addr)
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
