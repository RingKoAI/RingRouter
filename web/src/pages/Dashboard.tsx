import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Copy, Check } from 'lucide-react'
import { api } from '../lib/api'
import { useSite } from '../contexts/SiteContext'
import { Activity, Server, Zap } from 'lucide-react'

export default function Dashboard() {
  const { t } = useTranslation()
  const { siteName, apiBase, version } = useSite()
  const [status, setStatus] = useState<'ok' | 'error' | null>(null)
  const [copied, setCopied] = useState(false)

  useEffect(() => {
    api.get('/health').then((res) => {
      setStatus(res.status === 'ok' ? 'ok' : 'error')
    }).catch(() => setStatus('error'))
  }, [])

  const cards = [
    {
      label: t('dashboard.systemStatus'),
      value: status === 'ok' ? t('dashboard.online') : status === 'error' ? t('dashboard.offline') : '...',
      icon: Server,
      color: status === 'ok' ? 'text-success' : status === 'error' ? 'text-destructive' : 'text-muted-foreground',
      dot: status === 'ok' ? 'bg-success' : 'bg-destructive',
    },
    {
      label: t('dashboard.apiVersion'),
      value: 'v1',
      icon: Zap,
      color: 'text-primary',
      dot: 'bg-primary',
    },
    {
      label: t('dashboard.gatewayVersion'),
      value: version ? `v${version}` : '…',
      icon: Activity,
      color: 'text-muted-foreground',
      dot: 'bg-muted-foreground',
    },
  ]

  const curlCmd = `# ${t('dashboard.quickStartHint', { site: siteName })}
curl ${apiBase}/v1/chat/completions \\
  -H "Authorization: Bearer <your-key>" \\
  -H "Content-Type: application/json" \\
  -d '{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hello"}]}'`

  const copyCurl = async () => {
    try {
      await navigator.clipboard.writeText(curlCmd.replace(/\\\n\s*/g, ''))
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    } catch {
      // Clipboard unavailable (insecure context); the command stays selectable.
    }
  }

  return (
    <div className="max-w-5xl">
      <h2 className="text-xl font-semibold mb-6">{t('dashboard.title')}</h2>

      {/* Stat cards */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 mb-8">
        {cards.map((card) => {
          const Icon = card.icon
          return (
            <div key={card.label} className="bg-card rounded-2xl border border-border p-5 anim-fade-up">
              <div className="flex items-center justify-between mb-3">
                <span className="text-sm text-muted-foreground">{card.label}</span>
                <Icon size={18} className={card.color} strokeWidth={2} />
              </div>
              <div className="flex items-center gap-2">
                <span className={`w-2 h-2 rounded-full ${card.dot}`} />
                <p className="text-2xl font-semibold">{card.value}</p>
              </div>
            </div>
          )
        })}
      </div>

      {/* Quick start */}
      <div className="bg-card rounded-2xl border border-border p-6 anim-fade-up anim-delay-1">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h3 className="text-sm font-medium">{t('dashboard.quickStart')}</h3>
            <p className="text-xs text-muted-foreground mt-0.5">
              {t('dashboard.quickStartDesc', { site: siteName })}
            </p>
          </div>
          <button
            onClick={copyCurl}
            title={t('dashboard.copy')}
            className="p-2 rounded-lg text-muted-foreground hover:text-foreground hover:bg-muted transition-colors press cursor-pointer"
          >
            {copied ? <Check size={15} className="text-success" strokeWidth={2.2} /> : <Copy size={15} strokeWidth={2} />}
          </button>
        </div>
        <div className="bg-muted rounded-xl p-4 overflow-x-auto">
          <pre className="text-xs text-muted-foreground font-mono leading-relaxed">{curlCmd}</pre>
        </div>
        {version && (
          <p className="mt-3 text-xs text-muted-foreground">
            {siteName} · v{version}
          </p>
        )}
      </div>
    </div>
  )
}
