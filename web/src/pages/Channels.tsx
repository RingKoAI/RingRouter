import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Server, Plus, Search, MoreHorizontal, Pencil, Trash2,
  Power, RefreshCw, Globe, Key, Layers, TrendingUp, XCircle, CheckCircle2,
} from 'lucide-react'

interface Channel {
  id: number
  name: string
  type: string
  base_url: string
  models: string
  status: 'active' | 'disabled'
  priority: number
  modelCount?: number
}

const typeBadge: Record<string, { label: string; className: string }> = {
  openai: { label: 'OpenAI', className: 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400' },
  anthropic: { label: 'Claude', className: 'bg-orange-500/10 text-orange-600 dark:text-orange-400' },
  google: { label: 'Gemini', className: 'bg-blue-500/10 text-blue-600 dark:text-blue-400' },
}

export default function Channels() {
  const { t } = useTranslation()
  const [channels, setChannels] = useState<Channel[]>([])
  const [query, setQuery] = useState('')
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    // TODO: replace with real API
    const mock: Channel[] = [
      { id: 1, name: 'OpenAI 主力', type: 'openai', base_url: 'https://api.openai.com', models: 'gpt-4o,gpt-4o-mini,o1', status: 'active', priority: 100, modelCount: 3 },
      { id: 2, name: 'Claude 备用', type: 'anthropic', base_url: 'https://api.anthropic.com', models: 'claude-sonnet-4-5,claude-opus-4-1', status: 'active', priority: 50, modelCount: 2 },
      { id: 3, name: 'Gemini 实验', type: 'google', base_url: 'https://generativelanguage.googleapis.com', models: 'gemini-2.0-flash', status: 'disabled', priority: 10, modelCount: 1 },
    ]
    setTimeout(() => { setChannels(mock); setLoading(false) }, 300)
  }, [])

  const filtered = channels.filter((c) =>
    c.name.toLowerCase().includes(query.toLowerCase()) ||
    c.models.toLowerCase().includes(query.toLowerCase())
  )

  const stats = {
    total: channels.length,
    active: channels.filter((c) => c.status === 'active').length,
    models: new Set(channels.flatMap((c) => c.models.split(','))).size,
  }

  return (
    <div className="max-w-6xl">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 mb-6">
        <div>
          <h2 className="text-xl font-semibold">{t('channels.title')}</h2>
          <p className="text-sm text-muted-foreground mt-0.5">{t('channels.subtitle')}</p>
        </div>
        <div className="flex items-center gap-2">
          <button className="inline-flex items-center gap-2 px-3 py-2 text-sm border border-border rounded-lg hover:bg-muted transition-colors cursor-pointer">
            <RefreshCw size={15} strokeWidth={2} />
            <span className="hidden sm:inline">{t('channels.testAll')}</span>
          </button>
          <button className="inline-flex items-center gap-2 px-3 py-2 text-sm bg-primary text-primary-foreground rounded-lg hover:bg-primary-dark transition-colors cursor-pointer">
            <Plus size={15} strokeWidth={2} />
            {t('channels.add')}
          </button>
        </div>
      </div>

      {/* Stats strip */}
      <div className="grid grid-cols-3 gap-3 mb-6">
        <StatCard icon={Server} label={t('channels.stats.total')} value={stats.total} />
        <StatCard icon={CheckCircle2} label={t('channels.stats.active')} value={stats.active} tone="success" />
        <StatCard icon={Layers} label={t('channels.stats.models')} value={stats.models} tone="primary" />
      </div>

      {/* Search */}
      <div className="relative mb-4">
        <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
        <input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder={t('channels.search')}
          className="w-full pl-9 pr-3 py-2 text-sm bg-card border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary"
        />
      </div>

      {/* Channel list */}
      {loading ? (
        <div className="space-y-3">
          {[1, 2, 3].map((i) => (
            <div key={i} className="h-20 bg-card border border-border rounded-xl animate-pulse" />
          ))}
        </div>
      ) : (
        <div className="space-y-3">
          {filtered.map((ch) => (
            <div
              key={ch.id}
              className="group bg-card border border-border rounded-xl p-4 hover:border-primary/30 transition-colors"
            >
              <div className="flex items-start justify-between gap-3">
                {/* Left: identity */}
                <div className="flex items-start gap-3 min-w-0">
                  <div className={`w-9 h-9 rounded-lg flex items-center justify-center shrink-0 ${
                    ch.status === 'active' ? 'bg-primary/10' : 'bg-muted'
                  }`}>
                    <Server size={17} className={ch.status === 'active' ? 'text-primary' : 'text-muted-foreground'} strokeWidth={2} />
                  </div>
                  <div className="min-w-0">
                    <div className="flex items-center gap-2 flex-wrap">
                      <span className="font-medium text-sm">{ch.name}</span>
                      <TypeBadge type={ch.type} />
                      {ch.status === 'disabled' && (
                        <span className="inline-flex items-center gap-1 text-[11px] text-muted-foreground">
                          <XCircle size={12} /> {t('channels.disabled')}
                        </span>
                      )}
                    </div>
                    <div className="flex items-center gap-3 mt-1 text-xs text-muted-foreground">
                      <span className="inline-flex items-center gap-1 truncate">
                        <Globe size={11} /> {ch.base_url.replace(/^https?:\/\//, '')}
                      </span>
                    </div>
                    {/* Models */}
                    <div className="flex flex-wrap gap-1 mt-2">
                      {ch.models.split(',').slice(0, 4).map((m) => (
                        <code key={m} className="px-1.5 py-0.5 text-[11px] font-mono bg-muted rounded">
                          {m.trim()}
                        </code>
                      ))}
                      {ch.modelCount && ch.modelCount > 4 && (
                        <span className="text-[11px] text-muted-foreground self-center">
                          +{ch.modelCount - 4}
                        </span>
                      )}
                    </div>
                  </div>
                </div>

                {/* Right: priority + actions */}
                <div className="flex items-center gap-2 shrink-0">
                  <div className="hidden sm:flex flex-col items-end mr-1">
                    <span className="text-[10px] uppercase tracking-wide text-muted-foreground">{t('channels.priority')}</span>
                    <span className="text-sm font-semibold font-mono">{ch.priority}</span>
                  </div>
                  <button className="p-2 rounded-lg hover:bg-muted opacity-0 group-hover:opacity-100 focus-visible:opacity-100 transition-all cursor-pointer" title={t('channels.edit')}>
                    <Pencil size={15} className="text-muted-foreground" strokeWidth={2} />
                  </button>
                  <button className="p-2 rounded-lg hover:bg-muted opacity-0 group-hover:opacity-100 focus-visible:opacity-100 transition-all cursor-pointer" title={ch.status === 'active' ? t('channels.disable') : t('channels.enable')}>
                    <Power size={15} className="text-muted-foreground" strokeWidth={2} />
                  </button>
                  <button className="p-2 rounded-lg hover:bg-red-500/10 opacity-0 group-hover:opacity-100 focus-visible:opacity-100 transition-all cursor-pointer" title={t('channels.delete')}>
                    <Trash2 size={15} className="text-destructive" strokeWidth={2} />
                  </button>
                </div>
              </div>
            </div>
          ))}
          {filtered.length === 0 && (
            <div className="text-center py-16 bg-card border border-border rounded-xl">
              <Server size={32} className="mx-auto text-muted-foreground mb-3" strokeWidth={1.5} />
              <p className="text-sm text-muted-foreground">{t('channels.empty')}</p>
              <button className="mt-4 inline-flex items-center gap-2 px-3 py-2 text-sm bg-primary text-primary-foreground rounded-lg hover:bg-primary-dark transition-colors cursor-pointer">
                <Plus size={15} strokeWidth={2} /> {t('channels.add')}
              </button>
            </div>
          )}
        </div>
      )}
    </div>
  )
}

function StatCard({ icon: Icon, label, value, tone }: {
  icon: typeof Server
  label: string
  value: number
  tone?: 'success' | 'primary'
}) {
  const color = tone === 'success' ? 'text-success' : tone === 'primary' ? 'text-primary' : 'text-muted-foreground'
  return (
    <div className="bg-card border border-border rounded-xl p-4">
      <div className="flex items-center justify-between">
        <span className="text-xs text-muted-foreground">{label}</span>
        <Icon size={16} className={color} strokeWidth={2} />
      </div>
      <p className="text-xl font-semibold font-mono mt-1.5">{value}</p>
    </div>
  )
}

function TypeBadge({ type }: { type: string }) {
  const badge = typeBadge[type] || { label: type, className: 'bg-muted text-muted-foreground' }
  return (
    <span className={`px-1.5 py-0.5 text-[10px] font-medium rounded ${badge.className}`}>
      {badge.label}
    </span>
  )
}