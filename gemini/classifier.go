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

	// Deterministically reorder sections based on narrative type
	classification.OrderSections()

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

## Task

Classify this change and organize the hunks into a coherent narrative structure.

For each hunk, determine:
- **category**: refactoring (restructure without behavior change), systematic (mechanical changes like renames), core (essential logic change), noise (formatting, whitespace)
- **collapsed**: whether it can be collapsed in a diff viewer (true for noise, often true for systematic)

For the overall change, determine:
- **change_type**: bugfix, feature, refactor, chore, docs
- **narrative**: The storytelling pattern that best explains this change:
  - cause-effect: A problem leads to a fix (common for bugfixes)
  - core-periphery: Central change with supporting updates (common for features)
  - before-after: Transformation from old to new pattern (common for refactors)
  - rule-instances: A pattern applied in multiple places
  - entry-implementation: API/interface plus its implementation

Group hunks into sections with meaningful roles that tell the story of the change.

**Order sections to tell a coherent story.** The array order determines the reading order:
- cause-effect: problem → fix → test → supporting → cleanup
- core-periphery: core → supporting → cleanup
- before-after: cleanup (old pattern) → core (new pattern) → test → supporting
- rule-instances: pattern → core → supporting → cleanup
- entry-implementation: interface → core → test → supporting → cleanup

Rules:
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
