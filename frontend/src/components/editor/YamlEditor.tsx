import { useRef, useCallback } from 'react'
import Editor, { type OnMount } from '@monaco-editor/react'
import type { editor } from 'monaco-editor'
import { useTheme } from '@/hooks/useTheme.tsx'

// Known top-level YAML sections for config files
const TOP_LEVEL_SECTIONS = [
  'server',
  'admin',
  'auth',
  'database',
  'git',
  'agents',
  'review',
  'logging',
  'telemetry',
]

// Known top-level sections for rule files
const RULE_SECTIONS = ['version', 'rule_base', 'rules']

/**
 * Cursor position info for YAML editor
 * For config files: returns section name (server, admin, etc.)
 * For rule files: returns section + ruleId if cursor is inside a rule
 */
export interface CursorPosition {
  section: string | null
  ruleId?: string
}

interface YamlEditorProps {
  value: string
  onChange: (value: string) => void
  /** 
   * Callback for cursor position changes
   * Can receive either string (legacy) or CursorPosition object
   */
  onCursorChange?: (section: string | null) => void
  /**
   * Enhanced callback that provides more context for rule files
   * Returns ruleId when cursor is inside a rule definition
   */
  onCursorPositionChange?: (position: CursorPosition) => void
  readOnly?: boolean
  height?: string
}

/**
 * Monaco-based YAML editor component
 */
export function YamlEditor({
  value,
  onChange,
  onCursorChange,
  onCursorPositionChange,
  readOnly = false,
  height = '100%',
}: YamlEditorProps) {
  const { isDark } = useTheme()
  const editorRef = useRef<editor.IStandaloneCodeEditor | null>(null)

  // Find the top-level section for a given line number (for config files)
  const findSectionAtLine = useCallback((lineNumber: number, content: string): string | null => {
    const lines = content.split('\n')
    // Search backwards from current line to find the nearest top-level section
    for (let i = lineNumber - 1; i >= 0; i--) {
      const line = lines[i]
      if (!line) continue
      // Check if line starts with a top-level key (no leading whitespace)
      const match = line.match(/^([a-z_]+):/)
      if (match) {
        const key = match[1]
        if (TOP_LEVEL_SECTIONS.includes(key)) {
          return key
        }
      }
    }
    return null
  }, [])

  // Find cursor position with rule context (for rule files)
  const findCursorPosition = useCallback((lineNumber: number, content: string): CursorPosition => {
    const lines = content.split('\n')
    let section: string | null = null
    let ruleId: string | undefined
    let lastRuleId: string | undefined

    // Search backwards from current line
    for (let i = lineNumber - 1; i >= 0; i--) {
      const line = lines[i]
      if (!line) continue

      // Check for top-level section (no leading whitespace)
      const sectionMatch = line.match(/^([a-z_]+):/)
      if (sectionMatch) {
        const key = sectionMatch[1]
        if (RULE_SECTIONS.includes(key)) {
          section = key
          // If we're in the rules section, use the last found rule id
          if (key === 'rules') {
            ruleId = lastRuleId
          }
          break
        }
        // Also check config sections
        if (TOP_LEVEL_SECTIONS.includes(key)) {
          section = key
          break
        }
      }

      // Check for rule id pattern: "  - id: xxx" or "- id: xxx"
      const ruleIdMatch = line.match(/^\s*-\s*id:\s*(.+)$/)
      if (ruleIdMatch && !lastRuleId) {
        lastRuleId = ruleIdMatch[1].trim()
      }
    }

    return { section, ruleId }
  }, [])

  // Handle editor mount
  const handleEditorMount: OnMount = useCallback((editor) => {
    editorRef.current = editor

    // Listen to cursor position changes
    editor.onDidChangeCursorPosition((e) => {
      const content = editor.getValue()
      
      // Call legacy callback if provided
      if (onCursorChange) {
        const section = findSectionAtLine(e.position.lineNumber, content)
        onCursorChange(section)
      }
      
      // Call enhanced callback if provided
      if (onCursorPositionChange) {
        const position = findCursorPosition(e.position.lineNumber, content)
        onCursorPositionChange(position)
      }
    })
  }, [onCursorChange, onCursorPositionChange, findSectionAtLine, findCursorPosition])

  return (
    <Editor
      height={height}
      language="yaml"
      theme={isDark ? 'vs-dark' : 'light'}
      value={value}
      onChange={(v) => onChange(v || '')}
      onMount={handleEditorMount}
      options={{
        readOnly,
        minimap: { enabled: false },
        fontSize: 13,
        lineNumbers: 'on',
        scrollBeyondLastLine: false,
        automaticLayout: true,
        tabSize: 2,
        wordWrap: 'on',
        folding: true,
        renderLineHighlight: 'line',
        scrollbar: {
          verticalScrollbarSize: 8,
          horizontalScrollbarSize: 8,
        },
      }}
    />
  )
}
