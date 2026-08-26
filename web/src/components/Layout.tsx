import { useState, useEffect, useRef } from 'react'
import { Outlet, Link, useLocation, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  LayoutDashboard, Key, LogOut, Menu, Sun, Moon, Monitor,
  Globe, Check, Server, ChevronDown, Zap,
} from 'lucide-react'
import { useTheme } from '../contexts/ThemeContext'

/* ── Navigation model ─────────────────────────────────────────────────────── */

const navSections = [
  {
    labelKey: null as string | null,
    items: [{ path: '/dashboard', labelKey: 'layout.dashboard', icon: LayoutDashboard }],
  },
  {
    labelKey: 'layout.sectionGateway',
    items: [
      { path: '/channels', labelKey: 'layout.channels', icon: Server },
      { path: '/keys', labelKey: 'layout.apiKeys', icon: Key },
    ],
  },
]

const languages = [
  { code: 'zh', label: '简体中文' },
  { code: 'zh-TW', label: '繁體中文（台灣）' },
  { code: 'zh-HK', label: '繁體中文（香港）' },
  { code: 'en', label: 'English' },
]

const themes = [
  { value: 'light' as const, icon: Sun },
  { value: 'dark' as const, icon: Moon },
  { value: 'system' as const, icon: Monitor },
]

/* ── Layout ───────────────────────────────────────────────────────────────── */

