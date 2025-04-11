package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sui_ai_server/internal/ai/prompts"
	"sui_ai_server/internal/utils"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

// GenerateCodeChanges - Specific function for RAG refinement prompt to get code edits.
func (g *Generator) GenerateCodeChanges(ctx context.Context, userQuery string, contextFiles string) ([]GeneratedFile, error) {
	fullPrompt, ragSystemPrompt := prompts.GetSiteCodeChangePrompt(userQuery, contextFiles)

	req := openai.ChatCompletionRequest{
		Model: openai.GPT4o, // Or Claude 3 Opus, etc.
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: ragSystemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: fullPrompt},
		},
		ResponseFormat: &openai.ChatCompletionResponseFormat{ // Request JSON output
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		},
		MaxTokens:   4096, // Allow ample space for code changes
		Temperature: 0.3,  // Keep temperature low for focused edits
	}

	resp, err := g.client.CreateChatCompletion(ctx, req)

	if err != nil && utils.ShouldRetry(err) {
		log.Printf("OpenAI call for code changes failed, retrying... Error: %v", err)
		time.Sleep(2 * time.Second)
		resp, err = g.client.CreateChatCompletion(ctx, req)
	}

	if err != nil {
		return nil, fmt.Errorf("openai chat completion for code changes failed: %w", err)
	}
	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
		log.Printf("OpenAI usage for failed code change request: %+v", resp.Usage)
		return nil, errors.New("openai returned empty response for code changes")
	}

	// Parse the response (expecting JSON array, possibly wrapped)
	llmOutput := resp.Choices[0].Message.Content
	log.Printf("LLM raw output for code changes: %s", llmOutput)

	var changedFiles []GeneratedFile
	cleanedOutput := strings.TrimSpace(llmOutput)
	cleanedOutput = strings.TrimPrefix(cleanedOutput, "```json")
	cleanedOutput = strings.TrimSuffix(cleanedOutput, "```")
	cleanedOutput = strings.TrimSpace(cleanedOutput)

	err = json.Unmarshal([]byte(cleanedOutput), &changedFiles)
	if err != nil {
		keysToTry := []string{"files", "changes", "result", "code", "output"}
		parsed := false
		for _, key := range keysToTry {
			var wrapper map[string]json.RawMessage
			errWrapper := json.Unmarshal([]byte(cleanedOutput), &wrapper)
			if errWrapper == nil {
				if rawFiles, ok := wrapper[key]; ok {
					errInner := json.Unmarshal(rawFiles, &changedFiles)
					if errInner == nil { // Allow empty array if LLM correctly returns it
						log.Printf("Parsed LLM output for code changes assuming wrapped array structure with key '%s'.", key)
						err = nil // Clear the original array parsing error
						parsed = true
						break
					}
				}
			}
		}
		if !parsed {
			log.Printf("Failed to parse LLM JSON output for code changes. Original array error: %v. Cleaned output: %s", err, cleanedOutput)
			return nil, fmt.Errorf("failed to parse LLM JSON output for code changes: %w", err)
		}
	}

	log.Printf("LLM suggested %d file changes/additions.", len(changedFiles))

	return changedFiles, nil
}
