import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { Boxes, RefreshCw } from 'lucide-react'
import { api } from '../lib/api'

interface ChannelLite {
  id: number; name: string; protocol: string; models: string; group: string
  status: string; priority: number; model_mapping: string
}
interface GroupInfo { id: number; name: string; ratio: number; is_default: boolean }

const protoBadge: Record<string, string> = {
  openai: 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400',
  anthropic: 'bg-orange-500/10 text-orange-600 dark:text-orange-400',
  google: 'bg-blue-500/10 text-blue-600 dark:text-blue-400',
}

export default function Models() {
  const { t } = useTranslation()
  const [channels, setChannels] = useState<ChannelLite[]>([])
  const [groups, setGroups] = useState<GroupInfo[]>([])
  const [loading, setLoading] = useState(true)
  const [q, setQ] = useState('')

  useEffect(() => {
    Promise.all([
      api.get<{ channels: ChannelLite[] }>('/api/admin/channels'),
      api.get<{ groups: GroupInfo[] }>('/api/admin/groups'),
    ]).then(([c, g]) => { setChannels(c.channels ?? []); setGroups(g.groups ?? []) })
      .catch(() => {}).finally(() => setLoading(false))
  }, [])

  const rows = channels
    .flatMap((c) => c.models.split(',').map((m) => m.trim()).filter(Boolean)
      .map((m) => ({ model: m, channel: c })))
    .filter((r) => r.model.toLowerCase().includes(q.toLowerCase()))
  const uniqueModels = new Set(rows.map((r) => r.model))

  const mapping = (c: ChannelLite, m: string): string => {
    try {
      const mm = c.model_mapping ? JSON.parse(c.model_mapping) : {}
      return mm[m] ?? ''
    } catch { return '' }
  }

  return (
    <div className="max-w-5xl">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 mb-6">
        <div>
          <h2 className="text-xl font-semibold">{t('models.title')}</h2>
          <p className="text-sm text-muted-foreground mt-0.5">
            {t('models.subtitle', { models: uniqueModels.size, channels: channels.length })}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <input value={q} onChange={(e) => setQ(e.target.value)} placeholder={t('models.search')}
            className="min-h-[38px] px-3 border border-input rounded-lg text-sm bg-background" />
          <button onClick={() => window.location.reload()} className="p-2 border border-border rounded-lg hover:bg-muted cursor-pointer">
            <RefreshCw size={15} className={loading ? 'animate-spin' : ''} />
          </button>
        </div>
      </div>

      {/* Group ratio strip */}
      <div className="flex flex-wrap gap-2 mb-5">
        {groups.map((g) => (
          <span key={g.id} className="px-2.5 py-1 rounded-full bg-muted text-xs font-medium">
            {g.name} × {g.ratio}
          </span>
        ))}
      </div>

      <div className="border border-border rounded-xl overflow-hidden bg-card">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border bg-muted/50 text-left">
              <th className="px-4 py-3 font-medium text-muted-foreground">{t('models.colModel')}</th>
              <th className="px-4 py-3 font-medium text-muted-foreground">{t('models.colChannel')}</th>
              <th className="px-4 py-3 font-medium text-muted-foreground">{t('models.colProto')}</th>
              <th className="px-4 py-3 font-medium text-muted-foreground">{t('models.colGroup')}</th>
              <th className="px-4 py-3 font-medium text-muted-foreground">{t('models.colMapping')}</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr><td colSpan={5} className="px-4 py-8 text-center text-muted-foreground">…</td></tr>
            ) : rows.length === 0 ? (
              <tr><td colSpan={5} className="px-4 py-8 text-center text-muted-foreground">{t('models.empty')}</td></tr>
            ) : rows.map((r, i) => (
              <tr key={`${r.channel.id}-${r.model}-${i}`} className="border-b border-border last:border-0 hover:bg-muted/30">
                <td className="px-4 py-3 font-medium font-mono text-xs flex items-center gap-2">
                  <Boxes size={13} className="text-muted-foreground" />{r.model}
                </td>
                <td className="px-4 py-3">{r.channel.name}</td>
                <td className="px-4 py-3">
                  <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${protoBadge[r.channel.protocol] ?? 'bg-muted'}`}>
                    {r.channel.protocol}
                  </span>
                </td>
                <td className="px-4 py-3 text-xs">{r.channel.group}</td>
                <td className="px-4 py-3 font-mono text-xs text-muted-foreground">{mapping(r.channel, r.model) || '—'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
