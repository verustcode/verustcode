// Package report provides report generation functionality.
package report

import (
	"fmt"
	"sync"

	"go.uber.org/zap"

	"github.com/verustcode/verustcode/internal/dsl"
	"github.com/verustcode/verustcode/pkg/logger"
)

// DefaultReportsConfigDir is the default directory for report configurations.
// Report configurations should be loaded from filesystem, not from embedded files.
// Embedded files are only used for initial setup (verustcode serve --check).
const DefaultReportsConfigDir = "config/reports"

// ReportTypeDefinition defines a report type with its DSL configuration.
// Note: Sections are no longer predefined - they are dynamically generated
// by AI in Phase 1 based on project analysis.
type ReportTypeDefinition struct {
	ID          string            `json:"id"`          // Type identifier: wiki, security, architecture, etc.
	Name        string            `json:"name"`        // Display name
	Description string            `json:"description"` // Type description
	Config      *dsl.ReportConfig `json:"-"`           // DSL configuration (not serialized)
}

// reportTypeRegistry holds all registered report types
var (
	reportTypes     = make(map[string]*ReportTypeDefinition)
	reportTypesMu   sync.RWMutex
	reportTypeOrder []string // Maintains registration order for listing
	initialized     bool
)

// RegisterReportType registers a report type definition
func RegisterReportType(def *ReportTypeDefinition) {
	reportTypesMu.Lock()
	defer reportTypesMu.Unlock()

	if _, exists := reportTypes[def.ID]; !exists {
		reportTypeOrder = append(reportTypeOrder, def.ID)
	}
	reportTypes[def.ID] = def

	logger.Debug("Registered report type",
		zap.String("id", def.ID),
		zap.String("name", def.Name),
	)
}

// GetReportType returns a report type definition by ID
func GetReportType(typeID string) (*ReportTypeDefinition, error) {
	// Ensure types are initialized
	initReportTypes()

	reportTypesMu.RLock()
	defer reportTypesMu.RUnlock()

	if def, ok := reportTypes[typeID]; ok {
		return def, nil
	}
	return nil, fmt.Errorf("unknown report type: %s", typeID)
}

// ListReportTypes returns all registered report types in registration order
func ListReportTypes() []*ReportTypeDefinition {
	// Ensure types are initialized
	initReportTypes()

	reportTypesMu.RLock()
	defer reportTypesMu.RUnlock()

	result := make([]*ReportTypeDefinition, 0, len(reportTypeOrder))
	for _, id := range reportTypeOrder {
		if def, ok := reportTypes[id]; ok {
			result = append(result, def)
		}
	}
	return result
}

// initReportTypes initializes report types from filesystem configurations.
// This is called lazily to avoid initialization order issues.
// Note: Report configurations are loaded from filesystem (config/reports/),
// NOT from embedded files. Embedded files are only used for initial setup.
func initReportTypes() {
	reportTypesMu.Lock()
	if initialized {
		reportTypesMu.Unlock()
		return
	}
	initialized = true
	reportTypesMu.Unlock()

	logger.Debug("Initializing report types from filesystem",
		zap.String("dir", DefaultReportsConfigDir),
	)

	// Load report configurations from filesystem
	loader := dsl.NewReportLoader()
	configs, err := loader.LoadDir(DefaultReportsConfigDir)
	if err != nil {
		logger.Warn("Failed to load report configs from filesystem",
			zap.String("dir", DefaultReportsConfigDir),
			zap.Error(err),
		)
		return
	}

	if len(configs) == 0 {
		logger.Warn("No report configurations found in filesystem",
			zap.String("dir", DefaultReportsConfigDir),
			zap.String("hint", "Run 'verustcode serve --check' to initialize default report configurations"),
		)
		return
	}

	// Register each loaded configuration
	for _, config := range configs {
		RegisterReportType(&ReportTypeDefinition{
			ID:          config.ID,
			Name:        config.Name,
			Description: config.Description,
			Config:      config,
		})
	}

	logger.Info("Report types initialized from filesystem",
		zap.String("dir", DefaultReportsConfigDir),
		zap.Int("count", len(reportTypes)),
		zap.Strings("types", reportTypeOrder),
	)
}

