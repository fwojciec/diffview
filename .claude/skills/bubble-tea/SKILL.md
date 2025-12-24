---
name: bubble-tea
description: Patterns for building TUI applications with Bubble Tea (charmbracelet/bubbletea). Use when creating terminal UIs, pagers, or interactive CLI tools in Go. Covers Elm architecture, viewport scrolling, keyboard/mouse handling, Lipgloss styling, and golden file testing with teatest.
---

# Bubble Tea Patterns

## Elm Architecture

```go
type Model struct {
    content  string
    viewport viewport.Model
    ready    bool
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        if msg.String() == "q" {
            return m, tea.Quit
        }
    case tea.WindowSizeMsg:
        // Initialize viewport on first size message
        if !m.ready {
            m.viewport = viewport.New(msg.Width, msg.Height)
            m.viewport.SetContent(m.content)
            m.ready = true
        } else {
            m.viewport.Width = msg.Width
            m.viewport.Height = msg.Height
        }
    }
    var cmd tea.Cmd
    m.viewport, cmd = m.viewport.Update(msg)
    return m, cmd
}

func (m Model) View() string {
    if !m.ready {
        return "Loading..."
    }
    return m.viewport.View()
}
```

**Critical**: Wait for `tea.WindowSizeMsg` before initializing viewport - dimensions arrive async.

## Stdin Piping (`git diff | myapp`)

```go
func main() {
    stat, _ := os.Stdin.Stat()
    if stat.Mode()&os.ModeNamedPipe == 0 && stat.Size() == 0 {
        fmt.Println("Usage: git diff | diffview")
        os.Exit(1)
    }

    content, _ := io.ReadAll(os.Stdin)
    m := Model{content: string(content)}

    p := tea.NewProgram(m,
        tea.WithAltScreen(),       // Full-screen, restores on exit
        tea.WithMouseCellMotion(), // Mouse wheel support
    )
    p.Run()
}
```

## Keyboard Handling

**Simple matching:**
```go
case tea.KeyMsg:
    switch msg.String() {
    case "j", "down":
        m.viewport.LineDown(1)
    case "k", "up":
        m.viewport.LineUp(1)
    case "ctrl+d":
        m.viewport.HalfViewDown()
    case "ctrl+u":
        m.viewport.HalfViewUp()
    case "G":
        m.viewport.GotoBottom()
    case "q", "ctrl+c":
        return m, tea.Quit
    }
```

**Multi-key sequences (gg):**
```go
type Model struct {
    pendingKey string
    // ...
}

case tea.KeyMsg:
    if m.pendingKey == "g" && msg.String() == "g" {
        m.viewport.GotoTop()
        m.pendingKey = ""
        return m, nil
    }
    if msg.String() == "g" {
        m.pendingKey = "g"
        return m, nil
    }
    m.pendingKey = ""
```

**Customizable keymaps with bubbles/key:**
```go
import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
    Down key.Binding
    Up   key.Binding
    Quit key.Binding
}

var DefaultKeyMap = KeyMap{
    Down: key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j/↓", "down")),
    Up:   key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k/↑", "up")),
    Quit: key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}

// Usage: key.Matches(msg, m.keymap.Down)
```

## Viewport Built-in Keys

| Key | Action |
|-----|--------|
| `j/↓` | Line down |
| `k/↑` | Line up |
| `d/ctrl+d` | Half page down |
| `u/ctrl+u` | Half page up |
| `f/pgdn/space` | Page down |
| `b/pgup` | Page up |

## Lipgloss Styling

```go
import "github.com/charmbracelet/lipgloss"

// Diff line styles with adaptive colors
addedStyle := lipgloss.NewStyle().
    Foreground(lipgloss.AdaptiveColor{Light: "28", Dark: "34"}).
    Background(lipgloss.AdaptiveColor{Light: "194", Dark: "22"})

removedStyle := lipgloss.NewStyle().
    Foreground(lipgloss.AdaptiveColor{Light: "160", Dark: "203"}).
    Background(lipgloss.AdaptiveColor{Light: "224", Dark: "52"})

// Line numbers
lineNumStyle := lipgloss.NewStyle().
    Foreground(lipgloss.Color("245")).
    Width(6).
    Align(lipgloss.Right)

// Side-by-side layout
joined := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)

// Measure ANSI-aware width
width := lipgloss.Width(styledString)
```

**Layering styles** (syntax + diff): Render inner style first, wrap with outer.

## Header/Footer Pattern

```go
func (m Model) View() string {
    return fmt.Sprintf("%s\n%s\n%s",
        m.headerView(),
        m.viewport.View(),
        m.footerView(),
    )
}

// Calculate viewport height accounting for margins
case tea.WindowSizeMsg:
    headerHeight := lipgloss.Height(m.headerView())
    footerHeight := lipgloss.Height(m.footerView())
    m.viewport.Height = msg.Height - headerHeight - footerHeight
```

## Testing with Golden Files

**Package**: `github.com/charmbracelet/x/exp/teatest`

```go
import (
    "testing"
    "io"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/x/exp/teatest"
)

func TestView(t *testing.T) {
    m := NewModel(testContent)
    tm := teatest.NewTestModel(t, m,
        teatest.WithInitialTermSize(80, 24),
    )

    // Send key presses
    tm.Send(tea.KeyMsg{Type: tea.KeyDown})
    tm.Send(tea.KeyMsg{Runes: []rune{'j'}})

    // Wait for specific output
    teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
        return bytes.Contains(out, []byte("expected"))
    })

    tm.Send(tea.KeyMsg{Runes: []rune{'q'}})

    // Compare against golden file (testdata/TestView.golden)
    out, _ := io.ReadAll(tm.FinalOutput(t))
    teatest.RequireEqualOutput(t, out)
}
```

**Workflow**:
1. `go test -update` → creates/updates `testdata/TestName.golden`
2. Read `.golden` files to see terminal output (includes ANSI codes)
3. Tests fail with unified diff when output changes

## Gotchas

1. **Always return model** from Update, even if modified via receiver
2. **View() must be pure** - no side effects
3. **Commands run async** - don't assume order
4. **No line wrapping** - viewport truncates long lines
5. **Pass all messages to viewport** for built-in scrolling to work
6. **Never use `len(string)` for display width** - use `lipgloss.Width()` instead:
   ```go
   // WRONG: len() counts bytes, not display width
   padding := strings.Repeat(" ", maxWidth - len(line))

   // CORRECT: lipgloss.Width() handles Unicode properly
   padding := strings.Repeat(" ", maxWidth - lipgloss.Width(line))
   ```
   - `len("日本語")` = 9 bytes, but displays as 6 cells (CJK are double-width)
   - `len("emoji 😀")` = 10 bytes, but displays as 8 cells
   - `lipgloss.Width()` uses go-runewidth internally for correct display width
