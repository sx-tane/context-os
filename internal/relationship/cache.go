package relationship

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"context-os/domain/entities"
	"context-os/domain/types"
)

const assistCacheVersion = "relationship-assist/v1"

type cacheEnvelope struct {
	Relationships []types.Relationship `json:"relationships"`
}

// CachedAssistant stores accepted assistant relationships on disk by document content and entity IDs.
type CachedAssistant struct {
	dir       string
	assistant Assistant
}

// NewCachedAssistant wraps assistant with a disk cache rooted at dir.
func NewCachedAssistant(dir string, assistant Assistant) *CachedAssistant {
	return &CachedAssistant{dir: strings.TrimSpace(dir), assistant: assistant}
}

// Provider returns the wrapped assistant provider for accepted edge metadata.
func (c *CachedAssistant) Provider() string {
	if c == nil || c.assistant == nil {
		return ""
	}
	return assistProvider(c.assistant)
}

// ProposeRelationships returns cached proposals when present, otherwise it delegates and writes a cache entry.
func (c *CachedAssistant) ProposeRelationships(ctx context.Context, doc types.NormalizedDocument, canonical []entities.CanonicalEntity) ([]types.Relationship, error) {
	if c == nil || c.assistant == nil {
		return nil, nil
	}
	if c.dir == "" {
		return c.assistant.ProposeRelationships(ctx, doc, canonical)
	}

	path := filepath.Join(c.dir, assistCacheKey(c.Provider(), doc, canonical)+".json")
	if rels, ok := readAssistCache(path); ok {
		return rels, nil
	}

	rels, err := c.assistant.ProposeRelationships(ctx, doc, canonical)
	if err != nil {
		return nil, err
	}
	writeAssistCache(path, rels)
	return rels, nil
}

func readAssistCache(path string) ([]types.Relationship, bool) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, false
	}
	var envelope cacheEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, false
	}
	return envelope.Relationships, true
}

func writeAssistCache(path string, rels []types.Relationship) {
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return
	}
	data, err := json.MarshalIndent(cacheEnvelope{Relationships: rels}, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(filepath.Clean(path), append(data, '\n'), 0600)
}

func assistCacheKey(provider string, doc types.NormalizedDocument, canonical []entities.CanonicalEntity) string {
	entityIDs := make([]string, 0, len(canonical))
	for _, canonicalEntity := range canonical {
		entityIDs = append(entityIDs, canonicalEntity.Entity.ID)
	}
	sort.Strings(entityIDs)

	contentHash := strings.TrimSpace(doc.ContentHash)
	if contentHash == "" {
		sum := sha256.Sum256([]byte(doc.Title + "\n" + doc.Body))
		contentHash = hex.EncodeToString(sum[:])
	}

	keyMaterial := strings.Join([]string{
		assistCacheVersion,
		strings.TrimSpace(provider),
		contentHash,
		strings.Join(entityIDs, "\n"),
	}, "\n")
	sum := sha256.Sum256([]byte(keyMaterial))
	return hex.EncodeToString(sum[:])
}
