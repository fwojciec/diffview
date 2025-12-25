# Syntax highlighting for terminal diff viewers in Go

**Chroma is the clear choice for Go terminal syntax highlighting, and it composes well with Lipgloss—but you must layer them correctly.** The key insight from production implementations like delta is a two-pass architecture: first extract syntax highlighting foreground colors token-by-token, then apply diff background colors separately, and finally merge them at render time using Lipgloss. This approach keeps your diff backgrounds readable while preserving syntax coloring.

## Chroma provides everything you need for terminal output

Chroma offers **five terminal formatters** with different color depths: `terminal` (8-color), `terminal16`, `terminal256`, and `terminal16m` (true color/24-bit). The 256-color formatter generates sequences like `\033[38;5;Xm`, while true color outputs `\033[38;2;R;G;Bm` for precise RGB values.

The crucial API pattern for your use case is **token-by-token iteration** rather than quick.Highlight():

```go
lexer := lexers.Get("go")
lexer = chroma.Coalesce(lexer) // Reduces token count
style := styles.Get("monokai")

iterator, _ := lexer.Tokenise(nil, sourceCode)
for token := iterator(); token != chroma.EOF; token = iterator() {
    entry := style.Get(token.Type)
    // entry.Colour gives you the foreground color
    // entry.Bold, entry.Italic give you style attributes
    // token.Value is the actual text
}
```

For language detection from diff headers like `+++ b/src/foo.go`, Chroma provides `lexers.Match(filename)` which handles extension-based detection. Parse the diff header to extract the filename:

```go
func lexerFromDiffPath(diffPath string) chroma.Lexer {
    filename := strings.TrimPrefix(diffPath[4:], "b/")
    filename = filepath.Base(filename)
    if lexer := lexers.Match(filename); lexer != nil {
        return lexer
    }
    return lexers.Fallback
}
```

Content-based detection via `lexers.Analyse()` works as a fallback but has limited support—only some lexers implement the `Analyser` interface.

## Lipgloss preserves existing ANSI but has boundary issues

Lipgloss **does not strip or modify** existing ANSI escape sequences in input strings. When you call `style.Render()` on pre-styled content, the existing codes are preserved and Lipgloss wraps everything with its own sequences. However, **issues occur at line boundaries**—when Lipgloss truncates or pads styled content, reset sequences may be lost, causing colors to "bleed" into subsequent content.

The `lipgloss.Width()` function correctly ignores ANSI codes when measuring display width, which you'll need for proper layout calculations. For more complex ANSI manipulation, use the `github.com/charmbracelet/x/ansi` package which provides `ansi.Strip()`, `ansi.StringWidth()`, and `ansi.Truncate()`.

**The right order of operations is: apply syntax highlighting first, then wrap with Lipgloss for background colors.** But the cleanest approach avoids nesting ANSI entirely—extract Chroma's style information and generate your own Lipgloss styles:

```go
type StyledSegment struct {
    Text       string
    Foreground lipgloss.Color
    Background lipgloss.Color
    Bold       bool
}

func getSegments(code, language string, diffLineType LineType) []StyledSegment {
    lexer := lexers.Get(language)
    style := styles.Get("monokai")
    iterator, _ := lexer.Tokenise(nil, code)
    
    bgColor := getDiffBackground(diffLineType) // Your diff colors
    var segments []StyledSegment
    
    for token := iterator(); token != chroma.EOF; token = iterator() {
        entry := style.Get(token.Type)
        segments = append(segments, StyledSegment{
            Text:       token.Value,
            Foreground: lipgloss.Color(entry.Colour.String()),
            Background: bgColor,
            Bold:       entry.Bold == chroma.Yes,
        })
    }
    return segments
}

func renderSegments(segments []StyledSegment) string {
    var result strings.Builder
    for _, seg := range segments {
        s := lipgloss.NewStyle().
            Foreground(seg.Foreground).
            Background(seg.Background)
        if seg.Bold {
            s = s.Bold(true)
        }
        result.WriteString(s.Render(seg.Text))
    }
    return result.String()
}
```

## Terminal color detection requires coordination between libraries

**Chroma does not auto-detect terminal capabilities**—you must explicitly choose the formatter. Lipgloss and termenv do detect capabilities via environment variables: `COLORTERM=truecolor` triggers 24-bit mode, `TERM=xterm-256color` triggers 256-color mode.

The safe pattern is to detect once and match both libraries:

```go
func getFormatterName() string {
    if ct := os.Getenv("COLORTERM"); ct == "truecolor" || ct == "24bit" {
        return "terminal16m"
    }
    if strings.Contains(os.Getenv("TERM"), "256") {
        return "terminal256"
    }
    return "terminal16"
}

// In initialization
lipgloss.SetColorProfile(termenv.TrueColor) // Match to your Chroma formatter
```

For graceful degradation, Lipgloss provides `CompleteColor` which specifies fallbacks explicitly:

```go
lipgloss.CompleteColor{
    TrueColor: "#0000FF",
    ANSI256:   "21",
    ANSI:      "4",
}
```

## Creating overlay-friendly themes for diff backgrounds

The core challenge is that syntax colors must remain readable on green (added), red (deleted), and neutral backgrounds. Delta's solution: **use very dark, desaturated backgrounds** that don't compete with syntax foreground colors:

```go
var (
    MinusBackground = lipgloss.Color("#3f0001") // Very dark red
    PlusBackground  = lipgloss.Color("#004000") // Very dark green
    ContextBg       = lipgloss.NoColor{}        // Terminal default
)
```

