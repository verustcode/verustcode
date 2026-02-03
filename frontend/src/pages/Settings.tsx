import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Save, RotateCcw, AlertCircle, CheckCircle, XCircle, Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { LoadingSpinner } from '@/components/common/LoadingSpinner'
import { FieldHelpIcon } from '@/components/common/FieldHelpIcon'
import { cn } from '@/lib/utils'
import { api } from '@/lib/api'
import { toast } from '@/hooks/useToast'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Checkbox } from '@/components/ui/checkbox'
import { Card, CardContent } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { SecretInput } from '@/components/common/SecretInput'
import { TagInput } from '@/components/common/TagInput'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'

// Settings data structure
interface SettingsData {
  app?: {
    subtitle?: string
  }
  git?: {
    providers?: Array<{
      type: string
      url?: string
      token?: string
      webhook_secret?: string
      insecure_skip_verify?: boolean
    }>
  }
  agents?: Record<string, {
    cli_path?: string
    api_key?: string
    timeout?: number
    default_model?: string
    fallback_models?: string[]
    extra_args?: string
  }>
  review?: {
    workspace?: string
    max_concurrent?: number
    retention_days?: number
    max_retries?: number
    retry_delay?: number
    output_language?: string
    output_metadata?: {
      show_agent?: boolean
      show_model?: boolean
      custom_text?: string
    }
  }
  report?: {
    workspace?: string
    max_concurrent?: number
    max_retries?: number
    retry_delay?: number
    output_language?: string
  }
  notifications?: {
    channel?: string
    events?: string[]
    webhook?: { url?: string; secret?: string }
    email?: { smtp_host?: string; smtp_port?: number; username?: string; password?: string; from?: string; to?: string[] }
    slack?: { webhook_url?: string; channel?: string }
    feishu?: { webhook_url?: string; secret?: string }
  }
}

/**
 * Settings management page - editable form for runtime configuration
 */
