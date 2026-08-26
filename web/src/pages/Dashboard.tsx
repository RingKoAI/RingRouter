import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { api } from '../lib/api'
import { Activity, Server, Zap } from 'lucide-react'

export default function Dashboard() {
  const { t } = useTranslation()
  const [status, setStatus] = useState<'ok' | 'error' | null>(null)

  useEffect(() => {
    api.get('/health').then((res) => {
      setStatus(res.status === 'ok' ? 'ok' : 'error')
    }).catch(() => setStatus('error'))
  }, [])

  const cards = [
    {
      label: t('dashboard.systemStatus'),
      value: status === 'ok' ? t('dashboard.online') : t('dashboard.offline'),
      icon: Server,
      color: status === 'ok' ? 'text-green-500' : 'text-red-500',
    },
    {
      label: t('dashboard.apiVersion'),
      value: 'v1',
      icon: Zap,
      color: 'text-primary',
    },
    {
      label: t('dashboard.uptime'),
      value: '--',
      icon: Activity,
      color: 'text-text-muted',
    },
  ]

  return (
    <div>
      <h2 className="text-xl font-semibold mb-6">{t('dashboard.title')}</h2>

      <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-4 mb-8">
        {cards.map((card) => {
          const Icon = card.icon
          return (
            <div key={card.label} className="bg-white dark:bg-surface rounded-xl border border-border p-5">
              <div className="flex items-center justify-between mb-2">
                <span className="text-sm text-text-muted">{card.label}</span>
                <Icon size={20} className={card.color} />
              </div>
              <p className="text-2xl font-semibold">{card.value}</p>
            </div>
          )
        })}
      </div>

      <div className="bg-white dark:bg-surface rounded-xl border border-border p-6">
        <h3 className="text-sm font-medium mb-4">{t('dashboard.quickStart')}</h3>
        <div className="bg-surface-alt dark:bg-surface-alt rounded-lg p-4 overflow-x-auto">
          <pre className="text-xs text-text-muted">
{`# Test your RingRouter instance
curl http://localhost:3000/v1/chat/completions \\
  -H "Authorization: Bearer <your-key>" \\
  -H "Content-Type: application/json" \\
  -d '{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hello"}]}'`}
          </pre>
        </div>
      </div>
    </div>
  )
}