package converter

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/getgauge/gauge-proto/go/gauge_messages"
	"github.com/google/uuid"
)

const (
	statusPassed  = "passed"
	statusFailed  = "failed"
	statusBroken  = "broken"
	statusSkipped = "skipped"
	stageFinished = "finished"
	stagePending  = "pending"
)

// Options control how Gauge results are mapped to Allure results.
type Options struct {
	ProjectRoot    string
	ScreenshotsDir string
	IssuePattern   string
	TMSPattern     string
	Host           string
	ReportName     string
	Now            func() time.Time
	ReadFile       func(path string) ([]byte, error)
	FileExists     func(path string) bool
	LookupEnv      func(key string) string
}

func (o *Options) withDefaults() {
	if o.Now == nil {
		o.Now = time.Now
	}
	if o.ReadFile == nil {
		o.ReadFile = os.ReadFile
	}
	if o.FileExists == nil {
		o.FileExists = func(path string) bool {
			st, err := os.Stat(path)
			return err == nil && !st.IsDir()
		}
	}
	if o.LookupEnv == nil {
		o.LookupEnv = os.Getenv
	}
	if o.Host == "" {
		if h, err := os.Hostname(); err == nil {
			o.Host = h
		}
	}
}

// Convert turns a Gauge suite result into Allure 2/3 compatible files.
func Convert(suite *gauge_messages.ProtoSuiteResult, opts Options) *Model {
	opts.withDefaults()
	model := &Model{
		Environment: environmentFrom(suite, opts),
		Categories:  defaultCategories(),
		Executor:    executorFrom(opts),
	}
	if suite == nil {
		return model
	}

	suiteStart, suiteStop := timeWindow(suite.GetTimestampISO(), suite.GetExecutionTime(), opts.Now())
	suiteContainer := TestResultContainer{
		UUID:  newUUID(),
		Name:  firstNonEmpty(suite.GetProjectName(), opts.ReportName, "Gauge Suite"),
		Start: suiteStart,
		Stop:  suiteStop,
	}
	suiteContainer.Befores = append(suiteContainer.Befores, hookFixtures("Before Suite", suite.GetPreHookFailure(), suite.GetPreHookMessages(), suite.GetPreHookScreenshotFiles(), suiteStart, model, opts)...)
	suiteContainer.Afters = append(suiteContainer.Afters, hookFixtures("After Suite", suite.GetPostHookFailure(), suite.GetPostHookMessages(), suite.GetPostHookScreenshotFiles(), suiteStop, model, opts)...)

	for _, specResult := range suite.GetSpecResults() {
		specContainer, tests := convertSpec(specResult, suite, model, opts)
		for _, test := range tests {
			suiteContainer.Children = append(suiteContainer.Children, test.UUID)
			model.Results = append(model.Results, test)
		}
		if specContainer.UUID != "" {
			model.Containers = append(model.Containers, specContainer)
		}
	}

	if len(suiteContainer.Children) > 0 || len(suiteContainer.Befores) > 0 || len(suiteContainer.Afters) > 0 {
		model.Containers = append([]TestResultContainer{suiteContainer}, model.Containers...)
	}
	return model
}

