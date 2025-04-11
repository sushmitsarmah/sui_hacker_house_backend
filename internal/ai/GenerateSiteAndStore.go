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

	"github.com/google/uuid"
	openai "github.com/sashabaranov/go-openai"
)

// GenerateSiteAndStore generates the site, stores it in Neo4j, and returns the project ID.
func (g *Generator) GenerateSiteAndStore(ctx context.Context, userPrompt, walletAddress string) (string, error) {
	projectID := uuid.New().String()
	log.Printf("Generating site for project %s, wallet %s", projectID, walletAddress)

	initialGenerationPromptTemplate := prompts.GetSiteGenerationPrompt()

	// 1. Construct the prompt using the template
	fullPrompt := fmt.Sprintf(initialGenerationPromptTemplate, userPrompt)

	log.Println("Full prompt for LLM:", fullPrompt) // Log the full prompt for debugging

	// 2. Call the LLM (e.g., OpenAI GPT-4o)
	resp, err := g.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: openai.GPT4oLatest, // Or another suitable model like Claude 3 Opus
			Messages: []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleSystem, Content: "You are a helpful AI assistant that generates code based on user prompts and specific formatting instructions."},
				{Role: openai.ChatMessageRoleUser, Content: fullPrompt},
			},
			// ResponseFormat: &openai.ChatCompletionResponseFormat{
			// 	Type: openai.ChatCompletionResponseFormatTypeJSONObject, // Expect LLM to wrap array in JSON object
			// },
			// MaxTokens:   4096, // Increased max tokens for potentially large codebases
			Temperature: 0.3, // Lower temperature for more predictable code generation
		},
	)

	// Basic retry logic example
	if err != nil && utils.ShouldRetry(err) {
		log.Printf("OpenAI call failed, retrying once after delay... Error: %v", err)
		time.Sleep(2 * time.Second)
		// Recreate the request struct for clarity in retry
		retryReq := openai.ChatCompletionRequest{
			Model: openai.GPT4o,
			Messages: []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleSystem, Content: "You are a helpful AI assistant that generates code based on user prompts and specific formatting instructions."},
				{Role: openai.ChatMessageRoleUser, Content: fullPrompt},
			},
			ResponseFormat: &openai.ChatCompletionResponseFormat{
				Type: openai.ChatCompletionResponseFormatTypeJSONObject,
			},
			MaxTokens:   4096,
			Temperature: 0.3,
		}
		resp, err = g.client.CreateChatCompletion(ctx, retryReq)
	}

	if err != nil {
		return "", fmt.Errorf("openai chat completion failed: %w", err)
	}

	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
		log.Printf("OpenAI usage for failed request: %+v", resp.Usage)
		return "", errors.New("openai returned empty response")
	}

	// 3. Parse the LLM response (expecting JSON array, possibly wrapped)
	llmOutput := resp.Choices[0].Message.Content
	log.Printf("LLM raw output for project %s: %s", projectID, llmOutput) // Log raw output for debugging

	var generatedFiles []GeneratedFile

	cleanedOutput := strings.TrimSpace(llmOutput)
	cleanedOutput = strings.TrimPrefix(cleanedOutput, "```json")
	cleanedOutput = strings.TrimSuffix(cleanedOutput, "```")
	cleanedOutput = strings.TrimSpace(cleanedOutput)

	// Attempt 1: Try parsing as an array (standard case if LLM returns multiple files)
	err = json.Unmarshal([]byte(cleanedOutput), &generatedFiles)
	if err == nil {
		log.Printf("Parsed LLM output as a JSON array for project %s.", projectID)
		// Successfully parsed as an array, proceed.
	} else {
		// If array parsing failed, it might be a single object or a wrapped array.
		log.Printf("Info: Failed to parse as array (%v), trying single object for project %s.", err, projectID)

		// Attempt 2: Try parsing as a single object
		var singleFile GeneratedFile
		errSingle := json.Unmarshal([]byte(cleanedOutput), &singleFile)
		if errSingle == nil {
			log.Printf("Parsed LLM output as a single JSON object for project %s.", projectID)
			// Success! Wrap the single object in a slice.
			generatedFiles = []GeneratedFile{singleFile}
			err = nil // Clear the error from the failed array parse attempt
		} else {
			// If single object parsing also failed, try the wrapped array logic (your original fallback)
			log.Printf("Info: Failed to parse as single object (%v), trying wrapped keys for project %s.", errSingle, projectID)

			// Attempt 3: Try parsing as an object containing the array
			keysToTry := []string{"files", "result", "code", "data", "output"}
			parsedWrapped := false
			for _, key := range keysToTry {
				var wrapper map[string]json.RawMessage
				errWrapper := json.Unmarshal([]byte(cleanedOutput), &wrapper)
				if errWrapper == nil {
					if rawFiles, ok := wrapper[key]; ok {
						// Attempt to unmarshal the inner value (which should be an array)
						errInner := json.Unmarshal(rawFiles, &generatedFiles)
						if errInner == nil && len(generatedFiles) > 0 {
							log.Printf("Parsed LLM output assuming wrapped array structure with key '%s' for project %s.", key, projectID)
							err = nil // Clear previous errors
							parsedWrapped = true
							break
						} else if errInner != nil {
							log.Printf("Debug: Wrapped key '%s' found for project %s, but inner unmarshal failed: %v. Raw inner JSON: %s", key, projectID, errInner, string(rawFiles))
						}
					}
				} else {
					log.Printf("Debug: Failed to unmarshal into wrapper map for project %s: %v", projectID, errWrapper)
				}
			}

			// If none of the attempts (array, single object, wrapped array) worked
			if !parsedWrapped && err != nil { // Keep err from original array attempt or errSingle if that's more relevant
				log.Printf("Failed to parse LLM JSON output for project %s. Array error: %v. Single object error: %v. Cleaned output: %s", projectID, err, errSingle, cleanedOutput)
				// Return or handle the final error - using the original array error 'err' for consistency with old code
				fmt.Printf("Error generating site: %v\n", fmt.Errorf("failed to parse LLM JSON output (tried array, single object, and common wrapped keys): %w", err))
				// return // Exit or return error
			}
		}
	}

	// If we reach here without returning an error, 'generatedFiles' should be populated.
	if err == nil {
		log.Printf("Successfully parsed LLM output for project %s. Number of files: %d", projectID, len(generatedFiles))
		if len(generatedFiles) > 0 {
			fmt.Printf("First file filename: %s\n", generatedFiles[0].Filename)
		}
	} else {
		// This case should ideally be covered by the error handling above, but as a fallback:
		fmt.Printf("An unexpected error occurred during parsing: %v\n", err)
	}

	// ---------------

	if len(generatedFiles) == 0 {
		log.Printf("LLM output parsed, but resulted in zero files for project %s.", projectID)
		return "", errors.New("LLM did not generate any files or parsing failed silently")
	}

	log.Printf("Successfully parsed %d files from LLM for project %s", len(generatedFiles), projectID)

	log.Println(generatedFiles)

	return projectID, nil
}
