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

func TestGenerator_Generate_ReturnsStoryAnalysis(t *testing.T) {
	t.Parallel()

	// Arrange: mock Gemini response returning a valid StoryAnalysis
	storyPayload := diffview.StoryAnalysis{
		ChangeType: "feature",
		Summary:    "Adds user authentication",
		Parts: []diffview.StoryPart{
			{Role: "core", HunkIDs: []string{"h1"}, Explanation: "Main auth logic"},
		},
	}
	payloadBytes, err := json.Marshal(storyPayload)
	require.NoError(t, err)

	mockClient := &gemini.MockGenerativeClient{
		GenerateContentFn: func(ctx context.Context, model string, contents []*gemini.Content, config *gemini.GenerateContentConfig) (*gemini.GenerateContentResponse, error) {
			// Return a response with the JSON payload
			response := &diffview.DiffAnalysis{
				Version: 1,
				Analyses: []diffview.Analysis{
					{Type: "story", Payload: payloadBytes},
				},
			}
			responseJSON, _ := json.Marshal(response)
			return &gemini.GenerateContentResponse{Text: string(responseJSON)}, nil
		},
	}

	gen := gemini.NewGenerator(mockClient, gemini.DefaultModel)
	hunks := []diffview.AnnotatedHunk{
		{ID: "h1", Hunk: diffview.Hunk{OldStart: 1, NewStart: 1, Lines: []diffview.Line{{Type: diffview.LineAdded, Content: "+func Auth() {}"}}}},
	}

	// Act
	result, err := gen.Generate(context.Background(), hunks)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 1, result.Version)
	require.Len(t, result.Analyses, 1)
	assert.Equal(t, "story", result.Analyses[0].Type)

	// Verify the payload can be unmarshaled to StoryAnalysis
	var gotStory diffview.StoryAnalysis
	err = json.Unmarshal(result.Analyses[0].Payload, &gotStory)
	require.NoError(t, err)
	assert.Equal(t, "feature", gotStory.ChangeType)
	assert.Equal(t, "Adds user authentication", gotStory.Summary)
}

func TestGenerator_Generate_PropagatesAPIError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("API rate limit exceeded")
	mockClient := &gemini.MockGenerativeClient{
		GenerateContentFn: func(ctx context.Context, model string, contents []*gemini.Content, config *gemini.GenerateContentConfig) (*gemini.GenerateContentResponse, error) {
			return nil, expectedErr
		},
	}

	gen := gemini.NewGenerator(mockClient, gemini.DefaultModel)
	hunks := []diffview.AnnotatedHunk{{ID: "h1"}}

	_, err := gen.Generate(context.Background(), hunks)

	require.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

func TestGenerator_Generate_ReturnsErrorOnInvalidJSON(t *testing.T) {
	t.Parallel()

	mockClient := &gemini.MockGenerativeClient{
		GenerateContentFn: func(ctx context.Context, model string, contents []*gemini.Content, config *gemini.GenerateContentConfig) (*gemini.GenerateContentResponse, error) {
			return &gemini.GenerateContentResponse{Text: "not valid json"}, nil
		},
	}

	gen := gemini.NewGenerator(mockClient, gemini.DefaultModel)
	hunks := []diffview.AnnotatedHunk{{ID: "h1"}}

	_, err := gen.Generate(context.Background(), hunks)

	require.Error(t, err)
}

func TestGenerator_Generate_ReturnsErrorOnNilResponse(t *testing.T) {
	t.Parallel()

	mockClient := &gemini.MockGenerativeClient{
		GenerateContentFn: func(ctx context.Context, model string, contents []*gemini.Content, config *gemini.GenerateContentConfig) (*gemini.GenerateContentResponse, error) {
			return nil, nil
		},
	}

	gen := gemini.NewGenerator(mockClient, gemini.DefaultModel)
	hunks := []diffview.AnnotatedHunk{{ID: "h1"}}

	_, err := gen.Generate(context.Background(), hunks)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil response")
}

func TestBuildPrompt_IncludesHunkIDs(t *testing.T) {
	t.Parallel()

	hunks := []diffview.AnnotatedHunk{
		{ID: "h1", Hunk: diffview.Hunk{Lines: []diffview.Line{{Content: "+added line"}}}},
		{ID: "h2", Hunk: diffview.Hunk{Lines: []diffview.Line{{Content: "-removed line"}}}},
	}

	prompt := gemini.BuildPrompt(hunks)

	assert.Contains(t, prompt, "[h1]")
	assert.Contains(t, prompt, "[h2]")
	assert.Contains(t, prompt, "+added line")
	assert.Contains(t, prompt, "-removed line")
}

func TestBuildPrompt_IncludesTaskInstructions(t *testing.T) {
	t.Parallel()

	hunks := []diffview.AnnotatedHunk{{ID: "h1"}}

	prompt := gemini.BuildPrompt(hunks)

	assert.Contains(t, prompt, "Classify this change")
	assert.Contains(t, prompt, "narrative structure")
	assert.Contains(t, prompt, `"version": 1`)
	assert.Contains(t, prompt, `"type": "story"`)
}

func TestBuildConfig_SetsTemperature(t *testing.T) {
	t.Parallel()

	config := gemini.BuildConfig()

	require.NotNil(t, config.Temperature)
	assert.InDelta(t, 0.4, *config.Temperature, 0.001)
}

func TestBuildConfig_SetsSystemInstruction(t *testing.T) {
	t.Parallel()

	config := gemini.BuildConfig()

	require.NotNil(t, config.SystemInstruction)
	require.Len(t, config.SystemInstruction.Parts, 1)
	assert.Contains(t, config.SystemInstruction.Parts[0].Text, "code change analyst")
}

func TestBuildConfig_SetsJSONResponseType(t *testing.T) {
	t.Parallel()

	config := gemini.BuildConfig()

	assert.Equal(t, "application/json", config.ResponseMIMEType)
}
