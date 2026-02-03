import { useParams, Link, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useState, useMemo, useCallback, useEffect } from 'react'
import {
  ArrowLeft,
  ExternalLink,
  Clock,
  Calendar,
  GitBranch,
  AlertCircle,
  Repeat,
  User,
  Plus,
  Minus,
  FileCode,
  Copy,
  Check,
  ChevronLeft,
  ChevronRight,
  RotateCw,
  ChevronDown,
  CheckCircle2,
  CircleDot,
  CircleAlert,
} from 'lucide-react'
import { JsonView } from 'react-json-view-lite'
import 'react-json-view-lite/dist/index.css'
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter'
import { oneDark, oneLight } from 'react-syntax-highlighter/dist/esm/styles/prism'
import { useAccordionState } from '@/hooks/useAccordionState'
import { MarkdownRenderer } from '@/components/common/MarkdownRenderer'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Skeleton } from '@/components/ui/skeleton'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import { Switch } from '@/components/ui/switch'
import { Label } from '@/components/ui/label'
import { StatusBadge } from '@/components/common/StatusBadge'
import { LogViewer } from '@/components/common/LogViewer'
import { api } from '@/lib/api'
import { toast } from '@/hooks/useToast'
import { formatDate, formatDuration, formatRepoUrl } from '@/lib/utils'
import type { Review, ReviewRule } from '@/types/review'

/**
 * Get result data from rule's results array
 */
function getResultData(rule: ReviewRule): Record<string, unknown> | null {
  if (!rule.results || rule.results.length === 0) {
    return null
  }
  return rule.results[0].data
}

/**
 * Severity badge colors
 */
const severityColors: Record<string, string> = {
  critical: 'bg-red-500/15 text-red-600 dark:text-red-400 border border-red-500/30',
  high: 'bg-orange-500/15 text-orange-600 dark:text-orange-400 border border-orange-500/30',
  medium: 'bg-amber-500/15 text-amber-600 dark:text-amber-400 border border-amber-500/30',
  low: 'bg-blue-500/15 text-blue-600 dark:text-blue-400 border border-blue-500/30',
  info: 'bg-sky-500/15 text-sky-600 dark:text-sky-400 border border-sky-500/30',
}


/**
 * History compare status configuration
 * Defines colors and icons for each status type
 */
const statusConfig: Record<string, { 
  label: string
  icon: React.ComponentType<{ className?: string }>
  className: string
}> = {
  fixed: {
    label: 'FIXED',
    icon: CheckCircle2,
    className: 'text-green-600 dark:text-green-400',
  },
  new: {
    label: 'NEW',
    icon: CircleAlert,
    className: 'text-red-600 dark:text-red-400',
  },
  persists: {
    label: 'PERSISTS',
    icon: CircleDot,
    className: 'text-amber-600 dark:text-amber-400',
  },
}

/**
 * Render a finding item with full schema support
 * Base fields are rendered explicitly, extra_fields are rendered dynamically via AdditionalFields
 */
function FindingItem({ 
  finding, 
  index,
  itemSchema 
}: { 
  finding: Record<string, unknown>
  index: number
  itemSchema?: Record<string, Record<string, unknown>>
}) {
  const { t } = useTranslation()
  
  // Detect dark mode for syntax highlighter
  const [isDark, setIsDark] = useState(false)
  useEffect(() => {
    setIsDark(document.documentElement.classList.contains('dark'))
    const observer = new MutationObserver((mutations) => {
      mutations.forEach((mutation) => {
        if (mutation.attributeName === 'class') {
          setIsDark(document.documentElement.classList.contains('dark'))
        }
      })
    })
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })
    return () => observer.disconnect()
  }, [])
  
  // Base schema fields only (defined in GetDefaultJSONSchema)
  const severity = (finding.severity as string) || 'info'
  const description = finding.description as string
  const location = finding.location as string
  const category = finding.category as string
  const suggestion = finding.suggestion as string
  const codeSnippet = finding.code_snippet as string
  
  // History compare status field (lowercase: fixed, new, persists)
  const status = (finding.status as string)?.toLowerCase()
  const statusInfo = status ? statusConfig[status] : null

  return (
    <div className="pl-4 py-3 pr-3 rounded-md bg-[hsl(var(--muted))]/30">
      {/* Header: index, status, severity, category */}
      <div className="flex items-center flex-wrap gap-2 mb-1">
        <span className="text-xs text-[hsl(var(--muted-foreground))]">#{index + 1}</span>
        {/* History compare status indicator */}
        {statusInfo && (
          <span className={`flex items-center gap-1 px-1.5 py-0.5 text-xs font-medium rounded ${statusInfo.className}`}>
            <statusInfo.icon className="h-3 w-3" />
            {statusInfo.label}
          </span>
        )}
        <span className={`px-1.5 py-0.5 text-xs font-medium rounded ${severityColors[severity] || severityColors.info}`}>
          {severity.toUpperCase()}
        </span>
        {category && (
          <Badge 
            variant="secondary" 
            className="text-xs font-normal bg-[hsl(var(--primary))]/8 text-[hsl(var(--primary))] border border-[hsl(var(--primary))]/15"
          >
            {category}
          </Badge>
        )}
      </div>
      
      {/* Location */}
      {location && (
        <div className="flex items-center gap-1.5 text-xs text-[hsl(var(--muted-foreground))] font-mono mb-1">
          <FileCode className="h-3.5 w-3.5 flex-shrink-0" />
          <span>{location}</span>
        </div>
      )}
      
      {/* Description */}
      <div className="text-sm leading-relaxed mt-2">{description}</div>
      
      {/* Code snippet with syntax highlighting */}
      {codeSnippet && (
        <div className="mt-2">
          <SyntaxHighlighter
            style={isDark ? oneDark : oneLight}
            language="text"
            PreTag="div"
            className="rounded-md text-xs !my-0 border border-[hsl(var(--border))]"
            wrapLines
          >
            {codeSnippet}
          </SyntaxHighlighter>
        </div>
      )}
      
      {/* Suggestion */}
      {suggestion && (
        <div className="mt-2 text-sm text-[hsl(var(--muted-foreground))]">
          <span className="font-medium">{t('reviews.detail.suggestion')}：</span>
          {suggestion}
        </div>
      )}
      
      {/* Render extra_fields dynamically from schema */}
      {itemSchema && (
        <AdditionalFields finding={finding} itemSchema={itemSchema} />
      )}
    </div>
  )
}

