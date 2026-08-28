import { useState, useEffect, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { KeyRound, Plus, Trash2, Power, RefreshCw, Copy, Check, Search } from 'lucide-react'
import { api, APIError } from '../lib/api'

interface Token {
  id: number
  name: string
  key_masked: string
  group: string
  status: 'active' | 'disabled'
  quota: number
  used_quota: number
  created_at: string
}

export default function Keys() {
  const { t } = useTranslation()
  const [tokens, setTokens] = useState<Token[]>([])
  const [loading, setLoading] = useState(true)
  const [creating, setCreating] = useState(false)
  const [name, setName] = useState('')
  const [newKey, setNewKey] = useState('')
  const [copied, setCopied] = useState(false)
  const [error, setError] = useState('')
  const [query, setQuery] = useState('')
  const [searchInput, setSearchInput] = useState('')

  const load = useCallback(async (q: string) => {
    setLoading(true)
    try {
      const d = await api.get<{ tokens: Token[] }>(`/api/tokens?q=${encodeURIComponent(q)}`)
      setTokens(d.tokens ?? [])
    } catch { setTokens([]) } finally { setLoading(false) }
  }, [])

  useEffect(() => { load('') }, [load])

  const search = () => {
    const q = searchInput.trim()
    setQuery(q)
    load(q)
  }

  const create = async () => {
    setError(''); setCreating(true); setNewKey('')
    try {
      const d = await api.post<{ token: { key: string } }>('/api/tokens', { name: name.trim() })
      setNewKey(d.token.key)
      setName('')
      await load(query)
    } catch (e) { setError(e instanceof APIError ? e.message : t('users.errNetwork')) }
    finally { setCreating(false) }
  }

  const toggle = async (tok: Token) => {
    try {
      await api.put(`/api/tokens/${tok.id}`, { status: tok.status === 'active' ? 'disabled' : 'active' })
      await load(query)
    } catch (e) { setError(e instanceof APIError ? e.message : '') }
  }

  const remove = async (id: number) => {
    try { await api.delete(`/api/tokens/${id}`); await load(query) }
    catch (e) { setError(e instanceof APIError ? e.message : '') }
  }

  const copy = () => {
    navigator.clipboard.writeText(newKey).then(() => { setCopied(true); setTimeout(() => setCopied(false), 1500) })
  }

  const inputCls = 'flex-1 min-h-[40px] px-3 py-2 border border-input rounded-lg text-sm bg-background focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary'

  return (
    <div className="max-w-4xl">
      <div className="mb-6">
        <h2 className="text-xl font-semibold">{t('keys.title')}</h2>
        <p className="text-sm text-muted-foreground mt-0.5">{t('keys.subtitle')}</p>
      </div>

      {/* Search */}
      <div className="relative mb-4">
        <Search size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
        <input value={searchInput} onChange={(e) => setSearchInput(e.target.value)}
          onKeyDown={(e) => { if (e.key === 'Enter') search() }}
          placeholder={t('keys.search')}
          className="w-full min-h-[40px] pl-9 pr-3 py-2 border border-input rounded-lg text-sm bg-background focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary" />
      </div>

      {/* Create */}
      <div className="bg-card border border-border rounded-2xl p-5 mb-4">
        <div className="flex gap-2">
          <input value={name} onChange={(e) => setName(e.target.value)} placeholder={t('keys.nameHint')} className={inputCls} maxLength={64}
            onKeyDown={(e) => { if (e.key === 'Enter' && name.trim()) create() }} />
          <button onClick={create} disabled={creating || !name.trim()}
            className="inline-flex items-center gap-2 min-h-[40px] px-4 text-sm bg-primary text-primary-foreground rounded-lg hover:bg-primary-dark disabled:opacity-50 transition-colors cursor-pointer whitespace-nowrap">
            {creating ? <RefreshCw size={14} className="animate-spin" /> : <Plus size={14} />}
            {t('keys.create')}
          </button>
        </div>
        {newKey && (
          <div className="mt-3 p-3 rounded-lg bg-emerald-500/10 border border-emerald-500/30">
            <p className="text-xs text-emerald-700 dark:text-emerald-300 mb-1.5 font-medium">{t('keys.onetime')}</p>
            <div className="flex items-center gap-2">
              <code className="flex-1 text-xs font-mono break-all">{newKey}</code>
              <button onClick={copy} className="p-1.5 rounded-lg hover:bg-emerald-500/20 cursor-pointer shrink-0">
                {copied ? <Check size={14} className="text-emerald-600" /> : <Copy size={14} className="text-emerald-600" />}
              </button>
            </div>
          </div>
        )}
        {error && <p className="text-sm text-destructive mt-2">{error}</p>}
      </div>

      {/* List */}
      <div className="border border-border rounded-xl overflow-hidden bg-card">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border bg-muted/50 text-left">
              <th className="px-4 py-3 font-medium text-muted-foreground">{t('users.colUser')}</th>
              <th className="px-4 py-3 font-medium text-muted-foreground">Key</th>
              <th className="px-4 py-3 font-medium text-muted-foreground">{t('users.colGroup')}</th>
              <th className="px-4 py-3 font-medium text-muted-foreground">{t('users.colStatus')}</th>
              <th className="px-4 py-3 font-medium text-muted-foreground text-right">{t('users.colActions')}</th>
            </tr>
          </thead>
          <tbody>
            {loading && tokens.length === 0 ? (
              <tr><td colSpan={5} className="px-4 py-8 text-center text-muted-foreground">…</td></tr>
            ) : tokens.length === 0 ? (
              <tr><td colSpan={5} className="px-4 py-8 text-center text-muted-foreground">{t('keys.empty')}</td></tr>
            ) : tokens.map((tok) => (
              <tr key={tok.id} className="border-b border-border last:border-0 hover:bg-muted/30 transition-colors">
                <td className="px-4 py-3 font-medium flex items-center gap-2"><KeyRound size={14} className="text-muted-foreground" />{tok.name}</td>
                <td className="px-4 py-3 font-mono text-xs text-muted-foreground">{tok.key_masked}</td>
                <td className="px-4 py-3">{tok.group}</td>
                <td className="px-4 py-3">
                  <span className={`inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium ${
                    tok.status === 'active' ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400' : 'bg-red-500/10 text-red-600 dark:text-red-400'}`}>
                    <span className={`w-1.5 h-1.5 rounded-full ${tok.status === 'active' ? 'bg-emerald-500' : 'bg-red-500'}`} />
                    {tok.status === 'active' ? t('users.stActive') : t('users.stDisabled')}
                  </span>
                </td>
                <td className="px-4 py-3">
                  <div className="flex justify-end gap-1">
                    <button title={tok.status === 'active' ? t('users.disable') : t('users.enable')} onClick={() => toggle(tok)}
                      className="p-1.5 rounded-lg hover:bg-muted text-muted-foreground hover:text-foreground cursor-pointer"><Power size={14} /></button>
                    <button title={t('users.delete')} onClick={() => remove(tok.id)}
                      className="p-1.5 rounded-lg hover:bg-red-500/10 text-muted-foreground hover:text-red-500 cursor-pointer"><Trash2 size={14} /></button>
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