func convertSpec(specResult *gauge_messages.ProtoSpecResult, suite *gauge_messages.ProtoSuiteResult, model *Model, opts Options) (TestResultContainer, []TestResult) {
	if specResult == nil {
		return TestResultContainer{}, nil
	}
	spec := specResult.GetProtoSpec()
	specName := specName(spec)
	start, stop := timeWindow(specResult.GetTimestampISO(), specResult.GetExecutionTime(), opts.Now())
	container := TestResultContainer{
		UUID:  newUUID(),
		Name:  specName,
		Start: start,
		Stop:  stop,
	}
	if spec != nil {
		container.Befores = append(container.Befores, specHookFixtures("Before Spec", spec.GetPreHookFailures(), spec.GetPreHookMessages(), spec.GetPreHookScreenshotFiles(), start, model, opts)...)
		container.Afters = append(container.Afters, specHookFixtures("After Spec", spec.GetPostHookFailures(), spec.GetPostHookMessages(), spec.GetPostHookScreenshotFiles(), stop, model, opts)...)
	}

	var tests []TestResult
	if hasParseErrors(specResult.GetErrors()) {
		test := parseErrorTest(specResult, suite, specName, start, stop, opts)
		container.Children = append(container.Children, test.UUID)
		tests = append(tests, test)
		return container, tests
	}
	if spec == nil {
		return container, tests
	}

	specTable := findSpecTable(spec)
	for _, item := range spec.GetItems() {
		switch item.GetItemType() {
		case gauge_messages.ProtoItem_Scenario:
			test := convertScenario(item.GetScenario(), nil, specResult, suite, specTable, start, model, opts)
			container.Children = append(container.Children, test.UUID)
			tests = append(tests, test)
		case gauge_messages.ProtoItem_TableDrivenScenario:
			td := item.GetTableDrivenScenario()
			if td == nil {
				continue
			}
			test := convertScenario(td.GetScenario(), td, specResult, suite, specTable, start, model, opts)
			container.Children = append(container.Children, test.UUID)
			tests = append(tests, test)
		}
	}
	return container, tests
}

func convertScenario(scenario *gauge_messages.ProtoScenario, tableDriven *gauge_messages.ProtoTableDrivenScenario, specResult *gauge_messages.ProtoSpecResult, suite *gauge_messages.ProtoSuiteResult, specTable *gauge_messages.ProtoTable, specStart int64, model *Model, opts Options) TestResult {
	spec := specResult.GetProtoSpec()
	name := scenarioName(scenario)
	start, stop := timeWindow(specResult.GetTimestampISO(), scenario.GetExecutionTime(), time.UnixMilli(specStart))
	if specStart > 0 {
		start = specStart
		stop = specStart + scenario.GetExecutionTime()
	}

	test := TestResult{
		UUID:     newUUID(),
		Name:     name,
		FullName: fullName(spec, name),
		Stage:    stageFinished,
		Start:    start,
		Stop:     stop,
	}
	status, details := scenarioStatus(scenario)
	test.Status = status
	test.StatusDetails = details
	if scenario != nil && scenario.GetRetriesCount() > 1 {
		if test.StatusDetails == nil {
			test.StatusDetails = &StatusDetails{}
		}
		test.StatusDetails.Flaky = true
	}

	params := scenarioParameters(tableDriven, specTable)
	test.Parameters = params
	test.TestCaseID = md5Hex(test.FullName)
	test.HistoryID = historyID(test.TestCaseID, params)

	test.Labels = scenarioLabels(spec, scenario, suite, opts)
	parsed := parseTags(mergeTags(spec, scenario), opts.IssuePattern, opts.TMSPattern)
	test.Labels = append(test.Labels, parsed.labels...)
	test.Links = append(test.Links, parsed.links...)

	if spec != nil && spec.GetFileName() != "" {
		test.Description = fmt.Sprintf("Specification: %s", spec.GetFileName())
	}

	cursor := start
	if scenario != nil {
		if len(scenario.GetPreHookMessages()) > 0 || scenario.GetPreHookFailure() != nil || len(scenario.GetPreHookScreenshotFiles()) > 0 {
			test.Steps = append(test.Steps, hookStep("Before Scenario", scenario.GetPreHookFailure(), scenario.GetPreHookMessages(), scenario.GetPreHookScreenshotFiles(), cursor, model, opts))
		}
		if ctx := convertItemGroup("Context", scenario.GetContexts(), cursor, model, opts); ctx != nil {
			test.Steps = append(test.Steps, *ctx)
			cursor = ctx.Stop
		}
		test.Steps = append(test.Steps, convertItems(scenario.GetScenarioItems(), cursor, model, opts)...)
		if len(test.Steps) > 0 {
			cursor = test.Steps[len(test.Steps)-1].Stop
		}
		if td := convertItemGroup("Teardown", scenario.GetTearDownSteps(), cursor, model, opts); td != nil {
			test.Steps = append(test.Steps, *td)
			cursor = td.Stop
		}
		if len(scenario.GetPostHookMessages()) > 0 || scenario.GetPostHookFailure() != nil || len(scenario.GetPostHookScreenshotFiles()) > 0 {
			test.Steps = append(test.Steps, hookStep("After Scenario", scenario.GetPostHookFailure(), scenario.GetPostHookMessages(), scenario.GetPostHookScreenshotFiles(), cursor, model, opts))
		}
	}

	if tableDriven != nil {
		if html := tableHTML(tableFromDriven(tableDriven, specTable)); html != "" {
			att := model.addBytes("Data table", "text/html", ".html", []byte(html))
			test.Attachments = append(test.Attachments, att)
		}
	}
	return test
}

