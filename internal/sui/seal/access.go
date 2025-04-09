// backend/seal/access.go
package seal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Client struct to interact with Seal API
type Client struct {
	apiKey     string
	endpoint   string
	httpClient *http.Client
}

// NewClient creates a new Seal API client.
func NewClient(apiKey, endpoint string) *Client {
	return &Client{
		apiKey:   apiKey,
		endpoint: endpoint,
		httpClient: &http.Client{
			Timeout: 15 * time.Second, // Set a reasonable timeout
		},
	}
}

// SealPolicyRequest defines the structure for creating a Seal policy.
// Adjust this based on the actual Seal API specification.
type SealPolicyRequest struct {
	Name        string                 `json:"name"`        // A unique name for the policy
	ContentCIDs []string               `json:"contentCids"` // List of CIDs protected by this policy
	AccessGroup map[string]interface{} `json:"accessGroup"` // NFT criteria
}

// SealPolicyResponse structure (if needed)
// type SealPolicyResponse struct { ... }


// RegisterPolicy registers a new access policy with Seal.
func (c *Client) RegisterPolicy(ctx context.Context, policyName, contentCID string, nftCriteria map[string]interface{}) error {
	if c.apiKey == "" || c.endpoint == "" {
		log.Println("WARN: Seal API Key or Endpoint not configured. Skipping policy registration.")
		// Depending on requirements, could return nil or an error here.
		// Returning nil for now to allow deployment even if Seal isn't fully set up.
		return nil // Or return errors.New("Seal client not configured")
	}

	apiURL := fmt.Sprintf("%s/v1/policies", c.endpoint) // Adjust API path as needed

	requestBody := SealPolicyRequest{
		Name:        policyName,
		ContentCIDs: []string{contentCID},
		AccessGroup: nftCriteria, // Contains NFT contract, network, logic etc.
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal Seal policy request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create Seal API request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey) // Assuming Bearer token auth

	log.Printf("Registering Seal policy '%s' for CID %s at %s", policyName, contentCID, apiURL)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request to Seal API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
        // Read response body for more details
        var bodyBytes []byte
        resp.Body.Read(bodyBytes)
        log.Printf("Seal API error response body: %s", string(bodyBytes))
		return fmt.Errorf("Seal API returned non-success status: %s", resp.Status)
	}

	log.Printf("Successfully registered Seal policy '%s' for CID %s", policyName, contentCID)
	// TODO: Parse response if it contains useful info (e.g., policy ID)
	return nil
}

// SealVerifyRequest structure (adjust based on API)
type SealVerifyRequest struct {
    WalletAddress string `json:"walletAddress"`
    ContentCID    string `json:"contentCid"`
}

// SealVerifyResponse structure (adjust based on API)
type SealVerifyResponse struct {
    HasAccess bool `json:"hasAccess"`
    // Add other fields if provided by the API
}

// VerifyAccess checks if a wallet has access to a specific CID via Seal.
// Note: Seal verification is often done client-side using their SDK.
// This backend implementation is for cases where backend verification is desired.
func (c *Client) VerifyAccess(ctx context.Context, walletAddress, contentCID string) (bool, error) {
	if c.apiKey == "" || c.endpoint == "" {
		log.Println("WARN: Seal API Key or Endpoint not configured. Assuming access denied for verification.")
		return false, fmt.Errorf("Seal client not configured")
	}

	// This endpoint is hypothetical - check Seal documentation for actual verification API
	apiURL := fmt.Sprintf("%s/v1/verify", c.endpoint)

    requestBody := SealVerifyRequest{
        WalletAddress: walletAddress,
        ContentCID:    contentCID,
    }

    jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return false, fmt.Errorf("failed to marshal Seal verify request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return false, fmt.Errorf("failed to create Seal verify API request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

    log.Printf("Verifying Seal access for wallet %s on CID %s via %s", walletAddress, contentCID, apiURL)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to send verify request to Seal API: %w", err)
	}
	defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        var bodyBytes []byte
        resp.Body.Read(bodyBytes)
        log.Printf("Seal verify API error response body: %s", string(bodyBytes))
        return false, fmt.Errorf("Seal verify API returned non-success status: %s", resp.Status)
    }

    var verifyResp SealVerifyResponse
    if err := json.NewDecoder(resp.Body).Decode(&verifyResp); err != nil {
        return false, fmt.Errorf("failed to decode Seal verify response: %w", err)
    }

	return verifyResp.HasAccess, nil
}
