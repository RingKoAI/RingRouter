import { useState, useEffect, useMemo } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Boxes, Search, Zap, ArrowRight } from 'lucide-react'
import { api } from '../lib/api'

interface PlazaGroup { name: string; ratio: number }
interface PlazaModel {
  model: string
  protocols: string[]
  channels: number
  groups: PlazaGroup[]
}

const protoBadge: Record<string, { label: string; cls: string }> = {
  openai: { label: 'OpenAI', cls: 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400' },
  anthropic: { label: 'Claude', cls: 'bg-orange-500/10 text-orange-600 dark:text-orange-400' },
  google: { label: 'Gemini', cls: 'bg-blue-500/10 text-blue-600 dark:text-blue-400' },
}

export default function ModelsPlaza() {
  const { t } = useTranslation()
  const [models, setModels] = useState<PlazaModel[]>([])
  const [loading, setLoading] = useState(true)
  const [q, setQ] = useState('')
  const [group, setGroup] = useState('')

  useEffect(() => {
    api.get<{ models: PlazaModel[] }>('/api/plaza')
      .then((d) => setModels(d.models ?? []))
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [])

  const groups = useMemo(() => {
    const s = new Map<string, number>()
    models.forEach((m) => m.groups.forEach((g) => s.set(g.name, g.ratio)))
    return [...s.entries()].sort()
  }, [models])

  const filtered = models.filter((m) =>
    m.model.toLowerCase().includes(q.toLowerCase()) &&
    (group === '' || m.groups.some((g) => g.name === group))
  )

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <header className="border-b border-border">
        <div className="max-w-6xl mx-auto px-4 h-16 flex items-center justify-between">
          <Link to="/" className="flex items-center gap-2 font-semibold">
            <div className="w-8 h-8 rounded-lg bg-primary flex items-center justify-center">
              <Zap size={15} className="text-primary-foreground" />
            </div>
            {t('plaza.title')}
          </Link>
          <Link to="/auth/login" className="inline-flex items-center gap-1.5 text-sm text-primary hover:underline">
            {t('auth.signIn')} <ArrowRight size={13} />
          </Link>
        </div>
      </header>

      <main className="max-w-6xl mx-auto px-4 py-10">
        {/* Hero */}
        <div className="text-center mb-8">
          <h1 className="text-3xl font-bold tracking-tight mb-2">{t('plaza.heading')}</h1>
          <p className="text-sm text-muted-foreground">
            {t('plaza.subtitle', { count: models.length })}
          </p>
        </div>

        {/* Filters */}
        <div className="flex flex-col sm:flex-row gap-2 mb-6">
          <div className="relative flex-1">
            <Search size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
            <input value={q} onChange={(e) => setQ(e.target.value)}
              placeholder={t('plaza.search')}
              className="w-full min-h-[42px] pl-9 pr-3 py-2 border border-input rounded-xl text-sm bg-background focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary" />
          </div>
          <div className="flex gap-1.5 flex-wrap">
            <button onClick={() => setGroup('')}
              className={`px-3 py-2 min-h-[42px] rounded-xl text-xs font-medium border transition-colors cursor-pointer ${
                group === '' ? 'border-primary bg-primary/5 text-primary' : 'border-border hover:border-primary/30'}`}>
              {t('plaza.allGroups')}
            </button>
            {groups.map(([name, ratio]) => (
              <button key={name} onClick={() => setGroup(name)}
                className={`px-3 py-2 min-h-[42px] rounded-xl text-xs font-medium border transition-colors cursor-pointer ${
                  group === name ? 'border-primary bg-primary/5 text-primary' : 'border-border hover:border-primary/30'}`}>
                {name} × {ratio}
              </button>
            ))}
          </div>
        </div>

        {/* Grid */}
        {loading ? (
          <div className="py-20 text-center text-muted-foreground text-sm">…</div>
        ) : filtered.length === 0 ? (
          <div className="py-20 text-center text-muted-foreground text-sm">{t('plaza.empty')}</div>
        ) : (
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {filtered.map((m) => (
              <div key={m.model} className="p-4 rounded-2xl border border-border bg-card hover:border-primary/30 transition-colors">
                <div className="flex items-start justify-between gap-2 mb-3">
                  <div className="flex items-center gap-2 min-w-0">
                    <Boxes size={15} className="text-primary shrink-0" />
                    <span className="font-mono text-sm font-medium truncate">{m.model}</span>
                  </div>
                </div>
                <div className="flex flex-wrap gap-1.5 mb-3">
                  {m.protocols.map((p) => (
                    <span key={p} className={`px-2 py-0.5 rounded-full text-xs font-medium ${protoBadge[p]?.cls ?? 'bg-muted'}`}>
                      {protoBadge[p]?.label ?? p}
                    </span>
                  ))}
                </div>
                <div className="flex items-center justify-between text-xs text-muted-foreground">
                  <span>{t('plaza.channels', { count: m.channels })}</span>
                  <span className="flex gap-1">
                    {m.groups.map((g) => (
                      <span key={g.name} className="px-1.5 py-0.5 rounded bg-muted font-mono">{g.name} ×{g.ratio}</span>
                    ))}
                  </span>
                </div>
              </div>
            ))}
          </div>
        )}
      </main>
    </div>
  )
}