func parseErrorTest(specResult *gauge_messages.ProtoSpecResult, suite *gauge_messages.ProtoSuiteResult, specName string, start, stop int64, opts Options) TestResult {
	var messages []string
	for _, e := range specResult.GetErrors() {
		kind := "Parse"
		if e.GetType() == gauge_messages.Error_VALIDATION_ERROR {
			kind = "Validation"
		}
		messages = append(messages, fmt.Sprintf("[%s Error] %s:%d %s", kind, e.GetFilename(), e.GetLineNumber(), e.GetMessage()))
	}
	msg := strings.Join(messages, "\n")
	test := TestResult{
		UUID:     newUUID(),
		Name:     specName,
		FullName: specName,
		Status:   statusBroken,
		Stage:    stageFinished,
		Start:    start,
		Stop:     stop,
		StatusDetails: &StatusDetails{
			Message: "Parse/Validation Errors",
			Trace:   msg,
		},
		Labels: scenarioLabels(specResult.GetProtoSpec(), nil, suite, opts),
	}
	test.TestCaseID = md5Hex(test.FullName)
	test.HistoryID = historyID(test.TestCaseID, nil)
	return test
}

func scenarioLabels(spec *gauge_messages.ProtoSpec, scenario *gauge_messages.ProtoScenario, suite *gauge_messages.ProtoSuiteResult, opts Options) []Label {
	labels := []Label{
		{Name: "framework", Value: "gauge"},
		{Name: "language", Value: "gauge"},
	}
	if opts.Host != "" {
		labels = append(labels, Label{Name: "host", Value: opts.Host})
	}
	parent := firstNonEmpty(suite.GetProjectName(), "Gauge")
	labels = append(labels, Label{Name: "parentSuite", Value: parent})
	if spec != nil {
		name := specName(spec)
		labels = append(labels,
			Label{Name: "suite", Value: name},
			Label{Name: "feature", Value: name},
			Label{Name: "package", Value: posixPath(spec.GetFileName())},
			Label{Name: "testClass", Value: name},
		)
	}
	if scenario != nil {
		labels = append(labels, Label{Name: "testMethod", Value: scenario.GetScenarioHeading()})
		if scenario.GetRetriesCount() > 1 {
			labels = append(labels, Label{Name: "tag", Value: fmt.Sprintf("retries:%d", scenario.GetRetriesCount())})
		}
	}
	if suite != nil && suite.GetEnvironment() != "" {
		labels = append(labels, Label{Name: "tag", Value: "env:" + suite.GetEnvironment()})
	}
	return labels
}

func scenarioStatus(scenario *gauge_messages.ProtoScenario) (string, *StatusDetails) {
	if scenario == nil {
		return statusSkipped, &StatusDetails{Message: "scenario is empty"}
	}
	switch scenario.GetExecutionStatus() {
	case gauge_messages.ExecutionStatus_PASSED:
		return statusPassed, nil
	case gauge_messages.ExecutionStatus_SKIPPED, gauge_messages.ExecutionStatus_NOTEXECUTED:
		return statusSkipped, &StatusDetails{Message: strings.Join(scenario.GetSkipErrors(), "\n")}
	default:
		msg, trace, broken := firstScenarioFailure(scenario)
		status := statusFailed
		if broken {
			status = statusBroken
		}
		return status, &StatusDetails{Message: msg, Trace: trace}
	}
}

