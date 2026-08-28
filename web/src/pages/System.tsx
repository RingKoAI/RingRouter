import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { Cpu, Database, HardDrive, MemoryStick, Clock, Globe } from 'lucide-react'
import { api } from '../lib/api'
import { useSite } from '../contexts/SiteContext'

interface Runtime {
  go_version?: string
  os?: string
  arch?: string
  num_goroutine?: number
  mem_alloc_mb?: number
  uptime_seconds?: number
  db_type?: string
  redis_enabled?: boolean
}

function Stat({ icon: Icon, label, value }: { icon: typeof Cpu; label: string; value: string }) {
  return (
    <div className="flex items-center gap-3 p-4 rounded-xl border border-border bg-card">
      <div className="w-9 h-9 rounded-lg bg-muted flex items-center justify-center text-muted-foreground shrink-0">
        <Icon size={16} />
      </div>
      <div className="min-w-0">
        <p className="text-xs text-muted-foreground">{label}</p>
        <p className="text-sm font-medium truncate">{value}</p>
      </div>
    </div>
  )
}

export default function System() {
  const { t } = useTranslation()
  const { siteName, version, usageMode } = useSite()
  const [rt, setRt] = useState<Runtime>({})

  useEffect(() => {
    api.get<Runtime>('/api/admin/system').then(setRt).catch(() => {})
  }, [])

  const uptime = (s?: number) => {
    if (!s && s !== 0) return '—'
    const h = Math.floor(s / 3600), m = Math.floor((s % 3600) / 60)
    return h > 0 ? `${h}h ${m}m` : `${m}m`
  }

  return (
    <div className="max-w-3xl">
      <div className="mb-6">
        <h2 className="text-xl font-semibold">{t('system.title')}</h2>
        <p className="text-sm text-muted-foreground mt-0.5">{t('system.subtitle')}</p>
      </div>

      <div className="grid gap-3 sm:grid-cols-2">
        <Stat icon={Globe} label={t('system.site')} value={`${siteName} · v${version}`} />
        <Stat icon={Cpu} label={t('system.usageMode')} value={usageMode} />
        <Stat icon={Database} label={t('system.db')} value={rt.db_type || '—'} />
        <Stat icon={HardDrive} label="Redis" value={rt.redis_enabled ? t('system.enabled') : t('system.disabled')} />
        <Stat icon={Cpu} label="Go" value={rt.go_version || '—'} />
        <Stat icon={MemoryStick} label={t('system.mem')} value={rt.mem_alloc_mb !== undefined ? `${rt.mem_alloc_mb} MB` : '—'} />
        <Stat icon={Clock} label={t('system.uptime')} value={uptime(rt.uptime_seconds)} />
        <Stat icon={Cpu} label="Runtime" value={rt.os && rt.arch ? `${rt.os}/${rt.arch}` : '—'} />
      </div>
    </div>
  )
}
