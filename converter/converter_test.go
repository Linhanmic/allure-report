package converter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/getgauge/gauge-proto/go/gauge_messages"
)

func TestConvertPassedScenarioWithSteps(t *testing.T) {
	suite := sampleSuite(passedScenario("登录成功",
		passedStep("打开首页"),
		passedStep("输入用户名 <user>"),
	))
	model := Convert(suite, testOpts(t))
	if len(model.Results) != 1 {
		t.Fatalf("expected 1 test, got %d", len(model.Results))
	}
	tr := model.Results[0]
	if tr.Name != "登录成功" {
		t.Fatalf("name: %s", tr.Name)
	}
	if tr.Status != statusPassed {
		t.Fatalf("status: %s", tr.Status)
	}
	if tr.TestCaseID == "" || tr.HistoryID == "" {
		t.Fatal("expected testCaseId and historyId")
	}
	if len(tr.Steps) != 2 {
		t.Fatalf("steps: %d", len(tr.Steps))
	}
	assertLabel(t, tr.Labels, "framework", "gauge")
	assertLabel(t, tr.Labels, "suite", "用户认证")
	assertLabel(t, tr.Labels, "parentSuite", "demo")
	if len(model.Containers) < 2 {
		t.Fatalf("expected suite and spec containers, got %d", len(model.Containers))
	}
}

