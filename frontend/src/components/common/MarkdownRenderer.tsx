import { useEffect, useRef, useState, useMemo, memo } from 'react'
import ReactMarkdown, { type Components } from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter'
import { oneDark, oneLight } from 'react-syntax-highlighter/dist/esm/styles/prism'
import mermaid from 'mermaid'
import { cn } from '@/lib/utils'

/**
 * Custom hook for IntersectionObserver-based lazy loading
 * Returns [ref, isVisible] - element becomes visible when it enters viewport
 * Once visible, stays visible (no re-hiding on scroll out)
 */
function useIntersectionObserver<T extends HTMLElement>(
  options: IntersectionObserverInit = {}
): [React.RefObject<T | null>, boolean] {
  const ref = useRef<T>(null)
  const [isVisible, setIsVisible] = useState(false)

  useEffect(() => {
    const element = ref.current
    if (!element || isVisible) return // Skip if already visible

    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          setIsVisible(true)
          // Once visible, disconnect observer - no need to watch anymore
          observer.disconnect()
        }
      },
      {
        rootMargin: '100px', // Load slightly before entering viewport
        threshold: 0,
        ...options,
      }
    )

    observer.observe(element)
    return () => observer.disconnect()
  }, [isVisible, options])

  return [ref, isVisible]
}

// Initialize mermaid with default config
// useMaxWidth: true ensures diagrams scale properly - wide diagrams fill container, narrow ones keep original ratio
const initializeMermaidTheme = () => {
  const isDark = typeof document !== 'undefined' && document.documentElement.classList.contains('dark')
  mermaid.initialize({
    startOnLoad: false,
    theme: isDark ? 'dark' : 'default',
    securityLevel: 'loose',
    fontFamily: 'inherit',
    flowchart: { useMaxWidth: true },
    sequence: { useMaxWidth: true },
    gantt: { useMaxWidth: true },
    journey: { useMaxWidth: true },
    class: { useMaxWidth: true },
    state: { useMaxWidth: true },
    er: { useMaxWidth: true },
    pie: { useMaxWidth: true },
  })
}

// Initial setup
initializeMermaidTheme()

// Watch for theme changes globally (only once)
if (typeof document !== 'undefined') {
  const observer = new MutationObserver((mutations) => {
    mutations.forEach((mutation) => {
      if (mutation.attributeName === 'class') {
        initializeMermaidTheme()
      }
    })
  })
  observer.observe(document.documentElement, {
    attributes: true,
    attributeFilter: ['class'],
  })
}

interface MarkdownRendererProps {
  content: string
  className?: string
}

/**
 * Preprocess content to handle nested code blocks
 * Sometimes AI generates content wrapped in outer code fences, or mermaid blocks are nested
 */
function preprocessContent(content: string): string {
  let processed = content.trim()

  // Remove outer markdown code fence wrapper if present
  // Pattern: content starts with ``` (possibly with language like markdown/md, or plain) and ends with ```
  // This handles cases where AI wraps the entire response in code fences
  const outerFencePatterns = [
    /^```(?:markdown|md)\s*\n([\s\S]*?)\n```\s*$/,  // ```markdown or ```md wrapper
    /^```\s*\n([\s\S]*?)\n```\s*$/,                   // plain ``` wrapper
  ]

  for (const pattern of outerFencePatterns) {
    const match = processed.match(pattern)
    if (match) {
      processed = match[1].trim()
      break
    }
  }

  return processed
}

// Counter for generating unique mermaid IDs
let mermaidIdCounter = 0

/**
 * Code block placeholder shown before lazy loading
 * Provides visual feedback that content will load
 */
function CodeBlockPlaceholder({ lineCount }: { lineCount: number }) {
  // Show approximate height based on line count
  const height = Math.max(60, Math.min(lineCount * 20, 200))
  return (
    <div
      className="rounded-md bg-[hsl(var(--muted))] my-4 animate-pulse flex items-center justify-center"
      style={{ height: `${height}px` }}
    >
      <span className="text-xs text-[hsl(var(--muted-foreground))]">Loading code...</span>
    </div>
  )
}

/**
 * Lazy-loaded syntax highlighter component
 * Only renders the actual highlighter when entering viewport
 */
