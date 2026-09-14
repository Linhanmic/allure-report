package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Linhanmic/allure-report/converter"
	"github.com/Linhanmic/allure-report/writer"
	"github.com/getgauge/gauge-proto/go/gauge_messages"
)

func TestCreateReportWritesResults(t *testing.T) {
	root := t.TempDir()
	projectRoot = root
	t.Setenv("GAUGE_PROJECT_ROOT", root)
	t.Setenv("gauge_reports_dir", "reports")
	t.Setenv("overwrite_reports", "true")
	t.Setenv("allure_report_generate", "false")

	resultsDir := filepath.Join(root, "reports", "allure-results")
	if err := os.MkdirAll(resultsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(resultsDir, "stale-result.json")
	if err := os.WriteFile(stale, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	createReport(sampleSuiteResult())

	entries, err := os.ReadDir(resultsDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatal("expected stale result to be archived when overwrite_reports=true")
	}
	archiveRoot := filepath.Join(root, "reports", "archive")
	if _, err := os.Stat(archiveRoot); err != nil {
		t.Fatalf("expected archive dir: %v", err)
	}
	found := false
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), "-result.json") {
			found = true
		}
	}
	if !found {
		t.Fatalf("no result json in %s: %v", resultsDir, names(entries))
	}
}

func TestLoadConfigRespectsCustomDirs(t *testing.T) {
	root := t.TempDir()
	projectRoot = root
	t.Setenv("gauge_reports_dir", "out")
	t.Setenv("overwrite_reports", "true")
	t.Setenv("allure_results_dir", "custom-results")
	t.Setenv("allure_report_dir", "custom-report")
	t.Setenv("allure_report_single_file", "false")
	t.Setenv("allure_report_language", "en")

	cfg := loadConfig()
	if filepath.Base(cfg.ResultsDir) != "custom-results" {
		t.Fatalf("results dir: %s", cfg.ResultsDir)
	}
	if filepath.Base(cfg.ReportDir) != "custom-report" {
		t.Fatalf("report dir: %s", cfg.ReportDir)
	}
	if cfg.SingleFile {
		t.Fatal("expected single file false")
	}
	if cfg.Language != "en" {
		t.Fatalf("language: %s", cfg.Language)
	}
}

func TestWriterRoundTripFromConverter(t *testing.T) {
	dir := t.TempDir()
	model := converter.Convert(&gauge_messages.ProtoSuiteResult{
		ProjectName: "demo",
		SpecResults: []*gauge_messages.ProtoSpecResult{{
			ProtoSpec: &gauge_messages.ProtoSpec{
				SpecHeading: "规格",
				Items: []*gauge_messages.ProtoItem{{
					ItemType: gauge_messages.ProtoItem_Scenario,
					Scenario: &gauge_messages.ProtoScenario{
						ScenarioHeading: "场景",
						ExecutionStatus: gauge_messages.ExecutionStatus_PASSED,
					},
				}},
			},
		}},
	}, converter.Options{Host: "h"})
	if err := writer.Write(model, dir); err != nil {
		t.Fatal(err)
	}
}

func sampleSuiteResult() *gauge_messages.SuiteExecutionResult {
	return &gauge_messages.SuiteExecutionResult{
		SuiteResult: &gauge_messages.ProtoSuiteResult{
			ProjectName: "demo",
			SpecResults: []*gauge_messages.ProtoSpecResult{{
				ProtoSpec: &gauge_messages.ProtoSpec{
					SpecHeading: "示例",
					FileName:    "specs/demo.spec",
					Items: []*gauge_messages.ProtoItem{{
						ItemType: gauge_messages.ProtoItem_Scenario,
						Scenario: &gauge_messages.ProtoScenario{
							ScenarioHeading: "通过",
							ExecutionStatus: gauge_messages.ExecutionStatus_PASSED,
							ExecutionTime:   12,
							ScenarioItems: []*gauge_messages.ProtoItem{{
								ItemType: gauge_messages.ProtoItem_Step,
								Step: &gauge_messages.ProtoStep{
									ActualText: "你好",
									StepExecutionResult: &gauge_messages.ProtoStepExecutionResult{
										ExecutionResult: &gauge_messages.ProtoExecutionResult{ExecutionTime: 5},
									},
								},
							}},
						},
					}},
				},
			}},
		},
	}
}

func names(entries []os.DirEntry) []string {
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.Name())
	}
	return out
}
