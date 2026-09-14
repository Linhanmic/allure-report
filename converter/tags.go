package converter

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var (
	severityValues = map[string]string{
		"blocker":  "blocker",
		"critical": "critical",
		"normal":   "normal",
		"minor":    "minor",
		"trivial":  "trivial",
	}
	keyValueTag = regexp.MustCompile(`^(?i)(severity|owner|lead|epic|feature|story|issue|tms|layer|package|link|allure_id|as_id)[:：=](.+)$`)
)

type parsedTags struct {
	labels []Label
	links  []Link
}

func parseTags(tags []string, issuePattern, tmsPattern string) parsedTags {
	out := parsedTags{}
	seen := map[string]struct{}{}
	for _, raw := range tags {
		tag := strings.TrimSpace(strings.TrimPrefix(raw, "@"))
		if tag == "" {
			continue
		}
		key := strings.ToLower(tag)
		if severity, ok := severityValues[key]; ok {
			out.labels = appendLabel(out.labels, "severity", severity, seen)
			continue
		}
		if m := keyValueTag.FindStringSubmatch(tag); m != nil {
			name := strings.ToLower(m[1])
			value := strings.TrimSpace(m[2])
			switch name {
			case "issue":
				out.links = append(out.links, makeIssueLink(value, issuePattern))
				out.labels = appendLabel(out.labels, "tag", value, seen)
			case "tms":
				out.links = append(out.links, makeTMSLink(value, tmsPattern))
				out.labels = appendLabel(out.labels, "tag", value, seen)
			case "link":
				out.links = append(out.links, parseCustomLink(value))
			case "allure_id", "as_id":
				out.labels = appendLabel(out.labels, "ALLURE_ID", value, seen)
			case "severity":
				if mapped, ok := severityValues[strings.ToLower(value)]; ok {
					value = mapped
				}
				out.labels = appendLabel(out.labels, "severity", value, seen)
			default:
				out.labels = appendLabel(out.labels, name, value, seen)
			}
			continue
		}
		out.labels = appendLabel(out.labels, "tag", tag, seen)
	}
	return out
}

func appendLabel(labels []Label, name, value string, seen map[string]struct{}) []Label {
	if value == "" {
		return labels
	}
	key := name + "\x00" + value
	if _, ok := seen[key]; ok {
		return labels
	}
	seen[key] = struct{}{}
	return append(labels, Label{Name: name, Value: value})
}

func makeIssueLink(value, pattern string) Link {
	return Link{Type: "issue", Name: value, URL: applyPattern(pattern, value)}
}

func makeTMSLink(value, pattern string) Link {
	return Link{Type: "tms", Name: value, URL: applyPattern(pattern, value)}
}

func parseCustomLink(value string) Link {
	name := value
	linkURL := value
	if parts := strings.SplitN(value, "|", 2); len(parts) == 2 {
		linkURL = strings.TrimSpace(parts[0])
		name = strings.TrimSpace(parts[1])
	}
	return Link{Type: "link", Name: name, URL: linkURL}
}

func applyPattern(pattern, value string) string {
	if strings.TrimSpace(pattern) == "" {
		if isURL(value) {
			return value
		}
		return value
	}
	if strings.Contains(pattern, "%s") {
		return fmt.Sprintf(pattern, value)
	}
	if strings.Contains(pattern, "{}") {
		return strings.Replace(pattern, "{}", value, 1)
	}
	if strings.HasSuffix(pattern, "/") {
		return pattern + url.PathEscape(value)
	}
	return pattern + value
}

func isURL(value string) bool {
	u, err := url.Parse(value)
	return err == nil && u.Scheme != "" && u.Host != ""
}
