import { useState } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { ArrowLeft } from 'lucide-react'
import { api } from '../lib/api'

export default function Login() {
  const { t } = useTranslation()
  const [apiKey, setApiKey] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const navigate = useNavigate()

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)

    try {
      const res = await api.get('/v1/models')
      if (res.data || res.object) {
        localStorage.setItem('ringrouter_token', apiKey)
        navigate('/dashboard')
      }
    } catch {
      setError(t('auth.invalidKey'))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-background px-4">
      <div className="w-full max-w-sm">
        {/* Brand */}
        <div className="flex flex-col items-center mb-8">
          <div className="w-12 h-12 rounded-xl bg-primary flex items-center justify-center mb-3">
            <span className="text-primary-foreground font-bold text-xl">R</span>
          </div>
          <h1 className="text-2xl font-bold">{t('auth.signInTitle')}</h1>
          <p className="text-sm text-muted-foreground mt-1">{t('auth.signInDesc')}</p>
        </div>

        <div className="bg-card rounded-xl border border-border p-6 shadow-sm">
          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label htmlFor="apikey" className="block text-sm font-medium mb-1.5">
                {t('auth.adminKey')}
              </label>
              <input
                id="apikey"
                type="password"
                value={apiKey}
                onChange={(e) => setApiKey(e.target.value)}
                className="w-full px-3 py-2 border border-border rounded-lg text-sm bg-background focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary"
                placeholder="sk-..."
                autoFocus
              />
            </div>
            {error && (
              <p className="text-sm text-destructive">{error}</p>
            )}
            <button
              type="submit"
              disabled={loading || !apiKey}
              className="w-full py-2.5 bg-primary text-primary-foreground rounded-lg text-sm font-medium hover:bg-primary-dark disabled:opacity-50 transition-colors cursor-pointer"
            >
              {loading ? t('auth.signingIn') : t('auth.signIn')}
            </button>
          </form>
        </div>

        <div className="mt-4 text-center">
          <Link
            to="/"
            className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
          >
            <ArrowLeft size={14} strokeWidth={2} />
            {t('auth.backHome')}
          </Link>
        </div>
      </div>
    </div>
  )
}