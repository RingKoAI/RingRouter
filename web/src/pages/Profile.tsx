import { useState, useEffect, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { Fingerprint, Plus, Trash2, RefreshCw, ShieldCheck } from 'lucide-react'
import { api, APIError } from '../lib/api'
import { registerPasskey, isPasskeySupported } from '../lib/webauthn'

interface Passkey {
  id: number
  name: string
  backup_eligible: boolean
  backup_state: boolean
  last_used_at: string
  created_at: string
}

export default function Profile() {
  const { t } = useTranslation()
  const [keys, setKeys] = useState<Passkey[]>([])
  const [loading, setLoading] = useState(true)
  const [registering, setRegistering] = useState(false)
  const [name, setName] = useState('')
  const [error, setError] = useState('')
  const [info, setInfo] = useState('')
  const [passkeySupported, setPasskeySupported] = useState(false)

  useEffect(() => { isPasskeySupported().then(setPasskeySupported) }, [])

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const d = await api.get<{ passkeys: Passkey[] }>('/api/auth/passkeys')
      setKeys(d.passkeys ?? [])
    } catch {
      setKeys([])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { load() }, [load])

  const register = async () => {
    setError(''); setInfo('')
    setRegistering(true)
    try {
      await registerPasskey(name.trim())
      setName('')
      setInfo(t('profile.passkeyAdded'))
      await load()
    } catch (e) {
      const msg = e instanceof Error ? e.message : ''
      setError(msg === 'passkey-cancelled' ? t('profile.passkeyCancelled') : t('profile.passkeyFailed'))
    } finally {
      setRegistering(false)
    }
  }

  const remove = async (id: number) => {
    setError(''); setInfo('')
    try {
      await api.delete(`/api/auth/passkeys/${id}`)
      await load()
    } catch (e) {
      setError(e instanceof APIError ? e.message : t('profile.passkeyFailed'))
    }
  }

  const inputCls =
    'flex-1 min-h-[40px] px-3 py-2 border border-input rounded-lg text-sm bg-background ' +
    'focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary transition-all'

  return (
    <div className="max-w-2xl">
      <div className="mb-6">
        <h2 className="text-xl font-semibold">{t('profile.title')}</h2>
        <p className="text-sm text-muted-foreground mt-0.5">{t('profile.passkeySubtitle')}</p>
      </div>

      {/* Register new */}
      <div className="bg-card border border-border rounded-2xl p-5 mb-6">
        <div className="flex items-center gap-2 mb-3">
          <ShieldCheck size={16} className="text-primary" />
          <h3 className="font-medium text-sm">{t('profile.addPasskey')}</h3>
        </div>
        <div className="flex flex-col sm:flex-row gap-2">
          <input
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder={t('profile.passkeyName')}
            className={inputCls}
            maxLength={64}
          />
          <button
            onClick={register}
            disabled={registering || !passkeySupported}
            className="inline-flex items-center justify-center gap-2 min-h-[40px] px-4 text-sm bg-primary text-primary-foreground rounded-lg hover:bg-primary-dark disabled:opacity-50 transition-colors cursor-pointer whitespace-nowrap"
          >
            {registering ? <RefreshCw size={14} className="animate-spin" /> : <Plus size={14} />}
            {t('profile.register')}
          </button>
        </div>
        {info && <p className="text-sm text-emerald-600 dark:text-emerald-400 mt-2">{info}</p>}
        {error && <p className="text-sm text-destructive mt-2" role="alert">{error}</p>}
      </div>

      {/* List */}
      <div className="border border-border rounded-xl overflow-hidden bg-card">
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border bg-muted/50 text-left">
                <th className="px-4 py-3 font-medium text-muted-foreground">{t('profile.colName')}</th>
                <th className="px-4 py-3 font-medium text-muted-foreground">{t('profile.colSynced')}</th>
                <th className="px-4 py-3 font-medium text-muted-foreground">{t('profile.colCreated')}</th>
                <th className="px-4 py-3 font-medium text-muted-foreground text-right">{t('profile.colActions')}</th>
              </tr>
            </thead>
            <tbody>
              {loading && keys.length === 0 ? (
                <tr><td colSpan={4} className="px-4 py-8 text-center text-muted-foreground">…</td></tr>
              ) : keys.length === 0 ? (
                <tr><td colSpan={4} className="px-4 py-8 text-center text-muted-foreground">{t('profile.noPasskeys')}</td></tr>
              ) : keys.map((k) => (
                <tr key={k.id} className="border-b border-border last:border-0 hover:bg-muted/30 transition-colors">
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-2 font-medium">
                      <Fingerprint size={14} className="text-muted-foreground" />
                      {k.name}
                    </div>
                  </td>
                  <td className="px-4 py-3 text-xs">
                    {k.backup_state
                      ? <span className="text-emerald-600 dark:text-emerald-400">{t('profile.synced')}</span>
                      : <span className="text-muted-foreground">{t('profile.deviceOnly')}</span>}
                  </td>
                  <td className="px-4 py-3 text-xs text-muted-foreground whitespace-nowrap">
                    {new Date(k.created_at).toLocaleDateString()}
                  </td>
                  <td className="px-4 py-3 text-right">
                    <button
                      onClick={() => remove(k.id)}
                      title={t('profile.delete')}
                      className="p-1.5 rounded-lg hover:bg-red-500/10 text-muted-foreground hover:text-red-500 transition-colors cursor-pointer"
                    >
                      <Trash2 size={14} />
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      <p className="text-xs text-muted-foreground mt-4">{t('profile.passkeyHint')}</p>
    </div>
  )
}
