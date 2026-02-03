// Package exporter provides report export functionality with pluggable exporters.
package exporter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/verustcode/verustcode/internal/model"
)

// ====================
// Tests for helpers.go - Utility Functions
// ====================

func TestEscapeJSONString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple string",
			input:    "hello",
			expected: `"hello"`,
		},
		{
			name:     "with quotes",
			input:    `say "hello"`,
			expected: `"say \"hello\""`,
		},
		{
			name:     "with backslash",
			input:    `path\to\file`,
			expected: `"path\\to\\file"`,
		},
		{
			name:     "with newline",
			input:    "line1\nline2",
			expected: `"line1\nline2"`,
		},
		{
			name:     "with script tag",
			input:    `<script>alert("xss")</script>`,
			expected: `"<script\>alert(\"xss\")<\/script>"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeJSONString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEscapeHTMLAttr(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple string",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "with ampersand",
			input:    "A & B",
			expected: "A &amp; B",
		},
		{
			name:     "with angle brackets",
			input:    "<tag>",
			expected: "&lt;tag&gt;",
		},
		{
			name:     "with quotes",
			input:    `say "hello"`,
			expected: "say &quot;hello&quot;",
		},
		{
			name:     "with single quote",
			input:    "it's",
			expected: "it&#39;s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeHTMLAttr(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEscapeYAMLString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple string",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "with backslash",
			input:    `path\to\file`,
			expected: `path\\to\\file`,
		},
		{
			name:     "with quotes",
			input:    `say "hello"`,
			expected: `say \"hello\"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeYAMLString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple filename",
			input:    "report",
			expected: "report",
		},
		{
			name:     "with spaces",
			input:    "my report",
			expected: "my_report",
		},
		{
			name:     "with unsafe characters",
			input:    "report:test/file*name?",
			expected: "report_test_file_name",
		},
		{
			name:     "with consecutive underscores",
			input:    "report__test",
			expected: "report_test",
		},
		{
			name:     "with leading/trailing underscores",
			input:    "_report_",
			expected: "report",
		},
		{
			name:     "too long filename",
			input:    strings.Repeat("a", 150),
			expected: strings.Repeat("a", 100),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeFilename(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCreateAnchor(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple title",
			input:    "Introduction",
			expected: "introduction",
		},
		{
			name:     "with spaces",
			input:    "Chapter One",
			expected: "chapter-one",
		},
		{
			name:     "with colon",
			input:    "Section 1: Overview",
			expected: "section-1-overview",
		},
		{
			name:     "with parentheses",
			input:    "Section (Part A)",
			expected: "section-part-a",
		},
		{
			name:     "with slash",
			input:    "Section 1/2",
			expected: "section-1-2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := createAnchor(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMergeSections(t *testing.T) {
	now := time.Now()
	report := &model.Report{
		ID:         "test-report-001",
		Title:      "Test Report",
		ReportType: "test",
		RepoURL:    "https://github.com/test/repo",
		Ref:        "main",
		CreatedAt:  now,
	}

	t.Run("empty sections", func(t *testing.T) {
		result := MergeSections(report, []model.ReportSection{})
		assert.Contains(t, result, "# Test Report")
		assert.Contains(t, result, "Table of Contents")
	})

	t.Run("single leaf section", func(t *testing.T) {
		sections := []model.ReportSection{
			{
				ID:           1,
				SectionID:    "intro",
				Title:        "Introduction",
				Content:      "This is the introduction.",
				IsLeaf:       true,
				SectionIndex: 0,
			},
		}

		result := MergeSections(report, sections)
		assert.Contains(t, result, "# Test Report")
		assert.Contains(t, result, "Introduction")
		assert.Contains(t, result, "This is the introduction.")
		assert.Contains(t, result, "1. [Introduction](#introduction)")
	})

	t.Run("parent with children", func(t *testing.T) {
		parentID := "parent"
		sections := []model.ReportSection{
			{
				ID:           1,
				SectionID:    parentID,
				Title:        "Architecture",
				IsLeaf:       false,
				SectionIndex: 0,
			},
			{
				ID:              2,
				SectionID:       "child1",
				ParentSectionID: &parentID,
				Title:           "Overview",
				Content:         "Architecture overview.",
				IsLeaf:          true,
				SectionIndex:    1,
			},
			{
				ID:              3,
				SectionID:       "child2",
				ParentSectionID: &parentID,
				Title:           "Details",
				Content:         "Architecture details.",
				IsLeaf:          true,
				SectionIndex:    2,
			},
		}

		result := MergeSections(report, sections)
		assert.Contains(t, result, "Architecture")
		assert.Contains(t, result, "Architecture overview.")
		assert.Contains(t, result, "Architecture details.")
		assert.Contains(t, result, "1. Architecture")
		assert.Contains(t, result, "   1.1. [Overview](#overview)")
		assert.Contains(t, result, "   1.2. [Details](#details)")
	})

	t.Run("section with failed status", func(t *testing.T) {
		sections := []model.ReportSection{
			{
				ID:           1,
				SectionID:    "failed",
				Title:        "Failed Section",
				Status:       model.SectionStatusFailed,
				ErrorMessage: "Generation failed",
				IsLeaf:       true,
				SectionIndex: 0,
			},
		}

		result := MergeSections(report, sections)
		assert.Contains(t, result, "Failed Section")
		assert.Contains(t, result, "Section generation failed")
		assert.Contains(t, result, "Error: Generation failed")
	})
}

// ====================
// Tests for interface.go - ExportManager
// ====================

func TestNewExportManager(t *testing.T) {
	manager := NewExportManager()
	assert.NotNil(t, manager)
	assert.NotNil(t, manager.exporters)
	assert.Empty(t, manager.exporters)
}

func TestExportManager_Register(t *testing.T) {
	manager := NewExportManager()
	exporter := NewMarkdownExporter()

	manager.Register(ExportFormatMarkdown, exporter)

	// Verify exporter was registered
	registered, err := manager.GetExporter(ExportFormatMarkdown)
	assert.NoError(t, err)
	assert.Equal(t, exporter, registered)
}

func TestExportManager_Export(t *testing.T) {
	manager := NewExportManager()
	exporter := NewMarkdownExporter()
	manager.Register(ExportFormatMarkdown, exporter)

	report := &model.Report{
		ID:         "test-001",
		Title:      "Test Report",
		ReportType: "test",
		CreatedAt:  time.Now(),
	}
	sections := []model.ReportSection{
		{
			ID:           1,
			SectionID:    "intro",
			Title:        "Introduction",
			Content:      "Test content",
			IsLeaf:       true,
			SectionIndex: 0,
		},
	}

	content, err := manager.Export(report, sections, ExportFormatMarkdown)
	assert.NoError(t, err)
	assert.NotEmpty(t, content)
	assert.Contains(t, content, "Test Report")
}

func TestExportManager_Export_UnsupportedFormat(t *testing.T) {
	manager := NewExportManager()

	report := &model.Report{ID: "test-001"}
	sections := []model.ReportSection{}

	_, err := manager.Export(report, sections, ExportFormatPDF)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported export format")
}

func TestExportManager_ExportToFile(t *testing.T) {
	manager := NewExportManager()
	exporter := NewMarkdownExporter()
	manager.Register(ExportFormatMarkdown, exporter)

	report := &model.Report{
		ID:         "test-001",
		Title:      "Test Report",
		ReportType: "test",
		CreatedAt:  time.Now(),
	}
	sections := []model.ReportSection{
		{
			ID:           1,
			SectionID:    "intro",
			Title:        "Introduction",
			Content:      "Test content",
			IsLeaf:       true,
			SectionIndex: 0,
		},
	}

	// Create temp directory
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "test-report.md")

	err := manager.ExportToFile(report, sections, outputPath, ExportFormatMarkdown)
	assert.NoError(t, err)

	// Verify file was created
	_, err = os.Stat(outputPath)
	assert.NoError(t, err)

	// Verify file content
	content, err := os.ReadFile(outputPath)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "Test Report")
}

