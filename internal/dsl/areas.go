// Package dsl provides DSL configuration parsing and validation.
// This file defines all available review areas and their groupings.
package dsl

// AreaGroup defines the category grouping for areas
type AreaGroup string

const (
	// AreaGroupCodeQuality focuses on code-level quality issues
	AreaGroupCodeQuality AreaGroup = "code-quality"

	// AreaGroupSecurity focuses on security vulnerabilities and risks
	AreaGroupSecurity AreaGroup = "security"

	// AreaGroupPerformance focuses on performance and efficiency optimization
	AreaGroupPerformance AreaGroup = "performance"

	// AreaGroupBackdoor focuses on malicious code detection
	AreaGroupBackdoor AreaGroup = "backdoor"

	// AreaGroupTesting focuses on testing-related suggestions
	AreaGroupTesting AreaGroup = "testing"

	// AreaGroupArchitecture focuses on architecture design and technical debt
	AreaGroupArchitecture AreaGroup = "architecture"

	// AreaGroupCompliance focuses on regulatory and license compliance
	AreaGroupCompliance AreaGroup = "compliance"

	// AreaGroupDocumentation focuses on code documentation quality
	AreaGroupDocumentation AreaGroup = "documentation"
)

// AreaDefinition defines information for a single area
type AreaDefinition struct {
	ID          string    // Area identifier (kebab-case format)
	Group       AreaGroup // Group this area belongs to
	Description string    // Concise core description
}

// AllAreas contains all available area definitions (organized by group)
var AllAreas = []AreaDefinition{
	// ==================== Code Quality ====================
	{ID: "business-logic", Group: AreaGroupCodeQuality, Description: "Business logic correctness"},
	{ID: "logic-errors", Group: AreaGroupCodeQuality, Description: "Logic error detection"},
	{ID: "edge-cases", Group: AreaGroupCodeQuality, Description: "Edge case handling"},
	{ID: "runtime-safety", Group: AreaGroupCodeQuality, Description: "Runtime safety"},
	{ID: "concurrency", Group: AreaGroupCodeQuality, Description: "Concurrency issues"},
	{ID: "code-style", Group: AreaGroupCodeQuality, Description: "Code style consistency"},
	{ID: "consistency", Group: AreaGroupCodeQuality, Description: "Code consistency"},
	{ID: "readability", Group: AreaGroupCodeQuality, Description: "Code readability"},
	{ID: "complexity", Group: AreaGroupCodeQuality, Description: "Code complexity"},
	{ID: "error-handling", Group: AreaGroupCodeQuality, Description: "Error handling"},
	{ID: "testability", Group: AreaGroupCodeQuality, Description: "Testability"},

	// ==================== Security ====================
	{ID: "security-vulnerabilities", Group: AreaGroupSecurity, Description: "Security vulnerabilities"},
	{ID: "injection-attacks", Group: AreaGroupSecurity, Description: "Injection attacks (SQL, command, code injection, etc.)"},
	{ID: "authentication", Group: AreaGroupSecurity, Description: "Authentication"},
	{ID: "authorization", Group: AreaGroupSecurity, Description: "Authorization"},
	{ID: "sensitive-data", Group: AreaGroupSecurity, Description: "Sensitive data protection"},
	{ID: "cryptography", Group: AreaGroupSecurity, Description: "Cryptography security"},
	{ID: "input-validation", Group: AreaGroupSecurity, Description: "Input validation"},
	{ID: "xss", Group: AreaGroupSecurity, Description: "Cross-site scripting (XSS)"},
	{ID: "csrf", Group: AreaGroupSecurity, Description: "Cross-site request forgery (CSRF)"},
	{ID: "secrets-management", Group: AreaGroupSecurity, Description: "Secrets management"},
	{ID: "secure-coding-practices", Group: AreaGroupSecurity, Description: "Secure coding practices"},
	{ID: "ssrf", Group: AreaGroupSecurity, Description: "Server-side request forgery (SSRF)"},
	{ID: "deserialization", Group: AreaGroupSecurity, Description: "Insecure deserialization"},
	{ID: "security-misconfiguration", Group: AreaGroupSecurity, Description: "Security misconfiguration"},
	{ID: "vulnerable-components", Group: AreaGroupSecurity, Description: "Vulnerable and outdated components"},
	{ID: "logging-monitoring", Group: AreaGroupSecurity, Description: "Security logging and monitoring"},

	// ==================== Performance ====================
	{ID: "performance", Group: AreaGroupPerformance, Description: "Performance optimization"},
	{ID: "efficiency", Group: AreaGroupPerformance, Description: "Efficiency improvement"},
	{ID: "memory-usage", Group: AreaGroupPerformance, Description: "Memory usage"},
	{ID: "cpu-usage", Group: AreaGroupPerformance, Description: "CPU usage"},
	{ID: "io-optimization", Group: AreaGroupPerformance, Description: "I/O optimization"},
	{ID: "database-optimization", Group: AreaGroupPerformance, Description: "Database optimization"},
	{ID: "caching", Group: AreaGroupPerformance, Description: "Caching strategy"},
	{ID: "algorithm-efficiency", Group: AreaGroupPerformance, Description: "Algorithm time complexity"},
	{ID: "n-plus-one-query", Group: AreaGroupPerformance, Description: "N+1 query detection"},
	{ID: "resource-leak", Group: AreaGroupPerformance, Description: "Resource leak (connections, file handles)"},

	// ==================== Backdoor Detection ====================
	{ID: "backdoor-detection", Group: AreaGroupBackdoor, Description: "Backdoor detection"},
	{ID: "malicious-code", Group: AreaGroupBackdoor, Description: "Malicious code"},
	{ID: "obfuscation", Group: AreaGroupBackdoor, Description: "Code obfuscation"},
	{ID: "hidden-access", Group: AreaGroupBackdoor, Description: "Hidden access"},
	{ID: "suspicious-patterns", Group: AreaGroupBackdoor, Description: "Suspicious patterns"},
	{ID: "unauthorized-access", Group: AreaGroupBackdoor, Description: "Unauthorized access"},

	// ==================== Testing ====================
	{ID: "test-suggestions", Group: AreaGroupTesting, Description: "Testing suggestions"},

	// ==================== Architecture & Design ====================
	{ID: "technical-debt", Group: AreaGroupArchitecture, Description: "Technical debt identification"},
	{ID: "architecture-design", Group: AreaGroupArchitecture, Description: "Architecture design"},
	{ID: "design-patterns", Group: AreaGroupArchitecture, Description: "Design patterns"},
	{ID: "modularity", Group: AreaGroupArchitecture, Description: "Modularity design"},
	{ID: "scalability", Group: AreaGroupArchitecture, Description: "Scalability"},
	{ID: "maintainability", Group: AreaGroupArchitecture, Description: "Maintainability"},

	// ==================== Compliance ====================
	{ID: "compliance", Group: AreaGroupCompliance, Description: "Compliance checking"},
	{ID: "license-compliance", Group: AreaGroupCompliance, Description: "License compliance"},
	{ID: "regulatory-compliance", Group: AreaGroupCompliance, Description: "Regulatory compliance"},
	{ID: "data-privacy", Group: AreaGroupCompliance, Description: "Data privacy compliance"},

	// ==================== Documentation ====================
	{ID: "documentation", Group: AreaGroupDocumentation, Description: "Documentation quality (comments, API docs, naming clarity)"},
}

