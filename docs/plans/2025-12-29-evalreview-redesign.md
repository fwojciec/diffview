# Evalreview Redesign

**Date:** 2025-12-29
**Status:** Approved

## Problem

The current evalreview UI is too busy. Split-panel layout crams too much information into limited space, making error analysis difficult. The tool should optimize for browsing outputs and spotting patterns, not complex interactions.

## Design Principles

From Hamel Husain's eval guidance:
- **Error analysis first** - the primary task is browsing outputs to find failure patterns
- **One-click judgments** - pass/fail should be trivial, not the focus
- **Custom tools enable 10x iteration speed** - worth investing in good UX

From frontend design:
- **Utilitarian aesthetic** - maximum content, minimal chrome
- **Full-screen focus** - one view at a time, no split panels
- **KISS** - remove unnecessary features

## Solution

Two full-screen views, toggled with `Tab`:

### Story View

Experience the classification as a reviewer would. Vertical split (resizable with `+`/`-`):

```
┌─────────────────────────────────────────────────────────┐
│ [fix] Index Mapping Logic                    section 2/5│
│ role: fix                                               │
│                                                         │
│ This is the core of the fix. The renderer is updated   │
│ to accept an index map...                              │
├─────────────────────────────────────────────────────────┤
│ ── bubbletea/eval.go ─────────────────────────── +13 -7 │
│ @@ -808,11 +809,12 @@ func (m *EvalModel) toggle...    │
│  808 809     m.updateViewportContent()                  │
│  811     -// filteredDiff returns a diff containing...  │
│  812     +// filteredDiffWithIndices returns a diff...  │
└─────────────────────────────────────────────────────────┘
[story] │ section 2/5 │ case 12/44 │ ○ unset │ [p]ass [f]ail
```

**Top pane:** Section title, role badge, explanation text
**Bottom pane:** Diff hunks for current section only, syntax highlighted

### Data View

Inspect the classification as structured data. Full tree, always expanded, scroll-only:

```
┌─────────────────────────────────────────────────────────┐
│ CLASSIFICATION                                          │
│                                                         │
│ change_type: bugfix                                     │
│ narrative:   cause-effect                               │
│ summary:     Fix incorrect metadata lookups in filtered │
│              diff views by mapping local hunk indices   │
│                                                         │
│ ── sections ────────────────────────────────────────────│
│                                                         │
│ [problem] Identify the Bug                              │
│   explanation: Shows the buggy code that caused...      │
│   hunks:                                                │
│     bubbletea/eval.go:H0    core      visible           │
│                                                         │
│ [fix] Index Mapping Logic                               │
│   explanation: This is the core of the fix...           │
│   hunks:                                                │
│     bubbletea/render.go:H0  core      collapsed         │
│     bubbletea/render.go:H1  core      visible           │
└─────────────────────────────────────────────────────────┘
[data] │ case 12/44 │ ○ unset │ [p]ass [f]ail
```

### Footer

Minimal, one line:
```
[story] │ section 2/5 │ case 12/44 │ ○ unset │ n/N case ]/[ section p/f judge
```

- Mode indicator: `[story]` or `[data]`
- Position: section X/Y (story view only), case X/Y
- Judgment state: `○ unset`, `✓ pass`, `✗ fail`
- Key hints

### Keybindings

| Key | Action |
|-----|--------|
| `Tab` | Toggle story ↔ data view |
| `j/k` | Scroll |
| `n/N` | Next/prev case |
| `]/[` | Next/prev section (story view) |
| `p` | Mark pass |
| `f` | Mark fail |
| `c` | Open critique modal |
| `+/-` | Resize story/diff split (story view) |
| `?` | Help |
| `q` | Quit |

## Removed Features

- Split panel layout (replaced with full-screen views)
- Case saving (use diffstory for that)
- Yank to clipboard
- Complex collapsed hunk toggling

## Future Enhancements

Separate issues:
- **Validation warnings in Data View** - highlight coverage gaps, empty sections, invalid indices
- **Quick filtering** - show only unreviewed cases, only failures, etc.

## Implementation Notes

- Reuse existing `StoryModel` primitives for Story View rendering
- Data View is new: render classification tree as styled text
- EvalModel simplifies significantly - remove panel switching, clipboard, case saver
