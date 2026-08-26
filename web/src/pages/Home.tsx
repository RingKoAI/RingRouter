import { Link } from 'react-router-dom'
import { Trans, useTranslation } from 'react-i18next'
import { ArrowRight, Zap, Shield, Layers, Globe } from 'lucide-react'

const featureIcons = [Zap, Layers, Shield, Globe] as const
const featureKeys = ['unified', 'multi', 'keys', 'self'] as const

export default function Home() {
  const { t } = useTranslation()

  return (
    <div className="min-h-screen bg-surface-alt dark:bg-surface-alt">
      {/* Header */}
      <header className="border-b border-border bg-white dark:bg-surface">
        <div className="max-w-6xl mx-auto px-4 md:px-6 h-16 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <div className="w-8 h-8 rounded-lg bg-primary flex items-center justify-center">
              <span className="text-white font-bold text-sm">R</span>
            </div>
            <span className="font-semibold text-lg hidden sm:inline">RingRouter</span>
          </div>
          <div className="flex items-center gap-2 md:gap-3">
            <Link
              to="/auth/login"
              className="px-3 md:px-4 py-2 text-sm text-text-muted hover:text-text transition-colors"
            >
              {t('home.signIn')}
            </Link>
            <Link
              to="/auth/login"
              className="px-3 md:px-4 py-2 text-sm bg-primary text-white rounded-lg hover:bg-primary-dark transition-colors"
            >
              {t('home.getStarted')}
            </Link>
          </div>
        </div>
      </header>

      {/* Hero */}
      <section className="max-w-6xl mx-auto px-4 md:px-6 pt-16 md:pt-24 pb-12 md:pb-16 text-center">
        <h1 className="text-3xl md:text-5xl font-bold tracking-tight mb-4">
          <Trans i18nKey="home.heroTitle">
            Self-Deployed <span className="text-primary">LLM Gateway</span>
          </Trans>
        </h1>
        <p className="text-base md:text-lg text-text-muted max-w-2xl mx-auto mb-8">
          {t('home.heroDesc')}
        </p>
        <div className="flex flex-col sm:flex-row items-center justify-center gap-3">
          <Link
            to="/auth/login"
            className="px-6 py-3 bg-primary text-white rounded-lg font-medium hover:bg-primary-dark transition-colors inline-flex items-center gap-2 w-full sm:w-auto justify-center"
          >
            {t('home.getStarted')} <ArrowRight size={18} />
          </Link>
          <a
            href="https://github.com/RingKoAI/RingRouter"
            target="_blank"
            rel="noopener noreferrer"
            className="px-6 py-3 border border-border rounded-lg font-medium text-text hover:bg-surface-alt dark:hover:bg-surface transition-colors w-full sm:w-auto text-center"
          >
            {t('home.viewGitHub')}
          </a>
        </div>
      </section>

      {/* Features */}
      <section className="max-w-6xl mx-auto px-4 md:px-6 pb-16 md:pb-24">
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 md:gap-6">
          {featureKeys.map((key, i) => {
            const Icon = featureIcons[i]
            return (
              <div
                key={key}
                className="bg-white dark:bg-surface rounded-xl border border-border p-5 md:p-6 hover:shadow-sm transition-shadow"
              >
                <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center mb-4">
                  <Icon size={20} className="text-primary" />
                </div>
                <h3 className="font-semibold mb-2">{t(`features.${key}.title`)}</h3>
                <p className="text-sm text-text-muted">{t(`features.${key}.desc`)}</p>
              </div>
            )
          })}
        </div>
      </section>

      {/* Footer */}
      <footer className="border-t border-border bg-white dark:bg-surface py-8">
        <div className="max-w-6xl mx-auto px-4 md:px-6 text-center text-sm text-text-muted">
          {t('home.footer')}
        </div>
      </footer>
    </div>
  )
}