// Package diffview provides domain types for parsing and viewing diffs.
package diffview

import (
	"context"
	"io/fs"
)

// Diff represents a complete diff containing one or more file changes.
type Diff struct {
	Files []FileDiff
}

// FileDiff represents changes to a single file.
type FileDiff struct {
	OldPath   string      // "a/file.go" or empty for new files
	NewPath   string      // "b/file.go" or empty for deleted files
	Operation FileOp      // Added, Deleted, Modified, Renamed, Copied
	IsBinary  bool        // Binary files have no hunks
	OldMode   fs.FileMode // 0 if unchanged
	NewMode   fs.FileMode // For permission changes
	Hunks     []Hunk
	Extended  []string // Raw extended headers for passthrough
}

// Stats returns the number of added and deleted lines in the file.
func (f FileDiff) Stats() (added, deleted int) {
	for _, hunk := range f.Hunks {
		for _, line := range hunk.Lines {
			switch line.Type {
			case LineAdded:
				added++
			case LineDeleted:
				deleted++
			}
		}
	}
	return added, deleted
}

// FileOp represents the type of operation performed on a file.
type FileOp int

// File operation types.
const (
	FileModified FileOp = iota
	FileAdded
	FileDeleted
	FileRenamed
	FileCopied
)

// Hunk represents a contiguous block of changes within a file.
type Hunk struct {
	OldStart int    // From @@ -X,...
	OldCount int    // From @@ -X,Y ...
	NewStart int    // From @@ ...,+X
	NewCount int    // From @@ ...,+X,Y
	Section  string // Optional function name after @@ ... @@
	Lines    []Line
}

// Line represents a single line within a hunk.
type Line struct {
	Type       LineType
	Content    string
	OldLineNum int  // 0 if line is Added
	NewLineNum int  // 0 if line is Deleted
	NoNewline  bool // "\ No newline at end of file" marker
}

// LineType represents the type of a diff line.
type LineType int

// Line types.
const (
	LineContext LineType = iota
	LineAdded
	LineDeleted
)

// Segment represents a portion of text within a line for word-level diffing.
// Used to highlight specific changed words/characters within modified lines.
type Segment struct {
	Text    string // The text content of this segment
	Changed bool   // True if this segment differs between old/new versions
}

// WordDiffer computes word-level differences between two strings.
type WordDiffer interface {
	// Diff returns segments for both the old and new strings,
	// marking which portions changed between them.
	Diff(old, new string) (oldSegs, newSegs []Segment)
}

// GitRunner provides access to git operations for extracting commit history.
type GitRunner interface {
	// Log returns commit hashes from the repository at repoPath, limited to n commits.
	// Deprecated: Use MergeCommits for PR-level extraction.
	Log(ctx context.Context, repoPath string, limit int) ([]string, error)
	// Show returns the diff for a specific commit hash.
	// Deprecated: Use DiffRange for PR-level extraction.
	Show(ctx context.Context, repoPath string, hash string) (string, error)
	// Message returns the commit message for a specific commit hash.
	// Deprecated: Use CommitsInRange for PR-level extraction.
	Message(ctx context.Context, repoPath string, hash string) (string, error)

	// MergeCommits returns merge commit hashes from the repository, limited to n commits.
	// Used to find PR boundaries in git history.
	MergeCommits(ctx context.Context, repoPath string, limit int) ([]string, error)
	// CommitsInRange returns commits between base and head (base exclusive, head inclusive).
	// For a merge commit, use merge^1..merge^2 to get all PR commits.
	CommitsInRange(ctx context.Context, repoPath, base, head string) ([]CommitBrief, error)
	// DiffRange returns the combined diff between base and head.
	// Uses three-dot notation (base...head) to show changes introduced by head since common ancestor.
	DiffRange(ctx context.Context, repoPath, base, head string) (string, error)
	// CurrentBranch returns the name of the currently checked out branch.
	CurrentBranch(ctx context.Context, repoPath string) (string, error)
	// MergeBase returns the best common ancestor commit between two refs.
	MergeBase(ctx context.Context, repoPath, ref1, ref2 string) (string, error)
	// DefaultBranch returns the default branch name from origin/HEAD.
	// Returns an error if no remote is configured.
	DefaultBranch(ctx context.Context, repoPath string) (string, error)
}
