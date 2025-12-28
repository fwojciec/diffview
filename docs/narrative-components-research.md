# Research: Narrative Components and Ordering for Code Review Storytelling

## Current Implementation Summary

### Narratives (5 types)
| Narrative | Description | Use Case |
|-----------|-------------|----------|
| cause-effect | A problem leads to a fix | Bugfixes |
| core-periphery | Central change with supporting updates | Features |
| before-after | Transformation from old to new pattern | Refactors |
| rule-instances | A pattern applied in multiple places | Systematic changes |
| entry-implementation | API/interface plus its implementation | New APIs |

### Roles (9 types)
| Role | Description |
|------|-------------|
| problem | Code that demonstrates the issue being fixed |
| fix | The actual solution code |
| test | Test code validating the change |
| core | Essential logic change |
| supporting | Updates needed due to core change |
| rule | The pattern being applied |
| exception | Cases that don't follow the rule |
| integration | API/interface definitions |
| cleanup | Removed/replaced code |

### Current Orderings
- **cause-effect**: problem → fix → test → supporting → cleanup
- **core-periphery**: core → supporting → cleanup
- **before-after**: cleanup → core → test → supporting
- **rule-instances**: rule → exception → core → supporting → cleanup
- **entry-implementation**: integration → core → test → supporting → cleanup

### Current Primitives
Only **diff hunks** - classified into sections with categories (refactoring, systematic, core, noise).

### Available but Unused Data
- Full commit history (only first message used)
- Git blame information
- File contents at any revision
- PR description/comments (not captured)

---

## Research Findings

### 1. Cognitive Science: How Developers Process Code Changes

**Key findings from academic research:**

