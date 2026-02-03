package check

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
)

// Report collects and displays check results
type Report struct {
	FileResults       []FileCheckResult
	ValidationResults []ValidationResult
}

// NewReport creates a new report
func NewReport() *Report {
	return &Report{
		FileResults:       make([]FileCheckResult, 0),
		ValidationResults: make([]ValidationResult, 0),
	}
}

// AddFileResult adds a file check result
func (r *Report) AddFileResult(result FileCheckResult) {
	r.FileResults = append(r.FileResults, result)
}

// AddValidationResult adds a validation result
func (r *Report) AddValidationResult(result ValidationResult) {
	r.ValidationResults = append(r.ValidationResults, result)
}

// Print prints the final summary report
func (r *Report) Print() {
	// Print separator
	r.printSeparator()

	// Calculate summary
	summary := r.calculateSummary()

	// Print final status
	r.printSummary(summary)
}

// ReportSummary holds the summary statistics
type ReportSummary struct {
	TotalFiles       int
	FilesExist       int
	FilesCreated     int
	FilesMissing     int
	TotalValidations int
	ValidationsValid int
	ValidationErrors int
	HasErrors        bool
	HasWarnings      bool
}

// calculateSummary calculates the summary from all results
func (r *Report) calculateSummary() ReportSummary {
	summary := ReportSummary{}

	// File results
	summary.TotalFiles = len(r.FileResults)
	for _, result := range r.FileResults {
		if result.Exists || result.Created {
			if result.Created {
				summary.FilesCreated++
			}
			summary.FilesExist++
		} else {
			summary.FilesMissing++
		}
		if result.Error != nil {
			summary.HasErrors = true
		}
	}

	// Validation results
	summary.TotalValidations = len(r.ValidationResults)
	for _, result := range r.ValidationResults {
		if result.Valid {
			summary.ValidationsValid++
		} else {
			summary.ValidationErrors++
			if result.Error != nil {
				summary.HasErrors = true
			}
		}
		if len(result.Warnings) > 0 {
			summary.HasWarnings = true
		}
	}

	return summary
}

// printSeparator prints a separator line
func (r *Report) printSeparator() {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))
	fmt.Println(style.Render(strings.Repeat("─", 50)))
}

// printSummary prints the final summary
func (r *Report) printSummary(summary ReportSummary) {
	green := color.New(color.FgGreen, color.Bold)
	yellow := color.New(color.FgYellow, color.Bold)
	red := color.New(color.FgRed, color.Bold)

	// Determine overall status
	if summary.HasErrors {
		red.Print("✗ Check completed")
	} else if summary.HasWarnings || summary.FilesMissing > 0 {
		yellow.Print("⚠ Check completed")
	} else {
		green.Print("✓ Check completed")
	}

	// Build status details
	var details []string

	if summary.FilesCreated > 0 {
		details = append(details, fmt.Sprintf("%d file(s) created", summary.FilesCreated))
	}
	if summary.FilesMissing > 0 {
		details = append(details, fmt.Sprintf("%d file(s) missing", summary.FilesMissing))
	}
	if summary.ValidationErrors > 0 {
		details = append(details, fmt.Sprintf("%d validation error(s)", summary.ValidationErrors))
	}

	if len(details) > 0 {
		fmt.Printf(" (%s)\n", strings.Join(details, ", "))
	} else {
		fmt.Println(" - All checks passed")
	}
}

// PrintDetailedReport prints a detailed report with all sections
func (r *Report) PrintDetailedReport() {
	// Print header box
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("12")).
		Padding(0, 2).
		Width(50).
		Align(lipgloss.Center)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15"))

	fmt.Println(boxStyle.Render(titleStyle.Render("VerustCode Environment Check Report")))
	fmt.Println()

	// File section
	r.printFileSection()
	fmt.Println()

	// Validation section
	r.printValidationSection()
	fmt.Println()

	// Summary
	r.Print()
}

// printFileSection prints the file check section
func (r *Report) printFileSection() {
	sectionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("14"))

	fmt.Println(sectionStyle.Render("📁 File Check"))

	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)
	red := color.New(color.FgRed)

	for _, result := range r.FileResults {
		if result.Error != nil {
			red.Printf("  ✗ %s: %v\n", result.Path, result.Error)
		} else if result.Exists {
			if result.Created {
				green.Printf("  ✓ %s (created)\n", result.Path)
			} else {
				green.Printf("  ✓ %s\n", result.Path)
			}
		} else {
			yellow.Printf("  ⚠ %s does not exist\n", result.Path)
		}
	}
}

// printValidationSection prints the validation section
func (r *Report) printValidationSection() {
	sectionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("14"))

	fmt.Println(sectionStyle.Render("📝 Configuration Validation"))

	green := color.New(color.FgGreen)
	red := color.New(color.FgRed)
	yellow := color.New(color.FgYellow)

	for _, result := range r.ValidationResults {
		if result.Valid {
			if result.RuleCount > 0 {
				green.Printf("  ✓ %s (%d rules)\n", result.Path, result.RuleCount)
			} else {
				green.Printf("  ✓ %s\n", result.Path)
			}
		} else if result.Error != nil {
			red.Printf("  ✗ %s: %v\n", result.Path, result.Error)
		}

		for _, warning := range result.Warnings {
			yellow.Printf("    └─ %s\n", warning)
		}
	}
}
