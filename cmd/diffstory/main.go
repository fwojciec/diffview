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
	"github.com/fwojciec/diffview/jsonl"
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
// It first tries to extract PR-level cases from merge commits.
// If no merge commits are found, it falls back to individual commits.
func (c *Collector) Run(ctx context.Context) error {
	// Try PR-level extraction first
	mergeHashes, err := c.Git.MergeCommits(ctx, c.RepoPath, c.Limit)
	if err != nil {
		return err
	}

	if len(mergeHashes) > 0 {
		return c.runPRLevel(ctx, mergeHashes)
	}

	// Fall back to commit-level extraction
	return c.runCommitLevel(ctx)
}

// runPRLevel extracts PR-level cases from merge commits.
func (c *Collector) runPRLevel(ctx context.Context, mergeHashes []string) error {
	parser := gitdiff.NewParser()
	encoder := json.NewEncoder(c.Output)

	for _, mergeHash := range mergeHashes {
		// Get the merge commit message to extract branch name
		mergeMessage, err := c.Git.Message(ctx, c.RepoPath, mergeHash)
		if err != nil {
			return err
		}

		branch := ParseBranchFromMergeMessage(mergeMessage)

		// Get commits in the PR (merge^1..merge^2)
		base := mergeHash + "^1"
		head := mergeHash + "^2"

		commits, err := c.Git.CommitsInRange(ctx, c.RepoPath, base, head)
		if err != nil {
			return err
		}

		// Get combined diff for the PR
		diffText, err := c.Git.DiffRange(ctx, c.RepoPath, base, head)
		if err != nil {
			return err
		}

		diff, err := parser.Parse(strings.NewReader(diffText))
		if err != nil {
			return err
		}

		// Skip PRs with no files
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

		evalCase := diffview.EvalCase{
			Input: diffview.ClassificationInput{
				Repo:    c.RepoName,
				Branch:  branch,
				Commits: commits,
				Diff:    *diff,
			},
			Story: nil,
		}
		if err := encoder.Encode(evalCase); err != nil {
			return err
		}
	}

	return nil
}

// runCommitLevel extracts individual commit cases (fallback mode).
func (c *Collector) runCommitLevel(ctx context.Context) error {
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
				Repo: c.RepoName,
				Commits: []diffview.CommitBrief{
					{Hash: hash, Message: message},
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

// ParseBranchFromMergeMessage extracts the branch name from a GitHub merge commit message.
// Format: "Merge pull request #N from user/branch-name"
func ParseBranchFromMergeMessage(message string) string {
	// Only parse the first line (merge messages may have additional body text)
	firstLine := message
	if idx := strings.IndexByte(message, '\n'); idx != -1 {
		firstLine = message[:idx]
	}

	const prefix = "Merge pull request #"
	if !strings.HasPrefix(firstLine, prefix) {
		return ""
	}
	// Find "from user/branch"
	fromIdx := strings.Index(firstLine, " from ")
	if fromIdx == -1 {
		return ""
	}
	userBranch := firstLine[fromIdx+6:] // Skip " from "
	// Extract branch name after "user/"
	slashIdx := strings.Index(userBranch, "/")
	if slashIdx == -1 {
		return userBranch
	}
	return userBranch[slashIdx+1:]
}

// ClassifyRunner classifies eval cases using an LLM classifier.
type ClassifyRunner struct {
	Output     io.Writer
	Cases      []diffview.EvalCase
	Classifier diffview.StoryClassifier
}

// Run classifies each case and writes JSONL output.
func (c *ClassifyRunner) Run(ctx context.Context) error {
	encoder := json.NewEncoder(c.Output)

	for i := range c.Cases {
		evalCase := c.Cases[i]

		// Skip cases that already have a story
		if evalCase.Story == nil {
			story, err := c.Classifier.Classify(ctx, evalCase.Input)
			if err != nil {
				return err
			}
			evalCase.Story = story
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
		return fmt.Errorf("usage: diffstory <command> [options]\n\nCommands:\n  analyze   Analyze a diff file\n  collect   Extract diffs from git history\n  classify  Classify eval cases from JSONL")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	switch os.Args[1] {
	case "analyze":
		return runAnalyze(ctx)
	case "collect":
		return runCollect(ctx)
	case "classify":
		return runClassify(ctx)
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
	maxLines := fs.Int("max-lines", 2000, "Maximum lines changed (skip larger PRs/commits)")

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

func runClassify(ctx context.Context) error {
	if len(os.Args) < 3 {
		return fmt.Errorf("usage: diffstory classify <input.jsonl>")
	}

	inputPath := os.Args[2]

	// Check for API key
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("GEMINI_API_KEY environment variable required")
	}

	// Load cases from JSONL
	loader := jsonl.NewLoader()
	cases, err := loader.Load(inputPath)
	if err != nil {
		return fmt.Errorf("failed to load cases: %w", err)
	}

	if len(cases) == 0 {
		return fmt.Errorf("no cases found in %s", inputPath)
	}

	// Set up Gemini classifier
	client, err := gemini.NewClient(ctx, apiKey)
	if err != nil {
		return fmt.Errorf("failed to create Gemini client: %w", err)
	}
	defer client.Close()

	classifier := gemini.NewClassifier(client, gemini.DefaultModel)

	runner := &ClassifyRunner{
		Output:     os.Stdout,
		Cases:      cases,
		Classifier: classifier,
	}

	return runner.Run(ctx)
}