interface LazyCodeBlockProps {
  code: string
  language: string
  isDark: boolean
}

function LazyCodeBlock({ code, language, isDark }: LazyCodeBlockProps) {
  const [ref, isVisible] = useIntersectionObserver<HTMLDivElement>()
  const lineCount = code.split('\n').length

  return (
    <div ref={ref}>
      {isVisible ? (
        <SyntaxHighlighter
          style={isDark ? oneDark : oneLight}
          language={language || 'text'}
          PreTag="div"
          className="rounded-md text-sm !my-4 border border-[hsl(var(--border))]"
          showLineNumbers={lineCount > 3}
          wrapLines
        >
          {code}
        </SyntaxHighlighter>
      ) : (
        <CodeBlockPlaceholder lineCount={lineCount} />
      )}
    </div>
  )
}

/**
 * Mermaid diagram placeholder
 */
function MermaidPlaceholder() {
  return (
    <div className="rounded-md bg-[hsl(var(--muted))] my-4 h-32 animate-pulse flex items-center justify-center">
      <span className="text-xs text-[hsl(var(--muted-foreground))]">Loading diagram...</span>
    </div>
  )
}

/**
 * SVG Icons for Mermaid diagram controls (inline SVG for consistency with exported HTML)
 * These are Lucide icons as inline SVG strings
 */
const MermaidIcons = {
  zoomIn: `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><line x1="21" x2="16.65" y1="21" y2="16.65"/><line x1="11" x2="11" y1="8" y2="14"/><line x1="8" x2="14" y1="11" y2="11"/></svg>`,
  zoomOut: `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><line x1="21" x2="16.65" y1="21" y2="16.65"/><line x1="8" x2="14" y1="11" y2="11"/></svg>`,
  reset: `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/><path d="M3 3v5h5"/></svg>`,
  maximize: `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="15 3 21 3 21 9"/><polyline points="9 21 3 21 3 15"/><line x1="21" x2="14" y1="3" y2="10"/><line x1="3" x2="10" y1="21" y2="14"/></svg>`,
  close: `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>`,
}

/**
 * Icon button component for Mermaid controls
 */
function MermaidIconButton({ 
  icon, 
  onClick, 
  title,
  className = ''
}: { 
  icon: string
  onClick: () => void
  title: string
  className?: string
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      title={title}
      className={cn(
        'p-1.5 rounded-md transition-colors hover:bg-[hsl(var(--accent))] text-[hsl(var(--muted-foreground))] hover:text-[hsl(var(--foreground))]',
        className
      )}
      dangerouslySetInnerHTML={{ __html: icon }}
    />
  )
}

// Zoom limits for Mermaid diagrams
const MERMAID_ZOOM_MIN = 0.5  // 50%
const MERMAID_ZOOM_MAX = 2.5  // 250%
const MERMAID_ZOOM_STEP = 0.1 // 10% per step

/**
 * Mermaid preview modal component with wheel zoom support
 * Uses original SVG from Mermaid (which respects useMaxWidth setting)
 */
