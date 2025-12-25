# Unified Theme System Design

**Date**: 2025-12-25
**Status**: Approved
**Epic**: To be created

## Problem

Colors are defined in multiple places with inconsistencies:
- `lipgloss/theme.go`: DarkTheme/LightTheme with very dark backgrounds
- `bubbletea/viewer.go`: defaultStyles() with different subtle backgrounds
- `bubbletea/viewer.go`: statusBarView() with hardcoded colors
- `chroma/tokenizer.go`: One Dark palette, disconnected from themes

Adding syntax highlighting would compound this problem by introducing another color palette that doesn't harmonize with diff colors.

## Solution

A unified `Palette` type that serves as the single source of truth for all colors. Everything derives from the 18 semantic colors in the palette.

## Architecture

### Layer Separation (Delta Pattern)

Following delta's proven approach:
1. **Syntax highlighting** provides foreground colors
2. **Diff styling** provides background colors
3. **Merge at render time** - no runtime color adjustment

### Domain Types (Root Package)

```go
// Color represents a hex color value like "#RRGGBB"
type Color string

// Palette defines the semantic color vocabulary for the application.
type Palette struct {
    // Base colors (2)
    Background Color
    Foreground Color

    // Diff semantics (4)
    Added    Color
    Deleted  Color
    Modified Color
    Context  Color

    // Syntax highlighting (9)
    Keyword     Color
    String      Color
    Number      Color
    Comment     Color
    Operator    Color
    Function    Color
    Type        Color
    Constant    Color
    Punctuation Color

    // UI chrome (3)
    UIBackground Color
    UIForeground Color
    UIAccent     Color
}

// Theme provides the complete visual configuration.
type Theme interface {
    Palette() Palette
    Styles() Styles  // Generated from Palette
}
```

### Theme Implementation (`lipgloss/` Package)

```go
type theme struct {
    palette diffview.Palette
    styles  diffview.Styles   // Generated from palette
    syntax  *chroma.Style     // Generated from palette
}

func DefaultTheme() diffview.Theme {
    p := diffview.Palette{
        Background: "#1e1e2e",
        Foreground: "#cdd6f4",
        // ... 18 colors total
    }
    return newTheme(p)
}

func TestTheme() diffview.Theme {
    // Stable, predictable colors for testing
}

func newTheme(p diffview.Palette) *theme {
    return &theme{
        palette: p,
        styles:  stylesFromPalette(p),
        syntax:  syntaxFromPalette(p),
    }
}
```

### Chroma Integration (`chroma/` Package)

Root package can't import chroma, so chroma provides the generator:

```go
// chroma/style.go
func StyleFromPalette(p diffview.Palette) *chroma.Style {
    return chroma.MustNewStyle("diffview",
        chroma.StyleEntries{
            chroma.Background:   "bg:" + string(p.Background),
            chroma.Text:         string(p.Foreground),
            chroma.Keyword:      "bold " + string(p.Keyword),
            chroma.String:       string(p.String),
            chroma.Number:       string(p.Number),
            chroma.Comment:      "italic " + string(p.Comment),
            chroma.Operator:     string(p.Operator),
            chroma.NameFunction: string(p.Function),
            chroma.KeywordType:  string(p.Type),
            chroma.NameConstant: string(p.Constant),
            chroma.Punctuation:  string(p.Punctuation),
        },
    )
}
```

### Viewer Integration (`bubbletea/` Package)

```go
type Viewer struct {
    theme       diffview.Theme
    syntaxStyle *chroma.Style  // Cached
}

func WithTheme(t diffview.Theme) Option {
    return func(v *Viewer) {
        v.theme = t
        v.syntaxStyle = chroma.StyleFromPalette(t.Palette())
    }
}
```

## Testing Strategy

### Principles

1. **Behavior tests use `TestTheme()`** - decouples from aesthetic changes
2. **Always use explicit renderer** - no terminal auto-detection
3. **Check content, not colors** for most tests
4. **Color tests verify ANSI presence** - not specific RGB values

### Test Theme

```go
func TestTheme() diffview.Theme {
    return newTheme(diffview.Palette{
        Background:   "#000000",
        Foreground:   "#ffffff",
        Added:        "#00ff00",  // Pure green
        Deleted:      "#ff0000",  // Pure red
        // ... predictable, stable values
    })
}
```

### Test Patterns

**Behavior tests**:
```go
m := NewModel(diff, WithTheme(lipgloss.TestTheme()))
// Assert on content, not colors
```

**Color integration tests**:
```go
m := NewModel(diff,
    WithTheme(lipgloss.TestTheme()),
    WithRenderer(trueColorRenderer()),
)
// Assert: bytes.Contains(out, []byte("48;2;"))
```

## Design Decisions

1. **No light/dark mode switching** - Users pick a theme that matches their terminal
2. **Colors only in Palette** - Bold/italic handled in implementation layer
3. **Minimal UI chrome (3 colors)** - Derives from base, accent for emphasis
4. **Ben Johnson pattern** - Palette in root, implementations in dependency packages

## Migration Path

1. Add `Palette` type to root package, update `Theme` interface
2. Implement palette-based theme in `lipgloss/`
3. Add `StyleFromPalette` to `chroma/`
4. Update viewer to use `WithTheme()` option
5. Wire in `cmd/diffview/main.go`
6. Remove all hardcoded colors

## References

- [Theme Research](../theme-research.md) - Deep research on delta, Neovim, Helix patterns
- [Bubble Tea Skill](../../.claude/skills/bubble-tea/SKILL.md) - Testing patterns
