package config

import (
	"os"
	"testing"

	"golang.org/x/text/language"
)

// TestParseLanguage tests ParseLanguage function
func TestParseLanguage(t *testing.T) {
	tests := []struct {
		name        string
		langTag     string
		expectError bool
		checkTag    func(*testing.T, *LanguageConfig)
	}{
		{
			name:        "Valid English tag",
			langTag:     "en",
			expectError: false,
			checkTag: func(t *testing.T, lc *LanguageConfig) {
				if lc.String() != "en" {
					t.Errorf("Expected 'en', got '%s'", lc.String())
				}
			},
		},
		{
			name:        "Valid Chinese Simplified tag",
			langTag:     "zh-CN",
			expectError: false,
			checkTag: func(t *testing.T, lc *LanguageConfig) {
				if lc.String() != "zh-CN" {
					t.Errorf("Expected 'zh-CN', got '%s'", lc.String())
				}
			},
		},
		{
			name:        "Empty tag (should default to English)",
			langTag:     "",
			expectError: false,
			checkTag: func(t *testing.T, lc *LanguageConfig) {
				if lc == nil {
					t.Error("LanguageConfig should not be nil")
				}
				// Should default to English
				if lc.Tag() != language.English {
					t.Errorf("Expected English default, got %s", lc.Tag())
				}
			},
		},
		{
			name:        "Invalid tag (should default to English)",
			langTag:     "invalid-tag",
			expectError: false,
			checkTag: func(t *testing.T, lc *LanguageConfig) {
				// Should default to English
				if lc.Tag() != language.English {
					t.Errorf("Expected English default, got %s", lc.Tag())
				}
			},
		},
		{
			name:        "Uppercase tag",
			langTag:     "EN",
			expectError: false,
			checkTag: func(t *testing.T, lc *LanguageConfig) {
				if lc.String() != "en" {
					t.Errorf("Expected 'en', got '%s'", lc.String())
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lc, err := ParseLanguage(tt.langTag)
			if (err != nil) != tt.expectError {
				t.Errorf("ParseLanguage() error = %v, expectError = %v", err, tt.expectError)
				return
			}
			if lc != nil && tt.checkTag != nil {
				tt.checkTag(t, lc)
			}
		})
	}
}

// TestLanguageConfig_Tag tests Tag method
func TestLanguageConfig_Tag(t *testing.T) {
	lc, err := ParseLanguage("en")
	if err != nil {
		t.Fatalf("ParseLanguage failed: %v", err)
	}

	tag := lc.Tag()
	if tag != language.English {
		t.Errorf("Tag() = %v, want English", tag)
	}
}

// TestLanguageConfig_String tests String method
func TestLanguageConfig_String(t *testing.T) {
	tests := []struct {
		name     string
		langTag  string
		expected string
	}{
		{"English", "en", "en"},
		{"Chinese Simplified", "zh-CN", "zh-CN"},
		{"Japanese", "ja", "ja"},
		{"Korean", "ko", "ko"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lc, err := ParseLanguage(tt.langTag)
			if err != nil {
				t.Fatalf("ParseLanguage failed: %v", err)
			}
			if lc.String() != tt.expected {
				t.Errorf("String() = %s, want %s", lc.String(), tt.expected)
			}
		})
	}
}

// TestLanguageConfig_DisplayName tests DisplayName method
func TestLanguageConfig_DisplayName(t *testing.T) {
	tests := []struct {
		name     string
		langTag  string
		expected string
	}{
		{"English", "en", "en"},
		{"Chinese", "zh-CN", "zh"},
		{"Japanese", "ja", "ja"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lc, err := ParseLanguage(tt.langTag)
			if err != nil {
				t.Fatalf("ParseLanguage failed: %v", err)
			}
			displayName := lc.DisplayName()
			if displayName != tt.expected {
				t.Errorf("DisplayName() = %s, want %s", displayName, tt.expected)
			}
		})
	}
}

// TestLanguageConfig_PromptInstruction tests PromptInstruction method
func TestLanguageConfig_PromptInstruction(t *testing.T) {
	tests := []struct {
		name     string
		langTag  string
		expected string
	}{
		{"Chinese", "zh", "Chinese (Simplified Chinese preferred)"},
		{"English", "en", "English"},
		{"Japanese", "ja", "Japanese"},
		{"Korean", "ko", "Korean"},
		{"French", "fr", "French"},
		{"German", "de", "German"},
		{"Spanish", "es", "Spanish"},
		{"Portuguese", "pt", "Portuguese"},
		{"Russian", "ru", "Russian"},
		{"Arabic", "ar", "Arabic"},
		{"Unknown", "xyz", "English"}, // Invalid language tag defaults to English
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lc, err := ParseLanguage(tt.langTag)
			if err != nil {
				t.Fatalf("ParseLanguage failed: %v", err)
			}
			instruction := lc.PromptInstruction()
			if instruction != tt.expected {
				t.Errorf("PromptInstruction() = %s, want %s", instruction, tt.expected)
			}
		})
	}
}

