import { useState } from 'react'
import { Outlet, Link, useLocation, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { LayoutDashboard, Key, LogOut, Menu, Sun, Moon, Monitor, Globe, Check, Server } from 'lucide-react'
import { useTheme } from '../contexts/ThemeContext'

const navItems = [
  { path: '/dashboard', labelKey: 'layout.dashboard', icon: LayoutDashboard },
  { path: '/channels', labelKey: 'layout.channels', icon: Server },
  { path: '/keys', labelKey: 'layout.apiKeys', icon: Key },
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

export default function Layout() {
  const { t, i18n } = useTranslation()
  const location = useLocation()
  const navigate = useNavigate()
  const { theme, setTheme } = useTheme()
  const [sidebarOpen, setSidebarOpen] = useState(false)
  const [themeOpen, setThemeOpen] = useState(false)
  const [langOpen, setLangOpen] = useState(false)

  const handleLogout = () => {
    localStorage.removeItem('ringrouter_token')
    navigate('/')
  }

  const currentLang = languages.find((l) => l.code === i18n.language) || languages[3]
  const themeIcon = theme === 'dark' ? Moon : theme === 'light' ? Sun : Monitor

  const closeAll = () => { setThemeOpen(false); setLangOpen(false) }

  const sidebarContent = (
    <div className="flex flex-col h-full">
      {/* Brand */}
      <div className="px-4 py-5 border-b border-border">
        <div className="flex items-center gap-2.5">
          <div className="w-8 h-8 rounded-lg bg-primary flex items-center justify-center shrink-0">
            <span className="text-primary-foreground font-bold text-sm">R</span>
          </div>
          <div>
            <h1 className="text-base font-semibold leading-tight">RingRouter</h1>
            <p className="text-[11px] text-muted-foreground">{t('layout.subtitle')}</p>
          </div>
        </div>
      </div>

      {/* Nav */}
      <nav className="flex-1 p-2.5">
        {navItems.map((item) => {
          const Icon = item.icon
          const isActive = location.pathname.startsWith(item.path)
          return (
            <Link
              key={item.path}
              to={item.path}
              onClick={() => setSidebarOpen(false)}
              className={`flex items-center gap-3 px-3 py-2 rounded-md mb-0.5 text-sm transition-colors ${
                isActive
                  ? 'bg-primary/10 text-primary font-medium'
                  : 'text-muted-foreground hover:bg-muted hover:text-foreground'
              }`}
            >
              <Icon size={18} strokeWidth={2} />
              {t(item.labelKey)}
            </Link>
          )
        })}
      </nav>

      {/* Bottom controls */}
      <div className="p-2.5 border-t border-border space-y-0.5">
        {/* Theme selector */}
        <div className="relative">
          <button
            onClick={() => { setThemeOpen(!themeOpen); setLangOpen(false) }}
            className="flex items-center gap-3 px-3 py-2 rounded-md text-sm text-muted-foreground hover:bg-muted hover:text-foreground w-full transition-colors"
          >
            <themeIcon size={18} strokeWidth={2} />
            {t(`theme.${theme}`)}
          </button>
          {themeOpen && (
            <div className="absolute bottom-full left-0 mb-1 bg-card border border-border rounded-lg shadow-lg py-1 w-full z-50">
              {themes.map((th) => {
                const Icon = th.icon
                return (
                  <button
                    key={th.value}
                    onClick={() => { setTheme(th.value); closeAll() }}
                    className={`flex items-center justify-between px-3 py-1.5 text-sm w-full transition-colors ${
                      theme === th.value
                        ? 'text-primary font-medium'
                        : 'text-muted-foreground hover:bg-muted hover:text-foreground'
                    }`}
                  >
                    <span className="flex items-center gap-2">
                      <Icon size={14} strokeWidth={2} />
                      {t('theme.' + th.value)}
                    </span>
                    {theme === th.value && <Check size={14} />}
                  </button>
                )
              })}
            </div>
          )}
        </div>

        {/* Language selector */}
        <div className="relative">
          <button
            onClick={() => { setLangOpen(!langOpen); setThemeOpen(false) }}
            className="flex items-center gap-3 px-3 py-2 rounded-md text-sm text-muted-foreground hover:bg-muted hover:text-foreground w-full transition-colors"
          >
            <Globe size={18} strokeWidth={2} />
            {currentLang.label}
          </button>
          {langOpen && (
            <div className="absolute bottom-full left-0 mb-1 bg-card border border-border rounded-lg shadow-lg py-1 w-full z-50">
              {languages.map((lang) => (
                <button
                  key={lang.code}
                  onClick={() => { i18n.changeLanguage(lang.code); closeAll() }}
                  className={`flex items-center justify-between px-3 py-1.5 text-sm w-full transition-colors ${
                    i18n.language === lang.code
                      ? 'text-primary font-medium'
                      : 'text-muted-foreground hover:bg-muted hover:text-foreground'
                  }`}
                >
                  {lang.label}
                  {i18n.language === lang.code && <Check size={14} />}
                </button>
              ))}
            </div>
          )}
        </div>

        {/* Logout */}
        <button
          onClick={handleLogout}
          className="flex items-center gap-3 px-3 py-2 rounded-md text-sm text-muted-foreground hover:bg-muted hover:text-foreground w-full transition-colors"
        >
          <LogOut size={18} strokeWidth={2} />
          {t('layout.logout')}
        </button>
      </div>
    </div>
  )

  return (
    <div className="flex h-screen bg-background">
      {/* Desktop sidebar */}
      <aside className="hidden md:flex w-64 bg-card border-r border-border flex-col shrink-0">
        {sidebarContent}
      </aside>

      {/* Mobile sidebar overlay */}
      {sidebarOpen && (
        <div className="md:hidden fixed inset-0 z-40">
          <div
            className="sidebar-overlay absolute inset-0 bg-black/50"
            onClick={() => setSidebarOpen(false)}
          />
          <div className="sidebar-panel absolute left-0 top-0 bottom-0 w-64 bg-card border-r border-border z-50">
            {sidebarContent}
          </div>
        </div>
      )}

      {/* Main */}
      <main className="flex-1 overflow-auto flex flex-col">
        {/* Mobile header */}
        <div className="md:hidden sticky top-0 z-30 flex items-center gap-3 px-4 h-14 border-b border-border bg-card/80 backdrop-blur">
          <button
            onClick={() => setSidebarOpen(true)}
            className="p-1.5 rounded-md hover:bg-muted transition-colors cursor-pointer"
            aria-label="Open menu"
          >
            <Menu size={20} strokeWidth={2} />
          </button>
          <span className="font-semibold text-primary">RingRouter</span>
        </div>

        {/* Content */}
        <div className="p-4 md:p-6 flex-1">
          <Outlet />
        </div>
      </main>
    </div>
  )
}