function MermaidPreviewModal({ 
  svg, 
  onClose 
}: { 
  svg: string
  onClose: () => void 
}) {
  const bodyRef = useRef<HTMLDivElement>(null)
  const [scale, setScale] = useState(1)
  
  // Zoom controls with limits
  const zoomIn = () => setScale(s => Math.min(s + MERMAID_ZOOM_STEP, MERMAID_ZOOM_MAX))
  const zoomOut = () => setScale(s => Math.max(s - MERMAID_ZOOM_STEP, MERMAID_ZOOM_MIN))
  const resetZoom = () => setScale(1)

  // Close on escape key
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose()
      }
    }
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [onClose])

  // Prevent body scroll when modal is open
  useEffect(() => {
    document.body.style.overflow = 'hidden'
    return () => {
      document.body.style.overflow = ''
    }
  }, [])

  // Handle wheel zoom in modal body
  useEffect(() => {
    const bodyEl = bodyRef.current
    if (!bodyEl) return

    const handleWheel = (e: WheelEvent) => {
      // Check if Ctrl/Cmd is pressed or if it's a pinch gesture (ctrlKey is true for pinch on Mac)
      if (e.ctrlKey || e.metaKey) {
        e.preventDefault()
        // deltaY < 0 means scroll up (zoom in), > 0 means scroll down (zoom out)
        const delta = e.deltaY < 0 ? MERMAID_ZOOM_STEP : -MERMAID_ZOOM_STEP
        setScale(s => Math.max(MERMAID_ZOOM_MIN, Math.min(MERMAID_ZOOM_MAX, s + delta)))
      }
    }

    bodyEl.addEventListener('wheel', handleWheel, { passive: false })
    return () => bodyEl.removeEventListener('wheel', handleWheel)
  }, [])

  return (
    <div 
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/80"
      onClick={onClose}
    >
      {/* Modal content */}
      <div 
        className="relative w-[90vw] h-[90vh] bg-[hsl(var(--background))] rounded-lg overflow-hidden flex flex-col"
        onClick={e => e.stopPropagation()}
      >
        {/* Header with controls */}
        <div className="flex items-center justify-between px-4 py-3 border-b border-[hsl(var(--border))]">
          <div className="flex items-center gap-1">
            <MermaidIconButton icon={MermaidIcons.zoomOut} onClick={zoomOut} title="Zoom Out" />
            <span className="text-sm text-[hsl(var(--muted-foreground))] min-w-[4rem] text-center">
              {Math.round(scale * 100)}%
            </span>
            <MermaidIconButton icon={MermaidIcons.zoomIn} onClick={zoomIn} title="Zoom In" />
            <MermaidIconButton icon={MermaidIcons.reset} onClick={resetZoom} title="Reset Zoom" />
          </div>
          <span className="text-xs text-[hsl(var(--muted-foreground))]">
            Ctrl/Cmd + Scroll to zoom
          </span>
          <button
            type="button"
            onClick={onClose}
            className="p-2 rounded-md hover:bg-[hsl(var(--accent))] text-[hsl(var(--muted-foreground))] hover:text-[hsl(var(--foreground))] transition-colors"
            title="Close"
            dangerouslySetInnerHTML={{ __html: MermaidIcons.close }}
          />
        </div>
        {/* Diagram container with wheel zoom - SVG uses Mermaid's useMaxWidth for proper sizing */}
        <div 
          ref={bodyRef}
          className="flex-1 overflow-auto p-4 flex"
        >
          <div 
            className="transition-transform duration-200 [&>svg]:max-w-full flex justify-center m-auto"
            style={{ transform: `scale(${scale})`, transformOrigin: 'center top' }}
            dangerouslySetInnerHTML={{ __html: svg }}
          />
        </div>
      </div>
    </div>
  )
}


/**
 * Mermaid diagram component with lazy loading, zoom controls and preview
 * Renders mermaid diagrams from code blocks only when in viewport
 * Default display is width-adaptive for better browsing experience
 */
