import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from 'react'
import { api } from '../lib/api'

export interface SiteInfo {
  siteName: string
  usageMode: 'external' | 'self' | 'demo'
  version: string
  smtpConfigured: boolean
  turnstileEnabled: boolean
  turnstileSitekey: string
  passkeyEnabled: boolean
}

const DEFAULT_SITE: SiteInfo = {
  siteName: 'RingRouter',
  usageMode: 'external',
  version: '',
  smtpConfigured: false,
  turnstileEnabled: false,
  turnstileSitekey: '',
  passkeyEnabled: false,
}

interface SiteContextValue extends SiteInfo {
  /** Base URL for API requests, derived from the current origin. */
  apiBase: string
  refresh: () => Promise<void>
}

const SiteContext = createContext<SiteContextValue>({
  ...DEFAULT_SITE,
  apiBase: '',
  refresh: async () => {},
})

export function SiteProvider({ children }: { children: ReactNode }) {
  const [info, setInfo] = useState<SiteInfo>(DEFAULT_SITE)

  const refresh = useCallback(async () => {
    try {
      const d = await api.get<{
        site_name?: string
        usage_mode?: SiteInfo['usageMode']
        version?: string
        smtp_configured?: boolean
        turnstile_enabled?: boolean
        turnstile_sitekey?: string
        passkey_enabled?: boolean
      }>('/api/status')
      setInfo({
        siteName: d.site_name?.trim() || DEFAULT_SITE.siteName,
        usageMode: d.usage_mode || DEFAULT_SITE.usageMode,
        version: d.version || '',
        smtpConfigured: Boolean(d.smtp_configured),
        turnstileEnabled: Boolean(d.turnstile_enabled),
        turnstileSitekey: d.turnstile_sitekey || '',
        passkeyEnabled: Boolean(d.passkey_enabled),
      })
    } catch {
      // Backend unreachable: keep defaults so the UI stays branded.
    }
  }, [])

  useEffect(() => {
    refresh()
  }, [refresh])

  // Keep the document title in sync with the configured site name.
  useEffect(() => {
    document.title = info.siteName
  }, [info.siteName])

  const apiBase = typeof window !== 'undefined' ? window.location.origin : ''

  return (
    <SiteContext.Provider value={{ ...info, apiBase, refresh }}>
      {children}
    </SiteContext.Provider>
  )
}

export function useSite() {
  return useContext(SiteContext)
}
