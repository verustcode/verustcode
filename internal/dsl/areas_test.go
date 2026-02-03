package dsl

import (
	"testing"
)

// TestGetAreasByGroup tests the GetAreasByGroup function
func TestGetAreasByGroup(t *testing.T) {
	tests := []struct {
		name     string
		group    AreaGroup
		minCount int // Minimum expected count (to avoid brittle tests)
	}{
		{
			name:     "code quality group",
			group:    AreaGroupCodeQuality,
			minCount: 10, // At least 10 areas in code quality
		},
		{
			name:     "security group",
			group:    AreaGroupSecurity,
			minCount: 10, // At least 10 areas in security
		},
		{
			name:     "performance group",
			group:    AreaGroupPerformance,
			minCount: 5, // At least 5 areas in performance
		},
		{
			name:     "backdoor group",
			group:    AreaGroupBackdoor,
			minCount: 5, // At least 5 areas in backdoor
		},
		{
			name:     "testing group",
			group:    AreaGroupTesting,
			minCount: 1, // At least 1 area in testing
		},
		{
			name:     "architecture group",
			group:    AreaGroupArchitecture,
			minCount: 5, // At least 5 areas in architecture
		},
		{
			name:     "compliance group",
			group:    AreaGroupCompliance,
			minCount: 3, // At least 3 areas in compliance
		},
		{
			name:     "documentation group",
			group:    AreaGroupDocumentation,
			minCount: 1, // At least 1 area in documentation
		},
		{
			name:     "non-existent group",
			group:    AreaGroup("non-existent"),
			minCount: 0, // Should return empty slice
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			areas := GetAreasByGroup(tt.group)
			if len(areas) < tt.minCount {
				t.Errorf("GetAreasByGroup(%v) returned %d areas, want at least %d",
					tt.group, len(areas), tt.minCount)
			}

			// Verify all returned areas belong to the correct group
			for _, area := range areas {
				if area.Group != tt.group {
					t.Errorf("Area %s has group %v, want %v", area.ID, area.Group, tt.group)
				}
			}
		})
	}
}

// TestGetAreaDescription tests the GetAreaDescription function
func TestGetAreaDescription(t *testing.T) {
	tests := []struct {
		name       string
		areaID     string
		wantDesc   string
		wantExists bool
	}{
		{
			name:       "valid area - business-logic",
			areaID:     "business-logic",
			wantDesc:   "Business logic correctness",
			wantExists: true,
		},
		{
			name:       "valid area - security-vulnerabilities",
			areaID:     "security-vulnerabilities",
			wantDesc:   "Security vulnerabilities",
			wantExists: true,
		},
		{
			name:       "valid area - performance",
			areaID:     "performance",
			wantDesc:   "Performance optimization",
			wantExists: true,
		},
		{
			name:       "non-existent area",
			areaID:     "non-existent-area",
			wantDesc:   "",
			wantExists: false,
		},
		{
			name:       "empty area ID",
			areaID:     "",
			wantDesc:   "",
			wantExists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			desc, exists := GetAreaDescription(tt.areaID)
			if exists != tt.wantExists {
				t.Errorf("GetAreaDescription(%q) exists = %v, want %v",
					tt.areaID, exists, tt.wantExists)
			}
			if desc != tt.wantDesc {
				t.Errorf("GetAreaDescription(%q) = %q, want %q",
					tt.areaID, desc, tt.wantDesc)
			}
		})
	}
}

// TestGetAreaGroup tests the GetAreaGroup function
func TestGetAreaGroup(t *testing.T) {
	tests := []struct {
		name       string
		areaID     string
		wantGroup  AreaGroup
		wantExists bool
	}{
		{
			name:       "code quality area",
			areaID:     "business-logic",
			wantGroup:  AreaGroupCodeQuality,
			wantExists: true,
		},
		{
			name:       "security area",
			areaID:     "injection-attacks",
			wantGroup:  AreaGroupSecurity,
			wantExists: true,
		},
		{
			name:       "performance area",
			areaID:     "memory-usage",
			wantGroup:  AreaGroupPerformance,
			wantExists: true,
		},
		{
			name:       "backdoor area",
			areaID:     "malicious-code",
			wantGroup:  AreaGroupBackdoor,
			wantExists: true,
		},
		{
			name:       "testing area",
			areaID:     "test-suggestions",
			wantGroup:  AreaGroupTesting,
			wantExists: true,
		},
		{
			name:       "architecture area",
			areaID:     "technical-debt",
			wantGroup:  AreaGroupArchitecture,
			wantExists: true,
		},
		{
			name:       "compliance area",
			areaID:     "license-compliance",
			wantGroup:  AreaGroupCompliance,
			wantExists: true,
		},
		{
			name:       "documentation area",
			areaID:     "documentation",
			wantGroup:  AreaGroupDocumentation,
			wantExists: true,
		},
		{
			name:       "non-existent area",
			areaID:     "non-existent",
			wantGroup:  "",
			wantExists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			group, exists := GetAreaGroup(tt.areaID)
			if exists != tt.wantExists {
				t.Errorf("GetAreaGroup(%q) exists = %v, want %v",
					tt.areaID, exists, tt.wantExists)
			}
			if group != tt.wantGroup {
				t.Errorf("GetAreaGroup(%q) = %v, want %v",
					tt.areaID, group, tt.wantGroup)
			}
		})
	}
}

