# Enhanced Intro Slide Design

## Goal

Help reviewers build mental models and intuition for different types of code changes by adding classification context and schematic diagrams to the intro slide.

## Current State

The intro slide shows only:
- Summary (one sentence)
- Section list (numbered titles)
- Navigation hint

## Proposed Layout

```
┌─────────────────────────────────────────────────────────┐
│  [feature] Integrate PR metadata into classification    │  ← Summary with change type
│  pipeline to provide author intent for better narratives.│
├─────────────────────────────────────────────────────────┤
│  Core-periphery: A central change with supporting       │  ← Pattern explanation
│  updates radiating outward.                             │
│                                                         │
│          ╭────────────╮                                 │
│          │ supporting │                                 │  ← Diagram
│          ╰─────┬──────╯                                 │
│                │                                        │
│  ╭──────╮    ╭─┴───╮    ╭─────────╮                     │
│  │ test │────│core │────│ cleanup │                     │
│  ╰──────╯    ╰─────╯    ╰─────────╯                     │
├─────────────────────────────────────────────────────────┤
│  Sections:                                              │
│    1. [core] Classification Input Schema                │  ← Section list with roles
│    2. [supporting] Metadata Formatting and AI Guidance  │
│    3. [test] Validation Tests                           │
│    4. [cleanup] Project Status Updates                  │
├─────────────────────────────────────────────────────────┤
│  [s] next section                                       │  ← Navigation hint
└─────────────────────────────────────────────────────────┘
```

## Components

### 1. Summary with Change Type Prefix

Format: `[change_type] summary text`

Change types: `feature`, `bugfix`, `refactor`, `chore`, `docs`

### 2. Pattern Explanation

One-sentence description of the narrative pattern:

| Narrative | Explanation |
|-----------|-------------|
| cause-effect | "A problem leading to its fix and verification." |
| core-periphery | "A central change with supporting updates radiating outward." |
| entry-implementation | "An interface contract followed by its implementation." |
| before-after | "A transformation from an old pattern to a new one." |

### 3. Narrative Diagrams

#### Cause-Effect (linear flow)
```
╭─────────╮     ╭─────╮     ╭──────╮
│ problem │ ──→ │ fix │ ──→ │ test │
╰─────────╯     ╰─────╯     ╰──────╯
```
Note: `problem` section optional; if missing, start with `fix`.

#### Entry-Implementation (two towers)
```
╭───────────╮         ╭──────╮
│ interface │  ─────→ │ core │
╰───────────╯         ╰──────╯
```

#### Before-After (transformation)
```
╭─────────╮         ╭───────╮
│ cleanup │  ═════> │ core  │
╰─────────╯         ╰───────╯
   before             after
```

#### Core-Periphery (hub and spokes)
```
       ╭────────────╮
       │ supporting │
       ╰──────┬─────╯
              │
╭──────╮    ╭─┴───╮    ╭─────────╮
│ test │────│core │────│ cleanup │
╰──────╯    ╰─────╯    ╰─────────╯
```

Diagrams only show roles that exist in the classification.

### 4. Section List with Roles

Format: `N. [role] Title`

Maps diagram nodes to specific sections in this diff.

## Implementation

### Files

| File | Change |
|------|--------|
| `bubbletea/story.go` | Update `renderIntro()` to call new diagram renderer |
| `bubbletea/intro.go` (new) | Diagram rendering functions, pattern explanations map |

### Dependencies

- **Lipgloss** (existing) - linear, two-column, before-after diagrams
- **ntcharts Canvas** (new, optional) - hub-and-spoke diagram; can defer with simplified Lipgloss fallback

### Testing

- Golden file tests for each diagram type
- Test with varying section counts
- Test fallback when narrative is empty/unknown

## Decisions Made

- **No separate header line** - change type prefixes summary instead
- **Roles in diagram, titles in list** - teaches pattern vocabulary while grounding in specifics
- **Descriptive explanations only** - no prescriptive "how to review" instructions
- **No evolution field** - section structure already embodies the narrative
- **Graceful fallback** - unknown/empty narrative degrades to current behavior
