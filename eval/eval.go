// Package eval provides test helpers for LLM-as-judge evaluation patterns.
package eval

import (
	"os"
	"testing"

	"github.com/fwojciec/diffview"
)

// Eval provides assertion helpers for LLM-based test evaluation.
type Eval struct {
	judge diffview.RubricJudge
}

// New creates a new Eval with the given judge.
func New(judge diffview.RubricJudge) *Eval {
	return &Eval{judge: judge}
}

// AssertRubric evaluates whether the output satisfies the given criterion.
// If the criterion is not satisfied, the test is marked as failed.
func (e *Eval) AssertRubric(tb testing.TB, criterion, output string) {
	tb.Helper()

	result, err := e.judge.Judge(tb.Context(), criterion, output)
	if err != nil {
		tb.Errorf("rubric evaluation failed: %v", err)
		return
	}

	if !result.Passed {
		tb.Errorf("rubric criterion not satisfied: %q\nReasoning: %s", criterion, result.Reasoning)
	}
}

// SkipUnlessEvals skips the test unless GOEVALS environment variable is set.
// Use at the start of eval tests to make them opt-in.
func SkipUnlessEvals(tb testing.TB) {
	tb.Helper()
	if os.Getenv("GOEVALS") == "" {
		tb.Skip("GOEVALS not set")
	}
}
