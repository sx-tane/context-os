package aiworker_test

import (
	"os"
	"path/filepath"
	"testing"

	"context-os/internal/aiworker"
)

// TestEmbeddingCacheGetMissOnEmptyDir verifies that Get returns a miss when the cache dir is empty string.
func TestEmbeddingCacheGetMissOnEmptyDir(t *testing.T) {
	c := aiworker.NewEmbeddingCache("")
	if vec, ok := c.Get("hello"); ok || vec != nil {
		t.Errorf("EmbeddingCache.Get() with empty dir = %v, %v; want nil, false", vec, ok)
	}
}

// TestEmbeddingCacheSetNoopOnEmptyDir verifies that Set is a no-op and returns nil when dir is empty.
func TestEmbeddingCacheSetNoopOnEmptyDir(t *testing.T) {
	c := aiworker.NewEmbeddingCache("")
	if err := c.Set("hello", []float64{0.1, 0.2}); err != nil {
		t.Errorf("EmbeddingCache.Set() with empty dir error = %v; want nil", err)
	}
}

// TestEmbeddingCacheRoundTrip verifies that a vector written with Set is returned by Get.
func TestEmbeddingCacheRoundTrip(t *testing.T) {
	dir := t.TempDir()
	c := aiworker.NewEmbeddingCache(dir)
	want := []float64{0.1, 0.5, 0.9}

	if err := c.Set("hello world", want); err != nil {
		t.Fatalf("EmbeddingCache.Set() error = %v", err)
	}

	got, ok := c.Get("hello world")
	if !ok {
		t.Fatalf("EmbeddingCache.Get() hit = false; want true")
	}
	if len(got) != len(want) {
		t.Fatalf("EmbeddingCache.Get() len = %d; want %d", len(got), len(want))
	}
	for i, v := range want {
		if got[i] != v {
			t.Errorf("EmbeddingCache.Get()[%d] = %v; want %v", i, got[i], v)
		}
	}
}

// TestEmbeddingCacheMissOnUnknownText verifies that Get returns a miss for a text that was never written.
func TestEmbeddingCacheMissOnUnknownText(t *testing.T) {
	dir := t.TempDir()
	c := aiworker.NewEmbeddingCache(dir)

	if vec, ok := c.Get("never written"); ok || vec != nil {
		t.Errorf("EmbeddingCache.Get() unknown text = %v, %v; want nil, false", vec, ok)
	}
}

// TestCacheKeyIsStable verifies that CacheKey returns the same value for the same input.
func TestCacheKeyIsStable(t *testing.T) {
	a := aiworker.CacheKey("test text")
	b := aiworker.CacheKey("test text")
	if a != b {
		t.Errorf("CacheKey() = %q, %q; want identical", a, b)
	}
}

// TestCacheKeyFileExists verifies that the cache file path uses the CacheKey as its name.
func TestCacheKeyFileExists(t *testing.T) {
	dir := t.TempDir()
	c := aiworker.NewEmbeddingCache(dir)
	text := "check file name"

	if err := c.Set(text, []float64{1.0}); err != nil {
		t.Fatalf("EmbeddingCache.Set() error = %v", err)
	}

	expected := filepath.Join(dir, aiworker.CacheKey(text)+".json")
	if _, err := os.Stat(expected); err != nil {
		t.Errorf("expected cache file %s does not exist: %v", expected, err)
	}
}
