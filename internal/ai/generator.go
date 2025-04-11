package ai

import (

	// Added for determineFileType

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