func TestExportManager_GenerateFilename(t *testing.T) {
	manager := NewExportManager()
	exporter := NewMarkdownExporter()
	manager.Register(ExportFormatMarkdown, exporter)

	t.Run("with title", func(t *testing.T) {
		report := &model.Report{
			Title:      "My Test Report",
			ReportType: "test",
		}
		filename := manager.GenerateFilename(report, ExportFormatMarkdown)
		assert.Equal(t, "My_Test_Report.md", filename)
	})

	t.Run("without title", func(t *testing.T) {
		report := &model.Report{
			ReportType: "test",
		}
		filename := manager.GenerateFilename(report, ExportFormatMarkdown)
		assert.Equal(t, "test-report.md", filename)
	})

	t.Run("unsafe characters in title", func(t *testing.T) {
		report := &model.Report{
			Title:      "Report: Test/File*Name?",
			ReportType: "test",
		}
		filename := manager.GenerateFilename(report, ExportFormatMarkdown)
		assert.Equal(t, "Report_Test_File_Name.md", filename)
	})

	t.Run("fallback format", func(t *testing.T) {
		manager := NewExportManager() // No exporters registered
		report := &model.Report{Title: "Test"}
		filename := manager.GenerateFilename(report, ExportFormatPDF)
		assert.Equal(t, "Test.pdf", filename)
	})
}

