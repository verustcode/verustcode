import { useState, useEffect } from 'react'
import { Navigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useForm } from 'react-hook-form'
import { Eye, EyeOff, LogIn, AlertTriangle, AlertCircle } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { LanguageSwitch } from '@/components/common/LanguageSwitch'
import { ThemeToggle } from '@/components/layout/ThemeToggle'
import { LoadingSpinner } from '@/components/common/LoadingSpinner'
import { VerustCodeLogo } from '@/components/common/VerustCodeLogo'
import { useAuth } from '@/hooks/useAuth'
import { api } from '@/lib/api'

interface LoginForm {
  username: string
  password: string
}

// LocalStorage key for remember me preference
const REMEMBER_ME_KEY = 'scopeview_remember_me'

/**
 * Login page component
 */
export default function Login() {
  const { t } = useTranslation()
  const { login, loading, isAuthenticated } = useAuth()
  const [showPassword, setShowPassword] = useState(false)
  // Initialize rememberMe state from localStorage
  const [rememberMe, setRememberMe] = useState(() => {
    return localStorage.getItem(REMEMBER_ME_KEY) === 'true'
  })
  const [error, setError] = useState<string | null>(null)
  const [needsSetup, setNeedsSetup] = useState<boolean | null>(null)

  // Check if password setup is needed on component mount
  useEffect(() => {
    const checkSetupStatus = async () => {
      try {
        const response = await api.auth.checkSetupStatus()
        setNeedsSetup(response.needs_setup)
      } catch (error) {
        // If API returns 404, password is already set
        setNeedsSetup(false)
      }
    }
    checkSetupStatus()
  }, [])

  // Save rememberMe preference to localStorage when it changes
  const handleRememberMeChange = (checked: boolean) => {
    setRememberMe(checked)
    localStorage.setItem(REMEMBER_ME_KEY, String(checked))
    console.log('[Login] Remember me preference saved:', checked)
  }

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<LoginForm>({
    defaultValues: {
      username: '',
      password: '',
    },
  })

  // Redirect to setup page if password needs to be set
  if (needsSetup === true) {
    return <Navigate to="/admin/setup-password" replace />
  }

  // Redirect if already authenticated
  if (isAuthenticated) {
    return <Navigate to="/admin" replace />
  }

  // Show loading while checking setup status
  if (needsSetup === null) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <LoadingSpinner size="lg" />
      </div>
    )
  }

  const onSubmit = async (data: LoginForm) => {
    setError(null)
    try {
      await login(data.username, data.password, rememberMe)
    } catch {
      setError(t('auth.invalidCredentials'))
    }
  }

  return (
    <div className="flex min-h-screen flex-col items-center justify-center bg-[hsl(var(--background))] p-4">
      {/* Theme and language controls */}
      <div className="absolute right-4 top-3 flex items-center gap-2">
        <ThemeToggle />
        <LanguageSwitch />
      </div>

      {/* Login card */}
      <Card className="w-full max-w-md">
        <CardHeader className="space-y-2 text-center">
          {/* Logo */}
          <div className="mb-2 flex justify-center">
            <VerustCodeLogo size="xl" />
          </div>
          <CardTitle className="text-2xl font-bold">{t('auth.loginTitle')}</CardTitle>
          {/* Security warning */}
          <div className="flex items-center justify-center gap-2 rounded-md bg-amber-500/10 px-3 py-2 text-sm text-amber-600 dark:text-amber-400">
            <AlertTriangle className="h-4 w-4 flex-shrink-0" />
            <span>{t('auth.securityWarning')}</span>
          </div>
        </CardHeader>

        <CardContent>
          <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
            {/* Username field */}
            <div className="space-y-2">
              <Label htmlFor="username" className="mb-1 block">{t('auth.username')}</Label>
              <Input
                id="username"
                type="text"
                placeholder={t('auth.username')}
                autoComplete="username"
                {...register('username', { required: true })}
                className={errors.username ? 'border-[hsl(var(--destructive))]' : ''}
              />
            </div>

            {/* Password field */}
            <div className="space-y-2.5">
              <Label htmlFor="password" className="mb-1 block">{t('auth.password')}</Label>
              <div className="relative">
                <Input
                  id="password"
                  type={showPassword ? 'text' : 'password'}
                  placeholder={t('auth.password')}
                  autoComplete="current-password"
                  {...register('password', { required: true })}
                  className={errors.password ? 'border-[hsl(var(--destructive))]' : ''}
                />
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  className="absolute right-0 top-0 h-full px-3 hover:bg-transparent"
                  onClick={() => setShowPassword(!showPassword)}
                >
                  {showPassword ? (
                    <EyeOff className="h-4 w-4 text-[hsl(var(--muted-foreground))]" />
                  ) : (
                    <Eye className="h-4 w-4 text-[hsl(var(--muted-foreground))]" />
                  )}
                </Button>
              </div>
            </div>

            {/* Remember me checkbox */}
            <div className="flex items-center space-x-2">
              <Checkbox
                id="rememberMe"
                checked={rememberMe}
                onCheckedChange={(checked) => handleRememberMeChange(checked === true)}
              />
              <Label
                htmlFor="rememberMe"
                className="text-sm font-normal cursor-pointer text-[hsl(var(--muted-foreground))]"
              >
                {t('auth.rememberMe')}
              </Label>
            </div>

            {/* Error message area with fixed height to prevent layout shift */}
            <div className="h-6 flex items-center -mt-2">
              {error && (
                <div className="flex items-center gap-2 w-full rounded-md bg-[hsl(var(--destructive))]/10 px-2 py-1 text-xs text-[hsl(var(--destructive))]">
                  <AlertCircle className="h-3.5 w-3.5 flex-shrink-0" />
                  <span>{error}</span>
                </div>
              )}
            </div>

            {/* Submit button */}
            <Button
              type="submit"
              className="w-full"
              disabled={isSubmitting || loading}
            >
              {(isSubmitting || loading) ? (
                <LoadingSpinner size="sm" className="text-[hsl(var(--primary-foreground))]" />
              ) : (
                <>
                  <LogIn className="mr-2 h-4 w-4" />
                  {t('auth.login')}
                </>
              )}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}
