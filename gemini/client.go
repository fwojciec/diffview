package gemini

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/genai"
)

// DefaultModel is the recommended Gemini model for story classification.
// gemini-3-flash-preview offers superior code understanding for diff analysis.
const DefaultModel = "gemini-3-flash-preview"

// Client wraps the Gemini genai.Client.
type Client struct {
	client *genai.Client
}

// NewClient creates a new Client with the given API key.
func NewClient(ctx context.Context, apiKey string) (*Client, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		return nil, err
	}
	return &Client{client: client}, nil
}

// Close is a no-op for the new genai SDK (no cleanup needed).
func (c *Client) Close() error {
	return nil
}

// GenerateContent implements GenerativeClient by delegating to the genai.Client.
func (c *Client) GenerateContent(ctx context.Context, model string, contents []*Content, config *GenerateContentConfig) (*GenerateContentResponse, error) {
	// Convert our types to genai types
	genaiContents := make([]*genai.Content, len(contents))
	for i, content := range contents {
		parts := make([]*genai.Part, len(content.Parts))
		for j, part := range content.Parts {
			parts[j] = &genai.Part{Text: part.Text}
		}
		genaiContents[i] = &genai.Content{Parts: parts}
	}

	genaiConfig := &genai.GenerateContentConfig{
		ResponseMIMEType: config.ResponseMIMEType,
	}
	if config.Temperature != nil {
		genaiConfig.Temperature = config.Temperature
	}
	if config.SystemInstruction != nil {
		parts := make([]*genai.Part, len(config.SystemInstruction.Parts))
		for i, part := range config.SystemInstruction.Parts {
			parts[i] = &genai.Part{Text: part.Text}
		}
		genaiConfig.SystemInstruction = &genai.Content{Parts: parts}
	}
	if config.ResponseSchema != nil {
		genaiConfig.ResponseSchema = convertSchema(config.ResponseSchema)
	}
	if config.ThinkingLevel != "" {
		genaiConfig.ThinkingConfig = &genai.ThinkingConfig{
			ThinkingLevel: genai.ThinkingLevel(config.ThinkingLevel),
		}
	}

	result, err := c.client.Models.GenerateContent(ctx, model, genaiContents, genaiConfig)
	if err != nil {
		return nil, wrapAPIError(err)
	}

	return &GenerateContentResponse{Text: result.Text()}, nil
}

// wrapAPIError converts genai.APIError to our APIError type for retry handling.
func wrapAPIError(err error) error {
	var apiErr *genai.APIError
	if errors.As(err, &apiErr) {
		return &APIError{
			StatusCode: apiErr.Code,
			Message:    fmt.Sprintf("gemini API error (HTTP %d): %s", apiErr.Code, apiErr.Message),
		}
	}
	return err
}

// convertSchema recursively converts our Schema to genai.Schema.
func convertSchema(s *Schema) *genai.Schema {
	if s == nil {
		return nil
	}
	gs := &genai.Schema{
		Type:             genai.Type(s.Type),
		Enum:             s.Enum,
		Required:         s.Required,
		PropertyOrdering: s.PropertyOrdering,
		Description:      s.Description,
	}
	if s.Properties != nil {
		gs.Properties = make(map[string]*genai.Schema, len(s.Properties))
		for k, v := range s.Properties {
			gs.Properties[k] = convertSchema(v)
		}
	}
	if s.Items != nil {
		gs.Items = convertSchema(s.Items)
	}
	return gs
}

// Compile-time check that Client implements GenerativeClient.
var _ GenerativeClient = (*Client)(nil)
