import { useTranslation } from 'react-i18next'
import { KeyRound, Power, Trash2 } from 'lucide-react'
import { type Token } from './use-tokens'
import { TableSkeleton } from '../../components/ui/primitives'

interface Props {
  tokens: Token[]
  loading: boolean
  onToggle: (t: Token) => void
  onDelete: (id: number) => void
}

export default function KeyTable({ tokens, loading, onToggle, onDelete }: Props) {
  const { t } = useTranslation()

  if (loading && tokens.length === 0) {
    return <div className="p-4"><TableSkeleton rows={5} cols={5} /></div>
  }

  if (tokens.length === 0) {
    return null
  }

  return (
    <div className="border border-border rounded-xl overflow-hidden bg-card">
      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border bg-muted/50 text-left">
              <th className="px-4 py-3 font-medium text-muted-foreground">{t('users.colUser')}</th>
              <th className="px-4 py-3 font-medium text-muted-foreground">Key</th>
              <th className="px-4 py-3 font-medium text-muted-foreground">{t('users.colGroup')}</th>
              <th className="px-4 py-3 font-medium text-muted-foreground">{t('users.colStatus')}</th>
              <th className="px-4 py-3 font-medium text-muted-foreground text-right">{t('users.colActions')}</th>
            </tr>
          </thead>
          <tbody>
            {tokens.map((tok) => (
              <tr key={tok.id} className="border-b border-border last:border-0 hover:bg-muted/30 transition-colors">
                <td className="px-4 py-3 font-medium"><span className="flex items-center gap-2"><KeyRound size={14} className="text-muted-foreground" />{tok.name}</span></td>
                <td className="px-4 py-3 font-mono text-xs text-muted-foreground">{tok.key_masked}</td>
                <td className="px-4 py-3">{tok.group}</td>
                <td className="px-4 py-3">
                  <span className={`inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium ${
                    tok.status === 'active' ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400' : 'bg-red-500/10 text-red-600 dark:text-red-400'}`}>
                    <span className={`w-1.5 h-1.5 rounded-full ${tok.status === 'active' ? 'bg-emerald-500' : 'bg-red-500'}`} />
                    {tok.status === 'active' ? t('users.stActive') : t('users.stDisabled')}
                  </span>
                </td>
                <td className="px-4 py-3">
                  <div className="flex justify-end gap-1">
                    <button title={tok.status === 'active' ? t('users.disable') : t('users.enable')} onClick={() => onToggle(tok)}
                      className="p-1.5 rounded-lg hover:bg-muted text-muted-foreground hover:text-foreground cursor-pointer"><Power size={14} /></button>
                    <button title={t('users.delete')} onClick={() => onDelete(tok.id)}
                      className="p-1.5 rounded-lg hover:bg-red-500/10 text-muted-foreground hover:text-red-500 cursor-pointer"><Trash2 size={14} /></button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}