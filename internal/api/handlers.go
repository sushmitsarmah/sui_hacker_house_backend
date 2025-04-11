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
	// "sui_ai_server/sui/walrus" // Make sure context is imported

	"github.com/gin-gonic/gin"
)

// APIHandler holds dependencies for API endpoints.
type APIHandler struct {
	aiGenerator *ai.Generator
	// neo4jService   *neo4j.Service
	// walrusDeployer *walrus.Deployer
	// sealClient     *seal.Client
	// ragService     *rag.RAGService
	// suiService     *sui.Service // Service for Sui interactions
	suiNetwork string // Network name (e.g., devnet) for context
}

// NewAPIHandler initializes a new API handler with its dependencies.
func NewAPIHandler(
	aiGen *ai.Generator,
	// neo4jSvc *neo4j.Service,
	// walrusDep *walrus.Deployer,
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
		// walrusDeployer: walrusDep,
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
}

// GET /project/:id/files
// func (h *APIHandler) GetProjectFiles(c *gin.Context) {
// 	projectID := c.Param("id")
// 	if projectID == "" {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Project ID is required in path"})
// 		return
// 	}

// 	log.Printf("Fetching files for project ID: %s", projectID)
// 	files, err := h.neo4jService.GetProjectFiles(c.Request.Context(), projectID)
// 	if err != nil {
// 		log.Printf("Error fetching project files for %s: %v", projectID, err)
// 		if errors.Is(err, neo4j.ErrProjectNotFound) {
// 			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
// 			return
// 		}
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve project files"})
// 		return
// 	}

// 	// If files map is empty and no error, project exists but is empty. Still return 200 OK.
// 	c.JSON(http.StatusOK, files) // files is map[string]string
// }

// POST /project/:id/deploy
// func (h *APIHandler) DeployProject(c *gin.Context) {
// 	projectID := c.Param("id")
// 	var req DeployRequest // Request body might just need wallet, or be empty if ID is in path
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
// 		return
// 	}

// 	// Optional: Validate project ID from path matches body if both present
// 	// if req.ProjectID != "" && req.ProjectID != projectID { ... }

// 	log.Printf("Received manual deploy request for project %s by wallet %s", projectID, req.Wallet)

// 	// --- Ownership / Permission Check ---
// 	// TODO: Implement NFT ownership check using Sui SDK/RPC via h.suiService
// 	// 1. Determine required NFT type based on project or system settings.
// 	// 2. Call h.suiService.CheckNFTOwnership(ctx, req.Wallet, requiredNFTType)
// 	// Handle errors and forbidden status.
// 	log.Printf("WARN: Skipping NFT ownership check for deployment of project %s.", projectID) // Placeholder

// 	// --- Retrieve Files ---
// 	// files, err := h.neo4jServmap[string][string]{}ice.GetProjectFiles(c.Request.Context(), projectID)
// 	files := map[string]string{}
// 	err := errors.New("this is an error message")
// 	if err != nil {
// 		log.Printf("Error fetching project files for deployment %s: %v", projectID, err)
// 		if errors.Is(err, neo4j.ErrProjectNotFound) {
// 			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found, cannot deploy"})
// 			return
// 		}
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve project files for deployment"})
// 		return
// 	}
// 	if len(files) == 0 {
// 		log.Printf("Project %s has no files, cannot deploy.", projectID)
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Project contains no files to deploy"})
// 		return
// 	}

// 	// --- Trigger Deployment ---
// 	cid, err := h.walrusDeployer.DeployFiles(c.Request.Context(), files)
// 	if err != nil {
// 		log.Printf("Error deploying project %s to Walrus: %v", projectID, err)
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deploy project to Walrus"})
// 		return
// 	}
// 	log.Printf("Project %s deployed successfully. CID: %s", projectID, cid)

// 	// --- Register Access Control ---
// 	policyName := fmt.Sprintf("project-%s-access", projectID)
// 	// Define NFT criteria based on your system's logic (e.g., ownership of the subscription NFT)
// 	nftCriteria := map[string]interface{}{
// 		"groupLogic":    "owner_of",                                 // Placeholder logic
// 		"nftIdentifier": "YOUR_SEAL_NFT_COLLECTION_ID_OR_POLICY_ID", // Replace with actual Seal config
// 		// Add chain, contract address etc. as required by Seal
// 		"chain": fmt.Sprintf("sui-%s", h.suiNetwork), // Example: "sui-devnet"
// 		// "contractAddress": "YOUR_SUBSCRIPTION_NFT_CONTRACT_ADDRESS",
// 	}
// 	err = h.sealClient.RegisterPolicy(c.Request.Context(), policyName, cid, nftCriteria)
// 	if err != nil {
// 		// Log warning, but deployment succeeded. Don't fail the request here unless Seal is critical.
// 		log.Printf("WARN: Deployment succeeded (CID: %s), but failed to register Seal policy for project %s: %v", cid, projectID, err)
// 	} else {
// 		log.Printf("Seal access policy '%s' registered for CID %s", policyName, cid)
// 	}

