import { Link } from 'react-router-dom'
import { Trans, useTranslation } from 'react-i18next'
import {
  ArrowRight, Zap, Layers, Shield, Globe, Copy, Check,
  Settings, Link2, BarChart3, Sparkles, ChevronRight, Infinity as InfinityIcon,
} from 'lucide-react'
import { useState } from 'react'
import PublicNav from '../components/PublicNav'
import SiteFooter from '../components/SiteFooter'
import UserAvatar from '../components/UserAvatar'
import { useSite } from '../contexts/SiteContext'
import { useAuth } from '../contexts/AuthContext'
import { Button } from '../components/ui/button'
import { Badge } from '../components/ui/badge'

/* ── Code demo ──────────────────────────────────────────────────────────── */

/** One protocol example: tab metadata + runnable curl + response preview. */
interface ProtocolDemo {
  name: string
  chip: string
  endpoint: string
  curl: string[]
  response: string[]
}

const protocolDemos: ProtocolDemo[] = [
  {
    name: 'Chat', chip: 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400',
    endpoint: '/v1/chat/completions',
    curl: [
      'curl -X POST "/v1/chat/completions" \\',
      '  -H "Authorization: Bearer sk-••••" \\',
      "  -H 'Content-Type: application/json' \\",
      '  -d \'{',
      '       "model": "your-model",',
      '       "messages": [',
      '         { "role": "user", "content": "..." }',
      '       ]',
      "     }'",
    ],
    response: [
      '{',
      '  "choices": [{',
      '    "message": { "content": "Chat request routed." },',
      '    "finish_reason": "stop"',
      '  }],',
      '  "usage": { "total_tokens": 27 }',
      '}',
    ],
  },
  {
    name: 'Responses', chip: 'bg-sky-500/10 text-sky-600 dark:text-sky-400',
    endpoint: '/v1/responses',
    curl: [
      'curl -X POST "/v1/responses" \\',
      '  -H "Authorization: Bearer sk-••••" \\',
      "  -H 'Content-Type: application/json' \\",
      '  -d \'{',
      '       "model": "your-model",',
      '       "input": "..."',
      "     }'",
    ],
    response: [
      '{',
      '  "output": [{',
      '    "type": "message",',
      '    "content": [{ "type": "output_text", "text": "..." }]',
      '  }],',
      '  "usage": { "input_tokens": 18, "output_tokens": 9 }',
      '}',
    ],
  },
  {
    name: 'Claude', chip: 'bg-amber-500/10 text-amber-600 dark:text-amber-400',
    endpoint: '/v1/messages',
    curl: [
      'curl -X POST "/v1/messages" \\',
      '  -H "x-api-key: sk-••••" \\',
      '  -H "anthropic-version: 2023-06-01" \\',
      "  -H 'Content-Type: application/json' \\",
      '  -d \'{',
      '       "model": "your-model",',
      '       "max_tokens": 1024,',
      '       "messages": [',
      '         { "role": "user", "content": "..." }',
      '       ]',
      "     }'",
    ],
    response: [
      '{',
      '  "content": [{',
      '    "type": "text",',
      '    "text": "Chat request routed."',
      '  }],',
      '  "usage": { "input_tokens": 18, "output_tokens": 9 }',
      '}',
    ],
  },
  {
    name: 'Gemini', chip: 'bg-violet-500/10 text-violet-600 dark:text-violet-400',
    endpoint: '/v1beta/models/{model}:generateContent',
    curl: [
      'curl -X POST',
      '  "/v1beta/models/your-model:generateContent" \\',
      '  -H "x-goog-api-key: sk-••••" \\',
      "  -H 'Content-Type: application/json' \\",
      '  -d \'{',
      '       "contents": [{',
      '         "parts": [{ "text": "..." }]',
      '       ]',
      "     }'",
    ],
    response: [
      '{',
      '  "candidates": [{',
      '    "content": { "parts": [{ "text": "..." }] }',
      '  }],',
      '  "usageMetadata": { "totalTokenCount": 27 }',
      '}',
    ],
  },
]

