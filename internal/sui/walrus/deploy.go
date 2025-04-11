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

// DeployFiles takes a map of filename->content, saves them, runs npm install, npm build, site-builder, and walrus publish.
func (d *Deployer) DeployFiles(ctx context.Context) (string, error) {
	// 1. Create a temporary directory for the project files
	tempDir := "tmp"

	// 3. Run npm install
	npmInstallCmd := exec.CommandContext(ctx, "npm", "install")
	npmInstallCmd.Dir = tempDir // Set working directory to our temp folder
	var npmInstallStdErr bytes.Buffer
	npmInstallCmd.Stderr = &npmInstallStdErr

	log.Printf("Running npm install in %s", tempDir)
	if err := npmInstallCmd.Run(); err != nil {
		log.Printf("npm install stderr: %s", npmInstallStdErr.String())
		return "", fmt.Errorf("npm install failed: %w (stderr: %s)", err, npmInstallStdErr.String())
	}
	log.Println("npm install completed successfully.")

	// 4. Run npm run build
	npmBuildCmd := exec.CommandContext(ctx, "npm", "run", "build")
	npmBuildCmd.Dir = tempDir // Set working directory to our temp folder
	var npmBuildStdErr bytes.Buffer
	npmBuildCmd.Stderr = &npmBuildStdErr

	log.Printf("Running npm run build in %s", tempDir)
	if err := npmBuildCmd.Run(); err != nil {
		log.Printf("npm run build stderr: %s", npmBuildStdErr.String())
		return "", fmt.Errorf("npm run build failed: %w (stderr: %s)", err, npmBuildStdErr.String())
	}
	log.Println("npm run build completed successfully.")

	// 5. The build output should now be in tempDir/dist
	distDir := filepath.Join(tempDir, "dist")
	if _, err := os.Stat(distDir); os.IsNotExist(err) {
		return "", fmt.Errorf("build process did not create expected dist directory at %s", distDir)
	}

	// 8. Get Wal token
	getWal := exec.CommandContext(ctx, d.walrusCLIPath, "get-wal")
	var publishStdOut, publishStdErr bytes.Buffer
	getWal.Stdout = &publishStdOut
	getWal.Stderr = &publishStdErr

	// 6. Run site-builder with the dist directory as input
	// builderCmd := exec.CommandContext(ctx, d.siteBuilderPath, distDir) // Use dist directory as input
	sitesConfigPath := "sites-config.yaml"
	builderCmd := exec.CommandContext(
		ctx,
		d.siteBuilderPath,
		"--config",
		sitesConfigPath,
		"publish",
		distDir,
		"--epochs",
		"2",
	)
	var builderStdOut, builderStdErr bytes.Buffer
	builderCmd.Stderr = &builderStdErr
	builderCmd.Stdout = &builderStdOut

	log.Printf("Running site-builder with tmp/dist folder: %s", builderCmd.String())
	if err := builderCmd.Run(); err != nil {
		log.Printf("site-builder stderr: %s", builderStdErr.String())
		return "", fmt.Errorf("site-builder failed: %w (stderr: %s)", err, builderStdErr.String())
	}
	log.Println("site-builder completed successfully.")

	println("site-builder stdout: ", builderStdOut.String())

	// Extract the site object ID from the output
	builderOutput := builderStdOut.String()
	log.Printf("site-builder stdout: %s", builderOutput)
	siteObjectID := extractSiteObjectID(builderOutput)
	if siteObjectID == "" {
		return "", fmt.Errorf("failed to extract site object ID from site-builder output")
	}

	log.Printf("Site object ID: %s", siteObjectID)
	log.Println("site-builder completed successfully.")

	// Since we now want to return the site object ID instead of a CID,
	// we'll skip the walrus publish step and return the site object ID directly
	return siteObjectID, nil
}

// extractSiteObjectID parses the output of site-builder to find the site object ID.
func extractSiteObjectID(output string) string {
	// Looking for the line with "New site object ID: 0x..."
	lines := strings.Split(output, "\n")
	prefix := "New site object ID: "

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, prefix) {
			// Extract the ID which follows the prefix
			objectID := strings.TrimPrefix(line, prefix)
			return objectID
		}
	}

	return ""
}
