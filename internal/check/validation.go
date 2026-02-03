package check

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/fatih/color"
	"gopkg.in/yaml.v3"

	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/internal/dsl"
	"github.com/verustcode/verustcode/internal/llm"

	// Import LLM client implementations to register their factories
	_ "github.com/verustcode/verustcode/internal/llm/cursor"
	_ "github.com/verustcode/verustcode/internal/llm/gemini"
	_ "github.com/verustcode/verustcode/internal/llm/qoder"
)

// ValidationResult represents the result of a config validation
type ValidationResult struct {
	Path      string
	Valid     bool
	RuleCount int // for rules file
	Error     error
	Warnings  []string
}

// AgentValidationResult represents the result of an agent availability check
type AgentValidationResult struct {
	AgentName    string
	CLIAvailable bool
	APIKeySet    bool
	CLIPath      string
	Error        error
}

// validateConfigs validates all configuration files and returns error if agent validation fails
func (c *Checker) validateConfigs() error {
	// Validate bootstrap.yaml
	bootstrapResult := c.validateBootstrapYaml()
	c.report.AddValidationResult(bootstrapResult)
	printValidationResult(bootstrapResult)

	if !bootstrapResult.Valid {
		return fmt.Errorf("bootstrap.yaml validation failed: %w", bootstrapResult.Error)
	}

	// Validate default.yaml (review rules) and agents
	rulesResult, agentResults := c.validateRulesYaml()
	c.report.AddValidationResult(rulesResult)
	printValidationResult(rulesResult)

	// Print and check agent validation results
	if len(agentResults) > 0 {
		c.printAgentValidationResults(agentResults)
		for _, ar := range agentResults {
			if ar.Error != nil {
				return fmt.Errorf("agent '%s' validation failed: %w", ar.AgentName, ar.Error)
			}
		}
	}

	if !rulesResult.Valid {
		return fmt.Errorf("default.yaml validation failed: %w", rulesResult.Error)
	}

	// Validate schemas
	schemaResult := c.validateSchemas()
	c.report.AddValidationResult(schemaResult)
	printValidationResult(schemaResult)

	if !schemaResult.Valid {
		return fmt.Errorf("schema validation failed: %w", schemaResult.Error)
	}

	return nil
}

// validateBootstrapYaml validates the bootstrap configuration file
func (c *Checker) validateBootstrapYaml() ValidationResult {
	path := c.BootstrapPath()
	result := ValidationResult{Path: path}

	// Check if file exists
	if !fileExists(path) {
		result.Valid = false
		result.Error = fmt.Errorf("file does not exist")
		return result
	}

	// Try to load the bootstrap config
	_, err := config.LoadBootstrap(path)
	if err != nil {
		result.Valid = false
		result.Error = fmt.Errorf("format error: %v", err)
		return result
	}

	result.Valid = true
	return result
}

// validateConfigYaml validates the main configuration file (legacy, for migration)
func (c *Checker) validateConfigYaml() ValidationResult {
	path := c.ConfigPath()
	result := ValidationResult{Path: path}

	// Check if file exists
	if !fileExists(path) {
		result.Valid = false
		result.Error = fmt.Errorf("file does not exist")
		return result
	}

	// Try to load the config
	_, err := config.Load(path)
	if err != nil {
		result.Valid = false
		result.Error = fmt.Errorf("format error: %v", err)
		return result
	}

	result.Valid = true
	return result
}

// validateRulesYaml validates the review rules configuration file and agents
func (c *Checker) validateRulesYaml() (ValidationResult, []AgentValidationResult) {
	path := c.ReviewPath()
	result := ValidationResult{Path: path}

	// Check if file exists
	if !fileExists(path) {
		result.Valid = false
		result.Error = fmt.Errorf("file does not exist")
		return result, nil
	}

	// Use DSL loader for validation (strict mode)
	loader := dsl.NewStrictLoader()
	rulesConfig, err := loader.Load(path)
	if err != nil {
		result.Valid = false
		result.Error = fmt.Errorf("format error: %v", err)
		return result, nil
	}

	result.Valid = true
	result.RuleCount = len(rulesConfig.Rules)

	// Extract agents used in DSL (for info purposes)
	usedAgents := c.extractAgentNames(rulesConfig)
	if len(usedAgents) > 0 {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("Uses agents: %v - configure them via admin interface", usedAgents))
	}

	// Agent validation is now done at runtime since config is in database
	// Skip agent validation during startup check
	return result, nil
}

// extractAgentNames extracts unique agent names from DSL configuration
func (c *Checker) extractAgentNames(rulesConfig *dsl.ReviewRulesConfig) []string {
	agentSet := make(map[string]struct{})

	// Get default agent from rule_base
	defaultAgent := rulesConfig.RuleBase.Agent.GetType()

	// If default agent is set, add it to the set
	if defaultAgent != "" {
		agentSet[defaultAgent] = struct{}{}
	}

	// Iterate rules to collect agents
	for _, rule := range rulesConfig.Rules {
		agentType := rule.Agent.GetType()
		if agentType != "" {
			// Rule has explicit agent override
			agentSet[agentType] = struct{}{}
		}
		// If rule doesn't have agent, it uses defaultAgent (already added above)
	}

	// Convert set to slice
	agents := make([]string, 0, len(agentSet))
	for agent := range agentSet {
		agents = append(agents, agent)
	}

	return agents
}

