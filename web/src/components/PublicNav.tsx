import { useEffect, useRef, useState } from 'react'
import { Link, NavLink, useNavigate, useLocation } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Zap, LayoutDashboard, Menu, X, ArrowRight, ArrowUpRight } from 'lucide-react'
import ThemeLangActions from './ThemeLangActions'
import UserAvatar from './UserAvatar'
import { useAuth } from '../contexts/AuthContext'
import { useSite } from '../contexts/SiteContext'

/** Public site links shared by desktop nav and the mobile drawer. */
function usePublicLinks() {
  const { t } = useTranslation()
  return [
    { to: '/', label: t('nav.home'), end: true },
    { to: '/models', label: t('plaza.title'), end: false },
    { to: '/about', label: t('nav.about'), end: false },
    { href: 'https://github.com/RingKoAI/RingRouter', label: t('nav.docs'), external: true },
  ]
}

/**
 * Navigation bar shared by the public pages (home, model plaza, about).
 * Desktop: animated active indicator + theme/language + session CTA.
 * Mobile: hamburger opening a full drawer with everything inside.
 */
export default function PublicNav() {
  const { t } = useTranslation()
  const { siteName } = useSite()
  const { user } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const links = usePublicLinks()

  const [scrolled, setScrolled] = useState(false)
  const [drawerOpen, setDrawerOpen] = useState(false)
  const drawerRef = useRef<HTMLDivElement>(null)

  // Elevation after scrolling past the hero edge.
  useEffect(() => {
    const onScroll = () => setScrolled(window.scrollY > 8)
    onScroll()
    window.addEventListener('scroll', onScroll, { passive: true })
    return () => window.removeEventListener('scroll', onScroll)
  }, [])

  // Close the drawer on navigation and on Escape; lock body scroll while open.
  useEffect(() => { setDrawerOpen(false) }, [location.pathname])
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => { if (e.key === 'Escape') setDrawerOpen(false) }
    document.addEventListener('keydown', onKey)
    document.body.style.overflow = drawerOpen ? 'hidden' : ''
    return () => {
      document.removeEventListener('keydown', onKey)
      document.body.style.overflow = ''
    }
  }, [drawerOpen])

  const isExternal = (l: (typeof links)[number]) => 'external' in l && l.external
  const activePath = (l: (typeof links)[number]) =>
    'end' in l && l.end ? location.pathname === l.to : location.pathname.startsWith(String(l.to))

  const desktopLink = (l: (typeof links)[number]) => {
    const active = activePath(l)
    const inner = (
      <span className={`relative px-3 py-2 text-sm transition-colors cursor-pointer rounded-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40
        ${active ? 'text-foreground font-medium' : 'text-muted-foreground hover:text-foreground'}`}>
        {active && (
          <span className="absolute inset-x-3 -bottom-[1px] h-[2px] rounded-full bg-primary" />
        )}
        {l.label}
      </span>
    )
    if (isExternal(l)) {
      return (
        <a key={l.label} href={l.href} target="_blank" rel="noreferrer"
          className="text-sm text-muted-foreground hover:text-foreground transition-colors px-3 py-2 rounded-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40">
          {l.label}
        </a>
      )
    }
    return (
      <NavLink key={l.to} to={String(l.to)} end={'end' in l ? l.end : false} className="group">
        {inner}
      </NavLink>
    )
  }

  return (
    <>
      <header
        className={`sticky top-0 z-40 border-b bg-background/80 backdrop-blur-md transition-[box-shadow,border-color] duration-300 ${
          scrolled ? 'border-border shadow-[0_1px_12px_rgba(0,0,0,0.06)] dark:shadow-[0_1px_12px_rgba(0,0,0,0.4)]' : 'border-transparent'
        }`}
      >
        <div className="max-w-6xl mx-auto px-4 h-16 flex items-center gap-2">

          {/* Brand */}
          <Link to="/" className="flex items-center gap-2.5 shrink-0 group focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 rounded-xl pr-2">
            <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-primary to-primary/85 flex items-center justify-center shadow-md shadow-primary/25 transition-transform duration-300 group-hover:scale-105 group-hover:rotate-3">
              <Zap size={15} className="text-primary-foreground" strokeWidth={2.5} />
            </div>
            <span className="font-semibold tracking-tight">{siteName}</span>
          </Link>

          {/* Desktop links */}
          <nav className="hidden md:flex items-center gap-0.5 ml-2" aria-label="Primary">
            {links.map(desktopLink)}
          </nav>

          <div className="flex-1" />

          {/* Desktop actions */}
          <div className="hidden md:flex items-center gap-2">
            <a href="https://github.com/RingKoAI/RingRouter" target="_blank" rel="noreferrer"
              className="inline-flex items-center gap-1 px-2.5 py-2 rounded-lg text-sm text-muted-foreground hover:text-foreground hover:bg-muted transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40">
              GitHub <ArrowUpRight size={12} />
            </a>
            <ThemeLangActions compact />
            {user ? (
              <div className="flex items-center gap-2">
                <button
                  onClick={() => navigate('/dash/overview')}
                  className="inline-flex items-center gap-1.5 min-h-[36px] px-3.5 rounded-xl text-sm font-medium bg-primary text-primary-foreground hover:bg-primary-dark transition-all hover:shadow-lg hover:shadow-primary/25 active:scale-[0.98] cursor-pointer"
                >
                  <LayoutDashboard size={14} />
                  {t('nav.console')}
                </button>
                <UserAvatar />
              </div>
            ) : (
              <div className="flex items-center gap-1.5">
                <button
                  onClick={() => navigate('/auth/login')}
                  className="min-h-[36px] px-3.5 rounded-xl text-sm text-muted-foreground hover:text-foreground hover:bg-muted transition-colors cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
                >
                  {t('auth.signIn')}
                </button>
                <button
                  onClick={() => navigate('/auth/register')}
                  className="inline-flex items-center gap-1 min-h-[36px] px-4 rounded-xl text-sm font-medium bg-primary text-primary-foreground transition-all hover:shadow-lg hover:shadow-primary/25 active:scale-[0.98] cursor-pointer"
                >
                  {t('auth.getStarted')}
                  <ArrowRight size={13} className="transition-transform duration-200 " />
                </button>
              </div>
            )}
          </div>

          {/* Mobile hamburger */}
          <div className="md:hidden flex items-center gap-1">
            <ThemeLangActions compact />
            <button
              onClick={() => setDrawerOpen(!drawerOpen)}
              aria-label={drawerOpen ? 'Close menu' : 'Open menu'}
              aria-expanded={drawerOpen}
              className="p-2 rounded-lg text-muted-foreground hover:text-foreground hover:bg-muted transition-colors cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
            >
              {drawerOpen ? <X size={19} /> : <Menu size={19} />}
            </button>
          </div>
        </div>
      </header>

      {/* Mobile drawer */}
      <div
        className={`fixed inset-0 z-50 md:hidden transition-opacity duration-200 ${
          drawerOpen ? 'opacity-100' : 'opacity-0 pointer-events-none'
        }`}
        aria-hidden={!drawerOpen}
      >
        <div className="absolute inset-0 bg-black/50 backdrop-blur-sm" onClick={() => setDrawerOpen(false)} />
        <div
          ref={drawerRef}
          className={`absolute right-0 top-0 bottom-0 w-72 max-w-[85vw] bg-card border-l border-border shadow-2xl p-4 flex flex-col transition-transform duration-300 ${
            drawerOpen ? 'translate-x-0' : 'translate-x-full'
          }`}
          role="dialog"
          aria-modal="true"
        >
          <div className="flex items-center justify-between mb-6 px-1">
            <span className="font-semibold">{siteName}</span>
            <button onClick={() => setDrawerOpen(false)}
              className="p-2 rounded-lg text-muted-foreground hover:text-foreground hover:bg-muted transition-colors cursor-pointer">
              <X size={18} />
            </button>
          </div>

          <nav className="flex flex-col gap-1" aria-label="Mobile">
            {links.map((l) => {
              const active = activePath(l)
              if (isExternal(l)) {
                return (
                  <a key={l.label} href={l.href} target="_blank" rel="noreferrer"
                    className="px-3 py-3 rounded-xl text-sm text-muted-foreground hover:text-foreground hover:bg-muted transition-colors">
                    {l.label}
                  </a>
                )
              }
              return (
                <NavLink key={l.to} to={String(l.to)} end={'end' in l ? l.end : false}
                  className={`px-3 py-3 rounded-xl text-sm transition-colors ${
                    active ? 'bg-primary/10 text-primary font-medium' : 'text-muted-foreground hover:text-foreground hover:bg-muted'
                  }`}>
                  {l.label}
                </NavLink>
              )
            })}
          </nav>

          <div className="h-px bg-border my-4" />

          {user ? (
            <div className="flex flex-col gap-2">
              <button onClick={() => { setDrawerOpen(false); navigate('/dash/overview') }}
                className="inline-flex items-center justify-center gap-1.5 min-h-[44px] rounded-xl text-sm font-medium bg-primary text-primary-foreground active:scale-[0.98] transition-transform cursor-pointer">
                <LayoutDashboard size={15} /> {t('nav.console')}
              </button>
              <div className="flex justify-center py-2"><UserAvatar /></div>
            </div>
          ) : (
            <div className="flex flex-col gap-2">
              <button onClick={() => { setDrawerOpen(false); navigate('/auth/login') }}
                className="min-h-[44px] rounded-xl text-sm text-muted-foreground hover:bg-muted transition-colors cursor-pointer">
                {t('auth.signIn')}
              </button>
              <button onClick={() => { setDrawerOpen(false); navigate('/auth/register') }}
                className="inline-flex items-center justify-center gap-1 min-h-[44px] rounded-xl text-sm font-medium bg-primary text-primary-foreground active:scale-[0.98] transition-transform cursor-pointer">
                {t('auth.getStarted')} <ArrowRight size={14} />
              </button>
            </div>
          )}
        </div>
      </div>
    </>
  )
}
