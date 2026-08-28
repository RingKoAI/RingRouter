import { useState, useEffect, useCallback } from 'react'
import { toast } from 'sonner'
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
 * one-time reveal, enable/disable toggle, delete. All mutations toast.
 */
export function useTokens() {
  const [tokens, setTokens] = useState<Token[]>([])
  const [loading, setLoading] = useState(true)
  const [creating, setCreating] = useState(false)
  const [newKey, setNewKey] = useState('')
  const [error, setError] = useState('')

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

  const create = useCallback(async (name: string): Promise<boolean> => {
    setError(''); setCreating(true); setNewKey('')
    try {
      const d = await api.post<{ token: { key: string } }>('/api/tokens', { name: name.trim() })
      setNewKey(d.token.key)
      toast.success('Key created')
      await load('')
      return true
    } catch (e) {
      setError(e instanceof APIError ? e.message : 'network error')
      return false
    } finally {
      setCreating(false)
    }
  }, [load])

  const toggle = useCallback(async (tok: Token) => {
    try {
      await api.put(`/api/tokens/${tok.id}`, { status: tok.status === 'active' ? 'disabled' : 'active' })
      toast.success(tok.status === 'active' ? 'Key disabled' : 'Key enabled')
      await load('')
    } catch (e) {
      if (e instanceof APIError) toast.error(e.message)
    }
  }, [load])

  const remove = useCallback(async (id: number) => {
    try {
      await api.delete(`/api/tokens/${id}`)
      toast.success('Key deleted')
      await load('')
    } catch (e) {
      if (e instanceof APIError) toast.error(e.message)
    }
  }, [load])

  return { tokens, loading, creating, newKey, error, create, toggle, remove, reload: load }
}
