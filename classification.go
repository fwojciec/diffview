package diffview

import "context"

// CommitInfo captures metadata about a commit for classification.
type CommitInfo struct {
	Hash    string `json:"Hash"`
	Repo    string `json:"Repo"`
	Message string `json:"Message"`
}

// ClassificationInput is the complete input for story classification.
type ClassificationInput struct {
	Commit CommitInfo `json:"Commit"`
	Diff   Diff       `json:"Diff"`
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
