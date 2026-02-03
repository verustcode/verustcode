// Package exporter provides report export functionality with pluggable exporters.
package exporter

import (
	"fmt"
	"strings"

	"github.com/verustcode/verustcode/internal/model"
	"github.com/verustcode/verustcode/internal/report/assets"
)

// escapeJSONString escapes a string for safe JSON embedding
func escapeJSONString(s string) string {
	// Use Go's default JSON escaping by building manually
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	// Escape script tags to prevent XSS
	s = strings.ReplaceAll(s, "</script>", "<\\/script>")
	s = strings.ReplaceAll(s, "<script>", "<script\\>")
	return "\"" + s + "\""
}

// escapeHTMLAttr escapes a string for safe HTML attribute embedding
func escapeHTMLAttr(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	return s
}

// escapeYAMLString escapes special characters in YAML strings
func escapeYAMLString(s string) string {
	// Escape double quotes and backslashes
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}

// MergeSections merges report sections into a single Markdown document
func MergeSections(report *model.Report, sections []model.ReportSection) string {
	var sb strings.Builder

	// Report title
	sb.WriteString("# ")
	if report.Title != "" {
		sb.WriteString(report.Title)
	} else {
		sb.WriteString("Code Analysis Report")
	}
	sb.WriteString("\n\n")

	// Build parent-child relationships map
	parentSections := make(map[string]*model.ReportSection) // parent_id -> parent section
	childrenMap := make(map[string][]model.ReportSection)   // parent_id -> children
	var topLevelSections []model.ReportSection

	for i := range sections {
		section := &sections[i]
		if section.ParentSectionID == nil {
			// Top-level section (could be parent or standalone leaf)
			topLevelSections = append(topLevelSections, *section)
			if !section.IsLeaf {
				parentSections[section.SectionID] = section
			}
		} else {
			// Subsection
			parentID := *section.ParentSectionID
			childrenMap[parentID] = append(childrenMap[parentID], *section)
		}
	}

	// Table of contents - supports hierarchical structure
	sb.WriteString("## Table of Contents\n\n")
	parentNum := 0
	for _, section := range topLevelSections {
		parentNum++
		anchor := createAnchor(section.Title)

		if section.IsLeaf {
			// Standalone section (no subsections)
			sb.WriteString(fmt.Sprintf("%d. [%s](#%s)\n", parentNum, section.Title, anchor))
		} else {
			// Parent section with subsections
			sb.WriteString(fmt.Sprintf("%d. %s\n", parentNum, section.Title))
			if children, ok := childrenMap[section.SectionID]; ok {
				for j, child := range children {
					childAnchor := createAnchor(child.Title)
					sb.WriteString(fmt.Sprintf("   %d.%d. [%s](#%s)\n", parentNum, j+1, child.Title, childAnchor))
				}
			}
		}
	}
	sb.WriteString("\n---\n\n")

	// Section contents - organized by hierarchy
	for _, section := range topLevelSections {
		if section.IsLeaf {
			// Standalone leaf section - output content directly
			if section.Content != "" {
				sb.WriteString(section.Content)
				sb.WriteString("\n\n---\n\n")
			} else if section.Status == model.SectionStatusFailed {
				sb.WriteString(fmt.Sprintf("## %s\n\n", section.Title))
				sb.WriteString("*Section generation failed*\n\n")
				if section.ErrorMessage != "" {
					sb.WriteString(fmt.Sprintf("Error: %s\n\n", section.ErrorMessage))
				}
				sb.WriteString("---\n\n")
			}
		} else {
			// Parent section - output children's content
			if children, ok := childrenMap[section.SectionID]; ok {
				for _, child := range children {
					if child.Content != "" {
						sb.WriteString(child.Content)
						sb.WriteString("\n\n---\n\n")
					} else if child.Status == model.SectionStatusFailed {
						sb.WriteString(fmt.Sprintf("### %s\n\n", child.Title))
						sb.WriteString("*Section generation failed*\n\n")
						if child.ErrorMessage != "" {
							sb.WriteString(fmt.Sprintf("Error: %s\n\n", child.ErrorMessage))
						}
						sb.WriteString("---\n\n")
					}
				}
			}
		}
	}

	return strings.TrimSuffix(sb.String(), "---\n\n")
}

// createAnchor creates an anchor-friendly ID from a title
func createAnchor(title string) string {
	// Convert to lowercase and replace spaces with hyphens
	anchor := strings.ToLower(title)
	anchor = strings.ReplaceAll(anchor, " ", "-")
	// Remove special characters that might break anchors
	anchor = strings.ReplaceAll(anchor, ":", "")
	anchor = strings.ReplaceAll(anchor, "(", "")
	anchor = strings.ReplaceAll(anchor, ")", "")
	anchor = strings.ReplaceAll(anchor, "/", "-")
	return anchor
}

// getLogoSVG returns the logo SVG with specified width and height
func getLogoSVG(width, height int) string {
	// assets.LogoSVG has viewBox="0 0 100 100", we add width and height attributes
	logoContent := strings.TrimSpace(assets.LogoSVG)
	// Insert width and height after the opening <svg tag
	logoContent = strings.Replace(logoContent, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"`,
		fmt.Sprintf(`<svg width="%d" height="%d" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"`, width, height), 1)
	return logoContent
}
