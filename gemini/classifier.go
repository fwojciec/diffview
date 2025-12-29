package gemini

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"time"

	"github.com/fwojciec/diffview"
)

// Compile-time interface verification.
var _ diffview.StoryClassifier = (*Classifier)(nil)

// DefaultClassifyTimeout is the default timeout for a single classify call.
const DefaultClassifyTimeout = 60 * time.Second

// Classifier implements diffview.StoryClassifier using Google Gemini.
type Classifier struct {
	client       GenerativeClient
	model        string
	formatter    diffview.PromptFormatter
	timeout      time.Duration
	maxRetries   int
	baseDelay    time.Duration
	maxDelay     time.Duration
	retryEnabled bool
}

// ClassifierOption configures a Classifier.
type ClassifierOption func(*Classifier)

// WithTimeout sets the timeout for API calls.
func WithTimeout(d time.Duration) ClassifierOption {
	return func(c *Classifier) {
		c.timeout = d
	}
}

// WithRetry enables retry logic with exponential backoff.
// maxRetries is the maximum number of attempts (including the first).
// baseDelay is the initial delay between retries.
// maxDelay is the maximum delay between retries.
func WithRetry(maxRetries int, baseDelay, maxDelay time.Duration) ClassifierOption {
	return func(c *Classifier) {
		c.maxRetries = maxRetries
		c.baseDelay = baseDelay
		c.maxDelay = maxDelay
		c.retryEnabled = true
	}
}

// NewClassifier creates a new Classifier.
func NewClassifier(client GenerativeClient, model string, opts ...ClassifierOption) *Classifier {
	c := &Classifier{
		client:    client,
		model:     model,
		formatter: &diffview.DefaultFormatter{},
		timeout:   DefaultClassifyTimeout,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Classify produces a StoryClassification from classification input.
func (c *Classifier) Classify(ctx context.Context, input diffview.ClassificationInput) (*diffview.StoryClassification, error) {
	// Apply timeout to context
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	formattedInput := c.formatter.Format(input)
	prompt := BuildClassificationPrompt(formattedInput)

	contents := []*Content{{
		Parts: []*Part{{Text: prompt}},
	}}

	config := BuildClassificationConfig()

	var resp *GenerateContentResponse
	var lastErr error

	maxAttempts := 1
	if c.retryEnabled {
		maxAttempts = c.maxRetries
	}

	for attempt := range maxAttempts {
		resp, lastErr = c.client.GenerateContent(ctx, c.model, contents, config)
		if lastErr == nil {
			break
		}

		if !c.isRetryable(lastErr) {
			return nil, lastErr
		}

		if attempt < maxAttempts-1 {
			delay := c.backoffDelay(attempt)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
				// Continue retry loop
			}
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("gemini: max retries exceeded: %w", lastErr)
	}
	if resp == nil {
		return nil, fmt.Errorf("gemini: returned nil response")
	}

	var classification diffview.StoryClassification
	if err := json.Unmarshal([]byte(resp.Text), &classification); err != nil {
		return nil, fmt.Errorf("gemini: failed to parse response: %w", err)
	}

	return &classification, nil
}

// isRetryable determines if an error should trigger a retry.
// Retryable errors: 429 (rate limit), 500 (server error), 503 (unavailable).
func (c *Classifier) isRetryable(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case 429, 500, 503:
			return true
		}
	}
	return false
}

// backoffDelay calculates exponential backoff delay with jitter.
func (c *Classifier) backoffDelay(attempt int) time.Duration {
	baseMs := float64(c.baseDelay.Milliseconds())
	maxMs := float64(c.maxDelay.Milliseconds())
	delay := math.Min(baseMs*math.Pow(2, float64(attempt)), maxMs)
	jitter := rand.Float64() * baseMs * 0.3
	return time.Duration(delay+jitter) * time.Millisecond
}

