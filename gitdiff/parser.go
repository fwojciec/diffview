// Package gitdiff implements diff parsing using bluekeyes/go-gitdiff.
package gitdiff

import (
	"io"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
	"github.com/fwojciec/diffview"
)

// Compile-time interface verification.
var _ diffview.Parser = (*Parser)(nil)

// Parser parses unified diff content using go-gitdiff.
type Parser struct{}

// NewParser creates a new Parser.
func NewParser() *Parser {
	return &Parser{}
}

// Parse reads diff content and returns the parsed result.
func (p *Parser) Parse(r io.Reader) (*diffview.Diff, error) {
	files, _, err := gitdiff.Parse(r)
	if err != nil {
		return nil, err
	}

	result := &diffview.Diff{
		Files: make([]diffview.FileDiff, 0, len(files)),
	}

	for _, f := range files {
		fileDiff := convertFile(f)
		result.Files = append(result.Files, fileDiff)
	}

	return result, nil
}

func convertFile(f *gitdiff.File) diffview.FileDiff {
	fd := diffview.FileDiff{
		OldPath:  f.OldName,
		NewPath:  f.NewName,
		IsBinary: f.IsBinary,
		OldMode:  f.OldMode,
		NewMode:  f.NewMode,
		// Extended field is not populated - go-gitdiff parses extended headers
		// into structured fields rather than exposing raw header lines.
	}

	// Determine file operation
	switch {
	case f.IsNew:
		fd.Operation = diffview.FileAdded
	case f.IsDelete:
		fd.Operation = diffview.FileDeleted
	case f.IsRename:
		fd.Operation = diffview.FileRenamed
	case f.IsCopy:
		fd.Operation = diffview.FileCopied
	default:
		fd.Operation = diffview.FileModified
	}

	// Convert text fragments to hunks
	fd.Hunks = make([]diffview.Hunk, 0, len(f.TextFragments))
	for _, frag := range f.TextFragments {
		hunk := convertFragment(frag)
		fd.Hunks = append(fd.Hunks, hunk)
	}

	return fd
}

func convertFragment(frag *gitdiff.TextFragment) diffview.Hunk {
	hunk := diffview.Hunk{
		OldStart: int(frag.OldPosition),
		OldCount: int(frag.OldLines),
		NewStart: int(frag.NewPosition),
		NewCount: int(frag.NewLines),
		Section:  frag.Comment,
	}

	// Track line numbers for old and new files
	oldLineNum := int(frag.OldPosition)
	newLineNum := int(frag.NewPosition)

	for _, l := range frag.Lines {
		line := diffview.Line{
			Content:   l.Line,
			NoNewline: l.NoEOL(),
		}

		switch l.Op {
		case gitdiff.OpContext:
			line.Type = diffview.LineContext
			line.OldLineNum = oldLineNum
			line.NewLineNum = newLineNum
			oldLineNum++
			newLineNum++
		case gitdiff.OpAdd:
			line.Type = diffview.LineAdded
			line.OldLineNum = 0
			line.NewLineNum = newLineNum
			newLineNum++
		case gitdiff.OpDelete:
			line.Type = diffview.LineDeleted
			line.OldLineNum = oldLineNum
			line.NewLineNum = 0
			oldLineNum++
		}

		hunk.Lines = append(hunk.Lines, line)
	}

	return hunk
}
