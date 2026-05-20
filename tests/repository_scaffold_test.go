package tests

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestRepositoryScaffoldPathsExist(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to determine test file path")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), ".."))

	requiredDirectories := []string{
		"apps",
		"internal",
		"domain",
		"storage",
		"docs",
		"scripts",
		"migrations",
		"prompts",
		"docker",
		"tests",
		"internal/source",
		"internal/ingestion",
		"internal/normalization",
		"internal/classification",
		"internal/extraction",
		"internal/identity",
		"internal/relationship",
		"internal/graph",
		"internal/reasoning",
		"internal/execution",
		"internal/presentation",
		"domain/contracts",
		"domain/events",
		"domain/pipelines",
		"domain/entities",
		"domain/types",
	}

	for _, dir := range requiredDirectories {
		path := filepath.Join(root, dir)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("expected directory %q to exist: %v", dir, err)
		}
		if !info.IsDir() {
			t.Fatalf("expected %q to be a directory", dir)
		}
	}

	requiredFiles := []string{
		"apps/frontend/package.json",
		"apps/frontend/bun.lockb",
		"apps/api/main.go",
		"apps/ai-worker/pyproject.toml",
	}

	for _, file := range requiredFiles {
		path := filepath.Join(root, file)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("expected file %q to exist: %v", file, err)
		}
		if info.IsDir() {
			t.Fatalf("expected %q to be a file", file)
		}
	}
}