// BuildClassificationPrompt creates the user prompt for classification.
// Note: JSON schema is provided via ResponseSchema, not in the prompt (per Google's recommendation).
func BuildClassificationPrompt(formattedInput string) string {
	return fmt.Sprintf(`Analyze this code change and classify it into a structured narrative.

%s

## Why Narrative Structure Matters

Code reviews are cognitively demanding. Research shows that developers process changes more effectively when presented as stories rather than lists. Each narrative follows a three-act structure:

- **Exposition**: Context and setup (what exists, what's the problem)
- **Confrontation**: The change itself (the fix, new feature, transformation)
- **Resolution**: Validation and cleanup (tests proving it works, supporting changes)

## Classifying the Change

Determine the **change_type** (bugfix, feature, refactor, chore, docs) and select a **narrative** that best tells the story:

1. **Is it fixing a bug or issue?** (change_type: bugfix) → cause-effect
   - Shows the problem, then the fix, then proof it works
   - Exposition: the buggy code (problem)
   - Confrontation: the fix
   - Resolution: tests validating the fix

2. **Is it replacing an old pattern with a new one?** (change_type: refactor) → before-after
   - Shows the transformation from old to new
   - Exposition: what's being removed (cleanup)
   - Confrontation: the new pattern (core)
   - Resolution: tests proving the new pattern works

3. **Is it adding a new API/interface with implementation?** (change_type: feature) → entry-implementation
   - Shows the contract first, then the implementation
   - Exposition: the interface/API (interface)
   - Confrontation: the implementation (core)
   - Resolution: tests and supporting changes

4. **Is it applying the same pattern in multiple places?** (change_type: refactor) → rule-instances
   - Shows the pattern, then its applications
   - Exposition: the pattern (pattern)
   - Confrontation: applications of the pattern (core)
   - Resolution: tests validating the applications

5. **Otherwise (feature, enhancement, general change)?** (change_type: feature/chore/docs) → core-periphery
   - Shows the central change and its ripple effects
   - Exposition: the core change (core)
   - Confrontation: supporting updates (supporting)
   - Resolution: tests and cleanup

## Section Ordering: Two-Pass Process

The array order in your output determines reading order. Follow this two-pass approach:

### Pass 1: Narrative-Driven Ordering
Start with the standard ordering for your chosen narrative:
- cause-effect: problem → fix → test → supporting → cleanup
- core-periphery: core → supporting → test → cleanup
- before-after: cleanup (old pattern) → core (new pattern) → supporting → test
- rule-instances: pattern → core → test → supporting → cleanup
- entry-implementation: interface → core → test → supporting → cleanup

Principles for this ordering:
1. **Context before detail**: Show "why" before "what" (exposition before action)
2. **High-impact first**: Core changes before peripheral ones
3. **Tests as validation**: Tests belong near the end as proof (resolution/denouement)

### Pass 2: Sink Fully-Collapsed Sections
After establishing narrative order, identify sections where EVERY hunk is collapsed=true. These are "empty slides" in the story - they contain no visible content for the reviewer.

**Move fully-collapsed sections to the very end**, preserving their relative order. This prevents "empty slides" from interrupting the narrative flow.

Example: If your narrative order produces [problem, fix, cleanup, test] but "cleanup" has all hunks collapsed, the final order should be [problem, fix, test, cleanup].

## Classifying Hunks

For each hunk, determine:
- **category**: refactoring (restructure without behavior change), systematic (mechanical changes like renames), core (essential logic change), noise (formatting, whitespace)
- **collapsed**: whether to collapse in a diff viewer (true for noise, often true for systematic; never collapse tests - they verify intent and are essential for review)

Group hunks into sections with meaningful roles that tell the story of the change.

## Rules
- Every hunk from the input must appear in exactly one section
- hunk_index is 0-based within each file
- collapse_text provides a summary when collapsed is true`, formattedInput)
}

// BuildClassificationConfig returns config for classification calls.
// Note: Temperature is intentionally omitted to use Gemini 3's default (1.0).
// Lower temperatures can cause looping/degraded performance with Gemini 3 models.
func BuildClassificationConfig() *GenerateContentConfig {
	return &GenerateContentConfig{
		SystemInstruction: &Content{
			Parts: []*Part{{
				Text: `You are a code change analyst specializing in helping developers understand and review code changes.

Your role is to:
1. Classify the type of change (bugfix, feature, refactor, etc.)
2. Identify the narrative pattern that best explains the change
3. Organize hunks into logical sections that tell a coherent story
4. Categorize each hunk by its role in the change

When PR title and description are provided, use them to understand the author's intent. The PR description often explains why the change was made and what problem it solves.

Be precise and consistent. Focus on helping a reviewer quickly understand the change.`,
			}},
		},
		ResponseMIMEType: "application/json",
		ResponseSchema:   classificationSchema(),
		ThinkingLevel:    "MEDIUM", // Balanced reasoning for code classification
	}
}

// classificationSchema returns the JSON schema for StoryClassification output.
// Using ResponseSchema instead of prompt-embedded schema improves output quality.
func classificationSchema() *Schema {
	return &Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"change_type": {
				Type:        "string",
				Enum:        []string{"bugfix", "feature", "refactor", "chore", "docs"},
				Description: "Primary classification of the code change",
			},
			"narrative": {
				Type:        "string",
				Enum:        []string{"cause-effect", "core-periphery", "before-after", "rule-instances", "entry-implementation"},
				Description: "The storytelling pattern that best explains this change",
			},
			"summary": {
				Type:        "string",
				Description: "One sentence describing what this change does",
			},
			"sections": {
				Type:        "array",
				Description: "Ordered sections grouping related hunks",
				Items: &Schema{
					Type: "object",
					Properties: map[string]*Schema{
						"role": {
							Type:        "string",
							Enum:        []string{"problem", "fix", "test", "core", "supporting", "pattern", "interface", "cleanup"},
							Description: "The section's role in the narrative",
						},
						"title": {
							Type:        "string",
							Description: "Human-readable section title",
						},
						"explanation": {
							Type:        "string",
							Description: "Why this section matters in the narrative",
						},
						"hunks": {
							Type:        "array",
							Description: "References to hunks in this section",
							Items: &Schema{
								Type: "object",
								Properties: map[string]*Schema{
									"file": {
										Type:        "string",
										Description: "Path to the file",
									},
									"hunk_index": {
										Type:        "integer",
										Description: "0-based hunk index within the file",
									},
									"category": {
										Type:        "string",
										Enum:        []string{"refactoring", "systematic", "core", "noise"},
										Description: "Category of change",
									},
									"collapsed": {
										Type:        "boolean",
										Description: "Whether to collapse in diff viewer",
									},
									"collapse_text": {
										Type:        "string",
										Description: "Summary text when collapsed",
									},
								},
								Required:         []string{"file", "hunk_index", "category", "collapsed"},
								PropertyOrdering: []string{"file", "hunk_index", "category", "collapsed", "collapse_text"},
							},
						},
					},
					Required:         []string{"role", "title", "hunks", "explanation"},
					PropertyOrdering: []string{"role", "title", "hunks", "explanation"},
				},
			},
		},
		Required:         []string{"change_type", "narrative", "summary", "sections"},
		PropertyOrdering: []string{"change_type", "narrative", "summary", "sections"},
	}
}
