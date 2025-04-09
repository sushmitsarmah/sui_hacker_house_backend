// backend/walrus/deploy.go
package walrus

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Deployer struct {
	siteBuilderPath string
	walrusCLIPath   string
	// Add fields for wallet management / WAL token funding if needed
}

func NewDeployer(siteBuilderPath, walrusCLIPath string) *Deployer {
	return &Deployer{
		siteBuilderPath: siteBuilderPath,
		walrusCLIPath:   walrusCLIPath,
	}
}

// DeployFiles takes a map of filename->content, saves them, runs site-builder, and walrus publish.
func (d *Deployer) DeployFiles(ctx context.Context, files map[string]string) (string, error) {
	// 1. Create a temporary directory for the project files
	tempDir, err := os.MkdirTemp("", "walrus-deploy-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir) // Clean up the temporary directory

	log.Printf("Created temporary directory for deployment: %s", tempDir)

	// 2. Write files to the temporary directory
	for name, content := range files {
		filePath := filepath.Join(tempDir, name)
		// Ensure subdirectories exist (if any specified in filename like 'js/app.js')
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return "", fmt.Errorf("failed to create subdirectories for %s: %w", name, err)
		}
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return "", fmt.Errorf("failed to write file %s: %w", name, err)
		}
	}
	log.Printf("Wrote %d files to %s", len(files), tempDir)


	// 3. Run site-builder (if necessary - depends on what site-builder does)
	// Assuming site-builder takes the source dir and outputs to a 'dist' subdir or similar
	// Adjust command based on actual site-builder usage
	builderCmd := exec.CommandContext(ctx, d.siteBuilderPath, tempDir) // Example usage
    var builderStdErr bytes.Buffer
    builderCmd.Stderr = &builderStdErr
	log.Printf("Running site-builder: %s", builderCmd.String())
	if err := builderCmd.Run(); err != nil {
        log.Printf("site-builder stderr: %s", builderStdErr.String())
		return "", fmt.Errorf("site-builder failed: %w (stderr: %s)", err, builderStdErr.String())
	}
	log.Println("site-builder completed successfully.")

    // Determine the directory to publish (might be tempDir or a sub-directory like 'dist')
    publishDir := tempDir // Assume site-builder modifies in-place or we publish source
    // If site-builder creates an output dir: publishDir = filepath.Join(tempDir, "dist")


	// 4. Run walrus publish
	// TODO: Add WAL token funding logic here if required before publishing.
	// This might involve calling another CLI command, interacting with a wallet service, etc.

	// Command: walrus publish <directory>
	publishCmd := exec.CommandContext(ctx, d.walrusCLIPath, "publish", publishDir)
	var publishStdOut, publishStdErr bytes.Buffer
	publishCmd.Stdout = &publishStdOut
	publishCmd.Stderr = &publishStdErr

	log.Printf("Running walrus publish: %s", publishCmd.String())
	if err := publishCmd.Run(); err != nil {
        log.Printf("walrus publish stderr: %s", publishStdErr.String())
		return "", fmt.Errorf("walrus publish failed: %w (stderr: %s)", err, publishStdErr.String())
	}

    output := publishStdOut.String()
	log.Printf("walrus publish stdout: %s", output)


	// 5. Parse CID from walrus publish output
	// Example: Output might be "Published to ipfs://<CID>" or just <CID>
    cid := extractCID(output) // Implement this parsing function
    if cid == "" {
        log.Printf("walrus publish stderr: %s", publishStdErr.String()) // Log stderr too if CID extraction fails
        return "", fmt.Errorf("failed to extract CID from walrus publish output: %s", output)
    }

	log.Printf("Extracted CID: %s", cid)
	return cid, nil
}

// extractCID parses the output of `walrus publish` to find the CID.
// This needs to be adapted based on the *actual* output format.
func extractCID(output string) string {
    // Option 1: Simple prefix check (if output is like "Published: bafy...")
    prefix := "Published: "
    if strings.HasPrefix(output, prefix) {
        return strings.TrimSpace(strings.TrimPrefix(output, prefix))
    }

    // Option 2: Look for ipfs:// prefix
    ipfsPrefix := "ipfs://"
    if idx := strings.Index(output, ipfsPrefix); idx != -1 {
        line := output[idx:] // Get the rest of the line/output from ipfs://
        parts := strings.Fields(line) // Split by space
        if len(parts) > 0 {
            cid := strings.TrimPrefix(parts[0], ipfsPrefix)
            return cid
        }
    }

    // Option 3: Assume the last word is the CID (less robust)
    lines := strings.Split(strings.TrimSpace(output), "\n")
    if len(lines) > 0 {
        lastLine := lines[len(lines)-1]
        parts := strings.Fields(lastLine)
        if len(parts) > 0 {
            // Basic check if it looks like a CID (e.g., starts with "bafy" or "Qm")
            potentialCID := parts[len(parts)-1]
            if strings.HasPrefix(potentialCID, "bafy") || strings.HasPrefix(potentialCID, "Qm") {
                 return potentialCID
            }
        }
    }


    // Add more robust parsing based on actual CLI output format
    log.Printf("WARN: Could not extract CID using known patterns from output: %s", output)
    return "" // Indicate failure
}
