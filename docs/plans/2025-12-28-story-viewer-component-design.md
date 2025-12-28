# Story Viewer Component Design

Reusable Bubble Tea component for story-aware diff viewing with section navigation, hunk collapsing, and category-based styling.

## Overview

**Purpose**: Display diffs with narrative structure from LLM classification. Enables reviewers to navigate by story sections rather than just files/hunks.

**Scope**: This issue delivers the `bubbletea/StoryModel` component only. Follow-up issues will wire it into `diffstory` binary and `evalreview`.

## Data Model

```go
// StoryModel displays a diff with story-aware navigation and styling.
type StoryModel struct {
    diff    *diffview.Diff
    story   *diffview.StoryClassification

    // Pre-computed mappings (built on construction)
    hunkToSection    map[hunkKey]int    // file:hunkIdx → section index
    sectionPositions []int              // line numbers where sections start
    collapsedHunks   map[hunkKey]bool   // tracks runtime collapse state

    // UI state
    viewport viewport.Model
    keymap   StoryKeyMap
    styles   diffview.Styles
    width    int
    ready    bool
}

type hunkKey struct {
    file      string
    hunkIndex int
}
```

### Collapse Logic

1. If `HunkRef.Collapsed == true` → collapsed initially
2. If `HunkRef.Category == "noise"` → collapsed by default
3. User can toggle any hunk with `o`, or all with `z`

## Rendering

### Collapsed Hunks

Render as single styled line instead of full content:

```
@@ -50,8 +52,10 @@ func Validate
  ▸ [refactoring] Renamed variable for clarity
```

Uses `HunkRef.CollapseText` for the summary line.

### Category Styling

| Category | Style |
|----------|-------|
| `core` | Normal (prominent) |
| `refactoring` | Dimmed foreground |
| `systematic` | Dimmed foreground |
| `noise` | Very dimmed, collapsed by default |

Applied to hunk header and content lines.

### Section Indicator

Status bar shows current section (no inline section headers for MVP):

```
file 2/5 │ hunk 3/8 │ section 2/3: Core Changes │ 45% │ ...
```

## Keybindings

### Section Navigation (new)
| Key | Action |
|-----|--------|
| `s` | Jump to next section |
| `S` | Jump to previous section |

### Hunk Collapsing (new)
| Key | Action |
|-----|--------|
| `o` | Toggle collapse on current hunk |
| `z` | Toggle all collapsed/expanded |

### Existing (preserved)
| Key | Action |
|-----|--------|
| `j/k` | Scroll up/down |
| `n/N` | Next/prev hunk |
| `]/[` | Next/prev file |
| `ctrl+u/d` | Half page up/down |
| `gg/G` | Top/bottom |
| `q` | Quit |

## Architecture

Following Ben Johnson Standard Package Layout.

### File Structure

```
bubbletea/
├── viewer.go          # Existing diff viewer (unchanged)
├── story.go           # NEW: StoryModel
├── story_keymap.go    # NEW: StoryKeyMap
├── story_test.go      # NEW: Tests
├── render.go          # Extract shared rendering, parameterized
└── eval.go            # Existing (future integration)
```

### Shared Rendering

Extract `renderDiff` to `render.go` with parameters for:
- Collapse state (`map[hunkKey]bool`)
- Category lookup (`map[hunkKey]string`)
- Category styles

Both `Model` and `StoryModel` use the same renderer with different parameters.

## Testing Strategy

1. **Unit tests** for position computation
   - `computeSectionPositions`
   - `buildHunkLookup`
   - Collapse state initialization

2. **Golden file tests** for rendering
   - Collapsed hunk output
   - Category styling

3. **Integration tests** for navigation
   - Section jumping
   - Collapse toggling

## Validation Criteria

- [ ] Can navigate between sections with `s/S`
- [ ] Collapsed hunks render as single line showing CollapseText
- [ ] Noise category hunks collapsed by default
- [ ] Category affects hunk visual treatment (dimmed)
- [ ] Section indicator shows in status bar
- [ ] `make validate` passes
