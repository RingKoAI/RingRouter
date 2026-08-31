import { useState, useEffect, useCallback } from 'react'
import { toast } from 'sonner'
import { useTranslation } from 'react-i18next'
import { api, APIError } from '../../lib/api'

export interface Token {
  id: number
  name: string
  key_masked: string
  group: string
  status: 'active' | 'disabled'
  quota: number
  used_quota: number
  created_at: string
}

export type StatusFilter = 'all' | 'active' | 'disabled'

/**
 * Token list state: fetch (with server-side name search), create with
 * one-time reveal (the freshly minted key is RETURNED to the caller — the
 * backend never shows it again), enable/disable toggle, delete. All
 * mutations toast.
 */
export function useTokens() {
  const { t } = useTranslation()
  const [tokens, setTokens] = useState<Token[]>([])
  const [loading, setLoading] = useState(true)
  const [creating, setCreating] = useState(false)

  const load = useCallback(async (q: string) => {
    setLoading(true)
    try {
      const d = await api.get<{ tokens: Token[] }>(`/api/tokens?q=${encodeURIComponent(q)}`)
      setTokens(d.tokens ?? [])
    } catch {
      setTokens([])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { load('') }, [load])

  // Returns the one-time plaintext key on success, or null on failure (the
  // error message is already surfaced through the UI via the return path).
  const create = useCallback(async (name: string): Promise<{ key: string } | { error: string } | null> => {
    setCreating(true)
    try {
      const d = await api.post<{ token: { key: string } }>('/api/tokens', { name: name.trim() })
      toast.success(t('keys.created'))
      await load('')
      return { key: d.token.key }
    } catch (e) {
      return { error: e instanceof APIError ? e.message : t('keys.networkError') }
    } finally {
      setCreating(false)
    }
  }, [load, t])

  const toggle = useCallback(async (tok: Token) => {
    try {
      await api.put(`/api/tokens/${tok.id}`, { status: tok.status === 'active' ? 'disabled' : 'active' })
      toast.success(tok.status === 'active' ? t('keys.disabledToast') : t('keys.enabledToast'))
      await load('')
    } catch (e) {
      if (e instanceof APIError) toast.error(e.message)
    }
  }, [load, t])

  const remove = useCallback(async (id: number) => {
    try {
      await api.delete(`/api/tokens/${id}`)
      toast.success(t('keys.deleted'))
      await load('')
    } catch (e) {
      if (e instanceof APIError) toast.error(e.message)
    }
  }, [load, t])

  return { tokens, loading, creating, create, toggle, remove, reload: load }
}
