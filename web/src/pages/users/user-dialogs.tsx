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
  query: string
  page: number
}

export default function UserDialogs({ dialog, onClose, onRefresh }: Props) {
  const { t } = useTranslation()
  const [value, setValue] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    if (dialog.kind !== 'none') {
      setValue(dialog.kind === 'quota' ? String(dialog.user.quota) : '')
      setError('')
    }
  }, [dialog])

  if (dialog.kind === 'none') return null

  const submit = async () => {
    setBusy(true); setError('')
    const id = dialog.user.id
    try {
      if (dialog.kind === 'password') {
        await api.put(`/api/admin/users/${id}/password`, { password: value })
        toast.success(t('users.pwdReset'))
      } else if (dialog.kind === 'quota') {
        const n = parseInt(value, 10)
        if (Number.isNaN(n)) throw new APIError(400, 'invalid number')
        await api.put(`/api/admin/users/${id}/quota`, { quota: n })
        toast.success('Quota updated')
      } else if (dialog.kind === 'delete') {
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
      <div className="relative w-full max-w-sm bg-card border border-border rounded-2xl p-5 shadow-lg anim-fade-up">
        <div className="flex items-start justify-between mb-3">
          <h3 className="font-semibold">
            {dialog.kind === 'password' && t('users.resetPwd')}
            {dialog.kind === 'quota' && t('users.setQuota')}
            {dialog.kind === 'delete' && t('users.delete')}
          </h3>
          <button onClick={onClose} className="p-1 rounded-lg hover:bg-muted cursor-pointer"><X size={15} /></button>
        </div>
        <p className="text-sm text-muted-foreground mb-3">
          {dialog.kind === 'delete'
            ? t('users.confirmDelete', { name: dialog.user.username })
            : t('users.target', { name: dialog.user.username })}
        </p>

        {dialog.kind === 'password' && (
          <input type="text" value={value} onChange={(e) => setValue(e.target.value)} placeholder={t('users.newPwdHint')} className={inputCls} autoFocus />
        )}
        {dialog.kind === 'quota' && (
          <input type="text" inputMode="numeric" value={value} onChange={(e) => setValue(e.target.value.replace(/[^\d-]/g, ''))} placeholder={t('users.quotaHint')} className={inputCls} autoFocus />
        )}

        {error && <p className="text-sm text-destructive mt-2" role="alert">{error}</p>}

        <div className="flex gap-2 mt-4">
          <button onClick={submit} disabled={busy || (dialog.kind === 'password' && value.length < 8)}
            className="flex-1 min-h-[40px] text-sm bg-primary text-primary-foreground rounded-lg hover:bg-primary-dark disabled:opacity-50 transition-colors cursor-pointer inline-flex items-center justify-center gap-1.5">
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