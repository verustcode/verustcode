import { useTranslation } from 'react-i18next'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Checkbox } from '@/components/ui/checkbox'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { TagInput } from '@/components/common/TagInput'
import { FieldHelpIcon } from '@/components/common/FieldHelpIcon'
import { Separator } from '@/components/ui/separator'
import type { ReportConfig } from '@/types/report'
import { FileText, Settings, Layers, BookOpen, FileCode } from 'lucide-react'

interface ReportTypeFormProps {
  value: ReportConfig
  onChange: (value: ReportConfig) => void
  disabled?: boolean
}

/**
 * Report type form component for editing report configuration
 * Provides a user-friendly form interface for all ReportConfig fields
 */
export function ReportTypeForm({ value, onChange, disabled = false }: ReportTypeFormProps) {
  const { t } = useTranslation()

  // Helper function to update nested fields
  const updateField = (path: string[], newValue: any) => {
    // Deep clone the entire object to avoid mutation
    const updated = JSON.parse(JSON.stringify(value))
    let current: any = updated
    
    // Navigate to the parent of the target field
    for (let i = 0; i < path.length - 1; i++) {
      const key = path[i]
      if (!current[key]) {
        current[key] = {}
      }
      current = current[key]
    }
    
    // Set the final value
    const lastKey = path[path.length - 1]
    current[lastKey] = newValue
    
    onChange(updated)
  }

  // Get nested value safely
  const getNestedValue = (path: string[]): any => {
    let current: any = value
    for (const key of path) {
      if (current && typeof current === 'object') {
        current = current[key]
      } else {
        return undefined
      }
    }
    return current
  }

  return (
    <div className="space-y-6 pb-4">
      {/* Basic Information */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <FileText className="h-4 w-4" />
            {t('reportTypes.form.basicInfo')}
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          {/* ID */}
          <div className="space-y-2">
            <Label htmlFor="id" className="flex items-center gap-1">
              {t('reportTypes.form.id')}
              <span className="text-red-500">*</span>
              <FieldHelpIcon content={t('reportTypes.form.idHelp')} />
            </Label>
            <Input
              id="id"
              value={value.id || ''}
              onChange={(e) => updateField(['id'], e.target.value)}
              disabled={disabled}
              placeholder="wiki"
            />
          </div>

          {/* Name */}
          <div className="space-y-2">
            <Label htmlFor="name" className="flex items-center gap-1">
              {t('reportTypes.form.name')}
              <span className="text-red-500">*</span>
              <FieldHelpIcon content={t('reportTypes.form.nameHelp')} />
            </Label>
            <Input
              id="name"
              value={value.name || ''}
              onChange={(e) => updateField(['name'], e.target.value)}
              disabled={disabled}
              placeholder={t('reportTypes.form.namePlaceholder')}
            />
          </div>

          {/* Description */}
          <div className="space-y-2">
            <Label htmlFor="description" className="flex items-center gap-1">
              {t('reportTypes.form.description')}
              <FieldHelpIcon content={t('reportTypes.form.descriptionHelp')} />
            </Label>
            <Textarea
              id="description"
              value={value.description || ''}
              onChange={(e) => updateField(['description'], e.target.value)}
              disabled={disabled}
              placeholder={t('reportTypes.form.descriptionPlaceholder')}
              rows={3}
            />
          </div>

          {/* Version */}
          <div className="space-y-2">
            <Label htmlFor="version" className="flex items-center gap-1">
              {t('reportTypes.form.version')}
              <FieldHelpIcon content={t('reportTypes.form.versionHelp')} />
            </Label>
            <Input
              id="version"
              value={value.version || '1.0'}
              onChange={(e) => updateField(['version'], e.target.value)}
              disabled={disabled}
              placeholder="1.0"
            />
          </div>
        </CardContent>
      </Card>

      {/* Agent Configuration */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <Settings className="h-4 w-4" />
            {t('reportTypes.form.agentConfig')}
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          {/* Agent Type */}
          <div className="space-y-2">
            <Label htmlFor="agent-type" className="flex items-center gap-1">
              {t('reportTypes.form.agentType')}
              <FieldHelpIcon content={t('reportTypes.form.agentTypeHelp')} />
            </Label>
            <Select
              value={getNestedValue(['agent', 'type']) || 'cursor'}
              onValueChange={(val) => updateField(['agent', 'type'], val)}
              disabled={disabled}
            >
              <SelectTrigger id="agent-type">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="cursor">Cursor</SelectItem>
                <SelectItem value="gemini">Gemini</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {/* Agent Model */}
          <div className="space-y-2">
            <Label htmlFor="agent-model" className="flex items-center gap-1">
              {t('reportTypes.form.agentModel')}
              <FieldHelpIcon content={t('reportTypes.form.agentModelHelp')} />
            </Label>
            <Input
              id="agent-model"
              value={getNestedValue(['agent', 'model']) || ''}
              onChange={(e) => updateField(['agent', 'model'], e.target.value)}
              disabled={disabled}
              placeholder="sonnet-4.5"
            />
          </div>
        </CardContent>
      </Card>

      {/* Output Style Configuration */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <FileCode className="h-4 w-4" />
            {t('reportTypes.form.outputStyle')}
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          {/* Tone */}
          <div className="space-y-2">
            <Label htmlFor="tone" className="flex items-center gap-1">
              {t('reportTypes.form.tone')}
              <FieldHelpIcon content={t('reportTypes.form.toneHelp')} />
            </Label>
            <Select
              value={getNestedValue(['output', 'style', 'tone']) || 'professional'}
              onValueChange={(val) => updateField(['output', 'style', 'tone'], val)}
              disabled={disabled}
            >
              <SelectTrigger id="tone">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="professional">{t('reportTypes.form.toneProfessional')}</SelectItem>
                <SelectItem value="friendly">{t('reportTypes.form.toneFriendly')}</SelectItem>
                <SelectItem value="technical">{t('reportTypes.form.toneTechnical')}</SelectItem>
                <SelectItem value="strict">{t('reportTypes.form.toneStrict')}</SelectItem>
                <SelectItem value="constructive">{t('reportTypes.form.toneConstructive')}</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {/* Language */}
          <div className="space-y-2">
            <Label htmlFor="language" className="flex items-center gap-1">
              {t('reportTypes.form.language')}
              <FieldHelpIcon content={t('reportTypes.form.languageHelp')} />
            </Label>
            <Select
              value={getNestedValue(['output', 'style', 'language']) || 'zh-cn'}
              onValueChange={(val) => updateField(['output', 'style', 'language'], val)}
              disabled={disabled}
            >
              <SelectTrigger id="language">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="zh-cn">简体中文</SelectItem>
                <SelectItem value="zh-tw">繁體中文</SelectItem>
                <SelectItem value="en">English</SelectItem>
                <SelectItem value="ja">日本語</SelectItem>
                <SelectItem value="ko">한국어</SelectItem>
                <SelectItem value="fr">Français</SelectItem>
                <SelectItem value="de">Deutsch</SelectItem>
                <SelectItem value="es">Español</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {/* Heading Level */}
          <div className="space-y-2">
            <Label htmlFor="heading-level" className="flex items-center gap-1">
              {t('reportTypes.form.headingLevel')}
              <FieldHelpIcon content={t('reportTypes.form.headingLevelHelp')} />
            </Label>
            <Select
              value={String(getNestedValue(['output', 'style', 'heading_level']) || 2)}
              onValueChange={(val) => updateField(['output', 'style', 'heading_level'], parseInt(val))}
              disabled={disabled}
            >
              <SelectTrigger id="heading-level">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="1">1</SelectItem>
                <SelectItem value="2">2</SelectItem>
                <SelectItem value="3">3</SelectItem>
                <SelectItem value="4">4</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {/* Max Section Length */}
          <div className="space-y-2">
            <Label htmlFor="max-section-length" className="flex items-center gap-1">
              {t('reportTypes.form.maxSectionLength')}
              <FieldHelpIcon content={t('reportTypes.form.maxSectionLengthHelp')} />
            </Label>
            <Input
              id="max-section-length"
              type="number"
              min={500}
              value={getNestedValue(['output', 'style', 'max_section_length']) || ''}
              onChange={(e) => updateField(['output', 'style', 'max_section_length'], e.target.value ? parseInt(e.target.value) : undefined)}
              disabled={disabled}
              placeholder="5000"
            />
          </div>

          <Separator />

          {/* Boolean options */}
          <div className="space-y-3">
            <div className="flex items-center space-x-2">
              <Checkbox
                id="concise"
                checked={getNestedValue(['output', 'style', 'concise']) ?? true}
                onCheckedChange={(checked) => updateField(['output', 'style', 'concise'], checked)}
                disabled={disabled}
              />
              <Label htmlFor="concise" className="flex items-center gap-1 cursor-pointer">
                {t('reportTypes.form.concise')}
                <FieldHelpIcon content={t('reportTypes.form.conciseHelp')} />
              </Label>
            </div>

            <div className="flex items-center space-x-2">
              <Checkbox
                id="no-emoji"
                checked={getNestedValue(['output', 'style', 'no_emoji']) ?? true}
                onCheckedChange={(checked) => updateField(['output', 'style', 'no_emoji'], checked)}
                disabled={disabled}
              />
              <Label htmlFor="no-emoji" className="flex items-center gap-1 cursor-pointer">
                {t('reportTypes.form.noEmoji')}
                <FieldHelpIcon content={t('reportTypes.form.noEmojiHelp')} />
              </Label>
            </div>

            <div className="flex items-center space-x-2">
              <Checkbox
                id="use-mermaid"
                checked={getNestedValue(['output', 'style', 'use_mermaid']) ?? true}
                onCheckedChange={(checked) => updateField(['output', 'style', 'use_mermaid'], checked)}
                disabled={disabled}
              />
              <Label htmlFor="use-mermaid" className="flex items-center gap-1 cursor-pointer">
                {t('reportTypes.form.useMermaid')}
                <FieldHelpIcon content={t('reportTypes.form.useMermaidHelp')} />
              </Label>
            </div>

            <div className="flex items-center space-x-2">
              <Checkbox
                id="include-line-numbers"
                checked={getNestedValue(['output', 'style', 'include_line_numbers']) ?? true}
                onCheckedChange={(checked) => updateField(['output', 'style', 'include_line_numbers'], checked)}
                disabled={disabled}
              />
              <Label htmlFor="include-line-numbers" className="flex items-center gap-1 cursor-pointer">
                {t('reportTypes.form.includeLineNumbers')}
                <FieldHelpIcon content={t('reportTypes.form.includeLineNumbersHelp')} />
              </Label>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Phase 1: Structure Generation */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <Layers className="h-4 w-4" />
            {t('reportTypes.form.phase1Structure')}
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          {/* Description */}
          <div className="space-y-2">
            <Label htmlFor="structure-description" className="flex items-center gap-1">
              {t('reportTypes.form.description')}
              <FieldHelpIcon content={t('reportTypes.form.structureDescriptionHelp')} />
            </Label>
            <Textarea
              id="structure-description"
              value={getNestedValue(['structure', 'description']) || ''}
              onChange={(e) => updateField(['structure', 'description'], e.target.value)}
              disabled={disabled}
              placeholder={t('reportTypes.form.structureDescriptionPlaceholder')}
              rows={3}
            />
          </div>

          {/* Nested */}
          <div className="flex items-center space-x-2">
            <Checkbox
              id="structure-nested"
              checked={getNestedValue(['structure', 'nested']) ?? false}
              onCheckedChange={(checked) => updateField(['structure', 'nested'], checked)}
              disabled={disabled}
            />
            <Label htmlFor="structure-nested" className="flex items-center gap-1 cursor-pointer">
              {t('reportTypes.form.nested')}
              <FieldHelpIcon content={t('reportTypes.form.nestedHelp')} />
            </Label>
          </div>

          {/* Topics */}
          <div className="space-y-2">
            <Label className="flex items-center gap-1">
              {t('reportTypes.form.topics')}
              <FieldHelpIcon content={t('reportTypes.form.topicsHelp')} />
            </Label>
            <TagInput
              value={getNestedValue(['structure', 'goals', 'topics']) || []}
              onChange={(val) => updateField(['structure', 'goals', 'topics'], val)}
              disabled={disabled}
              placeholder={t('reportTypes.form.topicsPlaceholder')}
            />
          </div>

          {/* Avoid */}
          <div className="space-y-2">
            <Label className="flex items-center gap-1">
              {t('reportTypes.form.avoid')}
              <FieldHelpIcon content={t('reportTypes.form.avoidHelp')} />
            </Label>
            <TagInput
              value={getNestedValue(['structure', 'goals', 'avoid']) || []}
              onChange={(val) => updateField(['structure', 'goals', 'avoid'], val)}
              disabled={disabled}
              placeholder={t('reportTypes.form.avoidPlaceholder')}
            />
          </div>

          {/* Constraints */}
          <div className="space-y-2">
            <Label className="flex items-center gap-1">
              {t('reportTypes.form.constraints')}
              <FieldHelpIcon content={t('reportTypes.form.constraintsHelp')} />
            </Label>
            <TagInput
              value={getNestedValue(['structure', 'constraints']) || []}
              onChange={(val) => updateField(['structure', 'constraints'], val)}
              disabled={disabled}
              placeholder={t('reportTypes.form.constraintsPlaceholder')}
            />
          </div>

          {/* Reference Docs */}
          <div className="space-y-2">
            <Label className="flex items-center gap-1">
              <BookOpen className="h-3.5 w-3.5" />
              {t('reportTypes.form.referenceDocs')}
              <FieldHelpIcon content={t('reportTypes.form.referenceDocsHelp')} />
            </Label>
            <TagInput
              value={getNestedValue(['structure', 'reference_docs']) || []}
              onChange={(val) => updateField(['structure', 'reference_docs'], val)}
              disabled={disabled}
              placeholder={t('reportTypes.form.referenceDocsPlaceholder')}
            />
          </div>
        </CardContent>
      </Card>

      {/* Phase 2: Section Content Generation */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <FileText className="h-4 w-4" />
            {t('reportTypes.form.phase2Section')}
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          {/* Description */}
          <div className="space-y-2">
            <Label htmlFor="section-description" className="flex items-center gap-1">
              {t('reportTypes.form.description')}
              <FieldHelpIcon content={t('reportTypes.form.sectionDescriptionHelp')} />
            </Label>
            <Textarea
              id="section-description"
              value={getNestedValue(['section', 'description']) || ''}
              onChange={(e) => updateField(['section', 'description'], e.target.value)}
              disabled={disabled}
              placeholder={t('reportTypes.form.sectionDescriptionPlaceholder')}
              rows={3}
            />
          </div>

          {/* Topics */}
          <div className="space-y-2">
            <Label className="flex items-center gap-1">
              {t('reportTypes.form.topics')}
              <FieldHelpIcon content={t('reportTypes.form.topicsHelp')} />
            </Label>
            <TagInput
              value={getNestedValue(['section', 'goals', 'topics']) || []}
              onChange={(val) => updateField(['section', 'goals', 'topics'], val)}
              disabled={disabled}
              placeholder={t('reportTypes.form.topicsPlaceholder')}
            />
          </div>

          {/* Avoid */}
          <div className="space-y-2">
            <Label className="flex items-center gap-1">
              {t('reportTypes.form.avoid')}
              <FieldHelpIcon content={t('reportTypes.form.avoidHelp')} />
            </Label>
            <TagInput
              value={getNestedValue(['section', 'goals', 'avoid']) || []}
              onChange={(val) => updateField(['section', 'goals', 'avoid'], val)}
              disabled={disabled}
              placeholder={t('reportTypes.form.avoidPlaceholder')}
            />
          </div>

          {/* Constraints */}
          <div className="space-y-2">
            <Label className="flex items-center gap-1">
              {t('reportTypes.form.constraints')}
              <FieldHelpIcon content={t('reportTypes.form.constraintsHelp')} />
            </Label>
            <TagInput
              value={getNestedValue(['section', 'constraints']) || []}
              onChange={(val) => updateField(['section', 'constraints'], val)}
              disabled={disabled}
              placeholder={t('reportTypes.form.constraintsPlaceholder')}
            />
          </div>

          {/* Reference Docs */}
          <div className="space-y-2">
            <Label className="flex items-center gap-1">
              <BookOpen className="h-3.5 w-3.5" />
              {t('reportTypes.form.referenceDocs')}
              <FieldHelpIcon content={t('reportTypes.form.referenceDocsHelp')} />
            </Label>
            <TagInput
              value={getNestedValue(['section', 'reference_docs']) || []}
              onChange={(val) => updateField(['section', 'reference_docs'], val)}
              disabled={disabled}
              placeholder={t('reportTypes.form.referenceDocsPlaceholder')}
            />
          </div>

          <Separator />

          {/* Section Summary Configuration */}
          <div className="space-y-2">
            <Label htmlFor="summary-max-length" className="flex items-center gap-1">
              {t('reportTypes.form.summaryMaxLength')}
              <FieldHelpIcon content={t('reportTypes.form.summaryMaxLengthHelp')} />
            </Label>
            <Input
              id="summary-max-length"
              type="number"
              min={50}
              value={getNestedValue(['section', 'summary', 'max_length']) || 200}
              onChange={(e) => updateField(['section', 'summary', 'max_length'], parseInt(e.target.value))}
              disabled={disabled}
              placeholder="200"
            />
          </div>
        </CardContent>
      </Card>

      {/* Phase 3: Report Summary Generation */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <FileText className="h-4 w-4" />
            {t('reportTypes.form.phase3Summary')}
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          {/* Description */}
          <div className="space-y-2">
            <Label htmlFor="summary-description" className="flex items-center gap-1">
              {t('reportTypes.form.description')}
              <FieldHelpIcon content={t('reportTypes.form.summaryDescriptionHelp')} />
            </Label>
            <Textarea
              id="summary-description"
              value={getNestedValue(['summary', 'description']) || ''}
              onChange={(e) => updateField(['summary', 'description'], e.target.value)}
              disabled={disabled}
              placeholder={t('reportTypes.form.summaryDescriptionPlaceholder')}
              rows={3}
            />
          </div>

          {/* Topics */}
          <div className="space-y-2">
            <Label className="flex items-center gap-1">
              {t('reportTypes.form.topics')}
              <FieldHelpIcon content={t('reportTypes.form.topicsHelp')} />
            </Label>
            <TagInput
              value={getNestedValue(['summary', 'goals', 'topics']) || []}
              onChange={(val) => updateField(['summary', 'goals', 'topics'], val)}
              disabled={disabled}
              placeholder={t('reportTypes.form.topicsPlaceholder')}
            />
          </div>

          {/* Avoid */}
          <div className="space-y-2">
            <Label className="flex items-center gap-1">
              {t('reportTypes.form.avoid')}
              <FieldHelpIcon content={t('reportTypes.form.avoidHelp')} />
            </Label>
            <TagInput
              value={getNestedValue(['summary', 'goals', 'avoid']) || []}
              onChange={(val) => updateField(['summary', 'goals', 'avoid'], val)}
              disabled={disabled}
              placeholder={t('reportTypes.form.avoidPlaceholder')}
            />
          </div>

          {/* Constraints */}
          <div className="space-y-2">
            <Label className="flex items-center gap-1">
              {t('reportTypes.form.constraints')}
              <FieldHelpIcon content={t('reportTypes.form.constraintsHelp')} />
            </Label>
            <TagInput
              value={getNestedValue(['summary', 'constraints']) || []}
              onChange={(val) => updateField(['summary', 'constraints'], val)}
              disabled={disabled}
              placeholder={t('reportTypes.form.constraintsPlaceholder')}
            />
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
