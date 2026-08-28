import { useState, useEffect } from 'react'
import { Outlet, Link, useLocation, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  MessageSquare, Gamepad2,
  LayoutDashboard, BarChart3, Key, FileText, ClipboardList,
  Wallet, User,
  Server, Cpu, Users, Ticket, CreditCard, Info, Settings,
  LogOut, Menu, Zap, Megaphone,
} from 'lucide-react'
import ThemeLangActions from './ThemeLangActions'
import UserAvatar from './UserAvatar'
import { useAuth } from '../contexts/AuthContext'
import { useSite } from '../contexts/SiteContext'
import { api } from '../lib/api'

/* ── Navigation model ─────────────────────────────────────────────────────── */

interface NavItem {
  path: string
  labelKey: string
  icon: typeof MessageSquare
}

const chatNav: NavItem[] = [
  { path: '/dash/chat', labelKey: 'nav.chat', icon: MessageSquare },
  { path: '/dash/playground', labelKey: 'nav.playground', icon: Gamepad2 },
]

const generalNav: NavItem[] = [
  { path: '/dash/overview', labelKey: 'nav.overview', icon: LayoutDashboard },
  { path: '/dash/data', labelKey: 'nav.dataBoard', icon: BarChart3 },
  { path: '/dash/keys', labelKey: 'nav.apiKeys', icon: Key },
  { path: '/dash/logs', labelKey: 'nav.usageLogs', icon: FileText },
  { path: '/dash/task-logs', labelKey: 'nav.taskLogs', icon: ClipboardList },
]

const personalNav: NavItem[] = [
  { path: '/dash/wallet', labelKey: 'nav.wallet', icon: Wallet },
  { path: '/dash/profile', labelKey: 'nav.profile', icon: User },
]

const adminNav: NavItem[] = [
  { path: '/dash/manage/channels', labelKey: 'nav.channels', icon: Server },
  { path: '/dash/manage/models', labelKey: 'nav.models', icon: Cpu },
  { path: '/dash/manage/users', labelKey: 'nav.users', icon: Users },
  { path: '/dash/manage/codes', labelKey: 'nav.codes', icon: Ticket },
  { path: '/dash/manage/subscriptions', labelKey: 'nav.subscriptions', icon: CreditCard },
  { path: '/dash/manage/system', labelKey: 'nav.systemInfo', icon: Info },
  { path: '/dash/manage/settings', labelKey: 'nav.settings', icon: Settings },
]

/* ── Layout ───────────────────────────────────────────────────────────────── */

