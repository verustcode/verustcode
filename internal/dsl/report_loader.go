// Package dsl provides DSL parsing for report configuration.
package dsl

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"github.com/verustcode/verustcode/pkg/errors"
	"github.com/verustcode/verustcode/pkg/logger"
)

// ReportLoader loads report DSL configurations from files.
type ReportLoader struct {
	// configs stores loaded report configurations by ID
	configs map[string]*ReportConfig
	mu      sync.RWMutex
}

// NewReportLoader creates a new report loader.
func NewReportLoader() *ReportLoader {
	return &ReportLoader{
		configs: make(map[string]*ReportConfig),
	}
}

// LoadFile loads a single report configuration file.
func (l *ReportLoader) LoadFile(path string) (*ReportConfig, error) {
	logger.Debug("Loading report configuration",
		zap.String("path", path),
	)

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.New(errors.ErrCodeConfigNotFound,
				"report configuration file not found: "+path)
		}
		return nil, errors.Wrap(errors.ErrCodeConfigInvalid,
			"failed to read report configuration file", err)
	}

	// Expand environment variables (reuse the same function as review DSL)
	// Supports ${VAR_NAME} syntax with allowed prefixes: SCOPEVIEW_, CI_, GITHUB_, GITLAB_, CUSTOM_REVIEW_
	expanded := expandEnvVars(string(data))

	// Parse YAML
	var config ReportConfig
	if err := yaml.Unmarshal([]byte(expanded), &config); err != nil {
		return nil, errors.Wrap(errors.ErrCodeConfigInvalid,
			"failed to parse report configuration YAML", err)
	}

	// Validate
	if err := l.validate(&config); err != nil {
		return nil, err
	}

	// Apply defaults
	config.ApplyDefaults()

	// Register
	l.mu.Lock()
	l.configs[config.ID] = &config
	l.mu.Unlock()

	logger.Info("Loaded report configuration",
		zap.String("path", path),
		zap.String("id", config.ID),
		zap.String("name", config.Name),
	)

	return &config, nil
}

// LoadDir loads all report configurations from a directory.
func (l *ReportLoader) LoadDir(dir string) ([]*ReportConfig, error) {
	logger.Debug("Loading report configurations from directory",
		zap.String("dir", dir),
	)

	// Find all YAML files
	patterns := []string{"*.yaml", "*.yml"}
	var files []string

	for _, pattern := range patterns {
		matches, err := filepath.Glob(filepath.Join(dir, pattern))
		if err != nil {
			return nil, errors.Wrap(errors.ErrCodeInternal,
				"failed to glob directory", err)
		}
		files = append(files, matches...)
	}

	if len(files) == 0 {
		logger.Warn("No report configuration files found in directory",
			zap.String("dir", dir),
		)
		return nil, nil
	}

	var configs []*ReportConfig
	for _, file := range files {
		config, err := l.LoadFile(file)
		if err != nil {
			logger.Warn("Failed to load report configuration file, skipping",
				zap.String("file", file),
				zap.Error(err),
			)
			continue
		}
		configs = append(configs, config)
	}

	logger.Info("Loaded report configurations from directory",
		zap.String("dir", dir),
		zap.Int("count", len(configs)),
	)

	return configs, nil
}

// LoadFromBytes loads a report configuration from YAML bytes.
func (l *ReportLoader) LoadFromBytes(data []byte) (*ReportConfig, error) {
	// Expand environment variables
	expanded := expandEnvVars(string(data))

	var config ReportConfig
	if err := yaml.Unmarshal([]byte(expanded), &config); err != nil {
		return nil, errors.Wrap(errors.ErrCodeConfigInvalid,
			"failed to parse report configuration YAML", err)
	}

	// Validate
	if err := l.validate(&config); err != nil {
		return nil, err
	}

	// Apply defaults
	config.ApplyDefaults()

	// Register
	l.mu.Lock()
	l.configs[config.ID] = &config
	l.mu.Unlock()

	return &config, nil
}

// Get returns a loaded report configuration by ID.
func (l *ReportLoader) Get(id string) (*ReportConfig, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	config, ok := l.configs[id]
	return config, ok
}

// List returns all loaded report configurations.
func (l *ReportLoader) List() []*ReportConfig {
	l.mu.RLock()
	defer l.mu.RUnlock()

	result := make([]*ReportConfig, 0, len(l.configs))
	for _, config := range l.configs {
		result = append(result, config)
	}
	return result
}