/**
 * Check if a string looks like a URL
 */
function isUrl(str: string): boolean {
  return /^https?:\/\//i.test(str)
}

/**
 * Render a field value with smart formatting based on field name and type
 * - cwe_id: renders as CWE link
 * - cve_id: renders as CVE link  
 * - URL strings: renders as clickable links
 * - Arrays of URLs: renders as list of links
 * - Other arrays: renders as comma-separated or list
 */
function SmartFieldValue({ 
  fieldKey, 
  value, 
}: { 
  fieldKey: string
  value: unknown
}) {
  const stringValue = typeof value === 'string' ? value : ''
  
  // Handle CWE ID - link to CWE database
  if (fieldKey === 'cwe_id' && stringValue) {
    const cweNumber = stringValue.replace(/^CWE-/i, '')
    return (
      <a 
        href={`https://cwe.mitre.org/data/definitions/${cweNumber}.html`}
        target="_blank"
        rel="noopener noreferrer"
        className="text-[hsl(var(--primary))] hover:underline"
      >
        {stringValue}
      </a>
    )
  }
  
  // Handle CVE ID - link to CVE database
  if (fieldKey === 'cve_id' && stringValue) {
    return (
      <a 
        href={`https://cve.mitre.org/cgi-bin/cvename.cgi?name=${stringValue}`}
        target="_blank"
        rel="noopener noreferrer"
        className="text-[hsl(var(--primary))] hover:underline"
      >
        {stringValue}
      </a>
    )
  }
  
  // Handle array of strings (potentially URLs)
  if (Array.isArray(value)) {
    const hasUrls = value.some(item => typeof item === 'string' && isUrl(item))
    
    if (hasUrls) {
      return (
        <ul className="text-xs space-y-0.5 pl-4 mt-1">
          {value.map((item, i) => {
            const itemStr = String(item)
            if (isUrl(itemStr)) {
              return (
                <li key={i}>
                  <a 
                    href={itemStr}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-[hsl(var(--primary))] hover:underline break-all"
                  >
                    {itemStr}
                  </a>
                </li>
              )
            }
            return <li key={i}>{itemStr}</li>
          })}
        </ul>
      )
    }
    
    // Non-URL array: render inline
    return <span>{value.join(', ')}</span>
  }
  
  // Handle single URL string
  if (typeof value === 'string' && isUrl(value)) {
    return (
      <a 
        href={value}
        target="_blank"
        rel="noopener noreferrer"
        className="text-[hsl(var(--primary))] hover:underline break-all"
      >
        {value}
      </a>
    )
  }
  
  // Handle object
  if (typeof value === 'object' && value !== null) {
    return <span>{JSON.stringify(value)}</span>
  }
  
  // Default: render as string
  return <span>{String(value)}</span>
}

/**
 * Render additional fields that are defined in schema but not explicitly handled
 * These are extra_fields from DSL configuration
 */
function AdditionalFields({ 
  finding, 
  itemSchema 
}: { 
  finding: Record<string, unknown>
  itemSchema: Record<string, Record<string, unknown>>
}) {
  // Base schema fields that are explicitly handled in FindingItem
  const handledFields = new Set([
    'severity', 'title', 'description', 'category', 'location', 
    'suggestion', 'code_snippet', 'status'
  ])
  
  const additionalFields = Object.entries(itemSchema).filter(
    ([key]) => !handledFields.has(key) && finding[key] !== undefined && finding[key] !== null
  )
  
  if (additionalFields.length === 0) {
    return null
  }
  
  return (
    <div className="mt-2 space-y-1.5">
      {additionalFields.map(([key, propSchema]) => {
        const value = finding[key]
        const label = (propSchema.description as string) || key.replace(/_/g, ' ')
        
        return (
          <div key={key} className="text-xs text-[hsl(var(--muted-foreground))]">
            <span className="font-medium capitalize">{label}:</span>{' '}
            <SmartFieldValue fieldKey={key} value={value} />
          </div>
        )
      })}
    </div>
  )
}

/**
 * Schema-driven renderer
 * Renders data based on schema.properties when available
 * Uses schema description for labels and type for rendering decisions
 */