export default function Layout() {
  const { t } = useTranslation()
  const location = useLocation()
  const navigate = useNavigate()
  const { user, logout } = useAuth()
  const { siteName } = useSite()
  const [sidebarOpen, setSidebarOpen] = useState(false)
  const [annOpen, setAnnOpen] = useState(false)
  const [announcement, setAnnouncement] = useState<string | null>(null)

  useEffect(() => { setSidebarOpen(false) }, [location.pathname])

  useEffect(() => {
    api.get<{ announcement: string }>('/api/announcement')
      .then((d) => setAnnouncement(d.announcement || ''))
      .catch(() => setAnnouncement(''))
  }, [])

  const handleLogout = async () => {
    await logout()
    navigate('/')
  }

  const isActive = (path: string) =>
    location.pathname === path ||
    (path !== '/' && location.pathname.startsWith(path + '/'))

  const allItems = [...chatNav, ...generalNav, ...personalNav, ...adminNav]
  const current = allItems.find((i) => isActive(i.path))
  const pageTitle = current ? t(current.labelKey) : t('nav.overview')

  /* ── Sidebar ── */

  const renderNavItems = (items: NavItem[]) => (
    <ul className="space-y-0.5">
      {items.map((item) => {
        const active = isActive(item.path)
        const Icon = item.icon
        return (
          <li key={item.path}>
            <Link
              to={item.path}
              className={`group relative flex items-center gap-2.5 px-2.5 py-2 rounded-lg text-[13px] transition-all duration-150 ${
                active
                  ? 'bg-primary/10 text-primary font-medium'
                  : 'text-muted-foreground hover:bg-muted hover:text-foreground'
              }`}
            >
              <span
                className={`absolute left-0 top-1/2 -translate-y-1/2 -translate-x-3 w-1 h-4 rounded-full bg-primary transition-all duration-150 ${
                  active ? 'opacity-100 scale-100' : 'opacity-0 scale-0'
                }`}
              />
              <Icon
                size={16}
                strokeWidth={active ? 2.2 : 1.8}
                className={`transition-transform duration-150 group-hover:scale-110 ${active ? 'text-primary' : ''}`}
              />
              {t(item.labelKey)}
            </Link>
          </li>
        )
      })}
    </ul>
  )

  const Section = ({ labelKey, items }: { labelKey: string; items: NavItem[] }) => (
    <div>
      <p className="px-2.5 mb-1.5 text-[10px] font-semibold uppercase tracking-[0.08em] text-muted-foreground/70">
        {t(labelKey)}
      </p>
      {renderNavItems(items)}
    </div>
  )

  const sidebarContent = (
    <div className="flex flex-col h-full">
      {/* Brand */}
      <div className="h-16 px-4 flex items-center gap-3 border-b border-border shrink-0">
        <div className="relative w-9 h-9 rounded-xl bg-primary flex items-center justify-center shrink-0 shadow-sm">
          <Zap size={17} className="text-primary-foreground" strokeWidth={2.5} />
        </div>
        <div className="min-w-0">
          <div className="flex items-center gap-1.5">
            <span className="font-semibold text-[15px] leading-tight tracking-tight truncate">{siteName}</span>
            <span className="px-1.5 py-px text-[9px] font-mono font-medium rounded bg-muted text-muted-foreground leading-4">
              v0.1
            </span>
          </div>
          <p className="text-[11px] text-muted-foreground leading-tight truncate">{t('layout.subtitle')}</p>
        </div>
      </div>

      {/* Nav */}
      <nav className="flex-1 overflow-y-auto px-3 py-4 space-y-5">
        {renderNavItems(chatNav)}
        <Section labelKey="nav.sectionGeneral" items={generalNav} />
        <Section labelKey="nav.sectionPersonal" items={personalNav} />
        {user?.role === 'admin' && (
          <Section labelKey="nav.sectionAdmin" items={adminNav} />
        )}
      </nav>

      {/* User card */}
      <div className="p-3 border-t border-border shrink-0">
        <div className="flex items-center gap-2.5 px-1.5 py-1.5">
          <div className="w-8 h-8 rounded-full bg-gradient-to-br from-primary to-primary-dark flex items-center justify-center shrink-0">
            <span className="text-primary-foreground text-xs font-semibold uppercase">
              {(user?.username || 'a').charAt(0)}
            </span>
          </div>
          <div className="min-w-0 flex-1">
            <p className="text-[13px] font-medium leading-tight truncate">{user?.username || '—'}</p>
            <p className="text-[11px] text-muted-foreground leading-tight capitalize">
              {user?.role || ''}
            </p>
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

  return (
    <div className="flex h-screen bg-background overflow-hidden">
      <aside className="hidden md:flex w-60 bg-card border-r border-border flex-col shrink-0">
        {sidebarContent}
      </aside>

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

      <div className="flex-1 flex flex-col min-w-0">
        <header className="h-16 shrink-0 border-b border-border bg-card/70 backdrop-blur flex items-center gap-3 px-4 md:px-6">
          <button
            onClick={() => setSidebarOpen(true)}
            className="md:hidden p-2 -ml-2 rounded-lg hover:bg-muted transition-colors cursor-pointer"
            aria-label="Open menu"
          >
            <Menu size={19} strokeWidth={2} />
          </button>

          <h1 className="text-[15px] font-semibold tracking-tight truncate">{pageTitle}</h1>

          <div className="flex-1" />

          {announcement && announcement.trim() !== '' && (
            <div className="relative">
              <button
                onClick={() => setAnnOpen(!annOpen)}
                title={t('layout.announcement')}
                aria-label={t('layout.announcement')}
                className={`relative p-2 rounded-lg transition-colors cursor-pointer ${
                  annOpen
                    ? 'bg-muted text-foreground'
                    : 'text-muted-foreground hover:bg-muted hover:text-foreground'
                }`}
              >
                <Megaphone size={17} strokeWidth={2} />
                <span className="absolute top-1.5 right-1.5 w-1.5 h-1.5 rounded-full bg-accent" />
              </button>
              {annOpen && (
                <div className="absolute right-0 top-full mt-2 w-80 bg-card border border-border rounded-xl shadow-lg p-4 z-50 animate-[popIn_150ms_ease]">
                  <div className="flex items-center gap-2 mb-2">
                    <Megaphone size={14} className="text-primary" strokeWidth={2.2} />
                    <p className="text-[13px] font-semibold">{t('layout.announcement')}</p>
                  </div>
                  <p className="text-[13px] text-muted-foreground leading-relaxed whitespace-pre-wrap">
                    {announcement}
                  </p>
                </div>
              )}
            </div>
          )}

          <ThemeLangActions />
          <UserAvatar />
        </header>

        <main className="flex-1 overflow-auto">
          <div className="p-4 md:p-6 max-w-6xl">
            <div key={location.pathname} className="flex-1 anim-fade-up"><Outlet /></div>
          </div>
        </main>
      </div>
    </div>
  )
}
