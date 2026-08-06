import { Badge } from '@appica/ui-react/badge'
import { Input } from '@appica/ui-react/input'
import { AlertCircle, Download, FileText, Refresh, Trash } from '@appica/icons-react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { api } from '@/api/services'
import { AppButton } from '@/components/common/app-button'
import { EmptyState, ErrorState, Message } from '@/components/common/feedback'
import { PageHeader } from '@/components/common/page-header'
import { Panel } from '@/components/common/panel'
import { errorMessage } from '@/lib/error-message'

type Filters = { startAt: string; endAt: string; method: string; route: string; statusGroup: string; requestId: string; keyword: string }

const inputClass = 'h-10 w-full rounded-lg border border-neutral-300 bg-white px-3 text-sm dark:border-neutral-700 dark:bg-neutral-900'

function initialFilters(): Filters {
  const end = new Date()
  const start = new Date(end.getTime() - 24 * 60 * 60 * 1000)
  return { startAt: toDateTimeLocal(start), endAt: toDateTimeLocal(end), method: '', route: '', statusGroup: '', requestId: '', keyword: '' }
}

function toDateTimeLocal(date: Date) { const offset = date.getTimezoneOffset() * 60000; return new Date(date.getTime() - offset).toISOString().slice(0, 16) }
function toApiTime(value: string) { return value ? new Date(value).toISOString() : undefined }
function apiFilters(filters: Filters) { return { startAt: toApiTime(filters.startAt), endAt: toApiTime(filters.endAt), method: filters.method, route: filters.route, statusGroup: filters.statusGroup, requestId: filters.requestId, keyword: filters.keyword } }
function formatDate(value: string) { return value ? new Date(value).toLocaleString() : '-' }
function statusVariant(status: number): 'soft' | 'outline' | 'error' | 'success' { if (status >= 500) return 'error'; if (status >= 400) return 'outline'; if (status < 300) return 'success'; return 'soft' }