// validateAgents validates availability of all used agents
func (c *Checker) validateAgents(agentNames []string, cfg *config.Config) []AgentValidationResult {
	results := make([]AgentValidationResult, 0, len(agentNames))

	for _, agentName := range agentNames {
		result := AgentValidationResult{
			AgentName: agentName,
		}

		// Get agent configuration from database settings
		agentDetail := cfg.GetAgent(agentName)
		if agentDetail == nil {
			result.Error = fmt.Errorf("agent '%s' is used in review rules but not configured in settings", agentName)
			results = append(results, result)
			continue
		}

		// Check CLI path
		cliPath := agentDetail.CLIPath
		if cliPath == "" {
			// Use default CLI name
			cliPath = agentName
		}
		result.CLIPath = cliPath

		// Check if CLI tool is available
		resolvedPath, err := exec.LookPath(cliPath)
		if err != nil {
			// Try with default name pattern (e.g., "cursor-agent" for cursor)
			defaultName := getDefaultCLIName(agentName)
			resolvedPath, err = exec.LookPath(defaultName)
			if err != nil {
				result.CLIAvailable = false
				result.Error = fmt.Errorf("CLI tool not found: '%s' (tried: %s, %s)", agentName, cliPath, defaultName)
				results = append(results, result)
				continue
			}
			result.CLIPath = resolvedPath
		} else {
			result.CLIPath = resolvedPath
		}
		result.CLIAvailable = true

		// Check if API key is configured
		apiKey := agentDetail.APIKey
		if apiKey == "" {
			result.APIKeySet = false
			result.Error = fmt.Errorf("API key not configured for agent '%s' in settings", agentName)
			results = append(results, result)
			continue
		}
		result.APIKeySet = true

		// Additional validation: try to create LLM client to verify configuration
		llmConfig := llm.NewClientConfig(agentName)
		llmConfig.CLIPath = result.CLIPath
		llmConfig.APIKey = apiKey

		client, err := llm.Create(agentName, llmConfig)
		if err != nil {
			result.Error = fmt.Errorf("failed to create LLM client for agent '%s': %v", agentName, err)
			results = append(results, result)
			continue
		}
		client.Close()

		// All checks passed
		results = append(results, result)
	}

	return results
}

// getDefaultCLIName returns the default CLI command name for an agent
func getDefaultCLIName(agentName string) string {
	switch agentName {
	case "cursor":
		return "cursor-agent"
	case "gemini":
		return "gemini"
	case "qoder":
		return "qodercli"
	default:
		return agentName
	}
}

// printAgentValidationResults prints agent validation results
func (c *Checker) printAgentValidationResults(results []AgentValidationResult) {
	green := color.New(color.FgGreen)
	red := color.New(color.FgRed)

	fmt.Println()
	fmt.Println("Agent Availability:")

	for _, r := range results {
		if r.Error != nil {
			red.Printf("  ✗ %s: %v\n", r.AgentName, r.Error)
		} else {
			green.Printf("  ✓ %s (CLI: %s, API Key: configured)\n", r.AgentName, r.CLIPath)
		}
	}
}

// validateSchemas validates the embedded default JSON schema
// Since schemas are now embedded in code (dsl.GetDefaultJSONSchema()), this validates
// that the embedded schema is properly structured and can be used for output formatting.
func (c *Checker) validateSchemas() ValidationResult {
	result := ValidationResult{Path: "embedded:default-schema"}

	// Get the embedded default schema
	schema := dsl.GetDefaultJSONSchema()
	if schema == nil {
		result.Valid = false
		result.Error = fmt.Errorf("embedded default schema is nil")
		return result
	}

	// Validate schema structure
	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		result.Valid = false
		result.Error = fmt.Errorf("embedded schema missing 'properties'")
		return result
	}

	// Check required top-level properties
	if _, ok := properties["summary"]; !ok {
		result.Valid = false
		result.Error = fmt.Errorf("embedded schema missing 'summary' property")
		return result
	}

	findings, ok := properties["findings"].(map[string]interface{})
	if !ok {
		result.Valid = false
		result.Error = fmt.Errorf("embedded schema missing 'findings' property")
		return result
	}

	// Check findings has items with properties
	items, ok := findings["items"].(map[string]interface{})
	if !ok {
		result.Valid = false
		result.Error = fmt.Errorf("embedded schema 'findings' missing 'items'")
		return result
	}

	if _, ok := items["properties"].(map[string]interface{}); !ok {
		result.Valid = false
		result.Error = fmt.Errorf("embedded schema 'findings.items' missing 'properties'")
		return result
	}

	result.Valid = true
	return result
}

// validateYamlSyntax validates YAML syntax
func validateYamlSyntax(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("cannot read file: %w", err)
	}

	var content interface{}
	if err := yaml.Unmarshal(data, &content); err != nil {
		return fmt.Errorf("YAML syntax error: %w", err)
	}

	return nil
}

// printValidationResult prints the validation result
func printValidationResult(result ValidationResult) {
	green := color.New(color.FgGreen)
	red := color.New(color.FgRed)
	yellow := color.New(color.FgYellow)

	if result.Valid {
		if result.RuleCount > 0 {
			green.Printf("  ✓ %s (%d rules)\n", result.Path, result.RuleCount)
		} else {
			green.Printf("  ✓ %s\n", result.Path)
		}
	} else if result.Error != nil {
		red.Printf("  ✗ %s: %v\n", result.Path, result.Error)
	} else {
		yellow.Printf("  ⚠ %s\n", result.Path)
	}

	// Print warnings if any
	for _, warning := range result.Warnings {
		yellow.Printf("    └─ %s\n", warning)
	}
}
