package utils

import (
	"log"
	"os"
	"path/filepath"
	"sui_ai_server/internal/types"
	"sui_ai_server/internal/utils"
)

func SaveFilesDisk(projectID string, generatedFiles []types.GeneratedFile) {
	filesCount := 0
	for _, fileData := range generatedFiles {
		fileType := fileData.Type
		if fileType == "" {
			fileType = utils.DetermineFileType(fileData.Filename) // Fallback
		}

		// Create the full directory path within the tmp directory
		fullDirPath := filepath.Join("tmp", filepath.Dir(fileData.Filename))
		if err := os.MkdirAll(fullDirPath, os.ModePerm); err != nil {
			log.Printf("Failed to create directory path: %v", err)
			continue
		}

		// Construct the full file path
		filePath := filepath.Join("tmp", fileData.Filename)

		// Write the file content
		if err := os.WriteFile(filePath, []byte(fileData.Content), 0644); err != nil {
			log.Printf("Failed to write file %s: %v", filePath, err)
			continue
		}

		log.Printf("File saved: %s", filePath)
		filesCount++
	}
	log.Printf("Successfully stored project %s: %d files created", projectID, filesCount)
	if filesCount != len(generatedFiles) {
		log.Printf("WARN: Mismatch between parsed files (%d) and stored files (%d) for project %s.", len(generatedFiles), filesCount, projectID)
	}
}

func SaveToRAG(projectID string, generatedFiles []types.GeneratedFile) {
	filesCount := 0
	embeddingsCount := 0
	// 4. Create Project node in Neo4j
	// err = g.neo4jService.CreateProject(ctx, projectID, walletAddress, userPrompt)
	// if err != nil {
	// 	return "", fmt.Errorf("failed to create project node in Neo4j: %w", err)
	// }

	// 5. Create File nodes and Embeddings in Neo4j
	// filesCount := 0
	// embeddingsCount := 0
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
}
