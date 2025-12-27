# DiffStory Evaluation System Design

A system for evaluating LLM-based diff classification using Hamel Husain's eval methodology.

## Problem

We need to validate whether an LLM can correctly classify code diffs into:
- **Hunk categories**: refactoring, systematic, core logic, noise
- **Change types**: bugfix, feature, refactor, chore, docs
- **Narrative patterns**: cause-effect, core-periphery, before-after, rule-instances, entry-implementation

Before building the full narrative diff viewer, we need an eval system to iterate on the classification prompt.

## Methodology

Following Hamel's approach:

1. **Binary pass/fail scoring** - not Likert scales
2. **Detailed critiques** - "detailed enough for a new employee to understand"
3. **Open Coding first** - freeform critiques for 30-50 examples
4. **Axial Coding second** - LLM groups critiques into failure taxonomy
5. **Single domain expert** - benevolent dictator (Filip)

Sources:
- https://hamel.dev/blog/posts/llm-judge/index.html
- https://hamel.dev/blog/posts/evals-faq/why-is-error-analysis-so-important-in-llm-evals-and-how-is-it-performed.html

## Domain Types

New/extended types in root package (`diffview.go`):

```go
// CommitInfo captures metadata about a commit for classification.
type CommitInfo struct {
    Hash    string
    Repo    string
    Message string
}

// ClassificationInput is the complete input for story classification.
type ClassificationInput struct {
    Commit CommitInfo
    Diff   Diff
}

// StoryClassification is the LLM's structured output.
type StoryClassification struct {
    ChangeType string    `json:"change_type"`
    Narrative  string    `json:"narrative"`
    Summary    string    `json:"summary"`
    Sections   []Section `json:"sections"`
}

type Section struct {
    Role        string    `json:"role"`
    Title       string    `json:"title"`
    Hunks       []HunkRef `json:"hunks"`
    Explanation string    `json:"explanation"`
}

type HunkRef struct {
    File         string `json:"file"`
    HunkIndex    int    `json:"hunk_index"`
    Category     string `json:"category"`
    Collapsed    bool   `json:"collapsed"`
    CollapseText string `json:"collapse_text,omitempty"`
}

// EvalCase for human review.
type EvalCase struct {
    Input ClassificationInput  `json:"input"`
    Story *StoryClassification `json:"story"`
}

// StoryClassifier produces structured classification from diff + commit info.
type StoryClassifier interface {
    Classify(ctx context.Context, input ClassificationInput) (*StoryClassification, error)
}
```

## Classification Prompt

**Input format**:
```
<commit_message>
[Original commit message]
</commit_message>

<diff>
=== FILE: pkg/auth/login.go (modified) ===

--- HUNK H1 (@@ -45,6 +45,10 @@) ---
[hunk content with +/- lines]

--- HUNK H2 (@@ -82,3 +86,7 @@) ---
[hunk content]

=== FILE: pkg/auth/login_test.go (added) ===
...
</diff>
```

**Output format** (JSON):
```json
{
  "change_type": "bugfix|feature|refactor|chore|docs",
  "narrative": "cause-effect|core-periphery|before-after|rule-instances|entry-implementation",
  "summary": "One sentence describing the change",
  "sections": [
    {
      "role": "problem|fix|test|core|supporting|rule|exception|integration|cleanup",
      "title": "Human-readable section title",
      "hunks": [
        {
          "file": "path/to/file.go",
          "hunk_index": 0,
          "category": "refactoring|systematic|core|noise",
          "collapsed": false,
          "collapse_text": null
        }
      ],
      "explanation": "Why this section matters in the narrative"
    }
  ]
}
```

**Validation rules**:
- All referenced hunk IDs must exist in input
- Every input hunk must appear in exactly one section
- JSON schema validation

## Evalreview Enhancements

Current gaps:
- `ModeCritique` exists but has no text input
- Critiques display truncated to 30 chars

Required changes:

1. **Critique text input**
   - `[c]` enters critique mode
   - Full textarea for detailed critiques
   - `Esc` saves and exits

2. **Critique display**
   - Show full critique in story panel
   - Visual indicator: unjudged / pass / fail / has-critique

3. **Navigation helpers**
   - `[u]` jump to next unjudged case
   - Filter to show only failures (for axial coding)
   - Export critiques to markdown

**UI flow**:
```
Review Mode                    Critique Mode
┌─────────────────────┐       ┌─────────────────────┐
│ DIFF               │       │ CRITIQUE            │
│ [diff content]     │       │                     │
├─────────────────────┤  [c]  │ [text area with     │
│ STORY              │ ────► │  existing critique  │
│ [LLM output]       │       │  or empty]          │
├─────────────────────┤       │                     │
│ ○ Pass ● Fail      │ ◄──── │ [Esc] save & exit   │
│ Critique: "..."    │  Esc  │                     │
└─────────────────────┘       └─────────────────────┘
```

## Diff Collection

**Sources**: diffview and locdoc git histories

**Target composition** (~50 total):

| Type | Target Count | Signal |
|------|--------------|--------|
| Bugfix | 8-10 | "fix" in message |
| Feature | 15-20 | "add", "implement" |
| Refactor | 8-10 | "refactor", "rename" |
| Chore | 5-8 | deps, CI changes |
| Docs | 3-5 | doc file changes |

**Filter criteria**:
- Skip < 5 lines changed (trivial)
- Skip > 500 lines changed (too noisy for initial eval)
- Include mix of single-file and multi-file

**Git metadata extraction** - extend `git.Runner`:
```go
func (r *Runner) Message(ctx context.Context, repoPath, hash string) (string, error)
```

## Workflow

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  1. COLLECT     │    │  2. CLASSIFY    │    │  3. EVALUATE    │
│                 │    │                 │    │                 │
│  git histories  │───►│  run prompt on  │───►│  pass/fail +    │
│  → JSONL        │    │  each diff      │    │  detailed       │
│  (story: null)  │    │  → fill story   │    │  critique       │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                                      │
                                                      ▼
                              ┌─────────────────────────────────┐
                              │  4. AXIAL CODING                │
                              │                                 │
                              │  LLM groups critiques →         │
                              │  failure taxonomy               │
                              └─────────────────────────────────┘
```

## Implementation Phases

| Phase | Deliverable | Scope |
|-------|-------------|-------|
| **1. Domain types** | New types in root package | Small |
| **2. Git metadata** | Add `Message()` to GitRunner | Small |
| **3. Evalreview fix** | Critique text input | Medium |
| **4. Collection update** | `diffstory collect` uses new types | Small |
| **5. Prompt formatter** | Renders ClassificationInput for LLM | Small |
| **6. Classification prompt** | Prompt engineering | Medium |
| **7. Collect & classify** | 50 diffs with stories | Batch run |
| **8. Human review** | 30-50 judgments with critiques | Manual |
| **9. Axial coding** | Failure taxonomy | LLM-assisted |

## Success Criteria

After first eval round:
- Can determine if classification prompt produces usable output
- Have ~30 detailed critiques explaining failures
- Emergent failure taxonomy to guide prompt iteration

## Open Questions

1. **Which LLM for classification?** Gemini (existing integration) vs Claude vs other
2. **Prompt iteration strategy?** How many rounds before taxonomy stabilizes?
3. **Threshold for "good enough"?** What pass rate indicates readiness for viewer integration?