func firstScenarioFailure(scenario *gauge_messages.ProtoScenario) (string, string, bool) {
	if scenario.GetPreHookFailure() != nil {
		return scenario.GetPreHookFailure().GetErrorMessage(), scenario.GetPreHookFailure().GetStackTrace(), true
	}
	if msg, trace, broken, ok := firstItemFailure(scenario.GetContexts()); ok {
		return msg, trace, broken
	}
	if msg, trace, broken, ok := firstItemFailure(scenario.GetScenarioItems()); ok {
		return msg, trace, broken
	}
	if msg, trace, broken, ok := firstItemFailure(scenario.GetTearDownSteps()); ok {
		return msg, trace, broken
	}
	if scenario.GetPostHookFailure() != nil {
		return scenario.GetPostHookFailure().GetErrorMessage(), scenario.GetPostHookFailure().GetStackTrace(), true
	}
	return "Scenario failed", "", false
}

func firstItemFailure(items []*gauge_messages.ProtoItem) (string, string, bool, bool) {
	for _, item := range items {
		switch item.GetItemType() {
		case gauge_messages.ProtoItem_Step:
			if msg, trace, broken, ok := stepFailure(item.GetStep()); ok {
				return msg, trace, broken, true
			}
		case gauge_messages.ProtoItem_Concept:
			concept := item.GetConcept()
			if concept.GetConceptExecutionResult().GetPreHookFailure() != nil {
				f := concept.GetConceptExecutionResult().GetPreHookFailure()
				return f.GetErrorMessage(), f.GetStackTrace(), true, true
			}
			if msg, trace, broken, ok := firstItemFailure(concept.GetSteps()); ok {
				return msg, trace, broken, true
			}
			if result := concept.GetConceptExecutionResult().GetExecutionResult(); result != nil && result.GetFailed() {
				return result.GetErrorMessage(), result.GetStackTrace(), result.GetErrorType() != gauge_messages.ProtoExecutionResult_ASSERTION, true
			}
		}
	}
	return "", "", false, false
}

func stepFailure(step *gauge_messages.ProtoStep) (string, string, bool, bool) {
	if step == nil || step.GetStepExecutionResult() == nil {
		return "", "", false, false
	}
	ser := step.GetStepExecutionResult()
	if ser.GetPreHookFailure() != nil {
		return ser.GetPreHookFailure().GetErrorMessage(), ser.GetPreHookFailure().GetStackTrace(), true, true
	}
	if result := ser.GetExecutionResult(); result != nil && result.GetFailed() {
		return result.GetErrorMessage(), result.GetStackTrace(), result.GetErrorType() != gauge_messages.ProtoExecutionResult_ASSERTION, true
	}
	if ser.GetPostHookFailure() != nil {
		return ser.GetPostHookFailure().GetErrorMessage(), ser.GetPostHookFailure().GetStackTrace(), true, true
	}
	return "", "", false, false
}

func scenarioParameters(tableDriven *gauge_messages.ProtoTableDrivenScenario, specTable *gauge_messages.ProtoTable) []Parameter {
	if tableDriven == nil {
		return nil
	}
	var params []Parameter
	if tableDriven.GetIsSpecTableDriven() {
		params = append(params, rowParameters(specTable, int(tableDriven.GetTableRowIndex()), "spec")...)
	}
	if tableDriven.GetIsScenarioTableDriven() {
		table := tableDriven.GetScenarioDataTable()
		if table == nil && tableDriven.GetScenarioTableRow() != nil {
			params = append(params, rowParametersFromPair(tableDriven.GetScenarioTableRow(), int(tableDriven.GetScenarioTableRowIndex()), "scenario")...)
		} else {
			params = append(params, rowParameters(table, int(tableDriven.GetScenarioTableRowIndex()), "scenario")...)
		}
	}
	return params
}

