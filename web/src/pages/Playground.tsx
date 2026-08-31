import { useState, useRef, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { Send, Loader2, Eraser } from 'lucide-react'

interface Msg { role: 'user' | 'assistant'; content: string }

export default function Playground() {
  const { t } = useTranslation()
  const [model, setModel] = useState('')
  const [input, setInput] = useState('')
  const [msgs, setMsgs] = useState<Msg[]>([])
  const [streaming, setStreaming] = useState(false)
  const [error, setError] = useState('')
  const bottomRef = useRef<HTMLDivElement>(null)

  // The API key lives in sessionStorage only: it vanishes when the tab
  // closes, shrinking the window in which XSS or a shared machine could
  // harvest it (the rest of the app uses HttpOnly cookies).
  const [key, setKey] = useState(() => sessionStorage.getItem('rr_playground_key') ?? '')

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [msgs])

  const send = async () => {
    const text = input.trim()
    if (!text || streaming) return
    if (!key.trim() || !model.trim()) {
      setError(t('pg.needKeyModel'))
      return
    }
    setError('')
    sessionStorage.setItem('rr_playground_key', key.trim())

    const history = [...msgs, { role: 'user' as const, content: text }]
    setMsgs(history)
    setInput('')
    setStreaming(true)

    try {
      const res = await fetch('/v1/chat/completions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${key.trim()}` },
        body: JSON.stringify({
          model: model.trim(),
          messages: history.map((m) => ({ role: m.role, content: m.content })),
          stream: true,
        }),
      })
      if (!res.ok || !res.body) {
        const d = await res.json().catch(() => ({}))
        throw new Error(d.error?.message || d.error || `HTTP ${res.status}`)
      }

      const reader = res.body.getReader()
      const decoder = new TextDecoder()
      let buf = ''
      let acc = ''
      setMsgs([...history, { role: 'assistant', content: '' }])

      for (;;) {
        const { done, value } = await reader.read()
        if (done) break
        buf += decoder.decode(value, { stream: true })
        const lines = buf.split('\n')
        buf = lines.pop() ?? ''
        for (const line of lines) {
          const s = line.trim()
          if (!s.startsWith('data:')) continue
          const payload = s.slice(5).trim()
          if (payload === '[DONE]') continue
          try {
            const j = JSON.parse(payload)
            const delta = j.choices?.[0]?.delta?.content ?? j.choices?.[0]?.message?.content ?? ''
            if (delta) {
              acc += delta
              setMsgs([...history, { role: 'assistant', content: acc }])
            }
          } catch { /* partial frame */ }
        }
      }
      if (!acc) setMsgs([...history, { role: 'assistant', content: t('pg.emptyResponse') }])
    } catch (e) {
      setError(e instanceof Error ? e.message : t('users.errNetwork'))
      setMsgs(history)
    } finally {
      setStreaming(false)
    }
  }

  const inputCls = 'min-h-[38px] px-3 py-2 border border-input rounded-lg text-sm bg-background focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary'

  return (
    <div className="max-w-3xl">
      <div className="mb-4">
        <h2 className="text-xl font-semibold">{t('pg.title')}</h2>
        <p className="text-sm text-muted-foreground mt-0.5">{t('pg.subtitle')}</p>
      </div>

      {/* Config */}
      <div className="flex flex-col sm:flex-row gap-2 mb-4">
        <input className={`${inputCls} flex-1 font-mono text-xs`} placeholder={t('pg.apiKey')}
          value={key} onChange={(e) => setKey(e.target.value)} type="password" />
        <input className={`${inputCls} sm:w-56 font-mono text-xs`} placeholder={t('pg.model')}
          value={model} onChange={(e) => setModel(e.target.value)} />
      </div>

      {/* Conversation */}
      <div className="border border-border rounded-xl bg-card min-h-[320px] max-h-[60vh] overflow-y-auto p-4 space-y-3 mb-4">
        {msgs.length === 0 ? (
          <div className="text-sm text-muted-foreground text-center py-16">{t('pg.hint')}</div>
        ) : msgs.map((m, i) => (
          <div key={i} className={`flex ${m.role === 'user' ? 'justify-end' : 'justify-start'}`}>
            <div className={`max-w-[85%] px-3.5 py-2.5 rounded-2xl text-sm whitespace-pre-wrap break-words ${
              m.role === 'user' ? 'bg-primary text-primary-foreground rounded-br-md' : 'bg-muted rounded-bl-md'}`}>
              {m.content || '…'}
            </div>
          </div>
        ))}
        <div ref={bottomRef} />
      </div>

      {error && <p className="text-sm text-destructive mb-3" role="alert">{error}</p>}

      {/* Input */}
      <div className="flex gap-2">
        <button onClick={() => { setMsgs([]); setError('') }} title={t('pg.clear')}
          className="p-2.5 border border-border rounded-lg hover:bg-muted cursor-pointer shrink-0"><Eraser size={15} /></button>
        <textarea value={input} onChange={(e) => setInput(e.target.value)}
          onKeyDown={(e) => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); send() } }}
          placeholder={t('pg.placeholder')} rows={2}
          className="flex-1 px-3 py-2 border border-input rounded-lg text-sm bg-background resize-none focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary" />
        <button onClick={send} disabled={streaming || !input.trim()}
          className="px-4 rounded-lg bg-primary text-primary-foreground hover:bg-primary-dark disabled:opacity-50 cursor-pointer transition-colors shrink-0">
          {streaming ? <Loader2 size={15} className="animate-spin" /> : <Send size={15} />}
        </button>
      </div>
    </div>
  )
}
