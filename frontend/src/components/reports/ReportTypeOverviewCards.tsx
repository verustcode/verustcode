import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import yaml from 'js-yaml'
import { FileText, Settings, Layers, BookOpen, FileCode, Check, X } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'
import type { ReportConfig } from '@/types/report'

interface ReportTypeOverviewCardsProps {
  content: string
}

/**
 * Boolean indicator component
 */
function BooleanIndicator({ value, label }: { value?: boolean; label: string }) {
  if (value === undefined) return null
  return (
    <div className="flex items-center gap-1.5 text-sm">
      {value ? (
        <Check className="h-3.5 w-3.5 text-emerald-500" />
      ) : (
        <X className="h-3.5 w-3.5 text-[hsl(var(--muted-foreground))]" />
      )}
      <span className={value ? 'text-[hsl(var(--foreground))]' : 'text-[hsl(var(--muted-foreground))]'}>
        {label}
      </span>
    </div>
  )
}

/**
 * Section title component for consistent styling
 */
function SectionTitle({ children, className }: { children: React.ReactNode; className?: string }) {
  return (
    <h4 className={cn('text-sm font-semibold text-[hsl(var(--foreground))] mb-2 flex items-center gap-1.5', className)}>
      {children}
    </h4>
  )
}

/**
 * Report type overview cards component
 * Displays parsed YAML content as read-only cards
 */
