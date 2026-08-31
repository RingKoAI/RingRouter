import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { Users, Server, Package, Activity, FileText, RefreshCw, AlertTriangle } from 'lucide-react'
import { api } from '../lib/api'

function Card({ icon: Icon, label, value, tone, loading }: {
  icon: typeof Users
  label: string
  value: number | null
  tone?: string
  loading?: boolean
}) {
  return (
    <div className="p-4 rounded-xl border border-border bg-card">
      <div className={`w-9 h-9 rounded-lg flex items-center justify-center mb-3 ${tone ?? 'bg-muted text-muted-foreground'}`}>
        <Icon size={16} />
      </div>
      {loading ? (
        <div className="h-8 w-14 rounded bg-muted animate-pulse mt-0.5" />
      ) : (
        <p className="text-2xl font-bold font-mono">{value ?? '—'}</p>
      )}
      <p className="text-xs text-muted-foreground mt-0.5">{label}</p>
    </div>
  )
}

export default function DataBoard() {
  const { t } = useTranslation()
  const [stats, setStats] = useState<Record<string, number | null>>({
    users: null, channels: null, plans: null, subs: null, logs: null,
  })
  const [loading, setLoading] = useState(true)
  const [failed, setFailed] = useState(false)

  const load = () => {
    setLoading(true); setFailed(false)
    // Each source degrades independently; a missing/unauthorized endpoint
    // shows — instead of blanking the whole board.
    const count = (d: { length?: number } | null) => (d ? (d.length ?? 0) : null)
    Promise.allSettled([
      api.get<{ pagination: { total: number } }>('/api/admin/users?page=1&page_size=1'),
      api.get<{ channels: unknown[] }>('/api/admin/channels'),
      api.get<{ plans: unknown[] }>('/api/admin/plans'),
      api.get<{ subscriptions: unknown[] }>('/api/admin/subscriptions'),
      api.get<{ logs: unknown[]; pagination: { total: number } }>('/api/admin/logs?page=1&page_size=1'),
    ]).then(([u, c, p, s, l]) => {
      setStats({
        users: u.status === 'fulfilled' ? (u.value.pagination?.total ?? 0) : null,
        channels: c.status === 'fulfilled' ? count(c.value.channels) : null,
        plans: p.status === 'fulfilled' ? count(p.value.plans) : null,
        subs: s.status === 'fulfilled' ? count(s.value.subscriptions) : null,
        logs: l.status === 'fulfilled' ? (l.value.pagination?.total ?? 0) : null,
      })
      if ([u, c, p, s, l].every((r) => r.status === 'rejected')) setFailed(true)
      setLoading(false)
    })
  }

  useEffect(load, [])

  return (
    <div className="max-w-5xl">
      <div className="flex items-start justify-between gap-3 mb-6">
        <div>
          <h2 className="text-xl font-semibold">{t('data.title')}</h2>
          <p className="text-sm text-muted-foreground mt-0.5">{t('data.subtitle')}</p>
        </div>
        <button onClick={load} className="p-2 rounded-lg border border-border hover:bg-muted transition-colors cursor-pointer" title={t('channels.refresh')}>
          <RefreshCw size={15} className={loading ? 'animate-spin' : ''} />
        </button>
      </div>

      {failed && !loading && (
        <div className="mb-4 flex items-center gap-2 p-3 rounded-lg bg-destructive/10 text-destructive text-sm">
          <AlertTriangle size={15} /> {t('data.loadFailed')}
        </div>
      )}

      <div className="grid gap-3 grid-cols-2 lg:grid-cols-5">
        <Card icon={Users} label={t('data.users')} value={stats.users} tone="bg-primary/10 text-primary" loading={loading} />
        <Card icon={Server} label={t('data.channels')} value={stats.channels} loading={loading} />
        <Card icon={Package} label={t('data.plans')} value={stats.plans} loading={loading} />
        <Card icon={Activity} label={t('data.subs')} value={stats.subs} loading={loading} />
        <Card icon={FileText} label={t('data.requests')} value={stats.logs} loading={loading} />
      </div>

      <div className="mt-6 p-5 rounded-xl border border-dashed border-border bg-muted/30 text-sm text-muted-foreground">
        {t('data.moreHint')}
      </div>
    </div>
  )
}
