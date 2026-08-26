import { useState } from 'react'
import { Outlet, Link, useLocation, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { LayoutDashboard, Key, LogOut, Menu, X, Sun, Moon, Monitor, Globe } from 'lucide-react'
import { useTheme } from '../contexts/ThemeContext'

const navItems = [
  { path: '/dashboard', labelKey: 'layout.dashboard', icon: LayoutDashboard },
  { path: '/keys', labelKey: 'layout.apiKeys', icon: Key },
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

  const toggleLang = () => {
    const next = i18n.language === 'zh' ? 'en' : 'zh'
    i18n.changeLanguage(next)
    setLangOpen(false)
  }

  const themeIcon = theme === 'dark' ? Moon : theme === 'light' ? Sun : Monitor

  const sidebar = (
    <div className="flex flex-col h-full">
      {/* Brand */}
      <div className="p-4 border-b border-border">
        <h1 className="text-lg font-bold text-primary">RingRouter</h1>
        <p className="text-xs text-text-muted">{t('layout.subtitle')}</p>
      </div>

      {/* Nav */}
      <nav className="flex-1 p-2">
        {navItems.map((item) => {
          const Icon = item.icon
          const isActive = location.pathname.startsWith(item.path)
          return (
            <Link
              key={item.path}
              to={item.path}
              onClick={() => setSidebarOpen(false)}
              className={`flex items-center gap-3 px-3 py-2 rounded-lg mb-1 text-sm transition-colors ${
                isActive
                  ? 'bg-primary/10 text-primary font-medium'
                  : 'text-text-muted hover:bg-surface-alt hover:text-text'
              }`}
            >
              <Icon size={18} />
              {t(item.labelKey)}
            </Link>
          )
        })}
      </nav>

      {/* Bottom controls */}
      <div className="p-2 border-t border-border space-y-1">
        {/* Theme toggle */}
        <div className="relative">
          <button
            onClick={() => { setThemeOpen(!themeOpen); setLangOpen(false) }}
            className="flex items-center gap-3 px-3 py-2 rounded-lg text-sm text-text-muted hover:bg-surface-alt hover:text-text w-full transition-colors"
          >
            <themeIcon size={18} />
            {t(`theme.${theme}`)}
          </button>
          {themeOpen && (
            <div className="absolute bottom-full left-2 mb-1 bg-white dark:bg-gray-800 border border-border rounded-lg shadow-lg py-1 w-40 z-50">
              {(['light', 'dark', 'system'] as const).map((t) => (
                <button
                  key={t}
                  onClick={() => { setTheme(t); setThemeOpen(false) }}
                  className={`flex items-center gap-2 px-3 py-1.5 text-sm w-full text-left hover:bg-surface-alt transition-colors ${
                    theme === t ? 'text-primary font-medium' : 'text-text-muted'
                  }`}
                >
                  {t === 'light' ? <Sun size={14} /> : t === 'dark' ? <Moon size={14} /> : <Monitor size={14} />}
                  {t('theme.' + t)}
                </button>
              ))}
            </div>
          )}
        </div>

        {/* Language toggle */}
        <div className="relative">
          <button
            onClick={() => { setLangOpen(!langOpen); setThemeOpen(false) }}
            className="flex items-center gap-3 px-3 py-2 rounded-lg text-sm text-text-muted hover:bg-surface-alt hover:text-text w-full transition-colors"
          >
            <Globe size={18} />
            {i18n.language === 'zh' ? '中文' : 'English'}
          </button>
          {langOpen && (
            <div className="absolute bottom-full left-2 mb-1 bg-white dark:bg-gray-800 border border-border rounded-lg shadow-lg py-1 w-40 z-50">
              <button
                onClick={() => { i18n.changeLanguage('zh'); setLangOpen(false) }}
                className={`block px-3 py-1.5 text-sm w-full text-left hover:bg-surface-alt transition-colors ${
                  i18n.language === 'zh' ? 'text-primary font-medium' : 'text-text-muted'
                }`}
              >
                中文
              </button>
              <button
                onClick={() => { i18n.changeLanguage('en'); setLangOpen(false) }}
                className={`block px-3 py-1.5 text-sm w-full text-left hover:bg-surface-alt transition-colors ${
                  i18n.language === 'en' ? 'text-primary font-medium' : 'text-text-muted'
                }`}
              >
                English
              </button>
            </div>
          )}
        </div>

        {/* Logout */}
        <button
          onClick={handleLogout}
          className="flex items-center gap-3 px-3 py-2 rounded-lg text-sm text-text-muted hover:bg-surface-alt hover:text-text w-full transition-colors"
        >
          <LogOut size={18} />
          {t('layout.logout')}
        </button>
      </div>
    </div>
  )

  return (
    <div className="flex h-screen">
      {/* Desktop sidebar */}
      <aside className="hidden md:flex w-64 bg-white dark:bg-surface border-r border-border flex-col shrink-0">
        {sidebar}
      </aside>

      {/* Mobile sidebar overlay */}
      {sidebarOpen && (
        <div className="md:hidden fixed inset-0 z-40">
          <div
            className="sidebar-overlay absolute inset-0 bg-black/50"
            onClick={() => setSidebarOpen(false)}
          />
          <div className="sidebar-panel absolute left-0 top-0 bottom-0 w-64 bg-white dark:bg-surface border-r border-border z-50 translate-x-0">
            {sidebar}
          </div>
        </div>
      )}

      {/* Main */}
      <main className="flex-1 overflow-auto flex flex-col">
        {/* Mobile header */}
        <div className="md:hidden flex items-center gap-3 p-4 border-b border-border bg-white dark:bg-surface">
          <button
            onClick={() => setSidebarOpen(true)}
            className="p-1.5 rounded-lg hover:bg-surface-alt transition-colors"
          >
            <Menu size={20} />
          </button>
          <h1 className="font-semibold text-primary">RingRouter</h1>
        </div>

        {/* Content */}
        <div className="p-4 md:p-6 flex-1">
          <Outlet />
        </div>
      </main>
    </div>
  )
}