// ListIDs returns all loaded report configuration IDs.
func (l *ReportLoader) ListIDs() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	ids := make([]string, 0, len(l.configs))
	for id := range l.configs {
		ids = append(ids, id)
	}
	return ids
}

// validTones defines the valid tone values for section style.
var validTones = map[string]bool{
	"professional": true,
	"friendly":     true,
	"technical":    true,
	"strict":       true,
	"constructive": true,
}

// maxReferenceDocs is the maximum number of reference documents allowed.
const maxReferenceDocs = 10

// minSectionLength is the minimum value for max_section_length if specified.
const minSectionLength = 500

// maxSectionSummaryLength is the maximum value for section.summary.max_length.
const maxSectionSummaryLength = 1000

// validate validates a report configuration.
func (l *ReportLoader) validate(config *ReportConfig) error {
	if config.ID == "" {
		return errors.New(errors.ErrCodeConfigInvalid,
			"report configuration missing required field: id")
	}

	if config.Name == "" {
		return errors.New(errors.ErrCodeConfigInvalid,
			fmt.Sprintf("report configuration '%s' missing required field: name", config.ID))
	}

	// Validate output.style.tone if specified
	if config.Output.Style.Tone != "" {
		if !validTones[config.Output.Style.Tone] {
			logger.Warn("Unknown output style tone, using default",
				zap.String("id", config.ID),
				zap.String("tone", config.Output.Style.Tone),
			)
		}
	}

	// Validate output.style.heading_level (1-4)
	if hl := config.Output.Style.HeadingLevel; hl != 0 && (hl < 1 || hl > 4) {
		return errors.New(errors.ErrCodeConfigInvalid,
			fmt.Sprintf("report configuration '%s': output.style.heading_level must be between 1 and 4, got %d", config.ID, hl))
	}

	// Validate output.style.max_section_length (>= 500 if specified)
	if msl := config.Output.Style.MaxSectionLength; msl != 0 && msl < minSectionLength {
		return errors.New(errors.ErrCodeConfigInvalid,
			fmt.Sprintf("report configuration '%s': output.style.max_section_length must be at least %d, got %d", config.ID, minSectionLength, msl))
	}

	// Validate structure.reference_docs count
	if len(config.Structure.ReferenceDocs) > maxReferenceDocs {
		return errors.New(errors.ErrCodeConfigInvalid,
			fmt.Sprintf("report configuration '%s': structure.reference_docs exceeds maximum of %d files", config.ID, maxReferenceDocs))
	}

	// Validate section.reference_docs count
	if len(config.Section.ReferenceDocs) > maxReferenceDocs {
		return errors.New(errors.ErrCodeConfigInvalid,
			fmt.Sprintf("report configuration '%s': section.reference_docs exceeds maximum of %d files", config.ID, maxReferenceDocs))
	}

	// Validate section.summary.max_length (must be positive and <= 1000 if specified)
	if sml := config.Section.Summary.MaxLength; sml != 0 {
		if sml < 0 {
			return errors.New(errors.ErrCodeConfigInvalid,
				fmt.Sprintf("report configuration '%s': section.summary.max_length must be non-negative, got %d", config.ID, sml))
		}
		if sml > maxSectionSummaryLength {
			return errors.New(errors.ErrCodeConfigInvalid,
				fmt.Sprintf("report configuration '%s': section.summary.max_length must not exceed %d, got %d", config.ID, maxSectionSummaryLength, sml))
		}
	}

	return nil
}

// Clear removes all loaded configurations.
func (l *ReportLoader) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.configs = make(map[string]*ReportConfig)
}

// globalReportLoader is the default report loader instance.
var globalReportLoader = NewReportLoader()

// GetReportLoader returns the global report loader instance.
func GetReportLoader() *ReportLoader {
	return globalReportLoader
}

// LoadReportConfig loads a report configuration from a file using the global loader.
func LoadReportConfig(path string) (*ReportConfig, error) {
	return globalReportLoader.LoadFile(path)
}

// LoadReportConfigs loads all report configurations from a directory using the global loader.
func LoadReportConfigs(dir string) ([]*ReportConfig, error) {
	return globalReportLoader.LoadDir(dir)
}

// GetReportConfig returns a loaded report configuration by ID from the global loader.
func GetReportConfig(id string) (*ReportConfig, bool) {
	return globalReportLoader.Get(id)
}

// ListReportConfigs returns all loaded report configurations from the global loader.
func ListReportConfigs() []*ReportConfig {
	return globalReportLoader.List()
}
