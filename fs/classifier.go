package fs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/fwojciec/diffview"
)

// Compile-time interface verification.
var _ diffview.StoryClassifier = (*Classifier)(nil)

// Classifier wraps a StoryClassifier with file-based caching.
type Classifier struct {
	inner    diffview.StoryClassifier
	cacheDir string
}

// NewClassifier creates a new caching classifier.
func NewClassifier(inner diffview.StoryClassifier, cacheDir string) *Classifier {
	return &Classifier{
		inner:    inner,
		cacheDir: cacheDir,
	}
}

// Classify returns a cached classification or delegates to inner classifier.
func (c *Classifier) Classify(ctx context.Context, input diffview.ClassificationInput) (*diffview.StoryClassification, error) {
	hash := c.hashInput(input)

	// Check cache
	if cached, err := c.loadFromCache(hash); err == nil {
		return cached, nil
	}

	// Cache miss - delegate to inner
	result, err := c.inner.Classify(ctx, input)
	if err != nil {
		return nil, err
	}

	// Store in cache (best-effort)
	_ = c.saveToCache(hash, result)

	return result, nil
}

func (c *Classifier) hashInput(input diffview.ClassificationInput) string {
	data, _ := json.Marshal(input)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func (c *Classifier) cachePath(hash string) string {
	return filepath.Join(c.cacheDir, hash+".json")
}

func (c *Classifier) loadFromCache(hash string) (*diffview.StoryClassification, error) {
	data, err := os.ReadFile(c.cachePath(hash))
	if err != nil {
		return nil, err
	}

	var result diffview.StoryClassification
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Classifier) saveToCache(hash string, result *diffview.StoryClassification) error {
	if err := os.MkdirAll(c.cacheDir, 0755); err != nil {
		return err
	}

	data, err := json.Marshal(result)
	if err != nil {
		return err
	}

	return os.WriteFile(c.cachePath(hash), data, 0644)
}
