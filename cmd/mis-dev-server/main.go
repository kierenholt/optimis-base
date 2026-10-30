package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"optimis/internal/host"
	"optimis/internal/sdk"
)

func main() {
	var pluginDir string
	var persona string
	var port string
	var dbPath string
	var dryRun bool

	flag.StringVar(&pluginDir, "plugin", ".", "Path to plugin directory to mount in hot-reload mode")
	flag.StringVar(&persona, "persona", "stf_tch_7a", "Initial active staff persona ID")
	flag.StringVar(&port, "port", "18080", "HTTP port to listen on")
	flag.StringVar(&dbPath, "db", "", "Optional explicit SQLite core.db path (defaults to temp synthetic DB)")
	flag.BoolVar(&dryRun, "dry-run", false, "Initialize synthetic school DB, mount plugin, verify readiness, and exit 0")

	_ = flag.CommandLine.Parse(sdk.ReorderFlagsFirst(os.Args[1:], map[string]bool{"dry-run": true}))

	targetDir := strings.TrimSpace(pluginDir)
	if (targetDir == "" || targetDir == ".") && flag.NArg() > 0 {
		targetDir = strings.TrimSpace(flag.Arg(0))
	}
	absPluginDir, err := filepath.Abs(targetDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mis-dev-server failed to resolve plugin dir: %v\n", err)
		os.Exit(1)
	}

	manifest, err := sdk.ReadAndValidateManifest(absPluginDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mis-dev-server manifest validation failed: %v\n", err)
		os.Exit(1)
	}

	workDir, err := os.MkdirTemp("", "mis-dev-server-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "mis-dev-server failed to create workdir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(workDir)

	parentPluginsDir := filepath.Dir(absPluginDir)
	pluginFolder := filepath.Base(absPluginDir)

	h, err := host.NewHost(host.Config{
		DataDirPath:      filepath.Join(workDir, "data"),
		SchoolURN:        "100001",
		HostKMSKeyHandle: "kms_dev_server_key_01",
		PluginsDir:       parentPluginsDir,
		CoreDBPath:       strings.TrimSpace(dbPath),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "mis-dev-server failed to initialize host: %v\n", err)
		os.Exit(1)
	}
	defer h.Close()

	h.SetStructuredStdout(os.Stdout)

	ctx := context.Background()
	if err := sdk.SeedSyntheticDevCohort(ctx, h.DAL()); err != nil {
		fmt.Fprintf(os.Stderr, "mis-dev-server failed to seed synthetic school cohort: %v\n", err)
		os.Exit(1)
	}

	if strings.TrimSpace(persona) != "" {
		h.SetActiveDevPersona(persona)
	}

	if err := h.LoadPlugin(pluginFolder); err != nil {
		fmt.Fprintf(os.Stderr, "mis-dev-server failed to load plugin %s: %v\n", manifest.ID, err)
		os.Exit(1)
	}

	fmt.Fprintf(
		os.Stdout,
		"type=dev_server_init status=READY plugin=%s persona=%s port=%s\n",
		manifest.ID,
		strings.TrimSpace(persona),
		strings.TrimSpace(port),
	)

	if dryRun {
		return
	}

	handler, err := h.NewHTTPHandler()
	if err != nil {
		fmt.Fprintf(os.Stderr, "mis-dev-server handler init error: %v\n", err)
		os.Exit(1)
	}
	addr := "127.0.0.1:" + strings.TrimSpace(port)
	if err := http.ListenAndServe(addr, handler); err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, "mis-dev-server HTTP error: %v\n", err)
		os.Exit(1)
	}
}
