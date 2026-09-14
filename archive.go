package main

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/Linhanmic/allure-report/logger"
)

func prepareReportDirs(cfg pluginConfig, reportsRoot string) error {
	if cfg.UseCustomDirs {
		return ensureDirs(cfg.ResultsDir, cfg.ReportDir)
	}
	if shouldOverwriteReports() {
		stamp := time.Now().Format(timeFormat)
		archiveDir := filepath.Join(reportsRoot, "archive", stamp)
		if err := archiveDirIfExists(cfg.ResultsDir, filepath.Join(archiveDir, allureResultsDirName)); err != nil {
			return err
		}
		if err := archiveDirIfExists(cfg.ReportDir, filepath.Join(archiveDir, allureReportDirName)); err != nil {
			return err
		}
		cleanOldArchives(filepath.Join(reportsRoot, "archive"))
	}
	return ensureDirs(cfg.ResultsDir, cfg.ReportDir)
}

func archiveDirIfExists(src, dst string) error {
	if !dirExists(src) || !dirHasContent(src) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dst), dirPermissions); err != nil {
		return err
	}
	return os.Rename(src, dst)
}

func cleanOldArchives(archiveRoot string) {
	maxCount := envInt("allure_archive_max_count", 0)
	if maxCount <= 0 {
		return
	}
	entries, err := os.ReadDir(archiveRoot)
	if err != nil {
		return
	}
	dirs := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		}
	}
	if len(dirs) <= maxCount {
		return
	}
	sort.Strings(dirs)
	toRemove := dirs[:len(dirs)-maxCount]
	for _, name := range toRemove {
		path := filepath.Join(archiveRoot, name)
		if err := os.RemoveAll(path); err != nil {
			logger.Warn("Failed to remove old archive %s: %s", path, err)
		} else {
			logger.Debug("Removed old archive: %s", path)
		}
	}
}

func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func ensureDirs(resultsDir, reportDir string) error {
	if err := os.MkdirAll(resultsDir, dirPermissions); err != nil {
		return err
	}
	if err := os.MkdirAll(reportDir, dirPermissions); err != nil {
		return err
	}
	return nil
}

func dirExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}

func dirHasContent(path string) bool {
	entries, err := os.ReadDir(path)
	return err == nil && len(entries) > 0
}
