import { useState } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Loader2, Info, Eye, EyeOff } from 'lucide-react'
import AuthLayout from '../components/AuthLayout'
import Turnstile from '../components/Turnstile'
import { api, APIError } from '../lib/api'
import { useAuth } from '../contexts/AuthContext'
import { useSite } from '../contexts/SiteContext'

export default function Register() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { refresh } = useAuth()
  const { turnstileEnabled, turnstileSitekey } = useSite()

  const [username, setUsername] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [confirm, setConfirm] = useState('')
  const [turnstileToken, setTurnstileToken] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const inputCls =
    'w-full min-h-[44px] px-3.5 py-2.5 border border-input rounded-xl text-sm bg-background ' +
    'focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary transition-all duration-200'

  const mismatch = confirm !== '' && password !== confirm
  const valid =
    username.trim() !== '' && email.trim() !== '' && password.length >= 8 && !mismatch

  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (mismatch) return
    setError('')
    setLoading(true)
    try {
      await api.post('/api/auth/register', {
        username,
        email,
        password,
        cf_turnstile_response: turnstileToken || undefined,
      })
      await refresh()
      navigate('/dash')
    } catch (err) {
      setError(err instanceof APIError ? err.message : t('auth.networkError'))
    } finally {
      setLoading(false)
    }
  }

  return (
    <AuthLayout>
      <div className="mb-8 anim-fade-up">
        <h1 className="text-2xl font-bold tracking-tight">{t('auth.registerTitle')}</h1>
        <p className="text-sm text-muted-foreground mt-1">{t('auth.registerDesc')}</p>
      </div>

      <div className="bg-card rounded-2xl border border-border p-6 shadow-sm anim-fade-up anim-delay-1">
        <form onSubmit={submit} className="space-y-4">
          <div className="anim-fade-up anim-delay-2">
            <label htmlFor="username" className="block text-sm font-medium mb-1.5">
              {t('auth.username')}
            </label>
            <input
              id="username"
              type="text"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              className={inputCls}
              placeholder="username"
              autoComplete="username"
              autoFocus
            />
            <p className="mt-1.5 text-xs text-muted-foreground">{t('auth.usernameHint')}</p>
          </div>

          <div className="anim-fade-up anim-delay-3">
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
            />
          </div>

          <div className="anim-fade-up anim-delay-4">
            <label htmlFor="password" className="block text-sm font-medium mb-1.5">
              {t('auth.password')}
            </label>
            <div className="relative">
              <input
                id="password"
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

          <div className="anim-fade-up anim-delay-5">
            <label htmlFor="confirm" className="block text-sm font-medium mb-1.5">
              {t('auth.confirmPassword')}
            </label>
            <input
              id="confirm"
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

          <div className="flex items-start gap-2 p-3 rounded-xl bg-muted text-xs text-muted-foreground leading-relaxed anim-fade-up anim-delay-5">
            <Info size={14} className="shrink-0 mt-0.5" />
            <span>{t('auth.firstUserAdminHint')}</span>
          </div>

          {/* Turnstile */}
          {turnstileEnabled && turnstileSitekey && (
            <div className="flex justify-center anim-fade-up anim-delay-5">
              <Turnstile
                sitekey={turnstileSitekey}
                action="register"
                onVerify={setTurnstileToken}
                onExpire={() => setTurnstileToken('')}
              />
            </div>
          )}

          {error && (
            <p className="text-sm text-destructive animate-[popIn_150ms_ease]" role="alert">
              {error}
            </p>
          )}

          <button
            type="submit"
            disabled={loading || !valid || (turnstileEnabled && !turnstileToken)}
            className="w-full min-h-[44px] py-2.5 bg-primary text-primary-foreground rounded-xl text-sm font-medium hover:bg-primary-dark disabled:opacity-50 transition-colors press cursor-pointer inline-flex items-center justify-center gap-2 anim-fade-up anim-delay-6 whitespace-nowrap"
          >
            {loading && <Loader2 size={14} className="animate-spin" />}
            {loading ? t('auth.registering') : t('auth.register')}
          </button>
        </form>

        <p className="mt-5 text-center text-sm text-muted-foreground anim-fade-in anim-delay-6">
          {t('auth.hasAccount')}{' '}
          <Link to="/auth/login" className="text-primary hover:underline cursor-pointer">
            {t('auth.signIn')}
          </Link>
        </p>
      </div>
    </AuthLayout>
  )
}
