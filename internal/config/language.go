// Package config provides configuration management for the application.
package config

import (
	"os"
	"strings"

	"golang.org/x/text/language"
)

// LanguageConfig provides language-related configuration utilities
type LanguageConfig struct {
	tag language.Tag
}

// ParseLanguage parses and validates an ISO language tag.
// If the tag is empty, it defaults to English.
// Returns the validated language tag or defaults to English if invalid.
func ParseLanguage(langTag string) (*LanguageConfig, error) {
	var tag language.Tag
	var err error

	if langTag == "" {
		// Default to English if no language specified
		tag = language.English
	} else {
		// Parse the provided language tag
		tag, err = language.Parse(langTag)
		if err != nil {
			// Try to match with common language codes
			tag, err = language.Parse(strings.ToLower(langTag))
			if err != nil {
				// Default to English if parsing fails
				tag = language.English
			}
		}
	}

	return &LanguageConfig{tag: tag}, nil
}

// Tag returns the underlying language tag
func (lc *LanguageConfig) Tag() language.Tag {
	return lc.tag
}

// String returns the language tag as a string (e.g., "en", "zh-CN")
func (lc *LanguageConfig) String() string {
	return lc.tag.String()
}

// DisplayName returns the display name of the language in English
// (e.g., "English", "Chinese")
func (lc *LanguageConfig) DisplayName() string {
	base, _ := lc.tag.Base()
	return base.String()
}

// PromptInstruction returns the language name to add to prompts
func (lc *LanguageConfig) PromptInstruction() string {
	// Map common language tags to human-readable language names
	base, _ := lc.tag.Base()
	switch base.String() {
	case "zh":
		return "Chinese (Simplified Chinese preferred)"
	case "en":
		return "English"
	case "ja":
		return "Japanese"
	case "ko":
		return "Korean"
	case "fr":
		return "French"
	case "de":
		return "German"
	case "es":
		return "Spanish"
	case "pt":
		return "Portuguese"
	case "ru":
		return "Russian"
	case "ar":
		return "Arabic"
	default:
		// For other languages, use the tag directly
		return lc.tag.String()
	}
}

// detectSystemLanguage attempts to detect the system language from environment variables
func detectSystemLanguage() language.Tag {
	// Check common environment variables for language setting
	envVars := []string{"LANG", "LANGUAGE", "LC_ALL", "LC_MESSAGES"}

	for _, envVar := range envVars {
		if val := os.Getenv(envVar); val != "" {
			// Parse the locale string (e.g., "en_US.UTF-8" -> "en")
			langPart := strings.Split(val, ".")[0]            // Remove encoding
			langPart = strings.Replace(langPart, "_", "-", 1) // Convert to BCP 47 format

			if tag, err := language.Parse(langPart); err == nil {
				return tag
			}
		}
	}

	// Default to English if no system language detected
	return language.English
}

// GetOutputLanguage returns the configured output language or detects from system
func (c *ReviewConfig) GetOutputLanguage() (*LanguageConfig, error) {
	return ParseLanguage(c.OutputLanguage)
}

// ValidLanguageCodes returns a list of commonly supported language codes
func ValidLanguageCodes() []string {
	return []string{
		"en",    // English
		"zh-cn", // Simplified Chinese
		"zh-tw", // Traditional Chinese
		"ja",    // Japanese
		"ko",    // Korean
		"fr",    // French
		"de",    // German
		"es",    // Spanish
		"pt",    // Portuguese
		"ru",    // Russian
		"ar",    // Arabic
		"it",    // Italian
		"nl",    // Dutch
		"pl",    // Polish
		"tr",    // Turkish
		"vi",    // Vietnamese
		"th",    // Thai
		"id",    // Indonesian
	}
}
