import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { X, Check, Loader2 } from 'lucide-react'
import { toast } from 'sonner'
import { APIError } from '../../lib/api'
import { type UserDialog } from './use-users'
import { api } from '../../lib/api'

interface Props {
  dialog: UserDialog
  onClose: () => void
  onRefresh: () => void
  groups: string[]
  query: string
  page: number
}

export default function UserDialogs({ dialog, onClose, onRefresh, groups }: Props) {
  const { t } = useTranslation()
  const [value, setValue] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [createForm, setCreateForm] = useState({ username: '', email: '', password: '', role: 'user', group: 'default' })

  useEffect(() => {
    if (dialog.kind !== 'none') {
      setValue(dialog.kind === 'quota' ? String(dialog.user.quota) : '')
      setError('')
      if (dialog.kind === 'create') {
        setCreateForm({ username: '', email: '', password: '', role: 'user', group: groups[0] || 'default' })
      }
    }
  }, [dialog, groups])

  if (dialog.kind === 'none') return null

  const submit = async () => {
    setBusy(true); setError('')
    try {
      if (dialog.kind === 'create') {
        if (!/^[a-zA-Z0-9_-]{3,32}$/.test(createForm.username.trim())) {
          setError(t('users.invalidUsername'))
          return
        }
        if (createForm.password.length < 8 || createForm.password.length > 72) {
          setError(t('users.invalidPassword'))
          return
        }
        await api.post('/api/admin/users', {
          username: createForm.username.trim(), email: createForm.email.trim(),
          password: createForm.password, role: createForm.role, group: createForm.group,
        })
        toast.success(t('users.created'))
      } else if (dialog.kind === 'password') {
        const id = dialog.user.id
        await api.put(`/api/admin/users/${id}/password`, { password: value })
        toast.success(t('users.pwdReset'))
      } else if (dialog.kind === 'quota') {
        const n = parseInt(value, 10)
        if (Number.isNaN(n)) { setError(t('users.invalidNumber')); return }
        const id = dialog.user.id
        await api.put(`/api/admin/users/${id}/quota`, { quota: n })
        toast.success(t('users.quotaUpdated'))
      } else if (dialog.kind === 'delete') {
        const id = dialog.user.id
        await api.delete(`/api/admin/users/${id}`)
        toast.success(t('users.deleted'))
      }
      onClose()
      onRefresh()
    } catch (e) {
      setError(e instanceof APIError ? e.message : t('users.errNetwork'))
    } finally { setBusy(false) }
  }

  const inputCls = 'w-full min-h-[40px] px-3 py-2 border border-input rounded-lg text-sm bg-background focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary'

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div className="absolute inset-0 bg-black/50" onClick={() => !busy && onClose()} />
      <div className={`relative bg-card border-border shadow-lg anim-fade-up ${
        dialog.kind === 'create'
          ? 'absolute right-0 top-0 bottom-0 w-full max-w-md border-l p-6 rounded-l-2xl overflow-y-auto rr-scroll-thin animate-[slideIn_220ms_ease]'
          : 'w-full max-w-sm border rounded-2xl p-5'
      }`}>
        <div className="flex items-start justify-between mb-3">
          <h3 className="font-semibold">
            {dialog.kind === 'create' && t('users.create')}
            {dialog.kind === 'password' && t('users.resetPwd')}
            {dialog.kind === 'quota' && t('users.setQuota')}
            {dialog.kind === 'delete' && t('users.delete')}
          </h3>
          <button onClick={onClose} className="p-1 rounded-lg hover:bg-muted cursor-pointer"><X size={15} /></button>
        </div>
        <p className="text-sm text-muted-foreground mb-3">
          {dialog.kind === 'create'
            ? t('users.createHint')
            : dialog.kind === 'delete'
            ? t('users.confirmDelete', { name: dialog.user.username })
            : t('users.target', { name: dialog.user.username })}
        </p>

        {dialog.kind === 'create' && (
          <div className="space-y-2">
            <input className={inputCls} value={createForm.username} maxLength={32}
              onChange={(e) => setCreateForm({ ...createForm, username: e.target.value })}
              placeholder={t('users.username')} autoFocus autoComplete="username" />
            <input className={inputCls} value={createForm.email} maxLength={256}
              onChange={(e) => setCreateForm({ ...createForm, email: e.target.value })}
              placeholder={t('users.email')} autoComplete="email" />
            <input className={inputCls} type="password" value={createForm.password} maxLength={72}
              onChange={(e) => setCreateForm({ ...createForm, password: e.target.value })}
              placeholder={t('users.newPwdHint')} autoComplete="new-password" />
            <div className="grid grid-cols-2 gap-2">
              <select className={inputCls} value={createForm.role}
                onChange={(e) => setCreateForm({ ...createForm, role: e.target.value })}>
                <option value="user">{t('users.memberRole')}</option>
                <option value="admin">{t('users.adminRole')}</option>
              </select>
              <select className={inputCls} value={createForm.group}
                onChange={(e) => setCreateForm({ ...createForm, group: e.target.value })}>
                {(groups.length ? groups : ['default']).map((g) => <option key={g} value={g}>{g}</option>)}
              </select>
            </div>
          </div>
        )}
        {dialog.kind === 'password' && (
          <input type="password" value={value} onChange={(e) => setValue(e.target.value)} placeholder={t('users.newPwdHint')} className={inputCls} autoFocus />
        )}
        {dialog.kind === 'quota' && (
          <input type="text" inputMode="numeric" value={value} onChange={(e) => setValue(e.target.value.replace(/[^\d-]/g, ''))} placeholder={t('users.quotaHint')} className={inputCls} autoFocus />
        )}

        {error && <p className="text-sm text-destructive mt-2" role="alert">{error}</p>}

        <div className="flex gap-2 mt-4">
          <button onClick={submit} disabled={busy || (dialog.kind === 'password' && value.length < 8) || (dialog.kind === 'create' && (!createForm.username.trim() || !createForm.email.trim() || !createForm.password))}
            className="flex-1 min-h-[40px] text-sm bg-primary text-primary-foreground rounded-lg hover:bg-primary-dark disabled:opacity-50 transition-colors cursor-pointer inline-flex items-center justify-center gap-1.5 whitespace-nowrap">
            {busy ? <Loader2 size={14} className="animate-spin" /> : <Check size={14} />}
            {t('users.confirm')}
          </button>
          <button onClick={onClose} disabled={busy}
            className="flex-1 min-h-[40px] text-sm border border-input rounded-lg hover:bg-muted transition-colors cursor-pointer">{t('users.cancel')}</button>
        </div>
      </div>
    </div>
  )
}
