# AI agents can control TUIs through PTY + virtual terminal architecture

The most practical approach for an AI-controllable Go/Bubble Tea diff review tool combines **PTY-based process control with a gRPC sidecar server** that exposes state and accepts commands. Unlike browser automation's mature DevTools Protocol, terminal automation remains fragmented—but Bubble Tea's architecture offers unique advantages that can be exploited for AI interaction without screen scraping.

Three viable patterns emerge: (1) the **PTY + virtual terminal** approach used by Claude Plays Pokemon, where you spawn the TUI in a pseudo-terminal and parse its output through a headless terminal emulator; (2) the **gRPC sidecar** pattern, where your Bubble Tea app runs both a TUI and an API server, using `tea.Program.Send()` to inject external commands; (3) **tmux control mode** as an intermediary layer, providing screen capture via `capture-pane` and input injection via `send-keys`. For a Go/Bubble Tea tool, pattern #2 offers the best reliability and lowest latency, while pattern #1 provides fallback compatibility with any TUI.

---

## Terminal emulators vary wildly in automation support

**Kitty offers the most complete remote control API** among modern terminals. Its JSON-over-DCS protocol communicates via Unix sockets, supporting `get-text` to read screen contents, `send-text` for input injection, and full window/tab management. Enable it with `allow_remote_control yes` and `listen_on unix:/tmp/mykitty` in kitty.conf, then control externally:

```bash
# Read screen contents
kitty @ get-text --match "title:MyApp"
# Send input to specific window
kitty @ send-text --match "pid:12345" "hello world"
# List all windows as JSON
kitty @ ls
```

**WezTerm provides Lua scripting** with `action_callback` enabling arbitrary code execution on key events, plus a CLI (`wezterm cli send-text`, `list`, `split-pane`) for external control. It lacks Kitty's `get-text` equivalent—you cannot read screen contents directly.

**Ghostty is actively developing automation capabilities** but currently offers only basic `ghostty +new-window` IPC. The team is debating between Kitty-style control sequences, platform-native IPC (AppleScript/D-Bus), or cross-platform Unix domain sockets. **Alacritty intentionally omits automation features**—its minimalist philosophy delegates multiplexing to tmux/screen.

| Feature | Kitty | WezTerm | Ghostty | Alacritty |
|---------|-------|---------|---------|-----------|
| Read screen | ✅ `get-text` | ❌ | ❌ | ❌ |
| Send input | ✅ `send-text` | ✅ CLI | ❌ | ❌ |
| IPC protocol | Unix socket + DCS | Unix socket | Planned | None |
| Scripting | Python kittens | Lua callbacks | None | None |

The **OSC 52 escape sequence** works across most terminals for clipboard access—even over SSH—making it useful for data extraction: `printf "\033]52;c;$(echo -n 'data' | base64)\a"`.

---

## Claude Plays Pokemon reveals the AI-TUI feedback loop pattern

The Claude Plays Pokemon project demonstrates the canonical architecture for AI control of visual applications. Its **four-step continuous cycle** provides a template:

1. **State capture**: PyBoy emulator provides direct RAM access (`pyboy.memory[0xC345]`) plus screenshots via `pyboy.screen.image`. A RAM overlay system parses memory to extract player coordinates, team HP, inventory—structured data that eliminates vision model uncertainty.

2. **Prompt composition**: System assembles tool definitions, knowledge base context, summarized history, and current state into the prompt.

3. **Tool execution**: Claude returns tool calls (`use_emulator` for button presses, `navigator` for pathfinding, `update_knowledge_base` for memory), which the system executes.

4. **Memory management**: Short-term conversation history (30 messages), long-term knowledge base dictionary, automatic summarization when context exceeds limits.

**Critical insight**: The project uses **turn-based mode** where the emulator only advances when AI sends inputs—not real-time. This eliminates timing races and simplifies the feedback loop. For a diff review tool, this maps naturally: the TUI waits for AI commands rather than expecting real-time interaction.

Key repositories implementing this pattern:
- `davidhershey/ClaudePlaysPokemonStarter` — Official Anthropic starter
- `cicero225/llm_pokemon_scaffold` — Multi-model support with `memory_reader.py` for RAM extraction
- Desktop automation: `simular-ai/Agent-S` (72.6% on OSWorld benchmark), `HuggingFace ScreenEnv` with Docker isolation

**LLMs struggle with raw ASCII art** due to 1D tokenization losing 2D spatial relationships. For terminal output, structured state extraction (coordinates, selections, text content) dramatically outperforms asking the model to interpret rendered screens.

