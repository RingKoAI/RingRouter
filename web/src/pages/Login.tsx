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
    <div className="min-h-screen flex items-center justify-center bg-surface-alt dark:bg-surface-alt px-4">
      <div className="w-full max-w-sm">
        <div className="bg-white dark:bg-surface rounded-xl shadow-sm border border-border p-6 md:p-8">
          <div className="text-center mb-6">
            <h1 className="text-2xl font-bold text-primary">{t('auth.signInTitle')}</h1>
            <p className="text-sm text-text-muted mt-1">{t('auth.signInDesc')}</p>
          </div>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label htmlFor="apikey" className="block text-sm font-medium text-text mb-1">
                {t('auth.adminKey')}
              </label>
              <input
                id="apikey"
                type="password"
                value={apiKey}
                onChange={(e) => setApiKey(e.target.value)}
                className="w-full px-3 py-2 border border-border rounded-lg text-sm bg-white dark:bg-surface-alt focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary"
                autoFocus
              />
            </div>
            {error && <p className="text-sm text-red-500">{error}</p>}
            <button
              type="submit"
              disabled={loading || !apiKey}
              className="w-full py-2.5 bg-primary text-white rounded-lg text-sm font-medium hover:bg-primary-dark disabled:opacity-50 transition-colors"
            >
              {loading ? t('auth.signingIn') : t('auth.signIn')}
            </button>
          </form>
          <div className="mt-4 text-center">
            <Link
              to="/"
              className="inline-flex items-center gap-1 text-sm text-text-muted hover:text-text transition-colors"
            >
              <ArrowLeft size={14} />
              {t('auth.backHome')}
            </Link>
          </div>
        </div>
      </div>
    </div>
  )
}