// TestDetectSystemLanguage tests detectSystemLanguage function
func TestDetectSystemLanguage(t *testing.T) {
	// Save original environment
	originalLANG := os.Getenv("LANG")
	originalLANGUAGE := os.Getenv("LANGUAGE")
	originalLC_ALL := os.Getenv("LC_ALL")
	originalLC_MESSAGES := os.Getenv("LC_MESSAGES")

	// Restore environment after test
	defer func() {
		if originalLANG != "" {
			os.Setenv("LANG", originalLANG)
		} else {
			os.Unsetenv("LANG")
		}
		if originalLANGUAGE != "" {
			os.Setenv("LANGUAGE", originalLANGUAGE)
		} else {
			os.Unsetenv("LANGUAGE")
		}
		if originalLC_ALL != "" {
			os.Setenv("LC_ALL", originalLC_ALL)
		} else {
			os.Unsetenv("LC_ALL")
		}
		if originalLC_MESSAGES != "" {
			os.Setenv("LC_MESSAGES", originalLC_MESSAGES)
		} else {
			os.Unsetenv("LC_MESSAGES")
		}
	}()

	tests := []struct {
		name        string
		setEnv      map[string]string
		expectLang  language.Tag
		description string
	}{
		{
			name: "LANG set to en_US.UTF-8",
			setEnv: map[string]string{
				"LANG": "en_US.UTF-8",
			},
			expectLang:  language.English,
			description: "Should detect English from LANG",
		},
		{
			name: "LANG set to zh_CN.UTF-8",
			setEnv: map[string]string{
				"LANG": "zh_CN.UTF-8",
			},
			expectLang:  language.Chinese,
			description: "Should detect Chinese from LANG",
		},
		{
			name: "LANGUAGE set",
			setEnv: map[string]string{
				"LANGUAGE": "ja_JP.UTF-8",
			},
			expectLang:  language.Japanese,
			description: "Should detect Japanese from LANGUAGE",
		},
		{
			name:        "No env vars set",
			setEnv:      map[string]string{},
			expectLang:  language.English,
			description: "Should default to English",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all env vars first
			os.Unsetenv("LANG")
			os.Unsetenv("LANGUAGE")
			os.Unsetenv("LC_ALL")
			os.Unsetenv("LC_MESSAGES")

			// Set test env vars
			for key, value := range tt.setEnv {
				os.Setenv(key, value)
			}

			tag := detectSystemLanguage()
			base, _ := tag.Base()
			expectedBase, _ := tt.expectLang.Base()

			if base != expectedBase {
				t.Errorf("detectSystemLanguage() = %s, want %s (%s)", base, expectedBase, tt.description)
			}
		})
	}
}

// TestValidLanguageCodes tests ValidLanguageCodes function
func TestValidLanguageCodes(t *testing.T) {
	codes := ValidLanguageCodes()

	if len(codes) == 0 {
		t.Error("ValidLanguageCodes() should return non-empty slice")
	}

	// Check that common languages are included
	expectedCodes := []string{"en", "zh-cn", "ja", "ko"}
	for _, expected := range expectedCodes {
		found := false
		for _, code := range codes {
			if code == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("ValidLanguageCodes() missing expected code: %s", expected)
		}
	}
}

// TestGetOutputLanguage tests GetOutputLanguage method
func TestGetOutputLanguage(t *testing.T) {
	tests := []struct {
		name        string
		outputLang  string
		expectError bool
		checkResult func(*testing.T, *LanguageConfig)
	}{
		{
			name:        "Valid language",
			outputLang:  "en",
			expectError: false,
			checkResult: func(t *testing.T, lc *LanguageConfig) {
				if lc.String() != "en" {
					t.Errorf("Expected 'en', got '%s'", lc.String())
				}
			},
		},
		{
			name:        "Empty language",
			outputLang:  "",
			expectError: false,
			checkResult: func(t *testing.T, lc *LanguageConfig) {
				if lc == nil {
					t.Error("LanguageConfig should not be nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &ReviewConfig{
				OutputLanguage: tt.outputLang,
			}

			lc, err := cfg.GetOutputLanguage()
			if (err != nil) != tt.expectError {
				t.Errorf("GetOutputLanguage() error = %v, expectError = %v", err, tt.expectError)
				return
			}
			if lc != nil && tt.checkResult != nil {
				tt.checkResult(t, lc)
			}
		})
	}
}
