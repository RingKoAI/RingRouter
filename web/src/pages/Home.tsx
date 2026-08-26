import { Link } from 'react-router-dom'
import { Trans, useTranslation } from 'react-i18next'
import { ArrowRight, Zap, Shield, Layers, Globe } from 'lucide-react'

const featureIcons = [Zap, Layers, Shield, Globe] as const
const featureKeys = ['unified', 'multi', 'keys', 'self'] as const

export default function Home() {
  const { t } = useTranslation()

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <header className="sticky top-0 z-30 border-b border-border bg-background/80 backdrop-blur">
        <div className="max-w-6xl mx-auto px-4 md:px-6 h-16 flex items-center justify-between">
          <div className="flex items-center gap-2.5">
            <div className="w-8 h-8 rounded-lg bg-primary flex items-center justify-center">
              <span className="text-primary-foreground font-bold text-sm">R</span>
            </div>
            <span className="font-semibold text-lg hidden sm:inline">RingRouter</span>
          </div>
          <div className="flex items-center gap-2 md:gap-3">
            <Link
              to="/auth/login"
              className="px-3 md:px-4 py-2 text-sm text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
            >
              {t('home.signIn')}
            </Link>
            <Link
              to="/auth/login"
              className="px-3 md:px-4 py-2 text-sm bg-primary text-primary-foreground rounded-lg hover:bg-primary-dark transition-colors cursor-pointer"
            >
              {t('home.getStarted')}
            </Link>
          </div>
        </div>
      </header>

      {/* Hero */}
      <section className="relative max-w-6xl mx-auto px-4 md:px-6 pt-20 md:pt-32 pb-16 md:pb-20 text-center">
        {/* Glow background */}
        <div
          className="absolute inset-0 -z-10 flex items-start justify-center"
          aria-hidden="true"
        >
          <div className="w-[600px] h-[400px] bg-primary/20 rounded-full blur-3xl opacity-50" />
        </div>

        <div className="inline-flex items-center gap-2 px-3 py-1 mb-6 rounded-full border border-border bg-card text-xs text-muted-foreground">
          <span className="w-1.5 h-1.5 rounded-full bg-success" />
          OpenAI-compatible · Self-hosted
        </div>

        <h1 className="text-4xl md:text-6xl font-bold tracking-tight mb-5 leading-[1.1]">
          <Trans i18nKey="home.heroTitle">
            Self-Deployed <span className="text-primary">LLM Gateway</span>
          </Trans>
        </h1>
        <p className="text-base md:text-lg text-muted-foreground max-w-2xl mx-auto mb-10 leading-relaxed">
          {t('home.heroDesc')}
        </p>
        <div className="flex flex-col sm:flex-row items-center justify-center gap-3">
          <Link
            to="/auth/login"
            className="px-6 py-3 bg-primary text-primary-foreground rounded-lg font-medium hover:bg-primary-dark transition-colors inline-flex items-center gap-2 w-full sm:w-auto justify-center cursor-pointer"
          >
            {t('home.getStarted')} <ArrowRight size={18} strokeWidth={2} />
          </Link>
          <a
            href="https://github.com/RingKoAI/RingRouter"
            target="_blank"
            rel="noopener noreferrer"
            className="px-6 py-3 border border-border rounded-lg font-medium hover:bg-muted transition-colors w-full sm:w-auto text-center cursor-pointer"
          >
            {t('home.viewGitHub')}
          </a>
        </div>
      </section>

      {/* Features */}
      <section className="max-w-6xl mx-auto px-4 md:px-6 pb-20 md:pb-32">
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 md:gap-6">
          {featureKeys.map((key, i) => {
            const Icon = featureIcons[i]
            return (
              <div
                key={key}
                className="group bg-card rounded-xl border border-border p-5 md:p-6 hover:border-primary/30 transition-colors"
              >
                <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center mb-4 group-hover:bg-primary/15 transition-colors">
                  <Icon size={20} className="text-primary" strokeWidth={2} />
                </div>
                <h3 className="font-semibold mb-2">{t(`features.${key}.title`)}</h3>
                <p className="text-sm text-muted-foreground leading-relaxed">
                  {t(`features.${key}.desc`)}
                </p>
              </div>
            )
          })}
        </div>
      </section>

      {/* Footer */}
      <footer className="border-t border-border py-8">
        <div className="max-w-6xl mx-auto px-4 md:px-6 text-center text-sm text-muted-foreground">
          {t('home.footer')}
        </div>
      </footer>
    </div>
  )
}