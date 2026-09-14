package generator

import (
	"os"
	"path/filepath"
)

const bundledDirName = "bundled"

func resolveBundledDir(opts Options) string {
	if opts.BundledDir != "" {
		return opts.BundledDir
	}
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	pluginRoot := filepath.Dir(filepath.Dir(exe))
	return filepath.Join(pluginRoot, bundledDirName)
}

func bundledAvailable(bundledDir string) bool {
	if bundledDir == "" {
		return false
	}
	cli := filepath.Join(bundledDir, "node_modules", "allure", "cli.js")
	script := filepath.Join(bundledDir, "generate.mjs")
	if _, err := os.Stat(cli); err != nil {
		return false
	}
	if _, err := os.Stat(script); err != nil {
		return false
	}
	return true
}
