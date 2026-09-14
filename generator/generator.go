package generator

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	dirPermissions  = 0o755
	filePermissions = 0o644
)

// ReportPlugin 定义单个报告插件的配置
type ReportPlugin struct {
	ID      string                 // 插件唯一标识，如 "awesome", "classic"
	Import  string                 // 插件模块路径，如 "@allurereport/plugin-awesome"
	Type    string                 // 插件类型：awesome, classic, allure2, dashboard, csv, log
	Options map[string]interface{} // 插件特定选项
}

// Options control how the Allure 3 HTML report is generated.
type Options struct {
	ResultsDir    string
	ReportDir     string
	ReportName    string
	Language      string
	Theme         string
	SingleFile    bool
	HistoryPath   string         // 历史记录文件路径
	AppendHistory bool           // 是否追加历史记录
	HistoryLimit  int            // 历史记录限制数量
	Plugins       []ReportPlugin // 多插件配置
	LookPath      func(file string) (string, error)
	Command       func(name string, args ...string) *exec.Cmd
}

type Result struct {
	Generated bool
	Command   string
	Output    string
}

func (o *Options) withDefaults() {
	if o.LookPath == nil {
		o.LookPath = exec.LookPath
	}
	if o.Command == nil {
		o.Command = exec.Command
	}
	if o.ReportName == "" {
		o.ReportName = "Gauge Allure Report"
	}
	if o.Language == "" {
		o.Language = "zh"
	}
	if o.Theme == "" {
		o.Theme = "auto"
	}
}

// Generate creates an Allure 3 Awesome HTML report from allure-results.
func Generate(opts Options) (*Result, error) {
	opts.withDefaults()
	if strings.TrimSpace(opts.ResultsDir) == "" {
		return nil, fmt.Errorf("results dir is empty")
	}
	if strings.TrimSpace(opts.ReportDir) == "" {
		return nil, fmt.Errorf("report dir is empty")
	}
	if err := os.MkdirAll(opts.ReportDir, dirPermissions); err != nil {
		return nil, fmt.Errorf("create report dir: %w", err)
	}

	configPath, err := writeConfig(opts)
	if err != nil {
		return nil, err
	}
	defer os.Remove(configPath)

	attempts := commands(opts, configPath)
	if len(attempts) == 0 {
		return &Result{Generated: false}, fmt.Errorf("no report generator available: install Allure 3 CLI (allure) or Node.js (npx allure@3)")
	}

	var lastErr error
	var lastOut string
	var lastCmd string
	for _, attempt := range attempts {
		cmd := opts.Command(attempt[0], attempt[1:]...)
		out, err := cmd.CombinedOutput()
		lastOut = string(out)
		lastCmd = strings.Join(attempt, " ")
		if err == nil {
			return &Result{Generated: true, Command: lastCmd, Output: lastOut}, nil
		}
		lastErr = err
	}
	return &Result{Generated: false, Command: lastCmd, Output: lastOut}, fmt.Errorf("allure generate failed: %w\n%s", lastErr, lastOut)
}

func commands(opts Options, configPath string) [][]string {
	awesomeArgs := awesomeCLIArgs(opts, configPath)
	generateArgs := []string{
		"generate", opts.ResultsDir,
		"--output", opts.ReportDir,
		"--config", configPath,
		"--name", opts.ReportName,
	}

	var out [][]string
	if path, err := opts.LookPath("allure"); err == nil {
		out = append(out, append([]string{path}, awesomeArgs...))
		out = append(out, append([]string{path}, generateArgs...))
	}
	if path, err := opts.LookPath("npx"); err == nil {
		out = append(out, append([]string{path, "--yes", "allure@3"}, awesomeArgs...))
		out = append(out, append([]string{path, "--yes", "allure@3"}, generateArgs...))
	}
	return out
}

func awesomeCLIArgs(opts Options, configPath string) []string {
	args := []string{
		"awesome", opts.ResultsDir,
		"--output", opts.ReportDir,
		"--config", configPath,
		"--name", opts.ReportName,
		"--report-language", opts.Language,
	}
	if opts.SingleFile {
		args = append(args, "--single-file")
	}
	return args
}

func writeConfig(opts Options) (string, error) {
	// 构建 plugins 部分
	pluginsParts := make([]string, 0)
	for _, plugin := range opts.Plugins {
		pluginsParts = append(pluginsParts, generatePluginSection(plugin))
	}

	// 如果没有配置插件，使用默认的 awesome 插件
	if len(pluginsParts) == 0 {
		single := "false"
		if opts.SingleFile {
			single = "true"
		}
		pluginsParts = append(pluginsParts, fmt.Sprintf(`    awesome: {
      options: {
        reportName: %q,
        singleFile: %s,
        reportLanguage: %q,
        theme: %q,
        open: false
      }
    }`, opts.ReportName, single, opts.Language, opts.Theme))
	}

	// 构建 historyPath 部分
	historyPart := ""
	if opts.HistoryPath != "" {
		historyPart = fmt.Sprintf(`  historyPath: %q,
  appendHistory: %v,`, opts.HistoryPath, opts.AppendHistory)
		if opts.HistoryLimit > 0 {
			historyPart += fmt.Sprintf(`
  historyLimit: %d,`, opts.HistoryLimit)
		}
	}

	content := fmt.Sprintf(`export default {
  name: %q,
  output: %q,
%s
  plugins: {
%s
  }
};
`, opts.ReportName, filepath.ToSlash(opts.ReportDir), historyPart,
		strings.Join(pluginsParts, ",\n"))

	path := filepath.Join(os.TempDir(), fmt.Sprintf("allurerc-%d.mjs", os.Getpid()))
	if err := os.WriteFile(path, []byte(content), filePermissions); err != nil {
		return "", fmt.Errorf("write allurerc.mjs: %w", err)
	}
	return path, nil
}

// generatePluginSection 生成单个插件的配置部分
func generatePluginSection(plugin ReportPlugin) string {
	optionsJSON := formatOptions(plugin.Options)

	if plugin.Import != "" {
		return fmt.Sprintf(`    %s: {
      import: %q,
      options: %s
    }`, plugin.ID, plugin.Import, optionsJSON)
	}

	return fmt.Sprintf(`    %s: {
      options: %s
    }`, plugin.ID, optionsJSON)
}

// formatOptions 格式化选项为 JavaScript 对象字符串
func formatOptions(options map[string]interface{}) string {
	if len(options) == 0 {
		return "{}"
	}

	parts := make([]string, 0)
	for k, v := range options {
		switch val := v.(type) {
		case string:
			parts = append(parts, fmt.Sprintf(`        %s: %q`, k, val))
		case bool:
			if val {
				parts = append(parts, fmt.Sprintf(`        %s: true`, k))
			} else {
				parts = append(parts, fmt.Sprintf(`        %s: false`, k))
			}
		case int:
			parts = append(parts, fmt.Sprintf(`        %s: %d`, k, val))
		case []string:
			items := make([]string, len(val))
			for i, s := range val {
				items[i] = fmt.Sprintf(`%q`, s)
			}
			parts = append(parts, fmt.Sprintf(`        %s: [%s]`, k, strings.Join(items, ", ")))
		}
	}

	return "{\n" + strings.Join(parts, ",\n") + "\n      }"
}
