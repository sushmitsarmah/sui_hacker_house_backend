package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"path/filepath" // Added for determineFileType
	"strings"

	// "sui_ai_server/db/neo4j"
	"time" // Added for potential retries or delays

	"github.com/google/uuid"
	openai "github.com/sashabaranov/go-openai"
)

// GeneratedFile represents the structure expected from the LLM for each file.
type GeneratedFile struct {
	Filename string `json:"filename"`
	Type     string `json:"type"` // e.g., "tsx", "css", "json"
	Content  string `json:"content"`
}

type Generator struct {
	client *openai.Client
	// neo4jService     *neo4j.Service
	embeddingModelID string
}

func NewGenerator(apiKey string, embeddingModel string) *Generator {
	// func NewGenerator(apiKey string, neo4jSvc *neo4j.Service, embeddingModel string) *Generator {
	// Add basic retry logic to the HTTP client used by OpenAI
	// Note: go-openai doesn't directly expose easy retry config on the default client.
	// For robust retries, consider using a library like hashicorp/go-retryablehttp
	// or implementing a custom transport.
	// config := openai.DefaultConfig(apiKey)
	// config.HTTPClient = &http.Client{ ... custom transport ... }
	// client := openai.NewClientWithConfig(config)

	client := openai.NewClient(apiKey) // Sticking with default for now
	return &Generator{
		client: client,
		// neo4jService:     neo4jSvc,
		embeddingModelID: embeddingModel,
	}
}

// Constant for the initial generation prompt template
const initialGenerationPromptTemplate = `You are a full-stack site generator AI.

A user has submitted the following project description:

---
"%s"
---

Please create a **multi-file project** based on the following rules:

1.  **Frontend Framework**: React + TypeScript (Vite)
2.  **Styling**: TailwindCSS, consistent color theme:
    *   Primary: #1A73E8
    *   Accent: #FF6F61
    *   Background: #F9FAFB
    *   Font: Inter, sans-serif
3.  **Layout**: Responsive grid, cards with soft shadows and rounded corners
4.  **Animations**: Use Framer Motion for subtle entry effects on buttons, cards, and modals
5.  **Pages to Include** (at minimum):
    *   ` + "`index.tsx`" + `: landing page with hero section, feature highlights
    *   ` + "`about.tsx`" + `: about the site/project
    *   ` + "`components/Navbar.tsx`" + `, ` + "`Footer.tsx`" + `
    *   ` + "`App.tsx`" + `: wrap routes and layout
    *   ` + "`main.tsx`" + `: app root
    *   ` + "`tailwind.config.ts`" + `: theme customization
    *   ` + "`vite.config.ts`" + `: default Vite config

Respond with a structured array of files in the following format:

` + "```json" + `
[
  {
    "filename": "src/App.tsx",
    "type": "tsx",
    "content": "..."
  },
  {
    "filename": "src/components/Navbar.tsx",
    "type": "tsx",
    "content": "..."
  },
  ...
]
` + "```" + `

Only include code — no extra explanation. Your output will be parsed and saved as project files.`

// GenerateSiteAndStore generates the site, stores it in Neo4j, and returns the project ID.
func (g *Generator) GenerateSiteAndStore(ctx context.Context, userPrompt, walletAddress string) (string, error) {
	projectID := uuid.New().String()
	log.Printf("Generating site for project %s, wallet %s", projectID, walletAddress)

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
	if err != nil && shouldRetry(err) {
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

	// 4. Create Project node in Neo4j
	// err = g.neo4jService.CreateProject(ctx, projectID, walletAddress, userPrompt)
	// if err != nil {
	// 	return "", fmt.Errorf("failed to create project node in Neo4j: %w", err)
	// }

	// 5. Create File nodes and Embeddings in Neo4j
	filesCount := 0
	embeddingsCount := 0
	// for _, fileData := range generatedFiles {
	// 	fileType := fileData.Type
	// 	if fileType == "" {
	// 		fileType = g.determineFileType(fileData.Filename) // Fallback
	// 	}

	// 	fileID, err := g.neo4jService.CreateFile(ctx, projectID, fileData.Filename, fileData.Content, fileType)
	// 	if err != nil {
	// 		log.Printf("ERROR: Failed to create file node '%s' in Neo4j for project %s: %v. Continuing...", fileData.Filename, projectID, err)
	// 		continue // Skip embedding for this failed file
	// 	}
	// 	filesCount++

	// 	embedding, err := g.GenerateEmbedding(ctx, fileData.Content)
	// 	if err != nil {
	// 		log.Printf("WARN: Failed to generate embedding for file %s (ID: %s) in project %s: %v", fileData.Filename, fileID, projectID, err)
	// 		continue // Continue without embedding for this file
	// 	}
	// 	if len(embedding) == 0 {
	// 		log.Printf("WARN: Skipping embedding storage for file %s (ID: %s) due to empty embedding vector.", fileData.Filename, fileID)
	// 		continue
	// 	}

	// 	err = g.neo4jService.CreateEmbedding(ctx, fileID, embedding)
	// 	if err != nil {
	// 		log.Printf("WARN: Failed to store embedding for file %s (ID: %s) in project %s: %v", fileData.Filename, fileID, projectID, err)
	// 		// Continue, but log the issue.
	// 	} else {
	// 		embeddingsCount++
	// 	}
	// }

	log.Printf("Successfully stored project %s: %d files created, %d embeddings stored in Neo4j.", projectID, filesCount, embeddingsCount)
	if filesCount != len(generatedFiles) {
		log.Printf("WARN: Mismatch between parsed files (%d) and stored files (%d) for project %s.", len(generatedFiles), filesCount, projectID)
	}

	return projectID, nil
}

// GenerateEmbedding creates a vector embedding for the given text.
func (g *Generator) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	if g.embeddingModelID == "" {
		return nil, errors.New("embedding model ID is not configured")
	}
	if text == "" {
		// Return empty slice, Neo4j create embedding logic should handle this
		return []float32{}, nil
	}

	model := openai.EmbeddingModel(g.embeddingModelID)
	req := openai.EmbeddingRequest{
		Input: []string{text},
		Model: model,
	}

	resp, err := g.client.CreateEmbeddings(ctx, req)
	// Add retry logic here too if needed
	if err != nil && shouldRetry(err) {
		log.Printf("OpenAI embedding failed, retrying... Error: %v", err)
		time.Sleep(1 * time.Second)
		resp, err = g.client.CreateEmbeddings(ctx, req)
	}

	if err != nil {
		return nil, fmt.Errorf("openai embedding failed: %w", err)
	}

	if len(resp.Data) == 0 || len(resp.Data[0].Embedding) == 0 {
		return nil, errors.New("openai returned empty embedding")
	}

	return resp.Data[0].Embedding, nil
}