For Chroma themes, create a custom style that omits background colors entirely so your diff backgrounds show through:

```go
overlayStyle := chroma.MustNewStyle("overlay", chroma.StyleEntries{
    chroma.Background:  "noinherit", // Critical: no background
    chroma.Keyword:     "bold #ff79c6",
    chroma.String:      "#f1fa8c",
    chroma.Comment:     "italic #6272a4",
    chroma.Number:      "#bd93f9",
    chroma.NameFunction: "#50fa7b",
})
```

For adapting to terminal light/dark modes, Lipgloss provides `AdaptiveColor` which automatically selects based on detected background. You'd combine this with your overlay logic to select appropriate foreground intensities.

## Performance realities with diff hunks

Chroma uses regex-based lexers that process text linearly—**no full file context is required**, but partial files can cause issues. If a diff hunk starts inside a multi-line string or comment, tokenization will be incorrect. There's no pause/resume API.

Practical mitigation strategies:

1. **File-level caching**: Highlight the entire original and modified files once, cache results by content hash, then slice into the cached tokens for each hunk
2. **Lazy highlighting**: Only process lines currently in the viewport
3. **Background processing with timeout**: Use goroutines with fallback to plain text if highlighting takes too long

```go
type HighlightCache struct {
    sync.RWMutex
    cache map[string][]chroma.Token // key: filepath + content hash
}

// For async highlighting with fallback
select {
case tokens := <-highlightResult:
    return renderTokens(tokens)
case <-time.After(100 * time.Millisecond):
    return plainText // Fallback
}
```

The `chroma.Coalesce(lexer)` wrapper merges adjacent identical tokens, reducing memory and iteration overhead—always use it.

## Real-world implementation patterns from production tools

**Lazygit** deliberately avoids built-in syntax highlighting because line-by-line staging requires exact correspondence between displayed and actual diff lines. It delegates to external pagers like delta. This is worth considering if you need staging functionality.

**Glamour** (used by Glow for markdown rendering) demonstrates clean Chroma integration:

```go
err := quick.Highlight(writer, code, language, "terminal256", themeName)
```

But for your overlay case, Glamour's approach doesn't help since it doesn't handle custom backgrounds.

**Delta** (Rust) provides the architectural blueprint you want—a two-pass style composition:

1. **Pass 1**: Compute syntax styles → array of (foreground color, text segment) pairs
2. **Pass 2**: Compute diff styles → array of (background color, emphasis level) pairs  
3. **Merge**: Output where background comes from diff, foreground from syntax

Delta's style syntax makes this explicit:
```
minus-style = syntax "#3f0001"  # "syntax" = use syntax fg, "#3f0001" = bg
plus-style = syntax "#004000"
```

## Handling diff-specific preprocessing

Strip the diff prefix (`+`, `-`, or space) before highlighting, then re-add it with appropriate styling:

```go
func highlightDiffLine(line string, lineType LineType, lexer chroma.Lexer) string {
    prefix := line[0:1]  // "+", "-", or " "
    content := line[1:]  // Actual code
    
    // Highlight content without prefix
    tokens := tokenize(content, lexer)
    
    // Build output with diff styling
    bg := getDiffBackground(lineType)
    prefixStyle := lipgloss.NewStyle().
        Background(bg).
        Foreground(getPrefixColor(lineType))
    
    var result strings.Builder
    result.WriteString(prefixStyle.Render(prefix))
    
    for _, tok := range tokens {
        style := lipgloss.NewStyle().
            Background(bg).
            Foreground(lipgloss.Color(tok.Color))
        result.WriteString(style.Render(tok.Text))
    }
    return result.String()
}
```

For ANSI escape sequences, foreground and background codes **do compose independently**—you can set them in separate sequences and they stack. Each styled segment should end with a reset (`\x1b[0m`) to prevent bleeding.

## Testing syntax-highlighted output

Use golden file/snapshot testing with forced color profiles for reproducible CI results:

```go
func TestDiffHighlight(t *testing.T) {
    // Force consistent color output in CI
    lipgloss.SetColorProfile(termenv.TrueColor)
    
    result := highlightDiff(testInput)
    
    g := goldie.New(t, goldie.WithFixtureDir("testdata/golden"))
    g.Assert(t, "expected_output", []byte(result))
}
```

For readable test failures, escape ANSI sequences: `strings.ReplaceAll(s, "\x1b", "ESC")`.

## Recommended implementation architecture

```go
// Core types
type Token struct {
    Text       string
    Foreground lipgloss.Color
    Bold       bool
}

type DiffLine struct {
    Type     LineType // Added, Removed, Context
    Prefix   string   // "+", "-", " "
    Tokens   []Token
    LineNums struct { Old, New int }
}

// Rendering pipeline
func RenderDiff(hunk Hunk, language string) string {
    lexer := lexers.Match(hunk.Filename)
    style := getOverlayStyle()
    
    var output strings.Builder
    for _, line := range hunk.Lines {
        bg := getDiffBackground(line.Type)
        
        // Tokenize content (without prefix)
        tokens := tokenize(line.Content, lexer, style)
        
        // Render with layered styling
        output.WriteString(renderLine(line.Prefix, tokens, bg))
        output.WriteString("\n")
    }
    return output.String()
}
```

This architecture separates concerns cleanly: Chroma handles tokenization, you control the styling logic, and Lipgloss handles the actual ANSI generation with proper color profile awareness.
