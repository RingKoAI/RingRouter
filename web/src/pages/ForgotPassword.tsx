import { useState } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Mail, Loader2, ArrowLeft, Lock, Eye, EyeOff, Check } from 'lucide-react'
import AuthLayout from '../components/AuthLayout'
import { api, APIError } from '../lib/api'
import { useSite } from '../contexts/SiteContext'

type Step = 'email' | 'reset'

export default function ForgotPassword() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { smtpConfigured } = useSite()

  const [step, setStep] = useState<Step>('email')
  const [email, setEmail] = useState('')
  const [code, setCode] = useState('')
  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [sent, setSent] = useState(false)

  const inputCls =
    'w-full min-h-[44px] px-3.5 py-2.5 border border-input rounded-xl text-sm bg-background ' +
    'focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary transition-all duration-200'

  const sendCode = async () => {
    setError('')
    setLoading(true)
    try {
      await api.post('/api/auth/code', { email: email.trim() })
      setSent(true)
    } catch (err) {
      setError(err instanceof APIError ? err.message : t('auth.networkError'))
    } finally {
      setLoading(false)
    }
  }

  const resetPassword = async (e: React.FormEvent) => {
    e.preventDefault()
    if (password !== confirm) {
      setError(t('auth.passwordMismatch'))
      return
    }
    setError('')
    setLoading(true)
    try {
      await api.post('/api/auth/reset-password', {
        email: email.trim(),
        code,
        password,
      })
      navigate('/auth/login', { replace: true })
    } catch (err) {
      setError(err instanceof APIError ? err.message : t('auth.networkError'))
    } finally {
      setLoading(false)
    }
  }

  const mismatch = confirm !== '' && password !== confirm
  const resetValid = code.trim().length === 6 && password.length >= 8 && !mismatch

  /* SMTP not configured — show contact admin */
  if (!smtpConfigured) {
    return (
      <AuthLayout>
        <div className="text-center anim-fade-up">
          <div className="w-14 h-14 rounded-2xl bg-muted flex items-center justify-center mx-auto mb-4">
            <Mail size={24} className="text-muted-foreground" />
          </div>
          <h1 className="text-2xl font-bold tracking-tight mb-2">{t('auth.forgotPassword')}</h1>
          <p className="text-sm text-muted-foreground leading-relaxed mb-6">
            {t('auth.forgotNoEmail')}
          </p>
          <Link
            to="/auth/login"
            className="inline-flex items-center gap-1.5 text-sm text-primary hover:underline whitespace-nowrap"
          >
            <ArrowLeft size={14} />
            {t('auth.signIn')}
          </Link>
        </div>
      </AuthLayout>
    )
  }

  return (
    <AuthLayout>
      <div className="mb-8 anim-fade-up">
        <h1 className="text-2xl font-bold tracking-tight">{t('auth.forgotPassword')}</h1>
        <p className="text-sm text-muted-foreground mt-1">{t('auth.forgotDesc')}</p>
      </div>

      <div className="bg-card rounded-2xl border border-border p-6 shadow-sm anim-fade-up anim-delay-1">
        {step === 'email' ? (
          <div className="space-y-4 anim-form-in">
            <div>
              <label htmlFor="email" className="block text-sm font-medium mb-1.5">
                {t('auth.email')}
              </label>
              <input
                id="email"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className={inputCls}
                placeholder="you@example.com"
                autoComplete="email"
                autoFocus
              />
            </div>

            {sent && (
              <p className="text-sm text-emerald-600 dark:text-emerald-400 animate-[popIn_150ms_ease]">
                {t('auth.codeSent')}
              </p>
            )}

            {error && (
              <p className="text-sm text-destructive animate-[popIn_150ms_ease]" role="alert">
                {error}
              </p>
            )}

            <button
              onClick={() => {
                setError('')
                sendCode()
              }}
              disabled={loading || !email.trim()}
              className="w-full min-h-[44px] py-2.5 bg-primary text-primary-foreground rounded-xl text-sm font-medium hover:bg-primary-dark disabled:opacity-50 transition-colors press cursor-pointer inline-flex items-center justify-center gap-2 whitespace-nowrap"
            >
              {loading && <Loader2 size={14} className="animate-spin" />}
              {loading ? t('auth.sending') : t('auth.sendCode')}
            </button>

            {sent && (
              <button
                onClick={() => setStep('reset')}
                className="w-full min-h-[44px] py-2.5 border border-input rounded-xl text-sm font-medium hover:bg-muted transition-colors cursor-pointer whitespace-nowrap"
              >
                {t('auth.nextStep')}
              </button>
            )}
          </div>
        ) : (
          <form onSubmit={resetPassword} className="space-y-4 anim-form-in">
            <div>
              <label htmlFor="code" className="block text-sm font-medium mb-1.5">
                {t('auth.code')}
              </label>
              <input
                id="code"
                type="text"
                inputMode="numeric"
                maxLength={6}
                value={code}
                onChange={(e) => setCode(e.target.value.replace(/\D/g, ''))}
                className={`${inputCls} font-mono tracking-widest`}
                placeholder="000000"
                autoFocus
              />
            </div>
            <div>
              <label htmlFor="newPassword" className="block text-sm font-medium mb-1.5">
                {t('auth.newPassword')}
              </label>
              <div className="relative">
                <input
                  id="newPassword"
                  type={showPassword ? 'text' : 'password'}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className={`${inputCls} pr-11`}
                  placeholder="••••••••"
                  autoComplete="new-password"
                />
                <button
                  type="button"
                  onClick={() => setShowPassword(!showPassword)}
                  aria-label={showPassword ? t('auth.hidePassword') : t('auth.showPassword')}
                  className="absolute right-2 top-1/2 -translate-y-1/2 p-1.5 rounded-lg text-muted-foreground hover:text-foreground hover:bg-muted transition-colors cursor-pointer"
                >
                  {showPassword ? <EyeOff size={15} strokeWidth={2} /> : <Eye size={15} strokeWidth={2} />}
                </button>
              </div>
              <p className="mt-1.5 text-xs text-muted-foreground">{t('auth.passwordHint')}</p>
            </div>
            <div>
              <label htmlFor="confirmNew" className="block text-sm font-medium mb-1.5">
                {t('auth.confirmPassword')}
              </label>
              <input
                id="confirmNew"
                type="password"
                value={confirm}
                onChange={(e) => setConfirm(e.target.value)}
                className={`${inputCls} ${mismatch ? 'border-destructive focus:border-destructive focus:ring-destructive/30' : ''}`}
                placeholder="••••••••"
                autoComplete="new-password"
              />
              {mismatch && (
                <p className="mt-1.5 text-xs text-destructive animate-[popIn_150ms_ease]">
                  {t('auth.passwordMismatch')}
                </p>
              )}
            </div>

            {error && (
              <p className="text-sm text-destructive animate-[popIn_150ms_ease]" role="alert">
                {error}
              </p>
            )}

            <button
              type="submit"
              disabled={loading || !resetValid}
              className="w-full min-h-[44px] py-2.5 bg-primary text-primary-foreground rounded-xl text-sm font-medium hover:bg-primary-dark disabled:opacity-50 transition-colors press cursor-pointer inline-flex items-center justify-center gap-2 whitespace-nowrap"
            >
              {loading && <Loader2 size={14} className="animate-spin" />}
              {loading ? t('auth.resetting') : t('auth.resetPassword')}
            </button>
          </form>
        )}

        <p className="mt-5 text-center text-sm text-muted-foreground anim-fade-in">
          <Link to="/auth/login" className="inline-flex items-center gap-1 text-primary hover:underline cursor-pointer">
            <ArrowLeft size={12} />
            {t('auth.backToLogin')}
          </Link>
        </p>
      </div>
    </AuthLayout>
  )
}
