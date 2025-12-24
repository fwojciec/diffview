package main_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/fwojciec/diffview"
	main "github.com/fwojciec/diffview/cmd/diffview"
	"github.com/fwojciec/diffview/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApp_Run_Success(t *testing.T) {
	t.Parallel()

	input := "diff --git a/file.txt b/file.txt\n"
	expectedDiff := &diffview.Diff{
		Files: []diffview.FileDiff{{OldPath: "a/file.txt"}},
	}

	var parsedInput string
	var viewedDiff *diffview.Diff

	app := &main.App{
		Stdin: strings.NewReader(input),
		Parser: &mock.Parser{
			ParseFn: func(r io.Reader) (*diffview.Diff, error) {
				data, _ := io.ReadAll(r)
				parsedInput = string(data)
				return expectedDiff, nil
			},
		},
		Viewer: &mock.Viewer{
			ViewFn: func(ctx context.Context, diff *diffview.Diff) error {
				viewedDiff = diff
				return nil
			},
		},
	}

	err := app.Run(context.Background())

	require.NoError(t, err)
	assert.Equal(t, input, parsedInput, "parser should receive stdin content")
	assert.Equal(t, expectedDiff, viewedDiff, "viewer should receive parsed diff")
}

func TestApp_Run_ParseError(t *testing.T) {
	t.Parallel()

	parseErr := errors.New("invalid diff format")
	app := &main.App{
		Stdin: strings.NewReader("invalid content"),
		Parser: &mock.Parser{
			ParseFn: func(r io.Reader) (*diffview.Diff, error) {
				return nil, parseErr
			},
		},
		Viewer: &mock.Viewer{},
	}

	err := app.Run(context.Background())

	require.Error(t, err)
	assert.Equal(t, parseErr, err)
}

func TestApp_Run_ViewError(t *testing.T) {
	t.Parallel()

	viewErr := errors.New("terminal error")
	app := &main.App{
		Stdin: strings.NewReader("valid diff content"),
		Parser: &mock.Parser{
			ParseFn: func(r io.Reader) (*diffview.Diff, error) {
				return &diffview.Diff{Files: []diffview.FileDiff{{}}}, nil
			},
		},
		Viewer: &mock.Viewer{
			ViewFn: func(ctx context.Context, diff *diffview.Diff) error {
				return viewErr
			},
		},
	}

	err := app.Run(context.Background())

	require.Error(t, err)
	assert.Equal(t, viewErr, err)
}

func TestApp_Run_EmptyDiff(t *testing.T) {
	t.Parallel()

	viewerCalled := false
	app := &main.App{
		Stdin: strings.NewReader(""),
		Parser: &mock.Parser{
			ParseFn: func(r io.Reader) (*diffview.Diff, error) {
				return &diffview.Diff{Files: nil}, nil
			},
		},
		Viewer: &mock.Viewer{
			ViewFn: func(ctx context.Context, diff *diffview.Diff) error {
				viewerCalled = true
				return nil
			},
		},
	}

	err := app.Run(context.Background())

	require.ErrorIs(t, err, main.ErrNoChanges)
	assert.False(t, viewerCalled, "viewer should not be called for empty diff")
}
