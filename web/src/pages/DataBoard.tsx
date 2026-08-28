import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { Users, Server, Package, KeyRound, FileText, Activity } from 'lucide-react'
import { api } from '../lib/api'

function Card({ icon: Icon, label, value, tone }: { icon: typeof Users; label: string; value: number; tone?: string }) {
  return (
    <div className="p-4 rounded-xl border border-border bg-card">
      <div className={`w-9 h-9 rounded-lg flex items-center justify-center mb-3 ${tone ?? 'bg-muted text-muted-foreground'}`}>
        <Icon size={16} />
      </div>
      <p className="text-2xl font-bold">{value}</p>
      <p className="text-xs text-muted-foreground mt-0.5">{label}</p>
    </div>
  )
}

export default function DataBoard() {
  const { t } = useTranslation()
  const [stats, setStats] = useState({ users: 0, channels: 0, plans: 0, tokens: 0, logs: 0, subs: 0 })

  useEffect(() => {
    Promise.all([
      api.get<{ pagination: { total: number } }>('/api/admin/users?page=1&page_size=1'),
      api.get<{ channels: unknown[] }>('/api/admin/channels'),
      api.get<{ plans: unknown[] }>('/api/admin/plans'),
      api.get<{ subscriptions: unknown[] }>('/api/admin/subscriptions'),
      api.get<{ logs: unknown[]; pagination: { total: number } }>('/api/admin/logs?page=1&page_size=1'),
      api.get<{ tokens: unknown[] }>('/api/tokens').catch(() => null),
    ]).then(([u, c, p, s, l]) => {
      setStats({
        users: u.pagination?.total ?? 0,
        channels: c.channels?.length ?? 0,
        plans: p.plans?.length ?? 0,
        subs: s.subscriptions?.length ?? 0,
        logs: l.pagination?.total ?? 0,
        tokens: 0,
      })
    }).catch(() => {})
  }, [])

  return (
    <div className="max-w-5xl">
      <div className="mb-6">
        <h2 className="text-xl font-semibold">{t('data.title')}</h2>
        <p className="text-sm text-muted-foreground mt-0.5">{t('data.subtitle')}</p>
      </div>

      <div className="grid gap-3 grid-cols-2 lg:grid-cols-5">
        <Card icon={Users} label={t('data.users')} value={stats.users} tone="bg-primary/10 text-primary" />
        <Card icon={Server} label={t('data.channels')} value={stats.channels} />
        <Card icon={Package} label={t('data.plans')} value={stats.plans} />
        <Card icon={Activity} label={t('data.subs')} value={stats.subs} />
        <Card icon={FileText} label={t('data.requests')} value={stats.logs} />
      </div>

      <div className="mt-6 p-5 rounded-xl border border-dashed border-border bg-muted/30 text-sm text-muted-foreground">
        {t('data.moreHint')}
      </div>
    </div>
  )
}
