# Word-Level Diff Highlighting: Production Implementation Guide

**Neither GitHub, GitLab, VS Code, nor JetBrains use diff-match-patch for code diffing.** All major tools implement Myers algorithm variants with custom tokenization, and critically, diff-match-patch's semantic cleanup is prose-optimized—making it suboptimal for code. The winning strategy: tokenize code into identifiers, operators, and literals first, then apply array-based diffing to avoid partial identifier highlighting entirely.

## Production tools use two-phase diffing

Every major code review tool follows the same architectural pattern: line-level diff first, then intra-line highlighting on paired lines that pass a similarity threshold.

**GitLab's implementation** (fully open source in `lib/gitlab/diff/`) uses:
- `PairSelector` to match deleted/added line pairs for word-level comparison
- `CharDiff` for character-level comparison on those paired lines
- `InlineDiffMarker` to wrap changed sections in HTML spans
- Redis-backed `HighlightCache` for performance

**VS Code uses Myers' O(ND) algorithm** with two implementations:
- Legacy `DiffComputer` produces `charChanges` arrays within each `ILineChange`
- New `DefaultLinesDiffComputer` (default since v1.82) adds code move detection
- Key configuration: **5-second hard timeout** for character-level diff computation

**JetBrains computes "inner fragments"** via `ComparisonManager.compareLinesInner()`, with word-boundary awareness that VS Code lacks. Their highlight colors are derived algorithmically—word highlight is computed as a "brighter" version of line background color.

| Tool | Algorithm | Granularity | Open Source |
|------|-----------|-------------|-------------|
| GitLab | Custom + Myers | Character-level | Yes |
| VS Code | Myers O(ND) | Character-level | Yes |
| JetBrains | Custom | Word-boundary aware | Partial |
| Delta | Myers + Levenshtein | Word-level | Yes |
| Difftastic | Dijkstra on AST DAG | Structural | Yes |

## Tokenize first to avoid partial identifier highlighting

The core problem with character-level diffing: `myVariable` vs `myValue` highlights `myVa` as common and `riable`→`lue` as changed. This is confusing for code review.

**The solution is array-based diffing on tokens:**

```go
// Generic code tokenization pattern
tokenRegex := `([a-zA-Z_][a-zA-Z0-9_]*)|` +  // identifiers
              `([0-9]+\.?[0-9]*)|` +          // numbers
              `("[^"]*"|'[^']*')|` +          // string literals
              `([+\-*/=<>!&|^%]+)|` +         // operators
              `([(){}\[\];,.])|` +            // punctuation
              `(\s+)`                          // whitespace
```

**jsdiff's `diffWords()`** implements this pattern, treating "each word and each punctuation mark as a token." When you diff `["myVariable"]` against `["myValue"]` at the array level, the entire token is marked as changed—no partial highlighting.

VS Code notably does **not** tokenize by words—it runs character-level LCS on modified lines. JetBrains is word-boundary aware. This is the key differentiator in highlighting quality.

## Similarity thresholds determine when word-diff helps

If two lines are completely different, word-level highlighting produces noise. Production tools skip intra-line diffing below certain thresholds:

**Levenshtein ratio formula:** `(len1 + len2 - editDistance) / (len1 + len2)`

| Ratio | Action |
|-------|--------|
| < 0.3 | Skip word-diff entirely—show as full replacement |
| 0.3–0.5 | Word-diff optional, may be noisy |
| 0.5–0.8 | Show word-diff—changes are meaningful |
| > 0.8 | Definitely show word-diff—highlight small changes |

**Jaccard similarity of tokens** provides a faster alternative: `|A ∩ B| / |A ∪ B|` where A and B are token sets. Common thresholds: **0.4–0.5** as minimum for word-level diff, **0.7** as "highly similar."

GitLab's `PairSelector` class implements this threshold logic, though the exact values aren't publicly documented. VS Code's approach is simpler—compute character diff if any, with the 5-second timeout as the primary safeguard.

## DiffCleanupSemantic is wrong for code

diff-match-patch's `DiffCleanupSemantic` eliminates "semantically trivial equalities"—short common substrings that interrupt larger changes. It tries to align edits to **word boundaries for prose readability**.

**How it fails for code:**
- Optimized for English prose, not identifier boundaries
- Known bug: `text` vs `test` doesn't clean up properly (GitHub issue #104)
- No awareness that `userName` and `user_Name` should be treated as atomic tokens
- `DiffCleanupSemanticLossless` helps but doesn't understand code structure

**There is no code-aware cleanup equivalent in the library.** The correct solution: tokenize before diffing so cleanup isn't needed.

If you must use diff-match-patch at character level, the library provides a workaround: map each line/word to a unique Unicode character, run character diff, then map back. But this adds complexity that token-based diffing avoids.

## Algorithm recommendations for your Go TUI

**Histogram diff** (Git's `--diff-algorithm=histogram`) produces the cleanest results for code. It extends patience diff by matching lines with lowest occurrence count, not just unique lines. Available in Git 1.7.7+ and jgit.

**For word-level highlighting specifically:**

1. **Delta's approach** (dandavison/delta): Line-based Myers diff, then Levenshtein edit inference for word highlighting within modified line pairs
2. **diffr's approach**: "Works hunk by hunk, recomputing the diff on a word-by-word basis" using Myers LCS
3. **Patdiff** (Jane Street, OCaml): Patience diff with "good word-level diffing out of the box"

**Go libraries to consider:**
- `sergi/go-diff` — Go port of diff-match-patch
- `pmezard/go-difflib` — Python difflib port, provides `SequenceMatcher`
- Direct Myers implementation with custom tokenization

The most robust pattern:
```
1. Line-level diff (patience or histogram)
2. For each changed line pair:
   - Compute similarity ratio
   - If ratio >= 0.4: tokenize both lines, diff token arrays
   - Else: show as full line replacement
