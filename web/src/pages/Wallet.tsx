import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { Wallet as WalletIcon, Sparkles } from 'lucide-react'
import { api } from '../lib/api'
import { useAuth } from '../contexts/AuthContext'

interface Subscription {
  id: number; plan_name: string; quota: number; group: string
  expires_at: string; status: string; created_at: string
}

export default function Wallet() {
  const { t } = useTranslation()
  const { user } = useAuth()
  const [subs, setSubs] = useState<Subscription[]>([])

  useEffect(() => {
    api.get<{ subscriptions: Subscription[] }>('/api/subscriptions/me')
      .then((d) => setSubs(d.subscriptions ?? []))
      .catch(() => {})
  }, [])

  const quota = user?.quota ?? 0

  return (
    <div className="max-w-3xl">
      <div className="mb-6">
        <h2 className="text-xl font-semibold">{t('wallet.title')}</h2>
        <p className="text-sm text-muted-foreground mt-0.5">{t('wallet.subtitle')}</p>
      </div>

      {/* Balance card */}
      <div className="rounded-2xl border border-border bg-card p-6 mb-6 relative overflow-hidden">
        <div className="absolute -top-12 -right-12 w-40 h-40 rounded-full bg-primary/5" />
        <div className="flex items-center gap-2 text-muted-foreground text-sm mb-2">
          <WalletIcon size={15} />
          {t('wallet.balance')}
        </div>
        <div className="text-4xl font-bold tracking-tight">
          {quota === -1 ? (
            <span className="text-primary">{t('users.unlimited')}</span>
          ) : (
            <>{quota.toLocaleString()}</>
          )}
        </div>
        <p className="text-xs text-muted-foreground mt-2">
          {t('wallet.group')}: <span className="font-medium text-foreground">{user?.group ?? 'default'}</span>
        </p>
      </div>

      {/* Subscription history */}
      <h3 className="font-medium text-sm mb-3">{t('wallet.subscriptions')}</h3>
      <div className="border border-border rounded-xl overflow-hidden bg-card">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border bg-muted/50 text-left">
              <th className="px-4 py-3 font-medium text-muted-foreground">{t('plans.plan')}</th>
              <th className="px-4 py-3 font-medium text-muted-foreground">{t('plans.expires')}</th>
              <th className="px-4 py-3 font-medium text-muted-foreground">{t('users.colStatus')}</th>
            </tr>
          </thead>
          <tbody>
            {subs.length === 0 ? (
              <tr><td colSpan={3} className="px-4 py-8 text-center text-muted-foreground">{t('wallet.noSubs')}</td></tr>
            ) : subs.map((s) => (
              <tr key={s.id} className="border-b border-border last:border-0">
                <td className="px-4 py-3 font-medium flex items-center gap-2">
                  <Sparkles size={13} className="text-primary" />{s.plan_name}
                </td>
                <td className="px-4 py-3 text-xs text-muted-foreground">
                  {s.expires_at === '0001-01-01T00:00:00Z' ? t('plans.perpetual') : new Date(s.expires_at).toLocaleDateString()}
                </td>
                <td className="px-4 py-3">
                  <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${
                    s.status === 'active' ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400'
                    : s.status === 'expired' ? 'bg-amber-500/10 text-amber-600 dark:text-amber-400'
                    : 'bg-muted text-muted-foreground'}`}>{s.status}</span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
