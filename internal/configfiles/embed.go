// Package configfiles provides embedded configuration files for VerustCode.
// These files are used as templates for initializing user configuration.
package configfiles

import (
	"embed"
	"os"
	"path/filepath"
)

// Embedded configuration files
// Note: Runtime settings are managed via database and Settings page
//
//go:embed bootstrap.example.yaml
//go:embed default.example.yaml
//go:embed all:reports
var configFS embed.FS

// GetBootstrapExample returns the example bootstrap configuration file content
func GetBootstrapExample() ([]byte, error) {
	return configFS.ReadFile("bootstrap.example.yaml")
}

// GetReviewExample returns the example review rules file content (default.example.yaml)
func GetReviewExample() ([]byte, error) {
	return configFS.ReadFile("default.example.yaml")
}

// GetReportConfig returns the specified report configuration content
func GetReportConfig(name string) ([]byte, error) {
	return configFS.ReadFile(filepath.Join("reports", name))
}

// ListReportConfigs returns a list of available report configuration names
func ListReportConfigs() []string {
	entries, err := configFS.ReadDir("reports")
	if err != nil {
		return nil
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".yaml" {
			names = append(names, entry.Name())
		}
	}
	return names
}

// GetAllReportConfigs returns all embedded report configurations as a map of name to content
func GetAllReportConfigs() (map[string][]byte, error) {
	configs := make(map[string][]byte)
	for _, name := range ListReportConfigs() {
		data, err := GetReportConfig(name)
		if err != nil {
			return nil, err
		}
		configs[name] = data
	}
	return configs, nil
}

// InitReportConfigs initializes report configurations from embedded files to target directory.
// This function copies embedded report YAML files to the specified directory if they don't exist.
// It is used during initial setup (verustcode serve --check) to create default report configurations.
func InitReportConfigs(targetDir string) (int, error) {
	// Get all embedded report configurations
	configs, err := GetAllReportConfigs()
	if err != nil {
		return 0, err
	}

	created := 0
	for name, data := range configs {
		targetPath := filepath.Join(targetDir, name)

		// Skip if file already exists
		if _, err := os.Stat(targetPath); err == nil {
			continue
		}

		// Write file
		if err := os.WriteFile(targetPath, data, 0644); err != nil {
			return created, err
		}
		created++
	}

	return created, nil
}

// ReportConfigExists checks if a report config file exists in the target directory
func ReportConfigExists(targetDir string) bool {
	entries, err := os.ReadDir(targetDir)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".yaml" {
			return true
		}
	}
	return false
}