```

## Performance is not a concern for typical diffs

diff-match-patch achieves **~55ms for large text diffs** in optimized implementations. Myers algorithm is O(ND) where D is edit distance—essentially quadratic worst-case, but "very fast for similar inputs, which is quite common."

**VS Code's safeguards:**
- **5000ms timeout** for overall diff computation
- **50MB max file size** default
- **20,000 character limit** per line for syntax highlighting
- Character-level diff may be skipped for very long lines

**Caching strategies for TUI:**
- LRU cache keyed by `hash(old_line + new_line)` → word diff result
- Pre-compute for visible viewport ± 50 lines
- Lazy computation: show line-level immediately, add word highlighting async

**Fallback chain:**
```
1. Try token-level diff with 100ms timeout
2. If similarity < 0.3: line-level only
3. If > 10 change regions per line: simplify to line-level
4. If timeout: abort and use line-level
```

GitLab uses Redis-backed `HighlightCache` and suppresses files over **5000 lines**—you can implement simpler in-memory caching for a TUI.

## Edge case handling from production tools

**Whitespace-only changes:** Git's `diff.colorMovedWs = allow-indentation-change` ignores indentation for moved code detection. Difftastic "understands when whitespace matters, and when it's just an indentation change." Best practice: offer a toggle to show/hide whitespace-only changes.

**String literals:** Difftastic notes that "changing large string literals is a challenge—syntactically they're single atoms, but users sometimes want a word-level diff." Recommendation: detect string literals as tokens, but apply character-level diff *within* them. Treat escape sequences (`\n`, `\"`, `\\`) as single tokens.

**Long lines with minimal changes:** Delta and diffr both solve this with intra-line word diffing. For lines over 500–1000 characters, consider character-level only or horizontal scrolling.

**Renamed identifiers:** Text-based diffs will show all occurrences as changed—this is unavoidable without semantic analysis. Structural diff tools (difftastic, GumTree) detect renames via AST matching.

**Unicode:** diff-match-patch has known issues with Unicode surrogates. Work on grapheme clusters, not code points. Test with emoji and non-ASCII identifiers.

## Recommended implementation for your Go TUI

```go
// Core architecture
type LinePair struct {
    Old, New string
    Similarity float64
    WordDiff []DiffSegment // nil if similarity too low
}

func computeWordDiff(old, new string) *LinePair {
    sim := levenshteinRatio(old, new)
    if sim < 0.4 {
        return &LinePair{old, new, sim, nil}
    }
    
    oldTokens := tokenizeCode(old)
    newTokens := tokenizeCode(new)
    diff := myersDiffArrays(oldTokens, newTokens)
    
    return &LinePair{old, new, sim, diff}
}
```

Key parameters to expose:
- Similarity threshold (default 0.4)
- Max line length for word-diff (default 1000)
- Computation timeout (default 100ms per line pair)
- Show/hide whitespace toggle

The most impactful change from your current approach: **replace character-level diff-match-patch with token-based array diffing**. This single change eliminates partial identifier highlighting and produces cleaner, more readable diffs.

## Conclusion

Production diff tools converge on a consistent pattern: Myers-variant algorithms, token-based (not character-based) word diffing, and similarity thresholds to avoid noisy highlighting. For your Go TUI, drop `DiffCleanupSemantic` in favor of proper code tokenization before diffing—this is the single highest-impact improvement. Use similarity thresholds around **0.4** to decide when word-level highlighting helps, implement a simple LRU cache for computed diffs, and add a fallback timeout for pathological cases.
