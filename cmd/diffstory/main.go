package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	charmlipgloss "github.com/charmbracelet/lipgloss"
	"github.com/fwojciec/diffstory"
	"github.com/fwojciec/diffstory/bubbletea"
	"github.com/fwojciec/diffstory/chroma"
	"github.com/fwojciec/diffstory/fs"
	"github.com/fwojciec/diffstory/gemini"
	"github.com/fwojciec/diffstory/git"
	"github.com/fwojciec/diffstory/gitdiff"
	"github.com/fwojciec/diffstory/jsonl"
	"github.com/fwojciec/diffstory/lipgloss"
	"github.com/fwojciec/diffstory/worddiff"
)

// ErrNoChanges is returned when the diff contains no changes to analyze.
var ErrNoChanges = errors.New("no changes to analyze")

// ErrInvalidRange is returned when a commit range argument is malformed.
var ErrInvalidRange = errors.New("invalid commit range: expected format like 'main...feature' or 'HEAD~3..HEAD'")

// ErrOnBaseBranch is returned when running in branch mode while on the base branch.
var ErrOnBaseBranch = errors.New("already on base branch, no changes to show")

// ParseRange parses a git commit range specification into its components.
// Supports both two-dot (A..B) and three-dot (A...B) notation.
// Returns base ref, head ref, and any error.
func ParseRange(rangeSpec string) (base, head string, err error) {
	// Try three-dot first (must check before two-dot since "..." contains "..")
	if idx := strings.Index(rangeSpec, "..."); idx != -1 {
		base = rangeSpec[:idx]
		head = rangeSpec[idx+3:]
	} else if idx := strings.Index(rangeSpec, ".."); idx != -1 {
		base = rangeSpec[:idx]
		head = rangeSpec[idx+2:]
	} else {
		return "", "", ErrInvalidRange
	}

	if base == "" || head == "" {
		return "", "", ErrInvalidRange
	}

	return base, head, nil
}

// App encapsulates the application logic for testing.
type App struct {
	GitRunner  diffview.GitRunner       // Git runner for git operations
	RepoPath   string                   // Repository path
	BaseBranch string                   // Base branch (auto-detected if empty)
	Range      string                   // Raw commit range (e.g., "main...feature"), overrides BaseBranch
	Classifier diffview.StoryClassifier // Classifier for story generation
}

// Run parses the diff input and classifies it.
// Returns the parsed diff and classification for TUI display.
func (a *App) Run(ctx context.Context) (*diffview.Diff, *diffview.StoryClassification, error) {
	// Get diff from git - use raw Range if provided, otherwise use BaseBranch...HEAD
	var diffStr string
	var err error
	if a.Range != "" {
		diffStr, err = a.GitRunner.Diff(ctx, a.RepoPath, a.Range)
	} else {
		diffStr, err = a.GitRunner.DiffRange(ctx, a.RepoPath, a.BaseBranch, "HEAD")
	}
	if err != nil {
		return nil, nil, err
	}

	parser := gitdiff.NewParser()
	diff, err := parser.Parse(strings.NewReader(diffStr))
	if err != nil {
		return nil, nil, err
	}

	if len(diff.Files) == 0 {
		return nil, nil, ErrNoChanges
	}

	// Build classification input with the parsed diff
	classInput := diffview.ClassificationInput{
		Diff: *diff,
	}

	classification, err := a.Classifier.Classify(ctx, classInput)
	if err != nil {
		return nil, nil, err
	}

	return diff, classification, nil
}

// spinner displays a progress indicator on stderr while a long-running operation executes.
type spinner struct {
	frames   []string
	interval time.Duration
	message  string
	w        io.Writer
	stop     chan struct{}
	done     chan struct{}
}

// newSpinner creates a spinner that writes to the given writer.
func newSpinner(w io.Writer, message string) *spinner {
	return &spinner{
		frames:   []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		interval: 80 * time.Millisecond,
		message:  message,
		w:        w,
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
	}
}

// Start begins the spinner animation in a goroutine.
func (s *spinner) Start() {
	go func() {
		defer close(s.done)
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		frame := 0
		// Print initial frame immediately
		fmt.Fprintf(s.w, "\r%s %s", s.frames[frame], s.message)
		frame = (frame + 1) % len(s.frames)

		for {
			select {
			case <-s.stop:
				// Clear the spinner line using display width for Unicode correctness
				clearLen := charmlipgloss.Width(s.frames[0]) + 1 + charmlipgloss.Width(s.message)
				fmt.Fprintf(s.w, "\r%s\r", strings.Repeat(" ", clearLen))
				return
			case <-ticker.C:
				fmt.Fprintf(s.w, "\r%s %s", s.frames[frame], s.message)
				frame = (frame + 1) % len(s.frames)
			}
		}
	}()
}

// Stop halts the spinner and clears its output.
func (s *spinner) Stop() {
	close(s.stop)
	<-s.done
}

