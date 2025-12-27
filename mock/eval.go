package mock

import (
	"context"

	"github.com/fwojciec/diffview"
)

// Compile-time interface verification.
var (
	_ diffview.EvalCaseLoader = (*EvalCaseLoader)(nil)
	_ diffview.JudgmentStore  = (*JudgmentStore)(nil)
	_ diffview.RubricJudge    = (*RubricJudge)(nil)
)

// EvalCaseLoader is a mock implementation of diffview.EvalCaseLoader.
type EvalCaseLoader struct {
	LoadFn func(path string) ([]diffview.EvalCase, error)
}

func (l *EvalCaseLoader) Load(path string) ([]diffview.EvalCase, error) {
	return l.LoadFn(path)
}

// JudgmentStore is a mock implementation of diffview.JudgmentStore.
type JudgmentStore struct {
	LoadFn func(path string) ([]diffview.Judgment, error)
	SaveFn func(path string, judgments []diffview.Judgment) error
}

func (s *JudgmentStore) Load(path string) ([]diffview.Judgment, error) {
	return s.LoadFn(path)
}

func (s *JudgmentStore) Save(path string, judgments []diffview.Judgment) error {
	return s.SaveFn(path, judgments)
}

// RubricJudge is a mock implementation of diffview.RubricJudge.
type RubricJudge struct {
	JudgeFn func(ctx context.Context, criterion, output string) (*diffview.RubricResult, error)
}

func (j *RubricJudge) Judge(ctx context.Context, criterion, output string) (*diffview.RubricResult, error) {
	return j.JudgeFn(ctx, criterion, output)
}