1. **Narrative improves recall**: A study on semi-automated storytelling for software histories found that "participants were 16% more successful at recalling code history information, and had 30% less error when assessing correctness" when using story-view vs. list-view formats. ([IEEE Xplore](https://ieeexplore.ieee.org/document/10714574/))

2. **Cognitive load matters**: Research shows code review "requires deeper engagement of higher-level cognitive processes (e.g., decision making and analysis) than code comprehension alone." Checklists and structured approaches help reduce cognitive load. ([Springer](https://link.springer.com/article/10.1007/s10664-022-10123-8))

3. **Primacy/Recency effects**: Items at the beginning and end of a sequence are remembered best. File ordering in code reviews shows significant correlation with comment frequency - reviewers leave more comments on files shown earlier. ([arXiv](https://arxiv.org/html/2306.06956))

4. **Working memory limits**: Humans can process ~7±2 pieces of information at once. Progressive disclosure works by revealing information gradually, matching brain capacity. ([IxDF](https://www.interaction-design.org/literature/topics/progressive-disclosure))

5. **Narrative activates more brain regions**: Stories engage language, emotion, experience, and motor skills areas simultaneously, making information more memorable than isolated facts. ([Number Analytics](https://www.numberanalytics.com/blog/cognitive-science-narrative))

### 2. File Ordering Research

**Study: "Assessing the Impact of File Ordering Strategies on Code Review"** ([arXiv](https://arxiv.org/html/2306.06956))

Tested three strategies on 51,566 code reviews:
- **Alphabetical**: Current default in GitHub, Gerrit, JetBrains
- **Random**: Baseline
- **Code Diff**: Files ordered by lines changed (descending)

**Results:**
- Alphabetical significantly outperforms Random
- Code Diff significantly outperforms Alphabetical (by MRR and nDCG metrics)
- **Key insight**: Files with more changes should appear first

**Implications for our narratives:**
- "Core" changes should come before "supporting" changes (validates our pattern)
- However, **context before detail** is also valuable (problem before fix)
- The tension: should we show the most-changed files first, or the most-contextual?

### 3. Industry Tool Patterns

**Reviewable** ([docs](https://docs.reviewable.io/files.html)):
- Hierarchical file organization (files before subdirectories)
- Automatic grouping: reverted files, test files, renamed files, vendored deps
- Custom file groups via completion condition scripts
- Progressive disclosure: single-file mode for large diffs, auto-collapsed sections
- Diff suppression for binary/minified/oversized files

**Gerrit**:
- Topics for grouping related changes
- Patch-based review (each commit as a patch set)
- Selection-based comments (not line-based)
- Version comparison (diff v0 vs v1 of a change)
- OWNERS files for directory-based reviewers

**Stacked PRs** ([Michaela Greiler](https://www.michaelagreiler.com/stacked-pull-requests/)):
- Break features into logical layers (DB → business logic → UI)
- Each PR builds on previous one
- Creates natural progression for reviewers
- "Commits represent developer's progress; stacked PRs represent reviewer's comprehension"

### 4. Storytelling in Code Review

**Doppler Engineering** ([blog](https://www.doppler.com/blog/improving-code-reviews-with-storytelling)):
- Treat git history as narrative, not just log
- Preparatory groundwork first (refactors, setup)
- Logical progression where each commit builds on previous
- Focused changes addressing specific concerns

**Best practices from industry:**
- Optimal PR size: ~50 lines (40% faster to merge than 250 lines) ([Swarmia](https://www.swarmia.com/blog/why-small-pull-requests-are-better/))
- Never mix unrelated changes
- Handle refactors in separate PRs ([Artsy](https://artsy.github.io/blog/2021/03/09/strategies-for-small-focused-pull-requests/))
- Explain "why" not just "what" in commit messages ([Corgibytes](https://corgibytes.com/blog/2019/03/20/commit-messages/))

### 5. Narrative Structures

**Freytag's Pyramid** (5 acts):
1. Exposition (setup)
2. Rising action (complications)
3. Climax (turning point)
4. Falling action (consequences)
5. Denouement (resolution)

**Three-Act Structure**:
1. Setup (introduce context)
2. Confrontation (the problem/change)
3. Resolution (outcome/tests)

**Application to code review:**

| Literary Structure | Code Review Mapping |
|-------------------|---------------------|
| Exposition | Context: the code before, the problem |
| Rising Action | The fix being applied |
| Climax | Core behavioral change |
| Falling Action | Ripple effects, supporting changes |
| Resolution | Tests proving it works |

---

## Primitives Evaluation

### Current Primitive: Diff Hunks

**Strengths:**
- Direct representation of what changed
- Universal across all git workflows
- Easy to parse and display

**Weaknesses:**
- No "before" context beyond diff lines
- No evolution story (how we got here)
- No semantic information about code relationships

### Candidate Primitives

| Primitive | Value | Complexity | Recommendation |
|-----------|-------|------------|----------------|
| **Commit progression** | High - shows evolution of thought | Medium | **YES**: Already have commits, just need UI |
| **PR description** | High - author's intent | Low | **YES**: Easy API integration |
| **Before snapshots** | Medium - context for "before-after" | Low | **MAYBE**: Only for specific narratives |
| **Blame data** | Low - mostly historical curiosity | Low | **NO**: Adds noise, not story |
| **Unchanged context** | Medium - related code | High | **NO**: Requires semantic analysis |
| **Test output** | Medium - behavioral proof | Medium | **LATER**: CI integration complexity |
| **Type information** | Low - LSP complexity | High | **NO**: Out of scope |

### Recommended New Primitives

#### 1. Commit Progression (Priority: High)
**Value proposition**: Shows how the change evolved, supports stacked PR mental model.

**Schema change**:
```go
type ClassificationInput struct {
    // ... existing fields
    CommitProgression []CommitStep `json:"commit_progression,omitempty"`
}

type CommitStep struct {
    Hash     string    `json:"hash"`
    Message  string    `json:"message"`
    Summary  string    `json:"summary"`      // LLM-generated summary
    Role     string    `json:"role"`         // Same as section roles
    Hunks    []HunkRef `json:"hunks"`        // Hunks introduced in this commit
}
```

**UX concept**: "Commit mode" toggle showing changes step-by-step instead of squashed.

#### 2. PR Description (Priority: High)
**Value proposition**: Captures author's stated intent and context.

**Schema change**:
```go
type ClassificationInput struct {
    // ... existing fields
    PRDescription string `json:"pr_description,omitempty"`
    PRTitle       string `json:"pr_title,omitempty"`
}
```

**Usage**: Include in classifier prompt for better narrative selection.

---

## Narrative Analysis

### Validated Narratives

| Narrative | Keep? | Rationale |
|-----------|-------|-----------|
| cause-effect | ✅ YES | Maps to three-act structure, natural for bugfixes |
| core-periphery | ✅ YES | Matches "focal change + ripples" pattern |
| before-after | ✅ YES | Clear transformation story, good for refactors |
| rule-instances | ⚠️ MAYBE | Useful but rare; could merge with core-periphery |
| entry-implementation | ✅ YES | Common API pattern, distinct from core-periphery |

### Role Analysis

| Role | Keep? | Rationale |
|------|-------|-----------|
| problem | ✅ YES | Critical for cause-effect exposition |
| fix | ✅ YES | Core of cause-effect resolution |
| test | ✅ YES | Universal validation role |
| core | ✅ YES | Central to most narratives |
| supporting | ✅ YES | Ripple effects, well understood |
| rule | ⚠️ RENAME | "pattern" is clearer; merge into core-periphery? |
| exception | ⚠️ MERGE | Rare; merge with "supporting" |
| integration | ⚠️ RENAME | "interface" is clearer |
| cleanup | ✅ YES | Distinct role: removal/replacement |

### Ordering Principles

Based on research, orderings should follow these principles:

1. **Context before detail**: Show the "why" before the "what" (matches exposition → action)
2. **High-impact first**: Files with more changes should appear earlier (arXiv finding)
3. **Tests as validation**: Tests belong at the end as "proof" (denouement)
4. **Cleanup position varies**:
   - In "before-after": cleanup first (shows what's being removed)
   - In others: cleanup last (shows what was cleaned up after)

### Narrative Selection Decision Tree

```
START: What is the primary nature of this change?
│
├─► Is it fixing a bug or issue?
│   └─► YES: Use cause-effect
│       (problem → fix → test → supporting → cleanup)
│
├─► Is it replacing an old pattern with a new one?
│   └─► YES: Use before-after
│       (cleanup → core → supporting → test)
│
├─► Is it adding a new API/interface with implementation?
│   └─► YES: Use entry-implementation
│       (interface → implementation → test → supporting → cleanup)
│
├─► Is it applying the same pattern in multiple places?
│   └─► YES: Use rule-instances
│       (pattern → instances → test → cleanup)
│
└─► Otherwise (feature, enhancement, general change):
    └─► Use core-periphery
        (core → supporting → cleanup)
```

**Heuristics for Edge Cases:**

1. **Mixed changes**: If a PR has both bug fix and feature work, choose the narrative that best describes the *primary* change
2. **Refactors without replacement**: Pure refactors that don't replace a pattern → use core-periphery
3. **Tests-only changes**: Use core-periphery with tests as the "core"
4. **Chore/config changes**: Use core-periphery (simple, catches all)

### Recommended Orderings

| Narrative | Current | Recommended | Rationale |
|-----------|---------|-------------|-----------|
| cause-effect | problem → fix → test → supporting → cleanup | **Keep** | Follows Freytag perfectly |
| core-periphery | core → supporting → cleanup | **Keep** | High-impact first, ripples after |
| before-after | cleanup → core → test → supporting | cleanup → core → supporting → test | Tests validate the "after" state |
| rule-instances | rule → exception → core → supporting → cleanup | pattern → instances → test → cleanup | Clearer naming |
| entry-implementation | integration → core → test → supporting → cleanup | interface → implementation → test → supporting → cleanup | Clearer naming |

---

## Implementation Recommendations

### Phase 1: Schema Refinements (Low effort)

1. **Rename roles for clarity**:
   - `rule` → `pattern`
   - `integration` → `interface`

2. **Merge `exception` into `supporting`**:
   - Exceptions are a type of supporting change
   - Reduces role proliferation

3. **Add PR metadata fields**:
   ```go
   PRTitle       string `json:"pr_title,omitempty"`
   PRDescription string `json:"pr_description,omitempty"`
   ```

### Phase 2: Classifier Prompt Updates (Medium effort)

1. **Update narrative descriptions** with cognitive principles:
   - Reference "exposition → action → resolution" framing
   - Explain why each ordering works

2. **Add classification guidelines**:
   - Decision tree for narrative selection
   - Examples of each narrative type

3. **Include PR description** in classification context

### Phase 3: Commit Progression Support (Higher effort)

1. **Enhance ClassificationInput**:
   - Add `CommitProgression` field
   - Link hunks to originating commits

2. **Add viewer mode**:
   - "Story mode" (current, squashed)
   - "Evolution mode" (commit-by-commit)

3. **LLM summarization**:
   - Generate per-commit summaries
   - Assign roles to commits, not just hunks

---

## Sources

- [Semi-automated storytelling for software histories](https://ieeexplore.ieee.org/document/10714574/) - IEEE study on narrative vs list views
- [Assessing the Impact of File Ordering Strategies](https://arxiv.org/html/2306.06956) - arXiv study on file ordering
- [Reviewable docs on files](https://docs.reviewable.io/files.html) - Industry patterns
- [Stacked pull requests](https://www.michaelagreiler.com/stacked-pull-requests/) - Dr. Michaela Greiler
- [Improving Code Reviews with Storytelling](https://www.doppler.com/blog/improving-code-reviews-with-storytelling) - Doppler Engineering
- [Why small pull requests are better](https://www.swarmia.com/blog/why-small-pull-requests-are-better/) - Swarmia
- [Software as storytelling: A systematic literature review](https://www.sciencedirect.com/science/article/abs/pii/S157401372200051X) - Academic review
- [Cognitive Science of Narrative](https://www.numberanalytics.com/blog/cognitive-science-narrative) - Narrative cognition
- [Progressive Disclosure](https://www.interaction-design.org/literature/topics/progressive-disclosure) - IxDF