func TestExportManager_SupportedFormats(t *testing.T) {
	manager := NewExportManager()
	assert.Empty(t, manager.SupportedFormats())

	manager.Register(ExportFormatMarkdown, NewMarkdownExporter())
	manager.Register(ExportFormatHTML, NewHTMLExporter())

	formats := manager.SupportedFormats()
	assert.Len(t, formats, 2)
	assert.Contains(t, formats, ExportFormatMarkdown)
	assert.Contains(t, formats, ExportFormatHTML)
}

func TestExportManager_GetExporter(t *testing.T) {
	manager := NewExportManager()
	exporter := NewMarkdownExporter()
	manager.Register(ExportFormatMarkdown, exporter)

	got, err := manager.GetExporter(ExportFormatMarkdown)
	assert.NoError(t, err)
	assert.Equal(t, exporter, got)

	_, err = manager.GetExporter(ExportFormatPDF)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no exporter registered")
}

// ====================
// Tests for markdown.go - MarkdownExporter
// ====================

func TestNewMarkdownExporter(t *testing.T) {
	exporter := NewMarkdownExporter()
	assert.NotNil(t, exporter)
}

func TestMarkdownExporter_Name(t *testing.T) {
	exporter := NewMarkdownExporter()
	assert.Equal(t, "Markdown", exporter.Name())
}

func TestMarkdownExporter_FileExtension(t *testing.T) {
	exporter := NewMarkdownExporter()
	assert.Equal(t, ".md", exporter.FileExtension())
}

func TestMarkdownExporter_Export(t *testing.T) {
	exporter := NewMarkdownExporter()
	now := time.Now()

	report := &model.Report{
		ID:         "test-001",
		Title:      "Test Report",
		ReportType: "test",
		RepoURL:    "https://github.com/test/repo",
		Ref:        "main",
		CreatedAt:  now,
		Duration:   1000,
	}

	sections := []model.ReportSection{
		{
			ID:           1,
			SectionID:    "intro",
			Title:        "Introduction",
			Content:      "# Introduction\n\nThis is the intro.",
			IsLeaf:       true,
			SectionIndex: 0,
		},
	}

	content, err := exporter.Export(report, sections)
	assert.NoError(t, err)
	assert.NotEmpty(t, content)

	// Verify YAML front matter
	assert.Contains(t, content, "---")
	assert.Contains(t, content, "title:")
	assert.Contains(t, content, "report_type:")
	assert.Contains(t, content, "repository:")
	assert.Contains(t, content, "ref:")
	assert.Contains(t, content, "generated_at:")
	assert.Contains(t, content, "generation_time_ms:")
	assert.Contains(t, content, "generator: VerustCode")

	// Verify content
	assert.Contains(t, content, "Introduction")
	assert.Contains(t, content, "This is the intro.")

	// Verify footer
	assert.Contains(t, content, "Generated by [VerustCode]")
}

func TestMarkdownExporter_Export_WithExistingContent(t *testing.T) {
	exporter := NewMarkdownExporter()

	report := &model.Report{
		ID:         "test-001",
		Title:      "Test Report",
		ReportType: "test",
		Content:    "# Pre-generated Content\n\nThis content already exists.",
		CreatedAt:  time.Now(),
	}

	sections := []model.ReportSection{}

	content, err := exporter.Export(report, sections)
	assert.NoError(t, err)
	assert.Contains(t, content, "Pre-generated Content")
	assert.Contains(t, content, "This content already exists.")
}

// ====================
// Tests for html.go - HTMLExporter
// ====================

