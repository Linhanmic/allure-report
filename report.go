package main

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Linhanmic/allure-report/converter"
	"github.com/Linhanmic/allure-report/generator"
	"github.com/Linhanmic/allure-report/logger"
	"github.com/Linhanmic/allure-report/writer"
	"github.com/getgauge/gauge-proto/go/gauge_messages"
)

const (
	pluginID                    = "allure-report"
	defaultReportsDir           = "reports"
	gaugeReportsDirEnvName      = "gauge_reports_dir"
	overwriteReportsEnvProperty = "overwrite_reports"
	pluginActionEnv             = pluginID + "_action"
	executionAction             = "execution"
	gaugeProjectRootEnv         = "GAUGE_PROJECT_ROOT"
	timeFormat                  = "2006-01-02_15.04.05"
	dirPermissions              = 0o755
)

var projectRoot string

func createReport(suiteResult *gauge_messages.SuiteExecutionResult) {
	cfg := loadConfig()
	if shouldOverwriteReports() {
		_ = os.RemoveAll(cfg.ResultsDir)
		_ = os.RemoveAll(cfg.ReportDir)
	}
	if err := os.MkdirAll(cfg.ResultsDir, dirPermissions); err != nil {
		logger.Fatal("Failed to create results directory %s: %s", cfg.ResultsDir, err)
	}
	if err := os.MkdirAll(cfg.ReportDir, dirPermissions); err != nil {
		logger.Fatal("Failed to create report directory %s: %s", cfg.ReportDir, err)
	}

	model := converter.Convert(suiteResult.GetSuiteResult(), converter.Options{
		ProjectRoot:  projectRoot,
		IssuePattern: cfg.IssuePattern,
		TMSPattern:   cfg.TMSPattern,
		ReportName:   cfg.ReportName,
	})
	if err := writer.Write(model, cfg.ResultsDir); err != nil {
		logger.Fatal("Failed to write Allure results: %s", err)
	}
	logger.Info("Successfully generated allure-results to => %s", cfg.ResultsDir)

	if !cfg.GenerateHTML {
		return
	}
	result, err := generator.Generate(generator.Options{
		ResultsDir: cfg.ResultsDir,
		ReportDir:  cfg.ReportDir,
		ReportName: cfg.ReportName,
		Language:   cfg.Language,
		Theme:      cfg.Theme,
		SingleFile: cfg.SingleFile,
	})
	if err != nil {
		logger.Error("Allure 3 HTML report was not generated: %s", err)
		logger.Info("Raw Allure results are available at %s. Install Allure 3 (`npm i -g allure`) or Node.js (`npx allure`) and re-run, or execute: allure awesome %s --output %s", cfg.ResultsDir, cfg.ResultsDir, cfg.ReportDir)
		return
	}
	if result != nil && result.Generated {
		logger.Info("Successfully generated Allure 3 report to => %s", cfg.ReportDir)
	}
}

type pluginConfig struct {
	ResultsDir   string
	ReportDir    string
	ReportName   string
	Language     string
	Theme        string
	IssuePattern string
	TMSPattern   string
	SingleFile   bool
	GenerateHTML bool
}

func loadConfig() pluginConfig {
	reportsDir := os.Getenv(gaugeReportsDirEnvName)
	if reportsDir == "" {
		reportsDir = defaultReportsDir
	}
	if !filepath.IsAbs(reportsDir) && projectRoot != "" {
		reportsDir = filepath.Join(projectRoot, reportsDir)
	}
	stamp := ""
	if !shouldOverwriteReports() {
		stamp = time.Now().Format(timeFormat)
	}
	base := reportsDir
	if stamp != "" {
		base = filepath.Join(reportsDir, stamp)
	}

	resultsDir := envOr("allure_results_dir", filepath.Join(base, "allure-results"))
	reportDir := envOr("allure_report_dir", filepath.Join(base, "allure-report"))
	if !filepath.IsAbs(resultsDir) && projectRoot != "" {
		resultsDir = filepath.Join(projectRoot, resultsDir)
	}
	if !filepath.IsAbs(reportDir) && projectRoot != "" {
		reportDir = filepath.Join(projectRoot, reportDir)
	}

	return pluginConfig{
		ResultsDir:   resultsDir,
		ReportDir:    reportDir,
		ReportName:   envOr("allure_report_name", "Gauge Allure Report"),
		Language:     envOr("allure_report_language", "zh"),
		Theme:        envOr("allure_report_theme", "auto"),
		IssuePattern: os.Getenv("allure_issue_pattern"),
		TMSPattern:   os.Getenv("allure_tms_pattern"),
		SingleFile:   envBool("allure_report_single_file", true),
		GenerateHTML: envBool("allure_report_generate", true),
	}
}

func findPluginAndProjectRoot() {
	projectRoot = os.Getenv(gaugeProjectRootEnv)
	if projectRoot == "" {
		cwd, err := os.Getwd()
		if err != nil {
			logger.Fatal("Environment variable '%s' is not set.", gaugeProjectRootEnv)
		}
		projectRoot = cwd
	}
}

func shouldOverwriteReports() bool {
	v := strings.TrimSpace(os.Getenv(overwriteReportsEnvProperty))
	if v == "" {
		return true
	}
	return strings.EqualFold(v, "true")
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	switch strings.ToLower(v) {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return fallback
	}
}
