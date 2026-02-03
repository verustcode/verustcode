// Package check provides interactive environment checking and initialization.
// It helps users set up their local VerustCode configuration properly.
package check

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"

	"github.com/verustcode/verustcode/internal/config"
)

// CheckResult represents the result of a non-interactive environment check
type CheckResult struct {
	// Success indicates whether all required checks passed
	Success bool
	// Errors contains critical errors that prevent server startup
	Errors []string
	// Warnings contains non-critical issues that don't block startup
	Warnings []string
	// Suggestions contains helpful tips for fixing issues
	Suggestions []string
}

// Checker handles environment checking and initialization
type Checker struct {
	// configDir is the base directory for configuration files
	configDir string
	// report collects check results for final output
	report *Report
	// theme for consistent styling
	theme *huh.Theme
}

// NewChecker creates a new environment checker
func NewChecker() *Checker {
	return &Checker{
		configDir: "config",
		report:    NewReport(),
		theme:     huh.ThemeCharm(),
	}
}

// Run executes the full environment check
func (c *Checker) Run() error {
	// Print header
	c.printHeader()

	// Step 1: Check and create configuration files
	fmt.Println()
	printSection("Checking configuration files")
	if err := c.checkFiles(); err != nil {
		return fmt.Errorf("file check failed: %w", err)
	}

	// Step 1.1: Check and create reports configuration directory
	fmt.Println()
	printSection("Checking reports configuration")
	if err := c.checkReportsDir(); err != nil {
		return fmt.Errorf("reports config check failed: %w", err)
	}

	// Step 2: Validate configuration files and agents
	fmt.Println()
	printSection("Validating configuration formats")
	if err := c.validateConfigs(); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	// Step 3: Print final report
	fmt.Println()
	c.report.Print()

	return nil
}

// printHeader prints the welcome header
func (c *Checker) printHeader() {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("12")).
		MarginBottom(1)

	fmt.Println(titleStyle.Render("🔍 VerustCode Environment Check"))
}

// printSection prints a section header
func printSection(title string) {
	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15"))
	fmt.Println(style.Render(title + "..."))
}

// RequiredFiles returns the list of required configuration files
func (c *Checker) RequiredFiles() []FileConfig {
	return []FileConfig{
		{
			Path:        filepath.Join(c.configDir, "bootstrap.yaml"),
			Description: "Bootstrap configuration file (server, admin, logging)",
			Template:    TemplateBootstrap,
		},
		{
			Path:        filepath.Join(c.configDir, "reviews", "default.yaml"),
			Description: "Default review rules file",
			Template:    TemplateRules,
		},
	}
}

// BootstrapPath returns the path to the bootstrap config file
func (c *Checker) BootstrapPath() string {
	return filepath.Join(c.configDir, "bootstrap.yaml")
}

// ConfigPath returns the path to the legacy config file (used for migration only)
func (c *Checker) ConfigPath() string {
	return filepath.Join(c.configDir, "config.yaml")
}

// ReviewPath returns the path to the default review rules file
func (c *Checker) ReviewPath() string {
	return filepath.Join(c.configDir, "reviews", "default.yaml")
}

// ReportsDir returns the path to the reports configuration directory
func (c *Checker) ReportsDir() string {
	return filepath.Join(c.configDir, "reports")
}

// confirmCreate asks user to confirm file creation
func confirmCreate(path string) (bool, error) {
	var confirm bool
	err := huh.NewConfirm().
		Title(fmt.Sprintf("Create %s from template?", path)).
		Affirmative("Yes").
		Negative("No").
		Value(&confirm).
		Run()
	if err != nil {
		return false, err
	}
	return confirm, nil
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ensureDir creates directory if it doesn't exist
func ensureDir(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}
	return nil
}