func TestNewHTMLExporter(t *testing.T) {
	exporter := NewHTMLExporter()
	assert.NotNil(t, exporter)
}

func TestHTMLExporter_Name(t *testing.T) {
	exporter := NewHTMLExporter()
	assert.Equal(t, "HTML", exporter.Name())
}

func TestHTMLExporter_FileExtension(t *testing.T) {
	exporter := NewHTMLExporter()
	assert.Equal(t, ".html", exporter.FileExtension())
}

func TestHTMLExporter_Export(t *testing.T) {
	exporter := NewHTMLExporter()

	report := &model.Report{
		ID:         "test-001",
		Title:      "Test Report",
		ReportType: "test",
		RepoURL:    "https://github.com/test/repo",
		Ref:        "main",
		CreatedAt:  time.Now(),
	}

	sections := []model.ReportSection{
		{
			ID:           1,
			SectionID:    "intro",
			Title:        "Introduction",
			Content:      "# Introduction\n\nTest content.",
			IsLeaf:       true,
			SectionIndex: 0,
		},
	}

	content, err := exporter.Export(report, sections)
	assert.NoError(t, err)
	assert.NotEmpty(t, content)

	// Verify HTML structure
	assert.Contains(t, content, "<!DOCTYPE html>")
	assert.Contains(t, content, "<html")
	assert.Contains(t, content, "Test Report")
	assert.Contains(t, content, "Introduction")
}

func TestHTMLExporter_Export_WithHierarchy(t *testing.T) {
	exporter := NewHTMLExporter()

	report := &model.Report{
		ID:         "test-001",
		Title:      "Test Report",
		ReportType: "test",
		CreatedAt:  time.Now(),
	}

	parentID := "parent"
	sections := []model.ReportSection{
		{
			ID:           1,
			SectionID:    parentID,
			Title:        "Parent Section",
			IsLeaf:       false,
			SectionIndex: 0,
		},
		{
			ID:              2,
			SectionID:       "child1",
			ParentSectionID: &parentID,
			Title:           "Child 1",
			Content:         "Child content 1",
			IsLeaf:          true,
			SectionIndex:    1,
		},
	}

	content, err := exporter.Export(report, sections)
	assert.NoError(t, err)
	assert.Contains(t, content, "Parent Section")
	assert.Contains(t, content, "Child 1")
	assert.Contains(t, content, "Child content 1")
}

// ====================
// Tests for pdf.go - PDFExporter
// ====================

func TestNewPDFExporter(t *testing.T) {
	exporter := NewPDFExporter()
	assert.NotNil(t, exporter)
	assert.NotNil(t, exporter.options)
}

func TestDefaultPDFOptions(t *testing.T) {
	opts := DefaultPDFOptions()
	assert.Equal(t, 8.27, opts.PaperWidth)
	assert.Equal(t, 11.69, opts.PaperHeight)
	assert.Equal(t, 0.71, opts.MarginTop)
	assert.Equal(t, 0.59, opts.MarginBottom)
	assert.Equal(t, 0.79, opts.MarginLeft)
	assert.Equal(t, 0.79, opts.MarginRight)
	assert.True(t, opts.DisplayHeaderFooter)
	assert.True(t, opts.PrintBackground)
	assert.Equal(t, 1.0, opts.Scale)
	assert.Equal(t, 120*time.Second, opts.Timeout)
}

func TestNewPDFExporterWithOptions(t *testing.T) {
	opts := PDFOptions{
		PaperWidth:  10.0,
		PaperHeight: 12.0,
		Scale:       1.5,
	}
	exporter := NewPDFExporterWithOptions(opts)
	assert.NotNil(t, exporter)
	assert.Equal(t, 10.0, exporter.options.PaperWidth)
	assert.Equal(t, 12.0, exporter.options.PaperHeight)
	assert.Equal(t, 1.5, exporter.options.Scale)
}

func TestPDFExporter_Name(t *testing.T) {
	exporter := NewPDFExporter()
	assert.Equal(t, "PDF", exporter.Name())
}

func TestPDFExporter_FileExtension(t *testing.T) {
	exporter := NewPDFExporter()
	assert.Equal(t, ".pdf", exporter.FileExtension())
}

// Note: PDFExporter.Export() requires Chrome/Chromium, so we skip actual execution tests
// The Export method would be tested in integration tests with proper Chrome setup
