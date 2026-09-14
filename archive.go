package main

import (
	"os"
	"path/filepath"
	"time"
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
