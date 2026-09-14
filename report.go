package main

import (
	"fmt"
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
	allureResultsDirName        = "allure-results"
	allureReportDirName         = "allure-report"
	gaugeReportsDirEnvName      = "gauge_reports_dir"
	gaugeScreenshotsDirEnvName  = "gauge_screenshots_dir"
	overwriteReportsEnvProperty = "overwrite_reports"
	pluginActionEnv             = pluginID + "_action"
	executionAction             = "execution"
	gaugeProjectRootEnv         = "GAUGE_PROJECT_ROOT"
	timeFormat                  = "2006-01-02_15.04.05"
	dirPermissions              = 0o755

	// 报告插件类型常量
	PluginAwesome   = "awesome"
	PluginClassic   = "classic"
	PluginAllure2   = "allure2"
	PluginDashboard = "dashboard"
	PluginCSV       = "csv"
	PluginLog       = "log"
)

var projectRoot string

func createReport(suiteResult *gauge_messages.SuiteExecutionResult) {
	cfg := loadConfig()
	logger.Debug("Config: results=%s report=%s overwrite=%v custom=%v", cfg.ResultsDir, cfg.ReportDir, shouldOverwriteReports(), cfg.UseCustomDirs)

	if err := prepareReportDirs(cfg, cfg.ReportsRoot); err != nil {
		logger.Fatal("Failed to prepare report directories: %s", err)
	}

	screenshotsDir := os.Getenv(gaugeScreenshotsDirEnvName)
	if screenshotsDir != "" && !filepath.IsAbs(screenshotsDir) && projectRoot != "" {
		screenshotsDir = filepath.Join(projectRoot, screenshotsDir)
	}
	if screenshotsDir != "" {
		logger.Debug("Screenshots directory: %s", screenshotsDir)
	}

	model := converter.Convert(suiteResult.GetSuiteResult(), converter.Options{
		ProjectRoot:    projectRoot,
		ScreenshotsDir: screenshotsDir,
		IssuePattern:   cfg.IssuePattern,
		TMSPattern:     cfg.TMSPattern,
		ReportName:     cfg.ReportName,
	})
	if err := writer.Write(model, cfg.ResultsDir); err != nil {
		logger.Fatal("Failed to write Allure results: %s", err)
	}
	logger.Info("Successfully generated allure-results to => %s", cfg.ResultsDir)

	if !cfg.GenerateHTML {
		logger.Debug("HTML generation disabled, skipping")
		return
	}

	// 解析多报告插件配置
	plugins := parsePlugins(cfg)
	if len(plugins) == 0 {
		logger.Warn("No report plugins enabled, using default awesome plugin")
		plugins = []generator.ReportPlugin{
			{
				ID:   "awesome",
				Type: PluginAwesome,
				Options: map[string]interface{}{
					"reportName":     cfg.ReportName,
					"singleFile":     cfg.SingleFile,
					"reportLanguage": cfg.Language,
					"theme":          cfg.Theme,
					"open":           false,
				},
			},
		}
	}

	// 生成报告
	result, err := generator.Generate(generator.Options{
		ResultsDir:    cfg.ResultsDir,
		ReportDir:     cfg.ReportDir,
		ReportName:    cfg.ReportName,
		Language:      cfg.Language,
		Theme:         cfg.Theme,
		SingleFile:    cfg.SingleFile,
		HistoryPath:   cfg.HistoryPath,
		AppendHistory: cfg.AppendHistory,
		HistoryLimit:  cfg.HistoryLimit,
		Plugins:       plugins,
	})
	if err != nil {
		logger.Error("Allure 3 HTML report was not generated: %s", err)
		logger.Info("Raw Allure results are available at %s. Install Allure 3 CLI or Node.js (npx allure@3) and run: allure awesome %s --output %s --single-file", cfg.ResultsDir, cfg.ResultsDir, cfg.ReportDir)
		return
	}
	if result != nil && result.Generated {
		logger.Info("Successfully generated Allure 3 report to => %s", cfg.ReportDir)
	}
}

type pluginConfig struct {
	ReportsRoot    string
	ResultsDir     string
	ReportDir      string
	ReportName     string
	Language       string
	Theme          string
	IssuePattern   string
	TMSPattern     string
	SingleFile     bool
	GenerateHTML   bool
	UseCustomDirs  bool
	HistoryPath    string
	AppendHistory  bool
	HistoryLimit   int
}