func rowParameters(table *gauge_messages.ProtoTable, rowIndex int, prefix string) []Parameter {
	if table == nil || table.GetHeaders() == nil {
		return nil
	}
	headers := table.GetHeaders().GetCells()
	rows := table.GetRows()
	if rowIndex < 0 || rowIndex >= len(rows) {
		return nil
	}
	cells := rows[rowIndex].GetCells()
	var params []Parameter
	for i, header := range headers {
		value := ""
		if i < len(cells) {
			value = cells[i]
		}
		name := header
		if prefix != "" {
			name = fmt.Sprintf("%s.%s", prefix, header)
		}
		params = append(params, Parameter{Name: name, Value: value})
	}
	return params
}

func rowParametersFromPair(rowTable *gauge_messages.ProtoTable, rowIndex int, prefix string) []Parameter {
	return rowParameters(rowTable, rowIndex, prefix)
}

func tableFromDriven(tableDriven *gauge_messages.ProtoTableDrivenScenario, specTable *gauge_messages.ProtoTable) *gauge_messages.ProtoTable {
	if tableDriven.GetIsScenarioTableDriven() && tableDriven.GetScenarioDataTable() != nil {
		return tableDriven.GetScenarioDataTable()
	}
	if tableDriven.GetIsSpecTableDriven() {
		return specTable
	}
	return tableDriven.GetScenarioTableRow()
}

func findSpecTable(spec *gauge_messages.ProtoSpec) *gauge_messages.ProtoTable {
	for _, item := range spec.GetItems() {
		if item.GetItemType() == gauge_messages.ProtoItem_Table {
			return item.GetTable()
		}
	}
	return nil
}

func mergeTags(spec *gauge_messages.ProtoSpec, scenario *gauge_messages.ProtoScenario) []string {
	var tags []string
	if spec != nil {
		tags = append(tags, spec.GetTags()...)
	}
	if scenario != nil {
		tags = append(tags, scenario.GetTags()...)
	}
	return tags
}

func specName(spec *gauge_messages.ProtoSpec) string {
	if spec == nil {
		return "specification"
	}
	if strings.TrimSpace(spec.GetSpecHeading()) != "" {
		return spec.GetSpecHeading()
	}
	return filepath.Base(spec.GetFileName())
}

func scenarioName(scenario *gauge_messages.ProtoScenario) string {
	if scenario == nil || strings.TrimSpace(scenario.GetScenarioHeading()) == "" {
		return "scenario"
	}
	return scenario.GetScenarioHeading()
}

func fullName(spec *gauge_messages.ProtoSpec, scenario string) string {
	file := ""
	if spec != nil {
		file = posixPath(spec.GetFileName())
		if file == "" {
			file = specName(spec)
		}
	}
	return strings.Trim(file+"#"+scenario, "#")
}

func hasParseErrors(errors []*gauge_messages.Error) bool {
	for _, e := range errors {
		if e.GetType() == gauge_messages.Error_PARSE_ERROR || e.GetType() == gauge_messages.Error_VALIDATION_ERROR {
			return true
		}
	}
	return false
}

func hookFixtures(name string, failure *gauge_messages.ProtoHookFailure, messages, screenshots []string, at int64, model *Model, opts Options) []Fixture {
	if failure == nil && len(messages) == 0 && len(screenshots) == 0 {
		return nil
	}
	fx := Fixture{
		Name:   name,
		Status: statusPassed,
		Stage:  stageFinished,
		Start:  at,
		Stop:   at,
	}
	if failure != nil {
		fx.Status = statusBroken
		fx.StatusDetails = &StatusDetails{Message: failure.GetErrorMessage(), Trace: failure.GetStackTrace()}
		if failure.GetFailureScreenshotFile() != "" {
			fx.Attachments = append(fx.Attachments, model.addScreenshot("Failure screenshot", failure.GetFailureScreenshotFile(), opts))
		}
	}
	for _, msg := range messages {
		fx.Steps = append(fx.Steps, Step{Name: msg, Status: statusPassed, Stage: stageFinished, Start: at, Stop: at})
	}
	for i, shot := range screenshots {
		fx.Attachments = append(fx.Attachments, model.addScreenshot(fmt.Sprintf("Screenshot %d", i+1), shot, opts))
	}
	return []Fixture{fx}
}

