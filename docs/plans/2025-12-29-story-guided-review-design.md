# Story-Guided Review Mode for Evalreview

## Problem

The current evalreview UI shows a raw diff panel and a story text panel side-by-side. Reviewers must mentally map the story sections to the diff hunks. The classification output should *drive* the review experience, not just display metadata.

## Solution

Add **story mode** to EvalModel as the default viewing mode. Story mode provides section-by-section navigation through the classification, filtering the diff to show only hunks relevant to the current section.

## Design

### Mode Structure

- **Story mode (default)**: Section-by-section navigation with filtered diff view
- **Raw mode**: Full diff display (current behavior), accessible via toggle
- **Fallback**: Cases without a Story automatically use raw mode

Toggle with `m` key. Status bar indicates current mode.

### Story Mode Layout

**Top panel (larger)**: Section-filtered diff view
- Shows only hunks belonging to the current section
- Uses same rendering as StoryModel (collapsing, category styling)
- Section header at top: `[role] Title - Explanation...`

**Bottom panel (smaller)**: Section progress indicator
- Compact view: `Section 2/5: ✓ ✓ ● ○ ○`
- Current section title visible

**Status bar**: `story mode | section 2/5 | p:pass f:fail m:raw mode`

### Navigation & Progress

**Section navigation**:
- `s` / `S`: Next/previous section
- Navigating away marks section as "reviewed" (implicit)
- Cannot navigate past last section - must judge

**Within-section scrolling**:
- `j/k`, `ctrl+d/u`, `g/G`: Standard viewport scrolling

**Review flow**:
1. Start at section 1
2. Navigate through sections with `s`, each auto-marked reviewed
3. After last section, `s` does nothing
4. `p` or `f` records judgment, auto-advances to next case

**Backtracking**: `S` goes back. Revisiting doesn't un-review.

### Implementation Approach

Extract key behaviors from StoryModel rather than embedding it:

1. **Reuse `renderDiff` with filtering** - StoryModel's `filteredDiff()` logic
2. **Reuse collapse/category maps** - Same construction as StoryModel
3. **New state in EvalModel**:
   ```go
   storyMode        bool
   activeSection    int
   reviewedSections map[int]bool  // reset per case
   collapsedHunks   map[hunkKey]bool
   hunkCategories   map[hunkKey]string
   collapseText     map[hunkKey]string
   ```

**Why not embed StoryModel?** EvalModel coordinates two viewports, tracks judgments, handles case navigation - different lifecycle. Sharing rendering logic is cleaner.

**Key methods to add**:
- `rebuildStoryMaps()` - builds hunk maps when case changes
- `filteredDiff()` - returns diff with only current section's hunks
- `markSectionReviewed()` - called on section navigation
- `renderSectionProgress()` - checkmark indicator for bottom panel

### Keybindings (Story Mode)

| Key | Action |
|-----|--------|
| `s` | Next section |
| `S` | Previous section |
| `m` | Toggle to raw mode |
| `z` | Toggle LLM-collapsed hunks |
| `j/k` | Scroll up/down |
| `p/f` | Pass/fail judgment |

## Validation

- [ ] Story mode is default when case has a Story
- [ ] Can navigate section-by-section with `s/S`
- [ ] Current section's hunks shown (others filtered out)
- [ ] Section header shows role and explanation
- [ ] Progress indicator shows reviewed/current/pending sections
- [ ] Can toggle to raw mode with `m`
- [ ] Collapsed hunks render as single line with collapse text
- [ ] `z` toggles LLM-collapsed hunks in current section
- [ ] Pass/fail judgment works in both modes
- [ ] `make validate` passes
