import { Badge } from '@appica/ui-react/badge'
import { Dialog, DialogBody, DialogClose, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@appica/ui-react/dialog'
import { Copy, ExternalLink, Refresh } from '@appica/icons-react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { api } from '@/api/services'
import type { BuildInfo, UpdateChangeType, UpdateStatus } from '@/api/types'
import { AppButton } from '@/components/common/app-button'
import { ErrorState } from '@/components/common/feedback'
import { i18n } from '@/i18n'
import { formatDateTime } from '@/i18n/format'
import { copyTextToClipboard } from '@/lib/clipboard'
import { errorMessage } from '@/lib/error-message'

const updateQueryKey = ['admin', 'updates'] as const

export function AppVersionStatus({ isAdmin }: { isAdmin: boolean }) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const version = useQuery({ queryKey: ['app', 'version'], queryFn: api.version, staleTime: Number.POSITIVE_INFINITY })
  const updates = useQuery({ queryKey: updateQueryKey, queryFn: () => api.updates(), enabled: isAdmin, staleTime: 6 * 60 * 60 * 1000 })
  const current = updates.data?.current ?? version.data
  const statusText = updateStatusText(t, current, isAdmin, updates.data, updates.isPending, Boolean(updates.error))
  const versionText = current ? `${current.version} · ${t(`updates.channels.${current.channel}`, { defaultValue: current.channel })}` : t('updates.versionUnavailable')
  const highlighted = Boolean(updates.data?.update_available)

  const content = (
    <>
      <span className="block truncate text-[11px] font-medium text-neutral-500 dark:text-neutral-400">{versionText}</span>
      <span className={`block truncate text-[10px] font-semibold ${highlighted ? 'text-amber-600 dark:text-amber-400' : 'text-neutral-400 dark:text-neutral-500'}`}>{statusText}</span>
    </>
  )

  return (
    <>
      {isAdmin ? (
        <button type="button" className="block max-w-full rounded text-left outline-none hover:opacity-80 focus-visible:ring-2 focus-visible:ring-emerald-600" aria-label={t('updates.openDetails')} onClick={() => setOpen(true)}>
          {content}
        </button>
      ) : <span className="block max-w-full">{content}</span>}
      {isAdmin && <UpdateDialog open={open} onOpenChange={setOpen} current={current} status={updates.data} loading={updates.isPending} error={updates.error} />}
    </>
  )
}

