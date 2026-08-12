import { Home, Lock } from '@appica/icons-react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { AppButton } from '@/components/common/app-button'

function StatusPage({ code, title, description }: { code: string; title: string; description: string }) {
  const { t } = useTranslation()
  return (
    <div className="grid min-h-[calc(100vh-8rem)] place-items-center px-4 text-center">
      <div><p className="font-mono text-sm font-bold tracking-[0.3em] text-emerald-700 dark:text-emerald-400">{code}</p><span className="mx-auto mt-5 grid size-16 place-items-center rounded-2xl bg-neutral-100 text-neutral-500 dark:bg-neutral-800"><Lock size={28} /></span><h1 className="mt-5 text-3xl font-bold tracking-tight">{title}</h1><p className="mx-auto mt-2 max-w-md text-neutral-500">{description}</p><AppButton className="mt-7" render={<Link to="/" />}><Home size={17} />{t('statusPages.returnHome')}</AppButton></div>
    </div>
  )
}

export function ForbiddenPage() { const { t } = useTranslation(); return <StatusPage code="403" title={t('statusPages.forbiddenTitle')} description={t('statusPages.forbiddenDescription')} /> }
export function NotFoundPage() { const { t } = useTranslation(); return <StatusPage code="404" title={t('statusPages.notFoundTitle')} description={t('statusPages.notFoundDescription')} /> }