func TestConvertScreenshotFromGaugeScreenshotsDir(t *testing.T) {
	screenshotsDir := t.TempDir()
	shotName := "screenshot-123.png"
	if err := os.WriteFile(filepath.Join(screenshotsDir, shotName), []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	step := &gauge_messages.ProtoItem{
		ItemType: gauge_messages.ProtoItem_Step,
		Step: &gauge_messages.ProtoStep{
			ActualText: "失败步骤",
			StepExecutionResult: &gauge_messages.ProtoStepExecutionResult{
				ExecutionResult: &gauge_messages.ProtoExecutionResult{
					Failed:                true,
					ErrorMessage:          "boom",
					FailureScreenshotFile: shotName,
				},
			},
		},
	}
	suite := sampleSuite(scenarioWith("截图场景", gauge_messages.ExecutionStatus_FAILED, step))
	model := Convert(suite, Options{ScreenshotsDir: screenshotsDir, FileExists: func(p string) bool {
		_, err := os.Stat(p)
		return err == nil
	}})
	if len(model.Files) == 0 {
		t.Fatal("expected screenshot bytes from gauge_screenshots_dir")
	}
	if model.Results[0].Steps[0].Attachments[0].Name != shotName {
		t.Fatalf("attachment name: %s", model.Results[0].Steps[0].Attachments[0].Name)
	}
}

func TestConvertFailedAssertionAndScreenshot(t *testing.T) {
	shot := filepath.Join(t.TempDir(), "fail.png")
	if err := os.WriteFile(shot, []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	step := &gauge_messages.ProtoItem{
		ItemType: gauge_messages.ProtoItem_Step,
		Step: &gauge_messages.ProtoStep{
			ActualText: "校验余额",
			StepExecutionResult: &gauge_messages.ProtoStepExecutionResult{
				ExecutionResult: &gauge_messages.ProtoExecutionResult{
					Failed:                true,
					ErrorMessage:          "expected 100 got 0",
					StackTrace:            "at steps/balance.go:12",
					ErrorType:             gauge_messages.ProtoExecutionResult_ASSERTION,
					ExecutionTime:         40,
					FailureScreenshotFile: shot,
					Message:               []string{"实际余额为 0"},
				},
			},
		},
	}
	suite := sampleSuite(scenarioWith("余额校验", gauge_messages.ExecutionStatus_FAILED, step))
	model := Convert(suite, testOpts(t))
	tr := model.Results[0]
	if tr.Status != statusFailed {
		t.Fatalf("status: %s", tr.Status)
	}
	if tr.StatusDetails == nil || !strings.Contains(tr.StatusDetails.Message, "expected 100") {
		t.Fatalf("status details: %+v", tr.StatusDetails)
	}
	if len(tr.Steps) != 1 {
		t.Fatalf("steps: %d", len(tr.Steps))
	}
	if len(tr.Steps[0].Attachments) == 0 {
		t.Fatal("expected failure screenshot attachment")
	}
	if len(model.Files) == 0 {
		t.Fatal("expected screenshot file bytes")
	}
	if len(tr.Steps[0].Steps) != 1 || tr.Steps[0].Steps[0].Name != "实际余额为 0" {
		t.Fatalf("custom message not mapped: %+v", tr.Steps[0].Steps)
	}
}

func TestConvertVerificationFailureIsBroken(t *testing.T) {
	step := &gauge_messages.ProtoItem{
		ItemType: gauge_messages.ProtoItem_Step,
		Step: &gauge_messages.ProtoStep{
			ActualText: "调用接口",
			StepExecutionResult: &gauge_messages.ProtoStepExecutionResult{
				ExecutionResult: &gauge_messages.ProtoExecutionResult{
					Failed:       true,
					ErrorMessage: "connection refused",
					ErrorType:    gauge_messages.ProtoExecutionResult_VERIFICATION,
				},
			},
		},
	}
	suite := sampleSuite(scenarioWith("接口异常", gauge_messages.ExecutionStatus_FAILED, step))
	model := Convert(suite, testOpts(t))
	if model.Results[0].Status != statusBroken {
		t.Fatalf("status: %s", model.Results[0].Status)
	}
}

func TestRetriesDoNotMarkFlakyByDefault(t *testing.T) {
	scn := passedScenario("重试场景")
	scn.RetriesCount = 1
	model := Convert(sampleSuite(scn), testOpts(t))
	if model.Results[0].StatusDetails != nil && model.Results[0].StatusDetails.Flaky {
		t.Fatal("retriesCount=1 should not mark test as flaky")
	}
	scn.RetriesCount = 2
	model = Convert(sampleSuite(scn), testOpts(t))
	if model.Results[0].StatusDetails == nil || !model.Results[0].StatusDetails.Flaky {
		t.Fatal("retriesCount>1 should mark test as flaky")
	}
}

func TestConvertSkippedScenario(t *testing.T) {
	scn := &gauge_messages.ProtoScenario{
		ScenarioHeading: "尚未实现",
		ExecutionStatus: gauge_messages.ExecutionStatus_SKIPPED,
		SkipErrors:      []string{"No implementation found"},
		ExecutionTime:   0,
	}
	suite := sampleSuite(scn)
	model := Convert(suite, testOpts(t))
	if model.Results[0].Status != statusSkipped {
		t.Fatalf("status: %s", model.Results[0].Status)
	}
	if model.Results[0].StatusDetails.Message != "No implementation found" {
		t.Fatalf("skip message: %s", model.Results[0].StatusDetails.Message)
	}
}

func TestConvertTableDrivenParameters(t *testing.T) {
	table := &gauge_messages.ProtoTable{
		Headers: &gauge_messages.ProtoTableRow{Cells: []string{"user", "role"}},
		Rows: []*gauge_messages.ProtoTableRow{
			{Cells: []string{"alice", "admin"}},
			{Cells: []string{"bob", "user"}},
		},
	}
	scn := passedScenario("数据驱动登录")
	suite := &gauge_messages.ProtoSuiteResult{
		ProjectName:   "demo",
		Environment:   "default",
		TimestampISO:  "2026-09-14T06:00:00Z",
		ExecutionTime: 100,
		SpecResults: []*gauge_messages.ProtoSpecResult{{
			TimestampISO:  "2026-09-14T06:00:00Z",
			ExecutionTime: 100,
			ProtoSpec: &gauge_messages.ProtoSpec{
				SpecHeading: "用户认证",
				FileName:    "specs/login.spec",
				Items: []*gauge_messages.ProtoItem{
					{ItemType: gauge_messages.ProtoItem_Table, Table: table},
					{
						ItemType: gauge_messages.ProtoItem_TableDrivenScenario,
						TableDrivenScenario: &gauge_messages.ProtoTableDrivenScenario{
							Scenario:          scn,
							TableRowIndex:     1,
							IsSpecTableDriven: true,
						},
					},
				},
			},
		}},
	}
	model := Convert(suite, testOpts(t))
	tr := model.Results[0]
	if !hasParam(tr.Parameters, "spec.user", "bob") || !hasParam(tr.Parameters, "spec.role", "user") {
		t.Fatalf("parameters: %+v", tr.Parameters)
	}
	if len(tr.Attachments) == 0 {
		t.Fatal("expected data table attachment")
	}
}

func TestConvertConceptNestedSteps(t *testing.T) {
	concept := &gauge_messages.ProtoItem{
		ItemType: gauge_messages.ProtoItem_Concept,
		Concept: &gauge_messages.ProtoConcept{
			ConceptStep: &gauge_messages.ProtoStep{ActualText: "完成结账"},
			Steps: []*gauge_messages.ProtoItem{
				passedStep("选择支付方式"),
				passedStep("确认订单"),
			},
			ConceptExecutionResult: &gauge_messages.ProtoStepExecutionResult{
				ExecutionResult: &gauge_messages.ProtoExecutionResult{ExecutionTime: 80},
			},
		},
	}
	suite := sampleSuite(passedScenario("购物", concept))
	model := Convert(suite, testOpts(t))
	if len(model.Results[0].Steps) != 1 {
		t.Fatalf("expected concept as one step, got %d", len(model.Results[0].Steps))
	}
	if got := len(model.Results[0].Steps[0].Steps); got != 2 {
		t.Fatalf("nested steps: %d", got)
	}
}

func TestConvertTagsToLabelsAndLinks(t *testing.T) {
	scn := passedScenario("下单")
	scn.Tags = []string{"severity:critical", "owner:qa", "issue:BUG-12", "smoke"}
	spec := sampleSuite(scn)
	spec.SpecResults[0].ProtoSpec.Tags = []string{"epic:交易", "feature:订单"}
	model := Convert(spec, Options{
		ProjectRoot:  t.TempDir(),
		IssuePattern: "https://jira.example.com/%s",
		Now:          func() time.Time { return time.Date(2026, 9, 14, 6, 0, 0, 0, time.UTC) },
		ReadFile:     func(string) ([]byte, error) { return nil, os.ErrNotExist },
		FileExists:   func(string) bool { return false },
		LookupEnv:    func(string) string { return "" },
		Host:         "ci-host",
	})
	tr := model.Results[0]
	assertLabel(t, tr.Labels, "severity", "critical")
	assertLabel(t, tr.Labels, "owner", "qa")
	assertLabel(t, tr.Labels, "epic", "交易")
	assertLabel(t, tr.Labels, "feature", "订单")
	assertLabel(t, tr.Labels, "tag", "smoke")
	if len(tr.Links) != 1 || tr.Links[0].URL != "https://jira.example.com/BUG-12" {
		t.Fatalf("links: %+v", tr.Links)
	}
}

func TestConvertHooksAsFixtures(t *testing.T) {
	suite := sampleSuite(passedScenario("ok"))
	suite.PreHookMessages = []string{"suite setup"}
	suite.PreHookFailure = &gauge_messages.ProtoHookFailure{ErrorMessage: "db down", StackTrace: "hook.go:1"}
	suite.SpecResults[0].ProtoSpec.PostHookMessages = []string{"spec teardown"}
	model := Convert(suite, testOpts(t))
	if len(model.Containers) == 0 || len(model.Containers[0].Befores) == 0 {
		t.Fatal("expected suite before fixture")
	}
	before := model.Containers[0].Befores[0]
	if before.Status != statusBroken {
		t.Fatalf("before status: %s", before.Status)
	}
	found := false
	for _, c := range model.Containers {
		if len(c.Afters) > 0 {
			found = true
		}
	}
	if !found {
		t.Fatal("expected spec after fixture")
	}
}

func TestConvertParseErrors(t *testing.T) {
	suite := &gauge_messages.ProtoSuiteResult{
		ProjectName: "demo",
		SpecResults: []*gauge_messages.ProtoSpecResult{{
			ProtoSpec: &gauge_messages.ProtoSpec{SpecHeading: "坏规格", FileName: "specs/bad.spec"},
			Errors: []*gauge_messages.Error{{
				Type:       gauge_messages.Error_PARSE_ERROR,
				Filename:   "specs/bad.spec",
				LineNumber: 3,
				Message:    "Step is not defined",
			}},
		}},
	}
	model := Convert(suite, testOpts(t))
	if model.Results[0].Status != statusBroken {
		t.Fatalf("status: %s", model.Results[0].Status)
	}
}

func TestExecutorFromGitHub(t *testing.T) {
	suite := sampleSuite(passedScenario("ok"))
	model := Convert(suite, Options{
		Now:        func() time.Time { return time.Unix(0, 0).UTC() },
		ReadFile:   func(string) ([]byte, error) { return nil, os.ErrNotExist },
		FileExists: func(string) bool { return false },
		LookupEnv: func(key string) string {
			return map[string]string{
				"GITHUB_ACTIONS":    "true",
				"GITHUB_SERVER_URL": "https://github.com",
				"GITHUB_REPOSITORY": "acme/app",
				"GITHUB_RUN_ID":     "99",
				"GITHUB_RUN_NUMBER": "7",
				"GITHUB_WORKFLOW":   "ci",
			}[key]
		},
	})
	if model.Executor == nil || model.Executor.Type != "github" {
		t.Fatalf("executor: %+v", model.Executor)
	}
	if model.Executor.BuildURL != "https://github.com/acme/app/actions/runs/99" {
		t.Fatalf("build url: %s", model.Executor.BuildURL)
	}
}

func testOpts(t *testing.T) Options {
	t.Helper()
	return Options{
		ProjectRoot: t.TempDir(),
		Host:        "test-host",
		Now:         func() time.Time { return time.Date(2026, 9, 14, 6, 0, 0, 0, time.UTC) },
		ReadFile:    os.ReadFile,
		FileExists: func(path string) bool {
			st, err := os.Stat(path)
			return err == nil && !st.IsDir()
		},
		LookupEnv: func(string) string { return "" },
	}
}

func sampleSuite(scn *gauge_messages.ProtoScenario) *gauge_messages.ProtoSuiteResult {
	return &gauge_messages.ProtoSuiteResult{
		ProjectName:   "demo",
		Environment:   "default",
		TimestampISO:  "2026-09-14T06:00:00Z",
		ExecutionTime: 120,
		SpecResults: []*gauge_messages.ProtoSpecResult{{
			TimestampISO:  "2026-09-14T06:00:00Z",
			ExecutionTime: 120,
			ScenarioCount: 1,
			ProtoSpec: &gauge_messages.ProtoSpec{
				SpecHeading: "用户认证",
				FileName:    "specs/login.spec",
				Items: []*gauge_messages.ProtoItem{{
					ItemType: gauge_messages.ProtoItem_Scenario,
					Scenario: scn,
				}},
			},
		}},
	}
}

func passedScenario(name string, items ...*gauge_messages.ProtoItem) *gauge_messages.ProtoScenario {
	return scenarioWith(name, gauge_messages.ExecutionStatus_PASSED, items...)
}

func scenarioWith(name string, status gauge_messages.ExecutionStatus, items ...*gauge_messages.ProtoItem) *gauge_messages.ProtoScenario {
	return &gauge_messages.ProtoScenario{
		ScenarioHeading: name,
		ExecutionStatus: status,
		ExecutionTime:   90,
		ScenarioItems:   items,
	}
}

func passedStep(text string) *gauge_messages.ProtoItem {
	return &gauge_messages.ProtoItem{
		ItemType: gauge_messages.ProtoItem_Step,
		Step: &gauge_messages.ProtoStep{
			ActualText: text,
			Fragments: []*gauge_messages.Fragment{
				{FragmentType: gauge_messages.Fragment_Text, Text: text},
			},
			StepExecutionResult: &gauge_messages.ProtoStepExecutionResult{
				ExecutionResult: &gauge_messages.ProtoExecutionResult{ExecutionTime: 10},
			},
		},
	}
}

func assertLabel(t *testing.T, labels []Label, name, value string) {
	t.Helper()
	for _, l := range labels {
		if l.Name == name && l.Value == value {
			return
		}
	}
	t.Fatalf("missing label %s=%s in %+v", name, value, labels)
}

func hasParam(params []Parameter, name, value string) bool {
	for _, p := range params {
		if p.Name == name && p.Value == value {
			return true
		}
	}
	return false
}

func TestSpecNestingFlat(t *testing.T) {
	pSuite, sSuite, subSuite := specNesting("specs/login.spec")
	if pSuite != "" {
		t.Fatalf("parentSuite: %s", pSuite)
	}
	if sSuite != "login" {
		t.Fatalf("suite: %s", sSuite)
	}
	if subSuite != "" {
		t.Fatalf("subSuite: %s", subSuite)
	}
}

func TestSpecNestingOneLevel(t *testing.T) {
	pSuite, sSuite, subSuite := specNesting("specs/auth/login.spec")
	if pSuite != "auth" {
		t.Fatalf("parentSuite: %s", pSuite)
	}
	if sSuite != "login" {
		t.Fatalf("suite: %s", sSuite)
	}
	if subSuite != "" {
		t.Fatalf("subSuite: %s", subSuite)
	}
}

func TestSpecNestingTwoLevels(t *testing.T) {
	pSuite, sSuite, subSuite := specNesting("specs/api/users/create.spec")
	if pSuite != "api" {
		t.Fatalf("parentSuite: %s", pSuite)
	}
	if sSuite != "users" {
		t.Fatalf("suite: %s", sSuite)
	}
	if subSuite != "create" {
		t.Fatalf("subSuite: %s", subSuite)
	}
}

func TestSpecNestingDeepNesting(t *testing.T) {
	pSuite, sSuite, subSuite := specNesting("specs/a/b/c/d.spec")
	if pSuite != "a" {
		t.Fatalf("parentSuite: %s", pSuite)
	}
	if sSuite != "b" {
		t.Fatalf("suite: %s", sSuite)
	}
	if subSuite != "c/d" {
		t.Fatalf("subSuite: %s", subSuite)
	}
}

func TestConvertNestedDirectoryLabels(t *testing.T) {
	suite := &gauge_messages.ProtoSuiteResult{
		ProjectName:   "demo",
		TimestampISO:  "2026-09-14T06:00:00Z",
		ExecutionTime: 120,
		SpecResults: []*gauge_messages.ProtoSpecResult{{
			TimestampISO:  "2026-09-14T06:00:00Z",
			ExecutionTime: 120,
			ProtoSpec: &gauge_messages.ProtoSpec{
				SpecHeading: "登录验证",
				FileName:    "specs/auth/login.spec",
				Items: []*gauge_messages.ProtoItem{{
					ItemType: gauge_messages.ProtoItem_Scenario,
					Scenario: passedScenario("登录成功"),
				}},
			},
		}},
	}
	model := Convert(suite, testOpts(t))
	tr := model.Results[0]
	assertLabel(t, tr.Labels, "parentSuite", "auth")
	assertLabel(t, tr.Labels, "suite", "login")
	assertLabel(t, tr.Labels, "feature", "登录验证")
	assertLabel(t, tr.Labels, "package", "specs/auth/login.spec")
}

func TestConvertNestedDirectoryWithSubSuite(t *testing.T) {
	suite := &gauge_messages.ProtoSuiteResult{
		ProjectName:   "demo",
		TimestampISO:  "2026-09-14T06:00:00Z",
		ExecutionTime: 120,
		SpecResults: []*gauge_messages.ProtoSpecResult{{
			TimestampISO:  "2026-09-14T06:00:00Z",
			ExecutionTime: 120,
			ProtoSpec: &gauge_messages.ProtoSpec{
				SpecHeading: "创建用户",
				FileName:    "specs/api/users/create.spec",
				Items: []*gauge_messages.ProtoItem{{
					ItemType: gauge_messages.ProtoItem_Scenario,
					Scenario: passedScenario("创建成功"),
				}},
			},
		}},
	}
	model := Convert(suite, testOpts(t))
	tr := model.Results[0]
	assertLabel(t, tr.Labels, "parentSuite", "api")
	assertLabel(t, tr.Labels, "suite", "users")
	assertLabel(t, tr.Labels, "subSuite", "create")
	assertLabel(t, tr.Labels, "package", "specs/api/users/create.spec")
}

func TestConvertFlatSpecUsesSpecHeading(t *testing.T) {
	suite := sampleSuite(passedScenario("登录成功"))
	model := Convert(suite, testOpts(t))
	tr := model.Results[0]
	assertLabel(t, tr.Labels, "parentSuite", "demo")
	assertLabel(t, tr.Labels, "suite", "用户认证")
}

func TestFullNameStripsSpecsPrefix(t *testing.T) {
	spec := &gauge_messages.ProtoSpec{
		FileName: "specs/auth/login.spec",
	}
	got := fullName(spec, "登录成功")
	if got != "auth/login.spec#登录成功" {
		t.Fatalf("fullName: %s", got)
	}
}
