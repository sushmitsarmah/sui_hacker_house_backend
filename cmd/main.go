package main

import (
	"context"
	"errors" // Import errors
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	// "github.com/neo4j/neo4j-go-driver/v5/neo4j"

	// Viper included via config package

	"sui_ai_server/config"
	"sui_ai_server/internal/ai"
	"sui_ai_server/internal/api"

	// neo4jRepo "sui_ai_server/db/neo4j" // Alias to avoid name collision
	// "sui_ai_server/events"
	// "sui_ai_server/rag"
	// "sui_ai_server/sui/seal" // Import sui service package
	"sui_ai_server/internal/sui/walrus"
)

func main() {
	// --- Load .env file ---
	// This loads environment variables from a .env file in the current directory
	// or parent directories. It's crucial to do this BEFORE viper loads config.
	err := godotenv.Load()
	if err != nil {
		// It's common for .env to not exist (e.g., in production), so only log a warning
		// if the error is something other than "file not found".
		if !os.IsNotExist(err) {
			log.Printf("Warning: Error loading .env file: %v", err)
		} else {
			log.Println("Info: .env file not found, relying on system environment variables.")
		}
	} else {
		log.Println("Info: Loaded environment variables from .env file.")
	}
	// --- End loading .env ---

	// --- Configuration Loading ---
	cfg, err := config.LoadConfig(".") // Load from config.yaml or env vars
	if err != nil {
		log.Fatalf("Cannot load config: %v", err)
	}

	// --- Dependency Initialization ---
	// _ctx, cancel := context.WithCancel(context.Background())
	// defer cancel()

	// Neo4j Driver
	// driver, err := neo4j.NewDriverWithContext(
	// 	cfg.Neo4jURI,
	// 	neo4j.BasicAuth(cfg.Neo4jUser, cfg.Neo4jPassword, ""),
	// 	// Optional: Configure pool size, timeouts etc.
	// 	func(config *neo4j.Config) {
	// 		config.MaxConnectionPoolSize = 50                      // Example pool size
	// 		config.ConnectionAcquisitionTimeout = 15 * time.Second // Example timeout
	// 	},
	// )
	// if err != nil {
	// 	log.Fatalf("Could not create neo4j driver: %v", err)
	// }
	// defer driver.Close(ctx) // Ensure driver is closed on exit
	// Verify connectivity on startup
	// err = driver.VerifyConnectivity(ctx)
	// Use GetServerInfo for a more comprehensive check if needed
	// _, err = driver.GetServerInfo(ctx)
	// if err != nil {
	// 	log.Fatalf("Neo4j connectivity verification failed: %v", err)
	// }
	// log.Println("Neo4j connection established")

	// Repositories and Services
	// neo4jService := neo4jRepo.NewService(driver)

	// Ensure Neo4j Indexes exist (run once on startup)
	// log.Println("Ensuring Neo4j indexes exist...")
	// if err := neo4jService.SetupIndexes(ctx); err != nil {
	// 	// Log warning but continue - app might function with reduced performance or errors later
	// 	log.Printf("WARN: Failed to setup all Neo4j indexes: %v", err)
	// } else {
	// 	log.Println("Neo4j indexes setup successfully.")
	// }

	// Initialize AI Client (OpenAI or local)
	aiGenerator := ai.NewGenerator(cfg.OpenAIKey, cfg.EmbeddingModelID) // Pass Neo4j service for storage
	// aiGenerator := ai.NewGenerator(cfg.OpenAIKey, neo4jService, cfg.EmbeddingModelID) // Pass Neo4j service for storage

	// Initialize RAG Service
	// ragService := rag.NewRAGService(neo4jService, aiGenerator, cfg.EmbeddingModelID) // AI Generator needed for embeddings

	// Initialize Walrus Deployer
	walrusDeployer := walrus.NewDeployer(cfg.SiteBuilderPath, cfg.WalrusCLIPath) // Add wallet/token logic if needed

	// Initialize Seal Client
	// sealClient := seal.NewClient(cfg.SealAPIKey, cfg.SealEndpoint) // Adjust with actual SDK/API details

	// Initialize Sui Event Listener
	// Ensure the event type string from config is correct
	// eventListener := events.NewSuiEventListener(
	// 	cfg.SuiRPC,
	// 	cfg.SiteDeployedEventType, // Use specific config key
	// 	walrusDeployer,
	// 	sealClient,
	// 	neo4jService,
	// 	cfg.SuiNetwork, // Pass network for context if needed by handlers
	// )

	// Initialize API Handlers (pass all dependencies)
	apiHandler := api.NewAPIHandler(
		aiGenerator,
		// neo4jService,
		walrusDeployer,
		// sealClient,
		// ragService,
		cfg.SuiNetwork,           // Pass network name
		cfg.SuiRPC,               // Pass RPC URL for Sui Service
		cfg.SuinsContractAddress, // Pass SUINS contract address
		cfg.SuinsNftType,         // Pass SUINS NFT type string
	)

	// --- Start Services ---

	// Start Event Listener in a separate goroutine
	// eventListenerActive := true // Assume active unless config/init fails
	// if cfg.SuiRPC == "" || cfg.SiteDeployedEventType == "" {
	// 	log.Println("WARN: Sui RPC endpoint or SiteDeployed Event Type not configured. Event listener will be inactive.")
	// 	eventListenerActive = false
	// }

	// if eventListenerActive {
	// 	go func() {
	// 		log.Println("Starting Sui Event Listener...")
	// 		// Pass the main context to the listener
	// 		listenerErr := eventListener.StartListening(ctx)
	// 		// Only log error if the context wasn't cancelled externally
	// 		if listenerErr != nil && !errors.Is(listenerErr, context.Canceled) {
	// 			log.Printf("ERROR: Sui Event Listener stopped unexpectedly: %v", listenerErr)
	// 			// Consider more robust handling: panic, trigger alert, attempt restart
	// 			// If listener is critical, cancel the main context to stop the app
	// 			cancel()
	// 		} else if errors.Is(listenerErr, context.Canceled) {
	// 			log.Println("Sui Event Listener stopping due to context cancellation.")
	// 		} else {
	// 			log.Println("Sui Event Listener stopped.")
	// 		}
	// 	}()
	// }

	// Start API Server
	// Select Gin mode based on an environment variable or config (e.g., APP_ENV=production)
	appEnv := os.Getenv("APP_ENV")
	if appEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
		log.Println("Running in Gin Debug Mode")
	}

	router := gin.New()        // Use gin.New() for more control over middleware
	router.Use(gin.Logger())   // Add structured logger middleware
	router.Use(gin.Recovery()) // Add panic recovery middleware

	// Configure CORS properly for your frontend origin
	// import "github.com/gin-contrib/cors"
	// config := cors.DefaultConfig()
	// config.AllowOrigins = []string{"http://localhost:3000", "https://your-frontend-domain.com"} // List allowed origins
	// config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	// router.Use(cors.New(config))

	api.RegisterRoutes(router, apiHandler) // Register API endpoints

	server := &http.Server{
		Addr:    cfg.ServerAddress,
		Handler: router,
		// Set timeouts to prevent slow client attacks
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting API server on %s\n", cfg.ServerAddress)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("API server listen error: %s\n", err)
		}
		log.Println("API server has stopped listening.")
	}()

	// --- Graceful Shutdown ---
	quit := make(chan os.Signal, 1) // Buffered channel
	// Notify channel on SIGINT or SIGTERM
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	// Block until a signal is received
	sig := <-quit
	log.Printf("Received signal: %s. Shutting down server...", sig)

	// Create a context with timeout for shutdown
	shutdownCtx, serverCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer serverCancel()

	// Signal background tasks (like event listener) to stop by cancelling the main context
	log.Println("Cancelling main application context...")
	// cancel()

	// Attempt to gracefully shutdown the HTTP server
	log.Println("Shutting down API server...")
	if err := server.Shutdown(shutdownCtx); err != nil {
		// Error from closing listeners, or context timeout:
		log.Printf("API server forced shutdown error: %v", err)
	} else {
		log.Println("API server gracefully stopped.")
	}

	// Optional: Add WaitGroup or similar mechanism to wait for critical goroutines (like listener) to finish cleanup
	// e.g., listener.Wait()

	log.Println("Application exiting.")
}
