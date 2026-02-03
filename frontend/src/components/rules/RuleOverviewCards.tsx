import { useEffect, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import yaml from 'js-yaml'
import { Bot, FileText, MessageSquare, Webhook, Terminal, Check, X, Repeat, Layers, FileCode, BookOpen } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'
import type { ReviewRulesConfig, ReviewRuleConfig, RuleBaseConfig, OutputItemConfig, ConstraintsConfig, OutputConfig, MultiRunConfig, AgentConfig } from '@/types/rule'
import { useState } from 'react'

/**
 * Agent display info returned by getAgentDisplayInfo
 */
interface AgentDisplayInfo {
  agentType: string
  model?: string
}

/**
 * Get display info for agent configuration
 * Handles both new object format {type, model} and legacy string format
 * Returns an object with agentType and optional model for flexible rendering
 */
function getAgentDisplayInfo(agent?: AgentConfig | string): AgentDisplayInfo {
  if (!agent) return { agentType: 'cursor' }
  if (typeof agent === 'string') return { agentType: agent }
  // New object format: return agentType and model separately
  const agentType = agent.type || 'cursor'
  return { agentType, model: agent.model }
}

/**
 * Get agent type from configuration
 */
function getAgentType(agent?: AgentConfig | string): string {
  if (!agent) return 'cursor'
  if (typeof agent === 'string') return agent
  return agent.type || 'cursor'
}

// Severity level colors
const SEVERITY_COLORS: Record<string, string> = {
  critical: 'bg-red-500/15 text-red-600 dark:text-red-400 border-red-500/30',
  high: 'bg-orange-500/15 text-orange-600 dark:text-orange-400 border-orange-500/30',
  medium: 'bg-amber-500/15 text-amber-600 dark:text-amber-400 border-amber-500/30',
  low: 'bg-blue-500/15 text-blue-600 dark:text-blue-400 border-blue-500/30',
  info: 'bg-sky-500/15 text-sky-600 dark:text-sky-400 border-sky-500/30',
}

interface RuleOverviewCardsProps {
  content: string
  activeRuleId?: string | null
}

/**
 * Get output channel icon component
 */
function OutputIcon({ type }: { type: OutputItemConfig['type'] }) {
  switch (type) {
    case 'console':
      return <Terminal className="h-4 w-4" />
    case 'file':
      return <FileText className="h-4 w-4" />
    case 'comment':
      return <MessageSquare className="h-4 w-4" />
    case 'webhook':
      return <Webhook className="h-4 w-4" />
  }
}

/**
 * Severity badge component
 */
function SeverityBadge({ level }: { level: string }) {
  return (
    <Badge
      variant="outline"
      className={cn('text-sm font-medium', SEVERITY_COLORS[level] || SEVERITY_COLORS.medium)}
    >
      {level}
    </Badge>
  )
}

/**
 * Section title component for consistent styling
 */
function SectionTitle({ children, className }: { children: React.ReactNode; className?: string }) {
  return (
    <h4 className={cn('text-sm font-semibold text-[hsl(var(--foreground))] uppercase tracking-wider mb-2 flex items-center gap-1.5', className)}>
      {children}
    </h4>
  )
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
 * Output channel item component with detailed info
 * Uses vertical layout for all channel types
 */
function OutputChannelItem({ item, t }: { item: OutputItemConfig; t: (key: string) => string }) {
  const isFile = item.type === 'file'
  const isComment = item.type === 'comment'
  const isWebhook = item.type === 'webhook'

  // Check if channel has any details to show
  const hasFileDetails = isFile && (item.dir || item.overwrite !== undefined)
  const hasCommentDetails = isComment && (item.overwrite !== undefined || item.mode || item.marker_prefix)
  const hasWebhookDetails = isWebhook && (item.url || item.timeout || item.max_retries)
  const hasDetails = hasFileDetails || hasCommentDetails || hasWebhookDetails

  return (
    <div className="flex items-start gap-2.5 rounded-lg border border-[hsl(var(--border))]/50 bg-[hsl(var(--muted))]/30 px-3 py-2">
      <div className="mt-0.5 rounded-md bg-[hsl(var(--primary))]/10 p-1.5 text-[hsl(var(--primary))]">
        <OutputIcon type={item.type} />
      </div>
      <div className="flex-1 min-w-0">
        <span className="text-sm font-medium">{item.type}</span>
        {hasDetails && (
          <div className="text-xs text-[hsl(var(--muted-foreground))] space-y-0.5 mt-0.5">
            {/* File channel */}
            {isFile && (
              <>
                {item.dir && <p className="truncate">{t('rules.outputDir')}: {item.dir}</p>}
                {item.overwrite !== undefined && (
                  <p>{item.overwrite ? t('rules.overwrite') : t('rules.append')}</p>
                )}
              </>
            )}
            {/* Comment channel */}
            {isComment && (
              <>
                {item.overwrite !== undefined && (
                  <p>{item.overwrite ? t('rules.overwrite') : t('rules.append')}</p>
                )}
                {item.mode && <p>{t('rules.mode')}: {item.mode}</p>}
                {item.marker_prefix && <p className="truncate">{t('rules.markerPrefix')}: {item.marker_prefix}</p>}
              </>
            )}
            {/* Webhook channel */}
            {isWebhook && (
              <>
                {item.url && <p className="truncate">{item.url}</p>}
                {item.timeout && <p>{t('rules.timeout')}: {item.timeout}s</p>}
                {item.max_retries && <p>{t('rules.maxRetries')}: {item.max_retries}</p>}
              </>
            )}
          </div>
        )}
      </div>
    </div>
  )
}

/**
 * Constraints section component
 */
function ConstraintsSection({ constraints, t }: { constraints?: ConstraintsConfig; t: (key: string) => string }) {
  if (!constraints) return null
  
  const { scope_control, focus_on_issues_only, severity, duplicates } = constraints
  const hasScopeControl = scope_control && scope_control.length > 0
  const hasFocusOnIssuesOnly = focus_on_issues_only !== undefined
  const hasSeverity = severity && severity.min_report
  const hasDuplicates = duplicates && (duplicates.suppress_similar !== undefined || duplicates.similarity !== undefined)
  
  if (!hasScopeControl && !hasFocusOnIssuesOnly && !hasSeverity && !hasDuplicates) return null

  return (
    <div className="space-y-3">
      {/* Scope Control */}
      {hasScopeControl && (
        <div className="space-y-1.5">
          <span className="text-sm font-medium text-[hsl(var(--foreground))]">{t('rules.scopeControl')}:</span>
          <ul className="space-y-1 pl-3">
            {scope_control!.map((item, idx) => (
              <li key={idx} className="text-sm text-[hsl(var(--muted-foreground))] list-disc list-inside">
                {item}
              </li>
            ))}
          </ul>
        </div>
      )}

      {/* Focus on Issues Only */}
      {hasFocusOnIssuesOnly && (
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium text-[hsl(var(--foreground))]">{t('rules.focusOnIssuesOnly')}:</span>
          <Badge variant="outline" className={cn('text-xs', focus_on_issues_only ? 'text-emerald-600 dark:text-emerald-400 border-emerald-500/30' : 'text-[hsl(var(--muted-foreground))] border-[hsl(var(--border))]')}>
            {focus_on_issues_only ? t('common.yes') : t('common.no')}
          </Badge>
        </div>
      )}
      
      {/* Severity Configuration - Only min_report is configurable */}
      {hasSeverity && (
        <div className="space-y-1.5">
          <span className="text-sm font-medium text-[hsl(var(--foreground))]">{t('rules.severity')}:</span>
          <div className="flex flex-wrap items-center gap-2 pl-3">
            <div className="flex items-center gap-1.5">
              <span className="text-sm text-[hsl(var(--muted-foreground))]">{t('rules.minReport')}:</span>
              <SeverityBadge level={severity!.min_report!} />
            </div>
          </div>
        </div>
      )}
      
      {/* Duplicates Configuration */}
      {hasDuplicates && (
        <div className="flex flex-wrap items-center gap-3 pl-3">
          <BooleanIndicator value={duplicates!.suppress_similar} label={t('rules.suppressSimilar')} />
          {duplicates!.similarity !== undefined && (
            <span className="text-sm text-[hsl(var(--muted-foreground))]">
              {t('rules.similarity')}: {(duplicates!.similarity * 100).toFixed(0)}%
            </span>
          )}
        </div>
      )}
    </div>
  )
}

/**
 * Output section component
 * Displays format, schema, style, and output channels
 */
function OutputSection({ output, t, showTitle = true }: { output?: OutputConfig; t: (key: string) => string; showTitle?: boolean }) {
  if (!output) return null
  
  const { format, style, schema, channels } = output
  const hasStyle = style && (style.tone || style.concise !== undefined || style.no_emoji !== undefined || style.no_date !== undefined || style.language)
  const hasChannels = channels && channels.length > 0
  const hasSchema = schema && schema.extra_fields && schema.extra_fields.length > 0
  
  if (!format && !hasStyle && !hasChannels && !hasSchema) return null

  return (
    <div className="space-y-3">
      {showTitle && <SectionTitle>{t('rules.output')}</SectionTitle>}
      
      {/* Format - 顶级 */}
      {format && (
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium text-[hsl(var(--foreground))]">{t('rules.format')}:</span>
          <Badge variant="outline" className="text-sm">{format}</Badge>
        </div>
      )}
      
      {/* Extra Fields - 顶级 */}
      {hasSchema && (
        <div className="space-y-1.5">
          <span className="text-sm font-medium text-[hsl(var(--foreground))]">{t('rules.extraFields')}:</span>
          <div className="space-y-1 pl-3">
            {schema!.extra_fields!.map((field, idx) => (
              <div key={idx} className="flex items-center gap-1.5">
                <FileCode className="h-3 w-3 text-[hsl(var(--muted-foreground))]" />
                <Badge variant="outline" className="text-xs font-mono">
                  {field.name}: {field.type}
                  {field.required && <span className="text-red-500 ml-1">*</span>}
                </Badge>
                {field.description && (
                  <span className="text-xs text-[hsl(var(--muted-foreground))]">- {field.description}</span>
                )}
              </div>
            ))}
          </div>
        </div>
      )}
      
      {/* Style - 顶级 */}
      {hasStyle && (
        <div className="space-y-1.5">
          <span className="text-sm font-medium text-[hsl(var(--foreground))]">{t('rules.style')}:</span>
          <div className="space-y-1.5 pl-3">
            {/* Tone */}
            {style.tone && (
              <div className="flex items-center gap-2">
                <span className="text-sm text-[hsl(var(--muted-foreground))]">{t('rules.tone')}:</span>
                <Badge variant="outline" className="text-sm">{style.tone}</Badge>
              </div>
            )}
            {/* Language */}
            {style.language && (
              <div className="flex items-center gap-2">
                <span className="text-sm text-[hsl(var(--muted-foreground))]">{t('rules.language')}:</span>
                <Badge variant="outline" className="text-sm">{style.language}</Badge>
              </div>
            )}
            {/* Boolean flags */}
            {style.concise !== undefined && (
              <div className="flex items-center gap-2">
                <span className="text-sm text-[hsl(var(--muted-foreground))]">{t('rules.concise')}:</span>
                <Badge variant="outline" className={cn('text-xs', style.concise ? 'text-emerald-600 dark:text-emerald-400 border-emerald-500/30' : 'text-[hsl(var(--muted-foreground))] border-[hsl(var(--border))]')}>
                  {style.concise ? t('common.yes') : t('common.no')}
                </Badge>
              </div>
            )}
            {style.no_emoji !== undefined && (
              <div className="flex items-center gap-2">
                <span className="text-sm text-[hsl(var(--muted-foreground))]">{t('rules.noEmoji')}:</span>
                <Badge variant="outline" className={cn('text-xs', style.no_emoji ? 'text-emerald-600 dark:text-emerald-400 border-emerald-500/30' : 'text-[hsl(var(--muted-foreground))] border-[hsl(var(--border))]')}>
                  {style.no_emoji ? t('common.yes') : t('common.no')}
                </Badge>
              </div>
            )}
            {style.no_date !== undefined && (
              <div className="flex items-center gap-2">
                <span className="text-sm text-[hsl(var(--muted-foreground))]">{t('rules.noDate')}:</span>
                <Badge variant="outline" className={cn('text-xs', style.no_date ? 'text-emerald-600 dark:text-emerald-400 border-emerald-500/30' : 'text-[hsl(var(--muted-foreground))] border-[hsl(var(--border))]')}>
                  {style.no_date ? t('common.yes') : t('common.no')}
                </Badge>
              </div>
            )}
          </div>
        </div>
      )}
      
      {/* Output Channels - 顶级 */}
      {hasChannels && (
        <div className="space-y-1.5">
          <span className="text-sm font-medium text-[hsl(var(--foreground))]">{t('rules.outputChannels')}:</span>
          <div className="grid gap-1.5 sm:grid-cols-2 mt-2">
            {channels!.map((item: OutputItemConfig, idx: number) => (
              <OutputChannelItem key={idx} item={item} t={t} />
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

/**
 * Multi-run section component
 * Multi-run is automatically enabled when runs >= 2 (max 3 runs)
 */
function MultiRunSection({ multiRun, t }: { multiRun?: MultiRunConfig; t: (key: string) => string }) {
  // Multi-run is enabled when runs >= 2
  const isEnabled = multiRun && multiRun.runs !== undefined && multiRun.runs >= 2
  if (!isEnabled) return null

  return (
    <div className="space-y-2">
      <SectionTitle>
        <div className="flex items-center gap-1.5">
          <Repeat className="h-3.5 w-3.5" />
          {t('rules.multiRun')}
        </div>
      </SectionTitle>
      <div className="flex flex-wrap items-center gap-3">
        <BooleanIndicator value={true} label={t('rules.enabled')} />
        {multiRun.runs && (
          <div className="flex items-center gap-1.5">
            <span className="text-sm text-[hsl(var(--muted-foreground))]">{t('rules.runs')}:</span>
            <Badge variant="secondary" className="text-sm">{multiRun.runs}</Badge>
          </div>
        )}
        {multiRun.merge_model && (
          <div className="flex items-center gap-1.5">
            <span className="text-sm text-[hsl(var(--muted-foreground))]">{t('rules.mergeModel')}:</span>
            <Badge variant="secondary" className="text-sm font-mono">{multiRun.merge_model}</Badge>
          </div>
        )}
      </div>
      {multiRun.models && multiRun.models.length > 0 && (
        <div className="space-y-1.5">
          <span className="text-sm text-[hsl(var(--muted-foreground))]">{t('rules.models')}:</span>
          <div className="flex flex-wrap gap-1.5">
            {multiRun.models.map((model, idx) => (
              <Badge key={idx} variant="outline" className="text-sm font-mono">{model}</Badge>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

/**
 * Rule base summary card component - Enhanced version with complete config display
 */
function RuleBaseCard({ ruleBase }: { ruleBase?: RuleBaseConfig }) {
  const { t } = useTranslation()
  
  if (!ruleBase) return null

  const agentInfo = getAgentDisplayInfo(ruleBase.agent)
  const hasConstraints = ruleBase.constraints && (
    ruleBase.constraints.scope_control?.length ||
    ruleBase.constraints.severity ||
    ruleBase.constraints.duplicates
  )
  const hasOutput = ruleBase.output && (
    ruleBase.output.format ||
    ruleBase.output.style ||
    (ruleBase.output.schema?.extra_fields && ruleBase.output.schema.extra_fields.length > 0) ||
    ruleBase.output.channels?.length
  )

  return (
    <div className="mb-6">
      {/* Title row - aligned with Rules List title style */}
      <div className="flex items-center gap-2 mb-4 pb-2 border-b border-[hsl(var(--border))]/50">
        <div className="rounded-md bg-[hsl(var(--primary))]/10 p-1.5 text-[hsl(var(--primary))]">
          <Layers className="h-4 w-4" />
        </div>
        <h3 className="text-sm font-semibold text-[hsl(var(--foreground))] uppercase tracking-wide">
          {t('rules.ruleBaseTitle')}
        </h3>
      </div>
      
      {/* Content - using Card for each section like RuleCard style */}
      <Card className="border-[hsl(var(--border))]/70">
        <CardContent className="pt-5 space-y-5">
          {/* Basic Info Row - Agent and Model as separate badges */}
          <div className="flex flex-wrap items-center gap-4 text-base">
            <div className="flex items-center gap-2">
              <span className="text-xs uppercase tracking-wide text-[hsl(var(--muted-foreground))]">{t('rules.agent')}:</span>
              <Badge variant="secondary" className="font-mono text-sm bg-[hsl(var(--primary))]/10 text-[hsl(var(--primary))] border-[hsl(var(--primary))]/20">
                <Bot className="mr-1.5 h-3.5 w-3.5" />
                {agentInfo.agentType}
              </Badge>
            </div>
            {agentInfo.model && (
              <div className="flex items-center gap-2">
                <span className="text-xs uppercase tracking-wide text-[hsl(var(--muted-foreground))]">{t('rules.model')}:</span>
                <Badge variant="outline" className="font-mono text-sm">
                  {agentInfo.model}
                </Badge>
              </div>
            )}
          </div>

          {/* Output Section */}
          {hasOutput && (
            <div className="rounded-lg border border-[hsl(var(--border))]/40 bg-[hsl(var(--background))]/50 p-4">
              <OutputSection output={ruleBase.output} t={t} />
            </div>
          )}
        </CardContent>
        {/* Constraints Section */}
        {hasConstraints && (
          <div className="px-6 pb-6">
            <SectionTitle className="mb-2">{t('rules.constraints')}</SectionTitle>
            <div className="rounded-lg border border-[hsl(var(--border))]/40 bg-[hsl(var(--background))]/50 p-4">
              <ConstraintsSection constraints={ruleBase.constraints} t={t} />
            </div>
          </div>
        )}
      </Card>
    </div>
  )
}

/**
 * Individual rule card component - Enhanced version with complete config display
 */
function RuleCard({ 
  rule, 
  defaultAgentType, 
  isActive,
  id,
}: { 
  rule: ReviewRuleConfig
  defaultAgentType?: string
  isActive?: boolean
  id: string
}) {
  const { t } = useTranslation()
  
  // Computed values with defaults from rule_base
  // Use rule's agent if specified, otherwise fall back to default agent type
  const agentInfo = rule.agent 
    ? getAgentDisplayInfo(rule.agent) 
    : { agentType: defaultAgentType || 'cursor' }
  const minSeverity = rule.constraints?.severity?.min_report
  const areas = rule.goals?.areas || []
  const avoidList = rule.goals?.avoid || []
  
  // Check if rule has custom output config
  const hasCustomOutput = rule.output && (
    rule.output.format ||
    rule.output.style ||
    (rule.output.schema?.extra_fields && rule.output.schema.extra_fields.length > 0) ||
    rule.output.channels?.length
  )
  
  // Check if rule has constraints
  const hasConstraints = rule.constraints && (
    rule.constraints.scope_control?.length ||
    rule.constraints.severity ||
    rule.constraints.duplicates
  )

  return (
    <Card 
      id={id}
      className={cn(
        'mb-4 transition-all duration-200',
        isActive 
          ? '!border-2 !border-[hsl(var(--primary))] dark:!border-[hsl(var(--primary))]' 
          : '!border !border-[hsl(var(--border))]/70'
      )}
    >
      <CardHeader className="pb-4">
        {/* Title row with clear hierarchy */}
        <div className="flex items-start justify-between gap-4">
          <div className="flex-1 min-w-0">
            <CardTitle className="text-lg font-bold tracking-tight text-[hsl(var(--foreground))]">
              {rule.id}
            </CardTitle>
            <p className="text-sm text-[hsl(var(--muted-foreground))] mt-1">{rule.role}</p>
          </div>
          <div className="flex items-center gap-2 flex-shrink-0">
            {/* Agent badge */}
            <Badge variant="secondary" className="font-mono text-xs bg-[hsl(var(--muted))]/80">
              <Bot className="mr-1 h-3 w-3" />
              {agentInfo.agentType}
            </Badge>
            {/* Model badge - only show if model is configured */}
            {agentInfo.model && (
              <Badge variant="outline" className="font-mono text-xs">
                {agentInfo.model}
              </Badge>
            )}
            {minSeverity && <SeverityBadge level={minSeverity} />}
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-5">
        {/* Description */}
        {rule.description && (
          <p className="text-sm text-[hsl(var(--muted-foreground))] leading-relaxed">
            {rule.description.trim()}
          </p>
        )}

        {/* Reference Documentation */}
        {rule.reference_docs && rule.reference_docs.length > 0 && (
          <div className="space-y-2.5 rounded-lg border border-blue-500/20 bg-blue-500/5 p-3">
            <SectionTitle>
              <BookOpen className="h-3.5 w-3.5 text-blue-500" />
              {t('rules.referenceDocs')}
              <span className="text-[10px] font-normal text-[hsl(var(--muted-foreground))] ml-1">
                ({rule.reference_docs.length}/5)
              </span>
            </SectionTitle>
            <ul className="space-y-1.5">
              {rule.reference_docs.map((doc, idx) => (
                <li key={idx} className="flex items-center gap-2 text-[hsl(var(--muted-foreground))]">
                  <FileText className="h-3.5 w-3.5 flex-shrink-0 text-blue-400" />
                  <span className="font-mono text-xs truncate">{doc}</span>
                </li>
              ))}
            </ul>
          </div>
        )}

        {/* Focus Areas - Show all without folding */}
        {areas.length > 0 && (
          <div className="space-y-2.5">
            <SectionTitle>{t('rules.areas')}</SectionTitle>
            <div className="flex flex-wrap gap-1.5">
              {areas.map((area) => (
                <Badge 
                  key={area} 
                  variant="secondary" 
                  className="text-xs font-normal bg-[hsl(var(--primary))]/8 text-[hsl(var(--primary))] border border-[hsl(var(--primary))]/15"
                >
                  {area}
                </Badge>
              ))}
            </div>
          </div>
        )}

        {/* Multi-Run Section */}
        {rule.multi_run && (
          <div className="rounded-lg border border-purple-500/20 bg-purple-500/5 p-3">
            <MultiRunSection multiRun={rule.multi_run} t={t} />
          </div>
        )}

        {/* Custom Output Section */}
        {hasCustomOutput && (
          <div className="rounded-lg border border-[hsl(var(--border))]/50 bg-[hsl(var(--muted))]/20 p-3">
            <SectionTitle>
              <FileText className="h-3.5 w-3.5" />
              {t('rules.output')}
            </SectionTitle>
            <div className="mt-2">
              <OutputSection output={rule.output} t={t} showTitle={false} />
            </div>
          </div>
        )}
      </CardContent>
      {/* Constraints Section */}
      {hasConstraints && (
        <div className="px-6 pb-6">
          <SectionTitle className="mb-2">{t('rules.constraints')}</SectionTitle>
          <div className="rounded-lg border border-[hsl(var(--border))]/50 bg-[hsl(var(--muted))]/20 p-3">
            <ConstraintsSection constraints={rule.constraints} t={t} />
          </div>
        </div>
      )}
      {/* Avoid Items - Show all items */}
      {avoidList.length > 0 && (
        <div className="px-6 pb-6">
          <SectionTitle className="mb-2">
            {t('rules.avoid')}
          </SectionTitle>
          <div className="rounded-lg border border-red-500/20 bg-red-500/5 p-3">
            <ul className="space-y-1.5">
              {avoidList.map((item, idx) => (
                <li key={idx} className="text-[hsl(var(--muted-foreground))]">
                  <span className="text-sm">{item}</span>
                </li>
              ))}
            </ul>
          </div>
        </div>
      )}
    </Card>
  )
}

/**
 * Rule overview cards component
 * Displays parsed YAML content as read-only cards
 */
export function RuleOverviewCards({ content, activeRuleId }: RuleOverviewCardsProps) {
  const { t } = useTranslation()
  const [config, setConfig] = useState<ReviewRulesConfig>({ rules: [] })
  const [parseError, setParseError] = useState<string | null>(null)
  const containerRef = useRef<HTMLDivElement>(null)

  // Parse YAML content
  useEffect(() => {
    try {
      const parsed = yaml.load(content) as ReviewRulesConfig
      if (parsed && typeof parsed === 'object') {
        setConfig({
          version: parsed.version || '1.0',
          rule_base: parsed.rule_base || {},
          rules: parsed.rules || [],
        })
        setParseError(null)
      }
    } catch (err) {
      setParseError(err instanceof Error ? err.message : 'Parse error')
    }
  }, [content])

  // Scroll to active rule when it changes
  useEffect(() => {
    if (activeRuleId && containerRef.current) {
      const elementId = `rule-card-${activeRuleId}`
      const element = document.getElementById(elementId)
      if (element) {
        element.scrollIntoView({ behavior: 'smooth', block: 'center' })
      }
    }
  }, [activeRuleId])

  // Show parse error with hint to switch to YAML mode
  if (parseError) {
    return (
      <div className="flex h-full flex-col items-center justify-center text-[hsl(var(--destructive))]">
        <p>YAML Parse Error: {parseError}</p>
        <p className="mt-2 text-sm">{t('rules.switchToYamlHint')}</p>
      </div>
    )
  }

  // Get defaults from rule_base
  const defaultAgentType = getAgentType(config.rule_base?.agent)

  return (
    <div className="space-y-6 pr-4" ref={containerRef}>
      {/* Rule Base Summary */}
      <RuleBaseCard ruleBase={config.rule_base} />

      {/* Rules List */}
      {config.rules.length === 0 ? (
        <div className="py-12 text-center text-[hsl(var(--muted-foreground))]">
          {t('rules.noRules')}
        </div>
      ) : (
        <div>
          <div className="flex items-center gap-2 mb-4 pb-2 border-b border-[hsl(var(--border))]/50">
            <h3 className="text-sm font-semibold text-[hsl(var(--foreground))] uppercase tracking-wide">
              {t('rules.ruleList')}
            </h3>
            <Badge variant="outline" className="text-xs font-normal">
              {config.rules.length}
            </Badge>
          </div>
          <div className="space-y-4">
            {config.rules.map((rule) => (
              <RuleCard
                key={rule.id}
                id={`rule-card-${rule.id}`}
                rule={rule}
                defaultAgentType={defaultAgentType}
                isActive={activeRuleId === rule.id}
              />
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