---

## Bubble Tea enables native AI integration through Program.Send()

Bubble Tea's Elm Architecture provides natural hooks for AI control without screen scraping. The **`tea.Program.Send(msg)`** method injects messages into the event loop from external goroutines—the key to AI integration:

```go
// Create program with custom I/O for headless operation
var outputBuf bytes.Buffer
p := tea.NewProgram(model,
    tea.WithInput(nil),           // Disable stdin
    tea.WithOutput(&outputBuf),   // Capture output
    tea.WithoutRenderer(),        // Disable TUI rendering
)

go p.Run()

// AI agent injects commands
p.Send(tea.KeyMsg{Type: tea.KeyDown})
p.Send(customAICommand{action: "select_diff", index: 3})
```

**The teatest package** demonstrates proven patterns: `NewTestModel()` wraps programs with custom I/O, `Send()` injects messages, `WaitFor()` polls for conditions, and `FinalModel()` returns actual model state for assertions. This architecture adapts directly to AI interaction.

For **dual-interface operation** (TUI + API simultaneously), use Wish middleware for SSH-accessible Bubble Tea apps, or run a gRPC server alongside the TUI:

```go
type DiffReviewApp struct {
    model     *DiffModel
    program   *tea.Program
}

// gRPC service exposes state and accepts commands
func (s *DiffReviewServer) GetState(ctx context.Context, req *Empty) (*StateResponse, error) {
    return &StateResponse{
        CurrentFile:    s.app.model.CurrentFile,
        SelectedHunks:  s.app.model.SelectedHunks,
        CursorPosition: s.app.model.Cursor,
    }, nil
}

func (s *DiffReviewServer) ExecuteAction(ctx context.Context, req *ActionRequest) (*Response, error) {
    s.app.program.Send(aiCommand{action: req.Action, params: req.Params})
    // Wait for state update, return new state
    return &Response{Success: true, NewState: s.app.model.ToProto()}, nil
}
```

**State synchronization** follows the single-source-of-truth pattern: after any mutation, reload from the authoritative data source and update UI state in one place.

---

## Three approaches to terminal state extraction

### PTY + Virtual Terminal Emulator (Most Universal)

Spawn the TUI in a pseudo-terminal using `creack/pty`, then parse output through a headless terminal emulator:

```go
import (
    "github.com/creack/pty"
    "github.com/hinshun/vt10x"
)

cmd := exec.Command("./my-tui-app")
ptmx, _ := pty.StartWithSize(cmd, &pty.Winsize{Rows: 24, Cols: 80})

// Virtual terminal maintains screen state
vt := vt10x.New()
go func() {
    io.Copy(vt, ptmx)  // Feed PTY output to virtual terminal
}()

// Extract structured state
screen := vt.String()           // Full screen as text
cell := vt.Cell(10, 5)          // Individual cell (char + attributes)
cursor := vt.Cursor()           // Cursor position
```

**Go virtual terminal libraries**:
- `hinshun/vt10x` — VT100 emulation with `Cell(x,y)` access, cursor state, `String()` dump
- `taigrr/bubbleterm` — Headless emulator with Bubble Tea integration
- `tcell-term` — Terminal emulator rendering to tcell surface

### tmux Control Mode (Intermediary Layer)

tmux's `-CC` flag provides a text-based protocol for programmatic control:

```bash
tmux -CC new-session -d -s ai_session "./my-tui-app"
# Capture screen contents
tmux capture-pane -t ai_session -p -e  # -e preserves ANSI colors
# Send input
tmux send-keys -t ai_session "j"       # Down arrow
tmux send-keys -t ai_session Enter
```

Control mode wraps output in `%begin`/`%end` guards and sends notifications (`%output`, `%layout-change`). The `libtmux` Python library provides an OOP wrapper; for Go, shell out to tmux commands.

### ANSI Parsing Libraries

For parsing raw terminal output:
- `github.com/leaanthony/go-ansi-parser` — Returns `[]*StyledText` with colors, attributes, offsets
- `github.com/charmbracelet/x/ansi` — Charm's parser with `DecodeSequence` functions
- `github.com/ktr0731/go-ansisgr` — SGR-specific iterator, integrates with tcell

---

## Expect-style automation provides pattern matching

Netflix's `go-expect` and Google's `goexpect` bring Expect patterns to Go for send/wait automation:

