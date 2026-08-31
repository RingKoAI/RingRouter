import { useState, useEffect, useMemo, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import {
  Server, Plus, Search, Pencil, Trash2, Power, RefreshCw, Globe, Layers,
  XCircle, CheckCircle2, X, Loader2, Users,
} from 'lucide-react'
import { api, APIError } from '../lib/api'

/* ── Wire types (backend internal/handler/channels.go) ─────────────────── */

interface Channel {
  id: number
  name: string
  protocol: string // "type" on write; echoed as protocol
  base_url: string
  models: string
  model_mapping: string
  group: string
  status: 'active' | 'disabled'
  priority: number
  weight: number
  remark: string
  created_at: string
  updated_at: string
  api_key_masked?: string
}

interface GroupInfo { name: string; ratio: number; is_default: boolean }

interface ChannelForm {
  name: string
  type: string
  base_url: string
  api_key: string
  models: string
  group: string
  priority: string
  status: 'active' | 'disabled'
  model_mapping: string
}

const emptyForm: ChannelForm = {
  name: '', type: 'openai', base_url: '', api_key: '', models: '',
  group: 'default', priority: '0', status: 'active', model_mapping: '',
}

const typeBadge: Record<string, { label: string; className: string }> = {
  openai: { label: 'OpenAI', className: 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400' },
  anthropic: { label: 'Claude', className: 'bg-orange-500/10 text-orange-600 dark:text-orange-400' },
  google: { label: 'Gemini', className: 'bg-blue-500/10 text-blue-600 dark:text-blue-400' },
}

const typePlaceholder: Record<string, string> = {
  openai: 'https://api.openai.com (留空使用官方默认)',
  anthropic: 'https://api.anthropic.com (留空使用官方默认)',
  google: 'https://generativelanguage.googleapis.com (留空使用官方默认)',
}

export default function Channels() {
  const { t } = useTranslation()
  const [channels, setChannels] = useState<Channel[]>([])
  const [groups, setGroups] = useState<GroupInfo[]>([])
  const [query, setQuery] = useState('')
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)

  // Dialog state: editing holds the channel being edited (null = closed).
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editing, setEditing] = useState<Channel | null>(null)
  const [form, setForm] = useState<ChannelForm>(emptyForm)
  const [formErr, setFormErr] = useState('')
  const [deleting, setDeleting] = useState<Channel | null>(null)
  const [busyId, setBusyId] = useState<number | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const [c, g] = await Promise.all([
        api.get<{ channels: Channel[] }>('/api/admin/channels'),
        api.get<{ groups: GroupInfo[] }>('/api/admin/groups').catch(() => ({ groups: [] as GroupInfo[] })),
      ])
      setChannels(c.channels ?? [])
      setGroups(g.groups ?? [])
    } catch {
      setChannels([])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { load() }, [load])

  const filtered = useMemo(() =>
    channels.filter((c) =>
      !query ||
      c.name.toLowerCase().includes(query.toLowerCase()) ||
      c.models.toLowerCase().includes(query.toLowerCase())
    ), [channels, query])

  const stats = {
    total: channels.length,
    active: channels.filter((c) => c.status === 'active').length,
    models: new Set(channels.flatMap((c) => c.models.split(',').map((m) => m.trim())).filter(Boolean)).size,
  }

  /* ── Dialog helpers ── */

  const openCreate = () => {
    setEditing(null)
    setForm({ ...emptyForm, group: groups.find((g) => g.is_default)?.name ?? 'default' })
    setFormErr('')
    setDialogOpen(true)
  }

  const openEdit = (ch: Channel) => {
    setEditing(ch)
    setForm({
      name: ch.name,
      type: ch.protocol,
      base_url: ch.base_url,
      api_key: '',
      models: ch.models,
      group: ch.group || 'default',
      priority: String(ch.priority),
      status: ch.status,
      model_mapping: ch.model_mapping || '',
    })
    setFormErr('')
    setDialogOpen(true)
  }

  const valid = form.name.trim() !== '' && form.models.trim() !== '' &&
    (editing !== null || form.api_key.trim() !== '')

  const submit = async () => {
    setFormErr(''); setSaving(true)
    try {
      const priority = parseInt(form.priority || '0', 10)
      const payload: Record<string, unknown> = {
        name: form.name.trim(),
        type: form.type,
        models: form.models,
        group: form.group || 'default',
        priority: Number.isFinite(priority) ? priority : 0,
        status: form.status,
      }
      if (form.base_url.trim()) payload.base_url = form.base_url.trim()
      if (form.api_key.trim()) payload.api_key = form.api_key.trim()
      if (form.model_mapping.trim()) payload.model_mapping = form.model_mapping.trim()
      if (editing) {
        await api.put(`/api/admin/channels/${editing.id}`, payload)
        toast.success(t('channels.saved'))
      } else {
        await api.post('/api/admin/channels', payload)
        toast.success(t('channels.created'))
      }
      setDialogOpen(false)
      await load()
    } catch (e) {
      setFormErr(e instanceof APIError ? e.message : t('channels.saveFailed'))
    } finally {
      setSaving(false)
    }
  }

  const toggle = async (ch: Channel) => {
    setBusyId(ch.id)
    try {
      await api.put(`/api/admin/channels/${ch.id}`, {
        status: ch.status === 'active' ? 'disabled' : 'active',
      })
      toast.success(ch.status === 'active' ? t('channels.disabledToast') : t('channels.enabledToast'))
      await load()
    } catch (e) {
      if (e instanceof APIError) toast.error(e.message)
    } finally {
      setBusyId(null)
    }
  }

  const confirmDelete = async () => {
    if (!deleting) return
    try {
      await api.delete(`/api/admin/channels/${deleting.id}`)
      toast.success(t('channels.deleted'))
      setDeleting(null)
      await load()
    } catch (e) {
      if (e instanceof APIError) toast.error(e.message)
    }
  }

  const inputCls =
    'w-full min-h-[40px] px-3 py-2 border border-input rounded-lg text-sm bg-background ' +
    'focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary transition-all'

  return (
    <div className="max-w-6xl">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 mb-6">
        <div>
          <h2 className="text-xl font-semibold">{t('channels.title')}</h2>
          <p className="text-sm text-muted-foreground mt-0.5">{t('channels.subtitle')}</p>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={load}
            className="inline-flex items-center gap-2 px-3 py-2 text-sm border border-border rounded-lg hover:bg-muted transition-colors cursor-pointer whitespace-nowrap"
          >
            <RefreshCw size={15} strokeWidth={2} className={loading ? 'animate-spin' : ''} />
            <span className="hidden sm:inline">{t('channels.refresh')}</span>
          </button>
          <button
            onClick={openCreate}
            className="inline-flex items-center gap-2 px-3 py-2 text-sm bg-primary text-primary-foreground rounded-lg hover:bg-primary-dark transition-colors cursor-pointer whitespace-nowrap"
          >
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
            <div key={i} className="h-24 bg-card border border-border rounded-xl animate-pulse" />
          ))}
        </div>
      ) : (
        <div className="space-y-3">
          {filtered.map((ch) => {
            const modelList = ch.models.split(',').map((m) => m.trim()).filter(Boolean)
            return (
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
                      {busyId === ch.id
                        ? <Loader2 size={17} className="text-muted-foreground animate-spin" />
                        : <Server size={17} className={ch.status === 'active' ? 'text-primary' : 'text-muted-foreground'} strokeWidth={2} />}
                    </div>
                    <div className="min-w-0">
                      <div className="flex items-center gap-2 flex-wrap">
                        <span className="font-medium text-sm">{ch.name}</span>
                        <TypeBadge type={ch.protocol} />
                        {ch.status === 'disabled' && (
                          <span className="inline-flex items-center gap-1 text-[11px] text-muted-foreground whitespace-nowrap">
                            <XCircle size={12} /> {t('channels.disabled')}
                          </span>
                        )}
                      </div>
                      <div className="flex items-center gap-3 mt-1 text-xs text-muted-foreground flex-wrap">
                        <span className="inline-flex items-center gap-1 truncate">
                          <Globe size={11} /> {ch.base_url ? ch.base_url.replace(/^https?:\/\//, '') : t('channels.officialDefault')}
                        </span>
                        <span className="inline-flex items-center gap-1">
                          <Users size={11} /> {ch.group || 'default'}
                        </span>
                      </div>
                      {/* Models */}
                      <div className="flex flex-wrap gap-1 mt-2">
                        {modelList.slice(0, 4).map((m) => (
                          <code key={m} className="px-1.5 py-0.5 text-[11px] font-mono bg-muted rounded">{m}</code>
                        ))}
                        {modelList.length > 4 && (
                          <span className="text-[11px] text-muted-foreground self-center">
                            +{modelList.length - 4}
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
                    <button onClick={() => openEdit(ch)} className="p-2 rounded-lg hover:bg-muted opacity-0 group-hover:opacity-100 focus-visible:opacity-100 transition-all cursor-pointer" title={t('channels.edit')}>
                      <Pencil size={15} className="text-muted-foreground" strokeWidth={2} />
                    </button>
                    <button onClick={() => toggle(ch)} disabled={busyId === ch.id} className="p-2 rounded-lg hover:bg-muted opacity-0 group-hover:opacity-100 focus-visible:opacity-100 transition-all cursor-pointer disabled:opacity-30" title={ch.status === 'active' ? t('channels.disable') : t('channels.enable')}>
                      <Power size={15} className="text-muted-foreground" strokeWidth={2} />
                    </button>
                    <button onClick={() => setDeleting(ch)} className="p-2 rounded-lg hover:bg-red-500/10 opacity-0 group-hover:opacity-100 focus-visible:opacity-100 transition-all cursor-pointer" title={t('channels.delete')}>
                      <Trash2 size={15} className="text-destructive" strokeWidth={2} />
                    </button>
                  </div>
                </div>
              </div>
            )
          })}
          {filtered.length === 0 && (
            <div className="text-center py-16 bg-card border border-border rounded-xl">
              <Server size={32} className="mx-auto text-muted-foreground mb-3" strokeWidth={1.5} />
              <p className="text-sm text-muted-foreground">{t('channels.empty')}</p>
              <button onClick={openCreate} className="mt-4 inline-flex items-center gap-2 px-3 py-2 text-sm bg-primary text-primary-foreground rounded-lg hover:bg-primary-dark transition-colors cursor-pointer whitespace-nowrap">
                <Plus size={15} strokeWidth={2} /> {t('channels.add')}
              </button>
            </div>
          )}
        </div>
      )}

      {/* Create / edit dialog */}
      {dialogOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
          <div className="absolute inset-0 bg-black/50" onClick={() => !saving && setDialogOpen(false)} />
          <div className="relative w-full max-w-md bg-card border border-border rounded-2xl p-5 shadow-lg anim-fade-up max-h-[90vh] overflow-y-auto rr-scroll-thin">
            <div className="flex items-start justify-between mb-4">
              <h3 className="font-semibold">{editing ? t('channels.editTitle') : t('channels.addTitle')}</h3>
              <button onClick={() => setDialogOpen(false)} className="p-1 rounded-lg hover:bg-muted cursor-pointer"><X size={15} /></button>
            </div>
            <div className="space-y-3">
              <input className={inputCls} value={form.name} maxLength={64}
                onChange={(e) => setForm({ ...form, name: e.target.value })}
                placeholder={t('channels.fName')} />
              <div className="grid grid-cols-2 gap-2">
                <select className={inputCls} value={form.type}
                  onChange={(e) => setForm({ ...form, type: e.target.value })}>
                  <option value="openai">OpenAI</option>
                  <option value="anthropic">Anthropic</option>
                  <option value="google">Google Gemini</option>
                </select>
                <select className={inputCls} value={form.group}
                  onChange={(e) => setForm({ ...form, group: e.target.value })}>
                  {(groups.length ? groups : [{ name: 'default', ratio: 1, is_default: true }]).map((g) => (
                    <option key={g.name} value={g.name}>{g.name}{g.is_default ? ` · ${t('channels.defaultGroup')}` : ''}</option>
                  ))}
                </select>
              </div>
              <input className={inputCls} value={form.base_url} maxLength={256}
                onChange={(e) => setForm({ ...form, base_url: e.target.value })}
                placeholder={typePlaceholder[form.type] ?? 'https://'} />
              <input className={`${inputCls} font-mono`} type="password" value={form.api_key} autoComplete="new-password"
                onChange={(e) => setForm({ ...form, api_key: e.target.value })}
                placeholder={editing ? t('channels.fKeyKeep') : t('channels.fKey')} />
              <textarea className={`${inputCls} min-h-[64px]`} value={form.models}
                onChange={(e) => setForm({ ...form, models: e.target.value })}
                placeholder={t('channels.fModels')} />
              <div className="grid grid-cols-2 gap-2">
                <input className={`${inputCls} font-mono`} value={form.priority} inputMode="numeric"
                  onChange={(e) => setForm({ ...form, priority: e.target.value.replace(/[^\d-]/g, '') })}
                  placeholder={t('channels.fPriority')} />
                <select className={inputCls} value={form.status}
                  onChange={(e) => setForm({ ...form, status: e.target.value as 'active' | 'disabled' })}>
                  <option value="active">{t('channels.statusActive')}</option>
                  <option value="disabled">{t('channels.statusDisabled')}</option>
                </select>
              </div>
              <input className={`${inputCls} font-mono text-xs`} value={form.model_mapping} maxLength={2048}
                onChange={(e) => setForm({ ...form, model_mapping: e.target.value })}
                placeholder={t('channels.fMapping')} />
              {formErr && <p className="text-sm text-destructive" role="alert">{formErr}</p>}
              <button onClick={submit} disabled={saving || !valid}
                className="w-full min-h-[42px] text-sm bg-primary text-primary-foreground rounded-xl hover:bg-primary-dark disabled:opacity-50 transition-colors cursor-pointer inline-flex items-center justify-center gap-2 whitespace-nowrap">
                {saving && <Loader2 size={14} className="animate-spin" />}
                {editing ? t('channels.save') : t('channels.create')}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Delete confirm dialog */}
      {deleting && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
          <div className="absolute inset-0 bg-black/50" onClick={() => setDeleting(null)} />
          <div className="relative w-full max-w-sm bg-card border border-border rounded-2xl p-5 shadow-lg anim-fade-up">
            <h3 className="font-semibold mb-2">{t('channels.deleteTitle')}</h3>
            <p className="text-sm text-muted-foreground mb-4">
              {t('channels.deleteDesc', { name: deleting.name })}
            </p>
            <div className="flex gap-2 justify-end">
              <button onClick={() => setDeleting(null)} className="px-4 min-h-[38px] text-sm border border-border rounded-lg hover:bg-muted transition-colors cursor-pointer">
                {t('users.cancel')}
              </button>
              <button onClick={confirmDelete} className="px-4 min-h-[38px] text-sm bg-destructive text-white rounded-lg hover:bg-destructive/90 transition-colors cursor-pointer">
                {t('channels.delete')}
              </button>
            </div>
          </div>
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
