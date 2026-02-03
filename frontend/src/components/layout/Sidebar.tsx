import { useTranslation } from 'react-i18next'
import { BarChart3, GitPullRequestDraft, FileText, TrendingUp, ShieldCheck, FolderGit2, Settings2, Bug, FileType } from 'lucide-react'
import { FaGlobe, /* FaXTwitter, */ FaGithub } from 'react-icons/fa6'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  SidebarContainer,
  SidebarHeader,
  SidebarNav,
  type NavItem,
} from '@/components/ui/sidebar-nav'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { useAppMeta } from '@/hooks/useAppMeta'

// Social media links configuration
const socialLinks = [
  { icon: FaGlobe, href: 'https://www.verustcode.com/', label: 'Website' },
  // { icon: FaXTwitter, href: 'https://x.com/example', label: 'X' }, // 暂时隐藏
  { icon: FaGithub, href: 'https://github.com/verustcode/verustcode', label: 'GitHub' },
]

interface SidebarProps {
  collapsed: boolean
  onToggle: () => void
}

/**
 * Sidebar - Application sidebar
 * Uses SidebarNav component for standardized navigation
 */
export function Sidebar({ collapsed, onToggle }: SidebarProps) {
  const { t } = useTranslation()
  const { appName, subtitle } = useAppMeta()

  // Navigation menu items configuration - using refined icons
  // Separators divide the menu into sections
  // Group structure: Dashboard | Reviews + Statistics + Findings + Repositories + Rules | Reports + Report Types | Config
  const navItems: NavItem[] = [
    { path: '/admin', icon: BarChart3, label: t('nav.dashboard'), end: true, separator: true },
    { path: '/admin/reviews', icon: GitPullRequestDraft, label: t('nav.reviews') },
    { path: '/admin/statistics', icon: TrendingUp, label: t('nav.statistics') },
    { path: '/admin/findings', icon: Bug, label: t('nav.findings') },
    { path: '/admin/repositories', icon: FolderGit2, label: t('nav.repositories') },
    { path: '/admin/rules', icon: ShieldCheck, label: t('nav.rules'), separator: true },
    { path: '/admin/reports', icon: FileText, label: t('nav.reports') },
    { path: '/admin/report-types', icon: FileType, label: t('nav.reportTypes'), separator: true },
    { path: '/admin/settings', icon: Settings2, label: t('nav.settings') },
  ]

  return (
    <SidebarContainer collapsed={collapsed}>
      {/* Header contains Logo, title, subtitle and collapse button */}
      <SidebarHeader
        collapsed={collapsed}
        title={appName}
        subtitle={subtitle}
        onToggle={onToggle}
      />

      {/* Navigation menu */}
      <ScrollArea className="flex-1 py-4">
        <SidebarNav
          items={navItems}
          collapsed={collapsed}
          className={collapsed ? 'px-3' : 'px-2'}
        />
      </ScrollArea>

      {/* Social media links */}
      <div className={`border-t border-[hsl(var(--border))] py-4 ${collapsed ? 'px-3' : 'px-4'}`}>
        <div className={`flex ${collapsed ? 'flex-col items-center gap-4' : 'flex-row justify-center gap-6'}`}>
          {socialLinks.map((link) => {
            const Icon = link.icon
            const linkElement = (
              <a
                key={link.label}
                href={link.href}
                target="_blank"
                rel="noopener noreferrer"
                className="text-[hsl(var(--muted-foreground))] hover:text-[hsl(var(--foreground))] transition-colors"
                aria-label={link.label}
              >
                <Icon className="h-5 w-5" />
              </a>
            )

            if (collapsed) {
              return (
                <Tooltip key={link.label} delayDuration={0}>
                  <TooltipTrigger asChild>{linkElement}</TooltipTrigger>
                  <TooltipContent side="right" sideOffset={10}>
                    {link.label}
                  </TooltipContent>
                </Tooltip>
              )
            }

            return linkElement
          })}
        </div>
      </div>
    </SidebarContainer>
  )
}
