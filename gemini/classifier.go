package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fwojciec/diffview"
)

// Compile-time interface verification.
var _ diffview.StoryClassifier = (*Classifier)(nil)

// DefaultClassifyTimeout is the default timeout for a single classify call.
const DefaultClassifyTimeout = 60 * time.Second

// Classifier implements diffview.StoryClassifier using Google Gemini.
type Classifier struct {
	client    GenerativeClient
	model     string
	formatter diffview.PromptFormatter
	timeout   time.Duration
}

// ClassifierOption configures a Classifier.
type ClassifierOption func(*Classifier)

// WithTimeout sets the timeout for API calls.
func WithTimeout(d time.Duration) ClassifierOption {
	return func(c *Classifier) {
		c.timeout = d
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

	resp, err := c.client.GenerateContent(ctx, c.model, contents, config)
	if err != nil {
		return nil, err
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

// BuildClassificationPrompt creates the user prompt for classification.
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
- cause-effect: problem → fix → test → supporting
- core-periphery: core → supporting → noise
- before-after: old pattern removal → new pattern → test
- rule-instances: rule definition → instances
- entry-implementation: API/entry → implementation → test

Respond with JSON matching this schema:
{
  "change_type": "bugfix|feature|refactor|chore|docs",
  "narrative": "cause-effect|core-periphery|before-after|rule-instances|entry-implementation",
  "summary": "One sentence describing what this change does",
  "sections": [
    {
      "role": "problem|fix|test|core|supporting|rule|exception|integration|cleanup",
      "title": "Human-readable section title",
      "hunks": [
        {
          "file": "path/to/file.go",
          "hunk_index": 0,
          "category": "refactoring|systematic|core|noise",
          "collapsed": false,
          "collapse_text": null
        }
      ],
      "explanation": "Why this section matters in the narrative"
    }
  ]
}

Rules:
- Every hunk from the input must appear in exactly one section
- hunk_index is 0-based within each file
- collapse_text provides a summary when collapsed is true`, formattedInput)
}

// BuildClassificationConfig returns config for classification calls.
func BuildClassificationConfig() *GenerateContentConfig {
	temp := float32(0.3) // Lower temperature for more consistent classification
	return &GenerateContentConfig{
		SystemInstruction: &Content{
			Parts: []*Part{{
				Text: `You are a code change analyst specializing in helping developers understand and review code changes.

Your role is to:
1. Classify the type of change (bugfix, feature, refactor, etc.)
2. Identify the narrative pattern that best explains the change
3. Organize hunks into logical sections that tell a coherent story
4. Categorize each hunk by its role in the change

Be precise and consistent. Focus on helping a reviewer quickly understand the change.`,
			}},
		},
		Temperature:      &temp,
		ResponseMIMEType: "application/json",
	}
}
