package gemini_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

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
		Commit: diffview.CommitInfo{
			Hash:    "abc123",
			Repo:    "test",
			Message: "Fix token expiry",
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
		Commit: diffview.CommitInfo{Message: "test"},
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
		Commit: diffview.CommitInfo{Message: "test"},
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
		Commit: diffview.CommitInfo{Message: "test"},
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

func TestBuildClassificationConfig_SetsLowerTemperature(t *testing.T) {
	t.Parallel()

	config := gemini.BuildClassificationConfig()

	require.NotNil(t, config.Temperature)
	assert.InDelta(t, 0.3, *config.Temperature, 0.001)
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
