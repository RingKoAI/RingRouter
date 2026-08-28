import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { KeyRound, Search } from 'lucide-react'
import { PageHeader, EmptyState } from '../../components/ui/primitives'
import { useTokens, type StatusFilter } from './use-tokens'
import KeysSidebar from './sidebar'
import KeyTable from './key-table'
import CreateKeyCard from './create-card'

export default function Keys() {
  const { t } = useTranslation()
  const { tokens, loading, create, toggle, remove, reload } = useTokens()
  const [searchInput, setSearchInput] = useState('')
  const [query, setQuery] = useState('')
  const [filter, setFilter] = useState<StatusFilter>('all')
  const [group, setGroup] = useState('')

  const groups = useMemo(() => [...new Set(tokens.map((t) => t.group))], [tokens])

  const filtered = useMemo(() =>
    tokens.filter((t) => {
      if (filter !== 'all' && t.status !== filter) return false
      if (group && t.group !== group) return false
      if (query && !t.name.toLowerCase().includes(query.toLowerCase())) return false
      return true
    }), [tokens, filter, group, query])

  const search = () => {
    setQuery(searchInput.trim())
    setFilter('all')
    setGroup('')
  }

  return (
    <div className="flex gap-6">
      <KeysSidebar
        tokens={tokens}
        filter={filter}
        onFilter={setFilter}
        group={group}
        onPickGroup={setGroup}
        groups={groups}
      />

      <div className="flex-1 min-w-0">
        <PageHeader title={t('keys.title')} subtitle={t('keys.subtitle')} />

        <div className="relative mb-4">
          <Search size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
          <input
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
            onKeyDown={(e) => { if (e.key === 'Enter') search() }}
            placeholder={t('keys.search')}
            className="w-full min-h-[40px] pl-9 pr-3 py-2 border border-input rounded-lg text-sm bg-background focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary"
          />
        </div>

        <CreateKeyCard onCreated={async (name) => create(name)} />

        <KeyTable tokens={filtered} loading={loading} onToggle={toggle} onDelete={remove} />

        {!loading && tokens.length === 0 && (
          <EmptyState icon={KeyRound} title={t('keys.empty')} hint={t('keys.emptyHint')} />
        )}
      </div>
    </div>
  )
}