import { Badge } from '@appica/ui-react/badge'
import { ArrowLeft, FileText } from '@appica/icons-react'
import { useQuery } from '@tanstack/react-query'
import { Link, useParams } from 'react-router-dom'
import { api } from '@/api/services'
import { AppButton } from '@/components/common/app-button'
import { ErrorState } from '@/components/common/feedback'
import { PageHeader } from '@/components/common/page-header'
import { Panel } from '@/components/common/panel'
import { errorMessage } from '@/lib/error-message'

export function EventDetailPage() {
  const id = Number(useParams().id)
  const event = useQuery({ queryKey: ['webhooks', 'event', id], queryFn: () => api.event(id), enabled: Number.isFinite(id) && id > 0 })
  if (event.isPending) return <div className="grid min-h-[50vh] place-items-center text-sm text-neutral-500">正在加载消息…</div>
  if (event.error || !event.data) return <div className="mx-auto max-w-4xl px-4 py-10"><ErrorState message={errorMessage(event.error, '消息不存在')} /></div>
  const item = event.data
  return (
    <div className="mx-auto max-w-5xl space-y-7 px-4 py-10 sm:px-6">
      <PageHeader eyebrow="Webhook Event" title={item.title || 'Webhook 消息'} description={item.summary || '原始 Webhook 消息详情'} action={<AppButton render={<Link to="/events" />} size="sm" variant="outline"><ArrowLeft size={16} />返回消息流</AppButton>} />
      <Panel title="消息信息" icon={<FileText size={20} />}>
        <dl className="grid gap-4 sm:grid-cols-4">
          <div><dt className="text-xs text-neutral-500">App</dt><dd className="mt-1 font-semibold">{item.app_code}</dd></div>
          <div><dt className="text-xs text-neutral-500">原始类型</dt><dd className="mt-1 font-mono text-sm">{item.source_event_type || 'unknown'}</dd></div>
          <div><dt className="text-xs text-neutral-500">展示类型</dt><dd className="mt-1"><Badge variant={item.is_fallback ? 'outline' : 'soft'}>{item.is_fallback ? '默认类型' : item.display_event_type}</Badge></dd></div>
          <div><dt className="text-xs text-neutral-500">接收时间</dt><dd className="mt-1 text-sm">{new Date(item.received_at).toLocaleString()}</dd></div>
        </dl>
      </Panel>
      <Panel title="标准化展示" description="该区域由 App 类型渲染器提供，当前默认类型展示通用摘要。">
        <pre className="overflow-auto rounded-xl bg-neutral-950 p-5 text-sm leading-6 text-emerald-100">{JSON.stringify(item.presentation, null, 2)}</pre>
      </Panel>
      <Panel title="原始消息" description="原始内容按纯文本处理，不执行其中的 HTML 或脚本。">
        <pre className="max-h-[620px] overflow-auto rounded-xl bg-neutral-950 p-5 text-sm leading-6 text-neutral-200">{item.raw_body || '消息体为空'}</pre>
      </Panel>
    </div>
  )
}
