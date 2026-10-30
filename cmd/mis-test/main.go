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
	var persona string
	flag.StringVar(&pluginDir, "plugin", "", "Path to plugin directory containing manifest.json and tests/")
	flag.StringVar(&persona, "persona", "stf_tch_7a", "Default staff persona ID for t.rpc calls")
	_ = flag.CommandLine.Parse(sdk.ReorderFlagsFirst(os.Args[1:], nil))

	targetDir := strings.TrimSpace(pluginDir)
	if targetDir == "" && flag.NArg() > 0 {
		targetDir = strings.TrimSpace(flag.Arg(0))
	}
	if targetDir == "" {
		targetDir = "."
	}

	if _, err := sdk.RunPluginTests(targetDir, persona, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "mis-test failed: %v\n", err)
		os.Exit(1)
	}
}