func specHookFixtures(name string, failures []*gauge_messages.ProtoHookFailure, messages, screenshots []string, at int64, model *Model, opts Options) []Fixture {
	var out []Fixture
	if len(failures) == 0 && (len(messages) > 0 || len(screenshots) > 0) {
		return hookFixtures(name, nil, messages, screenshots, at, model, opts)
	}
	for i, failure := range failures {
		label := name
		if len(failures) > 1 {
			label = fmt.Sprintf("%s #%d", name, i+1)
		}
		out = append(out, hookFixtures(label, failure, messages, screenshots, at, model, opts)...)
		messages = nil
		screenshots = nil
	}
	return out
}

func hookStep(name string, failure *gauge_messages.ProtoHookFailure, messages, screenshots []string, at int64, model *Model, opts Options) Step {
	step := Step{Name: name, Status: statusPassed, Stage: stageFinished, Start: at, Stop: at}
	if failure != nil {
		step.Status = statusBroken
		step.StatusDetails = &StatusDetails{Message: failure.GetErrorMessage(), Trace: failure.GetStackTrace()}
		if failure.GetFailureScreenshotFile() != "" {
			step.Attachments = append(step.Attachments, model.addScreenshot("Failure screenshot", failure.GetFailureScreenshotFile(), opts))
		}
	}
	for _, msg := range messages {
		step.Steps = append(step.Steps, Step{Name: msg, Status: statusPassed, Stage: stageFinished, Start: at, Stop: at})
	}
	for i, shot := range screenshots {
		step.Attachments = append(step.Attachments, model.addScreenshot(fmt.Sprintf("Screenshot %d", i+1), shot, opts))
	}
	return step
}

func environmentFrom(suite *gauge_messages.ProtoSuiteResult, opts Options) map[string]string {
	env := map[string]string{
		"framework": "gauge",
	}
	if suite != nil {
		if suite.GetProjectName() != "" {
			env["project"] = suite.GetProjectName()
		}
		if suite.GetEnvironment() != "" {
			env["environment"] = suite.GetEnvironment()
		}
		if suite.GetTags() != "" {
			env["tags"] = suite.GetTags()
		}
	}
	if opts.Host != "" {
		env["host"] = opts.Host
	}
	if v := opts.LookupEnv("GAUGE_LANGUAGE"); v != "" {
		env["language"] = v
	}
	return env
}

func defaultCategories() []Category {
	return []Category{
		{Name: "Assertion failures", MatchedStatuses: []string{statusFailed}},
		{Name: "Broken tests", MatchedStatuses: []string{statusBroken}},
		{Name: "Skipped tests", MatchedStatuses: []string{statusSkipped}},
	}
}

func executorFrom(opts Options) *Executor {
	if opts.LookupEnv("GITHUB_ACTIONS") == "true" {
		server := strings.TrimRight(opts.LookupEnv("GITHUB_SERVER_URL"), "/")
		repo := opts.LookupEnv("GITHUB_REPOSITORY")
		runID := opts.LookupEnv("GITHUB_RUN_ID")
		buildURL := fmt.Sprintf("%s/%s/actions/runs/%s", server, repo, runID)
		return &Executor{
			Name:       "GitHub Actions",
			Type:       "github",
			URL:        server,
			BuildOrder: opts.LookupEnv("GITHUB_RUN_NUMBER"),
			BuildName:  firstNonEmpty(opts.LookupEnv("GITHUB_WORKFLOW"), opts.LookupEnv("GITHUB_JOB")),
			BuildURL:   buildURL,
			ReportName: firstNonEmpty(opts.ReportName, "Allure Report"),
		}
	}
	if opts.LookupEnv("JENKINS_URL") != "" || opts.LookupEnv("BUILD_URL") != "" {
		return &Executor{
			Name:       "Jenkins",
			Type:       "jenkins",
			URL:        opts.LookupEnv("JENKINS_URL"),
			BuildOrder: opts.LookupEnv("BUILD_NUMBER"),
			BuildName:  opts.LookupEnv("JOB_NAME"),
			BuildURL:   opts.LookupEnv("BUILD_URL"),
			ReportName: firstNonEmpty(opts.ReportName, "Allure Report"),
		}
	}
	return nil
}

