package diffview

import "time"

// EvalCase represents a case for evaluation: a diff with its LLM-generated classification.
type EvalCase struct {
	Input ClassificationInput  `json:"input"` // The input for classification
	Story *StoryClassification `json:"story"` // The LLM-generated classification (nil if not yet classified)
}

// Judgment represents a human reviewer's evaluation of an EvalCase.
type Judgment struct {
	CaseID   string    `json:"case_id"`   // Links to EvalCase.Input.CaseID() (repo/branch)
	Index    int       `json:"index"`     // Position in input file (0-based)
	Judged   bool      `json:"judged"`    // Whether pass/fail has been explicitly set
	Pass     bool      `json:"pass"`      // Whether the classification is acceptable
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

// Clipboard provides copy-to-clipboard functionality.
type Clipboard interface {
	Copy(content string) error
}