export function AdminLogsPage() {
  const queryClient = useQueryClient()
  const [filters, setFilters] = useState<Filters>(initialFilters)
  const [selectedId, setSelectedId] = useState<number | null>(null)
  const [clearText, setClearText] = useState('')
  const current = apiFilters(filters)
  const logs = useQuery({ queryKey: ['admin', 'api-logs', current], queryFn: () => api.apiLogs(current) })
  const summary = useQuery({ queryKey: ['admin', 'api-log-summary', current], queryFn: () => api.apiLogSummary(current) })
  const detail = useQuery({ queryKey: ['admin', 'api-log', selectedId], queryFn: () => api.apiLog(selectedId as number), enabled: selectedId !== null })
  const clear = useMutation({ mutationFn: () => api.clearApiLogs({ ...current, confirmation: 'CLEAR_LOGS' }), onSuccess: () => { setClearText(''); void queryClient.invalidateQueries({ queryKey: ['admin', 'api-logs'] }); void queryClient.invalidateQueries({ queryKey: ['admin', 'api-log-summary'] }) } })
  const exportLogs = useMutation({ mutationFn: (format: 'csv' | 'jsonl') => api.exportApiLogs({ ...current, format }), onSuccess: ({ blob, filename }) => { const url = URL.createObjectURL(blob); const anchor = document.createElement('a'); anchor.href = url; anchor.download = filename.replaceAll('"', ''); anchor.click(); URL.revokeObjectURL(url) } })

  function update<K extends keyof Filters>(key: K, value: Filters[K]) { setFilters((previous) => ({ ...previous, [key]: value })) }
  function clearLogs() { if (clearText !== 'CLEAR_LOGS') return; if (window.confirm('确定清空当前筛选范围内的接口日志吗？该操作不可恢复。')) clear.mutate() }

  return (
    <div className="mx-auto max-w-7xl space-y-7 px-4 py-10 sm:px-6">
      <PageHeader eyebrow="Administration / Observability" title="接口日志" description="查询后端接口调用情况，定位错误、慢请求和 Webhook 接收问题。" action={<AppButton size="sm" variant="outline" disabled={logs.isFetching} onClick={() => void logs.refetch()}><Refresh size={16} />刷新</AppButton>} />
      {(clear.error || exportLogs.error) && <Message variant="error" title={errorMessage(clear.error || exportLogs.error, '日志操作失败')} />}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-5">
        {[['请求总数', summary.data?.total ?? 0], ['成功请求', summary.data?.success_count ?? 0], ['客户端错误', summary.data?.client_errors ?? 0], ['服务端错误', summary.data?.server_errors ?? 0], ['平均耗时', `${Math.round(summary.data?.average_ms ?? 0)} ms`]].map(([label, value]) => <div key={label} className="app-panel p-5"><p className="text-xs font-medium text-neutral-500">{label}</p><p className="mt-2 text-2xl font-bold tracking-tight">{value}</p></div>)}
      </div>
      <Panel title="筛选条件" icon={<FileText size={20} />}>
        <div className="grid gap-4 md:grid-cols-3 lg:grid-cols-4">
          <label className="space-y-2 text-sm font-medium"><span>开始时间</span><input className={inputClass} type="datetime-local" value={filters.startAt} onChange={(event) => update('startAt', event.target.value)} /></label>
          <label className="space-y-2 text-sm font-medium"><span>结束时间</span><input className={inputClass} type="datetime-local" value={filters.endAt} onChange={(event) => update('endAt', event.target.value)} /></label>
          <label className="space-y-2 text-sm font-medium"><span>Method</span><select className={inputClass} value={filters.method} onChange={(event) => update('method', event.target.value)}><option value="">全部</option>{['GET', 'POST', 'PUT', 'DELETE', 'PATCH'].map((method) => <option key={method}>{method}</option>)}</select></label>
          <label className="space-y-2 text-sm font-medium"><span>状态</span><select className={inputClass} value={filters.statusGroup} onChange={(event) => update('statusGroup', event.target.value)}><option value="">全部</option><option value="2xx">2xx 成功</option><option value="3xx">3xx 重定向</option><option value="4xx">4xx 客户端错误</option><option value="5xx">5xx 服务端错误</option></select></label>
          <label className="space-y-2 text-sm font-medium md:col-span-2"><span>路由</span><Input value={filters.route} onChange={(event) => update('route', event.target.value)} placeholder="例如 /api/webhooks/events" /></label>
          <label className="space-y-2 text-sm font-medium"><span>Request ID</span><Input value={filters.requestId} onChange={(event) => update('requestId', event.target.value)} placeholder="精确匹配" /></label>
          <label className="space-y-2 text-sm font-medium"><span>错误关键字</span><Input value={filters.keyword} onChange={(event) => update('keyword', event.target.value)} placeholder="搜索错误信息" /></label>
        </div>
        <div className="mt-5 flex flex-wrap gap-2"><AppButton size="sm" variant="outline" disabled={exportLogs.isPending} onClick={() => exportLogs.mutate('csv')}><Download size={16} />导出 CSV</AppButton><AppButton size="sm" variant="outline" disabled={exportLogs.isPending} onClick={() => exportLogs.mutate('jsonl')}><Download size={16} />导出 JSONL</AppButton></div>
      </Panel>
      <Panel title="日志列表" description={logs.data ? `共 ${logs.data.total} 条` : '正在加载…'}>
        {logs.isPending ? <div className="py-12 text-center text-sm text-neutral-500">正在加载日志…</div> : logs.error ? <ErrorState message={errorMessage(logs.error)} onRetry={() => void logs.refetch()} /> : !logs.data?.items.length ? <EmptyState title="暂无接口日志" description="调整筛选范围或等待新的接口请求。" /> : <div className="overflow-x-auto"><table className="w-full min-w-[920px] text-left text-sm"><thead><tr className="border-b border-neutral-200 text-xs text-neutral-500 dark:border-neutral-800"><th className="px-2 py-3">时间</th><th className="px-2 py-3">状态</th><th className="px-2 py-3">方法</th><th className="px-2 py-3">路由</th><th className="px-2 py-3">耗时</th><th className="px-2 py-3">用户 / App</th><th className="px-2 py-3">Request ID</th></tr></thead><tbody>{logs.data.items.map((item) => <tr key={item.id} className="cursor-pointer border-b border-neutral-100 transition hover:bg-neutral-50 dark:border-neutral-900 dark:hover:bg-neutral-900" onClick={() => setSelectedId(item.id)}><td className="whitespace-nowrap px-2 py-3 text-xs text-neutral-500">{formatDate(item.occurred_at)}</td><td className="px-2 py-3"><Badge variant={statusVariant(item.status_code)} size="sm">{item.status_code}</Badge></td><td className="px-2 py-3 font-mono text-xs font-semibold">{item.method}</td><td className="max-w-[280px] truncate px-2 py-3 font-mono text-xs">{item.route}</td><td className="whitespace-nowrap px-2 py-3 text-xs">{item.latency_ms} ms</td><td className="px-2 py-3 text-xs">{item.username || item.app_code || item.client_ip || '-'}</td><td className="max-w-[160px] truncate px-2 py-3 font-mono text-xs text-neutral-500">{item.request_id}</td></tr>)}</tbody></table></div>}
        {logs.data?.has_more && <div className="mt-4 text-center text-xs text-neutral-500">当前页面还有更多日志，可通过游标继续加载。</div>}
      </Panel>
      <Panel title="清空日志" description="清空操作只影响当前筛选范围，管理员操作会保留在不可清除的审计记录中。" icon={<Trash size={20} />}>
        <div className="flex flex-col gap-4 sm:flex-row sm:items-end"><label className="max-w-md flex-1 space-y-2 text-sm font-medium"><span>输入 CLEAR_LOGS 确认</span><Input value={clearText} onChange={(event) => setClearText(event.target.value)} placeholder="CLEAR_LOGS" /></label><AppButton variant="destructive" disabled={clearText !== 'CLEAR_LOGS' || clear.isPending} onClick={clearLogs}><Trash size={16} />清空当前范围</AppButton></div>
      </Panel>
      {selectedId !== null && <div className="fixed inset-0 z-50 flex justify-end bg-black/30" onClick={() => setSelectedId(null)}><aside className="h-full w-full max-w-2xl overflow-y-auto bg-white p-6 shadow-2xl dark:bg-neutral-950" onClick={(event) => event.stopPropagation()}><div className="flex items-center justify-between"><div><p className="text-xs font-semibold uppercase tracking-widest text-emerald-700">Log Detail</p><h2 className="mt-1 text-xl font-bold">接口调用详情</h2></div><AppButton size="sm" variant="outline" onClick={() => setSelectedId(null)}>关闭</AppButton></div>{detail.isPending ? <p className="mt-8 text-sm text-neutral-500">正在加载详情…</p> : detail.error ? <div className="mt-8"><Message variant="error" title={errorMessage(detail.error)} /></div> : detail.data ? <div className="mt-6 space-y-5"><div className="grid grid-cols-2 gap-3 text-sm">{[['时间', formatDate(detail.data.occurred_at)], ['状态', String(detail.data.status_code)], ['耗时', `${detail.data.latency_ms} ms`], ['Request ID', detail.data.request_id], ['用户', detail.data.username || '-'], ['App', detail.data.app_code || '-'], ['IP', detail.data.client_ip || '-'], ['User-Agent', detail.data.user_agent || '-']].map(([label, value]) => <div key={label} className="rounded-lg bg-neutral-50 p-3 dark:bg-neutral-900"><p className="text-xs text-neutral-500">{label}</p><p className="mt-1 break-all text-sm font-medium">{value}</p></div>)}</div>{detail.data.error_message && <div className="rounded-lg bg-red-50 p-4 text-sm text-red-800 dark:bg-red-950/30 dark:text-red-200"><div className="flex gap-2"><AlertCircle size={17} />{detail.data.error_message}</div></div>}<pre className="overflow-auto rounded-xl bg-neutral-950 p-4 text-xs leading-6 text-emerald-100">{JSON.stringify(detail.data, null, 2)}</pre></div> : null}</aside></div>}
    </div>
  )
}