```go
import "github.com/Netflix/go-expect"

c, _ := expect.NewConsole(expect.WithStdout(os.Stdout))
defer c.Close()

cmd := exec.Command("./diff-review-tool")
cmd.Stdin, cmd.Stdout, cmd.Stderr = c.Tty(), c.Tty(), c.Tty()
cmd.Start()

// Wait for prompt, then send command
c.ExpectString("Select file to review:")
c.SendLine("j")  // Down
c.SendLine("j")  // Down  
c.SendLine(" ")  // Select
c.ExpectString("Hunk 1/5")
```

**Google's goexpect** adds `ExpectBatch()` for sequences and native SSH spawning. This works for any TUI but requires knowing expected output patterns—brittle if the interface changes.

---

## VHS and asciinema serve dual purposes for AI and marketing

**VHS** (charmbracelet/vhs) uses ttyd + headless Chrome + xterm.js to record terminals:

```tape
Output demo.gif
Set FontSize 14
Set Width 1200

Type "go run main.go"
Enter
Sleep 1s
Down@500ms 3
Type " "
Sleep 500ms
```

VHS can output `.txt` or `.ascii` for golden file testing, enabling the **same tape scripts to generate both marketing GIFs and CI test fixtures**. However, VHS is unidirectional—it cannot read output and conditionally respond.

**asciinema's asciicast format** (NDJSON) captures timing and supports markers for navigation:

```json
{"version": 2, "width": 80, "height": 24}
[0.5, "o", "Select file: \u001b[1mREADME.md\u001b[0m"]
[1.2, "i", "j"]
[1.5, "m", "file_selected"]
```

Event codes: `o` (output), `i` (input), `r` (resize), `m` (marker). The JavaScript player provides `seek()`, `getDuration()`, and marker events—useful for building interactive documentation from recorded AI sessions.

**Dual-purpose workflow**:
1. Record AI-TUI interactions to asciicast with markers at key moments
2. Use recordings as training data for fine-tuning prompts
3. Export segments as GIFs via VHS or asciinema2gif for documentation
4. Generate golden files from recordings for regression testing

---

## Recommended architecture for AI-controllable Bubble Tea apps

The optimal architecture combines Bubble Tea's native capabilities with a gRPC sidecar:

```
┌─────────────────────────────────────────────────────────┐
│                    Bubble Tea App                        │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐  │
│  │   Model     │◄───│  Update()   │◄───│  Messages   │  │
│  │  (state)    │    │             │    │   Channel   │  │
│  └──────┬──────┘    └─────────────┘    └──────▲──────┘  │
│         │                                      │         │
│         ▼                                      │         │
│  ┌─────────────┐                    ┌─────────────────┐ │
│  │   View()    │                    │ program.Send()  │ │
│  │  (render)   │                    │ (external msgs) │ │
│  └──────┬──────┘                    └────────▲────────┘ │
│         │                                     │         │
└─────────┼─────────────────────────────────────┼─────────┘
          │                                     │
          ▼                                     │
    ┌───────────┐                      ┌────────────────┐
    │  Terminal │                      │  gRPC Server   │
    │  (human)  │                      │ (Unix socket)  │
    └───────────┘                      └────────▲───────┘
                                                │
                                       ┌────────┴───────┐
                                       │   AI Agent     │
                                       │ (Claude Code)  │
                                       └────────────────┘
```

**Implementation outline**:

```go
// proto/diffreviewer.proto
service DiffReviewer {
    rpc GetState(Empty) returns (DiffState);
    rpc SendCommand(Command) returns (CommandResult);
    rpc StreamState(Empty) returns (stream DiffState);
}

message DiffState {
    string current_file = 1;
    repeated Hunk hunks = 2;
    int32 cursor_position = 3;
    repeated int32 selected_hunks = 4;
    string mode = 5;  // "file_list", "hunk_view", "commit"
}

message Command {
    string action = 1;  // "navigate", "select", "approve", "reject"
    map<string, string> params = 2;
}
```

```go
// main.go
func main() {
    model := NewDiffModel(diffs)
    program := tea.NewProgram(model)
    
    // Start gRPC server on Unix socket
    go func() {
        lis, _ := net.Listen("unix", "/tmp/diffreview.sock")
        srv := grpc.NewServer()
        pb.RegisterDiffReviewerServer(srv, &Server{
            model:   model,
            program: program,
        })
        srv.Serve(lis)
    }()
    
    // Run TUI (blocks)
    program.Run()
}

// Custom message type for AI commands
type aiCommand struct {
    action string
    params map[string]string
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case aiCommand:
        return m.handleAICommand(msg)
    case tea.KeyMsg:
        return m.handleKeyPress(msg)
    }
    return m, nil
}
```

