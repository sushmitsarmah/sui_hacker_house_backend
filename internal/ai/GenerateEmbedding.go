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
	if err != nil && utils.ShouldRetry(err) {
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
