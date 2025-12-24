# Diff Renderer Design

## Problem

The viewer doesn't render diffs properly. It dumps raw `line.Content` with no formatting:
- No file headers (`--- a/file`, `+++ b/file`)
- No hunk headers (`@@ -1,5 +1,10 @@`)
- No line prefixes (`+`/`-`/` `)
- No colors (styling system exists but isn't used)

## Architecture

```
diffview/
├── renderer.go         # Renderer interface + RenderResult
├── lipgloss/
│   ├── theme.go        # existing themes
│   └── renderer.go     # implements diffview.Renderer
├── bubbletea/
│   └── viewer.go       # accepts Renderer (injected)
└── cmd/diffview/
    └── main.go         # wires theme → renderer → viewer
```

Per Ben Johnson's Standard Package Layout:
- Interface in root package (no impl-to-impl coupling)
- `bubbletea/` only imports `diffview` (root)
- `lipgloss/` only imports `diffview` (root)
- Main does the wiring

## Domain Types

Add `diffview/renderer.go`:

```go
package diffview

// Renderer converts a Diff to styled output for display.
type Renderer interface {
    Render(diff *Diff) RenderResult
    SetTheme(theme Theme)
}

// RenderResult contains rendered output and navigation metadata.
type RenderResult struct {
    Content       string
    HunkPositions []int
    FilePositions []int
}
```

## Lipgloss Implementation

Add `lipgloss/renderer.go`:

```go
package lipgloss

import (
    "fmt"
    "strings"

    "github.com/charmbracelet/lipgloss"
    "github.com/fwojciec/diffview"
)

var _ diffview.Renderer = (*Renderer)(nil)

type Renderer struct {
    styles     diffview.Styles
    fileHeader lipgloss.Style
    hunkHeader lipgloss.Style
    added      lipgloss.Style
    deleted    lipgloss.Style
    context    lipgloss.Style
}

func NewRenderer(theme diffview.Theme) *Renderer {
    r := &Renderer{}
    r.SetTheme(theme)
    return r
}

func (r *Renderer) SetTheme(theme diffview.Theme) {
    s := theme.Styles()
    r.styles = s
    r.fileHeader = styleFromPair(s.FileHeader)
    r.hunkHeader = styleFromPair(s.HunkHeader)
    r.added = styleFromPair(s.Added)
    r.deleted = styleFromPair(s.Deleted)
    r.context = styleFromPair(s.Context)
}

func styleFromPair(cp diffview.ColorPair) lipgloss.Style {
    s := lipgloss.NewStyle()
    if cp.Foreground != "" {
        s = s.Foreground(lipgloss.Color(cp.Foreground))
    }
    if cp.Background != "" {
        s = s.Background(lipgloss.Color(cp.Background))
    }
    return s
}

func (r *Renderer) Render(diff *diffview.Diff) diffview.RenderResult {
    if diff == nil {
        return diffview.RenderResult{}
    }

    var sb strings.Builder
    var hunkPositions, filePositions []int
    lineNum := 0

    for _, file := range diff.Files {
        if len(file.Hunks) == 0 {
            continue
        }

        // File header
        filePositions = append(filePositions, lineNum)
        sb.WriteString(r.fileHeader.Render("--- " + file.OldPath))
        sb.WriteString("\n")
        lineNum++
        sb.WriteString(r.fileHeader.Render("+++ " + file.NewPath))
        sb.WriteString("\n")
        lineNum++

        for _, hunk := range file.Hunks {
            // Hunk header
            hunkPositions = append(hunkPositions, lineNum)
            header := fmt.Sprintf("@@ -%d,%d +%d,%d @@",
                hunk.OldStart, hunk.OldCount,
                hunk.NewStart, hunk.NewCount)
            if hunk.Section != "" {
                header += " " + hunk.Section
            }
            sb.WriteString(r.hunkHeader.Render(header))
            sb.WriteString("\n")
            lineNum++

            // Lines
            for _, line := range hunk.Lines {
                sb.WriteString(r.renderLine(line))
                sb.WriteString("\n")
                lineNum++
            }
        }
    }

    return diffview.RenderResult{
        Content:       sb.String(),
        HunkPositions: hunkPositions,
        FilePositions: filePositions,
    }
}

func (r *Renderer) renderLine(line diffview.Line) string {
    switch line.Type {
    case diffview.LineAdded:
        return r.added.Render("+" + line.Content)
    case diffview.LineDeleted:
        return r.deleted.Render("-" + line.Content)
    default:
        return r.context.Render(" " + line.Content)
    }
}
```

## Viewer Integration

Update `bubbletea/viewer.go`:

```go
type Viewer struct {
    renderer    diffview.Renderer
    programOpts []tea.ProgramOption
}

func NewViewer(renderer diffview.Renderer, opts ...ViewerOption) *Viewer {
    v := &Viewer{renderer: renderer}
    for _, opt := range opts {
        opt(v)
    }
    return v
}

func (v *Viewer) View(ctx context.Context, diff *diffview.Diff) error {
    result := v.renderer.Render(diff)
    m := NewModel(result)
    // ... rest unchanged
}

func NewModel(result diffview.RenderResult) Model {
    return Model{
        content:       result.Content,
        hunkPositions: result.HunkPositions,
        filePositions: result.FilePositions,
        keymap:        DefaultKeyMap(),
    }
}
```

## Main Wiring

Update `cmd/diffview/main.go`:

```go
// Wire dependencies
theme := lipgloss.DefaultTheme()
renderer := lipgloss.NewRenderer(theme)

app := &App{
    Stdin:  os.Stdin,
    Parser: gitdiff.NewParser(),
    Viewer: bubbletea.NewViewer(renderer),
}
```

## Output Format

```
--- a/path/to/file.go
+++ b/path/to/file.go
@@ -10,5 +10,7 @@ func Example()
 context line
-deleted line
+added line
 context line
```

| Element | Format | Style |
|---------|--------|-------|
| File header | `--- a/old` / `+++ b/new` | Yellow + background |
| Hunk header | `@@ -10,5 +10,7 @@ section` | Blue |
| Context line | ` content` (space prefix) | Gray |
| Added line | `+content` | Green |
| Deleted line | `-content` | Red |

## Key Decisions

1. **Renderer interface in root** - No impl-to-impl coupling between packages
2. **RenderResult struct** - Extensible return type for content + positions
3. **SetTheme() method** - Explicit runtime theme switching
4. **Pre-computed styles** - Updated via SetTheme(), not recomputed each render
5. **Traditional git diff format** - Familiar, handles edge cases naturally

## Files Changed

- `diffview/renderer.go` - new (interface + RenderResult)
- `lipgloss/renderer.go` - new (implementation)
- `bubbletea/viewer.go` - accept Renderer, simplify Model
- `cmd/diffview/main.go` - wire renderer
