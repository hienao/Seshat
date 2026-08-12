import { Spinner } from '@appica/ui-react/spinner'
import { useQuery } from '@tanstack/react-query'
import { useEffect } from 'react'
import { useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { api } from '@/api/services'
import { ErrorState } from '@/components/common/feedback'
import { EventCard } from '@/components/webhook/event-card'
import { errorMessage } from '@/lib/error-message'

export function PublicEventPage() {
  const { t } = useTranslation()
  const token = useParams().token ?? ''
  const event = useQuery({
    queryKey: ['public-event', token],
    queryFn: () => api.publicEvent(token),
    enabled: token.length >= 32,
    retry: false,
  })

  useEffect(() => {
    const previousTitle = document.title
    const robots = document.createElement('meta')
    robots.name = 'robots'
    robots.content = 'noindex,nofollow,noarchive'
    const referrer = document.createElement('meta')
    referrer.name = 'referrer'
    referrer.content = 'no-referrer'
    document.head.append(robots, referrer)
    document.title = event.data?.title || t('events.defaultTitle')
    return () => {
      robots.remove()
      referrer.remove()
      document.title = previousTitle
    }
  }, [event.data?.title, t])

  return (
    <main className="min-h-screen bg-neutral-50 px-4 py-10 dark:bg-neutral-950 sm:px-6 sm:py-16">
      <div className="mx-auto max-w-4xl">
        {event.isPending ? <div className="grid min-h-[50vh] place-items-center"><Spinner className="size-8" aria-label={t('events.loading')} /></div>
          : event.error || !event.data ? <div className="grid min-h-[50vh] place-items-center"><ErrorState message={errorMessage(event.error, t('events.missing'))} /></div>
            : <EventCard event={event.data} detail />}
      </div>
    </main>
  )
}
