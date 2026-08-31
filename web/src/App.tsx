import { useEffect, useState } from 'react'
import { Routes, Route, Navigate, Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Hammer, SearchX } from 'lucide-react'
import { api } from './lib/api'
import { useAuth } from './contexts/AuthContext'
import Home from './pages/Home'
import Login from './pages/Login'
import Register from './pages/Register'
import ForgotPassword from './pages/ForgotPassword'
import Setup from './pages/Setup'
import Dashboard from './pages/Dashboard'
import Channels from './pages/Channels'
import Users from './pages/users'
import Profile from './pages/Profile'
import Keys from './pages/keys'
import Logs from './pages/logs'
import Settings from './pages/Settings'
import Subscriptions from './pages/Subscriptions'
import Models from './pages/Models'
import System from './pages/System'
import DataBoard from './pages/DataBoard'
import Wallet from './pages/Wallet'
import Playground from './pages/Playground'
import ModelsPlaza from './pages/ModelsPlaza'
import About from './pages/About'
import Layout from './components/Layout'
import Toaster from './components/Toaster'

function FullScreenSpinner() {
  return (
    <div className="min-h-screen flex items-center justify-center bg-background">
      <div className="w-6 h-6 rounded-full border-2 border-border border-t-primary animate-spin" />
    </div>
  )
}

function useSetupStatus() {
  const [status, setStatus] = useState<'loading' | 'needed' | 'done' | 'error'>('loading')
  useEffect(() => {
    api.get<{ needed: boolean }>('/api/setup/status')
      .then((d) => setStatus(d.needed ? 'needed' : 'done'))
      // A failed status check must not be treated as "setup complete".
      // Otherwise a fresh instance can bypass the wizard into a broken login.
      .catch(() => setStatus('error'))
  }, [])
  return status
}

function SetupUnavailable() {
  const { t } = useTranslation()
  return (
    <div className="min-h-screen flex items-center justify-center bg-background px-4 text-center">
      <div className="max-w-sm space-y-3">
        <h1 className="text-lg font-semibold">{t('common.serviceUnavailable')}</h1>
        <p className="text-sm text-muted-foreground">{t('common.serviceUnavailableDesc')}</p>
        <button onClick={() => window.location.reload()}
          className="min-h-[40px] px-4 rounded-xl bg-primary text-primary-foreground text-sm hover:bg-primary-dark transition-colors cursor-pointer whitespace-nowrap">
          {t('common.retry')}
        </button>
      </div>
    </div>
  )
}

function SetupGate({ children }: { children: React.ReactNode }) {
  const status = useSetupStatus()
  if (status === 'loading') return <FullScreenSpinner />
  if (status === 'error') return <SetupUnavailable />
  if (status === 'needed') return <Navigate to="/setup" replace />
  return <>{children}</>
}

function BlockSetupWhenDone({ children }: { children: React.ReactNode }) {
  const status = useSetupStatus()
  if (status === 'loading') return <FullScreenSpinner />
  if (status === 'error') return <SetupUnavailable />
  if (status === 'done') return <Navigate to="/auth/login" replace />
  return <>{children}</>
}

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { user, loading } = useAuth()
  if (loading) return <FullScreenSpinner />
  if (!user) return <Navigate to="/auth/login" replace />
  return <>{children}</>
}

// Admin pages are hidden from the sidebar, but the URL is still typeable:
// this guard fails fast (the backend re-checks authorization on every API
// call regardless) instead of letting a member page half-load into 403s.
function AdminRoute({ children }: { children: React.ReactNode }) {
  const { user, loading } = useAuth()
  if (loading) return <FullScreenSpinner />
  if (!user) return <Navigate to="/auth/login" replace />
  if (user.role !== 'admin') return <Navigate to="/dash/overview" replace />
  return <>{children}</>
}

function Placeholder({ titleKey }: { titleKey: string }) {
  const { t } = useTranslation()
  return (
    <div className="flex flex-col items-center justify-center py-20 text-center">
      <div className="w-14 h-14 rounded-2xl bg-muted flex items-center justify-center mb-4">
        <Hammer size={22} className="text-muted-foreground" />
      </div>
      <h2 className="text-lg font-semibold mb-1">{t(titleKey)}</h2>
      <p className="text-sm text-muted-foreground">{t('common.comingSoon')}</p>
    </div>
  )
}

function NotFound() {
  const { t } = useTranslation()
  return (
    <div className="min-h-screen flex flex-col items-center justify-center bg-background gap-4 px-4 text-center">
      <div className="w-16 h-16 rounded-2xl bg-muted flex items-center justify-center">
        <SearchX size={26} className="text-muted-foreground" />
      </div>
      <h1 className="text-2xl font-bold tracking-tight">{t('common.notFoundTitle')}</h1>
      <p className="text-sm text-muted-foreground max-w-sm">{t('common.notFoundDesc')}</p>
      <Link
        to="/"
        className="inline-flex items-center gap-2 min-h-[40px] px-5 text-sm bg-primary text-primary-foreground rounded-xl hover:bg-primary-dark transition-colors whitespace-nowrap"
      >
        {t('common.backHome')}
      </Link>
    </div>
  )
}

export default function App() {
  return (
    <>
    <Toaster />
    <Routes>
      <Route path="/setup" element={<BlockSetupWhenDone><Setup /></BlockSetupWhenDone>} />
      <Route path="/" element={<SetupGate><Home /></SetupGate>} />
      <Route path="/models" element={<SetupGate><ModelsPlaza /></SetupGate>} />
      <Route path="/about" element={<SetupGate><About /></SetupGate>} />
      <Route path="/auth/login" element={<SetupGate><Login /></SetupGate>} />
      <Route path="/auth/register" element={<SetupGate><Register /></SetupGate>} />
      <Route path="/auth/forgot" element={<SetupGate><ForgotPassword /></SetupGate>} />

      <Route path="/dash" element={<ProtectedRoute><Layout /></ProtectedRoute>}>
        <Route index element={<Navigate to="/dash/overview" replace />} />
        {/* Chat */}
        <Route path="chat" element={<Placeholder titleKey="nav.chat" />} />
        <Route path="playground" element={<Playground />} />
        {/* General */}
        <Route path="overview" element={<Dashboard />} />
        <Route path="data" element={<DataBoard />} />
        <Route path="keys" element={<Keys />} />
        <Route path="logs" element={<Logs />} />
        <Route path="task-logs" element={<Placeholder titleKey="nav.taskLogs" />} />
        {/* Personal */}
        <Route path="wallet" element={<Wallet />} />
        <Route path="profile" element={<Profile />} />
        {/* Admin */}
        <Route path="manage/channels" element={<AdminRoute><Channels /></AdminRoute>} />
        <Route path="manage/models" element={<AdminRoute><Models /></AdminRoute>} />
        <Route path="manage/users" element={<AdminRoute><Users /></AdminRoute>} />
        <Route path="manage/codes" element={<AdminRoute><Placeholder titleKey="nav.codes" /></AdminRoute>} />
        <Route path="manage/subscriptions" element={<AdminRoute><Subscriptions /></AdminRoute>} />
        <Route path="manage/system" element={<AdminRoute><System /></AdminRoute>} />
        <Route path="manage/settings" element={<AdminRoute><Settings /></AdminRoute>} />
      </Route>

      <Route path="*" element={<NotFound />} />
    </Routes>
    </>
  )
}
