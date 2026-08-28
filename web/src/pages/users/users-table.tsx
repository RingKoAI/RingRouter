import { useTranslation } from 'react-i18next'
import { Users as UsersIcon, ShieldCheck, Shield, Power, KeyRound, Coins, Trash2 } from 'lucide-react'
import { type User } from './use-users'
import { EmptyState } from '../../components/ui/primitives'

interface Props {
  users: User[]
  loading: boolean
  busyId: number
  groups: string[]
  onRole: (u: User, role: 'admin' | 'user') => void
  onStatus: (u: User, status: 'active' | 'disabled') => void
  onGroup: (u: User, group: string) => void
  onQuota: (u: User) => void
  onPassword: (u: User) => void
  onDelete: (u: User) => void
}

function IconBtn({ title, onClick, children, busy, danger }: {
  title: string; onClick: () => void; children: React.ReactNode; busy?: boolean; danger?: boolean
}) {
  return (
    <button title={title} onClick={onClick} disabled={busy}
      className={`p-1.5 rounded-lg transition-colors cursor-pointer disabled:opacity-40 ${
        danger ? 'hover:bg-red-500/10 text-muted-foreground hover:text-red-500' : 'hover:bg-muted text-muted-foreground hover:text-foreground'}`}>
      {children}
    </button>
  )
}

export default function UsersTable({ users, loading, busyId, groups, onRole, onStatus, onGroup, onQuota, onPassword, onDelete }: Props) {
  const { t } = useTranslation()

  return (
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
              <tr>
                <td colSpan={8} className="p-0">
                  {loading ? <div className="px-4 py-10 text-center text-muted-foreground">…</div>
                    : <EmptyState icon={UsersIcon} title={t('users.empty')} />}
                </td>
              </tr>
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
                  <select value={u.group} disabled={busyId === u.id || groups.length === 0}
                    onChange={(e) => onGroup(u, e.target.value)}
                    className="px-2 py-1 border border-input rounded-lg text-xs bg-background cursor-pointer hover:border-primary/50">
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
                  <button onClick={() => onQuota(u)} title={t('users.setQuota')}
                    className="font-mono text-xs hover:text-primary transition-colors cursor-pointer">
                    {u.quota === -1 ? t('users.unlimited') : u.quota}
                  </button>
                </td>
                <td className="px-4 py-3 text-xs text-muted-foreground whitespace-nowrap">
                  {new Date(u.created_at).toLocaleDateString()}
                </td>
                <td className="px-4 py-3">
                  <div className="flex items-center justify-end gap-1">
                    <IconBtn title={u.role === 'admin' ? t('users.demote') : t('users.promote')} busy={busyId === u.id}
                      onClick={() => onRole(u, u.role === 'admin' ? 'user' : 'admin')}><ShieldCheck size={14} /></IconBtn>
                    <IconBtn title={u.status === 'active' ? t('users.disable') : t('users.enable')} busy={busyId === u.id}
                      onClick={() => onStatus(u, u.status === 'active' ? 'disabled' : 'active')}><Power size={14} /></IconBtn>
                    <IconBtn title={t('users.resetPwd')} onClick={() => onPassword(u)}><KeyRound size={14} /></IconBtn>
                    <IconBtn title={t('users.setQuota')} onClick={() => onQuota(u)}><Coins size={14} /></IconBtn>
                    <IconBtn title={t('users.delete')} danger onClick={() => onDelete(u)}><Trash2 size={14} /></IconBtn>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
