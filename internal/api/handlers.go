package api

import (
	// "errors" // Import errors
	// "fmt"
	"log"
	"net/http"

	// "strings"          // Import strings
	"sui_ai_server/internal/ai" // Import ai package
	"sui_ai_server/internal/types"

	// "sui_ai_server/db/neo4j"
	// "sui_ai_server/rag"
	// "sui_ai_server/sui" // NEW: Import sui interaction package
	// "sui_ai_server/sui/seal"
	"sui_ai_server/internal/sui/walrus" // Make sure context is imported

	"github.com/gin-gonic/gin"
)

// APIHandler holds dependencies for API endpoints.
type APIHandler struct {
	aiGenerator *ai.Generator
	// neo4jService   *neo4j.Service
	walrusDeployer *walrus.Deployer
	// sealClient     *seal.Client
	// ragService     *rag.RAGService
	// suiService     *sui.Service // Service for Sui interactions
	suiNetwork string // Network name (e.g., devnet) for context
}

// NewAPIHandler initializes a new API handler with its dependencies.
func NewAPIHandler(
	aiGen *ai.Generator,
	// neo4jSvc *neo4j.Service,
	walrusDep *walrus.Deployer,
	// sealCli *seal.Client,
	// ragSvc *rag.RAGService,
	suiNet string, // Network name (e.g., devnet)
	suiRpcUrl string, // RPC endpoint needed by SuiService
	suinsContractAddr string, // SUINS contract address needed by SuiService
	suinsNftType string, // SUINS NFT type needed by SuiService
) *APIHandler {
	// Initialize the Sui Service here
	// suiSvc, err := sui.NewService(suiRpcUrl, suinsContractAddr, suinsNftType)
	// if err != nil {
	// 	// Log warning and continue - some endpoints might fail if SuiService is nil
	// 	log.Printf("WARN: Failed to initialize Sui Service: %v. SUINS verification and potentially other Sui interactions might fail.", err)
	// 	suiSvc = nil // Explicitly set to nil on error
	// }

	return &APIHandler{
		aiGenerator: aiGen,
		// neo4jService:   neo4jSvc,
		walrusDeployer: walrusDep,
		// sealClient:     sealCli,
		// ragService:     ragSvc,
		// suiService:     suiSvc, // Assign the initialized (or nil) Sui Service
		suiNetwork: suiNet,
	}
}

// --- Structs for API Requests/Responses ---

type GenerateRequest struct {
	Prompt string `json:"prompt" binding:"required"`
	Wallet string `json:"wallet" binding:"required"` // Wallet address of the user
}

type GenerateResponse struct {
	ProjectID string `json:"projectId"`
}

type DeployRequest struct {
	ProjectID string `json:"projectId" binding:"required"`
	Wallet    string `json:"wallet" binding:"required"` // Wallet address confirming ownership/trigger
}

type DeployResponse struct {
	CID string `json:"cid"`
}

type RAGQueryRequest struct {
	Query string `json:"query" binding:"required"`
}

type RAGQueryResponse struct { // For text answers
	Answer string `json:"answer"`
}

type RefineCodeResponse struct { // For code change suggestions
	Files []types.GeneratedFile `json:"files"` // Return the array of file objects
}

type RegisterSuinsRequest struct {
	ProjectID string `json:"projectId" binding:"required"`
	SuinsName string `json:"suinsName" binding:"required,hostname_rfc1123"` // e.g., "mycoolsite.sui" - added basic validation
	Wallet    string `json:"wallet" binding:"required,hexadecimal"`         // Wallet claiming ownership - added basic validation
}

type RegisterSuinsResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// --- API Handlers ---

// POST /project/generate
func (h *APIHandler) GenerateSite(c *gin.Context) {
	var req GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	// Optional: Basic validation for wallet address format?
	// if !isValidSuiAddress(req.Wallet) { ... }

	log.Printf("Received generation request for wallet %s", req.Wallet)

	projectID, err := h.aiGenerator.GenerateSiteAndStore(c.Request.Context(), req.Prompt, req.Wallet)
	if err != nil {
		log.Printf("Error generating site for wallet %s: %v", req.Wallet, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate site"})
		return
	}

	log.Printf("Site generation successful for wallet %s. Project ID: %s", req.Wallet, projectID)
	c.JSON(http.StatusCreated, GenerateResponse{ProjectID: projectID}) // Use 201 Created

	cid, err := h.walrusDeployer.DeployFiles(c.Request.Context())
	if err != nil {
		log.Printf("Error deploying project %s to Walrus: %v", projectID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deploy project to Walrus"})
		return
	}
	log.Printf("Project %s deployed successfully. CID: %s", projectID, cid)

}
