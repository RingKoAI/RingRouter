import { useState, useEffect, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { Package, Plus, Trash2, UserPlus, RefreshCw, X, Check } from 'lucide-react'
import { api, APIError } from '../lib/api'

interface Plan {
  id: number; name: string; description: string; price_cents: number
  quota: number; group: string; duration_days: number; status: string
}
interface Subscription {
  id: number; user_id: number; plan_name: string; group: string
  quota: number; expires_at: string; status: string; created_at: string
}
interface UserLite { id: number; username: string; email: string }

type Dialog =
  | { kind: 'none' }
  | { kind: 'plan' }
  | { kind: 'grant' }

export default function Subscriptions() {
  const { t } = useTranslation()
  const [plans, setPlans] = useState<Plan[]>([])
  const [subs, setSubs] = useState<Subscription[]>([])
  const [users, setUsers] = useState<UserLite[]>([])
  const [loading, setLoading] = useState(true)
  const [dialog, setDialog] = useState<Dialog>({ kind: 'none' })
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  // plan form
  const [pf, setPf] = useState({ name: '', description: '', price: '', quota: '', group: 'default', days: '30' })
  // grant form
  const [gf, setGf] = useState({ userId: 0, planId: 0 })

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const [p, s, u] = await Promise.all([
        api.get<{ plans: Plan[] }>('/api/admin/plans'),
        api.get<{ subscriptions: Subscription[] }>('/api/admin/subscriptions'),
        api.get<{ users: UserLite[] }>('/api/admin/users?page=1&page_size=100'),
      ])
      setPlans(p.plans ?? [])
      setSubs(s.subscriptions ?? [])
      setUsers(u.users ?? [])
    } catch { /* keep empty */ } finally { setLoading(false) }
  }, [])

  useEffect(() => { load() }, [load])

  const createPlan = async () => {
    setBusy(true); setError('')
    try {
      await api.post('/api/admin/plans', {
        name: pf.name.trim(), description: pf.description.trim() || undefined,
        price_cents: Math.round((parseFloat(pf.price || '0') || 0) * 100),
        quota: pf.quota.trim() === '' ? 0 : parseInt(pf.quota, 10),
        group: pf.group, duration_days: parseInt(pf.days || '30', 10),
      })
      setDialog({ kind: 'none' })
      setPf({ name: '', description: '', price: '', quota: '', group: 'default', days: '30' })
      await load()
    } catch (e) { setError(e instanceof APIError ? e.message : '') } finally { setBusy(false) }
  }

  const grant = async () => {
    setBusy(true); setError('')
    try {
      await api.post('/api/admin/subscriptions', { user_id: gf.userId, plan_id: gf.planId })
      setDialog({ kind: 'none' })
      await load()
    } catch (e) { setError(e instanceof APIError ? e.message : '') } finally { setBusy(false) }
  }

  const delPlan = async (id: number) => {
    try { await api.delete(`/api/admin/plans/${id}`); await load() }
    catch (e) { setError(e instanceof APIError ? e.message : '') }
  }

  const price = (cents: number) => cents === 0 ? t('plans.free') : `$${(cents / 100).toFixed(2)}`
  const inputCls = 'w-full min-h-[40px] px-3 py-2 border border-input rounded-lg text-sm bg-background focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary'

  return (
    <div className="max-w-5xl">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h2 className="text-xl font-semibold">{t('plans.title')}</h2>
          <p className="text-sm text-muted-foreground mt-0.5">{t('plans.subtitle')}</p>
        </div>
        <div className="flex gap-2">
          <button onClick={() => setDialog({ kind: 'grant' })} disabled={plans.length === 0 || users.length === 0}
            className="inline-flex items-center gap-2 px-3 py-2 text-sm border border-border rounded-lg hover:bg-muted disabled:opacity-50 cursor-pointer transition-colors">
            <UserPlus size={15} /> {t('plans.grant')}
          </button>
          <button onClick={() => setDialog({ kind: 'plan' })}
            className="inline-flex items-center gap-2 px-3 py-2 text-sm bg-primary text-primary-foreground rounded-lg hover:bg-primary-dark cursor-pointer transition-colors">
            <Plus size={15} /> {t('plans.add')}
          </button>
        </div>
      </div>

      {error && <p className="text-sm text-destructive mb-4">{error}</p>}

      {/* Plans */}
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3 mb-8">
        {loading && plans.length === 0 ? (
          <div className="py-8 text-center text-muted-foreground text-sm col-span-full">…</div>
        ) : plans.length === 0 ? (
          <div className="py-8 text-center text-muted-foreground text-sm col-span-full">{t('plans.empty')}</div>
        ) : plans.map((p) => (
          <div key={p.id} className={`bg-card border rounded-2xl p-4 ${p.status === 'active' ? 'border-border' : 'border-border opacity-60'}`}>
            <div className="flex items-start justify-between">
              <div>
                <div className="flex items-center gap-2">
                  <Package size={15} className="text-primary" />
                  <span className="font-semibold">{p.name}</span>
                </div>
                {p.description && <p className="text-xs text-muted-foreground mt-1">{p.description}</p>}
              </div>
              <button onClick={() => delPlan(p.id)} title={t('users.delete')}
                className="p-1.5 rounded-lg hover:bg-red-500/10 text-muted-foreground hover:text-red-500 cursor-pointer"><Trash2 size={13} /></button>
            </div>
            <div className="mt-3 text-lg font-bold">{price(p.price_cents)}</div>
            <div className="flex flex-wrap gap-1.5 mt-2">
              <span className="px-2 py-0.5 rounded-full bg-muted text-xs">{t('users.colGroup')}: {p.group}</span>
              <span className="px-2 py-0.5 rounded-full bg-muted text-xs">{p.quota === -1 ? t('users.unlimited') : `${p.quota}`}</span>
              <span className="px-2 py-0.5 rounded-full bg-muted text-xs">{p.duration_days === 0 ? t('plans.perpetual') : `${p.duration_days}d`}</span>
            </div>
          </div>
        ))}
      </div>

      {/* Subscriptions */}
      <h3 className="font-medium mb-3 text-sm">{t('plans.grants')}</h3>
      <div className="border border-border rounded-xl overflow-hidden bg-card">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border bg-muted/50 text-left">
              <th className="px-4 py-3 font-medium text-muted-foreground">#</th>
              <th className="px-4 py-3 font-medium text-muted-foreground">{t('plans.plan')}</th>
              <th className="px-4 py-3 font-medium text-muted-foreground">{t('plans.user')}</th>
              <th className="px-4 py-3 font-medium text-muted-foreground">{t('plans.expires')}</th>
              <th className="px-4 py-3 font-medium text-muted-foreground">{t('users.colStatus')}</th>
            </tr>
          </thead>
          <tbody>
            {subs.length === 0 ? (
              <tr><td colSpan={5} className="px-4 py-8 text-center text-muted-foreground">{t('plans.noGrants')}</td></tr>
            ) : subs.map((s) => (
              <tr key={s.id} className="border-b border-border last:border-0 hover:bg-muted/30">
                <td className="px-4 py-3 text-muted-foreground">{s.id}</td>
                <td className="px-4 py-3 font-medium">{s.plan_name}</td>
                <td className="px-4 py-3">{users.find((u) => u.id === s.user_id)?.username ?? `#${s.user_id}`}</td>
                <td className="px-4 py-3 text-xs text-muted-foreground">
                  {s.expires_at === '0001-01-01T00:00:00Z' ? t('plans.perpetual') : new Date(s.expires_at).toLocaleDateString()}
                </td>
                <td className="px-4 py-3">
                  <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${
                    s.status === 'active' ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400'
                    : s.status === 'expired' ? 'bg-amber-500/10 text-amber-600 dark:text-amber-400'
                    : 'bg-muted text-muted-foreground'}`}>
                    {s.status}
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Dialogs */}
      {dialog.kind !== 'none' && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
          <div className="absolute inset-0 bg-black/50" onClick={() => !busy && setDialog({ kind: 'none' })} />
          <div className="relative w-full max-w-sm bg-card border border-border rounded-2xl p-5 shadow-lg">
            <div className="flex items-center justify-between mb-4">
              <h3 className="font-semibold">{dialog.kind === 'plan' ? t('plans.add') : t('plans.grant')}</h3>
              <button onClick={() => setDialog({ kind: 'none' })} className="p-1 rounded-lg hover:bg-muted cursor-pointer"><X size={15} /></button>
            </div>
            {dialog.kind === 'plan' ? (
              <div className="space-y-3">
                <input className={inputCls} placeholder={t('plans.fName')} value={pf.name} onChange={(e) => setPf({ ...pf, name: e.target.value })} />
                <input className={inputCls} placeholder={t('plans.fDesc')} value={pf.description} onChange={(e) => setPf({ ...pf, description: e.target.value })} />
                <div className="grid grid-cols-2 gap-2">
                  <input className={inputCls} placeholder={t('plans.fPrice')} value={pf.price} inputMode="decimal"
                    onChange={(e) => setPf({ ...pf, price: e.target.value.replace(/[^\d.]/g, '') })} />
                  <input className={inputCls} placeholder={t('plans.fQuota')} value={pf.quota} inputMode="numeric"
                    onChange={(e) => setPf({ ...pf, quota: e.target.value.replace(/[^\d-]/g, '') })} />
                </div>
                <div className="grid grid-cols-2 gap-2">
                  <input className={inputCls} placeholder={t('plans.fGroup')} value={pf.group} onChange={(e) => setPf({ ...pf, group: e.target.value })} />
                  <input className={inputCls} placeholder={t('plans.fDays')} value={pf.days} inputMode="numeric"
                    onChange={(e) => setPf({ ...pf, days: e.target.value.replace(/\D/g, '') })} />
                </div>
                <button onClick={createPlan} disabled={busy || !pf.name.trim()}
                  className="w-full min-h-[40px] text-sm bg-primary text-primary-foreground rounded-lg hover:bg-primary-dark disabled:opacity-50 cursor-pointer inline-flex items-center justify-center gap-1.5">
                  {busy ? <RefreshCw size={14} className="animate-spin" /> : <Check size={14} />} {t('users.confirm')}
                </button>
              </div>
            ) : (
              <div className="space-y-3">
                <select className={inputCls} value={gf.userId} onChange={(e) => setGf({ ...gf, userId: parseInt(e.target.value, 10) })}>
                  <option value={0}>{t('plans.selUser')}</option>
                  {users.map((u) => <option key={u.id} value={u.id}>{u.username}</option>)}
                </select>
                <select className={inputCls} value={gf.planId} onChange={(e) => setGf({ ...gf, planId: parseInt(e.target.value, 10) })}>
                  <option value={0}>{t('plans.selPlan')}</option>
                  {plans.map((p) => <option key={p.id} value={p.id}>{p.name}</option>)}
                </select>
                <button onClick={grant} disabled={busy || !gf.userId || !gf.planId}
                  className="w-full min-h-[40px] text-sm bg-primary text-primary-foreground rounded-lg hover:bg-primary-dark disabled:opacity-50 cursor-pointer inline-flex items-center justify-center gap-1.5">
                  {busy ? <RefreshCw size={14} className="animate-spin" /> : <Check size={14} />} {t('users.confirm')}
                </button>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
