# Evalreview TUI Design

Design for a terminal UI to review LLM-generated diff stories and record pass/fail judgments.

## Overview

**Purpose**: Evaluate LLM-generated story analyses by displaying the original diff alongside the generated story, allowing human reviewers to mark each as pass/fail with optional critique.

**Workflow**: Sequential review - go through cases one by one, judging each before moving on.

## Data Model

### Input Format (`eval/cases/*.jsonl`)

Each line is a self-contained case with diff hunks and the LLM-generated story:

```go
type EvalCase struct {
    Commit string                   `json:"commit"`
    Hunks  []diffview.AnnotatedHunk `json:"hunks"`
    Story  diffview.StoryAnalysis   `json:"story"`
}
```

### Output Format (`eval/cases/*-judgments.jsonl`)

Separate file for judgments, keeping source data immutable:

```go
type Judgment struct {
    Commit   string    `json:"commit"`     // Links to EvalCase
    Index    int       `json:"index"`      // Position in input file
    Pass     bool      `json:"pass"`
    Critique string    `json:"critique"`   // Free-text, empty if pass
    JudgedAt time.Time `json:"judged_at"`
}
```

Naming convention: input `foo.jsonl` → judgments `foo-judgments.jsonl` (same directory).

### Loading Behavior

- On startup, load all cases from input file
- If judgments file exists, load and merge (match by commit hash)
- Cases with existing judgments shown as reviewed but editable
- Status bar shows "5/23 reviewed" progress

## UI Layout

Vertical stack, responsive to terminal height:

```
┌─────────────────────────────────────────────┐
│ DIFF [active]           file 2/5  hunk 3/8  │  ← Panel header
│  @@ -10,4 +10,6 @@                          │
│   func foo() {                              │
│ -    old code                               │
│ +    new code                               │
│                                             │
├─────────────────────────────────────────────┤
│ STORY                                       │  ← Panel header
│ [bugfix] Fix null pointer in parser         │
│                                             │
│ Core changes:                               │
│ • parser.go:h0 - Added nil check before...  │
│                                             │
├─────────────────────────────────────────────┤
│ ○ Pass  ○ Fail    Critique: [not set]       │  ← Judgment bar
├─────────────────────────────────────────────┤
│ case 5/23 │ [d]iff [s]tory [p]ass [f]ail    │  ← Status bar
│           │ [c]ritique [j/k]nav [q]uit      │
└─────────────────────────────────────────────┘
```

### Height Distribution (40-line terminal)

- Diff panel: 50% (20 lines)
- Story panel: 35% (14 lines)
- Judgment bar: 1 line
- Status bar: 2 lines
- Borders/headers: 3 lines

## Modes

### Review Mode (default)

Navigate cases, scroll panels, record judgments. Active panel indicated by `[active]` tag in header.

### Critique Mode

Text input for critique. Enter with `c`, exit with `Esc`.

## Keyboard Controls

### Review Mode

| Key | Action |
|-----|--------|
| `j` / `k` | Next / previous case |
| `d` | Set diff as active panel |
| `s` | Set story as active panel |
| `ctrl+d` / `ctrl+u` | Scroll active panel half-page |
| `g g` | Scroll active panel to top |
| `G` | Scroll active panel to bottom |
| `n` / `N` | Next / previous hunk (diff active) |
| `p` | Mark current case as pass |
| `f` | Mark current case as fail |
| `c` | Enter critique mode |
| `q` | Quit (auto-saves) |

### Critique Mode

| Key | Action |
|-----|--------|
| Normal typing | Edit critique text |
| `Esc` | Exit critique mode |
| `Enter` | Newline in critique |

### Auto-save Behavior

- Judgments saved immediately on `p` or `f`
- Critique saved when exiting critique mode
- No explicit save command needed

## Architecture

Following Ben Johnson Standard Package Layout.

### Root Package (`diffview/`)

New domain types:

```go
// evalreview.go
type EvalCase struct {
    Commit string
    Hunks  []AnnotatedHunk
    Story  StoryAnalysis
}

type Judgment struct {
    Commit   string
    Index    int
    Pass     bool
    Critique string
    JudgedAt time.Time
}

type EvalCaseLoader interface {
    Load(path string) ([]EvalCase, error)
}

type JudgmentStore interface {
    Load(path string) ([]Judgment, error)
    Save(path string, judgments []Judgment) error
}
```

### File Structure

```
diffview/
├── evalreview.go          # Domain types and interfaces
├── jsonl/
│   ├── loader.go          # EvalCaseLoader implementation
│   └── store.go           # JudgmentStore implementation
├── bubbletea/
│   ├── viewer.go          # Existing diff viewer (unchanged)
│   ├── eval.go            # New EvalModel
│   ├── eval_test.go       # Tests for eval TUI
│   └── eval_keymap.go     # Keybindings for eval TUI
├── mock/
│   └── eval.go            # Mocks for new interfaces
└── cmd/evalreview/
    └── main.go            # CLI entry point
```

### EvalModel Structure

```go
type EvalModel struct {
    // Data
    cases        []diffview.EvalCase
    judgments    map[string]*diffview.Judgment
    currentIndex int

    // UI Components
    diffViewport  viewport.Model
    storyViewport viewport.Model
    critiqueInput textarea.Model

    // State
    activePanel   Panel  // PanelDiff or PanelStory
    mode          Mode   // ModeReview or ModeCritique

    // Rendering
    theme         diffview.Theme
    width, height int

    // Persistence
    store         diffview.JudgmentStore
    outputPath    string

    // Keybindings
    keymap        EvalKeyMap
}
```

## Testing Strategy

Using `teatest` for golden file testing, following existing patterns.

### Test Coverage

1. Navigation (j/k, edge cases at first/last)
2. Panel switching (d/s, active indicator)
3. Judgment recording (p/f, state updates)
4. Mode switching (c for critique, Esc to return)
5. Persistence (judgments saved on action)

### JSONL Tests

- Load valid JSONL
- Handle malformed lines gracefully
- Round-trip: save then load preserves data

## MVP Scope

### In Scope

- Load enriched JSONL (cases + stories)
- Vertical 3-panel layout
- Sequential navigation (j/k)
- Panel scrolling with active toggle (d/s)
- Pass/fail judgment (p/f)
- Critique text input
- Auto-save to separate file
- Progress indicator

### Out of Scope (future)

- Split-pane resizing
- Filtering by judgment status
- Search within cases
- Undo last judgment
- Statistics dashboard
- Jump to case by ID

## Validation Criteria

- [ ] `evalreview eval/cases/test.jsonl` launches TUI
- [ ] Can navigate between 3+ cases with j/k
- [ ] Can scroll diff and story independently
- [ ] Can record pass/fail judgment
- [ ] Can enter and save critique text
- [ ] Judgments persist to `*-judgments.jsonl`
- [ ] Re-launching loads previous judgments
- [ ] `make validate` passes
