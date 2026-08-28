import { useTheme } from '../contexts/ThemeContext'
import { Toaster as SonnerToaster } from 'sonner'

/**
 * Global toast surface. Mounted once at the app root; pages call
 * `toast.success/error(...)` from sonner for uniform feedback.
 */
export default function Toaster() {
  const { resolved } = useTheme()
  return (
    <SonnerToaster
      theme={resolved}
      position="top-center"
      richColors
      closeButton
      toastOptions={{
        style: {
          borderRadius: '0.75rem',
          border: '1px solid var(--color-border)',
        },
      }}
    />
  )
}
