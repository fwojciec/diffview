# Diff Story Generator: Design Document

An offline analysis tool that takes a git diff and produces a structured "story" - classifying the change type and segmenting hunks into narrative roles with explanations.

## Goals

1. Validate that LLM-generated stories actually help code review
2. Build an eval suite to systematically improve story quality
3. Create the foundation for future viewer integration

## Non-Goals (for now)

- Viewer integration (comes after we validate stories are useful)
- Real-time analysis (batch/offline first)
- Beads/hooks integration (future layers)

## Core Insight

The LLM's job is two tasks:

1. **Classification:** What type of change is this? (bugfix, feature, refactor, etc.)
2. **Segmentation:** Which hunks belong to which narrative role?

Different change types map to different narrative structures:

| Pattern | Structure | Best for |
|---------|-----------|----------|
| Cause → Effect | Bug report → Fix → Test | Bugfixes |
| Core → Periphery | Central logic → Supporting changes | Features |
| Entry → Implementation | API/interface → Internal logic → Helpers | Understanding flow |
| Before → After | Old approach → Transformation → New approach | Refactoring |

## Data Model

### Input (deterministic Go parsing)

```go
type AnnotatedDiff struct {
    Hunks []AnnotatedHunk
}

type AnnotatedHunk struct {
    ID       string   // "h1", "h2", etc.
    File     string   // "src/auth.go"
    OldStart int
    NewStart int
    Lines    []string // The actual diff lines
}
```

### Output (JSON from LLM)

```go
// Extensible container for future analysis types
type DiffAnalysis struct {
    Version   string           // Schema version for evolution
    DiffMeta  DiffMetadata     // Source, commit, etc.
    Hunks     []AnnotatedHunk  // The parsed input (stable)
    Analyses  []Analysis       // Extensible list of analysis types
}

type Analysis struct {
    Type    string          // "story", "architecture", "risks", etc.
    Payload json.RawMessage // Type-specific structure
}

// Current focus - Story analysis
type StoryAnalysis struct {
    ChangeType string      // "bugfix", "feature", "refactor", "chore"
    Summary    string      // One sentence
    Parts      []StoryPart
}

type StoryPart struct {
    Role        string   // "core", "supporting", "test", "cleanup"
    HunkIDs     []string // ["h1", "h3"]
    Explanation string   // Markdown, 1-3 sentences
}
```

The extensible `Analyses` array allows future additions (architectural analysis, risk assessment, etc.) without breaking changes.

## Eval Infrastructure

Following Hamel Husain's methodology, adapted for Go.

### Directory Structure

```
eval/
├── cases/         # Test cases (JSONL with diff paths + rubrics)
├── runs/          # Results per run (JSONL, append-only)
└── evals.db       # SQLite for queryable state (what needs review)
```

### Test Integration (Mattermost pattern)

```go
func TestStoryGeneration(t *testing.T) {
    if os.Getenv("GOEVALS") == "" {
        t.Skip("GOEVALS not set")
    }

    result := generateStory(loadDiff("testdata/feature-add.diff"))

    // LLM-as-judge rubrics
    assertRubric(t, "correctly classifies change type", result)
    assertRubric(t, "identifies core hunks vs supporting", result)
    assertRubric(t, "explanation is technically accurate", result)
}
```

### TUI Reviewer

Reuses existing diff viewer components:

```
┌─────────────────────────────────────────────┐
│ Diff (existing viewer with Chroma)          │
├─────────────────────────────────────────────┤
│ Generated Story (viewport)                  │
├─────────────────────────────────────────────┤
│ Judgment: [Pass] [Fail]                     │
│ Critique: [text input]                      │
├─────────────────────────────────────────────┤
│ [←Prev] [→Next] [Save] [q]uit              │
└─────────────────────────────────────────────┘
```

## CLI Design

```bash
# Generate story for a diff
diffstory analyze <diff-file-or-stdin>
# Output: JSON story to stdout

# Collect diffs from git history
diffstory collect <repo-path> --limit=50
# Output: Creates eval cases in eval/cases/

# Batch analysis
diffstory batch eval/cases/*.jsonl > eval/runs/2025-01-15.jsonl
```

## Prompt Design

### Input Format

```
You are analyzing a git diff to help a human reviewer understand the change.

## Hunks

[h1] src/auth/token.go:15-28 (added)
```go
+func ValidateToken(token string) error {
+    ...
+}
```

[h2] src/auth/token.go:45-52 (modified)
...

## Task

Classify this change and segment the hunks into a narrative structure.

Respond with JSON matching this schema:
{schema}
```

### Prompt Iteration Strategy

1. Start simple, observe failures
2. Add few-shot examples from domain expert judgments
3. Rubrics emerge from error analysis

## LLM Choice

- **Gemini 2.0 Flash** (GA) for iteration speed
- **Gemini 2.0 Pro** (preview) for quality comparison
- Reference `../locdoc` for Gemini client implementation

Using Gemini while Claude writes code provides different perspective in the loop.

## Implementation Phases

### Phase 1: Minimal viable story generator
- Parse diff → assign hunk IDs (reuse existing `gitdiff` package)
- Gemini client (adapt from locdoc)
- `diffstory analyze` command
- JSON output to stdout

### Phase 2: Collection tooling
- `diffstory collect` command
- Extract diffs from git history (diffview, locdoc repos)
- Generate eval cases (JSONL)

### Phase 3: TUI reviewer
- Bubble Tea app reusing existing diff viewer
- Pass/fail judgment + critique input
- Saves to SQLite/JSONL

### Phase 4: Eval integration
- LLM-as-judge assertions in `go test`
- Rubrics derived from critiques
- `GOEVALS=1 go test` for CI

### Phase 5: Iterate
- Error analysis → failure taxonomy
- Prompt refinement
- Few-shot examples from good outputs

## Package Structure

Following Ben Johnson Standard Package Layout:

```
diffview/
├── diffview.go          # Existing domain types (Diff, Hunk, etc.)
├── story.go             # NEW: Story domain types (in root, not subpackage)
├── generator.go         # NEW: StoryGenerator interface
│
├── gemini/              # Named after dependency (Gemini API)
│   └── generator.go     # Implements StoryGenerator
│
├── gitdiff/             # Existing
├── chroma/              # Existing
├── bubbletea/           # Existing
├── worddiff/            # Existing
├── lipgloss/            # Existing
├── mock/                # Existing (add StoryGenerator mock)
│
├── cmd/diffview/        # Existing CLI
├── cmd/diffstory/       # NEW: CLI for story generation
├── cmd/evalreview/      # NEW: TUI for eval review
│
└── eval/                # Data directory (not a Go package)
    ├── cases/           # JSONL test cases
    └── runs/            # Result files
```

**Key decisions:**
- Story domain types go in **root package** (not `story/` - that would be concept-named)
- `gemini/` named after the external dependency
- `eval/` is a data directory, orthogonal to the deterministic codebase

## References

- [Intelligent Diff Presentation](../intellingent-diff-presentation.md) - Cognitive science foundations
- [From Diff Viewer to Feedback Interface](../from-diff-viewer-to-feedback-interface.md) - Vision document
- [Go-centric LLM Evals](../llm-evals-go.md) - Eval infrastructure research
- `../locdoc` - Gemini client implementation reference
