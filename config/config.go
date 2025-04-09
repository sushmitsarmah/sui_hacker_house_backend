package config

import (
	"fmt"
	"log" // Import log

	"github.com/spf13/viper"
)

// Config holds all configuration for the application.
// Mapstructure tags are used to map environment variables and config file keys.
type Config struct {
	// Server Configuration
	ServerAddress string `mapstructure:"SERVER_ADDRESS"` // e.g., ":8080"

	// Neo4j Configuration
	Neo4jURI      string `mapstructure:"NEO4J_URI"`      // e.g., "neo4j://localhost:7687" or "neo4j+s://instance.databases.neo4j.io"
	Neo4jUser     string `mapstructure:"NEO4J_USER"`     // e.g., "neo4j"
	Neo4jPassword string `mapstructure:"NEO4J_PASSWORD"` // Database user password

	// AI Configuration
	OpenAIKey        string `mapstructure:"OPENAI_API_KEY"`     // API key for OpenAI
	EmbeddingModelID string `mapstructure:"EMBEDDING_MODEL_ID"` // e.g., "text-embedding-ada-002", "text-embedding-3-small"

	// Deployment Tools Configuration
	SiteBuilderPath string `mapstructure:"SITE_BUILDER_PATH"` // Path to the site-builder executable
	WalrusCLIPath   string `mapstructure:"WALRUS_CLI_PATH"`   // Path to the walrus CLI executable

	// Seal Access Control Configuration
	SealAPIKey   string `mapstructure:"SEAL_API_KEY"`  // API key for Seal service
	SealEndpoint string `mapstructure:"SEAL_ENDPOINT"` // API endpoint for Seal service (e.g., "https://api.seal.xyz")

	// Sui Blockchain Configuration
	SuiRPC                string `mapstructure:"SUI_RPC_ENDPOINT"`             // Sui network RPC endpoint URL
	SuiNetwork            string `mapstructure:"SUI_NETWORK"`                  // Network identifier (e.g., "devnet", "testnet", "mainnet")
	SiteDeployedEventType string `mapstructure:"SUI_SITE_DEPLOYED_EVENT_TYPE"` // Full event type string (e.g., "0xPKG::MODULE::SiteDeployed")

	// SUINS Integration Configuration
	SuinsContractAddress string `mapstructure:"SUINS_CONTRACT_ADDRESS"` // Package/Object ID of the SUINS registry contract
	SuinsNftType         string `mapstructure:"SUINS_NFT_TYPE"`         // Full NFT Type string for SUINS ownership (e.g., "0xPKG::suins::Suins")
}

// LoadConfig reads configuration from file and environment variables.
func LoadConfig(path string) (config Config, err error) {
	viper.AddConfigPath(path)     // Path to look for the config file in
	viper.SetConfigName("config") // Name of config file (without extension)
	viper.SetConfigType("yaml")   // REQUIRED if the config file does not have the extension in the name

	viper.AutomaticEnv() // Read environment variables that match keys

	// Attempt to read the config file
	err = viper.ReadInConfig()
	if err != nil {
		// If config file not found, log it but continue if env vars might be set
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Println("Config file ('config.yaml') not found in specified path, relying solely on environment variables.")
		} else {
			// If another error occurred reading the config file, return it
			return Config{}, fmt.Errorf("error reading config file: %w", err)
		}
	} else {
		log.Printf("Using configuration file: %s", viper.ConfigFileUsed())
	}

	// Unmarshal the configuration into the Config struct
	err = viper.Unmarshal(&config)
	if err != nil {
		return Config{}, fmt.Errorf("unable to decode config into struct: %w", err)
	}

	// Optional: Add validation logic here for required fields
	if config.SuiRPC == "" {
		log.Println("WARN: SUI_RPC_ENDPOINT is not set.")
		// Potentially return an error if critical: return Config{}, errors.New("SUI_RPC_ENDPOINT is required")
	}
	// Add more validation as needed...

	return
}
