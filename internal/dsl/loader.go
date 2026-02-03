package dsl

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"go.uber.org/zap"

	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/pkg/errors"
	"github.com/verustcode/verustcode/pkg/logger"
)

// Loader loads DSL configuration from files
type Loader struct {
	parser *Parser
}

// NewLoader creates a new DSL loader
func NewLoader() *Loader {
	return &Loader{
		parser: NewParser(),
	}
}

// NewStrictLoader creates a new DSL loader with strict validation
func NewStrictLoader() *Loader {
	return &Loader{
		parser: NewStrictParser(),
	}
}

// Load loads and parses a DSL configuration file
func (l *Loader) Load(path string) (*ReviewRulesConfig, error) {
	logger.Debug("Loading DSL configuration",
		zap.String("path", path),
	)

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.New(errors.ErrCodeConfigNotFound,
				"configuration file not found: "+path)
		}
		return nil, errors.Wrap(errors.ErrCodeConfigInvalid,
			"failed to read configuration file", err)
	}

	// Expand environment variables
	expanded := expandEnvVars(string(data))

	// Parse configuration
	config, err := l.parser.Parse([]byte(expanded))
	if err != nil {
		return nil, err
	}

	logger.Info("Loaded DSL configuration",
		zap.String("path", path),
		zap.Int("rules", len(config.Rules)),
		zap.Strings("rule_ids", config.GetRuleIDs()),
	)

	return config, nil
}

// LoadFromDir loads all DSL configuration files from a directory
func (l *Loader) LoadFromDir(dir string) ([]*ReviewRulesConfig, error) {
	logger.Debug("Loading DSL configurations from directory",
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
		return nil, errors.New(errors.ErrCodeConfigNotFound,
			"no configuration files found in directory: "+dir)
	}

	var configs []*ReviewRulesConfig
	for _, file := range files {
		config, err := l.Load(file)
		if err != nil {
			logger.Warn("Failed to load configuration file, skipping",
				zap.String("file", file),
				zap.Error(err),
			)
			continue
		}
		configs = append(configs, config)
	}

	if len(configs) == 0 {
		return nil, errors.New(errors.ErrCodeConfigInvalid,
			"no valid configuration files found in directory: "+dir)
	}

	logger.Info("Loaded DSL configurations from directory",
		zap.String("dir", dir),
		zap.Int("files", len(configs)),
	)

	return configs, nil
}

// MergeConfigs merges multiple ReviewRulesConfig into one
func MergeConfigs(configs ...*ReviewRulesConfig) *ReviewRulesConfig {
	if len(configs) == 0 {
		return &ReviewRulesConfig{
			Rules: []ReviewRuleConfig{},
		}
	}

	result := &ReviewRulesConfig{
		Version:  configs[0].Version,
		RuleBase: configs[0].RuleBase,
		Rules:    []ReviewRuleConfig{},
	}

	// Track IDs to avoid duplicates
	ids := make(map[string]bool)

	for _, config := range configs {
		for _, rule := range config.Rules {
			if ids[rule.ID] {
				logger.Warn("Duplicate rule ID found during merge, skipping",
					zap.String("id", rule.ID),
				)
				continue
			}
			ids[rule.ID] = true
			result.Rules = append(result.Rules, rule)
		}
	}

	return result
}

// allowedEnvVarPrefixes defines the allowed environment variable prefixes
// Only environment variables with these prefixes can be expanded in configuration files
// This is a security measure to prevent accidental exposure of sensitive system variables
var allowedEnvVarPrefixes = []string{
	"SCOPEVIEW_",     // VerustCode-specific variables
	"CI_",            // CI/CD environment variables
	"GITHUB_",        // GitHub Actions variables
	"GITLAB_",        // GitLab CI variables
	"CUSTOM_REVIEW_", // Custom review variables
}

// isAllowedEnvVar checks if an environment variable name is allowed to be expanded
func isAllowedEnvVar(varName string) bool {
	for _, prefix := range allowedEnvVarPrefixes {
		if strings.HasPrefix(varName, prefix) {
			return true
		}
	}
	// Log warning for blocked variable
	logger.Warn("Environment variable blocked by whitelist",
		zap.String("var_name", varName),
		zap.Strings("allowed_prefixes", allowedEnvVarPrefixes),
	)
	return false
}