// determineFileType provides a fallback if the LLM doesn't specify a type.
func (g *Generator) determineFileType(filename string) string {
	lowerFilename := strings.ToLower(filename)
	ext := filepath.Ext(lowerFilename)
	switch ext {
	case ".html":
		return "HTML"
	case ".css":
		return "CSS"
	case ".js":
		return "JavaScript"
	case ".jsx":
		return "JSX"
	case ".ts":
		return "TypeScript"
	case ".tsx":
		return "TSX"
	case ".json":
		return "JSON"
	case ".md":
		return "Markdown"
	case ".txt":
		return "Text"
	case ".yaml", ".yml":
		return "YAML"
	case ".toml":
		return "TOML"
	case ".sh":
		return "Shell"
	case ".py":
		return "Python"
	case ".go":
		return "Go"
	case ".env":
		return "Env"
	case ".gitignore":
		return "GitIgnore"
	case ".svg":
		return "SVG"
	case ".png", ".jpg", ".jpeg", ".gif", ".webp":
		return "Image" // May not want embeddings for images
	default:
		// Try getting type from common config file names
		base := filepath.Base(lowerFilename)
		if strings.Contains(base, "dockerfile") {
			return "Dockerfile"
		}
		if strings.Contains(base, "vite.config") {
			return "Config"
		} // Generic config
		if strings.Contains(base, "tailwind.config") {
			return "Config"
		}
		if strings.Contains(base, "package.json") {
			return "JSON"
		}
		if strings.Contains(base, "tsconfig.json") {
			return "JSON"
		}

		return "Unknown"
	}
}

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

	if err != nil && shouldRetry(err) {
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

// GenerateCodeChanges - Specific function for RAG refinement prompt to get code edits.
func (g *Generator) GenerateCodeChanges(ctx context.Context, userQuery string, contextFiles string) ([]GeneratedFile, error) {
	ragSystemPrompt := `You are a code assistant helping to **update an existing project**. Respond ONLY with the JSON array containing modified or new files as requested.`
	ragPromptTemplate := `
User's instruction:
---
%s
---

Here are the most relevant existing files from the project:
---
%s
---

Please respond with updated or new files in the following format:
` + "```json" + `
[
  {
    "filename": "src/components/Hero.tsx",
    "type": "tsx",
    "content": "..."
  },
  {
    "filename": "src/components/Testimonials.tsx",
    "type": "tsx",
    "content": "..."
  }
]
` + "```" + `

Only return the modified or newly added files. Do not include duplicates or files that were not changed.`

	fullPrompt := fmt.Sprintf(ragPromptTemplate, userQuery, contextFiles)

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

	if err != nil && shouldRetry(err) {
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

// Simple retry check (customize as needed)
func shouldRetry(err error) bool {
	if err == nil {
		return false
	}
	// Example: Retry on specific transient errors like rate limits or server errors
	errMsg := strings.ToLower(err.Error())
	if strings.Contains(errMsg, "rate limit") ||
		strings.Contains(errMsg, "500 internal server error") ||
		strings.Contains(errMsg, "502 bad gateway") ||
		strings.Contains(errMsg, "503 service unavailable") ||
		strings.Contains(errMsg, "504 gateway timeout") ||
		strings.Contains(errMsg, "timeout") ||
		strings.Contains(errMsg, "connection reset by peer") ||
		strings.Contains(errMsg, "context deadline exceeded") { // Context deadline might indicate temporary overload
		return true
	}
	// Check for specific OpenAI error types if available in the client library
	// var openAIErr *openai.APIError
	// if errors.As(err, &openAIErr) {
	//     if openAIErr.HTTPStatusCode >= 500 || openAIErr.HTTPStatusCode == 429 {
	//         return true
	//     }
	// }
	return false
}
