package tests

import (
	"os"            // used to stat paths and check they exist on disk
	"path/filepath" // used to build OS-safe paths relative to the repo root
	"runtime"       // used to find the source file's absolute path at test time
	"testing"       // provides the testing.T type and test helpers
)

// TestRepositoryScaffoldPathsExist confirms that all required directories and files
// exist at the expected locations in the repository layout.
func TestRepositoryScaffoldPathsExist(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0) // get the absolute path of this test file at runtime
	if !ok {
		t.Fatal("failed to determine test file path") // this should never fail in a normal Go toolchain
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..")) // navigate one level up from tests/ to the repo root

	requiredDirectories := []string{ // all directories that must exist for the scaffold to be valid
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

	for _, dir := range requiredDirectories { // verify every directory in the list actually exists
		path := filepath.Join(root, dir)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("expected directory %q to exist: %v", dir, err) // stat failed — directory is missing
		}
		if !info.IsDir() {
			t.Fatalf("expected %q to be a directory", dir) // path exists but is a file, not a directory
		}
	}

	requiredFiles := []string{ // key files that must exist for each app to be runnable
		"apps/frontend/package.json",
		"apps/frontend/bun.lockb",
		"apps/api/main.go",
		"apps/ai-worker/pyproject.toml",
	}

	for _, file := range requiredFiles { // verify every required file actually exists
		path := filepath.Join(root, file)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("expected file %q to exist: %v", file, err) // stat failed — file is missing
		}
		if info.IsDir() {
			t.Fatalf("expected %q to be a file", file) // path exists but is a directory, not a file
		}
	}
}
