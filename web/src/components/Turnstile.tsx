import { useEffect, useRef, useCallback } from 'react'

declare global {
  interface Window {
    turnstile?: {
      render: (container: string | HTMLElement, options: Record<string, any>) => string
      reset: (widgetId: string) => void
      remove: (widgetId: string) => void
    }
  }
}

interface TurnstileProps {
  sitekey: string
  action?: string
  onVerify: (token: string) => void
  onExpire?: () => void
  theme?: 'light' | 'dark' | 'auto'
}

export default function Turnstile({ sitekey, action, onVerify, onExpire, theme = 'auto' }: TurnstileProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const widgetIdRef = useRef<string | null>(null)

  const handleVerify = useCallback((token: string) => {
    onVerify(token)
  }, [onVerify])

  const handleExpire = useCallback(() => {
    onExpire?.()
  }, [onExpire])

  useEffect(() => {
    if (!containerRef.current || !window.turnstile) return

    widgetIdRef.current = window.turnstile.render(containerRef.current, {
      sitekey,
      action,
      theme,
      callback: handleVerify,
      'expired-callback': handleExpire,
      'error-callback': handleExpire,
    })

    return () => {
      if (widgetIdRef.current && window.turnstile) {
        window.turnstile.remove(widgetIdRef.current)
        widgetIdRef.current = null
      }
    }
  }, [sitekey, action, theme, handleVerify, handleExpire])

  return <div ref={containerRef} className="cf-turnstile" />
}
