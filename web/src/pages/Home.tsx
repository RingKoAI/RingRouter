import { Link } from 'react-router-dom'
import { ArrowRight, Zap, Shield, Layers, Globe } from 'lucide-react'

export default function Home() {
  const features = [
    {
      icon: Zap,
      title: 'Unified API',
      desc: 'One API endpoint for all LLM providers. Switch models without changing your code.',
    },
    {
      icon: Layers,
      title: 'Multi-Provider',
      desc: 'Route to OpenAI, Claude, Gemini, Azure, and more. Add new providers in seconds.',
    },
    {
      icon: Shield,
      title: 'API Key Management',
      desc: 'Create, revoke, and rate-limit API keys. Track usage per key and per user.',
    },
    {
      icon: Globe,
      title: 'Self-Deployed',
      desc: 'Runs on your own infrastructure. No data leaves your network. Single binary deployment.',
    },
  ]

  return (
    <div className="min-h-screen bg-surface-alt">
      {/* Header */}
      <header className="border-b border-border bg-white">
        <div className="max-w-6xl mx-auto px-6 h-16 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <div className="w-8 h-8 rounded-lg bg-primary flex items-center justify-center">
              <span className="text-white font-bold text-sm">R</span>
            </div>
            <span className="font-semibold text-lg">RingRouter</span>
          </div>
          <div className="flex items-center gap-3">
            <Link
              to="/login"
              className="px-4 py-2 text-sm text-text-muted hover:text-text transition-colors"
            >
              Sign In
            </Link>
            <Link
              to="/login"
              className="px-4 py-2 text-sm bg-primary text-white rounded-lg hover:bg-primary-dark transition-colors"
            >
              Get Started
            </Link>
          </div>
        </div>
      </header>

      {/* Hero */}
      <section className="max-w-6xl mx-auto px-6 pt-24 pb-16 text-center">
        <h1 className="text-5xl font-bold tracking-tight mb-4">
          Self-Deployed{' '}
          <span className="text-primary">LLM Gateway</span>
        </h1>
        <p className="text-lg text-text-muted max-w-2xl mx-auto mb-8">
          Route requests to any AI provider through a single OpenAI-compatible API.
          Deploy on your own server, keep your data private.
        </p>
        <div className="flex items-center justify-center gap-3">
          <Link
            to="/login"
            className="px-6 py-3 bg-primary text-white rounded-lg font-medium hover:bg-primary-dark transition-colors inline-flex items-center gap-2"
          >
            Get Started <ArrowRight size={18} />
          </Link>
          <a
            href="https://github.com/RingKoAI/RingRouter"
            target="_blank"
            rel="noopener noreferrer"
            className="px-6 py-3 border border-border rounded-lg font-medium text-text hover:bg-surface-alt transition-colors"
          >
            View on GitHub
          </a>
        </div>
      </section>

      {/* Features */}
      <section className="max-w-6xl mx-auto px-6 pb-24">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          {features.map((f) => {
            const Icon = f.icon
            return (
              <div
                key={f.title}
                className="bg-white rounded-xl border border-border p-6 hover:shadow-sm transition-shadow"
              >
                <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center mb-4">
                  <Icon size={20} className="text-primary" />
                </div>
                <h3 className="font-semibold mb-2">{f.title}</h3>
                <p className="text-sm text-text-muted">{f.desc}</p>
              </div>
            )
          })}
        </div>
      </section>

      {/* Footer */}
      <footer className="border-t border-border bg-white py-8">
        <div className="max-w-6xl mx-auto px-6 text-center text-sm text-text-muted">
          RingRouter — MIT Licensed. Built with Go + React.
        </div>
      </footer>
    </div>
  )
}