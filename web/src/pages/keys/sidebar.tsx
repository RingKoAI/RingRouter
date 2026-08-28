import { useTranslation } from 'react-i18next'
import { KeyRound, Power, User as UserIcon, Layers } from 'lucide-react'
import { type Token, type StatusFilter } from './use-tokens'

interface Props {
  tokens: Token[]
  filter: StatusFilter
  onFilter: (f: StatusFilter) => void
  onPickGroup: (g: string) => void
  group: string
  groups: string[]
}

function Stat({ icon: Icon, label, value }: {
  icon: typeof KeyRound; label: string; value: number | string
}) {
  return (
    <div className="p-3 rounded-xl bg-muted/50">
      <div className="flex items-center gap-1.5 text-muted-foreground mb-1">
        <Icon size={13} />
        <span className="text-[11px]">{label}</span>
      </div>
      <p className="text-lg font-semibold leading-none">{value}</p>
    </div>
  )
}

/**
 * Page-level sidebar for the API Keys view: quick stats plus filters that
 * slice the visible table without refetching.
 */
export default function KeysSidebar({ tokens, filter, onFilter, group, onPickGroup, groups }: Props) {
  const { t } = useTranslation()

  const stats = {
    total: tokens.length,
    active: tokens.filter((x) => x.status === 'active').length,
    groups: new Set(tokens.map((x) => x.group)).size,
  }

  const base =
    'w-full text-left px-3 py-2 rounded-lg text-[13px] transition-colors cursor-pointer flex items-center gap-2'

  const itemCls = (on: boolean) =>
    `${base} ${on ? 'bg-primary/10 text-primary font-medium' : 'text-muted-foreground hover:bg-muted hover:text-foreground'}`

  return (
    <aside className="hidden lg:flex w-56 shrink-0 flex-col gap-3">
      {/* Stats */}
      <div className="grid grid-cols-2 gap-2">
        <Stat icon={KeyRound} label={t('keys.sbTotal')} value={stats.total} />
        <Stat icon={Power} label={t('keys.sbActive')} value={stats.active} />
      </div>
      <div className="grid grid-cols-1">
        <Stat icon={Layers} label={t('keys.sbGroups')} value={stats.groups} />
      </div>

      {/* Status filter */}
      <div className="border border-border rounded-xl bg-card p-1.5 space-y-0.5">
        <p className="px-3 pt-1 pb-1.5 text-[10px] font-semibold uppercase tracking-[0.08em] text-muted-foreground/70">
          {t('keys.sbStatus')}
        </p>
        <button className={itemCls(filter === 'all')} onClick={() => onFilter('all')}>
          <KeyRound size={14} /> {t('keys.sbAll')}
        </button>
        <button className={itemCls(filter === 'active')} onClick={() => onFilter('active')}>
          <Power size={14} /> {t('keys.sbActive')}
        </button>
        <button className={itemCls(filter === 'disabled')} onClick={() => onFilter('disabled')}>
          <UserIcon size={14} /> {t('keys.sbDisabled')}
        </button>
      </div>

      {/* Group filter */}
      {groups.length > 0 && (
        <div className="border border-border rounded-xl bg-card p-1.5 space-y-0.5">
          <p className="px-3 pt-1 pb-1.5 text-[10px] font-semibold uppercase tracking-[0.08em] text-muted-foreground/70">
            {t('keys.sbGroup')}
          </p>
          <button className={itemCls(group === '')} onClick={() => onPickGroup('')}>
            <Layers size={14} /> {t('keys.sbAll')}
          </button>
          {groups.map((g) => (
            <button key={g} className={itemCls(group === g)} onClick={() => onPickGroup(g)}>
              <Layers size={14} /> <span className="truncate">{g}</span>
            </button>
          ))}
        </div>
      )}
    </aside>
  )
}