// TestIsValidArea tests the IsValidArea function
func TestIsValidArea(t *testing.T) {
	tests := []struct {
		name   string
		areaID string
		want   bool
	}{
		{
			name:   "valid area - business-logic",
			areaID: "business-logic",
			want:   true,
		},
		{
			name:   "valid area - xss",
			areaID: "xss",
			want:   true,
		},
		{
			name:   "valid area - caching",
			areaID: "caching",
			want:   true,
		},
		{
			name:   "non-existent area",
			areaID: "non-existent-area",
			want:   false,
		},
		{
			name:   "empty area ID",
			areaID: "",
			want:   false,
		},
		{
			name:   "similar but invalid area",
			areaID: "business_logic", // underscore instead of hyphen
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidArea(tt.areaID)
			if got != tt.want {
				t.Errorf("IsValidArea(%q) = %v, want %v", tt.areaID, got, tt.want)
			}
		})
	}
}

// TestGetAllGroups tests the GetAllGroups function
func TestGetAllGroups(t *testing.T) {
	groups := GetAllGroups()

	// Verify expected number of groups
	expectedGroups := 8
	if len(groups) != expectedGroups {
		t.Errorf("GetAllGroups() returned %d groups, want %d", len(groups), expectedGroups)
	}

	// Verify all expected groups are present
	expectedGroupSet := map[AreaGroup]bool{
		AreaGroupCodeQuality:   false,
		AreaGroupSecurity:      false,
		AreaGroupPerformance:   false,
		AreaGroupBackdoor:      false,
		AreaGroupTesting:       false,
		AreaGroupArchitecture:  false,
		AreaGroupCompliance:    false,
		AreaGroupDocumentation: false,
	}

	for _, group := range groups {
		if _, exists := expectedGroupSet[group]; !exists {
			t.Errorf("GetAllGroups() returned unexpected group: %v", group)
		}
		expectedGroupSet[group] = true
	}

	// Check all expected groups were found
	for group, found := range expectedGroupSet {
		if !found {
			t.Errorf("GetAllGroups() missing expected group: %v", group)
		}
	}
}

// TestAllAreasInitialization tests that AllAreas is properly initialized
func TestAllAreasInitialization(t *testing.T) {
	// Verify AllAreas is not empty
	if len(AllAreas) == 0 {
		t.Error("AllAreas is empty")
	}

	// Verify each area has required fields
	for i, area := range AllAreas {
		if area.ID == "" {
			t.Errorf("AllAreas[%d] has empty ID", i)
		}
		if area.Group == "" {
			t.Errorf("AllAreas[%d] (%s) has empty Group", i, area.ID)
		}
		if area.Description == "" {
			t.Errorf("AllAreas[%d] (%s) has empty Description", i, area.ID)
		}
	}

	// Verify no duplicate IDs
	seenIDs := make(map[string]bool)
	for _, area := range AllAreas {
		if seenIDs[area.ID] {
			t.Errorf("Duplicate area ID found: %s", area.ID)
		}
		seenIDs[area.ID] = true
	}
}

// TestAreaMapsConsistency tests that the mapping tables are consistent
func TestAreaMapsConsistency(t *testing.T) {
	// Verify AreaDescriptions and AreaGroups have the same number of entries as AllAreas
	if len(AreaDescriptions) != len(AllAreas) {
		t.Errorf("AreaDescriptions has %d entries, AllAreas has %d",
			len(AreaDescriptions), len(AllAreas))
	}

	if len(AreaGroups) != len(AllAreas) {
		t.Errorf("AreaGroups has %d entries, AllAreas has %d",
			len(AreaGroups), len(AllAreas))
	}

	// Verify consistency between AllAreas and maps
	for _, area := range AllAreas {
		// Check AreaDescriptions
		if desc, exists := AreaDescriptions[area.ID]; !exists {
			t.Errorf("Area %s not found in AreaDescriptions", area.ID)
		} else if desc != area.Description {
			t.Errorf("AreaDescriptions[%s] = %q, want %q",
				area.ID, desc, area.Description)
		}

		// Check AreaGroups
		if group, exists := AreaGroups[area.ID]; !exists {
			t.Errorf("Area %s not found in AreaGroups", area.ID)
		} else if group != area.Group {
			t.Errorf("AreaGroups[%s] = %v, want %v",
				area.ID, group, area.Group)
		}

		// Check AreasByGroup
		found := false
		for _, groupArea := range AreasByGroup[area.Group] {
			if groupArea.ID == area.ID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Area %s not found in AreasByGroup[%v]",
				area.ID, area.Group)
		}
	}
}
