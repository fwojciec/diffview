package gemini

import (
	"context"

	"google.golang.org/genai"
)

// DefaultModel is the recommended Gemini model for story generation.
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

	result, err := c.client.Models.GenerateContent(ctx, model, genaiContents, genaiConfig)
	if err != nil {
		return nil, err
	}

	return &GenerateContentResponse{Text: result.Text()}, nil
}

// Compile-time check that Client implements GenerativeClient.
var _ GenerativeClient = (*Client)(nil)
