package ai

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sui_ai_server/internal/utils"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

// GenerateWithContext is useful for pure Q&A RAG where the answer is text.
func (g *Generator) GenerateWithContext(ctx context.Context, systemPrompt string, userPrompt string, contextText string) (string, error) {
	fullUserPrompt := fmt.Sprintf("User Query: %s\n\nRelevant Context from Project Files:\n%s", userPrompt, contextText)

	resp, err := g.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: openai.GPT4o, // Or preferred model
			Messages: []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
				{Role: openai.ChatMessageRoleUser, Content: fullUserPrompt},
			},
			MaxTokens:   1500,
			Temperature: 0.7,
		},
	)

	if err != nil && utils.ShouldRetry(err) {
		log.Printf("OpenAI text generation with context failed, retrying... Error: %v", err)
		time.Sleep(1 * time.Second)
		// resp, err = g.client.CreateChatCompletion(ctx)
	}

	if err != nil {
		return "", fmt.Errorf("openai chat completion with context failed: %w", err)
	}

	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
		log.Printf("OpenAI usage for failed context query: %+v", resp.Usage)
		return "", errors.New("openai returned empty response for context query")
	}

	return resp.Choices[0].Message.Content, nil
}
