package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"optimis/internal/sdk"
)

func main() {
	var pluginDir string
	flag.StringVar(&pluginDir, "plugin", "", "Path to plugin directory containing manifest.json")
	flag.Parse()

	targetDir := strings.TrimSpace(pluginDir)
	if targetDir == "" && flag.NArg() > 0 {
		targetDir = strings.TrimSpace(flag.Arg(0))
	}
	if targetDir == "" {
		targetDir = "."
	}

	if _, err := sdk.VerifyPluginDir(targetDir, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "mis-verify failed: %v\n", err)
		os.Exit(1)
	}
}
