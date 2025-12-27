package diffview

import "context"

// RubricResult represents the outcome of an LLM-as-judge evaluation.
type RubricResult struct {
	Passed    bool   // Whether the output satisfied the criterion
	Reasoning string // LLM's explanation for the judgment
}

// RubricJudge evaluates text output against natural language criteria.
// Used for LLM-as-judge testing patterns.
type RubricJudge interface {
	// Judge evaluates whether the output satisfies the given criterion.
	Judge(ctx context.Context, criterion, output string) (*RubricResult, error)
}
