import { useState, useEffect } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Loader2, Eye, EyeOff, Fingerprint, KeyRound, X } from 'lucide-react'
import AuthLayout from '../components/AuthLayout'
import Turnstile from '../components/Turnstile'
import { api, APIError } from '../lib/api'
import { loginWithPasskey, registerPasskey, isPasskeySupported } from '../lib/webauthn'
import { useAuth } from '../contexts/AuthContext'
import { useSite } from '../contexts/SiteContext'

export default function Login() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { refresh } = useAuth()
  const { turnstileEnabled, turnstileSitekey, smtpConfigured, passkeyEnabled } = useSite()

  const [account, setAccount] = useState('')
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [turnstileToken, setTurnstileToken] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [passkeyBusy, setPasskeyBusy] = useState(false)
  const [enrollOpen, setEnrollOpen] = useState(false)
  const [enrollAccount, setEnrollAccount] = useState('')
  const [enrollPassword, setEnrollPassword] = useState('')
  const [enrollName, setEnrollName] = useState('')
  const [enrollBusy, setEnrollBusy] = useState(false)
  const [enrollInfo, setEnrollInfo] = useState('')
  const [passkeySupported, setPasskeySupported] = useState(false)

  useEffect(() => { isPasskeySupported().then(setPasskeySupported) }, [])

  // Prefill the enrollment dialog with whatever account the user typed.
  const openEnroll = () => {
    setEnrollAccount(account)
    setEnrollPassword('')
    setEnrollName('')
    setEnrollInfo('')
    setError('')
    setEnrollOpen(true)
  }

  // Verify the password first (obtains a session), then immediately run the
  // WebAuthn registration ceremony with that session.
  const enroll = async () => {
    setError(''); setEnrollInfo(''); setEnrollBusy(true)
    try {
      await api.post('/api/auth/login', {
        account: enrollAccount.trim(),
        password: enrollPassword,
        cf_turnstile_response: turnstileToken || undefined,
      })
      await registerPasskey(enrollName.trim())
      setEnrollInfo(t('auth.passkeyEnrolled'))
      setEnrollPassword('')
    } catch (err) {
      const msg = err instanceof Error ? err.message : ''
      if (msg === 'passkey-cancelled') setError('')
      else if (err instanceof APIError) setError(err.message)
      else setError(t('auth.passkeyFailed'))
    } finally {
      setEnrollBusy(false)
    }
  }

  const inputCls =
    'w-full min-h-[44px] px-3.5 py-2.5 border border-input rounded-xl text-sm bg-background ' +
    'focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary transition-all duration-200'

  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      await api.post('/api/auth/login', {
        account,
        password,
        cf_turnstile_response: turnstileToken || undefined,
      })
      await refresh()
      navigate('/dash/overview')
    } catch (err) {
      setError(err instanceof APIError ? err.message : t('auth.networkError'))
    } finally {
      setLoading(false)
    }
  }

  const valid = account.trim() !== '' && password !== ''
  const submitDisabled = loading || !valid || (turnstileEnabled && !turnstileToken)

  const passkeyLogin = async () => {
    setError('')
    setPasskeyBusy(true)
    try {
      await loginWithPasskey(account.trim() || undefined)
      await refresh()
      navigate('/dash/overview')
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'passkey-failed'
      if (msg === 'passkey-cancelled') setError('')
      else setError(t('auth.passkeyFailed'))
    } finally {
      setPasskeyBusy(false)
    }
  }

  return (
    <AuthLayout>
      <div className="mb-8 anim-fade-up">
        <h1 className="text-2xl font-bold tracking-tight">{t('auth.signInTitle')}</h1>
        <p className="text-sm text-muted-foreground mt-1">{t('auth.signInDesc')}</p>
      </div>

      <div className="bg-card rounded-2xl border border-border p-6 shadow-sm anim-fade-up anim-delay-1">
        <form onSubmit={submit} className="space-y-4 anim-form-in">
          <div>
            <label htmlFor="account" className="block text-sm font-medium mb-1.5">
              {t('auth.account')}
            </label>
            <input
              id="account"
              type="text"
              value={account}
              onChange={(e) => setAccount(e.target.value)}
              className={inputCls}
              placeholder={t('auth.accountPlaceholder')}
              autoComplete="username"
              autoFocus
            />
          </div>
          <div>
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
                autoComplete="current-password"
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
          </div>

          {/* Turnstile */}
          {turnstileEnabled && turnstileSitekey && (
            <div className="flex justify-center">
              <Turnstile
                sitekey={turnstileSitekey}
                action="login"
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
            disabled={submitDisabled}
            className="w-full min-h-[44px] py-2.5 bg-primary text-primary-foreground rounded-xl text-sm font-medium hover:bg-primary-dark disabled:opacity-50 transition-colors press cursor-pointer inline-flex items-center justify-center gap-2 whitespace-nowrap"
          >
            {loading && <Loader2 size={14} className="animate-spin" />}
            {loading ? t('auth.signingIn') : t('auth.signIn')}
          </button>

          {/* Passkey (WebAuthn) login */}
          {passkeyEnabled && passkeySupported && (
            <>
              <div className="flex items-center gap-3 pt-1">
                <span className="h-px flex-1 bg-border" />
                <span className="text-xs text-muted-foreground">{t('auth.or')}</span>
                <span className="h-px flex-1 bg-border" />
              </div>
              <button
                type="button"
                onClick={passkeyLogin}
                disabled={passkeyBusy || loading}
                className="w-full min-h-[44px] py-2.5 border border-input rounded-xl text-sm font-medium hover:bg-muted disabled:opacity-50 transition-colors cursor-pointer inline-flex items-center justify-center gap-2 whitespace-nowrap"
              >
                {passkeyBusy ? <Loader2 size={15} className="animate-spin" /> : <Fingerprint size={15} />}
                {t('auth.passkeyLogin')}
              </button>
              <button
                type="button"
                onClick={openEnroll}
                className="w-full text-xs text-muted-foreground hover:text-primary transition-colors cursor-pointer inline-flex items-center justify-center gap-1.5 whitespace-nowrap"
              >
                <KeyRound size={12} />
                {t('auth.passkeyEnroll')}
              </button>
            </>
          )}
        </form>

        {/* Passkey enrollment dialog: verify password, then register a key */}
        {enrollOpen && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
            <div className="absolute inset-0 bg-black/50" onClick={() => !enrollBusy && setEnrollOpen(false)} />
            <div className="relative w-full max-w-sm bg-card border border-border rounded-2xl p-5 shadow-lg anim-fade-up">
              <div className="flex items-start justify-between mb-3">
                <h3 className="font-semibold">{t('auth.passkeyEnrollTitle')}</h3>
                <button onClick={() => setEnrollOpen(false)} className="p-1 rounded-lg hover:bg-muted cursor-pointer"><X size={15} /></button>
              </div>
              <p className="text-sm text-muted-foreground mb-4">{t('auth.passkeyEnrollDesc')}</p>
              <div className="space-y-3">
                <input value={enrollAccount} onChange={(e) => setEnrollAccount(e.target.value)}
                  placeholder={t('auth.account')} autoComplete="username"
                  className="w-full min-h-[42px] px-3.5 py-2.5 border border-input rounded-xl text-sm bg-background focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary" />
                <input type="password" value={enrollPassword} onChange={(e) => setEnrollPassword(e.target.value)}
                  placeholder={t('auth.password')} autoComplete="current-password"
                  className="w-full min-h-[42px] px-3.5 py-2.5 border border-input rounded-xl text-sm bg-background focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary" />
                <input value={enrollName} onChange={(e) => setEnrollName(e.target.value)}
                  placeholder={t('profile.passkeyName')} maxLength={64}
                  className="w-full min-h-[42px] px-3.5 py-2.5 border border-input rounded-xl text-sm bg-background focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary" />
                {error && enrollOpen && <p className="text-sm text-destructive" role="alert">{error}</p>}
                {enrollInfo && <p className="text-sm text-emerald-600 dark:text-emerald-400">{enrollInfo}</p>}
                <button onClick={enroll} disabled={enrollBusy || !enrollAccount.trim() || enrollPassword.length < 1}
                  className="w-full min-h-[44px] text-sm bg-primary text-primary-foreground rounded-xl hover:bg-primary-dark disabled:opacity-50 transition-colors cursor-pointer inline-flex items-center justify-center gap-2 whitespace-nowrap">
                  {enrollBusy ? <Loader2 size={14} className="animate-spin" /> : <Fingerprint size={14} />}
                  {t('auth.passkeyEnrollAction')}
                </button>
              </div>
            </div>
          </div>
        )}

        <p className="mt-5 text-center text-sm text-muted-foreground anim-fade-in">
          {t('auth.noAccount')}{' '}
          <Link to="/auth/register" className="text-primary hover:underline cursor-pointer">
            {t('auth.createAccount')}
          </Link>
        </p>
        <p className="mt-2 text-center text-xs anim-fade-in">
          <Link to="/auth/forgot" className="text-muted-foreground hover:text-foreground transition-colors cursor-pointer">
            {smtpConfigured ? t('auth.forgotPassword') : t('auth.contactAdmin')}
          </Link>
        </p>
      </div>
    </AuthLayout>
  )
}
