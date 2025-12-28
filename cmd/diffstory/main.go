package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/fwojciec/diffview"
	"github.com/fwojciec/diffview/gemini"
	"github.com/fwojciec/diffview/gitdiff"
)

// ErrNoChanges is returned when the diff contains no changes to analyze.
var ErrNoChanges = errors.New("no changes to analyze")

// App encapsulates the application logic for testing.
type App struct {
	Input     io.Reader // Read diff from stdin (if FilePath is empty)
	FilePath  string    // Read diff from file (takes precedence over Input)
	Output    io.Writer
	Generator diffview.StoryGenerator
}

// Run parses the diff input and outputs the analysis as JSON.
func (a *App) Run(ctx context.Context) error {
	var input io.Reader
	if a.FilePath != "" {
		f, err := os.Open(a.FilePath)
		if err != nil {
			return err
		}
		defer f.Close()
		input = f
	} else {
		input = a.Input
	}

	parser := gitdiff.NewParser()
	diff, err := parser.Parse(input)
	if err != nil {
		return err
	}

	if len(diff.Files) == 0 {
		return ErrNoChanges
	}

	// Annotate hunks with IDs that include file path for context
	var annotated []diffview.AnnotatedHunk
	for _, file := range diff.Files {
		filePath := file.NewPath
		if filePath == "" {
			filePath = file.OldPath // For deleted files
		}
		for hunkIdx, hunk := range file.Hunks {
			annotated = append(annotated, diffview.AnnotatedHunk{
				ID:   fmt.Sprintf("%s:h%d", filePath, hunkIdx),
				Hunk: hunk,
			})
		}
	}

	if len(annotated) == 0 {
		return ErrNoChanges
	}

	analysis, err := a.Generator.Generate(ctx, annotated)
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(a.Output)
	encoder.SetIndent("", "  ")
	return encoder.Encode(analysis)
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 2 {
		return fmt.Errorf("usage: diffstory analyze [path/to/diff.patch]\n       or: git diff | diffstory analyze")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	switch os.Args[1] {
	case "analyze":
		return runAnalyze(ctx)
	default:
		return fmt.Errorf("unknown command: %s", os.Args[1])
	}
}

func runAnalyze(ctx context.Context) error {
	// Check for API key
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("GEMINI_API_KEY environment variable required")
	}

	// Set up Gemini client
	client, err := gemini.NewClient(ctx, apiKey)
	if err != nil {
		return fmt.Errorf("failed to create Gemini client: %w", err)
	}
	defer client.Close()

	gen := gemini.NewGenerator(client, gemini.DefaultModel)

	app := &App{
		Output:    os.Stdout,
		Generator: gen,
	}

	// Check for file path argument
	if len(os.Args) >= 3 {
		app.FilePath = os.Args[2]
	} else {
		// Check if stdin is a pipe
		stat, err := os.Stdin.Stat()
		if err != nil {
			return fmt.Errorf("error checking stdin: %w", err)
		}
		if (stat.Mode() & os.ModeCharDevice) != 0 {
			return fmt.Errorf("usage: diffstory analyze [path/to/diff.patch]\n       or: git diff | diffstory analyze")
		}
		app.Input = os.Stdin
	}

	return app.Run(ctx)
}
