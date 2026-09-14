package converter

// Allure 2/3 compatible result model written as *-result.json / *-container.json.

type Model struct {
	Results     []TestResult
	Containers  []TestResultContainer
	Files       []File
	Environment map[string]string
	Categories  []Category
	Executor    *Executor
}

type File struct {
	Name        string
	ContentType string
	Bytes       []byte
	SourcePath  string
}

type TestResult struct {
	UUID            string         `json:"uuid"`
	HistoryID       string         `json:"historyId,omitempty"`
	TestCaseID      string         `json:"testCaseId,omitempty"`
	FullName        string         `json:"fullName,omitempty"`
	Name            string         `json:"name"`
	Status          string         `json:"status"`
	StatusDetails   *StatusDetails `json:"statusDetails,omitempty"`
	Stage           string         `json:"stage,omitempty"`
	Description     string         `json:"description,omitempty"`
	DescriptionHTML string         `json:"descriptionHtml,omitempty"`
	Start           int64          `json:"start,omitempty"`
	Stop            int64          `json:"stop,omitempty"`
	Labels          []Label        `json:"labels,omitempty"`
	Links           []Link         `json:"links,omitempty"`
	Parameters      []Parameter    `json:"parameters,omitempty"`
	Steps           []Step         `json:"steps,omitempty"`
	Attachments     []Attachment   `json:"attachments,omitempty"`
}

type Step struct {
	Name          string         `json:"name"`
	Status        string         `json:"status,omitempty"`
	StatusDetails *StatusDetails `json:"statusDetails,omitempty"`
	Stage         string         `json:"stage,omitempty"`
	Start         int64          `json:"start,omitempty"`
	Stop          int64          `json:"stop,omitempty"`
	Steps         []Step         `json:"steps,omitempty"`
	Attachments   []Attachment   `json:"attachments,omitempty"`
	Parameters    []Parameter    `json:"parameters,omitempty"`
}

type TestResultContainer struct {
	UUID     string    `json:"uuid"`
	Name     string    `json:"name,omitempty"`
	Children []string  `json:"children,omitempty"`
	Befores  []Fixture `json:"befores,omitempty"`
	Afters   []Fixture `json:"afters,omitempty"`
	Start    int64     `json:"start,omitempty"`
	Stop     int64     `json:"stop,omitempty"`
}

type Fixture struct {
	Name          string         `json:"name"`
	Status        string         `json:"status,omitempty"`
	StatusDetails *StatusDetails `json:"statusDetails,omitempty"`
	Stage         string         `json:"stage,omitempty"`
	Start         int64          `json:"start,omitempty"`
	Stop          int64          `json:"stop,omitempty"`
	Steps         []Step         `json:"steps,omitempty"`
	Attachments   []Attachment   `json:"attachments,omitempty"`
	Parameters    []Parameter    `json:"parameters,omitempty"`
}

type StatusDetails struct {
	Known   bool   `json:"known,omitempty"`
	Muted   bool   `json:"muted,omitempty"`
	Flaky   bool   `json:"flaky,omitempty"`
	Message string `json:"message,omitempty"`
	Trace   string `json:"trace,omitempty"`
}

type Label struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Link struct {
	Type string `json:"type,omitempty"`
	Name string `json:"name,omitempty"`
	URL  string `json:"url"`
}

type Parameter struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Attachment struct {
	Name   string `json:"name"`
	Type   string `json:"type,omitempty"`
	Source string `json:"source"`
}

type Category struct {
	Name            string   `json:"name"`
	MatchedStatuses []string `json:"matchedStatuses,omitempty"`
	MessageRegex    string   `json:"messageRegex,omitempty"`
	TraceRegex      string   `json:"traceRegex,omitempty"`
}

type Executor struct {
	Name       string `json:"name,omitempty"`
	Type       string `json:"type,omitempty"`
	URL        string `json:"url,omitempty"`
	BuildOrder string `json:"buildOrder,omitempty"`
	BuildName  string `json:"buildName,omitempty"`
	BuildURL   string `json:"buildUrl,omitempty"`
	ReportName string `json:"reportName,omitempty"`
	ReportURL  string `json:"reportUrl,omitempty"`
}
