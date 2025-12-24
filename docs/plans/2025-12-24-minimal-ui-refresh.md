# Minimal UI Refresh: Less Is More

Design direction for diffview's visual overhaul, moving from traditional diff presentation toward a cleaner interface optimized for reviewing AI-generated code.

## Core Insight

The current UI answers: "What lines changed?"

AI code reviewers need to answer: "Did the AI do what I asked?"

Traditional diff viewers inherit `diff -u` conventions designed for humans reviewing other humans' code. That mental model doesn't apply when reviewing AI output, where comprehension and verification matter more than line-by-line approval.

## Design Principles

1. **Whitespace is information** - What you don't show communicates hierarchy
2. **One signal per change** - Don't stack gutter + background + word highlighting
3. **Remove artifacts** - Hunk headers are git plumbing, not comprehension aids
4. **De-emphasize metadata** - Line numbers rarely matter for AI code review

## Changes

### Immediate (P2)

| Issue | Change |
|-------|--------|
| diffview-3xh | Whitespace between hunks |
| diffview-bt0 | Whitespace before file headers |
| diffview-arm | Reduce background intensity (subtle tint, not full saturation) |
| diffview-9mj | Line numbers only for changed lines, muted style |
| diffview-75w | Minimal status bar, keybindings in `?` popup |
| diffview-831 | Remove hunk headers entirely (blocked by 3xh) |

### Research (P3)

| Issue | Topic |
|-------|-------|
| diffview-206 | Collapsible sections UI primitive |
| diffview-dn2 | Syntax highlighting (approach matters with new visual direction) |

## Future: Intelligence Layer

Long-term vision includes an LLM-powered layer that can:
- Generate semantic labels for changes ("Modified error handling in fetchUser")
- Reorder hunks to tell a coherent story
- Identify signal vs noise (what's important vs trivial)
- Incorporate agent workflow logs as context

The current work establishes UI primitives that will support this direction.

## Deferred

Closed theme variant issues (colorblind, high-contrast, github-style) - will revisit once new visual primitives are in place. The infrastructure (theme flag, color system) remains valuable.
