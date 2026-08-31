import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'
import { Zap, ExternalLink } from 'lucide-react'
import { useSite } from '../contexts/SiteContext'

interface FooterLink {
  label: string
  href: string
  external?: boolean
}

interface FooterColumn {
  titleKey: string
  links: FooterLink[]
}

/** Link columns (new-api footer style): about / docs / related projects. */
function useFooterColumns(): FooterColumn[] {
  const { t } = useTranslation()
  return [
    {
      titleKey: 'footer.colAbout',
      links: [
        { label: t('footer.linkAbout'), href: '/about' },
        { label: 'GitHub', href: 'https://github.com/RingKoAI/RingRouter', external: true },
      ],
    },
    {
      titleKey: 'footer.colDocs',
      links: [
        { label: t('footer.linkReadme'), href: 'https://github.com/RingKoAI/RingRouter/blob/main/README.md', external: true },
        { label: t('footer.linkApi'), href: 'https://github.com/RingKoAI/RingRouter#使用方法', external: true },
      ],
    },
    {
      titleKey: 'footer.colRelated',
      links: [
        { label: 'one-api', href: 'https://github.com/songquanpeng/one-api', external: true },
        { label: 'new-api', href: 'https://github.com/QuantumNous/new-api', external: true },
      ],
    },
  ]
}

/**
 * Public-site footer, new-api style: brand column, three link columns,
 * an instance copyright row, and the upstream project attribution
 * (AGPL-3.0 keeps the attribution chain intact).
 */
export default function SiteFooter() {
  const { t } = useTranslation()
  const { siteName, version } = useSite()
  const columns = useFooterColumns()
  const year = new Date().getFullYear()

  return (
    <footer className="border-t border-border/50 relative z-10">
      <div className="max-w-6xl mx-auto px-4 md:px-6 py-10 md:py-12">
        <div className="flex flex-col justify-between gap-8 md:flex-row md:gap-16">

          {/* Brand column */}
          <div className="shrink-0 max-w-xs">
            <Link to="/" className="group flex items-center gap-2.5">
              <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-primary to-primary/85 flex items-center justify-center shadow-md shadow-primary/25 transition-transform duration-300 group-hover:scale-105 group-hover:rotate-3">
                <Zap size={15} className="text-primary-foreground" strokeWidth={2.5} />
              </div>
              <span className="font-semibold tracking-tight">{siteName}</span>
              {version && (
                <span className="px-1.5 py-px text-[9px] font-mono rounded bg-muted text-muted-foreground leading-4">
                  v{version}
                </span>
              )}
            </Link>
            <p className="text-[13px] text-muted-foreground leading-relaxed mt-3">
              {t('footer.tagline')}
            </p>
          </div>

          {/* Link columns */}
          <div className="grid grid-cols-2 sm:grid-cols-3 gap-8">
            {columns.map((col) => (
              <div key={col.titleKey}>
                <p className="text-xs font-semibold uppercase tracking-[0.08em] text-muted-foreground/70 mb-3">
                  {t(col.titleKey)}
                </p>
                <ul className="space-y-2.5">
                  {col.links.map((l) => (
                    <li key={l.href}>
                      {l.external ? (
                        <a href={l.href} target="_blank" rel="noopener noreferrer"
                          className="text-sm text-muted-foreground hover:text-foreground transition-colors inline-flex items-center gap-1">
                          {l.label}
                        </a>
                      ) : (
                        <Link to={l.href} className="text-sm text-muted-foreground hover:text-foreground transition-colors">
                          {l.label}
                        </Link>
                      )}
                    </li>
                  ))}
                </ul>
              </div>
            ))}
          </div>
        </div>

        {/* Copyright + attribution rows */}
        <div className="border-t border-border/50 mt-8 pt-6 flex flex-col sm:flex-row items-center justify-between gap-2 text-xs text-muted-foreground/60">
          <p className="whitespace-nowrap">© {year} {siteName}. {t('footer.rights')}</p>
          <p className="text-muted-foreground/45 flex items-center gap-1 flex-wrap justify-center whitespace-nowrap">
            © {year}{' '}
            <a href="https://github.com/RingKoAI/RingRouter" target="_blank" rel="noopener noreferrer"
              className="text-foreground/70 hover:text-foreground font-medium transition-colors">
              RingRouter
            </a>
            . {t('footer.attribution')}
            <a href="https://github.com/RingKoAI/RingRouter/blob/main/LICENSE" target="_blank" rel="noopener noreferrer"
              className="inline-flex items-center gap-1 hover:text-foreground transition-colors">
              <ExternalLink size={11} /> AGPL-3.0
            </a>
          </p>
        </div>
      </div>
    </footer>
  )
}
