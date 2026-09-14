package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrepareReportDirsArchivesOnOverwrite(t *testing.T) {
	root := t.TempDir()
	resultsDir := filepath.Join(root, "reports", "allure-results")
	reportDir := filepath.Join(root, "reports", "allure-report")
	stale := filepath.Join(resultsDir, "stale-result.json")
	if err := os.MkdirAll(resultsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stale, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reportDir, "index.html"), []byte("<html></html>"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := pluginConfig{
		ReportsRoot:   filepath.Join(root, "reports"),
		ResultsDir:    resultsDir,
		ReportDir:     reportDir,
		UseCustomDirs: false,
	}
	if err := prepareReportDirs(cfg, cfg.ReportsRoot); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatal("expected stale result to be archived")
	}
	archiveRoot := filepath.Join(root, "reports", "archive")
	entries, err := os.ReadDir(archiveRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one archive entry, got %d", len(entries))
	}
	archivedResults := filepath.Join(archiveRoot, entries[0].Name(), "allure-results", "stale-result.json")
	if _, err := os.Stat(archivedResults); err != nil {
		t.Fatalf("archived results missing: %v", err)
	}
	archivedHTML := filepath.Join(archiveRoot, entries[0].Name(), "allure-report", "index.html")
	if _, err := os.Stat(archivedHTML); err != nil {
		t.Fatalf("archived report missing: %v", err)
	}
}

func TestLoadConfigTimestampedRunDir(t *testing.T) {
	root := t.TempDir()
	projectRoot = root
	t.Setenv("GAUGE_PROJECT_ROOT", root)
	t.Setenv("gauge_reports_dir", "reports")
	t.Setenv("overwrite_reports", "false")

	cfg := loadConfig()
	if !strings.Contains(cfg.ResultsDir, filepath.Join("allure-report")) {
		t.Fatalf("expected timestamped run dir, got %s", cfg.ResultsDir)
	}
	if filepath.Base(filepath.Dir(cfg.ReportDir)) != filepath.Base(filepath.Dir(cfg.ResultsDir)) {
		t.Fatalf("results and report should share run dir: %s vs %s", cfg.ResultsDir, cfg.ReportDir)
	}
}