function SchemaRenderer({ data, schema }: { data: Record<string, unknown>; schema: Record<string, unknown> }) {
  // Get properties from schema
  const properties = schema.properties as Record<string, Record<string, unknown>> | undefined
  
  if (!properties) {
    // Fallback: no properties defined, render as key-value pairs
    return (
      <div className="space-y-2">
        {Object.entries(data).map(([key, value]) => (
          <div key={key} className="text-sm">
            <span className="font-medium text-[hsl(var(--muted-foreground))]">{key}: </span>
            <span>{typeof value === 'object' ? JSON.stringify(value) : String(value)}</span>
          </div>
        ))}
      </div>
    )
  }
  
  // Render based on schema properties order
  // Sort keys to ensure summary appears before findings
  const propertyKeys = Object.keys(properties)
  const orderedKeys = [...propertyKeys].sort((a, b) => {
    if (a === 'summary') return -1
    if (b === 'summary') return 1
    if (a === 'findings') return 1
    if (b === 'findings') return -1
    return 0
  })
  
  return (
    <div className="space-y-4">
      {orderedKeys.map((propKey) => {
        const propSchema = properties[propKey]
        const value = data[propKey]
        
        if (value === undefined || value === null) {
          return null
        }
        
        const propType = propSchema.type as string
        const propDescription = propSchema.description as string
        const itemsSchema = propSchema.items as Record<string, unknown> | undefined
        
        // Handle string type (summary uses MarkdownRenderer)
        if (propType === 'string' && typeof value === 'string') {
          if (propKey === 'summary') {
            return (
              <div key={propKey}>
                <MarkdownRenderer content={value} className="text-sm" />
                <Separator className="my-4" />
              </div>
            )
          }
          // Use description as label if available
          const label = propDescription || propKey
          return (
            <div key={propKey} className="text-sm">
              <span className="font-medium text-[hsl(var(--muted-foreground))]">{label}: </span>
              <span>{value}</span>
            </div>
          )
        }
        
        // Handle array type (findings with severity use FindingItem)
        if (propType === 'array' && Array.isArray(value)) {
          const itemProps = itemsSchema?.properties as Record<string, Record<string, unknown>> | undefined
          const hasSeverity = itemProps?.severity !== undefined
          
          if (hasSeverity && value.length > 0) {
            // Render as findings list without summary title
            return (
              <div key={propKey}>
                <div className="space-y-4">
                  {value.map((item, index) => (
                    <FindingItem 
                      key={index} 
                      finding={item as Record<string, unknown>} 
                      index={index}
                      itemSchema={itemProps}
                    />
                  ))}
                </div>
              </div>
            )
          }
          
          // Generic array rendering with description label
          const label = propDescription || propKey
          return (
            <div key={propKey} className="text-sm">
              <div className="font-medium text-[hsl(var(--muted-foreground))] mb-1">{label}:</div>
              <ul className="list-disc list-inside pl-2 space-y-1">
                {value.map((item, index) => (
                  <li key={index}>{typeof item === 'object' ? JSON.stringify(item) : String(item)}</li>
                ))}
              </ul>
            </div>
          )
        }
        
        // Handle object type
        if (propType === 'object' && typeof value === 'object') {
          const label = propDescription || propKey
          return (
            <div key={propKey} className="text-sm">
              <div className="font-medium text-[hsl(var(--muted-foreground))] mb-1">{label}:</div>
              <pre className="pl-2 text-xs">{JSON.stringify(value, null, 2)}</pre>
            </div>
          )
        }
        
        // Default: render as string with description label
        const label = propDescription || propKey
        return (
          <div key={propKey} className="text-sm">
            <span className="font-medium text-[hsl(var(--muted-foreground))]">{label}: </span>
            <span>{String(value)}</span>
          </div>
        )
      })}
    </div>
  )
}

/**
 * Custom styles for JsonView using fixed class names defined in index.css
 */
const customJsonStyle = {
  container: 'json-view-container',
  basicChildStyle: 'json-view-basic-child',
  childFieldsContainer: 'json-view-child-container',
  label: 'json-view-label',
  clickableLabel: 'json-view-label', // Use same style as label
  stringValue: 'json-view-string',
  numberValue: 'json-view-number',
  booleanValue: 'json-view-boolean',
  nullValue: 'json-view-null',
  undefinedValue: 'json-view-undefined',
  otherValue: 'json-view-other',
  punctuation: 'json-view-punctuation',
  expandIcon: 'json-view-expand-icon',
  collapseIcon: 'json-view-collapse-icon',
  collapsedContent: 'json-view-collapsed-content',
  noQuotesForStringValues: false,
  quotesForFieldNames: false,
  stringifyStringValues: false,
  ariaLables: {
    collapseJson: 'collapse',
    expandJson: 'expand'
  }
}

/**
 * JSON view renderer - simple tree view, always expanded
 */
function JsonDataView({ data }: { data: Record<string, unknown> }) {
  return (
    <div className="rounded-md bg-[hsl(var(--muted))] p-4 text-xs overflow-auto">
      <JsonView
        data={data}
        shouldExpandNode={() => true}
        style={customJsonStyle}
      />
    </div>
  )
}

/**
 * Schema-driven text formatter for copying
 * Converts data to friendly text based on schema properties
 */
function schemaToFriendlyText(
  data: Record<string, unknown>, 
  schema: Record<string, unknown> | undefined
): string {
  const lines: string[] = []
  const properties = schema?.properties as Record<string, Record<string, unknown>> | undefined
  
  if (!properties) {
    // Fallback: no schema, use simple key-value format
    return fallbackToFriendlyText(data)
  }
  
  // Process properties in schema order
  const propertyKeys = Object.keys(properties)
  
  for (const propKey of propertyKeys) {
    const propSchema = properties[propKey]
    const value = data[propKey]
    
    if (value === undefined || value === null) {
      continue
    }
    
    const propType = propSchema.type as string
    const propDescription = propSchema.description as string
    const itemsSchema = propSchema.items as Record<string, unknown> | undefined
    
    // Handle string type
    if (propType === 'string' && typeof value === 'string') {
      if (propKey === 'summary') {
        // Summary: output directly without label
        lines.push(value)
        lines.push('')
      } else {
        // Other strings: use description as label if available
        const label = propDescription || propKey
        lines.push(`**${label}**: ${value}`)
      }
      continue
    }
    
    // Handle array type (findings)
    if (propType === 'array' && Array.isArray(value) && value.length > 0) {
      const itemProps = itemsSchema?.properties as Record<string, Record<string, unknown>> | undefined
      const arrayDescription = propSchema.description as string
      
      lines.push(`## ${arrayDescription || propKey} (${value.length})`)
      lines.push('')
      
      value.forEach((item, index) => {
        if (typeof item === 'object' && item !== null) {
          const itemObj = item as Record<string, unknown>
          lines.push(formatFindingItem(itemObj, index, itemProps))
          lines.push('')
        } else {
          lines.push(`- ${String(item)}`)
        }
      })
      continue
    }
    
    // Handle object type
    if (propType === 'object' && typeof value === 'object') {
      const label = propDescription || propKey
      lines.push(`**${label}**:`)
      lines.push(JSON.stringify(value, null, 2))
      lines.push('')
      continue
    }
    
    // Default: output as string
    const label = propDescription || propKey
    lines.push(`**${label}**: ${String(value)}`)
  }
  
  return lines.join('\n').trim() || JSON.stringify(data, null, 2)
}