func loadConfig() pluginConfig {
	reportsDir := os.Getenv(gaugeReportsDirEnvName)
	if reportsDir == "" {
		reportsDir = defaultReportsDir
	}
	if !filepath.IsAbs(reportsDir) && projectRoot != "" {
		reportsDir = filepath.Join(projectRoot, reportsDir)
	}

	customResults := strings.TrimSpace(os.Getenv("allure_results_dir"))
	customReport := strings.TrimSpace(os.Getenv("allure_report_dir"))
	useCustom := customResults != "" || customReport != ""

	var resultsDir, reportDir string
	if useCustom {
		base := reportsDir
		resultsDir = envOr("allure_results_dir", filepath.Join(base, allureResultsDirName))
		reportDir = envOr("allure_report_dir", filepath.Join(base, allureReportDirName))
	} else if shouldOverwriteReports() {
		resultsDir = filepath.Join(reportsDir, allureResultsDirName)
		reportDir = filepath.Join(reportsDir, allureReportDirName)
	} else {
		stamp := time.Now().Format(timeFormat)
		runDir := filepath.Join(reportsDir, allureReportDirName, stamp)
		resultsDir = filepath.Join(runDir, allureResultsDirName)
		reportDir = filepath.Join(runDir, allureReportDirName)
	}

	if !filepath.IsAbs(resultsDir) && projectRoot != "" {
		resultsDir = filepath.Join(projectRoot, resultsDir)
	}
	if !filepath.IsAbs(reportDir) && projectRoot != "" {
		reportDir = filepath.Join(projectRoot, reportDir)
	}

	cfg := pluginConfig{
		ReportsRoot:   reportsDir,
		ResultsDir:    resultsDir,
		ReportDir:     reportDir,
		ReportName:    envOr("allure_report_name", "Gauge Allure Report"),
		Language:      envOr("allure_report_language", "zh"),
		Theme:         envOr("allure_report_theme", "auto"),
		IssuePattern:  os.Getenv("allure_issue_pattern"),
		TMSPattern:    os.Getenv("allure_tms_pattern"),
		SingleFile:    envBool("allure_report_single_file", true),
		GenerateHTML:  envBool("allure_report_generate", true),
		UseCustomDirs: useCustom,
		HistoryPath:   envOr("allure_history_path", ""),
		AppendHistory: envBool("allure_history_append", true),
		HistoryLimit:  envInt("allure_history_limit", 0),
	}
	validateConfig(cfg)
	return cfg
}

// parsePlugins 解析多报告插件配置
func parsePlugins(cfg pluginConfig) []generator.ReportPlugin {
	// 获取启用的报告类型列表
	reportsStr := envOr("allure_reports", "awesome")
	reportTypes := strings.Split(reportsStr, ",")

	var plugins []generator.ReportPlugin
	pluginCount := make(map[string]int)

	for _, reportType := range reportTypes {
		reportType = strings.TrimSpace(reportType)
		if reportType == "" {
			continue
		}

		// 检查是否启用
		if !envBool("allure_"+reportType+"_enabled", true) {
			continue
		}

		// 生成插件 ID
		pluginID := reportType
		if pluginCount[reportType] > 0 {
			pluginID = fmt.Sprintf("%s_%d", reportType, pluginCount[reportType])
		}
		pluginCount[reportType]++

		plugin := generator.ReportPlugin{
			ID:      pluginID,
			Type:    reportType,
			Import:  getPluginImport(reportType),
			Options: buildPluginOptions(reportType, cfg),
		}

		plugins = append(plugins, plugin)
	}

	return plugins
}

// getPluginImport 返回插件的模块路径
func getPluginImport(pluginType string) string {
	switch pluginType {
	case PluginAwesome:
		return "@allurereport/plugin-awesome"
	default:
		return "" // 内置插件不需要 import
	}
}

// buildPluginOptions 构建插件选项
func buildPluginOptions(pluginType string, cfg pluginConfig) map[string]interface{} {
	options := map[string]interface{}{
		"reportName":     envOr("allure_"+pluginType+"_name", cfg.ReportName),
		"singleFile":     cfg.SingleFile,
		"reportLanguage": cfg.Language,
	}

	switch pluginType {
	case PluginAwesome:
		options["theme"] = cfg.Theme
		options["open"] = false
		// 支持 groupBy 配置
		if groupBy := os.Getenv("allure_awesome_group_by"); groupBy != "" {
			options["groupBy"] = strings.Split(groupBy, ",")
		}
	case PluginClassic, PluginAllure2, PluginDashboard:
		// 这些插件使用基本选项
	case PluginCSV:
		options["fileName"] = envOr("allure_csv_filename", "report.csv")
	case PluginLog:
		options["groupBy"] = "none"
	}

	return options
}

func validateConfig(cfg pluginConfig) {
	validThemes := map[string]bool{"light": true, "dark": true, "auto": true}
	if !validThemes[cfg.Theme] {
		logger.Warn("Invalid allure_report_theme '%s', using 'auto'. Valid values: light, dark, auto", cfg.Theme)
	}
	if cfg.ResultsDir == cfg.ReportDir && cfg.ResultsDir != "" {
		logger.Warn("Results dir and report dir are the same: %s", cfg.ResultsDir)
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
