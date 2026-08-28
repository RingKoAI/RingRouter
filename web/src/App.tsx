import { useEffect, useState } from 'react'
import { Routes, Route, Navigate } from 'react-router-dom'
import { api } from './lib/api'
import { useAuth } from './contexts/AuthContext'
import Home from './pages/Home'
import Login from './pages/Login'
import Register from './pages/Register'
import ForgotPassword from './pages/ForgotPassword'
import Setup from './pages/Setup'
import Dashboard from './pages/Dashboard'
import Channels from './pages/Channels'
import Users from './pages/Users'
import Profile from './pages/Profile'
import Keys from './pages/Keys'
import Logs from './pages/Logs'
import Settings from './pages/Settings'
import Subscriptions from './pages/Subscriptions'
import Models from './pages/Models'
import System from './pages/System'
import DataBoard from './pages/DataBoard'
import Wallet from './pages/Wallet'
import Playground from './pages/Playground'
import ModelsPlaza from './pages/ModelsPlaza'
import Layout from './components/Layout'

function FullScreenSpinner() {
  return (
    <div className="min-h-screen flex items-center justify-center bg-background">
      <div className="w-6 h-6 rounded-full border-2 border-border border-t-primary animate-spin" />
    </div>
  )
}

function useSetupStatus() {
  const [status, setStatus] = useState<'loading' | 'needed' | 'done'>('loading')
  useEffect(() => {
    api.get<{ needed: boolean }>('/api/setup/status')
      .then((d) => setStatus(d.needed ? 'needed' : 'done'))
      .catch(() => setStatus('done'))
  }, [])
  return status
}

function SetupGate({ children }: { children: React.ReactNode }) {
  const status = useSetupStatus()
  if (status === 'loading') return <FullScreenSpinner />
  if (status === 'needed') return <Navigate to="/setup" replace />
  return <>{children}</>
}

function BlockSetupWhenDone({ children }: { children: React.ReactNode }) {
  const status = useSetupStatus()
  if (status === 'loading') return <FullScreenSpinner />
  if (status === 'done') return <Navigate to="/auth/login" replace />
  return <>{children}</>
}

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { user, loading } = useAuth()
  if (loading) return <FullScreenSpinner />
  if (!user) return <Navigate to="/auth/login" replace />
  return <>{children}</>
}

function Placeholder({ title }: { title: string }) {
  return (
    <div className="flex flex-col items-center justify-center py-20 text-center">
      <div className="w-14 h-14 rounded-2xl bg-muted flex items-center justify-center mb-4">
        <span className="text-2xl">🚧</span>
      </div>
      <h2 className="text-lg font-semibold mb-1">{title}</h2>
      <p className="text-sm text-muted-foreground">即将上线</p>
    </div>
  )
}

export default function App() {
  return (
    <Routes>
      <Route path="/setup" element={<BlockSetupWhenDone><Setup /></BlockSetupWhenDone>} />
      <Route path="/" element={<SetupGate><Home /></SetupGate>} />
      <Route path="/models" element={<SetupGate><ModelsPlaza /></SetupGate>} />
      <Route path="/auth/login" element={<SetupGate><Login /></SetupGate>} />
      <Route path="/auth/register" element={<SetupGate><Register /></SetupGate>} />
      <Route path="/auth/forgot" element={<SetupGate><ForgotPassword /></SetupGate>} />

      <Route path="/dash" element={<ProtectedRoute><Layout /></ProtectedRoute>}>
        <Route index element={<Navigate to="/dash/overview" replace />} />
        {/* Chat */}
        <Route path="chat" element={<Placeholder title="聊天" />} />
        <Route path="playground" element={<Playground />} />
        {/* General */}
        <Route path="overview" element={<Dashboard />} />
        <Route path="data" element={<DataBoard />} />
        <Route path="keys" element={<Keys />} />
        <Route path="logs" element={<Logs />} />
        <Route path="task-logs" element={<Placeholder title="任务日志" />} />
        {/* Personal */}
        <Route path="wallet" element={<Wallet />} />
        <Route path="profile" element={<Profile />} />
        {/* Admin */}
        <Route path="manage/channels" element={<Channels />} />
        <Route path="manage/models" element={<Models />} />
        <Route path="manage/users" element={<Users />} />
        <Route path="manage/codes" element={<Placeholder title="兑换码" />} />
        <Route path="manage/subscriptions" element={<Subscriptions />} />
        <Route path="manage/system" element={<System />} />
        <Route path="manage/settings" element={<Settings />} />
      </Route>

      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}
