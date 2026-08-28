import { Link, NavLink, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Zap, LayoutDashboard } from 'lucide-react'
import ThemeLangActions from './ThemeLangActions'
import UserAvatar from './UserAvatar'
import { useAuth } from '../contexts/AuthContext'
import { useSite } from '../contexts/SiteContext'

/**
 * Navigation bar shared by the public pages (home, model plaza, about).
 * Mirrors the dashboard shell: brand, primary links, theme/language
 * actions, and a session-aware CTA (sign in vs. console).
 */
export default function PublicNav() {
  const { t } = useTranslation()
  const { siteName } = useSite()
  const { user } = useAuth()
  const navigate = useNavigate()

  const linkCls = ({ isActive }: { isActive: boolean }) =>
    `text-sm transition-colors cursor-pointer ${
      isActive ? 'text-foreground font-medium' : 'text-muted-foreground hover:text-foreground'
    }`

  return (
    <header className="sticky top-0 z-40 border-b border-border bg-background/80 backdrop-blur-md">
      <div className="max-w-6xl mx-auto px-4 h-16 flex items-center gap-4">
        {/* Brand */}
        <Link to="/" className="flex items-center gap-2.5 shrink-0">
          <div className="w-8 h-8 rounded-lg bg-primary flex items-center justify-center shadow-sm">
            <Zap size={15} className="text-primary-foreground" strokeWidth={2.5} />
          </div>
          <span className="font-semibold tracking-tight">{siteName}</span>
        </Link>

        {/* Primary links */}
        <nav className="flex items-center gap-1 ml-4">
          <NavLink to="/" end className={linkCls}>
            <span className="px-2.5 py-2">{t('nav.home')}</span>
          </NavLink>
          <NavLink to="/models" className={linkCls}>
            <span className="px-2.5 py-2">{t('plaza.title')}</span>
          </NavLink>
          <NavLink to="/about" className={linkCls}>
            <span className="px-2.5 py-2">{t('nav.about')}</span>
          </NavLink>
          <a href="https://github.com/RingKoAI/RingRouter" target="_blank" rel="noreferrer"
            className="text-sm text-muted-foreground hover:text-foreground transition-colors px-2.5 py-2">
            {t('nav.docs')}
          </a>
        </nav>

        <div className="flex-1" />

        {/* Theme + language */}
        <ThemeLangActions compact />

        {/* Session-aware CTA */}
        {user ? (
          <div className="flex items-center gap-2">
            <button
              onClick={() => navigate('/dash/overview')}
              className="inline-flex items-center gap-1.5 min-h-[36px] px-3.5 rounded-xl text-sm font-medium bg-primary text-primary-foreground hover:bg-primary-dark transition-colors cursor-pointer shadow-sm"
            >
              <LayoutDashboard size={14} />
              <span className="hidden sm:inline">{t('nav.console')}</span>
            </button>
            <UserAvatar />
          </div>
        ) : (
          <div className="flex items-center gap-2">
            <button
              onClick={() => navigate('/auth/login')}
              className="min-h-[36px] px-3.5 rounded-xl text-sm text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
            >
              {t('auth.signIn')}
            </button>
            <button
              onClick={() => navigate('/auth/register')}
              className="hidden sm:inline-flex min-h-[36px] items-center px-3.5 rounded-xl text-sm font-medium bg-primary text-primary-foreground hover:bg-primary-dark transition-colors cursor-pointer shadow-sm"
            >
              {t('auth.createAccount')}
            </button>
          </div>
        )}
      </div>
    </header>
  )
}
