import { useState } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { ArrowLeft } from 'lucide-react'
import { api } from '../lib/api'

export default function Login() {
  const [apiKey, setApiKey] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const navigate = useNavigate()

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)

    try {
      // Validate the API key by calling /v1/models
      const res = await api.get('/v1/models')
      if (res.data || res.object) {
        localStorage.setItem('ringrouter_token', apiKey)
        navigate('/dashboard')
      }
    } catch {
      setError('Invalid API key. Please try again.')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-surface-alt">
      <div className="w-full max-w-sm">
        <div className="bg-white rounded-xl shadow-sm border border-border p-8">
          <div className="text-center mb-6">
            <h1 className="text-2xl font-bold text-primary">RingRouter</h1>
            <p className="text-sm text-text-muted mt-1">Sign in with your API key</p>
          </div>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label htmlFor="apikey" className="block text-sm font-medium text-text mb-1">
                Admin API Key
              </label>
              <input
                id="apikey"
                type="password"
                value={apiKey}
                onChange={(e) => setApiKey(e.target.value)}
                className="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary"
                placeholder="sk-..."
                autoFocus
              />
            </div>
            {error && <p className="text-sm text-red-500">{error}</p>}
            <button
              type="submit"
              disabled={loading || !apiKey}
              className="w-full py-2 bg-primary text-white rounded-lg text-sm font-medium hover:bg-primary-dark disabled:opacity-50 transition-colors"
            >
              {loading ? 'Signing in...' : 'Sign In'}
            </button>
          </form>
          <div className="mt-4 text-center">
            <Link
              to="/"
              className="inline-flex items-center gap-1 text-sm text-text-muted hover:text-text transition-colors"
            >
              <ArrowLeft size={14} />
              Back to Home
            </Link>
          </div>
        </div>
      </div>
    </div>
  )
}