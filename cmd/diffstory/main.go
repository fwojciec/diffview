package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	charmlipgloss "github.com/charmbracelet/lipgloss"
	"github.com/fwojciec/diffview"
	"github.com/fwojciec/diffview/bubbletea"
	"github.com/fwojciec/diffview/chroma"
	"github.com/fwojciec/diffview/fs"
	"github.com/fwojciec/diffview/gemini"
	"github.com/fwojciec/diffview/git"
	"github.com/fwojciec/diffview/gitdiff"
	"github.com/fwojciec/diffview/lipgloss"
	"github.com/fwojciec/diffview/worddiff"
)

// ErrNoChanges is returned when the diff contains no changes to analyze.
var ErrNoChanges = errors.New("no changes to analyze")

// ErrOnBaseBranch is returned when running in branch mode while on the base branch.
var ErrOnBaseBranch = errors.New("already on base branch, no changes to show")

// App encapsulates the application logic for testing.
type App struct {
	Input      io.Reader                // Read diff from stdin (if FilePath is empty)
	FilePath   string                   // Read diff from file (takes precedence over Input)
	GitRunner  diffview.GitRunner       // Git runner for branch mode
	RepoPath   string                   // Repository path for branch mode
	BaseBranch string                   // Base branch to compare against (e.g., "main")
	Classifier diffview.StoryClassifier // Classifier for story generation
}

// Run parses the diff input and classifies it.
// Returns the parsed diff and classification for TUI display.
func (a *App) Run(ctx context.Context) (*diffview.Diff, *diffview.StoryClassification, error) {
	var input io.Reader
	if a.FilePath != "" {
		f, err := os.Open(a.FilePath)
		if err != nil {
			return nil, nil, err
		}
		defer f.Close()
		input = f
	} else if a.GitRunner != nil {
		// Branch mode: get diff from git
		diffStr, err := a.GitRunner.DiffRange(ctx, a.RepoPath, a.BaseBranch, "HEAD")
		if err != nil {
			return nil, nil, err
		}
		input = strings.NewReader(diffStr)
	} else {
		input = a.Input
	}

	parser := gitdiff.NewParser()
	diff, err := parser.Parse(input)
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

// ErrNoInput is returned when no diff input is provided.
var ErrNoInput = errors.New("no input: pipe a diff or provide a file path")

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

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Check for API key
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("GEMINI_API_KEY environment variable required")
	}

	// Set up Gemini client and classifier
	client, err := gemini.NewClient(ctx, apiKey)
	if err != nil {
		return fmt.Errorf("failed to create Gemini client: %w", err)
	}
	defer client.Close()

	geminiClassifier := gemini.NewClassifier(client, gemini.DefaultModel)
	classifier := fs.NewClassifier(geminiClassifier, fs.DefaultCacheDir())

	app := &App{
		Classifier: classifier,
	}

	// Check for file path argument
	if len(os.Args) >= 2 {
		app.FilePath = os.Args[1]
	} else {
		// Check if stdin is a pipe
		stat, err := os.Stdin.Stat()
		if err != nil {
			return fmt.Errorf("error checking stdin: %w", err)
		}
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			// Stdin is a pipe, use pipe mode
			app.Input = os.Stdin
		} else {
			// No pipe, use branch mode
			gitRunner := git.NewRunner()
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current directory: %w", err)
			}

			// Check if we're on the base branch
			currentBranch, err := gitRunner.CurrentBranch(ctx, cwd)
			if err != nil {
				return fmt.Errorf("failed to get current branch: %w", err)
			}
			baseBranch := os.Getenv("DIFFSTORY_BASE_BRANCH")
			if baseBranch == "" {
				baseBranch = "main"
			}
			if currentBranch == baseBranch {
				return ErrOnBaseBranch
			}

			app.GitRunner = gitRunner
			app.RepoPath = cwd
			app.BaseBranch = baseBranch
		}
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

	// Set up syntax highlighting
	theme := lipgloss.DefaultTheme()
	detector := chroma.NewDetector()
	tokenizer, err := chroma.NewTokenizer(chroma.StyleFromPalette(theme.Palette()))
	if err != nil {
		return fmt.Errorf("failed to set up syntax highlighting: %w", err)
	}

	// Launch StoryModel TUI
	m := bubbletea.NewStoryModel(diff, classification,
		bubbletea.WithStoryTheme(theme),
		bubbletea.WithStoryLanguageDetector(detector),
		bubbletea.WithStoryTokenizer(tokenizer),
		bubbletea.WithStoryWordDiffer(worddiff.NewDiffer()),
	)
	p := tea.NewProgram(m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
		tea.WithContext(ctx),
	)

	_, err = p.Run()
	return err
}
