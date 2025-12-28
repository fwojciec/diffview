# Classification Cache Design

Cache diffstory classification results to avoid redundant LLM calls.

## Decisions

| Decision | Choice |
|----------|--------|
| Cache key | SHA256 hash of JSON-serialized `ClassificationInput` |
| Storage format | Individual JSON files per entry |
| Cache location | `$XDG_CACHE_HOME/diffstory/` or `~/.cache/diffstory/` |
| Entry content | `StoryClassification` directly (no metadata wrapper) |
| Architecture | `fs.Classifier` wrapper implementing `StoryClassifier` |

## Package Structure

```
diffview/
├── fs/
│   ├── fs.go              # Cache directory resolution
│   ├── classifier.go      # Classifier implementing StoryClassifier
│   └── classifier_test.go
└── cmd/diffstory/
    └── main.go            # Wires fs.NewClassifier(geminiClassifier, cacheDir)
```

## Core Flow

```go
func (c *Classifier) Classify(ctx context.Context, input ClassificationInput) (*StoryClassification, error) {
    hash := hashInput(input)  // SHA256 of JSON-serialized input

    if cached, err := c.loadFromCache(hash); err == nil {
        return cached, nil
    }

    result, err := c.inner.Classify(ctx, input)
    if err != nil {
        return nil, err
    }

    _ = c.saveToCache(hash, result)  // Best-effort, ignore errors
    return result, nil
}
```

## File Layout

- Directory: `~/.cache/diffstory/` (or XDG override)
- Files: `<hash>.json` where hash is 64-char hex SHA256
- Permissions: `0755` directory, `0644` files
- Directory created on first write via `os.MkdirAll`

## Wiring

```go
geminiClassifier := gemini.NewClassifier(client, gemini.DefaultModel)
classifier := fs.NewClassifier(geminiClassifier, fs.DefaultCacheDir())
```

## Testing Strategy

- Use `t.TempDir()` for isolated cache per test
- Mock inner classifier to verify:
  - Cache miss → delegates to inner, stores result
  - Cache hit → returns cached, inner not called
  - Different input → new inner call
  - Corrupted cache file → treated as miss