function UpdateDialog({ open, onOpenChange, current, status, loading, error }: {
  open: boolean
  onOpenChange: (open: boolean) => void
  current?: BuildInfo
  status?: UpdateStatus
  loading: boolean
  error: unknown
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [copied, setCopied] = useState(false)
  const [copyFailed, setCopyFailed] = useState(false)
  const refresh = useMutation({
    mutationFn: () => api.updates(true),
    onSuccess: (data) => queryClient.setQueryData(updateQueryKey, data),
  })
  const locale = i18n.resolvedLanguage === 'zh-CN' ? 'zh-CN' : 'en'
  const activeStatus = refresh.data ?? status
  const activeError = activeStatus ? undefined : (refresh.error ?? error)
  const activeCurrent = activeStatus?.current ?? current

  const copyImageTag = async () => {
    if (!activeStatus?.image_tag) return
    setCopyFailed(false)
    try {
      await copyTextToClipboard(activeStatus.image_tag)
      setCopied(true)
      window.setTimeout(() => setCopied(false), 1600)
    } catch {
      setCopyFailed(true)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>{t('updates.title')}</DialogTitle>
          <DialogDescription>{t('updates.description')}</DialogDescription>
        </DialogHeader>
        <DialogBody className="max-h-[70vh] space-y-5 overflow-y-auto">
          <div className="flex flex-wrap items-center gap-2 rounded-xl bg-neutral-50 p-4 dark:bg-neutral-900">
            <Badge variant="outline">{t('updates.current')}: {activeCurrent?.version ?? '—'}</Badge>
            {activeCurrent && <Badge variant="soft">{t(`updates.channels.${activeCurrent.channel}`, { defaultValue: activeCurrent.channel })}</Badge>}
            {activeStatus?.latest_version && <Badge variant={activeStatus.update_available ? 'primary' : 'soft'}>{t('updates.latest')}: {activeStatus.latest_version}</Badge>}
          </div>

          {(loading || refresh.isPending) && <p className="py-8 text-center text-sm text-neutral-500">{t('updates.checking')}</p>}
          {!loading && Boolean(activeError) && <ErrorState message={errorMessage(activeError, t('updates.checkFailed'))} onRetry={() => refresh.mutate()} />}
          {!loading && !activeError && activeStatus && !activeStatus.supported && <p className="rounded-xl bg-neutral-50 p-4 text-sm text-neutral-600 dark:bg-neutral-900 dark:text-neutral-300">{t('updates.developmentBuild')}</p>}
          {!loading && !activeError && activeStatus?.supported && !activeStatus.update_available && (
            <p className="rounded-xl bg-emerald-50 p-4 text-sm font-medium text-emerald-800 dark:bg-emerald-950/50 dark:text-emerald-300">{t('updates.upToDate')}</p>
          )}
          {!loading && !activeError && activeStatus?.update_available && (
            <div className="space-y-4">
              <p className="text-sm text-neutral-600 dark:text-neutral-300">{t('updates.rangeDescription', { current: activeStatus.current.version, latest: activeStatus.latest_version, count: activeStatus.releases.length })}</p>
              {activeStatus.releases.map((release) => (
                <article key={release.version} className="rounded-xl border border-neutral-200 p-4 dark:border-neutral-800">
                  <div className="flex flex-wrap items-start justify-between gap-2">
                    <div>
                      <h3 className="font-semibold text-neutral-950 dark:text-white">{release.version}</h3>
                      <p className="mt-1 text-sm text-neutral-600 dark:text-neutral-300">{release.summary[locale]}</p>
                    </div>
                    {release.published_at && <span className="text-xs text-neutral-400">{formatDateTime(release.published_at)}</span>}
                  </div>
                  <ul className="mt-3 space-y-2">
                    {release.changes.map((change) => (
                      <li key={change.id} className="flex items-start gap-2 text-sm text-neutral-700 dark:text-neutral-300">
                        <Badge className="mt-0.5 shrink-0" size="sm" variant="outline">{t(`updates.changeTypes.${change.type}` as `updates.changeTypes.${UpdateChangeType}`)}</Badge>
                        <span>{change.text[locale]}</span>
                      </li>
                    ))}
                  </ul>
                  {release.upgrade_notes[locale].length > 0 && (
                    <div className="mt-4 rounded-lg bg-amber-50 p-3 text-sm text-amber-900 dark:bg-amber-950/40 dark:text-amber-200">
                      <p className="font-semibold">{t('updates.upgradeNotes')}</p>
                      <ul className="mt-1 list-disc space-y-1 pl-5">{release.upgrade_notes[locale].map((note) => <li key={note}>{note}</li>)}</ul>
                    </div>
                  )}
                </article>
              ))}
              <div className="flex flex-wrap gap-2">
                <AppButton variant="outline" onClick={() => void copyImageTag()}><Copy size={16} />{t(copied ? 'common.actions.copied' : 'updates.copyImageTag')}</AppButton>
                {activeStatus.releases.at(-1)?.release_url && <AppButton render={<a href={activeStatus.releases.at(-1)?.release_url} target="_blank" rel="noreferrer" />} variant="outline"><ExternalLink size={16} />{t('updates.viewRelease')}</AppButton>}
              </div>
              {copyFailed && <p className="text-sm text-red-600">{t('common.feedback.copyFailed')}</p>}
            </div>
          )}
          {activeStatus?.checked_at && <p className="text-xs text-neutral-400">{t('updates.checkedAt', { time: formatDateTime(activeStatus.checked_at) })}</p>}
        </DialogBody>
        <DialogFooter>
          <AppButton variant="outline" disabled={refresh.isPending} onClick={() => refresh.mutate()}><Refresh size={16} />{t(refresh.isPending ? 'updates.checking' : 'updates.checkNow')}</AppButton>
          <DialogClose render={<AppButton>{t('common.actions.close')}</AppButton>} />
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function updateStatusText(t: ReturnType<typeof useTranslation>['t'], current: BuildInfo | undefined, isAdmin: boolean, status: UpdateStatus | undefined, pending: boolean, failed: boolean) {
  if (!current) return t('updates.loadingVersion')
  if (!isAdmin) return t(current.channel === 'dev' ? 'updates.developmentBuild' : 'updates.currentVersion')
  if (pending) return t('updates.checking')
  if (failed) return t('updates.checkUnavailable')
  if (!status?.supported) return t('updates.developmentBuild')
  if (status.update_available) return t('updates.newVersion', { version: status.latest_version })
  return t('updates.upToDate')
}