/**
 * Format a field value for markdown output with smart handling
 */
function formatFieldForMarkdown(_key: string, value: unknown): string {
  if (value === undefined || value === null) return ''
  
  // Handle arrays
  if (Array.isArray(value)) {
    if (value.length === 0) return ''
    // Check if array contains URLs
    const hasUrls = value.some(item => typeof item === 'string' && isUrl(String(item)))
    if (hasUrls) {
      return value.map(item => `   - ${String(item)}`).join('\n')
    }
    return value.join(', ')
  }
  
  // Handle objects
  if (typeof value === 'object') {
    return JSON.stringify(value)
  }
  
  return String(value)
}

/**
 * Format a single finding item based on its properties
 * Uses schema-driven approach - no hardcoded extra_fields
 */
function formatFindingItem(
  item: Record<string, unknown>, 
  index: number,
  itemProps: Record<string, Record<string, unknown>> | undefined
): string {
  const lines: string[] = []
  
  // Base schema fields that are always handled
  const baseFields = new Set([
    'severity', 'title', 'description', 'category', 'location', 
    'suggestion', 'code_snippet', 'status'
  ])
  
  // Extract base fields
  const severity = (item.severity as string) || 'info'
  const title = item.title as string
  const description = item.description as string
  const location = item.location as string
  const category = item.category as string
  const suggestion = item.suggestion as string
  const codeSnippet = item.code_snippet as string
  
  // History compare status
  const status = (item.status as string)?.toUpperCase()
  
  // Title line with status, severity and title/description
  const titleParts: string[] = [`### ${index + 1}.`]
  if (status) {
    titleParts.push(`[${status}]`)
  }
  titleParts.push(`[${severity.toUpperCase()}]`)
  titleParts.push(title || description || 'No description')
  lines.push(titleParts.join(' '))
  
  // Description (if title exists and description is different)
  if (title && description && title !== description) {
    lines.push(description)
  }
  
  // Location
  if (location) {
    lines.push(`📁 ${location}`)
  }
  
  // Category
  if (category) {
    lines.push(`🏷️ Category: ${category}`)
  }
  
  // Code snippet
  if (codeSnippet) {
    lines.push('')
    lines.push('```')
    lines.push(codeSnippet)
    lines.push('```')
  }
  
  // Suggestion
  if (suggestion) {
    lines.push(`💡 ${suggestion}`)
  }
  
  // Handle all extra_fields from schema dynamically
  if (itemProps) {
    for (const [key, propSchema] of Object.entries(itemProps)) {
      if (baseFields.has(key)) continue
      const value = item[key]
      if (value === undefined || value === null) continue
      
      const label = (propSchema.description as string) || key.replace(/_/g, ' ')
      const formattedValue = formatFieldForMarkdown(key, value)
      
      if (!formattedValue) continue
      
      // Check if value is multi-line (array formatted as list)
      if (formattedValue.includes('\n')) {
        lines.push(`📌 ${label}:`)
        lines.push(formattedValue)
      } else {
        lines.push(`📌 ${label}: ${formattedValue}`)
      }
    }
  }
  
  return lines.join('\n')
}

/**
 * Fallback text formatter when no schema is available
 */
function fallbackToFriendlyText(data: Record<string, unknown>): string {
  const lines: string[] = []
  
  const summary = data.summary as string
  if (summary) {
    lines.push(summary)
    lines.push('')
  }

  const findings = data.findings as Record<string, unknown>[]
  if (findings && findings.length > 0) {
    lines.push(`## Findings (${findings.length})`)
    lines.push('')
    findings.forEach((finding, index) => {
      lines.push(formatFindingItem(finding, index, undefined))
      lines.push('')
    })
  }
  
  // Handle other top-level fields
  for (const [key, value] of Object.entries(data)) {
    if (key === 'summary' || key === 'findings') continue
    if (value === undefined || value === null) continue
    
    if (typeof value === 'object') {
      lines.push(`**${key}**: ${JSON.stringify(value, null, 2)}`)
    } else {
      lines.push(`**${key}**: ${String(value)}`)
    }
  }

  return lines.join('\n').trim() || JSON.stringify(data, null, 2)
}

/**
 * Review result content component with view toggle
 */
