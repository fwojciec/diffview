# Diff Story Classifier: Plan 1

A three-layer classification system for transforming raw git diffs into comprehensible narratives.

## Problem

Traditional diffs present changes file-by-file, forcing reviewers to mentally reconstruct the "story" of what happened. Research shows this approach:
- Scatters logical changes across files, increasing cognitive load
- Treats all hunks equally, regardless of importance
- Provides no guidance on reading order
- Misses opportunities to collapse repetitive patterns (9.3x concision possible)

## Solution: Three-Layer Classification

### Layer 1: Hunk Classification

Classify each hunk by its nature. This determines presentation strategy.

| Category | Signal | Presentation Strategy |
|----------|--------|----------------------|
| **Refactoring** | Structure change, no behavior change (rename, move, extract, inline) | Collapse to operation description: "Method `foo` moved to `Bar` class" |
| **Systematic** | Same pattern applied ≥3 times across hunks | Collapse to rule + exceptions: "All `lock()` calls now use `lock(timeout)`" |
| **Core Logic** | Actual behavior change - new algorithms, bug fixes, feature code | Full context, expanded view, show first |
| **Noise** | Imports, whitespace, formatting, generated code, trivial config | Auto-fold, show count only: "12 import changes (hidden)" |

**Classification signals for LLM:**
- Refactoring: AST structure changes but logic equivalent; variable/function/class renamed; code moved between files/scopes
- Systematic: Multiple hunks share transformation pattern; consistent API change; cross-cutting concern (logging, error handling)
- Core Logic: Control flow changes; new conditionals; algorithm modifications; business logic
- Noise: Only import statements; only whitespace; auto-generated markers; lockfile changes

### Layer 2: Change Type

Infer the overall PR/commit type from hunk distribution and commit message.

| Type | Hunk Distribution | Commit Message Signals |
|------|-------------------|----------------------|
| **Bugfix** | Core logic focused on fix; may have test additions | "fix", "bug", "issue", "crash", "error", issue references |
| **Feature** | Core logic = new code; supporting integration hunks | "add", "implement", "feature", "support", "enable" |
| **Refactor** | Dominated by refactoring hunks; minimal core logic | "refactor", "rename", "move", "extract", "clean" |
| **Chore** | Dominated by noise; deps, CI, config | "chore", "deps", "ci", "config", "update dependencies" |
| **Docs** | Only documentation files | "docs", "readme", "documentation" |

**Inference rule:**
```
if 80%+ hunks are noise → Chore
if 80%+ hunks are refactoring → Refactor
if commit message signals bugfix AND has core logic → Bugfix
if has new files with core logic → Feature
else → Feature (default)
```

### Layer 3: Narrative Pattern

Select arrangement strategy based on change type.

| Pattern | Use When | Arrangement |
|---------|----------|-------------|
| **Cause → Effect** | Bugfix | 1. Problem context (what was wrong) 2. The fix (core logic) 3. Verification (tests) |
| **Core → Periphery** | Feature | 1. Main new logic 2. Supporting changes 3. Integration glue 4. Noise (folded) |
| **Before → After** | Refactor | 1. Summary of transformation 2. Key structural changes 3. Mechanical follow-ons |
| **Rule → Instances** | Systematic-heavy | 1. Inferred rule in natural language 2. Representative examples 3. Exceptions/anomalies |
| **Entry → Implementation** | Complex feature | 1. Public API/interface 2. Internal implementation 3. Helper utilities |

**Pattern selection:**
```
if change_type == Bugfix → Cause → Effect
if change_type == Refactor → Before → After
if systematic_hunks > 50% of non-noise → Rule → Instances
if change_type == Feature AND has_new_public_api → Entry → Implementation
if change_type == Feature → Core → Periphery
```

## Output Schema

```go
type DiffStory struct {
    ChangeType    string      // "bugfix", "feature", "refactor", "chore", "docs"
    Narrative     string      // "cause-effect", "core-periphery", "before-after", "rule-instances", "entry-implementation"
    Summary       string      // One sentence: "Fixes auth token expiry bug by adding refresh logic"

    Sections      []Section   // Ordered for narrative flow
}

type Section struct {
    Role          string      // "problem", "fix", "test", "core", "supporting", "rule", "exception", etc.
    Title         string      // "The Bug", "The Fix", "API Changes", etc.
    Hunks         []HunkRef   // References to actual diff hunks
    Explanation   string      // Why this section matters
}

type HunkRef struct {
    File          string
    HunkIndex     int
    Category      string      // "refactoring", "systematic", "core", "noise"
    Collapsed     bool        // Whether to show collapsed by default
    CollapseText  string      // If collapsed: "Renamed foo → bar"
}
```

## Systematic Rule Inference

For hunks classified as "systematic", infer and describe the rule:

**Rule structure:** "For all [Scope], [Transformation] occurred, except [Exceptions]"

**Examples:**
- "All `database.Query` calls now include a `context.Context` first argument"
- "All error returns in `pkg/auth` now wrap with `fmt.Errorf`"
- "All test files added `t.Parallel()` at function start"

**Exception surfacing:** Highlight where the pattern *wasn't* applied - these are often bugs or missed updates.

**Thresholds (from LSDiff research):**
- Minimum support: 3 instances to qualify as systematic
- Minimum accuracy: 75% (exceptions < 25% of matches)

## Implementation Phases

### Phase 1: Hunk Classifier
- Input: Raw diff hunks with file context
- Output: Category per hunk (refactoring/systematic/core/noise)
- Eval: Human-labeled sample of real PRs

### Phase 2: Change Type Inference
- Input: Hunk categories + commit message
- Output: Change type + confidence
- Eval: Compare to conventional commit labels where available

### Phase 3: Narrative Arrangement
- Input: Classified hunks + change type
- Output: Ordered sections with explanations
- Eval: A/B test comprehension speed vs file-ordered

### Phase 4: Systematic Rule Inference
- Input: Hunks classified as systematic
- Output: Natural language rule + exceptions
- Eval: Rule accuracy, exception recall

## Open Questions

1. **Granularity:** Classify at hunk level or file level? Hunks can mix categories within a file.

2. **Context window:** How much surrounding code does the LLM need to classify accurately?

3. **Confidence thresholds:** When should the system fall back to basic file-ordered view?

4. **Multi-category hunks:** A hunk might be both refactoring AND core logic. How to handle?

5. **Cross-file systematic detection:** Need to see all hunks together to detect patterns. Chunking strategy?

## References

- LSDiff: Logical Structural Differencing (9.3x concision via rule inference)
- RefMerge: Refactoring-aware merge tool (17 refactoring types)
- Letovsky's code comprehension framework (specification/implementation/annotation layers)
- docs/intellingent-diff-presentation.md (cognitive science foundations)
- docs/from-diff-viewer-to-feedback-interface.md (philosophical framing)
