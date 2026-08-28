import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  Zap, User, Mail, Rocket, Loader2, Check, ArrowRight, ArrowLeft,
  Server, Users, Gift, Fingerprint,
} from 'lucide-react'
import { api, APIError } from '../lib/api'
import ThemeLangActions from '../components/ThemeLangActions'
import { useSite } from '../contexts/SiteContext'

type Step = 0 | 1 | 2 | 3

type UsageMode = 'external' | 'self' | 'demo'

interface SMTPForm {
  host: string
  port: string
  username: string
  password: string
  from: string
  testTo: string
}

const emptySMTP: SMTPForm = { host: '', port: '', username: '', password: '', from: '', testTo: '' }

const usageModes: { value: UsageMode; icon: typeof Users; titleKey: string; descKey: string }[] = [
  { value: 'external', icon: Users, titleKey: 'setup.mode.external.title', descKey: 'setup.mode.external.desc' },
  { value: 'self', icon: Server, titleKey: 'setup.mode.self.title', descKey: 'setup.mode.self.desc' },
  { value: 'demo', icon: Gift, titleKey: 'setup.mode.demo.title', descKey: 'setup.mode.demo.desc' },
]

const stepMeta = [
  { icon: User, labelKey: 'setup.steps.admin' },
  { icon: Mail, labelKey: 'setup.steps.smtp' },
  { icon: Fingerprint, labelKey: 'setup.steps.passkey' },
  { icon: Rocket, labelKey: 'setup.steps.mode' },
] as const

