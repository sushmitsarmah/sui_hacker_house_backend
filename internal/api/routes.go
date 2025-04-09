package api

import (
	"net/http" // Import net/http

	"github.com/gin-gonic/gin"
)

// RegisterRoutes sets up the API endpoints and groups them logically.
func RegisterRoutes(router *gin.Engine, h *APIHandler) {

	// --- Project Lifecycle ---
	// Group related project actions under /project
	projectGroup := router.Group("/project")
	{
		projectGroup.POST("/generate", h.GenerateSite) // Generate a new project from a prompt
		// projectGroup.GET("/:id/files", h.GetProjectFiles) // Get the files for a specific project
		// projectGroup.POST("/:id/deploy", h.DeployProject) // Trigger deployment for a specific project
	}

	// --- RAG (Retrieval-Augmented Generation) Endpoints ---
	// Group RAG actions under /rag/:projectId
	// ragGroup := router.Group("/rag/:projectId")
	// {
	// 	ragGroup.POST("/query", h.QueryProjectRAG)    // Get a text-based answer about the project code
	// 	ragGroup.POST("/refine", h.RefineProjectCode) // Get code modification suggestions for the project
	// }

	// --- SUINS (Sui Name Service) Integration ---
	// Group SUINS actions under /suins
	// suinsGroup := router.Group("/suins")
	// {
	// 	suinsGroup.POST("/register", h.RegisterSuins) // Register (map) a SUINS name to a project
	// 	// Optional future endpoint:
	// 	// GET /suins/{name} -> Find project details by SUINS name
	// 	// suinsGroup.GET("/:name", h.GetProjectBySuins) // Needs handler implementation in api/handlers.go and likely neo4j/graph.go
	// }

	// --- Access Control & Utilities ---
	// Endpoint for backend-based access check using Seal (less common than client-side check)
	// router.GET("/access/:cid", h.CheckAccess) // Requires ?wallet=<address> query parameter

	// --- Simple Health Check ---
	// Basic health endpoint to check if the service is running
	router.GET("/health", func(c *gin.Context) {
		// TODO: Implement deeper health checks:
		// - Neo4j connectivity (e.g., ping or simple query)
		// - AI client status (if possible)
		// - Sui RPC connectivity
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

}
