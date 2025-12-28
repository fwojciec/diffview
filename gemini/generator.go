package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fwojciec/diffview"
)

// Compile-time interface verification.
var _ diffview.StoryGenerator = (*Generator)(nil)

// Generator implements diffview.StoryGenerator using Google Gemini.
type Generator struct {
	client GenerativeClient
	model  string
}

// NewGenerator creates a new Generator.
func NewGenerator(client GenerativeClient, model string) *Generator {
	return &Generator{client: client, model: model}
}

// Generate creates a DiffAnalysis from annotated hunks.
func (g *Generator) Generate(ctx context.Context, hunks []diffview.AnnotatedHunk) (*diffview.DiffAnalysis, error) {
	prompt := BuildPrompt(hunks)

	contents := []*Content{{
		Parts: []*Part{{Text: prompt}},
	}}

	config := BuildConfig()

	resp, err := g.client.GenerateContent(ctx, g.model, contents, config)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("gemini: returned nil response")
	}

	var analysis diffview.DiffAnalysis
	if err := json.Unmarshal([]byte(resp.Text), &analysis); err != nil {
		return nil, fmt.Errorf("gemini: failed to parse response: %w", err)
	}

	return &analysis, nil
}

// BuildPrompt creates the user prompt for the Gemini API.
func BuildPrompt(hunks []diffview.AnnotatedHunk) string {
	var sb strings.Builder
	sb.WriteString("You are analyzing a git diff to help a human reviewer understand the change.\n\n")
	sb.WriteString("## Hunks\n\n")

	for _, h := range hunks {
		fmt.Fprintf(&sb, "[%s]\n", h.ID)
		for _, line := range h.Hunk.Lines {
			sb.WriteString(line.Content)
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	sb.WriteString("## Task\n\n")
	sb.WriteString("Classify this change and segment the hunks into a narrative structure.\n\n")
	sb.WriteString("Respond with JSON matching this schema:\n")
	sb.WriteString(`{
  "version": 1,
  "analyses": [{
    "type": "story",
    "payload": {
      "changeType": "feature|bugfix|refactor|chore",
      "summary": "One sentence description",
      "parts": [{"role": "core|supporting|test|cleanup", "hunkIDs": ["h1"], "explanation": "..."}]
    }
  }]
}
`)

	return sb.String()
}

// BuildConfig returns the GenerateContentConfig for Gemini API calls.
func BuildConfig() *GenerateContentConfig {
	temp := float32(0.4)
	return &GenerateContentConfig{
		SystemInstruction: &Content{
			Parts: []*Part{{
				Text: `You are a code change analyst. Your role is to analyze git diffs and produce structured narratives that help reviewers understand the change.

Classify the change type (bugfix, feature, refactor, chore) and segment hunks by their role (core, supporting, test, cleanup).`,
			}},
		},
		Temperature:      &temp,
		ResponseMIMEType: "application/json",
	}
}

// GenerativeClient abstracts the Gemini API for testing.
type GenerativeClient interface {
	GenerateContent(ctx context.Context, model string, contents []*Content, config *GenerateContentConfig) (*GenerateContentResponse, error)
}

// Content represents a message in a Gemini conversation.
type Content struct {
	Parts []*Part
}

// Part represents a part of a message.
type Part struct {
	Text string
}

// GenerateContentConfig holds configuration for content generation.
type GenerateContentConfig struct {
	SystemInstruction *Content
	Temperature       *float32
	ResponseMIMEType  string
	ResponseSchema    *Schema
	ThinkingLevel     string // "", "MINIMAL", "LOW", "MEDIUM", "HIGH"
}

// Schema represents the structure for controlled JSON generation.
type Schema struct {
	Type             string             // object, array, string, integer, number, boolean
	Properties       map[string]*Schema // For object types
	Items            *Schema            // For array types
	Enum             []string           // For string enums
	Required         []string           // Required property names
	PropertyOrdering []string           // Order of properties in output
	Description      string             // Field description
}

// GenerateContentResponse holds the response from content generation.
type GenerateContentResponse struct {
	Text string
}

// MockGenerativeClient is a mock implementation of GenerativeClient for testing.
type MockGenerativeClient struct {
	GenerateContentFn func(ctx context.Context, model string, contents []*Content, config *GenerateContentConfig) (*GenerateContentResponse, error)
}

func (m *MockGenerativeClient) GenerateContent(ctx context.Context, model string, contents []*Content, config *GenerateContentConfig) (*GenerateContentResponse, error) {
	return m.GenerateContentFn(ctx, model, contents, config)
}

// APIError represents an error from the Gemini API with HTTP status code.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return e.Message
}

// NewAPIError creates a new APIError with the given status code and message.
func NewAPIError(statusCode int, message string) *APIError {
	return &APIError{StatusCode: statusCode, Message: message}
}