/** Lightweight curl/JSON token colouring for one line. */
function hl(line: string) {
  const parts = line.match(/"[^"]*"|'[^']*'|\S+/g) ?? []
  return parts.map((tok, i) => {
    let cls = 'text-foreground/85'
    if (tok === 'curl') cls = 'text-emerald-600 dark:text-emerald-400'
    else if (tok === 'POST') cls = 'text-sky-600 dark:text-sky-400'
    else if (tok.startsWith('-') && !tok.startsWith('--')) cls = 'text-amber-600 dark:text-amber-400'
    else if (/^["']/.test(tok)) cls = 'text-primary'
    else if (/^-?\d+(\.\d+)?$/.test(tok)) cls = 'text-amber-600 dark:text-amber-400'
    return (
      <span key={i} className={cls}>
        {tok}
        {i < parts.length - 1 ? ' ' : ''}
      </span>
    )
  })
}

function CodeDemo() {
  const [tab, setTab] = useState(0)
  const [copied, setCopied] = useState(false)
  const demo = protocolDemos[tab]

  // Copy text = the exact multi-line curl (continuations included), runnable.
  const curlCmd = demo.curl.join('\n')

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
        {/* Protocol tabs — actually switchable */}
        <div className="flex items-center gap-1 border-b border-border/50 px-4 pt-3 bg-muted/30 overflow-x-auto rr-scroll-thin" role="tablist" aria-label="Protocol">
          {protocolDemos.map((p, i) => (
            <button
              key={p.name}
              role="tab"
              aria-selected={i === tab}
              onClick={() => { setTab(i); setCopied(false) }}
              className={`px-3 py-1.5 text-xs font-medium rounded-t-lg transition-all whitespace-nowrap cursor-pointer ${
                i === tab
                  ? `${p.chip} border-b-2 border-current shadow-sm`
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
            <span className="text-xs text-muted-foreground font-mono break-all">{demo.endpoint}</span>
          </div>

          {/* Request */}
          <div className="flex items-center gap-2 mb-2">
            <span className="text-[10px] font-bold uppercase tracking-[0.12em] text-muted-foreground/60">REQUEST</span>
            <span className="text-[10px] text-muted-foreground/40 font-mono">curl</span>
          </div>

          <div className="relative rounded-xl bg-muted/40 border border-border/40 overflow-hidden group">
            {/* One source line per row: no mid-token wrapping, horizontal
                scroll only when a single line cannot fit. */}
            <pre className="p-4 text-[12px] leading-relaxed font-mono overflow-x-auto rr-scroll-thin">
              {demo.curl.map((line, i) => (
                <code key={i} className="block whitespace-pre">{hl(line)}</code>
              ))}
            </pre>
            <button
              onClick={handleCopy}
              title="Copy"
              className="absolute top-2 right-2 p-1.5 rounded-md bg-card/80 border border-border/50 hover:bg-muted transition-all opacity-0 group-hover:opacity-100 focus-visible:opacity-100 cursor-pointer"
            >
              {copied ? <Check size={12} className="text-emerald-500" /> : <Copy size={12} className="text-muted-foreground" />}
            </button>
          </div>

          {/* Response */}
          <div className="mt-4">
            <span className="text-[10px] font-bold uppercase tracking-[0.12em] text-muted-foreground/60">RESPONSE</span>
            <div className="mt-2 rounded-xl bg-muted/40 border border-border/40 overflow-hidden">
              <pre className="p-4 text-[12px] leading-relaxed font-mono overflow-x-auto rr-scroll-thin">
                {demo.response.map((line, i) => (
                  <code key={i} className="block whitespace-pre text-foreground/80">{hl(line)}</code>
                ))}
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
      <PublicNav />

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
          <Button
            size="lg"
            render={<Link to="/auth/login" />}
            className="shadow-xl shadow-primary/25 hover:shadow-primary/35 transition-all px-8 gap-2"
          >
            {t('home.getStarted')} <ArrowRight size={16} />
          </Button>
          <Button
            variant="outline"
            size="lg"
            render={<a href="https://github.com/RingKoAI/RingRouter" target="_blank" rel="noopener noreferrer" />}
            className="border-border/50 hover:bg-muted/50"
          >
            {t('home.viewGitHub')}
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
            { value: '∞', key: 'control', gradient: 'from-violet-500/10 to-violet-500/5', icon: InfinityIcon },
          ].map((s) => (
            <div key={s.key} className={`relative p-5 rounded-2xl border border-border/40 bg-gradient-to-br ${s.gradient} backdrop-blur-sm text-center hover:border-primary/20 transition-colors`}>
              <div className="text-3xl md:text-4xl font-bold bg-gradient-to-br from-foreground to-foreground/70 bg-clip-text text-transparent flex items-center justify-center">
                {s.icon ? <s.icon className="!text-foreground/80" strokeWidth={2.2} /> : s.value}
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
            { label: 'OpenAI', dot: 'bg-emerald-500' },
            { label: 'Claude', dot: 'bg-orange-500' },
            { label: 'Gemini', dot: 'bg-blue-500' },
            { label: 'DeepSeek', dot: 'bg-violet-500' },
            { label: 'Qwen', dot: 'bg-red-500' },
            { label: 'Llama', dot: 'bg-sky-500' },
          ].map((p) => (
            <span
              key={p.label}
              className="inline-flex items-center gap-1.5 px-4 py-2 rounded-full border border-border/40 bg-card/50 text-sm font-medium text-muted-foreground hover:border-primary/30 hover:text-foreground hover:bg-card hover:shadow-md transition-all whitespace-nowrap"
            >
              <span className={`w-2 h-2 rounded-full ${p.dot}`} aria-hidden="true" />
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
            <Button
              size="lg"
              render={<Link to="/auth/login" />}
              className="shadow-xl shadow-primary/25 px-8 gap-2"
            >
              {t('home.getStarted')} <ArrowRight size={16} />
            </Button>
          </div>
        </div>
      </section>

      {/* ── Footer ── */}
      <SiteFooter />
    </div>
  )
}
