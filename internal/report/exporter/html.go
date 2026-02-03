// Package exporter provides report export functionality with pluggable exporters.
package exporter

import (
	"fmt"
	"strings"

	"github.com/verustcode/verustcode/internal/model"
	"github.com/verustcode/verustcode/internal/report/assets"
)

// HTMLExporter exports reports to HTML format
type HTMLExporter struct{}

// NewHTMLExporter creates a new HTML exporter
func NewHTMLExporter() *HTMLExporter {
	return &HTMLExporter{}
}

// Export exports a report to a self-contained HTML file
// The HTML includes left sidebar navigation, right content area with markdown rendering,
// syntax highlighting, and dark/light theme toggle
func (e *HTMLExporter) Export(rpt *model.Report, sections []model.ReportSection) (string, error) {
	// Build section tree for hierarchical navigation
	type sectionNode struct {
		ID              string
		SectionID       string
		ParentSectionID string
		Title           string
		Content         string
		IsLeaf          bool
		Children        []sectionNode
	}

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

	// Build parent and child nodes
	parentNodes := []sectionNode{}
	childMap := make(map[string][]sectionNode)

	for _, s := range sortedSections {
		node := sectionNode{
			ID:        fmt.Sprintf("%d", s.ID),
			SectionID: s.SectionID,
			Title:     s.Title,
			Content:   s.Content,
			IsLeaf:    s.IsLeaf,
			Children:  []sectionNode{},
		}
		if s.ParentSectionID != nil && *s.ParentSectionID != "" {
			node.ParentSectionID = *s.ParentSectionID
			childMap[*s.ParentSectionID] = append(childMap[*s.ParentSectionID], node)
		} else {
			parentNodes = append(parentNodes, node)
		}
	}

	// Attach children to parents
	for i := range parentNodes {
		if children, ok := childMap[parentNodes[i].SectionID]; ok {
			parentNodes[i].Children = children
		}
	}

	// Build sections JSON for JavaScript
	var sectionsJSON strings.Builder
	sectionsJSON.WriteString("[")
	for i, p := range parentNodes {
		if i > 0 {
			sectionsJSON.WriteString(",")
		}
		sectionsJSON.WriteString(fmt.Sprintf(`{"id":"%s","sectionId":"%s","title":%s,"content":%s,"isLeaf":%t,"children":[`,
			p.ID, p.SectionID, escapeJSONString(p.Title), escapeJSONString(p.Content), p.IsLeaf || len(p.Children) == 0))
		for j, c := range p.Children {
			if j > 0 {
				sectionsJSON.WriteString(",")
			}
			sectionsJSON.WriteString(fmt.Sprintf(`{"id":"%s","sectionId":"%s","title":%s,"content":%s,"isLeaf":true}`,
				c.ID, c.SectionID, escapeJSONString(c.Title), escapeJSONString(c.Content)))
		}
		sectionsJSON.WriteString("]}")
	}
	sectionsJSON.WriteString("]")

	// Generate HTML
	html := e.generateHTMLTemplate(rpt, sectionsJSON.String())
	return html, nil
}

// Name returns the human-readable name of this exporter
func (e *HTMLExporter) Name() string {
	return "HTML"
}

// FileExtension returns the file extension for HTML files
func (e *HTMLExporter) FileExtension() string {
	return ".html"
}

