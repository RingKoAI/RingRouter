import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { Boxes, RefreshCw, Coins, X, Check } from 'lucide-react'
import { api, APIError } from '../lib/api'

interface ChannelLite {
  id: number; name: string; protocol: string; models: string; group: string
  status: string; priority: number; model_mapping: string
}
interface GroupInfo { id: number; name: string; ratio: number; is_default: boolean }
interface MetaForm { vendor: string; description: string; input_price: string; output_price: string; cache_price: string; context_window: string }

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
  const [editModel, setEditModel] = useState('')
  const [mf, setMf] = useState<MetaForm>({ vendor: '', description: '', input_price: '', output_price: '', cache_price: '', context_window: '' })
  const [metaBusy, setMetaBusy] = useState(false)
  const [metaErr, setMetaErr] = useState('')

  const openMeta = async (model: string) => {
    setEditModel(model); setMetaErr('')
    try {
      const d = await api.get<{ metas: { name: string; vendor: string; description: string; input_price: number; output_price: number; cache_price: number; context_window: number }[] }>('/api/admin/models')
      const m = (d.metas ?? []).find((x) => x.name === model)
      setMf(m ? {
        vendor: m.vendor, description: m.description,
        input_price: String(m.input_price || ''), output_price: String(m.output_price || ''),
        cache_price: String(m.cache_price || ''), context_window: String(m.context_window || ''),
      } : { vendor: 'openai', description: '', input_price: '', output_price: '', cache_price: '', context_window: '' })
    } catch {
      setMf({ vendor: 'openai', description: '', input_price: '', output_price: '', cache_price: '', context_window: '' })
    }
  }

  const saveMeta = async () => {
    setMetaBusy(true); setMetaErr('')
    try {
      await api.put(`/api/admin/models/${encodeURIComponent(editModel)}`, {
        vendor: mf.vendor.trim() || undefined,
        description: mf.description.trim() || undefined,
        input_price: mf.input_price === '' ? undefined : parseFloat(mf.input_price),
        output_price: mf.output_price === '' ? undefined : parseFloat(mf.output_price),
        cache_price: mf.cache_price === '' ? undefined : parseFloat(mf.cache_price),
        context_window: mf.context_window === '' ? undefined : parseInt(mf.context_window, 10),
      })
      setEditModel('')
    } catch (e) { setMetaErr(e instanceof APIError ? e.message : '') } finally { setMetaBusy(false) }
  }

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
                <td className="px-4 py-3 text-right">
                  <button onClick={() => openMeta(r.model)} title={t('models.setPrice')}
                    className="p-1.5 rounded-lg hover:bg-muted text-muted-foreground hover:text-foreground cursor-pointer"><Coins size={14} /></button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Meta editor dialog */}
      {editModel && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
          <div className="absolute inset-0 bg-black/50" onClick={() => !metaBusy && setEditModel('')} />
          <div className="relative w-full max-w-sm bg-card border border-border rounded-2xl p-5 shadow-lg anim-fade-up">
            <div className="flex items-start justify-between mb-3">
              <h3 className="font-semibold text-sm font-mono break-all">{editModel}</h3>
              <button onClick={() => setEditModel('')} className="p-1 rounded-lg hover:bg-muted cursor-pointer"><X size={15} /></button>
            </div>
            <p className="text-xs text-muted-foreground mb-3">{t('models.priceUnit')}</p>
            <div className="space-y-2.5">
              <input value={mf.vendor} onChange={(e) => setMf({ ...mf, vendor: e.target.value })} placeholder={t('models.fVendor')} className={inputCls} />
              <input value={mf.description} onChange={(e) => setMf({ ...mf, description: e.target.value })} placeholder={t('models.fDesc')} className={inputCls} />
              <div className="grid grid-cols-3 gap-2">
                <input value={mf.input_price} onChange={(e) => setMf({ ...mf, input_price: e.target.value.replace(/[^\d.]/g, '') })} placeholder={t('plaza.input')} inputMode="decimal" className={inputCls} />
                <input value={mf.output_price} onChange={(e) => setMf({ ...mf, output_price: e.target.value.replace(/[^\d.]/g, '') })} placeholder={t('plaza.output')} inputMode="decimal" className={inputCls} />
                <input value={mf.cache_price} onChange={(e) => setMf({ ...mf, cache_price: e.target.value.replace(/[^\d.]/g, '') })} placeholder={t('plaza.cache')} inputMode="decimal" className={inputCls} />
              </div>
              <input value={mf.context_window} onChange={(e) => setMf({ ...mf, context_window: e.target.value.replace(/\D/g, '') })} placeholder={t('models.fCtx')} inputMode="numeric" className={inputCls} />
              {metaErr && <p className="text-sm text-destructive">{metaErr}</p>}
              <button onClick={saveMeta} disabled={metaBusy}
                className="w-full min-h-[40px] text-sm bg-primary text-primary-foreground rounded-lg hover:bg-primary-dark disabled:opacity-50 cursor-pointer inline-flex items-center justify-center gap-1.5">
                <Check size={14} /> {t('users.confirm')}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
