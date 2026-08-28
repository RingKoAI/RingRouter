import { useTranslation } from 'react-i18next'
import { Zap, Server, ShieldCheck, Boxes, Fingerprint, Package, ArrowRight } from 'lucide-react'

export default function About() {
  const { t } = useTranslation()

  const features = [
    { icon: Server, key: 'about.f1' },
    { icon: Boxes, key: 'about.f2' },
    { icon: ShieldCheck, key: 'about.f3' },
    { icon: Fingerprint, key: 'about.f4' },
    { icon: Package, key: 'about.f5' },
  ]

  return (
    <main className="max-w-3xl mx-auto px-4 py-16">
      {/* Hero */}
      <div className="text-center mb-12">
        <div className="w-14 h-14 rounded-2xl bg-primary flex items-center justify-center mx-auto mb-4 shadow-lg shadow-primary/20">
          <Zap size={24} className="text-primary-foreground" strokeWidth={2.5} />
        </div>
        <h1 className="text-3xl font-bold tracking-tight mb-3">RingRouter</h1>
        <p className="text-muted-foreground leading-relaxed max-w-xl mx-auto">{t('about.desc')}</p>
      </div>

      {/* Features */}
      <div className="grid sm:grid-cols-2 gap-3 mb-12">
        {features.map(({ icon: Icon, key }) => (
          <div key={key} className="flex items-start gap-3 p-4 rounded-2xl border border-border bg-card">
            <div className="w-9 h-9 rounded-lg bg-primary/10 text-primary flex items-center justify-center shrink-0">
              <Icon size={16} />
            </div>
            <p className="text-sm leading-relaxed text-muted-foreground">{t(key)}</p>
          </div>
        ))}
      </div>

      {/* Links */}
      <div className="flex flex-wrap justify-center gap-3">
        <a href="https://github.com/RingKoAI/RingRouter" target="_blank" rel="noreferrer"
          className="inline-flex items-center gap-1.5 min-h-[42px] px-5 rounded-xl text-sm font-medium bg-primary text-primary-foreground hover:bg-primary-dark transition-colors cursor-pointer">
          GitHub <ArrowRight size={14} />
        </a>
        <a href="https://github.com/RingKoAI/RingRouter/blob/main/README.md" target="_blank" rel="noreferrer"
          className="inline-flex items-center gap-1.5 min-h-[42px] px-5 rounded-xl text-sm font-medium border border-border hover:bg-muted transition-colors cursor-pointer">
          {t('nav.docs')} <ArrowRight size={14} />
        </a>
      </div>

      <p className="text-center text-xs text-muted-foreground mt-12">
        AGPL-3.0 · Inspired by{' '}
        <a className="underline hover:text-foreground" href="https://github.com/songquanpeng/one-api" target="_blank" rel="noreferrer">one-api</a>
      </p>
    </main>
  )
}
