# Evolution Mode Design

Design for enriching diff classification with commit history context.

## Problem

Current diff view shows squashed changes. The development journey - how the code evolved commit-by-commit - is lost. This matters because:

1. **LLM classification** - Commit sequence provides intent signals that help the LLM understand and classify changes better
2. **Reviewer insight** - Understanding "first tried X, then refactored to Y" helps validate the final implementation

## Research Basis

- Doppler: "treat git history as narrative, not just log"
- Stacked PRs: "reviewers understand code better in logical layers"
- Studies show 16% better recall with story vs list format

## Design

### Input Enrichment

Extend `CommitBrief` to include per-commit diffs:

```go
type CommitBrief struct {
    Hash    string `json:"hash"`
    Message string `json:"message"`
    Diff    *Diff  `json:"diff,omitempty"`  // Per-commit changes
}
```

The `Diff` field is optional for backward compatibility. When present, it contains the diff for just that commit (not the cumulative/squashed diff).

### Output Extension

Add optional evolutionary insight to classification:

```go
type StoryClassification struct {
    ChangeType string    `json:"change_type"`
    Narrative  string    `json:"narrative"`
    Summary    string    `json:"summary"`
    Sections   []Section `json:"sections"`
    Evolution  string    `json:"evolution,omitempty"` // NEW: development journey
}
```

The `Evolution` field captures high-level evolutionary insight when the commit history reveals something meaningful:
- "Implementation started with naive approach, then optimized after benchmarking"
- "Feature added in commit 1, edge cases handled in commits 2-3"
- Empty when commit history doesn't add insight (single commit, mechanical changes)

### Section-Level Integration

Section explanations can naturally reference evolution:
- "Refactored from map-based caching to sync.Pool (commits 2→4)"
- "Test coverage added after implementation was stable (commit 3)"

No new fields needed - the LLM weaves history into existing explanation text.

### Classifier Prompt Update

Update the classification prompt to:
1. Consider commit sequence when determining narrative structure
2. Reference specific commits in section explanations when relevant
3. Generate `Evolution` field when history provides meaningful insight

## Implementation Scope

### In Scope
- Schema changes to `CommitBrief` and `StoryClassification`
- Extraction pipeline changes to capture per-commit diffs
- Classifier prompt updates

### Out of Scope (Future Work)
- Separate "evolution mode" UI toggle
- Hunk-level time travel navigation
- Per-hunk commit history annotations

## Validation

- [ ] Schema changes compile and serialize correctly
- [ ] Extraction captures per-commit diffs
- [ ] Classifier produces evolutionary insights for multi-commit PRs
- [ ] Existing single-commit cases continue to work
- [ ] `make validate` passes
