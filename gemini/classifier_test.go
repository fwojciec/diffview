package gemini_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/fwojciec/diffview"
	"github.com/fwojciec/diffview/gemini"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClassifier_Classify_ReturnsStoryClassification(t *testing.T) {
	t.Parallel()

	// Arrange: mock Gemini response returning a valid StoryClassification
	expectedClassification := diffview.StoryClassification{
		ChangeType: "bugfix",
		Narrative:  "cause-effect",
		Summary:    "Fix token expiry handling",
		Sections: []diffview.Section{
			{
				Role:        "fix",
				Title:       "Token Validation",
				Explanation: "Adds expiry check before validation",
				Hunks: []diffview.HunkRef{
					{File: "auth.go", HunkIndex: 0, Category: "core", Collapsed: false},
				},
			},
		},
	}
	responseJSON, err := json.Marshal(expectedClassification)
	require.NoError(t, err)

	mockClient := &gemini.MockGenerativeClient{
		GenerateContentFn: func(ctx context.Context, model string, contents []*gemini.Content, config *gemini.GenerateContentConfig) (*gemini.GenerateContentResponse, error) {
			return &gemini.GenerateContentResponse{Text: string(responseJSON)}, nil
		},
	}

	classifier := gemini.NewClassifier(mockClient, gemini.DefaultModel)
	input := diffview.ClassificationInput{
		Repo: "test",
		Commits: []diffview.CommitBrief{
			{Hash: "abc123", Message: "Fix token expiry"},
		},
		Diff: diffview.Diff{
			Files: []diffview.FileDiff{
				{
					NewPath:   "auth.go",
					Operation: diffview.FileModified,
					Hunks: []diffview.Hunk{
						{
							OldStart: 10,
							OldCount: 5,
							NewStart: 10,
							NewCount: 8,
							Lines: []diffview.Line{
								{Type: diffview.LineAdded, Content: "+if expired { return err }"},
							},
						},
					},
				},
			},
		},
	}

	// Act
	result, err := classifier.Classify(context.Background(), input)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "bugfix", result.ChangeType)
	assert.Equal(t, "cause-effect", result.Narrative)
	assert.Equal(t, "Fix token expiry handling", result.Summary)
	require.Len(t, result.Sections, 1)
	assert.Equal(t, "fix", result.Sections[0].Role)
}

func TestClassifier_Classify_PropagatesAPIError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("API rate limit exceeded")
	mockClient := &gemini.MockGenerativeClient{
		GenerateContentFn: func(ctx context.Context, model string, contents []*gemini.Content, config *gemini.GenerateContentConfig) (*gemini.GenerateContentResponse, error) {
			return nil, expectedErr
		},
	}

	classifier := gemini.NewClassifier(mockClient, gemini.DefaultModel)
	input := diffview.ClassificationInput{
		Commits: []diffview.CommitBrief{{Message: "test"}},
	}

	_, err := classifier.Classify(context.Background(), input)

	require.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

func TestClassifier_Classify_ReturnsErrorOnInvalidJSON(t *testing.T) {
	t.Parallel()

	mockClient := &gemini.MockGenerativeClient{
		GenerateContentFn: func(ctx context.Context, model string, contents []*gemini.Content, config *gemini.GenerateContentConfig) (*gemini.GenerateContentResponse, error) {
			return &gemini.GenerateContentResponse{Text: "not valid json"}, nil
		},
	}

	classifier := gemini.NewClassifier(mockClient, gemini.DefaultModel)
	input := diffview.ClassificationInput{
		Commits: []diffview.CommitBrief{{Message: "test"}},
	}

	_, err := classifier.Classify(context.Background(), input)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse")
}

func TestClassifier_Classify_ReturnsErrorOnNilResponse(t *testing.T) {
	t.Parallel()

	mockClient := &gemini.MockGenerativeClient{
		GenerateContentFn: func(ctx context.Context, model string, contents []*gemini.Content, config *gemini.GenerateContentConfig) (*gemini.GenerateContentResponse, error) {
			return nil, nil
		},
	}

	classifier := gemini.NewClassifier(mockClient, gemini.DefaultModel)
	input := diffview.ClassificationInput{
		Commits: []diffview.CommitBrief{{Message: "test"}},
	}

	_, err := classifier.Classify(context.Background(), input)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil response")
}

