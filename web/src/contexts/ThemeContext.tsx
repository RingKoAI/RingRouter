import { createContext, useContext, useEffect, useState, type ReactNode } from 'react'

type Theme = 'light' | 'dark' | 'system'

interface ThemeContextValue {
  theme: Theme
  setTheme: (t: Theme) => void
  resolved: 'light' | 'dark'
}

const ThemeContext = createContext<ThemeContextValue>({
  theme: 'system',
  setTheme: () => {},
  resolved: 'light',
})

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [theme, setThemeState] = useState<Theme>(() => {
    return (localStorage.getItem('ringrouter_theme') as Theme) || 'system'
  })

  const getResolved = (t: Theme): 'light' | 'dark' => {
    if (t === 'system') {
      return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
    }
    return t
  }

  const [resolved, setResolved] = useState<'light' | 'dark'>(() => getResolved(theme))

  const setTheme = (t: Theme) => {
    setThemeState(t)
    localStorage.setItem('ringrouter_theme', t)
  }

  useEffect(() => {
    const r = getResolved(theme)
    setResolved(r)
    document.documentElement.classList.toggle('dark', r === 'dark')

    if (theme === 'system') {
      const mq = window.matchMedia('(prefers-color-scheme: dark)')
      const handler = (e: MediaQueryListEvent) => {
        const r = e.matches ? 'dark' : 'light'
        setResolved(r)
        document.documentElement.classList.toggle('dark', r === 'dark')
      }
      mq.addEventListener('change', handler)
      return () => mq.removeEventListener('change', handler)
    }
  }, [theme])

  return (
    <ThemeContext.Provider value={{ theme, setTheme, resolved }}>
      {children}
    </ThemeContext.Provider>
  )
}

export function useTheme() {
  return useContext(ThemeContext)
}