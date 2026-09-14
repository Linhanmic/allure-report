package converter

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/getgauge/gauge-proto/go/gauge_messages"
)

func convertItems(items []*gauge_messages.ProtoItem, start int64, model *Model, opts Options) []Step {
	cursor := start
	var steps []Step
	for _, item := range items {
		step, ok := convertItem(item, cursor, model, opts)
		if !ok {
			continue
		}
		steps = append(steps, step)
		if step.Stop > 0 {
			cursor = step.Stop
		}
	}
	return steps
}

func convertItemGroup(name string, items []*gauge_messages.ProtoItem, start int64, model *Model, opts Options) *Step {
	children := convertItems(items, start, model, opts)
	if len(children) == 0 {
		return nil
	}
	stop := start
	status := statusPassed
	for _, child := range children {
		if child.Stop > stop {
			stop = child.Stop
		}
		status = worstStatus(status, child.Status)
	}
	return &Step{
		Name:   name,
		Status: status,
		Stage:  stageFinished,
		Start:  start,
		Stop:   stop,
		Steps:  children,
	}
}

func convertItem(item *gauge_messages.ProtoItem, start int64, model *Model, opts Options) (Step, bool) {
	if item == nil {
		return Step{}, false
	}
	switch item.GetItemType() {
	case gauge_messages.ProtoItem_Step:
		return convertStep(item.GetStep(), start, model, opts)
	case gauge_messages.ProtoItem_Concept:
		return convertConcept(item.GetConcept(), start, model, opts)
	default:
		return Step{}, false
	}
}

func convertStep(step *gauge_messages.ProtoStep, start int64, model *Model, opts Options) (Step, bool) {
	if step == nil {
		return Step{}, false
	}
	duration := int64(0)
	if step.GetStepExecutionResult() != nil && step.GetStepExecutionResult().GetExecutionResult() != nil {
		duration = step.GetStepExecutionResult().GetExecutionResult().GetExecutionTime()
	}
	out := Step{
		Name:  firstNonEmpty(step.GetActualText(), step.GetParsedText(), "step"),
		Start: start,
		Stop:  start + duration,
		Stage: stageFinished,
	}
	out.Status, out.StatusDetails = executionStatus(step.GetStepExecutionResult())
	out.Parameters = stepParameters(step)
	out.Attachments = stepAttachments(step, model, opts)
	for _, msg := range messagesOf(step) {
		out.Steps = append(out.Steps, Step{
			Name:   msg,
			Status: statusPassed,
			Stage:  stageFinished,
			Start:  start,
			Stop:   start,
		})
	}
	if html := fragmentsTableHTML(step); html != "" {
		out.Attachments = append(out.Attachments, model.addBytes("Table", "text/html", ".html", []byte(html)))
	}
	return out, true
}

func convertConcept(concept *gauge_messages.ProtoConcept, start int64, model *Model, opts Options) (Step, bool) {
	if concept == nil {
		return Step{}, false
	}
	name := "concept"
	if concept.GetConceptStep() != nil {
		name = firstNonEmpty(concept.GetConceptStep().GetActualText(), concept.GetConceptStep().GetParsedText(), name)
	}
	duration := int64(0)
	if concept.GetConceptExecutionResult() != nil && concept.GetConceptExecutionResult().GetExecutionResult() != nil {
		duration = concept.GetConceptExecutionResult().GetExecutionResult().GetExecutionTime()
	}
	out := Step{
		Name:  name,
		Start: start,
		Stop:  start + duration,
		Stage: stageFinished,
		Steps: convertItems(concept.GetSteps(), start, model, opts),
	}
	out.Status, out.StatusDetails = executionStatus(concept.GetConceptExecutionResult())
	if concept.GetConceptStep() != nil {
		out.Parameters = stepParameters(concept.GetConceptStep())
		out.Attachments = append(out.Attachments, stepAttachments(concept.GetConceptStep(), model, opts)...)
	}
	if out.Status == "" {
		status := statusPassed
		for _, child := range out.Steps {
			status = worstStatus(status, child.Status)
		}
		out.Status = status
	}
	return out, true
}

func executionStatus(result *gauge_messages.ProtoStepExecutionResult) (string, *StatusDetails) {
	if result == nil {
		return statusSkipped, nil
	}
	if result.GetSkipped() {
		return statusSkipped, &StatusDetails{Message: result.GetSkippedReason()}
	}
	if result.GetPreHookFailure() != nil {
		f := result.GetPreHookFailure()
		return statusBroken, &StatusDetails{Message: f.GetErrorMessage(), Trace: f.GetStackTrace()}
	}
	if exec := result.GetExecutionResult(); exec != nil {
		if exec.GetFailed() {
			status := statusFailed
			if exec.GetErrorType() != gauge_messages.ProtoExecutionResult_ASSERTION {
				status = statusBroken
			}
			return status, &StatusDetails{Message: exec.GetErrorMessage(), Trace: exec.GetStackTrace()}
		}
		if exec.GetSkipScenario() {
			return statusSkipped, &StatusDetails{Message: exec.GetErrorMessage()}
		}
		return statusPassed, nil
	}
	if result.GetPostHookFailure() != nil {
		f := result.GetPostHookFailure()
		return statusBroken, &StatusDetails{Message: f.GetErrorMessage(), Trace: f.GetStackTrace()}
	}
	return statusPassed, nil
}

