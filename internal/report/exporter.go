// Package report provides report generation functionality.
package report

import (
	"fmt"

	"github.com/verustcode/verustcode/internal/model"
	"github.com/verustcode/verustcode/internal/report/exporter"
)

// ExportFormat represents the export format (re-exported for backward compatibility)
type ExportFormat = exporter.ExportFormat

const (
	// ExportFormatMarkdown represents Markdown format
	ExportFormatMarkdown = exporter.ExportFormatMarkdown
	// ExportFormatJSON represents JSON format
	ExportFormatJSON = exporter.ExportFormatJSON
	// ExportFormatHTML represents HTML format
	ExportFormatHTML = exporter.ExportFormatHTML
	// ExportFormatPDF represents PDF format
	ExportFormatPDF = exporter.ExportFormatPDF
)

// Exporter handles exporting reports to various formats (backward compatibility wrapper)
type Exporter struct {
	manager *exporter.ExportManager
}

// NewExporter creates a new report exporter with default exporters registered
func NewExporter() *Exporter {
	manager := exporter.NewExportManager()

	// Register default exporters
	manager.Register(ExportFormatMarkdown, exporter.NewMarkdownExporter())
	manager.Register(ExportFormatHTML, exporter.NewHTMLExporter())
	manager.Register(ExportFormatPDF, exporter.NewPDFExporter())

	return &Exporter{
		manager: manager,
	}
}

// ExportToMarkdown exports a report to Markdown format (backward compatibility)
func (e *Exporter) ExportToMarkdown(report *model.Report, sections []model.ReportSection) (string, error) {
	return e.manager.Export(report, sections, ExportFormatMarkdown)
}

// ExportToHTML exports a report to HTML format (backward compatibility)
func (e *Exporter) ExportToHTML(report *model.Report, sections []model.ReportSection) (string, error) {
	return e.manager.Export(report, sections, ExportFormatHTML)
}

// ExportToPDF exports a report to PDF format and returns binary data
func (e *Exporter) ExportToPDF(report *model.Report, sections []model.ReportSection) ([]byte, error) {
	exp, err := e.manager.GetExporter(ExportFormatPDF)
	if err != nil {
		return nil, err
	}
	// Type assert to PDFExporter to use ExportToPDF method
	pdfExp, ok := exp.(*exporter.PDFExporter)
	if !ok {
		return nil, fmt.Errorf("PDF exporter is not of expected type")
	}
	return pdfExp.ExportToPDF(report, sections)
}

// ExportToFile exports a report to a file (backward compatibility)
func (e *Exporter) ExportToFile(report *model.Report, sections []model.ReportSection, outputPath string, format ExportFormat) error {
	return e.manager.ExportToFile(report, sections, outputPath, format)
}

// GenerateFilename generates a filename for the exported report (backward compatibility)
func (e *Exporter) GenerateFilename(report *model.Report, format ExportFormat) string {
	return e.manager.GenerateFilename(report, format)
}

// GetExportMetadata returns metadata about the export (backward compatibility)
func (e *Exporter) GetExportMetadata(report *model.Report) *ExportMetadata {
	return &ExportMetadata{
		ReportID:     report.ID,
		Title:        report.Title,
		ReportType:   report.ReportType,
		Repository:   report.RepoURL,
		Ref:          report.Ref,
		GeneratedAt:  report.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		Generator:    "VerustCode",
		ContentSize:  len(report.Content),
		SectionCount: report.TotalSections,
	}
}

// ExportMetadata represents metadata about an exported report
type ExportMetadata struct {
	ReportID     string `json:"report_id"`
	Title        string `json:"title"`
	ReportType   string `json:"report_type"`
	Repository   string `json:"repository"`
	Ref          string `json:"ref"`
	GeneratedAt  string `json:"generated_at"`
	Generator    string `json:"generator"`
	ContentSize  int    `json:"content_size"`
	SectionCount int    `json:"section_count"`
}
