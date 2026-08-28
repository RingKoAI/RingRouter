import { useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { LogOut, User, ChevronDown } from 'lucide-react'
import { useAuth } from '../contexts/AuthContext'

/**
 * User avatar with dropdown menu. Shows the user's initial in a circle;
 * clicking opens a dropdown with username, role, and logout action.
 */
export default function UserAvatar() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { user, logout } = useAuth()
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    const onDown = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('mousedown', onDown)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onDown)
      document.removeEventListener('keydown', onKey)
    }
  }, [open])

  if (!user) return null

  const initial = (user.username || user.display_name || 'U').charAt(0).toUpperCase()

  const handleLogout = async () => {
    setOpen(false)
    await logout()
    navigate('/')
  }

  return (
    <div className="relative" ref={ref}>
      <button
        onClick={() => setOpen(!open)}
        className="flex items-center gap-1.5 p-1 rounded-lg hover:bg-muted transition-colors cursor-pointer"
        aria-label={user.username}
      >
        <div className="w-8 h-8 rounded-full bg-gradient-to-br from-primary to-primary-dark flex items-center justify-center shrink-0">
          <span className="text-primary-foreground text-xs font-semibold">{initial}</span>
        </div>
        <ChevronDown
          size={12}
          strokeWidth={2}
          className={`text-muted-foreground transition-transform duration-150 hidden sm:block ${open ? 'rotate-180' : ''}`}
        />
      </button>

      {open && (
        <div className="absolute right-0 top-full mt-2 w-52 bg-card border border-border rounded-xl shadow-lg py-1.5 z-50 animate-[popIn_150ms_ease]">
          {/* User info */}
          <div className="px-3.5 py-2.5 border-b border-border">
            <p className="text-[13px] font-medium leading-tight truncate">{user.username}</p>
            {user.email && (
              <p className="text-[11px] text-muted-foreground leading-tight truncate mt-0.5">{user.email}</p>
            )}
            <span className="inline-block mt-1 px-1.5 py-px text-[9px] font-mono font-medium rounded bg-primary/10 text-primary capitalize">
              {user.role}
            </span>
          </div>

          {/* Actions */}
          <div className="py-1">
            <button
              onClick={() => {
                setOpen(false)
                navigate('/dash/overview')
              }}
              className="flex items-center gap-2.5 px-3.5 py-2 text-[13px] w-full text-muted-foreground hover:bg-muted hover:text-foreground transition-colors cursor-pointer"
            >
              <User size={14} strokeWidth={2} />
              {t('layout.dashboard')}
            </button>
            <button
              onClick={handleLogout}
              className="flex items-center gap-2.5 px-3.5 py-2 text-[13px] w-full text-muted-foreground hover:bg-destructive/10 hover:text-destructive transition-colors cursor-pointer"
            >
              <LogOut size={14} strokeWidth={2} />
              {t('layout.logout')}
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