// 	// --- Update Database ---
// 	// err = h.neo4jService.UpdateProjectCID(c.Request.Context(), projectID, cid)
// 	// if err != nil {
// 	// 	// Log warning, but main operations succeeded.
// 	// 	log.Printf("WARN: Failed to update project %s with CID %s in Neo4j after deployment: %v", projectID, cid, err)
// 	// }

// 	c.JSON(http.StatusOK, DeployResponse{CID: cid})
// }

// GET /access/:cid - Backend check for Seal access (optional)
// func (h *APIHandler) CheckAccess(c *gin.Context) {
// 	cid := c.Param("cid")
// 	wallet := c.Query("wallet") // Wallet address of the viewer
// 	if cid == "" || wallet == "" {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "CID path parameter and wallet query parameter are required"})
// 		return
// 	}

// 	// Optional: Validate CID and wallet format

// 	log.Printf("Checking access for CID %s by wallet %s", cid, wallet)
// 	hasAccess, err := h.sealClient.VerifyAccess(c.Request.Context(), wallet, cid)
// 	if err != nil {
// 		log.Printf("Error verifying Seal access for CID %s, wallet %s: %v", cid, wallet, err)
// 		// Return generic forbidden on error to avoid leaking info
// 		c.JSON(http.StatusForbidden, gin.H{"access": false, "message": "Access denied or verification failed"})
// 		return
// 	}

// 	if !hasAccess {
// 		log.Printf("Access denied for CID %s by wallet %s via Seal", cid, wallet)
// 		c.JSON(http.StatusForbidden, gin.H{"access": false, "message": "Access denied based on policy requirements"})
// 		return
// 	}

// 	log.Printf("Access granted for CID %s by wallet %s via Seal", cid, wallet)
// 	c.JSON(http.StatusOK, gin.H{"access": true})
// }

// POST /rag/:projectId/query - Handler for TEXT-based RAG answers
// func (h *APIHandler) QueryProjectRAG(c *gin.Context) {
// 	projectID := c.Param("projectId")
// 	var req RAGQueryRequest
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
// 		return
// 	}

// 	log.Printf("Received RAG text query for project %s", projectID)

// 	answer, err := h.ragService.QueryProject(c.Request.Context(), projectID, req.Query)
// 	if err != nil {
// 		log.Printf("Error processing RAG text query for project %s: %v", projectID, err)
// 		errMsg := "Failed to process RAG query"
// 		status := http.StatusInternalServerError
// 		if errors.Is(err, neo4j.ErrProjectNotFound) {
// 			errMsg = fmt.Sprintf("Project '%s' not found", projectID)
// 			status = http.StatusNotFound
// 		} // Add checks for other specific errors if needed
// 		c.JSON(status, gin.H{"error": errMsg})
// 		return
// 	}

// 	c.JSON(http.StatusOK, RAGQueryResponse{Answer: answer})
// }

// POST /rag/:projectId/refine - Handler for RAG-based CODE refinement
// func (h *APIHandler) RefineProjectCode(c *gin.Context) {
// 	projectID := c.Param("projectId")
// 	var req RAGQueryRequest // Reuse the same request struct for the query
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
// 		return
// 	}

// 	log.Printf("Received RAG code refinement request for project %s", projectID)

// 	suggestedFiles, err := h.ragService.RefineProjectCode(c.Request.Context(), projectID, req.Query)
// 	if err != nil {
// 		log.Printf("Error processing RAG code refinement for project %s: %v", projectID, err)
// 		errMsg := "Failed to process code refinement request"
// 		status := http.StatusInternalServerError
// 		if errors.Is(err, neo4j.ErrProjectNotFound) {
// 			errMsg = fmt.Sprintf("Project '%s' not found", projectID)
// 			status = http.StatusNotFound
// 		} // Add checks for other specific errors
// 		c.JSON(status, gin.H{"error": errMsg})
// 		return
// 	}