export default function Settings() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()

  // Settings state
  const [settings, setSettings] = useState<SettingsData>({})
  const [originalSettings, setOriginalSettings] = useState<SettingsData>({})
  const [hasChanges, setHasChanges] = useState(false)

  // Git provider test states
  const [testingProvider, setTestingProvider] = useState<string | null>(null)
  const [testResult, setTestResult] = useState<Record<string, { success: boolean; message: string } | null>>({})

  // Agent test states
  const [testingAgent, setTestingAgent] = useState<string | null>(null)
  const [agentTestResult, setAgentTestResult] = useState<Record<string, { success: boolean; message: string; data?: string } | null>>({})

  // Notification test states
  const [testingNotification, setTestingNotification] = useState(false)
  const [notificationTestResult, setNotificationTestResult] = useState<{ success: boolean; message: string } | null>(null)

  // Test git provider connection
  const testGitProvider = async (providerType: string) => {
    const provider = getProvider(providerType)
    const token = provider.token
    
    if (!token) {
      setTestResult(prev => ({ ...prev, [providerType]: { success: false, message: t('settings.tokenRequired') } }))
      return
    }
    
    // For GitLab/Gitea, URL is required
    if ((providerType === 'gitlab' || providerType === 'gitea') && !provider.url) {
      setTestResult(prev => ({ ...prev, [providerType]: { success: false, message: t('settings.urlRequired') } }))
      return
    }
    
    setTestingProvider(providerType)
    setTestResult(prev => ({ ...prev, [providerType]: null }))
    
    try {
      const result = await api.admin.settings.testGitProvider({
        type: providerType,
        url: provider.url,
        token: token,
        insecure_skip_verify: provider.insecure_skip_verify,
      })
      setTestResult(prev => ({ ...prev, [providerType]: result }))
    } catch (error: unknown) {
      const errorMessage = error instanceof Error ? error.message : t('settings.testFailed')
      setTestResult(prev => ({ ...prev, [providerType]: { success: false, message: errorMessage } }))
    } finally {
      setTestingProvider(null)
    }
  }

  // Test agent connection
  const testAgent = async (agentName: string) => {
    const agent = getAgent(agentName)
    
    setTestingAgent(agentName)
    setAgentTestResult(prev => ({ ...prev, [agentName]: null }))
    
    try {
      const testParams: {
        name: string
        cli_path?: string
        api_key?: string
        default_model?: string
        fallback_models?: string[]
        timeout?: number
      } = {
        name: agentName,
        cli_path: agent.cli_path,
        api_key: agent.api_key,
        timeout: agent.timeout,
      }
      
      // Qoder doesn't support model parameters
      if (agentName !== 'qoder') {
        testParams.default_model = agent.default_model
        testParams.fallback_models = agent.fallback_models
      }
      
      const result = await api.admin.settings.testAgent(testParams)
      setAgentTestResult(prev => ({ ...prev, [agentName]: result }))
    } catch (error: unknown) {
      const errorMessage = error instanceof Error ? error.message : t('settings.testFailed')
      setAgentTestResult(prev => ({ ...prev, [agentName]: { success: false, message: errorMessage } }))
    } finally {
      setTestingAgent(null)
    }
  }

  // Test notification configuration
  const testNotification = async () => {
    const channel = settings.notifications?.channel
    if (!channel) return

    setTestingNotification(true)
    setNotificationTestResult(null)

    try {
      const webhookConfig = getNotificationConfig('webhook') as { url?: string; secret?: string }
      const emailConfig = getNotificationConfig('email') as { smtp_host?: string; smtp_port?: number; username?: string; password?: string; from?: string; to?: string[] }
      const slackConfig = getNotificationConfig('slack') as { webhook_url?: string; channel?: string }
      const feishuConfig = getNotificationConfig('feishu') as { webhook_url?: string; secret?: string }

      const result = await api.admin.settings.testNotification({
        channel,
        webhook_url: webhookConfig.url,
        webhook_secret: webhookConfig.secret,
        smtp_host: emailConfig.smtp_host,
        smtp_port: emailConfig.smtp_port,
        smtp_username: emailConfig.username,
        smtp_password: emailConfig.password,
        email_from: emailConfig.from,
        email_to: emailConfig.to,
        slack_webhook_url: slackConfig.webhook_url,
        slack_channel: slackConfig.channel,
        feishu_webhook_url: feishuConfig.webhook_url,
        feishu_secret: feishuConfig.secret,
      })
      setNotificationTestResult(result)
    } catch (error: unknown) {
      const errorMessage = error instanceof Error ? error.message : t('settings.testFailed')
      setNotificationTestResult({ success: false, message: errorMessage })
    } finally {
      setTestingNotification(false)
    }
  }

  // Check if notification config has valid required fields (including masked values)
  const hasValidNotificationConfig = (): boolean => {
    const channel = settings.notifications?.channel
    if (!channel) return false

    switch (channel) {
      case 'webhook': {
        const config = getNotificationConfig('webhook') as { url?: string }
        return !!config.url
      }
      case 'email': {
        const config = getNotificationConfig('email') as { smtp_host?: string; from?: string }
        return !!config.smtp_host && !!config.from
      }
      case 'slack': {
        const config = getNotificationConfig('slack') as { webhook_url?: string }
        return !!config.webhook_url
      }
      case 'feishu': {
        const config = getNotificationConfig('feishu') as { webhook_url?: string }
        return !!config.webhook_url
      }
      default:
        return false
    }
  }

  // Fetch settings
  const { data: settingsData, isLoading } = useQuery({
    queryKey: ['admin', 'settings'],
    queryFn: () => api.admin.settings.getAll(),
  })

  // Load settings into state
  useEffect(() => {
    if (settingsData?.settings) {
      const data = settingsData.settings as unknown as SettingsData
      setSettings(data)
      setOriginalSettings(JSON.parse(JSON.stringify(data)))
      setHasChanges(false)
    }
  }, [settingsData])

  // Apply mutation
  const applyMutation = useMutation({
    mutationFn: (data: Record<string, Record<string, unknown>>) =>
      api.admin.settings.apply(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'settings'] })
      queryClient.invalidateQueries({ queryKey: ['app', 'meta'] })
      toast({
        title: t('settings.applySuccess'),
        description: t('settings.applySuccessDesc'),
      })
      setOriginalSettings(JSON.parse(JSON.stringify(settings)))
      setHasChanges(false)
    },
    onError: () => {
      toast({
        variant: 'destructive',
        title: t('common.error'),
        description: t('settings.applyFailed'),
      })
    },
  })

  // Update settings helper
  const updateSettings = (category: keyof SettingsData, key: string, value: unknown) => {
    setSettings(prev => {
      const newSettings = { ...prev }
      if (!newSettings[category]) {
        newSettings[category] = {} as any
      }
      (newSettings[category] as any)[key] = value
      return newSettings
    })
    setHasChanges(true)
  }


  // Handle apply
  const handleApply = () => {
    applyMutation.mutate(settings as unknown as Record<string, Record<string, unknown>>)
  }

  // Handle reset
  const handleReset = () => {
    setSettings(JSON.parse(JSON.stringify(originalSettings)))
    setHasChanges(false)
  }

  // Get provider by type (with array check for corrupted data)
  const getProvider = (type: string) => {
    const providers = settings.git?.providers
    const found = Array.isArray(providers) ? providers.find(p => p.type === type) : undefined
    return found || { type }
  }

  // Update provider
  const updateProvider = (type: string, field: string, value: unknown) => {
    setSettings(prev => {
      const newSettings = { ...prev }
      if (!newSettings.git) newSettings.git = {}
      if (!newSettings.git.providers) newSettings.git.providers = []
      
      const idx = newSettings.git.providers.findIndex(p => p.type === type)
      if (idx >= 0) {
        (newSettings.git.providers[idx] as any)[field] = value
      } else {
        newSettings.git.providers.push({ type, [field]: value } as any)
      }
      return newSettings
    })
    setHasChanges(true)
  }

  // Get agent config
  const getAgent = (name: string) => {
    return settings.agents?.[name] || {}
  }

  // Check if API key exists (non-empty)
  // Masked values are handled by the server, similar to git provider tokens
  const hasValidApiKey = (agentName: string): boolean => {
    const agent = getAgent(agentName)
    const apiKey = agent.api_key
    
    // If not exists or empty, return false
    // Masked values (containing ****) are allowed - server will fetch real key from database
    return !!(apiKey && apiKey.trim() !== '')
  }

  // Update agent
  const updateAgent = (name: string, field: string, value: unknown) => {
    setSettings(prev => {
      const newSettings = { ...prev }
      if (!newSettings.agents) newSettings.agents = {}
      if (!newSettings.agents[name]) newSettings.agents[name] = {}
      ;(newSettings.agents[name] as any)[field] = value
      return newSettings
    })
    setHasChanges(true)
  }

  // Update notification channel config (e.g., notifications.webhook.url)
  const updateNotificationConfig = (channel: 'webhook' | 'email' | 'slack' | 'feishu', field: string, value: unknown) => {
    setSettings(prev => {
      const newSettings = { ...prev }
      if (!newSettings.notifications) newSettings.notifications = {}
      
      // Ensure channel config is an object, not a string
      const currentConfig = newSettings.notifications[channel]
      if (typeof currentConfig !== 'object' || currentConfig === null || Array.isArray(currentConfig)) {
        // If it's a string, try to parse it
        if (typeof currentConfig === 'string') {
          try {
            const parsed = JSON.parse(currentConfig)
            newSettings.notifications[channel] = typeof parsed === 'object' && parsed !== null && !Array.isArray(parsed) ? parsed : {}
          } catch {
            newSettings.notifications[channel] = {}
          }
        } else {
          newSettings.notifications[channel] = {}
        }
      }
      
      ;(newSettings.notifications[channel] as any)[field] = value
      return newSettings
    })
    setHasChanges(true)
  }

  // Get notification channel config
  const getNotificationConfig = (channel: 'webhook' | 'email' | 'slack' | 'feishu') => {
    const config = settings.notifications?.[channel]
    if (!config) return {}
    
    // If it's a string, try to parse it as an object
    if (typeof config === 'string') {
      try {
        const parsed = JSON.parse(config)
        return typeof parsed === 'object' && parsed !== null && !Array.isArray(parsed) ? parsed : {}
      } catch {
        return {}
      }
    }
    
    // If it's not an object, return empty object
    if (typeof config !== 'object' || config === null || Array.isArray(config)) {
      return {}
    }
    
    return config
  }

  if (isLoading) {
    return (
      <div className="flex h-[calc(100vh-8rem)] items-center justify-center">
        <LoadingSpinner size="lg" />
      </div>
    )
  }

  return (
    <div className="flex h-[calc(100vh-7rem)] flex-col">
      <div className="flex-1 overflow-auto pr-4">
        <div className="space-y-8">
          {/* App Settings */}
          <section>
            <h2 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider mb-4">
              {t('settings.app')}
            </h2>
            <Card>
              <CardContent className="p-6">
                <div className="grid gap-3">
                  <div className="grid gap-1.5">
                    <Label className="text-sm font-medium text-[hsl(var(--foreground))]">{t('config.subtitle')}</Label>
                    <Input
                      value={settings.app?.subtitle || ''}
                      onChange={(e) => updateSettings('app', 'subtitle', e.target.value)}
                      placeholder="AI Code Review"
                    />
                  </div>
                </div>
              </CardContent>
            </Card>
          </section>

          {/* Git Providers */}
          <section>
            <h2 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider mb-4">
              {t('config.gitProviders')}
            </h2>
            <Card>
              <CardContent className="p-6">
              <Tabs defaultValue="github">
                <TabsList>
                  <TabsTrigger value="github">GitHub</TabsTrigger>
                  <TabsTrigger value="gitlab">GitLab</TabsTrigger>
                  <TabsTrigger value="gitea">Gitea</TabsTrigger>
                </TabsList>

                <TabsContent value="github" className="space-y-4 pt-4">
                  <div className="grid gap-3 sm:grid-cols-2">
                    <div className="grid gap-1.5">
                      <div className="flex items-center gap-1.5">
                        <Label className="text-sm font-medium text-[hsl(var(--foreground))]">{t('config.githubToken')}<span className="text-red-500 ml-0.5">*</span></Label>
                        <FieldHelpIcon content={t('config.tokenDesc')} />
                      </div>
                      <SecretInput
                        value={getProvider('github').token || ''}
                        onChange={(value) => updateProvider('github', 'token', value)}
                        placeholder="ghp_..."
                      />
                    </div>
                    <div className="grid gap-1.5">
                      <div className="flex items-center gap-1.5">
                        <Label className="text-sm font-medium text-[hsl(var(--foreground))]">{t('config.webhookSecret')}</Label>
                        <FieldHelpIcon content={t('config.webhookSecretDesc')} />
                      </div>
                      <SecretInput
                        value={getProvider('github').webhook_secret || ''}
                        onChange={(value) => updateProvider('github', 'webhook_secret', value)}
                        placeholder="Webhook secret"
                      />
                    </div>
                  </div>
                  <div className="flex items-center gap-2 pt-2">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => testGitProvider('github')}
                      disabled={testingProvider === 'github' || !getProvider('github').token}
                    >
                      {testingProvider === 'github' ? (
                        <Loader2 className="mr-2 h-3 w-3 animate-spin" />
                      ) : null}
                      {t('settings.testConnection')}
                    </Button>
                    {testResult.github && (
                      <span className={cn(
                        'flex items-center gap-1 text-sm',
                        testResult.github.success ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400'
                      )}>
                        {testResult.github.success ? (
                          <CheckCircle className="h-4 w-4" />
                        ) : (
                          <XCircle className="h-4 w-4" />
                        )}
                        {testResult.github.message}
                      </span>
                    )}
                  </div>
                </TabsContent>

                <TabsContent value="gitlab" className="space-y-4 pt-4">
                  <div className="grid gap-1.5">
                    <Label className="text-sm font-medium text-[hsl(var(--foreground))]">{t('config.url')}<span className="text-red-500 ml-0.5">*</span></Label>
                    <Input
                      value={getProvider('gitlab').url || ''}
                      onChange={(e) => updateProvider('gitlab', 'url', e.target.value)}
                      placeholder="https://gitlab.com"
                    />
                  </div>
                  <div className="grid gap-3 sm:grid-cols-2">
                    <div className="grid gap-1.5">
                      <div className="flex items-center gap-1.5">
                        <Label className="text-sm font-medium text-[hsl(var(--foreground))]">{t('config.token')}<span className="text-red-500 ml-0.5">*</span></Label>
                        <FieldHelpIcon content={t('config.tokenDesc')} />
                      </div>
                      <SecretInput
                        value={getProvider('gitlab').token || ''}
                        onChange={(value) => updateProvider('gitlab', 'token', value)}
                        placeholder="glpat-..."
                      />
                    </div>
                    <div className="grid gap-1.5">
                      <div className="flex items-center gap-1.5">
                        <Label className="text-sm font-medium text-[hsl(var(--foreground))]">{t('config.webhookSecret')}</Label>
                        <FieldHelpIcon content={t('config.webhookSecretDesc')} />
                      </div>
                      <SecretInput
                        value={getProvider('gitlab').webhook_secret || ''}
                        onChange={(value) => updateProvider('gitlab', 'webhook_secret', value)}
                        placeholder="Webhook secret"
                      />
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <Checkbox
                      id="gitlab-insecure"
                      checked={getProvider('gitlab').insecure_skip_verify || false}
                      onCheckedChange={(checked) => updateProvider('gitlab', 'insecure_skip_verify', checked)}
                    />
                    <Label htmlFor="gitlab-insecure" className="text-sm font-medium text-[hsl(var(--foreground))]">{t('config.insecureSkipVerify')}</Label>
                  </div>
                  <div className="flex items-center gap-2 pt-2">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => testGitProvider('gitlab')}
                      disabled={testingProvider === 'gitlab' || !getProvider('gitlab').token}
                    >
                      {testingProvider === 'gitlab' ? (
                        <Loader2 className="mr-2 h-3 w-3 animate-spin" />
                      ) : null}
                      {t('settings.testConnection')}
                    </Button>
                    {testResult.gitlab && (
                      <span className={cn(
                        'flex items-center gap-1 text-sm',
                        testResult.gitlab.success ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400'
                      )}>
                        {testResult.gitlab.success ? (
                          <CheckCircle className="h-4 w-4" />
                        ) : (
                          <XCircle className="h-4 w-4" />
                        )}
                        {testResult.gitlab.message}
                      </span>
                    )}
                  </div>
                </TabsContent>

                <TabsContent value="gitea" className="space-y-4 pt-4">
                  <div className="grid gap-1.5">
                    <Label className="text-sm font-medium text-[hsl(var(--foreground))]">{t('config.url')}<span className="text-red-500 ml-0.5">*</span></Label>
                    <Input
                      value={getProvider('gitea').url || ''}
                      onChange={(e) => updateProvider('gitea', 'url', e.target.value)}
                      placeholder="https://gitea.com"
                    />
                  </div>
                  <div className="grid gap-3 sm:grid-cols-2">
                    <div className="grid gap-1.5">
                      <div className="flex items-center gap-1.5">
                        <Label className="text-sm font-medium text-[hsl(var(--foreground))]">{t('config.token')}<span className="text-red-500 ml-0.5">*</span></Label>
                        <FieldHelpIcon content={t('config.tokenDesc')} />
                      </div>
                      <SecretInput
                        value={getProvider('gitea').token || ''}
                        onChange={(value) => updateProvider('gitea', 'token', value)}
                        placeholder="Gitea token..."
                      />
                    </div>
                    <div className="grid gap-1.5">
                      <div className="flex items-center gap-1.5">
                        <Label className="text-sm font-medium text-[hsl(var(--foreground))]">{t('config.webhookSecret')}</Label>
                        <FieldHelpIcon content={t('config.webhookSecretDesc')} />
                      </div>
                      <SecretInput
                        value={getProvider('gitea').webhook_secret || ''}
                        onChange={(value) => updateProvider('gitea', 'webhook_secret', value)}
                        placeholder="Webhook secret"
                      />
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <Checkbox
                      id="gitea-insecure"
                      checked={getProvider('gitea').insecure_skip_verify || false}
                      onCheckedChange={(checked) => updateProvider('gitea', 'insecure_skip_verify', checked)}
                    />
                    <Label htmlFor="gitea-insecure" className="text-sm font-medium text-[hsl(var(--foreground))]">{t('config.insecureSkipVerify')}</Label>
                  </div>
                  <div className="flex items-center gap-2 pt-2">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => testGitProvider('gitea')}
                      disabled={testingProvider === 'gitea' || !getProvider('gitea').token}
                    >
                      {testingProvider === 'gitea' ? (
                        <Loader2 className="mr-2 h-3 w-3 animate-spin" />
                      ) : null}
                      {t('settings.testConnection')}
                    </Button>
                    {testResult.gitea && (
                      <span className={cn(
                        'flex items-center gap-1 text-sm',
                        testResult.gitea.success ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400'
                      )}>
                        {testResult.gitea.success ? (
                          <CheckCircle className="h-4 w-4" />
                        ) : (
                          <XCircle className="h-4 w-4" />
                        )}
                        {testResult.gitea.message}
                      </span>
                    )}
                  </div>
                </TabsContent>
              </Tabs>
              </CardContent>
            </Card>
          </section>

          {/* Agents */}
          <section>
            <h2 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider mb-4">
              {t('config.agents')}
            </h2>
            <Card>
              <CardContent className="p-6">
              <Tabs defaultValue="cursor">
                <TabsList>
                  <TabsTrigger value="cursor">Cursor</TabsTrigger>
                  <TabsTrigger value="gemini">Gemini</TabsTrigger>
                  <TabsTrigger value="qoder">Qoder</TabsTrigger>
                  <TabsTrigger value="mock">Mock</TabsTrigger>
                </TabsList>

                {['cursor', 'gemini', 'qoder', 'mock'].map((agent) => (
                  <TabsContent key={agent} value={agent} className="space-y-4 pt-4">
                    {agent !== 'mock' && (
                      <>
                        <div className={agent === 'qoder' ? 'grid grid-cols-1 gap-3' : 'grid grid-cols-2 gap-3'}>
                          <div className="grid gap-1.5">
                            <Label className="text-sm font-medium text-[hsl(var(--foreground))]">{t('config.cliPath')}</Label>
                            <Input
                              value={getAgent(agent).cli_path || ''}
                              onChange={(e) => updateAgent(agent, 'cli_path', e.target.value)}
                              placeholder={agent === 'cursor' ? 'cursor-agent' : agent}
                            />
                          </div>
                          {agent !== 'qoder' && (
                            <div className="grid gap-1.5">
                              <Label className="text-sm font-medium text-[hsl(var(--foreground))]">{t('config.defaultModel')}</Label>
                              <Input
                                value={getAgent(agent).default_model || ''}
                                onChange={(e) => updateAgent(agent, 'default_model', e.target.value)}
                                placeholder={agent === 'cursor' ? 'composer-1' : ''}
                              />
                            </div>
                          )}
                        </div>
                        <div className={agent === 'qoder' ? 'grid grid-cols-1 gap-3' : 'grid grid-cols-2 gap-3'}>
                          <div className="grid gap-1.5">
                            <div className="flex items-center gap-1.5">
                              <Label className="text-sm font-medium text-[hsl(var(--foreground))]">{t('config.apiKey')}<span className="text-red-500 ml-0.5">*</span></Label>
                              <FieldHelpIcon content={t('config.apiKeyDesc')} />
                            </div>
                            <SecretInput
                              value={getAgent(agent).api_key || ''}
                              onChange={(value) => updateAgent(agent, 'api_key', value)}
                              placeholder="API Key"
                            />
                          </div>
                          {agent !== 'qoder' && (
                            <div className="grid gap-1.5">
                              <Label className="text-sm font-medium text-[hsl(var(--foreground))]">{t('config.fallbackModels')}</Label>
                              <Input
                                defaultValue={(getAgent(agent).fallback_models || []).join(' ')}
                                onBlur={(e) => {
                                  const models = e.target.value.split(/\s+/).filter(Boolean)
                                  updateAgent(agent, 'fallback_models', models)
                                }}
                                placeholder={t('config.fallbackModelsPlaceholder')}
                              />
                            </div>
                          )}
                        </div>
                        <div className="grid grid-cols-2 gap-3">
                          <div className="grid gap-1.5">
                            <Label className="text-sm font-medium text-[hsl(var(--foreground))]">{t('config.timeout')}</Label>
                            <Input
                              type="number"
                              value={getAgent(agent).timeout || 600}
                              onChange={(e) => updateAgent(agent, 'timeout', parseInt(e.target.value) || 600)}
                              placeholder="600"
                            />
                          </div>
                          <div className="grid gap-1.5">
                            <Label className="text-sm font-medium text-[hsl(var(--foreground))]">{t('config.extraArgs')}</Label>
                            <Input
                              value={getAgent(agent).extra_args || ''}
                              onChange={(e) => updateAgent(agent, 'extra_args', e.target.value)}
                              placeholder="--debug --verbose"
                            />
                          </div>
                        </div>
                      </>
                    )}
                    {/* Test connection button */}
                    {agent !== 'mock' && (
                      <div className="flex items-center gap-2 pt-2">
                        <TooltipProvider>
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <span>
                                <Button
                                  variant="outline"
                                  size="sm"
                                  onClick={() => testAgent(agent)}
                                  disabled={testingAgent === agent || !hasValidApiKey(agent)}
                                >
                                  {testingAgent === agent ? (
                                    <Loader2 className="mr-2 h-3 w-3 animate-spin" />
                                  ) : null}
                                  {t('settings.testConnection')}
                                </Button>
                              </span>
                            </TooltipTrigger>
                            {!hasValidApiKey(agent) && (
                              <TooltipContent>
                                <p>{t('settings.apiKeyRequired')}</p>
                              </TooltipContent>
                            )}
                          </Tooltip>
                        </TooltipProvider>
                        {agentTestResult[agent] && (
                          <span className={cn(
                            'flex items-center gap-1 text-sm',
                            agentTestResult[agent]?.success ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400'
                          )}>
                            {agentTestResult[agent]?.success ? (
                              <CheckCircle className="h-4 w-4" />
                            ) : (
                              <XCircle className="h-4 w-4" />
                            )}
                            {agentTestResult[agent]?.message}
                          </span>
                        )}
                      </div>
                    )}
                  </TabsContent>
                ))}
              </Tabs>
              </CardContent>
            </Card>
          </section>

          {/* Review Settings */}
          <section>
            <h2 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider mb-4">
              {t('config.review')}
            </h2>
            <Card>
              <CardContent className="p-6">
              <div className="grid gap-3 sm:grid-cols-2">
                <div className="grid gap-1.5">
                  <Label className="text-sm font-medium text-[hsl(var(--foreground))]">{t('config.workspace')}</Label>
                  <Input
                    value={settings.review?.workspace || './workspace'}
                    onChange={(e) => updateSettings('review', 'workspace', e.target.value)}
                    placeholder="./workspace"
                  />
                </div>
                <div className="grid gap-1.5">
                  <Label className="text-sm font-medium text-[hsl(var(--foreground))]">{t('config.maxConcurrent')}</Label>
                  <Input
                    type="number"
                    value={settings.review?.max_concurrent || 3}
                    onChange={(e) => updateSettings('review', 'max_concurrent', parseInt(e.target.value) || 3)}
                    placeholder="3"
                  />
                </div>
                <div className="grid gap-1.5">
                  <Label className="text-sm font-medium text-[hsl(var(--foreground))]">{t('config.retentionDays')}</Label>
                  <Input
                    type="number"
                    value={settings.review?.retention_days || 30}
                    onChange={(e) => updateSettings('review', 'retention_days', parseInt(e.target.value) || 30)}
                    placeholder="30"
                  />
                </div>
                <div className="grid gap-1.5">
                  <Label className="text-sm font-medium text-[hsl(var(--foreground))]">{t('config.outputLanguage')}</Label>
                  <Select
                    value={settings.review?.output_language || 'en'}
                    onValueChange={(v) => updateSettings('review', 'output_language', v)}
                  >
                    <SelectTrigger>
                      <SelectValue placeholder="English" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="en">English</SelectItem>
                      <SelectItem value="zh-cn">中文 (简体)</SelectItem>
                      <SelectItem value="zh-tw">中文 (繁體)</SelectItem>
                      <SelectItem value="ja">日本語</SelectItem>
                      <SelectItem value="ko">한국어</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className="grid gap-1.5">
                  <Label className="text-sm font-medium text-[hsl(var(--foreground))]">{t('config.maxRetries')}</Label>
                  <Input
                    type="number"
                    value={settings.review?.max_retries || 3}
                    onChange={(e) => updateSettings('review', 'max_retries', parseInt(e.target.value) || 3)}
                    placeholder="3"
                  />
                </div>
                <div className="grid gap-1.5">
                  <Label className="text-sm font-medium text-[hsl(var(--foreground))]">{t('config.retryDelay')}</Label>
                  <Input
                    type="number"
                    value={settings.review?.retry_delay || 5}
                    onChange={(e) => updateSettings('review', 'retry_delay', parseInt(e.target.value) || 5)}
                    placeholder="5"
                  />
                </div>
              </div>
              {/* Output Metadata Settings */}
              <div className="mt-6 pt-6 border-t border-border/50">
                <Label className="text-sm font-medium text-[hsl(var(--foreground))] mb-3 block">{t('config.outputMetadata')}</Label>
                <div className="grid gap-3">
                  <div className="flex items-center gap-2">
                    <Checkbox
                      id="show_agent"
                      checked={settings.review?.output_metadata?.show_agent !== false}
                      onCheckedChange={(checked) => updateSettings('review', 'output_metadata', {
                        ...settings.review?.output_metadata,
                        show_agent: checked === true
                      })}
                    />
                    <Label htmlFor="show_agent" className="text-sm text-[hsl(var(--foreground))]">{t('config.showAgent')}</Label>
                  </div>
                  <div className="flex items-center gap-2">
                    <Checkbox
                      id="show_model"
                      checked={settings.review?.output_metadata?.show_model !== false}
                      onCheckedChange={(checked) => updateSettings('review', 'output_metadata', {
                        ...settings.review?.output_metadata,
                        show_model: checked === true
                      })}
                    />
                    <Label htmlFor="show_model" className="text-sm text-[hsl(var(--foreground))]">{t('config.showModel')}</Label>
                  </div>
                  <div className="grid gap-1.5">
                    <Label className="text-sm font-medium text-[hsl(var(--foreground))]">{t('config.customText')}</Label>
                    <Input
                      value={settings.review?.output_metadata?.custom_text || ''}
                      onChange={(e) => updateSettings('review', 'output_metadata', {
                        ...settings.review?.output_metadata,
                        custom_text: e.target.value
                      })}
                      placeholder="Generated by [VerustCode](https://github.com/verustcode/verustcode)"
                    />
                  </div>
                </div>
              </div>
              </CardContent>
            </Card>
          </section>

          {/* Report Settings */}
          <section>
            <h2 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider mb-4">
              {t('config.report')}
            </h2>
            <Card>
              <CardContent className="p-6">
              <div className="grid gap-3 sm:grid-cols-2">
                <div className="grid gap-1.5">
                  <Label className="text-sm font-medium text-[hsl(var(--foreground))]">{t('config.workspace')}</Label>
                  <Input
                    value={settings.report?.workspace || './report_workspace'}
                    onChange={(e) => updateSettings('report', 'workspace', e.target.value)}
                    placeholder="./report_workspace"
                  />
                </div>
                <div className="grid gap-1.5">
                  <Label className="text-sm font-medium text-[hsl(var(--foreground))]">{t('config.maxConcurrent')}</Label>
                  <Input
                    type="number"
                    value={settings.report?.max_concurrent || 2}
                    onChange={(e) => updateSettings('report', 'max_concurrent', parseInt(e.target.value) || 2)}
                    placeholder="2"
                  />
                </div>
                <div className="grid gap-1.5">
                  <Label className="text-sm font-medium text-[hsl(var(--foreground))]">{t('config.maxRetries')}</Label>
                  <Input
                    type="number"
                    value={settings.report?.max_retries || 3}
                    onChange={(e) => updateSettings('report', 'max_retries', parseInt(e.target.value) || 3)}
                    placeholder="3"
                  />
                </div>
                <div className="grid gap-1.5">
                  <Label className="text-sm font-medium text-[hsl(var(--foreground))]">{t('config.retryDelay')}</Label>
                  <Input
                    type="number"
                    value={settings.report?.retry_delay || 10}
                    onChange={(e) => updateSettings('report', 'retry_delay', parseInt(e.target.value) || 10)}
                    placeholder="10"
                  />
                </div>
                <div className="grid gap-1.5">
                  <Label className="text-sm font-medium text-[hsl(var(--foreground))]">{t('config.outputLanguage')}</Label>
                  <Select
                    value={settings.report?.output_language || 'en'}
                    onValueChange={(v) => updateSettings('report', 'output_language', v)}
                  >
                    <SelectTrigger>
                      <SelectValue placeholder="English" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="en">English</SelectItem>
                      <SelectItem value="zh-cn">中文 (简体)</SelectItem>
                      <SelectItem value="zh-tw">中文 (繁體)</SelectItem>
                      <SelectItem value="ja">日本語</SelectItem>
                      <SelectItem value="ko">한국어</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>
              </CardContent>
            </Card>
          </section>

          {/* Notifications */}
          <section>
            <h2 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider mb-4">
              {t('config.notifications')}
            </h2>
            <Card>
              <CardContent className="p-6">
                <div className="space-y-4">
                <div className="grid gap-1.5">
                  <Label className="text-sm font-medium text-[hsl(var(--foreground))]">{t('config.notificationChannel')}</Label>
                  <Select
                    value={settings.notifications?.channel || '__none__'}
                    onValueChange={(v) => {
                      const newChannel = v === '__none__' ? '' : v
                      updateSettings('notifications', 'channel', newChannel)
                      // Auto-select both failed events when first enabling notifications
                      if (newChannel && (!settings.notifications?.events || settings.notifications.events.length === 0)) {
                        updateSettings('notifications', 'events', ['review_failed', 'report_failed'])
                      } else if (newChannel) {
                        // Ensure failed events are always present
                        const current = settings.notifications?.events || []
                        const requiredEvents = ['review_failed', 'report_failed']
                        const missingEvents = requiredEvents.filter(e => !current.includes(e))
                        if (missingEvents.length > 0) {
                          updateSettings('notifications', 'events', [...current, ...missingEvents])
                        }
                      }
                    }}
                  >
                    <SelectTrigger>
                      <SelectValue placeholder={t('config.notificationDisabled')} />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="__none__">{t('config.notificationDisabled')}</SelectItem>
                      <SelectItem value="webhook">Webhook</SelectItem>
                      <SelectItem value="email">Email</SelectItem>
                      <SelectItem value="slack">Slack</SelectItem>
                      <SelectItem value="feishu">{t('config.feishu')}</SelectItem>
                    </SelectContent>
                  </Select>
                  {/* Webhook Config */}
                  {settings.notifications?.channel === 'webhook' && (
                    <div className="grid gap-3 sm:grid-cols-2 mt-2">
                      <div className="grid gap-1.5">
                        <Label className="text-sm font-medium text-[hsl(var(--foreground))]">{t('config.webhookUrl')}<span className="text-red-500 ml-0.5">*</span></Label>
                        <Input
                          value={(getNotificationConfig('webhook') as { url?: string }).url || ''}
                          onChange={(e) => updateNotificationConfig('webhook', 'url', e.target.value)}
                          placeholder="https://example.com/webhook"
                        />
                      </div>
                      <div className="grid gap-1.5">
                        <Label className="text-sm font-medium text-[hsl(var(--foreground))]">{t('config.webhookSecret')} ({t('config.optional')})</Label>
                        <SecretInput
                          value={(getNotificationConfig('webhook') as { secret?: string }).secret || ''}
                          onChange={(value) => updateNotificationConfig('webhook', 'secret', value)}
                          placeholder="Webhook secret"
                        />
                      </div>
                    </div>
                  )}

                  {/* Email Config */}
                  {settings.notifications?.channel === 'email' && (
                    <div className="space-y-3 mt-2">
                      <div className="grid gap-3 sm:grid-cols-2">
                        <div className="grid gap-1.5">
                          <Label className="text-sm font-medium text-[hsl(var(--foreground))]">{t('config.smtpHost')}<span className="text-red-500 ml-0.5">*</span></Label>
                          <Input
                            value={(getNotificationConfig('email') as { smtp_host?: string }).smtp_host || ''}
                            onChange={(e) => updateNotificationConfig('email', 'smtp_host', e.target.value)}
                            placeholder="smtp.example.com"
                          />
                        </div>
                        <div className="grid gap-1.5">
                          <Label className="text-sm font-medium text-[hsl(var(--foreground))]">{t('config.smtpPort')}<span className="text-red-500 ml-0.5">*</span></Label>
                          <Input
                            type="number"
                            value={(getNotificationConfig('email') as { smtp_port?: number }).smtp_port || 587}
                            onChange={(e) => updateNotificationConfig('email', 'smtp_port', parseInt(e.target.value) || 587)}
                            placeholder="587"
                          />
                        </div>
                      </div>
                      <div className="grid gap-3 sm:grid-cols-2">
                        <div className="grid gap-1.5">
                          <Label className="text-sm font-medium text-[hsl(var(--foreground))]">{t('config.smtpUsername')}</Label>
                          <Input
                            value={(getNotificationConfig('email') as { username?: string }).username || ''}
                            onChange={(e) => updateNotificationConfig('email', 'username', e.target.value)}
                            placeholder="username"
                          />
                        </div>
                        <div className="grid gap-1.5">
                          <Label className="text-sm font-medium text-[hsl(var(--foreground))]">{t('config.smtpPassword')}</Label>
                          <SecretInput
                            value={(getNotificationConfig('email') as { password?: string }).password || ''}
                            onChange={(value) => updateNotificationConfig('email', 'password', value)}
                            placeholder="password"
                          />
                        </div>
                      </div>
                      <div className="grid gap-3 sm:grid-cols-2">
                        <div className="grid gap-1.5">
                          <Label className="text-sm font-medium text-[hsl(var(--foreground))]">{t('config.emailFrom')}<span className="text-red-500 ml-0.5">*</span></Label>
                          <Input
                            value={(getNotificationConfig('email') as { from?: string }).from || ''}
                            onChange={(e) => updateNotificationConfig('email', 'from', e.target.value)}
                            placeholder="noreply@example.com"
                          />
                        </div>
                      </div>
                      <div className="grid gap-1.5">
                        <Label className="text-sm font-medium text-[hsl(var(--foreground))]">{t('config.emailTo')}</Label>
                        <TagInput
                          value={(getNotificationConfig('email') as { to?: string[] }).to || []}
                          onChange={(value) => updateNotificationConfig('email', 'to', value)}
                          placeholder={t('config.addEmail')}
                        />
                      </div>
                    </div>
                  )}

                  {/* Slack Config */}
                  {settings.notifications?.channel === 'slack' && (
                    <div className="grid gap-3 sm:grid-cols-2 mt-2">
                      <div className="grid gap-1.5">
                        <Label className="text-sm font-medium text-[hsl(var(--foreground))]">{t('config.slackWebhookUrl')}<span className="text-red-500 ml-0.5">*</span></Label>
                        <SecretInput
                          value={(getNotificationConfig('slack') as { webhook_url?: string }).webhook_url || ''}
                          onChange={(value) => updateNotificationConfig('slack', 'webhook_url', value)}
                          placeholder="https://hooks.slack.com/services/..."
                        />
                      </div>
                      <div className="grid gap-1.5">
                        <Label className="text-sm font-medium text-[hsl(var(--foreground))]">{t('config.slackChannel')} ({t('config.optional')})</Label>
                        <Input
                          value={(getNotificationConfig('slack') as { channel?: string }).channel || ''}
                          onChange={(e) => updateNotificationConfig('slack', 'channel', e.target.value)}
                          placeholder="#channel"
                        />
                      </div>
                    </div>
                  )}

                  {/* Feishu Config */}
                  {settings.notifications?.channel === 'feishu' && (
                    <div className="grid gap-3 sm:grid-cols-2 mt-2">
                      <div className="grid gap-1.5">
                        <Label className="text-sm font-medium text-[hsl(var(--foreground))]">{t('config.feishuWebhookUrl')}<span className="text-red-500 ml-0.5">*</span></Label>
                        <SecretInput
                          value={(getNotificationConfig('feishu') as { webhook_url?: string }).webhook_url || ''}
                          onChange={(value) => updateNotificationConfig('feishu', 'webhook_url', value)}
                          placeholder="https://open.feishu.cn/open-apis/bot/v2/hook/..."
                        />
                      </div>
                      <div className="grid gap-1.5">
                        <Label className="text-sm font-medium text-[hsl(var(--foreground))]">{t('config.feishuSecret')} ({t('config.optional')})</Label>
                        <SecretInput
                          value={(getNotificationConfig('feishu') as { secret?: string }).secret || ''}
                          onChange={(value) => updateNotificationConfig('feishu', 'secret', value)}
                          placeholder="Feishu sign secret"
                        />
                      </div>
                    </div>
                  )}
                </div>

                {/* Notification Events - always visible */}
                <div className="grid gap-2">
                  <Label className="text-sm font-medium text-[hsl(var(--foreground))]">{t('config.notificationEvents')}</Label>
                  <div className="grid grid-cols-2 gap-2">
                    <div className="flex items-center gap-2">
                      <Checkbox
                        id="event-review-failed"
                        checked={true}
                        disabled={true}
                      />
                      <Label htmlFor="event-review-failed" className="text-sm font-medium text-[hsl(var(--foreground))] cursor-not-allowed opacity-70">{t('config.reviewFailed')}</Label>
                    </div>
                    <div className="flex items-center gap-2">
                      <Checkbox
                        id="event-review-completed"
                        checked={settings.notifications?.events?.includes('review_completed') || false}
                        onCheckedChange={(checked) => {
                          const current = settings.notifications?.events || []
                          const newEvents = checked 
                            ? [...current.filter(e => e !== 'review_completed'), 'review_completed']
                            : current.filter(e => e !== 'review_completed')
                          updateSettings('notifications', 'events', newEvents)
                        }}
                      />
                      <Label htmlFor="event-review-completed" className="text-sm font-medium text-[hsl(var(--foreground))] cursor-pointer">{t('config.reviewCompleted')}</Label>
                    </div>
                    <div className="flex items-center gap-2">
                      <Checkbox
                        id="event-report-failed"
                        checked={true}
                        disabled={true}
                      />
                      <Label htmlFor="event-report-failed" className="text-sm font-medium text-[hsl(var(--foreground))] cursor-not-allowed opacity-70">{t('config.reportFailed')}</Label>
                    </div>
                    <div className="flex items-center gap-2">
                      <Checkbox
                        id="event-report-completed"
                        checked={settings.notifications?.events?.includes('report_completed') || false}
                        onCheckedChange={(checked) => {
                          const current = settings.notifications?.events || []
                          const newEvents = checked 
                            ? [...current.filter(e => e !== 'report_completed'), 'report_completed']
                            : current.filter(e => e !== 'report_completed')
                          updateSettings('notifications', 'events', newEvents)
                        }}
                      />
                      <Label htmlFor="event-report-completed" className="text-sm font-medium text-[hsl(var(--foreground))] cursor-pointer">{t('config.reportCompleted')}</Label>
                    </div>
                  </div>
                </div>

                {/* Test Notification Button */}
                {settings.notifications?.channel && settings.notifications.channel !== 'none' && (
                  <div className="flex items-center gap-2 pt-2">
                    <TooltipProvider>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <span>
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={testNotification}
                              disabled={testingNotification || !hasValidNotificationConfig()}
                            >
                              {testingNotification ? (
                                <Loader2 className="mr-2 h-3 w-3 animate-spin" />
                              ) : null}
                              {t('settings.testConnection')}
                            </Button>
                          </span>
                        </TooltipTrigger>
                        {!hasValidNotificationConfig() && (
                          <TooltipContent>
                            <p>{t('settings.notificationRequiredFields')}</p>
                          </TooltipContent>
                        )}
                      </Tooltip>
                    </TooltipProvider>
                    {notificationTestResult && (
                      <span className={cn(
                        'flex items-center gap-1 text-sm',
                        notificationTestResult.success ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400'
                      )}>
                        {notificationTestResult.success ? (
                          <CheckCircle className="h-4 w-4" />
                        ) : (
                          <XCircle className="h-4 w-4" />
                        )}
                        {notificationTestResult.message}
                      </span>
                    )}
                  </div>
                )}
              </div>
              </CardContent>
            </Card>
          </section>
        </div>
      </div>

      {/* Actions */}
      <div className="mt-4 flex items-center justify-between gap-3 border-t pt-4">
        {/* Left side: Status indicator */}
        <div className="flex items-center gap-2">
          {hasChanges && (
            <div className="flex items-center gap-2 text-sm text-amber-600 dark:text-amber-400">
              <AlertCircle className="h-4 w-4" />
              {t('settings.unsavedChanges')}
            </div>
          )}
        </div>

        {/* Right side: Action buttons */}
        <div className="flex items-center gap-3">
          <Button size="sm" variant="outline" onClick={handleReset} disabled={!hasChanges}>
            <RotateCcw className="mr-2 h-4 w-4" />
            {t('common.reset')}
          </Button>
          <Button size="sm" onClick={handleApply} disabled={!hasChanges || applyMutation.isPending}>
            {applyMutation.isPending ? (
              <LoadingSpinner size="sm" className="text-[hsl(var(--primary-foreground))]" />
            ) : (
              <>
                <Save className="mr-2 h-4 w-4" />
                {t('settings.apply')}
              </>
            )}
          </Button>
        </div>
      </div>
    </div>
  )
}