// expandEnvVars replaces ${VAR_NAME} patterns with environment variable values
// Only variables matching the allowed prefixes will be expanded for security
// Only matches ${VAR_NAME} format (not $VAR_NAME) to avoid conflicts with special characters
func expandEnvVars(content string) string {
	// Match ${VAR_NAME} patterns only (not $VAR_NAME to avoid conflicts with special characters)
	re := regexp.MustCompile(`\$\{([^}]+)\}`)

	return re.ReplaceAllStringFunc(content, func(match string) string {
		// Extract variable name from ${VAR_NAME}
		varName := match[2 : len(match)-1]

		// Support default values: ${VAR_NAME:-default}
		parts := strings.SplitN(varName, ":-", 2)
		varName = parts[0]

		// Check if the variable is allowed by whitelist
		if !isAllowedEnvVar(varName) {
			// Return default value if provided, otherwise return the original match
			if len(parts) > 1 {
				return parts[1]
			}
			return match // Return original unexpanded variable
		}

		if value := os.Getenv(varName); value != "" {
			logger.Debug("Expanded environment variable",
				zap.String("var_name", varName),
			)
			return value
		}

		// Return default value if provided
		if len(parts) > 1 {
			return parts[1]
		}

		return ""
	})
}

// ValidateFile validates a DSL configuration file without loading it
func (l *Loader) ValidateFile(path string) error {
	_, err := l.Load(path)
	return err
}

// FindConfigFile searches for a configuration file in common locations
func FindConfigFile(name string) string {
	// Common locations to search
	candidates := []string{
		name,
		filepath.Join("config", name),
		filepath.Join(".", name),
		filepath.Join(os.Getenv("HOME"), ".config", "verustcode", name),
		filepath.Join("/etc", "verustcode", name),
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// LoadFromRepoRoot loads review configuration from .verust-review.yaml at repository root
// This has the highest priority in the configuration loading order.
// Returns nil if the config file does not exist.
func (l *Loader) LoadFromRepoRoot(repoPath string) (*ReviewRulesConfig, error) {
	rootPath := filepath.Join(repoPath, config.RepoRootReviewPath)

	// Check if root config exists
	if _, err := os.Stat(rootPath); os.IsNotExist(err) {
		logger.Debug("No review config found at repository root",
			zap.String("repo_path", repoPath),
			zap.String("expected_path", rootPath),
		)
		return nil, nil
	}

	logger.Info("Found review config at repository root",
		zap.String("path", rootPath),
	)

	return l.Load(rootPath)
}

// LoadFromRepoEmbedded loads review configuration from embedded .verustcode/review.yaml in repository
// Deprecated: Use LoadFromRepoRoot instead. This method is kept for backward compatibility.
// Returns nil if the embedded config file does not exist
func (l *Loader) LoadFromRepoEmbedded(repoPath string) (*ReviewRulesConfig, error) {
	embeddedPath := filepath.Join(repoPath, config.RepoEmbeddedReviewPath)

	// Check if embedded config exists
	if _, err := os.Stat(embeddedPath); os.IsNotExist(err) {
		logger.Debug("No embedded review config found in repository",
			zap.String("repo_path", repoPath),
			zap.String("expected_path", embeddedPath),
		)
		return nil, nil
	}

	logger.Info("Found embedded review config in repository",
		zap.String("path", embeddedPath),
	)

	return l.Load(embeddedPath)
}

// LoadReviewFile loads a specific review file from the reviews directory
func (l *Loader) LoadReviewFile(fileName string) (*ReviewRulesConfig, error) {
	filePath := filepath.Join(config.ReviewsDir, fileName)
	return l.Load(filePath)
}

// LoadDefaultReviewConfig loads the default review configuration (default.yaml)
func (l *Loader) LoadDefaultReviewConfig() (*ReviewRulesConfig, error) {
	return l.LoadReviewFile(config.DefaultReviewFile)
}

// ListReviewFiles lists all available review configuration files in the reviews directory
// Returns file names (not full paths), sorted alphabetically
func ListReviewFiles() ([]string, error) {
	reviewsDir := config.ReviewsDir

	// Check if directory exists
	if _, err := os.Stat(reviewsDir); os.IsNotExist(err) {
		logger.Warn("Reviews directory does not exist",
			zap.String("dir", reviewsDir),
		)
		return []string{}, nil
	}

	// Find all YAML files
	patterns := []string{"*.yaml", "*.yml"}
	var files []string

	for _, pattern := range patterns {
		matches, err := filepath.Glob(filepath.Join(reviewsDir, pattern))
		if err != nil {
			return nil, errors.Wrap(errors.ErrCodeInternal,
				"failed to glob reviews directory", err)
		}
		for _, match := range matches {
			// Skip example files
			baseName := filepath.Base(match)
			if strings.Contains(baseName, ".example.") {
				continue
			}
			files = append(files, baseName)
		}
	}

	// Sort alphabetically
	sort.Strings(files)

	logger.Debug("Listed review files",
		zap.String("dir", reviewsDir),
		zap.Strings("files", files),
	)

	return files, nil
}

// ReviewFileExists checks if a review file exists in the reviews directory
func ReviewFileExists(fileName string) bool {
	filePath := filepath.Join(config.ReviewsDir, fileName)
	_, err := os.Stat(filePath)
	return err == nil
}