function MermaidDiagram({ code }: { code: string }) {
  const [ref, isInViewport] = useIntersectionObserver<HTMLDivElement>()
  const [svg, setSvg] = useState<string>('')
  const [error, setError] = useState<string | null>(null)
  const [scale, setScale] = useState(1)
  const [showPreview, setShowPreview] = useState(false)
  // Use a ref to store the unique ID so it doesn't change on re-renders
  const idRef = useRef<string>('')
  // Track if we've already rendered once
  const hasRendered = useRef(false)

  // Zoom controls with limits
  const zoomIn = () => setScale(s => Math.min(s + MERMAID_ZOOM_STEP, MERMAID_ZOOM_MAX))
  const zoomOut = () => setScale(s => Math.max(s - MERMAID_ZOOM_STEP, MERMAID_ZOOM_MIN))
  const resetZoom = () => setScale(1)

  useEffect(() => {
    // Only render when visible and hasn't rendered yet
    if (!isInViewport || hasRendered.current) return

    // Generate a unique ID only once when the component mounts
    if (!idRef.current) {
      idRef.current = `mermaid-${Date.now()}-${++mermaidIdCounter}`
    }

    const renderDiagram = async () => {
      if (!code.trim()) return

      try {
        // Generate a new unique ID for each render to avoid Mermaid's ID caching issue
        const renderId = `${idRef.current}-${Date.now()}`
        // Mermaid with useMaxWidth: true generates SVG with proper sizing automatically
        const { svg: rawSvg } = await mermaid.render(renderId, code.trim())
        setSvg(rawSvg)
        setError(null)
        hasRendered.current = true
      } catch (err) {
        console.error('Mermaid render error:', err)
        setError(err instanceof Error ? err.message : 'Failed to render diagram')
        hasRendered.current = true
      }
    }

    renderDiagram()
  }, [code, isInViewport])

  // Show placeholder until in viewport
  if (!isInViewport && !svg && !error) {
    return <div ref={ref}><MermaidPlaceholder /></div>
  }

  if (error) {
    return (
      <div ref={ref} className="rounded-md border border-red-300 bg-red-50 dark:border-red-800 dark:bg-red-950 p-4">
        <p className="text-sm text-red-600 dark:text-red-400 mb-2">
          Failed to render Mermaid diagram:
        </p>
        <pre className="text-xs text-red-500 dark:text-red-400 whitespace-pre-wrap">
          {error}
        </pre>
        <details className="mt-2">
          <summary className="text-xs text-red-500 dark:text-red-400 cursor-pointer">
            Show source
          </summary>
          <pre className="mt-2 text-xs bg-red-100 dark:bg-red-900 p-2 rounded overflow-x-auto">
            {code}
          </pre>
        </details>
      </div>
    )
  }

  // Show placeholder while rendering (after visible but before svg ready)
  if (!svg) {
    return <div ref={ref}><MermaidPlaceholder /></div>
  }

  return (
    <>
      <div className="relative my-4 rounded-md border border-[hsl(var(--border))] bg-[hsl(var(--card))] overflow-hidden group">
        {/* Toolbar - visible on hover */}
        <div className="absolute top-2 right-2 z-10 flex items-center gap-1 rounded-md border border-[hsl(var(--border))] bg-[hsl(var(--background))] p-1 opacity-0 group-hover:opacity-100 transition-opacity shadow-sm">
          <MermaidIconButton icon={MermaidIcons.zoomOut} onClick={zoomOut} title="Zoom Out" />
          <span className="text-xs text-[hsl(var(--muted-foreground))] min-w-[3rem] text-center">
            {Math.round(scale * 100)}%
          </span>
          <MermaidIconButton icon={MermaidIcons.zoomIn} onClick={zoomIn} title="Zoom In" />
          <MermaidIconButton icon={MermaidIcons.reset} onClick={resetZoom} title="Reset Zoom" />
          <div className="w-px h-4 bg-[hsl(var(--border))] mx-1" />
          <MermaidIconButton icon={MermaidIcons.maximize} onClick={() => setShowPreview(true)} title="Preview" />
        </div>
        {/* Diagram container - Mermaid's useMaxWidth handles sizing automatically */}
        <div className="overflow-auto p-4 flex justify-center">
          <div 
            className="transition-transform duration-200 w-full [&>svg]:w-full [&>svg]:min-w-[600px]"
            style={{ transform: `scale(${scale})`, transformOrigin: 'center top' }}
            dangerouslySetInnerHTML={{ __html: svg }}
          />
        </div>
      </div>
      {/* Preview Modal - uses original SVG for better zoom experience */}
      {showPreview && (
        <MermaidPreviewModal svg={svg} onClose={() => setShowPreview(false)} />
      )}
    </>
  )
}

/**
 * MarkdownRenderer component
 * Renders markdown content with syntax highlighting and Mermaid diagram support
 * Wrapped with React.memo to prevent unnecessary re-renders when props haven't changed
 */
