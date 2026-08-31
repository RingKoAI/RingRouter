import { useTranslation } from 'react-i18next'
import { useState } from 'react'
import { Copy, Check, Loader2 } from 'lucide-react'
import { toast } from 'sonner'

/**
 * Create-key card. On success the freshly minted secret is shown once with a
 * copy affordance — it is never retrievable again.
 */
export default function CreateKeyCard({ onCreated }: {
  onCreated: (name: string) => Promise<{ key: string } | { error: string } | null>
}) {
  const { t } = useTranslation()
  const [name, setName] = useState('')
  const [busy, setBusy] = useState(false)
  const [newKey, setNewKey] = useState('')
  const [copied, setCopied] = useState(false)
  const [error, setError] = useState('')

  const submit = async () => {
    setBusy(true); setError(''); setNewKey(''); setCopied(false)
    const result = await onCreated(name)
    setBusy(false)
    if (result && 'key' in result && result.key) {
      setNewKey(result.key)
      setName('')
    } else if (result && 'error' in result) {
      setError(result.error)
    }
  }

  const copy = () => {
    navigator.clipboard.writeText(newKey).then(() => {
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    })
  }

  const inputCls =
    'flex-1 min-h-[40px] px-3 py-2 border border-input rounded-lg text-sm bg-background focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary'

  return (
    <div className="bg-card border border-border rounded-2xl p-5 mb-4">
      <div className="flex gap-2">
        <input
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder={t('keys.nameHint')}
          className={inputCls}
          maxLength={64}
          onKeyDown={(e) => { if (e.key === 'Enter' && name.trim()) submit() }}
        />
        <button
          onClick={submit}
          disabled={busy || !name.trim()}
          className="inline-flex items-center gap-2 min-h-[40px] px-4 text-sm bg-primary text-primary-foreground rounded-lg hover:bg-primary-dark disabled:opacity-50 transition-colors cursor-pointer whitespace-nowrap"
        >
          {busy ? <Loader2 size={13} className="animate-spin" /> : null}
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
      {error && <p className="text-sm text-destructive mt-2" role="alert">{error}</p>}
    </div>
  )
}
