// benchmark-runner.go
// Identity resolution benchmark harness for ContextOS.
// Run with: go test ./internal/identity/... -run BenchmarkIdentityResolution -v
//
// Usage: populate BenchmarkCases from the CSV template, call RunBenchmark, inspect BenchmarkResult.

package identity

import (
	"encoding/csv"
	"os"
	"strings"
)

// BenchmarkCase represents a single alias-to-canonical evaluation pair.
type BenchmarkCase struct {
	Alias         string
	Canonical     string
	Language      string
	Source        string
	ExpectedMatch bool
	Notes         string
}

// BenchmarkResult holds per-layer and overall evaluation metrics.
type BenchmarkResult struct {
	TotalCases        int
	LayerPrecision    map[string]float64
	OverallPrecision  float64
	OverallRecall     float64
	ConflictRate      float64
	FalseMergeRate    float64
	UnresolvedRate    float64
	HumanQueueCount   int
	ConflictDetails   []ConflictDetail
}

// ConflictDetail records one conflicting pair for post-analysis.
type ConflictDetail struct {
	Alias       string
	Canonical   string
	LayerA      string
	LayerB      string
	LayerAMerge bool
	LayerBMerge bool
	Notes       string
}

// LoadBenchmarkCasesFromCSV reads the benchmark-dataset-template.csv format.
// Expected columns: alias,canonical,language,source,expected_match,notes
func LoadBenchmarkCasesFromCSV(path string) ([]BenchmarkCase, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	rows, err := r.ReadAll()
	if err != nil {
		return nil, err
	}

	cases := make([]BenchmarkCase, 0, len(rows)-1)
	for _, row := range rows[1:] { // skip header
		if len(row) < 6 {
			continue
		}
		cases = append(cases, BenchmarkCase{
			Alias:         strings.TrimSpace(row[0]),
			Canonical:     strings.TrimSpace(row[1]),
			Language:      strings.TrimSpace(row[2]),
			Source:        strings.TrimSpace(row[3]),
			ExpectedMatch: strings.EqualFold(strings.TrimSpace(row[4]), "true"),
			Notes:         strings.TrimSpace(row[5]),
		})
	}
	return cases, nil
}

// RunBenchmark evaluates cases against the provided Resolver and returns metrics.
// Resolver is your internal/identity resolution function:
//
//	type Resolver func(alias string) (canonical string, confidence float64, layer string, err error)
func RunBenchmark(cases []BenchmarkCase, resolve func(alias string) (canonical string, confidence float64, layer string, err error)) BenchmarkResult {
	result := BenchmarkResult{
		TotalCases:     len(cases),
		LayerPrecision: make(map[string]float64),
	}

	layerCorrect := make(map[string]int)
	layerTotal := make(map[string]int)
	correct := 0
	resolved := 0
	conflicts := 0

	for _, c := range cases {
		got, _, layer, err := resolve(c.Alias)
		if err != nil || got == "" {
			continue
		}
		resolved++
		layerTotal[layer]++
		merged := strings.EqualFold(got, c.Canonical)
		if merged == c.ExpectedMatch {
			correct++
			layerCorrect[layer]++
		} else if c.ExpectedMatch && !merged {
			conflicts++ // missed merge = conflict
		} else if !c.ExpectedMatch && merged {
			conflicts++ // false merge = conflict
		}
	}

	if resolved > 0 {
		result.OverallPrecision = float64(correct) / float64(resolved)
		result.ConflictRate = float64(conflicts) / float64(resolved)
		result.UnresolvedRate = float64(len(cases)-resolved) / float64(len(cases))
	}

	for layer, total := range layerTotal {
		if total > 0 {
			result.LayerPrecision[layer] = float64(layerCorrect[layer]) / float64(total)
		}
	}

	return result
}
