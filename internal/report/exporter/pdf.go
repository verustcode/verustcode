// Package exporter provides report export functionality with pluggable exporters.
package exporter

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"go.uber.org/zap"

	"github.com/verustcode/verustcode/internal/model"
	"github.com/verustcode/verustcode/internal/report/assets"
	"github.com/verustcode/verustcode/pkg/logger"
)

// PDFOptions contains configuration for PDF generation
type PDFOptions struct {
	// Paper dimensions in inches (A4: 8.27 x 11.69)
	PaperWidth  float64
	PaperHeight float64

	// Margins in inches
	MarginTop    float64
	MarginBottom float64
	MarginLeft   float64
	MarginRight  float64

	// Header and footer
	DisplayHeaderFooter bool
	HeaderTemplate      string
	FooterTemplate      string

	// Print background colors and images
	PrintBackground bool

	// Scale of the webpage rendering (1.0 = 100%)
	Scale float64

	// Timeout for PDF generation
	Timeout time.Duration
}

// DefaultPDFOptions returns default PDF options for A4 paper
func DefaultPDFOptions() PDFOptions {
	return PDFOptions{
		// A4 paper size in inches
		PaperWidth:  8.27,
		PaperHeight: 11.69,

		// Margins in inches
		// MarginTop增加20%以增加页眉与正文的距离: 0.59 -> 0.71 (~18mm)
		MarginTop:    0.71, // ~18mm (increased 20% for better header spacing)
		MarginBottom: 0.59, // ~15mm
		MarginLeft:   0.79, // ~20mm
		MarginRight:  0.79, // ~20mm

		DisplayHeaderFooter: true,
		PrintBackground:     true,
		Scale:               1.0,
		Timeout:             120 * time.Second,
	}
}

// PDFExporter exports reports to PDF format using Chrome headless
type PDFExporter struct {
	options PDFOptions
}

// NewPDFExporter creates a new PDF exporter with default options
func NewPDFExporter() *PDFExporter {
	return &PDFExporter{
		options: DefaultPDFOptions(),
	}
}

// NewPDFExporterWithOptions creates a new PDF exporter with custom options
func NewPDFExporterWithOptions(opts PDFOptions) *PDFExporter {
	return &PDFExporter{
		options: opts,
	}
}

// Export exports a report to PDF format
// Note: This returns a base64 encoded PDF as string for interface compatibility
// For binary PDF data, use ExportToPDF instead
func (e *PDFExporter) Export(report *model.Report, sections []model.ReportSection) (string, error) {
	pdfData, err := e.ExportToPDF(report, sections)
	if err != nil {
		return "", err
	}
	// Return as string (binary data) - caller should handle appropriately
	return string(pdfData), nil
}

