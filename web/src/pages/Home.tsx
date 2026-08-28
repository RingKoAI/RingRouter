import { Link } from 'react-router-dom'
import { Trans, useTranslation } from 'react-i18next'
import {
  ArrowRight, Zap, Layers, Shield, Globe, Copy, Check,
  Settings, Link2, BarChart3, Sparkles, ChevronRight,
} from 'lucide-react'
import { useState } from 'react'
import ThemeLangActions from '../components/ThemeLangActions'
import UserAvatar from '../components/UserAvatar'
import { useSite } from '../contexts/SiteContext'
import { useAuth } from '../contexts/AuthContext'
import { Button } from '../components/ui/button'
import { Badge } from '../components/ui/badge'

/* ── Code demo ──────────────────────────────────────────────────────────── */

function CodeDemo() {
  const [copied, setCopied] = useState(false)

  const curlCmd = `curl -X POST \\
  "/v1/chat/completions" \\
  -H "Authorization: Bearer sk-••••" \\
  -d '{
    "model": "your-model",
    "messages": [
      { "role": "user", "content": "..." }
    ]
  }'`

  const responseBody = `{
  "choices": [{
    "message": {
      "content": "Chat request routed."
    }
  }],
  "usage": { "total_tokens": 27 }
}`

  const handleCopy = () => {
    navigator.clipboard.writeText(curlCmd)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <div className="relative mx-auto max-w-3xl mt-14 anim-fade-up anim-delay-4">
      {/* Glow behind card */}
      <div className="absolute -inset-4 bg-gradient-to-r from-primary/20 via-accent/20 to-primary/20 rounded-3xl blur-2xl opacity-40" />

      <div className="relative rounded-2xl border border-border/50 bg-card/80 backdrop-blur-xl shadow-2xl shadow-primary/5 overflow-hidden">
        {/* Protocol tabs */}
        <div className="flex items-center gap-1 border-b border-border/50 px-4 pt-3 bg-muted/30 overflow-x-auto">
          {[
            { name: 'Chat', color: 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400' },
            { name: 'Responses', color: 'bg-sky-500/10 text-sky-600 dark:text-sky-400' },
            { name: 'Claude', color: 'bg-amber-500/10 text-amber-600 dark:text-amber-400' },
            { name: 'Gemini', color: 'bg-violet-500/10 text-violet-600 dark:text-violet-400' },
          ].map((p, i) => (
            <button
              key={p.name}
              className={`px-3 py-1.5 text-xs font-medium rounded-t-lg transition-all whitespace-nowrap ${
                i === 0
                  ? `${p.color} border-b-2 border-current shadow-sm`
                  : 'text-muted-foreground hover:text-foreground hover:bg-muted/50'
              }`}
            >
              {p.name}
            </button>
          ))}
        </div>

        <div className="p-5 md:p-6">
          {/* Request header */}
          <div className="flex items-center gap-2 mb-3">
            <span className="text-[10px] font-bold tracking-wider text-emerald-600 dark:text-emerald-400 bg-emerald-500/10 px-2 py-0.5 rounded-md font-mono">
              POST
            </span>
            <span className="text-xs text-muted-foreground font-mono">/v1/chat/completions</span>
          </div>

          {/* Request */}
          <div className="flex items-center gap-2 mb-2">
            <span className="text-[10px] font-bold uppercase tracking-[0.12em] text-muted-foreground/60">REQUEST</span>
            <span className="text-[10px] text-muted-foreground/40 font-mono">curl</span>
          </div>

          <div className="relative rounded-xl bg-muted/40 border border-border/40 overflow-hidden group">
            <pre className="p-4 text-[12px] leading-relaxed font-mono text-foreground/90 overflow-x-auto">
              <code>
                <span className="text-emerald-600 dark:text-emerald-400">curl</span>{' '}
                <span className="text-amber-600 dark:text-amber-400">-X</span>{' '}
                <span className="text-sky-600 dark:text-sky-400">POST</span>{' '}
                <span className="text-primary">"/v1/chat/completions"</span>{' '}
                <span className="text-emerald-600 dark:text-emerald-400">-H</span>{' '}
                <span className="text-primary">"Authorization: Bearer sk-••••"</span>{' '}
                <span className="text-emerald-600 dark:text-emerald-400">-d</span>{' '}
                <span className="text-muted-foreground">'{`{`}</span>{'\n'}
                {'  '}"model": <span className="text-primary">"your-model"</span>,{'\n'}
                {'  '}"messages": [{'\n'}
                {'    '}{`{`} "role": <span className="text-primary">"user"</span>, "content": <span className="text-primary">"..."</span> {`}`},{'\n'}
                {'  '}]{'\n'}
                <span className="text-muted-foreground">{`}`}'</span>
              </code>
            </pre>
            <button
              onClick={handleCopy}
              className="absolute top-2 right-2 p-1.5 rounded-md bg-card/80 border border-border/50 hover:bg-muted transition-all opacity-0 group-hover:opacity-100"
            >
              {copied ? <Check size={12} className="text-emerald-500" /> : <Copy size={12} className="text-muted-foreground" />}
            </button>
          </div>

          {/* Response */}
          <div className="mt-4">
            <span className="text-[10px] font-bold uppercase tracking-[0.12em] text-muted-foreground/60">RESPONSE</span>
            <div className="mt-2 rounded-xl bg-muted/40 border border-border/40 overflow-hidden">
              <pre className="p-4 text-[12px] leading-relaxed font-mono text-foreground/80 overflow-x-auto">
                <code>
                  <span className="text-muted-foreground">{`{`}</span>{'\n'}
                  {'  '}"choices": [{'{ '}"message": {'{ '}"content":{'\n'}
                  {'    '}<span className="text-emerald-600 dark:text-emerald-400">"Chat request routed."</span>{'\n'}
                  {'  '}{'} '}]{'} '}],{'\n'}
                  {'  '}"usage": {'{ '}"total_tokens": <span className="text-amber-600 dark:text-amber-400">27</span>{' }'}{'\n'}
                  <span className="text-muted-foreground">{`}`}</span>
                </code>
              </pre>
            </div>
          </div>

          {/* Metrics */}
          <div className="flex items-center gap-5 mt-4 pt-4 border-t border-border/40">
            <div className="flex items-baseline gap-1.5">
              <span className="text-lg font-bold font-mono text-primary tabular-nums">142</span>
              <span className="text-[10px] text-muted-foreground font-medium">MS</span>
            </div>
            <div className="w-px h-4 bg-border/40" />
            <div className="flex items-baseline gap-1.5">
              <span className="text-lg font-bold font-mono text-primary tabular-nums">27</span>
              <span className="text-[10px] text-muted-foreground font-medium">TOKENS</span>
            </div>
            <div className="w-px h-4 bg-border/40" />
            <div className="flex items-baseline gap-1.5">
              <span className="text-[10px] text-muted-foreground font-medium">COST</span>
              <span className="text-lg font-bold font-mono text-primary tabular-nums">$0.00081</span>
            </div>
            <div className="ml-auto">
              <Badge variant="outline" className="text-[10px] font-mono border-emerald-500/30 text-emerald-600 dark:text-emerald-400 bg-emerald-500/5">
                STREAM · SSE
              </Badge>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

/* ── Home page ──────────────────────────────────────────────────────────── */

export default function Home() {
  const { t } = useTranslation()
  const { siteName } = useSite()
  const { user } = useAuth()

  const features = [
    { icon: Zap, color: 'from-amber-500 to-orange-500', bg: 'bg-amber-500/10', key: 'speed' },
    { icon: Shield, color: 'from-emerald-500 to-teal-500', bg: 'bg-emerald-500/10', key: 'secure' },
    { icon: Globe, color: 'from-sky-500 to-blue-500', bg: 'bg-sky-500/10', key: 'global' },
    { icon: Layers, color: 'from-violet-500 to-purple-500', bg: 'bg-violet-500/10', key: 'dev' },
  ]

  const workflow = [
    { icon: Settings, key: 'configure', num: '01' },
    { icon: Link2, key: 'connect', num: '02' },
    { icon: BarChart3, key: 'monitor', num: '03' },
  ]

  return (
    <div className="min-h-screen bg-background relative overflow-hidden">
      {/* ── Grid background pattern ── */}
      <div className="absolute inset-0 -z-10" aria-hidden="true">
        <div
          className="absolute inset-0 opacity-[0.03] dark:opacity-[0.05]"
          style={{
            backgroundImage: 'linear-gradient(var(--color-foreground) 1px, transparent 1px), linear-gradient(90deg, var(--color-foreground) 1px, transparent 1px)',
            backgroundSize: '64px 64px',
            maskImage: 'radial-gradient(ellipse 80% 50% at 50% 30%, black, transparent)',
            WebkitMaskImage: 'radial-gradient(ellipse 80% 50% at 50% 30%, black, transparent)',
          }}
        />
      </div>

      {/* ── Header ── */}
      <header className="sticky top-0 z-30 border-b border-border/50 bg-background/70 backdrop-blur-xl">
        <div className="max-w-6xl mx-auto px-4 md:px-6 h-16 flex items-center justify-between">
          <Link to="/" className="flex items-center gap-2.5 group">
            <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-primary to-primary/80 flex items-center justify-center shadow-lg shadow-primary/20 group-hover:shadow-primary/30 transition-shadow">
              <Zap size={16} className="text-primary-foreground" strokeWidth={2.5} />
            </div>
            <span className="font-semibold text-lg hidden sm:inline tracking-tight">{siteName}</span>
          </Link>
          <div className="flex items-center gap-2 md:gap-3">
            <ThemeLangActions compact />
            {user ? (
              <UserAvatar />
            ) : (
              <>
                <Button variant="ghost" size="sm" asChild className="text-muted-foreground">
                  <Link to="/models">{t('home.plaza')}</Link>
                </Button>
                <Button variant="ghost" size="sm" asChild className="text-muted-foreground">
                  <Link to="/auth/login">{t('home.signIn')}</Link>
                </Button>
                <Button size="sm" asChild className="shadow-lg shadow-primary/20 hover:shadow-primary/30 transition-shadow">
                  <Link to="/auth/login">{t('home.getStarted')}</Link>
                </Button>
              </>
            )}
          </div>
        </div>
      </header>

      {/* ── Hero ── */}
      <section className="relative max-w-6xl mx-auto px-4 md:px-6 pt-24 md:pt-32 pb-8 text-center">
        {/* Multi-layer glow */}
        <div className="absolute top-0 left-1/2 -translate-x-1/2 w-[800px] h-[600px] -z-10" aria-hidden="true">
          <div className="absolute inset-0 bg-gradient-to-b from-primary/20 via-primary/5 to-transparent rounded-full blur-3xl" />
          <div className="absolute top-20 left-1/4 w-96 h-96 bg-accent/10 rounded-full blur-3xl" />
          <div className="absolute top-10 right-1/4 w-80 h-80 bg-sky-500/10 rounded-full blur-3xl" />
        </div>

        <Badge variant="secondary" className="mb-8 anim-fade-up border-primary/20 bg-primary/5 text-primary px-3 py-1">
          <Sparkles size={12} className="mr-1" />
          {t('home.badge')}
        </Badge>

        <h1 className="text-4xl md:text-6xl lg:text-7xl font-bold tracking-tight mb-6 leading-[1.08] anim-fade-up anim-delay-1">
          <Trans i18nKey="home.heroTitle">
            {siteName} — <span className="bg-gradient-to-r from-primary via-primary/80 to-accent bg-clip-text text-transparent">LLM Gateway</span>
          </Trans>
        </h1>

        <p className="text-base md:text-lg text-muted-foreground max-w-2xl mx-auto mb-10 leading-relaxed anim-fade-up anim-delay-2">
          {t('home.heroDesc')}
        </p>

        {/* CTA */}
        <div className="flex flex-col sm:flex-row items-center justify-center gap-3 anim-fade-up anim-delay-3">
          <Button size="lg" asChild className="shadow-xl shadow-primary/25 hover:shadow-primary/35 transition-all px-8">
            <Link to="/auth/login" className="gap-2">
              {t('home.getStarted')} <ArrowRight size={16} />
            </Link>
          </Button>
          <Button variant="outline" size="lg" asChild className="border-border/50 hover:bg-muted/50">
            <a href="https://github.com/RingKoAI/RingRouter" target="_blank" rel="noopener noreferrer">
              {t('home.viewGitHub')}
            </a>
          </Button>
        </div>

        <CodeDemo />
      </section>

      {/* ── Supported apps ── */}
      <section className="max-w-6xl mx-auto px-4 md:px-6 py-16 text-center anim-fade-up anim-delay-5">
        <p className="text-xs font-semibold uppercase tracking-[0.15em] text-muted-foreground/50 mb-5">
          {t('home.compatibleApps')}
        </p>
        <div className="flex items-center justify-center gap-3 flex-wrap">
          {[
            { name: 'Cherry Studio', url: 'https://cherry-ai.com/' },
            { name: 'CC Switch', url: 'https://ccswitch.io/' },
          ].map((app) => (
            <a
              key={app.name}
              href={app.url}
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center gap-2 px-5 py-2.5 rounded-xl border border-border/40 bg-card/50 hover:border-primary/30 hover:bg-card hover:shadow-md transition-all text-sm font-medium"
            >
              {app.name}
              <ChevronRight size={14} className="text-muted-foreground" />
            </a>
          ))}
          <span className="text-sm text-muted-foreground/40 px-2">+</span>
        </div>
      </section>

      {/* ── Stats ── */}
      <section className="max-w-6xl mx-auto px-4 md:px-6 pb-20">
        <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
          {[
            { value: '10+', key: 'adapters', gradient: 'from-primary/10 to-primary/5' },
            { value: '10+', key: 'billing', gradient: 'from-accent/10 to-accent/5' },
            { value: '10+', key: 'routing', gradient: 'from-sky-500/10 to-sky-500/5' },
            { value: '∞', key: 'control', gradient: 'from-violet-500/10 to-violet-500/5' },
          ].map((s) => (
            <div key={s.key} className={`relative p-5 rounded-2xl border border-border/40 bg-gradient-to-br ${s.gradient} backdrop-blur-sm text-center hover:border-primary/20 transition-colors`}>
              <div className="text-3xl md:text-4xl font-bold bg-gradient-to-br from-foreground to-foreground/70 bg-clip-text text-transparent">
                {s.value}
              </div>
              <div className="text-xs text-muted-foreground mt-2 font-medium">{t(`home.stats.${s.key}`)}</div>
            </div>
          ))}
        </div>
      </section>

      {/* ── Protocol showcase ── */}
      <section className="max-w-6xl mx-auto px-4 md:px-6 pb-20">
        <div className="flex flex-wrap items-center justify-center gap-3">
          {[
            { label: 'OpenAI', icon: '🟢' },
            { label: 'Claude', icon: '🟠' },
            { label: 'Gemini', icon: '🔵' },
            { label: 'DeepSeek', icon: '🟣' },
            { label: 'Qwen', icon: '🔴' },
            { label: 'Llama', icon: '🦙' },
          ].map((p) => (
            <span
              key={p.label}
              className="inline-flex items-center gap-1.5 px-4 py-2 rounded-full border border-border/40 bg-card/50 text-sm font-medium text-muted-foreground hover:border-primary/30 hover:text-foreground hover:bg-card hover:shadow-md transition-all"
            >
              <span>{p.icon}</span>
              {p.label}
            </span>
          ))}
        </div>
      </section>

      {/* ── Features ── */}
      <section className="max-w-6xl mx-auto px-4 md:px-6 pb-24">
        <div className="text-center mb-12">
          <h2 className="text-2xl md:text-4xl font-bold tracking-tight">
            <Trans i18nKey="home.featuresTitle">
              <span className="bg-gradient-to-r from-primary to-accent bg-clip-text text-transparent">Built for developers</span>, designed for scale
            </Trans>
          </h2>
        </div>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          {features.map(({ icon: Icon, color, bg, key }, i) => (
            <div
              key={key}
              className={`group relative p-6 rounded-2xl border border-border/40 bg-card/50 backdrop-blur-sm hover:border-primary/20 hover:shadow-lg hover:shadow-primary/5 transition-all duration-300 anim-fade-up anim-delay-${i + 1}`}
            >
              <div className={`w-12 h-12 rounded-xl ${bg} flex items-center justify-center mb-4 group-hover:scale-110 transition-transform duration-300`}>
                <Icon size={22} className="text-foreground/80" strokeWidth={1.8} />
              </div>
              <h3 className="font-semibold text-base mb-2">{t(`home.features.${key}.title`)}</h3>
              <p className="text-sm text-muted-foreground leading-relaxed">
                {t(`home.features.${key}.desc`)}
              </p>
              <div className={`absolute top-0 right-0 w-32 h-32 bg-gradient-to-br ${color} opacity-0 group-hover:opacity-5 rounded-2xl blur-2xl transition-opacity duration-500 -z-10`} />
            </div>
          ))}
        </div>
      </section>

      {/* ── Workflow ── */}
      <section className="max-w-6xl mx-auto px-4 md:px-6 pb-24">
        <div className="text-center mb-12">
          <p className="text-xs font-semibold uppercase tracking-[0.15em] text-primary mb-3">
            {t('home.workflowLabel')}
          </p>
          <h2 className="text-2xl md:text-4xl font-bold tracking-tight">
            {t('home.workflowTitle')}
          </h2>
        </div>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6 relative">
          {/* Connection line */}
          <div className="hidden md:block absolute top-10 left-[calc(16.67%+24px)] right-[calc(16.67%+24px)] h-px bg-gradient-to-r from-primary/30 via-accent/30 to-primary/30" />

          {workflow.map(({ icon: Icon, key, num }, i) => (
            <div key={key} className={`relative text-center anim-fade-up anim-delay-${i + 1}`}>
              <div className="relative w-20 h-20 mx-auto mb-5">
                <div className="absolute inset-0 rounded-2xl bg-gradient-to-br from-primary/10 to-accent/10 rotate-3 group-hover:rotate-6 transition-transform" />
                <div className="relative w-full h-full rounded-2xl bg-card border border-border/40 flex items-center justify-center shadow-lg shadow-primary/5">
                  <Icon size={24} className="text-primary" strokeWidth={1.8} />
                </div>
                <span className="absolute -top-2 -right-2 w-6 h-6 rounded-full bg-primary text-primary-foreground text-[10px] font-bold flex items-center justify-center shadow-md">
                  {num}
                </span>
              </div>
              <h3 className="font-semibold mb-2 text-base">{t(`home.workflow.${key}.title`)}</h3>
              <p className="text-sm text-muted-foreground leading-relaxed max-w-xs mx-auto">
                {t(`home.workflow.${key}.desc`)}
              </p>
            </div>
          ))}
        </div>
      </section>

      {/* ── CTA Banner ── */}
      <section className="max-w-6xl mx-auto px-4 md:px-6 pb-24">
        <div className="relative rounded-3xl border border-border/30 bg-gradient-to-br from-primary/5 via-card to-accent/5 p-8 md:p-12 text-center overflow-hidden">
          <div className="absolute -top-20 -right-20 w-64 h-64 bg-primary/10 rounded-full blur-3xl" />
          <div className="absolute -bottom-20 -left-20 w-64 h-64 bg-accent/10 rounded-full blur-3xl" />
          <div className="relative">
            <h2 className="text-2xl md:text-3xl font-bold tracking-tight mb-3">
              {t('home.getStarted')} {siteName}
            </h2>
            <p className="text-muted-foreground mb-6 max-w-lg mx-auto">
              {t('home.heroDesc')}
            </p>
            <Button size="lg" asChild className="shadow-xl shadow-primary/25 px-8">
              <Link to="/auth/login" className="gap-2">
                {t('home.getStarted')} <ArrowRight size={16} />
              </Link>
            </Button>
          </div>
        </div>
      </section>

      {/* ── Footer ── */}
      <footer className="border-t border-border/50 py-8">
        <div className="max-w-6xl mx-auto px-4 md:px-6 flex flex-col md:flex-row items-center justify-between gap-4">
          <div className="flex items-center gap-2">
            <div className="w-6 h-6 rounded-md bg-gradient-to-br from-primary to-primary/80 flex items-center justify-center">
              <Zap size={12} className="text-primary-foreground" strokeWidth={2.5} />
            </div>
            <span className="text-sm font-medium tracking-tight">{siteName}</span>
          </div>
          <p className="text-xs text-muted-foreground/60">
            {t('home.footer')}
          </p>
        </div>
      </footer>
    </div>
  )
}