function ReviewResultContent({ 
  rule, 
  copiedId, 
  onCopy 
}: { 
  rule: ReviewRule
  copiedId: string | null
  onCopy: (text: string, id: string) => void
}) {
  const { t } = useTranslation()
  const [showJson, setShowJson] = useState(false)
  
  const resultData = getResultData(rule)

  // Fetch default schema for text formatting
  // Schema is now embedded in backend, always use 'default'
  const { data: schema } = useQuery({
    queryKey: ['schema', 'default'],
    queryFn: () => api.schemas.get('default'),
    staleTime: Infinity, // Schema doesn't change often
  })

  const handleCopyText = useCallback(() => {
    if (resultData) {
      const text = schemaToFriendlyText(resultData, schema)
      onCopy(text, `${rule.id}-result-text`)
    }
  }, [resultData, schema, rule.id, onCopy])

  const handleCopyJson = useCallback(() => {
    if (resultData) {
      onCopy(JSON.stringify(resultData, null, 2), `${rule.id}-result-json`)
    }
  }, [resultData, rule.id, onCopy])

  if (!resultData) {
    return (
      <div className="text-sm text-[hsl(var(--muted-foreground))] italic">
        {t('reviews.detail.noResultData')}
      </div>
    )
  }

  const isCopied = copiedId === `${rule.id}-result-text` || copiedId === `${rule.id}-result-json`
  const hasSchema = !!schema

  // No schema: show JSON only (no toggle)
  if (!hasSchema) {
    return (
      <div>
        {/* Header with copy dropdown only */}
        <div className="flex items-center justify-end mt-1 mb-3">
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="sm" className="h-7 px-2">
                {isCopied ? (
                  <Check className="h-3.5 w-3.5 text-green-500 mr-1" />
                ) : (
                  <Copy className="h-3.5 w-3.5 mr-1" />
                )}
                <span className="text-xs">{isCopied ? t('reviews.detail.copied') : t('reviews.detail.copyAs')}</span>
                <ChevronDown className="h-3 w-3 ml-1" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={handleCopyText}>
                {t('reviews.detail.copyAsText')}
              </DropdownMenuItem>
              <DropdownMenuItem onClick={handleCopyJson}>
                {t('reviews.detail.copyAsJson')}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>

        {/* JSON view */}
        <JsonDataView data={resultData} />
      </div>
    )
  }

  // Has schema: show friendly view by default with Switch to toggle JSON
  return (
    <div>
      {/* Header with Switch toggle and copy dropdown */}
      <div className="flex items-center justify-end gap-4 mt-1 mb-3">
        {/* JSON Switch toggle */}
        <div className="flex items-center gap-2">
          <Switch 
            id={`json-view-${rule.id}`}
            checked={showJson} 
            onCheckedChange={setShowJson} 
          />
          <Label htmlFor={`json-view-${rule.id}`} className="text-xs cursor-pointer">
            JSON
          </Label>
        </div>

        {/* Copy dropdown */}
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="sm" className="h-7 px-2">
              {isCopied ? (
                <Check className="h-3.5 w-3.5 text-green-500 mr-1" />
              ) : (
                <Copy className="h-3.5 w-3.5 mr-1" />
              )}
              <span className="text-xs">{isCopied ? t('reviews.detail.copied') : t('reviews.detail.copyAs')}</span>
              <ChevronDown className="h-3 w-3 ml-1" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onClick={handleCopyText}>
              {t('reviews.detail.copyAsText')}
            </DropdownMenuItem>
            <DropdownMenuItem onClick={handleCopyJson}>
              {t('reviews.detail.copyAsJson')}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>

      {/* Content based on Switch state */}
      {showJson ? (
        <JsonDataView data={resultData} />
      ) : (
        <div className="rounded-md bg-[hsl(var(--muted))] p-4">
          <SchemaRenderer data={resultData} schema={schema} />
        </div>
      )}
    </div>
  )
}

/**
 * Render prompt content (still using the old approach for prompt)
 */
function PromptContent({ content }: { content: string }) {
  return (
    <MarkdownRenderer
      content={content}
      className="rounded-md bg-[hsl(var(--muted))] p-4 text-sm"
    />
  )
}

/**
 * Review detail page component
 */
