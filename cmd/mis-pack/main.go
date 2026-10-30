package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"optimis/internal/sdk"
)

func main() {
	var pluginDir string
	var outPath string
	flag.StringVar(&pluginDir, "plugin", "", "Path to plugin directory containing manifest.json")
	flag.StringVar(&outPath, "out", "", "Output path for .mispkg bundle")
	flag.StringVar(&outPath, "o", "", "Output path for .mispkg bundle (shorthand)")
	_ = flag.CommandLine.Parse(sdk.ReorderFlagsFirst(os.Args[1:], nil))

	targetDir := strings.TrimSpace(pluginDir)
	if targetDir == "" && flag.NArg() > 0 {
		targetDir = strings.TrimSpace(flag.Arg(0))
	}
	if targetDir == "" {
		targetDir = "."
	}

	targetOut := strings.TrimSpace(outPath)
	if targetOut == "" {
		manifest, err := sdk.ReadAndValidateManifest(targetDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "mis-pack failed: %v\n", err)
			os.Exit(1)
		}
		targetOut = filepath.Join(targetDir, fmt.Sprintf("%s-%s.mispkg", manifest.ID, manifest.Version))
	}

	if _, err := sdk.PackPlugin(targetDir, targetOut, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "mis-pack failed: %v\n", err)
		os.Exit(1)
	}
}
