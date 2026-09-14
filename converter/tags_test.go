package converter

import "testing"

func TestParseTags(t *testing.T) {
	got := parseTags([]string{
		"@severity:blocker",
		"owner=alice",
		"issue:ABC-1",
		"tms:TC-9",
		"link:https://example.com|文档",
		"critical",
		"smoke",
		"",
	}, "https://issues/%s", "https://tms/%s")

	wantLabels := map[string]string{
		"severity": "blocker",
		"owner":    "alice",
		"tag":      "smoke",
	}
	for name, value := range wantLabels {
		found := false
		for _, l := range got.labels {
			if l.Name == name && l.Value == value {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing label %s=%s in %+v", name, value, got.labels)
		}
	}
	if len(got.links) != 3 {
		t.Fatalf("links: %+v", got.links)
	}
}

func TestBareSeverityTag(t *testing.T) {
	got := parseTags([]string{"minor"}, "", "")
	assertLabel(t, got.labels, "severity", "minor")
}