// ExportToPDF exports a report to PDF format and returns binary data
func (e *PDFExporter) ExportToPDF(report *model.Report, sections []model.ReportSection) ([]byte, error) {
	// 记录开始时间用于计算总耗时
	startTime := time.Now()

	logger.Info("[PDF Export] Starting PDF export",
		zap.String("report_id", report.ID),
		zap.String("title", report.Title),
		zap.Int("sections_count", len(sections)),
		zap.Duration("timeout", e.options.Timeout),
	)

	// Generate print-optimized HTML
	logger.Debug("[PDF Export] Generating print-optimized HTML",
		zap.String("report_id", report.ID),
	)
	htmlStartTime := time.Now()
	html := e.generatePrintHTML(report, sections)
	logger.Debug("[PDF Export] HTML generation completed",
		zap.String("report_id", report.ID),
		zap.Int("html_size", len(html)),
		zap.Duration("duration", time.Since(htmlStartTime)),
	)

	// Generate header/footer templates
	headerTemplate, footerTemplate := e.generateHeaderFooter(report)
	logger.Debug("[PDF Export] Header/footer templates generated",
		zap.String("report_id", report.ID),
		zap.Int("header_size", len(headerTemplate)),
		zap.Int("footer_size", len(footerTemplate)),
	)

	// Write HTML to temporary file (avoids data URL size limits)
	logger.Debug("[PDF Export] Creating temporary HTML file",
		zap.String("report_id", report.ID),
	)
	tmpFile, err := os.CreateTemp("", "verustcode-pdf-*.html")
	if err != nil {
		logger.Error("[PDF Export] Failed to create temp file",
			zap.String("report_id", report.ID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath) // Clean up temp file

	if _, err := tmpFile.WriteString(html); err != nil {
		tmpFile.Close()
		logger.Error("[PDF Export] Failed to write HTML to temp file",
			zap.String("report_id", report.ID),
			zap.String("temp_path", tmpPath),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to write temp file: %w", err)
	}
	tmpFile.Close()
	logger.Debug("[PDF Export] Temporary HTML file created",
		zap.String("report_id", report.ID),
		zap.String("temp_path", tmpPath),
		zap.Int("file_size", len(html)),
	)

	// Create Chrome context with timeout
	logger.Debug("[PDF Export] Creating Chrome context",
		zap.String("report_id", report.ID),
		zap.Duration("timeout", e.options.Timeout),
	)
	ctx, cancel := context.WithTimeout(context.Background(), e.options.Timeout)
	defer cancel()

	// Configure Chrome options
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-software-rasterizer", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("headless", true),
	)

	// Check for custom Chrome path
	chromePath := os.Getenv("CHROME_PATH")
	if chromePath != "" {
		opts = append(opts, chromedp.ExecPath(chromePath))
		logger.Debug("[PDF Export] Using custom Chrome path",
			zap.String("report_id", report.ID),
			zap.String("chrome_path", chromePath),
		)
	} else {
		logger.Debug("[PDF Export] Using default Chrome path",
			zap.String("report_id", report.ID),
		)
	}

	logger.Debug("[PDF Export] Initializing Chrome allocator",
		zap.String("report_id", report.ID),
	)

	// Increase WebSocket URL timeout (default is 20s which may not be enough for slow systems)
	opts = append(opts, chromedp.WSURLReadTimeout(60*time.Second))

	allocCtx, allocCancel := chromedp.NewExecAllocator(ctx, opts...)
	defer allocCancel()

	// Create browser context
	logger.Debug("[PDF Export] Creating browser context",
		zap.String("report_id", report.ID),
	)
	browserCtx, browserCancel := chromedp.NewContext(allocCtx,
		chromedp.WithLogf(func(format string, args ...interface{}) {
			logger.Debug(fmt.Sprintf("[PDF Export] chromedp: "+format, args...))
		}),
	)
	defer browserCancel()

	var pdfData []byte

	// Navigate to HTML file and print to PDF
	fileURL := "file://" + tmpPath
	logger.Debug("[PDF Export] Starting Chrome navigation",
		zap.String("report_id", report.ID),
		zap.String("file_url", fileURL),
	)

	chromeStartTime := time.Now()
	err = chromedp.Run(browserCtx,
		// Navigate to the HTML file
		chromedp.ActionFunc(func(ctx context.Context) error {
			logger.Debug("[PDF Export] Navigating to HTML file",
				zap.String("report_id", report.ID),
			)
			return nil
		}),
		chromedp.Navigate(fileURL),

		// Wait for content to be ready
		chromedp.ActionFunc(func(ctx context.Context) error {
			logger.Debug("[PDF Export] Waiting for body element",
				zap.String("report_id", report.ID),
			)
			return nil
		}),
		chromedp.WaitReady("body"),

		// Wait for JavaScript to execute (Mermaid rendering, syntax highlighting)
		// Use a longer wait to ensure all async operations complete
		chromedp.ActionFunc(func(ctx context.Context) error {
			logger.Debug("[PDF Export] Waiting for JavaScript execution (3s)",
				zap.String("report_id", report.ID),
			)
			return nil
		}),
		chromedp.Sleep(3*time.Second),

		// Generate PDF
		chromedp.ActionFunc(func(ctx context.Context) error {
			logger.Debug("[PDF Export] Starting PDF generation with Chrome",
				zap.String("report_id", report.ID),
				zap.Float64("paper_width", e.options.PaperWidth),
				zap.Float64("paper_height", e.options.PaperHeight),
				zap.Float64("margin_top", e.options.MarginTop),
				zap.Float64("margin_bottom", e.options.MarginBottom),
				zap.Float64("scale", e.options.Scale),
			)

			pdfStartTime := time.Now()
			var err error
			pdfData, _, err = page.PrintToPDF().
				WithPaperWidth(e.options.PaperWidth).
				WithPaperHeight(e.options.PaperHeight).
				WithMarginTop(e.options.MarginTop).
				WithMarginBottom(e.options.MarginBottom).
				WithMarginLeft(e.options.MarginLeft).
				WithMarginRight(e.options.MarginRight).
				WithDisplayHeaderFooter(e.options.DisplayHeaderFooter).
				WithHeaderTemplate(headerTemplate).
				WithFooterTemplate(footerTemplate).
				WithPrintBackground(e.options.PrintBackground).
				WithScale(e.options.Scale).
				WithPreferCSSPageSize(false).
				Do(ctx)

			if err != nil {
				logger.Error("[PDF Export] Chrome PrintToPDF failed",
					zap.String("report_id", report.ID),
					zap.Error(err),
					zap.Duration("duration", time.Since(pdfStartTime)),
				)
			} else {
				logger.Debug("[PDF Export] Chrome PrintToPDF completed",
					zap.String("report_id", report.ID),
					zap.Int("pdf_size", len(pdfData)),
					zap.Duration("duration", time.Since(pdfStartTime)),
				)
			}
			return err
		}),
	)

	if err != nil {
		logger.Error("[PDF Export] Failed to generate PDF",
			zap.String("report_id", report.ID),
			zap.Error(err),
			zap.Duration("chrome_duration", time.Since(chromeStartTime)),
			zap.Duration("total_duration", time.Since(startTime)),
		)
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	logger.Info("[PDF Export] PDF export completed successfully",
		zap.String("report_id", report.ID),
		zap.Int("pdf_size_bytes", len(pdfData)),
		zap.String("pdf_size_human", formatBytes(len(pdfData))),
		zap.Duration("chrome_duration", time.Since(chromeStartTime)),
		zap.Duration("total_duration", time.Since(startTime)),
	)

	return pdfData, nil
}

// formatBytes converts bytes to human-readable format
func formatBytes(bytes int) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := int64(bytes) / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// Name returns the human-readable name of this exporter
func (e *PDFExporter) Name() string {
	return "PDF"
}

// FileExtension returns the file extension for PDF files
func (e *PDFExporter) FileExtension() string {
	return ".pdf"
}

// generateHeaderFooter creates header and footer HTML templates
func (e *PDFExporter) generateHeaderFooter(report *model.Report) (header, footer string) {
	title := report.Title
	if title == "" {
		title = fmt.Sprintf("%s Report", report.ReportType)
	}

	// Header template with Logo + brand name on left only (no title on right)
	// Chrome uses specific CSS classes for page info: pageNumber, totalPages, title, url, date
	// Note: Using inline SVG for the logo in PDF header - larger size for better visibility
	// Logo尺寸放大20%: 28->34, 产品名称字体放大20%: 14px->17px
	// 增加页眉padding以增加高度和与正文的距离
	header = fmt.Sprintf(`
		<div style="width:100%%; padding:8px 20px 12px 20px; font-size:12px; font-family:system-ui,-apple-system,sans-serif; color:#666; display:flex; justify-content:flex-start; align-items:center;">
			<span style="display:flex; align-items:center; gap:12px;">
				%s
				<span style="font-weight:600; font-size:17px; color:#1E40AF;">Verust Code</span>
			</span>
		</div>
	`, getLogoSVG(34, 34))

	// Footer template with page numbers
	footer = `
		<div style="width:100%; padding:0 20px; font-size:9px; font-family:system-ui,-apple-system,sans-serif; color:#666; display:flex; justify-content:space-between; align-items:center;">
			<span>Generated by VerustCode</span>
			<span>Page <span class="pageNumber"></span> of <span class="totalPages"></span></span>
		</div>
	`

	return header, footer
}

// generatePrintHTML creates a print-optimized HTML document
func (e *PDFExporter) generatePrintHTML(report *model.Report, sections []model.ReportSection) string {
	title := report.Title
	if title == "" {
		title = fmt.Sprintf("%s Report", report.ReportType)
	}

	// Build section content for linear display (no sidebar, all sections expanded)
	var contentBuilder strings.Builder

	// Sort sections by index
	sortedSections := make([]model.ReportSection, len(sections))
	copy(sortedSections, sections)
	for i := 0; i < len(sortedSections)-1; i++ {
		for j := i + 1; j < len(sortedSections); j++ {
			if sortedSections[i].SectionIndex > sortedSections[j].SectionIndex {
				sortedSections[i], sortedSections[j] = sortedSections[j], sortedSections[i]
			}
		}
	}

	// Build parent-child map
	parentSections := []model.ReportSection{}
	childMap := make(map[string][]model.ReportSection)

	for _, s := range sortedSections {
		if s.ParentSectionID == nil || *s.ParentSectionID == "" {
			parentSections = append(parentSections, s)
		} else {
			childMap[*s.ParentSectionID] = append(childMap[*s.ParentSectionID], s)
		}
	}

	// Generate sections JSON for JavaScript rendering
	var sectionsJSON strings.Builder
	sectionsJSON.WriteString("[")
	for i, p := range parentSections {
		if i > 0 {
			sectionsJSON.WriteString(",")
		}
		children := childMap[p.SectionID]
		sectionsJSON.WriteString(fmt.Sprintf(`{"id":"%d","sectionId":"%s","title":%s,"content":%s,"isLeaf":%t,"children":[`,
			p.ID, p.SectionID, escapeJSONString(p.Title), escapeJSONString(p.Content), p.IsLeaf || len(children) == 0))
		for j, c := range children {
			if j > 0 {
				sectionsJSON.WriteString(",")
			}
			sectionsJSON.WriteString(fmt.Sprintf(`{"id":"%d","sectionId":"%s","title":%s,"content":%s,"isLeaf":true}`,
				c.ID, c.SectionID, escapeJSONString(c.Title), escapeJSONString(c.Content)))
		}
		sectionsJSON.WriteString("]}")
	}
	sectionsJSON.WriteString("]")

	// Build the HTML content
	contentBuilder.WriteString(e.generatePrintHTMLTemplate(title, report, sectionsJSON.String()))

	return contentBuilder.String()
}

// generatePrintHTMLTemplate creates the full HTML document for PDF printing
func (e *PDFExporter) generatePrintHTMLTemplate(title string, report *model.Report, sectionsJSON string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s</title>
    <!-- Embedded Marked.js for Markdown rendering -->
    <script>%s</script>
    <!-- Embedded Prism.js for code syntax highlighting -->
    <style>%s</style>
    <script>%s</script>
    <!-- Embedded Mermaid.js for diagram rendering -->
    <script>%s</script>
    <style>
        /* Base styles for PDF print */
        * {
            box-sizing: border-box;
            margin: 0;
            padding: 0;
        }
        
        body {
            font-family: system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Noto Sans SC', 'Microsoft YaHei', sans-serif;
            font-size: 10pt;
            line-height: 1.6;
            color: #1a1a1a;
            background: white;
            padding: 0;
            text-align: left;
        }

        /* Main container */
        .container {
            max-width: 100%%;
            padding: 0;
        }

        /* Report header */
        .report-header {
            text-align: center;
            padding-bottom: 20px;
            margin-bottom: 20px;
            border-bottom: 2px solid #2563eb;
        }

        .report-logo {
            margin-bottom: 12px;
        }

        .report-header h1 {
            font-size: 18pt;
            font-weight: 700;
            color: #1e40af;
            margin-bottom: 8px;
        }

        .report-meta {
            font-size: 9pt;
            color: #666;
        }

        .report-meta span {
            margin: 0 10px;
        }

        /* Table of Contents */
        .toc {
            background: #f8fafc;
            border: 1px solid #e2e8f0;
            border-radius: 6px;
            padding: 15px 20px;
            margin-bottom: 25px;
            page-break-after: always;
        }

        .toc h2 {
            font-size: 12pt;
            font-weight: 600;
            color: #1e40af;
            margin-bottom: 12px;
            padding-bottom: 8px;
            border-bottom: 1px solid #e2e8f0;
        }

        .toc-list {
            list-style: none;
            padding: 0;
        }

        .toc-list li {
            padding: 4px 0;
            font-size: 10pt;
        }

        .toc-list .toc-parent {
            font-weight: 600;
            color: #334155;
        }

        .toc-list .toc-child {
            padding-left: 20px;
            color: #64748b;
        }

        /* Section content */
        .section {
            margin-bottom: 20px;
            page-break-inside: auto;
        }

        .section-title {
            font-size: 14pt;
            font-weight: 700;
            color: #1e40af;
            padding: 10px 0;
            margin-bottom: 10px;
            border-bottom: 2px solid #2563eb;
            page-break-after: avoid;
        }

        .subsection-title {
            font-size: 12pt;
            font-weight: 600;
            color: #334155;
            padding: 8px 0;
            margin: 15px 0 10px 0;
            border-left: 4px solid #2563eb;
            padding-left: 12px;
            page-break-after: avoid;
        }

        /* Markdown content styles */
        .markdown-content h1 {
            font-size: 14pt;
            font-weight: 700;
            color: #1e40af;
            margin: 20px 0 10px 0;
            padding-bottom: 6px;
            border-bottom: 2px solid #2563eb;
            page-break-after: avoid;
        }

        .markdown-content h2 {
            font-size: 12pt;
            font-weight: 700;
            color: #1e3a8a;
            margin: 18px 0 8px 0;
            padding-bottom: 4px;
            border-bottom: 1px solid #e2e8f0;
            page-break-after: avoid;
        }

        .markdown-content h3 {
            font-size: 11pt;
            font-weight: 600;
            color: #334155;
            margin: 15px 0 6px 0;
            padding-left: 10px;
            border-left: 3px solid #2563eb;
            page-break-after: avoid;
        }

        .markdown-content h4 {
            font-size: 10pt;
            font-weight: 600;
            color: #475569;
            margin: 12px 0 5px 0;
            page-break-after: avoid;
        }

        .markdown-content p {
            margin: 6px 0;
            text-align: left;
            word-wrap: break-word;
        }

        .markdown-content ul, .markdown-content ol {
            margin: 6px 0;
            padding-left: 20px;
            text-align: left;
        }

        .markdown-content li {
            margin: 3px 0;
            text-align: left;
        }

        .markdown-content blockquote {
            border-left: 3px solid #2563eb;
            padding-left: 12px;
            margin: 10px 0;
            color: #64748b;
            font-style: italic;
            background: #f8fafc;
            padding: 8px 12px;
        }

        /* Code blocks */
        .markdown-content code {
            font-family: 'JetBrains Mono', 'Fira Code', 'SF Mono', Consolas, monospace;
            font-size: 8pt;
            background: #f1f5f9;
            padding: 1px 4px;
            border-radius: 3px;
            color: #c7254e;
        }

        .markdown-content pre {
            margin: 10px 0;
            border-radius: 6px;
            overflow-x: auto;
            page-break-inside: avoid;
        }

        .markdown-content pre code {
            display: block;
            padding: 12px;
            background: #1e293b;
            color: #e2e8f0;
            font-size: 8pt;
            line-height: 1.5;
            overflow-x: auto;
        }

        /* Tables */
        .markdown-content table {
            width: 100%%;
            border-collapse: collapse;
            margin: 10px 0;
            font-size: 9pt;
            page-break-inside: avoid;
        }

        .markdown-content th,
        .markdown-content td {
            border: 1px solid #e2e8f0;
            padding: 6px 10px;
            text-align: left;
        }

        .markdown-content th {
            background: #f1f5f9;
            font-weight: 600;
            color: #334155;
        }

        .markdown-content tr:nth-child(even) {
            background: #f8fafc;
        }

        /* Links */
        .markdown-content a {
            color: #2563eb;
            text-decoration: none;
        }

        /* Horizontal rule */
        .markdown-content hr {
            border: none;
            border-top: 1px solid #e2e8f0;
            margin: 15px 0;
        }

        /* Mermaid diagrams */
        .mermaid-container {
            margin: 15px 0;
            padding: 15px;
            background: #f8fafc;
            border: 1px solid #e2e8f0;
            border-radius: 6px;
            text-align: center;
            page-break-inside: auto;
        }

        .mermaid-container svg {
            max-width: 100%%;
            height: auto;
        }

        /* Print-specific styles */
        @media print {
            body {
                font-size: 10pt;
                -webkit-print-color-adjust: exact;
                print-color-adjust: exact;
            }

            .page-break {
                page-break-before: always;
            }

            h1, h2, h3, h4, h5, h6 {
                page-break-after: avoid;
            }

            pre, blockquote, table, figure {
                page-break-inside: avoid;
            }

            p, li {
                orphans: 3;
                widows: 3;
            }
        }

        /* Loading indicator (hidden after render) */
        .loading {
            display: none;
        }
    </style>
</head>
<body>
    <div class="container">
        <!-- Report Header -->
        <div class="report-header">
            <h1>%s</h1>
            <div class="report-meta">
                <span>Repo: %s</span>
                <span>Ref: %s</span>
                <span>Date: %s</span>
            </div>
        </div>

        <!-- Table of Contents -->
        <div class="toc" id="toc"></div>

        <!-- Content sections -->
        <div id="content"></div>
    </div>

    <script>
        // Report sections data
        const sections = %s;
        
        // Initialize Mermaid with light theme for PDF
        mermaid.initialize({
            startOnLoad: false,
            theme: 'default',
            securityLevel: 'loose',
            fontFamily: 'system-ui, -apple-system, sans-serif',
            flowchart: { useMaxWidth: true, htmlLabels: true },
            sequence: { useMaxWidth: true },
            gantt: { useMaxWidth: true },
            class: { useMaxWidth: true },
            state: { useMaxWidth: true },
            er: { useMaxWidth: true },
            pie: { useMaxWidth: true }
        });

        // Generate Table of Contents
        function generateTOC() {
            const tocEl = document.getElementById('toc');
            let html = '<h2>Table of Contents</h2><ul class="toc-list">';
            
            sections.forEach((parent, idx) => {
                const hasChildren = parent.children && parent.children.length > 0;
                html += '<li class="toc-parent">' + (idx + 1) + '. ' + escapeHtml(parent.title) + '</li>';
                
                if (hasChildren) {
                    parent.children.forEach((child, cidx) => {
                        html += '<li class="toc-child">' + (idx + 1) + '.' + (cidx + 1) + ' ' + escapeHtml(child.title) + '</li>';
                    });
                }
            });
            
            html += '</ul>';
            tocEl.innerHTML = html;
        }

        // Render all sections
        async function renderContent() {
            const contentEl = document.getElementById('content');
            let html = '';

            for (const parent of sections) {
                const hasChildren = parent.children && parent.children.length > 0;

                if (!hasChildren && parent.content) {
                    // Leaf section with content
                    html += '<div class="section">';
                    html += '<div class="markdown-content">' + marked.parse(parent.content) + '</div>';
                    html += '</div>';
                } else if (hasChildren) {
                    // Parent with children
                    for (const child of parent.children) {
                        if (child.content) {
                            html += '<div class="section">';
                            html += '<div class="markdown-content">' + marked.parse(child.content) + '</div>';
                            html += '</div>';
                        }
                    }
                }
            }

            contentEl.innerHTML = html;

            // Apply syntax highlighting
            document.querySelectorAll('pre code').forEach(block => {
                if (!block.classList.contains('language-mermaid')) {
                    Prism.highlightElement(block);
                }
            });

            // Render Mermaid diagrams
            await renderMermaidDiagrams();
        }

        // Render Mermaid diagrams
        async function renderMermaidDiagrams() {
            const mermaidBlocks = document.querySelectorAll('pre code.language-mermaid');
            let counter = 0;

            for (const block of mermaidBlocks) {
                const code = block.textContent;
                const pre = block.parentElement;

                try {
                    const id = 'mermaid-pdf-' + (++counter);
                    const { svg } = await mermaid.render(id, code);

                    const container = document.createElement('div');
                    container.className = 'mermaid-container';
                    container.innerHTML = svg;

                    pre.replaceWith(container);
                } catch (err) {
                    console.error('Mermaid render error:', err);
                    pre.style.cssText = 'border:1px solid #f87171;background:#fef2f2;padding:12px;border-radius:6px;';
                    block.innerHTML = '<span style="color:#dc2626;">Diagram render failed</span>\\n' + escapeHtml(code);
                }
            }
        }

        // Escape HTML
        function escapeHtml(text) {
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        }

        // Initialize - use multiple strategies to ensure execution
        async function init() {
            try {
                generateTOC();
                await renderContent();
                console.log('PDF content rendered successfully');
            } catch (err) {
                console.error('PDF render error:', err);
            }
        }

        // Try DOMContentLoaded first
        if (document.readyState === 'loading') {
            document.addEventListener('DOMContentLoaded', init);
        } else {
            // DOM already loaded, run immediately
            init();
        }
    </script>
</body>
</html>`,
		escapeHTMLAttr(title), // <title>
		assets.MarkedJS,       // Marked.js
		assets.PrismCSSLight,  // Prism.js CSS (light for PDF)
		assets.PrismJS,        // Prism.js
		assets.MermaidJS,      // Mermaid.js
		escapeHTMLAttr(title), // header h1
		escapeHTMLAttr(report.RepoURL),
		escapeHTMLAttr(report.Ref),
		report.CreatedAt.Format("2006-01-02 15:04"),
		sectionsJSON)
}