func stepParameters(step *gauge_messages.ProtoStep) []Parameter {
	var params []Parameter
	for _, fragment := range step.GetFragments() {
		if fragment.GetFragmentType() != gauge_messages.Fragment_Parameter || fragment.GetParameter() == nil {
			continue
		}
		p := fragment.GetParameter()
		name := firstNonEmpty(p.GetName(), parameterTypeName(p.GetParameterType()))
		value := p.GetValue()
		if p.GetTable() != nil {
			value = tableInline(p.GetTable())
		}
		params = append(params, Parameter{Name: name, Value: value})
	}
	return params
}

func stepAttachments(step *gauge_messages.ProtoStep, model *Model, opts Options) []Attachment {
	var atts []Attachment
	if result := step.GetStepExecutionResult().GetExecutionResult(); result != nil {
		if result.GetFailureScreenshotFile() != "" {
			atts = append(atts, model.addScreenshot("Failure screenshot", result.GetFailureScreenshotFile(), opts))
		}
		for i, shot := range result.GetScreenshotFiles() {
			atts = append(atts, model.addScreenshot(fmt.Sprintf("Screenshot %d", i+1), shot, opts))
		}
	}
	for i, shot := range step.GetPreHookScreenshotFiles() {
		atts = append(atts, model.addScreenshot(fmt.Sprintf("Before screenshot %d", i+1), shot, opts))
	}
	for i, shot := range step.GetPostHookScreenshotFiles() {
		atts = append(atts, model.addScreenshot(fmt.Sprintf("After screenshot %d", i+1), shot, opts))
	}
	return filterEmptyAttachments(atts)
}

func messagesOf(step *gauge_messages.ProtoStep) []string {
	var msgs []string
	msgs = append(msgs, step.GetPreHookMessages()...)
	if result := step.GetStepExecutionResult().GetExecutionResult(); result != nil {
		msgs = append(msgs, result.GetMessage()...)
	}
	msgs = append(msgs, step.GetPostHookMessages()...)
	return msgs
}

func fragmentsTableHTML(step *gauge_messages.ProtoStep) string {
	for _, fragment := range step.GetFragments() {
		if fragment.GetParameter() != nil && fragment.GetParameter().GetTable() != nil {
			return tableHTML(fragment.GetParameter().GetTable())
		}
	}
	return ""
}

func tableInline(table *gauge_messages.ProtoTable) string {
	if table == nil {
		return ""
	}
	var rows []string
	if headers := table.GetHeaders(); headers != nil {
		rows = append(rows, strings.Join(headers.GetCells(), " | "))
	}
	for _, row := range table.GetRows() {
		rows = append(rows, strings.Join(row.GetCells(), " | "))
	}
	return strings.Join(rows, "\n")
}

func parameterTypeName(t gauge_messages.Parameter_ParameterType) string {
	switch t {
	case gauge_messages.Parameter_Dynamic:
		return "dynamic"
	case gauge_messages.Parameter_Special_String:
		return "file"
	case gauge_messages.Parameter_Special_Table, gauge_messages.Parameter_Table:
		return "table"
	case gauge_messages.Parameter_Multiline_String:
		return "multiline"
	default:
		return "arg"
	}
}

func filterEmptyAttachments(atts []Attachment) []Attachment {
	var out []Attachment
	for _, att := range atts {
		if att.Source != "" {
			out = append(out, att)
		}
	}
	return out
}

func worstStatus(current, next string) string {
	rank := map[string]int{
		statusPassed:  1,
		statusSkipped: 2,
		statusFailed:  3,
		statusBroken:  4,
	}
	if rank[next] > rank[current] {
		return next
	}
	return current
}

func md5Hex(value string) string {
	sum := md5.Sum([]byte(value))
	return hex.EncodeToString(sum[:])
}

func historyID(testCaseID string, params []Parameter) string {
	cloned := append([]Parameter(nil), params...)
	sort.Slice(cloned, func(i, j int) bool {
		if cloned[i].Name == cloned[j].Name {
			return cloned[i].Value < cloned[j].Value
		}
		return cloned[i].Name < cloned[j].Name
	})
	var b strings.Builder
	for _, p := range cloned {
		b.WriteString(p.Name)
		b.WriteByte(':')
		b.WriteString(p.Value)
		b.WriteByte(',')
	}
	return testCaseID + ":" + md5Hex(b.String())
}
