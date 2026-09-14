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

// Options control how the Allure 3 HTML report is generated.
type Options struct {
	ResultsDir string
	ReportDir  string
	ReportName string
	Language   string
	Theme      string
	SingleFile bool
	LookPath   func(file string) (string, error)
	Command    func(name string, args ...string) *exec.Cmd
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
	single := "false"
	if opts.SingleFile {
		single = "true"
	}
	content := fmt.Sprintf(`export default {
  name: %q,
  output: %q,
  plugins: {
    awesome: {
      options: {
        reportName: %q,
        singleFile: %s,
        reportLanguage: %q,
        theme: %q,
        open: false
      }
    }
  }
};
`, opts.ReportName, filepath.ToSlash(opts.ReportDir), opts.ReportName, single, opts.Language, opts.Theme)

	path := filepath.Join(os.TempDir(), fmt.Sprintf("allurerc-%d.mjs", os.Getpid()))
	if err := os.WriteFile(path, []byte(content), filePermissions); err != nil {
		return "", fmt.Errorf("write allurerc.mjs: %w", err)
	}
	return path, nil
}