// generateHTMLTemplate creates the self-contained HTML document
func (e *HTMLExporter) generateHTMLTemplate(rpt *model.Report, sectionsJSON string) string {
	title := rpt.Title
	if title == "" {
		title = fmt.Sprintf("%s Report", rpt.ReportType)
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s</title>
    <!-- Embedded Marked.js for Markdown rendering -->
    <script>%s</script>
    <!-- Embedded Prism.js for code syntax highlighting -->
    <style id="prism-light-style">%s</style>
    <style id="prism-dark-style">%s</style>
    <script>%s</script>
    <!-- Embedded Mermaid.js for diagram rendering -->
    <script>%s</script>
    <style>
        /* CSS Variables - Light Mode */
        :root {
            --background: hsl(0 0%% 99%%);
            --foreground: hsl(220 15%% 15%%);
            --card: hsl(0 0%% 100%%);
            --card-foreground: hsl(220 15%% 15%%);
            --muted: hsl(220 14%% 96%%);
            --muted-foreground: hsl(220 10%% 40%%);
            --accent: hsl(220 60%% 94%%);
            --accent-foreground: hsl(220 60%% 25%%);
            --border: hsl(220 15%% 90%%);
            --primary: hsl(220 60%% 50%%);
            --primary-foreground: hsl(0 0%% 100%%);
            --radius: 0.375rem;
        }
        /* Dark Mode */
        .dark {
            --background: hsl(220 20%% 7%%);
            --foreground: hsl(220 10%% 92%%);
            --card: hsl(220 18%% 10%%);
            --card-foreground: hsl(220 10%% 92%%);
            --muted: hsl(220 15%% 13%%);
            --muted-foreground: hsl(220 8%% 55%%);
            --accent: hsl(220 55%% 18%%);
            --accent-foreground: hsl(220 55%% 85%%);
            --border: hsl(220 15%% 18%%);
            --primary: hsl(220 55%% 55%%);
            --primary-foreground: hsl(0 0%% 100%%);
        }
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            font-family: system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            font-size: 0.875rem;
            line-height: 1.5;
            background: var(--background);
            color: var(--foreground);
        }
        /* Layout */
        .container {
            display: flex;
            height: 100vh;
            overflow: hidden;
        }
        /* Header */
        .header {
            padding: 1rem 1.5rem;
            background: var(--card);
            border-bottom: 1px solid var(--border);
            display: flex;
            justify-content: space-between;
            align-items: center;
        }
        .header h1 {
            font-size: 1.25rem;
            font-weight: 600;
        }
        .header-meta {
            font-size: 0.75rem;
            color: var(--muted-foreground);
        }
        .theme-toggle {
            display: flex;
            align-items: center;
            justify-content: center;
            width: 2.25rem;
            height: 2.25rem;
            padding: 0;
            border: 1px solid var(--border);
            border-radius: var(--radius);
            background: var(--muted);
            color: var(--foreground);
            cursor: pointer;
        }
        .theme-toggle:hover {
            background: var(--accent);
        }
        .theme-toggle svg {
            width: 1rem;
            height: 1rem;
        }
        /* Sidebar */
        .sidebar-wrapper {
            position: relative;
            display: flex;
            flex-shrink: 0;
        }
        .sidebar {
            width: 280px;
            flex-shrink: 0;
            border-right: 1px solid var(--border);
            background: var(--card);
            display: flex;
            flex-direction: column;
            overflow: hidden;
            transition: width 0.3s ease, opacity 0.3s ease;
        }
        .sidebar.collapsed {
            width: 0;
            opacity: 0;
            border-right: none;
        }
        /* Sidebar toggle button - inline in header */
        .sidebar-toggle {
            padding: 0.25rem;
            display: flex;
            align-items: center;
            justify-content: center;
            background: transparent;
            border: none;
            border-radius: var(--radius);
            cursor: pointer;
            color: var(--muted-foreground);
            transition: background 0.15s, color 0.15s;
        }
        .sidebar-toggle:hover {
            background: var(--background);
            color: var(--foreground);
        }
        /* Floating expand button when sidebar is collapsed */
        .sidebar-expand-btn {
            position: fixed;
            left: 0.5rem;
            top: 50%%;
            transform: translateY(-50%%);
            z-index: 30;
            padding: 0.5rem;
            display: none;
            align-items: center;
            justify-content: center;
            background: var(--card);
            border: 1px solid var(--border);
            border-radius: var(--radius);
            cursor: pointer;
            color: var(--muted-foreground);
            box-shadow: 0 2px 8px rgba(0,0,0,0.1);
            transition: background 0.15s, color 0.15s;
        }
        .sidebar-expand-btn:hover {
            background: var(--accent);
            color: var(--foreground);
        }
        .sidebar-expand-btn.show {
            display: flex;
        }
        .sidebar-header {
            padding: 0.75rem 1rem;
            background: var(--muted);
            border-bottom: 1px solid var(--border);
            font-size: 0.875rem;
            font-weight: 500;
            display: flex;
            justify-content: space-between;
            align-items: center;
        }
        .sidebar-header-actions {
            display: flex;
            align-items: center;
            gap: 0.25rem;
        }
        .sidebar-content {
            flex: 1;
            overflow-y: auto;
        }
        .section-group {
            border-bottom: 1px solid var(--border);
        }
        .section-parent {
            width: 100%%;
            padding: 0.5rem 1rem;
            display: flex;
            align-items: center;
            gap: 0.5rem;
            background: hsl(var(--muted) / 0.5);
            border: none;
            font-size: 0.875rem;
            font-weight: 600;
            text-transform: uppercase;
            letter-spacing: 0.05em;
            color: var(--foreground);
            cursor: pointer;
            text-align: left;
        }
        .section-parent:hover {
            background: var(--muted);
        }
        .section-parent .chevron {
            display: inline-flex;
            flex-shrink: 0;
        }
        .section-item {
            width: 100%%;
            padding: 0.625rem 1rem;
            padding-left: 2rem;
            display: flex;
            align-items: center;
            gap: 0.5rem;
            background: transparent;
            border: none;
            border-bottom: 1px solid var(--border);
            font-size: 0.875rem;
            color: var(--foreground);
            cursor: pointer;
            text-align: left;
        }
        .section-item:last-child {
            border-bottom: none;
        }
        .section-item:hover {
            background: var(--muted);
        }
        .section-item.active {
            background: var(--accent);
            color: var(--accent-foreground);
        }
        .section-item.leaf-parent {
            padding-left: 1rem;
        }
        .status-dot {
            width: 0.5rem;
            height: 0.5rem;
            border-radius: 50%%;
            background: hsl(142 70%% 45%%);
            flex-shrink: 0;
        }
        .children-container {
            display: none;
        }
        .children-container.show {
            display: block;
        }
        /* Main Content */
        .main {
            flex: 1;
            display: flex;
            flex-direction: column;
            overflow: hidden;
        }
        .content {
            flex: 1;
            overflow-y: auto;
            padding: 1.5rem;
        }
        .content-inner {
            max-width: 900px;
            margin: 0 auto;
            padding: 1.5rem;
        }
        /* Mermaid containers use full width within content-inner */
        .content-inner .mermaid-container {
            width: 100%%;
        }
        /* Markdown Styles */
        .markdown h1 { font-size: 1.875rem; font-weight: 700; margin: 2rem 0 1rem; padding-bottom: 0.5rem; border-bottom: 2px solid var(--primary); }
        .markdown h2 { font-size: 1.5rem; font-weight: 700; margin: 1.75rem 0 0.75rem; padding-bottom: 0.375rem; border-bottom: 1px solid var(--border); }
        .markdown h3 { font-size: 1.25rem; font-weight: 600; margin: 1.5rem 0 0.5rem; padding-left: 0.75rem; border-left: 4px solid var(--primary); }
        .markdown h4 { font-size: 1.125rem; font-weight: 600; margin: 1.25rem 0 0.5rem; color: var(--muted-foreground); }
        .markdown p { margin: 0.5rem 0; line-height: 1.75; }
        .markdown ul, .markdown ol { margin: 0.5rem 0; padding-left: 1.5rem; }
        .markdown li { margin: 0.25rem 0; }
        .markdown blockquote { border-left: 4px solid var(--primary); padding-left: 1rem; margin: 1rem 0; font-style: italic; color: var(--muted-foreground); }
        /* Inline code - not inside pre (Prism uses language-* classes for code blocks) */
        .markdown code:not([class*="language-"]) { background: var(--muted); padding: 0.125rem 0.375rem; border-radius: 0.25rem; font-family: monospace; font-size: 0.875em; }
        .markdown pre { margin: 1rem 0; border-radius: 0.5rem; overflow: hidden; border: 1px solid var(--border); }
        /* Code blocks - let highlight.js theme control background and colors */
        .markdown pre code { display: block; padding: 1rem; border-radius: 0.5rem; background: transparent; }
        .markdown table { width: 100%%; border-collapse: collapse; margin: 1rem 0; }
        .markdown th, .markdown td { border: 1px solid var(--border); padding: 0.5rem 1rem; text-align: left; }
        .markdown th { background: var(--muted); font-weight: 600; }
        .markdown a { color: var(--primary); text-decoration: none; }
        .markdown a:hover { text-decoration: underline; }
        .markdown hr { border: none; border-top: 1px solid var(--border); margin: 1.5rem 0; }
        /* Scrollbar */
        ::-webkit-scrollbar { width: 8px; height: 8px; }
        ::-webkit-scrollbar-track { background: var(--muted); }
        ::-webkit-scrollbar-thumb { background: var(--muted-foreground); opacity: 0.3; border-radius: 4px; }
        /* Collapse button */
        .collapse-all {
            padding: 0.25rem;
            background: transparent;
            border: none;
            color: var(--muted-foreground);
            cursor: pointer;
            border-radius: var(--radius);
        }
        .collapse-all:hover {
            background: var(--background);
        }
        /* Mermaid diagram container with zoom controls */
        .mermaid-container {
            position: relative;
            margin: 1rem 0;
            border-radius: var(--radius);
            border: 1px solid var(--border);
            background: var(--card);
            overflow: hidden;
        }
        .mermaid-container:hover .mermaid-toolbar {
            opacity: 1;
        }
        .mermaid-toolbar {
            position: absolute;
            top: 0.5rem;
            right: 0.5rem;
            z-index: 10;
            display: flex;
            align-items: center;
            gap: 0.25rem;
            padding: 0.25rem;
            border-radius: var(--radius);
            border: 1px solid var(--border);
            background: var(--background);
            opacity: 0;
            transition: opacity 0.2s;
            box-shadow: 0 1px 3px rgba(0,0,0,0.1);
        }
        .mermaid-toolbar-btn {
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 0.375rem;
            border: none;
            border-radius: var(--radius);
            background: transparent;
            color: var(--muted-foreground);
            cursor: pointer;
            transition: background 0.15s, color 0.15s;
        }
        .mermaid-toolbar-btn:hover {
            background: var(--accent);
            color: var(--foreground);
        }
        .mermaid-toolbar-btn svg {
            width: 16px;
            height: 16px;
        }
        .mermaid-zoom-label {
            font-size: 0.75rem;
            color: var(--muted-foreground);
            min-width: 3rem;
            text-align: center;
        }
        .mermaid-toolbar-divider {
            width: 1px;
            height: 1rem;
            background: var(--border);
            margin: 0 0.25rem;
        }
        .mermaid-content {
            overflow: auto;
            padding: 1rem;
            display: flex;
            justify-content: center;
        }
        .mermaid-svg-wrapper {
            transition: transform 0.2s;
            transform-origin: center top;
            min-width: 100%%;
        }
        /* Mermaid's useMaxWidth handles sizing - allow diagrams to use available space */
        .mermaid-svg-wrapper svg {
            width: 100%%;
            height: auto;
            min-width: 600px;
        }
        /* Mermaid preview modal */
        .mermaid-modal {
            position: fixed;
            inset: 0;
            z-index: 100;
            display: flex;
            align-items: center;
            justify-content: center;
            background: rgba(0,0,0,0.8);
        }
        .mermaid-modal-content {
            position: relative;
            width: 90vw;
            height: 90vh;
            background: var(--background);
            border-radius: var(--radius);
            overflow: hidden;
            display: flex;
            flex-direction: column;
        }
        .mermaid-modal-header {
            display: flex;
            align-items: center;
            justify-content: space-between;
            padding: 0.75rem 1rem;
            border-bottom: 1px solid var(--border);
        }
        .mermaid-modal-controls {
            display: flex;
            align-items: center;
            gap: 0.25rem;
        }
        .mermaid-modal-close {
            padding: 0.5rem;
            border: none;
            border-radius: var(--radius);
            background: transparent;
            color: var(--muted-foreground);
            cursor: pointer;
            transition: background 0.15s, color 0.15s;
        }
        .mermaid-modal-close:hover {
            background: var(--accent);
            color: var(--foreground);
        }
        .mermaid-modal-close svg {
            width: 20px;
            height: 20px;
        }
        .mermaid-modal-body {
            flex: 1;
            overflow: auto;
            padding: 1rem;
            display: flex;
        }
        .mermaid-modal-svg-wrapper {
            transition: transform 0.2s;
            transform-origin: center top;
            display: flex;
            justify-content: center;
            margin: auto;
        }
        /* Mermaid's useMaxWidth handles sizing - just ensure it doesn't overflow */
        .mermaid-modal-svg-wrapper svg {
            max-width: 100%%;
            height: auto;
        }
    </style>
</head>
<body>
    <div class="container">
        <!-- Floating expand button when sidebar is collapsed -->
        <button class="sidebar-expand-btn" onclick="toggleSidebar()" title="Expand Sidebar" id="sidebar-expand-btn"></button>
        <div class="sidebar-wrapper">
            <aside class="sidebar" id="sidebar">
                <div class="sidebar-header">
                    <span>Sections</span>
                    <div class="sidebar-header-actions">
                        <button class="collapse-all" onclick="toggleAllSections()" title="Expand/Collapse All" id="collapse-all-btn"></button>
                        <button class="sidebar-toggle" onclick="toggleSidebar()" title="Collapse Sidebar" id="sidebar-toggle"></button>
                    </div>
                </div>
                <div class="sidebar-content" id="sidebar-content"></div>
            </aside>
        </div>
        <main class="main">
            <header class="header">
                <div style="display:flex; align-items:center; gap:12px;">
                    <!-- Logo -->
                    %s
                    <div>
                        <h1>%s</h1>
                        <div class="header-meta">%s · %s</div>
                    </div>
                </div>
                <button class="theme-toggle" id="theme-toggle-btn" onclick="toggleTheme()" title="Toggle Theme"></button>
            </header>
            <div class="content">
                <div class="content-inner markdown" id="content"></div>
            </div>
        </main>
    </div>
    <script>
        // Report sections data
        const sections = %s;
        
        // State
        let selectedId = null;
        let expandedParents = new Set();
        let mermaidCounter = 0;
        let sidebarCollapsed = false;
        
        // Initialize Mermaid
        // Initialize Mermaid with useMaxWidth for proper diagram sizing
        // Wide diagrams fill container, narrow ones keep original proportions
        function initMermaid() {
            const isDark = document.body.classList.contains('dark');
            mermaid.initialize({
                startOnLoad: false,
                theme: isDark ? 'dark' : 'default',
                securityLevel: 'loose',
                fontFamily: 'system-ui, -apple-system, sans-serif',
                flowchart: { useMaxWidth: true },
                sequence: { useMaxWidth: true },
                gantt: { useMaxWidth: true },
                journey: { useMaxWidth: true },
                class: { useMaxWidth: true },
                state: { useMaxWidth: true },
                er: { useMaxWidth: true },
                pie: { useMaxWidth: true }
            });
        }
        
        // SVG Icons (same as frontend for consistency)
        const MermaidIcons = {
            zoomIn: '<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><line x1="21" x2="16.65" y1="21" y2="16.65"/><line x1="11" x2="11" y1="8" y2="14"/><line x1="8" x2="14" y1="11" y2="11"/></svg>',
            zoomOut: '<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><line x1="21" x2="16.65" y1="21" y2="16.65"/><line x1="8" x2="14" y1="11" y2="11"/></svg>',
            reset: '<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/><path d="M3 3v5h5"/></svg>',
            maximize: '<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="15 3 21 3 21 9"/><polyline points="9 21 3 21 3 15"/><line x1="21" x2="14" y1="3" y2="10"/><line x1="3" x2="10" y1="21" y2="14"/></svg>',
            close: '<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>'
        };
        
        // Sidebar Icons (same as frontend for consistency)
        const SidebarIcons = {
            chevronRight: '<svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m9 18 6-6-6-6"/></svg>',
            chevronDown: '<svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m6 9 6 6 6-6"/></svg>',
            chevronsUpDown: '<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m7 15 5 5 5-5"/><path d="m7 9 5-5 5 5"/></svg>',
            panelLeftClose: '<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="18" height="18" x="3" y="3" rx="2"/><path d="M9 3v18"/><path d="m16 15-3-3 3-3"/></svg>',
            panelLeftOpen: '<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="18" height="18" x="3" y="3" rx="2"/><path d="M9 3v18"/><path d="m14 9 3 3-3 3"/></svg>'
        };
        
        // Theme Icons (same as frontend for consistency)
        const ThemeIcons = {
            sun: '<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="4"/><path d="M12 2v2"/><path d="M12 20v2"/><path d="m4.93 4.93 1.41 1.41"/><path d="m17.66 17.66 1.41 1.41"/><path d="M2 12h2"/><path d="M20 12h2"/><path d="m6.34 17.66-1.41 1.41"/><path d="m19.07 4.93-1.41 1.41"/></svg>',
            moon: '<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9Z"/></svg>'
        };
        
        // Store mermaid diagram states
        const mermaidStates = new Map();
        
        // Zoom limits (same as frontend for consistency)
        const MERMAID_ZOOM_MIN = 0.5;  // 50%%
        const MERMAID_ZOOM_MAX = 2.5;  // 250%%
        const MERMAID_ZOOM_STEP = 0.1; // 10%% per step
        
        // Open mermaid preview modal
        // SVG from Mermaid with useMaxWidth: true already has proper sizing
        function openMermaidPreview(svgContent) {
            // Create modal
            const modal = document.createElement('div');
            modal.className = 'mermaid-modal';
            modal.id = 'mermaid-preview-modal';
            
            modal.innerHTML = 
                '<div class="mermaid-modal-content" onclick="event.stopPropagation()">' +
                    '<div class="mermaid-modal-header">' +
                        '<div class="mermaid-modal-controls">' +
                            '<button class="mermaid-toolbar-btn" onclick="mermaidModalZoom(-MERMAID_ZOOM_STEP)" title="Zoom Out">' + MermaidIcons.zoomOut + '</button>' +
                            '<span class="mermaid-zoom-label mermaid-modal-zoom-label">100%%</span>' +
                            '<button class="mermaid-toolbar-btn" onclick="mermaidModalZoom(MERMAID_ZOOM_STEP)" title="Zoom In">' + MermaidIcons.zoomIn + '</button>' +
                            '<button class="mermaid-toolbar-btn" onclick="mermaidModalReset()" title="Reset Zoom">' + MermaidIcons.reset + '</button>' +
                        '</div>' +
                        '<span style="font-size:0.75rem;color:var(--muted-foreground)">Ctrl/Cmd + Scroll to zoom</span>' +
                        '<button class="mermaid-modal-close" onclick="closeMermaidPreview()" title="Close">' + MermaidIcons.close + '</button>' +
                    '</div>' +
                    '<div class="mermaid-modal-body" id="mermaid-modal-body">' +
                        '<div class="mermaid-modal-svg-wrapper">' + svgContent + '</div>' +
                    '</div>' +
                '</div>';
            
            modal.onclick = () => closeMermaidPreview();
            document.body.appendChild(modal);
            document.body.style.overflow = 'hidden';
            
            // Store modal state
            window.mermaidModalScale = 1;
            
            // Add wheel zoom handler
            const modalBody = document.getElementById('mermaid-modal-body');
            if (modalBody) {
                modalBody.addEventListener('wheel', function(e) {
                    if (e.ctrlKey || e.metaKey) {
                        e.preventDefault();
                        const delta = e.deltaY < 0 ? MERMAID_ZOOM_STEP : -MERMAID_ZOOM_STEP;
                        mermaidModalZoom(delta);
                    }
                }, { passive: false });
            }
        }
        
        // Modal zoom controls
        function mermaidModalZoom(delta) {
            window.mermaidModalScale = Math.max(MERMAID_ZOOM_MIN, Math.min(MERMAID_ZOOM_MAX, window.mermaidModalScale + delta));
            const modal = document.getElementById('mermaid-preview-modal');
            if (modal) {
                const wrapper = modal.querySelector('.mermaid-modal-svg-wrapper');
                const label = modal.querySelector('.mermaid-modal-zoom-label');
                if (wrapper) wrapper.style.transform = 'scale(' + window.mermaidModalScale + ')';
                if (label) label.textContent = Math.round(window.mermaidModalScale * 100) + '%%';
            }
        }
        
        function mermaidModalReset() {
            window.mermaidModalScale = 1;
            mermaidModalZoom(0);
        }
        
        // Close mermaid preview modal
        function closeMermaidPreview() {
            const modal = document.getElementById('mermaid-preview-modal');
            if (modal) {
                modal.remove();
                document.body.style.overflow = '';
            }
        }
        
        // Handle escape key for modal
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') closeMermaidPreview();
        });
        
        // Mermaid zoom controls for inline diagrams
        function mermaidZoom(containerId, delta) {
            const state = mermaidStates.get(containerId) || { scale: 1 };
            state.scale = Math.max(MERMAID_ZOOM_MIN, Math.min(MERMAID_ZOOM_MAX, state.scale + delta));
            mermaidStates.set(containerId, state);
            
            const container = document.getElementById(containerId);
            if (container) {
                const wrapper = container.querySelector('.mermaid-svg-wrapper');
                const label = container.querySelector('.mermaid-zoom-label');
                if (wrapper) wrapper.style.transform = 'scale(' + state.scale + ')';
                if (label) label.textContent = Math.round(state.scale * 100) + '%%';
            }
        }
        
        function mermaidResetZoom(containerId) {
            mermaidStates.set(containerId, { scale: 1 });
            mermaidZoom(containerId, 0);
        }
        
        // Render Mermaid diagrams in content
        async function renderMermaidDiagrams() {
            const contentEl = document.getElementById('content');
            // Find all code blocks with language-mermaid class
            const mermaidBlocks = contentEl.querySelectorAll('pre code.language-mermaid');
            
            for (const block of mermaidBlocks) {
                const code = block.textContent;
                const pre = block.parentElement;
                
                try {
                    const id = 'mermaid-' + (++mermaidCounter);
                    const containerId = 'mermaid-container-' + mermaidCounter;
                    // Mermaid with useMaxWidth: true generates SVG with proper sizing
                    const { svg } = await mermaid.render(id, code);
                    
                    // Initialize state (store svg for preview)
                    mermaidStates.set(containerId, { scale: 1, svg: svg });
                    
                    // Create container with toolbar
                    const container = document.createElement('div');
                    container.className = 'mermaid-container';
                    container.id = containerId;
                    container.innerHTML = 
                        '<div class="mermaid-toolbar">' +
                            '<button class="mermaid-toolbar-btn" onclick="mermaidZoom(\'' + containerId + '\', -MERMAID_ZOOM_STEP)" title="Zoom Out">' + MermaidIcons.zoomOut + '</button>' +
                            '<span class="mermaid-zoom-label">100%%</span>' +
                            '<button class="mermaid-toolbar-btn" onclick="mermaidZoom(\'' + containerId + '\', MERMAID_ZOOM_STEP)" title="Zoom In">' + MermaidIcons.zoomIn + '</button>' +
                            '<button class="mermaid-toolbar-btn" onclick="mermaidResetZoom(\'' + containerId + '\')" title="Reset Zoom">' + MermaidIcons.reset + '</button>' +
                            '<div class="mermaid-toolbar-divider"></div>' +
                            '<button class="mermaid-toolbar-btn" onclick="openMermaidPreview(mermaidStates.get(\'' + containerId + '\').svg)" title="Preview">' + MermaidIcons.maximize + '</button>' +
                        '</div>' +
                        '<div class="mermaid-content">' +
                            '<div class="mermaid-svg-wrapper">' + svg + '</div>' +
                        '</div>';
                    
                    // Replace pre element with the rendered diagram
                    pre.replaceWith(container);
                } catch (err) {
                    console.error('Mermaid render error:', err);
                    // Show error message
                    pre.style.cssText = 'border:1px solid #f87171;background:#fef2f2;padding:1rem;border-radius:0.5rem;';
                    block.innerHTML = '<span style="color:#dc2626;">Failed to render diagram:</span>\n' + escapeHtml(code);
                }
            }
        }
        
        // Check if there are any collapsible sections (parents with children)
        function hasCollapsibleSections() {
            return sections.some(s => s.children && s.children.length > 0);
        }
        
        // Initialize
        document.addEventListener('DOMContentLoaded', () => {
            initMermaid();
            // Set initial theme styles
            updateThemeStyles();
            // Set initial theme toggle icon
            updateThemeIcon();
            // Set initial collapse-all button icon, hide if no collapsible sections
            const collapseBtn = document.getElementById('collapse-all-btn');
            if (collapseBtn) {
                if (hasCollapsibleSections()) {
                    collapseBtn.innerHTML = SidebarIcons.chevronsUpDown;
                    collapseBtn.style.display = '';
                } else {
                    // Hide collapse button if no collapsible sections
                    collapseBtn.style.display = 'none';
                }
            }
            // Set initial sidebar toggle button icon
            const toggleBtn = document.getElementById('sidebar-toggle');
            if (toggleBtn) {
                toggleBtn.innerHTML = SidebarIcons.panelLeftClose;
            }
            renderSidebar();
            // Select first leaf section
            if (sections.length > 0) {
                const first = sections[0];
                if (first.isLeaf || first.children.length === 0) {
                    selectSection(first.id, first.content);
                } else if (first.children.length > 0) {
                    expandedParents.add(first.sectionId);
                    selectSection(first.children[0].id, first.children[0].content);
                }
                renderSidebar();
            }
        });
        
        // Render sidebar
        function renderSidebar() {
            const container = document.getElementById('sidebar-content');
            container.innerHTML = '';
            
            sections.forEach(parent => {
                const group = document.createElement('div');
                group.className = 'section-group';
                
                const hasChildren = parent.children && parent.children.length > 0;
                const isExpanded = expandedParents.has(parent.sectionId);
                
                if (hasChildren) {
                    // Collapsible parent
                    const btn = document.createElement('button');
                    btn.className = 'section-parent' + (isExpanded ? ' expanded' : '');
                    const chevronIcon = isExpanded ? SidebarIcons.chevronDown : SidebarIcons.chevronRight;
                    btn.innerHTML = '<span class="chevron">' + chevronIcon + '</span> ' + escapeHtml(parent.title);
                    btn.onclick = () => {
                        if (expandedParents.has(parent.sectionId)) {
                            expandedParents.delete(parent.sectionId);
                        } else {
                            expandedParents.add(parent.sectionId);
                        }
                        renderSidebar();
                    };
                    group.appendChild(btn);
                    
                    // Children container
                    const childrenDiv = document.createElement('div');
                    childrenDiv.className = 'children-container' + (isExpanded ? ' show' : '');
                    parent.children.forEach(child => {
                        const item = document.createElement('button');
                        item.className = 'section-item' + (selectedId === child.id ? ' active' : '');
                        item.innerHTML = '<span class="status-dot"></span> ' + escapeHtml(child.title);
                        item.onclick = () => selectSection(child.id, child.content);
                        childrenDiv.appendChild(item);
                    });
                    group.appendChild(childrenDiv);
                } else {
                    // Leaf parent - clickable
                    const item = document.createElement('button');
                    item.className = 'section-item leaf-parent' + (selectedId === parent.id ? ' active' : '');
                    item.innerHTML = '<span class="status-dot"></span> ' + escapeHtml(parent.title);
                    item.onclick = () => selectSection(parent.id, parent.content);
                    group.appendChild(item);
                }
                
                container.appendChild(group);
            });
        }
        
        // Select section
        async function selectSection(id, content) {
            selectedId = id;
            renderSidebar();
            
            const contentEl = document.getElementById('content');
            if (content) {
                contentEl.innerHTML = marked.parse(content);
                // Highlight code blocks (except mermaid)
                contentEl.querySelectorAll('pre code').forEach(block => {
                    if (!block.classList.contains('language-mermaid')) {
                        Prism.highlightElement(block);
                    }
                });
                // Render Mermaid diagrams
                await renderMermaidDiagrams();
            } else {
                contentEl.innerHTML = '<p style="text-align:center;color:var(--muted-foreground)">No content available</p>';
            }
            // Scroll to top
            document.querySelector('.content').scrollTop = 0;
        }
        
        // Toggle all sections
        function toggleAllSections() {
            const parentIds = sections.filter(s => s.children && s.children.length > 0).map(s => s.sectionId);
            if (expandedParents.size > 0) {
                expandedParents.clear();
            } else {
                parentIds.forEach(id => expandedParents.add(id));
            }
            renderSidebar();
        }
        
        // Toggle sidebar collapse/expand
        function toggleSidebar() {
            sidebarCollapsed = !sidebarCollapsed;
            const sidebar = document.getElementById('sidebar');
            const toggleBtn = document.getElementById('sidebar-toggle');
            const expandBtn = document.getElementById('sidebar-expand-btn');
            if (sidebar) {
                if (sidebarCollapsed) {
                    sidebar.classList.add('collapsed');
                } else {
                    sidebar.classList.remove('collapsed');
                }
            }
            if (toggleBtn) {
                toggleBtn.innerHTML = SidebarIcons.panelLeftClose;
                toggleBtn.title = 'Collapse Sidebar';
            }
            // Show/hide floating expand button when sidebar is collapsed
            if (expandBtn) {
                if (sidebarCollapsed) {
                    expandBtn.classList.add('show');
                    expandBtn.innerHTML = SidebarIcons.panelLeftOpen;
                } else {
                    expandBtn.classList.remove('show');
                }
            }
        }
        
        // Update theme toggle icon based on current theme
        function updateThemeIcon() {
            const isDark = document.body.classList.contains('dark');
            const btn = document.getElementById('theme-toggle-btn');
            if (btn) {
                btn.innerHTML = isDark ? ThemeIcons.sun : ThemeIcons.moon;
            }
        }
        
        // Update theme styles based on current theme
        function updateThemeStyles() {
            const isDark = document.body.classList.contains('dark');
            const lightStyle = document.getElementById('prism-light-style');
            const darkStyle = document.getElementById('prism-dark-style');
            if (lightStyle) lightStyle.disabled = isDark;
            if (darkStyle) darkStyle.disabled = !isDark;
        }

        // Theme toggle
        async function toggleTheme() {
            document.body.classList.toggle('dark');
            updateThemeStyles();
            // Update theme toggle icon
            updateThemeIcon();
            // Re-highlight code blocks
            document.querySelectorAll('pre code').forEach(block => {
                if (!block.classList.contains('language-mermaid')) {
                    Prism.highlightElement(block);
                }
            });
            // Re-initialize Mermaid with new theme and re-render diagrams
            initMermaid();
            // Re-render current section to update mermaid diagrams
            const currentSection = findSectionById(selectedId);
            if (currentSection && currentSection.content) {
                const contentEl = document.getElementById('content');
                contentEl.innerHTML = marked.parse(currentSection.content);
                document.querySelectorAll('pre code').forEach(block => {
                    if (!block.classList.contains('language-mermaid')) {
                        Prism.highlightElement(block);
                    }
                });
                await renderMermaidDiagrams();
            }
        }
        
        // Find section by ID
        function findSectionById(id) {
            for (const parent of sections) {
                if (parent.id === id) return parent;
                for (const child of (parent.children || [])) {
                    if (child.id === id) return child;
                }
            }
            return null;
        }
        
        // Escape HTML
        function escapeHtml(text) {
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        }
    </script>
</body>
</html>`,
		escapeHTMLAttr(title), // <title>
		assets.MarkedJS,       // Marked.js
		assets.PrismCSSLight,  // Prism.js light CSS
		assets.PrismCSSDark,   // Prism.js dark CSS
		assets.PrismJS,        // Prism.js
		assets.MermaidJS,      // Mermaid.js
		getLogoSVG(32, 32),    // Logo SVG
		escapeHTMLAttr(title), // header h1
		escapeHTMLAttr(rpt.RepoURL),
		escapeHTMLAttr(rpt.Ref),
		sectionsJSON)
}
