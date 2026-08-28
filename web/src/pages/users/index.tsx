import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Search, RefreshCw, ChevronLeft, ChevronRight } from 'lucide-react'
import { PageHeader } from '../../components/ui/primitives'
import { api } from '../../lib/api'
import { useUsers, type UserDialog } from './use-users'
import UsersTable from './users-table'
import UserDialogs from './user-dialogs'

export default function Users() {
  const { t } = useTranslation()
  const { users, page, setPage, loading, busyId, groups, load, act } = useUsers()
  const [query, setQuery] = useState('')
  const [searchInput, setSearchInput] = useState('')
  const [dialog, setDialog] = useState<UserDialog>({ kind: 'none' })

  const totalPages = Math.max(1, Math.ceil(page.total / page.page_size))

  const refresh = () => load(query, page.page)

  const open = (d: UserDialog) => setDialog(d)

  return (
    <div>
      <PageHeader title={t('users.title')} subtitle={t('users.subtitle')} actions={
        <button onClick={refresh}
          className="inline-flex items-center gap-2 px-3 py-2 text-sm border border-border rounded-lg hover:bg-muted transition-colors cursor-pointer">
          <RefreshCw size={15} strokeWidth={2} className={loading ? 'animate-spin' : ''} />
          <span className="hidden sm:inline">{t('users.refresh')}</span>
        </button>
      } />

      <div className="relative mb-4">
        <Search size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
        <input value={searchInput} onChange={(e) => setSearchInput(e.target.value)}
          onKeyDown={(e) => { if (e.key === 'Enter') { setQuery(searchInput.trim()); setPage({ ...page, page: 1 }); load(searchInput.trim(), 1) } }}
          placeholder={t('users.search')}
          className="w-full min-h-[40px] pl-9 pr-3 py-2 border border-input rounded-lg text-sm bg-background focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary" />
      </div>

      <UsersTable
        users={users} loading={loading} busyId={busyId} groups={groups}
        onRole={(u, role) => act(u, () => api.put(`/api/admin/users/${u.id}/role`, { role }).then(() => { setDialog({ kind: 'none' }); refresh() }))}
        onStatus={(u, status) => act(u, () => api.put(`/api/admin/users/${u.id}/status`, { status }).then(refresh))}
        onGroup={(u, group) => act(u, () => api.put(`/api/admin/users/${u.id}/group`, { group }).then(refresh))}
        onQuota={(u) => open({ kind: 'quota', user: u })}
        onPassword={(u) => open({ kind: 'password', user: u })}
        onDelete={(u) => open({ kind: 'delete', user: u })}
      />

      <div className="flex items-center justify-between mt-4 text-sm text-muted-foreground">
        <span>{t('users.pageInfo', { total: page.total, page: page.page, pages: totalPages })}</span>
        <div className="flex items-center gap-2">
          <button disabled={page.page <= 1}
            onClick={() => { const p = page.page - 1; setPage({ ...page, page: p }); load(query, p) }}
            className="p-2 border border-border rounded-lg hover:bg-muted disabled:opacity-40 cursor-pointer disabled:cursor-not-allowed"><ChevronLeft size={15} /></button>
          <button disabled={page.page >= totalPages}
            onClick={() => { const p = page.page + 1; setPage({ ...page, page: p }); load(query, p) }}
            className="p-2 border border-border rounded-lg hover:bg-muted disabled:opacity-40 cursor-pointer disabled:cursor-not-allowed"><ChevronRight size={15} /></button>
        </div>
      </div>

      <UserDialogs dialog={dialog} onClose={() => setDialog({ kind: 'none' })} onRefresh={refresh} query={query} page={page.page} />
    </div>
  )
}