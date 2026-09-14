package writer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Linhanmic/allure-report/converter"
)

func TestWriteResults(t *testing.T) {
	dir := t.TempDir()
	model := &converter.Model{
		Results: []converter.TestResult{{
			UUID:     "11111111-1111-1111-1111-111111111111",
			Name:     "登录",
			Status:   "passed",
			FullName: "specs/login.spec#登录",
			Labels:   []converter.Label{{Name: "framework", Value: "gauge"}},
		}},
		Containers: []converter.TestResultContainer{{
			UUID:     "22222222-2222-2222-2222-222222222222",
			Name:     "用户认证",
			Children: []string{"11111111-1111-1111-1111-111111111111"},
		}},
		Files: []converter.File{{
			Name:        "shot-attachment.txt",
			ContentType: "text/plain",
			Bytes:       []byte("hello"),
		}},
		Environment: map[string]string{"framework": "gauge", "project": "demo"},
		Categories:  []converter.Category{{Name: "Broken tests", MatchedStatuses: []string{"broken"}}},
		Executor:    &converter.Executor{Name: "GitHub Actions", Type: "github"},
	}
	if err := Write(model, dir); err != nil {
		t.Fatal(err)
	}

	mustExist(t, filepath.Join(dir, "11111111-1111-1111-1111-111111111111-result.json"))
	mustExist(t, filepath.Join(dir, "22222222-2222-2222-2222-222222222222-container.json"))
	mustExist(t, filepath.Join(dir, "shot-attachment.txt"))
	mustExist(t, filepath.Join(dir, "environment.properties"))
	mustExist(t, filepath.Join(dir, "categories.json"))
	mustExist(t, filepath.Join(dir, "executor.json"))

	raw, err := os.ReadFile(filepath.Join(dir, "11111111-1111-1111-1111-111111111111-result.json"))
	if err != nil {
		t.Fatal(err)
	}
	var tr converter.TestResult
	if err := json.Unmarshal(raw, &tr); err != nil {
		t.Fatal(err)
	}
	if tr.Name != "登录" || tr.Status != "passed" {
		t.Fatalf("result: %+v", tr)
	}
}

func mustExist(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("missing %s: %v", path, err)
	}
}
