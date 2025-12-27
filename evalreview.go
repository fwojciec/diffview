package diffview

import "time"

// EvalCase represents a case for evaluation: a diff with its LLM-generated story analysis.
type EvalCase struct {
	Commit string          `json:"commit"` // Git commit hash
	Hunks  []AnnotatedHunk `json:"hunks"`  // The diff hunks
	Story  StoryAnalysis   `json:"story"`  // The LLM-generated analysis to evaluate
}

// Judgment represents a human reviewer's evaluation of an EvalCase.
type Judgment struct {
	Commit   string    `json:"commit"`    // Links to EvalCase.Commit
	Index    int       `json:"index"`     // Position in input file (0-based)
	Pass     bool      `json:"pass"`      // Whether the story analysis is acceptable
	Critique string    `json:"critique"`  // Explanation for failure (empty if pass)
	JudgedAt time.Time `json:"judged_at"` // When judgment was recorded
}

// EvalCaseLoader loads evaluation cases from a source.
type EvalCaseLoader interface {
	Load(path string) ([]EvalCase, error)
}

// JudgmentStore persists and retrieves judgments.
type JudgmentStore interface {
	Load(path string) ([]Judgment, error)
	Save(path string, judgments []Judgment) error
}
