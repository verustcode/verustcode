// Package exporter provides report export functionality with pluggable exporters.
package exporter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"go.uber.org/zap"

	"github.com/verustcode/verustcode/internal/model"
	"github.com/verustcode/verustcode/pkg/logger"
)

// ExportFormat represents the export format type
type ExportFormat string

const (
	// ExportFormatMarkdown represents Markdown format
	ExportFormatMarkdown ExportFormat = "markdown"
	// ExportFormatJSON represents JSON format
	ExportFormatJSON ExportFormat = "json"
	// ExportFormatHTML represents HTML format
	ExportFormatHTML ExportFormat = "html"
	// ExportFormatPDF represents PDF format
	ExportFormatPDF ExportFormat = "pdf"
)

// ReportExporter defines the interface for report exporters
type ReportExporter interface {
	// Export exports a report to string content
	Export(report *model.Report, sections []model.ReportSection) (string, error)
	// Name returns the human-readable name of the exporter (e.g., "Markdown", "HTML")
	Name() string
	// FileExtension returns the file extension for this format (e.g., ".md", ".html")
	FileExtension() string
}

// ExportManager manages all registered exporters
type ExportManager struct {
	exporters map[ExportFormat]ReportExporter
	mu        sync.RWMutex
}

// NewExportManager creates a new export manager
func NewExportManager() *ExportManager {
	return &ExportManager{
		exporters: make(map[ExportFormat]ReportExporter),
	}
}

// Register registers an exporter for a specific format
func (m *ExportManager) Register(format ExportFormat, exporter ReportExporter) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.exporters[format] = exporter
	logger.Debug("Registered report exporter",
		zap.String("format", string(format)),
		zap.String("name", exporter.Name()),
	)
}

// Export exports a report using the specified format
func (m *ExportManager) Export(report *model.Report, sections []model.ReportSection, format ExportFormat) (string, error) {
	m.mu.RLock()
	exporter, ok := m.exporters[format]
	m.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("unsupported export format: %s", format)
	}

	logger.Debug("Exporting report",
		zap.String("report_id", report.ID),
		zap.String("format", string(format)),
		zap.String("exporter", exporter.Name()),
	)

	content, err := exporter.Export(report, sections)
	if err != nil {
		return "", fmt.Errorf("failed to export report with %s exporter: %w", exporter.Name(), err)
	}

	return content, nil
}

// ExportToFile exports a report to a file
func (m *ExportManager) ExportToFile(report *model.Report, sections []model.ReportSection, outputPath string, format ExportFormat) error {
	content, err := m.Export(report, sections, format)
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	logger.Info("Report exported to file",
		zap.String("report_id", report.ID),
		zap.String("format", string(format)),
		zap.String("path", outputPath),
	)

	return nil
}

// GenerateFilename generates a filename for the exported report
func (m *ExportManager) GenerateFilename(report *model.Report, format ExportFormat) string {
	m.mu.RLock()
	exporter, ok := m.exporters[format]
	m.mu.RUnlock()

	// Base name from title or report type
	baseName := report.Title
	if baseName == "" {
		baseName = fmt.Sprintf("%s-report", report.ReportType)
	}

	// Sanitize filename
	baseName = sanitizeFilename(baseName)

	// Get extension from exporter if available
	if ok {
		return baseName + exporter.FileExtension()
	}

	// Fallback to format-based extension
	switch format {
	case ExportFormatMarkdown:
		return baseName + ".md"
	case ExportFormatJSON:
		return baseName + ".json"
	case ExportFormatHTML:
		return baseName + ".html"
	case ExportFormatPDF:
		return baseName + ".pdf"
	default:
		return baseName + ".txt"
	}
}

// SupportedFormats returns a list of all supported export formats
func (m *ExportManager) SupportedFormats() []ExportFormat {
	m.mu.RLock()
	defer m.mu.RUnlock()

	formats := make([]ExportFormat, 0, len(m.exporters))
	for format := range m.exporters {
		formats = append(formats, format)
	}
	return formats
}

// GetExporter returns the exporter for a specific format
func (m *ExportManager) GetExporter(format ExportFormat) (ReportExporter, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	exporter, ok := m.exporters[format]
	if !ok {
		return nil, fmt.Errorf("no exporter registered for format: %s", format)
	}
	return exporter, nil
}

// sanitizeFilename removes unsafe characters from filename
func sanitizeFilename(name string) string {
	// Replace unsafe characters
	unsafe := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", " "}
	result := name
	for _, char := range unsafe {
		result = strings.ReplaceAll(result, char, "_")
	}

	// Remove consecutive underscores
	for strings.Contains(result, "__") {
		result = strings.ReplaceAll(result, "__", "_")
	}

	// Trim underscores
	result = strings.Trim(result, "_")

	// Limit length
	if len(result) > 100 {
		result = result[:100]
	}

	return result
}
