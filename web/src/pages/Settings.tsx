import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { Save, Loader2, Fingerprint, Mail } from 'lucide-react'
import { api, APIError } from '../lib/api'

interface SettingsData {
  site_name: string
  announcement: string
  usage_mode: string
  plaza_public: boolean
  smtp: { host: string; port: number; username: string; from: string; has_password: boolean; enabled: boolean }
  passkey: { enabled: boolean; rp_id: string; rp_origins: string }
}

export default function Settings() {
  const { t } = useTranslation()
  const [data, setData] = useState<SettingsData | null>(null)
  const [smtpPassword, setSmtpPassword] = useState('')
  const [saving, setSaving] = useState(false)
  const [info, setInfo] = useState('')
  const [error, setError] = useState('')

  useEffect(() => {
    api.get<SettingsData>('/api/admin/settings').then(setData).catch(() => {})
  }, [])

  const save = async () => {
    if (!data) return
    setSaving(true); setInfo(''); setError('')
    try {
      const patch: Record<string, unknown> = {
        site_name: data.site_name,
        announcement: data.announcement,
        usage_mode: data.usage_mode,
        plaza_public: data.plaza_public,
        passkey: { enabled: data.passkey.enabled, rp_id: data.passkey.rp_id, rp_origins: data.passkey.rp_origins },
      }
      if (data.smtp.host) {
        patch.smtp = {
          host: data.smtp.host, port: data.smtp.port,
          username: data.smtp.username, from: data.smtp.from,
          ...(smtpPassword ? { password: smtpPassword } : {}),
        }
      }
      await api.put('/api/admin/settings', patch)
      setInfo(t('settings.saved'))
      setSmtpPassword('')
      const fresh = await api.get<SettingsData>('/api/admin/settings')
      setData(fresh)
    } catch (e) {
      setError(e instanceof APIError ? e.message : t('users.errNetwork'))
    } finally { setSaving(false) }
  }

  if (!data) {
    return <div className="py-20 text-center text-muted-foreground text-sm">…</div>
  }

  const inputCls = 'w-full min-h-[40px] px-3 py-2 border border-input rounded-lg text-sm bg-background focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary'

  return (
    <div className="max-w-2xl space-y-6">
      <div>
        <h2 className="text-xl font-semibold">{t('settings.title')}</h2>
        <p className="text-sm text-muted-foreground mt-0.5">{t('settings.subtitle')}</p>
      </div>

      {/* Site */}
      <section className="bg-card border border-border rounded-2xl p-5 space-y-4">
        <h3 className="font-medium text-sm">{t('settings.site')}</h3>
        <div>
          <label className="block text-sm mb-1.5">{t('setup.site.name')}</label>
          <input className={inputCls} value={data.site_name} maxLength={64}
            onChange={(e) => setData({ ...data, site_name: e.target.value })} />
        </div>
        <label className="flex items-center justify-between p-3 rounded-lg bg-muted cursor-pointer">
          <div>
            <p className="text-sm font-medium">{t('settings.plazaPublic')}</p>
            <p className="text-xs text-muted-foreground">{t('settings.plazaPublicDesc')}</p>
          </div>
          <input
            type="checkbox"
            checked={data.plaza_public}
            onChange={(e) => setData({ ...data, plaza_public: e.target.checked })}
            className="w-4 h-4 accent-[var(--color-primary)] cursor-pointer"
          />
        </label>
        <div>
          <label className="block text-sm mb-1.5">{t('settings.announcement')}</label>
          <textarea className={`${inputCls} min-h-[72px] resize-y`} value={data.announcement} maxLength={1024}
            onChange={(e) => setData({ ...data, announcement: e.target.value })} />
        </div>
        <div>
          <label className="block text-sm mb-1.5">{t('settings.usageMode')}</label>
          <div className="flex gap-2">
            {(['external', 'self', 'demo'] as const).map((m) => (
              <button key={m} onClick={() => setData({ ...data, usage_mode: m })}
                className={`flex-1 min-h-[38px] text-sm rounded-lg border transition-colors cursor-pointer ${
                  data.usage_mode === m ? 'border-primary bg-primary/5 text-primary font-medium' : 'border-border hover:border-primary/30'}`}>
                {t(`setup.mode.${m}.title`)}
              </button>
            ))}
          </div>
        </div>
      </section>

      {/* SMTP */}
      <section className="bg-card border border-border rounded-2xl p-5 space-y-4">
        <div className="flex items-center gap-2">
          <Mail size={15} className="text-primary" />
          <h3 className="font-medium text-sm">{t('setup.smtp.title')}</h3>
          {data.smtp.host ? (
            <span className="text-xs text-emerald-600 dark:text-emerald-400">{t('settings.configured')}</span>
          ) : (
            <span className="text-xs text-muted-foreground">{t('settings.notConfigured')}</span>
          )}
        </div>
        <div className="grid grid-cols-[1fr_96px] gap-2">
          <div>
            <label className="block text-sm mb-1.5">{t('setup.smtp.host')}</label>
            <input className={inputCls} value={data.smtp.host} onChange={(e) => setData({ ...data, smtp: { ...data.smtp, host: e.target.value } })} placeholder="smtp.example.com" />
          </div>
          <div>
            <label className="block text-sm mb-1.5">{t('setup.smtp.port')}</label>
            <input className={inputCls} value={data.smtp.port || ''} inputMode="numeric"
              onChange={(e) => setData({ ...data, smtp: { ...data.smtp, port: parseInt(e.target.value.replace(/\D/g, '') || '0', 10) } })} />
          </div>
        </div>
        <div className="grid grid-cols-2 gap-2">
          <div>
            <label className="block text-sm mb-1.5">{t('setup.smtp.username')}</label>
            <input className={inputCls} value={data.smtp.username} onChange={(e) => setData({ ...data, smtp: { ...data.smtp, username: e.target.value } })} />
          </div>
          <div>
            <label className="block text-sm mb-1.5">{t('setup.smtp.password')}</label>
            <input type="password" className={inputCls} value={smtpPassword} onChange={(e) => setSmtpPassword(e.target.value)}
              placeholder={data.smtp.has_password ? '••••••••' : ''} />
          </div>
        </div>
        <div>
          <label className="block text-sm mb-1.5">{t('setup.smtp.from')}</label>
          <input className={inputCls} value={data.smtp.from} onChange={(e) => setData({ ...data, smtp: { ...data.smtp, from: e.target.value } })} />
        </div>
      </section>

      {/* Passkey */}
      <section className="bg-card border border-border rounded-2xl p-5 space-y-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Fingerprint size={15} className="text-primary" />
            <h3 className="font-medium text-sm">{t('setup.passkey.title')}</h3>
          </div>
          <label className="flex items-center gap-2 text-sm cursor-pointer">
            <input type="checkbox" checked={data.passkey.enabled}
              onChange={(e) => setData({ ...data, passkey: { ...data.passkey, enabled: e.target.checked } })}
              className="w-4 h-4 accent-[var(--color-primary)] cursor-pointer" />
            {t('setup.passkey.enable')}
          </label>
        </div>
        <div>
          <label className="block text-sm mb-1.5">{t('setup.passkey.rpId')}</label>
          <input className={inputCls} value={data.passkey.rp_id} disabled={!data.passkey.enabled}
            onChange={(e) => setData({ ...data, passkey: { ...data.passkey, rp_id: e.target.value } })} placeholder="example.com" />
        </div>
        <div>
          <label className="block text-sm mb-1.5">{t('setup.passkey.origins')}</label>
          <input className={inputCls} value={data.passkey.rp_origins} disabled={!data.passkey.enabled}
            onChange={(e) => setData({ ...data, passkey: { ...data.passkey, rp_origins: e.target.value } })} placeholder="https://example.com" />
          <p className="text-xs text-muted-foreground mt-1.5">{t('setup.passkey.originsHint')}</p>
        </div>
      </section>

      {info && <p className="text-sm text-emerald-600 dark:text-emerald-400">{info}</p>}
      {error && <p className="text-sm text-destructive" role="alert">{error}</p>}

      <button onClick={save} disabled={saving}
        className="inline-flex items-center gap-2 min-h-[42px] px-6 text-sm bg-primary text-primary-foreground rounded-lg hover:bg-primary-dark disabled:opacity-50 transition-colors cursor-pointer">
        {saving && <Loader2 size={14} className="animate-spin" />}
        <Save size={14} />
        {t('settings.save')}
      </button>
    </div>
  )
}
