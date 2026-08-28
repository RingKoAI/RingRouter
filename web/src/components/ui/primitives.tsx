/**
 * Shared UI primitives: skeleton loading rows, empty states, and page
 * headers so every dashboard page reads and feels the same.
 */

export function Skeleton({ className = '' }: { className?: string }) {
  return <div className={`animate-pulse rounded-md bg-muted ${className}`} />
}

/** Table skeleton: header band + N body rows with shimmering cells. */
export function TableSkeleton({ rows = 6, cols = 5 }: { rows?: number; cols?: number }) {
  return (
    <div className="border border-border rounded-xl overflow-hidden bg-card">
      <div className="flex gap-4 px-4 py-3 bg-muted/50">
        {Array.from({ length: cols }).map((_, i) => (
          <Skeleton key={i} className="h-3.5 flex-1" />
        ))}
      </div>
      {Array.from({ length: rows }).map((_, r) => (
        <div key={r} className="flex gap-4 px-4 py-3.5 border-t border-border" style={{ opacity: 1 - r * 0.08 }}>
          {Array.from({ length: cols }).map((_, c) => (
            <Skeleton key={c} className={`h-3.5 ${c === 0 ? 'flex-[1.4]' : 'flex-1'}`} />
          ))}
        </div>
      ))}
    </div>
  )
}

/** Card-grid skeleton for tile layouts (plans, plaza cards). */
export function CardsSkeleton({ count = 6 }: { count?: number }) {
  return (
    <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
      {Array.from({ length: count }).map((_, i) => (
        <div key={i} className="p-4 rounded-2xl border border-border bg-card space-y-3">
          <div className="flex justify-between">
            <Skeleton className="h-4 w-1/2" />
            <Skeleton className="h-4 w-6" />
          </div>
          <Skeleton className="h-3 w-3/4" />
          <div className="flex gap-2">
            <Skeleton className="h-3 w-14" />
            <Skeleton className="h-3 w-10" />
          </div>
        </div>
      ))}
    </div>
  )
}

/** Unified empty state with icon, copy, and an optional action. */
export function EmptyState({ icon: Icon, title, hint, action }: {
  icon: React.ComponentType<{ size?: number; className?: string }>
  title: string
  hint?: string
  action?: React.ReactNode
}) {
  return (
    <div className="flex flex-col items-center justify-center py-16 gap-3 text-center">
      <div className="w-14 h-14 rounded-2xl bg-muted flex items-center justify-center">
        <Icon size={22} className="text-muted-foreground" />
      </div>
      <div>
        <p className="text-sm font-medium">{title}</p>
        {hint && <p className="text-xs text-muted-foreground mt-1 max-w-xs mx-auto leading-relaxed">{hint}</p>}
      </div>
      {action}
    </div>
  )
}

/** Standard dashboard page header with title, subtitle, and right-side slot. */
export function PageHeader({ title, subtitle, actions }: {
  title: string
  subtitle?: string
  actions?: React.ReactNode
}) {
  return (
    <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 mb-6">
      <div>
        <h2 className="text-xl font-semibold tracking-tight">{title}</h2>
        {subtitle && <p className="text-sm text-muted-foreground mt-0.5">{subtitle}</p>}
      </div>
      {actions && <div className="flex items-center gap-2">{actions}</div>}
    </div>
  )
}
