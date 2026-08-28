import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Sun, Moon, Monitor, Globe, Check, ChevronDown } from 'lucide-react'
import { useTheme } from '../contexts/ThemeContext'

export const languages = [
  { code: 'zh', label: '简体中文' },
  { code: 'zh-TW', label: '繁體中文（台灣）' },
  { code: 'zh-HK', label: '繁體中文（香港）' },
  { code: 'en', label: 'English' },
]

/**
 * Theme cycle + language dropdown actions shared by the dashboard header,
 * the public home header, and the auth layout.
 */
export default function ThemeLangActions({ compact = false }: { compact?: boolean }) {
  const { t, i18n } = useTranslation()
  const { theme, setTheme, resolved } = useTheme()
  const [langOpen, setLangOpen] = useState(false)
  const langRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!langOpen) return
    const onDown = (e: MouseEvent) => {
      if (langRef.current && !langRef.current.contains(e.target as Node)) {
        setLangOpen(false)
      }
    }
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setLangOpen(false)
    }
    document.addEventListener('mousedown', onDown)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onDown)
      document.removeEventListener('keydown', onKey)
    }
  }, [langOpen])

  const cycleTheme = () => {
    const order = ['light', 'dark', 'system'] as const
    const idx = order.indexOf(theme)
    setTheme(order[(idx + 1) % order.length])
  }

  const ThemeIcon = theme === 'dark' ? Moon : theme === 'light' ? Sun : Monitor
  const nextTheme = theme === 'light' ? 'dark' : theme === 'dark' ? 'system' : 'light'
  const currentLang = languages.find((l) => l.code === i18n.language) || languages[3]

  return (
    <div className="flex items-center gap-1">
      {/* Theme cycle */}
      <button
        onClick={cycleTheme}
        title={`${t('theme.' + theme)} → ${t('theme.' + nextTheme)}`}
        aria-label={t('theme.' + theme)}
        className="relative p-2 rounded-lg text-muted-foreground hover:bg-muted hover:text-foreground transition-colors cursor-pointer"
      >
        <ThemeIcon size={compact ? 16 : 17} strokeWidth={2} />
        {theme === 'system' && (
          <span
            className={`absolute bottom-1 right-1 w-1.5 h-1.5 rounded-full ${
              resolved === 'dark' ? 'bg-primary' : 'bg-primary/60'
            }`}
          />
        )}
      </button>

      {/* Language dropdown */}
      <div className="relative" ref={langRef}>
        <button
          onClick={() => setLangOpen(!langOpen)}
          className="flex items-center gap-1 px-2 py-2 rounded-lg text-muted-foreground hover:bg-muted hover:text-foreground transition-colors cursor-pointer"
          title={currentLang.label}
          aria-label="Language"
        >
          <Globe size={compact ? 16 : 17} strokeWidth={2} />
          <ChevronDown
            size={12}
            strokeWidth={2}
            className={`transition-transform duration-150 ${langOpen ? 'rotate-180' : ''}`}
          />
        </button>

        {langOpen && (
          <div className="absolute right-0 top-full mt-2 w-44 bg-card border border-border rounded-xl shadow-lg py-1.5 z-50 animate-[popIn_150ms_ease]">
            {languages.map((lang) => (
              <button
                key={lang.code}
                onClick={() => {
                  i18n.changeLanguage(lang.code)
                  setLangOpen(false)
                }}
                className={`flex items-center justify-between px-3 py-2 text-[13px] w-full transition-colors cursor-pointer ${
                  i18n.language === lang.code
                    ? 'text-primary font-medium'
                    : 'text-muted-foreground hover:bg-muted hover:text-foreground'
                }`}
              >
                {lang.label}
                {i18n.language === lang.code && <Check size={14} strokeWidth={2.5} />}
              </button>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