**Why gRPC over Unix sockets**: ~20% faster than TCP (1.02µs vs 1.27µs per call), bidirectional streaming for real-time state updates, type-safe protobuf contracts, works within Claude Code's sandboxed environment.

---

## What a Terminal DevTools Protocol would look like

Drawing from Chrome DevTools Protocol, a hypothetical **Terminal DevTools Protocol (TDP)** would need:

**Domains**:
- `Terminal` — Screen state, size, cursor, scroll region, alternate buffer
- `Input` — Keystroke injection, mouse events, paste operations
- `Process` — PTY lifecycle, command execution, environment
- `Application` — Widget tree for structured TUIs (selection state, focus, content)
- `Recording` — Capture/replay, markers, timing

**Example commands**:
```json
// Get screen state
{"id": 1, "method": "Terminal.getScreenState"}
→ {"id": 1, "result": {"rows": [...cells...], "cursor": {"x": 5, "y": 10}, "size": {"cols": 80, "rows": 24}}}

// Inject input
{"id": 2, "method": "Input.sendKey", "params": {"key": "ArrowDown", "modifiers": []}}

// Subscribe to screen updates
{"id": 3, "method": "Terminal.enable"}
→ Events: {"method": "Terminal.screenUpdated", "params": {"changedRegions": [...]}}

// For Bubble Tea apps: access application state directly
{"id": 4, "method": "Application.getState"}
→ {"id": 4, "result": {"model": {"currentFile": "main.go", "selectedHunks": [1, 3]}}}
```

**Current gap**: No standard exists. Kitty's remote control is closest but terminal-specific. The pragmatic path is application-level APIs (gRPC sidecar) rather than waiting for terminal standardization.

---

## Trade-off comparison for implementation approaches

| Approach | Latency | Reliability | Complexity | Go Support | Works in Sandbox |
|----------|---------|-------------|------------|------------|------------------|
| **gRPC sidecar** | ~1µs | Highest | Medium | Excellent | ✅ Yes |
| **PTY + vt10x** | ~10ms | High | High | Good | ✅ Yes |
| **tmux control** | ~50ms | Medium | Low | Shell out | ✅ Yes |
| **Kitty remote** | ~5ms | High | Low | Shell out | ⚠️ Requires Kitty |
| **Screen scraping** | ~100ms | Low | Medium | Good | ✅ Yes |

**For Claude Code's sandboxed environment**: The gRPC sidecar approach works best—it's pure Go, uses Unix sockets (allowed in sandbox), and provides type-safe communication. The AI agent can call a wrapper CLI that connects to the socket:

```bash
# AI-callable commands
./diffreview-ctl get-state
./diffreview-ctl send-command --action=navigate --params='{"direction":"down"}'
./diffreview-ctl stream-state  # For real-time updates
```

---

## Key repositories and resources

**Terminal emulator automation**:
- Kitty remote control: https://sw.kovidgoyal.net/kitty/remote-control/
- WezTerm CLI: https://wezfurlong.org/wezterm/cli/
- Ghostty discussions: https://github.com/ghostty-org/ghostty/discussions/2353

**AI-terminal interaction**:
- `davidhershey/ClaudePlaysPokemonStarter` — Official Anthropic Pokemon starter
- `cicero225/llm_pokemon_scaffold` — Multi-model scaffold with memory reader
- `simular-ai/Agent-S` — Desktop automation (72.6% OSWorld)

**Bubble Tea and Go TUI**:
- `charmbracelet/bubbletea` — Core framework
- `charmbracelet/x/exp/teatest` — Testing library
- `charmbracelet/wish` — SSH middleware for Bubble Tea
- `charmbracelet/vhs` — Terminal recording

**PTY and terminal emulation**:
- `creack/pty` — Go PTY library (Unix)
- `aymanbagabas/go-pty` — Cross-platform including ConPTY
- `Netflix/go-expect` — Expect patterns for Go
- `hinshun/vt10x` — VT100 emulator with state access
- `taigrr/bubbleterm` — Headless terminal with Bubble Tea integration

**ANSI parsing**:
- `leaanthony/go-ansi-parser` — Structured styled text output
- `charmbracelet/x/ansi` — Charm's ANSI toolkit

The combination of Bubble Tea's `Program.Send()`, a gRPC sidecar server, and protobuf-defined state provides the most robust architecture for an AI-controllable diff review tool—offering both interactive TUI use and programmatic control without screen scraping or terminal-specific dependencies.