func (m *Model) addScreenshot(name, path string, opts Options) Attachment {
	if strings.TrimSpace(path) == "" {
		return Attachment{}
	}
	resolved := resolveFile(path, opts)
	if resolved == "" {
		return Attachment{}
	}
	ext := filepath.Ext(resolved)
	if ext == "" {
		ext = ".png"
	}
	displayName := screenshotDisplayName(name, resolved)
	file := File{
		Name:        newUUID() + "-attachment" + ext,
		ContentType: mimeFromExt(ext),
		SourcePath:  resolved,
	}
	if data, err := opts.ReadFile(resolved); err == nil {
		file.Bytes = data
		file.SourcePath = ""
	}
	m.Files = append(m.Files, file)
	return Attachment{Name: displayName, Type: file.ContentType, Source: file.Name}
}

func screenshotDisplayName(name, resolved string) string {
	base := filepath.Base(resolved)
	if base != "" && base != "." {
		return base
	}
	return name
}

func (m *Model) addBytes(name, contentType, ext string, data []byte) Attachment {
	file := File{
		Name:        newUUID() + "-attachment" + ext,
		ContentType: contentType,
		Bytes:       data,
	}
	m.Files = append(m.Files, file)
	return Attachment{Name: name, Type: contentType, Source: file.Name}
}

func resolveFile(path string, opts Options) string {
	if path == "" {
		return ""
	}
	base := filepath.Base(path)
	candidates := make([]string, 0, 6)
	if opts.ScreenshotsDir != "" {
		candidates = append(candidates, filepath.Join(opts.ScreenshotsDir, base))
	}
	candidates = append(candidates, path)
	if !filepath.IsAbs(path) && opts.ProjectRoot != "" {
		candidates = append(candidates,
			filepath.Join(opts.ProjectRoot, path),
			filepath.Join(opts.ProjectRoot, ".gauge", "screenshots", base),
		)
	}
	for _, candidate := range candidates {
		if opts.FileExists(candidate) {
			return candidate
		}
	}
	return ""
}

func mimeFromExt(ext string) string {
	switch strings.ToLower(ext) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".html", ".htm":
		return "text/html"
	case ".txt", ".log":
		return "text/plain"
	case ".json":
		return "application/json"
	case ".xml":
		return "application/xml"
	case ".mp4":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	default:
		return "application/octet-stream"
	}
}

func tableHTML(table *gauge_messages.ProtoTable) string {
	if table == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString("<table>")
	if headers := table.GetHeaders(); headers != nil && len(headers.GetCells()) > 0 {
		b.WriteString("<thead><tr>")
		for _, cell := range headers.GetCells() {
			fmt.Fprintf(&b, "<th>%s</th>", html.EscapeString(cell))
		}
		b.WriteString("</tr></thead>")
	}
	b.WriteString("<tbody>")
	for _, row := range table.GetRows() {
		b.WriteString("<tr>")
		for _, cell := range row.GetCells() {
			fmt.Fprintf(&b, "<td>%s</td>", html.EscapeString(cell))
		}
		b.WriteString("</tr>")
	}
	b.WriteString("</tbody></table>")
	return b.String()
}

func timeWindow(iso string, durationMs int64, fallback time.Time) (int64, int64) {
	start := fallback.UnixMilli()
	if iso != "" {
		if t, err := time.Parse(time.RFC3339Nano, iso); err == nil {
			start = t.UnixMilli()
		} else if t, err := time.Parse(time.RFC3339, iso); err == nil {
			start = t.UnixMilli()
		}
	}
	if durationMs < 0 {
		durationMs = 0
	}
	return start, start + durationMs
}

func posixPath(path string) string {
	return strings.ReplaceAll(path, "\\", "/")
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func newUUID() string {
	return uuid.NewString()
}