// RunNonInteractive performs a non-interactive environment check.
// Unlike Run(), this method does not prompt for user input and does not create files.
// It returns a CheckResult with errors, warnings, and suggestions.
func (c *Checker) RunNonInteractive() *CheckResult {
	result := &CheckResult{
		Success:     true,
		Errors:      make([]string, 0),
		Warnings:    make([]string, 0),
		Suggestions: make([]string, 0),
	}

	// Step 1: Check if required configuration files exist
	c.checkFilesNonInteractive(result)

	// If required files are missing, return early with suggestions
	if !result.Success {
		result.Suggestions = append(result.Suggestions,
			"Run 'verustcode serve --check' to interactively create configuration files",
		)
		return result
	}

	// Step 2: Validate configuration file formats
	c.validateConfigsNonInteractive(result)

	// Step 3: Check credentials (as warnings, not errors)
	c.checkCredentialsNonInteractive(result)

	return result
}

// checkFilesNonInteractive checks if required configuration files exist
func (c *Checker) checkFilesNonInteractive(result *CheckResult) {
	// Bootstrap.yaml is required for server startup
	bootstrapPath := filepath.Join(c.configDir, "bootstrap.yaml")
	if !fileExists(bootstrapPath) {
		result.Success = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("Bootstrap configuration not found: %s", bootstrapPath))
	}

	// Review rules file is required
	reviewPath := filepath.Join(c.configDir, "reviews", "default.yaml")
	if !fileExists(reviewPath) {
		result.Success = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("Default review rules file not found: %s", reviewPath))
	}
}

// validateConfigsNonInteractive validates configuration file formats
func (c *Checker) validateConfigsNonInteractive(result *CheckResult) {
	// Validate bootstrap.yaml
	bootstrapResult := c.validateBootstrapYaml()
	if !bootstrapResult.Valid {
		result.Success = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("Invalid bootstrap.yaml: %v", bootstrapResult.Error))
	}

	// Validate default.yaml (review rules)
	rulesResult, _ := c.validateRulesYaml()
	if !rulesResult.Valid {
		result.Success = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("Invalid default.yaml: %v", rulesResult.Error))
	}
}

// checkCredentialsNonInteractive checks credentials as warnings (not blocking errors)
func (c *Checker) checkCredentialsNonInteractive(result *CheckResult) {
	// Load bootstrap config to check admin credentials
	bootstrapCfg, err := config.LoadBootstrap(c.BootstrapPath())
	if err != nil {
		// Bootstrap already validated, this shouldn't fail
		return
	}

	// Note: password_hash validation is intentionally skipped
	// Users can set password via Web UI after server starts

	// Note: jwt_secret will be auto-generated if empty during server startup
	// So we don't need to warn about it here

	// Only check if username is set when admin is enabled
	if bootstrapCfg.Admin != nil && bootstrapCfg.Admin.Enabled {
		if bootstrapCfg.Admin.Username == "" {
			result.Warnings = append(result.Warnings,
				"Admin username not set")
		}
	}

	// Note: Agent API keys are now stored in database and configured via admin interface
	// Skip agent credential check during startup
}

// PrintCheckResult prints the check result in a formatted way
func PrintCheckResult(result *CheckResult) {
	red := color.New(color.FgRed)
	yellow := color.New(color.FgYellow)
	cyan := color.New(color.FgCyan)

	// Print errors
	if len(result.Errors) > 0 {
		fmt.Println()
		red.Println("[ERROR] Environment check failed")
		fmt.Println()
		for _, err := range result.Errors {
			red.Printf("  ✗ %s\n", err)
		}
	}

	// Print warnings
	if len(result.Warnings) > 0 {
		fmt.Println()
		yellow.Println("[WARNING] Configuration warnings:")
		fmt.Println()
		for _, warn := range result.Warnings {
			yellow.Printf("  ⚠ %s\n", warn)
		}
	}

	// Print suggestions
	if len(result.Suggestions) > 0 {
		cyan.Println("\nTo fix these issues:")
		for _, suggestion := range result.Suggestions {
			fmt.Printf("  → %s\n", suggestion)
		}
	}

	fmt.Println()
}