export function ReportTypeOverviewCards({ content }: ReportTypeOverviewCardsProps) {
  const { t } = useTranslation()
  const [config, setConfig] = useState<ReportConfig | null>(null)
  const [parseError, setParseError] = useState<string | null>(null)

  // Parse YAML content
  useEffect(() => {
    try {
      const parsed = yaml.load(content) as ReportConfig
      if (parsed && typeof parsed === 'object') {
        setConfig(parsed)
        setParseError(null)
      }
    } catch (err) {
      setParseError(err instanceof Error ? err.message : 'Parse error')
    }
  }, [content])

  // Show parse error with hint to switch to YAML mode
  if (parseError) {
    return (
      <div className="flex h-full flex-col items-center justify-center text-[hsl(var(--destructive))]">
        <p>YAML Parse Error: {parseError}</p>
        <p className="mt-2 text-sm">{t('reportTypes.switchToYamlHint')}</p>
      </div>
    )
  }

  if (!config) {
    return (
      <div className="flex h-full flex-col items-center justify-center text-[hsl(var(--muted-foreground))]">
        <p>{t('reportTypes.noConfig')}</p>
      </div>
    )
  }

  return (
    <div className="space-y-6 pr-4">
      {/* Basic Information */}
      <Card className="border-[hsl(var(--border))]/70">
        <CardHeader className="pb-4">
          <div className="flex items-center gap-2">
            <div className="rounded-md bg-[hsl(var(--primary))]/10 p-1.5 text-[hsl(var(--primary))]">
              <FileText className="h-4 w-4" />
            </div>
            <CardTitle className="text-base">{t('reportTypes.form.basicInfo')}</CardTitle>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <span className="text-xs text-[hsl(var(--muted-foreground))]">
                {t('reportTypes.form.id')}
              </span>
              <p className="mt-1 font-mono text-sm font-medium">{config.id}</p>
            </div>
            <div>
              <span className="text-xs text-[hsl(var(--muted-foreground))]">
                {t('reportTypes.form.version')}
              </span>
              <p className="mt-1 font-mono text-sm">{config.version || '1.0'}</p>
            </div>
          </div>
          <div>
            <span className="text-xs text-[hsl(var(--muted-foreground))]">
              {t('reportTypes.form.name')}
            </span>
            <p className="mt-1 text-sm font-medium">{config.name}</p>
          </div>
          {config.description && (
            <div>
              <span className="text-xs text-[hsl(var(--muted-foreground))]">
                {t('reportTypes.form.description')}
              </span>
              <p className="mt-1 text-sm text-[hsl(var(--muted-foreground))] leading-relaxed">
                {config.description}
              </p>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Agent Configuration */}
      <Card className="border-[hsl(var(--border))]/70">
        <CardHeader className="pb-4">
          <div className="flex items-center gap-2">
            <div className="rounded-md bg-[hsl(var(--primary))]/10 p-1.5 text-[hsl(var(--primary))]">
              <Settings className="h-4 w-4" />
            </div>
            <CardTitle className="text-base">{t('reportTypes.form.agentConfig')}</CardTitle>
          </div>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="flex flex-wrap items-center gap-4">
            <div className="flex items-center gap-2">
              <span className="text-xs text-[hsl(var(--muted-foreground))]">
                {t('reportTypes.form.agentType')}:
              </span>
              <Badge variant="secondary" className="font-mono text-sm bg-[hsl(var(--primary))]/10 text-[hsl(var(--primary))] border-[hsl(var(--primary))]/20">
                {config.agent?.type || 'cursor'}
              </Badge>
            </div>
            {config.agent?.model && (
              <div className="flex items-center gap-2">
                <span className="text-xs text-[hsl(var(--muted-foreground))]">
                  {t('reportTypes.form.agentModel')}:
                </span>
                <Badge variant="outline" className="font-mono text-sm">
                  {config.agent.model}
                </Badge>
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Output Style Configuration */}
      {config.output?.style && (
        <Card className="border-[hsl(var(--border))]/70">
          <CardHeader className="pb-4">
            <div className="flex items-center gap-2">
              <div className="rounded-md bg-[hsl(var(--primary))]/10 p-1.5 text-[hsl(var(--primary))]">
                <FileCode className="h-4 w-4" />
              </div>
              <CardTitle className="text-base">{t('reportTypes.form.outputStyle')}</CardTitle>
            </div>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              {config.output.style.tone && (
                <div className="flex items-center gap-2">
                  <span className="text-xs text-[hsl(var(--muted-foreground))]">
                    {t('reportTypes.form.tone')}:
                  </span>
                  <Badge variant="outline" className="text-sm">
                    {config.output.style.tone}
                  </Badge>
                </div>
              )}
              {config.output.style.language && (
                <div className="flex items-center gap-2">
                  <span className="text-xs text-[hsl(var(--muted-foreground))]">
                    {t('reportTypes.form.language')}:
                  </span>
                  <Badge variant="outline" className="text-sm">
                    {config.output.style.language}
                  </Badge>
                </div>
              )}
              {config.output.style.heading_level !== undefined && (
                <div className="flex items-center gap-2">
                  <span className="text-xs text-[hsl(var(--muted-foreground))]">
                    {t('reportTypes.form.headingLevel')}:
                  </span>
                  <Badge variant="outline" className="text-sm">
                    {config.output.style.heading_level}
                  </Badge>
                </div>
              )}
              {config.output.style.max_section_length && (
                <div className="flex items-center gap-2">
                  <span className="text-xs text-[hsl(var(--muted-foreground))]">
                    {t('reportTypes.form.maxSectionLength')}:
                  </span>
                  <Badge variant="outline" className="text-sm">
                    {config.output.style.max_section_length}
                  </Badge>
                </div>
              )}
            </div>
            <div className="flex flex-wrap gap-3">
              <BooleanIndicator value={config.output.style.concise} label={t('reportTypes.form.concise')} />
              <BooleanIndicator value={config.output.style.no_emoji} label={t('reportTypes.form.noEmoji')} />
              <BooleanIndicator value={config.output.style.use_mermaid} label={t('reportTypes.form.useMermaid')} />
              <BooleanIndicator value={config.output.style.include_line_numbers} label={t('reportTypes.form.includeLineNumbers')} />
            </div>
          </CardContent>
        </Card>
      )}

      {/* Phase 1: Structure Generation */}
      <Card className="border-[hsl(var(--border))]/70">
        <CardHeader className="pb-4">
          <div className="flex items-center gap-2">
            <div className="rounded-md bg-[hsl(var(--primary))]/10 p-1.5 text-[hsl(var(--primary))]">
              <Layers className="h-4 w-4" />
            </div>
            <CardTitle className="text-base">{t('reportTypes.form.phase1Structure')}</CardTitle>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          {config.structure.description && (
            <div>
              <SectionTitle>{t('reportTypes.form.description')}</SectionTitle>
              <p className="text-sm text-[hsl(var(--muted-foreground))] leading-relaxed">
                {config.structure.description}
              </p>
            </div>
          )}

          {/* Nested Structure */}
          {config.structure.nested !== undefined && (
            <div className="flex items-center gap-2">
              <span className="text-sm font-medium text-[hsl(var(--foreground))]">{t('reportTypes.form.nested')}:</span>
              <Badge variant="outline" className={cn('text-xs', config.structure.nested ? 'text-emerald-600 dark:text-emerald-400 border-emerald-500/30' : 'text-[hsl(var(--muted-foreground))] border-[hsl(var(--border))]')}>
                {config.structure.nested ? t('common.yes') : t('common.no')}
              </Badge>
            </div>
          )}

          {config.structure.goals?.topics && config.structure.goals.topics.length > 0 && (
            <div>
              <SectionTitle>{t('reportTypes.form.topics')}</SectionTitle>
              <ul className="list-disc space-y-1 pl-6">
                {config.structure.goals.topics.map((topic, idx) => (
                  <li key={idx} className="text-sm text-[hsl(var(--muted-foreground))]">
                    {topic}
                  </li>
                ))}
              </ul>
            </div>
          )}

          {config.structure.constraints && config.structure.constraints.length > 0 && (
            <div>
              <SectionTitle>{t('reportTypes.form.constraints')}</SectionTitle>
              <ul className="list-disc space-y-1 pl-6">
                {config.structure.constraints.map((constraint, idx) => (
                  <li key={idx} className="text-sm text-[hsl(var(--muted-foreground))]">
                    {constraint}
                  </li>
                ))}
              </ul>
            </div>
          )}
        </CardContent>
        {config.structure.reference_docs && config.structure.reference_docs.length > 0 && (
          <div className="px-6 pb-6">
            <SectionTitle className="mb-2">
              <BookOpen className="h-3.5 w-3.5 text-blue-500" />
              {t('reportTypes.form.referenceDocs')}
            </SectionTitle>
            <div className="rounded-lg border border-blue-500/20 bg-blue-500/5 p-3">
              <ul className="space-y-1">
                {config.structure.reference_docs.map((doc, idx) => (
                  <li key={idx} className="flex items-center gap-2 text-[hsl(var(--muted-foreground))]">
                    <FileText className="h-3.5 w-3.5 flex-shrink-0 text-blue-400" />
                    <span className="font-mono text-xs">{doc}</span>
                  </li>
                ))}
              </ul>
            </div>
          </div>
        )}
        {config.structure.goals?.avoid && config.structure.goals.avoid.length > 0 && (
          <div className="px-6 pb-6">
            <SectionTitle className="mb-2">{t('reportTypes.form.avoid')}</SectionTitle>
            <div className="rounded-lg border border-red-500/20 bg-red-500/5 p-3">
              <ul className="space-y-1">
                {config.structure.goals.avoid.map((item, idx) => (
                  <li key={idx} className="text-sm text-[hsl(var(--muted-foreground))]">
                    {item}
                  </li>
                ))}
              </ul>
            </div>
          </div>
        )}
      </Card>

      {/* Phase 2: Section Content Generation */}
      <Card className="border-[hsl(var(--border))]/70">
        <CardHeader className="pb-4">
          <div className="flex items-center gap-2">
            <div className="rounded-md bg-[hsl(var(--primary))]/10 p-1.5 text-[hsl(var(--primary))]">
              <FileText className="h-4 w-4" />
            </div>
            <CardTitle className="text-base">{t('reportTypes.form.phase2Section')}</CardTitle>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          {config.section.description && (
            <div>
              <SectionTitle>{t('reportTypes.form.description')}</SectionTitle>
              <p className="text-sm text-[hsl(var(--muted-foreground))] leading-relaxed">
                {config.section.description}
              </p>
            </div>
          )}

          {config.section.goals?.topics && config.section.goals.topics.length > 0 && (
            <div>
              <SectionTitle>{t('reportTypes.form.topics')}</SectionTitle>
              <ul className="list-disc space-y-1 pl-6">
                {config.section.goals.topics.map((topic, idx) => (
                  <li key={idx} className="text-sm text-[hsl(var(--muted-foreground))]">
                    {topic}
                  </li>
                ))}
              </ul>
            </div>
          )}

          {config.section.constraints && config.section.constraints.length > 0 && (
            <div>
              <SectionTitle>{t('reportTypes.form.constraints')}</SectionTitle>
              <ul className="list-disc space-y-1 pl-6">
                {config.section.constraints.map((constraint, idx) => (
                  <li key={idx} className="text-sm text-[hsl(var(--muted-foreground))]">
                    {constraint}
                  </li>
                ))}
              </ul>
            </div>
          )}

          {config.section.summary && (
            <div className="flex items-center gap-2">
              <span className="text-sm font-medium text-[hsl(var(--foreground))]">
                {t('reportTypes.form.summaryMaxLength')}:
              </span>
              <Badge variant="outline" className="text-sm">
                {config.section.summary.max_length || 200}
              </Badge>
            </div>
          )}
        </CardContent>
        {config.section.reference_docs && config.section.reference_docs.length > 0 && (
          <div className="px-6 pb-6">
            <SectionTitle className="mb-2">
              <BookOpen className="h-3.5 w-3.5 text-blue-500" />
              {t('reportTypes.form.referenceDocs')}
            </SectionTitle>
            <div className="rounded-lg border border-blue-500/20 bg-blue-500/5 p-3">
              <ul className="space-y-1">
                {config.section.reference_docs.map((doc, idx) => (
                  <li key={idx} className="flex items-center gap-2 text-[hsl(var(--muted-foreground))]">
                    <FileText className="h-3.5 w-3.5 flex-shrink-0 text-blue-400" />
                    <span className="font-mono text-xs">{doc}</span>
                  </li>
                ))}
              </ul>
            </div>
          </div>
        )}
        {config.section.goals?.avoid && config.section.goals.avoid.length > 0 && (
          <div className="px-6 pb-6">
            <SectionTitle className="mb-2">{t('reportTypes.form.avoid')}</SectionTitle>
            <div className="rounded-lg border border-red-500/20 bg-red-500/5 p-3">
              <ul className="space-y-1">
                {config.section.goals.avoid.map((item, idx) => (
                  <li key={idx} className="text-sm text-[hsl(var(--muted-foreground))]">
                    {item}
                  </li>
                ))}
              </ul>
            </div>
          </div>
        )}
      </Card>

      {/* Phase 3: Report Summary Generation */}
      <Card className="border-[hsl(var(--border))]/70">
        <CardHeader className="pb-4">
          <div className="flex items-center gap-2">
            <div className="rounded-md bg-[hsl(var(--primary))]/10 p-1.5 text-[hsl(var(--primary))]">
              <FileText className="h-4 w-4" />
            </div>
            <CardTitle className="text-base">{t('reportTypes.form.phase3Summary')}</CardTitle>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          {config.summary.description && (
            <div>
              <SectionTitle>{t('reportTypes.form.description')}</SectionTitle>
              <p className="text-sm text-[hsl(var(--muted-foreground))] leading-relaxed">
                {config.summary.description}
              </p>
            </div>
          )}

          {config.summary.goals?.topics && config.summary.goals.topics.length > 0 && (
            <div>
              <SectionTitle>{t('reportTypes.form.topics')}</SectionTitle>
              <ul className="list-disc space-y-1 pl-6">
                {config.summary.goals.topics.map((topic, idx) => (
                  <li key={idx} className="text-sm text-[hsl(var(--muted-foreground))]">
                    {topic}
                  </li>
                ))}
              </ul>
            </div>
          )}

          {config.summary.constraints && config.summary.constraints.length > 0 && (
            <div>
              <SectionTitle>{t('reportTypes.form.constraints')}</SectionTitle>
              <ul className="list-disc space-y-1 pl-6">
                {config.summary.constraints.map((constraint, idx) => (
                  <li key={idx} className="text-sm text-[hsl(var(--muted-foreground))]">
                    {constraint}
                  </li>
                ))}
              </ul>
            </div>
          )}
        </CardContent>
        {config.summary.goals?.avoid && config.summary.goals.avoid.length > 0 && (
          <div className="px-6 pb-6">
            <SectionTitle className="mb-2">{t('reportTypes.form.avoid')}</SectionTitle>
            <div className="rounded-lg border border-red-500/20 bg-red-500/5 p-3">
              <ul className="space-y-1">
                {config.summary.goals.avoid.map((item, idx) => (
                  <li key={idx} className="text-sm text-[hsl(var(--muted-foreground))]">
                    {item}
                  </li>
                ))}
              </ul>
            </div>
          </div>
        )}
      </Card>
    </div>
  )
}
