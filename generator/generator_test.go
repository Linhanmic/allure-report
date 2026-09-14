package generator

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratePrefersBundledAllure(t *testing.T) {
	results := t.TempDir()
	report := t.TempDir()
	bundled := t.TempDir()
	if err := os.WriteFile(filepath.Join(results, "dummy-result.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(bundled, "node_modules", "allure"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bundled, "node_modules", "allure", "cli.js"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bundled, "generate.mjs"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	var got []string
	res, err := Generate(Options{
		ResultsDir: results,
		ReportDir:  report,
		ReportName: "Demo",
		Language:   "zh",
		SingleFile: true,
		BundledDir: bundled,
		LookPath: func(file string) (string, error) {
			switch file {
			case "node":
				return "/usr/bin/node", nil
			case "allure":
				return "/usr/bin/allure", nil
			}
			return "", os.ErrNotExist
		},
		Command: func(name string, args ...string) *exec.Cmd {
			got = append([]string{name}, args...)
			return exec.Command("true")
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Generated {
		t.Fatal("expected generated")
	}
	if got[0] != "/usr/bin/node" || !strings.Contains(got[1], "generate.mjs") {
		t.Fatalf("expected bundled node command, got: %v", got)
	}
}

func TestGenerateUsesAllureAwesome(t *testing.T) {
	results := t.TempDir()
	report := t.TempDir()
	if err := os.WriteFile(filepath.Join(results, "dummy-result.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	var got []string
	res, err := Generate(Options{
		ResultsDir: results,
		ReportDir:  report,
		ReportName: "Demo",
		Language:   "zh",
		SingleFile: true,
		BundledDir: t.TempDir(),
		LookPath: func(file string) (string, error) {
			if file == "allure" {
				return "/usr/bin/allure", nil
			}
			return "", os.ErrNotExist
		},
		Command: func(name string, args ...string) *exec.Cmd {
			got = append([]string{name}, args...)
			cmd := exec.Command("true")
			return cmd
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Generated {
		t.Fatal("expected generated")
	}
	joined := strings.Join(got, " ")
	if !strings.Contains(joined, "awesome") || !strings.Contains(joined, "--single-file") {
		t.Fatalf("command: %s", joined)
	}
	if _, err := os.Stat(filepath.Join(report, "allurerc.mjs")); err == nil {
		t.Fatal("config should not be written into the report directory")
	}
}

func TestGenerateFallsBackWhenMissing(t *testing.T) {
	_, err := Generate(Options{
		ResultsDir: t.TempDir(),
		ReportDir:  t.TempDir(),
		BundledDir: t.TempDir(),
		LookPath:   func(string) (string, error) { return "", os.ErrNotExist },
	})
	if err == nil {
		t.Fatal("expected error when allure and npx are missing")
	}
}
