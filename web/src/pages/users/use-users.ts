import { useState, useEffect, useCallback } from 'react'
import { toast } from 'sonner'
import { useTranslation } from 'react-i18next'
import { api, APIError } from '../../lib/api'

export interface User {
  id: number
  username: string
  email: string
  display_name: string
  role: 'admin' | 'user'
  group: string
  quota: number
  used_quota: number
  status: 'active' | 'disabled'
  created_at: string
}

export interface Pagination { page: number; page_size: number; total: number }

export type UserDialog =
  | { kind: 'none' }
  | { kind: 'create' }
  | { kind: 'password'; user: User }
  | { kind: 'quota'; user: User }
  | { kind: 'delete'; user: User }

/**
 * Admin user list: search + paginated fetch, and role/status/group/quota/
 * password mutations. Errors toast; success feedback supplied by callers.
 */
export function useUsers() {
  const { t } = useTranslation()
  const [users, setUsers] = useState<User[]>([])
  const [page, setPage] = useState<Pagination>({ page: 1, page_size: 20, total: 0 })
  const [loading, setLoading] = useState(true)
  const [busyId, setBusyId] = useState(0)
  const [groups, setGroups] = useState<string[]>([])

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

  const act = useCallback(async (user: User, fn: () => Promise<unknown>): Promise<void> => {
    setBusyId(user.id)
    try {
      await fn()
      toast.success(t('users.updated'))
    } catch (e) {
      if (e instanceof APIError) toast.error(e.message)
    } finally {
      setBusyId(0)
    }
  }, [])

  return { users, page, setPage, loading, busyId, groups, load, act }
}
