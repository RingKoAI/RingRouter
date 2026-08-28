import { useState, useEffect, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { RefreshCw, ChevronLeft, ChevronRight } from 'lucide-react'
import { api } from '../lib/api'

interface LogEntry {
  id: number
  model_name: string
  prompt_tokens: number
  completion_tokens: number
  elapsed_ms: number
  status: string
  error_msg: string
  ip: string
  created_at: string
}

const PAGE_SIZE = 20

export default function Logs() {
  const { t } = useTranslation()
  const [logs, setLogs] = useState<LogEntry[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [model, setModel] = useState('')
  const [status, setStatus] = useState('')
  const [loading, setLoading] = useState(true)

  const load = useCallback(async (p: number) => {
    setLoading(true)
    try {
      const q = new URLSearchParams({ page: String(p), page_size: String(PAGE_SIZE) })
      if (model.trim()) q.set('model', model.trim())
      if (status) q.set('status', status)
      const d = await api.get<{ logs: LogEntry[]; pagination: { total: number } }>(`/api/logs?${q}`)
      setLogs(d.logs ?? [])
      setTotal(d.pagination?.total ?? 0)
      setPage(p)
    } catch { setLogs([]) } finally { setLoading(false) }
  }, [model, status])

  useEffect(() => { load(1) }, [load])

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))
  const selectCls = 'min-h-[36px] px-2 border border-input rounded-lg text-sm bg-background cursor-pointer'

  return (
    <div className="max-w-5xl">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 mb-6">
        <div>
          <h2 className="text-xl font-semibold">{t('logs.title')}</h2>
          <p className="text-sm text-muted-foreground mt-0.5">{t('logs.subtitle')}</p>
        </div>
        <div className="flex items-center gap-2">
          <input value={model} onChange={(e) => setModel(e.target.value)}
            onKeyDown={(e) => { if (e.key === 'Enter') load(1) }}
            placeholder={t('logs.modelFilter')} className="min-h-[36px] px-3 border border-input rounded-lg text-sm bg-background" />
          <select value={status} onChange={(e) => { setStatus(e.target.value); setTimeout(() => load(1), 0) }} className={selectCls}>
            <option value="">{t('logs.allStatus')}</option>
            <option value="success">{t('logs.success')}</option>
            <option value="failed">{t('logs.failed')}</option>
          </select>
          <button onClick={() => load(page)} className="p-2 border border-border rounded-lg hover:bg-muted cursor-pointer">
            <RefreshCw size={15} className={loading ? 'animate-spin' : ''} />
          </button>
        </div>
      </div>

      <div className="border border-border rounded-xl overflow-hidden bg-card">
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border bg-muted/50 text-left">
                <th className="px-4 py-3 font-medium text-muted-foreground">{t('logs.colModel')}</th>
                <th className="px-4 py-3 font-medium text-muted-foreground">{t('logs.colTokens')}</th>
                <th className="px-4 py-3 font-medium text-muted-foreground">{t('logs.colElapsed')}</th>
                <th className="px-4 py-3 font-medium text-muted-foreground">{t('users.colStatus')}</th>
                <th className="px-4 py-3 font-medium text-muted-foreground">{t('logs.colTime')}</th>
              </tr>
            </thead>
            <tbody>
              {loading && logs.length === 0 ? (
                <tr><td colSpan={5} className="px-4 py-8 text-center text-muted-foreground">…</td></tr>
              ) : logs.length === 0 ? (
                <tr><td colSpan={5} className="px-4 py-8 text-center text-muted-foreground">{t('logs.empty')}</td></tr>
              ) : logs.map((l) => (
                <tr key={l.id} className="border-b border-border last:border-0 hover:bg-muted/30 transition-colors">
                  <td className="px-4 py-3 font-medium">{l.model_name}</td>
                  <td className="px-4 py-3 font-mono text-xs text-muted-foreground">
                    {l.prompt_tokens + l.completion_tokens > 0 ? `${l.prompt_tokens}+${l.completion_tokens}` : '—'}
                  </td>
                  <td className="px-4 py-3 font-mono text-xs text-muted-foreground">{l.elapsed_ms > 0 ? `${l.elapsed_ms}ms` : '—'}</td>
                  <td className="px-4 py-3">
                    <span className={`inline-flex px-2 py-0.5 rounded-full text-xs font-medium ${
                      l.status === 'success' ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400' : 'bg-red-500/10 text-red-600 dark:text-red-400'}`}>
                      {l.status === 'success' ? t('logs.success') : t('logs.failed')}
                    </span>
                    {l.status !== 'success' && l.error_msg && (
                      <p className="text-xs text-muted-foreground mt-1 max-w-[280px] truncate" title={l.error_msg}>{l.error_msg}</p>
                    )}
                  </td>
                  <td className="px-4 py-3 text-xs text-muted-foreground whitespace-nowrap">{new Date(l.created_at).toLocaleString()}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      <div className="flex items-center justify-between mt-4 text-sm text-muted-foreground">
        <span>{t('users.pageInfo', { total, page, pages: totalPages })}</span>
        <div className="flex gap-2">
          <button disabled={page <= 1} onClick={() => load(page - 1)}
            className="p-2 border border-border rounded-lg hover:bg-muted disabled:opacity-40 cursor-pointer disabled:cursor-not-allowed"><ChevronLeft size={15} /></button>
          <button disabled={page >= totalPages} onClick={() => load(page + 1)}
            className="p-2 border border-border rounded-lg hover:bg-muted disabled:opacity-40 cursor-pointer disabled:cursor-not-allowed"><ChevronRight size={15} /></button>
        </div>
      </div>
    </div>
  )
}