export default function Setup() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { siteName } = useSite()

  const [step, setStep] = useState<Step>(0)
  const [siteNameInput, setSiteNameInput] = useState('')
  const [username, setUsername] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [smtpEnabled, setSmtpEnabled] = useState(false)
  const [smtp, setSmtp] = useState<SMTPForm>(emptySMTP)
  const [passkeyEnabled, setPasskeyEnabled] = useState(false)
  const [passkeyRPID, setPasskeyRPID] = useState('')
  const [passkeyOrigins, setPasskeyOrigins] = useState('')
  const [mode, setMode] = useState<UsageMode>('external')
  const [error, setError] = useState('')
  const [info, setInfo] = useState('')
  const [testing, setTesting] = useState(false)
  const [submitting, setSubmitting] = useState(false)

  const inputCls =
    'w-full min-h-[44px] px-3.5 py-2.5 border border-input rounded-xl text-sm bg-background focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary transition-all duration-200'

  const mismatch = confirm !== '' && password !== confirm
  const step1Valid =
    /^[a-zA-Z0-9_-]{3,32}$/.test(username.trim()) &&
    /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.trim()) &&
    password.length >= 8 &&
    !mismatch

  const smtpValid =
    smtp.host.trim() !== '' && /^\d{1,5}$/.test(smtp.port) && smtp.from.trim() !== ''

  const passkeyValid =
    passkeyRPID.trim() !== '' && passkeyOrigins.trim() !== ''

  const enablePasskeyPrefill = () => {
    setPasskeyEnabled(true)
    if (!passkeyRPID) setPasskeyRPID(window.location.hostname || 'localhost')
    if (!passkeyOrigins) setPasskeyOrigins(window.location.origin || 'http://localhost:3000')
  }

  const testSMTP = async () => {
    setError('')
    setInfo('')
    setTesting(true)
    try {
      await api.post('/api/setup/test-smtp', {
        smtp: {
          host: smtp.host.trim(),
          port: parseInt(smtp.port, 10),
          username: smtp.username.trim(),
          password: smtp.password,
          from: smtp.from.trim(),
        },
        to: smtp.testTo.trim(),
      })
      setInfo(t('setup.smtp.testOk'))
    } catch (err) {
      setError(err instanceof APIError ? err.message : t('auth.networkError'))
    } finally {
      setTesting(false)
    }
  }

  const complete = async () => {
    setError('')
    setSubmitting(true)
    try {
      await api.post('/api/setup/complete', {
        username: username.trim(),
        email: email.trim(),
        password,
        site_name: siteNameInput.trim() || undefined,
        usage_mode: mode,
        smtp: smtpEnabled
          ? {
              host: smtp.host.trim(),
              port: parseInt(smtp.port, 10),
              username: smtp.username.trim(),
              password: smtp.password,
              from: smtp.from.trim(),
            }
          : undefined,
        passkey: {
          enabled: passkeyEnabled,
          rp_id: passkeyRPID.trim(),
          rp_origins: passkeyOrigins.trim(),
        },
      })
      navigate('/auth/login', { replace: true })
    } catch (err) {
      setError(err instanceof APIError ? err.message : t('auth.networkError'))
      setSubmitting(false)
    }
  }

  const next = () => {
    setError('')
    if (step === 0 && !step1Valid) {
      setError(t('setup.admin.invalid'))
      return
    }
    if (step === 1 && smtpEnabled && !smtpValid) {
      setError(t('setup.smtp.invalid'))
      return
    }
    if (step === 2 && passkeyEnabled && !passkeyValid) {
      setError(t('setup.passkey.invalid'))
      return
    }
    setStep((s) => Math.min(s + 1, 3) as Step)
  }

  const canTest =
    smtpEnabled && smtpValid && smtp.testTo.trim() !== '' && !testing

  return (
    <div className="min-h-screen bg-background flex flex-col items-center justify-center px-4 py-10 relative">
      {/* Top-right actions */}
      <div className="absolute top-4 right-4 z-10">
        <ThemeLangActions compact />
      </div>

      {/* Brand + title */}
      <div className="flex items-center gap-3 mb-3">
        <div className="w-10 h-10 rounded-xl bg-primary flex items-center justify-center shadow-sm">
          <Zap size={19} className="text-primary-foreground" strokeWidth={2.5} />
        </div>
        <div>
          <h1 className="text-xl font-bold tracking-tight leading-tight">{siteName}</h1>
          <p className="text-xs text-muted-foreground">{t('setup.subtitle')}</p>
        </div>
      </div>

      {/* Stepper */}
      <div className="flex items-center gap-0 mb-8 select-none">
        {stepMeta.map((s, i) => {
          const Icon = s.icon
          const done = i < step
          const active = i === step
          return (
            <div key={i} className="flex items-center">
              {i > 0 && (
                <div
                  className={`w-10 h-px mx-1 ${i <= step ? 'bg-primary' : 'bg-border'}`}
                />
              )}
              <div className="flex items-center gap-2">
                <div
                  className={`w-8 h-8 rounded-full flex items-center justify-center border transition-colors ${
                    active
                      ? 'bg-primary border-primary text-primary-foreground'
                      : done
                        ? 'bg-primary/10 border-primary/40 text-primary'
                        : 'bg-card border-border text-muted-foreground'
                  }`}
                >
                  {done ? <Check size={14} strokeWidth={2.5} /> : <Icon size={14} strokeWidth={2} />}
                </div>
                <span
                  className={`text-xs hidden sm:block ${
                    active ? 'text-foreground font-medium' : 'text-muted-foreground'
                  }`}
                >
                  {t(s.labelKey)}
                </span>
              </div>
            </div>
          )
        })}
      </div>

      {/* Card — keyed on step so transitions replay on navigation */}
      <div className="w-full max-w-md bg-card rounded-2xl border border-border p-6 shadow-sm anim-fade-up anim-delay-1">
        <div key={step} className="anim-form-in">
        {/* Step 1: site + admin account */}
        {step === 0 && (
          <div className="space-y-4">
            <div>
              <h2 className="font-semibold mb-1">{t('setup.admin.title')}</h2>
              <p className="text-[13px] text-muted-foreground mb-4">{t('setup.admin.desc')}</p>
            </div>
            <div>
              <label className="block text-sm font-medium mb-1.5">{t('setup.site.name')}</label>
              <input value={siteNameInput} onChange={(e) => setSiteNameInput(e.target.value)} className={inputCls} placeholder={siteName} maxLength={64} />
              <p className="mt-1.5 text-xs text-muted-foreground">{t('setup.site.nameHint')}</p>
            </div>
            <div>
              <label className="block text-sm font-medium mb-1.5">{t('auth.username')}</label>
              <input value={username} onChange={(e) => setUsername(e.target.value)} className={inputCls} placeholder="username" autoFocus />
              <p className="mt-1.5 text-xs text-muted-foreground">{t('auth.usernameHint')}</p>
            </div>
            <div>
              <label className="block text-sm font-medium mb-1.5">{t('auth.email')}</label>
              <input value={email} onChange={(e) => setEmail(e.target.value)} className={inputCls} placeholder="you@example.com" />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1.5">{t('auth.password')}</label>
              <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} className={inputCls} placeholder="••••••••" />
              <p className="mt-1.5 text-xs text-muted-foreground">{t('auth.passwordHint')}</p>
            </div>
            <div>
              <label className="block text-sm font-medium mb-1.5">{t('auth.confirmPassword')}</label>
              <input type="password" value={confirm} onChange={(e) => setConfirm(e.target.value)}
                className={`${inputCls} ${mismatch ? 'border-destructive focus:border-destructive focus:ring-destructive/30' : ''}`} placeholder="••••••••" />
              {mismatch && <p className="mt-1.5 text-xs text-destructive">{t('auth.passwordMismatch')}</p>}
            </div>
          </div>
        )}

        {/* Step 2: SMTP */}
        {step === 1 && (
          <div className="space-y-4">
            <div>
              <h2 className="font-semibold mb-1">{t('setup.smtp.title')}</h2>
              <p className="text-[13px] text-muted-foreground">{t('setup.smtp.desc')}</p>
            </div>

            <label className="flex items-center justify-between p-3 rounded-lg bg-muted cursor-pointer">
              <div>
                <p className="text-sm font-medium">{t('setup.smtp.enable')}</p>
                <p className="text-xs text-muted-foreground">{t('setup.smtp.enableDesc')}</p>
              </div>
              <input
                type="checkbox"
                checked={smtpEnabled}
                onChange={(e) => setSmtpEnabled(e.target.checked)}
                className="w-4 h-4 accent-[var(--color-primary)] cursor-pointer"
              />
            </label>

            {smtpEnabled && (
              <>
                <div className="grid grid-cols-[1fr_96px] gap-2">
                  <div>
                    <label className="block text-sm font-medium mb-1.5">{t('setup.smtp.host')}</label>
                    <input value={smtp.host} onChange={(e) => setSmtp({ ...smtp, host: e.target.value })} className={inputCls} placeholder="smtp.example.com" />
                  </div>
                  <div>
                    <label className="block text-sm font-medium mb-1.5">{t('setup.smtp.port')}</label>
                    <input value={smtp.port} onChange={(e) => setSmtp({ ...smtp, port: e.target.value.replace(/\D/g, '') })} className={inputCls} placeholder="587" inputMode="numeric" />
                  </div>
                </div>
                <div>
                  <label className="block text-sm font-medium mb-1.5">{t('setup.smtp.username')}</label>
                  <input value={smtp.username} onChange={(e) => setSmtp({ ...smtp, username: e.target.value })} className={inputCls} placeholder="postmaster@example.com" />
                </div>
                <div>
                  <label className="block text-sm font-medium mb-1.5">{t('setup.smtp.password')}</label>
                  <input type="password" value={smtp.password} onChange={(e) => setSmtp({ ...smtp, password: e.target.value })} className={inputCls} placeholder="••••••••" />
                </div>
                <div>
                  <label className="block text-sm font-medium mb-1.5">{t('setup.smtp.from')}</label>
                  <input value={smtp.from} onChange={(e) => setSmtp({ ...smtp, from: e.target.value })} className={inputCls} placeholder="ringrouter@example.com" />
                </div>
                <div>
                  <label className="block text-sm font-medium mb-1.5">{t('setup.smtp.testTo')}</label>
                  <div className="flex gap-2">
                    <input value={smtp.testTo} onChange={(e) => setSmtp({ ...smtp, testTo: e.target.value })} className={inputCls} placeholder="you@example.com" />
                    <button
                      type="button"
                      onClick={testSMTP}
                      disabled={!canTest}
                      className="shrink-0 min-h-[44px] px-3.5 py-2 border border-input rounded-xl text-[13px] text-muted-foreground hover:text-foreground hover:bg-muted transition-colors press disabled:opacity-50 cursor-pointer inline-flex items-center gap-1.5"
                    >
                      {testing && <Loader2 size={13} className="animate-spin" />}
                      {t('setup.smtp.test')}
                    </button>
                  </div>
                </div>
              </>
            )}
            {info && <p className="text-sm text-success">{info}</p>}
          </div>
        )}

        {/* Step 3: passkey */}
        {step === 2 && (
          <div className="space-y-4">
            <div>
              <h2 className="font-semibold mb-1">{t('setup.passkey.title')}</h2>
              <p className="text-[13px] text-muted-foreground">{t('setup.passkey.desc')}</p>
            </div>

            <label className="flex items-center justify-between p-3 rounded-lg bg-muted cursor-pointer">
              <div>
                <p className="text-sm font-medium">{t('setup.passkey.enable')}</p>
                <p className="text-xs text-muted-foreground">{t('setup.passkey.enableDesc')}</p>
              </div>
              <input
                type="checkbox"
                checked={passkeyEnabled}
                onChange={(e) => (e.target.checked ? enablePasskeyPrefill() : setPasskeyEnabled(false))}
                className="w-4 h-4 accent-[var(--color-primary)] cursor-pointer"
              />
            </label>

            {passkeyEnabled && (
              <>
                <div>
                  <label className="block text-sm font-medium mb-1.5">{t('setup.passkey.rpId')}</label>
                  <input value={passkeyRPID} onChange={(e) => setPasskeyRPID(e.target.value)} className={inputCls} placeholder="example.com" />
                  <p className="mt-1.5 text-xs text-muted-foreground">{t('setup.passkey.rpIdHint')}</p>
                </div>
                <div>
                  <label className="block text-sm font-medium mb-1.5">{t('setup.passkey.origins')}</label>
                  <input value={passkeyOrigins} onChange={(e) => setPasskeyOrigins(e.target.value)} className={inputCls} placeholder="https://example.com" />
                  <p className="mt-1.5 text-xs text-muted-foreground">{t('setup.passkey.originsHint')}</p>
                </div>
              </>
            )}
          </div>
        )}

        {/* Step 4: usage mode */}
        {step === 3 && (
          <div className="space-y-3">
            <div>
              <h2 className="font-semibold mb-1">{t('setup.mode.title')}</h2>
              <p className="text-[13px] text-muted-foreground">{t('setup.mode.desc')}</p>
            </div>
            {usageModes.map(({ value, icon: Icon, titleKey, descKey }) => (
              <button
                key={value}
                onClick={() => setMode(value)}
                className={`w-full flex items-start gap-3 p-4 rounded-2xl border text-left transition-all duration-200 press cursor-pointer ${
                  mode === value
                    ? 'border-primary bg-primary/5 ring-1 ring-primary/30'
                    : 'border-border hover:border-primary/30 bg-card'
                }`}
              >
                <div
                  className={`w-9 h-9 rounded-lg flex items-center justify-center shrink-0 ${
                    mode === value ? 'bg-primary/15 text-primary' : 'bg-muted text-muted-foreground'
                  }`}
                >
                  <Icon size={17} strokeWidth={2} />
                </div>
                <div className="min-w-0">
                  <p className="text-sm font-medium">{t(titleKey)}</p>
                  <p className="text-[13px] text-muted-foreground leading-relaxed mt-0.5">{t(descKey)}</p>
                </div>
                <div
                  className={`w-4 h-4 rounded-full border-2 shrink-0 mt-0.5 ml-auto ${
                    mode === value ? 'border-primary bg-primary' : 'border-border'
                  }`}
                />
              </button>
            ))}
          </div>
        )}

        {error && <p className="mt-4 text-sm text-destructive">{error}</p>}

        {/* Actions */}
        <div className="flex items-center gap-2 mt-6">
          {step > 0 && (
            <button
              onClick={() => { setStep((s) => (s - 1) as Step); setError('') }}
              className="px-4 py-2.5 min-h-[44px] border border-input rounded-xl text-sm text-muted-foreground hover:text-foreground hover:bg-muted transition-colors press cursor-pointer inline-flex items-center gap-1.5"
            >
              <ArrowLeft size={14} strokeWidth={2} />
              {t('setup.back')}
            </button>
          )}
          <div className="flex-1" />
          {step === 1 && !smtpEnabled && (
            <span className="text-xs text-muted-foreground mr-1">{t('setup.smtp.skipHint')}</span>
          )}
          {step === 2 && !passkeyEnabled && (
            <span className="text-xs text-muted-foreground mr-1">{t('setup.passkey.skipHint')}</span>
          )}
          {step < 3 ? (
            <button
              onClick={next}
              className="px-5 py-2.5 min-h-[44px] bg-primary text-primary-foreground rounded-xl text-sm font-medium hover:bg-primary-dark transition-colors press cursor-pointer inline-flex items-center gap-1.5"
            >
              {t('setup.next')}
              <ArrowRight size={14} strokeWidth={2} />
            </button>
          ) : (
            <button
              onClick={complete}
              disabled={submitting}
              className="px-5 py-2.5 min-h-[44px] bg-primary text-primary-foreground rounded-xl text-sm font-medium hover:bg-primary-dark disabled:opacity-50 transition-colors press cursor-pointer inline-flex items-center gap-1.5"
            >
              {submitting && <Loader2 size={14} className="animate-spin" />}
              {t('setup.finish')}
            </button>
          )}
        </div>
        </div>
      </div>

      <p className="mt-6 text-xs text-muted-foreground">{t('setup.footnote')}</p>
    </div>
  )
}
