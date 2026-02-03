// Package assets provides embedded JavaScript, CSS libraries, and images for HTML/PDF export.
package assets

import (
	_ "embed"
)

// Mermaid.js for diagram rendering
//
//go:embed mermaid.min.js
var MermaidJS string

// Marked.js for Markdown rendering
//
//go:embed marked.min.js
var MarkedJS string

// Prism.js for code syntax highlighting (matches react-syntax-highlighter Prism)
//
//go:embed prism.min.js
var PrismJS string

// Prism.js One Light theme CSS (matches react-syntax-highlighter oneLight)
//
//go:embed prism-one-light.min.css
var PrismCSSLight string

// Prism.js One Dark theme CSS (matches react-syntax-highlighter oneDark)
//
//go:embed prism-one-dark.min.css
var PrismCSSDark string

// VerustCode Logo SVG - Shield + Code design
// Used in PDF header and HTML report header
//
//go:embed logo.svg
var LogoSVG string