// 	// Return the array of suggested file changes/additions.
// 	// An empty array is a valid success response (no changes needed or no context found).
// 	log.Printf("Returning %d suggested file changes for project %s", len(suggestedFiles), projectID)
// 	c.JSON(http.StatusOK, RefineCodeResponse{Files: suggestedFiles})
// }

// POST /suins/register - Maps a SUINS name to a deployed Project ID/CID
// func (h *APIHandler) RegisterSuins(c *gin.Context) {
// 	var req RegisterSuinsRequest
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		// Use Gin's binding errors for more specific feedback
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
// 		return
// 	}

// 	log.Printf("Received request to map SUINS '%s' to project '%s' for wallet '%s'", req.SuinsName, req.ProjectID, req.Wallet)

// 	// --- Prerequisite Checks ---
// 	if h.suiService == nil {
// 		log.Printf("ERROR: Sui Service is not available (nil). Cannot verify SUINS ownership.")
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Service unavailable: Cannot connect to Sui network."})
// 		return
// 	}

// 	// Normalize SUINS name
// 	suinsName := strings.ToLower(strings.TrimSpace(req.SuinsName))
// 	// Optional: Add more robust SUINS name validation if rules exist
// 	if suinsName == "" {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "SUINS name cannot be empty"})
// 		return
// 	}

// 	// --- Verification Steps ---
// 	// 1. Verify the requesting wallet owns the SUINS name NFT on Sui
// 	isOwner, err := h.suiService.VerifySuinsOwnership(c.Request.Context(), req.Wallet, suinsName)
// 	if err != nil {
// 		log.Printf("Error verifying SUINS ownership for '%s' by wallet '%s': %v", suinsName, req.Wallet, err)
// 		// Avoid leaking detailed blockchain errors
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify SUINS ownership due to an internal error."})
// 		return
// 	}
// 	if !isOwner {
// 		log.Printf("Verification failed: Wallet '%s' does not own SUINS name '%s'", req.Wallet, suinsName)
// 		c.JSON(http.StatusForbidden, gin.H{"error": fmt.Sprintf("Wallet %s does not own the SUINS name %s", req.Wallet, suinsName)})
// 		return
// 	}
// 	log.Printf("Verification success: Wallet '%s' owns SUINS name '%s'", req.Wallet, suinsName)

// 	// 2. Optional but Recommended: Verify the target Project exists
// 	// This prevents mapping to non-existent projects.
// 	// exists, err := h.neo4jService.CheckProjectExistsRead(c.Request.Context(), req.ProjectID)
// 	exists := false
// 	err = errors.New("this is an error message")
// 	if err != nil {
// 		log.Printf("Error checking existence of project '%s' before SUINS mapping: %v", req.ProjectID, err)
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify target project"})
// 		return
// 	}
// 	if !exists {
// 		log.Printf("Attempt to map SUINS '%s' to non-existent project '%s'", suinsName, req.ProjectID)
// 		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Target project '%s' not found.", req.ProjectID)})
// 		return
// 	}

// 	// Optional: Verify the request wallet owns the project in Neo4j?
// 	// projectDetails, err := h.neo4jService.GetProjectDetails(c.Request.Context(), req.ProjectID)
// 	// if err == nil && projectDetails.Wallet != req.Wallet { ... return forbidden ... }

// 	// --- Store Mapping ---
// 	// err = h.neo4jService.MapSuinsNameToProject(c.Request.Context(), req.ProjectID, suinsName)
// 	err = errors.New("this is an error message")
// 	if err != nil {
// 		log.Printf("Error mapping SUINS '%s' to project '%s' in Neo4j: %v", suinsName, req.ProjectID, err)
// 		if errors.Is(err, neo4j.ErrSuinsNameTaken) {
// 			c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("SUINS name '%s' is already mapped to another project.", suinsName)})
// 			return
// 		}
// 		if errors.Is(err, neo4j.ErrProjectNotFound) {
// 			// Should have been caught by the CheckProjectExistsRead, but handle defensively
// 			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Project '%s' not found.", req.ProjectID)})
// 			return
// 		}
// 		// Generic internal error
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store SUINS mapping."})
// 		return
// 	}

// 	log.Printf("Successfully mapped SUINS '%s' to project '%s'", suinsName, req.ProjectID)
// 	c.JSON(http.StatusOK, RegisterSuinsResponse{
// 		Success: true,
// 		Message: fmt.Sprintf("Successfully mapped SUINS name '%s' to project '%s'.", suinsName, req.ProjectID),
// 	})
// }
