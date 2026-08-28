import { useState, useEffect, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Users as UsersIcon, Search, ShieldCheck, Shield, Power, Trash2,
  KeyRound, Coins, RefreshCw, ChevronLeft, ChevronRight, X, Check,
} from 'lucide-react'
import { toast } from 'sonner'
import { api, APIError } from '../lib/api'
import { TableSkeleton, EmptyState, PageHeader } from '../components/ui/primitives'

interface User {
  id: number
  username: string
  email: string
  display_name: string
  role: 'admin' | 'user'
  group: string
  quota: number
  status: 'active' | 'disabled'
  created_at: string
}

interface Pagination {
  page: number
  page_size: number
  total: number
}

type Dialog =
  | { kind: 'none' }
  | { kind: 'password'; user: User }
  | { kind: 'quota'; user: User }
  | { kind: 'delete'; user: User }

export default function Users() {
  const { t } = useTranslation()
  const [users, setUsers] = useState<User[]>([])
  const [page, setPage] = useState<Pagination>({ page: 1, page_size: 20, total: 0 })
  const [query, setQuery] = useState('')
  const [searchInput, setSearchInput] = useState('')
  const [groups, setGroups] = useState<string[]>([])
  const [loading, setLoading] = useState(true)
  const [busyId, setBusyId] = useState<number>(0)
  const [dialog, setDialog] = useState<Dialog>({ kind: 'none' })
  const [dialogValue, setDialogValue] = useState('')
  const [dialogError, setDialogError] = useState('')
  const [dialogBusy, setDialogBusy] = useState(false)

  const load = useCallback(async (q: string, p: number) => {
    setLoading(true)
    try {
      const d = await api.get<{ users: User[]; pagination: Pagination }>(
        `/api/admin/users?q=${encodeURIComponent(q)}&page=${p}&page_size=20`,
      )
      setUsers(d.users ?? [])
      setPage(d.pagination)
    } catch {
      setUsers([])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { load('', 1) }, [load])

  useEffect(() => {
    api.get<{ groups: { name: string }[] }>('/api/admin/groups')
      .then((d) => setGroups((d.groups ?? []).map((g) => g.name)))
      .catch(() => {})
  }, [])

  const act = async (user: User, fn: () => Promise<unknown>) => {
    setBusyId(user.id)
    try { await fn(); await load(query, page.page) } catch (e) {
      if (e instanceof APIError) toast.error(e.message)
    } finally { setBusyId(0) }
  }

  const totalPages = Math.max(1, Math.ceil(page.total / page.page_size))

  const openDialog = (d: Dialog, initial = '') => {
    setDialog(d); setDialogValue(initial); setDialogError('')
  }

  const submitDialog = async () => {
    if (dialog.kind === 'none') return
    setDialogBusy(true); setDialogError('')
    const id = dialog.user.id
    try {
      if (dialog.kind === 'password') {
        await api.put(`/api/admin/users/${id}/password`, { password: dialogValue })
      } else if (dialog.kind === 'quota') {
        const n = parseInt(dialogValue, 10)
        if (Number.isNaN(n)) throw new APIError(400, t('users.errQuota'))
        await api.put(`/api/admin/users/${id}/quota`, { quota: n })
      } else if (dialog.kind === 'delete') {
        await api.delete(`/api/admin/users/${id}`)
      }
      setDialog({ kind: 'none' })
      toast.success(dialog.kind === 'delete' ? t('users.deleted') : dialog.kind === 'password' ? t('users.pwdReset') : t('users.updated'))
      await load(query, page.page)
    } catch (e) {
      setDialogError(e instanceof APIError ? e.message : t('users.errNetwork'))
    } finally { setDialogBusy(false) }
  }

  const inputCls =
    'w-full min-h-[40px] px-3 py-2 border border-input rounded-lg text-sm bg-background ' +
    'focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary transition-all'

  return (
    <div className="max-w-6xl">
      <PageHeader title={t('users.title')} subtitle={t('users.subtitle')} actions={
        <button
          onClick={() => load(query, page.page)}
          className="inline-flex items-center gap-2 px-3 py-2 text-sm border border-border rounded-lg hover:bg-muted transition-colors cursor-pointer"
        >
          <RefreshCw size={15} strokeWidth={2} className={loading ? 'animate-spin' : ''} />
          <span className="hidden sm:inline">{t('users.refresh')}</span>
        </button>
      } />

      {/* Search */}
      <div className="relative mb-4">
        <Search size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
        <input
          value={searchInput}
          onChange={(e) => setSearchInput(e.target.value)}
          onKeyDown={(e) => { if (e.key === 'Enter') { setQuery(searchInput.trim()); setPage((p) => ({ ...p, page: 1 })); load(searchInput.trim(), 1) } }}
          placeholder={t('users.search')}
          className="w-full min-h-[40px] pl-9 pr-3 py-2 border border-input rounded-lg text-sm bg-background focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary"
        />
      </div>

      {/* Table */}
      <div className="border border-border rounded-xl overflow-hidden bg-card">
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border bg-muted/50 text-left">
                <th className="px-4 py-3 font-medium text-muted-foreground">ID</th>
                <th className="px-4 py-3 font-medium text-muted-foreground">{t('users.colUser')}</th>
                <th className="px-4 py-3 font-medium text-muted-foreground">{t('users.colRole')}</th>
                <th className="px-4 py-3 font-medium text-muted-foreground">{t('users.colGroup')}</th>
                <th className="px-4 py-3 font-medium text-muted-foreground">{t('users.colStatus')}</th>
                <th className="px-4 py-3 font-medium text-muted-foreground">{t('users.colQuota')}</th>
                <th className="px-4 py-3 font-medium text-muted-foreground">{t('users.colCreated')}</th>
                <th className="px-4 py-3 font-medium text-muted-foreground text-right">{t('users.colActions')}</th>
              </tr>
            </thead>
            <tbody>
              {users.length === 0 && (
                <tr><td colSpan={8} className="p-0">
                  {loading ? <div className="p-4"><TableSkeleton rows={6} cols={8} /></div>
                  : <EmptyState icon={UsersIcon} title={t('users.empty')} />}
                </td></tr>
              )}
              {users.map((u) => (
                <tr key={u.id} className="border-b border-border last:border-0 hover:bg-muted/30 transition-colors">
                  <td className="px-4 py-3 text-muted-foreground">{u.id}</td>
                  <td className="px-4 py-3">
                    <div className="font-medium flex items-center gap-2">
                      <UsersIcon size={14} className="text-muted-foreground" />
                      {u.username}
                    </div>
                    {u.email && <div className="text-xs text-muted-foreground mt-0.5">{u.email}</div>}
                  </td>
                  <td className="px-4 py-3">
                    <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium ${
                      u.role === 'admin'
                        ? 'bg-violet-500/10 text-violet-600 dark:text-violet-400'
                        : 'bg-muted text-muted-foreground'}`}>
                      {u.role === 'admin' ? <ShieldCheck size={12} /> : <Shield size={12} />}
                      {u.role === 'admin' ? t('users.roleAdmin') : t('users.roleUser')}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    <select
                      value={u.group}
                      disabled={busyId === u.id || groups.length === 0}
                      onChange={(e) => act(u, () => api.put(`/api/admin/users/${u.id}/group`, { group: e.target.value }))}
                      className="px-2 py-1 border border-input rounded-lg text-xs bg-background cursor-pointer hover:border-primary/50"
                    >
                      {groups.map((g) => <option key={g} value={g}>{g}</option>)}
                    </select>
                  </td>
                  <td className="px-4 py-3">
                    <span className={`inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium ${
                      u.status === 'active'
                        ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400'
                        : 'bg-red-500/10 text-red-600 dark:text-red-400'}`}>
                      <span className={`w-1.5 h-1.5 rounded-full ${u.status === 'active' ? 'bg-emerald-500' : 'bg-red-500'}`} />
                      {u.status === 'active' ? t('users.stActive') : t('users.stDisabled')}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    <button
                      onClick={() => openDialog({ kind: 'quota', user: u }, String(u.quota))}
                      className="font-mono text-xs hover:text-primary transition-colors cursor-pointer"
                      title={t('users.setQuota')}
                    >
                      {u.quota === -1 ? t('users.unlimited') : u.quota}
                    </button>
                  </td>
                  <td className="px-4 py-3 text-xs text-muted-foreground whitespace-nowrap">
                    {new Date(u.created_at).toLocaleDateString()}
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex items-center justify-end gap-1">
                      <IconBtn
                        title={u.role === 'admin' ? t('users.demote') : t('users.promote')}
                        busy={busyId === u.id}
                        onClick={() => act(u, () => api.put(`/api/admin/users/${u.id}/role`, { role: u.role === 'admin' ? 'user' : 'admin' }))}
                      >
                        <ShieldCheck size={14} />
                      </IconBtn>
                      <IconBtn
                        title={u.status === 'active' ? t('users.disable') : t('users.enable')}
                        busy={busyId === u.id}
                        onClick={() => act(u, () => api.put(`/api/admin/users/${u.id}/status`, { status: u.status === 'active' ? 'disabled' : 'active' }))}
                      >
                        <Power size={14} />
                      </IconBtn>
                      <IconBtn title={t('users.resetPwd')} onClick={() => openDialog({ kind: 'password', user: u })}>
                        <KeyRound size={14} />
                      </IconBtn>
                      <IconBtn title={t('users.setQuota')} onClick={() => openDialog({ kind: 'quota', user: u }, String(u.quota))}>
                        <Coins size={14} />
                      </IconBtn>
                      <IconBtn title={t('users.delete')} danger onClick={() => openDialog({ kind: 'delete', user: u })}>
                        <Trash2 size={14} />
                      </IconBtn>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Pagination */}
      <div className="flex items-center justify-between mt-4 text-sm text-muted-foreground">
        <span>{t('users.pageInfo', { total: page.total, page: page.page, pages: totalPages })}</span>
        <div className="flex items-center gap-2">
          <button
            disabled={page.page <= 1}
            onClick={() => { const p = page.page - 1; setPage({ ...page, page: p }); load(query, p) }}
            className="p-2 border border-border rounded-lg hover:bg-muted disabled:opacity-40 cursor-pointer disabled:cursor-not-allowed"
          ><ChevronLeft size={15} /></button>
          <button
            disabled={page.page >= totalPages}
            onClick={() => { const p = page.page + 1; setPage({ ...page, page: p }); load(query, p) }}
            className="p-2 border border-border rounded-lg hover:bg-muted disabled:opacity-40 cursor-pointer disabled:cursor-not-allowed"
          ><ChevronRight size={15} /></button>
        </div>
      </div>

      {/* Dialog */}
      {dialog.kind !== 'none' && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
          <div className="absolute inset-0 bg-black/50" onClick={() => !dialogBusy && setDialog({ kind: 'none' })} />
          <div className="relative w-full max-w-sm bg-card border border-border rounded-2xl p-5 shadow-lg anim-fade-up">
            <div className="flex items-start justify-between mb-3">
              <h3 className="font-semibold">
                {dialog.kind === 'password' && t('users.resetPwd')}
                {dialog.kind === 'quota' && t('users.setQuota')}
                {dialog.kind === 'delete' && t('users.delete')}
              </h3>
              <button onClick={() => setDialog({ kind: 'none' })} className="p-1 rounded-lg hover:bg-muted cursor-pointer"><X size={15} /></button>
            </div>
            <p className="text-sm text-muted-foreground mb-3">
              {dialog.kind === 'delete'
                ? t('users.confirmDelete', { name: dialog.user.username })
                : t('users.target', { name: dialog.user.username })}
            </p>

            {dialog.kind === 'password' && (
              <input
                type="text"
                value={dialogValue}
                onChange={(e) => setDialogValue(e.target.value)}
                placeholder={t('users.newPwdHint')}
                className={inputCls}
                autoFocus
              />
            )}
            {dialog.kind === 'quota' && (
              <input
                type="text"
                inputMode="numeric"
                value={dialogValue}
                onChange={(e) => setDialogValue(e.target.value.replace(/[^\d-]/g, ''))}
                placeholder={t('users.quotaHint')}
                className={inputCls}
                autoFocus
              />
            )}

            {dialogError && <p className="text-sm text-destructive mt-2" role="alert">{dialogError}</p>}

            <div className="flex gap-2 mt-4">
              <button
                onClick={submitDialog}
                disabled={dialogBusy || (dialog.kind === 'password' && dialogValue.length < 8) || (dialog.kind === 'quota' && dialogValue === '')}
                className="flex-1 min-h-[40px] text-sm bg-primary text-primary-foreground rounded-lg hover:bg-primary-dark disabled:opacity-50 transition-colors cursor-pointer inline-flex items-center justify-center gap-1.5"
              >
                <Check size={14} />
                {t('users.confirm')}
              </button>
              <button
                onClick={() => setDialog({ kind: 'none' })}
                disabled={dialogBusy}
                className="flex-1 min-h-[40px] text-sm border border-input rounded-lg hover:bg-muted transition-colors cursor-pointer"
              >{t('users.cancel')}</button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

function IconBtn({ title, onClick, children, danger, busy }: {
  title: string; onClick: () => void; children: React.ReactNode; danger?: boolean; busy?: boolean
}) {
  return (
    <button
      title={title}
      onClick={onClick}
      disabled={busy}
      className={`p-1.5 rounded-lg transition-colors cursor-pointer disabled:opacity-40 ${
        danger ? 'hover:bg-red-500/10 text-muted-foreground hover:text-red-500' : 'hover:bg-muted text-muted-foreground hover:text-foreground'}`}
    >
      {children}
    </button>
  )
}
