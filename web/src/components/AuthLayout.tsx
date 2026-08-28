import type { ReactNode } from 'react'
import { useTranslation, Trans } from 'react-i18next'
import { Link } from 'react-router-dom'
import { ArrowLeft, Zap, Route, Shield, Layers } from 'lucide-react'
import ThemeLangActions from './ThemeLangActions'
import UserAvatar from './UserAvatar'
import { useSite } from '../contexts/SiteContext'
import { useAuth } from '../contexts/AuthContext'

const introFeatures = [
  { icon: Route, key: 'routing' },
  { icon: Layers, key: 'formats' },
  { icon: Shield, key: 'privacy' },
] as const

/**
 * Split-screen auth shell: brand/intro panel on the left, form on the right.
 * Follows the active light/dark theme. Collapses to a single column on mobile.
 */
export default function AuthLayout({ children }: { children: ReactNode }) {
  const { t } = useTranslation()
  const { siteName } = useSite()
  const { user } = useAuth()

  return (
    <div className="min-h-screen bg-background lg:grid lg:grid-cols-2">
      {/* Left: intro panel (desktop only) */}
      <aside className="hidden lg:flex relative flex-col justify-between overflow-hidden bg-muted/40 border-r border-border p-10 xl:p-14">
        {/* Decorative background */}
        <div className="absolute inset-0 -z-0" aria-hidden="true">
          <div className="absolute -top-32 -left-32 w-[480px] h-[480px] bg-primary/15 rounded-full blur-3xl" />
          <div className="absolute -bottom-40 -right-24 w-[420px] h-[420px] bg-accent/10 rounded-full blur-3xl" />
          <div
            className="absolute inset-0 opacity-[0.35]"
            style={{
              backgroundImage:
                'linear-gradient(var(--color-border) 1px, transparent 1px), linear-gradient(90deg, var(--color-border) 1px, transparent 1px)',
              backgroundSize: '44px 44px',
              maskImage: 'radial-gradient(ellipse 70% 60% at 30% 40%, black, transparent)',
              WebkitMaskImage: 'radial-gradient(ellipse 70% 60% at 30% 40%, black, transparent)',
            }}
          />
        </div>

        {/* Brand */}
        <Link to="/" className="relative z-10 flex items-center gap-3 w-fit group anim-fade-up">
          <div className="w-10 h-10 rounded-xl bg-primary flex items-center justify-center shadow-sm group-hover:scale-105 transition-transform">
            <Zap size={19} className="text-primary-foreground" strokeWidth={2.5} />
          </div>
          <div>
            <span className="font-semibold text-lg tracking-tight block leading-tight">{siteName}</span>
            <span className="text-xs text-muted-foreground">{t('layout.subtitle')}</span>
          </div>
        </Link>

        {/* Headline + features */}
        <div className="relative z-10 max-w-md">
          <h2 className="text-3xl xl:text-4xl font-bold tracking-tight leading-tight mb-4 anim-fade-up anim-delay-1">
            <Trans i18nKey="authIntro.headline">
              One gateway, <span className="text-primary">every provider</span>.
            </Trans>
          </h2>
          <p className="text-muted-foreground leading-relaxed mb-10 anim-fade-up anim-delay-2">{t('authIntro.sub')}</p>

          <ul className="space-y-5">
            {introFeatures.map(({ icon: Icon, key }, i) => (
              <li key={key} className={`flex items-start gap-4 anim-fade-up anim-delay-${i + 3}`}>
                <div className="w-9 h-9 rounded-lg bg-card border border-border flex items-center justify-center shrink-0 shadow-sm">
                  <Icon size={17} className="text-primary" strokeWidth={2} />
                </div>
                <div>
                  <p className="text-sm font-medium">{t(`authIntro.${key}.title`)}</p>
                  <p className="text-[13px] text-muted-foreground leading-relaxed mt-0.5">
                    {t(`authIntro.${key}.desc`)}
                  </p>
                </div>
              </li>
            ))}
          </ul>
        </div>

        {/* Footnote */}
        <p className="relative z-10 text-xs text-muted-foreground anim-fade-in anim-delay-6">{t('authIntro.footnote')}</p>
      </aside>

      {/* Right: form column */}
      <main className="flex flex-col min-h-screen">
        {/* Mobile brand header */}
        <div className="lg:hidden flex items-center justify-between px-4 pt-4 anim-fade-in">
          <Link to="/" className="flex items-center gap-2.5">
            <div className="w-8 h-8 rounded-lg bg-primary flex items-center justify-center">
              <Zap size={15} className="text-primary-foreground" strokeWidth={2.5} />
            </div>
            <span className="font-semibold tracking-tight">{siteName}</span>
          </Link>
          <ThemeLangActions compact />
          {user && <UserAvatar />}
        </div>

        <div className="flex-1 flex items-center justify-center px-4 sm:px-6 py-10">
          <div className="w-full max-w-sm anim-form-in anim-delay-1">{children}</div>
        </div>

        <div className="px-4 pb-6 text-center anim-fade-in anim-delay-3">
          <Link
            to="/"
            className="inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors"
          >
            <ArrowLeft size={12} strokeWidth={2} />
            {t('auth.backHome')}
          </Link>
        </div>
      </main>
    </div>
  )
}