// isTerminal returns true if the given file is a terminal.
func isTerminal(f *os.File) bool {
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `Usage: diffstory [range | command]

Modes:
  (default)              Analyze current branch diff vs auto-detected base
  <range>                Analyze diff for specific commit range
  replay <file> [index]  Replay a saved eval case from JSONL file

Range examples:
  main...feature         Three-dot: changes on feature since diverging from main
  HEAD~3..HEAD           Two-dot: diff between two points

Examples:
  diffstory                      # Analyze current branch vs base
  diffstory main...feature       # Analyze specific branch comparison
  diffstory HEAD~3..HEAD         # Analyze last 3 commits
  diffstory replay cases.jsonl   # Replay first case
  diffstory replay cases.jsonl 2 # Replay third case (0-indexed)
`)
}

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Check for subcommand or range argument
	var rangeArg string
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "replay":
			return runReplay(ctx)
		case "-h", "--help", "help":
			usage()
			return nil
		default:
			// Validate as commit range - provides helpful error for malformed ranges
			if _, _, err := ParseRange(os.Args[1]); err != nil {
				return fmt.Errorf("unknown argument %q (use --help for usage)", os.Args[1])
			}
			rangeArg = os.Args[1]
		}
	}

	// Check for API key
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("GEMINI_API_KEY environment variable required")
	}

	// Set up git runner and detect repo
	gitRunner := git.NewRunner()
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	var baseBranch, currentBranch string
	if rangeArg == "" {
		// Branch mode: auto-detect base branch from origin/HEAD
		baseBranch, err = gitRunner.DefaultBranch(ctx, cwd)
		if err != nil {
			return fmt.Errorf("failed to detect base branch: %w", err)
		}

		// Check if we're on the base branch
		currentBranch, err = gitRunner.CurrentBranch(ctx, cwd)
		if err != nil {
			return fmt.Errorf("failed to get current branch: %w", err)
		}
		if currentBranch == baseBranch {
			return ErrOnBaseBranch
		}
	}

	// Set up Gemini client and classifier
	client, err := gemini.NewClient(ctx, apiKey)
	if err != nil {
		return fmt.Errorf("failed to create Gemini client: %w", err)
	}
	defer client.Close()

	geminiClassifier := gemini.NewClassifier(client, gemini.DefaultModel,
		gemini.WithValidationRetry(2)) // Retry once if LLM returns invalid hunk references
	classifier := fs.NewClassifier(geminiClassifier, fs.DefaultCacheDir())

	app := &App{
		GitRunner:  gitRunner,
		RepoPath:   cwd,
		BaseBranch: baseBranch,
		Range:      rangeArg,
		Classifier: classifier,
	}

	// Show spinner while processing (only if stderr is a terminal)
	var spin *spinner
	if isTerminal(os.Stderr) {
		spin = newSpinner(os.Stderr, "Classifying diff...")
		spin.Start()
	}

	diff, classification, err := app.Run(ctx)

	// Stop spinner before TUI or error output
	if spin != nil {
		spin.Stop()
	}

	if err != nil {
		return err
	}

	// Get commits for ClassificationInput
	var commits []diffview.CommitBrief
	var branchName string
	if rangeArg != "" {
		// Range mode: parse range and get commits
		base, head, parseErr := ParseRange(rangeArg)
		if parseErr == nil {
			commits, _ = gitRunner.CommitsInRange(ctx, cwd, base, head)
		}
		branchName = rangeArg // Use range as "branch" name for context
	} else {
		// Branch mode: use baseBranch...HEAD
		commits, _ = gitRunner.CommitsInRange(ctx, cwd, baseBranch, "HEAD")
		branchName = currentBranch
	}

	// Build ClassificationInput for case saving
	classInput := diffview.ClassificationInput{
		Repo:    filepath.Base(cwd),
		Branch:  branchName,
		Commits: commits,
		Diff:    *diff,
	}

	// Set up syntax highlighting
	theme := lipgloss.DefaultTheme()
	detector := chroma.NewDetector()
	tokenizer, err := chroma.NewTokenizer(chroma.StyleFromPalette(theme.Palette()))
	if err != nil {
		return fmt.Errorf("failed to set up syntax highlighting: %w", err)
	}

	// Curated cases go to a fixed location in cwd
	curatedPath := filepath.Join(cwd, "eval-curated.jsonl")

	// Launch StoryModel TUI
	m := bubbletea.NewStoryModel(diff, classification,
		bubbletea.WithStoryTheme(theme),
		bubbletea.WithStoryLanguageDetector(detector),
		bubbletea.WithStoryTokenizer(tokenizer),
		bubbletea.WithStoryWordDiffer(worddiff.NewDiffer()),
		bubbletea.WithIntroSlide(),
		bubbletea.WithStoryInput(classInput),
		bubbletea.WithStoryCaseSaver(jsonl.NewSaver(), curatedPath),
	)
	p := tea.NewProgram(m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
		tea.WithContext(ctx),
	)

	_, err = p.Run()
	return err
}

func runReplay(ctx context.Context) error {
	// Parse replay arguments: replay <file> [index]
	if len(os.Args) < 3 {
		return fmt.Errorf("replay requires a file path: diffstory replay <file.jsonl> [index]")
	}

	filePath := os.Args[2]
	index := 0
	if len(os.Args) > 3 {
		if _, err := fmt.Sscanf(os.Args[3], "%d", &index); err != nil {
			return fmt.Errorf("invalid index %q: must be a non-negative integer", os.Args[3])
		}
	}

	app := &ReplayApp{
		Loader:   jsonl.NewLoader(),
		FilePath: filePath,
		Index:    index,
	}

	diff, story, err := app.Run()
	if err != nil {
		return err
	}

	// Set up syntax highlighting
	theme := lipgloss.DefaultTheme()
	detector := chroma.NewDetector()
	tokenizer, err := chroma.NewTokenizer(chroma.StyleFromPalette(theme.Palette()))
	if err != nil {
		return fmt.Errorf("failed to set up syntax highlighting: %w", err)
	}

	// Launch StoryModel TUI (without case saving - this is replay mode)
	m := bubbletea.NewStoryModel(diff, story,
		bubbletea.WithStoryTheme(theme),
		bubbletea.WithStoryLanguageDetector(detector),
		bubbletea.WithStoryTokenizer(tokenizer),
		bubbletea.WithStoryWordDiffer(worddiff.NewDiffer()),
		bubbletea.WithIntroSlide(),
	)
	p := tea.NewProgram(m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
		tea.WithContext(ctx),
	)

	_, err = p.Run()
	return err
}
