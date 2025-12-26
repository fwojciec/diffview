package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/fwojciec/diffview"
	"github.com/fwojciec/diffview/bubbletea"
	"github.com/fwojciec/diffview/gitdiff"
	"github.com/fwojciec/diffview/lipgloss"
)

// ErrNoChanges is returned when the diff contains no changes to display.
var ErrNoChanges = errors.New("no changes to display")

// App encapsulates the application logic for testing.
type App struct {
	Stdin  io.Reader
	Parser diffview.Parser
	Viewer diffview.Viewer
}

// Run parses stdin and displays the diff.
func (a *App) Run(ctx context.Context) error {
	diff, err := a.Parser.Parse(a.Stdin)
	if err != nil {
		return err
	}
	if len(diff.Files) == 0 {
		return ErrNoChanges
	}
	return a.Viewer.View(ctx, diff)
}

func main() {
	// Check if stdin is a pipe (not a terminal)
	stat, err := os.Stdin.Stat()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error checking stdin:", err)
		os.Exit(1)
	}
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		fmt.Fprintln(os.Stderr, "Usage: git diff | diffview")
		os.Exit(1)
	}

	// Set up context with signal handling for graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	app := &App{
		Stdin:  os.Stdin,
		Parser: gitdiff.NewParser(),
		Viewer: bubbletea.NewViewer(lipgloss.DefaultTheme()),
	}

	if err := app.Run(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