// ScanReportTypesFromDir scans and returns report types from a directory.
// Unlike LoadReportTypesFromDir, this does NOT cache results - it reads fresh on every call.
// This is useful for API endpoints that need to reflect real-time changes to config files.
// Note: Duplicate IDs are deduplicated - last loaded wins (by filename order).
func ScanReportTypesFromDir(dir string) ([]*ReportTypeDefinition, error) {
	logger.Debug("Scanning report types from directory",
		zap.String("dir", dir),
	)

	loader := dsl.NewReportLoader()
	configs, err := loader.LoadDir(dir)
	if err != nil {
		return nil, err
	}

	// Use map to deduplicate by ID (last loaded wins)
	seen := make(map[string]*ReportTypeDefinition)
	for _, config := range configs {
		if _, exists := seen[config.ID]; exists {
			logger.Warn("Duplicate report type ID found, overwriting",
				zap.String("id", config.ID),
				zap.String("name", config.Name),
			)
		}
		seen[config.ID] = &ReportTypeDefinition{
			ID:          config.ID,
			Name:        config.Name,
			Description: config.Description,
			Config:      config,
		}
	}

	// Convert map to slice
	result := make([]*ReportTypeDefinition, 0, len(seen))
	for _, def := range seen {
		result = append(result, def)
	}

	logger.Debug("Scanned report types from directory",
		zap.String("dir", dir),
		zap.Int("count", len(result)),
	)

	return result, nil
}

// ScanReportTypeFromDir scans and returns a single report type by ID from a directory.
// This does NOT use cache - it reads fresh from filesystem on every call.
// Returns the report type definition or an error if not found.
func ScanReportTypeFromDir(dir string, typeID string) (*ReportTypeDefinition, error) {
	types, err := ScanReportTypesFromDir(dir)
	if err != nil {
		return nil, err
	}

	for _, t := range types {
		if t.ID == typeID {
			return t, nil
		}
	}

	return nil, fmt.Errorf("report type '%s' not found in %s", typeID, dir)
}

// ScanReportType scans and returns a single report type by ID from default config directory.
// This does NOT use cache - it reads fresh from filesystem on every call.
func ScanReportType(typeID string) (*ReportTypeDefinition, error) {
	return ScanReportTypeFromDir(DefaultReportsConfigDir, typeID)
}

// ScanReportConfig scans and returns the DSL configuration for a report type.
// This does NOT use cache - it reads fresh from filesystem on every call.
func ScanReportConfig(typeID string) (*dsl.ReportConfig, error) {
	def, err := ScanReportType(typeID)
	if err != nil {
		return nil, err
	}
	if def.Config == nil {
		return nil, fmt.Errorf("report type '%s' has no DSL configuration", typeID)
	}
	return def.Config, nil
}

// LoadReportTypesFromDir loads additional report types from a directory.
// This can be used to load user-defined report types.
func LoadReportTypesFromDir(dir string) error {
	logger.Debug("Loading report types from directory",
		zap.String("dir", dir),
	)

	loader := dsl.NewReportLoader()
	configs, err := loader.LoadDir(dir)
	if err != nil {
		return err
	}

	for _, config := range configs {
		RegisterReportType(&ReportTypeDefinition{
			ID:          config.ID,
			Name:        config.Name,
			Description: config.Description,
			Config:      config,
		})
	}

	logger.Info("Loaded report types from directory",
		zap.String("dir", dir),
		zap.Int("count", len(configs)),
	)

	return nil
}

// GetReportConfig returns the DSL configuration for a report type.
func GetReportConfig(typeID string) (*dsl.ReportConfig, error) {
	def, err := GetReportType(typeID)
	if err != nil {
		return nil, err
	}
	if def.Config == nil {
		return nil, fmt.Errorf("report type '%s' has no DSL configuration", typeID)
	}
	return def.Config, nil
}

// ResetReportTypes clears all registered report types and resets initialization.
// This is mainly useful for testing.
func ResetReportTypes() {
	reportTypesMu.Lock()
	defer reportTypesMu.Unlock()

	reportTypes = make(map[string]*ReportTypeDefinition)
	reportTypeOrder = nil
	initialized = false
}