export default function Layout() {
  const { t, i18n } = useTranslation()
  const location = useLocation()
  const navigate = useNavigate()
  const { theme, setTheme, resolved } = useTheme()
  const [sidebarOpen, setSidebarOpen] = useState(false)
  const [langOpen, setLangOpen] = useState(false)

  const langRef = useRef<HTMLDivElement>(null)

  // Close language menu on outside click / Escape.
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

  // Close drawer on route change.
  useEffect(() => { setSidebarOpen(false) }, [location.pathname])

  const handleLogout = () => {
    localStorage.removeItem('ringrouter_token')
    navigate('/')
  }

  const cycleTheme = () => {
    const order = ['light', 'dark', 'system'] as const
    const idx = order.indexOf(theme)
    setTheme(order[(idx + 1) % order.length])
  }

  const themeIcon = theme === 'dark' ? Moon : theme === 'light' ? Sun : Monitor
  const currentLang = languages.find((l) => l.code === i18n.language) || languages[3]

  const pageTitle =
    location.pathname.startsWith('/channels') ? t('layout.channels')
    : location.pathname.startsWith('/keys') ? t('layout.apiKeys')
    : t('layout.dashboard')

  /* ── Sidebar content (shared desktop / mobile) ── */

  const sidebarContent = (
    <div className="flex flex-col h-full">
      {/* Brand */}
      <div className="h-16 px-4 flex items-center gap-3 border-b border-border shrink-0">
        <div className="relative w-9 h-9 rounded-xl bg-primary flex items-center justify-center shrink-0 shadow-sm">
          <Zap size={17} className="text-primary-foreground" strokeWidth={2.5} />
        </div>
        <div className="min-w-0">
          <div className="flex items-center gap-1.5">
            <span className="font-semibold text-[15px] leading-tight tracking-tight">RingRouter</span>
            <span className="px-1.5 py-px text-[9px] font-mono font-medium rounded bg-muted text-muted-foreground leading-4">
              v0.1
            </span>
          </div>
          <p className="text-[11px] text-muted-foreground leading-tight truncate">{t('layout.subtitle')}</p>
        </div>
      </div>

      {/* Nav */}
      <nav className="flex-1 overflow-y-auto px-3 py-4 space-y-5">
        {navSections.map((section, si) => (
          <div key={si}>
            {section.labelKey && (
              <p className="px-2.5 mb-1.5 text-[10px] font-semibold uppercase tracking-[0.08em] text-muted-foreground/70">
                {t(section.labelKey)}
              </p>
            )}
            <ul className="space-y-0.5">
              {section.items.map((item) => {
                const Icon = item.icon
                const isActive = location.pathname.startsWith(item.path)
                return (
                  <li key={item.path}>
                    <Link
                      to={item.path}
                      className={`group relative flex items-center gap-2.5 px-2.5 py-2 rounded-lg text-[13px] transition-all duration-150 ${
                        isActive
                          ? 'bg-primary/10 text-primary font-medium'
                          : 'text-muted-foreground hover:bg-muted hover:text-foreground'
                      }`}
                    >
                      {/* Left active indicator */}
                      <span
                        className={`absolute left-0 top-1/2 -translate-y-1/2 -translate-x-3 w-1 h-4 rounded-full bg-primary transition-all duration-150 ${
                          isActive ? 'opacity-100 scale-100' : 'opacity-0 scale-0'
                        }`}
                      />
                      <Icon
                        size={16}
                        strokeWidth={isActive ? 2.2 : 1.8}
                        className={`transition-transform duration-150 group-hover:scale-110 ${
                          isActive ? 'text-primary' : ''
                        }`}
                      />
                      {t(item.labelKey)}
                    </Link>
                  </li>
                )
              })}
            </ul>
          </div>
        ))}
      </nav>

      {/* User card */}
      <div className="p-3 border-t border-border shrink-0">
        <div className="flex items-center gap-2.5 px-1.5 py-1.5">
          <div className="w-8 h-8 rounded-full bg-gradient-to-br from-primary to-primary-dark flex items-center justify-center shrink-0">
            <span className="text-primary-foreground text-xs font-semibold">A</span>
          </div>
          <div className="min-w-0 flex-1">
            <p className="text-[13px] font-medium leading-tight truncate">admin</p>
            <p className="text-[11px] text-muted-foreground leading-tight">Root</p>
          </div>
          <button
            onClick={handleLogout}
            title={t('layout.logout')}
            className="p-2 rounded-lg text-muted-foreground hover:bg-destructive/10 hover:text-destructive transition-colors cursor-pointer shrink-0"
          >
            <LogOut size={15} strokeWidth={2} />
          </button>
        </div>
      </div>
    </div>
  )

  /* ── Shell ── */

  return (
    <div className="flex h-screen bg-background overflow-hidden">
      {/* Desktop sidebar */}
      <aside className="hidden md:flex w-60 bg-card border-r border-border flex-col shrink-0">
        {sidebarContent}
      </aside>

      {/* Mobile drawer */}
      {sidebarOpen && (
        <div className="md:hidden fixed inset-0 z-50">
          <div
            className="absolute inset-0 bg-black/50 backdrop-blur-sm animate-[fadeIn_150ms_ease]"
            onClick={() => setSidebarOpen(false)}
          />
          <div className="absolute left-0 top-0 bottom-0 w-64 bg-card border-r border-border shadow-xl animate-[slideIn_200ms_ease]">
            {sidebarContent}
          </div>
        </div>
      )}

      {/* Main column */}
      <div className="flex-1 flex flex-col min-w-0">
        {/* Top header */}
        <header className="h-16 shrink-0 border-b border-border bg-card/70 backdrop-blur flex items-center gap-3 px-4 md:px-6">
          {/* Mobile: menu button */}
          <button
            onClick={() => setSidebarOpen(true)}
            className="md:hidden p-2 -ml-2 rounded-lg hover:bg-muted transition-colors cursor-pointer"
            aria-label="Open menu"
          >
            <Menu size={19} strokeWidth={2} />
          </button>

          {/* Page title */}
          <h1 className="text-[15px] font-semibold tracking-tight truncate">{pageTitle}</h1>

          <div className="flex-1" />

          {/* Theme cycle button */}
          <button
            onClick={cycleTheme}
            title={`${t('theme.' + theme)} → ${t('theme.' + (theme === 'light' ? 'dark' : theme === 'dark' ? 'system' : 'light'))}`}
            className="relative p-2 rounded-lg text-muted-foreground hover:bg-muted hover:text-foreground transition-colors cursor-pointer"
          >
            <themeIcon size={17} strokeWidth={2} />
            {theme === 'system' && (
              <span className={`absolute bottom-1 right-1 w-1.5 h-1.5 rounded-full ${resolved === 'dark' ? 'bg-primary' : 'bg-primary/60'}`} />
            )}
          </button>

          {/* Language dropdown */}
          <div className="relative" ref={langRef}>
            <button
              onClick={() => setLangOpen(!langOpen)}
              className="flex items-center gap-1 px-2 py-2 rounded-lg text-muted-foreground hover:bg-muted hover:text-foreground transition-colors cursor-pointer"
              title={currentLang.label}
            >
              <Globe size={17} strokeWidth={2} />
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
                    onClick={() => { i18n.changeLanguage(lang.code); setLangOpen(false) }}
                    className={`flex items-center justify-between px-3 py-2 text-[13px] w-full transition-colors ${
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
        </header>

        {/* Content */}
        <main className="flex-1 overflow-auto">
          <div className="p-4 md:p-6 max-w-6xl">
            <Outlet />
          </div>
        </main>
      </div>
    </div>
  )
}