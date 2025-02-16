package claude

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"caia-ai-cli/pkg/config"
)

// Client wraps the Anthropic API client
type Client struct {
	client *anthropic.Client
}

// NewClient creates a new Claude client
func NewClient() (*Client, error) {
	apiKey, err := config.GetAnthropicAPIKey()
	if err != nil {
		return nil, err
	}

	client := anthropic.NewClient(option.WithAPIKey(apiKey))

	return &Client{client: client}, nil
}

// SendMessage sends a message to Claude and returns the response
func (c *Client) SendMessage(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	stream := c.client.Messages.NewStreaming(ctx, anthropic.MessageNewParams{
		Model:     anthropic.F(anthropic.ModelClaude3_5SonnetLatest),
		MaxTokens: anthropic.F(int64(4096)),
		System:    anthropic.F([]anthropic.TextBlockParam{anthropic.NewTextBlock(systemPrompt)}),
		Messages:  anthropic.F([]anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock(userMessage))}),
	})

	var response string
	for stream.Next() {
		event := stream.Current()
		if delta, ok := event.Delta.(anthropic.ContentBlockDeltaEventDelta); ok {
			response += delta.Text
		}
	}

	if err := stream.Err(); err != nil {
		return "", fmt.Errorf("error sending message to Claude: %v", err)
	}

	if response == "" {
		return "", fmt.Errorf("received empty response from Claude")
	}

	return response, nil
}
