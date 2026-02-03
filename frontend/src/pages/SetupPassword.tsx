import { useState } from 'react'
import { Navigate, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useForm } from 'react-hook-form'
import { Eye, EyeOff, Lock, CheckCircle2, AlertCircle } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { LanguageSwitch } from '@/components/common/LanguageSwitch'
import { ThemeToggle } from '@/components/layout/ThemeToggle'
import { LoadingSpinner } from '@/components/common/LoadingSpinner'
import { VerustCodeLogo } from '@/components/common/VerustCodeLogo'
import { api } from '@/lib/api'
import { toast } from '@/hooks/useToast'

interface SetupPasswordForm {
  password: string
  confirmPassword: string
}

/**
 * Password setup page component for first-time admin password configuration
 */
export default function SetupPassword() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [showPassword, setShowPassword] = useState(false)
  const [showConfirmPassword, setShowConfirmPassword] = useState(false)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [needsSetup, setNeedsSetup] = useState<boolean | null>(null)

  const {
    register,
    handleSubmit,
    watch,
    formState: { errors },
  } = useForm<SetupPasswordForm>({
    defaultValues: {
      password: '',
      confirmPassword: '',
    },
  })

  // Check if setup is needed on component mount
  useState(() => {
    const checkStatus = async () => {
      try {
        const response = await api.auth.checkSetupStatus()
        setNeedsSetup(response.needs_setup)
      } catch (error: any) {
        // If API returns 404, password is already set (this is expected behavior)
        // The 404 is by design for security - to hide the setup API when not needed
        if (error?.response?.status === 404) {
          setNeedsSetup(false)
        } else {
          // For other errors, also assume setup is not needed
          setNeedsSetup(false)
        }
      }
    }
    checkStatus()
  })

  // Redirect to login if setup is not needed
  if (needsSetup === false) {
    return <Navigate to="/admin/login" replace />
  }

  // Show loading while checking status
  if (needsSetup === null) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <LoadingSpinner size="lg" />
      </div>
    )
  }

  const password = watch('password')

  // Password validation rules
  const passwordValidation = {
    minLength: password.length >= 8,
    hasUppercase: /[A-Z]/.test(password),
    hasLowercase: /[a-z]/.test(password),
    hasDigit: /\d/.test(password),
    hasSpecial: /[!@#$%^&*(),.?":{}|<>]/.test(password),
  }

  const isPasswordValid = Object.values(passwordValidation).every(Boolean)

  const onSubmit = async (data: SetupPasswordForm) => {
    // Check if passwords match
    if (data.password !== data.confirmPassword) {
      toast({
        variant: 'destructive',
        title: t('common.error'),
        description: t('setup.passwordMismatch'),
      })
      return
    }

    // Check if password meets requirements
    if (!isPasswordValid) {
      toast({
        variant: 'destructive',
        title: t('common.error'),
        description: t('setup.weakPassword'),
      })
      return
    }

    setIsSubmitting(true)
    try {
      await api.auth.setupPassword(data.password, data.confirmPassword)
      
      toast({
        title: t('common.success'),
        description: t('setup.success'),
      })

      // Redirect to login page after a short delay
      setTimeout(() => {
        navigate('/admin/login')
      }, 1500)
    } catch (error) {
      toast({
        variant: 'destructive',
        title: t('common.error'),
        description: t('setup.error'),
      })
      setIsSubmitting(false)
    }
  }

  return (
    <div className="flex min-h-screen flex-col items-center justify-center bg-[hsl(var(--background))] p-4">
      {/* Theme and language controls */}
      <div className="absolute right-4 top-3 flex items-center gap-2">
        <ThemeToggle />
        <LanguageSwitch />
      </div>

      {/* Setup card */}
      <Card className="w-full max-w-md">
        <CardHeader className="space-y-2 text-center">
          {/* Logo */}
          <div className="mb-2 flex justify-center">
            <VerustCodeLogo size="xl" />
          </div>
          <CardTitle className="text-2xl font-bold">{t('setup.title')}</CardTitle>
          <p className="text-sm text-[hsl(var(--muted-foreground))]">{t('setup.subtitle')}</p>
          <p className="text-sm font-medium text-[hsl(var(--foreground))]">{t('setup.usernameInfo')}</p>
        </CardHeader>

        <CardContent>
          <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
            {/* Password field */}
            <div className="space-y-2">
              <Label htmlFor="password" className="mb-1 block">{t('setup.password')}</Label>
              <div className="relative">
                <Input
                  id="password"
                  type={showPassword ? 'text' : 'password'}
                  placeholder={t('setup.password')}
                  autoComplete="new-password"
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

            {/* Confirm password field */}
            <div className="space-y-2">
              <Label htmlFor="confirmPassword" className="mb-1 block">{t('setup.confirmPassword')}</Label>
              <div className="relative">
                <Input
                  id="confirmPassword"
                  type={showConfirmPassword ? 'text' : 'password'}
                  placeholder={t('setup.confirmPassword')}
                  autoComplete="new-password"
                  {...register('confirmPassword', { required: true })}
                  className={errors.confirmPassword ? 'border-[hsl(var(--destructive))]' : ''}
                />
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  className="absolute right-0 top-0 h-full px-3 hover:bg-transparent"
                  onClick={() => setShowConfirmPassword(!showConfirmPassword)}
                >
                  {showConfirmPassword ? (
                    <EyeOff className="h-4 w-4 text-[hsl(var(--muted-foreground))]" />
                  ) : (
                    <Eye className="h-4 w-4 text-[hsl(var(--muted-foreground))]" />
                  )}
                </Button>
              </div>
            </div>

            {/* Password requirements */}
            <div className="rounded-md bg-[hsl(var(--muted))] p-3 text-sm">
              <div className="mb-2 font-medium text-[hsl(var(--foreground))]">
                {t('setup.requirements')}
              </div>
              <ul className="space-y-1">
                <li className="flex items-center gap-2">
                  {passwordValidation.minLength ? (
                    <CheckCircle2 className="h-4 w-4 text-green-600 dark:text-green-400" />
                  ) : (
                    <AlertCircle className="h-4 w-4 text-[hsl(var(--muted-foreground))]" />
                  )}
                  <span className={passwordValidation.minLength ? 'text-green-600 dark:text-green-400' : 'text-[hsl(var(--muted-foreground))]'}>
                    {t('setup.requirement1')}
                  </span>
                </li>
                <li className="flex items-center gap-2">
                  {passwordValidation.hasUppercase ? (
                    <CheckCircle2 className="h-4 w-4 text-green-600 dark:text-green-400" />
                  ) : (
                    <AlertCircle className="h-4 w-4 text-[hsl(var(--muted-foreground))]" />
                  )}
                  <span className={passwordValidation.hasUppercase ? 'text-green-600 dark:text-green-400' : 'text-[hsl(var(--muted-foreground))]'}>
                    {t('setup.requirement2')}
                  </span>
                </li>
                <li className="flex items-center gap-2">
                  {passwordValidation.hasLowercase ? (
                    <CheckCircle2 className="h-4 w-4 text-green-600 dark:text-green-400" />
                  ) : (
                    <AlertCircle className="h-4 w-4 text-[hsl(var(--muted-foreground))]" />
                  )}
                  <span className={passwordValidation.hasLowercase ? 'text-green-600 dark:text-green-400' : 'text-[hsl(var(--muted-foreground))]'}>
                    {t('setup.requirement3')}
                  </span>
                </li>
                <li className="flex items-center gap-2">
                  {passwordValidation.hasDigit ? (
                    <CheckCircle2 className="h-4 w-4 text-green-600 dark:text-green-400" />
                  ) : (
                    <AlertCircle className="h-4 w-4 text-[hsl(var(--muted-foreground))]" />
                  )}
                  <span className={passwordValidation.hasDigit ? 'text-green-600 dark:text-green-400' : 'text-[hsl(var(--muted-foreground))]'}>
                    {t('setup.requirement4')}
                  </span>
                </li>
                <li className="flex items-center gap-2">
                  {passwordValidation.hasSpecial ? (
                    <CheckCircle2 className="h-4 w-4 text-green-600 dark:text-green-400" />
                  ) : (
                    <AlertCircle className="h-4 w-4 text-[hsl(var(--muted-foreground))]" />
                  )}
                  <span className={passwordValidation.hasSpecial ? 'text-green-600 dark:text-green-400' : 'text-[hsl(var(--muted-foreground))]'}>
                    {t('setup.requirement5')}
                  </span>
                </li>
              </ul>
            </div>

            {/* Submit button */}
            <Button
              type="submit"
              className="w-full"
              disabled={isSubmitting || !isPasswordValid}
            >
              {isSubmitting ? (
                <>
                  <LoadingSpinner size="sm" className="mr-2 text-[hsl(var(--primary-foreground))]" />
                  {t('setup.setting')}
                </>
              ) : (
                <>
                  <Lock className="mr-2 h-4 w-4" />
                  {t('setup.submit')}
                </>
              )}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}

