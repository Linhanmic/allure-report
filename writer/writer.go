package writer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Linhanmic/allure-report/converter"
)

const (
	filePermissions = 0o644
	dirPermissions  = 0o755
)

// Write persists Allure result files into resultsDir.
func Write(model *converter.Model, resultsDir string) error {
	if model == nil {
		return fmt.Errorf("model is nil")
	}
	if err := os.MkdirAll(resultsDir, dirPermissions); err != nil {
		return fmt.Errorf("create results dir: %w", err)
	}
	for _, result := range model.Results {
		if err := writeJSON(filepath.Join(resultsDir, result.UUID+"-result.json"), result); err != nil {
			return err
		}
	}
	for _, container := range model.Containers {
		if err := writeJSON(filepath.Join(resultsDir, container.UUID+"-container.json"), container); err != nil {
			return err
		}
	}
	for _, file := range model.Files {
		if err := writeFile(resultsDir, file); err != nil {
			return err
		}
	}
	if len(model.Environment) > 0 {
		if err := os.WriteFile(filepath.Join(resultsDir, "environment.properties"), environmentBytes(model.Environment), filePermissions); err != nil {
			return fmt.Errorf("write environment.properties: %w", err)
		}
	}
	if len(model.Categories) > 0 {
		if err := writeJSON(filepath.Join(resultsDir, "categories.json"), model.Categories); err != nil {
			return err
		}
	}
	if model.Executor != nil {
		if err := writeJSON(filepath.Join(resultsDir, "executor.json"), model.Executor); err != nil {
			return err
		}
	}
	return nil
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", filepath.Base(path), err)
	}
	if err := os.WriteFile(path, data, filePermissions); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func writeFile(resultsDir string, file converter.File) error {
	if file.Name == "" {
		return nil
	}
	dest := filepath.Join(resultsDir, file.Name)
	if len(file.Bytes) > 0 {
		return os.WriteFile(dest, file.Bytes, filePermissions)
	}
	if file.SourcePath == "" {
		return nil
	}
	data, err := os.ReadFile(file.SourcePath)
	if err != nil {
		return fmt.Errorf("read attachment %s: %w", file.SourcePath, err)
	}
	return os.WriteFile(dest, data, filePermissions)
}

func environmentBytes(env map[string]string) []byte {
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(env[k])
		b.WriteByte('\n')
	}
	return []byte(b.String())
}
