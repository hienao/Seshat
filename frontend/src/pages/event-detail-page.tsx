import { Badge } from '@appica/ui-react/badge'
import { ArrowLeft, Bell, FileText } from '@appica/icons-react'
import { useQuery } from '@tanstack/react-query'
import { Link, useParams } from 'react-router-dom'
import { api } from '@/api/services'
import { AppButton } from '@/components/common/app-button'
import { ErrorState } from '@/components/common/feedback'
import { PageHeader } from '@/components/common/page-header'
import { Panel } from '@/components/common/panel'
import { EventCard } from '@/components/webhook/event-card'
import { errorMessage } from '@/lib/error-message'

export function EventDetailPage() {
  const id = Number(useParams().id)
  const event = useQuery({ queryKey: ['webhooks', 'event', id], queryFn: () => api.event(id), enabled: Number.isFinite(id) && id > 0 })
  const notificationStatus = useQuery({ queryKey: ['notifications', 'event', id], queryFn: () => api.eventNotificationStatus(id), enabled: Number.isFinite(id) && id > 0 })
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
      <Panel title="推送状态" description="该状态按消息接收时生成的推送任务和当前实例配置计算。" icon={<Bell size={20} />}>
        {notificationStatus.isPending ? <p className="text-sm text-neutral-500">正在加载推送状态…</p> : notificationStatus.error ? <ErrorState message={errorMessage(notificationStatus.error, '推送状态不可用')} onRetry={() => void notificationStatus.refetch()} /> : notificationStatus.data && <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between"><div><Badge variant={notificationStatus.data.state === 'succeeded' ? 'success' : notificationStatus.data.state === 'failed' ? 'error' : 'outline'}>{notificationStateLabel(notificationStatus.data.state)}</Badge><p className="mt-2 text-sm text-neutral-500">{notificationStatus.data.reason || notificationStatus.data.delivery?.last_error || '推送任务已创建并进入处理流程。'}</p></div>{notificationStatus.data.delivery && <div className="text-xs text-neutral-500">尝试 {notificationStatus.data.delivery.attempt_count} 次{notificationStatus.data.delivery.sent_at ? ` · ${new Date(notificationStatus.data.delivery.sent_at).toLocaleString()}` : ''}</div>}</div>}
      </Panel>
      <Panel title="标准化展示" description="根据消息类型展示媒体、播放、用户或系统信息，缺失字段会自动省略。">
        <EventCard event={item} detail />
      </Panel>
      <Panel title="原始消息" description="原始内容按纯文本处理，不执行其中的 HTML 或脚本。">
        <pre className="max-h-[620px] overflow-auto rounded-xl bg-neutral-950 p-5 text-sm leading-6 text-neutral-200">{item.raw_body || '消息体为空'}</pre>
      </Panel>
    </div>
  )
}

function notificationStateLabel(state: string) {
  const labels: Record<string, string> = {
    not_configured: '未配置渠道', channel_disabled: '渠道已停用', not_matched: '该类型未推送', not_created: '未创建任务',
    pending: '等待发送', sending: '发送中', retrying: '等待重试', succeeded: '推送成功', failed: '推送失败',
  }
  return labels[state] ?? state
}
