package utils

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/sashabaranov/go-openai"
)

// Simple retry check (customize as needed)
func ShouldRetry(err error) bool {
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
	var openAIErr *openai.APIError
	if errors.As(err, &openAIErr) {
		if openAIErr.HTTPStatusCode >= 500 || openAIErr.HTTPStatusCode == 429 {
			return true
		}
	}
	return false
}

// determineFileType provides a fallback if the LLM doesn't specify a type.
func DetermineFileType(filename string) string {
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
