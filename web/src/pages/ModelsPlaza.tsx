import { useState, useEffect, useMemo } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  Zap, Search, Copy, Check, LayoutGrid, List, ChevronLeft, ChevronRight, X, Info,
} from 'lucide-react'
import { api } from '../lib/api'

interface PlazaGroup {
  name: string; ratio: number
  input_price: number; output_price: number; cache_price: number
}
interface PlazaModel {
  model: string; vendor: string; description: string
  protocols: string[]; channels: number; groups: PlazaGroup[]
  context_window: number
  input_price: number; output_price: number; cache_price: number
  stats: { avg_ms: number; tps: number; samples: number }
}

const PAGE_SIZE = 20

// Vendor badge palette — text avatar instead of external logos.
const vendorStyle: Record<string, string> = {
  openai: 'bg-emerald-500/15 text-emerald-600 dark:text-emerald-400',
  anthropic: 'bg-orange-500/15 text-orange-600 dark:text-orange-400',
  google: 'bg-blue-500/15 text-blue-600 dark:text-blue-400',
  deepseek: 'bg-indigo-500/15 text-indigo-600 dark:text-indigo-400',
  zhipu: 'bg-sky-500/15 text-sky-600 dark:text-sky-400',
  moonshot: 'bg-purple-500/15 text-purple-600 dark:text-purple-400',
  minimax: 'bg-rose-500/15 text-rose-600 dark:text-rose-400',
  meta: 'bg-blue-600/15 text-blue-600 dark:text-blue-400',
  other: 'bg-muted text-muted-foreground',
}
const protoLabel: Record<string, string> = {
  openai: 'OpenAI', anthropic: 'Claude', google: 'Gemini',
}

const price = (v: number): string =>
  v <= 0 ? '—' : `$${v < 0.01 ? v.toPrecision(2) : v.toFixed(v < 1 ? 3 : 2)}`
const ctxLabel = (n: number): string =>
  n >= 1_000_000 ? `${(n / 1_000_000).toFixed(n % 1_000_000 === 0 ? 0 : 1)}M`
  : n >= 1000 ? `${Math.round(n / 1000)}K` : n > 0 ? String(n) : '—'