export const MarkdownRenderer = memo(function MarkdownRenderer({ content, className }: MarkdownRendererProps) {
  // Detect dark mode for syntax highlighter
  const [isDark, setIsDark] = useState(false)

  useEffect(() => {
    // Check initial theme
    setIsDark(document.documentElement.classList.contains('dark'))

    // Watch for theme changes
    const observer = new MutationObserver((mutations) => {
      mutations.forEach((mutation) => {
        if (mutation.attributeName === 'class') {
          setIsDark(document.documentElement.classList.contains('dark'))
        }
      })
    })

    observer.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ['class'],
    })

    return () => observer.disconnect()
  }, [])

  // Create components with access to isDark state - memoized to prevent unnecessary re-renders
  const components: Components = useMemo(() => ({
    // Custom pre handler - we handle code blocks in the code component
    pre({ children }) {
      return <>{children}</>
    },
    // Custom code block renderer with lazy-loaded syntax highlighting and mermaid support
    code({ className, children, ...props }) {
      const match = /language-(\w+)/.exec(className || '')
      const language = match ? match[1] : ''
      const codeString = String(children).replace(/\n$/, '')

      // Check if this is inline code by checking the parent element
      // In react-markdown, code blocks are wrapped in <pre><code>
      // We detect this by checking if the code has newlines or language class
      const hasNewlines = codeString.includes('\n')
      const hasLanguage = Boolean(match)
      const isBlock = hasNewlines || hasLanguage

      // Handle mermaid diagrams with lazy loading
      if (language === 'mermaid') {
        return <MermaidDiagram code={codeString} />
      }

      // Render code block with lazy-loaded syntax highlighting
      if (isBlock) {
        return (
          <LazyCodeBlock
            code={codeString}
            language={language || 'text'}
            isDark={isDark}
          />
        )
      }

      // Inline code - no lazy loading needed (small)
      return (
        <code
          className="rounded bg-[hsl(var(--muted))] px-1.5 py-0.5 text-sm font-mono"
          {...props}
        >
          {children}
        </code>
      )
    },
    // Custom table styling
    table({ children }) {
      return (
        <div className="overflow-x-auto my-4">
          <table className="min-w-full border-collapse border border-[hsl(var(--border))]">
            {children}
          </table>
        </div>
      )
    },
    th({ children }) {
      return (
        <th className="border border-[hsl(var(--border))] bg-[hsl(var(--muted))] px-4 py-2 text-left font-semibold">
          {children}
        </th>
      )
    },
    td({ children }) {
      return (
        <td className="border border-[hsl(var(--border))] px-4 py-2">
          {children}
        </td>
      )
    },
    // Custom link styling
    a({ href, children }) {
      return (
        <a
          href={href}
          target="_blank"
          rel="noopener noreferrer"
          className="text-[hsl(var(--primary))] hover:underline"
        >
          {children}
        </a>
      )
    },
    // Custom blockquote styling
    blockquote({ children }) {
      return (
        <blockquote className="border-l-4 border-[hsl(var(--primary))] pl-4 italic text-[hsl(var(--muted-foreground))]">
          {children}
        </blockquote>
      )
    },
    // Custom heading styles - enhanced visual hierarchy
    h1({ children }) {
      return (
        <h1 className="text-3xl font-bold mt-8 mb-4 pb-2 border-b-2 border-[hsl(var(--primary))] text-[hsl(var(--foreground))]">
          {children}
        </h1>
      )
    },
    h2({ children }) {
      return (
        <h2 className="text-2xl font-bold mt-7 mb-3 pb-1.5 border-b border-[hsl(var(--border))]">
          {children}
        </h2>
      )
    },
    h3({ children }) {
      return (
        <h3 className="text-xl font-semibold mt-6 mb-2 pl-3 border-l-4 border-[hsl(var(--primary)/0.5)]">
          {children}
        </h3>
      )
    },
    h4({ children }) {
      return (
        <h4 className="text-lg font-semibold mt-5 mb-2 text-[hsl(var(--muted-foreground))]">
          {children}
        </h4>
      )
    },
    // Custom list styling
    ul({ children }) {
      return <ul className="list-disc list-inside my-2 space-y-1">{children}</ul>
    },
    ol({ children }) {
      return <ol className="list-decimal list-inside my-2 space-y-1">{children}</ol>
    },
    // Custom paragraph styling
    p({ children }) {
      return <p className="my-2 leading-relaxed">{children}</p>
    },
    // Custom horizontal rule
    hr() {
      return <hr className="my-6 border-[hsl(var(--border))]" />
    },
  }), [isDark])

  // Preprocess content to handle nested code blocks - memoized to avoid reprocessing
  const processedContent = useMemo(() => preprocessContent(content), [content])

  return (
    <div className={cn('prose prose-sm dark:prose-invert max-w-none', className)}>
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        components={components}
      >
        {processedContent}
      </ReactMarkdown>
    </div>
  )
})
