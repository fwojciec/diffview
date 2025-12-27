package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/fwojciec/diffview"
	"github.com/fwojciec/diffview/gemini"
	"github.com/fwojciec/diffview/git"
	"github.com/fwojciec/diffview/gitdiff"
)

// ErrNoChanges is returned when the diff contains no changes to analyze.
var ErrNoChanges = errors.New("no changes to analyze")

// Collector extracts diffs from git history.
type Collector struct {
	Output   io.Writer
	RepoPath string
	RepoName string
	Limit    int
	MinLines int
	MaxLines int
	Git      diffview.GitRunner
}

// Run extracts diffs from git history and writes JSONL output.
func (c *Collector) Run(ctx context.Context) error {
	hashes, err := c.Git.Log(ctx, c.RepoPath, c.Limit)
	if err != nil {
		return err
	}

	parser := gitdiff.NewParser()
	encoder := json.NewEncoder(c.Output)

	for _, hash := range hashes {
		diffText, err := c.Git.Show(ctx, c.RepoPath, hash)
		if err != nil {
			return err
		}

		diff, err := parser.Parse(strings.NewReader(diffText))
		if err != nil {
			return err
		}

		// Skip commits with no files (e.g., merge commits)
		if len(diff.Files) == 0 {
			continue
		}

		// Count total lines changed
		totalLines := countLinesChanged(diff)

		// Apply line filters
		if c.MinLines > 0 && totalLines < c.MinLines {
			continue
		}
		if c.MaxLines > 0 && totalLines > c.MaxLines {
			continue
		}

		// Get commit message
		message, err := c.Git.Message(ctx, c.RepoPath, hash)
		if err != nil {
			return err
		}

		evalCase := diffview.EvalCase{
			Input: diffview.ClassificationInput{
				Commit: diffview.CommitInfo{
					Hash:    hash,
					Repo:    c.RepoName,
					Message: message,
				},
				Diff: *diff,
			},
			Story: nil, // Not classified yet
		}
		if err := encoder.Encode(evalCase); err != nil {
			return err
		}
	}

	return nil
}

// countLinesChanged returns the total number of added + deleted lines in a diff.
func countLinesChanged(diff *diffview.Diff) int {
	total := 0
	for _, file := range diff.Files {
		added, deleted := file.Stats()
		total += added + deleted
	}
	return total
}

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
		return fmt.Errorf("usage: diffstory <command> [options]\n\nCommands:\n  analyze  Analyze a diff file\n  collect  Extract diffs from git history")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	switch os.Args[1] {
	case "analyze":
		return runAnalyze(ctx)
	case "collect":
		return runCollect(ctx)
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

func runCollect(ctx context.Context) error {
	fs := flag.NewFlagSet("collect", flag.ExitOnError)
	limit := fs.Int("limit", 50, "Maximum number of commits to extract")
	repo := fs.String("repo", "", "Repository name (defaults to directory name)")
	minLines := fs.Int("min-lines", 5, "Minimum lines changed (skip smaller commits)")
	maxLines := fs.Int("max-lines", 500, "Maximum lines changed (skip larger commits)")

	if err := fs.Parse(os.Args[2:]); err != nil {
		return err
	}

	args := fs.Args()
	repoPath := "."
	if len(args) > 0 {
		repoPath = args[0]
	}

	// Derive repo name from path if not specified
	repoName := *repo
	if repoName == "" {
		absPath, err := filepath.Abs(repoPath)
		if err != nil {
			return fmt.Errorf("failed to resolve repo path: %w", err)
		}
		repoName = filepath.Base(absPath)
	}

	collector := &Collector{
		Output:   os.Stdout,
		RepoPath: repoPath,
		RepoName: repoName,
		Limit:    *limit,
		MinLines: *minLines,
		MaxLines: *maxLines,
		Git:      git.NewRunner(),
	}

	return collector.Run(ctx)
}
