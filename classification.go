package diffview

import "context"

// CommitBrief captures essential commit metadata for PR context.
type CommitBrief struct {
	Hash    string `json:"hash"`
	Message string `json:"message"`
}

// ClassificationInput is the complete input for story classification.
// It represents a PR's worth of changes: multiple commits with their combined diff.
type ClassificationInput struct {
	Repo    string        `json:"repo"`
	Branch  string        `json:"branch"`
	Commits []CommitBrief `json:"commits"`
	Diff    Diff          `json:"diff"`
}

// FirstCommitMessage returns the message of the first commit, or empty if none.
// This is a migration helper - prefer iterating Commits directly.
func (c ClassificationInput) FirstCommitMessage() string {
	if len(c.Commits) == 0 {
		return ""
	}
	return c.Commits[0].Message
}

// FirstCommitHash returns the hash of the first commit, or empty if none.
// This is a migration helper - prefer iterating Commits directly.
func (c ClassificationInput) FirstCommitHash() string {
	if len(c.Commits) == 0 {
		return ""
	}
	return c.Commits[0].Hash
}

// CaseID returns a unique identifier for this case using repo/branch format.
// This uniquely identifies a PR-level case for judgment linking.
func (c ClassificationInput) CaseID() string {
	return c.Repo + "/" + c.Branch
}

// StoryClassification is the LLM's structured output for a diff.
type StoryClassification struct {
	ChangeType string    `json:"change_type"` // bugfix, feature, refactor, chore, docs
	Narrative  string    `json:"narrative"`   // cause-effect, core-periphery, before-after, etc.
	Summary    string    `json:"summary"`     // One sentence describing the change
	Sections   []Section `json:"sections"`    // Ordered sections grouping related hunks
}

// Section groups related hunks with a narrative role.
type Section struct {
	Role        string    `json:"role"`        // problem, fix, test, core, supporting, etc.
	Title       string    `json:"title"`       // Human-readable section title
	Hunks       []HunkRef `json:"hunks"`       // References to hunks in this section
	Explanation string    `json:"explanation"` // Why this section matters
}

// HunkRef references a specific hunk with classification metadata.
type HunkRef struct {
	File         string `json:"file"`
	HunkIndex    int    `json:"hunk_index"`
	Category     string `json:"category"`                // refactoring, systematic, core, noise
	Collapsed    bool   `json:"collapsed"`               // Whether to collapse in viewer
	CollapseText string `json:"collapse_text,omitempty"` // Summary when collapsed
}

// StoryClassifier produces structured classification from diff + commit info.
type StoryClassifier interface {
	Classify(ctx context.Context, input ClassificationInput) (*StoryClassification, error)
}