func TestBuildClassificationPrompt_IncludesFormattedInput(t *testing.T) {
	t.Parallel()

	formattedInput := `<commit_message>
Test message
</commit_message>

<diff>
=== FILE: test.go (modified) ===
</diff>`

	prompt := gemini.BuildClassificationPrompt(formattedInput)

	assert.Contains(t, prompt, "<commit_message>")
	assert.Contains(t, prompt, "Test message")
	assert.Contains(t, prompt, "<diff>")
	assert.Contains(t, prompt, "test.go (modified)")
}

func TestBuildClassificationPrompt_IncludesInstructions(t *testing.T) {
	t.Parallel()

	prompt := gemini.BuildClassificationPrompt("test input")

	// Check key instruction elements
	assert.Contains(t, prompt, "change_type")
	assert.Contains(t, prompt, "narrative")
	assert.Contains(t, prompt, "cause-effect")
	assert.Contains(t, prompt, "core-periphery")
	assert.Contains(t, prompt, "sections")
	assert.Contains(t, prompt, "hunk_index")
}

func TestBuildClassificationConfig_UsesDefaultTemperature(t *testing.T) {
	t.Parallel()

	config := gemini.BuildClassificationConfig()

	// Temperature should be nil to use Gemini 3's default (1.0)
	// Lower temperatures can cause looping/degraded performance
	assert.Nil(t, config.Temperature)
}

func TestBuildClassificationConfig_SetsThinkingLevel(t *testing.T) {
	t.Parallel()

	config := gemini.BuildClassificationConfig()

	// MEDIUM provides balanced reasoning for code classification
	assert.Equal(t, "MEDIUM", config.ThinkingLevel)
}

func TestBuildClassificationConfig_SetsSystemInstruction(t *testing.T) {
	t.Parallel()

	config := gemini.BuildClassificationConfig()

	require.NotNil(t, config.SystemInstruction)
	require.Len(t, config.SystemInstruction.Parts, 1)
	assert.Contains(t, config.SystemInstruction.Parts[0].Text, "code change analyst")
}

func TestBuildClassificationConfig_SetsJSONResponseType(t *testing.T) {
	t.Parallel()

	config := gemini.BuildClassificationConfig()

	assert.Equal(t, "application/json", config.ResponseMIMEType)
}