export default function ReviewDetail() {
  const { t } = useTranslation()
  const { id } = useParams<{ id: string }>()
  const queryClient = useQueryClient()
  const navigate = useNavigate()

  // Fetch review details
  const { data: review, isLoading } = useQuery<Review>({
    queryKey: ['review', id],
    queryFn: () => api.reviews.get(id!) as Promise<Review>,
    enabled: !!id,
    refetchInterval: (query) => {
      const data = query.state.data as Review | undefined
      // Stop polling when review is done
      if (data?.status === 'completed' || data?.status === 'failed' || data?.status === 'cancelled') {
        return false
      }
      return 5000
    },
  })

  // Fetch reviews list for navigation (fetch a large page to get more reviews)
  const { data: reviewsData } = useQuery({
    queryKey: ['reviews', 1, 1000, 'all'],
    queryFn: () =>
      api.reviews.list({
        page: 1,
        page_size: 1000,
        status: undefined,
      }),
  })

  // Calculate previous and next review IDs based on created_at
  const { previousReviewId, nextReviewId } = useMemo(() => {
    if (!review || !reviewsData?.data) {
      return { previousReviewId: null, nextReviewId: null }
    }

    const reviews = reviewsData.data as Review[]
    const currentIndex = reviews.findIndex((r) => r.id === review.id)

    if (currentIndex === -1) {
      return { previousReviewId: null, nextReviewId: null }
    }

    // Reviews are sorted by created_at DESC (newest first)
    // Previous = newer (index - 1)
    // Next = older (index + 1)
    const previousReviewId = currentIndex > 0 ? reviews[currentIndex - 1].id : null
    const nextReviewId = currentIndex < reviews.length - 1 ? reviews[currentIndex + 1].id : null

    return { previousReviewId, nextReviewId }
  }, [review, reviewsData])

  // Copy state
  const [copiedId, setCopiedId] = useState<string | null>(null)

  // Accordion state for prompt/result sections (persisted per rule_id)
  const { getState: getAccordionState, setState: setAccordionState } = useAccordionState()

  // Copy handler
  const handleCopy = async (text: string, id: string) => {
    try {
      await navigator.clipboard.writeText(text)
      setCopiedId(id)
      setTimeout(() => setCopiedId(null), 2000)
    } catch (err) {
      console.error('Failed to copy:', err)
    }
  }

  // Retry review mutation
  const retryMutation = useMutation({
    mutationFn: () => api.reviews.retry(id!),
    onSuccess: () => {
      toast({
        title: t('reviews.retry.success'),
        description: t('reviews.retry.successDescription'),
      })
      // Invalidate queries to refresh data
      queryClient.invalidateQueries({ queryKey: ['review', id] })
      queryClient.invalidateQueries({ queryKey: ['reviews'] })
    },
    onError: () => {
      // Error is already handled by global toast in api.ts
    },
  })

  // Retry rule mutation - tracks which rule is being retried
  const [retryingRuleId, setRetryingRuleId] = useState<string | null>(null)
  const retryRuleMutation = useMutation({
    mutationFn: (ruleId: string) => api.reviews.retryRule(id!, ruleId),
    onSuccess: () => {
      toast({
        title: t('reviews.retryRule.success'),
        description: t('reviews.retryRule.successDescription'),
      })
      // Invalidate queries to refresh data
      queryClient.invalidateQueries({ queryKey: ['review', id] })
      queryClient.invalidateQueries({ queryKey: ['reviews'] })
      setRetryingRuleId(null)
    },
    onError: () => {
      // Error is already handled by global toast in api.ts
      setRetryingRuleId(null)
    },
  })

  // Handle rule retry
  const handleRetryRule = (ruleId: string) => {
    setRetryingRuleId(ruleId)
    retryRuleMutation.mutate(ruleId)
  }

  if (isLoading) {
    return (
      <div className="space-y-8">
        <Skeleton className="h-10 w-48" />
        <Skeleton className="h-48 w-full" />
        <Skeleton className="h-64 w-full" />
      </div>
    )
  }

  if (!review) {
    return (
      <div className="flex flex-col items-center justify-center py-12">
        <AlertCircle className="mb-4 h-12 w-12 text-[hsl(var(--muted-foreground))]" />
        <p className="text-lg">{t('errors.notFound')}</p>
        <Button variant="link" asChild className="mt-4">
          <Link to="/admin/reviews">{t('common.back')}</Link>
        </Button>
      </div>
    )
  }

  return (
    <div className="space-y-8">
      {/* Back button and title */}
      <div className="flex items-center gap-5">
        <Button variant="ghost" size="icon" asChild>
          <Link to="/admin/reviews">
            <ArrowLeft className="h-4 w-4" />
          </Link>
        </Button>
        <div className="flex-1">
          <h2 className="text-2xl font-bold text-[hsl(var(--foreground))]">
            {t('reviews.detail.title')} <span className="font-mono">#{review.id}</span>
          </h2>
          <p className="text-sm text-[hsl(var(--muted-foreground))] break-all">
            {formatRepoUrl(review.repo_url)}
          </p>
        </div>
        <div className="ml-auto flex items-center gap-3">
          {/* Navigation buttons */}
          <div className="flex items-center gap-1 border rounded-lg">
            <Button
              variant="ghost"
              size="sm"
              onClick={() => previousReviewId && navigate(`/admin/reviews/${previousReviewId}`)}
              disabled={!previousReviewId}
              className="rounded-r-none border-r"
              title={t('reviews.detail.previousReview')}
            >
              <ChevronLeft className="h-4 w-4 mr-1" />
              {t('reviews.detail.previousReview')}
            </Button>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => nextReviewId && navigate(`/admin/reviews/${nextReviewId}`)}
              disabled={!nextReviewId}
              className="rounded-l-none"
              title={t('reviews.detail.nextReview')}
            >
              {t('reviews.detail.nextReview')}
              <ChevronRight className="h-4 w-4 ml-1" />
            </Button>
          </div>
          
          {/* Status and retry group */}
          <div className="flex items-center border rounded-lg">
            <div className="px-3 py-1">
              <StatusBadge status={review.status} />
            </div>
            {review.retry_count > 0 && (
              <>
                <div className="h-5 w-px bg-[hsl(var(--border))]" />
                <span className="px-3 py-1.5 text-sm text-[hsl(var(--muted-foreground))]">
                  {t('reviews.retry.count', { count: review.retry_count })}
                </span>
              </>
            )}
            {review.status === 'failed' && (
              <>
                <div className="h-5 w-px bg-[hsl(var(--border))]" />
                <Button
                  size="sm"
                  variant="ghost"
                  onClick={() => retryMutation.mutate()}
                  disabled={retryMutation.isPending}
                  className="rounded-l-none h-full px-3 hover:bg-transparent hover:text-[hsl(var(--primary))]"
                >
                  <Repeat className={`mr-1.5 h-3.5 w-3.5 text-amber-500 ${retryMutation.isPending ? 'animate-spin' : ''}`} />
                  {retryMutation.isPending ? t('reviews.retry.retrying') : t('reviews.retry.button')}
                </Button>
              </>
            )}
          </div>

          {/* View logs button */}
          <LogViewer taskType="review" taskId={review.id} />
        </div>
      </div>

      {/* Basic info card */}
      <Card>
        <CardHeader>
          <CardTitle>{t('reviews.detail.basicInfo')}</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
            {/* Repository */}
            <div className="flex items-start gap-3">
              <GitBranch className="mt-0.5 h-4 w-4 text-[hsl(var(--muted-foreground))]" />
              <div>
                <div className="text-sm text-[hsl(var(--muted-foreground))]">
                  {t('reviews.detail.repository')}
                </div>
                <a
                  href={review.repo_url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="flex items-center gap-1 text-[hsl(var(--primary))] hover:underline break-all"
                >
                  {formatRepoUrl(review.repo_url)}
                  <ExternalLink className="h-3 w-3 flex-shrink-0" />
                </a>
              </div>
            </div>

            {/* Ref */}
            <div className="flex items-start gap-3">
              <GitBranch className="mt-0.5 h-4 w-4 text-[hsl(var(--muted-foreground))]" />
              <div>
                <div className="text-sm text-[hsl(var(--muted-foreground))]">
                  {t('reviews.detail.ref')}
                </div>
                <code className="rounded-md bg-[hsl(var(--muted))] px-2 py-1 text-sm">
                  {review.ref}
                </code>
              </div>
            </div>

            {/* PR URL */}
            {review.pr_url && (
              <div className="flex items-start gap-3">
                <ExternalLink className="mt-0.5 h-4 w-4 text-[hsl(var(--muted-foreground))]" />
                <div>
                  <div className="text-sm text-[hsl(var(--muted-foreground))]">
                    {t('reviews.detail.prUrl')}
                  </div>
                  <a
                    href={review.pr_url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="flex items-center gap-1 text-[hsl(var(--primary))] hover:underline"
                  >
                    PR #{review.pr_number}
                    <ExternalLink className="h-3 w-3" />
                  </a>
                </div>
              </div>
            )}

            {/* Source */}
            <div className="flex items-start gap-3">
              <Clock className="mt-0.5 h-4 w-4 text-[hsl(var(--muted-foreground))]" />
              <div>
                <div className="text-sm text-[hsl(var(--muted-foreground))]">
                  {t('reviews.detail.source')}
                </div>
                <Badge variant="outline">{review.source}</Badge>
              </div>
            </div>

            {/* Started at */}
            <div className="flex items-start gap-3">
              <Calendar className="mt-0.5 h-4 w-4 text-[hsl(var(--muted-foreground))]" />
              <div>
                <div className="text-sm text-[hsl(var(--muted-foreground))]">
                  {t('reviews.detail.startedAt')}
                </div>
                <div>{review.started_at ? formatDate(review.started_at) : '-'}</div>
              </div>
            </div>

            {/* Duration */}
            <div className="flex items-start gap-3">
              <Clock className="mt-0.5 h-4 w-4 text-[hsl(var(--muted-foreground))]" />
              <div>
                <div className="text-sm text-[hsl(var(--muted-foreground))]">
                  {t('common.duration')}
                </div>
                <div>{review.duration ? formatDuration(review.duration) : '-'}</div>
              </div>
            </div>

            {/* Author */}
            {review.author && (
              <div className="flex items-start gap-3">
                <User className="mt-0.5 h-4 w-4 text-[hsl(var(--muted-foreground))]" />
                <div>
                  <div className="text-sm text-[hsl(var(--muted-foreground))]">
                    {t('reviews.detail.author')}
                  </div>
                  <div className="font-medium">{review.author}</div>
                </div>
              </div>
            )}

            {/* Branch Created At */}
            {review.branch_created_at && (
              <div className="flex items-start gap-3">
                <GitBranch className="mt-0.5 h-4 w-4 text-[hsl(var(--muted-foreground))]" />
                <div>
                  <div className="text-sm text-[hsl(var(--muted-foreground))]">
                    {t('reviews.detail.branchCreatedAt')}
                  </div>
                  <div>{formatDate(review.branch_created_at)}</div>
                </div>
              </div>
            )}
          </div>

          {/* Diff Statistics */}
          {((review.lines_added ?? 0) > 0 || (review.lines_deleted ?? 0) > 0 || (review.files_changed ?? 0) > 0) && (
            <>
              <Separator className="my-6" />
              <div>
                <div className="mb-3 text-sm font-semibold text-[hsl(var(--muted-foreground))]">
                  {t('reviews.detail.diffStats')}
                </div>
                <div className="flex flex-wrap gap-6">
                  <div className="flex items-center gap-2">
                    <FileCode className="h-4 w-4 text-[hsl(var(--muted-foreground))]" />
                    <span className="text-sm text-[hsl(var(--muted-foreground))]">
                      {t('reviews.detail.filesChanged')}:
                    </span>
                    <span className="font-semibold">{review.files_changed}</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <Plus className="h-4 w-4 text-green-500" />
                    <span className="text-sm text-[hsl(var(--muted-foreground))]">
                      {t('reviews.detail.linesAdded')}:
                    </span>
                    <span className="font-semibold text-green-500">+{review.lines_added}</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <Minus className="h-4 w-4 text-red-500" />
                    <span className="text-sm text-[hsl(var(--muted-foreground))]">
                      {t('reviews.detail.linesDeleted')}:
                    </span>
                    <span className="font-semibold text-red-500">-{review.lines_deleted}</span>
                  </div>
                </div>
              </div>
            </>
          )}

          {/* Error message */}
          {review.error_message && (
            <>
              <Separator className="my-6" />
              <div className="rounded-md bg-[hsl(var(--destructive))]/10 p-4">
                <div className="mb-2 text-sm font-semibold text-[hsl(var(--destructive))]">
                  {t('reviews.detail.errorMessage')}
                </div>
                <pre className="whitespace-pre-wrap text-sm">{review.error_message}</pre>
              </div>
            </>
          )}
        </CardContent>
      </Card>


      {/* Rules */}
      {review.rules && review.rules.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>{t('reviews.detail.rules')}</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-6">
              {review.rules.map((rule, index) => (
                <div key={rule.id} className={index > 0 ? 'pt-6 border-t border-[hsl(var(--border))]' : ''}>
                  {/* Rule Header */}
                  <div className="flex flex-1 items-center justify-between mb-4">
                    <div className="flex items-center gap-4">
                      <span className="text-[hsl(var(--muted-foreground))]">
                        #{index + 1}
                      </span>
                      <span className="font-semibold">{rule.rule_id}</span>
                    </div>
                    <div className="flex items-center gap-5">
                      {rule.duration && (
                        <span className="text-sm text-[hsl(var(--muted-foreground))]">
                          {formatDuration(rule.duration)}
                        </span>
                      )}
                      {rule.multi_run_enabled && rule.multi_run_runs > 1 && rule.status === 'running' && (
                        <span className="text-sm text-[hsl(var(--muted-foreground))]">
                          {t('reviews.detail.multiRunProgress', { 
                            current: rule.current_run_index + 1, 
                            total: rule.multi_run_runs 
                          })}
                        </span>
                      )}
                      {/* Status + retry count + retry button 组合容器 */}
                      <div className="flex items-center border rounded-lg">
                        <div className="px-3 py-1">
                          <StatusBadge status={rule.status} showPulse={false} />
                        </div>
                        {rule.retry_count > 0 && (
                          <>
                            <div className="h-5 w-px bg-[hsl(var(--border))]" />
                            <span className="px-3 py-1.5 text-sm text-[hsl(var(--muted-foreground))]">
                              {t('reviews.retryRule.count', { count: rule.retry_count })}
                            </span>
                          </>
                        )}
                        {rule.status === 'failed' && (
                          <>
                            <div className="h-5 w-px bg-[hsl(var(--border))]" />
                            <Button
                              size="sm"
                              variant="ghost"
                              onClick={() => handleRetryRule(rule.rule_id)}
                              disabled={retryingRuleId === rule.rule_id}
                              className="rounded-l-none h-full px-3 hover:bg-transparent hover:text-[hsl(var(--primary))]"
                            >
                              <RotateCw className={`mr-1.5 h-3.5 w-3.5 text-amber-500 ${retryingRuleId === rule.rule_id ? 'animate-spin' : ''}`} />
                              {retryingRuleId === rule.rule_id ? t('reviews.retryRule.retrying') : t('reviews.retryRule.button')}
                            </Button>
                          </>
                        )}
                      </div>
                    </div>
                  </div>
                  
                  {/* Rule Content */}
                  <div className="space-y-5">
                      {/* Prompt Section - default collapsed, state persisted per rule_id */}
                      {rule.prompt && (
                        <div>
                          <Accordion 
                            type="single" 
                            collapsible 
                            value={getAccordionState(rule.rule_id, 'prompt') ? 'prompt' : ''}
                            onValueChange={(v) => setAccordionState(rule.rule_id, 'prompt', v === 'prompt')}
                          >
                            <AccordionItem value="prompt" className="border-none">
                              <AccordionTrigger className="text-sm font-semibold py-2 hover:no-underline">
                                <div className="flex items-center gap-2">
                                  <span>Prompt</span>
                                  <span
                                    role="button"
                                    tabIndex={0}
                                    onClick={(e) => {
                                      e.stopPropagation()
                                      handleCopy(rule.prompt!, `${rule.id}-prompt`)
                                    }}
                                    onKeyDown={(e) => {
                                      if (e.key === 'Enter' || e.key === ' ') {
                                        e.preventDefault()
                                        e.stopPropagation()
                                        handleCopy(rule.prompt!, `${rule.id}-prompt`)
                                      }
                                    }}
                                    className="p-1 rounded hover:bg-[hsl(var(--muted))] transition-colors cursor-pointer"
                                    title={t('common.copy')}
                                  >
                                    {copiedId === `${rule.id}-prompt` ? (
                                      <Check className="h-3 w-3 text-green-500" />
                                    ) : (
                                      <Copy className="h-3 w-3 text-[hsl(var(--muted-foreground))] opacity-50 hover:opacity-100" />
                                    )}
                                  </span>
                                </div>
                              </AccordionTrigger>
                              <AccordionContent>
                                <PromptContent content={rule.prompt} />
                              </AccordionContent>
                            </AccordionItem>
                          </Accordion>
                        </div>
                      )}

                      {/* Review Content - default expanded, state persisted per rule_id */}
                      {/* Now reads from results[].data instead of summary */}
                      {(rule.results && rule.results.length > 0) && (
                        <div>
                          <Accordion 
                            type="single" 
                            collapsible 
                            value={getAccordionState(rule.rule_id, 'result') ? 'result' : ''}
                            onValueChange={(v) => setAccordionState(rule.rule_id, 'result', v === 'result')}
                          >
                            <AccordionItem value="result" className="border-none">
                              <AccordionTrigger className="text-sm font-semibold py-2 hover:no-underline">
                                <span>{t('reviews.detail.reviewContent')}</span>
                              </AccordionTrigger>
                              <AccordionContent>
                                <ReviewResultContent 
                                  rule={rule}
                                  copiedId={copiedId}
                                  onCopy={handleCopy}
                                />
                              </AccordionContent>
                            </AccordionItem>
                          </Accordion>
                        </div>
                      )}

                      {/* Error */}
                      {rule.error_message && (
                        <div className="rounded-md bg-[hsl(var(--destructive))]/10 p-4">
                          <div className="mb-2 text-sm font-semibold text-[hsl(var(--destructive))]">
                            Error
                          </div>
                          <pre className="whitespace-pre-wrap text-sm">
                            {rule.error_message}
                          </pre>
                        </div>
                      )}
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
