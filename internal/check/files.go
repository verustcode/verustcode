package check

import (
	"fmt"
	"os"

	"github.com/fatih/color"

	configfiles "github.com/verustcode/verustcode/internal/configfiles"
)

// TemplateType represents the type of template file
type TemplateType int

const (
	TemplateBootstrap TemplateType = iota
	TemplateRules
)

// FileConfig represents a configuration file to check
type FileConfig struct {
	Path        string
	Description string
	Template    TemplateType
}

// FileCheckResult represents the result of a file check
type FileCheckResult struct {
	Path        string
	Exists      bool
	Created     bool
	Description string
	Error       error
}

// checkFiles checks all required configuration files
func (c *Checker) checkFiles() error {
	files := c.RequiredFiles()

	for _, file := range files {
		result := c.checkFile(file)
		c.report.AddFileResult(result)

		if result.Error != nil {
			return result.Error
		}
	}

	return nil
}

// checkFile checks a single file and prompts for creation if missing
func (c *Checker) checkFile(file FileConfig) FileCheckResult {
	result := FileCheckResult{
		Path:        file.Path,
		Description: file.Description,
	}

	// Check if file exists
	if fileExists(file.Path) {
		result.Exists = true
		printFileStatus(file.Path, true, false)
		return result
	}

	// File doesn't exist
	result.Exists = false
	printFileStatus(file.Path, false, false)

	// Ask user if they want to create it
	confirm, err := confirmCreate(file.Path)
	if err != nil {
		result.Error = fmt.Errorf("failed to get user confirmation: %w", err)
		return result
	}

	if !confirm {
		// User declined, just note it
		return result
	}

	// Get template content
	content, err := getTemplateContent(file.Template)
	if err != nil {
		result.Error = fmt.Errorf("failed to get template: %w", err)
		return result
	}

	// Create parent directory if needed
	if err := ensureDir(file.Path); err != nil {
		result.Error = err
		return result
	}

	// Write file
	if err := os.WriteFile(file.Path, content, 0644); err != nil {
		result.Error = fmt.Errorf("failed to create file %s: %w", file.Path, err)
		return result
	}

	result.Created = true
	printFileCreated(file.Path)

	return result
}

// getTemplateContent returns the embedded template content
func getTemplateContent(t TemplateType) ([]byte, error) {
	switch t {
	case TemplateBootstrap:
		return configfiles.GetBootstrapExample()
	case TemplateRules:
		return configfiles.GetReviewExample()
	default:
		return nil, fmt.Errorf("unknown template type: %d", t)
	}
}

// printFileStatus prints the status of a file check
func printFileStatus(path string, exists bool, created bool) {
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)

	if exists {
		green.Printf("  ✓ %s\n", path)
	} else if created {
		green.Printf("  ✓ %s (created)\n", path)
	} else {
		yellow.Printf("  ⚠ %s does not exist\n", path)
	}
}

// printFileCreated prints a message when a file is created
func printFileCreated(path string) {
	green := color.New(color.FgGreen)
	green.Printf("  ✓ Created %s\n", path)
}

// checkReportsDir checks and initializes the reports configuration directory
func (c *Checker) checkReportsDir() error {
	reportsDir := c.ReportsDir()

	// Check if directory exists and has yaml files
	if configfiles.ReportConfigExists(reportsDir) {
		printFileStatus(reportsDir, true, false)
		return nil
	}

	// Directory doesn't exist or is empty
	printFileStatus(reportsDir, false, false)

	// Ask user if they want to initialize it
	confirm, err := confirmCreate(reportsDir + " (report configs)")
	if err != nil {
		return fmt.Errorf("failed to get user confirmation: %w", err)
	}

	if !confirm {
		return nil
	}

	// Create directory if needed
	if err := os.MkdirAll(reportsDir, 0755); err != nil {
		return fmt.Errorf("failed to create reports directory: %w", err)
	}

	// Initialize report configs from embedded files
	created, err := configfiles.InitReportConfigs(reportsDir)
	if err != nil {
		return fmt.Errorf("failed to initialize report configs: %w", err)
	}

	green := color.New(color.FgGreen)
	green.Printf("  ✓ Created %d report config(s) in %s\n", created, reportsDir)

	return nil
}