export default function ModelsPlaza() {
  const { t } = useTranslation()
  const [models, setModels] = useState<PlazaModel[]>([])
  const [loading, setLoading] = useState(true)
  const [q, setQ] = useState('')
  const [group, setGroup] = useState('')
  const [sort, setSort] = useState<'name' | 'price'>('name')
  const [view, setView] = useState<'cards' | 'table'>('cards')
  const [page, setPage] = useState(1)
  const [copied, setCopied] = useState('')
  const [detail, setDetail] = useState<PlazaModel | null>(null)

  useEffect(() => {
    api.get<{ models: PlazaModel[] }>('/api/plaza')
      .then((d) => setModels(d.models ?? []))
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [])

  useEffect(() => { setPage(1) }, [q, group, sort])

  const groups = useMemo(() => {
    const s = new Set<string>()
    models.forEach((m) => m.groups.forEach((g) => s.add(g.name)))
    return [...s].sort()
  }, [models])

  const filtered = useMemo(() => {
    const out = models.filter((m) =>
      m.model.toLowerCase().includes(q.toLowerCase()) &&
      (group === '' || m.groups.some((g) => g.name === group)))
    out.sort((a, b) => sort === 'price'
      ? (a.input_price || Infinity) - (b.input_price || Infinity)
      : a.model.localeCompare(b.model))
    return out
  }, [models, q, group, sort])

  const totalPages = Math.max(1, Math.ceil(filtered.length / PAGE_SIZE))
  const pageItems = filtered.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE)

  // Alphabetical sections for the cards view.
  const sections = useMemo(() => {
    const map = new Map<string, PlazaModel[]>()
    for (const m of pageItems) {
      const letter = (m.model[0] ?? '#').toUpperCase()
      if (!map.has(letter)) map.set(letter, [])
      map.get(letter)!.push(m)
    }
    return [...map.entries()]
  }, [pageItems])

  const copy = (name: string) => {
    navigator.clipboard.writeText(name).then(() => {
      setCopied(name)
      setTimeout(() => setCopied(''), 1200)
    })
  }

  const inputCls = 'min-h-[40px] px-3 py-2 border border-input rounded-xl text-sm bg-background focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary'

  const VendorBadge = ({ v }: { v: string }) => (
    <span className={`px-2 py-0.5 rounded-full text-xs font-semibold uppercase tracking-wide ${vendorStyle[v] ?? vendorStyle.other}`}>
      {v}
    </span>
  )
  const PriceTriple = ({ m, g }: { m: PlazaModel; g?: PlazaGroup }) => (
    <div className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
      <span>{t('plaza.input')} <b className="text-foreground">{price(g ? g.input_price : m.input_price)}</b></span>
      <span>{t('plaza.output')} <b className="text-foreground">{price(g ? g.output_price : m.output_price)}</b></span>
      {(g ? g.cache_price : m.cache_price) > 0 && (
        <span>{t('plaza.cache')} <b className="text-foreground">{price(g ? g.cache_price : m.cache_price)}</b></span>
      )}
      <span className="text-muted-foreground/70">/1M</span>
    </div>
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
          <Link to="/auth/login" className="text-sm text-primary hover:underline">{t('auth.signIn')}</Link>
        </div>
      </header>

      <main className="max-w-6xl mx-auto px-4 py-8">
        {/* Hero */}
        <div className="text-center mb-8">
          <h1 className="text-3xl font-bold tracking-tight mb-2">{t('plaza.heading')}</h1>
          <p className="text-sm text-muted-foreground">
            {loading ? '…' : t('plaza.enabledCount', { count: models.length })}
          </p>
          <p className="text-sm text-muted-foreground mt-1">{t('plaza.desc')}</p>
        </div>

        {/* Toolbar */}
        <div className="flex flex-col lg:flex-row gap-2 mb-6">
          <div className="relative flex-1">
            <Search size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
            <input value={q} onChange={(e) => setQ(e.target.value)} placeholder={t('plaza.search')}
              className={`${inputCls} w-full pl-9`} />
          </div>
          <div className="flex gap-1.5 flex-wrap items-center">
            <button onClick={() => setGroup('')}
              className={`px-3 py-2 rounded-xl text-xs font-medium border transition-colors cursor-pointer ${
                group === '' ? 'border-primary bg-primary/5 text-primary' : 'border-border hover:border-primary/30'}`}>
              {t('plaza.allGroups')}
            </button>
            {groups.map((g) => (
              <button key={g} onClick={() => setGroup(g)}
                className={`px-3 py-2 rounded-xl text-xs font-medium border transition-colors cursor-pointer ${
                  group === g ? 'border-primary bg-primary/5 text-primary' : 'border-border hover:border-primary/30'}`}>
                {g}
              </button>
            ))}
            <select value={sort} onChange={(e) => setSort(e.target.value as 'name' | 'price')}
              className={`${inputCls} cursor-pointer`} title={t('plaza.sort')}>
              <option value="name">{t('plaza.sortName')}</option>
              <option value="price">{t('plaza.sortPrice')}</option>
            </select>
            <div className="flex rounded-xl border border-border overflow-hidden">
              <button onClick={() => setView('cards')} title={t('plaza.viewCards')}
                className={`px-2.5 py-2 cursor-pointer ${view === 'cards' ? 'bg-primary/10 text-primary' : 'text-muted-foreground hover:bg-muted'}`}><LayoutGrid size={15} /></button>
              <button onClick={() => setView('table')} title={t('plaza.viewTable')}
                className={`px-2.5 py-2 cursor-pointer ${view === 'table' ? 'bg-primary/10 text-primary' : 'text-muted-foreground hover:bg-muted'}`}><List size={15} /></button>
            </div>
          </div>
        </div>

        {/* Count */}
        <p className="text-sm text-muted-foreground mb-4">
          <b className="text-foreground">{filtered.length}</b> {t('plaza.modelUnit')}
        </p>

        {loading ? (
          <div className="py-20 text-center text-muted-foreground text-sm">…</div>
        ) : filtered.length === 0 ? (
          <div className="py-20 text-center text-muted-foreground text-sm">{t('plaza.empty')}</div>
        ) : view === 'cards' ? (
          /* Cards view with alphabetical sections */
          <div className="space-y-6">
            {sections.map(([letter, items]) => (
              <div key={letter}>
                <div className="flex items-center gap-3 mb-3">
                  <span className="w-7 h-7 rounded-lg bg-muted flex items-center justify-center text-xs font-bold text-muted-foreground">{letter}</span>
                  <span className="h-px flex-1 bg-border" />
                </div>
                <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
                  {items.map((m) => (
                    <div key={m.model} className="p-4 rounded-2xl border border-border bg-card hover:border-primary/30 transition-colors flex flex-col gap-3">
                      <div className="flex items-start justify-between gap-2">
                        <div className="min-w-0">
                          <h3 className="font-mono text-sm font-semibold truncate" title={m.model}>{m.model}</h3>
                          <div className="mt-1.5 flex items-center gap-1.5 flex-wrap">
                            <VendorBadge v={m.vendor} />
                            {m.protocols.map((p) => (
                              <span key={p} className="px-1.5 py-0.5 rounded bg-muted text-[11px]">{protoLabel[p] ?? p}</span>
                            ))}
                          </div>
                        </div>
                        <div className="flex gap-1 shrink-0">
                          <button onClick={() => setDetail(m)} title={t('plaza.detail')}
                            className="p-1.5 rounded-lg hover:bg-muted text-muted-foreground hover:text-foreground cursor-pointer"><Info size={14} /></button>
                          <button onClick={() => copy(m.model)} title={t('plaza.copy')}
                            className="p-1.5 rounded-lg hover:bg-muted text-muted-foreground hover:text-foreground cursor-pointer">
                            {copied === m.model ? <Check size={14} className="text-emerald-500" /> : <Copy size={14} />}
                          </button>
                        </div>
                      </div>
                      <PriceTriple m={m} />
                      <div className="flex items-center justify-between text-xs text-muted-foreground mt-auto">
                        <span>{t('plaza.ctx')} <b className="text-foreground">{ctxLabel(m.context_window)}</b></span>
                        {m.stats.samples > 0 && (
                          <span>{t('plaza.latency')} <b className="text-foreground">{(m.stats.avg_ms / 1000).toFixed(1)}s</b>
                            {m.stats.tps > 0 && <> · {Math.round(m.stats.tps)}t/s</>}
                          </span>
                        )}
                      </div>
                      <div className="flex flex-wrap gap-1">
                        {m.groups.map((g) => (
                          <span key={g.name} className="px-1.5 py-0.5 rounded bg-muted text-[11px] font-mono">{g.name} ×{g.ratio}</span>
                        ))}
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </div>
        ) : (
          /* Table view */
          <div className="border border-border rounded-xl overflow-hidden bg-card">
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border bg-muted/50 text-left">
                    <th className="px-4 py-3 font-medium text-muted-foreground">{t('plaza.colModel')}</th>
                    <th className="px-4 py-3 font-medium text-muted-foreground">{t('plaza.vendor')}</th>
                    <th className="px-4 py-3 font-medium text-muted-foreground">{t('plaza.input')}</th>
                    <th className="px-4 py-3 font-medium text-muted-foreground">{t('plaza.output')}</th>
                    <th className="px-4 py-3 font-medium text-muted-foreground">{t('plaza.ctxCol')}</th>
                    <th className="px-4 py-3 font-medium text-muted-foreground">{t('users.colGroup')}</th>
                    <th className="px-4 py-3 font-medium text-muted-foreground text-right"> </th>
                  </tr>
                </thead>
                <tbody>
                  {pageItems.map((m) => (
                    <tr key={m.model} className="border-b border-border last:border-0 hover:bg-muted/30">
                      <td className="px-4 py-3 font-mono text-xs font-medium">{m.model}</td>
                      <td className="px-4 py-3"><VendorBadge v={m.vendor} /></td>
                      <td className="px-4 py-3 font-mono text-xs">{price(m.input_price)}</td>
                      <td className="px-4 py-3 font-mono text-xs">{price(m.output_price)}</td>
                      <td className="px-4 py-3 text-xs">{ctxLabel(m.context_window)}</td>
                      <td className="px-4 py-3">
                        <div className="flex flex-wrap gap-1">
                          {m.groups.map((g) => (
                            <span key={g.name} className="px-1.5 py-0.5 rounded bg-muted text-[11px] font-mono" title={`${t('plaza.input')} ${price(g.input_price)}`}>{g.name}</span>
                          ))}
                        </div>
                      </td>
                      <td className="px-4 py-3 text-right">
                        <button onClick={() => copy(m.model)} className="p-1.5 rounded-lg hover:bg-muted text-muted-foreground cursor-pointer">
                          {copied === m.model ? <Check size={13} className="text-emerald-500" /> : <Copy size={13} />}
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {/* Pagination */}
        {totalPages > 1 && (
          <div className="flex items-center justify-between mt-6 text-sm text-muted-foreground">
            <span>{t('plaza.pageInfo', { page, pages: totalPages })}</span>
            <div className="flex gap-2">
              <button disabled={page <= 1} onClick={() => setPage(page - 1)}
                className="p-2 border border-border rounded-lg hover:bg-muted disabled:opacity-40 cursor-pointer disabled:cursor-not-allowed"><ChevronLeft size={15} /></button>
              <button disabled={page >= totalPages} onClick={() => setPage(page + 1)}
                className="p-2 border border-border rounded-lg hover:bg-muted disabled:opacity-40 cursor-pointer disabled:cursor-not-allowed"><ChevronRight size={15} /></button>
            </div>
          </div>
        )}
      </main>

      {/* Detail dialog */}
      {detail && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
          <div className="absolute inset-0 bg-black/50" onClick={() => setDetail(null)} />
          <div className="relative w-full max-w-md bg-card border border-border rounded-2xl p-5 shadow-lg anim-fade-up max-h-[80vh] overflow-y-auto">
            <div className="flex items-start justify-between mb-3">
              <div>
                <h3 className="font-mono font-semibold text-sm break-all">{detail.model}</h3>
                <div className="mt-1.5"><VendorBadge v={detail.vendor} /></div>
              </div>
              <button onClick={() => setDetail(null)} className="p-1 rounded-lg hover:bg-muted cursor-pointer"><X size={15} /></button>
            </div>
            <p className="text-sm text-muted-foreground mb-4">{detail.description || t('plaza.noDesc')}</p>
            <div className="space-y-3 text-sm">
              <div className="flex justify-between"><span className="text-muted-foreground">{t('plaza.ctx')}</span><b>{ctxLabel(detail.context_window)}</b></div>
              <div className="flex justify-between"><span className="text-muted-foreground">{t('plaza.channelsCol')}</span><b>{detail.channels}</b></div>
              {detail.stats.samples > 0 && <>
                <div className="flex justify-between"><span className="text-muted-foreground">{t('plaza.latency')}</span><b>{(detail.stats.avg_ms / 1000).toFixed(1)}s</b></div>
                {detail.stats.tps > 0 && <div className="flex justify-between"><span className="text-muted-foreground">{t('plaza.throughput')}</span><b>{Math.round(detail.stats.tps)} t/s</b></div>}
                <div className="flex justify-between"><span className="text-muted-foreground">{t('plaza.samples')}</span><b>{detail.stats.samples}</b></div>
              </>}
            </div>
            <div className="mt-4">
              <p className="text-xs font-medium text-muted-foreground mb-2">{t('plaza.groupPricing')}</p>
              <div className="space-y-1.5">
                {detail.groups.map((g) => (
                  <div key={g.name} className="flex items-center justify-between text-xs px-3 py-2 rounded-lg bg-muted/50">
                    <span className="font-mono font-medium">{g.name} ×{g.ratio}</span>
                    <span className="text-muted-foreground">
                      {t('plaza.input')} {price(g.input_price)} · {t('plaza.output')} {price(g.output_price)} /1M
                    </span>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