// AreasByGroup maps area groups to their areas
// key: AreaGroup, value: all AreaDefinitions in that group
var AreasByGroup map[AreaGroup][]AreaDefinition

// AreaDescriptions maps area IDs to their descriptions for quick lookup
// key: area ID, value: area description
var AreaDescriptions map[string]string

// AreaGroups maps area IDs to their groups for quick lookup
// key: area ID, value: AreaGroup
var AreaGroups map[string]AreaGroup

// init initializes the mapping tables
func init() {
	// Initialize maps
	AreasByGroup = make(map[AreaGroup][]AreaDefinition)
	AreaDescriptions = make(map[string]string)
	AreaGroups = make(map[string]AreaGroup)

	// Build mapping tables by iterating through all areas
	for _, area := range AllAreas {
		// Group areas by category
		AreasByGroup[area.Group] = append(AreasByGroup[area.Group], area)

		// Build ID -> Description mapping
		AreaDescriptions[area.ID] = area.Description

		// Build ID -> Group mapping
		AreaGroups[area.ID] = area.Group
	}
}

// GetAreasByGroup returns all areas in the specified group
func GetAreasByGroup(group AreaGroup) []AreaDefinition {
	return AreasByGroup[group]
}

// GetAreaDescription returns the description for the specified area
func GetAreaDescription(areaID string) (string, bool) {
	desc, exists := AreaDescriptions[areaID]
	return desc, exists
}

// GetAreaGroup returns the group for the specified area
func GetAreaGroup(areaID string) (AreaGroup, bool) {
	group, exists := AreaGroups[areaID]
	return group, exists
}

// IsValidArea checks if the specified area ID is valid (defined)
func IsValidArea(areaID string) bool {
	_, exists := AreaDescriptions[areaID]
	return exists
}

// GetAllGroups returns all area groups
func GetAllGroups() []AreaGroup {
	return []AreaGroup{
		AreaGroupCodeQuality,
		AreaGroupSecurity,
		AreaGroupPerformance,
		AreaGroupBackdoor,
		AreaGroupTesting,
		AreaGroupArchitecture,
		AreaGroupCompliance,
		AreaGroupDocumentation,
	}
}