func TestClassifier_Classify_TimesOutOnSlowAPI(t *testing.T) {
	t.Parallel()

	// Arrange: mock that blocks longer than timeout
	mockClient := &gemini.MockGenerativeClient{
		GenerateContentFn: func(ctx context.Context, model string, contents []*gemini.Content, config *gemini.GenerateContentConfig) (*gemini.GenerateContentResponse, error) {
			// Block until context is cancelled
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}

	// Use very short timeout for test
	timeout := 10 * time.Millisecond
	classifier := gemini.NewClassifier(mockClient, gemini.DefaultModel, gemini.WithTimeout(timeout))
	input := diffview.ClassificationInput{
		Commits: []diffview.CommitBrief{{Message: "test"}},
	}

	// Act
	_, err := classifier.Classify(context.Background(), input)

	// Assert
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestClassifier_Classify_RespectsCallerContextDeadline(t *testing.T) {
	t.Parallel()

	// Arrange: caller has shorter deadline than classifier timeout
	mockClient := &gemini.MockGenerativeClient{
		GenerateContentFn: func(ctx context.Context, model string, contents []*gemini.Content, config *gemini.GenerateContentConfig) (*gemini.GenerateContentResponse, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}

	// Classifier has long timeout, but caller context is short
	classifier := gemini.NewClassifier(mockClient, gemini.DefaultModel, gemini.WithTimeout(time.Hour))
	input := diffview.ClassificationInput{
		Commits: []diffview.CommitBrief{{Message: "test"}},
	}

	// Caller's context has very short deadline
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	// Act
	_, err := classifier.Classify(ctx, input)

	// Assert: should timeout from caller's context, not classifier's
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestClassifier_Classify_RetriesOnTransientErrors(t *testing.T) {
	t.Parallel()

	expectedClassification := diffview.StoryClassification{
		ChangeType: "feature",
		Narrative:  "core-periphery",
		Summary:    "Add new feature",
		Sections:   []diffview.Section{{Role: "core", Title: "Main", Hunks: nil, Explanation: "test"}},
	}
	responseJSON, err := json.Marshal(expectedClassification)
	require.NoError(t, err)

	callCount := 0
	mockClient := &gemini.MockGenerativeClient{
		GenerateContentFn: func(ctx context.Context, model string, contents []*gemini.Content, config *gemini.GenerateContentConfig) (*gemini.GenerateContentResponse, error) {
			callCount++
			if callCount <= 2 {
				// Fail with retryable error first two times
				return nil, gemini.NewAPIError(503, "service unavailable")
			}
			return &gemini.GenerateContentResponse{Text: string(responseJSON)}, nil
		},
	}

	classifier := gemini.NewClassifier(mockClient, gemini.DefaultModel,
		gemini.WithRetry(3, 1*time.Millisecond, 10*time.Millisecond))
	input := diffview.ClassificationInput{
		Commits: []diffview.CommitBrief{{Message: "test"}},
	}

	result, err := classifier.Classify(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, 3, callCount, "should have retried twice after initial failure")
	assert.Equal(t, "feature", result.ChangeType)
}

func TestClassifier_Classify_DoesNotRetryNonRetryableErrors(t *testing.T) {
	t.Parallel()

	callCount := 0
	mockClient := &gemini.MockGenerativeClient{
		GenerateContentFn: func(ctx context.Context, model string, contents []*gemini.Content, config *gemini.GenerateContentConfig) (*gemini.GenerateContentResponse, error) {
			callCount++
			return nil, gemini.NewAPIError(400, "bad request")
		},
	}

	classifier := gemini.NewClassifier(mockClient, gemini.DefaultModel,
		gemini.WithRetry(3, 1*time.Millisecond, 10*time.Millisecond))
	input := diffview.ClassificationInput{
		Commits: []diffview.CommitBrief{{Message: "test"}},
	}

	_, err := classifier.Classify(context.Background(), input)

	require.Error(t, err)
	assert.Equal(t, 1, callCount, "should not retry non-retryable errors")
}

func TestClassifier_Classify_FailsAfterMaxRetries(t *testing.T) {
	t.Parallel()

	callCount := 0
	mockClient := &gemini.MockGenerativeClient{
		GenerateContentFn: func(ctx context.Context, model string, contents []*gemini.Content, config *gemini.GenerateContentConfig) (*gemini.GenerateContentResponse, error) {
			callCount++
			return nil, gemini.NewAPIError(429, "rate limited")
		},
	}

	classifier := gemini.NewClassifier(mockClient, gemini.DefaultModel,
		gemini.WithRetry(3, 1*time.Millisecond, 10*time.Millisecond))
	input := diffview.ClassificationInput{
		Commits: []diffview.CommitBrief{{Message: "test"}},
	}

	_, err := classifier.Classify(context.Background(), input)

	require.Error(t, err)
	assert.Equal(t, 3, callCount, "should try maxRetries times")
	assert.Contains(t, err.Error(), "max retries exceeded")
}

func TestBuildClassificationConfig_SetsResponseSchema(t *testing.T) {
	t.Parallel()

	config := gemini.BuildClassificationConfig()

	require.NotNil(t, config.ResponseSchema)
	assert.Equal(t, "object", config.ResponseSchema.Type)
	require.NotNil(t, config.ResponseSchema.Properties)
	assert.Contains(t, config.ResponseSchema.Properties, "change_type")
	assert.Contains(t, config.ResponseSchema.Properties, "narrative")
	assert.Contains(t, config.ResponseSchema.Properties, "summary")
	assert.Contains(t, config.ResponseSchema.Properties, "sections")
}
